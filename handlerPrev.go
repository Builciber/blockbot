package main

import (
	"context"
	"fmt"
	"log"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

func (cfg *apiConfig) handlerPrev(ctx context.Context, b *bot.Bot, update *models.Update) {
	telegramId := update.CallbackQuery.From.ID
	cfg.userBalancesMu.RLock()
	userBalances, ok := cfg.usersBalances[telegramID(telegramId)]
	cfg.userBalancesMu.RUnlock()
	if !ok {
		return
	}
	if userBalances.currBalanceIdx == 0 {
		return
	}
	token := userBalances.balances[userBalances.currBalanceIdx-1]
	if token.MarketCap == "" {
		marketData, err := cfg.getMarketData(token.ContractAddress)
		if err != nil {
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: update.CallbackQuery.Message.Message.Chat.ID,
				Text:   "Something went wrong, please try again",
			})
			log.Println(err.Error())
			return
		}
		token.MarketCap = marketData.MarketCap
		token.Liquidity = marketData.Liquidity
		token.Intervals = marketData.Intervals
		token.Tag = marketData.Tag
	}
	inlineText, err := cfg.handlerShowBoughttokenPM(ctx, telegramId, token, userBalances.monBalance)
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
				{Text: "Home 🏠︎", CallbackData: "positions_home"},
				{Text: "Close ❌", CallbackData: "positions_close"},
			}, {
				{Text: token.Symbol, CallbackData: "positions_symbol"},
			}, {
				{Text: "◀️ Prev", CallbackData: "positions_prev"},
				{Text: "Next ▶️", CallbackData: "positions_next"},
			}, {
				{Text: fmt.Sprintf("Buy %v MON", pgNumericToString(buySellButtons.BuyButtonLeft)), CallbackData: "positions_buyLeft"},
				{Text: fmt.Sprintf("Buy %v MON", pgNumericToString(buySellButtons.BuyButtonRight)), CallbackData: "positions_buyRight"},
				{Text: "Buy X MON", CallbackData: "positions_buyX"},
			}, {
				{Text: fmt.Sprintf("Sell %v%%", buySellButtons.SellButtonLeft), CallbackData: "positions_sellLeft"},
				{Text: fmt.Sprintf("Sell %v%%", buySellButtons.SellButtonRight), CallbackData: "positions_sellRight"},
				{Text: "Sell X %", CallbackData: "positions_sellX"},
			}, {
				{Text: "Refresh ⟳", CallbackData: "positions_refresh"},
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
	cfg.userBalancesMu.Lock()
	ub := cfg.usersBalances[telegramID(telegramId)]
	ub.balances[ub.currBalanceIdx-1] = token
	ub.currBalanceIdx -= 1
	cfg.userBalancesMu.Unlock()
}
