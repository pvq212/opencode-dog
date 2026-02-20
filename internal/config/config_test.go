package config

import (
	"strings"
	"testing"
	"time"
)

// --- Load defaults ---

func TestLoad_Defaults(t *testing.T) {
	t.Setenv("DB_PASSWORD", "secret")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	checks := []struct {
		name string
		got  string
		want string
	}{
		{"ServerPort", cfg.ServerPort, "8080"},
		{"ServerHost", cfg.ServerHost, "0.0.0.0"},
		{"DBHost", cfg.DBHost, "localhost"},
		{"DBUser", cfg.DBUser, "opencode"},
		{"DBName", cfg.DBName, "opencode_dog"},
		{"DBSSLMode", cfg.DBSSLMode, "disable"},
		{"OpencodeConfigDir", cfg.OpencodeConfigDir, "/app/config"},
	}
	for _, c := range checks {
		if c.got != c.want {
			t.Errorf("%s = %q, want %q", c.name, c.got, c.want)
		}
	}

	if cfg.DBPort != 5432 {
		t.Errorf("DBPort = %d, want 5432", cfg.DBPort)
	}
	if cfg.DBPassword != "secret" {
		t.Errorf("DBPassword = %q, want %q", cfg.DBPassword, "secret")
	}
}

// --- Load with custom env vars ---

func TestLoad_CustomEnvVars(t *testing.T) {
	t.Setenv("DB_PASSWORD", "mypass")
	t.Setenv("SERVER_PORT", "9090")
	t.Setenv("SERVER_HOST", "127.0.0.1")
	t.Setenv("DB_HOST", "db.example.com")
	t.Setenv("DB_PORT", "5433")
	t.Setenv("DB_USER", "admin")
	t.Setenv("DB_NAME", "mydb")
	t.Setenv("DB_SSLMODE", "require")
	t.Setenv("OPENCODE_CONFIG_DIR", "/tmp/oc")
	t.Setenv("JWT_SECRET", "jwt-key")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.ServerPort != "9090" {
		t.Errorf("ServerPort = %q, want %q", cfg.ServerPort, "9090")
	}
	if cfg.ServerHost != "127.0.0.1" {
		t.Errorf("ServerHost = %q, want %q", cfg.ServerHost, "127.0.0.1")
	}
	if cfg.DBHost != "db.example.com" {
		t.Errorf("DBHost = %q, want %q", cfg.DBHost, "db.example.com")
	}
	if cfg.DBPort != 5433 {
		t.Errorf("DBPort = %d, want 5433", cfg.DBPort)
	}
	if cfg.DBUser != "admin" {
		t.Errorf("DBUser = %q, want %q", cfg.DBUser, "admin")
	}
	if cfg.DBName != "mydb" {
		t.Errorf("DBName = %q, want %q", cfg.DBName, "mydb")
	}
	if cfg.DBSSLMode != "require" {
		t.Errorf("DBSSLMode = %q, want %q", cfg.DBSSLMode, "require")
	}
	if cfg.OpencodeConfigDir != "/tmp/oc" {
		t.Errorf("OpencodeConfigDir = %q, want %q", cfg.OpencodeConfigDir, "/tmp/oc")
	}
	if cfg.JWTSecret != "jwt-key" {
		t.Errorf("JWTSecret = %q, want %q", cfg.JWTSecret, "jwt-key")
	}
}

// --- Load missing DB_PASSWORD ---

func TestLoad_MissingDBPassword(t *testing.T) {
	t.Setenv("DB_PASSWORD", "")

	cfg, err := Load()
	if err == nil {
		t.Fatal("Load() expected error for missing DB_PASSWORD, got nil")
	}
	if cfg != nil {
		t.Errorf("Load() returned non-nil config on error")
	}
	if !strings.Contains(err.Error(), "DB_PASSWORD") {
		t.Errorf("error = %q, want it to mention DB_PASSWORD", err.Error())
	}
}

// --- Load timeout/duration env vars ---

