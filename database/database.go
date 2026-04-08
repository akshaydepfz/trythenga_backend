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

func EnsureWaitersTable(db *sql.DB) error {
	createTableQuery := `
		CREATE TABLE IF NOT EXISTS waiters (
			id UUID PRIMARY KEY,
			restaurant_id UUID NOT NULL REFERENCES restaurants(id),
			name TEXT NOT NULL,
			phone TEXT,
			password_hash TEXT NOT NULL,
			role TEXT NOT NULL DEFAULT 'waiter',
			is_active BOOLEAN NOT NULL DEFAULT TRUE,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);
	`
	if _, err := db.Exec(createTableQuery); err != nil {
		return fmt.Errorf("failed to create waiters table: %w", err)
	}

	ensurePasswordHashColumnQuery := `
		ALTER TABLE waiters
		ADD COLUMN IF NOT EXISTS password_hash TEXT;
	`
	if _, err := db.Exec(ensurePasswordHashColumnQuery); err != nil {
		return fmt.Errorf("failed to add waiters.password_hash column: %w", err)
	}

	migrateLegacyPasswordQuery := `
		DO $$
		BEGIN
			IF EXISTS (
				SELECT 1
				FROM information_schema.columns
				WHERE table_schema = 'public'
				  AND table_name = 'waiters'
				  AND column_name = 'password'
			) THEN
				UPDATE waiters
				SET password_hash = password
				WHERE password_hash IS NULL AND password IS NOT NULL;
			END IF;
		END $$;
	`
	if _, err := db.Exec(migrateLegacyPasswordQuery); err != nil {
		return fmt.Errorf("failed to migrate waiters legacy password data: %w", err)
	}

	return nil
}
