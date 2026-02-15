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

// --- SSH Keys ---

func (d *DB) CreateSSHKey(ctx context.Context, k *SSHKey) error {
	return d.Pool.QueryRow(ctx,
		`INSERT INTO ssh_keys (name, private_key, public_key) VALUES ($1,$2,$3) RETURNING id, created_at`,
		k.Name, k.PrivateKey, k.PublicKey,
	).Scan(&k.ID, &k.CreatedAt)
}

func (d *DB) ListSSHKeys(ctx context.Context) ([]*SSHKey, error) {
	rows, err := d.Pool.Query(ctx, `SELECT id, name, public_key, created_at FROM ssh_keys ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var keys []*SSHKey
	for rows.Next() {
		k := &SSHKey{}
		if err := rows.Scan(&k.ID, &k.Name, &k.PublicKey, &k.CreatedAt); err != nil {
			return nil, err
		}
		keys = append(keys, k)
	}
	return keys, rows.Err()
}

func (d *DB) GetSSHKey(ctx context.Context, id string) (*SSHKey, error) {
	k := &SSHKey{}
	err := d.Pool.QueryRow(ctx, `SELECT id, name, private_key, public_key, created_at FROM ssh_keys WHERE id=$1`, id).
		Scan(&k.ID, &k.Name, &k.PrivateKey, &k.PublicKey, &k.CreatedAt)
	return k, err
}

func (d *DB) DeleteSSHKey(ctx context.Context, id string) error {
	_, err := d.Pool.Exec(ctx, `DELETE FROM ssh_keys WHERE id=$1`, id)
	return err
}

// --- Projects ---

func (d *DB) CreateProject(ctx context.Context, p *Project) error {
	return d.Pool.QueryRow(ctx,
		`INSERT INTO projects (name, ssh_url, ssh_key_id, default_branch, enabled) VALUES ($1,$2,$3,$4,$5) RETURNING id, created_at, updated_at`,
		p.Name, p.SSHURL, p.SSHKeyID, p.DefaultBranch, p.Enabled,
	).Scan(&p.ID, &p.CreatedAt, &p.UpdatedAt)
}

func (d *DB) ListProjects(ctx context.Context) ([]*Project, error) {
	rows, err := d.Pool.Query(ctx, `SELECT id, name, ssh_url, ssh_key_id, default_branch, enabled, created_at, updated_at FROM projects ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var projects []*Project
	for rows.Next() {
		p := &Project{}
		if err := rows.Scan(&p.ID, &p.Name, &p.SSHURL, &p.SSHKeyID, &p.DefaultBranch, &p.Enabled, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		projects = append(projects, p)
	}
	return projects, rows.Err()
}

func (d *DB) GetProject(ctx context.Context, id string) (*Project, error) {
	p := &Project{}
	err := d.Pool.QueryRow(ctx,
		`SELECT id, name, ssh_url, ssh_key_id, default_branch, enabled, created_at, updated_at FROM projects WHERE id=$1`, id).
		Scan(&p.ID, &p.Name, &p.SSHURL, &p.SSHKeyID, &p.DefaultBranch, &p.Enabled, &p.CreatedAt, &p.UpdatedAt)
	return p, err
}

func (d *DB) UpdateProject(ctx context.Context, p *Project) error {
	_, err := d.Pool.Exec(ctx,
		`UPDATE projects SET name=$2, ssh_url=$3, ssh_key_id=$4, default_branch=$5, enabled=$6 WHERE id=$1`,
		p.ID, p.Name, p.SSHURL, p.SSHKeyID, p.DefaultBranch, p.Enabled)
	return err
}

func (d *DB) DeleteProject(ctx context.Context, id string) error {
	_, err := d.Pool.Exec(ctx, `DELETE FROM projects WHERE id=$1`, id)
	return err
}

// --- Provider Configs ---

func (d *DB) CreateProviderConfig(ctx context.Context, pc *ProviderConfig) error {
	return d.Pool.QueryRow(ctx,
		`INSERT INTO provider_configs (project_id, provider_type, config, webhook_secret, webhook_path, enabled) VALUES ($1,$2,$3,$4,$5,$6) RETURNING id, created_at, updated_at`,
		pc.ProjectID, pc.ProviderType, pc.Config, pc.WebhookSecret, pc.WebhookPath, pc.Enabled,
	).Scan(&pc.ID, &pc.CreatedAt, &pc.UpdatedAt)
}

func (d *DB) ListProviderConfigs(ctx context.Context, projectID string) ([]*ProviderConfig, error) {
	rows, err := d.Pool.Query(ctx,
		`SELECT id, project_id, provider_type, config, webhook_secret, webhook_path, enabled, created_at, updated_at FROM provider_configs WHERE project_id=$1 ORDER BY created_at`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var configs []*ProviderConfig
	for rows.Next() {
		pc := &ProviderConfig{}
		if err := rows.Scan(&pc.ID, &pc.ProjectID, &pc.ProviderType, &pc.Config, &pc.WebhookSecret, &pc.WebhookPath, &pc.Enabled, &pc.CreatedAt, &pc.UpdatedAt); err != nil {
			return nil, err
		}
		configs = append(configs, pc)
	}
	return configs, rows.Err()
}

func (d *DB) GetProviderConfig(ctx context.Context, id string) (*ProviderConfig, error) {
	pc := &ProviderConfig{}
	err := d.Pool.QueryRow(ctx,
		`SELECT id, project_id, provider_type, config, webhook_secret, webhook_path, enabled, created_at, updated_at FROM provider_configs WHERE id=$1`, id).
		Scan(&pc.ID, &pc.ProjectID, &pc.ProviderType, &pc.Config, &pc.WebhookSecret, &pc.WebhookPath, &pc.Enabled, &pc.CreatedAt, &pc.UpdatedAt)
	return pc, err
}

func (d *DB) GetProviderConfigByPath(ctx context.Context, path string) (*ProviderConfig, error) {
	pc := &ProviderConfig{}
	err := d.Pool.QueryRow(ctx,
		`SELECT id, project_id, provider_type, config, webhook_secret, webhook_path, enabled, created_at, updated_at FROM provider_configs WHERE webhook_path=$1`, path).
		Scan(&pc.ID, &pc.ProjectID, &pc.ProviderType, &pc.Config, &pc.WebhookSecret, &pc.WebhookPath, &pc.Enabled, &pc.CreatedAt, &pc.UpdatedAt)
	return pc, err
}

func (d *DB) ListAllProviderConfigs(ctx context.Context) ([]*ProviderConfig, error) {
	rows, err := d.Pool.Query(ctx,
		`SELECT id, project_id, provider_type, config, webhook_secret, webhook_path, enabled, created_at, updated_at FROM provider_configs WHERE enabled=true ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var configs []*ProviderConfig
	for rows.Next() {
		pc := &ProviderConfig{}
		if err := rows.Scan(&pc.ID, &pc.ProjectID, &pc.ProviderType, &pc.Config, &pc.WebhookSecret, &pc.WebhookPath, &pc.Enabled, &pc.CreatedAt, &pc.UpdatedAt); err != nil {
			return nil, err
		}
		configs = append(configs, pc)
	}
	return configs, rows.Err()
}

func (d *DB) DeleteProviderConfig(ctx context.Context, id string) error {
	_, err := d.Pool.Exec(ctx, `DELETE FROM provider_configs WHERE id=$1`, id)
	return err
}

// --- Trigger Keywords ---

func (d *DB) SetTriggerKeywords(ctx context.Context, projectID string, keywords []TriggerKeyword) error {
	tx, err := d.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `DELETE FROM trigger_keywords WHERE project_id=$1`, projectID)
	if err != nil {
		return err
	}
	for _, kw := range keywords {
		_, err = tx.Exec(ctx,
			`INSERT INTO trigger_keywords (project_id, mode, keyword) VALUES ($1,$2,$3) ON CONFLICT (project_id, keyword) DO UPDATE SET mode=$2`,
			projectID, kw.Mode, kw.Keyword)
		if err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (d *DB) GetTriggerKeywords(ctx context.Context, projectID string) ([]*TriggerKeyword, error) {
	rows, err := d.Pool.Query(ctx,
		`SELECT id, project_id, mode, keyword, created_at FROM trigger_keywords WHERE project_id=$1 ORDER BY mode, keyword`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var keywords []*TriggerKeyword
	for rows.Next() {
		kw := &TriggerKeyword{}
		if err := rows.Scan(&kw.ID, &kw.ProjectID, &kw.Mode, &kw.Keyword, &kw.CreatedAt); err != nil {
			return nil, err
		}
		keywords = append(keywords, kw)
	}
	return keywords, rows.Err()
}

// --- Tasks ---

func (d *DB) CreateTask(ctx context.Context, t *Task) error {
	return d.Pool.QueryRow(ctx,
		`INSERT INTO tasks (project_id, provider_config_id, provider_type, trigger_mode, trigger_keyword, external_ref, title, message_body, author)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9) RETURNING id, created_at, updated_at`,
		t.ProjectID, t.ProviderConfigID, t.ProviderType, t.TriggerMode, t.TriggerKeyword,
		t.ExternalRef, t.Title, t.MessageBody, t.Author,
	).Scan(&t.ID, &t.CreatedAt, &t.UpdatedAt)
}

func (d *DB) UpdateTaskStatus(ctx context.Context, taskID string, status TaskStatus, result *string, errMsg *string) error {
	now := time.Now()
	var startedAt, completedAt *time.Time
	switch status {
	case TaskStatusProcessing:
		startedAt = &now
	case TaskStatusCompleted, TaskStatusFailed:
		completedAt = &now
	}
	_, err := d.Pool.Exec(ctx,
		`UPDATE tasks SET status=$2, result=$3, error_message=$4, started_at=COALESCE($5, started_at), completed_at=COALESCE($6, completed_at) WHERE id=$1`,
		taskID, status, result, errMsg, startedAt, completedAt)
	return err
}

func (d *DB) ListTasks(ctx context.Context, limit, offset int) ([]*Task, error) {
	rows, err := d.Pool.Query(ctx,
		`SELECT id, project_id, provider_config_id, provider_type, trigger_mode, trigger_keyword, external_ref, title, message_body, author, status, result, error_message, created_at, updated_at, started_at, completed_at
		 FROM tasks ORDER BY created_at DESC LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var tasks []*Task
	for rows.Next() {
		t := &Task{}
		if err := rows.Scan(&t.ID, &t.ProjectID, &t.ProviderConfigID, &t.ProviderType, &t.TriggerMode, &t.TriggerKeyword, &t.ExternalRef, &t.Title, &t.MessageBody, &t.Author, &t.Status, &t.Result, &t.ErrorMessage, &t.CreatedAt, &t.UpdatedAt, &t.StartedAt, &t.CompletedAt); err != nil {
			return nil, err
		}
		tasks = append(tasks, t)
	}
	return tasks, rows.Err()
}

func (d *DB) GetTask(ctx context.Context, id string) (*Task, error) {
	t := &Task{}
	err := d.Pool.QueryRow(ctx,
		`SELECT id, project_id, provider_config_id, provider_type, trigger_mode, trigger_keyword, external_ref, title, message_body, author, status, result, error_message, created_at, updated_at, started_at, completed_at
		 FROM tasks WHERE id=$1`, id).Scan(&t.ID, &t.ProjectID, &t.ProviderConfigID, &t.ProviderType, &t.TriggerMode, &t.TriggerKeyword, &t.ExternalRef, &t.Title, &t.MessageBody, &t.Author, &t.Status, &t.Result, &t.ErrorMessage, &t.CreatedAt, &t.UpdatedAt, &t.StartedAt, &t.CompletedAt)
	if err != nil {
		return nil, err
	}
	return t, nil
}

func (d *DB) CountTasks(ctx context.Context) (int, error) {
	var count int
	err := d.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM tasks`).Scan(&count)
	return count, err
}

// --- Webhook Dedup ---

func (d *DB) IsWebhookProcessed(ctx context.Context, eventUUID string) (bool, error) {
	var exists bool
	err := d.Pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM webhook_deliveries WHERE event_uuid=$1)`, eventUUID).Scan(&exists)
	return exists, err
}

func (d *DB) RecordWebhookDelivery(ctx context.Context, delivery *WebhookDelivery) error {
	_, err := d.Pool.Exec(ctx,
		`INSERT INTO webhook_deliveries (event_uuid, event_type, payload_hash) VALUES ($1,$2,$3) ON CONFLICT (event_uuid) DO NOTHING`,
		delivery.EventUUID, delivery.EventType, delivery.PayloadHash)
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

// --- Settings ---

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

// --- MCP Servers ---

func (d *DB) CreateMCPServer(ctx context.Context, m *MCPServer) error {
	return d.Pool.QueryRow(ctx,
		`INSERT INTO mcp_servers (name, type, package, command, args, env, enabled)
		 VALUES ($1,$2,$3,$4,$5,$6,$7) RETURNING id, created_at, updated_at`,
		m.Name, m.Type, m.Package, m.Command, m.Args, m.Env, m.Enabled,
	).Scan(&m.ID, &m.CreatedAt, &m.UpdatedAt)
}

func (d *DB) ListMCPServers(ctx context.Context) ([]*MCPServer, error) {
	rows, err := d.Pool.Query(ctx,
		`SELECT id, name, type, package, command, args, env, enabled, status, error_msg, created_at, updated_at
		 FROM mcp_servers ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var servers []*MCPServer
	for rows.Next() {
		m := &MCPServer{}
		if err := rows.Scan(&m.ID, &m.Name, &m.Type, &m.Package, &m.Command, &m.Args, &m.Env, &m.Enabled, &m.Status, &m.ErrorMsg, &m.CreatedAt, &m.UpdatedAt); err != nil {
			return nil, err
		}
		servers = append(servers, m)
	}
	return servers, rows.Err()
}

func (d *DB) ListEnabledMCPServers(ctx context.Context) ([]*MCPServer, error) {
	rows, err := d.Pool.Query(ctx,
		`SELECT id, name, type, package, command, args, env, enabled, status, error_msg, created_at, updated_at
		 FROM mcp_servers WHERE enabled=true AND status='installed' ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var servers []*MCPServer
	for rows.Next() {
		m := &MCPServer{}
		if err := rows.Scan(&m.ID, &m.Name, &m.Type, &m.Package, &m.Command, &m.Args, &m.Env, &m.Enabled, &m.Status, &m.ErrorMsg, &m.CreatedAt, &m.UpdatedAt); err != nil {
			return nil, err
		}
		servers = append(servers, m)
	}
	return servers, rows.Err()
}

func (d *DB) GetMCPServer(ctx context.Context, id string) (*MCPServer, error) {
	m := &MCPServer{}
	err := d.Pool.QueryRow(ctx,
		`SELECT id, name, type, package, command, args, env, enabled, status, error_msg, created_at, updated_at
		 FROM mcp_servers WHERE id=$1`, id).
		Scan(&m.ID, &m.Name, &m.Type, &m.Package, &m.Command, &m.Args, &m.Env, &m.Enabled, &m.Status, &m.ErrorMsg, &m.CreatedAt, &m.UpdatedAt)
	return m, err
}

func (d *DB) UpdateMCPServerStatus(ctx context.Context, id string, status MCPServerStatus, errMsg *string) error {
	_, err := d.Pool.Exec(ctx,
		`UPDATE mcp_servers SET status=$2, error_msg=$3 WHERE id=$1`,
		id, status, errMsg)
	return err
}

func (d *DB) UpdateMCPServer(ctx context.Context, m *MCPServer) error {
	_, err := d.Pool.Exec(ctx,
		`UPDATE mcp_servers SET name=$2, type=$3, package=$4, command=$5, args=$6, env=$7, enabled=$8 WHERE id=$1`,
		m.ID, m.Name, m.Type, m.Package, m.Command, m.Args, m.Env, m.Enabled)
	return err
}

func (d *DB) DeleteMCPServer(ctx context.Context, id string) error {
	_, err := d.Pool.Exec(ctx, `DELETE FROM mcp_servers WHERE id=$1`, id)
	return err
}

// --- Users ---

func (d *DB) CreateUser(ctx context.Context, u *User) error {
	return d.Pool.QueryRow(ctx,
		`INSERT INTO users (username, password_hash, display_name, role, enabled)
		 VALUES ($1,$2,$3,$4,$5) RETURNING id, created_at, updated_at`,
		u.Username, u.PasswordHash, u.DisplayName, u.Role, u.Enabled,
	).Scan(&u.ID, &u.CreatedAt, &u.UpdatedAt)
}

func (d *DB) GetUserByUsername(ctx context.Context, username string) (*User, error) {
	u := &User{}
	err := d.Pool.QueryRow(ctx,
		`SELECT id, username, password_hash, display_name, role, enabled, created_at, updated_at
		 FROM users WHERE username=$1`, username).
		Scan(&u.ID, &u.Username, &u.PasswordHash, &u.DisplayName, &u.Role, &u.Enabled, &u.CreatedAt, &u.UpdatedAt)
	return u, err
}

func (d *DB) GetUser(ctx context.Context, id string) (*User, error) {
	u := &User{}
	err := d.Pool.QueryRow(ctx,
		`SELECT id, username, password_hash, display_name, role, enabled, created_at, updated_at
		 FROM users WHERE id=$1`, id).
		Scan(&u.ID, &u.Username, &u.PasswordHash, &u.DisplayName, &u.Role, &u.Enabled, &u.CreatedAt, &u.UpdatedAt)
	return u, err
}

func (d *DB) ListUsers(ctx context.Context) ([]*User, error) {
	rows, err := d.Pool.Query(ctx,
		`SELECT id, username, password_hash, display_name, role, enabled, created_at, updated_at
		 FROM users ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var users []*User
	for rows.Next() {
		u := &User{}
		if err := rows.Scan(&u.ID, &u.Username, &u.PasswordHash, &u.DisplayName, &u.Role, &u.Enabled, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

func (d *DB) UpdateUser(ctx context.Context, u *User) error {
	_, err := d.Pool.Exec(ctx,
		`UPDATE users SET username=$2, display_name=$3, role=$4, enabled=$5 WHERE id=$1`,
		u.ID, u.Username, u.DisplayName, u.Role, u.Enabled)
	return err
}

func (d *DB) UpdateUserPassword(ctx context.Context, id string, hash string) error {
	_, err := d.Pool.Exec(ctx, `UPDATE users SET password_hash=$2 WHERE id=$1`, id, hash)
	return err
}

func (d *DB) DeleteUser(ctx context.Context, id string) error {
	_, err := d.Pool.Exec(ctx, `DELETE FROM users WHERE id=$1`, id)
	return err
}

func (d *DB) CountUsers(ctx context.Context) (int, error) {
	var count int
	err := d.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM users`).Scan(&count)
	return count, err
}
