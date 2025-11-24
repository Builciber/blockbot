-- +goose Up
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION getReferralData(telegramId BIGINT, OUT referralCode VARCHAR(8), OUT referralCount INTEGER, OUT referralEarnings NUMERIC)
AS $$
DECLARE
    userInfo users%ROWTYPE;
    refCount INTEGER;
BEGIN
    SELECT * FROM users
    INTO userInfo
    WHERE telegram_id = telegramId;

    SELECT COUNT(referrer_id) FROM users
    INTO refCount
    WHERE referrer_id = telegramId;

    referralCode = userInfo.referral_code;
    referralCount = refCount;
    referralEarnings = userInfo.referral_earnings;
END;
$$ LANGUAGE PLPGSQL;
-- +goose StatementEnd

-- +goose Down
DROP FUNCTION IF EXISTS getReferralData;