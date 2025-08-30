-- +goose Up
CREATE TABLE IF NOT EXISTS settings (
    telegram_id BIGINT PRIMARY KEY REFERENCES users(telegram_id) ON DELETE CASCADE,
    buy_slippage SMALLINT NOT NULL CONSTRAINT check_buy_slippage CHECK (buy_slippage > 0 AND buy_slippage < 76),
    sell_slippage SMALLINT NOT NULL CONSTRAINT check_sell_slippage CHECK (sell_slippage > 0 AND sell_slippage < 76),
    max_price_impact SMALLINT NOT NULL CONSTRAINT check_price_impact CHECK (max_price_impact > 0 AND max_price_impact < 101),
    priority_fee VARCHAR(10) NOT NULL CONSTRAINT check_priority_fee CHECK (priority_fee IN ('normal', 'fast', 'very fast', 'turbo')),
    auto_buy BOOLEAN NOT NULL DEFAULT FALSE,
    auto_buy_amount NUMERIC CONSTRAINT check_auto_buy_amount CHECK(auto_buy_amount > 0) DEFAULT 0.1,
    buy_button_left NUMERIC NOT NULL CONSTRAINT check_buy_button_left CHECK(buy_button_left > 0),
    buy_button_right NUMERIC NOT NULL CONSTRAINT check_buy_button_right CHECK(buy_button_right > 0),
    sell_button_left SMALLINT NOT NULL CONSTRAINT check_sell_button_left CHECK(sell_button_left > 0 AND sell_button_left < 101),
    sell_button_right SMALLINT NOT NULL CONSTRAINT check_sell_button_right CHECK(sell_button_right > 0 AND sell_button_right < 101),
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL
);

-- +goose Down
DROP TABLE IF EXISTS settings;