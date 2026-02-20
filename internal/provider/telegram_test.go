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

	"github.com/opencode-ai/opencode-dog/internal/db/dbmock"
)

// --- TelegramProvider Type ---

func TestTelegramProvider_Type(t *testing.T) {
	p := NewTelegramProvider(dbmock.New(), slog.Default())
	if p.Type() != ProviderTelegram {
		t.Errorf("Type() = %q, want %q", p.Type(), ProviderTelegram)
	}
}

// --- TelegramProvider ValidateConfig ---

func TestTelegramProvider_ValidateConfig_Valid(t *testing.T) {
	p := NewTelegramProvider(dbmock.New(), slog.Default())
	cfg := map[string]any{"bot_token": "123:ABC"}
	if err := p.ValidateConfig(cfg); err != nil {
		t.Errorf("ValidateConfig() error = %v", err)
	}
}

func TestTelegramProvider_ValidateConfig_MissingBotToken(t *testing.T) {
	p := NewTelegramProvider(dbmock.New(), slog.Default())
	err := p.ValidateConfig(map[string]any{})
	if err == nil {
		t.Fatal("expected error for missing bot_token")
	}
	if !strings.Contains(err.Error(), "bot_token") {
		t.Errorf("error = %q, want mention of bot_token", err.Error())
	}
}

// --- Telegram BuildHandler: method not allowed ---

func TestTelegramHandler_MethodNotAllowed(t *testing.T) {
	p := NewTelegramProvider(dbmock.New(), slog.Default())
	handler := p.BuildHandler("cfg-1", "secret", nil, nil)

	for _, method := range []string{http.MethodGet, http.MethodPut, http.MethodDelete} {
		req := httptest.NewRequest(method, "/hook/telegram/test", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("%s: status = %d, want %d", method, w.Code, http.StatusMethodNotAllowed)
		}
	}
}

// --- Telegram BuildHandler: secret token verification ---

func TestTelegramHandler_InvalidSecret(t *testing.T) {
	p := NewTelegramProvider(dbmock.New(), slog.Default())
	handler := p.BuildHandler("cfg-1", "correct-secret", nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/hook/telegram/test", strings.NewReader("{}"))
	req.Header.Set("X-Telegram-Bot-Api-Secret-Token", "wrong-secret")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", w.Code, http.StatusForbidden)
	}
}

func TestTelegramHandler_MissingSecret(t *testing.T) {
	p := NewTelegramProvider(dbmock.New(), slog.Default())
	handler := p.BuildHandler("cfg-1", "my-secret", nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/hook/telegram/test", strings.NewReader("{}"))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", w.Code, http.StatusForbidden)
	}
}

// --- Telegram BuildHandler: empty secret skips verification ---

