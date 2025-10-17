-- +goose Up
CREATE TABLE IF NOT EXISTS badge_receivers (
    telegram_id BIGINT PRIMARY KEY REFERENCES users(telegram_id),
    has_test_badge BOOLEAN NOT NULL DEFAULT FALSE,
    has_feedback_badge BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL
);

-- +goose Down
DROP TABLE IF EXISTS test_badge_receivers;