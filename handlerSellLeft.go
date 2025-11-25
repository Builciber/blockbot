package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

func (cfg *apiConfig) handlerSellLeft(ctx context.Context, b *bot.Bot, update *models.Update) {
	telegramId := update.CallbackQuery.From.ID
	cfg.userBalancesMu.RLock()
	userBalances, ok := cfg.usersBalances[telegramID(telegramId)]
	cfg.userBalancesMu.RUnlock()
	if !ok {
		return
	}
	processingMsg, _ := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.CallbackQuery.Message.Message.Chat.ID,
		Text:   "Processing request...",
	})
	buySellButtons, err := cfg.DB.GetBuySellButtons(ctx, telegramId)
	if err != nil {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.CallbackQuery.Message.Message.Chat.ID,
			Text:   "Something went wrong, please try again",
		})
		log.Println(err.Error())
		return
	}
	token := userBalances.balances[userBalances.currBalanceIdx]
	tokenDecimals := token.Decimals
	sellPercent := buySellButtons.SellButtonLeft
	executingMsg, _ := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.CallbackQuery.Message.Message.Chat.ID,
		Text:   "Executing sale...",
	})
	saleResult, err := cfg.handlerSell(ctx, telegramId, int(sellPercent), token.ContractAddress, uint8(tokenDecimals))
	if err != nil {
		errorMessage, found := strings.CutPrefix(err.Error(), "display to user: ")
		if found {
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: update.CallbackQuery.Message.Message.Chat.ID,
				Text:   errorMessage,
			})
			return
		}
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.CallbackQuery.Message.Message.Chat.ID,
			Text:   "Something went wrong, please try again",
		})
		log.Println(err.Error())
		return
	}
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    update.CallbackQuery.Message.Message.Chat.ID,
		ParseMode: models.ParseModeMarkdown,
		Text:      fmt.Sprintf("Sale successful: Sold *%v %s* for *%v MON*\n[View on the explorer](https://monadvision.com/tx/%s)", strings.Replace(displayDecimal(saleResult.SoldAmount, 3), ".", "\\.", 1), escapeMarkdown(token.Symbol), strings.Replace(displayDecimal(saleResult.ReceivedMon, 3), ".", "\\.", 1), saleResult.TxHash),
	})
	b.DeleteMessages(ctx, &bot.DeleteMessagesParams{
		ChatID:     update.CallbackQuery.Message.Message.Chat.ID,
		MessageIDs: []int{processingMsg.ID, executingMsg.ID},
	})
}
