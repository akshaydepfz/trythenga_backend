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

type TableHandler struct {
	tableService *service.TableService
}

func NewTableHandler(db *sql.DB) *TableHandler {
	floorRepo := repository.NewFloorRepository(db)
	tableRepo := repository.NewTableRepository(db)
	tableService := service.NewTableService(tableRepo, floorRepo)
	return &TableHandler{tableService: tableService}
}

func (h *TableHandler) CreateTable(w http.ResponseWriter, r *http.Request) {
	var payload models.Table
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		helper.JSONError(w, "invalid request payload", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), requestTimeout)
	defer cancel()

	created, err := h.tableService.CreateTable(ctx, payload)
	if err != nil {
		if service.IsTableValidationError(err) {
			helper.JSONError(w, err.Error(), http.StatusBadRequest)
			return
		}
		helper.JSONError(w, "failed to create table", http.StatusInternalServerError)
		return
	}
	helper.JSONResponseWithStatus(w, created, http.StatusCreated)
}

func (h *TableHandler) GetTablesByFloor(w http.ResponseWriter, r *http.Request) {
	floorID := strings.TrimSpace(r.URL.Query().Get("floor_id"))

	ctx, cancel := context.WithTimeout(r.Context(), requestTimeout)
	defer cancel()

	tables, err := h.tableService.GetTablesByFloor(ctx, floorID)
	if err != nil {
		if service.IsTableValidationError(err) {
			helper.JSONError(w, err.Error(), http.StatusBadRequest)
			return
		}
		helper.JSONError(w, "failed to fetch tables", http.StatusInternalServerError)
		return
	}
	helper.JSONResponse(w, tables)
}

func (h *TableHandler) GetTableByID(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if _, err := uuid.Parse(id); err != nil {
		helper.JSONError(w, "invalid table id", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), requestTimeout)
	defer cancel()

	table, err := h.tableService.GetTableByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			helper.JSONError(w, "table not found", http.StatusNotFound)
			return
		}
		helper.JSONError(w, "failed to fetch table", http.StatusInternalServerError)
		return
	}
	helper.JSONResponse(w, table)
}

func (h *TableHandler) UpdateTable(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if _, err := uuid.Parse(id); err != nil {
		helper.JSONError(w, "invalid table id", http.StatusBadRequest)
		return
	}

	var payload models.Table
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		helper.JSONError(w, "invalid request payload", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), requestTimeout)
	defer cancel()

	updated, err := h.tableService.UpdateTable(ctx, id, payload)
	if err != nil {
		if service.IsTableValidationError(err) {
			helper.JSONError(w, err.Error(), http.StatusBadRequest)
			return
		}
		if errors.Is(err, sql.ErrNoRows) {
			helper.JSONError(w, "table not found", http.StatusNotFound)
			return
		}
		helper.JSONError(w, "failed to update table", http.StatusInternalServerError)
		return
	}
	helper.JSONResponse(w, updated)
}

func (h *TableHandler) DeleteTable(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if _, err := uuid.Parse(id); err != nil {
		helper.JSONError(w, "invalid table id", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), requestTimeout)
	defer cancel()

	_, err := h.tableService.SoftDeleteTable(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			helper.JSONError(w, "table not found", http.StatusNotFound)
			return
		}
		helper.JSONError(w, "failed to delete table", http.StatusInternalServerError)
		return
	}
	helper.JSONResponse(w, map[string]bool{"deleted": true})
}

func (h *TableHandler) UpdateTablePosition(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if _, err := uuid.Parse(id); err != nil {
		helper.JSONError(w, "invalid table id", http.StatusBadRequest)
		return
	}

	var payload struct {
		PosX int `json:"pos_x"`
		PosY int `json:"pos_y"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		helper.JSONError(w, "invalid request payload", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), requestTimeout)
	defer cancel()

	updated, err := h.tableService.UpdateTablePosition(ctx, id, payload.PosX, payload.PosY)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			helper.JSONError(w, "table not found", http.StatusNotFound)
			return
		}
		helper.JSONError(w, "failed to update table position", http.StatusInternalServerError)
		return
	}
	helper.JSONResponse(w, updated)
}

func (h *TableHandler) UpdateTableStatus(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if _, err := uuid.Parse(id); err != nil {
		helper.JSONError(w, "invalid table id", http.StatusBadRequest)
		return
	}

	var payload struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		helper.JSONError(w, "invalid request payload", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), requestTimeout)
	defer cancel()

	updated, err := h.tableService.UpdateTableStatus(ctx, id, payload.Status)
	if err != nil {
		if service.IsTableValidationError(err) {
			helper.JSONError(w, err.Error(), http.StatusBadRequest)
			return
		}
		if errors.Is(err, sql.ErrNoRows) {
			helper.JSONError(w, "table not found", http.StatusNotFound)
			return
		}
		helper.JSONError(w, "failed to update table status", http.StatusInternalServerError)
		return
	}
	helper.JSONResponse(w, updated)
}

func (h *TableHandler) BulkUpdateTablePositions(w http.ResponseWriter, r *http.Request) {
	var payload []models.TablePositionUpdate
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		helper.JSONError(w, "invalid request payload", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), requestTimeout)
	defer cancel()

	if err := h.tableService.BulkUpdateTablePositions(ctx, payload); err != nil {
		if service.IsTableValidationError(err) {
			helper.JSONError(w, err.Error(), http.StatusBadRequest)
			return
		}
		if errors.Is(err, sql.ErrNoRows) {
			helper.JSONError(w, "one or more tables not found", http.StatusNotFound)
			return
		}
		helper.JSONError(w, "failed to bulk update table positions", http.StatusInternalServerError)
		return
	}
	helper.JSONResponse(w, map[string]bool{"updated": true})
}
