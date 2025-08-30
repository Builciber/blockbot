package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

type ExportWalletRespBody struct {
	TelegramID int64  `json:"telegram_id"`
	PrivateKey string `json:"private_key"`
}

func (cfg *apiConfig) handlerExportPrivateKeyProceed(ctx context.Context, b *bot.Bot, update *models.Update) {
	telegramID := update.CallbackQuery.From.ID
	exportWalletResp := ExportWalletRespBody{}
	err := WalletServiceCall("PUT", fmt.Sprintf("%s/v1/export", cfg.bwsOrigin), cfg.bwsApiKey, ReqBody{TelegramID: telegramID}, &exportWalletResp)
	if err != nil {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.CallbackQuery.Message.Message.Chat.ID,
			Text:   "Something went wrong, please try again",
		})
		log.Println(err.Error())
		return
	}
	mes, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    update.CallbackQuery.Message.Message.Chat.ID,
		ParseMode: models.ParseModeMarkdown,
		Text:      fmt.Sprintf("Your *Private Key* is:\n\n`%s` \\(Tap to copy\\)\n\nYou can now import the key into a wallet like Metamask\\. This message should auto\\-delete in one minute\\. If not delete this message when you're done\\.", exportWalletResp.PrivateKey),
	})
	if err != nil {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.CallbackQuery.Message.Message.Chat.ID,
			Text:   "Something went wrong, please try again",
		})
		log.Println(err.Error())
		return
	}
	b.DeleteMessage(ctx, &bot.DeleteMessageParams{
		ChatID:    update.CallbackQuery.Message.Message.Chat.ID,
		MessageID: update.CallbackQuery.Message.Message.ID,
	})
	time.Sleep(1 * time.Minute)
	b.DeleteMessage(ctx, &bot.DeleteMessageParams{
		ChatID:    update.CallbackQuery.Message.Message.Chat.ID,
		MessageID: mes.ID,
	})
}

func (cfg *apiConfig) handlerExportWallet(ctx context.Context, b *bot.Bot, update *models.Update) {
	keyboard := &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{
				{Text: "Proceed ✅", CallbackData: "wallet_export_proceed"},
				{Text: "Cancel ❌", CallbackData: "wallet_export_cancel"},
			},
		},
	}
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      update.CallbackQuery.Message.Message.Chat.ID,
		ParseMode:   models.ParseModeMarkdown,
		Text:        "*Your private key is about to be revealed*\\.\n\nDo NOT under any circumstances share your private key with anyone, NOT EVEN THE BLOCKBOT TEAM\\. Giving anyone your private keys give them *FULL CONTROL* over your funds\\.\n\nThe Blockbot team will *NEVER* ask for your private keys",
		ReplyMarkup: keyboard,
	})
}
