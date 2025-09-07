package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/big"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"github.com/Builciber/blockbot/internal/database"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/jackc/pgx/v5/pgconn"
)

type monorailBalancesResp struct {
	Address     string   `json:"address"`
	Balance     string   `json:"balance"`
	Categories  []string `json:"categories"`
	Decimals    string   `json:"decimals"`
	MonPerToken string   `json:"mon_per_token"`
	MonValue    string   `json:"mon_value"`
	Name        string   `json:"name"`
	Pconf       string   `json:"pconf"`
	Symbol      string   `json:"symbol"`
	UsdPerToken string   `json:"usd_per_token"`
}

type monorailTotalPortfolioResp struct {
	Value string `json:"value"`
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
	url := fmt.Sprintf("https://testnet-api.monorail.xyz/v1/wallet/%s/balances", walletAddress)
	res, err := http.Get(url)
	if err != nil {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.CallbackQuery.Message.Message.Chat.ID,
			Text:   "Something went wrong, please try again",
		})
		log.Println(err.Error())
		return
	}
	body, err := io.ReadAll(res.Body)
	if err != nil {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.CallbackQuery.Message.Message.Chat.ID,
			Text:   "Something went wrong, please try again",
		})
		log.Println(err.Error())
		return
	}
	res.Body.Close()
	if res.StatusCode > 299 {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.CallbackQuery.Message.Message.Chat.ID,
			Text:   "Something went wrong, please try again",
		})
		return
	}
	monorailRespBody := []monorailBalancesResp{}
	err = json.Unmarshal(body, &monorailRespBody)
	if err != nil {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.CallbackQuery.Message.Message.Chat.ID,
			Text:   "Something went wrong, please try again",
		})
		log.Println(err.Error())
		return
	}
	url = fmt.Sprintf("https://testnet-api.monorail.xyz/v1/portfolio/%s/value", walletAddress)
	res, err = http.Get(url)
	if err != nil {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.CallbackQuery.Message.Message.Chat.ID,
			Text:   "Something went wrong, please try again",
		})
		log.Println(err.Error())
		return
	}
	body, err = io.ReadAll(res.Body)
	if err != nil {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.CallbackQuery.Message.Message.Chat.ID,
			Text:   "Something went wrong, please try again",
		})
		log.Println(err.Error())
		return
	}
	res.Body.Close()
	if res.StatusCode > 299 {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.CallbackQuery.Message.Message.Chat.ID,
			Text:   "Something went wrong, please try again",
		})
		return
	}
	totalPortfolioValue := monorailTotalPortfolioResp{}
	err = json.Unmarshal(body, &totalPortfolioValue)
	if err != nil {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.CallbackQuery.Message.Message.Chat.ID,
			Text:   "Something went wrong, please try again",
		})
		log.Println(err.Error())
		return
	}
	const zeroAddress = "0x0000000000000000000000000000000000000000"
	if len(monorailRespBody) == 0 || (len(monorailRespBody) == 1 && monorailRespBody[0].Address == zeroAddress) {
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
	for _, token := range monorailRespBody {
		if token.Address == zeroAddress {
			monBalance = token.Balance
			break
		}
	}
	slices.SortFunc(monorailRespBody, func(tokenA, tokenB monorailBalancesResp) int {
		if tokenA.MonValue == "" || tokenB.MonValue == "" {
			return 0
		}
		tokenABalAsFloat, _ := new(big.Float).SetString(tokenA.MonValue)
		tokenBBalAsFloat, _ := new(big.Float).SetString(tokenB.MonValue)
		return tokenABalAsFloat.Cmp(tokenBBalAsFloat)
	})
	slices.Reverse(monorailRespBody)
	viewablePositions, err := cfg.removeHiddenPositions(ctx, telegramId, monorailRespBody)
	if err != nil {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.CallbackQuery.Message.Message.Chat.ID,
			Text:   "Something went wrong, please try again",
		})
		log.Println(err.Error())
		return
	}
	token := viewablePositions[0]
	pnlPercentFormatted := "N/A"
	pnlFormatted := "N/A"
	initialCostFormatted := "N/A"
	priceFormatted := "N/A"
	position, err := cfg.DB.CallGetPositionFunc(ctx, database.CallGetPositionFuncParams{Traderid: telegramId, Tokenaddress: token.Address})
	if err == nil && token.MonPerToken != "" {
		currPricePerToken, _ := new(big.Float).SetString(token.MonPerToken)
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
		replacer := strings.NewReplacer(".", "\\.", "-", "\\-", "+", "\\+")
		pnlPercentFormatted = formatPnl(pnlPercent.Text(byte('f'), 2))
		pnlPercentFormatted = replacer.Replace(pnlPercentFormatted)
		pnlFormatted = formatPnl(pnl.Text(byte('f'), 4))
		pnlFormatted = replacer.Replace(pnlFormatted)
		initialCostFormatted = strings.Replace(initialMonCost.Text(byte('f'), 4), ".", "\\.", 1)
		priceFormatted = strings.Replace(currPricePerToken.Text(byte('f'), 6), ".", "\\.", 1)
	}
	if err != nil {
		if pgErr, ok := err.(*pgconn.PgError); !(ok && pgErr.Code == "P0002") { // if not a `no_data_found` PL/pgsql error
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: update.CallbackQuery.Message.Message.Chat.ID,
				Text:   "Something went wrong, please try again",
			})
			log.Println(err.Error())
			return
		}
	}
	monValueFormatted := "N/A"
	usdValueFormatted := "N/A"
	if token.MonValue != "" {
		if val, _ := strconv.ParseFloat(token.MonValue, 64); val != 0 {
			monValue, _ := new(big.Float).SetString(token.MonValue)
			monValueFormatted = strings.Replace(monValue.Text(byte('f'), 4), ".", "\\.", 1)
		}
	}
	tokenAmount, _ := new(big.Float).SetString(token.Balance)
	if token.UsdPerToken != "" {
		if val, _ := strconv.ParseFloat(token.UsdPerToken, 64); val != 0 {
			usdPerToken, _ := new(big.Float).SetString(token.UsdPerToken)
			usdValue := usdPerToken.Mul(tokenAmount, usdPerToken)
			usdValueFormatted = strings.Replace(usdValue.Text(byte('f'), 2), ".", "\\.", 1)
		}
	}
	tokenBalance, _ := new(big.Float).SetString(token.Balance)
	tokenBalanceFormatted := strings.Replace(tokenBalance.Text(byte('f'), 4), ".", "\\.", 1)
	monBalanceAsFloat, _ := new(big.Float).SetString(monBalance)
	monBalanceFormatted := strings.Replace(monBalanceAsFloat.Text(byte('f'), 4), ".", "\\.", 1)
	totalPortfolioValueAsFloat, _ := new(big.Float).SetString(totalPortfolioValue.Value)
	totalPortfolioValueFormatted := strings.Replace(totalPortfolioValueAsFloat.Text(byte('f'), 4), ".", "\\.", 1)
	inlineText := fmt.Sprintf("*%s* \\| *%s* \\| `%s`\n\nPnL: *%s%% / %s MON*\nValue: *$%s / %s MON*\nPrice: *%s MON* \n\nInitial: *%s MON*\nToken Balance: *%s %s*\nWallet Balance: *%s MON*\nTotal Portfolio Value: *$%s*\n\n[*View Token on Explorer*](https://testnet.monadexplorer.com/token/%s) \\| [*Share Token*](https://t.me/Monad_BlockBot?start=st_%s)", token.Symbol, token.Name, token.Address, pnlPercentFormatted, pnlFormatted, usdValueFormatted, monValueFormatted, priceFormatted, initialCostFormatted, tokenBalanceFormatted, token.Symbol, monBalanceFormatted, totalPortfolioValueFormatted, token.Address, token.Address)
	buySellButtons, err := cfg.DB.GetBuySellButtons(ctx, telegramId)
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
				{Text: token.Symbol, CallbackData: "positions_symbol"},
			}, {
				{Text: "◀️ Prev", CallbackData: "positions_prev"},
				{Text: "Next ▶️", CallbackData: "positions_next"},
			}, {
				{Text: fmt.Sprintf("Buy %v MON", pgNumericToString(buySellButtons.BuyButtonLeft)), CallbackData: "positions_buyLeft"},
				{Text: fmt.Sprintf("Buy %v MON", pgNumericToString(buySellButtons.BuyButtonRight)), CallbackData: "positions_buyRight"},
				{Text: "Buy X MON", CallbackData: "positions_buyX"},
			}, {
				{Text: fmt.Sprintf("Sell %v%%", buySellButtons.SellButtonLeft), CallbackData: "positions_sellLeft"},
				{Text: fmt.Sprintf("Sell %v%%", buySellButtons.SellButtonRight), CallbackData: "positions_sellRight"},
				{Text: "Sell X %", CallbackData: "positions_sellX"},
			}, {
				{Text: "Refresh ⟳", CallbackData: "positions_refresh"},
			},
		},
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
		totalPortFolioValue: totalPortfolioValueFormatted,
		monBalance:          monBalanceFormatted,
	}
	cfg.userBalancesMu.Unlock()
}

func (cfg *apiConfig) removeHiddenPositions(ctx context.Context, telegramID int64, positions []monorailBalancesResp) ([]monorailBalancesResp, error) {
	zeroAddress := "0x0000000000000000000000000000000000000000"
	hiddenPositions, err := cfg.DB.GetHiddenPositions(ctx, telegramID)
	if err != nil {
		return nil, err
	}
	hiddenPositionsMap := make(map[string]bool)
	for _, address := range hiddenPositions {
		hiddenPositionsMap[address] = true
	}
	viewablePositions := make([]monorailBalancesResp, 0)
	for _, position := range positions {
		if !hiddenPositionsMap[position.Address] && position.Address != zeroAddress {
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
