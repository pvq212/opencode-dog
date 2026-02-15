package mcpmgr

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"os/exec"
	"time"

	"github.com/opencode-ai/opencode-gitlab-bot/internal/db"
)

type Manager struct {
	database *db.DB
	logger   *slog.Logger
}

func New(database *db.DB, logger *slog.Logger) *Manager {
	return &Manager{database: database, logger: logger}
}

func (m *Manager) Install(ctx context.Context, id string) error {
	srv, err := m.database.GetMCPServer(ctx, id)
	if err != nil {
		return fmt.Errorf("get mcp server: %w", err)
	}

	_ = m.database.UpdateMCPServerStatus(ctx, id, db.MCPStatusInstalling, nil)

	installCtx, cancel := context.WithTimeout(ctx, 3*time.Minute)
	defer cancel()

	var cmd *exec.Cmd
	switch srv.Type {
	case "npm":
		cmd = exec.CommandContext(installCtx, "npm", "install", "-g", srv.Package)
	case "binary":
		m.logger.Info("binary type mcp server, skipping install", "name", srv.Name)
		_ = m.database.UpdateMCPServerStatus(ctx, id, db.MCPStatusInstalled, nil)
		return nil
	default:
		errMsg := fmt.Sprintf("unknown mcp server type: %s", srv.Type)
		_ = m.database.UpdateMCPServerStatus(ctx, id, db.MCPStatusFailed, &errMsg)
		return fmt.Errorf("%s", errMsg)
	}

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	m.logger.Info("installing mcp server", "name", srv.Name, "package", srv.Package)

	if err := cmd.Run(); err != nil {
		errMsg := fmt.Sprintf("npm install failed: %v\n%s", err, stderr.String())
		_ = m.database.UpdateMCPServerStatus(ctx, id, db.MCPStatusFailed, &errMsg)
		return fmt.Errorf("%s", errMsg)
	}

	_ = m.database.UpdateMCPServerStatus(ctx, id, db.MCPStatusInstalled, nil)
	m.logger.Info("mcp server installed", "name", srv.Name)
	return nil
}

func (m *Manager) Uninstall(ctx context.Context, id string) error {
	srv, err := m.database.GetMCPServer(ctx, id)
	if err != nil {
		return fmt.Errorf("get mcp server: %w", err)
	}

	_ = m.database.UpdateMCPServerStatus(ctx, id, db.MCPStatusUninstalling, nil)

	if srv.Type == "npm" {
		uninstallCtx, cancel := context.WithTimeout(ctx, time.Minute)
		defer cancel()

		cmd := exec.CommandContext(uninstallCtx, "npm", "uninstall", "-g", srv.Package)
		var stderr bytes.Buffer
		cmd.Stderr = &stderr

		if err := cmd.Run(); err != nil {
			m.logger.Warn("npm uninstall failed", "name", srv.Name, "error", err, "stderr", stderr.String())
		}
	}

	return m.database.DeleteMCPServer(ctx, id)
}
