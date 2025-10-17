-- name: CreateBadgeReceiver :exec
INSERT INTO badge_receivers(telegram_id, has_test_badge, created_at, updated_at)
VALUES($1, $2, NOW()::TIMESTAMP, NOW()::TIMESTAMP);