func TestLoad_DurationEnvVars(t *testing.T) {
	t.Setenv("DB_PASSWORD", "secret")
	t.Setenv("SERVER_READ_HEADER_TIMEOUT", "5s")
	t.Setenv("SERVER_READ_TIMEOUT", "15s")
	t.Setenv("SERVER_WRITE_TIMEOUT", "45s")
	t.Setenv("SERVER_IDLE_TIMEOUT", "90s")
	t.Setenv("SERVER_SHUTDOWN_TIMEOUT", "10s")
	t.Setenv("DB_MAX_CONN_LIFETIME", "1h")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	checks := []struct {
		name string
		got  time.Duration
		want time.Duration
	}{
		{"ReadHeaderTimeout", cfg.ReadHeaderTimeout, 5 * time.Second},
		{"ReadTimeout", cfg.ReadTimeout, 15 * time.Second},
		{"WriteTimeout", cfg.WriteTimeout, 45 * time.Second},
		{"IdleTimeout", cfg.IdleTimeout, 90 * time.Second},
		{"ShutdownTimeout", cfg.ShutdownTimeout, 10 * time.Second},
		{"DBMaxConnLifetime", cfg.DBMaxConnLifetime, 1 * time.Hour},
	}
	for _, c := range checks {
		if c.got != c.want {
			t.Errorf("%s = %v, want %v", c.name, c.got, c.want)
		}
	}
}

// --- Load duration defaults ---

func TestLoad_DurationDefaults(t *testing.T) {
	t.Setenv("DB_PASSWORD", "secret")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	checks := []struct {
		name string
		got  time.Duration
		want time.Duration
	}{
		{"ReadHeaderTimeout", cfg.ReadHeaderTimeout, 10 * time.Second},
		{"ReadTimeout", cfg.ReadTimeout, 30 * time.Second},
		{"WriteTimeout", cfg.WriteTimeout, 60 * time.Second},
		{"IdleTimeout", cfg.IdleTimeout, 120 * time.Second},
		{"ShutdownTimeout", cfg.ShutdownTimeout, 30 * time.Second},
		{"DBMaxConnLifetime", cfg.DBMaxConnLifetime, 30 * time.Minute},
	}
	for _, c := range checks {
		if c.got != c.want {
			t.Errorf("%s = %v, want %v", c.name, c.got, c.want)
		}
	}
}

// --- Load int env vars ---

func TestLoad_IntEnvVars(t *testing.T) {
	t.Setenv("DB_PASSWORD", "secret")
	t.Setenv("DB_MAX_CONNS", "50")
	t.Setenv("DB_MIN_CONNS", "5")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.DBMaxConns != 50 {
		t.Errorf("DBMaxConns = %d, want 50", cfg.DBMaxConns)
	}
	if cfg.DBMinConns != 5 {
		t.Errorf("DBMinConns = %d, want 5", cfg.DBMinConns)
	}
}

// --- Load int defaults ---

func TestLoad_IntDefaults(t *testing.T) {
	t.Setenv("DB_PASSWORD", "secret")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.DBMaxConns != 20 {
		t.Errorf("DBMaxConns = %d, want 20", cfg.DBMaxConns)
	}
	if cfg.DBMinConns != 2 {
		t.Errorf("DBMinConns = %d, want 2", cfg.DBMinConns)
	}
}

// --- DSN ---

func TestDSN(t *testing.T) {
	cfg := &Config{
		DBUser:     "user",
		DBPassword: "pass",
		DBHost:     "host",
		DBPort:     5432,
		DBName:     "dbname",
		DBSSLMode:  "disable",
	}
	want := "postgres://user:pass@host:5432/dbname?sslmode=disable"
	got := cfg.DSN()
	if got != want {
		t.Errorf("DSN() = %q, want %q", got, want)
	}
}

func TestDSN_SpecialChars(t *testing.T) {
	cfg := &Config{
		DBUser:     "admin",
		DBPassword: "p@ss:word",
		DBHost:     "db.example.com",
		DBPort:     5433,
		DBName:     "my_db",
		DBSSLMode:  "require",
	}
	want := "postgres://admin:p@ss:word@db.example.com:5433/my_db?sslmode=require"
	got := cfg.DSN()
	if got != want {
		t.Errorf("DSN() = %q, want %q", got, want)
	}
}

// --- ListenAddr ---

func TestListenAddr(t *testing.T) {
	cfg := &Config{ServerHost: "0.0.0.0", ServerPort: "8080"}
	want := "0.0.0.0:8080"
	got := cfg.ListenAddr()
	if got != want {
		t.Errorf("ListenAddr() = %q, want %q", got, want)
	}
}

