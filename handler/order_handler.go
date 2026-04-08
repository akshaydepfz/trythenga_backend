package handler

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/google/uuid"

	"trythenga.com/helper"
	"trythenga.com/models"
	"trythenga.com/repository"
	"trythenga.com/service"
)

type OrderHandler struct {
	service *service.OrderService
}

func NewOrderHandler(db *sql.DB) *OrderHandler {
	repo := repository.NewOrderRepository(db)
	orderService := service.NewOrderService(repo)
	return &OrderHandler{service: orderService}
}

func (h *OrderHandler) CreateOrder(w http.ResponseWriter, r *http.Request) {
	var payload models.CreateOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		helper.JSONError(w, "invalid request payload", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), requestTimeout)
	defer cancel()

	created, err := h.service.CreateOrder(ctx, payload)
	if err != nil {
		if service.IsOrderValidationError(err) {
			helper.JSONError(w, err.Error(), http.StatusBadRequest)
			return
		}
		helper.JSONError(w, "failed to create order", http.StatusInternalServerError)
		return
	}
	helper.JSONResponseWithStatus(w, created, http.StatusCreated)
}

func (h *OrderHandler) GetOrdersByRestaurant(w http.ResponseWriter, r *http.Request) {
	restaurantID := strings.TrimSpace(r.URL.Query().Get("restaurant_id"))
	includeItems := false
	if includeItemsRaw := strings.TrimSpace(r.URL.Query().Get("include_items")); includeItemsRaw != "" {
		parsed, err := strconv.ParseBool(includeItemsRaw)
		if err != nil {
			helper.JSONError(w, "include_items must be true or false", http.StatusBadRequest)
			return
		}
		includeItems = parsed
	}

	ctx, cancel := context.WithTimeout(r.Context(), requestTimeout)
	defer cancel()

	orders, err := h.service.GetOrdersByRestaurant(ctx, restaurantID, includeItems)
	if err != nil {
		if service.IsOrderValidationError(err) {
			helper.JSONError(w, err.Error(), http.StatusBadRequest)
			return
		}
		helper.JSONError(w, "failed to fetch orders", http.StatusInternalServerError)
		return
	}
	helper.JSONResponse(w, orders)
}

func (h *OrderHandler) GetOrderByID(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(r.PathValue("id"))
	if _, err := uuid.Parse(id); err != nil {
		helper.JSONError(w, "invalid order id", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), requestTimeout)
	defer cancel()

	details, err := h.service.GetOrderDetails(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			helper.JSONError(w, "order not found", http.StatusNotFound)
			return
		}
		helper.JSONError(w, "failed to fetch order details", http.StatusInternalServerError)
		return
	}
	helper.JSONResponse(w, details)
}

func (h *OrderHandler) AddItemToOrder(w http.ResponseWriter, r *http.Request) {
	orderID := strings.TrimSpace(r.PathValue("id"))
	if _, err := uuid.Parse(orderID); err != nil {
		helper.JSONError(w, "invalid order id", http.StatusBadRequest)
		return
	}

	var payload models.AddOrderItemRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		helper.JSONError(w, "invalid request payload", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), requestTimeout)
	defer cancel()

	details, err := h.service.AddItemToOrder(ctx, orderID, payload)
	if err != nil {
		if service.IsOrderValidationError(err) {
			helper.JSONError(w, err.Error(), http.StatusBadRequest)
			return
		}
		if errors.Is(err, sql.ErrNoRows) {
			helper.JSONError(w, "order not found", http.StatusNotFound)
			return
		}
		helper.JSONError(w, "failed to add item to order", http.StatusInternalServerError)
		return
	}
	helper.JSONResponse(w, details)
}

func (h *OrderHandler) UpdateItemQuantity(w http.ResponseWriter, r *http.Request) {
	itemID := strings.TrimSpace(r.PathValue("item_id"))
	if _, err := uuid.Parse(itemID); err != nil {
		helper.JSONError(w, "invalid item id", http.StatusBadRequest)
		return
	}

	var payload models.UpdateOrderItemQuantityRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		helper.JSONError(w, "invalid request payload", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), requestTimeout)
	defer cancel()

	details, err := h.service.UpdateItemQuantity(ctx, itemID, payload.Quantity)
	if err != nil {
		if service.IsOrderValidationError(err) {
			helper.JSONError(w, err.Error(), http.StatusBadRequest)
			return
		}
		if errors.Is(err, sql.ErrNoRows) {
			helper.JSONError(w, "order item not found", http.StatusNotFound)
			return
		}
		helper.JSONError(w, "failed to update item quantity", http.StatusInternalServerError)
		return
	}
	helper.JSONResponse(w, details)
}

