-- +goose Up
alter table private_beta_testers add column if not exists sent_badge_msg boolean default false;

-- +goose Down
DROP COLUMN IF EXISTS sent_badge_msg;