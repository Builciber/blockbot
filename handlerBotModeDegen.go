package main

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"strings"
	"time"

	"github.com/Builciber/blockbot/internal/database"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

func (cfg *apiConfig) handlerBotModeDegen(ctx context.Context, b *bot.Bot, update *models.Update) {
	defer cfg.endInteraction(update.CallbackQuery.Message.Message)
	telegramID := update.CallbackQuery.From.ID
	cfg.mu.RLock()
	intSeq, ok := cfg.intSeqMap[chatID(update.CallbackQuery.Message.Message.Chat.ID)]
	cfg.mu.RUnlock()
	if !ok {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.CallbackQuery.Message.Message.Chat.ID,
			Text:   "interaction expired. Please use the /start command to continue",
		})
		return
	}
	var referrerID pgtype.Int8
	slice := strings.Split(intSeq.retValues[0], " ")
	if len(slice) == 2 {
		userTGID, err := cfg.DB.GetUserByRefCode(ctx, retrieveRefCode(slice[1]))
		if err != nil && err == pgx.ErrNoRows {
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: update.CallbackQuery.Message.Message.Chat.ID,
				Text:   "invalid referral code",
			})
			return
		}
		if err != nil {
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: update.CallbackQuery.Message.Message.Chat.ID,
				Text:   "something went wrong, please try again",
			})
			log.Println(err.Error())
			return
		}
		referrerID = pgtype.Int8{Int64: userTGID, Valid: true}
	}
	resp := createWalletRespBody{}
	err := WalletServiceCall("POST", fmt.Sprintf("%s/v1/create", cfg.bwsOrigin), cfg.bwsApiKey, ReqBody{TelegramID: telegramID}, &resp)
	if err != nil {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.CallbackQuery.Message.Message.Chat.ID,
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
				ChatID: update.CallbackQuery.Message.Message.Chat.ID,
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

	creationTime := time.Now()
	createUserParams := database.CreateUserParams{
		TelegramID:    telegramID,
		ChatID:        update.CallbackQuery.Message.Message.Chat.ID,
		WalletAddress: resp.WalletAddress,
		ReferrerID:    referrerID,
		ReferralCode:  refCode,
		CreatedAt:     pgtype.Timestamp{Time: creationTime, Valid: true},
		UpdatedAt:     pgtype.Timestamp{Time: creationTime, Valid: true},
	}
	createUserSettingsParams := database.CreateUserSettingsParams{
		TelegramID:      telegramID,
		BuySlippage:     15,
		SellSlippage:    15,
		MaxPriceImpact:  25,
		PriorityFee:     "turbo",
		AutoBuy:         false,
		AutoBuyAmount:   pgtype.Numeric{Int: big.NewInt(10), Exp: -1, Valid: true},
		BuyButtonLeft:   pgtype.Numeric{Int: big.NewInt(10), Exp: -1, Valid: true},
		BuyButtonRight:  pgtype.Numeric{Int: big.NewInt(50), Exp: -1, Valid: true},
		SellButtonLeft:  25,
		SellButtonRight: 100,
		CreatedAt:       pgtype.Timestamp{Time: creationTime, Valid: true},
		UpdatedAt:       pgtype.Timestamp{Time: creationTime, Valid: true},
	}
	err = cfg.createUserTx(ctx, createUserParams, createUserSettingsParams)
	if err != nil {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.CallbackQuery.Message.Message.Chat.ID,
			Text:   "Something went wrong, please try again",
		})
		log.Println(err.Error())
		return
	}
	responseText := fmt.Sprintf("⚡ Degen Mode locked in\\.\nI see you like things super fast\\. Respect\\. 🫡\n\nWe’ve tuned your engine for max performance:\n– Slippage: 15%%\n– Maximimum Acceptable Price Impact: 25%%\n– Transaction Priority: Turbo 🚀\n\nWant to tweak any of these later?\nYou can always tweak any of these individually via `/settings` or as a group via the `/changemode` command\\.\n\nAll systems are a go, your wallet is ready\\.\nYou’re now equipped to trade like a savage\\. To start trading, tap the address to copy it then send MON to it: \n\n`%s`", resp.WalletAddress)
	_, err = b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      update.CallbackQuery.Message.Message.Chat.ID,
		ParseMode:   models.ParseModeMarkdown,
		Text:        responseText,
		ReplyMarkup: homeKeyboard,
	})
	if err != nil {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.CallbackQuery.Message.Message.Chat.ID,
			Text:   "Something went wrong, please try again",
		})
		log.Println(err.Error())
		return
	}
	b.DeleteMessage(ctx, &bot.DeleteMessageParams{
		ChatID:    update.CallbackQuery.Message.Message.Chat.ID,
		MessageID: update.CallbackQuery.Message.Message.ID,
	})
	found := strings.Contains(intSeq.retValues[0], "_ca_")
	if found {
		tokenAddress := strings.Split(intSeq.retValues[0], "_ca_")[1][0:42]
		inputs := sharedTokenAddressFuncInputs{
			telegramId:   telegramID,
			chatId:       update.CallbackQuery.Message.Message.Chat.ID,
			tokenAddress: tokenAddress,
		}
		cfg.handleSharedTokenAddress(ctx, b, inputs)
		return
	}
}
