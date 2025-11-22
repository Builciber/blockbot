package main

import (
	"context"
	"log"
	"strings"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

func (cfg *apiConfig) handlerPositionRefresh(ctx context.Context, b *bot.Bot, update *models.Update) {
	telegramId := update.CallbackQuery.From.ID
	cfg.userBalancesMu.RLock()
	userBalances, ok := cfg.usersBalances[telegramID(telegramId)]
	cfg.userBalancesMu.RUnlock()
	if !ok {
		return
	}
	startIndex := (userBalances.steps - 1) * 10
	endIndex := userBalances.steps * 10
	tokens := userBalances.balances[startIndex:endIndex]
	err := cfg.fetchOverviewData(tokens)
	if err != nil {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.CallbackQuery.Message.Message.Chat.ID,
			Text:   "Something went wrong, please try again",
		})
		log.Println(err.Error())
		return
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
			}, {
				{Text: "Refresh ⟳", CallbackData: "positions_refresh"},
			},
		},
	}
	if len(tokens) <= 10 {
		keyboard = &models.InlineKeyboardMarkup{
			InlineKeyboard: [][]models.InlineKeyboardButton{
				{
					{Text: "Home 🏠︎", CallbackData: "positions_home"},
					{Text: "Close ❌", CallbackData: "positions_close"},
				}, {
					{Text: "Refresh ⟳", CallbackData: "positions_refresh"},
				},
			},
		}
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
	cfg.userBalancesMu.Unlock()
}

func readMonBalanceFromString(str string) string {
	substring := "Wallet Balance: *"
	index := strings.Index(str, substring)
	startIndex := index + len(substring)
	endIndex := startIndex
	for str[endIndex] != ' ' {
		endIndex++
	}
	monBalance := str[startIndex:endIndex]
	monBalance = strings.Replace(monBalance, "\\.", ".", 1)
	/*substring = "Total Portfolio Value: *$"
	index = strings.Index(str, substring)
	startIndex = index + len(substring)
	endIndex = startIndex
	for str[endIndex] != '*' {
		endIndex++
	}
	portfolioValue := str[startIndex:endIndex]*/
	return monBalance
}
