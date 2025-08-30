package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

func (cfg *apiConfig) handlerRefresh(ctx context.Context, b *bot.Bot, update *models.Update) {
	telegramID := update.CallbackQuery.From.ID
	walletAddress, err := cfg.DB.GetWalletAddress(ctx, telegramID)
	if err != nil {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.CallbackQuery.Message.Message.Chat.ID,
			Text:   "Something went wrong, please try again",
		})
		log.Println(err.Error())
		return
	}
	getBalanceResp := getBalanceRespBody{}
	err = WalletServiceCall("GET", fmt.Sprintf("%s/v1/balance", cfg.bwsOrigin), cfg.bwsApiKey, ReqBody{TelegramID: telegramID}, &getBalanceResp)
	if err != nil {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.CallbackQuery.Message.Message.Chat.ID,
			Text:   "Something went wrong whilst trying to refresh, please try again",
		})
		log.Println(err.Error())
		return
	}
	walletBalance := getBalanceResp.Balance
	keyboard := &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{
				{Text: "View on Monad Explorer 🔎", URL: "https://testnet.monadexplorer.com/address/" + walletAddress},
				{Text: "Close ❌", CallbackData: "wallet_close"},
			}, {
				{Text: "Deposit MON 🏦", CallbackData: "wallet_deposit"},
			}, {
				{Text: "Withdraw all MON 💵💵", CallbackData: "wallet_WA"},
				{Text: "Withdraw X MON 💵", CallbackData: "wallet_WX"},
			}, {
				{Text: "Recreate Wallet 💼", CallbackData: "wallet_recreate"},
				{Text: "Export Private Key 🗝️", CallbackData: "wallet_export"},
			}, {
				{Text: "Refresh ⟳", CallbackData: "wallet_refresh"},
			},
		},
	}
	b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:      update.CallbackQuery.Message.Message.Chat.ID,
		MessageID:   update.CallbackQuery.Message.Message.ID,
		ParseMode:   models.ParseModeMarkdown,
		Text:        fmt.Sprintf("*Trading Wallet Information*:\n\nAddress:  `%s`\nBalance:  *%s* MON\n\nTap the address to copy it and send MON to deposit", walletAddress, strings.Replace(walletBalance, ".", "\\.", 1)),
		ReplyMarkup: keyboard,
	})
}
