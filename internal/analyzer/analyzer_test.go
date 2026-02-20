package analyzer

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/opencode-ai/opencode-dog/internal/db"
	"github.com/opencode-ai/opencode-dog/internal/db/dbmock"
	"github.com/opencode-ai/opencode-dog/internal/provider"
)

// ---- ptrStr ----

func TestPtrStr_Empty(t *testing.T) {
	if ptrStr("") != nil {
		t.Fatal("expected nil for empty string")
	}
}

func TestPtrStr_NonEmpty(t *testing.T) {
	p := ptrStr("hello")
	if p == nil {
		t.Fatal("expected non-nil pointer")
	}
	if *p != "hello" {
		t.Fatalf("got %q, want %q", *p, "hello")
	}
}

// ---- matchKeyword ----

func TestMatchKeyword_Match(t *testing.T) {
	keywords := []*db.TriggerKeyword{
		{Keyword: "@opencode", Mode: "ask"},
	}
	kw, mode := matchKeyword("hey @opencode please help", keywords)
	if kw != "@opencode" {
		t.Fatalf("keyword: got %q, want %q", kw, "@opencode")
	}
	if mode != provider.ModeAsk {
		t.Fatalf("mode: got %q, want %q", mode, provider.ModeAsk)
	}
}

func TestMatchKeyword_CaseInsensitive(t *testing.T) {
	keywords := []*db.TriggerKeyword{
		{Keyword: "@OpenCode", Mode: "plan"},
	}
	kw, mode := matchKeyword("HEY @OPENCODE DO STUFF", keywords)
	if kw != "@OpenCode" {
		t.Fatalf("keyword: got %q, want %q", kw, "@OpenCode")
	}
	if mode != provider.ModePlan {
		t.Fatalf("mode: got %q, want %q", mode, provider.ModePlan)
	}
}

func TestMatchKeyword_NoMatch(t *testing.T) {
	keywords := []*db.TriggerKeyword{
		{Keyword: "@opencode", Mode: "ask"},
	}
	kw, mode := matchKeyword("nothing relevant here", keywords)
	if kw != "" || mode != "" {
		t.Fatalf("expected empty, got kw=%q mode=%q", kw, mode)
	}
}

func TestMatchKeyword_EmptyKeywords(t *testing.T) {
	kw, mode := matchKeyword("anything", nil)
	if kw != "" || mode != "" {
		t.Fatalf("expected empty, got kw=%q mode=%q", kw, mode)
	}
}

func TestMatchKeyword_FirstMatchWins(t *testing.T) {
	keywords := []*db.TriggerKeyword{
		{Keyword: "@plan", Mode: "plan"},
		{Keyword: "@do", Mode: "do"},
	}
	kw, mode := matchKeyword("please @plan and @do this", keywords)
	if kw != "@plan" {
		t.Fatalf("keyword: got %q, want %q", kw, "@plan")
	}
	if mode != provider.ModePlan {
		t.Fatalf("mode: got %q, want %q", mode, provider.ModePlan)
	}
}

// ---- buildPrompt ----

func TestBuildPrompt_AskMode(t *testing.T) {
	store := dbmock.New()
	a := &Analyzer{database: store, logger: slog.Default(), configDir: t.TempDir()}

	msg := &provider.IncomingMessage{
		Provider: provider.ProviderGitLab,
		Title:    "Bug Report",
		Author:   "alice",
		Body:     "Why is it broken?",
	}

	result := a.buildPrompt(context.Background(), msg, provider.ModeAsk)
	if !strings.Contains(result, "Answer the following question") {
		t.Fatalf("ask prompt missing expected prefix, got:\n%s", result)
	}
	if !strings.Contains(result, "Bug Report") {
		t.Fatal("prompt missing title")
	}
	if !strings.Contains(result, "@alice") {
		t.Fatal("prompt missing author")
	}
	if !strings.Contains(result, "Why is it broken?") {
		t.Fatal("prompt missing body")
	}
}

func TestBuildPrompt_PlanMode(t *testing.T) {
	store := dbmock.New()
	a := &Analyzer{database: store, logger: slog.Default(), configDir: t.TempDir()}
	msg := &provider.IncomingMessage{Provider: provider.ProviderSlack, Title: "Refactor", Author: "bob", Body: "plan this"}

	result := a.buildPrompt(context.Background(), msg, provider.ModePlan)
	if !strings.Contains(result, "implementation plan") {
		t.Fatalf("plan prompt missing expected prefix, got:\n%s", result)
	}
}

