-- name: InsertBetaTesters :copyfrom
INSERT INTO private_beta_testers(telegram_username)
VALUES ($1);

-- name: PrivateBetaTestersExists :one
SELECT EXISTS (SELECT 1 FROM private_beta_testers WHERE telegram_username IS NOT NULL LIMIT 1);

-- name: IsBetaTester :one
SELECT EXISTS (SELECT 1 FROM private_beta_testers WHERE telegram_username = $1);

-- name: GetUnmessagedUsers :many
SELECT * FROM private_beta_testers WHERE sent_badge_msg = FALSE;

-- name: GetBadgeMessageState :one
SELECT sent_badge_msg, sent_feedback_badge_msg, gave_feedback FROM private_beta_testers
WHERE telegram_username = $1;

-- name: SentTestBadgeMsg :exec
UPDATE private_beta_testers SET sent_badge_msg = TRUE
WHERE telegram_username = $1;

-- name: SentFeedBackBadgeMsg :exec
UPDATE private_beta_testers SET sent_feedback_badge_msg = TRUE
WHERE telegram_username = $1;

-- name: Gavefeedback :one
SELECT gave_feedback FROM private_beta_testers
WHERE telegram_username = $1;