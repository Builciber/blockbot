package main

import (
	"context"
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
	startIndex := (userBalances.steps - 1) * 10
	endIndex := userBalances.steps * 10
	/*if startIndex < 0 {
		return
	}*/
	if endIndex > len(userBalances.balances) {
		endIndex = len(userBalances.balances)
	}
	tokens := userBalances.balances[startIndex:endIndex]
	if tokens[0].MarketCap == "" {
		err := cfg.fetchOverviewData(tokens)
		if err != nil {
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: update.CallbackQuery.Message.Message.Chat.ID,
				Text:   "Something went wrong, please try again",
			})
			log.Println(err.Error())
			return
		}
	}
	inlineText, err := cfg.constructOverviewString(ctx, tokens, telegramId, startIndex, userBalances.totalPortFolioValue, userBalances.monBalance, len(userBalances.balances))
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
				{Text: "◀️ Prev", CallbackData: "positions_prev"},
				{Text: "Next ▶️", CallbackData: "positions_next"},
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
	idx := 0
	for i := startIndex; i < endIndex; i++ {
		ub.balances[i] = tokens[idx]
		idx++
	}
	if startIndex > 0 {
		ub.steps--
	}
	cfg.userBalancesMu.Unlock()
}
