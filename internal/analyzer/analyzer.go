package analyzer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/opencode-ai/opencode-gitlab-bot/internal/db"
	"github.com/opencode-ai/opencode-gitlab-bot/internal/provider"
)

type Analyzer struct {
	database  *db.DB
	registry  *provider.Registry
	logger    *slog.Logger
	configDir string
}

func New(database *db.DB, registry *provider.Registry, logger *slog.Logger, configDir string) *Analyzer {
	return &Analyzer{
		database:  database,
		registry:  registry,
		logger:    logger,
		configDir: configDir,
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

	ackBody := fmt.Sprintf("🔍 **OpenCode** received your request (%s mode).\n> Keyword: `%s` | Author: %s\n\n_Analyzing..._",
		matchedMode, matchedKeyword, msg.Author)
	if err := p.SendReply(ctx, cfgMap, msg, ackBody); err != nil {
		a.logger.Error("send ack failed", "error", err)
	}

	_ = a.database.UpdateTaskStatus(ctx, task.ID, db.TaskStatusProcessing, nil, nil)

	result, err := a.analyze(ctx, msg, matchedMode)
	if err != nil {
		a.logger.Error("analysis failed", "task_id", task.ID, "error", err)
		errMsg := err.Error()
		_ = a.database.UpdateTaskStatus(ctx, task.ID, db.TaskStatusFailed, nil, &errMsg)
		errReply := fmt.Sprintf("⚠️ **OpenCode** error:\n```\n%s\n```", err.Error())
		_ = p.SendReply(ctx, cfgMap, msg, errReply)
		return
	}

	_ = a.database.UpdateTaskStatus(ctx, task.ID, db.TaskStatusCompleted, &result, nil)

	replyBody := fmt.Sprintf("## 🤖 OpenCode Analysis\n\n%s\n\n---\n_%s mode | triggered by %s_", result, matchedMode, msg.Author)
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

func (a *Analyzer) analyze(ctx context.Context, msg *provider.IncomingMessage, mode provider.TriggerMode) (string, error) {
	prompt := buildPrompt(msg, mode)
	return a.runOpencode(ctx, prompt)
}

func (a *Analyzer) runOpencode(ctx context.Context, prompt string) (string, error) {
	if err := a.writeConfigFiles(ctx); err != nil {
		return "", fmt.Errorf("write config: %w", err)
	}

	binary := a.database.GetSettingString(ctx, "opencode_binary", "opencode")

	runCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(runCtx, binary, "-p", prompt, "-f", "json", "-q")
	cmd.Dir = a.configDir
	cmd.Env = a.buildEnv(ctx)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	a.logger.Info("running opencode", "binary", binary, "config_dir", a.configDir)

	if err := cmd.Run(); err != nil {
		stderrStr := stderr.String()
		if stderrStr != "" {
			return "", fmt.Errorf("opencode failed: %w\nstderr: %s", err, stderrStr)
		}
		return "", fmt.Errorf("opencode failed: %w", err)
	}

	output := strings.TrimSpace(stdout.String())
	if output == "" {
		return "", fmt.Errorf("opencode returned empty output")
	}
	return output, nil
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

func (a *Analyzer) buildEnv(ctx context.Context) []string {
	env := os.Environ()
	env = append(env, "HOME="+a.configDir)

	authPath := filepath.Join(a.configDir, "auth.json")
	data, err := os.ReadFile(authPath)
	if err != nil {
		return env
	}

	var authConfig map[string]any
	if err := json.Unmarshal(data, &authConfig); err != nil {
		return env
	}

	envKeys := map[string]string{
		"anthropic_api_key":  "ANTHROPIC_API_KEY",
		"openai_api_key":     "OPENAI_API_KEY",
		"gemini_api_key":     "GEMINI_API_KEY",
		"groq_api_key":       "GROQ_API_KEY",
		"openrouter_api_key": "OPENROUTER_API_KEY",
		"xai_api_key":        "XAI_API_KEY",
		"github_token":       "GITHUB_TOKEN",
	}

	for jsonKey, envKey := range envKeys {
		if val, ok := authConfig[jsonKey]; ok {
			if s, ok := val.(string); ok && s != "" {
				env = append(env, envKey+"="+s)
			}
		}
	}

	return env
}

func buildPrompt(msg *provider.IncomingMessage, mode provider.TriggerMode) string {
	var sb strings.Builder

	switch mode {
	case provider.ModeAsk:
		sb.WriteString("You are an expert software engineer. Answer the following question with a detailed, actionable response.\n\n")
	case provider.ModePlan:
		sb.WriteString("You are an expert software architect. Create a detailed implementation plan for the following request.\n\n")
	case provider.ModeDo:
		sb.WriteString("You are an expert software engineer. Provide the exact code changes needed to resolve the following issue.\n\n")
	default:
		sb.WriteString("You are an expert software engineer. Analyze the following and provide a detailed response.\n\n")
	}

	sb.WriteString(fmt.Sprintf("## Source: %s\n", msg.Provider))
	sb.WriteString(fmt.Sprintf("## Title: %s\n\n", msg.Title))
	sb.WriteString(fmt.Sprintf("### Message from @%s:\n%s\n\n", msg.Author, msg.Body))

	if msg.ExternalRef != "" {
		sb.WriteString(fmt.Sprintf("Reference: %s\n\n", msg.ExternalRef))
	}

	sb.WriteString("Format your response in Markdown.")
	return sb.String()
}

func ptrStr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
