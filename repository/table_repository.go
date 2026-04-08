package repository

import (
	"context"
	"database/sql"

	"trythenga.com/models"
)

type TableRepository struct {
	DB *sql.DB
}

func NewTableRepository(db *sql.DB) *TableRepository {
	return &TableRepository{DB: db}
}

func (r *TableRepository) CreateTable(ctx context.Context, table models.Table) (models.Table, error) {
	query := `
		INSERT INTO tables (id, restaurant_id, floor_id, table_number, capacity, status, pos_x, pos_y, shape, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, restaurant_id, floor_id, table_number, capacity, status, pos_x, pos_y, shape, is_active, created_at, updated_at;
	`
	row := r.DB.QueryRowContext(ctx, query, table.ID, table.RestaurantID, table.FloorID, table.TableNumber, table.Capacity, table.Status, table.PosX, table.PosY, table.Shape, table.IsActive)
	return scanTable(row)
}

func (r *TableRepository) GetTablesByFloor(ctx context.Context, floorID string) ([]models.Table, error) {
	query := `
		SELECT id, restaurant_id, floor_id, table_number, capacity, status, pos_x, pos_y, shape, is_active, created_at, updated_at
		FROM tables
		WHERE floor_id = $1 AND is_active = TRUE
		ORDER BY created_at DESC;
	`
	rows, err := r.DB.QueryContext(ctx, query, floorID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tables := make([]models.Table, 0)
	for rows.Next() {
		var table models.Table
		if err := rows.Scan(
			&table.ID,
			&table.RestaurantID,
			&table.FloorID,
			&table.TableNumber,
			&table.Capacity,
			&table.Status,
			&table.PosX,
			&table.PosY,
			&table.Shape,
			&table.IsActive,
			&table.CreatedAt,
			&table.UpdatedAt,
		); err != nil {
			return nil, err
		}
		tables = append(tables, table)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return tables, nil
}

func (r *TableRepository) GetTableByID(ctx context.Context, id string) (models.Table, error) {
	query := `
		SELECT id, restaurant_id, floor_id, table_number, capacity, status, pos_x, pos_y, shape, is_active, created_at, updated_at
		FROM tables
		WHERE id = $1 AND is_active = TRUE;
	`
	row := r.DB.QueryRowContext(ctx, query, id)
	return scanTable(row)
}

func (r *TableRepository) UpdateTable(ctx context.Context, table models.Table) (models.Table, error) {
	query := `
		UPDATE tables
		SET floor_id = $2,
			table_number = $3,
			capacity = $4,
			status = $5,
			pos_x = $6,
			pos_y = $7,
			shape = $8,
			updated_at = NOW()
		WHERE id = $1 AND is_active = TRUE
		RETURNING id, restaurant_id, floor_id, table_number, capacity, status, pos_x, pos_y, shape, is_active, created_at, updated_at;
	`
	row := r.DB.QueryRowContext(ctx, query, table.ID, table.FloorID, table.TableNumber, table.Capacity, table.Status, table.PosX, table.PosY, table.Shape)
	return scanTable(row)
}

func (r *TableRepository) SoftDeleteTable(ctx context.Context, id string) (models.Table, error) {
	query := `
		UPDATE tables
		SET is_active = FALSE, updated_at = NOW()
		WHERE id = $1 AND is_active = TRUE
		RETURNING id, restaurant_id, floor_id, table_number, capacity, status, pos_x, pos_y, shape, is_active, created_at, updated_at;
	`
	row := r.DB.QueryRowContext(ctx, query, id)
	return scanTable(row)
}

func (r *TableRepository) UpdateTablePosition(ctx context.Context, id string, posX, posY int) (models.Table, error) {
	query := `
		UPDATE tables
		SET pos_x = $2, pos_y = $3, updated_at = NOW()
		WHERE id = $1 AND is_active = TRUE
		RETURNING id, restaurant_id, floor_id, table_number, capacity, status, pos_x, pos_y, shape, is_active, created_at, updated_at;
	`
	row := r.DB.QueryRowContext(ctx, query, id, posX, posY)
	return scanTable(row)
}

func (r *TableRepository) UpdateTableStatus(ctx context.Context, id, status string) (models.Table, error) {
	query := `
		UPDATE tables
		SET status = $2, updated_at = NOW()
		WHERE id = $1 AND is_active = TRUE
		RETURNING id, restaurant_id, floor_id, table_number, capacity, status, pos_x, pos_y, shape, is_active, created_at, updated_at;
	`
	row := r.DB.QueryRowContext(ctx, query, id, status)
	return scanTable(row)
}

func (r *TableRepository) BulkUpdateTablePositions(ctx context.Context, updates []models.TablePositionUpdate) error {
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	query := `
		UPDATE tables
		SET pos_x = $2, pos_y = $3, updated_at = NOW()
		WHERE id = $1 AND is_active = TRUE;
	`
	for _, update := range updates {
		result, execErr := tx.ExecContext(ctx, query, update.ID, update.PosX, update.PosY)
		if execErr != nil {
			err = execErr
			return err
		}
		rowsAffected, raErr := result.RowsAffected()
		if raErr != nil {
			err = raErr
			return err
		}
		if rowsAffected == 0 {
			err = sql.ErrNoRows
			return err
		}
	}

	if commitErr := tx.Commit(); commitErr != nil {
		return commitErr
	}
	return nil
}

func (r *TableRepository) IsTableNumberDuplicateInFloor(ctx context.Context, floorID, tableNumber string, excludeTableID string) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1
			FROM tables
			WHERE floor_id = $1
			  AND table_number = $2
			  AND is_active = TRUE
			  AND (NULLIF($3, '') IS NULL OR id <> NULLIF($3, '')::uuid)
		);
	`
	var exists bool
	err := r.DB.QueryRowContext(ctx, query, floorID, tableNumber, excludeTableID).Scan(&exists)
	return exists, err
}

func scanTable(row *sql.Row) (models.Table, error) {
	var table models.Table
	err := row.Scan(
		&table.ID,
		&table.RestaurantID,
		&table.FloorID,
		&table.TableNumber,
		&table.Capacity,
		&table.Status,
		&table.PosX,
		&table.PosY,
		&table.Shape,
		&table.IsActive,
		&table.CreatedAt,
		&table.UpdatedAt,
	)
	return table, err
}
