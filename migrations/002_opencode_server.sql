-- Migration: OpenCode Server Mode
-- Description: Add configuration settings for OpenCode HTTP API client
-- Author: System Migration
-- Date: 2026-02-16

-- This migration adds three new settings required for OpenCode Server Mode:
-- 1. opencode_server_url: Base URL for the OpenCode HTTP API server
-- 2. opencode_server_auth_user: HTTP Basic Auth username
-- 3. opencode_server_auth_password: HTTP Basic Auth password

-- Note: These settings complement the existing opencode_auth_json, 
-- opencode_config_json, and opencode_ohmy_json settings, which are 
-- still used to generate config files for the OpenCode server.

-- The opencode_binary setting is now deprecated but kept for backward compatibility.

-- OpenCode server base URL (Docker service name by default)
INSERT INTO settings (key, value, description, created_at, updated_at)
VALUES (
    'opencode_server_url',
    '"http://opencode-server:4096"'::jsonb,
    'OpenCode server base URL for HTTP API client. Default uses Docker service name.',
    NOW(),
    NOW()
) ON CONFLICT (key) DO NOTHING;

-- OpenCode server Basic Auth username
INSERT INTO settings (key, value, description, created_at, updated_at)
VALUES (
    'opencode_server_auth_user',
    '"opencode"'::jsonb,
    'OpenCode server HTTP Basic Auth username. Should match OPENCODE_SERVER_USERNAME env var.',
    NOW(),
    NOW()
) ON CONFLICT (key) DO NOTHING;

-- OpenCode server Basic Auth password (REQUIRED - must be set via WebUI)
INSERT INTO settings (key, value, description, created_at, updated_at)
VALUES (
    'opencode_server_auth_password',
    '""'::jsonb,
    'OpenCode server HTTP Basic Auth password (required). Must match OPENCODE_SERVER_PASSWORD env var. Set this via Settings page in WebUI after first deployment.',
    NOW(),
    NOW()
) ON CONFLICT (key) DO NOTHING;

-- Verify migration
-- Run this after migration to check settings were created:
-- SELECT key, value, description FROM settings WHERE key LIKE 'opencode_server%';
