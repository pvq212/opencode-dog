package auth

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/opencode-ai/opencode-dog/internal/db"
	"github.com/opencode-ai/opencode-dog/internal/db/dbmock"
)

func newTestAuth(store *dbmock.Store) *Auth {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return New(store, logger, "test-secret")
}

func mustHashPassword(t *testing.T, pw string) string {
	t.Helper()
	h, err := HashPassword(pw)
	if err != nil {
		t.Fatalf("HashPassword(%q) failed: %v", pw, err)
	}
	return h
}

func seedUser(t *testing.T, store *dbmock.Store, username, password, role string, enabled bool) *db.User {
	t.Helper()
	u := &db.User{
		Username:     username,
		PasswordHash: mustHashPassword(t, password),
		DisplayName:  username,
		Role:         role,
		Enabled:      enabled,
	}
	if err := store.CreateUser(context.Background(), u); err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}
	return u
}

// --- HashPassword / CheckPassword ---

func TestHashPasswordAndCheck(t *testing.T) {
	hash, err := HashPassword("mypassword")
	if err != nil {
		t.Fatalf("HashPassword error: %v", err)
	}
	if !CheckPassword(hash, "mypassword") {
		t.Fatal("CheckPassword should return true for correct password")
	}
	if CheckPassword(hash, "wrongpassword") {
		t.Fatal("CheckPassword should return false for wrong password")
	}
}

func TestHashPasswordUniqueness(t *testing.T) {
	h1, _ := HashPassword("same")
	h2, _ := HashPassword("same")
	if h1 == h2 {
		t.Fatal("two hashes of same password should differ (bcrypt salt)")
	}
}

// --- Login ---

func TestLoginSuccess(t *testing.T) {
	store := dbmock.New()
	a := newTestAuth(store)
	seedUser(t, store, "alice", "pass123", db.RoleAdmin, true)

	token, user, err := a.Login(context.Background(), "alice", "pass123")
	if err != nil {
		t.Fatalf("Login error: %v", err)
	}
	if token == "" {
		t.Fatal("expected non-empty token")
	}
	if user.Username != "alice" {
		t.Fatalf("user.Username = %q, want %q", user.Username, "alice")
	}
}

