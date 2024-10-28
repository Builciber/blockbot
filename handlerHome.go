package main

import (
	"context"
	"fmt"
	"log"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

func (cfg *apiConfig) handlerHome(ctx context.Context, b *bot.Bot, update *models.Update) {
	walletAddress, err := cfg.DB.GetWalletAddress(ctx, update.Message.From.ID)
	if err != nil {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "Something went wrong, please try again",
		})
		log.Println(err.Error())
		return
	}
	keyboard := &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{
				{Text: "Wallet 💼", CallbackData: "home_wallet"},
			}, {
				{Text: "Referral 👥", CallbackData: "home_referral"},
			}, {
				{Text: "Pin 📌", CallbackData: "home_pin"},
			},
		},
	}
	_, err = b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      update.Message.Chat.ID,
		ParseMode:   models.ParseModeMarkdown,
		Text:        fmt.Sprintf("*Welcome to BlockBot*\nMonad's most powerful telegram trading bot\\. Built with speed, security and YOU in mind\n\nBelow is your newly generated trading wallet's address\\. To start trading, tap the address to copy it then send ETH to it:\n\n`%s`", walletAddress),
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

func handlerPinButton(ctx context.Context, b *bot.Bot, update *models.Update) {
	_, err := b.PinChatMessage(ctx, &bot.PinChatMessageParams{
		ChatID:    update.CallbackQuery.Message.Message.Chat.ID,
		MessageID: update.CallbackQuery.Message.Message.ID,
	})
	if err != nil {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.CallbackQuery.Message.Message.Chat.ID,
			Text:   "Failed to pin message",
		})
		log.Println(err.Error())
		return
	}
}

func (cfg *apiConfig) handlerHomeCallback(ctx context.Context, b *bot.Bot, update *models.Update) {
	b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
		CallbackQueryID: update.CallbackQuery.ID,
		ShowAlert:       false,
	})
	switch update.CallbackQuery.Data {
	case "home_wallet":
		cfg.handlerWalletButton(ctx, b, update)
	case "home_referral":
		cfg.handlerReferralButton(ctx, b, update)
	case "home_pin":
		handlerPinButton(ctx, b, update)
	}
}
