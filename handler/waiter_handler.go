package handler

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"trythenga.com/helper"
	"trythenga.com/models"
)

type WaiterRepository struct {
	DB *sql.DB
}

func NewWaiterRepository(db *sql.DB) *WaiterRepository {
	return &WaiterRepository{DB: db}
}

func (r *WaiterRepository) CreateWaiter(ctx context.Context, waiter models.Waiter) (models.Waiter, error) {
	query := `
		INSERT INTO waiters (
			id, restaurant_id, name, phone, password_hash, role, is_active
		)
		VALUES (
			$1, $2, $3, $4, $5, $6, $7
		)
		RETURNING id, restaurant_id, name, phone, role, is_active, created_at, updated_at;
	`

	row := r.DB.QueryRowContext(
		ctx,
		query,
		waiter.ID,
		waiter.RestaurantID,
		waiter.Name,
		waiter.Phone,
		waiter.PasswordHash,
		waiter.Role,
		waiter.IsActive,
	)

	return scanWaiter(row)
}

func (r *WaiterRepository) GetWaitersByRestaurant(ctx context.Context, restaurantID string) ([]models.Waiter, error) {
	query := `
		SELECT id, restaurant_id, name, phone, role, is_active, created_at, updated_at
		FROM waiters
		WHERE restaurant_id = $1 AND is_active = TRUE
		ORDER BY created_at DESC;
	`

	rows, err := r.DB.QueryContext(ctx, query, restaurantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	waiters := make([]models.Waiter, 0)
	for rows.Next() {
		var w models.Waiter
		if err := rows.Scan(
			&w.ID,
			&w.RestaurantID,
			&w.Name,
			&w.Phone,
			&w.Role,
			&w.IsActive,
			&w.CreatedAt,
			&w.UpdatedAt,
		); err != nil {
			return nil, err
		}
		waiters = append(waiters, w)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return waiters, nil
}

func (r *WaiterRepository) GetWaiterByID(ctx context.Context, id string) (models.Waiter, error) {
	query := `
		SELECT id, restaurant_id, name, phone, role, is_active, created_at, updated_at
		FROM waiters
		WHERE id = $1;
	`

	row := r.DB.QueryRowContext(ctx, query, id)
	return scanWaiter(row)
}

func (r *WaiterRepository) UpdateWaiter(ctx context.Context, waiter models.Waiter) (models.Waiter, error) {
	query := `
		UPDATE waiters
		SET name = $2,
			phone = $3,
			role = $4,
			is_active = $5,
			password_hash = COALESCE(NULLIF($6, ''), password_hash),
			updated_at = NOW()
		WHERE id = $1
		RETURNING id, restaurant_id, name, phone, role, is_active, created_at, updated_at;
	`

	row := r.DB.QueryRowContext(
		ctx,
		query,
		waiter.ID,
		waiter.Name,
		waiter.Phone,
		waiter.Role,
		waiter.IsActive,
		waiter.PasswordHash,
	)

	return scanWaiter(row)
}

func (r *WaiterRepository) SoftDeleteWaiter(ctx context.Context, id string) (models.Waiter, error) {
	query := `
		UPDATE waiters
		SET is_active = FALSE,
			updated_at = NOW()
		WHERE id = $1
		RETURNING id, restaurant_id, name, phone, role, is_active, created_at, updated_at;
	`

	row := r.DB.QueryRowContext(ctx, query, id)
	return scanWaiter(row)
}

func (r *WaiterRepository) GetWaiterByName(ctx context.Context, name string) (models.Waiter, error) {
	query := `
		SELECT id, restaurant_id, name, phone, role, is_active, created_at, updated_at, password_hash
		FROM waiters
		WHERE LOWER(name) = LOWER($1);
	`

	var waiter models.Waiter
	var hash string

	err := r.DB.QueryRowContext(ctx, query, name).Scan(
		&waiter.ID,
		&waiter.RestaurantID,
		&waiter.Name,
		&waiter.Phone,
		&waiter.Role,
		&waiter.IsActive,
		&waiter.CreatedAt,
		&waiter.UpdatedAt,
		&hash,
	)
	if err != nil {
		return models.Waiter{}, err
	}

	waiter.PasswordHash = hash
	return waiter, nil
}

type WaiterService struct {
	repo *WaiterRepository
}

func NewWaiterService(repo *WaiterRepository) *WaiterService {
	return &WaiterService{repo: repo}
}

func (s *WaiterService) CreateWaiter(ctx context.Context, waiter models.Waiter) (models.Waiter, error) {
	waiter.Name = strings.TrimSpace(waiter.Name)
	if waiter.Name == "" {
		return models.Waiter{}, errors.New("name is required")
	}
	waiter.RestaurantID = strings.TrimSpace(waiter.RestaurantID)
	if waiter.RestaurantID == "" {
		return models.Waiter{}, errors.New("restaurant_id is required")
	}

	if strings.TrimSpace(waiter.Role) == "" {
		waiter.Role = "waiter"
	}
	if !waiter.IsActive {
		waiter.IsActive = true
	}

	if strings.TrimSpace(waiter.Password) != "" {
		hash, err := s.hashPassword(waiter.Password)
		if err != nil {
			return models.Waiter{}, err
		}
		waiter.PasswordHash = hash
	}
	waiter.Password = ""

	waiter.ID = uuid.NewString()

	created, err := s.repo.CreateWaiter(ctx, waiter)
	if err != nil {
		return models.Waiter{}, err
	}
	return created, nil
}

func (s *WaiterService) GetWaitersByRestaurant(ctx context.Context, restaurantID string) ([]models.Waiter, error) {
	restaurantID = strings.TrimSpace(restaurantID)
	if restaurantID == "" {
		return nil, errors.New("restaurant_id is required")
	}
	return s.repo.GetWaitersByRestaurant(ctx, restaurantID)
}

func (s *WaiterService) GetWaiterByID(ctx context.Context, id string) (models.Waiter, error) {
	return s.repo.GetWaiterByID(ctx, id)
}

func (s *WaiterService) UpdateWaiter(ctx context.Context, id string, payload models.Waiter) (models.Waiter, error) {
	payload.Name = strings.TrimSpace(payload.Name)
	if payload.Name == "" {
		return models.Waiter{}, errors.New("name is required")
	}

	if strings.TrimSpace(payload.Role) == "" {
		payload.Role = "waiter"
	}

	if strings.TrimSpace(payload.Password) != "" {
		hash, err := s.hashPassword(payload.Password)
		if err != nil {
			return models.Waiter{}, err
		}
		payload.PasswordHash = hash
	}
	payload.Password = ""
	payload.ID = id

	return s.repo.UpdateWaiter(ctx, payload)
}

func (s *WaiterService) SoftDeleteWaiter(ctx context.Context, id string) (models.Waiter, error) {
	return s.repo.SoftDeleteWaiter(ctx, id)
}

func (s *WaiterService) Login(ctx context.Context, name, password string) (models.Waiter, bool, error) {
	name = strings.TrimSpace(name)
	password = strings.TrimSpace(password)
	if name == "" || password == "" {
		return models.Waiter{}, false, nil
	}

	waiter, err := s.repo.GetWaiterByName(ctx, name)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.Waiter{}, false, nil
		}
		return models.Waiter{}, false, err
	}

	if !waiter.IsActive {
		return models.Waiter{}, false, nil
	}

	if err := bcrypt.CompareHashAndPassword([]byte(waiter.PasswordHash), []byte(password)); err != nil {
		return models.Waiter{}, false, nil
	}

	waiter.Password = ""
	waiter.PasswordHash = ""

	return waiter, true, nil
}

