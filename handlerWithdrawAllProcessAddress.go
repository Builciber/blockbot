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

type withdrawAllReqBody struct {
	TelegramID int64  `json:"telegram_id"`
	WithdrawTo string `json:"withdraw_to"`
}

type withdrawRespBody struct {
	Success bool   `json:"success"`
	TxHash  string `json:"tx_hash"`
}

func (cfg *apiConfig) handlerWithdrawAllProcessAddress(ctx context.Context, b *bot.Bot, msg *models.Message) {
	defer cfg.endInteraction(msg)
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: msg.Chat.ID,
		Text:   "Processing... Please wait.",
		ReplyParameters: &models.ReplyParameters{
			MessageID:                msg.ID,
			AllowSendingWithoutReply: true,
		},
	})
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
	telegramID := msg.From.ID
	requestBody := withdrawAllReqBody{
		TelegramID: telegramID,
		WithdrawTo: msg.Text,
	}
	withdrawResp := withdrawRespBody{}
	err := WalletServiceCall("POST", fmt.Sprintf("%s/v1/withdraw_all", cfg.bwsOrigin), cfg.bwsApiKey, requestBody, &withdrawResp)
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
			Text:      fmt.Sprintf("✅✅✅✅\n\nWithdrawal Successful: [View on Monad explorer](https://monadvision\\.com/tx/%s)", withdrawResp.TxHash),
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
			Text:      fmt.Sprintf("❌❌❌❌\n\nWithdrawal Failed: [View on Monad Explorer](https://monadvision\\.com/tx/%s)", withdrawResp.TxHash),
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
