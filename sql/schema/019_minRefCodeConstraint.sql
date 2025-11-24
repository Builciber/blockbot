-- +goose Up
alter table users add constraint if not exists referral_code_min_length check(length(referral_code) >= 4);

-- +goose Down
alter table users drop constraint if exists referral_code_min_length;