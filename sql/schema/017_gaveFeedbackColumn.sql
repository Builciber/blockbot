-- +goose Up
alter table private_beta_testers add column if not exists gave_feedback boolean default false;

-- +goose Down
DROP COLUMN IF EXISTS gave_feedback;