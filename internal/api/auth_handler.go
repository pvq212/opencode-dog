package api

import (
	"encoding/json"
	"net/http"

	"github.com/opencode-ai/opencode-dog/internal/auth"
)

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
