package service

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"github.com/google/uuid"

	"trythenga.com/models"
	"trythenga.com/repository"
)

var allowedOrderStatuses = map[string]bool{
	"pending":   true,
	"preparing": true,
	"served":    true,
	"completed": true,
	"cancelled": true,
}

var allowedPaymentStatuses = map[string]bool{
	"unpaid": true,
	"paid":   true,
}

var allowedOrderItemStatuses = map[string]bool{
	"pending":   true,
	"preparing": true,
	"served":    true,
}

type OrderService struct {
	repo *repository.OrderRepository
}

func NewOrderService(repo *repository.OrderRepository) *OrderService {
	return &OrderService{repo: repo}
}

func (s *OrderService) CreateOrder(ctx context.Context, req models.CreateOrderRequest) (models.OrderDetails, error) {
	req.RestaurantID = strings.TrimSpace(req.RestaurantID)
	req.TableID = strings.TrimSpace(req.TableID)
	req.WaiterID = strings.TrimSpace(req.WaiterID)
	req.Notes = strings.TrimSpace(req.Notes)

	if req.RestaurantID == "" {
		return models.OrderDetails{}, errors.New("restaurant_id is required")
	}
	if req.TableID == "" {
		return models.OrderDetails{}, errors.New("table_id is required")
	}
	if req.WaiterID == "" {
		return models.OrderDetails{}, errors.New("waiter_id is required")
	}
	if len(req.Items) == 0 {
		return models.OrderDetails{}, errors.New("items are required")
	}

	if err := s.validateRestaurantRelations(ctx, req.RestaurantID, req.TableID, req.WaiterID); err != nil {
		return models.OrderDetails{}, err
	}

	orderItems := make([]models.OrderItem, 0, len(req.Items))
	totalAmount := 0.0
	for _, item := range req.Items {
		item.MenuItemID = strings.TrimSpace(item.MenuItemID)
		if item.MenuItemID == "" {
			return models.OrderDetails{}, errors.New("menu_item_id is required")
		}
		if item.Quantity <= 0 {
			return models.OrderDetails{}, errors.New("quantity must be greater than 0")
		}
		menuItem, err := s.repo.MenuItemByID(ctx, item.MenuItemID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return models.OrderDetails{}, errors.New("menu item not found")
			}
			return models.OrderDetails{}, err
		}
		if menuItem.RestaurantID != req.RestaurantID {
			return models.OrderDetails{}, errors.New("menu item does not belong to the restaurant")
		}
		totalPrice := menuItem.Price * float64(item.Quantity)
		totalAmount += totalPrice

		orderItems = append(orderItems, models.OrderItem{
			ID:         uuid.NewString(),
			MenuItemID: menuItem.ID,
			Name:       menuItem.Name,
			Price:      menuItem.Price,
			Quantity:   item.Quantity,
			TotalPrice: totalPrice,
			Status:     "pending",
		})
	}

	order := models.Order{
		ID:            uuid.NewString(),
		RestaurantID:  req.RestaurantID,
		TableID:       req.TableID,
		WaiterID:      req.WaiterID,
		Status:        "pending",
		PaymentStatus: "unpaid",
		TotalAmount:   totalAmount,
		Notes:         req.Notes,
	}

	createdOrder, createdItems, err := s.repo.CreateOrderTx(ctx, order, orderItems)
	if err != nil {
		return models.OrderDetails{}, err
	}
	return models.OrderDetails{Order: createdOrder, Items: createdItems}, nil
}

func (s *OrderService) GetOrdersByRestaurant(ctx context.Context, restaurantID string, includeItems bool) ([]models.Order, error) {
	restaurantID = strings.TrimSpace(restaurantID)
	if restaurantID == "" {
		return nil, errors.New("restaurant_id is required")
	}
	orders, err := s.repo.GetOrdersByRestaurant(ctx, restaurantID)
	if err != nil {
		return nil, err
	}
	if !includeItems || len(orders) == 0 {
		return orders, nil
	}

	orderIDs := make([]string, 0, len(orders))
	for _, order := range orders {
		orderIDs = append(orderIDs, order.ID)
	}

	itemsByOrderID, err := s.repo.GetItemsByOrderIDs(ctx, orderIDs)
	if err != nil {
		return nil, err
	}
	for i := range orders {
		orders[i].Items = itemsByOrderID[orders[i].ID]
	}
	return orders, nil
}

func (s *OrderService) GetOrderDetails(ctx context.Context, orderID string) (models.OrderDetails, error) {
	order, err := s.repo.GetOrderByID(ctx, orderID)
	if err != nil {
		return models.OrderDetails{}, err
	}
	items, err := s.repo.GetItemsByOrderID(ctx, orderID)
	if err != nil {
		return models.OrderDetails{}, err
	}
	return models.OrderDetails{Order: order, Items: items}, nil
}

