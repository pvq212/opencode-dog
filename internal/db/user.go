package db

import "context"

func (d *DB) CreateUser(ctx context.Context, u *User) error {
	return d.Pool.QueryRow(ctx,
		`INSERT INTO users (username, password_hash, display_name, role, enabled)
		 VALUES ($1,$2,$3,$4,$5) RETURNING id, created_at, updated_at`,
		u.Username, u.PasswordHash, u.DisplayName, u.Role, u.Enabled,
	).Scan(&u.ID, &u.CreatedAt, &u.UpdatedAt)
}

func (d *DB) GetUserByUsername(ctx context.Context, username string) (*User, error) {
	u := &User{}
	err := d.Pool.QueryRow(ctx,
		`SELECT id, username, password_hash, display_name, role, enabled, created_at, updated_at
		 FROM users WHERE username=$1`, username).
		Scan(&u.ID, &u.Username, &u.PasswordHash, &u.DisplayName, &u.Role, &u.Enabled, &u.CreatedAt, &u.UpdatedAt)
	return u, err
}

func (d *DB) GetUser(ctx context.Context, id string) (*User, error) {
	u := &User{}
	err := d.Pool.QueryRow(ctx,
		`SELECT id, username, password_hash, display_name, role, enabled, created_at, updated_at
		 FROM users WHERE id=$1`, id).
		Scan(&u.ID, &u.Username, &u.PasswordHash, &u.DisplayName, &u.Role, &u.Enabled, &u.CreatedAt, &u.UpdatedAt)
	return u, err
}

func (d *DB) ListUsers(ctx context.Context) ([]*User, error) {
	rows, err := d.Pool.Query(ctx,
		`SELECT id, username, password_hash, display_name, role, enabled, created_at, updated_at
		 FROM users ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var users []*User
	for rows.Next() {
		u := &User{}
		if err := rows.Scan(&u.ID, &u.Username, &u.PasswordHash, &u.DisplayName, &u.Role, &u.Enabled, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

func (d *DB) UpdateUser(ctx context.Context, u *User) error {
	_, err := d.Pool.Exec(ctx,
		`UPDATE users SET username=$2, display_name=$3, role=$4, enabled=$5 WHERE id=$1`,
		u.ID, u.Username, u.DisplayName, u.Role, u.Enabled)
	return err
}

func (d *DB) UpdateUserPassword(ctx context.Context, id string, hash string) error {
	_, err := d.Pool.Exec(ctx, `UPDATE users SET password_hash=$2 WHERE id=$1`, id, hash)
	return err
}

func (d *DB) DeleteUser(ctx context.Context, id string) error {
	_, err := d.Pool.Exec(ctx, `DELETE FROM users WHERE id=$1`, id)
	return err
}

func (d *DB) CountUsers(ctx context.Context) (int, error) {
	var count int
	err := d.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM users`).Scan(&count)
	return count, err
}
