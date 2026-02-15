package provider

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

type SlackProvider struct {
	logger     *slog.Logger
	httpClient *http.Client
}

func NewSlackProvider(logger *slog.Logger) *SlackProvider {
	return &SlackProvider{
		logger:     logger,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

func (s *SlackProvider) Type() ProviderType { return ProviderSlack }

func (s *SlackProvider) ValidateConfig(cfg map[string]any) error {
	required := []string{"bot_token", "signing_secret"}
	for _, k := range required {
		if _, ok := cfg[k]; !ok {
			return fmt.Errorf("missing required field: %s", k)
		}
	}
	return nil
}

type slackEvent struct {
	Token     string `json:"token"`
	Challenge string `json:"challenge"`
	Type      string `json:"type"`
	Event     struct {
		Type    string `json:"type"`
		Text    string `json:"text"`
		User    string `json:"user"`
		Channel string `json:"channel"`
		TS      string `json:"ts"`
		ThreadTS string `json:"thread_ts"`
	} `json:"event"`
}

type slackReplyMeta struct {
	Channel  string `json:"channel"`
	ThreadTS string `json:"thread_ts"`
}

func (s *SlackProvider) BuildHandler(providerCfgID string, secret string, cfg map[string]any, onMessage func(context.Context, *IncomingMessage)) http.Handler {
	signingSecret, _ := cfg["signing_secret"].(string)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}

		if signingSecret != "" && !s.verifySlackSignature(r, body, signingSecret) {
			s.logger.Warn("slack signature verification failed", "provider_cfg", providerCfgID)
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}

		var evt slackEvent
		if err := json.Unmarshal(body, &evt); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}

		if evt.Type == "url_verification" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{"challenge": evt.Challenge})
			return
		}

		w.WriteHeader(http.StatusOK)

		if evt.Type != "event_callback" {
			return
		}

		if evt.Event.Type != "message" && evt.Event.Type != "app_mention" {
			return
		}

		if evt.Event.User == "" {
			return
		}

		threadTS := evt.Event.ThreadTS
		if threadTS == "" {
			threadTS = evt.Event.TS
		}

		meta := slackReplyMeta{
			Channel:  evt.Event.Channel,
			ThreadTS: threadTS,
		}

		msg := &IncomingMessage{
			Provider:      ProviderSlack,
			ProviderCfgID: providerCfgID,
			ExternalRef:   fmt.Sprintf("slack://%s/%s", evt.Event.Channel, evt.Event.TS),
			Title:         fmt.Sprintf("Slack message in #%s", evt.Event.Channel),
			Body:          evt.Event.Text,
			Author:        evt.Event.User,
			ReplyMeta:     meta,
		}

		go onMessage(context.Background(), msg)
	})
}

func (s *SlackProvider) verifySlackSignature(r *http.Request, body []byte, signingSecret string) bool {
	timestamp := r.Header.Get("X-Slack-Request-Timestamp")
	sig := r.Header.Get("X-Slack-Signature")
	if timestamp == "" || sig == "" {
		return false
	}

	baseString := fmt.Sprintf("v0:%s:%s", timestamp, string(body))
	mac := hmac.New(sha256.New, []byte(signingSecret))
	mac.Write([]byte(baseString))
	expected := "v0=" + hex.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(expected), []byte(sig))
}

func (s *SlackProvider) SendReply(ctx context.Context, cfg map[string]any, msg *IncomingMessage, body string) error {
	botToken, _ := cfg["bot_token"].(string)
	if botToken == "" {
		return fmt.Errorf("missing bot_token in config")
	}

	var meta slackReplyMeta
	raw, _ := json.Marshal(msg.ReplyMeta)
	if err := json.Unmarshal(raw, &meta); err != nil {
		return fmt.Errorf("invalid reply meta: %w", err)
	}

	payload := map[string]any{
		"channel":   meta.Channel,
		"thread_ts": meta.ThreadTS,
		"text":      body,
		"mrkdwn":    true,
	}

	jsonBody, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://slack.com/api/chat.postMessage", bytes.NewReader(jsonBody))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+botToken)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("slack api call failed: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		OK    bool   `json:"ok"`
		Error string `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}
	if !result.OK {
		return fmt.Errorf("slack api error: %s", result.Error)
	}
	return nil
}

func init() {
	_ = strings.NewReader
}
