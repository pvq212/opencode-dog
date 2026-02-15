package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/opencode-ai/opencode-dog/internal/auth"
	"github.com/opencode-ai/opencode-dog/internal/db"
	"github.com/opencode-ai/opencode-dog/internal/mcpmgr"
)

type API struct {
	database *db.DB
	auth     *auth.Auth
	mcpMgr   *mcpmgr.Manager
	logger   *slog.Logger
}

func New(database *db.DB, a *auth.Auth, mcpMgr *mcpmgr.Manager, logger *slog.Logger) *API {
	return &API{database: database, auth: a, mcpMgr: mcpMgr, logger: logger}
}

func (a *API) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/auth/login", a.handleLogin)

	protected := http.NewServeMux()
	protected.HandleFunc("/api/auth/me", a.handleMe)
	protected.HandleFunc("/api/auth/password", a.handleChangePassword)

	protected.HandleFunc("/api/projects", a.handleProjects)
	protected.HandleFunc("/api/projects/", a.handleProjectDetail)
	protected.HandleFunc("/api/ssh-keys", a.handleSSHKeys)
	protected.HandleFunc("/api/ssh-keys/", a.handleSSHKeyDetail)
	protected.HandleFunc("/api/providers/", a.handleProviders)
	protected.HandleFunc("/api/keywords/", a.handleKeywords)
	protected.HandleFunc("/api/tasks", a.handleTasks)
	protected.HandleFunc("/api/tasks/", a.handleTaskDetail)

	protected.HandleFunc("/api/settings", a.handleSettings)
	protected.HandleFunc("/api/settings/", a.handleSettingDetail)
	protected.HandleFunc("/api/mcp-servers", a.handleMCPServers)
	protected.HandleFunc("/api/mcp-servers/", a.handleMCPServerDetail)
	protected.HandleFunc("/api/users", a.handleUsers)
	protected.HandleFunc("/api/users/", a.handleUserDetail)

	mux.Handle("/api/", a.auth.Middleware(protected))
}

func (a *API) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}

	token, user, err := a.auth.Login(r.Context(), req.Username, req.Password)
	if err != nil {
		writeErr(w, http.StatusUnauthorized, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"token": token,
		"user":  user,
	})
}

func (a *API) handleMe(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetUser(r.Context())
	if claims == nil {
		writeErr(w, http.StatusUnauthorized, "not authenticated")
		return
	}
	user, err := a.database.GetUser(r.Context(), claims.UserID)
	if err != nil {
		writeErr(w, http.StatusNotFound, "user not found")
		return
	}
	writeJSON(w, http.StatusOK, user)
}

