-- name: CreateBadgeReceiver :exec
INSERT INTO badge_receivers(telegram_id, has_test_badge, created_at, updated_at)
VALUES($1, $2, NOW()::TIMESTAMP, NOW()::TIMESTAMP);

-- name: SendTestBage :exec
UPDATE badge_receivers SET has_test_badge = TRUE
WHERE telegram_id = $1;

-- name: SendFeedbackBadge :exec
UPDATE badge_receivers SET has_feedback_badge = TRUE
WHERE telegram_id = $1;