package api

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/opencode-ai/opencode-dog/internal/db"
)

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
