package provider

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/opencode-ai/opencode-dog/internal/db/dbmock"
)

// --- SlackProvider Type ---

func TestSlackProvider_Type(t *testing.T) {
	p := NewSlackProvider(dbmock.New(), slog.Default())
	if p.Type() != ProviderSlack {
		t.Errorf("Type() = %q, want %q", p.Type(), ProviderSlack)
	}
}

// --- SlackProvider ValidateConfig ---

func TestSlackProvider_ValidateConfig_Valid(t *testing.T) {
	p := NewSlackProvider(dbmock.New(), slog.Default())
	cfg := map[string]any{"bot_token": "xoxb-xxx", "signing_secret": "sec"}
	if err := p.ValidateConfig(cfg); err != nil {
		t.Errorf("ValidateConfig() error = %v", err)
	}
}

func TestSlackProvider_ValidateConfig_MissingBotToken(t *testing.T) {
	p := NewSlackProvider(dbmock.New(), slog.Default())
	cfg := map[string]any{"signing_secret": "sec"}
	err := p.ValidateConfig(cfg)
	if err == nil {
		t.Fatal("expected error for missing bot_token")
	}
	if !strings.Contains(err.Error(), "bot_token") {
		t.Errorf("error = %q, want mention of bot_token", err.Error())
	}
}

func TestSlackProvider_ValidateConfig_MissingSigningSecret(t *testing.T) {
	p := NewSlackProvider(dbmock.New(), slog.Default())
	cfg := map[string]any{"bot_token": "xoxb-xxx"}
	err := p.ValidateConfig(cfg)
	if err == nil {
		t.Fatal("expected error for missing signing_secret")
	}
	if !strings.Contains(err.Error(), "signing_secret") {
		t.Errorf("error = %q, want mention of signing_secret", err.Error())
	}
}

// --- Slack BuildHandler: method not allowed ---

func TestSlackHandler_MethodNotAllowed(t *testing.T) {
	p := NewSlackProvider(dbmock.New(), slog.Default())
	handler := p.BuildHandler("cfg-1", "secret", map[string]any{}, nil)

	req := httptest.NewRequest(http.MethodGet, "/hook/slack/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

// --- Slack BuildHandler: url_verification ---

func TestSlackHandler_URLVerification(t *testing.T) {
	p := NewSlackProvider(dbmock.New(), slog.Default())
	handler := p.BuildHandler("cfg-1", "secret", map[string]any{}, nil)

	payload := map[string]any{
		"type":      "url_verification",
		"challenge": "test-challenge-123",
		"token":     "verification-token",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/hook/slack/test", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("response decode error = %v", err)
	}
	if resp["challenge"] != "test-challenge-123" {
		t.Errorf("challenge = %q, want %q", resp["challenge"], "test-challenge-123")
	}
}

// --- Slack BuildHandler: signature verification failure ---

func TestSlackHandler_SignatureFailure(t *testing.T) {
	p := NewSlackProvider(dbmock.New(), slog.Default())
	cfg := map[string]any{"signing_secret": "my-signing-secret"}
	handler := p.BuildHandler("cfg-1", "secret", cfg, nil)

	payload := `{"type":"event_callback","event":{"type":"message","text":"hello"}}`

	req := httptest.NewRequest(http.MethodPost, "/hook/slack/test", strings.NewReader(payload))
	req.Header.Set("X-Slack-Request-Timestamp", "1234567890")
	req.Header.Set("X-Slack-Signature", "v0=invalid_signature")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", w.Code, http.StatusForbidden)
	}
}

// --- Slack BuildHandler: missing signature headers ---

func TestSlackHandler_MissingSignatureHeaders(t *testing.T) {
	p := NewSlackProvider(dbmock.New(), slog.Default())
	cfg := map[string]any{"signing_secret": "my-signing-secret"}
	handler := p.BuildHandler("cfg-1", "secret", cfg, nil)

	payload := `{"type":"event_callback","event":{"type":"message","text":"hello"}}`

	req := httptest.NewRequest(http.MethodPost, "/hook/slack/test", strings.NewReader(payload))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", w.Code, http.StatusForbidden)
	}
}

// --- Slack BuildHandler: valid signature + message event ---

func slackSignature(secret, timestamp, body string) string {
	baseString := fmt.Sprintf("v0:%s:%s", timestamp, body)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(baseString))
	return "v0=" + hex.EncodeToString(mac.Sum(nil))
}

