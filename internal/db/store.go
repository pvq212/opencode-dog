// Package db provides PostgreSQL persistence via pgx v5.
//
// The Store interface abstracts all database operations, enabling unit testing
// of consumers (auth, analyzer, api, etc.) without a live database connection.
// The concrete DB type implements Store using pgx connection pooling.
package db

import (
	"context"
	"encoding/json"
	"time"
)

// Store defines the complete set of database operations used throughout the application.
// Each consuming package depends on this interface (or a subset of it) rather than the
// concrete *DB type, making dependency injection and testing straightforward.
type Store interface {
	// Close releases the underlying connection pool.
	Close()

	// RunMigrations executes SQL migration files from the given directory.
	RunMigrations(ctx context.Context, dir string) error

	// --- SSH Keys ---

	CreateSSHKey(ctx context.Context, k *SSHKey) error
	ListSSHKeys(ctx context.Context) ([]*SSHKey, error)
	GetSSHKey(ctx context.Context, id string) (*SSHKey, error)
	DeleteSSHKey(ctx context.Context, id string) error

	// --- Projects ---

	CreateProject(ctx context.Context, p *Project) error
	ListProjects(ctx context.Context) ([]*Project, error)
	GetProject(ctx context.Context, id string) (*Project, error)
	UpdateProject(ctx context.Context, p *Project) error
	DeleteProject(ctx context.Context, id string) error

	// --- Provider Configs ---

	CreateProviderConfig(ctx context.Context, pc *ProviderConfig) error
	ListProviderConfigs(ctx context.Context, projectID string) ([]*ProviderConfig, error)
	GetProviderConfig(ctx context.Context, id string) (*ProviderConfig, error)
	GetProviderConfigByPath(ctx context.Context, path string) (*ProviderConfig, error)
	ListAllProviderConfigs(ctx context.Context) ([]*ProviderConfig, error)
	DeleteProviderConfig(ctx context.Context, id string) error

	// --- Trigger Keywords ---

	SetTriggerKeywords(ctx context.Context, projectID string, keywords []TriggerKeyword) error
	GetTriggerKeywords(ctx context.Context, projectID string) ([]*TriggerKeyword, error)

	// --- Tasks ---

	CreateTask(ctx context.Context, t *Task) error
	UpdateTaskStatus(ctx context.Context, taskID string, status TaskStatus, result *string, errMsg *string) error
	ListTasks(ctx context.Context, limit, offset int) ([]*Task, error)
	GetTask(ctx context.Context, id string) (*Task, error)
	CountTasks(ctx context.Context) (int, error)

	// --- Webhook Dedup ---

	IsWebhookProcessed(ctx context.Context, eventUUID string) (bool, error)
	RecordWebhookDelivery(ctx context.Context, delivery *WebhookDelivery) error

	// --- Settings ---

	GetSetting(ctx context.Context, key string) (*Setting, error)
	GetSettingBool(ctx context.Context, key string, fallback bool) bool
	GetSettingString(ctx context.Context, key string, fallback string) string
	GetSettingInt(ctx context.Context, key string, fallback int) int
	GetSettingDuration(ctx context.Context, key string, fallback time.Duration) time.Duration
	SetSetting(ctx context.Context, key string, value json.RawMessage) error
	ListSettings(ctx context.Context) ([]*Setting, error)
	DeleteSetting(ctx context.Context, key string) error

	// --- MCP Servers ---

	CreateMCPServer(ctx context.Context, m *MCPServer) error
	ListMCPServers(ctx context.Context) ([]*MCPServer, error)
	ListEnabledMCPServers(ctx context.Context) ([]*MCPServer, error)
	GetMCPServer(ctx context.Context, id string) (*MCPServer, error)
	UpdateMCPServerStatus(ctx context.Context, id string, status MCPServerStatus, errMsg *string) error
	UpdateMCPServer(ctx context.Context, m *MCPServer) error
	DeleteMCPServer(ctx context.Context, id string) error

	// --- Users ---

	CreateUser(ctx context.Context, u *User) error
	GetUserByUsername(ctx context.Context, username string) (*User, error)
	GetUser(ctx context.Context, id string) (*User, error)
	ListUsers(ctx context.Context) ([]*User, error)
	UpdateUser(ctx context.Context, u *User) error
	UpdateUserPassword(ctx context.Context, id string, hash string) error
	DeleteUser(ctx context.Context, id string) error
	CountUsers(ctx context.Context) (int, error)
}

// Compile-time check: *DB must satisfy Store.
var _ Store = (*DB)(nil)
