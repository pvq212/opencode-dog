INSERT INTO settings (key, value)
VALUES ('opencode_server_url', '"http://opencode-server:4096"'::jsonb)
ON CONFLICT (key) DO NOTHING;

INSERT INTO settings (key, value)
VALUES ('opencode_server_auth_user', '"opencode"'::jsonb)
ON CONFLICT (key) DO NOTHING;

INSERT INTO settings (key, value)
VALUES ('opencode_server_auth_password', '""'::jsonb)
ON CONFLICT (key) DO NOTHING;
