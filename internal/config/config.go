package config

import (
	"fmt"
	"os"
	"strconv"
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

	// OpencodeConfigDir is the directory where opencode config files are stored.
	// auth.json, opencode.json, oh-my-opencode.json are generated/managed here.
	// Mounted as a Docker volume so configs persist across restarts.
	OpencodeConfigDir string

	JWTSecret string
}

func Load() (*Config, error) {
	cfg := &Config{
		ServerPort:        getEnv("SERVER_PORT", "8080"),
		ServerHost:        getEnv("SERVER_HOST", "0.0.0.0"),
		DBHost:            getEnv("DB_HOST", "localhost"),
		DBPort:            getEnvInt("DB_PORT", 5432),
		DBUser:            getEnv("DB_USER", "opencode"),
		DBPassword:        getEnv("DB_PASSWORD", ""),
		DBName:            getEnv("DB_NAME", "opencode_gitlab"),
		DBSSLMode:         getEnv("DB_SSLMODE", "disable"),
		OpencodeConfigDir: getEnv("OPENCODE_CONFIG_DIR", "/app/config"),
		JWTSecret:         getEnv("JWT_SECRET", ""),
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
