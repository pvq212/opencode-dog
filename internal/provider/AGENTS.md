# Provider 渠道層

## 概覽

多渠道 Webhook 接收的抽象層。定義 `Provider` 介面，各渠道獨立實作驗證、解析、回覆邏輯。

## 架構

```
Provider 介面（types.go）
    ├── GitLabProvider（gitlab.go）  — go-gitlab SDK 解析 Note Event
    ├── SlackProvider（slack.go）    — HMAC-SHA256 驗籤 + Event API
    └── TelegramProvider（telegram.go）— Secret Token 驗證 + Bot API
            ↕
    Registry（registry.go）— 以 sync.RWMutex 保護的 map[ProviderType]Provider
```

## Provider 介面

```go
type Provider interface {
    Type() ProviderType
    ValidateConfig(cfg map[string]any) error
    BuildHandler(providerCfgID, secret string, cfg map[string]any, onMessage func(ctx, *IncomingMessage)) http.Handler
    SendReply(ctx context.Context, cfg map[string]any, msg *IncomingMessage, body string) error
}
```

## 新增渠道步驟

1. 在 `types.go` 新增 `ProviderXxx ProviderType = "xxx"` 常數
2. 建立 `xxx.go`，實作 `Provider` 介面四個方法
3. 在 `internal/server/server.go` 的 `New()` 中呼叫 `registry.Register(provider.NewXxxProvider(logger))`
4. 對應的 Webhook 驗證邏輯寫在 `BuildHandler` 內

## 各渠道驗證方式

| 渠道 | 驗證方式 | 設定欄位 |
|------|----------|----------|
| GitLab | `X-Gitlab-Token` header 比對 | `webhook_secret` |
| Slack | `X-Slack-Signature` HMAC-SHA256 | `signing_secret` + `bot_token` |
| Telegram | `X-Telegram-Bot-Api-Secret-Token` header | `secret_token` + `bot_token` |

## 關鍵型別

- `IncomingMessage` — 統一的訊息結構，含 Provider/ProjectID/Body/Author/TriggerMode/ReplyMeta
- `TriggerMode` — `ask` / `plan` / `do` 三種模式
- `ReplyMeta` — `any` 型別，各渠道存不同回覆元資料（GitLab issue ID、Slack channel+ts、Telegram chat_id+message_id）

## 注意

- `Config` 欄位是 `json.RawMessage`，各渠道所需 key 不同，透過 `ConfigMap()` 轉 `map[string]any`
- `Registry` 用讀寫鎖，支援並行讀取
- Webhook 去重依賴 `db.IsWebhookProcessed()` + `db.RecordWebhookDelivery()`
