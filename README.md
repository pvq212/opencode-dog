<p align="center">
  <img src="https://raw.githubusercontent.com/opencode-ai/opencode/main/docs/images/logo.png" alt="OpenCode Dog" width="120" />
</p>

<h1 align="center">OpenCode Dog 🐕</h1>

<p align="center">
  <strong>把 AI Coding Agent 帶進你團隊的每一個對話窗</strong>
</p>

<p align="center">
  <a href="#-快速開始">快速開始</a> ·
  <a href="#-功能特色">功能特色</a> ·
  <a href="#-渠道接入">渠道接入</a> ·
  <a href="#-管理後台">管理後台</a> ·
  <a href="#-api-文件">API 文件</a> ·
  <a href="#-貢獻指南">貢獻指南</a>
</p>

<p align="center">
  <img src="https://img.shields.io/badge/Go-1.24-00ADD8?logo=go&logoColor=white" alt="Go 1.24" />
  <img src="https://img.shields.io/badge/React-19-61DAFB?logo=react&logoColor=white" alt="React 19" />
  <img src="https://img.shields.io/badge/PostgreSQL-16-4169E1?logo=postgresql&logoColor=white" alt="PostgreSQL 16" />
  <img src="https://img.shields.io/badge/Docker-Ready-2496ED?logo=docker&logoColor=white" alt="Docker" />
  <img src="https://img.shields.io/badge/License-MIT-green" alt="MIT License" />
</p>

---

## 這是什麼？

