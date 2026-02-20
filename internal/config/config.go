// Package config loads infrastructure settings from environment variables.
//
// Only infrastructure concerns (database, JWT, server port) belong here.
// All business configuration is stored in PostgreSQL and managed via the WebUI.
package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds infrastructure-only settings loaded from environment variables.
// All business configuration (providers, MCP servers, keywords, etc.) lives in
// the database and is managed through the WebUI.
type Config struct {
	ServerPort string
	ServerHost string

	DBHost     string
	DBPort     int
	DBUser     string
	DBPassword string
	DBName     string
	DBSSLMode  string

	// OpencodeConfigDir stores opencode config files (auth.json, .opencode.json,
	// oh-my-opencode.json) synced from DB. Bind-mounted to the OpenCode server
	// container so both services share the same configuration.
	OpencodeConfigDir string

	JWTSecret string

	ReadHeaderTimeout time.Duration
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
	ShutdownTimeout   time.Duration
	DBMaxConns        int32
	DBMinConns        int32
	DBMaxConnLifetime time.Duration
}

func Load() (*Config, error) {
	cfg := &Config{
		ServerPort:        getEnv("SERVER_PORT", "8080"),
		ServerHost:        getEnv("SERVER_HOST", "0.0.0.0"),
		DBHost:            getEnv("DB_HOST", "localhost"),
		DBPort:            getEnvInt("DB_PORT", 5432),
		DBUser:            getEnv("DB_USER", "opencode"),
		DBPassword:        getEnv("DB_PASSWORD", ""),
		DBName:            getEnv("DB_NAME", "opencode_dog"),
		DBSSLMode:         getEnv("DB_SSLMODE", "disable"),
		OpencodeConfigDir: getEnv("OPENCODE_CONFIG_DIR", "/app/config"),
		JWTSecret:         getEnv("JWT_SECRET", ""),
		ReadHeaderTimeout: getEnvDuration("SERVER_READ_HEADER_TIMEOUT", 10*time.Second),
		ReadTimeout:       getEnvDuration("SERVER_READ_TIMEOUT", 30*time.Second),
		WriteTimeout:      getEnvDuration("SERVER_WRITE_TIMEOUT", 60*time.Second),
		IdleTimeout:       getEnvDuration("SERVER_IDLE_TIMEOUT", 120*time.Second),
		ShutdownTimeout:   getEnvDuration("SERVER_SHUTDOWN_TIMEOUT", 30*time.Second),
		DBMaxConns:        int32(getEnvInt("DB_MAX_CONNS", 20)),
		DBMinConns:        int32(getEnvInt("DB_MIN_CONNS", 2)),
		DBMaxConnLifetime: getEnvDuration("DB_MAX_CONN_LIFETIME", 30*time.Minute),
	}
	if cfg.DBPassword == "" {
		return nil, fmt.Errorf("DB_PASSWORD is required")
	}
	return cfg, nil
}

func (c *Config) DSN() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		c.DBUser, c.DBPassword, c.DBHost, c.DBPort, c.DBName, c.DBSSLMode)
}

func (c *Config) ListenAddr() string {
	return fmt.Sprintf("%s:%s", c.ServerHost, c.ServerPort)
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	i, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return i
}

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return fallback
	}
	return d
}
