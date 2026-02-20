package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/opencode-ai/opencode-dog/internal/db"
)

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
