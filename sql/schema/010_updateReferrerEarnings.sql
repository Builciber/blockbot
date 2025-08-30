-- +goose Up
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION updateReferrerEarnings(telegramId BIGINT, referrerEarnings NUMERIC) RETURNS VOID AS $$
DECLARE
    userInfo users%rowtype;
BEGIN
    SELECT * FROM users
    INTO userInfo STRICT
    WHERE telegram_id = telegramId;

    IF userInfo.referrer_id IS NOT NULL THEN
        UPDATE users SET referral_earnings = referral_earnings + referrerEarnings, updated_at = NOW()::TIMESTAMP
        WHERE telegram_id = userInfo.referrer_id;
    END IF;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

-- +goose Down
DROP FUNCTION IF EXISTS updateReferrerEarnings;