func TestSlackHandler_ValidMessageEvent(t *testing.T) {
	p := NewSlackProvider(dbmock.New(), slog.Default())
	signingSecret := "test-signing-secret"
	cfg := map[string]any{"signing_secret": signingSecret}

	var mu sync.Mutex
	var received *IncomingMessage
	onMessage := func(_ context.Context, msg *IncomingMessage) {
		mu.Lock()
		received = msg
		mu.Unlock()
	}
	handler := p.BuildHandler("cfg-1", "secret", cfg, onMessage)

	payload := map[string]any{
		"type": "event_callback",
		"event": map[string]any{
			"type":    "message",
			"text":    "@opencode help me",
			"user":    "U123",
			"channel": "C456",
			"ts":      "1234567890.123456",
		},
	}
	body, _ := json.Marshal(payload)
	bodyStr := string(body)
	timestamp := "1234567890"
	sig := slackSignature(signingSecret, timestamp, bodyStr)

	req := httptest.NewRequest(http.MethodPost, "/hook/slack/test", strings.NewReader(bodyStr))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Slack-Request-Timestamp", timestamp)
	req.Header.Set("X-Slack-Signature", sig)
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
	if received.Provider != ProviderSlack {
		t.Errorf("Provider = %q, want %q", received.Provider, ProviderSlack)
	}
	if received.Body != "@opencode help me" {
		t.Errorf("Body = %q", received.Body)
	}
	if received.Author != "U123" {
		t.Errorf("Author = %q, want %q", received.Author, "U123")
	}
}

// --- Slack BuildHandler: non-event_callback type ---

func TestSlackHandler_NonEventCallback(t *testing.T) {
	p := NewSlackProvider(dbmock.New(), slog.Default())
	handler := p.BuildHandler("cfg-1", "secret", map[string]any{}, nil)

	payload := `{"type":"app_rate_limited"}`
	req := httptest.NewRequest(http.MethodPost, "/hook/slack/test", strings.NewReader(payload))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

// --- Slack BuildHandler: non-message event type ---

func TestSlackHandler_NonMessageEventType(t *testing.T) {
	p := NewSlackProvider(dbmock.New(), slog.Default())
	handler := p.BuildHandler("cfg-1", "secret", map[string]any{}, nil)

	payload := `{"type":"event_callback","event":{"type":"channel_created","channel":"C123"}}`
	req := httptest.NewRequest(http.MethodPost, "/hook/slack/test", strings.NewReader(payload))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

// --- Slack BuildHandler: empty user (bot message) ---

func TestSlackHandler_EmptyUser(t *testing.T) {
	p := NewSlackProvider(dbmock.New(), slog.Default())

	var called bool
	onMessage := func(_ context.Context, _ *IncomingMessage) {
		called = true
	}
	handler := p.BuildHandler("cfg-1", "secret", map[string]any{}, onMessage)

	payload := `{"type":"event_callback","event":{"type":"message","text":"bot msg","user":"","channel":"C123","ts":"123.456"}}`
	req := httptest.NewRequest(http.MethodPost, "/hook/slack/test", strings.NewReader(payload))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	time.Sleep(50 * time.Millisecond)
	if called {
		t.Error("onMessage should not be called for empty user")
	}
}

// --- Slack BuildHandler: malformed JSON ---

func TestSlackHandler_MalformedJSON(t *testing.T) {
	p := NewSlackProvider(dbmock.New(), slog.Default())
	handler := p.BuildHandler("cfg-1", "secret", map[string]any{}, nil)

	req := httptest.NewRequest(http.MethodPost, "/hook/slack/test", strings.NewReader("{invalid"))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

// --- Slack BuildHandler: thread_ts used when present ---

func TestSlackHandler_ThreadTS(t *testing.T) {
	p := NewSlackProvider(dbmock.New(), slog.Default())

	var mu sync.Mutex
	var received *IncomingMessage
	onMessage := func(_ context.Context, msg *IncomingMessage) {
		mu.Lock()
		received = msg
		mu.Unlock()
	}
	handler := p.BuildHandler("cfg-1", "secret", map[string]any{}, onMessage)

	payload := map[string]any{
		"type": "event_callback",
		"event": map[string]any{
			"type":      "message",
			"text":      "reply in thread",
			"user":      "U789",
			"channel":   "C456",
			"ts":        "111.222",
			"thread_ts": "100.000",
		},
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/hook/slack/test", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	time.Sleep(100 * time.Millisecond)
	mu.Lock()
	defer mu.Unlock()

	if received == nil {
		t.Fatal("onMessage was not called")
	}

	meta, ok := received.ReplyMeta.(slackReplyMeta)
	if !ok {
		t.Fatal("ReplyMeta is not slackReplyMeta")
	}
	if meta.ThreadTS != "100.000" {
		t.Errorf("ThreadTS = %q, want %q", meta.ThreadTS, "100.000")
	}
}

// --- Slack BuildHandler: no signing secret skips verification ---

func TestSlackHandler_NoSigningSecret(t *testing.T) {
	p := NewSlackProvider(dbmock.New(), slog.Default())
	handler := p.BuildHandler("cfg-1", "secret", map[string]any{}, nil)

	payload := `{"type":"app_rate_limited"}`
	req := httptest.NewRequest(http.MethodPost, "/hook/slack/test", strings.NewReader(payload))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d (should pass without signing_secret)", w.Code, http.StatusOK)
	}
}
