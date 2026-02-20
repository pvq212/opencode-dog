package db

import (
	"context"
	"encoding/json"
	"time"
)

func (d *DB) GetSetting(ctx context.Context, key string) (*Setting, error) {
	s := &Setting{}
	err := d.Pool.QueryRow(ctx,
		`SELECT key, value, updated_at FROM settings WHERE key=$1`, key).
		Scan(&s.Key, &s.Value, &s.UpdatedAt)
	return s, err
}

func (d *DB) GetSettingBool(ctx context.Context, key string, fallback bool) bool {
	s, err := d.GetSetting(ctx, key)
	if err != nil {
		return fallback
	}
	var v bool
	if err := json.Unmarshal(s.Value, &v); err != nil {
		return fallback
	}
	return v
}

func (d *DB) GetSettingString(ctx context.Context, key string, fallback string) string {
	s, err := d.GetSetting(ctx, key)
	if err != nil {
		return fallback
	}
	var v string
	if err := json.Unmarshal(s.Value, &v); err != nil {
		return fallback
	}
	return v
}

func (d *DB) GetSettingInt(ctx context.Context, key string, fallback int) int {
	s, err := d.GetSetting(ctx, key)
	if err != nil {
		return fallback
	}
	var v int
	if err := json.Unmarshal(s.Value, &v); err != nil {
		return fallback
	}
	return v
}

func (d *DB) GetSettingDuration(ctx context.Context, key string, fallback time.Duration) time.Duration {
	s, err := d.GetSetting(ctx, key)
	if err != nil {
		return fallback
	}
	var v string
	if err := json.Unmarshal(s.Value, &v); err != nil {
		return fallback
	}
	parsed, err := time.ParseDuration(v)
	if err != nil {
		return fallback
	}
	return parsed
}

func (d *DB) SetSetting(ctx context.Context, key string, value json.RawMessage) error {
	_, err := d.Pool.Exec(ctx,
		`INSERT INTO settings (key, value) VALUES ($1, $2)
		 ON CONFLICT (key) DO UPDATE SET value=$2`,
		key, value)
	return err
}

func (d *DB) ListSettings(ctx context.Context) ([]*Setting, error) {
	rows, err := d.Pool.Query(ctx, `SELECT key, value, updated_at FROM settings ORDER BY key`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var settings []*Setting
	for rows.Next() {
		s := &Setting{}
		if err := rows.Scan(&s.Key, &s.Value, &s.UpdatedAt); err != nil {
			return nil, err
		}
		settings = append(settings, s)
	}
	return settings, rows.Err()
}

func (d *DB) DeleteSetting(ctx context.Context, key string) error {
	_, err := d.Pool.Exec(ctx, `DELETE FROM settings WHERE key=$1`, key)
	return err
}
