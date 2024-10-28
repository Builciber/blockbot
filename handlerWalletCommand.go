package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

func (cfg *apiConfig) handlerWalletCommand(ctx context.Context, b *bot.Bot, update *models.Update) {
	telegramID := update.Message.From.ID
	walletAddress, err := cfg.DB.GetWalletAddress(ctx, telegramID)
	if err != nil {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "Something went wrong, please try again",
		})
		log.Println(err.Error())
		return
	}
	getBalanceResp := getBalanceRespBody{}
	err = WalletServiceCall("GET", "http://localhost:8080/v1/balance", cfg.bwsApiKey, ReqBody{TelegramID: telegramID}, &getBalanceResp)
	if err != nil {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "Something went wrong, please try again",
		})
		log.Println(err.Error())
		return
	}
	walletBalance := getBalanceResp.Balance
	keyboard := &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{
				{Text: "View on Etherscan 🔎", URL: "https://sepolia.etherscan.io/address/" + walletAddress},
				{Text: "Close ❌", CallbackData: "wallet_close"},
			}, {
				{Text: "Deposit ETH 🏦", CallbackData: "wallet_deposit"},
			}, {
				{Text: "Withdraw all ETH 💵💵", CallbackData: "wallet_WA"},
				{Text: "Withdraw X ETH 💵", CallbackData: "wallet_WX"},
			}, {
				{Text: "Recreate Wallet 💼", CallbackData: "wallet_recreate"},
				{Text: "Export Private Key 🗝️", CallbackData: "wallet_export"},
			}, {
				{Text: "Refresh ⟳", CallbackData: "wallet_refresh"},
			},
		},
	}
	_, err = b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      update.Message.Chat.ID,
		ParseMode:   models.ParseModeMarkdown,
		Text:        fmt.Sprintf("*Trading Wallet Information*:\n\nAddress:  `%s`\nBalance:  *%s* ETH\n\nTap the address to copy it and send ETH to deposit", walletAddress, strings.Replace(walletBalance, ".", "\\.", 1)),
		ReplyMarkup: keyboard,
	})
	if err != nil {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "Something went wrong, please try again",
		})
		log.Println(err.Error())
		return
	}
}
