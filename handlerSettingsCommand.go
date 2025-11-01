package main

import (
	"context"
	"log"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/jackc/pgx/v5"
)

func (cfg *apiConfig) handlerSettingsCommand(ctx context.Context, b *bot.Bot, update *models.Update) {
	telegramID := update.Message.From.ID
	settings, err := cfg.DB.GetUserSettings(ctx, telegramID)
	if err == pgx.ErrNoRows {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "Forbidden action",
		})
		log.Println(err.Error())
		return
	}
	if err != nil {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "Something went wrong, please try again",
		})
		log.Println(err.Error())
		return
	}
	settingsKeyboard, err := generateSettingsKeyboard(&settings)
	if err != nil {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "Something went wrong, please try again",
		})
		log.Println(err.Error())
		return
	}
	_, err = b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      update.Message.Chat.ID,
		ParseMode:   models.ParseModeMarkdown,
		Text:        settingsMessage,
		ReplyMarkup: settingsKeyboard,
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
