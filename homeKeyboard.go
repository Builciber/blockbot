package main

import (
	"fmt"
	"strings"

	"github.com/go-telegram/bot/models"
)

var homeKeyboard = &models.InlineKeyboardMarkup{
	InlineKeyboard: [][]models.InlineKeyboardButton{
		{
			{Text: "Buy 💳", CallbackData: "home_buy"},
			{Text: "Portfolio 📊", CallbackData: "home_portfolio"},
		},
		{
			{Text: "Wallet 💼", CallbackData: "home_wallet"},
			{Text: "Referrals 👥", CallbackData: "home_referral"},
		}, {
			{Text: "Pin 📌", CallbackData: "home_pin"},
			{Text: "Settings 🔧", CallbackData: "home_settings"},
		}, {
			{Text: "Refresh ⟳", CallbackData: "home_refresh"},
		},
	},
}

func generateHomeMessage(balance string) string {
	return fmt.Sprintf("You currently have a balance of *%s* MON\\.\n\nTo view and manage your open positions, tap the \"Portfolio 📊\" buttton\\.\n\nTo buy a token, enter a ticker or token address in chat\\.\n\nFor more information on your wallet and to export your seed phrase, tap the \"Wallet 💼\" button below\\.", strings.Replace(displayDecimal(balance, 3), ".", "\\.", 1))
}
