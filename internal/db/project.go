package db

import "context"

func (d *DB) CreateProject(ctx context.Context, p *Project) error {
	return d.Pool.QueryRow(ctx,
		`INSERT INTO projects (name, ssh_url, ssh_key_id, default_branch, enabled) VALUES ($1,$2,$3,$4,$5) RETURNING id, created_at, updated_at`,
		p.Name, p.SSHURL, p.SSHKeyID, p.DefaultBranch, p.Enabled,
	).Scan(&p.ID, &p.CreatedAt, &p.UpdatedAt)
}

func (d *DB) ListProjects(ctx context.Context) ([]*Project, error) {
	rows, err := d.Pool.Query(ctx, `SELECT id, name, ssh_url, ssh_key_id, default_branch, enabled, created_at, updated_at FROM projects ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var projects []*Project
	for rows.Next() {
		p := &Project{}
		if err := rows.Scan(&p.ID, &p.Name, &p.SSHURL, &p.SSHKeyID, &p.DefaultBranch, &p.Enabled, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		projects = append(projects, p)
	}
	return projects, rows.Err()
}

func (d *DB) GetProject(ctx context.Context, id string) (*Project, error) {
	p := &Project{}
	err := d.Pool.QueryRow(ctx,
		`SELECT id, name, ssh_url, ssh_key_id, default_branch, enabled, created_at, updated_at FROM projects WHERE id=$1`, id).
		Scan(&p.ID, &p.Name, &p.SSHURL, &p.SSHKeyID, &p.DefaultBranch, &p.Enabled, &p.CreatedAt, &p.UpdatedAt)
	return p, err
}

func (d *DB) UpdateProject(ctx context.Context, p *Project) error {
	_, err := d.Pool.Exec(ctx,
		`UPDATE projects SET name=$2, ssh_url=$3, ssh_key_id=$4, default_branch=$5, enabled=$6 WHERE id=$1`,
		p.ID, p.Name, p.SSHURL, p.SSHKeyID, p.DefaultBranch, p.Enabled)
	return err
}

func (d *DB) DeleteProject(ctx context.Context, id string) error {
	_, err := d.Pool.Exec(ctx, `DELETE FROM projects WHERE id=$1`, id)
	return err
}
