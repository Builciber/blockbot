-- name: CreateUserSettings :exec
INSERT INTO settings(telegram_id, buy_slippage, sell_slippage, max_price_impact, priority_fee, auto_buy, auto_buy_amount, buy_button_left, buy_button_right, sell_button_left, sell_button_right, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13);

-- name: GetUserSettings :one
SELECT * FROM settings
WHERE telegram_id = $1;

-- name: GetUsersSettings :many
SELECT * FROM settings;

-- name: UpdateBuySlippage :exec
UPDATE settings SET buy_slippage = $2, updated_at = $3
WHERE telegram_id = $1;

-- name: UpdateSellSlippage :exec
UPDATE settings SET sell_slippage = $2, updated_at = $3
WHERE telegram_id = $1;

-- name: UpdateMaxPriceImpact :exec
UPDATE settings SET max_price_impact = $2, updated_at = $3
WHERE telegram_id = $1;

-- name: UpdatePriorityFee :exec
UPDATE settings SET priority_fee = $2, updated_at = $3
WHERE telegram_id = $1;

-- name: UpdateAutoBuy :exec
UPDATE settings SET auto_buy = $2, updated_at = $3
WHERE telegram_id = $1;

-- name: UpdateAutoBuyAmount :exec
UPDATE settings SET auto_buy_amount = $2, updated_at = $3
WHERE telegram_id = $1;

-- name: UpdateBuyButtonLeft :exec
UPDATE settings SET buy_button_left = $2, updated_at = $3
WHERE telegram_id = $1;

-- name: UpdateBuyButtonRight :exec
UPDATE settings SET buy_button_right = $2, updated_at = $3
WHERE telegram_id = $1;

-- name: UpdateSellButtonLeft :exec
UPDATE settings SET sell_button_left = $2, updated_at = $3
WHERE telegram_id = $1;

-- name: UpdateSellButtonRight :exec
UPDATE settings SET sell_button_right = $2, updated_at = $3
WHERE telegram_id = $1;

-- name: GetTradeData :one
SELECT buySlippage, sellSlippage, maxPriceImpact, referrerFeePercent, referrerAddress, priorityFee FROM getTradeData($1);

-- name: GetBuySellButtons :one
SELECT buy_button_left, buy_button_right, sell_button_left, sell_button_right FROM settings
WHERE telegram_id = $1;

-- name: UpdateUserTradeSettings :exec
UPDATE settings SET buy_slippage = $2, sell_slippage = $3, max_price_impact = $4, priority_fee = $5, updated_at = NOW()::TIMESTAMP
WHERE telegram_id = $1;