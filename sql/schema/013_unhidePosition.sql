-- +goose Up
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION unhidePosition(traderId BIGINT, tokenAddress CHAR(42)) RETURNS VOID AS $$
DECLARE
    position positions%rowtype;
BEGIN
    SELECT * FROM positions
    INTO position
    WHERE trader = traderId AND token_address = tokenAddress;

    IF position.total_mon_cost IS NOT NULL AND position.total_mon_cost = -1 THEN
        DELETE FROM positions WHERE trader = traderId AND token_address = tokenAddress;
    ELSEIF position.total_mon_cost IS NOT NULL AND position.total_mon_cost > 0 THEN
        UPDATE positions SET is_hidden = FALSE, updated_at = NOW()::TIMESTAMP
        WHERE trader = traderId AND token_address = tokenAddress;
    END IF;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

-- +goose Down
DROP FUNCTION IF EXISTS unhidePosition;