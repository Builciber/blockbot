package main

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/Builciber/blockbot/internal/database"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/jackc/pgx/v5/pgtype"
)

func (cfg *apiConfig) handlerSetBuyButtonRight(ctx context.Context, b *bot.Bot, update *models.Update) {
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.CallbackQuery.Message.Message.Chat.ID,
		Text:   "Reply with your new setting for the right Buy Button in MON.",
		ReplyMarkup: models.ForceReply{
			ForceReply:            true,
			InputFieldPlaceholder: "1.5",
			Selective:             false,
		},
	})
	retValues := make([]string, 5)
	retValues[0] = strconv.FormatInt(int64(update.CallbackQuery.Message.Message.ID), 10)
	cfg.mu.Lock()
	cfg.intSeqMap[chatID(update.CallbackQuery.Message.Message.Chat.ID)] = &interactionSequence{
		funcSlice:   []interactionHandler{cfg.handlerProcessBuyButtonRight},
		retValues:   retValues,
		createdAt:   time.Now(),
		nextFuncIdx: 0,
	}
	cfg.mu.Unlock()
}

func (cfg *apiConfig) handlerProcessBuyButtonRight(ctx context.Context, b *bot.Bot, msg *models.Message) {
	defer cfg.endInteraction(msg)
	telegramID := msg.From.ID
	valid, _ := regexp.MatchString(`^[a-zA-Z]*$`, msg.Text)
	if ok, _ := regexp.MatchString(`^[0-9]*(.)?[0-9]*$`, msg.Text); valid || !ok {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: msg.Chat.ID,
			Text:   "Invalid right buy button value",
			ReplyParameters: &models.ReplyParameters{
				MessageID:                msg.ID,
				AllowSendingWithoutReply: true,
			},
		})
		return
	}
	if val, _ := strconv.ParseFloat(msg.Text, 64); val == 0.0 {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: msg.Chat.ID,
			Text:   "right buy button amount cannot be zero",
			ReplyParameters: &models.ReplyParameters{
				MessageID:                msg.ID,
				AllowSendingWithoutReply: true,
			},
		})
		return
	}
	var exp int32
	var intAmount int64
	var err error
	if strings.Contains(msg.Text, ".") {
		parts := strings.Split(msg.Text, ".")
		fractionalPart := parts[1]
		integralPart := parts[0]
		if val, _ := strconv.ParseInt(fractionalPart, 10, 64); val == 0 {
			exp = 0
			intAmount, _ = strconv.ParseInt(integralPart, 10, 64)
		}
		exp = int32(len(fractionalPart)) * -1
		intAmount, _ = strconv.ParseInt(strings.Join(parts, ""), 10, 64)
	} else {
		intAmount, err = strconv.ParseInt(msg.Text, 10, 64)
		if err != nil {
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: msg.Chat.ID,
				Text:   "Something went wrong, please try again",
			})
			log.Println(err.Error())
			return
		}
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
	err = cfg.DB.UpdateBuyButtonRight(ctx, database.UpdateBuyButtonRightParams{
		TelegramID:     telegramID,
		BuyButtonRight: pgtype.Numeric{Int: big.NewInt(intAmount), Exp: exp, Valid: true},
		UpdatedAt:      pgtype.Timestamp{Time: time.Now(), Valid: true},
	})
	if err != nil {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: msg.Chat.ID,
			Text:   "Something went wrong, please try again",
		})
		log.Println(err.Error())
		return
	}
	settings.BuyButtonRight = pgtype.Numeric{Int: big.NewInt(intAmount), Exp: exp, Valid: true}
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
		Text:   fmt.Sprintf("Right Buy Button value set to: %s MON", msg.Text),
	})
}
