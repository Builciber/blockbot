-- +goose Up
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION mutatePositionSell(traderId BIGINT, tokenAddress CHAR(42), token_amount NUMERIC) RETURNS VOID AS $$
DECLARE
    position positions%rowtype;
BEGIN
    SELECT * FROM positions
    INTO position
    WHERE trader = traderId AND token_address = tokenAddress;

    IF found THEN
        IF token_amount < position.total_token_amount THEN
            UPDATE positions SET total_mon_cost = total_mon_cost - (token_amount * (total_mon_cost / position.total_token_amount)), total_token_amount = total_token_amount - token_amount, updated_at = NOW()::TIMESTAMP
            WHERE trader = traderId AND token_address = tokenAddress;
        ELSE
            DELETE FROM positions
            WHERE trader = traderId AND token_address = tokenAddress;
        END IF;
    END IF;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

-- +goose Down
DROP FUNCTION IF EXISTS mutatePositionSell;