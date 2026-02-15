# 專案知識庫

## 概覽

多渠道 AI 程式碼分析機器人。透過 GitLab / Slack / Telegram Webhook 觸發 OpenCode CLI 分析，結果回覆至來源渠道。Go 後端 + React Admin 前端，前端編譯後以 `go:embed` 嵌入單一二進位檔。

## 架構流程

```
Webhook 請求 → Provider 層（驗證 + 解析）→ Analyzer（子程序呼叫 OpenCode CLI）→ 結果回寫渠道
                                                                ↕
                                                        PostgreSQL（所有設定 + 任務紀錄）
                                                                ↑
                                                     React Admin WebUI（嵌入 Go binary）
```

## 結構

```
opencode-dog/
├── cmd/server/main.go          # 入口點（config → server → 自動 migration → 啟動）
├── internal/
│   ├── config/                  # 環境變數載入（僅基礎設施設定）
│   ├── auth/                    # HMAC Token 認證 + RBAC 中介層（admin/editor/viewer）
│   ├── db/                      # PostgreSQL CRUD（pgx v5 連線池）— 536 行核心
│   ├── provider/                # 渠道抽象層（介面 + GitLab/Slack/Telegram 實作）
│   ├── analyzer/                # OpenCode CLI 子程序（寫入設定檔 → 執行 → 解析輸出）
│   ├── api/                     # REST API 端點 + Auth Middleware — 721 行
│   ├── mcp/                     # MCP Protocol 伺服器（mcp-go，提供 5 個 tool）
│   ├── mcpmgr/                  # MCP npm 套件安裝/移除管理
│   ├── server/                  # HTTP Server 組裝 + Webhook 路由註冊 + 優雅關閉
│   └── webui/                   # go:embed 嵌入前端靜態檔
├── web/                         # React Admin 前端（TypeScript + Vite + MUI）
├── migrations/001_init.sql      # 資料庫 Schema（9 張表 + 2 個 enum）
├── Dockerfile                   # 多階段建置（Go builder → Node.js 22 runtime）
└── docker-compose.yml           # PostgreSQL 16 + App 服務
```

## 快速定位

| 任務 | 位置 | 備註 |
|------|------|------|
| 新增渠道（如 Discord） | `internal/provider/` | 實作 `Provider` 介面，在 `server.go` 註冊 |
| 修改 API 端點 | `internal/api/api.go` | `RegisterRoutes()` 集中註冊，handler 同檔 |
| 修改資料庫 | `internal/db/db.go` + `models.go` | 手寫 SQL，無 ORM |
| 修改觸發邏輯 | `internal/analyzer/analyzer.go` | `matchKeyword()` + `buildPrompt()` |
| 新增 MCP Tool | `internal/mcp/server.go` | `registerTools()` 方法 |
| 前端頁面 | `web/src/resources/` | 每個 `.tsx` 對應一個 React Admin Resource |
| 認證邏輯 | `internal/auth/auth.go` | 自製 HMAC Token（非標準 JWT lib） |
| 環境變數 | `internal/config/config.go` | `getEnv()` 模式，預設值寫在程式內 |
| 資料庫 Schema | `migrations/001_init.sql` | 啟動時自動執行，冪等設計 |

## 技術棧

| 層級 | 技術 |
|------|------|
| 後端 | Go 1.24、net/http（標準庫）、pgx v5 |
| 前端 | React 19、React Admin 5.14、Vite 7、MUI 7、Monaco Editor |
| 資料庫 | PostgreSQL 16 |
| 認證 | HMAC Token（自製）、bcrypt 密碼雜湊 |
| 外部 SDK | go-gitlab v0.115、mcp-go v0.44 |
| 容器 | Docker 多階段建置（Go builder → Node.js 22 runtime） |

## 慣例

- **純標準庫 HTTP**：不用 gin/echo/chi，直接 `http.ServeMux` + `HandleFunc`
- **無 ORM**：手寫 SQL（pgx v5），模型定義在 `db/models.go`
- **結構化日誌**：全專案 `log/slog`（JSON handler），不用 `fmt.Println` 或 `log`
- **設定分離**：環境變數僅管基礎設施（DB、JWT），業務設定全在 PostgreSQL
- **前端嵌入**：Vite 輸出到 `internal/webui/dist/`，Go 用 `go:embed` 打包
- **無測試**：專案目前沒有 `_test.go` 或前端測試
- **無 CI/CD**：無 GitHub Actions / GitLab CI，僅 Docker Compose 部署
- **無 Makefile**：本機開發手動執行 `go run ./cmd/server`

## 禁止事項

- **不要用 `.env` 存業務設定** — 業務設定一律存 PostgreSQL，透過 WebUI 管理
- **不要新增第三方 HTTP 框架** — 保持標準庫 `net/http`
- **不要引入 ORM** — 維持手寫 SQL + pgx v5
- **不要用 `log` 或 `fmt.Println`** — 一律 `slog.Logger`
- **不要直接修改 `internal/webui/dist/`** — 那是建置產物，修改 `web/src/` 後重新編譯

## 安全注意

- `JWT_SECRET` 未設定時會生成隨機密鑰，重啟後所有 Token 失效
- 預設帳號 `admin/admin`，首次登入務必修改密碼
- Webhook 驗證各渠道獨立實作（GitLab token、Slack signing_secret、Telegram secret_token）
- SSH 私鑰存於資料庫（`ssh_keys` 表），注意存取控制

## 指令

```bash
# 本機開發
cd web && npm install && npm run build && cd ..
export DB_PASSWORD=xxx JWT_SECRET=xxx OPENCODE_CONFIG_DIR=/tmp/oc
go run ./cmd/server

# Docker 部署
cp .env.example .env   # 編輯 DB_PASSWORD + JWT_SECRET
docker compose up -d   # http://localhost:8080

# 前端開發（熱更新）
cd web && npm run dev   # 需後端同時執行

# 前端 Lint
cd web && npm run lint
```

## 注意事項

- `analyzer.go` 呼叫 OpenCode CLI 有 5 分鐘超時限制
- MCP 套件安裝有 3 分鐘超時限制
- Webhook 路由在啟動時從 DB 載入並註冊；新增 Provider 設定後需重啟才生效（或走 fallback `/hook/` 路由）
- `internal/mcp/server.go` 的 `handleGetTask` 用迴圈掃全表查找，非直接 ID 查詢
- 前端 `dataProvider.ts` 部分資源採用客戶端排序和分頁
- 認證是自製 HMAC（`auth/hmac.go`），非標準 JWT 函式庫
