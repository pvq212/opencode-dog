# OpenCode GitLab Bot

多渠道 AI 程式碼分析機器人 — 透過 GitLab / Slack / Telegram 的 Webhook 觸發 [OpenCode](https://github.com/opencode-ai/opencode) 自動排查問題，並將分析結果回覆至來源渠道。

## 功能特色

- **多渠道支援** — GitLab（Issue 留言）、Slack（Event Subscription）、Telegram（Bot Webhook）
- **可設定觸發關鍵字** — 依專案設定不同關鍵字對應不同模式（提問 `ask`、計畫 `plan`、執行 `do`）
- **OpenCode 整合** — 以子程序方式呼叫 OpenCode CLI，自動注入設定檔與 MCP 伺服器配置
- **React Admin 後台** — 完整的 Web UI 管理介面，含 RBAC 權限控制（Admin / Editor / Viewer）
- **MCP 伺服器管理** — 在後台安裝、啟用、停用 MCP npm 套件
- **設定檔線上編輯** — auth.json、.opencode.json、oh-my-opencode.json 可在 WebUI 直接編輯
- **全部設定進資料庫** — 渠道設定、觸發詞、MCP 配置皆存於 PostgreSQL，不依賴 .env
- **Docker 打包部署** — 含 Go + Node.js 執行環境，一鍵 docker compose 啟動

## 架構概覽

```
┌─────────────┐  ┌─────────────┐  ┌──────────────┐
│   GitLab    │  │    Slack    │  │   Telegram   │
│  Webhook    │  │  Webhook    │  │   Webhook    │
└──────┬──────┘  └──────┬──────┘  └──────┬───────┘
       │                │                │
       └────────────────┼────────────────┘
                        ▼
              ┌──────────────────┐
              │  Provider Layer  │  ← 抽象化渠道來源
              │ (gitlab/slack/tg)│
              └────────┬─────────┘
                       ▼
              ┌──────────────────┐
              │    Analyzer      │  ← 呼叫 OpenCode CLI
              │  (subprocess)    │
              └────────┬─────────┘
                       ▼
              ┌──────────────────┐
              │   PostgreSQL     │  ← 所有設定 & 任務紀錄
              └──────────────────┘
                       ▲
              ┌──────────────────┐
              │  React Admin UI  │  ← 嵌入 Go binary
              │   (WebUI)        │
              └──────────────────┘
```

## 快速開始

### 使用 Docker Compose（推薦）

```bash
# 1. 複製環境變數
cp .env.example .env

# 2. 修改 .env 中的密碼與 JWT 密鑰
#    DB_PASSWORD=your-strong-password
#    JWT_SECRET=your-random-secret-string

# 3. 啟動
docker compose up -d

# 4. 開啟瀏覽器
#    http://localhost:8080
#    預設帳號：admin / admin（首次登入請立即修改密碼）
```

### 本機開發

**前置需求：** Go 1.24+、Node.js 22+、PostgreSQL 16+

```bash
# 1. 建立資料庫
createdb opencode_gitlab

# 2. 執行遷移
psql -d opencode_gitlab -f migrations/001_init.sql

# 3. 編譯前端
cd web && npm install && npm run build && cd ..

# 4. 啟動伺服器
export DB_PASSWORD=your-password
export JWT_SECRET=your-secret
export OPENCODE_CONFIG_DIR=/tmp/opencode-config
go run ./cmd/server
```

## 環境變數

僅基礎設施相關的設定放在環境變數，其餘全部透過 WebUI 管理。

| 變數 | 說明 | 預設值 |
|------|------|--------|
| `SERVER_PORT` | 伺服器埠號 | `8080` |
| `SERVER_HOST` | 綁定地址 | `0.0.0.0` |
| `DB_HOST` | PostgreSQL 主機 | `localhost` |
| `DB_PORT` | PostgreSQL 埠號 | `5432` |
| `DB_USER` | 資料庫使用者 | `opencode` |
| `DB_PASSWORD` | 資料庫密碼 | — |
| `DB_NAME` | 資料庫名稱 | `opencode_gitlab` |
| `DB_SSLMODE` | SSL 模式 | `disable` |
| `JWT_SECRET` | JWT 簽名密鑰 | — |
| `OPENCODE_CONFIG_DIR` | OpenCode 設定檔目錄 | `/app/config` |

## WebUI 管理功能

登入後可在管理後台操作以下功能：

| 功能 | 說明 | 權限 |
|------|------|------|
| **Dashboard** | 系統概覽，專案數量、任務統計、近期任務 | 所有角色 |
| **Projects** | 管理專案（SSH URL、分支、啟停用） | Admin 可編輯 |
| **SSH Keys** | 管理 SSH 金鑰（用於 git clone） | Admin |
| **Tasks** | 檢視所有分析任務，含狀態、來源、結果 | 所有角色 |
| **Settings** | 系統設定、OpenCode 設定檔編輯（JSON 編輯器） | Admin |
| **MCP Servers** | 安裝 / 管理 MCP 伺服器（npm 套件） | Admin |
| **Users** | 使用者管理（RBAC：Admin / Editor / Viewer） | Admin |
| **Guides** | 各渠道接入教學（GitLab / Slack / Telegram） | 所有角色 |

## 渠道設定指南

### GitLab

1. 在 WebUI 建立 Project（填入 SSH URL）
2. 為該 Project 新增 Provider，類型選 `gitlab`，設定 `webhook_secret`
3. 前往 GitLab 專案 → Settings → Webhooks
4. URL 填入 `https://YOUR_DOMAIN/hook/gitlab/{project_id_prefix}`
5. Secret Token 填入步驟 2 設定的 `webhook_secret`
6. 勾選 **Note events**
7. 在 Issue 留言中輸入 `@opencode 請分析這個問題` 即可觸發

### Slack

1. 建立 Slack App，啟用 Event Subscriptions
2. 在 WebUI 新增 Provider，類型選 `slack`，設定 `bot_token` 和 `signing_secret`
3. Request URL 填入 `https://YOUR_DOMAIN/hook/slack/{project_id_prefix}`
4. 訂閱 `message.channels` 事件
5. 在頻道中 `@opencode 請分析這個問題` 即可觸發

### Telegram

1. 透過 [@BotFather](https://t.me/BotFather) 建立 Bot，取得 Token
2. 在 WebUI 新增 Provider，類型選 `telegram`，設定 `bot_token`
3. 設定 Webhook：`https://api.telegram.org/bot<TOKEN>/setWebhook?url=https://YOUR_DOMAIN/hook/telegram/{project_id_prefix}`
4. 在群組中 `@opencode 請分析這個問題` 即可觸發

## 觸發關鍵字

在 WebUI 的 Project 頁面可設定多組觸發關鍵字，每個關鍵字對應一種模式：

| 模式 | 說明 | 範例關鍵字 |
|------|------|-----------|
| `ask` | 提問模式 — 分析問題並回覆結果 | `@opencode`、`@ask` |
| `plan` | 計畫模式 — 產生修復計畫 | `@plan` |
| `do` | 執行模式 — 直接修改程式碼 | `@do` |

## 專案結構

```
opencode-gitlab-bot/
├── cmd/server/main.go              # 入口點（啟動時自動執行 migration）
├── internal/
│   ├── config/                     # 環境變數載入
│   ├── auth/                       # HMAC Token 認證 + RBAC 中介層
│   ├── db/                         # PostgreSQL CRUD（pgx v5）
│   ├── provider/                   # 渠道抽象層
│   │   ├── types.go                # Provider 介面定義
│   │   ├── registry.go             # Provider 註冊表
│   │   ├── gitlab.go               # GitLab Webhook 處理
│   │   ├── slack.go                # Slack Event 處理
│   │   └── telegram.go             # Telegram Update 處理
│   ├── analyzer/                   # OpenCode CLI 子程序呼叫
│   ├── mcpmgr/                     # MCP 套件安裝管理（npm）
│   ├── mcp/                        # MCP Protocol 伺服器
│   ├── api/                        # REST API + Auth Middleware
│   ├── server/                     # HTTP Server 組裝
│   └── webui/                      # 嵌入前端靜態檔（go:embed）
├── web/                            # React Admin 前端原始碼
├── migrations/                     # PostgreSQL Schema
├── Dockerfile                      # 多階段建置（Go → Node.js 22）
├── docker-compose.yml              # PostgreSQL + App 服務
└── .env.example                    # 環境變數範本
```

## 資料庫 Schema

| 資料表 | 用途 |
|--------|------|
| `projects` | 專案（SSH URL、分支、啟停用） |
| `ssh_keys` | SSH 金鑰（私鑰加密儲存） |
| `provider_configs` | 渠道配置（GitLab/Slack/Telegram + Webhook 路徑） |
| `trigger_keywords` | 觸發關鍵字（keyword → mode 映射） |
| `tasks` | 分析任務紀錄（狀態、結果、錯誤訊息） |
| `webhook_deliveries` | Webhook 事件去重 |
| `settings` | 系統設定（key-value JSONB，含 OpenCode 設定檔） |
| `mcp_servers` | MCP 伺服器定義（npm 套件、啟用狀態） |
| `users` | 使用者帳號（角色：admin / editor / viewer） |

## API 端點

所有 `/api/*` 端點需要 JWT Token（透過 `Authorization: Bearer <token>` 傳送）。

| 方法 | 路徑 | 說明 | 權限 |
|------|------|------|------|
| POST | `/api/auth/login` | 登入取得 Token | 公開 |
| GET | `/api/auth/me` | 取得目前使用者資訊 | 已登入 |
| PUT | `/api/auth/password` | 修改密碼 | 已登入 |
| GET/POST | `/api/projects` | 專案列表 / 建立 | 讀取：全部；寫入：Admin |
| GET/PUT/DELETE | `/api/projects/{id}` | 專案詳情 / 更新 / 刪除 | Admin |
| GET/POST | `/api/ssh-keys` | SSH 金鑰列表 / 建立 | Admin |
| DELETE | `/api/ssh-keys/{id}` | 刪除 SSH 金鑰 | Admin |
| GET/POST | `/api/providers/{projectId}` | 渠道配置列表 / 建立 | Admin |
| GET/POST | `/api/keywords/{projectId}` | 觸發關鍵字列表 / 建立 | Admin |
| GET | `/api/tasks` | 任務列表（支援分頁） | 已登入 |
| GET | `/api/tasks/{id}` | 任務詳情 | 已登入 |
| GET/PUT | `/api/settings` | 系統設定列表 / 更新 | Admin |
| GET/POST | `/api/mcp-servers` | MCP 伺服器列表 / 建立 | Admin |
| POST | `/api/mcp-servers/{id}/install` | 安裝 MCP 套件 | Admin |
| GET/POST | `/api/users` | 使用者列表 / 建立 | Admin |
| PUT/DELETE | `/api/users/{id}` | 更新 / 刪除使用者 | Admin |
| POST | `/hook/{provider}/{prefix}` | Webhook 接收端點 | 依 webhook_secret 驗證 |

## 技術棧

| 層級 | 技術 |
|------|------|
| 後端 | Go 1.24、net/http、pgx v5 |
| 前端 | React Admin、TypeScript、Vite、MUI、Monaco Editor |
| 資料庫 | PostgreSQL 16 |
| 認證 | HMAC Token（bcrypt 密碼雜湊） |
| GitLab SDK | [go-gitlab](https://github.com/xanzy/go-gitlab) v0.115 |
| MCP | [mcp-go](https://github.com/mark3labs/mcp-go) v0.44 |
| 容器 | Docker 多階段建置（Go builder → Node.js 22 runtime） |

## License

MIT
