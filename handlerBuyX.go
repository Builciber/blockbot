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

func (cfg *apiConfig) handlerBuyX(ctx context.Context, b *bot.Bot, update *models.Update) {
	telegramId := update.CallbackQuery.From.ID
	cfg.userBalancesMu.RLock()
	_, ok := cfg.usersBalances[telegramID(telegramId)]
	cfg.userBalancesMu.RUnlock()
	if !ok {
		return
	}
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.CallbackQuery.Message.Message.Chat.ID,
		Text:   "Enter how much MON you want to spend",
		ReplyMarkup: models.ForceReply{
			ForceReply:            true,
			InputFieldPlaceholder: "6.9420",
			Selective:             false,
		},
	})
	cfg.mu.Lock()
	cfg.intSeqMap[chatID(update.CallbackQuery.Message.Message.Chat.ID)] = &interactionSequence{
		funcSlice:   []interactionHandler{cfg.handlerProcessBuyX},
		retValues:   nil,
		createdAt:   time.Now(),
		nextFuncIdx: 0,
	}
	cfg.mu.Unlock()
}

func (cfg *apiConfig) handlerProcessBuyX(ctx context.Context, b *bot.Bot, msg *models.Message) {
	defer cfg.endInteraction(msg)
	telegramId := msg.From.ID
	cfg.userBalancesMu.RLock()
	userBalances, ok := cfg.usersBalances[telegramID(telegramId)]
	cfg.userBalancesMu.RUnlock()
	if !ok {
		return
	}
	valid, _ := regexp.MatchString(`^[a-zA-Z]*$`, msg.Text)
	if ok, _ := regexp.MatchString(`^[0-9]*(.)?[0-9]*$`, msg.Text); valid || !ok {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: msg.Chat.ID,
			Text:   "invalid spend amount",
			ReplyParameters: &models.ReplyParameters{
				MessageID:                msg.ID,
				AllowSendingWithoutReply: true,
			},
		})
		return
	}
	getBalanceResp := getBalanceRespBody{}
	err := WalletServiceCall("GET", fmt.Sprintf("%s/v1/balance", cfg.bwsOrigin), cfg.bwsApiKey, ReqBody{TelegramID: telegramId}, &getBalanceResp)
	if err != nil {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: msg.Chat.ID,
			Text:   "Something went wrong, please try again",
		})
		log.Println(err.Error())
		return
	}
	walletBalance := getBalanceResp.Balance
	amount, err := strconv.ParseFloat(msg.Text, 64)
	if err != nil {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: msg.Chat.ID,
			Text:   "invalid number",
			ReplyParameters: &models.ReplyParameters{
				MessageID:                msg.ID,
				AllowSendingWithoutReply: true,
			},
		})
		return
	}
	if amount == 0.0 {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: msg.Chat.ID,
			Text:   "spend value cannot be 0 MON",
			ReplyParameters: &models.ReplyParameters{
				MessageID:                msg.ID,
				AllowSendingWithoutReply: true,
			},
		})
		return
	}
	walletBalanceAsFloat, _ := strconv.ParseFloat(walletBalance, 64)
	if amount > walletBalanceAsFloat {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: msg.Chat.ID,
			Text:   "spend value cannot be greater than wallet balance",
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
	tokenDecimals, _ := strconv.Atoi(token.Decimals)
	executingMsg, _ := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: msg.Chat.ID,
		Text:   "Executing purchase...",
	})
	buyResult, err := cfg.handlerBuy(ctx, telegramId, msg.Text, token.Address, uint8(tokenDecimals))
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
		Text:      fmt.Sprintf("Purchase successful: Bought *%v %s* for *%v MON*\n[View on the explorer](https://testnet.monadexplorer.com/tx/%s)", strings.Replace(buyResult.BoughtAmount, ".", "\\.", 1), token.Symbol, strings.Replace(msg.Text, ".", "\\.", 1), buyResult.TxHash),
	})
	b.DeleteMessages(ctx, &bot.DeleteMessagesParams{
		ChatID:     msg.Chat.ID,
		MessageIDs: []int{processingMsg.ID, executingMsg.ID},
	})
}
