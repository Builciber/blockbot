-- +goose Up
-- +goose StatementBegin
CREATE or REPLACE function getTradeData(telegramId BIGINT, OUT buySlippage SMALLINT, OUT sellSlippage SMALLINT, OUT maxPriceImpact SMALLINT, OUT referrerFeePercent SMALLINT, OUT referrerAddress CHAR(42), OUT priorityFee VARCHAR(10))
AS $$
DECLARE
	userInfo users%ROWTYPE;
	refAddress users.wallet_address%TYPE;
	userSettings settings%ROWTYPE;
BEGIN
	SELECT * FROM users
	INTO userInfo WHERE telegram_id = telegramId;
	
	SELECT wallet_address FROM users
	INTO refAddress WHERE telegram_id = userInfo.referrer_id;
	
	SELECT * FROM settings
	INTO userSettings WHERE telegram_id = telegramId;
	
	IF refAddress IS NULL THEN
		referrerAddress = '0x0000000000000000000000000000000000000000';
	ELSE
		referrerAddress = refAddress;
	END IF;
	buySlippage = userSettings.buy_slippage;
	sellSlippage = userSettings.sell_slippage;
	maxPriceImpact = userSettings.max_price_impact;
	referrerFeePercent = userInfo.referrer_fee_percent;
	priorityFee = userSettings.priority_fee;
END;
$$ LANGUAGE PLPGSQL;
-- +goose StatementEnd

-- +goose Down
DROP FUNCTION IF EXISTS getTradeData;