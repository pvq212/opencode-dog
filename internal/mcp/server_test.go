package mcp

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/opencode-ai/opencode-dog/internal/db"
	"github.com/opencode-ai/opencode-dog/internal/db/dbmock"
)

func newTestServer(store *dbmock.Store) *Server {
	return NewServer(store, slog.Default())
}

func makeReq(args map[string]any) mcp.CallToolRequest {
	return mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: args,
		},
	}
}

func resultText(r *mcp.CallToolResult) string {
	if len(r.Content) == 0 {
		return ""
	}
	if tc, ok := r.Content[0].(mcp.TextContent); ok {
		return tc.Text
	}
	return ""
}

// ---- optionalInt ----

func TestOptionalInt_StringValue(t *testing.T) {
	req := makeReq(map[string]any{"limit": "42"})
	got := optionalInt(req, "limit", 10)
	if got != 42 {
		t.Fatalf("got %d, want 42", got)
	}
}

func TestOptionalInt_Float64Value(t *testing.T) {
	req := makeReq(map[string]any{"limit": float64(7)})
	got := optionalInt(req, "limit", 10)
	if got != 7 {
		t.Fatalf("got %d, want 7", got)
	}
}

func TestOptionalInt_MissingKey(t *testing.T) {
	req := makeReq(map[string]any{})
	got := optionalInt(req, "limit", 99)
	if got != 99 {
		t.Fatalf("got %d, want 99", got)
	}
}

func TestOptionalInt_InvalidString(t *testing.T) {
	req := makeReq(map[string]any{"limit": "abc"})
	got := optionalInt(req, "limit", 55)
	if got != 55 {
		t.Fatalf("got %d, want 55", got)
	}
}

func TestOptionalInt_NilArguments(t *testing.T) {
	req := mcp.CallToolRequest{}
	got := optionalInt(req, "limit", 33)
	if got != 33 {
		t.Fatalf("got %d, want 33", got)
	}
}

// ---- handleListProjects ----

func TestHandleListProjects_Empty(t *testing.T) {
	s := newTestServer(dbmock.New())
	result, err := s.handleListProjects(context.Background(), makeReq(nil))
	if err != nil {
		t.Fatal(err)
	}
	txt := resultText(result)
	if txt != "No projects configured." {
		t.Fatalf("got %q", txt)
	}
}

func TestHandleListProjects_WithProjects(t *testing.T) {
	store := dbmock.New()
	_ = store.CreateProject(context.Background(), &db.Project{
		Name:          "Alpha",
		SSHURL:        "git@example.com:alpha.git",
		DefaultBranch: "main",
		Enabled:       true,
	})
	_ = store.CreateProject(context.Background(), &db.Project{
		Name:          "Beta",
		SSHURL:        "git@example.com:beta.git",
		DefaultBranch: "develop",
		Enabled:       false,
	})

	s := newTestServer(store)
	result, err := s.handleListProjects(context.Background(), makeReq(nil))
	if err != nil {
		t.Fatal(err)
	}
	txt := resultText(result)
	if !strings.Contains(txt, "Alpha") {
		t.Fatal("missing Alpha")
	}
	if !strings.Contains(txt, "Beta") {
		t.Fatal("missing Beta")
	}
	if !strings.Contains(txt, "[enabled]") {
		t.Fatal("missing enabled status")
	}
	if !strings.Contains(txt, "[disabled]") {
		t.Fatal("missing disabled status")
	}
}

// ---- handleGetProject ----

func TestHandleGetProject_Found(t *testing.T) {
	store := dbmock.New()
	p := &db.Project{Name: "Test", SSHURL: "git@test.com:test.git", DefaultBranch: "main", Enabled: true}
	_ = store.CreateProject(context.Background(), p)

	s := newTestServer(store)
	result, err := s.handleGetProject(context.Background(), makeReq(map[string]any{"project_id": p.ID}))
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatalf("unexpected error: %s", resultText(result))
	}
	txt := resultText(result)
	if !strings.Contains(txt, "Test") {
		t.Fatalf("missing project name in: %s", txt)
	}
}

func TestHandleGetProject_NotFound(t *testing.T) {
	s := newTestServer(dbmock.New())
	result, err := s.handleGetProject(context.Background(), makeReq(map[string]any{"project_id": "nonexistent"}))
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Fatal("expected error result for missing project")
	}
	if !strings.Contains(resultText(result), "not found") {
		t.Fatalf("error text: %s", resultText(result))
	}
}

func TestHandleGetProject_MissingArg(t *testing.T) {
	s := newTestServer(dbmock.New())
	result, err := s.handleGetProject(context.Background(), makeReq(map[string]any{}))
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Fatal("expected error for missing project_id")
	}
}

// ---- handleListTasks ----

func TestHandleListTasks_Empty(t *testing.T) {
	s := newTestServer(dbmock.New())
	result, err := s.handleListTasks(context.Background(), makeReq(map[string]any{}))
	if err != nil {
		t.Fatal(err)
	}
	if resultText(result) != "No tasks found." {
		t.Fatalf("got %q", resultText(result))
	}
}

