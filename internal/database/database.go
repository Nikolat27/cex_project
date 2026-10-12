package database

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"os"

	_ "github.com/lib/pq"
	"github.com/prakis/loadenv"
)

type DB struct {
	db     *sql.DB
	Models *Models
}

func New() (*DB, error) {
	if err := loadenv.Load(".env"); err != nil {
		return nil, err
	}

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		return nil, errors.New("DATABASE_URL is not set")
	}

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}

	newModels := NewModels(db)

	newDB := &DB{db: db, Models: newModels}

	return newDB, nil
}

func (db *DB) Close() error {
	log.Println("Closing database connection...")
	return db.db.Close()
}

func (db *DB) Ping(ctx context.Context) error {
	return db.db.PingContext(ctx)
}
