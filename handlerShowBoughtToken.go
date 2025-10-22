package main

import (
	"context"
	"fmt"
	"math/big"
	"strings"

	"github.com/Builciber/blockbot/internal/database"
	"github.com/jackc/pgx/v5/pgconn"
)

func (cfg *apiConfig) showBoughtToken(ctx context.Context, telegramId int64, tokenAddress string, walletAddress string) (string, error) {
	token, err := cfg.getToken(tokenAddress, walletAddress)
	if err != nil {
		return "", err
	}
	getBalanceResp := getBalanceRespBody{}
	err = WalletServiceCall("GET", fmt.Sprintf("%s/v1/balance", cfg.bwsOrigin), cfg.bwsApiKey, ReqBody{TelegramID: telegramId}, &getBalanceResp)
	if err != nil {
		return "", err
	}
	monBalance := getBalanceResp.Balance
	monUsdPrice, err := getMONUSDPrice()
	if err != nil {
		return "", err
	}
	MonUsdPrice, _ := new(big.Float).SetString(monUsdPrice)
	pnlPercentFormatted := "N/A"
	pnlFormatted := "N/A"
	initialCostFormatted := "N/A"
	priceFormatted := "N/A"
	monValueFormatted := "N/A"
	usdValueFormatted := "N/A"
	position, err := cfg.DB.CallGetPositionFunc(ctx, database.CallGetPositionFuncParams{Traderid: telegramId, Tokenaddress: token.ContractAddress})
	if token.Price != "" && token.Price != "0" {
		tokenUsdPrice, _ := new(big.Float).SetString(token.Price)
		currPricePerToken := new(big.Float)
		currPricePerToken.Quo(tokenUsdPrice, MonUsdPrice) //price of token in MON
		tokenBalance, _ := new(big.Float).SetString(token.Balance)
		monValue := new(big.Float).Mul(currPricePerToken, tokenBalance)
		monValueFormatted = strings.Replace(formatFloat(monValue, 3), ".", "\\.", 1)
		usdValueFormatted = strings.Replace(displayDecimal(token.UsdValue, 2), ".", "\\.", 1)
		priceFormatted = strings.Replace(displayDecimal(token.Price, 4), ".", "\\.", 1)
		if err == nil {
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
			pnlPercentFormatted = formatPnl(formatFloat(pnlPercent, 2))
			pnlPercentFormatted = replacer.Replace(pnlPercentFormatted)
			pnlFormatted = formatPnl(formatFloat(pnl, 3))
			pnlFormatted = replacer.Replace(pnlFormatted)
			initialCostFormatted = strings.Replace(formatFloat(initialMonCost, 4), ".", "\\.", 1)
		}
	}
	if err != nil {
		if pgErr, ok := err.(*pgconn.PgError); !(ok && pgErr.Code == "P0002") { // if not a `no_data_found` PL/pgsql error
			return "", err
		}
	}
	tokenBalance, _ := new(big.Float).SetString(token.Balance)
	tokenBalanceFormatted := strings.Replace(formatFloat(tokenBalance, 4), ".", "\\.", 1)
	monBalanceAsFloat, _ := new(big.Float).SetString(monBalance)
	monBalanceFormatted := strings.Replace(formatFloat(monBalanceAsFloat, 4), ".", "\\.", 1)
	marketCapFormatted := strings.Replace(displayDecimal(token.MarketCap, 2), ".", "\\.", 1)
	liquidityFormatted := strings.Replace(displayDecimal(token.Liquidity, 4), ".", "\\.", 1)
	replacer := strings.NewReplacer(".", "\\.", "-", "\\-", "+", "\\+")
	interval30MinFormatted := replacer.Replace(formatPnl(displayDecimal(token.Intervals.Interval30Min.PriceChange, 2)))
	interval1HourFormatted := replacer.Replace(formatPnl(displayDecimal(token.Intervals.Interval1Hour.PriceChange, 2)))
	interval4HourFormatted := replacer.Replace(formatPnl(displayDecimal(token.Intervals.Interval4Hour.PriceChange, 2)))
	interval24HourFormatted := replacer.Replace(formatPnl(displayDecimal(token.Intervals.Interval24Hour.PriceChange, 2)))
	liquidityFieldKey := "Liquidity"
	liquidityFieldValue := "$" + liquidityFormatted
	launchpad := "None"
	if token.Tag == "Nadfun" {
		liquidityFieldKey = "TokensInBondingCurve"
		liquidityFieldValue = fmt.Sprintf("%s %s", liquidityFormatted, token.Symbol)
		launchpad = "Nadfun"
	}
	inlineText := fmt.Sprintf("*%s* \\| *%s* \\| `%s`\n\nPnL: *%s%% / %s MON*\nValue: *$%s / %s MON*\nMcap: *$%s*\nPrice: *$%s*\n%s: *%s*\nLaunchpad: *%s*\n30m: *%s%%*, 1h: *%s%%*, 4h: *%s%%*, 24h: *%s%%*\n\nInitial: *%s MON*\nToken Balance: *%s %s*\nWallet Balance: *%s MON*\n\n[*View Token on Explorer*](https://testnet.monadexplorer.com/token/%s) \\| [*Share Token*](https://t.me/Monad_BlockBot?start=st_%s)", escapeMarkdown(token.Symbol), escapeMarkdown(token.Name), token.ContractAddress, pnlPercentFormatted, pnlFormatted, usdValueFormatted, monValueFormatted, marketCapFormatted, priceFormatted, liquidityFieldKey, liquidityFieldValue, launchpad, interval30MinFormatted, interval1HourFormatted, interval4HourFormatted, interval24HourFormatted, initialCostFormatted, tokenBalanceFormatted, token.Symbol, monBalanceFormatted, token.ContractAddress, token.ContractAddress)
	return inlineText, nil
}