func TestBuildPrompt_DoMode(t *testing.T) {
	store := dbmock.New()
	a := &Analyzer{database: store, logger: slog.Default(), configDir: t.TempDir()}
	msg := &provider.IncomingMessage{Provider: provider.ProviderTelegram, Title: "Fix", Author: "charlie", Body: "fix it"}

	result := a.buildPrompt(context.Background(), msg, provider.ModeDo)
	if !strings.Contains(result, "exact code changes") {
		t.Fatalf("do prompt missing expected prefix, got:\n%s", result)
	}
}

func TestBuildPrompt_DefaultMode(t *testing.T) {
	store := dbmock.New()
	a := &Analyzer{database: store, logger: slog.Default(), configDir: t.TempDir()}
	msg := &provider.IncomingMessage{Provider: provider.ProviderGitLab, Title: "Q", Author: "dave", Body: "something"}

	result := a.buildPrompt(context.Background(), msg, "unknown")
	if !strings.Contains(result, "Analyze the following") {
		t.Fatalf("default prompt missing expected prefix, got:\n%s", result)
	}
}

func TestBuildPrompt_WithExternalRef(t *testing.T) {
	store := dbmock.New()
	a := &Analyzer{database: store, logger: slog.Default(), configDir: t.TempDir()}
	msg := &provider.IncomingMessage{
		Provider:    provider.ProviderGitLab,
		Title:       "T",
		Author:      "eve",
		Body:        "body",
		ExternalRef: "https://gitlab.com/issue/1",
	}

	result := a.buildPrompt(context.Background(), msg, provider.ModeAsk)
	if !strings.Contains(result, "https://gitlab.com/issue/1") {
		t.Fatal("prompt missing external ref")
	}
}

func TestBuildPrompt_CustomPromptFromSettings(t *testing.T) {
	store := dbmock.New()
	_ = store.SetSetting(context.Background(), "prompt_ask", mustJSON("Custom ask: "))

	a := &Analyzer{database: store, logger: slog.Default(), configDir: t.TempDir()}
	msg := &provider.IncomingMessage{Provider: provider.ProviderGitLab, Title: "T", Author: "f", Body: "b"}

	result := a.buildPrompt(context.Background(), msg, provider.ModeAsk)
	if !strings.HasPrefix(result, "Custom ask: ") {
		t.Fatalf("expected custom prefix, got:\n%s", result)
	}
}

// ---- HandleMessage integration ----

type fakeProvider struct {
	replies []string
}

func (f *fakeProvider) Type() provider.ProviderType           { return provider.ProviderGitLab }
func (f *fakeProvider) ValidateConfig(_ map[string]any) error { return nil }
func (f *fakeProvider) BuildHandler(_, _ string, _ map[string]any, _ func(context.Context, *provider.IncomingMessage)) http.Handler {
	return nil
}
func (f *fakeProvider) SendReply(_ context.Context, _ map[string]any, _ *provider.IncomingMessage, body string) error {
	f.replies = append(f.replies, body)
	return nil
}

func TestHandleMessage_FullFlow(t *testing.T) {
	ocServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "POST" && r.URL.Path == "/session":
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(Session{ID: "sess-1", Title: "test"})
		case r.Method == "POST" && strings.HasPrefix(r.URL.Path, "/session/sess-1/message"):
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(MessageResponse{
				Parts: []MessagePart{{Type: "text", Text: "analysis result"}},
			})
		case r.Method == "DELETE" && r.URL.Path == "/session/sess-1":
			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer ocServer.Close()

	store := dbmock.New()
	_ = store.SetSetting(context.Background(), "opencode_server_url", mustJSON(ocServer.URL))

	pcfg := &db.ProviderConfig{
		ProjectID:    "proj-1",
		ProviderType: "gitlab",
		Config:       json.RawMessage(`{}`),
		Enabled:      true,
	}
	_ = store.CreateProviderConfig(context.Background(), pcfg)

	_ = store.SetTriggerKeywords(context.Background(), "proj-1", []db.TriggerKeyword{
		{Keyword: "@opencode", Mode: "ask"},
	})

	fp := &fakeProvider{}
	registry := provider.NewRegistry(slog.Default())
	registry.Register(fp)

	logger := slog.Default()
	client := NewOpencodeClient(ocServer.URL, "user", "pass", 30*time.Second, logger)
	a := &Analyzer{
		database:       store,
		registry:       registry,
		logger:         logger,
		configDir:      t.TempDir(),
		opencodeClient: client,
	}

	msg := &provider.IncomingMessage{
		Provider:      provider.ProviderGitLab,
		ProviderCfgID: pcfg.ID,
		ProjectID:     "proj-1",
		Title:         "Test Issue",
		Body:          "hey @opencode help me",
		Author:        "tester",
		ExternalRef:   "https://gitlab.com/issue/42",
	}

	a.HandleMessage(context.Background(), msg)

	if len(store.Tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(store.Tasks))
	}
	task := store.Tasks[0]
	if task.Status != db.TaskStatusCompleted {
		t.Fatalf("task status: got %q, want %q", task.Status, db.TaskStatusCompleted)
	}
	if task.Result == nil || !strings.Contains(*task.Result, "analysis result") {
		t.Fatal("task result missing expected content")
	}

	if len(fp.replies) < 2 {
		t.Fatalf("expected at least 2 replies (ack + result), got %d", len(fp.replies))
	}
	if !strings.Contains(fp.replies[0], "received your request") {
		t.Fatalf("ack reply unexpected: %s", fp.replies[0])
	}
	if !strings.Contains(fp.replies[1], "analysis result") {
		t.Fatalf("result reply unexpected: %s", fp.replies[1])
	}
}

