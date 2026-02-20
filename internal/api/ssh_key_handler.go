package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/opencode-ai/opencode-dog/internal/db"
)

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
