package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PGConfig struct {
	PGHost     string `env:"POSTGRES_HOST" env-default:"db"`
	PGPort     string `env:"POSTGRES_PORT" env-default:"5432"`
	PGUser     string `env:"POSTGRES_USER" env-default:"postgres"`
	PGPassword string `env:"POSTGRES_PASSWORD" env-default:"postgres"`
	PGName     string `env:"POSTGRES_DB"` 
	PGMaxConns int32 `env:"POSTGRES_MAXCONNS" env-default:"20"`
	PGMinConns int32 `env:"POSTGRES_MINCONNS" env-default:"2"`
	PGMaxConnLifetime int32 `env:"POSTGRES_MAXCONNLIFETIME" env-default:"30"`
	PGMaxConnIdleTime int32 `env:"POSTGRES_MAXCONNIDLE" env-default:"5"`

	DSN string
}

func NewPostgres(ctx context.Context, cfg PGConfig) (*pgxpool.Pool, error) {
	db, err := pgxpool.ParseConfig(cfg.DSN)
	if err != nil {
		return nil, fmt.Errorf("parse database url: %w", err)
	}

	db.MaxConns = cfg.PGMaxConns
	db.MinConns = cfg.PGMinConns
	db.MaxConnLifetime = time.Duration(cfg.PGMaxConnLifetime) * time.Minute
	db.MaxConnIdleTime = time.Duration(cfg.PGMaxConnIdleTime) * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, db)
	if err != nil {
		return nil, fmt.Errorf("create pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}

	return pool, nil
}