-- name: CreateUser :exec
INSERT INTO users(telegram_id, wallet_address, referrer_id, referral_code, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6);

-- name: GetUser :one
SELECT * FROM users
WHERE telegram_id = $1;

-- name: GetUserByRefCode :one
SELECT telegram_id FROM users
WHERE referral_code = $1;

-- name: IsExistingUser :one
SELECT EXISTS (SELECT 1 FROM users WHERE users.telegram_id = $1 LIMIT 1);

-- name: IsExistingRefCode :one
SELECT EXISTS (SELECT 1 FROM users WHERE users.referral_code = $1 LIMIT 1);

-- name: GetWalletAddress :one
SELECT wallet_address FROM users
WHERE telegram_id = $1;

-- name: GetReferralCode :one
SELECT referral_code FROM users
WHERE telegram_id = $1;

-- name: GetReferralCount :one
SELECT COUNT(referrer_id) FROM users
WHERE referrer_id = $1;

-- name: UpdateWallet :exec
UPDATE users SET wallet_address = $2, updated_at = $3
WHERE telegram_id = $1;

-- name: UpdateReferrerEarnings :exec
SELECT updateReferrerEarnings(telegramId => $1, referrerEarnings => $2);

-- name: GetReferralData :one
SELECT referralCode, referralCount, referralEarnings FROM getReferralData(telegramId => $1);

-- name: GetBuyCommandParams :one
SELECT * FROM getBuyCommandParams(telegramId => $1);