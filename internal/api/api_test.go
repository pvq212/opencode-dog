package api

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/opencode-ai/opencode-dog/internal/auth"
	"github.com/opencode-ai/opencode-dog/internal/db"
	"github.com/opencode-ai/opencode-dog/internal/db/dbmock"
	"github.com/opencode-ai/opencode-dog/internal/mcpmgr"
)

// --- Test helpers ---

type testEnv struct {
	api   *API
	store *dbmock.Store
	auth  *auth.Auth
	mux   *http.ServeMux
}

func newTestEnv(t *testing.T) *testEnv {
	t.Helper()
	store := dbmock.New()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	a := auth.New(store, logger, "test-secret")
	mgr := mcpmgr.New(store, logger)
	api := New(store, a, mgr, logger)
	mux := http.NewServeMux()
	api.RegisterRoutes(mux)
	return &testEnv{api: api, store: store, auth: a, mux: mux}
}

func seedUser(t *testing.T, store *dbmock.Store, username, password, role string) *db.User {
	t.Helper()
	hash, err := auth.HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	u := &db.User{
		Username:     username,
		PasswordHash: hash,
		DisplayName:  username,
		Role:         role,
		Enabled:      true,
	}
	if err := store.CreateUser(context.Background(), u); err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	return u
}

func loginToken(t *testing.T, env *testEnv, username, password string) string {
	t.Helper()
	token, _, err := env.auth.Login(context.Background(), username, password)
	if err != nil {
		t.Fatalf("Login(%s): %v", username, err)
	}
	return token
}

func doRequest(env *testEnv, method, path string, body io.Reader, token string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, body)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	rec := httptest.NewRecorder()
	env.mux.ServeHTTP(rec, req)
	return rec
}

func jsonBody(v any) *bytes.Buffer {
	b, _ := json.Marshal(v)
	return bytes.NewBuffer(b)
}

func decodeJSON(t *testing.T, rec *httptest.ResponseRecorder, v any) {
	t.Helper()
	if err := json.NewDecoder(rec.Body).Decode(v); err != nil {
		t.Fatalf("decode response: %v (body: %s)", err, rec.Body.String())
	}
}

// --- Login handler ---

func TestLoginSuccess(t *testing.T) {
	env := newTestEnv(t)
	seedUser(t, env.store, "alice", "pass123", db.RoleAdmin)

	rec := doRequest(env, http.MethodPost, "/api/auth/login",
		jsonBody(map[string]string{"username": "alice", "password": "pass123"}), "")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp map[string]any
	decodeJSON(t, rec, &resp)
	if resp["token"] == nil || resp["token"] == "" {
		t.Fatal("expected token in response")
	}
}

func TestLoginWrongMethod(t *testing.T) {
	env := newTestEnv(t)
	rec := doRequest(env, http.MethodGet, "/api/auth/login", nil, "")
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rec.Code)
	}
}

