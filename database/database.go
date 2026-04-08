package database

import (
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"
)

func Connect() (*sql.DB, error) {

	const (
		host     = "ep-bold-unit-a1um73lm.ap-southeast-1.pg.koyeb.app"
		port     = 5432
		user     = "koyeb-adm"
		password = "npg_qY2uILyNb3sP"
		dbname   = "koyebdb"
	)

	connectionString := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=require", host, port, user, password, dbname)
	db, err := sql.Open("postgres", connectionString)
	if err != nil {
		return nil, fmt.Errorf("error opening database: %w", err)
	}

	if err = db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("error connecting to the database: %w", err)
	}

	return db, nil
}

func EnsureRestaurantPasswordColumn(db *sql.DB) error {
	query := `
		ALTER TABLE restaurants
		ADD COLUMN IF NOT EXISTS password_hash TEXT;
	`
	if _, err := db.Exec(query); err != nil {
		return fmt.Errorf("failed to alter restaurants table for password_hash: %w", err)
	}
	return nil
}
