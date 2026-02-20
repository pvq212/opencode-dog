package db

import (
	"context"
	"time"
)

func (d *DB) CreateTask(ctx context.Context, t *Task) error {
	return d.Pool.QueryRow(ctx,
		`INSERT INTO tasks (project_id, provider_config_id, provider_type, trigger_mode, trigger_keyword, external_ref, title, message_body, author)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9) RETURNING id, created_at, updated_at`,
		t.ProjectID, t.ProviderConfigID, t.ProviderType, t.TriggerMode, t.TriggerKeyword,
		t.ExternalRef, t.Title, t.MessageBody, t.Author,
	).Scan(&t.ID, &t.CreatedAt, &t.UpdatedAt)
}

func (d *DB) UpdateTaskStatus(ctx context.Context, taskID string, status TaskStatus, result *string, errMsg *string) error {
	now := time.Now()
	var startedAt, completedAt *time.Time
	switch status {
	case TaskStatusProcessing:
		startedAt = &now
	case TaskStatusCompleted, TaskStatusFailed:
		completedAt = &now
	}
	_, err := d.Pool.Exec(ctx,
		`UPDATE tasks SET status=$2, result=$3, error_message=$4, started_at=COALESCE($5, started_at), completed_at=COALESCE($6, completed_at) WHERE id=$1`,
		taskID, status, result, errMsg, startedAt, completedAt)
	return err
}

func (d *DB) ListTasks(ctx context.Context, limit, offset int) ([]*Task, error) {
	rows, err := d.Pool.Query(ctx,
		`SELECT id, project_id, provider_config_id, provider_type, trigger_mode, trigger_keyword, external_ref, title, message_body, author, status, result, error_message, created_at, updated_at, started_at, completed_at
		 FROM tasks ORDER BY created_at DESC LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var tasks []*Task
	for rows.Next() {
		t := &Task{}
		if err := rows.Scan(&t.ID, &t.ProjectID, &t.ProviderConfigID, &t.ProviderType, &t.TriggerMode, &t.TriggerKeyword, &t.ExternalRef, &t.Title, &t.MessageBody, &t.Author, &t.Status, &t.Result, &t.ErrorMessage, &t.CreatedAt, &t.UpdatedAt, &t.StartedAt, &t.CompletedAt); err != nil {
			return nil, err
		}
		tasks = append(tasks, t)
	}
	return tasks, rows.Err()
}

func (d *DB) GetTask(ctx context.Context, id string) (*Task, error) {
	t := &Task{}
	err := d.Pool.QueryRow(ctx,
		`SELECT id, project_id, provider_config_id, provider_type, trigger_mode, trigger_keyword, external_ref, title, message_body, author, status, result, error_message, created_at, updated_at, started_at, completed_at
		 FROM tasks WHERE id=$1`, id).Scan(&t.ID, &t.ProjectID, &t.ProviderConfigID, &t.ProviderType, &t.TriggerMode, &t.TriggerKeyword, &t.ExternalRef, &t.Title, &t.MessageBody, &t.Author, &t.Status, &t.Result, &t.ErrorMessage, &t.CreatedAt, &t.UpdatedAt, &t.StartedAt, &t.CompletedAt)
	if err != nil {
		return nil, err
	}
	return t, nil
}

func (d *DB) CountTasks(ctx context.Context) (int, error) {
	var count int
	err := d.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM tasks`).Scan(&count)
	return count, err
}
