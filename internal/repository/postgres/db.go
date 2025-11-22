package postgres

import (
	"fmt"
	"time"

	"github.com/V1merX/pr-reviewer-service/internal/config"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

func Open(cfg *config.Config) (*sqlx.DB, error) {
	if cfg == nil {
		return nil, fmt.Errorf("postgres: empty config")
	}

	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.Username,
		cfg.Database.Password,
		cfg.Database.DBName,
		cfg.Database.SSLMode,
	)

	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("postgres connect: %w", err)
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	return db, nil
}

func Close(db *sqlx.DB) {
	if db == nil {
		return
	}
	if err := db.Close(); err != nil {
		fmt.Printf("postgres: failed to close db: %v\n", err)
	}
}
