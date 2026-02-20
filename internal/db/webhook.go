package db

import "context"

func (d *DB) IsWebhookProcessed(ctx context.Context, eventUUID string) (bool, error) {
	var exists bool
	err := d.Pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM webhook_deliveries WHERE event_uuid=$1)`, eventUUID).Scan(&exists)
	return exists, err
}

func (d *DB) RecordWebhookDelivery(ctx context.Context, delivery *WebhookDelivery) error {
	_, err := d.Pool.Exec(ctx,
		`INSERT INTO webhook_deliveries (event_uuid, event_type, payload_hash) VALUES ($1,$2,$3) ON CONFLICT (event_uuid) DO NOTHING`,
		delivery.EventUUID, delivery.EventType, delivery.PayloadHash)
	return err
}
