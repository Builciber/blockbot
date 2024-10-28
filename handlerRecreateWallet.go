package main

import (
	"context"
	"fmt"
	"internal/database"
	"log"
	"time"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

type RecreateWalletRespBody struct {
	TelegramID    int64  `json:"telegram_id"`
	WalletAddress string `json:"wallet_address"`
}

func (cfg *apiConfig) handlerRecreateWalletProceed(ctx context.Context, b *bot.Bot, update *models.Update) {
	telegramID := update.CallbackQuery.From.ID
	exportWalletResp := ExportWalletRespBody{}
	err := WalletServiceCall("PUT", "http://localhost:8080/v1/export", cfg.bwsApiKey, ReqBody{TelegramID: telegramID}, &exportWalletResp)
	if err != nil {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.CallbackQuery.Message.Message.Chat.ID,
			Text:   "Something went wrong, please try again",
		})
		log.Println(err.Error())
		return
	}
	recreateCallResp := RecreateWalletRespBody{}
	err = WalletServiceCall("PUT", "http://localhost:8080/v1/recreate", cfg.bwsApiKey, ReqBody{TelegramID: telegramID}, &recreateCallResp)
	if err != nil {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.CallbackQuery.Message.Message.Chat.ID,
			Text:   "Something went wrong, please try again",
		})
		log.Println(err.Error())
		return
	}
	err = cfg.DB.UpdateWallet(ctx, database.UpdateWalletParams{
		TelegramID:    telegramID,
		WalletAddress: recreateCallResp.WalletAddress,
		UpdatedAt:     time.Now(),
	})
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
		Text:      fmt.Sprintf("*Wallet recreation successful:*\n\nYour new wallet's address is:\n`%s`\n\nYour *previous* wallet's private key is:\n`%s`\n\nIf you want to view your new wallet's private key use the 'Export Private Key' feature", recreateCallResp.WalletAddress, exportWalletResp.PrivateKey),
	})
	b.DeleteMessage(ctx, &bot.DeleteMessageParams{
		ChatID:    update.CallbackQuery.Message.Message.Chat.ID,
		MessageID: update.CallbackQuery.Message.Message.ID,
	})
}

func (cfg *apiConfig) handlerRecreateWallet(ctx context.Context, b *bot.Bot, update *models.Update) {
	keyboard := &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{
				{Text: "Proceed ✅", CallbackData: "wallet_recreate_proceed"},
				{Text: "Cancel ❌", CallbackData: "wallet_recreate_cancel"},
			},
		},
	}
	_, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      update.CallbackQuery.Message.Message.Chat.ID,
		ParseMode:   models.ParseModeMarkdown,
		Text:        "*Recreate wallet:*\n\n*You're about to recreate your trading wallet*\\.\n\nThis action removes your current wallet from our systems, replacing it with a new one\\.\n\nIf you choose to proceed, the private key of your *current wallet* will be displayed to you along with the wallet address of your *newly* generated wallet\\.",
		ReplyMarkup: keyboard,
	})
	if err != nil {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.CallbackQuery.Message.Message.Chat.ID,
			Text:   "Something went wrong, please try again",
		})
		log.Println(err.Error())
		return
	}
}
