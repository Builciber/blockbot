-- +goose Up
CREATE TABLE IF NOT EXISTS positions (
    trader BIGINT NOT NULL REFERENCES users(telegram_id) ON DELETE CASCADE,
    token_address CHAR(42) NOT NULL,
    total_mon_cost NUMERIC NOT NULL CONSTRAINT check_total_cost CHECK(total_mon_cost > 0 OR total_mon_cost = -1),
    total_token_amount NUMERIC NOT NULL CONSTRAINT check_total_amount CHECK(total_token_amount > 0 OR total_token_amount = -1),
    is_hidden BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    CONSTRAINT per_user_token_position UNIQUE(trader, token_address)
);

-- +goose Down
DROP TABLE IF EXISTS positions;