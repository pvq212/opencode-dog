package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/opencode-ai/opencode-dog/internal/db"
)

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
