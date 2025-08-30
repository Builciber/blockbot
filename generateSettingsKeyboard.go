package main

import (
	"fmt"

	"github.com/Builciber/blockbot/internal/database"
	"github.com/go-telegram/bot/models"
)

func generateSettingsKeyboard(userSettings *database.Setting) (models.InlineKeyboardMarkup, error) {
	buyButtonLeft, err := userSettings.BuyButtonLeft.Float64Value()
	if err != nil {
		return models.InlineKeyboardMarkup{}, err
	}
	buyButtonRight, err := userSettings.BuyButtonRight.Float64Value()
	if err != nil {
		return models.InlineKeyboardMarkup{}, err
	}
	/*autoBuyAmount, err := userSettings.AutoBuyAmount.Float64Value()
	if err != nil {
		return models.InlineKeyboardMarkup{}, err
	}
	autoBuy := "🔴 Disabled"
	if userSettings.AutoBuy {
		autoBuy = "🟢 Enabled"
	}*/

	return models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			/*{
				{Text: "--- Auto Buy ---", CallbackData: "settings_NoCallback"},
			},
			{
				{Text: autoBuy, CallbackData: "settings_AB"},
				{Text: fmt.Sprintf("✏️ %v MON", autoBuyAmount.Float64), CallbackData: "settings_ABConf"},
			},*/
			{
				{Text: "--- Buy Buttons Config ---", CallbackData: "settings_NoCallback"},
			},
			{
				{Text: fmt.Sprintf("✏️ Left: %v MON", buyButtonLeft.Float64), CallbackData: "settings_BBCLeft"},
				{Text: fmt.Sprintf("✏️ Right: %v MON", buyButtonRight.Float64), CallbackData: "settings_BBCRight"},
			},
			{
				{Text: "--- Sell Buttons Config ---", CallbackData: "settings_NoCallback"},
			},
			{
				{Text: fmt.Sprintf("✏️ Left: %v %%", userSettings.SellButtonLeft), CallbackData: "settings_SBCLeft"},
				{Text: fmt.Sprintf("✏️ Right: %v %%", userSettings.SellButtonRight), CallbackData: "settings_SBCRight"},
			},
			{
				{Text: "--- Slippage Config ---", CallbackData: "settings_NoCallback"},
			},
			{
				{Text: fmt.Sprintf("✏️ Buy: %v %%", userSettings.BuySlippage), CallbackData: "settings_SCBuy"},
				{Text: fmt.Sprintf("✏️ Sell: %v %%", userSettings.SellSlippage), CallbackData: "settings_SCSell"},
			},
			{
				{Text: fmt.Sprintf("✏️ Max Price Impact: %v %%", userSettings.MaxPriceImpact), CallbackData: "settings_MPI"},
			},
			{
				{Text: fmt.Sprintf("🔁 Transaction Priority: %s", userSettings.PriorityFee), CallbackData: "settings_PF"},
			},
			{
				{Text: "close ❌", CallbackData: "settings_close"},
			},
		},
	}, nil
}

var settingsMessage = `*Configure Your Settings*:

*BUY BUTTONS CONFIG*
Customize your buy buttons for the buy interface and portfolio interface\. Tap to edit\.

*SELL BUTTONS CONFIG*
Customize your sell buttons for the portfolio interface\. Tap to edit\.

*SLIPPAGE CONFIG*
Customize your slippage tolerance settings for buys and sells\. Tap to edit\.

*MAX PRICE IMPACT*
Max Price Impact is to protect against trades in pools with low liquidity\.

*TRANSACTION PRIORITY*
Configure your transaction speed, higher tx priority means higher fees paid to validators which in turn increases your transaction's speed\. Tap to toggle\.`
