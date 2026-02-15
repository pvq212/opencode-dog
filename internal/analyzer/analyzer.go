package analyzer

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/opencode-ai/opencode-dog/internal/db"
	"github.com/opencode-ai/opencode-dog/internal/provider"
)

type Analyzer struct {
	database       *db.DB
	registry       *provider.Registry
	logger         *slog.Logger
	configDir      string
	opencodeClient *OpencodeClient
}

func New(database *db.DB, registry *provider.Registry, logger *slog.Logger, configDir string) *Analyzer {
	ctx := context.Background()
	serverURL := database.GetSettingString(ctx, "opencode_server_url", "http://opencode-server:4096")
	authUser := database.GetSettingString(ctx, "opencode_server_auth_user", "opencode")
	authPass := database.GetSettingString(ctx, "opencode_server_auth_password", "")
	timeout := database.GetSettingDuration(ctx, "analyzer_timeout", 5*time.Minute)

	client := NewOpencodeClient(serverURL, authUser, authPass, timeout, logger)

	return &Analyzer{
		database:       database,
		registry:       registry,
		logger:         logger,
		configDir:      configDir,
		opencodeClient: client,
	}
}

func (a *Analyzer) HandleMessage(ctx context.Context, msg *provider.IncomingMessage) {
	pcfg, err := a.database.GetProviderConfig(ctx, msg.ProviderCfgID)
	if err != nil {
		a.logger.Error("provider config not found", "id", msg.ProviderCfgID, "error", err)
		return
	}

	keywords, err := a.database.GetTriggerKeywords(ctx, msg.ProjectID)
	if err != nil {
		a.logger.Error("get keywords failed", "error", err)
		return
	}

	matchedKeyword, matchedMode := matchKeyword(msg.Body, keywords)
	if matchedKeyword == "" {
		return
	}

	msg.TriggerKeyword = matchedKeyword
	msg.TriggerMode = matchedMode

	task := &db.Task{
		ProjectID:        ptrStr(msg.ProjectID),
		ProviderConfigID: ptrStr(msg.ProviderCfgID),
		ProviderType:     string(msg.Provider),
		TriggerMode:      string(msg.TriggerMode),
		TriggerKeyword:   msg.TriggerKeyword,
		ExternalRef:      msg.ExternalRef,
		Title:            msg.Title,
		MessageBody:      msg.Body,
		Author:           msg.Author,
	}

	if err := a.database.CreateTask(ctx, task); err != nil {
		a.logger.Error("create task failed", "error", err)
		return
	}

	a.logger.Info("task created",
		"id", task.ID,
		"provider", msg.Provider,
		"mode", matchedMode,
		"keyword", matchedKeyword,
		"author", msg.Author,
	)

	p, ok := a.registry.Get(msg.Provider)
	if !ok {
		a.logger.Error("provider not registered", "type", msg.Provider)
		return
	}
	cfgMap := pcfg.ConfigMap()

	tpl := a.database.GetSettingString(ctx, "analyzer_ack_template",
		"🔍 **OpenCode** received your request (%s mode).\n> Keyword: `%s` | Author: %s\n\n_Analyzing..._")
	ackBody := fmt.Sprintf(tpl, matchedMode, matchedKeyword, msg.Author)
	if err := p.SendReply(ctx, cfgMap, msg, ackBody); err != nil {
		a.logger.Error("send ack failed", "error", err)
	}

	_ = a.database.UpdateTaskStatus(ctx, task.ID, db.TaskStatusProcessing, nil, nil)

	result, err := a.analyze(ctx, msg, matchedMode)
	if err != nil {
		a.logger.Error("analysis failed", "task_id", task.ID, "error", err)
		errMsg := err.Error()
		_ = a.database.UpdateTaskStatus(ctx, task.ID, db.TaskStatusFailed, nil, &errMsg)
		tpl := a.database.GetSettingString(ctx, "analyzer_error_template",
			"⚠️ **OpenCode** error:\n```\n%s\n```")
		errReply := fmt.Sprintf(tpl, err.Error())
		_ = p.SendReply(ctx, cfgMap, msg, errReply)
		return
	}

	_ = a.database.UpdateTaskStatus(ctx, task.ID, db.TaskStatusCompleted, &result, nil)

	resultTpl := a.database.GetSettingString(ctx, "analyzer_result_template",
		"## 🤖 OpenCode Analysis\n\n%s\n\n---\n_%s mode | triggered by %s_")
	replyBody := fmt.Sprintf(resultTpl, result, matchedMode, msg.Author)
	if err := p.SendReply(ctx, cfgMap, msg, replyBody); err != nil {
		a.logger.Error("send result failed", "error", err)
	}
}

func matchKeyword(text string, keywords []*db.TriggerKeyword) (string, provider.TriggerMode) {
	lower := strings.ToLower(text)
	for _, kw := range keywords {
		if strings.Contains(lower, strings.ToLower(kw.Keyword)) {
			return kw.Keyword, provider.TriggerMode(kw.Mode)
		}
	}
	return "", ""
}