func TestHandleMessage_NoKeywordMatch(t *testing.T) {
	store := dbmock.New()

	pcfg := &db.ProviderConfig{
		ProjectID:    "proj-1",
		ProviderType: "gitlab",
		Config:       json.RawMessage(`{}`),
		Enabled:      true,
	}
	_ = store.CreateProviderConfig(context.Background(), pcfg)

	_ = store.SetTriggerKeywords(context.Background(), "proj-1", []db.TriggerKeyword{
		{Keyword: "@opencode", Mode: "ask"},
	})

	fp := &fakeProvider{}
	registry := provider.NewRegistry(slog.Default())
	registry.Register(fp)

	a := &Analyzer{
		database:  store,
		registry:  registry,
		logger:    slog.Default(),
		configDir: t.TempDir(),
	}

	msg := &provider.IncomingMessage{
		Provider:      provider.ProviderGitLab,
		ProviderCfgID: pcfg.ID,
		ProjectID:     "proj-1",
		Title:         "Just chatting",
		Body:          "no trigger here",
		Author:        "tester",
	}

	a.HandleMessage(context.Background(), msg)

	if len(store.Tasks) != 0 {
		t.Fatalf("expected 0 tasks, got %d", len(store.Tasks))
	}
	if len(fp.replies) != 0 {
		t.Fatalf("expected 0 replies, got %d", len(fp.replies))
	}
}

func TestHandleMessage_AnalysisError(t *testing.T) {
	ocServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "POST" && r.URL.Path == "/session":
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(Session{ID: "sess-err", Title: "test"})
		case r.Method == "POST" && strings.HasPrefix(r.URL.Path, "/session/sess-err/message"):
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(w, "server error")
		case r.Method == "DELETE":
			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer ocServer.Close()

	store := dbmock.New()

	pcfg := &db.ProviderConfig{
		ProjectID:    "proj-1",
		ProviderType: "gitlab",
		Config:       json.RawMessage(`{}`),
		Enabled:      true,
	}
	_ = store.CreateProviderConfig(context.Background(), pcfg)

	_ = store.SetTriggerKeywords(context.Background(), "proj-1", []db.TriggerKeyword{
		{Keyword: "@opencode", Mode: "do"},
	})

	fp := &fakeProvider{}
	registry := provider.NewRegistry(slog.Default())
	registry.Register(fp)

	logger := slog.Default()
	client := NewOpencodeClient(ocServer.URL, "user", "pass", 30*time.Second, logger)
	a := &Analyzer{
		database:       store,
		registry:       registry,
		logger:         logger,
		configDir:      t.TempDir(),
		opencodeClient: client,
	}

	msg := &provider.IncomingMessage{
		Provider:      provider.ProviderGitLab,
		ProviderCfgID: pcfg.ID,
		ProjectID:     "proj-1",
		Title:         "Fix bug",
		Body:          "hey @opencode fix this",
		Author:        "tester",
	}

	a.HandleMessage(context.Background(), msg)

	if len(store.Tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(store.Tasks))
	}
	if store.Tasks[0].Status != db.TaskStatusFailed {
		t.Fatalf("task status: got %q, want %q", store.Tasks[0].Status, db.TaskStatusFailed)
	}
	if len(fp.replies) < 2 {
		t.Fatalf("expected at least 2 replies (ack + error), got %d", len(fp.replies))
	}
	if !strings.Contains(fp.replies[1], "error") {
		t.Fatalf("error reply unexpected: %s", fp.replies[1])
	}
}

// ---- helpers ----

func mustJSON(v string) json.RawMessage {
	b, _ := json.Marshal(v)
	return b
}
