package main

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

func (cfg *apiConfig) handlerSellX(ctx context.Context, b *bot.Bot, update *models.Update) {
	telegramId := update.CallbackQuery.From.ID
	cfg.userBalancesMu.RLock()
	_, ok := cfg.usersBalances[telegramID(telegramId)]
	cfg.userBalancesMu.RUnlock()
	if !ok {
		return
	}
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.CallbackQuery.Message.Message.Chat.ID,
		Text:   "Enter the percentage you want to sell",
		ReplyMarkup: models.ForceReply{
			ForceReply:            true,
			InputFieldPlaceholder: "10",
			Selective:             false,
		},
	})
	cfg.mu.Lock()
	cfg.intSeqMap[chatID(update.CallbackQuery.Message.Message.Chat.ID)] = &interactionSequence{
		funcSlice:   []interactionHandler{cfg.handlerProcessSellX},
		retValues:   nil,
		createdAt:   time.Now(),
		nextFuncIdx: 0,
	}
	cfg.mu.Unlock()
}

func (cfg *apiConfig) handlerProcessSellX(ctx context.Context, b *bot.Bot, msg *models.Message) {
	defer cfg.endInteraction(msg)
	telegramId := msg.From.ID
	cfg.userBalancesMu.RLock()
	userBalances, ok := cfg.usersBalances[telegramID(telegramId)]
	cfg.userBalancesMu.RUnlock()
	if !ok {
		return
	}
	valid, _ := regexp.MatchString(`^[a-zA-Z]*$`, msg.Text)
	if ok, _ := regexp.MatchString(`^[0-9]*$`, msg.Text); valid || !ok {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: msg.Chat.ID,
			Text:   "Invalid percentage. Integer percentage expected",
			ReplyParameters: &models.ReplyParameters{
				MessageID:                msg.ID,
				AllowSendingWithoutReply: true,
			},
		})
		return
	}
	sellPercent, err := strconv.ParseInt(msg.Text, 10, 64)
	if err != nil {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: msg.Chat.ID,
			Text:   "Something went wrong, please try again",
		})
		log.Println(err.Error())
		return
	}
	if sellPercent < 1 || sellPercent > 100 {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: msg.Chat.ID,
			Text:   "Invalid percentage. Valid percentage range is 1 to 100 inclusive.",
			ReplyParameters: &models.ReplyParameters{
				MessageID:                msg.ID,
				AllowSendingWithoutReply: true,
			},
		})
		return
	}
	processingMsg, _ := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: msg.Chat.ID,
		Text:   "Processing request...",
	})
	token := userBalances.balances[userBalances.currBalanceIdx]
	tokenDecimals := token.Decimals
	executingMsg, _ := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: msg.Chat.ID,
		Text:   "Executing sale...",
	})
	saleResult, err := cfg.handlerSell(ctx, telegramId, int(sellPercent), token.ContractAddress, uint8(tokenDecimals))
	if err != nil {
		errorMessage, found := strings.CutPrefix(err.Error(), "display to user: ")
		if found {
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: msg.Chat.ID,
				Text:   errorMessage,
			})
			return
		}
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: msg.Chat.ID,
			Text:   "Something went wrong, please try again",
		})
		log.Println(err.Error())
		return
	}
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    msg.Chat.ID,
		ParseMode: models.ParseModeMarkdown,
		Text:      fmt.Sprintf("Sale successful: Sold *%v %s* for *%v MON*\n[View on the explorer](https://testnet.monadexplorer.com/tx/%s)", strings.Replace(displayDecimal(saleResult.SoldAmount, 3), ".", "\\.", 1), token.Symbol, strings.Replace(displayDecimal(saleResult.ReceivedMon, 3), ".", "\\.", 1), saleResult.TxHash),
	})
	b.DeleteMessages(ctx, &bot.DeleteMessagesParams{
		ChatID:     msg.Chat.ID,
		MessageIDs: []int{processingMsg.ID, executingMsg.ID},
	})
}