func (h *OrderHandler) RemoveItem(w http.ResponseWriter, r *http.Request) {
	itemID := strings.TrimSpace(r.PathValue("item_id"))
	if _, err := uuid.Parse(itemID); err != nil {
		helper.JSONError(w, "invalid item id", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), requestTimeout)
	defer cancel()

	details, err := h.service.RemoveItem(ctx, itemID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			helper.JSONError(w, "order item not found", http.StatusNotFound)
			return
		}
		helper.JSONError(w, "failed to remove item", http.StatusInternalServerError)
		return
	}
	helper.JSONResponse(w, details)
}

func (h *OrderHandler) UpdateOrderStatus(w http.ResponseWriter, r *http.Request) {
	orderID := strings.TrimSpace(r.PathValue("id"))
	if _, err := uuid.Parse(orderID); err != nil {
		helper.JSONError(w, "invalid order id", http.StatusBadRequest)
		return
	}

	var payload models.UpdateOrderStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		helper.JSONError(w, "invalid request payload", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), requestTimeout)
	defer cancel()

	updated, err := h.service.UpdateOrderStatus(ctx, orderID, payload.Status)
	if err != nil {
		if service.IsOrderValidationError(err) {
			helper.JSONError(w, err.Error(), http.StatusBadRequest)
			return
		}
		if errors.Is(err, sql.ErrNoRows) {
			helper.JSONError(w, "order not found", http.StatusNotFound)
			return
		}
		helper.JSONError(w, "failed to update order status", http.StatusInternalServerError)
		return
	}
	helper.JSONResponse(w, updated)
}

func (h *OrderHandler) UpdateItemStatus(w http.ResponseWriter, r *http.Request) {
	itemID := strings.TrimSpace(r.PathValue("item_id"))
	if _, err := uuid.Parse(itemID); err != nil {
		helper.JSONError(w, "invalid item id", http.StatusBadRequest)
		return
	}

	var payload models.UpdateOrderItemStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		helper.JSONError(w, "invalid request payload", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), requestTimeout)
	defer cancel()

	updated, err := h.service.UpdateOrderItemStatus(ctx, itemID, payload.Status)
	if err != nil {
		if service.IsOrderValidationError(err) {
			helper.JSONError(w, err.Error(), http.StatusBadRequest)
			return
		}
		if errors.Is(err, sql.ErrNoRows) {
			helper.JSONError(w, "order item not found", http.StatusNotFound)
			return
		}
		helper.JSONError(w, "failed to update item status", http.StatusInternalServerError)
		return
	}
	helper.JSONResponse(w, updated)
}

func (h *OrderHandler) CompletePayment(w http.ResponseWriter, r *http.Request) {
	orderID := strings.TrimSpace(r.PathValue("id"))
	if _, err := uuid.Parse(orderID); err != nil {
		helper.JSONError(w, "invalid order id", http.StatusBadRequest)
		return
	}

	var payload models.UpdatePaymentStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		helper.JSONError(w, "invalid request payload", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), requestTimeout)
	defer cancel()

	updated, err := h.service.CompletePayment(ctx, orderID, payload.PaymentStatus, payload.PaymentMethod)
	if err != nil {
		if service.IsOrderValidationError(err) {
			helper.JSONError(w, err.Error(), http.StatusBadRequest)
			return
		}
		if errors.Is(err, sql.ErrNoRows) {
			helper.JSONError(w, "order not found", http.StatusNotFound)
			return
		}
		helper.JSONError(w, "failed to update payment status", http.StatusInternalServerError)
		return
	}
	helper.JSONResponse(w, updated)
}

func (h *OrderHandler) GetCurrentOrderByTable(w http.ResponseWriter, r *http.Request) {
	tableID := strings.TrimSpace(r.PathValue("table_id"))
	if _, err := uuid.Parse(tableID); err != nil {
		helper.JSONError(w, "invalid table id", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), requestTimeout)
	defer cancel()

	details, err := h.service.GetCurrentOrderByTable(ctx, tableID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			helper.JSONError(w, "active order not found", http.StatusNotFound)
			return
		}
		helper.JSONError(w, "failed to fetch current order", http.StatusInternalServerError)
		return
	}
	helper.JSONResponse(w, details)
}
