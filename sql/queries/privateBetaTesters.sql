-- name: InsertBetaTesters :copyfrom
INSERT INTO private_beta_testers(telegram_username)
VALUES ($1);

-- name: PrivateBetaTestersExists :one
SELECT EXISTS (SELECT 1 FROM private_beta_testers WHERE telegram_username IS NOT NULL LIMIT 1);

-- name: IsBetaTester :one
SELECT EXISTS (SELECT 1 FROM private_beta_testers WHERE telegram_username = $1);