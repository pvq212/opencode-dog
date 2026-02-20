package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/opencode-ai/opencode-dog/internal/db"
)

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
