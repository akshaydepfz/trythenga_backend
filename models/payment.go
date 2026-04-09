package models

import "time"

type Payment struct {
	ID            string    `json:"id"`
	OrderID       string    `json:"order_id"`
	RestaurantID  string    `json:"restaurant_id"`
	Amount        float64   `json:"amount"`
	PaymentMethod string    `json:"payment_method"`
	PaymentStatus string    `json:"payment_status"`
	TransactionID string    `json:"transaction_id,omitempty"`
	PaidAt        time.Time `json:"paid_at"`
	CreatedAt     time.Time `json:"created_at"`
}

type CreatePaymentRequest struct {
	OrderID       string  `json:"order_id"`
	Amount        float64 `json:"amount"`
	PaymentMethod string  `json:"payment_method"`
	RestaurantID  string  `json:"restaurant_id,omitempty"`
	TransactionID string  `json:"transaction_id,omitempty"`
}

type UpdatePaymentRequest struct {
	PaymentStatus string `json:"payment_status"`
}

type PaymentSummary struct {
	OrderID     string  `json:"order_id"`
	TotalAmount float64 `json:"total_amount"`
	TotalPaid   float64 `json:"total_paid"`
	Remaining   float64 `json:"remaining"`
	Status      string  `json:"status"`
}
