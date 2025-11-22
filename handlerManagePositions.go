package main

import (
	"context"
	"log"
	"math/big"
	"slices"
	"strconv"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

type getPricesReq struct {
	TokenAddresses []string `json:"token_addresses"`
}

type getPricesResp struct {
	Address     string `json:"address"`
	MonPerToken string `json:"mon_per_token"`
	Decimals    int    `json:"decimals"`
	Protocol    int    `json:"protocol"`
}

func (cfg *apiConfig) handlerManagePositions(ctx context.Context, b *bot.Bot, update *models.Update) {
	telegramId := update.CallbackQuery.From.ID
	walletAddress, err := cfg.DB.GetWalletAddress(ctx, telegramId)
	if err != nil {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.CallbackQuery.Message.Message.Chat.ID,
			Text:   "Something went wrong, please try again",
		})
		log.Println(err.Error())
		return
	}
	tokens, netWorth, err := cfg.getWalletTokens(walletAddress)
	if err != nil {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.CallbackQuery.Message.Message.Chat.ID,
			Text:   "Something went wrong, please try again",
		})
		log.Println(err.Error())
		return
	}
	tokens, err = cfg.fillMissingPriceData(tokens)
	if err != nil {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.CallbackQuery.Message.Message.Chat.ID,
			Text:   "Something went wrong, please try again",
		})
		log.Println(err.Error())
		return
	}
	const zeroAddress = "0x0000000000000000000000000000000000000000"
	if len(tokens) == 0 || (len(tokens) == 1 && tokens[0].ContractAddress == zeroAddress) {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.CallbackQuery.Message.Message.Chat.ID,
			Text:   "You have no open positions. To buy a token and thus open a position, send it's ticker or token address in chat. Ensure you have enough MON for the purchase",
			ReplyParameters: &models.ReplyParameters{
				MessageID:                update.CallbackQuery.Message.Message.ID,
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
			ChatID: update.CallbackQuery.Message.Message.Chat.ID,
			Text:   "Something went wrong, please try again",
		})
		log.Println(err.Error())
		return
	}
	err = cfg.fetchOverviewData(viewablePositions)
	if err != nil {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.CallbackQuery.Message.Message.Chat.ID,
			Text:   "Something went wrong, please try again",
		})
		log.Println(err.Error())
		return
	}
	netWorthString := strconv.FormatFloat(netWorth, byte('f'), 2, 64)
	startIndex := 0
	inlineText, err := cfg.constructOverviewString(ctx, viewablePositions, telegramId, startIndex, netWorthString, monBalance, len(viewablePositions))
	if err != nil {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.CallbackQuery.Message.Message.Chat.ID,
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
		ChatID:      update.CallbackQuery.Message.Message.Chat.ID,
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
		totalPortFolioValue: netWorthString,
	}
	cfg.userBalancesMu.Unlock()
}

func (cfg *apiConfig) removeHiddenPositions(ctx context.Context, telegramID int64, positions []Token) ([]Token, error) {
	zeroAddress := "0x0000000000000000000000000000000000000000"
	hiddenPositions, err := cfg.DB.GetHiddenPositions(ctx, telegramID)
	if err != nil {
		return nil, err
	}
	hiddenPositionsMap := make(map[string]bool)
	for _, address := range hiddenPositions {
		hiddenPositionsMap[address] = true
	}
	viewablePositions := make([]Token, 0)
	for _, position := range positions {
		if !hiddenPositionsMap[position.ContractAddress] && position.ContractAddress != zeroAddress {
			viewablePositions = append(viewablePositions, position)
		}
	}
	return viewablePositions, nil
}

func (cfg *apiConfig) positionsViewCallback(ctx context.Context, b *bot.Bot, update *models.Update) {
	b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
		CallbackQueryID: update.CallbackQuery.ID,
		ShowAlert:       false,
	})
	switch update.CallbackQuery.Data {
	case "positions_home":
		cfg.handlerHomeButton(ctx, b, update)
	case "positions_close":
		cfg.handlerCloseButton(ctx, b, update)
	case "positions_prev":
		cfg.handlerPrev(ctx, b, update)
	case "positions_next":
		cfg.handlerNext(ctx, b, update)
	case "positions_buyLeft":
		cfg.handlerBuyLeft(ctx, b, update)
	case "positions_buyRight":
		cfg.handlerBuyRight(ctx, b, update)
	case "positions_buyX":
		cfg.handlerBuyX(ctx, b, update)
	case "positions_sellLeft":
		cfg.handlerSellLeft(ctx, b, update)
	case "positions_sellRight":
		cfg.handlerSellRight(ctx, b, update)
	case "positions_sellX":
		cfg.handlerSellX(ctx, b, update)
	case "positions_refresh":
		cfg.handlerPositionRefresh(ctx, b, update)
	}
}

func formatPnl(pnl string) string {
	if string(pnl[0]) != "-" {
		return "+" + pnl
	}
	return pnl
}
