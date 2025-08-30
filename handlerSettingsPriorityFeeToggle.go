package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/Builciber/blockbot/internal/database"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/jackc/pgx/v5/pgtype"
)

func (cfg *apiConfig) handlerPriorityFeeToggle(ctx context.Context, b *bot.Bot, update *models.Update) {
	telegramID := update.CallbackQuery.From.ID
	settings, err := cfg.DB.GetUserSettings(ctx, telegramID)
	if err != nil {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.CallbackQuery.Message.Message.Chat.ID,
			Text:   "Something went wrong, please try again",
		})
		log.Println(err.Error())
		return
	}
	var newPriorityFee string
	switch settings.PriorityFee {
	case "normal":
		newPriorityFee = "fast"
	case "fast":
		newPriorityFee = "very fast"
	case "very fast":
		newPriorityFee = "turbo"
	case "turbo":
		newPriorityFee = "normal"
	}
	settings.PriorityFee = newPriorityFee
	settingsKeyboard, err := generateSettingsKeyboard(&settings)
	if err != nil {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.CallbackQuery.Message.Message.Chat.ID,
			Text:   "Something went wrong, please try again",
		})
		log.Println(err.Error())
		return
	}
	err = cfg.DB.UpdatePriorityFee(ctx, database.UpdatePriorityFeeParams{
		TelegramID:  telegramID,
		PriorityFee: newPriorityFee,
		UpdatedAt:   pgtype.Timestamp{Time: time.Now(), Valid: true},
	})
	if err != nil {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.CallbackQuery.Message.Message.Chat.ID,
			Text:   "Something went wrong, please try again",
		})
		log.Println(err.Error())
		return
	}
	_, err = b.EditMessageReplyMarkup(ctx, &bot.EditMessageReplyMarkupParams{
		MessageID:   update.CallbackQuery.Message.Message.ID,
		ChatID:      update.CallbackQuery.Message.Message.Chat.ID,
		ReplyMarkup: settingsKeyboard,
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
		ChatID: update.CallbackQuery.Message.Message.Chat.ID,
		Text:   fmt.Sprintf("Priority fee set to '%s'.", newPriorityFee),
	})
}
