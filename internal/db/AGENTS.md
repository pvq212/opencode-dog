# 資料庫層

## 概覽

PostgreSQL CRUD 封裝（pgx v5 連線池）。手寫 SQL，無 ORM。所有資料存取都經過此層。

## 檔案

- `models.go` — 9 個結構體 + 2 組常數（TaskStatus、MCPServerStatus、角色）
- `db.go` — 536 行，全部 CRUD 方法 + 連線池初始化 + Migration 執行

## 連線池設定

```go
MaxConns: 20, MinConns: 2, MaxConnLifetime: 30 min
```

## 資料表對應

| 結構體 | 資料表 | 主要操作 |
|--------|--------|----------|
| `SSHKey` | `ssh_keys` | Create/List/Get/Delete |
| `Project` | `projects` | CRUD |
| `ProviderConfig` | `provider_configs` | Create/List/Get/GetByPath/ListAll/Delete |
| `TriggerKeyword` | `trigger_keywords` | Set（事務：先刪後插）/Get |
| `Task` | `tasks` | Create/UpdateStatus/List/Get/Count |
| `WebhookDelivery` | `webhook_deliveries` | IsProcessed/Record（ON CONFLICT DO NOTHING） |
| `Setting` | `settings` | Get/GetBool/GetString/Set（UPSERT）/List/Delete |
| `MCPServer` | `mcp_servers` | CRUD + UpdateStatus/ListEnabled |
| `User` | `users` | CRUD + GetByUsername/UpdatePassword/Count |

## 慣例

- 所有 ID 為 UUID，由 PostgreSQL `gen_random_uuid()` 產生
- 回傳值用指標切片 `[]*Model`
- `QueryRow` + `Scan` 模式讀取單行
- `Query` + `rows.Next()` + `Scan` 模式讀取多行
- 錯誤直接回傳，不包裝（呼叫端自行處理）
- `ListSSHKeys` 不回傳 `private_key`（安全考量）
- `User.PasswordHash` 標記 `json:"-"`（不序列化）

## 特殊方法

- `SetTriggerKeywords` — 用事務（`Begin` → 刪除舊的 → 插入新的 → `Commit`）
- `RunMigrations` — 直接讀取 SQL 檔案並 `Exec`
- `ConfigMap()` — `ProviderConfig` 上的輔助方法，將 JSON 轉 `map[string]any`
- `HashPayload` / `ToJSON` — 獨立工具函式

## 注意

- 無事務封裝工具，需要事務時直接操作 `Pool.Begin()`
- `ListTasks` 支援分頁（LIMIT/OFFSET），其他 List 方法不支援
- `Setting.Value` 型別為 `json.RawMessage`，存任意 JSON
