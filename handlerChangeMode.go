package main

import (
	"context"
	"fmt"
	"log"

	"github.com/Builciber/blockbot/internal/database"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

func (cfg *apiConfig) handlerChangeMode(ctx context.Context, b *bot.Bot, update *models.Update) {
	isUser, err := cfg.DB.IsExistingUser(ctx, update.Message.From.ID)
	if err != nil {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "Something went wrong, please try again",
		})
		log.Println(err.Error())
		return
	}
	if !isUser {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "Forbidden action",
		})
		return
	}
	welcomeMessage := fmt.Sprintf("Choose your mode %s:\n\n🛡 *Standard Mode* – For traders who like things smooth and steady\\.\n\n⚡ *Degen Mode* – For maniacs chasing milliseconds execution and meme magic\\.\n\n\\(You can always tweak this settings later… I won’t judge\\. 😎\\)\\.\n\nTap to choose and let’s get cooking\\.👇", bot.EscapeMarkdown(update.Message.From.Username))
	keyboard := &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{
				{Text: "🛡 Standard Mode", CallbackData: "change_mode_standard"},
				{Text: "⚡ Degen Mode", CallbackData: "change_mode_degen"},
			},
		},
	}
	_, err = b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      update.Message.Chat.ID,
		ParseMode:   models.ParseModeMarkdown,
		Text:        welcomeMessage,
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
	cfg.sendBadgeMessage(ctx, b, update)
}

func (cfg *apiConfig) changeModeViewCallback(ctx context.Context, b *bot.Bot, update *models.Update) {
	b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
		CallbackQueryID: update.CallbackQuery.ID,
		ShowAlert:       false,
	})
	switch update.CallbackQuery.Data {
	case "change_mode_standard":
		cfg.handlerChangeBotModeStandard(ctx, b, update)
	case "change_mode_degen":
		cfg.handlerChangeBotModeDegen(ctx, b, update)
	}
	cfg.sendBadgeMessage(ctx, b, update)
}

func (cfg *apiConfig) handlerChangeBotModeDegen(ctx context.Context, b *bot.Bot, update *models.Update) {
	telegramID := update.CallbackQuery.From.ID
	updateUserTradeSettingsParams := database.UpdateUserTradeSettingsParams{
		TelegramID:     telegramID,
		BuySlippage:    15,
		SellSlippage:   15,
		MaxPriceImpact: 25,
		PriorityFee:    "turbo",
	}
	err := cfg.DB.UpdateUserTradeSettings(ctx, updateUserTradeSettingsParams)
	if err != nil {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.CallbackQuery.Message.Message.Chat.ID,
			Text:   "Something went wrong, please try again",
		})
		log.Println(err.Error())
		return
	}
	responseText := "⚡ Degen Mode locked in\\.\nI see you like things super fast\\. Respect\\. 🫡\n\nWe’ve tuned your engine for max performance:\n– Slippage: 15%\n– Maximimum Acceptable Price Impact: 25%\n– Transaction Priority: Turbo 🚀\n\nWant to tweak any of these later?\nYou can always tweak any of these individually in Settings\\.\nYou’re now equipped to trade like a savage\\."
	_, err = b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      update.CallbackQuery.Message.Message.Chat.ID,
		ParseMode:   models.ParseModeMarkdown,
		Text:        responseText,
		ReplyMarkup: homeKeyboard,
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

func (cfg *apiConfig) handlerChangeBotModeStandard(ctx context.Context, b *bot.Bot, update *models.Update) {
	telegramID := update.CallbackQuery.From.ID
	updateUserTradeSettingsParams := database.UpdateUserTradeSettingsParams{
		TelegramID:     telegramID,
		BuySlippage:    1,
		SellSlippage:   1,
		MaxPriceImpact: 5,
		PriorityFee:    "normal",
	}
	err := cfg.DB.UpdateUserTradeSettings(ctx, updateUserTradeSettingsParams)
	if err != nil {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.CallbackQuery.Message.Message.Chat.ID,
			Text:   "Something went wrong, please try again",
		})
		log.Println(err.Error())
		return
	}
	responseText := "🛡 Standard Mode locked in\\.\nPlaying it sharp and steady\\. Respect that\\. 🎯\n\nWe’ve tuned your engine for max performance:\n– Slippage: 1%\n– Maximum Acceptable Price Impact: 5%\n– Transaction Priority: Normal ⏱\n\nWant to adjust these individually later?\nYou’ve got full control anytime in Settings\\.\nYou’re now equipped to trade with precision\\."
	_, err = b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      update.CallbackQuery.Message.Message.Chat.ID,
		ParseMode:   models.ParseModeMarkdown,
		Text:        responseText,
		ReplyMarkup: homeKeyboard,
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
