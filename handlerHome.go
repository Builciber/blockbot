package main

import (
	"context"
	"fmt"
	"log"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

func (cfg *apiConfig) handlerHome(ctx context.Context, b *bot.Bot, update *models.Update) {
	getBalanceResp := getBalanceRespBody{}
	err := WalletServiceCall("GET", fmt.Sprintf("%s/v1/balance", cfg.bwsOrigin), cfg.bwsApiKey, ReqBody{TelegramID: update.Message.From.ID}, &getBalanceResp)
	if err != nil && err.Error() == "User does not exist" {
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
	walletBalance := getBalanceResp.Balance
	_, err = b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      update.Message.Chat.ID,
		ParseMode:   models.ParseModeMarkdown,
		Text:        generateHomeMessage(walletBalance),
		ReplyMarkup: homeKeyboard,
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
	case "home_settings":
		cfg.handlerSettingsButton(ctx, b, update)
	case "home_refresh":
		cfg.handlerHomeRefresh(ctx, b, update)
	case "home_portfolio":
		cfg.handlerManagePositions(ctx, b, update)
	case "home_buy":
		cfg.handlerHomeBuy(ctx, b, update)
	case "home_buy_close":
		cfg.handlerCloseButton(ctx, b, update)
	}
	cfg.sendBadgeMessage(ctx, b, update)
}

func (cfg *apiConfig) handlerHomeBuy(ctx context.Context, b *bot.Bot, update *models.Update) {
	keyboard := &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{
				{Text: "Close", CallbackData: "home_buy_close"},
			},
		},
	}
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      update.CallbackQuery.Message.Message.Chat.ID,
		ParseMode:   models.ParseModeMarkdown,
		Text:        "To buy a token, send it's name, symbol or token address in chat",
		ReplyMarkup: keyboard,
	})
}

func (cfg *apiConfig) handlerHomeButton(ctx context.Context, b *bot.Bot, update *models.Update) {
	getBalanceResp := getBalanceRespBody{}
	err := WalletServiceCall("GET", fmt.Sprintf("%s/v1/balance", cfg.bwsOrigin), cfg.bwsApiKey, ReqBody{TelegramID: update.CallbackQuery.From.ID}, &getBalanceResp)
	if err != nil {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.CallbackQuery.Message.Message.Chat.ID,
			Text:   "Something went wrong, please try again",
		})
		log.Println(err.Error())
		return
	}
	walletBalance := getBalanceResp.Balance
	_, err = b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      update.CallbackQuery.Message.Message.Chat.ID,
		ParseMode:   models.ParseModeMarkdown,
		Text:        generateHomeMessage(walletBalance),
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
