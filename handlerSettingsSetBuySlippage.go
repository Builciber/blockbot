package main

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"time"

	"github.com/Builciber/blockbot/internal/database"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/jackc/pgx/v5/pgtype"
)

func (cfg *apiConfig) handlerSetBuySlippage(ctx context.Context, b *bot.Bot, update *models.Update) {
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.CallbackQuery.Message.Message.Chat.ID,
		Text:   "Reply with your new setting for the Buy Slippage in % (1 - 75%). Whole numbers only",
		ReplyMarkup: models.ForceReply{
			ForceReply:            true,
			InputFieldPlaceholder: "10",
			Selective:             false,
		},
	})
	retValues := make([]string, 5)
	retValues[0] = strconv.FormatInt(int64(update.CallbackQuery.Message.Message.ID), 10)
	cfg.mu.Lock()
	cfg.intSeqMap[chatID(update.CallbackQuery.Message.Message.Chat.ID)] = &interactionSequence{
		funcSlice:   []interactionHandler{cfg.handlerProcessBuySlippage},
		retValues:   retValues,
		createdAt:   time.Now(),
		nextFuncIdx: 0,
	}
	cfg.mu.Unlock()
}

func (cfg *apiConfig) handlerProcessBuySlippage(ctx context.Context, b *bot.Bot, msg *models.Message) {
	defer cfg.endInteraction(msg)
	telegramID := msg.From.ID
	valid, _ := regexp.MatchString(`^[a-zA-Z]*$`, msg.Text)
	if ok, _ := regexp.MatchString(`^[0-9]*$`, msg.Text); valid || !ok {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: msg.Chat.ID,
			Text:   "Invalid Buy Slippage percent. Integer percentage expected.",
			ReplyParameters: &models.ReplyParameters{
				MessageID:                msg.ID,
				AllowSendingWithoutReply: true,
			},
		})
		return
	}
	percent, err := strconv.ParseInt(msg.Text, 10, 64)
	if err != nil {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: msg.Chat.ID,
			Text:   "Something went wrong, please try again",
		})
		log.Println(err.Error())
		return
	}
	if percent < 1 || percent > 75 {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: msg.Chat.ID,
			Text:   "Invalid Buy Slippage percent. Valid percentage range is 1 to 75 inclusive.",
		})
		return
	}
	settings, err := cfg.DB.GetUserSettings(ctx, telegramID)
	if err != nil {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: msg.Chat.ID,
			Text:   "Something went wrong, please try again",
		})
		log.Println(err.Error())
		return
	}
	err = cfg.DB.UpdateBuySlippage(ctx, database.UpdateBuySlippageParams{
		TelegramID:  telegramID,
		BuySlippage: int16(percent),
		UpdatedAt:   pgtype.Timestamp{Time: time.Now(), Valid: true},
	})
	if err != nil {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: msg.Chat.ID,
			Text:   "Something went wrong, please try again",
		})
		log.Println(err.Error())
		return
	}
	settings.BuySlippage = int16(percent)
	settingsKeyboard, err := generateSettingsKeyboard(&settings)
	if err != nil {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: msg.Chat.ID,
			Text:   "Something went wrong, please try again",
		})
		log.Println(err.Error())
		return
	}
	cfg.mu.RLock()
	intSeq, ok := cfg.intSeqMap[chatID(msg.Chat.ID)]
	cfg.mu.RUnlock()
	if !ok {
		return
	}
	msgID, _ := strconv.ParseInt(intSeq.retValues[0], 10, 64)
	_, err = b.EditMessageReplyMarkup(ctx, &bot.EditMessageReplyMarkupParams{
		MessageID:   int(msgID),
		ChatID:      msg.Chat.ID,
		ReplyMarkup: settingsKeyboard,
	})
	if err != nil {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: msg.Chat.ID,
			Text:   "Something went wrong, please try again",
		})
		log.Println(err.Error())
		return
	}
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: msg.Chat.ID,
		Text:   fmt.Sprintf("Buy Slippage set to: %s%%", msg.Text),
	})
}