func (s *WaiterService) hashPassword(password string) (string, error) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(strings.TrimSpace(password)), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashed), nil
}

type WaiterHandler struct {
	service *WaiterService
}

func NewWaiterHandler(db *sql.DB) *WaiterHandler {
	repo := NewWaiterRepository(db)
	service := NewWaiterService(repo)
	return &WaiterHandler{service: service}
}

func (h *WaiterHandler) CreateWaiter(w http.ResponseWriter, r *http.Request) {
	var payload models.Waiter
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		helper.JSONError(w, "invalid request payload", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), requestTimeout)
	defer cancel()

	created, err := h.service.CreateWaiter(ctx, payload)
	if err != nil {
		if err.Error() == "name is required" || err.Error() == "restaurant_id is required" {
			helper.JSONError(w, err.Error(), http.StatusBadRequest)
			return
		}
		log.Printf("create waiter failed: %v", err)
		helper.JSONError(w, "failed to create waiter", http.StatusInternalServerError)
		return
	}

	helper.JSONResponseWithStatus(w, created, http.StatusCreated)
}

func (h *WaiterHandler) GetWaiters(w http.ResponseWriter, r *http.Request) {
	restaurantID := strings.TrimSpace(r.URL.Query().Get("restaurant_id"))

	ctx, cancel := context.WithTimeout(r.Context(), requestTimeout)
	defer cancel()

	waiters, err := h.service.GetWaitersByRestaurant(ctx, restaurantID)
	if err != nil {
		if err.Error() == "restaurant_id is required" {
			helper.JSONError(w, err.Error(), http.StatusBadRequest)
			return
		}
		helper.JSONError(w, "failed to fetch waiters", http.StatusInternalServerError)
		return
	}

	helper.JSONResponse(w, waiters)
}

