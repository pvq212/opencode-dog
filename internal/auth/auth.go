package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/opencode-ai/opencode-dog/internal/db"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrAccountDisabled    = errors.New("account disabled")
	ErrForbidden          = errors.New("forbidden")
)

type contextKey string

const userContextKey contextKey = "user"

type TokenClaims struct {
	UserID   string `json:"uid"`
	Username string `json:"usr"`
	Role     string `json:"rol"`
	Exp      int64  `json:"exp"`
}

type Auth struct {
	database *db.DB
	logger   *slog.Logger
	secret   []byte
	tokenTTL time.Duration
}

func New(database *db.DB, logger *slog.Logger, jwtSecret string) *Auth {
	if jwtSecret == "" {
		b := make([]byte, 32)
		_, _ = rand.Read(b)
		jwtSecret = hex.EncodeToString(b)
		logger.Warn("JWT_SECRET not set, generated random secret (tokens won't survive restart)")
	}
	return &Auth{
		database: database,
		logger:   logger,
		secret:   []byte(jwtSecret),
		tokenTTL: 24 * time.Hour,
	}
}

func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(hash), err
}

func CheckPassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

func (a *Auth) Login(ctx context.Context, username, password string) (string, *db.User, error) {
	user, err := a.database.GetUserByUsername(ctx, username)
	if err != nil {
		return "", nil, ErrInvalidCredentials
	}
	if !user.Enabled {
		return "", nil, ErrAccountDisabled
	}
	if !CheckPassword(user.PasswordHash, password) {
		return "", nil, ErrInvalidCredentials
	}
	token, err := a.generateToken(ctx, user)
	if err != nil {
		return "", nil, err
	}
	return token, user, nil
}

func (a *Auth) generateToken(ctx context.Context, user *db.User) (string, error) {
	ttl := a.database.GetSettingDuration(ctx, "token_ttl", a.tokenTTL)
	claims := TokenClaims{
		UserID:   user.ID,
		Username: user.Username,
		Role:     user.Role,
		Exp:      time.Now().Add(ttl).Unix(),
	}
	payload, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	return encodeHMAC(a.secret, payload), nil
}

func (a *Auth) validateToken(token string) (*TokenClaims, error) {
	payload, err := decodeHMAC(a.secret, token)
	if err != nil {
		return nil, err
	}
	var claims TokenClaims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, err
	}
	if time.Now().Unix() > claims.Exp {
		return nil, errors.New("token expired")
	}
	return &claims, nil
}

func (a *Auth) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := r.Header.Get("Authorization")
		if header == "" {
			writeAuthErr(w, http.StatusUnauthorized, "missing authorization header")
			return
		}
		token := strings.TrimPrefix(header, "Bearer ")
		if token == header {
			writeAuthErr(w, http.StatusUnauthorized, "invalid authorization format")
			return
		}
		claims, err := a.validateToken(token)
		if err != nil {
			writeAuthErr(w, http.StatusUnauthorized, "invalid token")
			return
		}
		ctx := context.WithValue(r.Context(), userContextKey, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func RequireRole(roles ...string) func(http.Handler) http.Handler {
	roleSet := make(map[string]bool, len(roles))
	for _, r := range roles {
		roleSet[r] = true
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := GetUser(r.Context())
			if claims == nil {
				writeAuthErr(w, http.StatusUnauthorized, "not authenticated")
				return
			}
			if !roleSet[claims.Role] {
				writeAuthErr(w, http.StatusForbidden, "insufficient permissions")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func GetUser(ctx context.Context) *TokenClaims {
	claims, _ := ctx.Value(userContextKey).(*TokenClaims)
	return claims
}

func (a *Auth) SeedDefaultAdmin(ctx context.Context) error {
	count, err := a.database.CountUsers(ctx)
	if err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	hash, err := HashPassword("admin")
	if err != nil {
		return err
	}
	user := &db.User{
		Username:     "admin",
		PasswordHash: hash,
		DisplayName:  "Administrator",
		Role:         db.RoleAdmin,
		Enabled:      true,
	}
	a.logger.Info("creating default admin user (username: admin, password: admin)")
	return a.database.CreateUser(ctx, user)
}

func writeAuthErr(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
