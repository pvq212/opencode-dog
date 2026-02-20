package db

import "context"

func (d *DB) SetTriggerKeywords(ctx context.Context, projectID string, keywords []TriggerKeyword) error {
	tx, err := d.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `DELETE FROM trigger_keywords WHERE project_id=$1`, projectID)
	if err != nil {
		return err
	}
	for _, kw := range keywords {
		_, err = tx.Exec(ctx,
			`INSERT INTO trigger_keywords (project_id, mode, keyword) VALUES ($1,$2,$3) ON CONFLICT (project_id, keyword) DO UPDATE SET mode=$2`,
			projectID, kw.Mode, kw.Keyword)
		if err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (d *DB) GetTriggerKeywords(ctx context.Context, projectID string) ([]*TriggerKeyword, error) {
	rows, err := d.Pool.Query(ctx,
		`SELECT id, project_id, mode, keyword, created_at FROM trigger_keywords WHERE project_id=$1 ORDER BY mode, keyword`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var keywords []*TriggerKeyword
	for rows.Next() {
		kw := &TriggerKeyword{}
		if err := rows.Scan(&kw.ID, &kw.ProjectID, &kw.Mode, &kw.Keyword, &kw.CreatedAt); err != nil {
			return nil, err
		}
		keywords = append(keywords, kw)
	}
	return keywords, rows.Err()
}