func TestLoginBadJSON(t *testing.T) {
	env := newTestEnv(t)
	rec := doRequest(env, http.MethodPost, "/api/auth/login",
		bytes.NewBufferString("{invalid"), "")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestLoginInvalidCredentials(t *testing.T) {
	env := newTestEnv(t)
	seedUser(t, env.store, "bob", "secret", db.RoleViewer)

	rec := doRequest(env, http.MethodPost, "/api/auth/login",
		jsonBody(map[string]string{"username": "bob", "password": "wrong"}), "")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

// --- Me handler ---

func TestMeAuthenticated(t *testing.T) {
	env := newTestEnv(t)
	u := seedUser(t, env.store, "alice", "pass", db.RoleAdmin)
	token := loginToken(t, env, "alice", "pass")

	rec := doRequest(env, http.MethodGet, "/api/auth/me", nil, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp db.User
	decodeJSON(t, rec, &resp)
	if resp.Username != u.Username {
		t.Fatalf("expected username %q, got %q", u.Username, resp.Username)
	}
}

func TestMeUnauthenticated(t *testing.T) {
	env := newTestEnv(t)
	rec := doRequest(env, http.MethodGet, "/api/auth/me", nil, "")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

// --- Change password ---

func TestChangePasswordSuccess(t *testing.T) {
	env := newTestEnv(t)
	seedUser(t, env.store, "alice", "oldpass", db.RoleAdmin)
	token := loginToken(t, env, "alice", "oldpass")

	rec := doRequest(env, http.MethodPut, "/api/auth/password",
		jsonBody(map[string]string{"old_password": "oldpass", "new_password": "newpass"}), token)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	_, _, err := env.auth.Login(context.Background(), "alice", "newpass")
	if err != nil {
		t.Fatalf("login with new password failed: %v", err)
	}
}

func TestChangePasswordWrongOld(t *testing.T) {
	env := newTestEnv(t)
	seedUser(t, env.store, "alice", "real", db.RoleAdmin)
	token := loginToken(t, env, "alice", "real")

	rec := doRequest(env, http.MethodPut, "/api/auth/password",
		jsonBody(map[string]string{"old_password": "wrong", "new_password": "new"}), token)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestChangePasswordWrongMethod(t *testing.T) {
	env := newTestEnv(t)
	seedUser(t, env.store, "alice", "pass", db.RoleAdmin)
	token := loginToken(t, env, "alice", "pass")

	rec := doRequest(env, http.MethodGet, "/api/auth/password", nil, token)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rec.Code)
	}
}

// --- Projects ---

func TestProjectsList(t *testing.T) {
	env := newTestEnv(t)
	seedUser(t, env.store, "admin", "pass", db.RoleAdmin)
	token := loginToken(t, env, "admin", "pass")

	env.store.Projects = []*db.Project{
		{ID: "p1", Name: "proj1", SSHURL: "git@example.com:a.git", DefaultBranch: "main", Enabled: true},
	}

	rec := doRequest(env, http.MethodGet, "/api/projects", nil, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var projects []db.Project
	decodeJSON(t, rec, &projects)
	if len(projects) != 1 {
		t.Fatalf("expected 1 project, got %d", len(projects))
	}
}

func TestProjectsListEmpty(t *testing.T) {
	env := newTestEnv(t)
	seedUser(t, env.store, "admin", "pass", db.RoleAdmin)
	token := loginToken(t, env, "admin", "pass")

	rec := doRequest(env, http.MethodGet, "/api/projects", nil, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var projects []db.Project
	decodeJSON(t, rec, &projects)
	if len(projects) != 0 {
		t.Fatalf("expected 0 projects, got %d", len(projects))
	}
}

func TestProjectsCreateAdmin(t *testing.T) {
	env := newTestEnv(t)
	seedUser(t, env.store, "admin", "pass", db.RoleAdmin)
	token := loginToken(t, env, "admin", "pass")

	body := jsonBody(map[string]string{"name": "new-proj", "ssh_url": "git@example.com:new.git"})
	rec := doRequest(env, http.MethodPost, "/api/projects", body, token)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	var p db.Project
	decodeJSON(t, rec, &p)
	if p.Name != "new-proj" {
		t.Fatalf("expected name 'new-proj', got %q", p.Name)
	}
	if !p.Enabled {
		t.Fatal("new project should be enabled")
	}
}

func TestProjectsCreateViewerForbidden(t *testing.T) {
	env := newTestEnv(t)
	seedUser(t, env.store, "viewer", "pass", db.RoleViewer)
	token := loginToken(t, env, "viewer", "pass")

	body := jsonBody(map[string]string{"name": "x", "ssh_url": "git@x.git"})
	rec := doRequest(env, http.MethodPost, "/api/projects", body, token)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}
}

func TestProjectsCreateMissingFields(t *testing.T) {
	env := newTestEnv(t)
	seedUser(t, env.store, "admin", "pass", db.RoleAdmin)
	token := loginToken(t, env, "admin", "pass")

	rec := doRequest(env, http.MethodPost, "/api/projects", jsonBody(map[string]string{"name": "x"}), token)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestProjectsCreateBadJSON(t *testing.T) {
	env := newTestEnv(t)
	seedUser(t, env.store, "admin", "pass", db.RoleAdmin)
	token := loginToken(t, env, "admin", "pass")

	rec := doRequest(env, http.MethodPost, "/api/projects", bytes.NewBufferString("{bad"), token)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestProjectDetailGet(t *testing.T) {
	env := newTestEnv(t)
	seedUser(t, env.store, "admin", "pass", db.RoleAdmin)
	token := loginToken(t, env, "admin", "pass")

	env.store.Projects = []*db.Project{
		{ID: "p1", Name: "proj1", SSHURL: "git@example.com:a.git", DefaultBranch: "main"},
	}

	rec := doRequest(env, http.MethodGet, "/api/projects/p1", nil, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestProjectDetailNotFound(t *testing.T) {
	env := newTestEnv(t)
	seedUser(t, env.store, "admin", "pass", db.RoleAdmin)
	token := loginToken(t, env, "admin", "pass")

	rec := doRequest(env, http.MethodGet, "/api/projects/nonexistent", nil, token)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestProjectDetailUpdate(t *testing.T) {
	env := newTestEnv(t)
	seedUser(t, env.store, "admin", "pass", db.RoleAdmin)
	token := loginToken(t, env, "admin", "pass")

	env.store.Projects = []*db.Project{
		{ID: "p1", Name: "old", SSHURL: "git@example.com:old.git", DefaultBranch: "main"},
	}

	body := jsonBody(map[string]string{"name": "updated", "ssh_url": "git@example.com:new.git", "default_branch": "develop"})
	rec := doRequest(env, http.MethodPut, "/api/projects/p1", body, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestProjectDetailDeleteAdmin(t *testing.T) {
	env := newTestEnv(t)
	seedUser(t, env.store, "admin", "pass", db.RoleAdmin)
	token := loginToken(t, env, "admin", "pass")

	env.store.Projects = []*db.Project{
		{ID: "p1", Name: "proj1", SSHURL: "git@example.com:a.git"},
	}

	rec := doRequest(env, http.MethodDelete, "/api/projects/p1", nil, token)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rec.Code)
	}
}

func TestProjectDetailDeleteEditorForbidden(t *testing.T) {
	env := newTestEnv(t)
	seedUser(t, env.store, "editor", "pass", db.RoleEditor)
	token := loginToken(t, env, "editor", "pass")

	rec := doRequest(env, http.MethodDelete, "/api/projects/p1", nil, token)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}
}

func TestProjectDetailMissingID(t *testing.T) {
	env := newTestEnv(t)
	seedUser(t, env.store, "admin", "pass", db.RoleAdmin)
	token := loginToken(t, env, "admin", "pass")

	rec := doRequest(env, http.MethodGet, "/api/projects/", nil, token)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

// --- SSH Keys ---

func TestSSHKeysList(t *testing.T) {
	env := newTestEnv(t)
	seedUser(t, env.store, "admin", "pass", db.RoleAdmin)
	token := loginToken(t, env, "admin", "pass")

	env.store.SSHKeys = []*db.SSHKey{
		{ID: "k1", Name: "my-key", PrivateKey: "SECRET", PublicKey: "ssh-rsa AAA"},
	}

	rec := doRequest(env, http.MethodGet, "/api/ssh-keys", nil, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var keys []db.SSHKey
	decodeJSON(t, rec, &keys)
	if len(keys) != 1 {
		t.Fatalf("expected 1 key, got %d", len(keys))
	}
	if keys[0].PrivateKey != "" {
		t.Fatal("private key should be stripped from list response")
	}
}

func TestSSHKeysCreate(t *testing.T) {
	env := newTestEnv(t)
	seedUser(t, env.store, "admin", "pass", db.RoleAdmin)
	token := loginToken(t, env, "admin", "pass")

	body := jsonBody(map[string]string{"name": "new-key", "private_key": "secret-key"})
	rec := doRequest(env, http.MethodPost, "/api/ssh-keys", body, token)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	var k db.SSHKey
	decodeJSON(t, rec, &k)
	if k.PrivateKey != "" {
		t.Fatal("private key should be stripped from create response")
	}
}

func TestSSHKeysCreateMissingFields(t *testing.T) {
	env := newTestEnv(t)
	seedUser(t, env.store, "admin", "pass", db.RoleAdmin)
	token := loginToken(t, env, "admin", "pass")

	rec := doRequest(env, http.MethodPost, "/api/ssh-keys", jsonBody(map[string]string{"name": "x"}), token)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestSSHKeysCreateViewerForbidden(t *testing.T) {
	env := newTestEnv(t)
	seedUser(t, env.store, "viewer", "pass", db.RoleViewer)
	token := loginToken(t, env, "viewer", "pass")

	rec := doRequest(env, http.MethodPost, "/api/ssh-keys",
		jsonBody(map[string]string{"name": "k", "private_key": "s"}), token)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}
}

func TestSSHKeyDelete(t *testing.T) {
	env := newTestEnv(t)
	seedUser(t, env.store, "admin", "pass", db.RoleAdmin)
	token := loginToken(t, env, "admin", "pass")

	env.store.SSHKeys = []*db.SSHKey{{ID: "k1", Name: "key"}}

	rec := doRequest(env, http.MethodDelete, "/api/ssh-keys/k1", nil, token)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rec.Code)
	}
}

func TestSSHKeyDeleteViewerForbidden(t *testing.T) {
	env := newTestEnv(t)
	seedUser(t, env.store, "viewer", "pass", db.RoleViewer)
	token := loginToken(t, env, "viewer", "pass")

	rec := doRequest(env, http.MethodDelete, "/api/ssh-keys/k1", nil, token)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}
}

// --- Providers ---

func TestProvidersList(t *testing.T) {
	env := newTestEnv(t)
	seedUser(t, env.store, "admin", "pass", db.RoleAdmin)
	token := loginToken(t, env, "admin", "pass")

	env.store.ProviderConfigs = []*db.ProviderConfig{
		{ID: "pc1", ProjectID: "p1", ProviderType: "gitlab", Enabled: true},
	}

	rec := doRequest(env, http.MethodGet, "/api/providers/p1", nil, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var configs []db.ProviderConfig
	decodeJSON(t, rec, &configs)
	if len(configs) != 1 {
		t.Fatalf("expected 1 config, got %d", len(configs))
	}
}

func TestProvidersCreate(t *testing.T) {
	env := newTestEnv(t)
	seedUser(t, env.store, "admin", "pass", db.RoleAdmin)
	token := loginToken(t, env, "admin", "pass")

	body := jsonBody(map[string]string{"provider_type": "slack"})
	rec := doRequest(env, http.MethodPost, "/api/providers/project123", body, token)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	var pc db.ProviderConfig
	decodeJSON(t, rec, &pc)
	if !pc.Enabled {
		t.Fatal("new provider should be enabled")
	}
}

func TestProvidersCreateViewerForbidden(t *testing.T) {
	env := newTestEnv(t)
	seedUser(t, env.store, "viewer", "pass", db.RoleViewer)
	token := loginToken(t, env, "viewer", "pass")

	body := jsonBody(map[string]string{"provider_type": "slack"})
	rec := doRequest(env, http.MethodPost, "/api/providers/p1", body, token)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}
}

func TestProvidersDelete(t *testing.T) {
	env := newTestEnv(t)
	seedUser(t, env.store, "admin", "pass", db.RoleAdmin)
	token := loginToken(t, env, "admin", "pass")

	env.store.ProviderConfigs = []*db.ProviderConfig{
		{ID: "pc1", ProjectID: "p1", ProviderType: "gitlab"},
	}

	rec := doRequest(env, http.MethodDelete, "/api/providers/p1/pc1", nil, token)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestProvidersDeleteMissingID(t *testing.T) {
	env := newTestEnv(t)
	seedUser(t, env.store, "admin", "pass", db.RoleAdmin)
	token := loginToken(t, env, "admin", "pass")

	rec := doRequest(env, http.MethodDelete, "/api/providers/p1", nil, token)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

// --- Keywords ---

func TestKeywordsList(t *testing.T) {
	env := newTestEnv(t)
	seedUser(t, env.store, "admin", "pass", db.RoleAdmin)
	token := loginToken(t, env, "admin", "pass")

	env.store.TriggerKeywords = []*db.TriggerKeyword{
		{ID: "kw1", ProjectID: "p1", Mode: "ask", Keyword: "@opencode"},
	}

	rec := doRequest(env, http.MethodGet, "/api/keywords/p1", nil, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestKeywordsUpdate(t *testing.T) {
	env := newTestEnv(t)
	seedUser(t, env.store, "admin", "pass", db.RoleAdmin)
	token := loginToken(t, env, "admin", "pass")

	keywords := []map[string]string{
		{"mode": "ask", "keyword": "@opencode"},
		{"mode": "do", "keyword": "@do"},
	}
	rec := doRequest(env, http.MethodPut, "/api/keywords/p1", jsonBody(keywords), token)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestKeywordsUpdateViewerForbidden(t *testing.T) {
	env := newTestEnv(t)
	seedUser(t, env.store, "viewer", "pass", db.RoleViewer)
	token := loginToken(t, env, "viewer", "pass")

	rec := doRequest(env, http.MethodPut, "/api/keywords/p1",
		jsonBody([]map[string]string{{"mode": "ask", "keyword": "@x"}}), token)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}
}

// --- Tasks ---

func TestTasksList(t *testing.T) {
	env := newTestEnv(t)
	seedUser(t, env.store, "admin", "pass", db.RoleAdmin)
	token := loginToken(t, env, "admin", "pass")

	now := time.Now()
	env.store.Tasks = []*db.Task{
		{ID: "t1", Title: "task1", Status: db.TaskStatusPending, CreatedAt: now},
		{ID: "t2", Title: "task2", Status: db.TaskStatusCompleted, CreatedAt: now},
	}

	rec := doRequest(env, http.MethodGet, "/api/tasks", nil, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var resp map[string]any
	decodeJSON(t, rec, &resp)
	tasks := resp["tasks"].([]any)
	if len(tasks) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(tasks))
	}
	total := resp["total"].(float64)
	if total != 2 {
		t.Fatalf("expected total 2, got %v", total)
	}
}

func TestTasksListWithPagination(t *testing.T) {
	env := newTestEnv(t)
	seedUser(t, env.store, "admin", "pass", db.RoleAdmin)
	token := loginToken(t, env, "admin", "pass")

	now := time.Now()
	for i := 0; i < 5; i++ {
		env.store.Tasks = append(env.store.Tasks, &db.Task{
			ID: "t" + string(rune('0'+i)), Title: "task", Status: db.TaskStatusPending, CreatedAt: now,
		})
	}

	rec := doRequest(env, http.MethodGet, "/api/tasks?limit=2&offset=1", nil, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestTasksWrongMethod(t *testing.T) {
	env := newTestEnv(t)
	seedUser(t, env.store, "admin", "pass", db.RoleAdmin)
	token := loginToken(t, env, "admin", "pass")

	rec := doRequest(env, http.MethodPost, "/api/tasks", jsonBody(map[string]string{}), token)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rec.Code)
	}
}

func TestTaskDetail(t *testing.T) {
	env := newTestEnv(t)
	seedUser(t, env.store, "admin", "pass", db.RoleAdmin)
	token := loginToken(t, env, "admin", "pass")

	env.store.Tasks = []*db.Task{
		{ID: "t1", Title: "task1", Status: db.TaskStatusCompleted, CreatedAt: time.Now()},
	}

	rec := doRequest(env, http.MethodGet, "/api/tasks/t1", nil, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestTaskDetailNotFound(t *testing.T) {
	env := newTestEnv(t)
	seedUser(t, env.store, "admin", "pass", db.RoleAdmin)
	token := loginToken(t, env, "admin", "pass")

	rec := doRequest(env, http.MethodGet, "/api/tasks/nonexistent", nil, token)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

// --- Settings ---

func TestSettingsList(t *testing.T) {
	env := newTestEnv(t)
	seedUser(t, env.store, "admin", "pass", db.RoleAdmin)
	token := loginToken(t, env, "admin", "pass")

	env.store.Settings = []*db.Setting{
		{Key: "theme", Value: json.RawMessage(`"dark"`)},
	}

	rec := doRequest(env, http.MethodGet, "/api/settings", nil, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestSettingsUpsert(t *testing.T) {
	env := newTestEnv(t)
	seedUser(t, env.store, "admin", "pass", db.RoleAdmin)
	token := loginToken(t, env, "admin", "pass")

	body := jsonBody(map[string]any{"key": "theme", "value": "dark"})
	rec := doRequest(env, http.MethodPut, "/api/settings", body, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestSettingsUpsertMissingKey(t *testing.T) {
	env := newTestEnv(t)
	seedUser(t, env.store, "admin", "pass", db.RoleAdmin)
	token := loginToken(t, env, "admin", "pass")

	body := jsonBody(map[string]any{"value": "dark"})
	rec := doRequest(env, http.MethodPut, "/api/settings", body, token)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestSettingsUpsertViewerForbidden(t *testing.T) {
	env := newTestEnv(t)
	seedUser(t, env.store, "viewer", "pass", db.RoleViewer)
	token := loginToken(t, env, "viewer", "pass")

	body := jsonBody(map[string]any{"key": "theme", "value": "dark"})
	rec := doRequest(env, http.MethodPut, "/api/settings", body, token)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}
}

func TestSettingDetailGet(t *testing.T) {
	env := newTestEnv(t)
	seedUser(t, env.store, "admin", "pass", db.RoleAdmin)
	token := loginToken(t, env, "admin", "pass")

	env.store.Settings = []*db.Setting{
		{Key: "theme", Value: json.RawMessage(`"dark"`)},
	}

	rec := doRequest(env, http.MethodGet, "/api/settings/theme", nil, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestSettingDetailNotFound(t *testing.T) {
	env := newTestEnv(t)
	seedUser(t, env.store, "admin", "pass", db.RoleAdmin)
	token := loginToken(t, env, "admin", "pass")

	rec := doRequest(env, http.MethodGet, "/api/settings/nonexistent", nil, token)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestSettingDetailDelete(t *testing.T) {
	env := newTestEnv(t)
	seedUser(t, env.store, "admin", "pass", db.RoleAdmin)
	token := loginToken(t, env, "admin", "pass")

	env.store.Settings = []*db.Setting{
		{Key: "theme", Value: json.RawMessage(`"dark"`)},
	}

	rec := doRequest(env, http.MethodDelete, "/api/settings/theme", nil, token)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rec.Code)
	}
}

func TestSettingDetailDeleteViewerForbidden(t *testing.T) {
	env := newTestEnv(t)
	seedUser(t, env.store, "viewer", "pass", db.RoleViewer)
	token := loginToken(t, env, "viewer", "pass")

	rec := doRequest(env, http.MethodDelete, "/api/settings/theme", nil, token)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}
}

// --- MCP Servers ---

func TestMCPServersList(t *testing.T) {
	env := newTestEnv(t)
	seedUser(t, env.store, "admin", "pass", db.RoleAdmin)
	token := loginToken(t, env, "admin", "pass")

	env.store.MCPServers = []*db.MCPServer{
		{ID: "m1", Name: "test-mcp", Package: "@test/mcp", Type: "npm", Enabled: true,
			Args: json.RawMessage("[]"), Env: json.RawMessage("{}")},
	}

	rec := doRequest(env, http.MethodGet, "/api/mcp-servers", nil, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestMCPServersCreate(t *testing.T) {
	env := newTestEnv(t)
	seedUser(t, env.store, "admin", "pass", db.RoleAdmin)
	token := loginToken(t, env, "admin", "pass")

	body := jsonBody(map[string]string{"name": "test-mcp", "package": "@test/mcp"})
	rec := doRequest(env, http.MethodPost, "/api/mcp-servers", body, token)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	var m db.MCPServer
	decodeJSON(t, rec, &m)
	if m.Type != "npm" {
		t.Fatalf("expected default type 'npm', got %q", m.Type)
	}
	if !m.Enabled {
		t.Fatal("new MCP server should be enabled")
	}
}

func TestMCPServersCreateMissingFields(t *testing.T) {
	env := newTestEnv(t)
	seedUser(t, env.store, "admin", "pass", db.RoleAdmin)
	token := loginToken(t, env, "admin", "pass")

	rec := doRequest(env, http.MethodPost, "/api/mcp-servers",
		jsonBody(map[string]string{"name": "x"}), token)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestMCPServersCreateViewerForbidden(t *testing.T) {
	env := newTestEnv(t)
	seedUser(t, env.store, "viewer", "pass", db.RoleViewer)
	token := loginToken(t, env, "viewer", "pass")

	body := jsonBody(map[string]string{"name": "x", "package": "y"})
	rec := doRequest(env, http.MethodPost, "/api/mcp-servers", body, token)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}
}

func TestMCPServerDetailGet(t *testing.T) {
	env := newTestEnv(t)
	seedUser(t, env.store, "admin", "pass", db.RoleAdmin)
	token := loginToken(t, env, "admin", "pass")

	env.store.MCPServers = []*db.MCPServer{
		{ID: "m1", Name: "test", Package: "pkg", Type: "npm",
			Args: json.RawMessage("[]"), Env: json.RawMessage("{}")},
	}

	rec := doRequest(env, http.MethodGet, "/api/mcp-servers/m1", nil, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestMCPServerDetailNotFound(t *testing.T) {
	env := newTestEnv(t)
	seedUser(t, env.store, "admin", "pass", db.RoleAdmin)
	token := loginToken(t, env, "admin", "pass")

	rec := doRequest(env, http.MethodGet, "/api/mcp-servers/nonexistent", nil, token)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestMCPServerDetailUpdate(t *testing.T) {
	env := newTestEnv(t)
	seedUser(t, env.store, "admin", "pass", db.RoleAdmin)
	token := loginToken(t, env, "admin", "pass")

	env.store.MCPServers = []*db.MCPServer{
		{ID: "m1", Name: "test", Package: "pkg", Type: "npm",
			Args: json.RawMessage("[]"), Env: json.RawMessage("{}")},
	}

	body := jsonBody(map[string]any{"name": "updated", "enabled": false})
	rec := doRequest(env, http.MethodPut, "/api/mcp-servers/m1", body, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestMCPServerDetailDelete(t *testing.T) {
	env := newTestEnv(t)
	seedUser(t, env.store, "admin", "pass", db.RoleAdmin)
	token := loginToken(t, env, "admin", "pass")

	env.store.MCPServers = []*db.MCPServer{
		{ID: "m1", Name: "test", Package: "pkg", Type: "npm",
			Args: json.RawMessage("[]"), Env: json.RawMessage("{}")},
	}

	rec := doRequest(env, http.MethodDelete, "/api/mcp-servers/m1", nil, token)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rec.Code)
	}
}

func TestMCPServerInstall(t *testing.T) {
	env := newTestEnv(t)
	seedUser(t, env.store, "admin", "pass", db.RoleAdmin)
	token := loginToken(t, env, "admin", "pass")

	env.store.MCPServers = []*db.MCPServer{
		{ID: "m1", Name: "test", Package: "pkg", Type: "npm",
			Args: json.RawMessage("[]"), Env: json.RawMessage("{}")},
	}

	rec := doRequest(env, http.MethodPost, "/api/mcp-servers/m1/install", nil, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	// allow goroutine to start and fail gracefully
	time.Sleep(50 * time.Millisecond)
}

// --- Users ---

func TestUsersList(t *testing.T) {
	env := newTestEnv(t)
	seedUser(t, env.store, "admin", "pass", db.RoleAdmin)
	token := loginToken(t, env, "admin", "pass")

	rec := doRequest(env, http.MethodGet, "/api/users", nil, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var users []db.User
	decodeJSON(t, rec, &users)
	if len(users) != 1 {
		t.Fatalf("expected 1 user (seeded admin), got %d", len(users))
	}
}

func TestUsersListViewerForbidden(t *testing.T) {
	env := newTestEnv(t)
	seedUser(t, env.store, "viewer", "pass", db.RoleViewer)
	token := loginToken(t, env, "viewer", "pass")

	rec := doRequest(env, http.MethodGet, "/api/users", nil, token)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}
}

func TestUsersCreate(t *testing.T) {
	env := newTestEnv(t)
	seedUser(t, env.store, "admin", "pass", db.RoleAdmin)
	token := loginToken(t, env, "admin", "pass")

	body := jsonBody(map[string]string{"username": "newuser", "password": "newpass", "role": "editor"})
	rec := doRequest(env, http.MethodPost, "/api/users", body, token)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestUsersCreateMissingFields(t *testing.T) {
	env := newTestEnv(t)
	seedUser(t, env.store, "admin", "pass", db.RoleAdmin)
	token := loginToken(t, env, "admin", "pass")

	rec := doRequest(env, http.MethodPost, "/api/users",
		jsonBody(map[string]string{"username": "x"}), token)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestUsersCreateDefaultRole(t *testing.T) {
	env := newTestEnv(t)
	seedUser(t, env.store, "admin", "pass", db.RoleAdmin)
	token := loginToken(t, env, "admin", "pass")

	body := jsonBody(map[string]string{"username": "newuser", "password": "pass"})
	rec := doRequest(env, http.MethodPost, "/api/users", body, token)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	var u db.User
	decodeJSON(t, rec, &u)
	if u.Role != db.RoleViewer {
		t.Fatalf("expected default role 'viewer', got %q", u.Role)
	}
}

func TestUserDetailGet(t *testing.T) {
	env := newTestEnv(t)
	u := seedUser(t, env.store, "admin", "pass", db.RoleAdmin)
	token := loginToken(t, env, "admin", "pass")

	rec := doRequest(env, http.MethodGet, "/api/users/"+u.ID, nil, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestUserDetailNotFound(t *testing.T) {
	env := newTestEnv(t)
	seedUser(t, env.store, "admin", "pass", db.RoleAdmin)
	token := loginToken(t, env, "admin", "pass")

	rec := doRequest(env, http.MethodGet, "/api/users/nonexistent", nil, token)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestUserDetailUpdate(t *testing.T) {
	env := newTestEnv(t)
	u := seedUser(t, env.store, "admin", "pass", db.RoleAdmin)
	token := loginToken(t, env, "admin", "pass")

	body := jsonBody(map[string]string{"display_name": "Updated Name", "role": "editor"})
	rec := doRequest(env, http.MethodPut, "/api/users/"+u.ID, body, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestUserDetailDelete(t *testing.T) {
	env := newTestEnv(t)
	seedUser(t, env.store, "admin", "pass", db.RoleAdmin)
	target := seedUser(t, env.store, "target", "pass", db.RoleViewer)
	token := loginToken(t, env, "admin", "pass")

	rec := doRequest(env, http.MethodDelete, "/api/users/"+target.ID, nil, token)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rec.Code)
	}
}

func TestUserDetailDeleteViewerForbidden(t *testing.T) {
	env := newTestEnv(t)
	seedUser(t, env.store, "viewer", "pass", db.RoleViewer)
	token := loginToken(t, env, "viewer", "pass")

	rec := doRequest(env, http.MethodDelete, "/api/users/some-id", nil, token)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}
}

// --- Unauthenticated access to protected routes ---

func TestProtectedRoutesRequireAuth(t *testing.T) {
	env := newTestEnv(t)

	routes := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/projects"},
		{http.MethodGet, "/api/tasks"},
		{http.MethodGet, "/api/settings"},
		{http.MethodGet, "/api/users"},
		{http.MethodGet, "/api/ssh-keys"},
		{http.MethodGet, "/api/mcp-servers"},
	}

	for _, r := range routes {
		rec := doRequest(env, r.method, r.path, nil, "")
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("%s %s: expected 401, got %d", r.method, r.path, rec.Code)
		}
	}
}

// --- Method not allowed checks ---

func TestMethodNotAllowed(t *testing.T) {
	env := newTestEnv(t)
	seedUser(t, env.store, "admin", "pass", db.RoleAdmin)
	token := loginToken(t, env, "admin", "pass")

	cases := []struct {
		method string
		path   string
	}{
		{http.MethodDelete, "/api/projects"},
		{http.MethodDelete, "/api/ssh-keys"},
		{http.MethodPut, "/api/tasks"},
		{http.MethodPost, "/api/settings"},
	}

	for _, c := range cases {
		rec := doRequest(env, c.method, c.path, nil, token)
		if rec.Code != http.StatusMethodNotAllowed {
			t.Errorf("%s %s: expected 405, got %d", c.method, c.path, rec.Code)
		}
	}
}
