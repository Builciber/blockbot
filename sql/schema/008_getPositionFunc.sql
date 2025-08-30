-- +goose Up
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION getPosition(traderId BIGINT, tokenAddress CHAR(42)) RETURNS positions AS $$
DECLARE
    position positions%rowtype;
BEGIN
    SELECT * FROM positions
    INTO position
    WHERE trader = traderId AND token_address = tokenAddress;

    IF position.total_mon_cost = -1 OR position.total_mon_cost IS NULL THEN
        RAISE no_data_found;
    ELSE
        return position;
    END IF;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

-- +goose Down
DROP FUNCTION IF EXISTS getPosition;