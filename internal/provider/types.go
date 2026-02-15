package provider

import (
	"context"
	"net/http"
)

type TriggerMode string

const (
	ModeAsk  TriggerMode = "ask"
	ModePlan TriggerMode = "plan"
	ModeDo   TriggerMode = "do"
)

type ProviderType string

const (
	ProviderGitLab   ProviderType = "gitlab"
	ProviderSlack    ProviderType = "slack"
	ProviderTelegram ProviderType = "telegram"
)

type IncomingMessage struct {
	Provider       ProviderType
	ProviderCfgID  string
	ProjectID      string
	ExternalRef    string
	Title          string
	Body           string
	Author         string
	TriggerMode    TriggerMode
	TriggerKeyword string
	ReplyMeta      any
}

type Provider interface {
	Type() ProviderType
	ValidateConfig(cfg map[string]any) error
	BuildHandler(providerCfgID string, secret string, cfg map[string]any, onMessage func(context.Context, *IncomingMessage)) http.Handler
	SendReply(ctx context.Context, cfg map[string]any, msg *IncomingMessage, body string) error
}
