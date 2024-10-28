package main

import (
	"context"
	"log"
	"regexp"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

func (cfg *apiConfig) handlerWithdrawProcessAddress(ctx context.Context, b *bot.Bot, msg *models.Message) {
	if ok, _ := regexp.MatchString(`^0x[0-9a-fA-F]{40}$`, msg.Text); !ok {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: msg.Chat.ID,
			Text:   "Invalid withdrawal wallet address",
			ReplyParameters: &models.ReplyParameters{
				MessageID:                msg.ID,
				AllowSendingWithoutReply: true,
			},
		})
		return
	}
	_, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: msg.Chat.ID,
		Text:   "Reply with the amount to withdraw",
		ReplyParameters: &models.ReplyParameters{
			MessageID:                msg.ID,
			AllowSendingWithoutReply: true,
		},
		ReplyMarkup: models.ForceReply{
			ForceReply:            true,
			InputFieldPlaceholder: "69.420",
			Selective:             false,
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
	cfg.mu.Lock()
	intSeq := cfg.intSeqMap[chatID(msg.Chat.ID)]
	intSeq.retValues[intSeq.nextFuncIdx] = msg.Text
	intSeq.nextFuncIdx++
	cfg.mu.Unlock()
}
