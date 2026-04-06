package handler

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"trythenga.com/helper"
	"trythenga.com/models"
)

const requestTimeout = 5 * time.Second

type RestaurantHandler struct {
	DB *sql.DB
}

func NewRestaurantHandler(db *sql.DB) *RestaurantHandler {
	return &RestaurantHandler{DB: db}
}

func (h *RestaurantHandler) CreateRestaurant(w http.ResponseWriter, r *http.Request) {
	var payload models.Restaurant
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		helper.JSONError(w, "invalid request payload", http.StatusBadRequest)
		return
	}

	if err := validateRestaurantPayload(payload); err != nil {
		helper.JSONError(w, err.Error(), http.StatusBadRequest)
		return
	}

	payload.ID = uuid.NewString()
	payload.Plan = normalizePlan(payload.Plan)
	payload.Status = normalizeStatus(payload.Status)

	ctx, cancel := context.WithTimeout(r.Context(), requestTimeout)
	defer cancel()

	query := `
		INSERT INTO restaurants (
			id, name, owner_name, phone, email, address, city, state, pincode, country,
			gst_number, logo_url, opening_time, closing_time, seating_capacity, plan, status
		)
		VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
			$11, $12, $13, $14, $15, $16, $17
		)
		RETURNING id, name, owner_name, phone, email, address, city, state, pincode, country,
			gst_number, logo_url, opening_time, closing_time, seating_capacity, plan, status, created_at, updated_at;
	`

	row := h.DB.QueryRowContext(
		ctx,
		query,
		payload.ID,
		payload.Name,
		payload.OwnerName,
		payload.Phone,
		payload.Email,
		payload.Address,
		payload.City,
		payload.State,
		payload.Pincode,
		payload.Country,
		payload.GSTNumber,
		payload.LogoURL,
		payload.OpeningTime,
		payload.ClosingTime,
		payload.SeatingCapacity,
		payload.Plan,
		payload.Status,
	)

	created, err := scanRestaurant(row)
	if err != nil {
		helper.JSONError(w, "failed to create restaurant", http.StatusInternalServerError)
		return
	}

	helper.JSONResponseWithStatus(w, created, http.StatusCreated)
}

func (h *RestaurantHandler) GetRestaurants(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), requestTimeout)
	defer cancel()

	baseQuery := `
		SELECT id, name, owner_name, phone, email, address, city, state, pincode, country,
			gst_number, logo_url, opening_time, closing_time, seating_capacity, plan, status, created_at, updated_at
		FROM restaurants
	`

	var filters []string
	var args []any
	argPos := 1

	if status := strings.TrimSpace(r.URL.Query().Get("status")); status != "" {
		filters = append(filters, "status = $"+itoa(argPos))
		args = append(args, strings.ToLower(status))
		argPos++
	}

	if search := strings.TrimSpace(r.URL.Query().Get("search")); search != "" {
		filters = append(filters, "LOWER(name) LIKE $"+itoa(argPos))
		args = append(args, "%"+strings.ToLower(search)+"%")
		argPos++
	}

	query := baseQuery
	if len(filters) > 0 {
		query += " WHERE " + strings.Join(filters, " AND ")
	}
	query += " ORDER BY created_at DESC"

	rows, err := h.DB.QueryContext(ctx, query, args...)
	if err != nil {
		helper.JSONError(w, "failed to fetch restaurants", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	restaurants := make([]models.Restaurant, 0)
	for rows.Next() {
		var restaurant models.Restaurant
		if err := rows.Scan(
			&restaurant.ID,
			&restaurant.Name,
			&restaurant.OwnerName,
			&restaurant.Phone,
			&restaurant.Email,
			&restaurant.Address,
			&restaurant.City,
			&restaurant.State,
			&restaurant.Pincode,
			&restaurant.Country,
			&restaurant.GSTNumber,
			&restaurant.LogoURL,
			&restaurant.OpeningTime,
			&restaurant.ClosingTime,
			&restaurant.SeatingCapacity,
			&restaurant.Plan,
			&restaurant.Status,
			&restaurant.CreatedAt,
			&restaurant.UpdatedAt,
		); err != nil {
			helper.JSONError(w, "failed to parse restaurants", http.StatusInternalServerError)
			return
		}
		restaurants = append(restaurants, restaurant)
	}

	if err := rows.Err(); err != nil {
		helper.JSONError(w, "failed to fetch restaurants", http.StatusInternalServerError)
		return
	}

	helper.JSONResponse(w, restaurants)
}

func (h *RestaurantHandler) GetRestaurantByID(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if _, err := uuid.Parse(id); err != nil {
		helper.JSONError(w, "invalid restaurant id", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), requestTimeout)
	defer cancel()

	query := `
		SELECT id, name, owner_name, phone, email, address, city, state, pincode, country,
			gst_number, logo_url, opening_time, closing_time, seating_capacity, plan, status, created_at, updated_at
		FROM restaurants
		WHERE id = $1;
	`

	row := h.DB.QueryRowContext(ctx, query, id)
	restaurant, err := scanRestaurant(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			helper.JSONError(w, "restaurant not found", http.StatusNotFound)
			return
		}
		helper.JSONError(w, "failed to fetch restaurant", http.StatusInternalServerError)
		return
	}

	helper.JSONResponse(w, restaurant)
}

