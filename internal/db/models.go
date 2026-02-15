package db

import (
	"encoding/json"
	"time"
)

type TaskStatus string

const (
	TaskStatusPending    TaskStatus = "pending"
	TaskStatusProcessing TaskStatus = "processing"
	TaskStatusCompleted  TaskStatus = "completed"
	TaskStatusFailed     TaskStatus = "failed"
	TaskStatusCancelled  TaskStatus = "cancelled"
)

type SSHKey struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	PrivateKey string    `json:"private_key,omitempty"`
	PublicKey  string    `json:"public_key"`
	CreatedAt  time.Time `json:"created_at"`
}

type Project struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	SSHURL        string    `json:"ssh_url"`
	SSHKeyID      *string   `json:"ssh_key_id,omitempty"`
	DefaultBranch string    `json:"default_branch"`
	Enabled       bool      `json:"enabled"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type ProviderConfig struct {
	ID            string          `json:"id"`
	ProjectID     string          `json:"project_id"`
	ProviderType  string          `json:"provider_type"`
	Config        json.RawMessage `json:"config"`
	WebhookSecret string          `json:"webhook_secret"`
	WebhookPath   string          `json:"webhook_path"`
	Enabled       bool            `json:"enabled"`
	CreatedAt     time.Time       `json:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at"`
}

func (pc *ProviderConfig) ConfigMap() map[string]any {
	m := make(map[string]any)
	_ = json.Unmarshal(pc.Config, &m)
	return m
}

type TriggerKeyword struct {
	ID        string    `json:"id"`
	ProjectID string    `json:"project_id"`
	Mode      string    `json:"mode"`
	Keyword   string    `json:"keyword"`
	CreatedAt time.Time `json:"created_at"`
}

type Task struct {
	ID               string     `json:"id"`
	ProjectID        *string    `json:"project_id,omitempty"`
	ProviderConfigID *string    `json:"provider_config_id,omitempty"`
	ProviderType     string     `json:"provider_type"`
	TriggerMode      string     `json:"trigger_mode"`
	TriggerKeyword   string     `json:"trigger_keyword"`
	ExternalRef      string     `json:"external_ref"`
	Title            string     `json:"title"`
	MessageBody      string     `json:"message_body"`
	Author           string     `json:"author"`
	Status           TaskStatus `json:"status"`
	Result           *string    `json:"result,omitempty"`
	ErrorMessage     *string    `json:"error_message,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
	StartedAt        *time.Time `json:"started_at,omitempty"`
	CompletedAt      *time.Time `json:"completed_at,omitempty"`
}

type WebhookDelivery struct {
	ID          string    `json:"id"`
	EventUUID   string    `json:"event_uuid"`
	EventType   string    `json:"event_type"`
	PayloadHash string    `json:"payload_hash"`
	Processed   bool      `json:"processed"`
	CreatedAt   time.Time `json:"created_at"`
}

type Setting struct {
	Key       string          `json:"key"`
	Value     json.RawMessage `json:"value"`
	UpdatedAt time.Time       `json:"updated_at"`
}

type MCPServerStatus string

const (
	MCPStatusPending      MCPServerStatus = "pending"
	MCPStatusInstalling   MCPServerStatus = "installing"
	MCPStatusInstalled    MCPServerStatus = "installed"
	MCPStatusFailed       MCPServerStatus = "failed"
	MCPStatusUninstalling MCPServerStatus = "uninstalling"
)

type MCPServer struct {
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	Type      string          `json:"type"`
	Package   string          `json:"package"`
	Command   string          `json:"command"`
	Args      json.RawMessage `json:"args"`
	Env       json.RawMessage `json:"env"`
	Enabled   bool            `json:"enabled"`
	Status    MCPServerStatus `json:"status"`
	ErrorMsg  *string         `json:"error_msg,omitempty"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}

const (
	RoleAdmin  = "admin"
	RoleEditor = "editor"
	RoleViewer = "viewer"
)

type User struct {
	ID           string    `json:"id"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"-"`
	DisplayName  string    `json:"display_name"`
	Role         string    `json:"role"`
	Enabled      bool      `json:"enabled"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}
