package provider

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

// --- GitLabProvider Type ---

func TestGitLabProvider_Type(t *testing.T) {
	p := NewGitLabProvider(slog.Default())
	if p.Type() != ProviderGitLab {
		t.Errorf("Type() = %q, want %q", p.Type(), ProviderGitLab)
	}
}

// --- GitLabProvider ValidateConfig ---

func TestGitLabProvider_ValidateConfig_Valid(t *testing.T) {
	p := NewGitLabProvider(slog.Default())
	cfg := map[string]any{"base_url": "https://gitlab.com", "token": "tok"}
	if err := p.ValidateConfig(cfg); err != nil {
		t.Errorf("ValidateConfig() error = %v", err)
	}
}

func TestGitLabProvider_ValidateConfig_MissingBaseURL(t *testing.T) {
	p := NewGitLabProvider(slog.Default())
	cfg := map[string]any{"token": "tok"}
	err := p.ValidateConfig(cfg)
	if err == nil {
		t.Fatal("ValidateConfig() expected error for missing base_url")
	}
	if !strings.Contains(err.Error(), "base_url") {
		t.Errorf("error = %q, want mention of base_url", err.Error())
	}
}

func TestGitLabProvider_ValidateConfig_MissingToken(t *testing.T) {
	p := NewGitLabProvider(slog.Default())
	cfg := map[string]any{"base_url": "https://gitlab.com"}
	err := p.ValidateConfig(cfg)
	if err == nil {
		t.Fatal("ValidateConfig() expected error for missing token")
	}
	if !strings.Contains(err.Error(), "token") {
		t.Errorf("error = %q, want mention of token", err.Error())
	}
}

func TestGitLabProvider_ValidateConfig_Empty(t *testing.T) {
	p := NewGitLabProvider(slog.Default())
	err := p.ValidateConfig(map[string]any{})
	if err == nil {
		t.Fatal("ValidateConfig() expected error for empty config")
	}
}

// --- GitLab BuildHandler: method not allowed ---

func TestGitLabHandler_MethodNotAllowed(t *testing.T) {
	p := NewGitLabProvider(slog.Default())
	handler := p.BuildHandler("cfg-1", "secret", nil, nil)

	for _, method := range []string{http.MethodGet, http.MethodPut, http.MethodDelete, http.MethodPatch} {
		req := httptest.NewRequest(method, "/hook/gitlab/test", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("%s: status = %d, want %d", method, w.Code, http.StatusMethodNotAllowed)
		}
	}
}

// --- GitLab BuildHandler: invalid token ---

func TestGitLabHandler_InvalidToken(t *testing.T) {
	p := NewGitLabProvider(slog.Default())
	handler := p.BuildHandler("cfg-1", "correct-secret", nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/hook/gitlab/test", nil)
	req.Header.Set("X-Gitlab-Token", "wrong-secret")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", w.Code, http.StatusForbidden)
	}
}

func TestGitLabHandler_MissingToken(t *testing.T) {
	p := NewGitLabProvider(slog.Default())
	handler := p.BuildHandler("cfg-1", "my-secret", nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/hook/gitlab/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", w.Code, http.StatusForbidden)
	}
}

// --- GitLab BuildHandler: non-note event ---

