// Package api implements REST API handlers for the admin WebUI.
//
// Routes are registered via RegisterRoutes() on a standard http.ServeMux.
// Public routes (login) are registered directly; all other routes are wrapped
// with auth.Middleware for Bearer token validation. Each handler performs
// inline RBAC checks via requireRole().
package api

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/opencode-ai/opencode-dog/internal/auth"
	"github.com/opencode-ai/opencode-dog/internal/db"
	"github.com/opencode-ai/opencode-dog/internal/mcpmgr"
)

type API struct {
	database db.Store
	auth     *auth.Auth
	mcpMgr   *mcpmgr.Manager
	logger   *slog.Logger
}

func New(database db.Store, a *auth.Auth, mcpMgr *mcpmgr.Manager, logger *slog.Logger) *API {
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
