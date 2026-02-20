package db

import "context"

func (d *DB) CreateSSHKey(ctx context.Context, k *SSHKey) error {
	return d.Pool.QueryRow(ctx,
		`INSERT INTO ssh_keys (name, private_key, public_key) VALUES ($1,$2,$3) RETURNING id, created_at`,
		k.Name, k.PrivateKey, k.PublicKey,
	).Scan(&k.ID, &k.CreatedAt)
}

func (d *DB) ListSSHKeys(ctx context.Context) ([]*SSHKey, error) {
	rows, err := d.Pool.Query(ctx, `SELECT id, name, public_key, created_at FROM ssh_keys ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var keys []*SSHKey
	for rows.Next() {
		k := &SSHKey{}
		if err := rows.Scan(&k.ID, &k.Name, &k.PublicKey, &k.CreatedAt); err != nil {
			return nil, err
		}
		keys = append(keys, k)
	}
	return keys, rows.Err()
}

func (d *DB) GetSSHKey(ctx context.Context, id string) (*SSHKey, error) {
	k := &SSHKey{}
	err := d.Pool.QueryRow(ctx, `SELECT id, name, private_key, public_key, created_at FROM ssh_keys WHERE id=$1`, id).
		Scan(&k.ID, &k.Name, &k.PrivateKey, &k.PublicKey, &k.CreatedAt)
	return k, err
}

func (d *DB) DeleteSSHKey(ctx context.Context, id string) error {
	_, err := d.Pool.Exec(ctx, `DELETE FROM ssh_keys WHERE id=$1`, id)
	return err
}
