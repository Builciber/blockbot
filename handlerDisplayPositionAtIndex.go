package main

import (
	"context"
	"log"
	"strconv"
	"strings"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

func (cfg *apiConfig) handlerDisplayPositionAtIndex(ctx context.Context, b *bot.Bot, update *models.Update) {
	b.DeleteMessage(ctx, &bot.DeleteMessageParams{
		ChatID:    update.Message.Chat.ID,
		MessageID: update.Message.ID,
	})
	msg := update.Message
	telegramId := msg.From.ID
	cfg.userBalancesMu.RLock()
	userBalances, ok := cfg.usersBalances[telegramID(telegramId)]
	cfg.userBalancesMu.RUnlock()
	if !ok {
		return
	}
	split := strings.Split(update.Message.Text, " ")
	str := split[1]
	indexString := strings.TrimPrefix(str, "position_")
	index, err := strconv.ParseInt(indexString, 10, 64)
	if err != nil {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: msg.Chat.ID,
			Text:   "Something went wrong, please try again",
		})
		log.Println(err.Error())
		return
	}
	token := userBalances.balances[index]
	monBalance := userBalances.monBalance
	inlineText, err := cfg.handlerShowBoughttokenPM(ctx, telegramId, token, monBalance)
	if err != nil {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: msg.Chat.ID,
			Text:   "Something went wrong, please try again",
		})
		log.Println(err.Error())
		return
	}
	buySellButtons, err := cfg.DB.GetBuySellButtons(ctx, telegramId)
	if err != nil {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: msg.Chat.ID,
			Text:   "Something went wrong, please try again",
		})
		log.Println(err.Error())
		return
	}
	keyboard := genQuickViewKeyboard(buySellButtons, token.Symbol)
	b.SendMessage(ctx, &bot.SendMessageParams{
		ParseMode:   models.ParseModeMarkdown,
		Text:        inlineText,
		ChatID:      msg.Chat.ID,
		ReplyMarkup: &keyboard,
	})
	cfg.userBalancesMu.RLock()
	ub := cfg.usersBalances[telegramID(telegramId)]
	ub.currBalanceIdx = int(index)
	cfg.userBalancesMu.RUnlock()
}
