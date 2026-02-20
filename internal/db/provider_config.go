package db

import "context"

func (d *DB) CreateProviderConfig(ctx context.Context, pc *ProviderConfig) error {
	return d.Pool.QueryRow(ctx,
		`INSERT INTO provider_configs (project_id, provider_type, config, webhook_secret, webhook_path, enabled) VALUES ($1,$2,$3,$4,$5,$6) RETURNING id, created_at, updated_at`,
		pc.ProjectID, pc.ProviderType, pc.Config, pc.WebhookSecret, pc.WebhookPath, pc.Enabled,
	).Scan(&pc.ID, &pc.CreatedAt, &pc.UpdatedAt)
}

func (d *DB) ListProviderConfigs(ctx context.Context, projectID string) ([]*ProviderConfig, error) {
	rows, err := d.Pool.Query(ctx,
		`SELECT id, project_id, provider_type, config, webhook_secret, webhook_path, enabled, created_at, updated_at FROM provider_configs WHERE project_id=$1 ORDER BY created_at`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var configs []*ProviderConfig
	for rows.Next() {
		pc := &ProviderConfig{}
		if err := rows.Scan(&pc.ID, &pc.ProjectID, &pc.ProviderType, &pc.Config, &pc.WebhookSecret, &pc.WebhookPath, &pc.Enabled, &pc.CreatedAt, &pc.UpdatedAt); err != nil {
			return nil, err
		}
		configs = append(configs, pc)
	}
	return configs, rows.Err()
}

func (d *DB) GetProviderConfig(ctx context.Context, id string) (*ProviderConfig, error) {
	pc := &ProviderConfig{}
	err := d.Pool.QueryRow(ctx,
		`SELECT id, project_id, provider_type, config, webhook_secret, webhook_path, enabled, created_at, updated_at FROM provider_configs WHERE id=$1`, id).
		Scan(&pc.ID, &pc.ProjectID, &pc.ProviderType, &pc.Config, &pc.WebhookSecret, &pc.WebhookPath, &pc.Enabled, &pc.CreatedAt, &pc.UpdatedAt)
	return pc, err
}

func (d *DB) GetProviderConfigByPath(ctx context.Context, path string) (*ProviderConfig, error) {
	pc := &ProviderConfig{}
	err := d.Pool.QueryRow(ctx,
		`SELECT id, project_id, provider_type, config, webhook_secret, webhook_path, enabled, created_at, updated_at FROM provider_configs WHERE webhook_path=$1`, path).
		Scan(&pc.ID, &pc.ProjectID, &pc.ProviderType, &pc.Config, &pc.WebhookSecret, &pc.WebhookPath, &pc.Enabled, &pc.CreatedAt, &pc.UpdatedAt)
	return pc, err
}

func (d *DB) ListAllProviderConfigs(ctx context.Context) ([]*ProviderConfig, error) {
	rows, err := d.Pool.Query(ctx,
		`SELECT id, project_id, provider_type, config, webhook_secret, webhook_path, enabled, created_at, updated_at FROM provider_configs WHERE enabled=true ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var configs []*ProviderConfig
	for rows.Next() {
		pc := &ProviderConfig{}
		if err := rows.Scan(&pc.ID, &pc.ProjectID, &pc.ProviderType, &pc.Config, &pc.WebhookSecret, &pc.WebhookPath, &pc.Enabled, &pc.CreatedAt, &pc.UpdatedAt); err != nil {
			return nil, err
		}
		configs = append(configs, pc)
	}
	return configs, rows.Err()
}

func (d *DB) DeleteProviderConfig(ctx context.Context, id string) error {
	_, err := d.Pool.Exec(ctx, `DELETE FROM provider_configs WHERE id=$1`, id)
	return err
}
