-- +goose Up
CREATE TABLE IF NOT EXISTS users (
    telegram_id BIGINT PRIMARY KEY,
    wallet_address CHAR(42) NOT NULL UNIQUE,
    referrer_id BIGINT,
    referral_code CHAR(8) NOT NULL UNIQUE,
    referrer_fee_percent SMALLINT NOT NULL CONSTRAINT check_referrer_fee_percent CHECK(referrer_fee_percent > 0 AND referrer_fee_percent < 76) DEFAULT 25,
    referral_earnings NUMERIC NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL
);

-- +goose Down
DROP TABLE IF EXISTS users;