func (s *OrderService) AddItemToOrder(ctx context.Context, orderID string, req models.AddOrderItemRequest) (models.OrderDetails, error) {
	req.MenuItemID = strings.TrimSpace(req.MenuItemID)
	if req.MenuItemID == "" {
		return models.OrderDetails{}, errors.New("menu_item_id is required")
	}
	if req.Quantity <= 0 {
		return models.OrderDetails{}, errors.New("quantity must be greater than 0")
	}

	order, err := s.repo.GetOrderByID(ctx, orderID)
	if err != nil {
		return models.OrderDetails{}, err
	}
	menuItem, err := s.repo.MenuItemByID(ctx, req.MenuItemID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.OrderDetails{}, errors.New("menu item not found")
		}
		return models.OrderDetails{}, err
	}
	if menuItem.RestaurantID != order.RestaurantID {
		return models.OrderDetails{}, errors.New("menu item does not belong to the restaurant")
	}

	newItem := models.OrderItem{
		ID:         uuid.NewString(),
		MenuItemID: menuItem.ID,
		Name:       menuItem.Name,
		Price:      menuItem.Price,
		Quantity:   req.Quantity,
		TotalPrice: menuItem.Price * float64(req.Quantity),
		Status:     "pending",
	}
	_, updatedOrder, err := s.repo.CreateOrderItem(ctx, orderID, newItem)
	if err != nil {
		return models.OrderDetails{}, err
	}
	items, err := s.repo.GetItemsByOrderID(ctx, orderID)
	if err != nil {
		return models.OrderDetails{}, err
	}
	return models.OrderDetails{Order: updatedOrder, Items: items}, nil
}

func (s *OrderService) UpdateItemQuantity(ctx context.Context, itemID string, quantity int) (models.OrderDetails, error) {
	if quantity <= 0 {
		return models.OrderDetails{}, errors.New("quantity must be greater than 0")
	}
	_, updatedOrder, err := s.repo.UpdateItemQuantity(ctx, itemID, quantity)
	if err != nil {
		return models.OrderDetails{}, err
	}
	items, err := s.repo.GetItemsByOrderID(ctx, updatedOrder.ID)
	if err != nil {
		return models.OrderDetails{}, err
	}
	return models.OrderDetails{Order: updatedOrder, Items: items}, nil
}

func (s *OrderService) RemoveItem(ctx context.Context, itemID string) (models.OrderDetails, error) {
	updatedOrder, err := s.repo.DeleteItem(ctx, itemID)
	if err != nil {
		return models.OrderDetails{}, err
	}
	items, err := s.repo.GetItemsByOrderID(ctx, updatedOrder.ID)
	if err != nil {
		return models.OrderDetails{}, err
	}
	return models.OrderDetails{Order: updatedOrder, Items: items}, nil
}

func (s *OrderService) UpdateOrderStatus(ctx context.Context, orderID, status string) (models.Order, error) {
	status = normalizeOrderStatus(status)
	if status == "" {
		return models.Order{}, errors.New("status is required")
	}
	if !allowedOrderStatuses[status] {
		return models.Order{}, errors.New("invalid status value")
	}
	return s.repo.UpdateOrderStatus(ctx, orderID, status)
}

func (s *OrderService) UpdateOrderItemStatus(ctx context.Context, itemID, status string) (models.OrderItem, error) {
	status = normalizeOrderStatus(status)
	if status == "" {
		return models.OrderItem{}, errors.New("status is required")
	}
	if !allowedOrderItemStatuses[status] {
		return models.OrderItem{}, errors.New("invalid item status value")
	}
	return s.repo.UpdateItemStatus(ctx, itemID, status)
}

func (s *OrderService) CompletePayment(ctx context.Context, orderID, paymentStatus, paymentMethod string) (models.Order, error) {
	paymentStatus = normalizeOrderStatus(paymentStatus)
	paymentMethod = strings.TrimSpace(paymentMethod)
	if paymentStatus == "" {
		return models.Order{}, errors.New("payment_status is required")
	}
	if !allowedPaymentStatuses[paymentStatus] {
		return models.Order{}, errors.New("invalid payment_status value")
	}
	return s.repo.UpdatePaymentStatus(ctx, orderID, paymentStatus, paymentMethod)
}

func (s *OrderService) GetCurrentOrderByTable(ctx context.Context, tableID string) (models.OrderDetails, error) {
	order, err := s.repo.GetCurrentOrderByTable(ctx, tableID)
	if err != nil {
		return models.OrderDetails{}, err
	}
	items, err := s.repo.GetItemsByOrderID(ctx, order.ID)
	if err != nil {
		return models.OrderDetails{}, err
	}
	return models.OrderDetails{Order: order, Items: items}, nil
}

func (s *OrderService) validateRestaurantRelations(ctx context.Context, restaurantID, tableID, waiterID string) error {
	table, err := s.repo.TableByID(ctx, tableID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return errors.New("table not found")
		}
		return err
	}
	if table.RestaurantID != restaurantID {
		return errors.New("table does not belong to the restaurant")
	}

	waiter, err := s.repo.WaiterByID(ctx, waiterID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return errors.New("waiter not found")
		}
		return err
	}
	if waiter.RestaurantID != restaurantID {
		return errors.New("waiter does not belong to the restaurant")
	}
	return nil
}

func normalizeOrderStatus(status string) string {
	return strings.TrimSpace(strings.ToLower(status))
}

func IsOrderValidationError(err error) bool {
	switch err.Error() {
	case "restaurant_id is required",
		"table_id is required",
		"waiter_id is required",
		"items are required",
		"menu_item_id is required",
		"quantity must be greater than 0",
		"menu item not found",
		"menu item does not belong to the restaurant",
		"table not found",
		"table does not belong to the restaurant",
		"waiter not found",
		"waiter does not belong to the restaurant",
		"status is required",
		"invalid status value",
		"invalid item status value",
		"payment_status is required",
		"invalid payment_status value":
		return true
	default:
		return false
	}
}
