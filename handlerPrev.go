package main

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"strconv"
	"strings"

	"github.com/Builciber/blockbot/internal/database"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/jackc/pgx/v5/pgconn"
)

func (cfg *apiConfig) handlerPrev(ctx context.Context, b *bot.Bot, update *models.Update) {
	telegramId := update.CallbackQuery.From.ID
	cfg.userBalancesMu.RLock()
	userBalances, ok := cfg.usersBalances[telegramID(telegramId)]
	cfg.userBalancesMu.RUnlock()
	if !ok {
		return
	}
	if userBalances.currBalanceIdx == 0 {
		return
	}
	token := userBalances.balances[userBalances.currBalanceIdx-1]
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
	inlineText := fmt.Sprintf("*%s* \\| *%s* \\| `%s`\n\nPnL: *%s%% / %s MON*\nValue: *$%s / %s* MON\nPrice: *%s MON* \n\nInitial: *%s MON*\nToken Balance: *%s %s*\nWallet Balance: *%s MON*\nTotal Portfolio Value: *$%s*\n\n[*View Token on Explorer*](https://testnet.monadexplorer.com/token/%s)", token.Symbol, token.Name, token.Address, pnlPercentFormatted, pnlFormatted, usdValueFormatted, monValueFormatted, priceFormatted, initialCostFormatted, tokenBalanceFormatted, token.Symbol, userBalances.monBalance, userBalances.totalPortFolioValue, token.Address)
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
	b.EditMessageText(ctx, &bot.EditMessageTextParams{
		MessageID:   update.CallbackQuery.Message.Message.ID,
		ParseMode:   models.ParseModeMarkdown,
		Text:        inlineText,
		ChatID:      update.CallbackQuery.Message.Message.Chat.ID,
		ReplyMarkup: keyboard,
	})
	cfg.userBalancesMu.Lock()
	ub := cfg.usersBalances[telegramID(telegramId)]
	ub.currBalanceIdx -= 1
	cfg.userBalancesMu.Unlock()
}
