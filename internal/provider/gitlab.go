package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	gogitlab "github.com/xanzy/go-gitlab"
)

type GitLabProvider struct {
	logger *slog.Logger
}

func NewGitLabProvider(logger *slog.Logger) *GitLabProvider {
	return &GitLabProvider{logger: logger}
}

func (g *GitLabProvider) Type() ProviderType { return ProviderGitLab }

func (g *GitLabProvider) ValidateConfig(cfg map[string]any) error {
	required := []string{"base_url", "token"}
	for _, k := range required {
		if _, ok := cfg[k]; !ok {
			return fmt.Errorf("missing required field: %s", k)
		}
	}
	return nil
}

type gitlabReplyMeta struct {
	ProjectID int `json:"project_id"`
	IssueIID  int `json:"issue_iid"`
}

func (g *GitLabProvider) BuildHandler(providerCfgID string, secret string, cfg map[string]any, onMessage func(context.Context, *IncomingMessage)) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		token := gogitlab.HookEventToken(r)
		if token != secret {
			g.logger.Warn("gitlab webhook token mismatch", "provider_cfg", providerCfgID)
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}

		eventType := gogitlab.HookEventType(r)
		if eventType != gogitlab.EventTypeNote && eventType != gogitlab.EventConfidentialNote {
			w.WriteHeader(http.StatusOK)
			return
		}

		payload, err := io.ReadAll(r.Body)
		if err != nil || len(payload) == 0 {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}

		event, err := gogitlab.ParseWebhook(eventType, payload)
		if err != nil {
			g.logger.Error("gitlab parse webhook failed", "error", err)
			http.Error(w, "unprocessable", http.StatusUnprocessableEntity)
			return
		}

		w.WriteHeader(http.StatusOK)

		issueComment, ok := event.(*gogitlab.IssueCommentEvent)
		if !ok || issueComment.ObjectAttributes.System {
			return
		}

		webURL := ""
		if issueComment.ObjectAttributes.URL != "" {
			webURL = issueComment.ObjectAttributes.URL
		} else {
			webURL = fmt.Sprintf("%s/-/issues/%d", issueComment.Project.WebURL, issueComment.Issue.IID)
		}

		meta := gitlabReplyMeta{
			ProjectID: issueComment.ProjectID,
			IssueIID:  issueComment.Issue.IID,
		}

		msg := &IncomingMessage{
			Provider:      ProviderGitLab,
			ProviderCfgID: providerCfgID,
			ExternalRef:   webURL,
			Title:         issueComment.Issue.Title,
			Body:          issueComment.ObjectAttributes.Note,
			Author:        issueComment.User.Username,
			ReplyMeta:     meta,
		}

		go onMessage(context.Background(), msg)
	})
}

func (g *GitLabProvider) SendReply(ctx context.Context, cfg map[string]any, msg *IncomingMessage, body string) error {
	baseURL, _ := cfg["base_url"].(string)
	token, _ := cfg["token"].(string)

	client, err := gogitlab.NewClient(token, gogitlab.WithBaseURL(baseURL))
	if err != nil {
		return fmt.Errorf("create gitlab client: %w", err)
	}

	var meta gitlabReplyMeta
	raw, _ := json.Marshal(msg.ReplyMeta)
	if err := json.Unmarshal(raw, &meta); err != nil {
		return fmt.Errorf("invalid reply meta: %w", err)
	}

	_, _, err = client.Notes.CreateIssueNote(
		meta.ProjectID,
		meta.IssueIID,
		&gogitlab.CreateIssueNoteOptions{Body: gogitlab.Ptr(body)},
	)
	return err
}
