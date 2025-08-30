-- name: CreatePosition :exec
INSERT INTO positions (trader, token_address, total_mon_cost, total_token_amount, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6);

-- name: GetPosition :one
SELECT is_hidden, total_mon_cost, total_token_amount, CAST(total_mon_cost / total_token_amount AS NUMERIC) AS average_mon_price FROM positions
WHERE trader = $1 AND token_address = $2;

-- name: GetUserPositions :many
SELECT token_address, total_mon_cost, CAST(total_mon_cost / total_token_amount AS NUMERIC) AS average_mon_price FROM positions
WHERE trader = $1 AND is_hidden = FALSE;

-- name: GetHiddenPositions :many
SELECT token_address FROM positions
WHERE trader = $1 AND is_hidden = TRUE;

-- name: UpdatePosition :exec
UPDATE positions SET total_mon_cost = $2, total_token_amount = $3, updated_at = $4
WHERE trader = $1;

-- name: DeleteUserPositions :exec
DELETE FROM positions WHERE trader = $1;

-- name: DeletePosition :exec
DELETE FROM positions WHERE trader = $1 AND token_address = $2;

-- name: HidePosition :exec
SELECT hidePosition(traderId => $1, tokenAddress => $2);

-- name: UnhidePosition :exec
SELECT unhidePosition(traderId => $1, tokenAddress => $2);

-- name: MutatePosition :exec
SELECT mutatePosition(traderId => $1, tokenAddress => $2, mon_cost => $3, token_amount => $4);

-- name: CallGetPositionFunc :one
SELECT trader, token_address, total_mon_cost, total_token_amount, CAST(total_mon_cost / total_token_amount AS NUMERIC) AS average_mon_price FROM getPosition($1, $2);

-- name: MutatePositionSell :exec
SELECT mutatePositionSell(traderId => $1, tokenAddress => $2, token_amount => $3);