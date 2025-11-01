package main

import (
	"context"
	"log"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

func (cfg *apiConfig) handlerSettingsButton(ctx context.Context, b *bot.Bot, update *models.Update) {
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
	settingsKeyboard, err := generateSettingsKeyboard(&settings)
	if err != nil {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.CallbackQuery.Message.Message.Chat.ID,
			Text:   "Something went wrong, please try again",
		})
		log.Println(err.Error())
		return
	}
	_, err = b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      update.CallbackQuery.Message.Message.Chat.ID,
		ParseMode:   models.ParseModeMarkdown,
		Text:        settingsMessage,
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
}

func (cfg *apiConfig) settingsViewCallback(ctx context.Context, b *bot.Bot, update *models.Update) {
	b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
		CallbackQueryID: update.CallbackQuery.ID,
		ShowAlert:       false,
	})

	switch update.CallbackQuery.Data {
	case "settings_AB":
		cfg.handlerToggleAutoBuy(ctx, b, update)
	case "settings_ABConf":
		cfg.handlerSetAutoBuyAmount(ctx, b, update)
	case "settings_BBCLeft":
		cfg.handlerSetBuyButtonLeft(ctx, b, update)
	case "settings_BBCRight":
		cfg.handlerSetBuyButtonRight(ctx, b, update)
	case "settings_SBCLeft":
		cfg.handlerSetSellButtonLeft(ctx, b, update)
	case "settings_SBCRight":
		cfg.handlerSetSellButtonRight(ctx, b, update)
	case "settings_SCBuy":
		cfg.handlerSetBuySlippage(ctx, b, update)
	case "settings_SCSell":
		cfg.handlerSetSellSlippage(ctx, b, update)
	case "settings_MPI":
		cfg.handlerSetMaxPriceImpact(ctx, b, update)
	case "settings_PF":
		cfg.handlerPriorityFeeToggle(ctx, b, update)
	case "settings_close":
		cfg.handlerCloseButton(ctx, b, update)
	}
}
