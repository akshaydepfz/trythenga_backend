package service

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"

	"trythenga.com/models"
	"trythenga.com/repository"
)

type FloorService struct {
	repo *repository.FloorRepository
}

func NewFloorService(repo *repository.FloorRepository) *FloorService {
	return &FloorService{repo: repo}
}

func (s *FloorService) CreateFloor(ctx context.Context, floor models.Floor) (models.Floor, error) {
	floor.RestaurantID = strings.TrimSpace(floor.RestaurantID)
	if floor.RestaurantID == "" {
		return models.Floor{}, errors.New("restaurant_id is required")
	}
	floor.Name = strings.TrimSpace(floor.Name)
	if floor.Name == "" {
		return models.Floor{}, errors.New("name is required")
	}
	floor.Description = strings.TrimSpace(floor.Description)
	floor.ID = uuid.NewString()
	return s.repo.CreateFloor(ctx, floor)
}

func (s *FloorService) GetFloorsByRestaurant(ctx context.Context, restaurantID string) ([]models.Floor, error) {
	restaurantID = strings.TrimSpace(restaurantID)
	if restaurantID == "" {
		return nil, errors.New("restaurant_id is required")
	}
	return s.repo.GetFloorsByRestaurant(ctx, restaurantID)
}

func (s *FloorService) GetFloorByID(ctx context.Context, id string) (models.Floor, error) {
	return s.repo.GetFloorByID(ctx, id)
}

func (s *FloorService) UpdateFloor(ctx context.Context, id string, floor models.Floor) (models.Floor, error) {
	floor.Name = strings.TrimSpace(floor.Name)
	if floor.Name == "" {
		return models.Floor{}, errors.New("name is required")
	}
	floor.Description = strings.TrimSpace(floor.Description)
	floor.ID = id
	return s.repo.UpdateFloor(ctx, floor)
}

func (s *FloorService) DeleteFloor(ctx context.Context, id string) error {
	if err := s.repo.DeleteFloor(ctx, id); err != nil {
		return err
	}
	return nil
}

func IsFloorValidationError(err error) bool {
	switch err.Error() {
	case "restaurant_id is required",
		"name is required":
		return true
	default:
		return false
	}
}