func (a *Analyzer) SyncConfig(ctx context.Context) {
	if err := a.writeConfigFiles(ctx); err != nil {
		a.logger.Warn("failed to sync opencode config files", "error", err)
		return
	}
	a.logger.Info("opencode config files synced to disk", "dir", a.configDir)
}

func (a *Analyzer) analyze(ctx context.Context, msg *provider.IncomingMessage, mode provider.TriggerMode) (string, error) {
	if err := a.writeConfigFiles(ctx); err != nil {
		return "", fmt.Errorf("write config: %w", err)
	}

	prompt := a.buildPrompt(ctx, msg, mode)
	return a.runOpencodeHTTP(ctx, prompt)
}

func (a *Analyzer) runOpencodeHTTP(ctx context.Context, prompt string) (string, error) {
	title := fmt.Sprintf("Analysis %s", time.Now().Format("2006-01-02 15:04:05"))
	session, err := a.opencodeClient.CreateSession(ctx, title)
	if err != nil {
		return "", fmt.Errorf("create session: %w", err)
	}

	defer func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if delErr := a.opencodeClient.DeleteSession(cleanupCtx, session.ID); delErr != nil {
			a.logger.Warn("failed to delete session", "session_id", session.ID, "error", delErr)
		}
	}()

	result, err := a.opencodeClient.SendMessage(ctx, session.ID, prompt)
	if err != nil {
		return "", fmt.Errorf("send message: %w", err)
	}

	return result, nil
}

func (a *Analyzer) writeConfigFiles(ctx context.Context) error {
	if err := os.MkdirAll(a.configDir, 0o755); err != nil {
		return err
	}

	files := map[string]string{
		"opencode_auth_json":   "auth.json",
		"opencode_config_json": ".opencode.json",
		"opencode_ohmy_json":   "oh-my-opencode.json",
	}

	for settingKey, filename := range files {
		setting, err := a.database.GetSetting(ctx, settingKey)
		if err != nil {
			continue
		}

		var content json.RawMessage
		if err := json.Unmarshal(setting.Value, &content); err != nil {
			content = setting.Value
		}

		configPath := filepath.Join(a.configDir, filename)
		if err := os.WriteFile(configPath, content, 0o600); err != nil {
			return fmt.Errorf("write %s: %w", filename, err)
		}
	}

	return a.injectMCPServers(ctx)
}

func (a *Analyzer) injectMCPServers(ctx context.Context) error {
	servers, err := a.database.ListEnabledMCPServers(ctx)
	if err != nil || len(servers) == 0 {
		return nil
	}

	configPath := filepath.Join(a.configDir, ".opencode.json")
	existing := make(map[string]any)

	data, err := os.ReadFile(configPath)
	if err == nil {
		_ = json.Unmarshal(data, &existing)
	}

	mcpServers := make(map[string]any, len(servers))
	for _, s := range servers {
		entry := map[string]any{
			"command": s.Command,
		}
		var args []string
		_ = json.Unmarshal(s.Args, &args)
		if len(args) > 0 {
			entry["args"] = args
		}
		var env map[string]string
		_ = json.Unmarshal(s.Env, &env)
		if len(env) > 0 {
			entry["env"] = env
		}
		mcpServers[s.Name] = entry
	}

	existing["mcpServers"] = mcpServers
	merged, err := json.MarshalIndent(existing, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	return os.WriteFile(configPath, merged, 0o600)
}

func (a *Analyzer) buildPrompt(ctx context.Context, msg *provider.IncomingMessage, mode provider.TriggerMode) string {
	var sb strings.Builder

	switch mode {
	case provider.ModeAsk:
		sb.WriteString(a.database.GetSettingString(ctx, "prompt_ask",
			"You are an expert software engineer. Answer the following question with a detailed, actionable response.\n\n"))
	case provider.ModePlan:
		sb.WriteString(a.database.GetSettingString(ctx, "prompt_plan",
			"You are an expert software architect. Create a detailed implementation plan for the following request.\n\n"))
	case provider.ModeDo:
		sb.WriteString(a.database.GetSettingString(ctx, "prompt_do",
			"You are an expert software engineer. Provide the exact code changes needed to resolve the following issue.\n\n"))
	default:
		sb.WriteString(a.database.GetSettingString(ctx, "prompt_default",
			"You are an expert software engineer. Analyze the following and provide a detailed response.\n\n"))
	}

	sb.WriteString(fmt.Sprintf("## Source: %s\n", msg.Provider))
	sb.WriteString(fmt.Sprintf("## Title: %s\n\n", msg.Title))
	sb.WriteString(fmt.Sprintf("### Message from @%s:\n%s\n\n", msg.Author, msg.Body))

	if msg.ExternalRef != "" {
		sb.WriteString(fmt.Sprintf("Reference: %s\n\n", msg.ExternalRef))
	}

	sb.WriteString(a.database.GetSettingString(ctx, "prompt_format_suffix", "Format your response in Markdown."))
	return sb.String()
}

func ptrStr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
