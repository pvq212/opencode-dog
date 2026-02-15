# React Admin 前端

## 概覽

TypeScript + React Admin 5.14 + Vite 7 + MUI 7。編譯產物輸出至 `../internal/webui/dist/`，由 Go 端 `go:embed` 嵌入。

## 結構

```
web/src/
├── main.tsx              # 掛載 React root
├── App.tsx               # React Admin 設定（Resources、CustomRoutes、權限）
├── Layout.tsx            # 自訂導覽列（含 Guides 連結）
├── Dashboard.tsx         # 首頁儀表板
├── theme.ts              # 深色/淺色主題定義
├── authProvider.ts       # JWT 認證（localStorage 存 token）
├── dataProvider.ts       # REST API 客戶端（自訂，非 ra-data-simple-rest）
└── resources/            # 各管理頁面
    ├── projects.tsx       # 專案 CRUD
    ├── sshKeys.tsx        # SSH 金鑰管理
    ├── providers.tsx      # 渠道設定
    ├── keywords.tsx       # 觸發關鍵字管理（自訂頁面，非標準 Resource）
    ├── tasks.tsx          # 任務列表/詳情
    ├── settings.tsx       # 系統設定（含 Monaco JSON 編輯器）
    ├── mcpServers.tsx     # MCP 伺服器管理
    ├── users.tsx          # 使用者 RBAC 管理
    └── Guides.tsx         # 渠道接入教學（Markdown 渲染）
```

## 關鍵檔案

### `dataProvider.ts`
- **非標準實作**：未使用 `ra-data-simple-rest`（雖有安裝），完全自訂
- 部分資源（非 tasks）用客戶端排序 + 分頁
- `tasks` 走伺服器端分頁（query string `limit`/`offset`）
- 匯出兩個非標準方法：`installMcpServer(id)` 和 `saveKeywords(projectId, keywords)`
- `keywords` 資源手動加 `id`（後端不回傳 id，前端用 index 補）

### `authProvider.ts`
- Token 存 `localStorage.token`
- User 物件存 `localStorage.user`
- 權限取自 `user.role`（admin / editor / viewer）

### `App.tsx`
- 權限控制：`viewer` 無法看到 create/edit 按鈕
- `admin` 獨有：SSH Keys、Settings、MCP Servers、Users
- `CustomRoutes`：`/guides` 和 `/keywords`（非標準 Resource）

## 建置

```bash
npm run build    # tsc -b && vite build → 輸出到 ../internal/webui/dist/
npm run dev      # 開發模式（需後端同時執行，API 代理至 localhost:8080）
npm run lint     # ESLint 檢查
```

## 慣例

- TypeScript strict mode
- ESLint 9 flat config（react-hooks + react-refresh）
- 無 Prettier（僅 ESLint 格式化）
- 無前端測試
- 深色主題為預設（`defaultTheme="dark"`）

## 注意

- Vite `base: "/"` — SPA 路由，Go 端需處理 fallback
- `settings` 的 `id` 用 `key` 欄位（非 UUID）
- Monaco Editor 用於 JSON 設定編輯（auth.json、.opencode.json、oh-my-opencode.json）
- `as any` 出現在 `dataProvider.ts` 的 delete 方法（React Admin 型別限制）
