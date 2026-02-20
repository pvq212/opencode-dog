package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/opencode-ai/opencode-dog/internal/auth"
	"github.com/opencode-ai/opencode-dog/internal/db"
)

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
