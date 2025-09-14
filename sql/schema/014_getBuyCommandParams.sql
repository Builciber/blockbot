-- +goose Up
-- +goose StatementBegin
CREATE or REPLACE function getBuyCommandParams(telegramId BIGINT, OUT buy_button_left NUMERIC, OUT buy_button_right NUMERIC, OUT sell_button_left SMALLINT, OUT sell_button_right SMALLINT, OUT walletAddress CHAR(42), OUT autoBuyEnabled BOOLEAN, OUT autoBuyAmount NUMERIC)
AS $$
DECLARE
	_walletAddress users.wallet_address%TYPE;
	userSettings settings%ROWTYPE;
BEGIN
	SELECT wallet_address FROM users
	INTO _walletAddress WHERE telegram_id = telegramId;
	
	SELECT * FROM settings
	INTO userSettings WHERE telegram_id = telegramId;

    walletAddress = _walletAddress;
    buy_button_left = userSettings.buy_button_left;
	buy_button_right = userSettings.buy_button_right;
	sell_button_left = userSettings.sell_button_left;
	sell_button_right = userSettings.sell_button_right;
    autoBuyEnabled = userSettings.auto_buy;
    autoBuyAmount = userSettings.auto_buy_amount;
END;
$$ LANGUAGE PLPGSQL;
-- +goose StatementEnd

-- +goose Down
DROP FUNCTION IF EXISTS getBuyCommandParams;