package main

import (
	"context"
	"fmt"
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
	walletAddress, err := cfg.DB.GetWalletAddress(ctx, telegramId)
	if err != nil {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.CallbackQuery.Message.Message.Chat.ID,
			Text:   "Something went wrong, please try again",
		})
		log.Println(err.Error())
		return
	}
	oldToken := userBalances.balances[userBalances.currBalanceIdx]
	token, err := cfg.getToken(oldToken.ContractAddress, walletAddress)
	if err != nil {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.CallbackQuery.Message.Message.Chat.ID,
			Text:   "Something went wrong, please try again",
		})
		log.Println(err.Error())
		return
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
	userBalances.balances[userBalances.currBalanceIdx] = token
	monBalanceFormatted := readMonBalanceFromString(inlineText)
	userBalances.monBalance = monBalanceFormatted
	cfg.userBalancesMu.Lock()
	cfg.usersBalances[telegramID(telegramId)] = userBalances
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
