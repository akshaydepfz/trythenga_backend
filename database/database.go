package database

import (
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"
)

func Connect() (*sql.DB, error) {

	const (
		host     = "ep-summer-dust-amkn8he2.c-5.us-east-1.pg.koyeb.app"
		port     = 5432
		user     = "koyeb-adm"
		password = "npg_UvWuyLqa93Ab"
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
