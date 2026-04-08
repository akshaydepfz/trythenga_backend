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

type FloorHandler struct {
	floorService *service.FloorService
}

func NewFloorHandler(db *sql.DB) *FloorHandler {
	floorRepo := repository.NewFloorRepository(db)
	floorService := service.NewFloorService(floorRepo)
	return &FloorHandler{floorService: floorService}
}

func (h *FloorHandler) CreateFloor(w http.ResponseWriter, r *http.Request) {
	var payload models.Floor
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		helper.JSONError(w, "invalid request payload", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), requestTimeout)
	defer cancel()

	created, err := h.floorService.CreateFloor(ctx, payload)
	if err != nil {
		if service.IsFloorValidationError(err) {
			helper.JSONError(w, err.Error(), http.StatusBadRequest)
			return
		}
		helper.JSONError(w, "failed to create floor", http.StatusInternalServerError)
		return
	}
	helper.JSONResponseWithStatus(w, created, http.StatusCreated)
}

func (h *FloorHandler) GetFloorsByRestaurant(w http.ResponseWriter, r *http.Request) {
	restaurantID := strings.TrimSpace(r.URL.Query().Get("restaurant_id"))

	ctx, cancel := context.WithTimeout(r.Context(), requestTimeout)
	defer cancel()

	floors, err := h.floorService.GetFloorsByRestaurant(ctx, restaurantID)
	if err != nil {
		if service.IsFloorValidationError(err) {
			helper.JSONError(w, err.Error(), http.StatusBadRequest)
			return
		}
		helper.JSONError(w, "failed to fetch floors", http.StatusInternalServerError)
		return
	}
	helper.JSONResponse(w, floors)
}

func (h *FloorHandler) GetFloorByID(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if _, err := uuid.Parse(id); err != nil {
		helper.JSONError(w, "invalid floor id", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), requestTimeout)
	defer cancel()

	floor, err := h.floorService.GetFloorByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			helper.JSONError(w, "floor not found", http.StatusNotFound)
			return
		}
		helper.JSONError(w, "failed to fetch floor", http.StatusInternalServerError)
		return
	}
	helper.JSONResponse(w, floor)
}

func (h *FloorHandler) UpdateFloor(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if _, err := uuid.Parse(id); err != nil {
		helper.JSONError(w, "invalid floor id", http.StatusBadRequest)
		return
	}

	var payload models.Floor
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		helper.JSONError(w, "invalid request payload", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), requestTimeout)
	defer cancel()

	updated, err := h.floorService.UpdateFloor(ctx, id, payload)
	if err != nil {
		if service.IsFloorValidationError(err) {
			helper.JSONError(w, err.Error(), http.StatusBadRequest)
			return
		}
		if errors.Is(err, sql.ErrNoRows) {
			helper.JSONError(w, "floor not found", http.StatusNotFound)
			return
		}
		helper.JSONError(w, "failed to update floor", http.StatusInternalServerError)
		return
	}
	helper.JSONResponse(w, updated)
}

func (h *FloorHandler) DeleteFloor(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if _, err := uuid.Parse(id); err != nil {
		helper.JSONError(w, "invalid floor id", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), requestTimeout)
	defer cancel()

	if err := h.floorService.DeleteFloor(ctx, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			helper.JSONError(w, "floor not found", http.StatusNotFound)
			return
		}
		helper.JSONError(w, "failed to delete floor", http.StatusInternalServerError)
		return
	}
	helper.JSONResponse(w, map[string]bool{"deleted": true})
}
