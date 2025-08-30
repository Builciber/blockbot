package main

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strings"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

type withdrawReqBody struct {
	TelegramID int64  `json:"telegram_id"`
	WithdrawTo string `json:"withdraw_to"`
	Amount     string `json:"amount"`
}

func (cfg *apiConfig) handlerWithdrawProcessAmount(ctx context.Context, b *bot.Bot, msg *models.Message) {
	defer cfg.endInteraction(msg)
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: msg.Chat.ID,
		Text:   "Processing... Please wait.",
		ReplyParameters: &models.ReplyParameters{
			MessageID:                msg.ID,
			AllowSendingWithoutReply: true,
		},
	})
	valid, _ := regexp.MatchString(`^[a-zA-Z]*$`, msg.Text)
	if ok, _ := regexp.MatchString(`^[0-9]*(.)?[0-9]*$`, msg.Text); valid || !ok {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: msg.Chat.ID,
			Text:   "Invalid withdrawal amount",
			ReplyParameters: &models.ReplyParameters{
				MessageID:                msg.ID,
				AllowSendingWithoutReply: true,
			},
		})
		return
	}
	telegramID := msg.From.ID
	cfg.mu.RLock()
	intSeq := cfg.intSeqMap[chatID(msg.Chat.ID)]
	cfg.mu.RUnlock()
	requestBody := withdrawReqBody{
		TelegramID: telegramID,
		WithdrawTo: intSeq.retValues[intSeq.nextFuncIdx-1],
		Amount:     msg.Text,
	}
	withdrawResp := withdrawRespBody{}
	err := WalletServiceCall("POST", fmt.Sprintf("%s/v1/withdraw", cfg.bwsOrigin), cfg.bwsApiKey, requestBody, &withdrawResp)
	if err != nil {
		if strings.HasPrefix(err.Error(), "User Error:") {
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: msg.Chat.ID,
				Text:   strings.TrimPrefix(err.Error(), "User Error: "),
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
	if withdrawResp.Success {
		_, err = b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:    msg.Chat.ID,
			ParseMode: models.ParseModeMarkdown,
			Text:      fmt.Sprintf("✅✅✅✅\n\nWithdrawal Successful: [View on Monad Explorer](https://testnet\\.monadexplorer\\.com/tx/%s)", withdrawResp.TxHash),
			ReplyParameters: &models.ReplyParameters{
				MessageID:                msg.ID,
				AllowSendingWithoutReply: true,
			},
		})
		if err != nil {
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: msg.Chat.ID,
				Text:   "Something went wrong, please try again",
			})
			log.Println(err.Error())
			return
		}
	} else {
		_, err = b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:    msg.Chat.ID,
			ParseMode: models.ParseModeMarkdown,
			Text:      fmt.Sprintf("❌❌❌❌\n\nWithdrawal Failed: [View on Monad Explorer](https://testnet\\.monadexplorer\\.com/tx/%s)", withdrawResp.TxHash),
			ReplyParameters: &models.ReplyParameters{
				MessageID:                msg.ID,
				AllowSendingWithoutReply: true,
			},
		})
		if err != nil {
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: msg.Chat.ID,
				Text:   "Something went wrong, please try again",
			})
			log.Println(err.Error())
			return
		}
	}
}

func (cfg *apiConfig) endInteraction(msg *models.Message) {
	cfg.mu.Lock()
	delete(cfg.intSeqMap, chatID(msg.Chat.ID))
	cfg.mu.Unlock()
}
