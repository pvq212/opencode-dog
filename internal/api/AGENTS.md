# REST API 層

## 概覽

721 行單檔。所有 REST 端點的路由註冊與 handler 實作，含認證中介層整合。

## 路由結構

```
/api/auth/login          ← 公開（無認證）
/api/**                  ← 全部經過 auth.Middleware（Bearer Token 驗證）
```

`RegisterRoutes(mux)` 集中註冊，公開路由直接掛 mux，受保護路由掛在內部 `protected` mux 再透過 `auth.Middleware` 包裝。

## Handler 命名規則

- 集合端點：`handleProjects`、`handleTasks`（內部依 `r.Method` 分 GET/POST）
- 明細端點：`handleProjectDetail`（內部依 `r.Method` 分 GET/PUT/DELETE）
- 從 URL path 取 ID：手動 `strings.TrimPrefix(r.URL.Path, "/api/xxx/")`

## 權限控制

API handler 內部檢查 `auth.GetUser(ctx).Role`：
- `admin` — 全部操作
- `editor` — 讀取 + 部分寫入
- `viewer` — 僅讀取

非 `auth.RequireRole()` 中介層模式，而是 handler 內 inline 檢查。

## 回應格式

- 成功：直接 `json.NewEncoder(w).Encode(data)`
- 錯誤：`http.Error()` 或自定 JSON `{"error": "message"}`
- Content-Type 由各 handler 自行設定
- React Admin 相容：部分 List 回應需含 `Content-Range` header

## 新增端點步驟

1. 在 `api.go` 新增 handler 方法
2. 在 `RegisterRoutes()` 的 `protected` mux 註冊路由
3. 內部做 Method 檢查（`r.Method == http.MethodGet` 等）
4. 權限檢查用 `auth.GetUser(r.Context())`

## 注意

- 無路由參數解析器，路徑解析靠 `strings.TrimPrefix`
- `/api/mcp-servers/{id}/install` 是 POST，觸發非同步安裝（`go a.mcpMgr.Install(...)`）
- 巢狀資源（providers、keywords）路徑含 projectId：`/api/providers/{projectId}`
- Tasks 的 List API 支援 query string 分頁（`limit`、`offset`、`status`、`provider_type`）
