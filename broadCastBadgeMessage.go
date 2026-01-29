package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/Builciber/blockbot/internal/database"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

var testerMessage = `Hey, it’s me — Blockie 

You’ve been part of my early runs, testing trades and helping me get sharper on Monad. I just wanted to say thank you for trusting me this early.

We’re getting ready for Mainnet, and I’ll be there on day one - faster, smarter, and smoother than ever.

Oh, and I’ve had a bit of a glow-up 👀
The UI/UX got a full upgrade – you’ll now see more token info like liquidity, market cap, launchpad source, and price changes across different timeframes.

Here’s a glimpse of what’s rolling out next:
• 💰 PnL Cards
• 🎯 Take-Profit & Stop-Loss
• 📈 Limit Orders
• 🔒 [REDACTED] — stay tuned.

Appreciate you being early, frens.💚💜

— From your favorite trading buddy, Blockie`

var feedbackMessage = `Yo, it’s Blockie again 

This is a special appreciation for providing feedback. I saw your feedback and it seriously helped.
You caught things I missed, shared insights that made me sharper, and helped me understand what real traders need.

Thanks for taking the time to speak up and shape how I grow.
Every bit of feedback made me better and brought the team closer to being mainnet ready.
Go ahead and let the TL know who built with Blockie first.

Couldn’t have done it without you
See you on mainnet day one.

— From your favorite trading buddy (and the BlockBot team)`

func (cfg *apiConfig) sendBadgeMessage(ctx context.Context, b *bot.Bot, update *models.Update) {
	var chatId int64
	var username string
	var telegramId int64
	if update.CallbackQuery == nil {
		chatId = update.Message.Chat.ID
		username = strings.ToLower(update.Message.From.Username)
		telegramId = update.Message.From.ID
	} else {
		chatId = update.CallbackQuery.Message.Message.Chat.ID
		username = strings.ToLower(update.CallbackQuery.Message.Message.From.Username)
		telegramId = update.CallbackQuery.Message.Message.From.ID
	}
	messageState, err := cfg.DB.GetBadgeMessageState(ctx, username)
	if err != nil {
		return
	}
	cfg.DB.UpdateUserChatId(ctx, database.UpdateUserChatIdParams{
		TelegramID: telegramId,
		ChatID:     chatId,
	})
	/*if !messageState.SentBadgeMsg.Bool {
		fileContent, _ := os.ReadFile("tester_badge.jpg")
		b.SendPhoto(ctx, &bot.SendPhotoParams{
			ChatID: chatId,
			Photo: &models.InputFileUpload{
				Filename: "tester_badge.jpg",
				Data:     bytes.NewReader(fileContent),
			},
		})
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatId,
			Text:   fmt.Sprintf(testerMessage, username),
		})
		cfg.DB.SentTestBadgeMsg(ctx, username)
	}*/
	if !messageState.SentFeedbackBadgeMsg.Bool && messageState.GaveFeedback.Bool {
		fileContent, _ := os.ReadFile("feedback_badge.jpg")
		b.SendPhoto(ctx, &bot.SendPhotoParams{
			ChatID: chatId,
			Photo: &models.InputFileUpload{
				Filename: "feedback_badge.jpg",
				Data:     bytes.NewReader(fileContent),
			},
		})
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatId,
			Text:   feedbackMessage,
		})
		cfg.updateFeedbackBadgeStatusTx(ctx, username, telegramId)
	}
}

func (cfg *apiConfig) sendTestBadgeToAllUsers(ctx context.Context, b *bot.Bot) {
	userIds, err := cfg.DB.GetUserIds(ctx)
	if err != nil {
		return
	}
	for _, userId := range userIds {
		fileContent, _ := os.ReadFile("tester_badge.jpg")
		_, err = b.SendPhoto(ctx, &bot.SendPhotoParams{
			ChatID: userId,
			Photo: &models.InputFileUpload{
				Filename: "tester_badge.jpg",
				Data:     bytes.NewReader(fileContent),
			},
		})
		if err != nil {
			return
		}
		_, err = b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: userId,
			Text:   testerMessage,
		})
		if err != nil {
			return
		}
		err = cfg.DB.CreateBadgeReceiver(ctx, database.CreateBadgeReceiverParams{
			TelegramID:   userId,
			HasTestBadge: true,
		})
		if err != nil {
			return
		}
		time.Sleep(500 * time.Millisecond)
	}
	cfg.DB.MarkAllSentBadgeMsgTrue(ctx)
}

func (cfg *apiConfig) updateFeedbackBadgeStatusTx(ctx context.Context, telegramUsername string, telegramId int64) error {
	tx, err := cfg.dbConn.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	qtx := cfg.DB.WithTx(tx)
	err = qtx.SentFeedBackBadgeMsg(ctx, telegramUsername)
	if err != nil {
		return err
	}
	err = qtx.SendFeedbackBadge(ctx, telegramId)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}

var shutdownMessage = `Blockbot Shutdown Notice ⚠️

Blockbot operations will officially shut down | Platform operations will cease.

Your private key is exported in the next message. Copy and keep it safe. We strongly advise all users to move their funds to a fresh wallet as an additional security measure.

[Please remember:
Never share your private keys or seed phrase with anyone. Your wallet security is your responsibility.]

Thank you to everyone who supported Blockbot.`

func (cfg *apiConfig) sendShutdownMessage(ctx context.Context, b *bot.Bot) error {
	client := &http.Client{}
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/v1/export/all", cfg.bwsOrigin), nil)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "ApiKey "+cfg.bwsApiKey)
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode > 299 {
		errMsg := ErrorResp{}
		err = json.Unmarshal(body, &errMsg)
		if err != nil {
			return err
		}
		return errors.New(errMsg.Error)
	}
	keys := []ExportWalletRespBody{}
	err = json.Unmarshal(body, &keys)
	if err != nil {
		return err
	}
	for _, wallet := range keys {
		_, err = b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: wallet.TelegramID,
			Text:   shutdownMessage,
		})
		if err != nil {
			lowercase := strings.ToLower(err.Error())
			if strings.Contains(lowercase, "bad request") || strings.Contains(lowercase, "forbidden") {
				log.Printf("chat not found or I was blocked by user with ID %d\n", wallet.TelegramID)
				continue
			}
			return err
		}
		_, err = b.SendMessage(ctx, &bot.SendMessageParams{
			ParseMode: models.ParseModeMarkdown,
			ChatID:    wallet.TelegramID,
			Text:      fmt.Sprintf("`%s` \\(Tap to copy\\)", wallet.PrivateKey),
		})
		if err != nil {
			return err
		}
		b, err := json.Marshal(ReqBody{TelegramID: wallet.TelegramID})
		if err != nil {
			return err
		}
		req, err := http.NewRequest("PUT", fmt.Sprintf("%s/v1/mark", cfg.bwsOrigin), bytes.NewBuffer(b))
		if err != nil {
			return err
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "ApiKey "+cfg.bwsApiKey)
		resp, err := client.Do(req)
		if err != nil {
			return err
		}
		if resp.StatusCode > 299 {
			errMsg := ErrorResp{}
			err = json.Unmarshal(body, &errMsg)
			if err != nil {
				return err
			}
			return errors.New(errMsg.Error)
		}
	}
	return nil
}

func (cfg *apiConfig) sendShutdownMessageWorker(ctx context.Context, b *bot.Bot) {
	err := cfg.sendShutdownMessage(ctx, b)
	if err != nil {
		log.Printf("failed to finish sending shutdown message: %s", err.Error())
		return
	}
	log.Println("finished sending shutdown message")
}
