package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strconv"
	"strings"

	"github.com/Builciber/blockbot/internal/database"
	"github.com/jackc/pgx/v5/pgconn"
)

func (cfg *apiConfig) showBoughtToken(ctx context.Context, telegramId int64, token monorailBalancesResp, walletAddress string) (string, error) {
	url := fmt.Sprintf("https://testnet-api.monorail.xyz/v1/portfolio/%s/value", walletAddress)
	res, err := http.Get(url)
	if err != nil {
		return "", err
	}
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return "", err
	}
	res.Body.Close()
	if res.StatusCode > 299 {
		return "", fmt.Errorf("non 2xx status code received")
	}
	totalPortfolioValue := monorailTotalPortfolioResp{}
	err = json.Unmarshal(body, &totalPortfolioValue)
	if err != nil {
		return "", err
	}
	getBalanceResp := getBalanceRespBody{}
	err = WalletServiceCall("GET", fmt.Sprintf("%s/v1/balance", cfg.bwsOrigin), cfg.bwsApiKey, ReqBody{TelegramID: telegramId}, &getBalanceResp)
	if err != nil {
		return "", err
	}
	monBalance := getBalanceResp.Balance
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
			return "", err
		}
	}
	monValueFormatted := "N/A"
	usdValueFormatted := "N/A"
	tokenAmount, _ := new(big.Float).SetString(token.Balance)
	if token.MonPerToken != "" {
		currPricePerToken, _ := new(big.Float).SetString(token.MonPerToken)
		usdPerToken, err := getTokenUSDPrice(currPricePerToken)
		if err != nil {
			return "", err
		}
		usdValue := new(big.Float).Mul(tokenAmount, usdPerToken)
		usdValueFormatted = strings.Replace(usdValue.Text(byte('f'), 2), ".", "\\.", 1)
	}
	if token.MonValue != "" {
		if val, _ := strconv.ParseFloat(token.MonValue, 64); val != 0 {
			monValue, _ := new(big.Float).SetString(token.MonValue)
			monValueFormatted = strings.Replace(monValue.Text(byte('f'), 4), ".", "\\.", 1)
		}
	}
	tokenBalance, _ := new(big.Float).SetString(token.Balance)
	tokenBalanceFormatted := strings.Replace(tokenBalance.Text(byte('f'), 4), ".", "\\.", 1)
	monBalanceAsFloat, _ := new(big.Float).SetString(monBalance)
	monBalanceFormatted := strings.Replace(monBalanceAsFloat.Text(byte('f'), 4), ".", "\\.", 1)
	totalPortfolioValueAsFloat, _ := new(big.Float).SetString(totalPortfolioValue.Value)
	totalPortfolioValueFormatted := strings.Replace(totalPortfolioValueAsFloat.Text(byte('f'), 4), ".", "\\.", 1)
	inlineText := fmt.Sprintf("*%s* \\| *%s* \\| `%s`\n\nPnL: *%s%% / %s MON*\nValue: *$%s / %s MON*\nPrice: *%s MON* \n\nInitial: *%s MON*\nToken Balance: *%s %s*\nWallet Balance: *%s MON*\nTotal Portfolio Value: *$%s*\n\n[*View Token on Explorer*](https://testnet.monadexplorer.com/token/%s) \\| [*Share Token*](https://t.me/Monad_TestBlockBot?start=st_%s)", token.Symbol, token.Name, token.Address, pnlPercentFormatted, pnlFormatted, usdValueFormatted, monValueFormatted, priceFormatted, initialCostFormatted, tokenBalanceFormatted, token.Symbol, monBalanceFormatted, totalPortfolioValueFormatted, token.Address, token.Address)
	return inlineText, nil
}