func TestHandleListTasks_WithPagination(t *testing.T) {
	store := dbmock.New()
	for i := 0; i < 5; i++ {
		_ = store.CreateTask(context.Background(), &db.Task{
			ProviderType:   "gitlab",
			TriggerMode:    "ask",
			TriggerKeyword: "@oc",
			Title:          "Task",
			Author:         "user",
			CreatedAt:      time.Now(),
		})
	}

	s := newTestServer(store)

	result, err := s.handleListTasks(context.Background(), makeReq(map[string]any{"limit": "2", "offset": "0"}))
	if err != nil {
		t.Fatal(err)
	}
	txt := resultText(result)
	count := strings.Count(txt, "### Task ")
	if count != 2 {
		t.Fatalf("expected 2 tasks, got %d in:\n%s", count, txt)
	}

	result2, err := s.handleListTasks(context.Background(), makeReq(map[string]any{"limit": float64(3), "offset": float64(3)}))
	if err != nil {
		t.Fatal(err)
	}
	txt2 := resultText(result2)
	count2 := strings.Count(txt2, "### Task ")
	if count2 != 2 {
		t.Fatalf("expected 2 tasks from offset 3, got %d in:\n%s", count2, txt2)
	}
}

func TestHandleListTasks_DefaultPagination(t *testing.T) {
	store := dbmock.New()
	for i := 0; i < 3; i++ {
		_ = store.CreateTask(context.Background(), &db.Task{
			ProviderType:   "slack",
			TriggerMode:    "plan",
			TriggerKeyword: "@plan",
			Title:          "T",
			Author:         "u",
			CreatedAt:      time.Now(),
		})
	}

	s := newTestServer(store)
	result, err := s.handleListTasks(context.Background(), makeReq(map[string]any{}))
	if err != nil {
		t.Fatal(err)
	}
	count := strings.Count(resultText(result), "### Task ")
	if count != 3 {
		t.Fatalf("expected 3 tasks with default limit, got %d", count)
	}
}

// ---- handleGetTask ----

func TestHandleGetTask_Found(t *testing.T) {
	store := dbmock.New()
	task := &db.Task{
		ProviderType:   "gitlab",
		TriggerMode:    "ask",
		TriggerKeyword: "@oc",
		Title:          "Test Task",
		Author:         "alice",
		MessageBody:    "body",
		CreatedAt:      time.Now(),
	}
	_ = store.CreateTask(context.Background(), task)

	s := newTestServer(store)
	result, err := s.handleGetTask(context.Background(), makeReq(map[string]any{"task_id": task.ID}))
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatalf("unexpected error: %s", resultText(result))
	}

	var decoded db.Task
	if jsonErr := json.Unmarshal([]byte(resultText(result)), &decoded); jsonErr != nil {
		t.Fatalf("invalid JSON: %v", jsonErr)
	}
	if decoded.Title != "Test Task" {
		t.Fatalf("title: got %q, want %q", decoded.Title, "Test Task")
	}
}

func TestHandleGetTask_NotFound(t *testing.T) {
	s := newTestServer(dbmock.New())
	result, err := s.handleGetTask(context.Background(), makeReq(map[string]any{"task_id": "nope"}))
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Fatal("expected error")
	}
	if !strings.Contains(resultText(result), "not found") {
		t.Fatalf("error text: %s", resultText(result))
	}
}

func TestHandleGetTask_MissingArg(t *testing.T) {
	s := newTestServer(dbmock.New())
	result, err := s.handleGetTask(context.Background(), makeReq(map[string]any{}))
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Fatal("expected error for missing task_id")
	}
}

// ---- handleListProviders ----

func TestHandleListProviders_Empty(t *testing.T) {
	s := newTestServer(dbmock.New())
	result, err := s.handleListProviders(context.Background(), makeReq(map[string]any{"project_id": "proj-1"}))
	if err != nil {
		t.Fatal(err)
	}
	if resultText(result) != "No providers configured for this project." {
		t.Fatalf("got %q", resultText(result))
	}
}

func TestHandleListProviders_WithProjectFilter(t *testing.T) {
	store := dbmock.New()
	_ = store.CreateProviderConfig(context.Background(), &db.ProviderConfig{
		ProjectID:    "proj-A",
		ProviderType: "gitlab",
		Config:       json.RawMessage(`{}`),
		WebhookPath:  "/hook/gitlab/a",
		Enabled:      true,
	})
	_ = store.CreateProviderConfig(context.Background(), &db.ProviderConfig{
		ProjectID:    "proj-A",
		ProviderType: "slack",
		Config:       json.RawMessage(`{}`),
		WebhookPath:  "/hook/slack/a",
		Enabled:      false,
	})
	_ = store.CreateProviderConfig(context.Background(), &db.ProviderConfig{
		ProjectID:    "proj-B",
		ProviderType: "telegram",
		Config:       json.RawMessage(`{}`),
		WebhookPath:  "/hook/telegram/b",
		Enabled:      true,
	})

	s := newTestServer(store)

	result, err := s.handleListProviders(context.Background(), makeReq(map[string]any{"project_id": "proj-A"}))
	if err != nil {
		t.Fatal(err)
	}
	txt := resultText(result)
	if !strings.Contains(txt, "gitlab") {
		t.Fatal("missing gitlab provider")
	}
	if !strings.Contains(txt, "slack") {
		t.Fatal("missing slack provider")
	}
	if strings.Contains(txt, "telegram") {
		t.Fatal("should not include proj-B's telegram provider")
	}
}

func TestHandleListProviders_MissingArg(t *testing.T) {
	s := newTestServer(dbmock.New())
	result, err := s.handleListProviders(context.Background(), makeReq(map[string]any{}))
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Fatal("expected error for missing project_id")
	}
}
