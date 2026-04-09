package repository

import (
	"context"
	"database/sql"

	"github.com/lib/pq"

	"trythenga.com/models"
)

type OrderRepository struct {
	DB *sql.DB
}

func NewOrderRepository(db *sql.DB) *OrderRepository {
	return &OrderRepository{DB: db}
}

func (r *OrderRepository) CreateOrderTx(ctx context.Context, order models.Order, items []models.OrderItem) (models.Order, []models.OrderItem, error) {
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return models.Order{}, nil, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	createdOrder, err := r.insertOrder(ctx, tx, order)
	if err != nil {
		return models.Order{}, nil, err
	}

	createdItems := make([]models.OrderItem, 0, len(items))
	for _, item := range items {
		item.OrderID = createdOrder.ID
		createdItem, createErr := r.insertOrderItem(ctx, tx, item)
		if createErr != nil {
			err = createErr
			return models.Order{}, nil, err
		}
		createdItems = append(createdItems, createdItem)
	}

	if _, err = tx.ExecContext(ctx, `UPDATE tables SET status = 'occupied', updated_at = NOW() WHERE id = $1`, createdOrder.TableID); err != nil {
		return models.Order{}, nil, err
	}

	if commitErr := tx.Commit(); commitErr != nil {
		return models.Order{}, nil, commitErr
	}
	return createdOrder, createdItems, nil
}

