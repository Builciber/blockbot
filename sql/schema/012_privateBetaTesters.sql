-- +goose Up
CREATE TABLE IF NOT EXISTS private_beta_testers (
    telegram_username VARCHAR(32) PRIMARY KEY
);

-- +goose Down
DROP TABLE IF EXISTS private_beta_testers;