func (h *WaiterHandler) GetWaitersByRestaurantID(w http.ResponseWriter, r *http.Request) {
	restaurantID := strings.TrimSpace(r.PathValue("restaurant_id"))
	if _, err := uuid.Parse(restaurantID); err != nil {
		helper.JSONError(w, "invalid restaurant id", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), requestTimeout)
	defer cancel()

	waiters, err := h.service.GetWaitersByRestaurant(ctx, restaurantID)
	if err != nil {
		if err.Error() == "restaurant_id is required" {
			helper.JSONError(w, err.Error(), http.StatusBadRequest)
			return
		}
		helper.JSONError(w, "failed to fetch waiters", http.StatusInternalServerError)
		return
	}

	helper.JSONResponse(w, waiters)
}

func (h *WaiterHandler) GetWaiterByID(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if _, err := uuid.Parse(id); err != nil {
		helper.JSONError(w, "invalid waiter id", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), requestTimeout)
	defer cancel()

	waiter, err := h.service.GetWaiterByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			helper.JSONError(w, "waiter not found", http.StatusNotFound)
			return
		}
		helper.JSONError(w, "failed to fetch waiter", http.StatusInternalServerError)
		return
	}

	helper.JSONResponse(w, waiter)
}

func (h *WaiterHandler) UpdateWaiter(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if _, err := uuid.Parse(id); err != nil {
		helper.JSONError(w, "invalid waiter id", http.StatusBadRequest)
		return
	}

	var payload models.Waiter
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		helper.JSONError(w, "invalid request payload", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), requestTimeout)
	defer cancel()

	updated, err := h.service.UpdateWaiter(ctx, id, payload)
	if err != nil {
		if err.Error() == "name is required" {
			helper.JSONError(w, err.Error(), http.StatusBadRequest)
			return
		}
		if errors.Is(err, sql.ErrNoRows) {
			helper.JSONError(w, "waiter not found", http.StatusNotFound)
			return
		}
		helper.JSONError(w, "failed to update waiter", http.StatusInternalServerError)
		return
	}

	helper.JSONResponse(w, updated)
}

func (h *WaiterHandler) DeleteWaiter(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if _, err := uuid.Parse(id); err != nil {
		helper.JSONError(w, "invalid waiter id", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), requestTimeout)
	defer cancel()

	_, err := h.service.SoftDeleteWaiter(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			helper.JSONError(w, "waiter not found", http.StatusNotFound)
			return
		}
		helper.JSONError(w, "failed to delete waiter", http.StatusInternalServerError)
		return
	}

	helper.JSONResponse(w, map[string]bool{"deleted": true})
}

func (h *WaiterHandler) LoginWaiter(w http.ResponseWriter, r *http.Request) {
	type loginRequest struct {
		Name     string `json:"name"`
		Password string `json:"password"`
	}

	type loginResponse struct {
		Login  bool           `json:"login"`
		Waiter *models.Waiter `json:"waiter,omitempty"`
	}

	var payload loginRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		helper.JSONError(w, "invalid request payload", http.StatusBadRequest)
		return
	}

	name := strings.TrimSpace(payload.Name)
	password := strings.TrimSpace(payload.Password)
	if name == "" || password == "" {
		helper.JSONResponse(w, loginResponse{Login: false})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), requestTimeout)
	defer cancel()

	waiter, ok, err := h.service.Login(ctx, name, password)
	if err != nil {
		helper.JSONError(w, "failed to verify credentials", http.StatusInternalServerError)
		return
	}

	if !ok {
		helper.JSONResponse(w, loginResponse{Login: false})
		return
	}

	helper.JSONResponse(w, loginResponse{
		Login:  true,
		Waiter: &waiter,
	})
}

func scanWaiter(row *sql.Row) (models.Waiter, error) {
	var w models.Waiter
	err := row.Scan(
		&w.ID,
		&w.RestaurantID,
		&w.Name,
		&w.Phone,
		&w.Role,
		&w.IsActive,
		&w.CreatedAt,
		&w.UpdatedAt,
	)
	return w, err
}

