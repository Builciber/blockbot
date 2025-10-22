package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"math/big"
	"math/rand"
	"regexp"
	"strings"
	"time"

	"github.com/Builciber/blockbot/internal/database"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/jackc/pgx/v5/pgconn"
)

func (cfg *apiConfig) handlerGeneratePnlCard(ctx context.Context, b *bot.Bot, update *models.Update) {
	telegramId := update.Message.From.ID
	isUser, err := cfg.DB.IsExistingUser(ctx, telegramId)
	if err != nil {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "Something went wrong, please try again",
		})
		log.Println(err.Error())
		return
	}
	if !isUser {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "Forbidden action",
		})
		return
	}
	split := strings.Split(update.Message.Text, " ")
	str := split[1]
	tokenAddress := strings.TrimPrefix(str, "pnlcard_")
	if ok, _ := regexp.MatchString(`^0x[0-9a-fA-F]{40}$`, tokenAddress); !ok {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "Invalid token address",
		})
		return
	}
	position, err := cfg.DB.CallGetPositionFunc(ctx, database.CallGetPositionFuncParams{Traderid: telegramId, Tokenaddress: tokenAddress})
	if err == nil {
		const zeroAddress = "0x0000000000000000000000000000000000000000"
		token, err := cfg.getToken(tokenAddress, zeroAddress)
		if err != nil {
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: update.Message.Chat.ID,
				Text:   "Something went wrong, please try again",
			})
			log.Println(err.Error())
			return
		}
		if token.Price != "" && token.Price != "0" {
			monPrice, err := getMONUSDPrice()
			if err != nil {
				b.SendMessage(ctx, &bot.SendMessageParams{
					ChatID: update.Message.Chat.ID,
					Text:   "Something went wrong, please try again",
				})
				log.Println(err.Error())
				return
			}
			MonUsdPrice, ok := new(big.Float).SetString(monPrice)
			if !ok {
				b.SendMessage(ctx, &bot.SendMessageParams{
					ChatID: update.Message.Chat.ID,
					Text:   "refresh this position to proceed",
				})
				log.Println("MONUSD price for user call missing")
				return
			}
			tokenUsdPrice, _ := new(big.Float).SetString(token.Price)
			currPricePerToken := new(big.Float)
			currPricePerToken.Quo(tokenUsdPrice, MonUsdPrice) //price of token in MON
			initialMonCost, _ := new(big.Float).SetString(pgNumericToString(position.TotalMonCost))
			totalTokenAmount, _ := new(big.Float).SetString(pgNumericToString(position.TotalTokenAmount))
			currentMonValue := new(big.Float)
			currentMonValue.Mul(currPricePerToken, totalTokenAmount)
			pnl := new(big.Float)
			pnl.Sub(currentMonValue, initialMonCost)
			ratio := new(big.Float)
			ratio.Quo(pnl, initialMonCost)
			pnlPercent := new(big.Float)
			pnlPercent.Mul(ratio, big.NewFloat(100))
			pnlPercentFormatted := formatPnl(formatFloat(pnlPercent, 2))
			idx := rand.Intn(len(profitBackground))
			imageBackground := profitBackground[idx]
			isProfit := true
			if pnlPercentFormatted[0] == '-' {
				idx = rand.Intn(len(lossBackgrounds))
				imageBackground = lossBackgrounds[idx]
				isProfit = false
			}
			refCode, err := cfg.DB.GetReferralCode(ctx, update.Message.From.ID)
			if err != nil {
				b.SendMessage(ctx, &bot.SendMessageParams{
					ChatID: update.Message.Chat.ID,
					Text:   "something went wrong, please try again",
				})
				log.Println(err.Error())
				return
			}
			duration := time.Now().Unix() - position.CreatedAt.Time.Unix()
			pnlData := PNLCardData{
				TokenPair:      fmt.Sprintf("%s/MON", token.Symbol),
				PercentageGain: pnlPercentFormatted + "%",
				TradeDuration:  formatDuration(time.Duration(duration) * time.Second),
				ReferralCode:   fmt.Sprintf("https://t.me/Monad_BlockBot?start=r_%s", refCode),
				IsProfit:       isProfit,
				BackgroundPath: imageBackground,
			}
			buf, err := generatePNLCard(pnlData)
			if err != nil {
				b.SendMessage(ctx, &bot.SendMessageParams{
					ChatID: update.Message.Chat.ID,
					Text:   "something went wrong, please try again",
				})
				log.Println(err.Error())
				return
			}
			b.DeleteMessage(ctx, &bot.DeleteMessageParams{
				ChatID:    update.Message.Chat.ID,
				MessageID: update.Message.ID,
			})
			params := &bot.SendPhotoParams{
				ChatID: update.Message.Chat.ID,
				Photo: &models.InputFileUpload{
					Filename: "pnl_card.png",
					Data:     bytes.NewReader(buf.Bytes()),
				},
			}
			b.SendPhoto(ctx, params)
			return
		}
	}
	if err != nil {
		if pgErr, ok := err.(*pgconn.PgError); !(ok && pgErr.Code == "P0002") { // if not a `no_data_found` PL/pgsql error
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: update.Message.Chat.ID,
				Text:   "Something went wrong, please try again",
			})
			log.Println(err.Error())
			return
		}
	}
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   "PnL information not available",
	})
}