func (a *API) handleChangePassword(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	claims := auth.GetUser(r.Context())
	if claims == nil {
		writeErr(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	var req struct {
		OldPassword string `json:"old_password"`
		NewPassword string `json:"new_password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}

	user, err := a.database.GetUser(r.Context(), claims.UserID)
	if err != nil {
		writeErr(w, http.StatusNotFound, "user not found")
		return
	}
	if !auth.CheckPassword(user.PasswordHash, req.OldPassword) {
		writeErr(w, http.StatusBadRequest, "old password incorrect")
		return
	}

	hash, err := auth.HashPassword(req.NewPassword)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "hash failed")
		return
	}
	if err := a.database.UpdateUserPassword(r.Context(), claims.UserID, hash); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (a *API) handleProjects(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		projects, err := a.database.ListProjects(r.Context())
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		if projects == nil {
			projects = []*db.Project{}
		}
		writeJSON(w, http.StatusOK, projects)

	case http.MethodPost:
		if !a.requireRole(w, r, db.RoleAdmin, db.RoleEditor) {
			return
		}
		var p db.Project
		if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid json")
			return
		}
		if p.Name == "" || p.SSHURL == "" {
			writeErr(w, http.StatusBadRequest, "name and ssh_url required")
			return
		}
		if p.DefaultBranch == "" {
			p.DefaultBranch = a.database.GetSettingString(r.Context(), "default_git_branch", "main")
		}
		p.Enabled = true
		if err := a.database.CreateProject(r.Context(), &p); err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, p)

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (a *API) handleProjectDetail(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/projects/")
	if id == "" {
		writeErr(w, http.StatusBadRequest, "missing project id")
		return
	}

	switch r.Method {
	case http.MethodGet:
		p, err := a.database.GetProject(r.Context(), id)
		if err != nil {
			writeErr(w, http.StatusNotFound, "project not found")
			return
		}
		writeJSON(w, http.StatusOK, p)

	case http.MethodPut:
		if !a.requireRole(w, r, db.RoleAdmin, db.RoleEditor) {
			return
		}
		var p db.Project
		if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid json")
			return
		}
		p.ID = id
		if err := a.database.UpdateProject(r.Context(), &p); err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, p)

	case http.MethodDelete:
		if !a.requireRole(w, r, db.RoleAdmin) {
			return
		}
		if err := a.database.DeleteProject(r.Context(), id); err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		w.WriteHeader(http.StatusNoContent)

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (a *API) handleSSHKeys(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		keys, err := a.database.ListSSHKeys(r.Context())
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		if keys == nil {
			keys = []*db.SSHKey{}
		}
		for _, k := range keys {
			k.PrivateKey = ""
		}
		writeJSON(w, http.StatusOK, keys)

	case http.MethodPost:
		if !a.requireRole(w, r, db.RoleAdmin) {
			return
		}
		var k db.SSHKey
		if err := json.NewDecoder(r.Body).Decode(&k); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid json")
			return
		}
		if k.Name == "" || k.PrivateKey == "" {
			writeErr(w, http.StatusBadRequest, "name and private_key required")
			return
		}
		if err := a.database.CreateSSHKey(r.Context(), &k); err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		k.PrivateKey = ""
		writeJSON(w, http.StatusCreated, k)

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (a *API) handleSSHKeyDetail(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/ssh-keys/")
	if r.Method == http.MethodDelete {
		if !a.requireRole(w, r, db.RoleAdmin) {
			return
		}
		if err := a.database.DeleteSSHKey(r.Context(), id); err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		w.WriteHeader(http.StatusNoContent)
		return
	}
	http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
}

func (a *API) handleProviders(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/providers/"), "/")
	projectID := parts[0]

	switch r.Method {
	case http.MethodGet:
		configs, err := a.database.ListProviderConfigs(r.Context(), projectID)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		if configs == nil {
			configs = []*db.ProviderConfig{}
		}
		writeJSON(w, http.StatusOK, configs)

	case http.MethodPost:
		if !a.requireRole(w, r, db.RoleAdmin, db.RoleEditor) {
			return
		}
		var pc db.ProviderConfig
		if err := json.NewDecoder(r.Body).Decode(&pc); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid json")
			return
		}
		pc.ProjectID = projectID
		pc.Enabled = true
		if pc.WebhookPath == "" {
			pc.WebhookPath = "/hook/" + pc.ProviderType + "/" + projectID[:8]
		}
		if err := a.database.CreateProviderConfig(r.Context(), &pc); err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, pc)

	case http.MethodDelete:
		if !a.requireRole(w, r, db.RoleAdmin) {
			return
		}
		if len(parts) > 1 {
			if err := a.database.DeleteProviderConfig(r.Context(), parts[1]); err != nil {
				writeErr(w, http.StatusInternalServerError, err.Error())
				return
			}
			w.WriteHeader(http.StatusNoContent)
			return
		}
		writeErr(w, http.StatusBadRequest, "missing config id")

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (a *API) handleKeywords(w http.ResponseWriter, r *http.Request) {
	projectID := strings.TrimPrefix(r.URL.Path, "/api/keywords/")

	switch r.Method {
	case http.MethodGet:
		keywords, err := a.database.GetTriggerKeywords(r.Context(), projectID)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		if keywords == nil {
			keywords = []*db.TriggerKeyword{}
		}
		writeJSON(w, http.StatusOK, keywords)

	case http.MethodPut:
		if !a.requireRole(w, r, db.RoleAdmin, db.RoleEditor) {
			return
		}
		var keywords []db.TriggerKeyword
		if err := json.NewDecoder(r.Body).Decode(&keywords); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid json")
			return
		}
		if err := a.database.SetTriggerKeywords(r.Context(), projectID, keywords); err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (a *API) handleTasks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	defaultLimit := a.database.GetSettingInt(r.Context(), "task_list_default_limit", 50)
	maxLimit := a.database.GetSettingInt(r.Context(), "task_list_max_limit", 100)
	if limit <= 0 || limit > maxLimit {
		limit = defaultLimit
	}

	tasks, err := a.database.ListTasks(r.Context(), limit, offset)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	count, _ := a.database.CountTasks(r.Context())

	if tasks == nil {
		tasks = []*db.Task{}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"tasks": tasks,
		"total": count,
	})
}

func (a *API) handleTaskDetail(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/tasks/")
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	task, err := a.database.GetTask(r.Context(), id)
	if err != nil {
		writeErr(w, http.StatusNotFound, "task not found")
		return
	}
	writeJSON(w, http.StatusOK, task)
}

func (a *API) handleSettings(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		settings, err := a.database.ListSettings(r.Context())
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		if settings == nil {
			settings = []*db.Setting{}
		}
		writeJSON(w, http.StatusOK, settings)

	case http.MethodPut:
		if !a.requireRole(w, r, db.RoleAdmin) {
			return
		}
		var req struct {
			Key   string          `json:"key"`
			Value json.RawMessage `json:"value"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid json")
			return
		}
		if req.Key == "" {
			writeErr(w, http.StatusBadRequest, "key is required")
			return
		}
		if err := a.database.SetSetting(r.Context(), req.Key, req.Value); err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (a *API) handleSettingDetail(w http.ResponseWriter, r *http.Request) {
	key := strings.TrimPrefix(r.URL.Path, "/api/settings/")

	switch r.Method {
	case http.MethodGet:
		s, err := a.database.GetSetting(r.Context(), key)
		if err != nil {
			writeErr(w, http.StatusNotFound, "setting not found")
			return
		}
		writeJSON(w, http.StatusOK, s)

	case http.MethodDelete:
		if !a.requireRole(w, r, db.RoleAdmin) {
			return
		}
		if err := a.database.DeleteSetting(r.Context(), key); err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		w.WriteHeader(http.StatusNoContent)

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (a *API) handleMCPServers(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		servers, err := a.database.ListMCPServers(r.Context())
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		if servers == nil {
			servers = []*db.MCPServer{}
		}
		writeJSON(w, http.StatusOK, servers)

	case http.MethodPost:
		if !a.requireRole(w, r, db.RoleAdmin) {
			return
		}
		var m db.MCPServer
		if err := json.NewDecoder(r.Body).Decode(&m); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid json")
			return
		}
		if m.Name == "" || m.Package == "" {
			writeErr(w, http.StatusBadRequest, "name and package required")
			return
		}
		if m.Type == "" {
			m.Type = "npm"
		}
		if m.Args == nil {
			m.Args = json.RawMessage("[]")
		}
		if m.Env == nil {
			m.Env = json.RawMessage("{}")
		}
		m.Enabled = true
		if err := a.database.CreateMCPServer(r.Context(), &m); err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}

		go func() {
			if err := a.mcpMgr.Install(r.Context(), m.ID); err != nil {
				a.logger.Error("mcp server install failed", "id", m.ID, "error", err)
			}
		}()

		writeJSON(w, http.StatusCreated, m)

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (a *API) handleMCPServerDetail(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/mcp-servers/"), "/")
	id := parts[0]

	if len(parts) > 1 && parts[1] == "install" && r.Method == http.MethodPost {
		if !a.requireRole(w, r, db.RoleAdmin) {
			return
		}
		go func() {
			if err := a.mcpMgr.Install(r.Context(), id); err != nil {
				a.logger.Error("mcp server install failed", "id", id, "error", err)
			}
		}()
		writeJSON(w, http.StatusOK, map[string]string{"status": "installing"})
		return
	}

	switch r.Method {
	case http.MethodGet:
		m, err := a.database.GetMCPServer(r.Context(), id)
		if err != nil {
			writeErr(w, http.StatusNotFound, "mcp server not found")
			return
		}
		writeJSON(w, http.StatusOK, m)

	case http.MethodPut:
		if !a.requireRole(w, r, db.RoleAdmin) {
			return
		}
		var m db.MCPServer
		if err := json.NewDecoder(r.Body).Decode(&m); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid json")
			return
		}
		m.ID = id
		if err := a.database.UpdateMCPServer(r.Context(), &m); err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, m)

	case http.MethodDelete:
		if !a.requireRole(w, r, db.RoleAdmin) {
			return
		}
		go func() {
			if err := a.mcpMgr.Uninstall(r.Context(), id); err != nil {
				a.logger.Error("mcp server uninstall failed", "id", id, "error", err)
			}
		}()
		w.WriteHeader(http.StatusNoContent)

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (a *API) handleUsers(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		if !a.requireRole(w, r, db.RoleAdmin) {
			return
		}
		users, err := a.database.ListUsers(r.Context())
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		if users == nil {
			users = []*db.User{}
		}
		writeJSON(w, http.StatusOK, users)

	case http.MethodPost:
		if !a.requireRole(w, r, db.RoleAdmin) {
			return
		}
		var req struct {
			Username    string `json:"username"`
			Password    string `json:"password"`
			DisplayName string `json:"display_name"`
			Role        string `json:"role"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid json")
			return
		}
		if req.Username == "" || req.Password == "" {
			writeErr(w, http.StatusBadRequest, "username and password required")
			return
		}
		if req.Role == "" {
			req.Role = db.RoleViewer
		}
		hash, err := auth.HashPassword(req.Password)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, "hash failed")
			return
		}
		u := &db.User{
			Username:     req.Username,
			PasswordHash: hash,
			DisplayName:  req.DisplayName,
			Role:         req.Role,
			Enabled:      true,
		}
		if err := a.database.CreateUser(r.Context(), u); err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, u)

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (a *API) handleUserDetail(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/users/")

	switch r.Method {
	case http.MethodGet:
		if !a.requireRole(w, r, db.RoleAdmin) {
			return
		}
		u, err := a.database.GetUser(r.Context(), id)
		if err != nil {
			writeErr(w, http.StatusNotFound, "user not found")
			return
		}
		writeJSON(w, http.StatusOK, u)

	case http.MethodPut:
		if !a.requireRole(w, r, db.RoleAdmin) {
			return
		}
		var u db.User
		if err := json.NewDecoder(r.Body).Decode(&u); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid json")
			return
		}
		u.ID = id
		if err := a.database.UpdateUser(r.Context(), &u); err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, u)

	case http.MethodDelete:
		if !a.requireRole(w, r, db.RoleAdmin) {
			return
		}
		if err := a.database.DeleteUser(r.Context(), id); err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		w.WriteHeader(http.StatusNoContent)

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (a *API) requireRole(w http.ResponseWriter, r *http.Request, roles ...string) bool {
	claims := auth.GetUser(r.Context())
	if claims == nil {
		writeErr(w, http.StatusUnauthorized, "not authenticated")
		return false
	}
	for _, role := range roles {
		if claims.Role == role {
			return true
		}
	}
	writeErr(w, http.StatusForbidden, "insufficient permissions")
	return false
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
