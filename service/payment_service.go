package service

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"

	"trythenga.com/models"
	"trythenga.com/repository"
)

var allowedCreatePaymentMethods = map[string]bool{
	"cash": true,
	"upi":  true,
	"card": true,
}

var allowedPaymentStates = map[string]bool{
	"success":  true,
	"failed":   true,
	"pending":  true,
	"refunded": true,
}

type PaymentService struct {
	repo *repository.PaymentRepository
}

func NewPaymentService(repo *repository.PaymentRepository) *PaymentService {
	return &PaymentService{repo: repo}
}

func (s *PaymentService) CreatePayment(ctx context.Context, req models.CreatePaymentRequest) (models.Payment, error) {
	req.OrderID = strings.TrimSpace(req.OrderID)
	req.PaymentMethod = normalizePaymentValue(req.PaymentMethod)
	req.RestaurantID = strings.TrimSpace(req.RestaurantID)
	req.TransactionID = strings.TrimSpace(req.TransactionID)

	if req.OrderID == "" {
		return models.Payment{}, errors.New("order_id is required")
	}
	if req.Amount <= 0 {
		return models.Payment{}, errors.New("amount must be greater than 0")
	}
	if req.PaymentMethod == "" {
		return models.Payment{}, errors.New("payment_method is required")
	}
	if !allowedCreatePaymentMethods[req.PaymentMethod] {
		return models.Payment{}, errors.New("invalid payment_method value")
	}

	order, err := s.repo.GetOrderByID(ctx, req.OrderID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.Payment{}, errors.New("order not found")
		}
		return models.Payment{}, err
	}

	if req.RestaurantID != "" && req.RestaurantID != order.RestaurantID {
		return models.Payment{}, errors.New("order does not belong to the restaurant")
	}

	payment := models.Payment{
		ID:            uuid.NewString(),
		OrderID:       req.OrderID,
		RestaurantID:  order.RestaurantID,
		Amount:        req.Amount,
		PaymentMethod: req.PaymentMethod,
		PaymentStatus: "success",
		TransactionID: req.TransactionID,
		PaidAt:        time.Now().UTC(),
	}
	created, _, _, err := s.repo.CreatePaymentTx(ctx, payment)
	if err != nil {
		return models.Payment{}, err
	}
	return created, nil
}

func (s *PaymentService) GetPaymentsByOrder(ctx context.Context, orderID string) ([]models.Payment, error) {
	orderID = strings.TrimSpace(orderID)
	if orderID == "" {
		return nil, errors.New("order_id is required")
	}
	if _, err := s.repo.GetOrderByID(ctx, orderID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("order not found")
		}
		return nil, err
	}
	return s.repo.GetPaymentsByOrder(ctx, orderID)
}

func (s *PaymentService) GetPaymentByID(ctx context.Context, id string) (models.Payment, error) {
	return s.repo.GetPaymentByID(ctx, id)
}

func (s *PaymentService) GetOrderByPaymentID(ctx context.Context, paymentID string) (models.OrderDetails, error) {
	paymentID = strings.TrimSpace(paymentID)
	if paymentID == "" {
		return models.OrderDetails{}, errors.New("payment_id is required")
	}

	order, err := s.repo.GetOrderByPaymentID(ctx, paymentID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.OrderDetails{}, errors.New("payment not found")
		}
		return models.OrderDetails{}, err
	}

	orderItems, err := s.repo.GetOrderItemsByOrderID(ctx, order.ID)
	if err != nil {
		return models.OrderDetails{}, err
	}

	return models.OrderDetails{
		Order: order,
		Items: orderItems,
	}, nil
}

func (s *PaymentService) UpdatePayment(ctx context.Context, id, status string) (models.Payment, error) {
	status = normalizePaymentValue(status)
	if status == "" {
		return models.Payment{}, errors.New("payment_status is required")
	}
	if !allowedPaymentStates[status] {
		return models.Payment{}, errors.New("invalid payment_status value")
	}
	return s.repo.UpdatePayment(ctx, id, status)
}

func (s *PaymentService) DeletePayment(ctx context.Context, id string) error {
	return s.repo.DeletePayment(ctx, id)
}

func (s *PaymentService) GetOrderPaymentSummary(ctx context.Context, orderID string) (models.PaymentSummary, error) {
	orderID = strings.TrimSpace(orderID)
	if orderID == "" {
		return models.PaymentSummary{}, errors.New("order_id is required")
	}

	order, err := s.repo.GetOrderByID(ctx, orderID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.PaymentSummary{}, errors.New("order not found")
		}
		return models.PaymentSummary{}, err
	}

	totalPaid, err := s.repo.GetTotalPaidByOrder(ctx, orderID)
	if err != nil {
		return models.PaymentSummary{}, err
	}

	remaining := order.TotalAmount - totalPaid
	if remaining < 0 {
		remaining = 0
	}

	status := "pending"
	if totalPaid <= 0 {
		status = "pending"
	} else if totalPaid < order.TotalAmount {
		status = "partial"
	} else {
		status = "paid"
	}

	return models.PaymentSummary{
		OrderID:     order.ID,
		TotalAmount: order.TotalAmount,
		TotalPaid:   totalPaid,
		Remaining:   remaining,
		Status:      status,
	}, nil
}

func normalizePaymentValue(value string) string {
	return strings.TrimSpace(strings.ToLower(value))
}

func IsPaymentValidationError(err error) bool {
	switch err.Error() {
	case "order_id is required",
		"payment_id is required",
		"amount must be greater than 0",
		"payment_method is required",
		"invalid payment_method value",
		"order does not belong to the restaurant",
		"order not found",
		"payment not found",
		"payment_status is required",
		"invalid payment_status value":
		return true
	default:
		return false
	}
}