**OpenCode Dog** 是一個多渠道 AI 程式碼分析服務——它把 [OpenCode](https://github.com/opencode-ai/opencode) 這隻忠實的 AI Coding Agent 帶到你團隊日常使用的 GitLab、Slack、Telegram 中。

PM 在 GitLab Issue 留言 `@opencode 幫我分析這個 bug`，幾分鐘後就能收到完整的分析報告。QA 在 Slack 頻道貼上錯誤訊息，AI 直接給出修復方案。不需要切換工具、不需要學新介面——**在你原本的工作流裡，直接獲得 AI 能力。**

```
💬 團隊對話窗                    🐕 OpenCode Dog                    🤖 OpenCode Server
───────────────────────────────────────────────────────────────────────────────
  PM: "@opencode 為什麼登入     →  解析訊息 → 觸發分析  →  AI 深度分析
       頁面一直轉圈？"              (HTTP API)                   程式碼庫
                                                                  │
  PM 收到完整分析報告  ←────────  格式化結果 ← 回傳分析   ←──────┘
  直接在 Issue 裡回覆
```

## ✨ 功能特色

### 🔌 多渠道整合

| 渠道 | 觸發方式 | 回覆位置 |
|------|----------|----------|
| **GitLab** | Issue / MR 留言 | 同一則 Issue / MR |
| **Slack** | 頻道訊息 | 同一頻道 thread |
| **Telegram** | 群組訊息 | 同一群組 |

### 🎯 三種觸發模式

```
@opencode 為什麼這個 API 回傳 500？    → ask  模式：分析問題、給出解釋
@plan    幫我規劃登入功能的重構方案       → plan 模式：產生實作計畫
@do      修復這個 null pointer exception → do   模式：直接產出程式碼修改
```

觸發關鍵字完全可自訂——在 WebUI 裡依專案設定不同的關鍵字和對應模式。

### 🛠 更多亮點

- **🖥 管理後台** — React Admin 打造的 WebUI，管理專案、渠道、使用者、MCP 伺服器
- **🔐 RBAC 權限** — Admin / Editor / Viewer 三級角色控制
- **📦 MCP 伺服器** — 在後台一鍵安裝 npm 套件，擴展 OpenCode 能力
- **⚙️ 線上設定** — auth.json、.opencode.json 等設定檔可在 WebUI 用 Monaco Editor 編輯
- **🗄 資料庫驅動** — 所有設定存 PostgreSQL，不依賴 .env 或設定檔
- **📦 單一部署** — Go binary 內嵌前端，搭配 Docker Compose 一鍵啟動

---

## 🚀 快速開始

### 使用 Docker Compose（推薦）

```bash
# 1. 複製專案
git clone https://github.com/anthropic-ai/opencode-dog.git
cd opencode-dog

# 2. 設定環境變數
cp .env.example .env
# 編輯 .env，設定 DB_PASSWORD、JWT_SECRET 和 OPENCODE_SERVER_PASSWORD

# 3. 啟動
docker compose up -d

# 4. 開啟瀏覽器 → http://localhost:8080
#    預設帳號：admin / admin（首次登入請立即修改密碼）
```

### 本機開發

<details>
<summary><strong>前置需求：Go 1.24+ · Node.js 22+ · PostgreSQL 16+</strong></summary>

```bash
# 1. 建立資料庫
createdb opencode_dog

# 2. 編譯前端
cd web && npm install && npm run build && cd ..

# 3. 啟動伺服器
export DB_PASSWORD=your-password
export JWT_SECRET=your-secret
export OPENCODE_CONFIG_DIR=/tmp/opencode-config
go run ./cmd/server

# 前端熱更新開發
cd web && npm run dev   # 需後端同時執行
```

</details>

---

## 🔌 渠道接入

### GitLab

1. 在 WebUI 建立 **Project**（填入 SSH URL）
2. 為該 Project 新增 **Provider**，類型選 `gitlab`，設定 `webhook_secret`
3. 前往 GitLab 專案 → **Settings → Webhooks**
4. URL：`https://YOUR_DOMAIN/hook/gitlab/{project_id_prefix}`
5. Secret Token：步驟 2 設定的 `webhook_secret`
6. 勾選 **Note events**
7. 在 Issue 留言中 `@opencode 請分析這個問題` 即可觸發 ✅

### Slack

1. 建立 **Slack App**，啟用 **Event Subscriptions**
2. 在 WebUI 新增 Provider，類型 `slack`，填入 `bot_token` 和 `signing_secret`
3. Request URL：`https://YOUR_DOMAIN/hook/slack/{project_id_prefix}`
4. 訂閱 `message.channels` 事件
5. 在頻道中 `@opencode 請分析這個問題` ✅

### Telegram

1. 透過 [@BotFather](https://t.me/BotFather) 建立 Bot
2. 在 WebUI 新增 Provider，類型 `telegram`，填入 `bot_token`
3. 設定 Webhook：
   ```
   https://api.telegram.org/bot<TOKEN>/setWebhook?url=https://YOUR_DOMAIN/hook/telegram/{project_id_prefix}
   ```
4. 在群組中 `@opencode 請分析這個問題` ✅

---

## 🖥 管理後台

內建完整的 React Admin 管理介面，登入後即可操作：

| 頁面 | 說明 | 權限 |
|------|------|------|
| **Dashboard** | 系統概覽 — 專案數量、任務統計、近期分析結果 | 全部 |
| **Projects** | 管理程式碼專案（SSH URL、分支、啟停用） | Admin 編輯 |
| **SSH Keys** | 管理 SSH 金鑰（用於 git clone 私有 repo） | Admin |
| **Tasks** | 所有分析任務的狀態追蹤與結果查看 | 全部 |
| **Settings** | 系統設定、OpenCode 設定檔（Monaco JSON 編輯器） | Admin |
| **MCP Servers** | 安裝 / 啟用 / 停用 MCP 伺服器（npm 套件） | Admin |
| **Users** | 使用者帳號管理（RBAC 角色分配） | Admin |
| **Guides** | 各渠道的接入設定教學 | 全部 |

---

## 🏗 架構

```
┌─────────────┐  ┌─────────────┐  ┌──────────────┐
│   GitLab    │  │    Slack    │  │   Telegram   │
│  Webhook    │  │  Webhook    │  │   Webhook    │
└──────┬──────┘  └──────┬──────┘  └──────┬───────┘
       │                │                │
       └────────────────┼────────────────┘
                        ▼
              ┌──────────────────┐
              │  Provider Layer  │  驗證 → 解析 → 統一格式
              └────────┬─────────┘
                       ▼
              ┌──────────────────┐
              │    Analyzer      │  HTTP API 呼叫
              └────────┬─────────┘
                       ▼
              ┌──────────────────┐
              │  OpenCode Server │  Docker 容器（port 4096）
              └──────────────────┘

              ┌──────────────────┐
              │   PostgreSQL     │  設定、任務、使用者
              └──────────────────┘
                       ▲
              ┌──────────────────┐
              │  React Admin UI  │  內嵌於 Go binary
              └──────────────────┘
```

### 專案結構

```
opencode-dog/
├── cmd/server/main.go              # 入口點
├── internal/
│   ├── config/                     # 環境變數載入
│   ├── auth/                       # HMAC Token 認證 + RBAC
│   ├── db/                         # PostgreSQL CRUD（pgx v5）
│   ├── provider/                   # 渠道抽象層
│   │   ├── types.go                #   Provider 介面定義
│   │   ├── registry.go             #   Provider 註冊表
│   │   ├── gitlab.go               #   GitLab 實作
│   │   ├── slack.go                #   Slack 實作
│   │   └── telegram.go             #   Telegram 實作
│   ├── analyzer/                   # OpenCode Server HTTP 客戶端
│   ├── api/                        # REST API 端點
│   ├── mcp/                        # MCP Protocol 伺服器
│   ├── mcpmgr/                     # MCP npm 套件管理
│   ├── server/                     # HTTP Server 組裝
│   └── webui/                      # 前端靜態檔（go:embed）
├── web/                            # React Admin 前端原始碼
├── migrations/                     # PostgreSQL Schema
├── Dockerfile                      # 多階段建置
└── docker-compose.yml              # 一鍵部署
```

### 技術棧

| 層級 | 技術 |
|------|------|
| 後端 | Go 1.24 · 標準庫 `net/http` · pgx v5 |
| 前端 | React 19 · React Admin 5.14 · Vite 7 · MUI 7 · Monaco Editor |
| 資料庫 | PostgreSQL 16 |
| AI 引擎 | [OpenCode](https://opencode.ai) Server（Docker 容器，HTTP API） |
| 認證 | HMAC Token · bcrypt |
| SDK | [go-gitlab](https://github.com/xanzy/go-gitlab) v0.115 · [mcp-go](https://github.com/mark3labs/mcp-go) v0.44 |
| 部署 | Docker Compose（PostgreSQL + OpenCode Server + App） |

---

## 📡 API 文件

所有 `/api/*` 端點需要 Bearer Token 認證（透過 `/api/auth/login` 取得）。

<details>
<summary><strong>展開完整 API 列表</strong></summary>

| 方法 | 路徑 | 說明 | 權限 |
|------|------|------|------|
| POST | `/api/auth/login` | 登入取得 Token | 公開 |
| GET | `/api/auth/me` | 目前使用者資訊 | 已登入 |
| PUT | `/api/auth/password` | 修改密碼 | 已登入 |
| GET · POST | `/api/projects` | 專案列表 / 建立 | 讀取：全部；寫入：Admin |
| GET · PUT · DELETE | `/api/projects/{id}` | 專案 CRUD | Admin |
| GET · POST | `/api/ssh-keys` | SSH 金鑰管理 | Admin |
| DELETE | `/api/ssh-keys/{id}` | 刪除金鑰 | Admin |
| GET · POST | `/api/providers/{projectId}` | 渠道配置管理 | Admin |
| GET · POST · PUT | `/api/keywords/{projectId}` | 觸發關鍵字管理 | Admin |
| GET | `/api/tasks` | 任務列表（支援分頁） | 已登入 |
| GET | `/api/tasks/{id}` | 任務詳情 | 已登入 |
| GET · PUT | `/api/settings` | 系統設定管理 | Admin |
| GET · POST | `/api/mcp-servers` | MCP 伺服器管理 | Admin |
| POST | `/api/mcp-servers/{id}/install` | 安裝 MCP 套件 | Admin |
| GET · POST | `/api/users` | 使用者管理 | Admin |
| PUT · DELETE | `/api/users/{id}` | 更新 / 刪除使用者 | Admin |
| POST | `/hook/{provider}/{prefix}` | Webhook 接收 | Secret 驗證 |

</details>

---

## ⚙️ 環境變數

僅基礎設施設定使用環境變數，業務設定全部在 WebUI 管理。

| 變數 | 說明 | 預設值 |
|------|------|--------|
| `SERVER_PORT` | 伺服器埠號 | `8080` |
| `SERVER_HOST` | 綁定地址 | `0.0.0.0` |
| `DB_HOST` | PostgreSQL 主機 | `localhost` |
| `DB_PORT` | PostgreSQL 埠號 | `5432` |
| `DB_USER` | 資料庫使用者 | `opencode` |
| `DB_PASSWORD` | 資料庫密碼 | **必填** |
| `DB_NAME` | 資料庫名稱 | `opencode_dog` |
| `DB_SSLMODE` | SSL 模式 | `disable` |
| `JWT_SECRET` | Token 簽名密鑰 | **建議設定**¹ |
| `OPENCODE_CONFIG_DIR` | OpenCode 設定檔目錄 | `/app/opencode-config` |
| `OPENCODE_SERVER_USERNAME` | OpenCode Server 認證用戶名 | `opencode` |
| `OPENCODE_SERVER_PASSWORD` | OpenCode Server 認證密碼 | **必填** |
| `OPENCODE_SERVER_PORT` | OpenCode Server 對外埠號 | `4096` |

> ¹ 未設定時自動生成隨機密鑰，重啟後所有 Token 失效。

---

## 🤝 貢獻指南

歡迎任何形式的貢獻！

### 開發流程

```bash
# Fork & Clone
git clone https://github.com/YOUR_USERNAME/opencode-dog.git
cd opencode-dog

# 啟動開發環境
docker compose up postgres -d          # 僅啟動資料庫
cd web && npm install && npm run build && cd ..
export DB_PASSWORD=dev JWT_SECRET=dev OPENCODE_CONFIG_DIR=/tmp/oc
go run ./cmd/server
```

### 新增渠道支援

想加入 Discord、LINE 或其他渠道？只需三步：

1. 在 `internal/provider/` 實作 `Provider` 介面
2. 在 `internal/server/server.go` 的 `New()` 中註冊
3. 提交 PR 🎉

### 開發規範

- **HTTP 框架**：使用標準庫 `net/http`，不引入第三方框架
- **資料庫**：手寫 SQL + pgx v5，不使用 ORM
- **日誌**：一律使用 `log/slog`
- **設定**：業務設定存資料庫，不存 .env

---

## 📋 Roadmap

- [ ] Discord 渠道支援
- [ ] LINE 渠道支援
- [ ] GitHub Issues / Discussions 渠道支援
- [ ] 任務佇列（支援並行分析）
- [ ] Webhook 管理頁面（新增後免重啟）
- [ ] 分析結果快取 & 搜尋
- [ ] OpenTelemetry 可觀測性整合

---

## 📄 License

[MIT](LICENSE) — 自由使用、修改、分發。

---

<p align="center">
  <sub>Built with ❤️ by the OpenCode community</sub>
</p>
