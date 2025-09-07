package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"regexp"
	"strings"
	"time"

	"github.com/Builciber/blockbot/internal/database"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

type createWalletRespBody struct {
	TelegramID    int64  `json:"telegram_id"`
	WalletAddress string `json:"wallet_address"`
}

func (cfg *apiConfig) handlerGenericStart(ctx context.Context, b *bot.Bot, update *models.Update) {
	telegramID := update.Message.From.ID
	isUser, err := cfg.DB.IsExistingUser(ctx, telegramID)
	if err != nil {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "Something went wrong, please try again",
		})
		log.Println(err.Error())
		log.Println(1)
		return
	}
	found := strings.Contains(update.Message.Text, "_ca_")
	if isUser && found {
		tokenAddress := strings.Split(update.Message.Text, "_ca_")[1][0:42]
		inputs := sharedTokenAddressFuncInputs{
			telegramId:   telegramID,
			chatId:       update.Message.Chat.ID,
			tokenAddress: tokenAddress,
		}
		b.DeleteMessage(ctx, &bot.DeleteMessageParams{
			ChatID:    update.Message.Chat.ID,
			MessageID: update.Message.ID,
		})
		cfg.handleSharedTokenAddress(ctx, b, inputs)
		return
	}
	if isUser {
		cfg.handlerHome(ctx, b, update)
		return
	}
	cfg.mu.Lock()
	cfg.intSeqMap[chatID(update.Message.Chat.ID)] = &interactionSequence{
		funcSlice:   nil,
		retValues:   []string{update.Message.Text},
		createdAt:   time.Now(),
		nextFuncIdx: 0,
	}
	cfg.mu.Unlock()
	cfg.handlerSelectMode(ctx, b, update)
}

func generateReferralCode() string {
	const alphabet = "abcdefghijklmnopqrstuvwxyz"
	builder := new(strings.Builder)
	for i := 0; i < 8; i++ {
		builder.WriteByte(alphabet[rand.Intn(26)])
	}
	return builder.String()
}

func (cfg *apiConfig) createUserTx(ctx context.Context, createUserParams database.CreateUserParams, createUserSettingsParam database.CreateUserSettingsParams) error {
	tx, err := cfg.dbConn.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	qtx := cfg.DB.WithTx(tx)
	err = qtx.CreateUser(ctx, createUserParams)
	if err != nil {
		return err
	}
	err = qtx.CreateUserSettings(ctx, createUserSettingsParam)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (cfg *apiConfig) handlerSelectMode(ctx context.Context, b *bot.Bot, update *models.Update) {
	welcomeMessage := fmt.Sprintf("Hey there %s, I’m Blockie \\- your personal trading buddy\\.\nWelcome to BlockBot, Monad’s most powerful Telegram trading bot\\. Built with speed, security, and *YOU* in mind\\.\n\nBefore we blast off, let's pick your style:\n\n🛡 *Standard Mode* – For traders who like things smooth and steady\\.\n\n⚡ *Degen Mode* – For maniacs chasing milliseconds execution and meme magic\\.\n\n\\(You can always change modes via the changemode command accesible via the menu button\\)\\.\n\nTap to choose and let’s get cooking\\.👇", strings.ReplaceAll(update.Message.From.Username, "_", "\\_"))
	keyboard := &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{
				{Text: "🛡 Standard Mode", CallbackData: "mode_standard"},
				{Text: "⚡ Degen Mode", CallbackData: "mode_degen"},
			},
		},
	}
	_, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      update.Message.Chat.ID,
		ParseMode:   models.ParseModeMarkdown,
		Text:        welcomeMessage,
		ReplyMarkup: keyboard,
	})
	if err != nil {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "Something went wrong, please try again",
		})
		log.Println(err.Error())
		log.Println(2)
		return
	}
}

func (cfg *apiConfig) modeViewCallback(ctx context.Context, b *bot.Bot, update *models.Update) {
	b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
		CallbackQueryID: update.CallbackQuery.ID,
		ShowAlert:       false,
	})
	switch update.CallbackQuery.Data {
	case "mode_standard":
		cfg.handlerBotModeStandard(ctx, b, update)
	case "mode_degen":
		cfg.handlerBotModeDegen(ctx, b, update)
	}
}

func (cfg *apiConfig) handlerStartParamCallback(ctx context.Context, b *bot.Bot, update *models.Update) {
	isBetaTester, err := cfg.DB.IsBetaTester(ctx, strings.ToLower(update.Message.From.Username))
	if err != nil {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "Something went wrong, please try again",
		})
		log.Println(err.Error())
		return
	}
	if !isBetaTester {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:    update.Message.Chat.ID,
			ParseMode: models.ParseModeMarkdown,
			Text:      "*Access Denied*\\.\n\nYou are not a whitelisted private Beta tester\\. Please come back when I'm open to the public\\.",
		})
		return
	}
	split := strings.Split(update.Message.Text, " ")
	if len(split) == 1 {
		cfg.handlerGenericStart(ctx, b, update)
		return
	}
	str := split[1]
	param := strings.Split(str, "_")
	switch param[0] {
	case "st":
		cfg.handleTokenShare(ctx, b, update)
	default:
		cfg.handlerGenericStart(ctx, b, update)
	}
}

func (cfg *apiConfig) handleTokenShare(ctx context.Context, b *bot.Bot, update *models.Update) {
	isUser, err := cfg.DB.IsExistingUser(ctx, update.Message.From.ID)
	if err != nil {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "Something went wrong, please try again",
		})
		log.Println(err.Error())
		log.Println(1)
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
	tokenAddress := strings.TrimPrefix(str, "st_")
	if ok, _ := regexp.MatchString(`^0x[0-9a-fA-F]{40}$`, tokenAddress); !ok {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "Invalid token address",
		})
		return
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
	b.DeleteMessage(ctx, &bot.DeleteMessageParams{
		ChatID:    update.Message.Chat.ID,
		MessageID: update.Message.ID,
	})
	text := bot.EscapeMarkdown(fmt.Sprintf("https://t.me/Monad_BlockBot?start=r_%s_ca_%s", refCode, tokenAddress))
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    update.Message.Chat.ID,
		ParseMode: models.ParseModeMarkdown,
		Text:      fmt.Sprintf("`%s`\n\n_Click the link above to copy it and share with friends\\!_", text),
	})
}
