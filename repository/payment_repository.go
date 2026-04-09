package repository

import (
	"context"
	"database/sql"
	"time"

	"trythenga.com/models"
)

type PaymentRepository struct {
	DB *sql.DB
}

func NewPaymentRepository(db *sql.DB) *PaymentRepository {
	return &PaymentRepository{DB: db}
}

func (r *PaymentRepository) CreatePaymentTx(ctx context.Context, payment models.Payment) (models.Payment, models.Order, float64, error) {
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return models.Payment{}, models.Order{}, 0, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	order, err := r.getOrderForPaymentTx(ctx, tx, payment.OrderID)
	if err != nil {
		return models.Payment{}, models.Order{}, 0, err
	}

	row := tx.QueryRowContext(ctx, `
		INSERT INTO payments (id, order_id, restaurant_id, amount, payment_method, payment_status, transaction_id, paid_at)
		VALUES ($1, $2, $3, $4, $5, $6, NULLIF($7, ''), $8)
		RETURNING id, order_id, restaurant_id, amount, payment_method, payment_status, COALESCE(transaction_id, ''), paid_at, created_at;
	`, payment.ID, payment.OrderID, payment.RestaurantID, payment.Amount, payment.PaymentMethod, payment.PaymentStatus, payment.TransactionID, payment.PaidAt)
	createdPayment, err := scanPayment(row)
	if err != nil {
		return models.Payment{}, models.Order{}, 0, err
	}

	totalPaid, err := r.getTotalPaidByOrderTx(ctx, tx, payment.OrderID)
	if err != nil {
		return models.Payment{}, models.Order{}, 0, err
	}

	if totalPaid >= order.TotalAmount {
		if _, err = tx.ExecContext(ctx, `
			UPDATE orders
			SET payment_status = 'paid', status = 'completed', updated_at = NOW()
			WHERE id = $1;
		`, order.ID); err != nil {
			return models.Payment{}, models.Order{}, 0, err
		}
		if _, err = tx.ExecContext(ctx, `UPDATE tables SET status = 'vacant', updated_at = NOW() WHERE id = $1`, order.TableID); err != nil {
			return models.Payment{}, models.Order{}, 0, err
		}
	}

	updatedOrder, err := r.getOrderByIDTx(ctx, tx, order.ID)
	if err != nil {
		return models.Payment{}, models.Order{}, 0, err
	}

	if commitErr := tx.Commit(); commitErr != nil {
		return models.Payment{}, models.Order{}, 0, commitErr
	}
	return createdPayment, updatedOrder, totalPaid, nil
}

func (r *PaymentRepository) GetPaymentsByOrder(ctx context.Context, orderID string) ([]models.Payment, error) {
	rows, err := r.DB.QueryContext(ctx, `
		SELECT id, order_id, restaurant_id, amount, payment_method, payment_status, COALESCE(transaction_id, ''), paid_at, created_at
		FROM payments
		WHERE order_id = $1
		ORDER BY created_at ASC;
	`, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	payments := make([]models.Payment, 0)
	for rows.Next() {
		payment, scanErr := scanPaymentFromRows(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		payments = append(payments, payment)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return payments, nil
}

func (r *PaymentRepository) GetPaymentByID(ctx context.Context, id string) (models.Payment, error) {
	row := r.DB.QueryRowContext(ctx, `
		SELECT id, order_id, restaurant_id, amount, payment_method, payment_status, COALESCE(transaction_id, ''), paid_at, created_at
		FROM payments
		WHERE id = $1;
	`, id)
	return scanPayment(row)
}

func (r *PaymentRepository) UpdatePayment(ctx context.Context, id, status string) (models.Payment, error) {
	row := r.DB.QueryRowContext(ctx, `
		UPDATE payments
		SET payment_status = $2
		WHERE id = $1
		RETURNING id, order_id, restaurant_id, amount, payment_method, payment_status, COALESCE(transaction_id, ''), paid_at, created_at;
	`, id, status)
	return scanPayment(row)
}

func (r *PaymentRepository) DeletePayment(ctx context.Context, id string) error {
	result, err := r.DB.ExecContext(ctx, `DELETE FROM payments WHERE id = $1`, id)
	if err != nil {
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *PaymentRepository) GetTotalPaidByOrder(ctx context.Context, orderID string) (float64, error) {
	row := r.DB.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(amount), 0)
		FROM payments
		WHERE order_id = $1;
	`, orderID)
	var total float64
	if err := row.Scan(&total); err != nil {
		return 0, err
	}
	return total, nil
}

func (r *PaymentRepository) GetOrderByID(ctx context.Context, orderID string) (models.Order, error) {
	row := r.DB.QueryRowContext(ctx, `
		SELECT id, restaurant_id, table_id, waiter_id, guest_count, order_number, status, payment_status, COALESCE(payment_method, ''), total_amount, COALESCE(notes, ''), created_at, updated_at
		FROM orders
		WHERE id = $1;
	`, orderID)
	return scanOrder(row)
}

func (r *PaymentRepository) getOrderForPaymentTx(ctx context.Context, tx *sql.Tx, orderID string) (models.Order, error) {
	row := tx.QueryRowContext(ctx, `
		SELECT id, restaurant_id, table_id, waiter_id, guest_count, order_number, status, payment_status, COALESCE(payment_method, ''), total_amount, COALESCE(notes, ''), created_at, updated_at
		FROM orders
		WHERE id = $1
		FOR UPDATE;
	`, orderID)
	return scanOrder(row)
}

func (r *PaymentRepository) getTotalPaidByOrderTx(ctx context.Context, tx *sql.Tx, orderID string) (float64, error) {
	row := tx.QueryRowContext(ctx, `SELECT COALESCE(SUM(amount), 0) FROM payments WHERE order_id = $1`, orderID)
	var total float64
	if err := row.Scan(&total); err != nil {
		return 0, err
	}
	return total, nil
}

func (r *PaymentRepository) getOrderByIDTx(ctx context.Context, tx *sql.Tx, orderID string) (models.Order, error) {
	row := tx.QueryRowContext(ctx, `
		SELECT id, restaurant_id, table_id, waiter_id, guest_count, order_number, status, payment_status, COALESCE(payment_method, ''), total_amount, COALESCE(notes, ''), created_at, updated_at
		FROM orders
		WHERE id = $1;
	`, orderID)
	return scanOrder(row)
}

func scanPayment(row *sql.Row) (models.Payment, error) {
	var payment models.Payment
	err := row.Scan(
		&payment.ID,
		&payment.OrderID,
		&payment.RestaurantID,
		&payment.Amount,
		&payment.PaymentMethod,
		&payment.PaymentStatus,
		&payment.TransactionID,
		&payment.PaidAt,
		&payment.CreatedAt,
	)
	return payment, err
}

func scanPaymentFromRows(rows *sql.Rows) (models.Payment, error) {
	var payment models.Payment
	err := rows.Scan(
		&payment.ID,
		&payment.OrderID,
		&payment.RestaurantID,
		&payment.Amount,
		&payment.PaymentMethod,
		&payment.PaymentStatus,
		&payment.TransactionID,
		&payment.PaidAt,
		&payment.CreatedAt,
	)
	return payment, err
}

func nowUTC() time.Time {
	return time.Now().UTC()
}