func TestLoginWrongPassword(t *testing.T) {
	store := dbmock.New()
	a := newTestAuth(store)
	seedUser(t, store, "bob", "correct", db.RoleViewer, true)

	_, _, err := a.Login(context.Background(), "bob", "wrong")
	if err != ErrInvalidCredentials {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestLoginUserNotFound(t *testing.T) {
	store := dbmock.New()
	a := newTestAuth(store)

	_, _, err := a.Login(context.Background(), "ghost", "pass")
	if err != ErrInvalidCredentials {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestLoginDisabledAccount(t *testing.T) {
	store := dbmock.New()
	a := newTestAuth(store)
	seedUser(t, store, "disabled", "pass", db.RoleEditor, false)

	_, _, err := a.Login(context.Background(), "disabled", "pass")
	if err != ErrAccountDisabled {
		t.Fatalf("expected ErrAccountDisabled, got %v", err)
	}
}

// --- generateToken / validateToken ---

func TestGenerateValidateRoundTrip(t *testing.T) {
	store := dbmock.New()
	a := newTestAuth(store)
	user := seedUser(t, store, "carol", "pw", db.RoleAdmin, true)

	token, err := a.generateToken(context.Background(), user)
	if err != nil {
		t.Fatalf("generateToken error: %v", err)
	}
	claims, err := a.validateToken(token)
	if err != nil {
		t.Fatalf("validateToken error: %v", err)
	}
	if claims.UserID != user.ID {
		t.Fatalf("claims.UserID = %q, want %q", claims.UserID, user.ID)
	}
	if claims.Username != "carol" {
		t.Fatalf("claims.Username = %q, want %q", claims.Username, "carol")
	}
	if claims.Role != db.RoleAdmin {
		t.Fatalf("claims.Role = %q, want %q", claims.Role, db.RoleAdmin)
	}
}

func TestValidateTokenExpired(t *testing.T) {
	store := dbmock.New()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	a := New(store, logger, "test-secret")
	a.tokenTTL = -1 * time.Hour

	user := seedUser(t, store, "expired", "pw", db.RoleViewer, true)
	token, err := a.generateToken(context.Background(), user)
	if err != nil {
		t.Fatalf("generateToken error: %v", err)
	}
	_, err = a.validateToken(token)
	if err == nil || err.Error() != "token expired" {
		t.Fatalf("expected 'token expired', got %v", err)
	}
}

func TestValidateTokenInvalid(t *testing.T) {
	store := dbmock.New()
	a := newTestAuth(store)

	_, err := a.validateToken("not-a-valid-token")
	if err == nil {
		t.Fatal("expected error for invalid token")
	}
}

func TestValidateTokenWrongSecret(t *testing.T) {
	store := dbmock.New()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	a1 := New(store, logger, "secret-1")
	a2 := New(store, logger, "secret-2")
	user := seedUser(t, store, "cross", "pw", db.RoleViewer, true)

	token, _ := a1.generateToken(context.Background(), user)
	_, err := a2.validateToken(token)
	if err == nil {
		t.Fatal("expected error validating token with wrong secret")
	}
}

// --- New with empty secret ---

func TestNewEmptySecretGeneratesRandom(t *testing.T) {
	store := dbmock.New()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	a := New(store, logger, "")
	if len(a.secret) == 0 {
		t.Fatal("expected non-empty secret when empty string passed")
	}
}

// --- Middleware ---

func TestMiddlewareValidToken(t *testing.T) {
	store := dbmock.New()
	a := newTestAuth(store)
	user := seedUser(t, store, "mw-user", "pw", db.RoleEditor, true)
	token, _, _ := a.Login(context.Background(), "mw-user", "pw")

	var capturedClaims *TokenClaims
	handler := a.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedClaims = GetUser(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	if capturedClaims == nil {
		t.Fatal("claims should be set in context")
	}
	if capturedClaims.UserID != user.ID {
		t.Fatalf("claims.UserID = %q, want %q", capturedClaims.UserID, user.ID)
	}
}

func TestMiddlewareMissingHeader(t *testing.T) {
	store := dbmock.New()
	a := newTestAuth(store)
	handler := a.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called")
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusUnauthorized)
	}
	assertJSONError(t, rr, "missing authorization header")
}

func TestMiddlewareInvalidFormat(t *testing.T) {
	store := dbmock.New()
	a := newTestAuth(store)
	handler := a.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called")
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Basic abc123")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusUnauthorized)
	}
	assertJSONError(t, rr, "invalid authorization format")
}

func TestMiddlewareInvalidToken(t *testing.T) {
	store := dbmock.New()
	a := newTestAuth(store)
	handler := a.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called")
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer invalid.token.here")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusUnauthorized)
	}
}

func TestMiddlewareExpiredToken(t *testing.T) {
	store := dbmock.New()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	a := New(store, logger, "test-secret")
	a.tokenTTL = -1 * time.Hour
	user := seedUser(t, store, "exp-mw", "pw", db.RoleViewer, true)
	token, _ := a.generateToken(context.Background(), user)

	handler := a.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called")
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusUnauthorized)
	}
}

// --- RequireRole ---

func TestRequireRoleAllowed(t *testing.T) {
	tests := []struct {
		name    string
		role    string
		allowed []string
		want    int
	}{
		{"admin allowed", db.RoleAdmin, []string{db.RoleAdmin}, http.StatusOK},
		{"editor in multi-role", db.RoleEditor, []string{db.RoleAdmin, db.RoleEditor}, http.StatusOK},
		{"viewer forbidden", db.RoleViewer, []string{db.RoleAdmin}, http.StatusForbidden},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			claims := &TokenClaims{UserID: "1", Username: "u", Role: tt.role}
			ctx := context.WithValue(context.Background(), userContextKey, claims)

			inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})
			handler := RequireRole(tt.allowed...)(inner)

			req := httptest.NewRequest(http.MethodGet, "/", nil).WithContext(ctx)
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			if rr.Code != tt.want {
				t.Fatalf("status = %d, want %d", rr.Code, tt.want)
			}
		})
	}
}

func TestRequireRoleNoClaims(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called")
	})
	handler := RequireRole(db.RoleAdmin)(inner)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusUnauthorized)
	}
	assertJSONError(t, rr, "not authenticated")
}

// --- GetUser ---

