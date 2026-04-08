package repository

import (
	"context"
	"database/sql"

	"trythenga.com/models"
)

type FloorRepository struct {
	DB *sql.DB
}

func NewFloorRepository(db *sql.DB) *FloorRepository {
	return &FloorRepository{DB: db}
}

func (r *FloorRepository) CreateFloor(ctx context.Context, floor models.Floor) (models.Floor, error) {
	query := `
		INSERT INTO floors (id, restaurant_id, name, description)
		VALUES ($1, $2, $3, NULLIF($4, ''))
		RETURNING id, restaurant_id, name, COALESCE(description, ''), created_at, updated_at;
	`
	row := r.DB.QueryRowContext(ctx, query, floor.ID, floor.RestaurantID, floor.Name, floor.Description)
	return scanFloor(row)
}

func (r *FloorRepository) GetFloorsByRestaurant(ctx context.Context, restaurantID string) ([]models.Floor, error) {
	query := `
		SELECT id, restaurant_id, name, COALESCE(description, ''), created_at, updated_at
		FROM floors
		WHERE restaurant_id = $1
		ORDER BY created_at DESC;
	`
	rows, err := r.DB.QueryContext(ctx, query, restaurantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	floors := make([]models.Floor, 0)
	for rows.Next() {
		var floor models.Floor
		if err := rows.Scan(
			&floor.ID,
			&floor.RestaurantID,
			&floor.Name,
			&floor.Description,
			&floor.CreatedAt,
			&floor.UpdatedAt,
		); err != nil {
			return nil, err
		}
		floors = append(floors, floor)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return floors, nil
}

func (r *FloorRepository) GetFloorByID(ctx context.Context, id string) (models.Floor, error) {
	query := `
		SELECT id, restaurant_id, name, COALESCE(description, ''), created_at, updated_at
		FROM floors
		WHERE id = $1;
	`
	row := r.DB.QueryRowContext(ctx, query, id)
	return scanFloor(row)
}

func (r *FloorRepository) UpdateFloor(ctx context.Context, floor models.Floor) (models.Floor, error) {
	query := `
		UPDATE floors
		SET name = $2, description = NULLIF($3, ''), updated_at = NOW()
		WHERE id = $1
		RETURNING id, restaurant_id, name, COALESCE(description, ''), created_at, updated_at;
	`
	row := r.DB.QueryRowContext(ctx, query, floor.ID, floor.Name, floor.Description)
	return scanFloor(row)
}

func (r *FloorRepository) DeleteFloor(ctx context.Context, id string) error {
	result, err := r.DB.ExecContext(ctx, "DELETE FROM floors WHERE id = $1", id)
	if err != nil {
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *FloorRepository) FloorExistsForRestaurant(ctx context.Context, floorID, restaurantID string) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1 FROM floors WHERE id = $1 AND restaurant_id = $2
		);
	`
	var exists bool
	err := r.DB.QueryRowContext(ctx, query, floorID, restaurantID).Scan(&exists)
	return exists, err
}

func scanFloor(row *sql.Row) (models.Floor, error) {
	var floor models.Floor
	err := row.Scan(
		&floor.ID,
		&floor.RestaurantID,
		&floor.Name,
		&floor.Description,
		&floor.CreatedAt,
		&floor.UpdatedAt,
	)
	return floor, err
}
