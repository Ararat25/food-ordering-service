package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"

	"avito-kitchen/internal/config"
)

// New - открывает соединение с базой и проверяет его пингом.
func New(ctx context.Context, cfg config.DatabaseConfig) (*sql.DB, error) {
	db, err := sql.Open("postgres", cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	db.SetMaxOpenConns(cfg.MaxOpenConns)       // максимум открытых соединений
	db.SetMaxIdleConns(cfg.MaxIdleConns)       // максимум простаивающих
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime) // сколько живёт соединение
	db.SetConnMaxIdleTime(cfg.ConnMaxIdleTime) // сколько простаивает

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := db.PingContext(pingCtx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return db, nil
}
