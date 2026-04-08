package handler

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/google/uuid"

	"trythenga.com/helper"
	"trythenga.com/models"
)

const maxMultipartSize = 6 * 1024 * 1024

type CategoryRepository struct {
	DB *sql.DB
}

func NewCategoryRepository(db *sql.DB) *CategoryRepository {
	return &CategoryRepository{DB: db}
}

func (r *CategoryRepository) CreateCategory(ctx context.Context, category models.Category) (models.Category, error) {
	query := `
		INSERT INTO categories (id, restaurant_id, name, description, is_active)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, restaurant_id, name, description, is_active, created_at, updated_at;
	`
	row := r.DB.QueryRowContext(ctx, query, category.ID, category.RestaurantID, category.Name, category.Description, category.IsActive)
	return scanCategory(row)
}

func (r *CategoryRepository) GetCategoriesByRestaurant(ctx context.Context, restaurantID string) ([]models.Category, error) {
	query := `
		SELECT id, restaurant_id, name, description, is_active, created_at, updated_at
		FROM categories
		WHERE restaurant_id = $1
		ORDER BY created_at DESC;
	`
	rows, err := r.DB.QueryContext(ctx, query, restaurantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	categories := make([]models.Category, 0)
	for rows.Next() {
		var category models.Category
		if err := rows.Scan(
			&category.ID,
			&category.RestaurantID,
			&category.Name,
			&category.Description,
			&category.IsActive,
			&category.CreatedAt,
			&category.UpdatedAt,
		); err != nil {
			return nil, err
		}
		categories = append(categories, category)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return categories, nil
}

func (r *CategoryRepository) UpdateCategory(ctx context.Context, category models.Category) (models.Category, error) {
	query := `
		UPDATE categories
		SET name = $2, description = $3, is_active = $4, updated_at = NOW()
		WHERE id = $1
		RETURNING id, restaurant_id, name, description, is_active, created_at, updated_at;
	`
	row := r.DB.QueryRowContext(ctx, query, category.ID, category.Name, category.Description, category.IsActive)
	return scanCategory(row)
}

func (r *CategoryRepository) SoftDeleteCategory(ctx context.Context, id string) (models.Category, error) {
	query := `
		UPDATE categories
		SET is_active = FALSE, updated_at = NOW()
		WHERE id = $1
		RETURNING id, restaurant_id, name, description, is_active, created_at, updated_at;
	`
	row := r.DB.QueryRowContext(ctx, query, id)
	return scanCategory(row)
}

func (r *CategoryRepository) GetCategoryByID(ctx context.Context, id string) (models.Category, error) {
	query := `
		SELECT id, restaurant_id, name, description, is_active, created_at, updated_at
		FROM categories
		WHERE id = $1;
	`
	row := r.DB.QueryRowContext(ctx, query, id)
	return scanCategory(row)
}

type MenuItemRepository struct {
	DB *sql.DB
}

func NewMenuItemRepository(db *sql.DB) *MenuItemRepository {
	return &MenuItemRepository{DB: db}
}

func (r *MenuItemRepository) CreateMenuItem(ctx context.Context, item models.MenuItem) (models.MenuItem, error) {
	query := `
		INSERT INTO menu_items (id, restaurant_id, category_id, name, description, price, is_available, image_url)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NULLIF($8, ''))
		RETURNING id, restaurant_id, category_id, name, description, price, is_available, image_url, created_at, updated_at;
	`
	row := r.DB.QueryRowContext(
		ctx,
		query,
		item.ID,
		item.RestaurantID,
		item.CategoryID,
		item.Name,
		item.Description,
		item.Price,
		item.IsAvailable,
		item.ImageURL,
	)
	return scanMenuItem(row)
}

func (r *MenuItemRepository) GetMenuItemsByRestaurant(ctx context.Context, restaurantID string) ([]models.MenuItem, error) {
	query := `
		SELECT id, restaurant_id, category_id, name, description, price, is_available, image_url, created_at, updated_at
		FROM menu_items
		WHERE restaurant_id = $1
		ORDER BY created_at DESC;
	`
	rows, err := r.DB.QueryContext(ctx, query, restaurantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]models.MenuItem, 0)
	for rows.Next() {
		item, err := scanMenuItemFromRows(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func (r *MenuItemRepository) GetMenuItemsByCategory(ctx context.Context, categoryID string) ([]models.MenuItem, error) {
	query := `
		SELECT id, restaurant_id, category_id, name, description, price, is_available, image_url, created_at, updated_at
		FROM menu_items
		WHERE category_id = $1
		ORDER BY created_at DESC;
	`
	rows, err := r.DB.QueryContext(ctx, query, categoryID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]models.MenuItem, 0)
	for rows.Next() {
		item, err := scanMenuItemFromRows(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func (r *MenuItemRepository) UpdateMenuItem(ctx context.Context, item models.MenuItem, hasImageUpdate bool) (models.MenuItem, error) {
	if hasImageUpdate {
		query := `
			UPDATE menu_items
			SET category_id = $2, name = $3, description = $4, price = $5, is_available = $6, image_url = NULLIF($7, ''), updated_at = NOW()
			WHERE id = $1
			RETURNING id, restaurant_id, category_id, name, description, price, is_available, image_url, created_at, updated_at;
		`
		row := r.DB.QueryRowContext(
			ctx,
			query,
			item.ID,
			item.CategoryID,
			item.Name,
			item.Description,
			item.Price,
			item.IsAvailable,
			item.ImageURL,
		)
		return scanMenuItem(row)
	}

	query := `
		UPDATE menu_items
		SET category_id = $2, name = $3, description = $4, price = $5, is_available = $6, updated_at = NOW()
		WHERE id = $1
		RETURNING id, restaurant_id, category_id, name, description, price, is_available, image_url, created_at, updated_at;
	`
	row := r.DB.QueryRowContext(
		ctx,
		query,
		item.ID,
		item.CategoryID,
		item.Name,
		item.Description,
		item.Price,
		item.IsAvailable,
	)
	return scanMenuItem(row)
}

func (r *MenuItemRepository) SoftDeleteMenuItem(ctx context.Context, id string) (models.MenuItem, error) {
	query := `
		UPDATE menu_items
		SET is_available = FALSE, updated_at = NOW()
		WHERE id = $1
		RETURNING id, restaurant_id, category_id, name, description, price, is_available, image_url, created_at, updated_at;
	`
	row := r.DB.QueryRowContext(ctx, query, id)
	return scanMenuItem(row)
}

type CategoryService struct {
	repo *CategoryRepository
}

func NewCategoryService(repo *CategoryRepository) *CategoryService {
	return &CategoryService{repo: repo}
}

func (s *CategoryService) CreateCategory(ctx context.Context, category models.Category) (models.Category, error) {
	category.RestaurantID = strings.TrimSpace(category.RestaurantID)
	if category.RestaurantID == "" {
		return models.Category{}, errors.New("restaurant_id is required")
	}
	category.Name = strings.TrimSpace(category.Name)
	if category.Name == "" {
		return models.Category{}, errors.New("name is required")
	}
	category.Description = strings.TrimSpace(category.Description)
	category.ID = uuid.NewString()
	if !category.IsActive {
		category.IsActive = true
	}
	return s.repo.CreateCategory(ctx, category)
}

func (s *CategoryService) GetCategoriesByRestaurant(ctx context.Context, restaurantID string) ([]models.Category, error) {
	restaurantID = strings.TrimSpace(restaurantID)
	if restaurantID == "" {
		return nil, errors.New("restaurant_id is required")
	}
	return s.repo.GetCategoriesByRestaurant(ctx, restaurantID)
}

func (s *CategoryService) UpdateCategory(ctx context.Context, id string, payload models.Category) (models.Category, error) {
	payload.Name = strings.TrimSpace(payload.Name)
	if payload.Name == "" {
		return models.Category{}, errors.New("name is required")
	}
	payload.Description = strings.TrimSpace(payload.Description)
	payload.ID = id
	return s.repo.UpdateCategory(ctx, payload)
}

func (s *CategoryService) SoftDeleteCategory(ctx context.Context, id string) (models.Category, error) {
	return s.repo.SoftDeleteCategory(ctx, id)
}

type MenuItemService struct {
	repo         *MenuItemRepository
	categoryRepo *CategoryRepository
}

func NewMenuItemService(repo *MenuItemRepository, categoryRepo *CategoryRepository) *MenuItemService {
	return &MenuItemService{repo: repo, categoryRepo: categoryRepo}
}

func (s *MenuItemService) CreateMenuItem(ctx context.Context, item models.MenuItem) (models.MenuItem, error) {
	if err := s.validateMenuItem(ctx, item.RestaurantID, item.CategoryID, item.Name, item.Price); err != nil {
		return models.MenuItem{}, err
	}
	item.ID = uuid.NewString()
	item.Name = strings.TrimSpace(item.Name)
	item.Description = strings.TrimSpace(item.Description)
	if !item.IsAvailable {
		item.IsAvailable = true
	}
	return s.repo.CreateMenuItem(ctx, item)
}

func (s *MenuItemService) GetMenuItemsByRestaurant(ctx context.Context, restaurantID string) ([]models.MenuItem, error) {
	restaurantID = strings.TrimSpace(restaurantID)
	if restaurantID == "" {
		return nil, errors.New("restaurant_id is required")
	}
	return s.repo.GetMenuItemsByRestaurant(ctx, restaurantID)
}

func (s *MenuItemService) GetMenuItemsByCategory(ctx context.Context, categoryID string) ([]models.MenuItem, error) {
	categoryID = strings.TrimSpace(categoryID)
	if categoryID == "" {
		return nil, errors.New("category_id is required")
	}
	return s.repo.GetMenuItemsByCategory(ctx, categoryID)
}

func (s *MenuItemService) UpdateMenuItem(ctx context.Context, id string, item models.MenuItem, hasImageUpdate bool) (models.MenuItem, error) {
	if err := s.validateMenuItem(ctx, item.RestaurantID, item.CategoryID, item.Name, item.Price); err != nil {
		return models.MenuItem{}, err
	}
	item.ID = id
	item.Name = strings.TrimSpace(item.Name)
	item.Description = strings.TrimSpace(item.Description)
	return s.repo.UpdateMenuItem(ctx, item, hasImageUpdate)
}

func (s *MenuItemService) SoftDeleteMenuItem(ctx context.Context, id string) (models.MenuItem, error) {
	return s.repo.SoftDeleteMenuItem(ctx, id)
}

func (s *MenuItemService) validateMenuItem(ctx context.Context, restaurantID, categoryID, name string, price float64) error {
	restaurantID = strings.TrimSpace(restaurantID)
	if restaurantID == "" {
		return errors.New("restaurant_id is required")
	}
	categoryID = strings.TrimSpace(categoryID)
	if categoryID == "" {
		return errors.New("category_id is required")
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return errors.New("name is required")
	}
	if price <= 0 {
		return errors.New("price must be greater than 0")
	}

	category, err := s.categoryRepo.GetCategoryByID(ctx, categoryID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return errors.New("category not found")
		}
		return err
	}
	if category.RestaurantID != restaurantID {
		return errors.New("category does not belong to the restaurant")
	}
	if !category.IsActive {
		return errors.New("category is not active")
	}
	return nil
}

type MenuHandler struct {
	categoryService *CategoryService
	menuItemService *MenuItemService
}

func NewMenuHandler(db *sql.DB) *MenuHandler {
	categoryRepo := NewCategoryRepository(db)
	menuItemRepo := NewMenuItemRepository(db)
	categoryService := NewCategoryService(categoryRepo)
	menuItemService := NewMenuItemService(menuItemRepo, categoryRepo)
	return &MenuHandler{
		categoryService: categoryService,
		menuItemService: menuItemService,
	}
}

func (h *MenuHandler) CreateCategory(w http.ResponseWriter, r *http.Request) {
	var payload models.Category
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		helper.JSONError(w, "invalid request payload", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), requestTimeout)
	defer cancel()

	created, err := h.categoryService.CreateCategory(ctx, payload)
	if err != nil {
		if isMenuValidationError(err) {
			helper.JSONError(w, err.Error(), http.StatusBadRequest)
			return
		}
		helper.JSONError(w, "failed to create category", http.StatusInternalServerError)
		return
	}
	helper.JSONResponseWithStatus(w, created, http.StatusCreated)
}

func (h *MenuHandler) GetCategoriesByRestaurant(w http.ResponseWriter, r *http.Request) {
	restaurantID := strings.TrimSpace(r.URL.Query().Get("restaurant_id"))

	ctx, cancel := context.WithTimeout(r.Context(), requestTimeout)
	defer cancel()

	categories, err := h.categoryService.GetCategoriesByRestaurant(ctx, restaurantID)
	if err != nil {
		if isMenuValidationError(err) {
			helper.JSONError(w, err.Error(), http.StatusBadRequest)
			return
		}
		helper.JSONError(w, "failed to fetch categories", http.StatusInternalServerError)
		return
	}
	helper.JSONResponse(w, categories)
}

func (h *MenuHandler) UpdateCategory(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if _, err := uuid.Parse(id); err != nil {
		helper.JSONError(w, "invalid category id", http.StatusBadRequest)
		return
	}

	var payload models.Category
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		helper.JSONError(w, "invalid request payload", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), requestTimeout)
	defer cancel()

	updated, err := h.categoryService.UpdateCategory(ctx, id, payload)
	if err != nil {
		if isMenuValidationError(err) {
			helper.JSONError(w, err.Error(), http.StatusBadRequest)
			return
		}
		if errors.Is(err, sql.ErrNoRows) {
			helper.JSONError(w, "category not found", http.StatusNotFound)
			return
		}
		helper.JSONError(w, "failed to update category", http.StatusInternalServerError)
		return
	}
	helper.JSONResponse(w, updated)
}

func (h *MenuHandler) DeleteCategory(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if _, err := uuid.Parse(id); err != nil {
		helper.JSONError(w, "invalid category id", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), requestTimeout)
	defer cancel()

	_, err := h.categoryService.SoftDeleteCategory(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			helper.JSONError(w, "category not found", http.StatusNotFound)
			return
		}
		helper.JSONError(w, "failed to delete category", http.StatusInternalServerError)
		return
	}
	helper.JSONResponse(w, map[string]bool{"deleted": true})
}

func (h *MenuHandler) CreateMenuItem(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(maxMultipartSize); err != nil {
		helper.JSONError(w, "invalid multipart form data", http.StatusBadRequest)
		return
	}

	price, err := strconv.ParseFloat(strings.TrimSpace(r.FormValue("price")), 64)
	if err != nil {
		helper.JSONError(w, "price must be a valid number", http.StatusBadRequest)
		return
	}

	isAvailable := true
	if raw := strings.TrimSpace(r.FormValue("is_available")); raw != "" {
		parsed, parseErr := strconv.ParseBool(raw)
		if parseErr != nil {
			helper.JSONError(w, "is_available must be a valid boolean", http.StatusBadRequest)
			return
		}
		isAvailable = parsed
	}

	imageURL := ""
	file, header, fileErr := r.FormFile("file")
	if fileErr == nil {
		defer file.Close()
		ctxUpload, cancelUpload := context.WithTimeout(r.Context(), requestTimeout)
		defer cancelUpload()
		imageURL, err = helper.UploadImageToS3(ctxUpload, file, header, "menu-images")
		if err != nil {
			helper.JSONError(w, err.Error(), http.StatusBadRequest)
			return
		}
	} else if !errors.Is(fileErr, http.ErrMissingFile) {
		helper.JSONError(w, "invalid image file", http.StatusBadRequest)
		return
	}

	payload := models.MenuItem{
		RestaurantID: strings.TrimSpace(r.FormValue("restaurant_id")),
		CategoryID:   strings.TrimSpace(r.FormValue("category_id")),
		Name:         strings.TrimSpace(r.FormValue("name")),
		Description:  strings.TrimSpace(r.FormValue("description")),
		Price:        price,
		IsAvailable:  isAvailable,
		ImageURL:     imageURL,
	}

	ctx, cancel := context.WithTimeout(r.Context(), requestTimeout)
	defer cancel()

	created, err := h.menuItemService.CreateMenuItem(ctx, payload)
	if err != nil {
		if isMenuValidationError(err) {
			helper.JSONError(w, err.Error(), http.StatusBadRequest)
			return
		}
		log.Printf("create menu item failed: %v", err)
		helper.JSONError(w, "failed to create menu item", http.StatusInternalServerError)
		return
	}
	helper.JSONResponseWithStatus(w, created, http.StatusCreated)
}

func (h *MenuHandler) GetMenuItemsByRestaurant(w http.ResponseWriter, r *http.Request) {
	restaurantID := strings.TrimSpace(r.URL.Query().Get("restaurant_id"))

	ctx, cancel := context.WithTimeout(r.Context(), requestTimeout)
	defer cancel()

	items, err := h.menuItemService.GetMenuItemsByRestaurant(ctx, restaurantID)
	if err != nil {
		if isMenuValidationError(err) {
			helper.JSONError(w, err.Error(), http.StatusBadRequest)
			return
		}
		helper.JSONError(w, "failed to fetch menu items", http.StatusInternalServerError)
		return
	}
	helper.JSONResponse(w, items)
}

func (h *MenuHandler) GetMenuItemsByCategory(w http.ResponseWriter, r *http.Request) {
	categoryID := strings.TrimSpace(r.PathValue("category_id"))
	if _, err := uuid.Parse(categoryID); err != nil {
		helper.JSONError(w, "invalid category id", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), requestTimeout)
	defer cancel()

	items, err := h.menuItemService.GetMenuItemsByCategory(ctx, categoryID)
	if err != nil {
		if isMenuValidationError(err) {
			helper.JSONError(w, err.Error(), http.StatusBadRequest)
			return
		}
		helper.JSONError(w, "failed to fetch menu items", http.StatusInternalServerError)
		return
	}
	helper.JSONResponse(w, items)
}

func (h *MenuHandler) UpdateMenuItem(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if _, err := uuid.Parse(id); err != nil {
		helper.JSONError(w, "invalid menu item id", http.StatusBadRequest)
		return
	}

	if err := r.ParseMultipartForm(maxMultipartSize); err != nil {
		helper.JSONError(w, "invalid multipart form data", http.StatusBadRequest)
		return
	}

	price, err := strconv.ParseFloat(strings.TrimSpace(r.FormValue("price")), 64)
	if err != nil {
		helper.JSONError(w, "price must be a valid number", http.StatusBadRequest)
		return
	}

	isAvailable := true
	if raw := strings.TrimSpace(r.FormValue("is_available")); raw != "" {
		parsed, parseErr := strconv.ParseBool(raw)
		if parseErr != nil {
			helper.JSONError(w, "is_available must be a valid boolean", http.StatusBadRequest)
			return
		}
		isAvailable = parsed
	}

	imageURL := ""
	hasImageUpdate := false
	file, header, fileErr := r.FormFile("file")
	if fileErr == nil {
		defer file.Close()
		ctxUpload, cancelUpload := context.WithTimeout(r.Context(), requestTimeout)
		defer cancelUpload()
		imageURL, err = helper.UploadImageToS3(ctxUpload, file, header, "menu-images")
		if err != nil {
			helper.JSONError(w, err.Error(), http.StatusBadRequest)
			return
		}
		hasImageUpdate = true
	} else if !errors.Is(fileErr, http.ErrMissingFile) {
		helper.JSONError(w, "invalid image file", http.StatusBadRequest)
		return
	}

	payload := models.MenuItem{
		RestaurantID: strings.TrimSpace(r.FormValue("restaurant_id")),
		CategoryID:   strings.TrimSpace(r.FormValue("category_id")),
		Name:         strings.TrimSpace(r.FormValue("name")),
		Description:  strings.TrimSpace(r.FormValue("description")),
		Price:        price,
		IsAvailable:  isAvailable,
		ImageURL:     imageURL,
	}

	ctx, cancel := context.WithTimeout(r.Context(), requestTimeout)
	defer cancel()

	updated, err := h.menuItemService.UpdateMenuItem(ctx, id, payload, hasImageUpdate)
	if err != nil {
		if isMenuValidationError(err) {
			helper.JSONError(w, err.Error(), http.StatusBadRequest)
			return
		}
		if errors.Is(err, sql.ErrNoRows) {
			helper.JSONError(w, "menu item not found", http.StatusNotFound)
			return
		}
		helper.JSONError(w, "failed to update menu item", http.StatusInternalServerError)
		return
	}
	helper.JSONResponse(w, updated)
}

func (h *MenuHandler) DeleteMenuItem(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if _, err := uuid.Parse(id); err != nil {
		helper.JSONError(w, "invalid menu item id", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), requestTimeout)
	defer cancel()

	_, err := h.menuItemService.SoftDeleteMenuItem(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			helper.JSONError(w, "menu item not found", http.StatusNotFound)
			return
		}
		helper.JSONError(w, "failed to delete menu item", http.StatusInternalServerError)
		return
	}
	helper.JSONResponse(w, map[string]bool{"deleted": true})
}

func scanCategory(row *sql.Row) (models.Category, error) {
	var category models.Category
	err := row.Scan(
		&category.ID,
		&category.RestaurantID,
		&category.Name,
		&category.Description,
		&category.IsActive,
		&category.CreatedAt,
		&category.UpdatedAt,
	)
	return category, err
}

func scanMenuItem(row *sql.Row) (models.MenuItem, error) {
	var item models.MenuItem
	var imageURL sql.NullString
	err := row.Scan(
		&item.ID,
		&item.RestaurantID,
		&item.CategoryID,
		&item.Name,
		&item.Description,
		&item.Price,
		&item.IsAvailable,
		&imageURL,
		&item.CreatedAt,
		&item.UpdatedAt,
	)
	if err != nil {
		return models.MenuItem{}, err
	}
	item.ImageURL = imageURL.String
	return item, nil
}

func scanMenuItemFromRows(rows *sql.Rows) (models.MenuItem, error) {
	var item models.MenuItem
	var imageURL sql.NullString
	err := rows.Scan(
		&item.ID,
		&item.RestaurantID,
		&item.CategoryID,
		&item.Name,
		&item.Description,
		&item.Price,
		&item.IsAvailable,
		&imageURL,
		&item.CreatedAt,
		&item.UpdatedAt,
	)
	if err != nil {
		return models.MenuItem{}, err
	}
	item.ImageURL = imageURL.String
	return item, nil
}

func isMenuValidationError(err error) bool {
	switch err.Error() {
	case "restaurant_id is required",
		"category_id is required",
		"name is required",
		"price must be greater than 0",
		"category not found",
		"category does not belong to the restaurant",
		"category is not active":
		return true
	default:
		return false
	}
}
