package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

func (cfg *apiConfig) handlerQuickViewRefresh(ctx context.Context, b *bot.Bot, update *models.Update) {
	telegramId := update.CallbackQuery.From.ID
	walletAddress, err := cfg.DB.GetWalletAddress(ctx, telegramId)
	if err != nil {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.CallbackQuery.Message.Message.Chat.ID,
			Text:   "Something went wrong, please try again",
		})
		log.Println(err.Error())
		return
	}
	msg := update.CallbackQuery.Message.Message
	splits := strings.Split(msg.Text, "|")
	withTokenAddress := strings.TrimPrefix(splits[2], " ")
	tokenAddress := withTokenAddress[0:42]
	tokenSymbol := splits[1]
	token, err := cfg.getToken(tokenAddress, walletAddress)
	if err != nil {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.CallbackQuery.Message.Message.Chat.ID,
			Text:   "Something went wrong, please try again",
		})
		log.Println(err.Error())
		return
	}
	getBalanceResp := getBalanceRespBody{}
	err = WalletServiceCall("GET", fmt.Sprintf("%s/v1/balance", cfg.bwsOrigin), cfg.bwsApiKey, ReqBody{TelegramID: telegramId}, &getBalanceResp)
	if err != nil {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.CallbackQuery.Message.Message.Chat.ID,
			Text:   "Something went wrong, please try again",
		})
		log.Println(err.Error())
		return
	}
	monBalance := getBalanceResp.Balance
	inlineText, err := cfg.handlerShowBoughttokenPM(ctx, telegramId, token, monBalance)
	if err != nil {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.CallbackQuery.Message.Message.Chat.ID,
			Text:   "Something went wrong, please try again",
		})
		log.Println(err.Error())
		return
	}
	buySellButtons, err := cfg.DB.GetBuySellButtons(ctx, telegramId)
	if err != nil {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.CallbackQuery.Message.Message.Chat.ID,
			Text:   "Something went wrong, please try again",
		})
		log.Println(err.Error())
		return
	}
	keyboard := &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{
				{Text: "Home 🏠︎", CallbackData: "quickView_home"},
				{Text: "Close ❌", CallbackData: "quickView_close"},
			}, {
				{Text: tokenSymbol, CallbackData: "quickView_symbol"},
			}, {
				{Text: fmt.Sprintf("Buy %v MON", pgNumericToString(buySellButtons.BuyButtonLeft)), CallbackData: "quickView_buyLeft"},
				{Text: fmt.Sprintf("Buy %v MON", pgNumericToString(buySellButtons.BuyButtonRight)), CallbackData: "quickView_buyRight"},
				{Text: "Buy X MON", CallbackData: "quickView_buyX"},
			}, {
				{Text: fmt.Sprintf("Sell %v%%", buySellButtons.SellButtonLeft), CallbackData: "quickView_sellLeft"},
				{Text: fmt.Sprintf("Sell %v%%", buySellButtons.SellButtonRight), CallbackData: "quickView_sellRight"},
				{Text: "Sell X %", CallbackData: "quickView_sellX"},
			}, {
				{Text: "Refresh ⟳", CallbackData: "quickView_refresh"},
			},
		},
	}
	b.EditMessageText(ctx, &bot.EditMessageTextParams{
		MessageID:   update.CallbackQuery.Message.Message.ID,
		ParseMode:   models.ParseModeMarkdown,
		Text:        inlineText,
		ChatID:      update.CallbackQuery.Message.Message.Chat.ID,
		ReplyMarkup: keyboard,
	})
}
