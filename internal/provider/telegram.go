package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/opencode-ai/opencode-dog/internal/db"
)

type TelegramProvider struct {
	database   db.Store
	logger     *slog.Logger
	httpClient *http.Client
	parseMode  string
}

func NewTelegramProvider(database db.Store, logger *slog.Logger) *TelegramProvider {
	timeout := database.GetSettingDuration(context.Background(), "telegram_http_timeout", 30*time.Second)
	parseMode := database.GetSettingString(context.Background(), "telegram_parse_mode", "Markdown")
	return &TelegramProvider{
		database:   database,
		logger:     logger,
		httpClient: &http.Client{Timeout: timeout},
		parseMode:  parseMode,
	}
}

func (t *TelegramProvider) Type() ProviderType { return ProviderTelegram }

func (t *TelegramProvider) ValidateConfig(cfg map[string]any) error {
	if _, ok := cfg["bot_token"]; !ok {
		return fmt.Errorf("missing required field: bot_token")
	}
	return nil
}

type telegramUpdate struct {
	UpdateID int `json:"update_id"`
	Message  *struct {
		MessageID int `json:"message_id"`
		From      struct {
			ID       int    `json:"id"`
			Username string `json:"username"`
		} `json:"from"`
		Chat struct {
			ID    int64  `json:"id"`
			Title string `json:"title"`
			Type  string `json:"type"`
		} `json:"chat"`
		Text string `json:"text"`
	} `json:"message"`
}

type telegramReplyMeta struct {
	ChatID    int64 `json:"chat_id"`
	MessageID int   `json:"message_id"`
}

func (t *TelegramProvider) BuildHandler(providerCfgID string, secret string, cfg map[string]any, onMessage func(context.Context, *IncomingMessage)) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		if secret != "" {
			token := r.Header.Get("X-Telegram-Bot-Api-Secret-Token")
			if token != secret {
				t.logger.Warn("telegram secret token mismatch", "provider_cfg", providerCfgID)
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}

		var update telegramUpdate
		if err := json.Unmarshal(body, &update); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}

		w.WriteHeader(http.StatusOK)

		if update.Message == nil || update.Message.Text == "" {
			return
		}

		meta := telegramReplyMeta{
			ChatID:    update.Message.Chat.ID,
			MessageID: update.Message.MessageID,
		}

		chatTitle := update.Message.Chat.Title
		if chatTitle == "" {
			chatTitle = fmt.Sprintf("Chat %d", update.Message.Chat.ID)
		}

		msg := &IncomingMessage{
			Provider:      ProviderTelegram,
			ProviderCfgID: providerCfgID,
			ExternalRef:   fmt.Sprintf("tg://chat/%d/msg/%d", update.Message.Chat.ID, update.Message.MessageID),
			Title:         chatTitle,
			Body:          update.Message.Text,
			Author:        update.Message.From.Username,
			ReplyMeta:     meta,
		}

		go onMessage(context.Background(), msg)
	})
}

func (t *TelegramProvider) SendReply(ctx context.Context, cfg map[string]any, msg *IncomingMessage, body string) error {
	botToken, _ := cfg["bot_token"].(string)
	if botToken == "" {
		return fmt.Errorf("missing bot_token in config")
	}

	var meta telegramReplyMeta
	raw, _ := json.Marshal(msg.ReplyMeta)
	if err := json.Unmarshal(raw, &meta); err != nil {
		return fmt.Errorf("invalid reply meta: %w", err)
	}

	payload := map[string]any{
		"chat_id":                  meta.ChatID,
		"reply_to_message_id":      meta.MessageID,
		"text":                     body,
		"parse_mode":               t.parseMode,
		"disable_web_page_preview": true,
	}

	jsonBody, _ := json.Marshal(payload)
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", botToken)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(jsonBody))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := t.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("telegram api call failed: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		OK          bool   `json:"ok"`
		Description string `json:"description"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}
	if !result.OK {
		return fmt.Errorf("telegram api error: %s", result.Description)
	}
	return nil
}
