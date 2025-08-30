-- +goose Up
CREATE OR REPLACE FUNCTION mutatePosition(traderId BIGINT, tokenAddress CHAR(42), mon_cost NUMERIC, token_amount NUMERIC) RETURNS VOID AS $$
DECLARE
    position positions%rowtype;
BEGIN
    SELECT * FROM positions
    INTO position
    WHERE trader = traderId AND token_address = tokenAddress;

    IF NOT found THEN
        INSERT INTO positions (trader, token_address, total_mon_cost, total_token_amount, created_at, updated_at)
        VALUES (traderId, tokenAddress, mon_cost, token_amount, NOW()::TIMESTAMP, NOW()::TIMESTAMP);
    ELSE
        UPDATE positions SET total_mon_cost = total_mon_cost + mon_cost, total_token_amount = total_token_amount + token_amount, updated_at = NOW()::TIMESTAMP
        WHERE trader = traderId AND token_address = tokenAddress;
    END IF;
END $$ LANGUAGE plpgsql;

-- +goose Down
DROP FUNCTION IF EXISTS mutatePosition;