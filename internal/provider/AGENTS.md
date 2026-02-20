# Provider Layer

## Overview

Multi-channel webhook abstraction. Defines `Provider` interface; each channel independently implements validation, parsing, and reply logic. 67.8% test coverage across 5 test files.

## Architecture

```
Provider interface (types.go)
    ├── GitLabProvider (gitlab.go)    — go-gitlab SDK parses Note Event
    ├── SlackProvider (slack.go)      — HMAC-SHA256 signature verification + Event API
    └── TelegramProvider (telegram.go)— Secret Token verification + Bot API
            ↕
    Registry (registry.go)           — sync.RWMutex-protected map[ProviderType]Provider
```

## Provider Interface

```go
type Provider interface {
    Type() ProviderType
    ValidateConfig(cfg map[string]any) error
    BuildHandler(providerCfgID, secret string, cfg map[string]any, onMessage func(ctx, *IncomingMessage)) http.Handler
    SendReply(ctx context.Context, cfg map[string]any, msg *IncomingMessage, body string) error
}
```

## Adding a New Channel

1. Add `ProviderXxx ProviderType = "xxx"` constant in `types.go`
2. Create `xxx.go`, implement all 4 `Provider` interface methods
3. In `internal/server/server.go` `New()`, call `registry.Register(provider.NewXxxProvider(logger))`
4. Webhook verification logic goes inside `BuildHandler`

## Channel Verification

| Channel | Verification | Config Fields |
|---------|-------------|---------------|
| GitLab | `X-Gitlab-Token` header match | `webhook_secret` |
| Slack | `X-Slack-Signature` HMAC-SHA256 | `signing_secret` + `bot_token` |
| Telegram | `X-Telegram-Bot-Api-Secret-Token` header | `secret_token` + `bot_token` |

## Key Types

- `IncomingMessage` — Unified message struct: Provider/ProjectID/Body/Author/TriggerMode/ReplyMeta
- `TriggerMode` — `ask` / `plan` / `do`
- `ReplyMeta` — `any` type, each channel stores different reply metadata (GitLab issue ID, Slack channel+ts, Telegram chat_id+message_id)

## Notes

- `Config` field is `json.RawMessage`; each channel needs different keys, accessed via `ConfigMap()` → `map[string]any`
- `Registry` uses RWMutex for concurrent read access
- Webhook dedup relies on `db.IsWebhookProcessed()` + `db.RecordWebhookDelivery()`
