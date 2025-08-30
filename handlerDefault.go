package main

import (
	"context"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

func (cfg *apiConfig) handlerDefault(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}
	if update.Message.Text == "" {
		return
	}
	if update.Message.ReplyToMessage == nil {
		cfg.handlerBuyCommand(ctx, b, update.Message)
		return
	}
	chatId := chatID(update.Message.Chat.ID)
	cfg.mu.RLock()
	intSeq, ok := cfg.intSeqMap[chatID(chatId)]
	cfg.mu.RUnlock()
	if !ok {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:    update.Message.Chat.ID,
			ParseMode: models.ParseModeMarkdown,
			Text:      "*Unknown Request*\n\nPress the Menu button to see a list of commands or use the '/home' command to open main menu",
			ReplyParameters: &models.ReplyParameters{
				MessageID:                update.Message.ID,
				AllowSendingWithoutReply: true,
			},
		})
		return
	}
	handler := intSeq.funcSlice[intSeq.nextFuncIdx]
	handler(ctx, b, update.Message)
}