func TestGetUserFromContext(t *testing.T) {
	claims := &TokenClaims{UserID: "uid-1", Username: "test", Role: db.RoleAdmin}
	ctx := context.WithValue(context.Background(), userContextKey, claims)
	got := GetUser(ctx)
	if got == nil {
		t.Fatal("expected claims, got nil")
	}
	if got.UserID != "uid-1" {
		t.Fatalf("UserID = %q, want %q", got.UserID, "uid-1")
	}
}

func TestGetUserFromEmptyContext(t *testing.T) {
	got := GetUser(context.Background())
	if got != nil {
		t.Fatalf("expected nil, got %+v", got)
	}
}

// --- SeedDefaultAdmin ---

func TestSeedDefaultAdminFirstRun(t *testing.T) {
	store := dbmock.New()
	a := newTestAuth(store)

	if err := a.SeedDefaultAdmin(context.Background()); err != nil {
		t.Fatalf("SeedDefaultAdmin error: %v", err)
	}
	if len(store.Users) != 1 {
		t.Fatalf("expected 1 user, got %d", len(store.Users))
	}
	admin := store.Users[0]
	if admin.Username != "admin" {
		t.Fatalf("username = %q, want %q", admin.Username, "admin")
	}
	if admin.Role != db.RoleAdmin {
		t.Fatalf("role = %q, want %q", admin.Role, db.RoleAdmin)
	}
	if !admin.Enabled {
		t.Fatal("admin should be enabled")
	}
	if !CheckPassword(admin.PasswordHash, "admin") {
		t.Fatal("default password should be 'admin'")
	}
}

func TestSeedDefaultAdminIdempotent(t *testing.T) {
	store := dbmock.New()
	a := newTestAuth(store)
	seedUser(t, store, "existing", "pw", db.RoleViewer, true)

	if err := a.SeedDefaultAdmin(context.Background()); err != nil {
		t.Fatalf("SeedDefaultAdmin error: %v", err)
	}
	if len(store.Users) != 1 {
		t.Fatalf("expected 1 user (no new user created), got %d", len(store.Users))
	}
	if store.Users[0].Username != "existing" {
		t.Fatalf("existing user should be preserved, got %q", store.Users[0].Username)
	}
}

func TestSeedDefaultAdminCountError(t *testing.T) {
	store := dbmock.New()
	store.ErrDefault = errForTest("db error")
	a := newTestAuth(store)

	err := a.SeedDefaultAdmin(context.Background())
	if err == nil {
		t.Fatal("expected error from CountUsers")
	}
}

// --- Token with custom TTL from settings ---

func TestGenerateTokenCustomTTL(t *testing.T) {
	store := dbmock.New()
	store.Settings = []*db.Setting{
		{Key: "token_ttl", Value: json.RawMessage(`"2h"`)},
	}
	a := newTestAuth(store)
	user := seedUser(t, store, "ttl-user", "pw", db.RoleViewer, true)

	token, err := a.generateToken(context.Background(), user)
	if err != nil {
		t.Fatalf("generateToken error: %v", err)
	}
	claims, err := a.validateToken(token)
	if err != nil {
		t.Fatalf("validateToken error: %v", err)
	}

	expectedMin := time.Now().Add(1*time.Hour + 55*time.Minute).Unix()
	expectedMax := time.Now().Add(2*time.Hour + 5*time.Minute).Unix()
	if claims.Exp < expectedMin || claims.Exp > expectedMax {
		t.Fatalf("token expiry %d not within expected 2h range", claims.Exp)
	}
}

// --- writeAuthErr ---

func TestWriteAuthErr(t *testing.T) {
	rr := httptest.NewRecorder()
	writeAuthErr(rr, http.StatusForbidden, "nope")

	if rr.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusForbidden)
	}
	ct := rr.Header().Get("Content-Type")
	if ct != "application/json" {
		t.Fatalf("Content-Type = %q, want %q", ct, "application/json")
	}
	assertJSONError(t, rr, "nope")
}

// --- helpers ---

func assertJSONError(t *testing.T, rr *httptest.ResponseRecorder, wantMsg string) {
	t.Helper()
	var body map[string]string
	if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}
	if body["error"] != wantMsg {
		t.Fatalf("error = %q, want %q", body["error"], wantMsg)
	}
}

type testError struct{ msg string }

func (e *testError) Error() string { return e.msg }

func errForTest(msg string) error { return &testError{msg: msg} }