func TestListenAddr_Custom(t *testing.T) {
	cfg := &Config{ServerHost: "127.0.0.1", ServerPort: "3000"}
	want := "127.0.0.1:3000"
	got := cfg.ListenAddr()
	if got != want {
		t.Errorf("ListenAddr() = %q, want %q", got, want)
	}
}

// --- getEnv ---

func TestGetEnv_Set(t *testing.T) {
	t.Setenv("TEST_GET_ENV", "value")
	got := getEnv("TEST_GET_ENV", "fallback")
	if got != "value" {
		t.Errorf("getEnv() = %q, want %q", got, "value")
	}
}

func TestGetEnv_Unset(t *testing.T) {
	got := getEnv("TEST_GET_ENV_UNSET_12345", "fallback")
	if got != "fallback" {
		t.Errorf("getEnv() = %q, want %q", got, "fallback")
	}
}

func TestGetEnv_Empty(t *testing.T) {
	t.Setenv("TEST_GET_ENV_EMPTY", "")
	got := getEnv("TEST_GET_ENV_EMPTY", "fallback")
	if got != "fallback" {
		t.Errorf("getEnv() with empty = %q, want %q", got, "fallback")
	}
}

// --- getEnvInt ---

func TestGetEnvInt_Valid(t *testing.T) {
	t.Setenv("TEST_INT", "42")
	got := getEnvInt("TEST_INT", 0)
	if got != 42 {
		t.Errorf("getEnvInt() = %d, want 42", got)
	}
}

func TestGetEnvInt_Invalid(t *testing.T) {
	t.Setenv("TEST_INT_BAD", "not-a-number")
	got := getEnvInt("TEST_INT_BAD", 99)
	if got != 99 {
		t.Errorf("getEnvInt() invalid = %d, want 99", got)
	}
}

func TestGetEnvInt_Empty(t *testing.T) {
	got := getEnvInt("TEST_INT_UNSET_12345", 77)
	if got != 77 {
		t.Errorf("getEnvInt() unset = %d, want 77", got)
	}
}

func TestGetEnvInt_Negative(t *testing.T) {
	t.Setenv("TEST_INT_NEG", "-5")
	got := getEnvInt("TEST_INT_NEG", 0)
	if got != -5 {
		t.Errorf("getEnvInt() = %d, want -5", got)
	}
}

func TestGetEnvInt_Zero(t *testing.T) {
	t.Setenv("TEST_INT_ZERO", "0")
	got := getEnvInt("TEST_INT_ZERO", 10)
	if got != 0 {
		t.Errorf("getEnvInt() = %d, want 0", got)
	}
}

// --- getEnvDuration ---

func TestGetEnvDuration_Valid(t *testing.T) {
	t.Setenv("TEST_DUR", "5m")
	got := getEnvDuration("TEST_DUR", time.Second)
	if got != 5*time.Minute {
		t.Errorf("getEnvDuration() = %v, want 5m", got)
	}
}

func TestGetEnvDuration_Invalid(t *testing.T) {
	t.Setenv("TEST_DUR_BAD", "not-a-duration")
	got := getEnvDuration("TEST_DUR_BAD", 3*time.Second)
	if got != 3*time.Second {
		t.Errorf("getEnvDuration() invalid = %v, want 3s", got)
	}
}

func TestGetEnvDuration_Empty(t *testing.T) {
	got := getEnvDuration("TEST_DUR_UNSET_12345", 7*time.Second)
	if got != 7*time.Second {
		t.Errorf("getEnvDuration() unset = %v, want 7s", got)
	}
}

func TestGetEnvDuration_Milliseconds(t *testing.T) {
	t.Setenv("TEST_DUR_MS", "500ms")
	got := getEnvDuration("TEST_DUR_MS", time.Second)
	if got != 500*time.Millisecond {
		t.Errorf("getEnvDuration() = %v, want 500ms", got)
	}
}

func TestGetEnvDuration_Hours(t *testing.T) {
	t.Setenv("TEST_DUR_H", "2h")
	got := getEnvDuration("TEST_DUR_H", time.Second)
	if got != 2*time.Hour {
		t.Errorf("getEnvDuration() = %v, want 2h", got)
	}
}
