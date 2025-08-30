-- name: InsertTransaction :exec
INSERT INTO transactions (trader, wallet_address, from_token, to_token, from_amount, to_amount, tx_hash, trade_unix_timestamp, webhook_event_id, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10);