package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/opencode-ai/opencode-dog/internal/db"
)

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
