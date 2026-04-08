package models

import "time"

type Order struct {
	ID            string      `json:"id"`
	RestaurantID  string      `json:"restaurant_id"`
	TableID       string      `json:"table_id"`
	WaiterID      string      `json:"waiter_id"`
	OrderNumber   int64       `json:"order_number"`
	Status        string      `json:"status"`
	PaymentStatus string      `json:"payment_status"`
	PaymentMethod string      `json:"payment_method"`
	TotalAmount   float64     `json:"total_amount"`
	Notes         string      `json:"notes"`
	Items         []OrderItem `json:"items,omitempty"`
	CreatedAt     time.Time   `json:"created_at"`
	UpdatedAt     time.Time   `json:"updated_at"`
}

type OrderItem struct {
	ID         string    `json:"id"`
	OrderID    string    `json:"order_id"`
	MenuItemID string    `json:"menu_item_id"`
	Name       string    `json:"name"`
	Price      float64   `json:"price"`
	Quantity   int       `json:"quantity"`
	TotalPrice float64   `json:"total_price"`
	Status     string    `json:"status"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type CreateOrderItemInput struct {
	MenuItemID string `json:"menu_item_id"`
	Quantity   int    `json:"quantity"`
}

type CreateOrderRequest struct {
	RestaurantID string                 `json:"restaurant_id"`
	TableID      string                 `json:"table_id"`
	WaiterID     string                 `json:"waiter_id"`
	Items        []CreateOrderItemInput `json:"items"`
	Notes        string                 `json:"notes"`
}

type AddOrderItemRequest struct {
	MenuItemID string `json:"menu_item_id"`
	Quantity   int    `json:"quantity"`
}

type UpdateOrderItemQuantityRequest struct {
	Quantity int `json:"quantity"`
}

type UpdateOrderStatusRequest struct {
	Status string `json:"status"`
}

type UpdateOrderItemStatusRequest struct {
	Status string `json:"status"`
}

type UpdatePaymentStatusRequest struct {
	PaymentStatus string `json:"payment_status"`
	PaymentMethod string `json:"payment_method"`
}

type OrderDetails struct {
	Order Order       `json:"order"`
	Items []OrderItem `json:"items"`
}
