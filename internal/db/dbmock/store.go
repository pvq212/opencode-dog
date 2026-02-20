package dbmock

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/opencode-ai/opencode-dog/internal/db"
)

// Store is an in-memory implementation of db.Store for unit testing.
// All fields are exported so tests can pre-populate data or inspect state.
type Store struct {
	mu sync.RWMutex

	SSHKeys         []*db.SSHKey
	Projects        []*db.Project
	ProviderConfigs []*db.ProviderConfig
	TriggerKeywords []*db.TriggerKeyword
	Tasks           []*db.Task
	Webhooks        []*db.WebhookDelivery
	Settings        []*db.Setting
	MCPServers      []*db.MCPServer
	Users           []*db.User

	// Error injection: set these to force specific methods to return errors.
	ErrDefault error

	idCounter int
}

var _ db.Store = (*Store)(nil)

func New() *Store {
	return &Store{}
}

func (s *Store) nextID() string {
	s.idCounter++
	return "mock-" + time.Now().Format("20060102") + "-" + json.Number(json.Number(string(rune('0'+s.idCounter)))).String()
}

func (s *Store) Close() {}

func (s *Store) RunMigrations(_ context.Context, _ string) error {
	return s.ErrDefault
}

// --- SSH Keys ---

func (s *Store) CreateSSHKey(_ context.Context, k *db.SSHKey) error {
	if s.ErrDefault != nil {
		return s.ErrDefault
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	k.ID = s.nextID()
	k.CreatedAt = time.Now()
	s.SSHKeys = append(s.SSHKeys, k)
	return nil
}

func (s *Store) ListSSHKeys(_ context.Context) ([]*db.SSHKey, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.SSHKeys, s.ErrDefault
}

func (s *Store) GetSSHKey(_ context.Context, id string) (*db.SSHKey, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, k := range s.SSHKeys {
		if k.ID == id {
			return k, nil
		}
	}
	return nil, errNotFound("ssh_key", id)
}

func (s *Store) DeleteSSHKey(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, k := range s.SSHKeys {
		if k.ID == id {
			s.SSHKeys = append(s.SSHKeys[:i], s.SSHKeys[i+1:]...)
			return nil
		}
	}
	return nil
}

// --- Projects ---

func (s *Store) CreateProject(_ context.Context, p *db.Project) error {
	if s.ErrDefault != nil {
		return s.ErrDefault
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	p.ID = s.nextID()
	now := time.Now()
	p.CreatedAt = now
	p.UpdatedAt = now
	s.Projects = append(s.Projects, p)
	return nil
}

func (s *Store) ListProjects(_ context.Context) ([]*db.Project, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Projects, s.ErrDefault
}

func (s *Store) GetProject(_ context.Context, id string) (*db.Project, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, p := range s.Projects {
		if p.ID == id {
			return p, nil
		}
	}
	return nil, errNotFound("project", id)
}

func (s *Store) UpdateProject(_ context.Context, p *db.Project) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, existing := range s.Projects {
		if existing.ID == p.ID {
			p.UpdatedAt = time.Now()
			s.Projects[i] = p
			return nil
		}
	}
	return errNotFound("project", p.ID)
}

func (s *Store) DeleteProject(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, p := range s.Projects {
		if p.ID == id {
			s.Projects = append(s.Projects[:i], s.Projects[i+1:]...)
			return nil
		}
	}
	return nil
}

// --- Provider Configs ---

func (s *Store) CreateProviderConfig(_ context.Context, pc *db.ProviderConfig) error {
	if s.ErrDefault != nil {
		return s.ErrDefault
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	pc.ID = s.nextID()
	now := time.Now()
	pc.CreatedAt = now
	pc.UpdatedAt = now
	s.ProviderConfigs = append(s.ProviderConfigs, pc)
	return nil
}

func (s *Store) ListProviderConfigs(_ context.Context, projectID string) ([]*db.ProviderConfig, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var result []*db.ProviderConfig
	for _, pc := range s.ProviderConfigs {
		if pc.ProjectID == projectID {
			result = append(result, pc)
		}
	}
	return result, s.ErrDefault
}

func (s *Store) GetProviderConfig(_ context.Context, id string) (*db.ProviderConfig, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, pc := range s.ProviderConfigs {
		if pc.ID == id {
			return pc, nil
		}
	}
	return nil, errNotFound("provider_config", id)
}

func (s *Store) GetProviderConfigByPath(_ context.Context, path string) (*db.ProviderConfig, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, pc := range s.ProviderConfigs {
		if pc.WebhookPath == path {
			return pc, nil
		}
	}
	return nil, errNotFound("provider_config_path", path)
}

func (s *Store) ListAllProviderConfigs(_ context.Context) ([]*db.ProviderConfig, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var result []*db.ProviderConfig
	for _, pc := range s.ProviderConfigs {
		if pc.Enabled {
			result = append(result, pc)
		}
	}
	return result, s.ErrDefault
}

func (s *Store) DeleteProviderConfig(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, pc := range s.ProviderConfigs {
		if pc.ID == id {
			s.ProviderConfigs = append(s.ProviderConfigs[:i], s.ProviderConfigs[i+1:]...)
			return nil
		}
	}
	return nil
}

// --- Trigger Keywords ---

func (s *Store) SetTriggerKeywords(_ context.Context, projectID string, keywords []db.TriggerKeyword) error {
	if s.ErrDefault != nil {
		return s.ErrDefault
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	filtered := s.TriggerKeywords[:0]
	for _, kw := range s.TriggerKeywords {
		if kw.ProjectID != projectID {
			filtered = append(filtered, kw)
		}
	}
	s.TriggerKeywords = filtered

	for i := range keywords {
		kw := &keywords[i]
		kw.ProjectID = projectID
		kw.ID = s.nextID()
		kw.CreatedAt = time.Now()
		s.TriggerKeywords = append(s.TriggerKeywords, kw)
	}
	return nil
}

func (s *Store) GetTriggerKeywords(_ context.Context, projectID string) ([]*db.TriggerKeyword, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var result []*db.TriggerKeyword
	for _, kw := range s.TriggerKeywords {
		if kw.ProjectID == projectID {
			result = append(result, kw)
		}
	}
	return result, s.ErrDefault
}

// --- Tasks ---

func (s *Store) CreateTask(_ context.Context, t *db.Task) error {
	if s.ErrDefault != nil {
		return s.ErrDefault
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	t.ID = s.nextID()
	now := time.Now()
	t.CreatedAt = now
	t.UpdatedAt = now
	t.Status = db.TaskStatusPending
	s.Tasks = append(s.Tasks, t)
	return nil
}

func (s *Store) UpdateTaskStatus(_ context.Context, taskID string, status db.TaskStatus, result *string, errMsg *string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, t := range s.Tasks {
		if t.ID == taskID {
			t.Status = status
			t.Result = result
			t.ErrorMessage = errMsg
			t.UpdatedAt = time.Now()
			now := time.Now()
			if status == db.TaskStatusProcessing {
				t.StartedAt = &now
			}
			if status == db.TaskStatusCompleted || status == db.TaskStatusFailed {
				t.CompletedAt = &now
			}
			return nil
		}
	}
	return errNotFound("task", taskID)
}

func (s *Store) ListTasks(_ context.Context, limit, offset int) ([]*db.Task, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if offset >= len(s.Tasks) {
		return nil, s.ErrDefault
	}
	end := offset + limit
	if end > len(s.Tasks) {
		end = len(s.Tasks)
	}
	return s.Tasks[offset:end], s.ErrDefault
}

func (s *Store) GetTask(_ context.Context, id string) (*db.Task, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, t := range s.Tasks {
		if t.ID == id {
			return t, nil
		}
	}
	return nil, errNotFound("task", id)
}

func (s *Store) CountTasks(_ context.Context) (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.Tasks), s.ErrDefault
}

// --- Webhook Dedup ---

func (s *Store) IsWebhookProcessed(_ context.Context, eventUUID string) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, w := range s.Webhooks {
		if w.EventUUID == eventUUID {
			return true, nil
		}
	}
	return false, s.ErrDefault
}

func (s *Store) RecordWebhookDelivery(_ context.Context, delivery *db.WebhookDelivery) error {
	if s.ErrDefault != nil {
		return s.ErrDefault
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	delivery.ID = s.nextID()
	delivery.CreatedAt = time.Now()
	s.Webhooks = append(s.Webhooks, delivery)
	return nil
}

// --- Settings ---

func (s *Store) GetSetting(_ context.Context, key string) (*db.Setting, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, st := range s.Settings {
		if st.Key == key {
			return st, nil
		}
	}
	return nil, errNotFound("setting", key)
}

func (s *Store) GetSettingBool(_ context.Context, key string, fallback bool) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, st := range s.Settings {
		if st.Key == key {
			var v bool
			if json.Unmarshal(st.Value, &v) == nil {
				return v
			}
			return fallback
		}
	}
	return fallback
}

func (s *Store) GetSettingString(_ context.Context, key string, fallback string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, st := range s.Settings {
		if st.Key == key {
			var v string
			if json.Unmarshal(st.Value, &v) == nil {
				return v
			}
			return fallback
		}
	}
	return fallback
}

func (s *Store) GetSettingInt(_ context.Context, key string, fallback int) int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, st := range s.Settings {
		if st.Key == key {
			var v int
			if json.Unmarshal(st.Value, &v) == nil {
				return v
			}
			return fallback
		}
	}
	return fallback
}

func (s *Store) GetSettingDuration(_ context.Context, key string, fallback time.Duration) time.Duration {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, st := range s.Settings {
		if st.Key == key {
			var v string
			if json.Unmarshal(st.Value, &v) == nil {
				if d, err := time.ParseDuration(v); err == nil {
					return d
				}
			}
			return fallback
		}
	}
	return fallback
}

func (s *Store) SetSetting(_ context.Context, key string, value json.RawMessage) error {
	if s.ErrDefault != nil {
		return s.ErrDefault
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, st := range s.Settings {
		if st.Key == key {
			st.Value = value
			st.UpdatedAt = time.Now()
			return nil
		}
	}
	s.Settings = append(s.Settings, &db.Setting{
		Key:       key,
		Value:     value,
		UpdatedAt: time.Now(),
	})
	return nil
}

func (s *Store) ListSettings(_ context.Context) ([]*db.Setting, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Settings, s.ErrDefault
}

func (s *Store) DeleteSetting(_ context.Context, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, st := range s.Settings {
		if st.Key == key {
			s.Settings = append(s.Settings[:i], s.Settings[i+1:]...)
			return nil
		}
	}
	return nil
}

// --- MCP Servers ---

func (s *Store) CreateMCPServer(_ context.Context, m *db.MCPServer) error {
	if s.ErrDefault != nil {
		return s.ErrDefault
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	m.ID = s.nextID()
	now := time.Now()
	m.CreatedAt = now
	m.UpdatedAt = now
	s.MCPServers = append(s.MCPServers, m)
	return nil
}

func (s *Store) ListMCPServers(_ context.Context) ([]*db.MCPServer, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.MCPServers, s.ErrDefault
}

func (s *Store) ListEnabledMCPServers(_ context.Context) ([]*db.MCPServer, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var result []*db.MCPServer
	for _, m := range s.MCPServers {
		if m.Enabled && m.Status == db.MCPStatusInstalled {
			result = append(result, m)
		}
	}
	return result, s.ErrDefault
}

func (s *Store) GetMCPServer(_ context.Context, id string) (*db.MCPServer, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, m := range s.MCPServers {
		if m.ID == id {
			return m, nil
		}
	}
	return nil, errNotFound("mcp_server", id)
}

func (s *Store) UpdateMCPServerStatus(_ context.Context, id string, status db.MCPServerStatus, errMsg *string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, m := range s.MCPServers {
		if m.ID == id {
			m.Status = status
			m.ErrorMsg = errMsg
			m.UpdatedAt = time.Now()
			return nil
		}
	}
	return errNotFound("mcp_server", id)
}

func (s *Store) UpdateMCPServer(_ context.Context, m *db.MCPServer) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, existing := range s.MCPServers {
		if existing.ID == m.ID {
			m.UpdatedAt = time.Now()
			s.MCPServers[i] = m
			return nil
		}
	}
	return errNotFound("mcp_server", m.ID)
}

func (s *Store) DeleteMCPServer(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, m := range s.MCPServers {
		if m.ID == id {
			s.MCPServers = append(s.MCPServers[:i], s.MCPServers[i+1:]...)
			return nil
		}
	}
	return nil
}

// --- Users ---

func (s *Store) CreateUser(_ context.Context, u *db.User) error {
	if s.ErrDefault != nil {
		return s.ErrDefault
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	u.ID = s.nextID()
	now := time.Now()
	u.CreatedAt = now
	u.UpdatedAt = now
	s.Users = append(s.Users, u)
	return nil
}

func (s *Store) GetUserByUsername(_ context.Context, username string) (*db.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, u := range s.Users {
		if u.Username == username {
			return u, nil
		}
	}
	return nil, errNotFound("user", username)
}

func (s *Store) GetUser(_ context.Context, id string) (*db.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, u := range s.Users {
		if u.ID == id {
			return u, nil
		}
	}
	return nil, errNotFound("user", id)
}

func (s *Store) ListUsers(_ context.Context) ([]*db.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Users, s.ErrDefault
}

func (s *Store) UpdateUser(_ context.Context, u *db.User) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, existing := range s.Users {
		if existing.ID == u.ID {
			u.UpdatedAt = time.Now()
			u.PasswordHash = existing.PasswordHash
			s.Users[i] = u
			return nil
		}
	}
	return errNotFound("user", u.ID)
}

func (s *Store) UpdateUserPassword(_ context.Context, id string, hash string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, u := range s.Users {
		if u.ID == id {
			u.PasswordHash = hash
			u.UpdatedAt = time.Now()
			return nil
		}
	}
	return errNotFound("user", id)
}

func (s *Store) DeleteUser(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, u := range s.Users {
		if u.ID == id {
			s.Users = append(s.Users[:i], s.Users[i+1:]...)
			return nil
		}
	}
	return nil
}

func (s *Store) CountUsers(_ context.Context) (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.Users), s.ErrDefault
}

func errNotFound(entity, id string) error {
	return &NotFoundError{Entity: entity, ID: id}
}

type NotFoundError struct {
	Entity string
	ID     string
}

func (e *NotFoundError) Error() string {
	return e.Entity + " not found: " + e.ID
}