func TestTelegramHandler_EmptySecretSkipsVerification(t *testing.T) {
	p := NewTelegramProvider(dbmock.New(), slog.Default())
	handler := p.BuildHandler("cfg-1", "", nil, nil)

	payload := `{"update_id":1,"message":null}`
	req := httptest.NewRequest(http.MethodPost, "/hook/telegram/test", strings.NewReader(payload))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

// --- Telegram BuildHandler: valid message ---

func TestTelegramHandler_ValidMessage(t *testing.T) {
	p := NewTelegramProvider(dbmock.New(), slog.Default())

	var mu sync.Mutex
	var received *IncomingMessage
	onMessage := func(_ context.Context, msg *IncomingMessage) {
		mu.Lock()
		received = msg
		mu.Unlock()
	}
	handler := p.BuildHandler("cfg-1", "secret", nil, onMessage)

	payload := map[string]any{
		"update_id": 100,
		"message": map[string]any{
			"message_id": 42,
			"from":       map[string]any{"id": 1001, "username": "alice"},
			"chat":       map[string]any{"id": 12345, "title": "Dev Chat", "type": "group"},
			"text":       "@opencode analyze this bug",
		},
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/hook/telegram/test", strings.NewReader(string(body)))
	req.Header.Set("X-Telegram-Bot-Api-Secret-Token", "secret")
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
	if received.Provider != ProviderTelegram {
		t.Errorf("Provider = %q, want %q", received.Provider, ProviderTelegram)
	}
	if received.ProviderCfgID != "cfg-1" {
		t.Errorf("ProviderCfgID = %q", received.ProviderCfgID)
	}
	if received.Body != "@opencode analyze this bug" {
		t.Errorf("Body = %q", received.Body)
	}
	if received.Author != "alice" {
		t.Errorf("Author = %q, want %q", received.Author, "alice")
	}
	if received.Title != "Dev Chat" {
		t.Errorf("Title = %q, want %q", received.Title, "Dev Chat")
	}
	if received.ExternalRef != "tg://chat/12345/msg/42" {
		t.Errorf("ExternalRef = %q", received.ExternalRef)
	}

	meta, ok := received.ReplyMeta.(telegramReplyMeta)
	if !ok {
		t.Fatal("ReplyMeta is not telegramReplyMeta")
	}
	if meta.ChatID != 12345 {
		t.Errorf("ChatID = %d, want 12345", meta.ChatID)
	}
	if meta.MessageID != 42 {
		t.Errorf("MessageID = %d, want 42", meta.MessageID)
	}
}

// --- Telegram BuildHandler: null message ---

func TestTelegramHandler_NullMessage(t *testing.T) {
	p := NewTelegramProvider(dbmock.New(), slog.Default())

	var called bool
	onMessage := func(_ context.Context, _ *IncomingMessage) {
		called = true
	}
	handler := p.BuildHandler("cfg-1", "secret", nil, onMessage)

	payload := `{"update_id":1}`
	req := httptest.NewRequest(http.MethodPost, "/hook/telegram/test", strings.NewReader(payload))
	req.Header.Set("X-Telegram-Bot-Api-Secret-Token", "secret")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	time.Sleep(50 * time.Millisecond)
	if called {
		t.Error("onMessage should not be called for null message")
	}
}

// --- Telegram BuildHandler: empty text ---

func TestTelegramHandler_EmptyText(t *testing.T) {
	p := NewTelegramProvider(dbmock.New(), slog.Default())

	var called bool
	onMessage := func(_ context.Context, _ *IncomingMessage) {
		called = true
	}
	handler := p.BuildHandler("cfg-1", "secret", nil, onMessage)

	payload := map[string]any{
		"update_id": 1,
		"message": map[string]any{
			"message_id": 1,
			"from":       map[string]any{"id": 1, "username": "user"},
			"chat":       map[string]any{"id": 1, "title": "Chat", "type": "group"},
			"text":       "",
		},
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/hook/telegram/test", strings.NewReader(string(body)))
	req.Header.Set("X-Telegram-Bot-Api-Secret-Token", "secret")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	time.Sleep(50 * time.Millisecond)
	if called {
		t.Error("onMessage should not be called for empty text")
	}
}

// --- Telegram BuildHandler: malformed JSON ---

func TestTelegramHandler_MalformedJSON(t *testing.T) {
	p := NewTelegramProvider(dbmock.New(), slog.Default())
	handler := p.BuildHandler("cfg-1", "secret", nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/hook/telegram/test", strings.NewReader("{bad"))
	req.Header.Set("X-Telegram-Bot-Api-Secret-Token", "secret")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

// --- Telegram BuildHandler: chat without title ---

func TestTelegramHandler_ChatWithoutTitle(t *testing.T) {
	p := NewTelegramProvider(dbmock.New(), slog.Default())

	var mu sync.Mutex
	var received *IncomingMessage
	onMessage := func(_ context.Context, msg *IncomingMessage) {
		mu.Lock()
		received = msg
		mu.Unlock()
	}
	handler := p.BuildHandler("cfg-1", "secret", nil, onMessage)

	payload := map[string]any{
		"update_id": 1,
		"message": map[string]any{
			"message_id": 1,
			"from":       map[string]any{"id": 1, "username": "user"},
			"chat":       map[string]any{"id": 99, "type": "private"},
			"text":       "hello",
		},
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/hook/telegram/test", strings.NewReader(string(body)))
	req.Header.Set("X-Telegram-Bot-Api-Secret-Token", "secret")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	time.Sleep(100 * time.Millisecond)
	mu.Lock()
	defer mu.Unlock()

	if received == nil {
		t.Fatal("onMessage was not called")
	}
	if received.Title != "Chat 99" {
		t.Errorf("Title = %q, want %q", received.Title, "Chat 99")
	}
}
