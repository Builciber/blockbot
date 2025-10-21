package main

import (
	"bytes"
	"context"
	"time"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

func sendPNLCard(ctx context.Context, b *bot.Bot, chatID int64, tokenPair string, percentageChange float64, duration time.Duration, referralCode string) error {
	var cardBuf *bytes.Buffer
	var err error

	if percentageChange >= 0 {
		cardBuf, err = generateProfitCard(tokenPair, percentageChange, duration, referralCode)
	} else {
		cardBuf, err = generateLossCard(tokenPair, percentageChange, duration, referralCode)
	}

	if err != nil {
		return err
	}

	params := &bot.SendPhotoParams{
		ChatID: chatID,
		Photo: &models.InputFileUpload{
			Filename: "pnl_card.png",
			Data:     bytes.NewReader(cardBuf.Bytes()),
		},
		Caption: "Your PnL Summary",
	}

	_, err = b.SendPhoto(ctx, params)
	return err
}
