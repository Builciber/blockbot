package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

func (cfg *apiConfig) handlerReferralButton(ctx context.Context, b *bot.Bot, update *models.Update) {
	telegramID := update.CallbackQuery.From.ID
	referralData, err := cfg.DB.GetReferralData(ctx, telegramID)
	if err != nil {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.CallbackQuery.Message.Message.Chat.ID,
			Text:   "Something went wrong, please try again",
		})
		log.Println(err.Error())
		return
	}
	referralCode, ok := referralData.Referralcode.(string)
	if !ok {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.CallbackQuery.Message.Message.Chat.ID,
			Text:   "Something went wrong, please try again",
		})
		log.Println("failed to parse referral code as string")
		return
	}
	referralEarnings := pgNumericToString(referralData.Referralearnings)
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    update.CallbackQuery.Message.Message.Chat.ID,
		ParseMode: models.ParseModeMarkdown,
		Text:      fmt.Sprintf("*Referral Information*\n\nYour referral link: https://t\\.me/Monad\\_BlockBot?start\\=r\\_%s\n\nReferrals: *%d*\n\nTotal referral earnings: *%s MON*", referralCode, referralData.Referralcount.Int32, strings.Replace(displayDecimal(referralEarnings, 2), ".", "\\.", 1)),
	})
}
