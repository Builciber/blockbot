-- +goose Up
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION hidePosition(traderId BIGINT, tokenAddress CHAR(42)) RETURNS VOID AS $$
DECLARE
    position positions%rowtype;
BEGIN
    SELECT * FROM positions
    INTO position
    WHERE trader = traderId AND token_address = tokenAddress;

    IF NOT found THEN
        INSERT INTO positions (trader, token_address, total_mon_cost, total_token_amount, is_hidden, created_at, updated_at)
        VALUES (traderId, tokenAddress, -1, -1, TRUE, NOW()::timestamp,  NOW()::timestamp);
    ELSE
        UPDATE positions SET is_hidden = TRUE, updated_at = NOW()::timestamp
        WHERE trader = traderId AND token_address = tokenAddress;
    END IF;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

-- +goose Down
DROP FUNCTION IF EXISTS hidePosition;