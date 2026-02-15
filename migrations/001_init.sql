CREATE EXTENSION IF NOT EXISTS "pgcrypto";

DO $$ BEGIN
    CREATE TYPE task_status AS ENUM ('pending','processing','completed','failed','cancelled');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

DO $$ BEGIN
    CREATE TYPE trigger_mode AS ENUM ('ask','plan','do');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

CREATE TABLE IF NOT EXISTS ssh_keys (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        TEXT NOT NULL,
    private_key TEXT NOT NULL,
    public_key  TEXT NOT NULL,
    created_at  TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS projects (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name            TEXT NOT NULL,
    ssh_url         TEXT NOT NULL,
    ssh_key_id      UUID REFERENCES ssh_keys(id) ON DELETE SET NULL,
    default_branch  TEXT NOT NULL DEFAULT 'main',
    enabled         BOOLEAN NOT NULL DEFAULT TRUE,
    created_at      TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS provider_configs (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id      UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    provider_type   TEXT NOT NULL,
    config          JSONB NOT NULL DEFAULT '{}',
    webhook_secret  TEXT NOT NULL DEFAULT '',
    webhook_path    TEXT NOT NULL DEFAULT '',
    enabled         BOOLEAN NOT NULL DEFAULT TRUE,
    created_at      TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS trigger_keywords (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id  UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    mode        trigger_mode NOT NULL,
    keyword     TEXT NOT NULL,
    created_at  TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    UNIQUE(project_id, keyword)
);

CREATE TABLE IF NOT EXISTS tasks (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id         UUID REFERENCES projects(id) ON DELETE SET NULL,
    provider_config_id UUID REFERENCES provider_configs(id) ON DELETE SET NULL,
    provider_type      TEXT NOT NULL,
    trigger_mode       TEXT NOT NULL DEFAULT 'ask',
    trigger_keyword    TEXT NOT NULL DEFAULT '',
    external_ref       TEXT NOT NULL DEFAULT '',
    title              TEXT NOT NULL DEFAULT '',
    message_body       TEXT NOT NULL DEFAULT '',
    author             TEXT NOT NULL DEFAULT '',
    status             task_status NOT NULL DEFAULT 'pending',
    result             TEXT,
    error_message      TEXT,
    created_at         TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    started_at         TIMESTAMP WITH TIME ZONE,
    completed_at       TIMESTAMP WITH TIME ZONE
);

CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks(status);
CREATE INDEX IF NOT EXISTS idx_tasks_project_id ON tasks(project_id);
CREATE INDEX IF NOT EXISTS idx_tasks_created_at ON tasks(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_provider_configs_project ON provider_configs(project_id);
CREATE INDEX IF NOT EXISTS idx_trigger_keywords_project ON trigger_keywords(project_id);

CREATE TABLE IF NOT EXISTS webhook_deliveries (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_uuid   TEXT UNIQUE NOT NULL,
    event_type   TEXT NOT NULL,
    payload_hash TEXT NOT NULL,
    processed    BOOLEAN NOT NULL DEFAULT FALSE,
    created_at   TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS settings (
    key        TEXT PRIMARY KEY,
    value      JSONB NOT NULL DEFAULT '{}',
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

INSERT INTO settings (key, value) VALUES
    ('mcp_enabled', 'true'::jsonb),
    ('mcp_endpoint', '"/mcp"'::jsonb),
    ('opencode_binary', '"opencode"'::jsonb),
    ('opencode_auth_json', '{}'::jsonb),
    ('opencode_config_json', '{}'::jsonb),
    ('opencode_ohmy_json', '{}'::jsonb)
ON CONFLICT (key) DO NOTHING;

INSERT INTO settings (key, value) VALUES
    ('analyzer_timeout', '"5m"'),
    ('analyzer_ack_template', '"🔍 **OpenCode** received your request (%s mode).\n> Keyword: `%s` | Author: %s\n\n_Analyzing..._"'),
    ('analyzer_error_template', '"⚠️ **OpenCode** error:\n```\n%s\n```"'),
    ('analyzer_result_template', '"## 🤖 OpenCode Analysis\n\n%s\n\n---\n_%s mode | triggered by %s_"'),
    ('prompt_ask', '"You are an expert software engineer. Answer the following question with a detailed, actionable response.\n\n"'),
    ('prompt_plan', '"You are an expert software architect. Create a detailed implementation plan for the following request.\n\n"'),
    ('prompt_do', '"You are an expert software engineer. Provide the exact code changes needed to resolve the following issue.\n\n"'),
    ('prompt_default', '"You are an expert software engineer. Analyze the following and provide a detailed response.\n\n"'),
    ('prompt_format_suffix', '"Format your response in Markdown."'),
    ('token_ttl', '"24h"'),
    ('mcp_install_timeout', '"3m"'),
    ('mcp_uninstall_timeout', '"1m"'),
    ('slack_http_timeout', '"30s"'),
    ('telegram_http_timeout', '"30s"'),
    ('telegram_parse_mode', '"Markdown"'),
    ('task_list_default_limit', '50'),
    ('task_list_max_limit', '100'),
    ('default_git_branch', '"main"')
ON CONFLICT (key) DO NOTHING;

DO $$ BEGIN
    CREATE TYPE mcp_server_status AS ENUM ('pending','installing','installed','failed','uninstalling');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

CREATE TABLE IF NOT EXISTS mcp_servers (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        TEXT NOT NULL UNIQUE,
    type        TEXT NOT NULL DEFAULT 'npm',
    package     TEXT NOT NULL,
    command     TEXT NOT NULL DEFAULT '',
    args        JSONB NOT NULL DEFAULT '[]',
    env         JSONB NOT NULL DEFAULT '{}',
    enabled     BOOLEAN NOT NULL DEFAULT TRUE,
    status      mcp_server_status NOT NULL DEFAULT 'pending',
    error_msg   TEXT,
    created_at  TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_mcp_servers_enabled ON mcp_servers(enabled);

CREATE TABLE IF NOT EXISTS users (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username      TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    display_name  TEXT NOT NULL DEFAULT '',
    role          TEXT NOT NULL DEFAULT 'viewer',
    enabled       BOOLEAN NOT NULL DEFAULT TRUE,
    created_at    TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN NEW.updated_at = NOW(); RETURN NEW; END;
$$ LANGUAGE plpgsql;

DO $$ BEGIN
    CREATE TRIGGER trg_projects_updated BEFORE UPDATE ON projects FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;
DO $$ BEGIN
    CREATE TRIGGER trg_provider_configs_updated BEFORE UPDATE ON provider_configs FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;
DO $$ BEGIN
    CREATE TRIGGER trg_tasks_updated BEFORE UPDATE ON tasks FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;
DO $$ BEGIN
    CREATE TRIGGER trg_settings_updated BEFORE UPDATE ON settings FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;
DO $$ BEGIN
    CREATE TRIGGER trg_mcp_servers_updated BEFORE UPDATE ON mcp_servers FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;
DO $$ BEGIN
    CREATE TRIGGER trg_users_updated BEFORE UPDATE ON users FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;
