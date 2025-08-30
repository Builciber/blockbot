-- +goose Up
CREATE TABLE IF NOT EXISTS transactions (
    trader BIGINT REFERENCES users(telegram_id) ON DELETE CASCADE,
    wallet_address CHAR(42) NOT NULL,
    from_token CHAR(42) NOT NULL,
    to_token CHAR(42) NOT NULL,
    from_amount NUMERIC NOT NULL,
    to_amount NUMERIC NOT NULL,
    tx_hash CHAR(66) NOT NULL UNIQUE,
    trade_unix_timestamp TIMESTAMP NOT NULL,
    webhook_event_id TEXT UNIQUE,
    created_at TIMESTAMP NOT NULL
);

-- +goose Down
DROP TABLE IF EXISTS transactions;