func (h *RestaurantHandler) UpdateRestaurant(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if _, err := uuid.Parse(id); err != nil {
		helper.JSONError(w, "invalid restaurant id", http.StatusBadRequest)
		return
	}

	var payload models.Restaurant
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		helper.JSONError(w, "invalid request payload", http.StatusBadRequest)
		return
	}

	if err := validateRestaurantPayload(payload); err != nil {
		helper.JSONError(w, err.Error(), http.StatusBadRequest)
		return
	}

	payload.Plan = normalizePlan(payload.Plan)
	payload.Status = normalizeStatus(payload.Status)

	ctx, cancel := context.WithTimeout(r.Context(), requestTimeout)
	defer cancel()

	query := `
		UPDATE restaurants
		SET name = $2,
			owner_name = $3,
			phone = $4,
			email = $5,
			address = $6,
			city = $7,
			state = $8,
			pincode = $9,
			country = $10,
			gst_number = $11,
			logo_url = $12,
			opening_time = $13,
			closing_time = $14,
			seating_capacity = $15,
			plan = $16,
			status = $17,
			updated_at = NOW()
		WHERE id = $1
		RETURNING id, name, owner_name, phone, email, address, city, state, pincode, country,
			gst_number, logo_url, opening_time, closing_time, seating_capacity, plan, status, created_at, updated_at;
	`

	row := h.DB.QueryRowContext(
		ctx,
		query,
		id,
		payload.Name,
		payload.OwnerName,
		payload.Phone,
		payload.Email,
		payload.Address,
		payload.City,
		payload.State,
		payload.Pincode,
		payload.Country,
		payload.GSTNumber,
		payload.LogoURL,
		payload.OpeningTime,
		payload.ClosingTime,
		payload.SeatingCapacity,
		payload.Plan,
		payload.Status,
	)

	updatedRestaurant, err := scanRestaurant(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			helper.JSONError(w, "restaurant not found", http.StatusNotFound)
			return
		}
		helper.JSONError(w, "failed to update restaurant", http.StatusInternalServerError)
		return
	}

	helper.JSONResponse(w, updatedRestaurant)
}

func (h *RestaurantHandler) DisableRestaurant(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if _, err := uuid.Parse(id); err != nil {
		helper.JSONError(w, "invalid restaurant id", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), requestTimeout)
	defer cancel()

	query := `
		UPDATE restaurants
		SET status = 'disabled',
			updated_at = NOW()
		WHERE id = $1
		RETURNING id, name, owner_name, phone, email, address, city, state, pincode, country,
			gst_number, logo_url, opening_time, closing_time, seating_capacity, plan, status, created_at, updated_at;
	`

	row := h.DB.QueryRowContext(ctx, query, id)
	updatedRestaurant, err := scanRestaurant(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			helper.JSONError(w, "restaurant not found", http.StatusNotFound)
			return
		}
		helper.JSONError(w, "failed to disable restaurant", http.StatusInternalServerError)
		return
	}

	helper.JSONResponse(w, updatedRestaurant)
}

func scanRestaurant(row *sql.Row) (models.Restaurant, error) {
	var restaurant models.Restaurant
	err := row.Scan(
		&restaurant.ID,
		&restaurant.Name,
		&restaurant.OwnerName,
		&restaurant.Phone,
		&restaurant.Email,
		&restaurant.Address,
		&restaurant.City,
		&restaurant.State,
		&restaurant.Pincode,
		&restaurant.Country,
		&restaurant.GSTNumber,
		&restaurant.LogoURL,
		&restaurant.OpeningTime,
		&restaurant.ClosingTime,
		&restaurant.SeatingCapacity,
		&restaurant.Plan,
		&restaurant.Status,
		&restaurant.CreatedAt,
		&restaurant.UpdatedAt,
	)
	return restaurant, err
}

func validateRestaurantPayload(restaurant models.Restaurant) error {
	if strings.TrimSpace(restaurant.Name) == "" {
		return errors.New("name is required")
	}
	if strings.TrimSpace(restaurant.OwnerName) == "" {
		return errors.New("owner_name is required")
	}
	if strings.TrimSpace(restaurant.Phone) == "" {
		return errors.New("phone is required")
	}
	if strings.TrimSpace(restaurant.Email) == "" {
		return errors.New("email is required")
	}
	if strings.TrimSpace(restaurant.Plan) != "" && !isValidPlan(restaurant.Plan) {
		return errors.New("plan must be one of: free, premium")
	}
	if strings.TrimSpace(restaurant.Status) != "" && !isValidStatus(restaurant.Status) {
		return errors.New("status must be one of: active, disabled")
	}
	return nil
}

func isValidPlan(plan string) bool {
	switch strings.ToLower(strings.TrimSpace(plan)) {
	case "free", "premium":
		return true
	default:
		return false
	}
}

func isValidStatus(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "active", "disabled":
		return true
	default:
		return false
	}
}

func normalizePlan(plan string) string {
	clean := strings.ToLower(strings.TrimSpace(plan))
	if clean == "" {
		return "free"
	}
	return clean
}

func normalizeStatus(status string) string {
	clean := strings.ToLower(strings.TrimSpace(status))
	if clean == "" {
		return "active"
	}
	return clean
}

func itoa(v int) string {
	return strconv.Itoa(v)
}
