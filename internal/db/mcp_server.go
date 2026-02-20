package db

import "context"

func (d *DB) CreateMCPServer(ctx context.Context, m *MCPServer) error {
	return d.Pool.QueryRow(ctx,
		`INSERT INTO mcp_servers (name, type, package, command, args, env, enabled)
		 VALUES ($1,$2,$3,$4,$5,$6,$7) RETURNING id, created_at, updated_at`,
		m.Name, m.Type, m.Package, m.Command, m.Args, m.Env, m.Enabled,
	).Scan(&m.ID, &m.CreatedAt, &m.UpdatedAt)
}

func (d *DB) ListMCPServers(ctx context.Context) ([]*MCPServer, error) {
	rows, err := d.Pool.Query(ctx,
		`SELECT id, name, type, package, command, args, env, enabled, status, error_msg, created_at, updated_at
		 FROM mcp_servers ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var servers []*MCPServer
	for rows.Next() {
		m := &MCPServer{}
		if err := rows.Scan(&m.ID, &m.Name, &m.Type, &m.Package, &m.Command, &m.Args, &m.Env, &m.Enabled, &m.Status, &m.ErrorMsg, &m.CreatedAt, &m.UpdatedAt); err != nil {
			return nil, err
		}
		servers = append(servers, m)
	}
	return servers, rows.Err()
}

func (d *DB) ListEnabledMCPServers(ctx context.Context) ([]*MCPServer, error) {
	rows, err := d.Pool.Query(ctx,
		`SELECT id, name, type, package, command, args, env, enabled, status, error_msg, created_at, updated_at
		 FROM mcp_servers WHERE enabled=true AND status='installed' ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var servers []*MCPServer
	for rows.Next() {
		m := &MCPServer{}
		if err := rows.Scan(&m.ID, &m.Name, &m.Type, &m.Package, &m.Command, &m.Args, &m.Env, &m.Enabled, &m.Status, &m.ErrorMsg, &m.CreatedAt, &m.UpdatedAt); err != nil {
			return nil, err
		}
		servers = append(servers, m)
	}
	return servers, rows.Err()
}

func (d *DB) GetMCPServer(ctx context.Context, id string) (*MCPServer, error) {
	m := &MCPServer{}
	err := d.Pool.QueryRow(ctx,
		`SELECT id, name, type, package, command, args, env, enabled, status, error_msg, created_at, updated_at
		 FROM mcp_servers WHERE id=$1`, id).
		Scan(&m.ID, &m.Name, &m.Type, &m.Package, &m.Command, &m.Args, &m.Env, &m.Enabled, &m.Status, &m.ErrorMsg, &m.CreatedAt, &m.UpdatedAt)
	return m, err
}

func (d *DB) UpdateMCPServerStatus(ctx context.Context, id string, status MCPServerStatus, errMsg *string) error {
	_, err := d.Pool.Exec(ctx,
		`UPDATE mcp_servers SET status=$2, error_msg=$3 WHERE id=$1`,
		id, status, errMsg)
	return err
}

func (d *DB) UpdateMCPServer(ctx context.Context, m *MCPServer) error {
	_, err := d.Pool.Exec(ctx,
		`UPDATE mcp_servers SET name=$2, type=$3, package=$4, command=$5, args=$6, env=$7, enabled=$8 WHERE id=$1`,
		m.ID, m.Name, m.Type, m.Package, m.Command, m.Args, m.Env, m.Enabled)
	return err
}

func (d *DB) DeleteMCPServer(ctx context.Context, id string) error {
	_, err := d.Pool.Exec(ctx, `DELETE FROM mcp_servers WHERE id=$1`, id)
	return err
}