func (r *OrderRepository) GetOrdersByRestaurant(ctx context.Context, restaurantID string) ([]models.Order, error) {
	query := `
		SELECT id, restaurant_id, table_id, COALESCE(table_number, ''), waiter_id, guest_count, order_number, status, payment_status, COALESCE(payment_method, ''), total_amount, COALESCE(notes, ''), created_at, updated_at
		FROM orders
		WHERE restaurant_id = $1 AND status <> 'cancelled'
		ORDER BY created_at DESC;
	`
	rows, err := r.DB.QueryContext(ctx, query, restaurantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	orders := make([]models.Order, 0)
	for rows.Next() {
		order, scanErr := scanOrderFromRows(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		orders = append(orders, order)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return orders, nil
}

func (r *OrderRepository) GetOrderByID(ctx context.Context, orderID string) (models.Order, error) {
	row := r.DB.QueryRowContext(ctx, `
		SELECT id, restaurant_id, table_id, COALESCE(table_number, ''), waiter_id, guest_count, order_number, status, payment_status, COALESCE(payment_method, ''), total_amount, COALESCE(notes, ''), created_at, updated_at
		FROM orders
		WHERE id = $1;
	`, orderID)
	return scanOrder(row)
}

func (r *OrderRepository) UpdateOrderStatus(ctx context.Context, orderID, status string) (models.Order, error) {
	row := r.DB.QueryRowContext(ctx, `
		UPDATE orders
		SET status = $2, updated_at = NOW()
		WHERE id = $1
		RETURNING id, restaurant_id, table_id, COALESCE(table_number, ''), waiter_id, guest_count, order_number, status, payment_status, COALESCE(payment_method, ''), total_amount, COALESCE(notes, ''), created_at, updated_at;
	`, orderID, status)
	return scanOrder(row)
}

func (r *OrderRepository) UpdatePaymentStatus(ctx context.Context, orderID, paymentStatus, paymentMethod string) (models.Order, error) {
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return models.Order{}, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	row := tx.QueryRowContext(ctx, `
		UPDATE orders
		SET payment_status = $2, payment_method = NULLIF($3, ''), updated_at = NOW()
		WHERE id = $1
		RETURNING id, restaurant_id, table_id, COALESCE(table_number, ''), waiter_id, guest_count, order_number, status, payment_status, COALESCE(payment_method, ''), total_amount, COALESCE(notes, ''), created_at, updated_at;
	`, orderID, paymentStatus, paymentMethod)
	updatedOrder, err := scanOrder(row)
	if err != nil {
		return models.Order{}, err
	}

	if paymentStatus == "paid" {
		if _, err = tx.ExecContext(ctx, `UPDATE tables SET status = 'vacant', updated_at = NOW() WHERE id = $1`, updatedOrder.TableID); err != nil {
			return models.Order{}, err
		}
	}

	if commitErr := tx.Commit(); commitErr != nil {
		return models.Order{}, commitErr
	}
	return updatedOrder, nil
}

func (r *OrderRepository) CreateOrderItem(ctx context.Context, orderID string, item models.OrderItem) (models.OrderItem, models.Order, error) {
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return models.OrderItem{}, models.Order{}, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	item.OrderID = orderID
	createdItem, err := r.insertOrderItem(ctx, tx, item)
	if err != nil {
		return models.OrderItem{}, models.Order{}, err
	}

	if err = r.recalculateOrderTotalTx(ctx, tx, orderID); err != nil {
		return models.OrderItem{}, models.Order{}, err
	}

	updatedOrder, err := r.getOrderByIDTx(ctx, tx, orderID)
	if err != nil {
		return models.OrderItem{}, models.Order{}, err
	}

	if commitErr := tx.Commit(); commitErr != nil {
		return models.OrderItem{}, models.Order{}, commitErr
	}
	return createdItem, updatedOrder, nil
}

func (r *OrderRepository) GetItemsByOrderID(ctx context.Context, orderID string) ([]models.OrderItem, error) {
	rows, err := r.DB.QueryContext(ctx, `
		SELECT id, order_id, menu_item_id, name, price, quantity, total_price, status, created_at, updated_at
		FROM order_items
		WHERE order_id = $1
		ORDER BY created_at ASC;
	`, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]models.OrderItem, 0)
	for rows.Next() {
		item, scanErr := scanOrderItemFromRows(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func (r *OrderRepository) GetItemsByOrderIDs(ctx context.Context, orderIDs []string) (map[string][]models.OrderItem, error) {
	itemsByOrderID := make(map[string][]models.OrderItem)
	if len(orderIDs) == 0 {
		return itemsByOrderID, nil
	}

	rows, err := r.DB.QueryContext(ctx, `
		SELECT id, order_id, menu_item_id, name, price, quantity, total_price, status, created_at, updated_at
		FROM order_items
		WHERE order_id = ANY($1::uuid[])
		ORDER BY created_at ASC;
	`, pq.Array(orderIDs))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		item, scanErr := scanOrderItemFromRows(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		itemsByOrderID[item.OrderID] = append(itemsByOrderID[item.OrderID], item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return itemsByOrderID, nil
}

func (r *OrderRepository) GetOrderItemByID(ctx context.Context, itemID string) (models.OrderItem, error) {
	row := r.DB.QueryRowContext(ctx, `
		SELECT id, order_id, menu_item_id, name, price, quantity, total_price, status, created_at, updated_at
		FROM order_items
		WHERE id = $1;
	`, itemID)
	return scanOrderItem(row)
}

func (r *OrderRepository) UpdateItemQuantity(ctx context.Context, itemID string, quantity int) (models.OrderItem, models.Order, error) {
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return models.OrderItem{}, models.Order{}, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	var orderID string
	if err = tx.QueryRowContext(ctx, `
		UPDATE order_items
		SET quantity = $2, total_price = price * $2, updated_at = NOW()
		WHERE id = $1
		RETURNING order_id;
	`, itemID, quantity).Scan(&orderID); err != nil {
		return models.OrderItem{}, models.Order{}, err
	}

	if err = r.recalculateOrderTotalTx(ctx, tx, orderID); err != nil {
		return models.OrderItem{}, models.Order{}, err
	}

	updatedItem, err := r.getOrderItemByIDTx(ctx, tx, itemID)
	if err != nil {
		return models.OrderItem{}, models.Order{}, err
	}
	updatedOrder, err := r.getOrderByIDTx(ctx, tx, orderID)
	if err != nil {
		return models.OrderItem{}, models.Order{}, err
	}

	if commitErr := tx.Commit(); commitErr != nil {
		return models.OrderItem{}, models.Order{}, commitErr
	}
	return updatedItem, updatedOrder, nil
}

func (r *OrderRepository) DeleteItem(ctx context.Context, itemID string) (models.Order, error) {
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return models.Order{}, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	var orderID string
	if err = tx.QueryRowContext(ctx, `SELECT order_id FROM order_items WHERE id = $1`, itemID).Scan(&orderID); err != nil {
		return models.Order{}, err
	}

	result, err := tx.ExecContext(ctx, `DELETE FROM order_items WHERE id = $1`, itemID)
	if err != nil {
		return models.Order{}, err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return models.Order{}, err
	}
	if rowsAffected == 0 {
		return models.Order{}, sql.ErrNoRows
	}

	if err = r.recalculateOrderTotalTx(ctx, tx, orderID); err != nil {
		return models.Order{}, err
	}

	updatedOrder, err := r.getOrderByIDTx(ctx, tx, orderID)
	if err != nil {
		return models.Order{}, err
	}

	if commitErr := tx.Commit(); commitErr != nil {
		return models.Order{}, commitErr
	}
	return updatedOrder, nil
}

func (r *OrderRepository) UpdateItemStatus(ctx context.Context, itemID, status string) (models.OrderItem, error) {
	row := r.DB.QueryRowContext(ctx, `
		UPDATE order_items
		SET status = $2, updated_at = NOW()
		WHERE id = $1
		RETURNING id, order_id, menu_item_id, name, price, quantity, total_price, status, created_at, updated_at;
	`, itemID, status)
	return scanOrderItem(row)
}

func (r *OrderRepository) GetCurrentOrderByTable(ctx context.Context, tableID string) (models.Order, error) {
	row := r.DB.QueryRowContext(ctx, `
		SELECT id, restaurant_id, table_id, COALESCE(table_number, ''), waiter_id, guest_count, order_number, status, payment_status, COALESCE(payment_method, ''), total_amount, COALESCE(notes, ''), created_at, updated_at
		FROM orders
		WHERE table_id = $1
		  AND status IN ('pending', 'preparing', 'served')
		ORDER BY created_at DESC
		LIMIT 1;
	`, tableID)
	return scanOrder(row)
}

func (r *OrderRepository) MenuItemByID(ctx context.Context, menuItemID string) (models.MenuItem, error) {
	row := r.DB.QueryRowContext(ctx, `
		SELECT id, restaurant_id, category_id, name, description, price, is_available, COALESCE(image_url, ''), created_at, updated_at
		FROM menu_items
		WHERE id = $1;
	`, menuItemID)
	var item models.MenuItem
	if err := row.Scan(
		&item.ID,
		&item.RestaurantID,
		&item.CategoryID,
		&item.Name,
		&item.Description,
		&item.Price,
		&item.IsAvailable,
		&item.ImageURL,
		&item.CreatedAt,
		&item.UpdatedAt,
	); err != nil {
		return models.MenuItem{}, err
	}
	return item, nil
}

func (r *OrderRepository) TableByID(ctx context.Context, tableID string) (models.Table, error) {
	row := r.DB.QueryRowContext(ctx, `
		SELECT id, restaurant_id, floor_id, table_number, capacity, status, pos_x, pos_y, shape, is_active, created_at, updated_at
		FROM tables
		WHERE id = $1 AND is_active = TRUE;
	`, tableID)
	return scanTable(row)
}

func (r *OrderRepository) WaiterByID(ctx context.Context, waiterID string) (models.Waiter, error) {
	row := r.DB.QueryRowContext(ctx, `
		SELECT id, restaurant_id, name, COALESCE(phone, ''), COALESCE(password_hash, ''), role, is_active, created_at, updated_at
		FROM waiters
		WHERE id = $1 AND is_active = TRUE;
	`, waiterID)
	var waiter models.Waiter
	if err := row.Scan(
		&waiter.ID,
		&waiter.RestaurantID,
		&waiter.Name,
		&waiter.Phone,
		&waiter.PasswordHash,
		&waiter.Role,
		&waiter.IsActive,
		&waiter.CreatedAt,
		&waiter.UpdatedAt,
	); err != nil {
		return models.Waiter{}, err
	}
	return waiter, nil
}

func (r *OrderRepository) insertOrder(ctx context.Context, tx *sql.Tx, order models.Order) (models.Order, error) {
	row := tx.QueryRowContext(ctx, `
		INSERT INTO orders (id, restaurant_id, table_id, table_number, waiter_id, guest_count, status, payment_status, total_amount, notes)
		VALUES ($1, $2, $3, (SELECT table_number FROM tables WHERE id = $3), $4, $5, $6, $7, $8, NULLIF($9, ''))
		RETURNING id, restaurant_id, table_id, COALESCE(table_number, ''), waiter_id, guest_count, order_number, status, payment_status, COALESCE(payment_method, ''), total_amount, COALESCE(notes, ''), created_at, updated_at;
	`, order.ID, order.RestaurantID, order.TableID, order.WaiterID, order.GuestCount, order.Status, order.PaymentStatus, order.TotalAmount, order.Notes)
	return scanOrder(row)
}

func (r *OrderRepository) insertOrderItem(ctx context.Context, tx *sql.Tx, item models.OrderItem) (models.OrderItem, error) {
	row := tx.QueryRowContext(ctx, `
		INSERT INTO order_items (id, order_id, menu_item_id, name, price, quantity, total_price, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, order_id, menu_item_id, name, price, quantity, total_price, status, created_at, updated_at;
	`, item.ID, item.OrderID, item.MenuItemID, item.Name, item.Price, item.Quantity, item.TotalPrice, item.Status)
	return scanOrderItem(row)
}

func (r *OrderRepository) recalculateOrderTotalTx(ctx context.Context, tx *sql.Tx, orderID string) error {
	_, err := tx.ExecContext(ctx, `
		UPDATE orders
		SET total_amount = COALESCE((SELECT SUM(total_price) FROM order_items WHERE order_id = $1), 0),
		    updated_at = NOW()
		WHERE id = $1;
	`, orderID)
	return err
}

func (r *OrderRepository) getOrderByIDTx(ctx context.Context, tx *sql.Tx, orderID string) (models.Order, error) {
	row := tx.QueryRowContext(ctx, `
		SELECT id, restaurant_id, table_id, COALESCE(table_number, ''), waiter_id, guest_count, order_number, status, payment_status, COALESCE(payment_method, ''), total_amount, COALESCE(notes, ''), created_at, updated_at
		FROM orders
		WHERE id = $1;
	`, orderID)
	return scanOrder(row)
}

func (r *OrderRepository) getOrderItemByIDTx(ctx context.Context, tx *sql.Tx, itemID string) (models.OrderItem, error) {
	row := tx.QueryRowContext(ctx, `
		SELECT id, order_id, menu_item_id, name, price, quantity, total_price, status, created_at, updated_at
		FROM order_items
		WHERE id = $1;
	`, itemID)
	return scanOrderItem(row)
}

func scanOrder(row *sql.Row) (models.Order, error) {
	var order models.Order
	err := row.Scan(
		&order.ID,
		&order.RestaurantID,
		&order.TableID,
		&order.TableNumber,
		&order.WaiterID,
		&order.GuestCount,
		&order.OrderNumber,
		&order.Status,
		&order.PaymentStatus,
		&order.PaymentMethod,
		&order.TotalAmount,
		&order.Notes,
		&order.CreatedAt,
		&order.UpdatedAt,
	)
	return order, err
}

func scanOrderFromRows(rows *sql.Rows) (models.Order, error) {
	var order models.Order
	err := rows.Scan(
		&order.ID,
		&order.RestaurantID,
		&order.TableID,
		&order.TableNumber,
		&order.WaiterID,
		&order.GuestCount,
		&order.OrderNumber,
		&order.Status,
		&order.PaymentStatus,
		&order.PaymentMethod,
		&order.TotalAmount,
		&order.Notes,
		&order.CreatedAt,
		&order.UpdatedAt,
	)
	return order, err
}

func scanOrderItem(row *sql.Row) (models.OrderItem, error) {
	var item models.OrderItem
	err := row.Scan(
		&item.ID,
		&item.OrderID,
		&item.MenuItemID,
		&item.Name,
		&item.Price,
		&item.Quantity,
		&item.TotalPrice,
		&item.Status,
		&item.CreatedAt,
		&item.UpdatedAt,
	)
	return item, err
}

func scanOrderItemFromRows(rows *sql.Rows) (models.OrderItem, error) {
	var item models.OrderItem
	err := rows.Scan(
		&item.ID,
		&item.OrderID,
		&item.MenuItemID,
		&item.Name,
		&item.Price,
		&item.Quantity,
		&item.TotalPrice,
		&item.Status,
		&item.CreatedAt,
		&item.UpdatedAt,
	)
	return item, err
}
