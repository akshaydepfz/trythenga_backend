package service

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"github.com/google/uuid"

	"trythenga.com/models"
	"trythenga.com/repository"
)

var allowedTableStatuses = map[string]bool{
	"vacant":      true,
	"occupied":    true,
	"reserved":    true,
	"payment_due": true,
}

var allowedTableShapes = map[string]bool{
	"square": true,
	"circle": true,
}

type TableService struct {
	repo      *repository.TableRepository
	floorRepo *repository.FloorRepository
}

func NewTableService(repo *repository.TableRepository, floorRepo *repository.FloorRepository) *TableService {
	return &TableService{repo: repo, floorRepo: floorRepo}
}

func (s *TableService) CreateTable(ctx context.Context, table models.Table) (models.Table, error) {
	if err := s.validateCreateOrUpdate(ctx, table, ""); err != nil {
		return models.Table{}, err
	}

	table.ID = uuid.NewString()
	table.Status = normalizeStatus(table.Status)
	table.Shape = normalizeShape(table.Shape)
	table.TableNumber = strings.TrimSpace(table.TableNumber)
	table.IsActive = true

	return s.repo.CreateTable(ctx, table)
}

func (s *TableService) GetTablesByFloor(ctx context.Context, floorID string) ([]models.Table, error) {
	floorID = strings.TrimSpace(floorID)
	if floorID == "" {
		return nil, errors.New("floor_id is required")
	}
	return s.repo.GetTablesByFloor(ctx, floorID)
}

func (s *TableService) GetTableByID(ctx context.Context, id string) (models.Table, error) {
	return s.repo.GetTableByID(ctx, id)
}

func (s *TableService) UpdateTable(ctx context.Context, id string, table models.Table) (models.Table, error) {
	if err := s.validateCreateOrUpdate(ctx, table, id); err != nil {
		return models.Table{}, err
	}

	table.ID = id
	table.Status = normalizeStatus(table.Status)
	table.Shape = normalizeShape(table.Shape)
	table.TableNumber = strings.TrimSpace(table.TableNumber)

	return s.repo.UpdateTable(ctx, table)
}

func (s *TableService) SoftDeleteTable(ctx context.Context, id string) (models.Table, error) {
	return s.repo.SoftDeleteTable(ctx, id)
}

func (s *TableService) UpdateTablePosition(ctx context.Context, id string, posX, posY int) (models.Table, error) {
	return s.repo.UpdateTablePosition(ctx, id, posX, posY)
}

func (s *TableService) UpdateTableStatus(ctx context.Context, id, status string) (models.Table, error) {
	status = strings.TrimSpace(strings.ToLower(status))
	if status == "" {
		return models.Table{}, errors.New("status is required")
	}
	if !allowedTableStatuses[status] {
		return models.Table{}, errors.New("invalid status value")
	}
	return s.repo.UpdateTableStatus(ctx, id, status)
}

func (s *TableService) BulkUpdateTablePositions(ctx context.Context, updates []models.TablePositionUpdate) error {
	if len(updates) == 0 {
		return errors.New("positions payload is required")
	}
	for i := range updates {
		updates[i].ID = strings.TrimSpace(updates[i].ID)
		if updates[i].ID == "" {
			return errors.New("id is required for position update")
		}
	}
	return s.repo.BulkUpdateTablePositions(ctx, updates)
}

func (s *TableService) validateCreateOrUpdate(ctx context.Context, table models.Table, excludeID string) error {
	table.RestaurantID = strings.TrimSpace(table.RestaurantID)
	if table.RestaurantID == "" {
		return errors.New("restaurant_id is required")
	}
	table.FloorID = strings.TrimSpace(table.FloorID)
	if table.FloorID == "" {
		return errors.New("floor_id is required")
	}
	table.TableNumber = strings.TrimSpace(table.TableNumber)
	if table.TableNumber == "" {
		return errors.New("table_number is required")
	}
	if table.Status != "" {
		if !allowedTableStatuses[normalizeStatus(table.Status)] {
			return errors.New("invalid status value")
		}
	}
	if table.Shape != "" {
		if !allowedTableShapes[normalizeShape(table.Shape)] {
			return errors.New("invalid shape value")
		}
	}
	exists, err := s.floorRepo.FloorExistsForRestaurant(ctx, table.FloorID, table.RestaurantID)
	if err != nil {
		return err
	}
	if !exists {
		return errors.New("floor does not belong to the restaurant")
	}
	duplicate, err := s.repo.IsTableNumberDuplicateInFloor(ctx, table.FloorID, table.TableNumber, excludeID)
	if err != nil {
		return err
	}
	if duplicate {
		return errors.New("table_number already exists in this floor")
	}
	return nil
}

func normalizeStatus(status string) string {
	status = strings.TrimSpace(strings.ToLower(status))
	if status == "" {
		return "vacant"
	}
	return status
}

func normalizeShape(shape string) string {
	shape = strings.TrimSpace(strings.ToLower(shape))
	if shape == "" {
		return "square"
	}
	return shape
}

func IsTableValidationError(err error) bool {
	if errors.Is(err, sql.ErrNoRows) {
		return false
	}
	switch err.Error() {
	case "restaurant_id is required",
		"floor_id is required",
		"table_number is required",
		"status is required",
		"invalid status value",
		"invalid shape value",
		"floor does not belong to the restaurant",
		"table_number already exists in this floor",
		"positions payload is required",
		"id is required for position update":
		return true
	default:
		return false
	}
}
