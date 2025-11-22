package main

import (
	"context"
	"log"
	"math/big"
	"slices"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/jackc/pgx/v5"
)

func (cfg *apiConfig) handlerPortfolioCommand(ctx context.Context, b *bot.Bot, update *models.Update) {
	msg := update.Message
	telegramId := msg.From.ID
	walletAddress, err := cfg.DB.GetWalletAddress(ctx, telegramId)
	if err == pgx.ErrNoRows {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "Forbidden action",
		})
		log.Println(err.Error())
		return
	}
	if err != nil {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "Something went wrong, please try again",
		})
		log.Println(err.Error())
		return
	}
	tokens, _, err := cfg.getWalletTokens(walletAddress)
	if err != nil {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "Something went wrong, please try again",
		})
		log.Println(err.Error())
		return
	}
	tokens, netWorth, err := cfg.fillMissingPriceData(tokens)
	if err != nil {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "Something went wrong, please try again",
		})
		log.Println(err.Error())
		return
	}
	const zeroAddress = "0x0000000000000000000000000000000000000000"
	if len(tokens) == 0 || (len(tokens) == 1 && tokens[0].ContractAddress == zeroAddress) {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "You have no open positions. To buy a token and thus open a position, send it's ticker or token address in chat. Ensure you have enough MON for the purchase",
			ReplyParameters: &models.ReplyParameters{
				MessageID:                update.Message.ID,
				AllowSendingWithoutReply: true,
			},
		})
		return
	}
	var monBalance string
	for _, token := range tokens {
		if token.ContractAddress == zeroAddress {
			monBalance = token.Balance
			break
		}
	}
	slices.SortFunc(tokens, func(tokenA, tokenB Token) int {
		if tokenA.UsdValue == "" || tokenB.UsdValue == "" {
			return 0
		}
		tokenAValAsFloat, _ := new(big.Float).SetString(tokenA.UsdValue)
		tokenBValAsFloat, _ := new(big.Float).SetString(tokenB.UsdValue)
		return tokenAValAsFloat.Cmp(tokenBValAsFloat)
	})
	slices.Reverse(tokens)
	viewablePositions, err := cfg.removeHiddenPositions(ctx, telegramId, tokens)
	if err != nil {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "Something went wrong, please try again",
		})
		log.Println(err.Error())
		return
	}
	err = cfg.fetchOverviewData(viewablePositions)
	if err != nil {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "Something went wrong, please try again",
		})
		log.Println(err.Error())
		return
	}
	startIndex := 0
	inlineText, err := cfg.constructOverviewString(ctx, viewablePositions, telegramId, startIndex, netWorth, monBalance, len(viewablePositions))
	if err != nil {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "Something went wrong, please try again",
		})
		log.Println(err.Error())
		return
	}
	keyboard := &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{
				{Text: "Home 🏠︎", CallbackData: "positions_home"},
				{Text: "Close ❌", CallbackData: "positions_close"},
			}, {
				{Text: "◀️ Prev", CallbackData: "positions_prev"},
				{Text: "Next ▶️", CallbackData: "positions_next"},
			},
		},
	}
	if len(viewablePositions) <= 10 {
		keyboard = &models.InlineKeyboardMarkup{
			InlineKeyboard: [][]models.InlineKeyboardButton{
				{
					{Text: "Home 🏠︎", CallbackData: "positions_home"},
					{Text: "Close ❌", CallbackData: "positions_close"},
				},
			},
		}
	}
	_, err = b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      update.Message.Chat.ID,
		ParseMode:   models.ParseModeMarkdown,
		Text:        inlineText,
		ReplyMarkup: keyboard,
	})
	if err != nil {
		log.Println(err.Error())
		return
	}
	cfg.userBalancesMu.Lock()
	cfg.usersBalances[telegramID(telegramId)] = &userBalances{
		balances:            viewablePositions,
		currBalanceIdx:      0,
		steps:               1,
		monBalance:          monBalance,
		totalPortFolioValue: netWorth,
	}
	cfg.userBalancesMu.Unlock()
}
