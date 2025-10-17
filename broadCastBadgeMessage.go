package main

import (
	"bytes"
	"context"
	"fmt"
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
	if !messageState.SentBadgeMsg.Bool {
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
	}
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
		cfg.DB.SentFeedBackBadgeMsg(ctx, username)
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
