package db

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type DB struct {
	Pool *pgxpool.Pool
}

func New(ctx context.Context, dsn string, maxConns int32, minConns int32, maxLifetime time.Duration) (*DB, error) {
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse dsn: %w", err)
	}
	config.MaxConns = maxConns
	config.MinConns = minConns
	config.MaxConnLifetime = maxLifetime

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("connect to database: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("ping database: %w", err)
	}
	return &DB{Pool: pool}, nil
}

func (d *DB) Close() { d.Pool.Close() }

func (d *DB) RunMigrations(ctx context.Context, dir string) error {
	sql, err := os.ReadFile(dir + "/001_init.sql")
	if err != nil {
		return fmt.Errorf("read migration: %w", err)
	}
	_, err = d.Pool.Exec(ctx, string(sql))
	return err
}

func HashPayload(payload []byte) string {
	h := sha256.Sum256(payload)
	return fmt.Sprintf("%x", h)
}

func ToJSON(v any) json.RawMessage {
	b, _ := json.Marshal(v)
	return b
}
