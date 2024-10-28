package main

import (
	"context"
	"database/sql"
	"internal/database"
	"log"
	"math/rand"
	"strings"
	"time"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

type createWalletRespBody struct {
	TelegramID    int64  `json:"telegram_id"`
	WalletAddress string `json:"wallet_address"`
}

func (cfg *apiConfig) handlerStart(ctx context.Context, b *bot.Bot, update *models.Update) {
	telegramID := update.Message.From.ID
	isUser, err := cfg.DB.IsExistingUser(ctx, telegramID)
	if err != nil {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "Something went wrong, please try again",
		})
		log.Println(err.Error())
		return
	}
	if isUser {
		cfg.handlerHome(ctx, b, update)
		return
	}

	resp := createWalletRespBody{}
	err = WalletServiceCall("POST", "http://localhost:8080/v1/create", cfg.bwsApiKey, ReqBody{TelegramID: telegramID}, &resp)
	if err != nil {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "Something went wrong, please try again",
		})
		log.Println(err.Error())
		return
	}

	refCode := generateReferralCode()
	for {
		used, err := cfg.DB.IsExistingRefCode(ctx, refCode)
		if err != nil {
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: update.Message.Chat.ID,
				Text:   "Something went wrong, please try again",
			})
			log.Println(err.Error())
			return
		}
		if used {
			refCode = generateReferralCode()
			continue
		}
		break
	}

	var referrerID sql.NullInt64
	slice := strings.Split(update.Message.Text, " ")
	if len(slice) == 2 {
		userTGID, err := cfg.DB.GetUserByRefCode(ctx, strings.TrimPrefix(slice[1], "r_"))
		if err != nil && err.Error() != "sql: no rows in result set" {
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: update.Message.Chat.ID,
				Text:   "Something went wrong, please try again",
			})
			log.Println(err.Error())
			return
		}
		if userTGID > 0 {
			referrerID = sql.NullInt64{Int64: userTGID, Valid: true}
		}
	}

	creationTime := time.Now()
	err = cfg.DB.CreateUser(ctx, database.CreateUserParams{
		TelegramID:    telegramID,
		WalletAddress: resp.WalletAddress,
		ReferrerID:    referrerID,
		ReferralCode:  refCode,
		CreatedAt:     creationTime,
		UpdatedAt:     creationTime,
	})
	if err != nil {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "Something went wrong, please try again",
		})
		log.Println(err.Error())
		return
	}
	cfg.handlerHome(ctx, b, update)
}

func generateReferralCode() string {
	const alphabet = "abcdefghijklmnopqrstuvwxyz"
	builder := new(strings.Builder)
	for i := 0; i < 8; i++ {
		builder.WriteByte(alphabet[rand.Intn(26)])
	}
	return builder.String()
}
