package analyzer

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"
)

type OpencodeClient struct {
	baseURL    string
	httpClient *http.Client
	authHeader string
	logger     *slog.Logger
}

func NewOpencodeClient(baseURL, username, password string, timeout time.Duration, logger *slog.Logger) *OpencodeClient {
	auth := base64.StdEncoding.EncodeToString([]byte(username + ":" + password))

	return &OpencodeClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		authHeader: "Basic " + auth,
		logger:     logger,
	}
}

type Session struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

type MessagePart struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type MessageRequest struct {
	Parts []MessagePart `json:"parts"`
}

type MessageResponse struct {
	Info struct {
		ID      string `json:"id"`
		Role    string `json:"role"`
		Content string `json:"content"`
		Error   *struct {
			Name    string `json:"name"`
			Message string `json:"message"`
		} `json:"error,omitempty"`
	} `json:"info"`
	Parts []MessagePart `json:"parts"`
}

func (c *OpencodeClient) CreateSession(ctx context.Context, title string) (*Session, error) {
	reqBody := map[string]string{"title": title}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/session", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", c.authHeader)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("create session failed: status %d: %s", resp.StatusCode, string(body))
	}

	var session Session
	if err := json.NewDecoder(resp.Body).Decode(&session); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	c.logger.Info("opencode session created", "session_id", session.ID, "title", title)
	return &session, nil
}

func (c *OpencodeClient) SendMessage(ctx context.Context, sessionID, prompt string) (string, error) {
	reqBody := MessageRequest{
		Parts: []MessagePart{
			{Type: "text", Text: prompt},
		},
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/session/%s/message", c.baseURL, sessionID)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", c.authHeader)

	c.logger.Info("sending message to opencode", "session_id", sessionID, "prompt_len", len(prompt))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("send message failed: status %d: %s", resp.StatusCode, string(body))
	}

	var msgResp MessageResponse
	if err := json.NewDecoder(resp.Body).Decode(&msgResp); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}

	if msgResp.Info.Error != nil {
		return "", fmt.Errorf("opencode error: %s: %s", msgResp.Info.Error.Name, msgResp.Info.Error.Message)
	}

	var result string
	for _, part := range msgResp.Parts {
		if part.Type == "text" {
			result += part.Text
		}
	}

	if result == "" {
		return "", fmt.Errorf("opencode returned empty response")
	}

	c.logger.Info("received response from opencode", "session_id", sessionID, "response_len", len(result))
	return result, nil
}

func (c *OpencodeClient) DeleteSession(ctx context.Context, sessionID string) error {
	url := fmt.Sprintf("%s/session/%s", c.baseURL, sessionID)
	req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Authorization", c.authHeader)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("delete session failed: status %d: %s", resp.StatusCode, string(body))
	}

	c.logger.Info("opencode session deleted", "session_id", sessionID)
	return nil
}
