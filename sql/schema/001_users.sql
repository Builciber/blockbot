-- +goose Up
CREATE TABLE IF NOT EXISTS users (
    telegram_id BIGINT PRIMARY KEY,
    wallet_address CHAR(42) NOT NULL UNIQUE,
    referrer_id BIGINT,
    referral_code CHAR(8) NOT NULL UNIQUE,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL
);

-- +goose Down
DROP TABLE IF EXISTS users;