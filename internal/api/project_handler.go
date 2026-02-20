package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/opencode-ai/opencode-dog/internal/db"
)

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
