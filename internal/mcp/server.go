package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/opencode-ai/opencode-gitlab-bot/internal/db"
)

type Server struct {
	mcpServer *server.MCPServer
	database  *db.DB
	logger    *slog.Logger
}

func NewServer(database *db.DB, logger *slog.Logger) *Server {
	s := &Server{
		database: database,
		logger:   logger,
	}

	s.mcpServer = server.NewMCPServer(
		"opencode-bot",
		"2.0.0",
		server.WithToolCapabilities(false),
	)

	s.registerTools()
	return s
}

func (s *Server) registerTools() {
	listProjects := mcp.NewTool("list_projects",
		mcp.WithDescription("List all configured projects"),
	)
	s.mcpServer.AddTool(listProjects, s.handleListProjects)

	getProject := mcp.NewTool("get_project",
		mcp.WithDescription("Get details of a project by ID"),
		mcp.WithString("project_id", mcp.Required(), mcp.Description("Project UUID")),
	)
	s.mcpServer.AddTool(getProject, s.handleGetProject)

	listTasks := mcp.NewTool("list_tasks",
		mcp.WithDescription("List recent analysis tasks"),
		mcp.WithString("limit", mcp.Description("Maximum number of tasks to return (default: 20)")),
		mcp.WithString("offset", mcp.Description("Offset for pagination (default: 0)")),
	)
	s.mcpServer.AddTool(listTasks, s.handleListTasks)

	getTask := mcp.NewTool("get_task",
		mcp.WithDescription("Get details and result of an analysis task"),
		mcp.WithString("task_id", mcp.Required(), mcp.Description("Task UUID")),
	)
	s.mcpServer.AddTool(getTask, s.handleGetTask)

	listProviders := mcp.NewTool("list_providers",
		mcp.WithDescription("List provider configurations for a project"),
		mcp.WithString("project_id", mcp.Required(), mcp.Description("Project UUID")),
	)
	s.mcpServer.AddTool(listProviders, s.handleListProviders)
}

func (s *Server) handleListProjects(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projects, err := s.database.ListProjects(ctx)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to list projects: %v", err)), nil
	}

	var sb strings.Builder
	for _, p := range projects {
		status := "enabled"
		if !p.Enabled {
			status = "disabled"
		}
		sb.WriteString(fmt.Sprintf("- **%s** (ID: %s) [%s]\n  SSH: %s | Branch: %s\n", p.Name, p.ID, status, p.SSHURL, p.DefaultBranch))
	}

	if sb.Len() == 0 {
		return mcp.NewToolResultText("No projects configured."), nil
	}
	return mcp.NewToolResultText(sb.String()), nil
}

func (s *Server) handleGetProject(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectID, err := request.RequireString("project_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	p, err := s.database.GetProject(ctx, projectID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("project not found: %v", err)), nil
	}

	data, _ := json.MarshalIndent(p, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}

func (s *Server) handleListTasks(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	limit := optionalInt(request, "limit", 20)
	offset := optionalInt(request, "offset", 0)

	tasks, err := s.database.ListTasks(ctx, limit, offset)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to list tasks: %v", err)), nil
	}

	var sb strings.Builder
	for _, t := range tasks {
		sb.WriteString(fmt.Sprintf("### Task %s\n", t.ID))
		sb.WriteString(fmt.Sprintf("- **Status**: %s\n", t.Status))
		sb.WriteString(fmt.Sprintf("- **Provider**: %s\n", t.ProviderType))
		sb.WriteString(fmt.Sprintf("- **Mode**: %s (keyword: %s)\n", t.TriggerMode, t.TriggerKeyword))
		sb.WriteString(fmt.Sprintf("- **Author**: %s\n", t.Author))
		sb.WriteString(fmt.Sprintf("- **Title**: %s\n", t.Title))
		sb.WriteString(fmt.Sprintf("- **Created**: %s\n\n", t.CreatedAt.Format("2006-01-02 15:04")))
	}

	if sb.Len() == 0 {
		return mcp.NewToolResultText("No tasks found."), nil
	}
	return mcp.NewToolResultText(sb.String()), nil
}

func (s *Server) handleGetTask(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	taskID, err := request.RequireString("task_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	tasks, err := s.database.ListTasks(ctx, 100, 0)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to query tasks: %v", err)), nil
	}

	for _, t := range tasks {
		if t.ID == taskID {
			data, _ := json.MarshalIndent(t, "", "  ")
			return mcp.NewToolResultText(string(data)), nil
		}
	}

	return mcp.NewToolResultError("task not found"), nil
}

func (s *Server) handleListProviders(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectID, err := request.RequireString("project_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	configs, err := s.database.ListProviderConfigs(ctx, projectID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to list providers: %v", err)), nil
	}

	var sb strings.Builder
	for _, pc := range configs {
		status := "enabled"
		if !pc.Enabled {
			status = "disabled"
		}
		sb.WriteString(fmt.Sprintf("- **%s** (ID: %s) [%s]\n  Webhook: %s\n", pc.ProviderType, pc.ID, status, pc.WebhookPath))
	}

	if sb.Len() == 0 {
		return mcp.NewToolResultText("No providers configured for this project."), nil
	}
	return mcp.NewToolResultText(sb.String()), nil
}

func (s *Server) GetServer() *server.MCPServer {
	return s.mcpServer
}

func optionalInt(req mcp.CallToolRequest, key string, fallback int) int {
	args := req.GetArguments()
	if v, ok := args[key]; ok {
		switch val := v.(type) {
		case string:
			if i, err := strconv.Atoi(val); err == nil {
				return i
			}
		case float64:
			return int(val)
		}
	}
	return fallback
}
