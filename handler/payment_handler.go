package handler

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"trythenga.com/helper"
	"trythenga.com/models"
	"trythenga.com/repository"
	"trythenga.com/service"
)

type PaymentHandler struct {
	service *service.PaymentService
}

func NewPaymentHandler(db *sql.DB) *PaymentHandler {
	repo := repository.NewPaymentRepository(db)
	paymentService := service.NewPaymentService(repo)
	return &PaymentHandler{service: paymentService}
}

func (h *PaymentHandler) CreatePayment(w http.ResponseWriter, r *http.Request) {
	var payload models.CreatePaymentRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		helper.JSONError(w, "invalid request payload", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), requestTimeout)
	defer cancel()

	created, err := h.service.CreatePayment(ctx, payload)
	if err != nil {
		if service.IsPaymentValidationError(err) {
			helper.JSONError(w, err.Error(), http.StatusBadRequest)
			return
		}
		helper.JSONError(w, "failed to create payment", http.StatusInternalServerError)
		return
	}
	helper.JSONResponseWithStatus(w, created, http.StatusCreated)
}

func (h *PaymentHandler) GetPaymentsByOrder(w http.ResponseWriter, r *http.Request) {
	orderID := strings.TrimSpace(r.URL.Query().Get("order_id"))
	ctx, cancel := context.WithTimeout(r.Context(), requestTimeout)
	defer cancel()

	payments, err := h.service.GetPaymentsByOrder(ctx, orderID)
	if err != nil {
		if service.IsPaymentValidationError(err) {
			if err.Error() == "order not found" {
				helper.JSONError(w, err.Error(), http.StatusNotFound)
				return
			}
			helper.JSONError(w, err.Error(), http.StatusBadRequest)
			return
		}
		helper.JSONError(w, "failed to fetch payments", http.StatusInternalServerError)
		return
	}
	helper.JSONResponse(w, payments)
}

func (h *PaymentHandler) GetPaymentByID(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(r.PathValue("id"))
	if _, err := uuid.Parse(id); err != nil {
		helper.JSONError(w, "invalid payment id", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), requestTimeout)
	defer cancel()

	payment, err := h.service.GetPaymentByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			helper.JSONError(w, "payment not found", http.StatusNotFound)
			return
		}
		helper.JSONError(w, "failed to fetch payment", http.StatusInternalServerError)
		return
	}
	helper.JSONResponse(w, payment)
}

func (h *PaymentHandler) GetOrderByPaymentID(w http.ResponseWriter, r *http.Request) {
	paymentID := strings.TrimSpace(r.PathValue("id"))
	if _, err := uuid.Parse(paymentID); err != nil {
		helper.JSONError(w, "invalid payment id", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), requestTimeout)
	defer cancel()

	orderDetails, err := h.service.GetOrderByPaymentID(ctx, paymentID)
	if err != nil {
		if service.IsPaymentValidationError(err) {
			if err.Error() == "payment not found" {
				helper.JSONError(w, err.Error(), http.StatusNotFound)
				return
			}
			helper.JSONError(w, err.Error(), http.StatusBadRequest)
			return
		}
		helper.JSONError(w, "failed to fetch order by payment id", http.StatusInternalServerError)
		return
	}

	helper.JSONResponse(w, orderDetails)
}

func (h *PaymentHandler) UpdatePayment(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(r.PathValue("id"))
	if _, err := uuid.Parse(id); err != nil {
		helper.JSONError(w, "invalid payment id", http.StatusBadRequest)
		return
	}

	var payload models.UpdatePaymentRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		helper.JSONError(w, "invalid request payload", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), requestTimeout)
	defer cancel()

	updated, err := h.service.UpdatePayment(ctx, id, payload.PaymentStatus)
	if err != nil {
		if service.IsPaymentValidationError(err) {
			helper.JSONError(w, err.Error(), http.StatusBadRequest)
			return
		}
		if errors.Is(err, sql.ErrNoRows) {
			helper.JSONError(w, "payment not found", http.StatusNotFound)
			return
		}
		helper.JSONError(w, "failed to update payment", http.StatusInternalServerError)
		return
	}
	helper.JSONResponse(w, updated)
}

func (h *PaymentHandler) DeletePayment(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(r.PathValue("id"))
	if _, err := uuid.Parse(id); err != nil {
		helper.JSONError(w, "invalid payment id", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), requestTimeout)
	defer cancel()

	if err := h.service.DeletePayment(ctx, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			helper.JSONError(w, "payment not found", http.StatusNotFound)
			return
		}
		helper.JSONError(w, "failed to delete payment", http.StatusInternalServerError)
		return
	}
	helper.JSONResponse(w, map[string]bool{"deleted": true})
}

func (h *PaymentHandler) GetOrderPaymentSummary(w http.ResponseWriter, r *http.Request) {
	orderID := strings.TrimSpace(r.PathValue("id"))
	if _, err := uuid.Parse(orderID); err != nil {
		helper.JSONError(w, "invalid order id", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), requestTimeout)
	defer cancel()

	summary, err := h.service.GetOrderPaymentSummary(ctx, orderID)
	if err != nil {
		if service.IsPaymentValidationError(err) {
			if err.Error() == "order not found" {
				helper.JSONError(w, err.Error(), http.StatusNotFound)
				return
			}
			helper.JSONError(w, err.Error(), http.StatusBadRequest)
			return
		}
		helper.JSONError(w, "failed to fetch payment summary", http.StatusInternalServerError)
		return
	}
	helper.JSONResponse(w, summary)
}
