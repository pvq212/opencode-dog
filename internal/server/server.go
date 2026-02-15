package server

import (
	"context"
	"errors"
	"fmt"
	"github.com/opencode-ai/opencode-dog/internal/analyzer"
	"github.com/opencode-ai/opencode-dog/internal/api"
	"github.com/opencode-ai/opencode-dog/internal/auth"
	"github.com/opencode-ai/opencode-dog/internal/config"
	"github.com/opencode-ai/opencode-dog/internal/db"
	mcpserver "github.com/opencode-ai/opencode-dog/internal/mcp"
	"github.com/opencode-ai/opencode-dog/internal/mcpmgr"
	"github.com/opencode-ai/opencode-dog/internal/provider"
	"github.com/opencode-ai/opencode-dog/internal/webui"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

type Server struct {
	cfg        *config.Config
	database   *db.DB
	registry   *provider.Registry
	analyzer   *analyzer.Analyzer
	auth       *auth.Auth
	mcpMgr     *mcpmgr.Manager
	logger     *slog.Logger
	httpServer *http.Server
}

func New(cfg *config.Config, logger *slog.Logger) (*Server, error) {
	ctx := context.Background()

	database, err := db.New(ctx, cfg.DSN(), cfg.DBMaxConns, cfg.DBMinConns, cfg.DBMaxConnLifetime)
	if err != nil {
		return nil, fmt.Errorf("init database: %w", err)
	}

	registry := provider.NewRegistry(logger)
	registry.Register(provider.NewGitLabProvider(logger))
	registry.Register(provider.NewSlackProvider(database, logger))
	registry.Register(provider.NewTelegramProvider(database, logger))

	a := analyzer.New(database, registry, logger, cfg.OpencodeConfigDir)
	authSvc := auth.New(database, logger, cfg.JWTSecret)
	mcpMgr := mcpmgr.New(database, logger)

	if err := authSvc.SeedDefaultAdmin(ctx); err != nil {
		logger.Warn("seed default admin failed", "error", err)
	}

	return &Server{
		cfg:      cfg,
		database: database,
		registry: registry,
		analyzer: a,
		auth:     authSvc,
		mcpMgr:   mcpMgr,
		logger:   logger,
	}, nil
}

func (s *Server) Start() error {
	mux := http.NewServeMux()

	apiHandler := api.New(s.database, s.auth, s.mcpMgr, s.logger)
	apiHandler.RegisterRoutes(mux)

	s.registerWebhookRoutes(mux)

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"status":"ok"}`)
	})

	mcpEnabled := s.database.GetSettingBool(context.Background(), "mcp_enabled", true)
	if mcpEnabled {
		mcpSrv := mcpserver.NewServer(s.database, s.logger)
		mcpEndpoint := s.database.GetSettingString(context.Background(), "mcp_endpoint", "/mcp")
		mux.Handle(mcpEndpoint, newMCPHTTPHandler(mcpSrv.GetServer()))
		s.logger.Info("MCP server enabled", "endpoint", mcpEndpoint)
	}

	webui.RegisterRoutes(mux)

	s.httpServer = &http.Server{
		Addr:              s.cfg.ListenAddr(),
		Handler:           mux,
		ReadHeaderTimeout: s.cfg.ReadHeaderTimeout,
		ReadTimeout:       s.cfg.ReadTimeout,
		WriteTimeout:      s.cfg.WriteTimeout,
		IdleTimeout:       s.cfg.IdleTimeout,
	}

	errCh := make(chan error, 1)
	go func() {
		s.logger.Info("server starting", "addr", s.cfg.ListenAddr())
		if err := s.httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-errCh:
		return fmt.Errorf("server error: %w", err)
	case sig := <-quit:
		s.logger.Info("shutdown signal received", "signal", sig.String())
	}

	return s.shutdown()
}

func (s *Server) registerWebhookRoutes(mux *http.ServeMux) {
	configs, err := s.database.ListAllProviderConfigs(context.Background())
	if err != nil {
		s.logger.Warn("failed to load provider configs, webhook routes not registered", "error", err)
		return
	}

	for _, pc := range configs {
		p, ok := s.registry.Get(provider.ProviderType(pc.ProviderType))
		if !ok {
			s.logger.Warn("unknown provider type", "type", pc.ProviderType)
			continue
		}

		cfgMap := pc.ConfigMap()
		path := pc.WebhookPath
		cfgID := pc.ID
		projectID := pc.ProjectID

		handler := p.BuildHandler(cfgID, pc.WebhookSecret, cfgMap, func(ctx context.Context, msg *provider.IncomingMessage) {
			msg.ProjectID = projectID
			msg.ProviderCfgID = cfgID
			s.analyzer.HandleMessage(ctx, msg)
		})

		mux.Handle(path, handler)
		s.logger.Info("webhook route registered",
			"path", path,
			"provider", pc.ProviderType,
			"project", pc.ProjectID,
		)
	}

	mux.HandleFunc("/hook/", func(w http.ResponseWriter, r *http.Request) {
		pc, err := s.database.GetProviderConfigByPath(r.Context(), r.URL.Path)
		if err != nil {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}

		p, ok := s.registry.Get(provider.ProviderType(pc.ProviderType))
		if !ok {
			http.Error(w, "unknown provider", http.StatusInternalServerError)
			return
		}

		cfgMap := pc.ConfigMap()
		handler := p.BuildHandler(pc.ID, pc.WebhookSecret, cfgMap, func(ctx context.Context, msg *provider.IncomingMessage) {
			msg.ProjectID = pc.ProjectID
			msg.ProviderCfgID = pc.ID
			s.analyzer.HandleMessage(ctx, msg)
		})
		handler.ServeHTTP(w, r)
	})
}

func (s *Server) shutdown() error {
	ctx, cancel := context.WithTimeout(context.Background(), s.cfg.ShutdownTimeout)
	defer cancel()
	s.logger.Info("shutting down server...")
	if err := s.httpServer.Shutdown(ctx); err != nil {
		return fmt.Errorf("server shutdown: %w", err)
	}
	s.database.Close()
	s.logger.Info("server stopped")
	return nil
}

func (s *Server) RunMigrations(ctx context.Context, dir string) error {
	return s.database.RunMigrations(ctx, dir)
}
