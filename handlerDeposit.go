package main

import (
	"context"
	"fmt"
	"log"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

func (cfg *apiConfig) handlerDeposit(ctx context.Context, b *bot.Bot, update *models.Update) {
	walletAddress, err := cfg.DB.GetWalletAddress(ctx, update.CallbackQuery.From.ID)
	if err != nil {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.CallbackQuery.Message.Message.Chat.ID,
			Text:   "Something went wrong, please try again",
		})
		log.Println(err.Error())
		return
	}
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    update.CallbackQuery.Message.Message.Chat.ID,
		ParseMode: models.ParseModeMarkdown,
		Text:      fmt.Sprintf("*Deposit*:\n\nTo deposit, send ETH to your trading wallet address below:\n\n`%s`", walletAddress),
	})
}