func TestGitLabHandler_NonNoteEvent(t *testing.T) {
	p := NewGitLabProvider(slog.Default())
	handler := p.BuildHandler("cfg-1", "secret", nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/hook/gitlab/test", strings.NewReader("{}"))
	req.Header.Set("X-Gitlab-Token", "secret")
	req.Header.Set("X-Gitlab-Event", "Push Hook")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

// --- GitLab BuildHandler: empty body for note event ---

func TestGitLabHandler_EmptyBody(t *testing.T) {
	p := NewGitLabProvider(slog.Default())
	handler := p.BuildHandler("cfg-1", "secret", nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/hook/gitlab/test", strings.NewReader(""))
	req.Header.Set("X-Gitlab-Token", "secret")
	req.Header.Set("X-Gitlab-Event", "Note Hook")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

// --- GitLab BuildHandler: system comment ---

func TestGitLabHandler_SystemComment(t *testing.T) {
	p := NewGitLabProvider(slog.Default())

	var called bool
	onMessage := func(_ context.Context, _ *IncomingMessage) {
		called = true
	}
	handler := p.BuildHandler("cfg-1", "secret", nil, onMessage)

	payload := map[string]any{
		"object_kind":   "note",
		"event_type":    "note",
		"user":          map[string]any{"username": "bot"},
		"project_id":    1,
		"project":       map[string]any{"web_url": "https://gitlab.com/test/proj"},
		"noteable_type": "Issue",
		"object_attributes": map[string]any{
			"note":          "system note",
			"system":        true,
			"noteable_type": "Issue",
			"url":           "https://gitlab.com/test/proj/-/issues/1#note_1",
		},
		"issue": map[string]any{
			"iid":   1,
			"title": "Test Issue",
		},
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/hook/gitlab/test", strings.NewReader(string(body)))
	req.Header.Set("X-Gitlab-Token", "secret")
	req.Header.Set("X-Gitlab-Event", "Note Hook")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	time.Sleep(50 * time.Millisecond)
	if called {
		t.Error("onMessage should not be called for system comment")
	}
}

// --- GitLab BuildHandler: valid issue comment ---

func TestGitLabHandler_ValidIssueComment(t *testing.T) {
	p := NewGitLabProvider(slog.Default())

	var mu sync.Mutex
	var received *IncomingMessage
	onMessage := func(_ context.Context, msg *IncomingMessage) {
		mu.Lock()
		received = msg
		mu.Unlock()
	}
	handler := p.BuildHandler("cfg-1", "secret", nil, onMessage)

	payload := map[string]any{
		"object_kind":   "note",
		"event_type":    "note",
		"user":          map[string]any{"username": "alice"},
		"project_id":    42,
		"project":       map[string]any{"web_url": "https://gitlab.com/test/proj"},
		"noteable_type": "Issue",
		"object_attributes": map[string]any{
			"note":          "@opencode analyze this",
			"system":        false,
			"noteable_type": "Issue",
			"url":           "https://gitlab.com/test/proj/-/issues/5#note_100",
		},
		"issue": map[string]any{
			"iid":   5,
			"title": "Bug Report",
		},
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/hook/gitlab/test", strings.NewReader(string(body)))
	req.Header.Set("X-Gitlab-Token", "secret")
	req.Header.Set("X-Gitlab-Event", "Note Hook")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	time.Sleep(100 * time.Millisecond)
	mu.Lock()
	defer mu.Unlock()

	if received == nil {
		t.Fatal("onMessage was not called")
	}
	if received.Provider != ProviderGitLab {
		t.Errorf("Provider = %q, want %q", received.Provider, ProviderGitLab)
	}
	if received.ProviderCfgID != "cfg-1" {
		t.Errorf("ProviderCfgID = %q, want %q", received.ProviderCfgID, "cfg-1")
	}
	if received.Author != "alice" {
		t.Errorf("Author = %q, want %q", received.Author, "alice")
	}
	if received.Body != "@opencode analyze this" {
		t.Errorf("Body = %q", received.Body)
	}
	if received.Title != "Bug Report" {
		t.Errorf("Title = %q, want %q", received.Title, "Bug Report")
	}
	if received.ExternalRef != "https://gitlab.com/test/proj/-/issues/5#note_100" {
		t.Errorf("ExternalRef = %q", received.ExternalRef)
	}
}

// --- GitLab BuildHandler: malformed JSON ---

func TestGitLabHandler_MalformedJSON(t *testing.T) {
	p := NewGitLabProvider(slog.Default())
	handler := p.BuildHandler("cfg-1", "secret", nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/hook/gitlab/test", strings.NewReader("{not valid json"))
	req.Header.Set("X-Gitlab-Token", "secret")
	req.Header.Set("X-Gitlab-Event", "Note Hook")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnprocessableEntity)
	}
}
