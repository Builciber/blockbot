package main

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/Builciber/blockbot/internal/database"
)

func (cfg *apiConfig) handlerShowBoughtTokenBrief(token *Token, position database.CallGetPositionFuncRow, monUsdPrice string, serial int, builder *strings.Builder) {
	MonUsdPrice, _ := new(big.Float).SetString(monUsdPrice)
	pnlPercentFormatted := "N/A"
	pnlFormatted := "N/A"
	priceFormatted := "N/A"
	monValueFormatted := "N/A"
	usdValueFormatted := "N/A"
	//position, err := cfg.DB.CallGetPositionFunc(ctx, database.CallGetPositionFuncParams{Traderid: telegramId, Tokenaddress: token.ContractAddress})
	if token.Price != "" && token.Price != "0" {
		tokenUsdPrice, _ := new(big.Float).SetString(token.Price)
		currPricePerToken := new(big.Float)
		currPricePerToken.Quo(tokenUsdPrice, MonUsdPrice) //price of token in MON
		tokenBalance, _ := new(big.Float).SetString(token.Balance)
		monValue := new(big.Float).Mul(currPricePerToken, tokenBalance)
		monValueFormatted = strings.Replace(formatFloat(monValue, 3), ".", "\\.", 1)
		usdValueAsFloat, _ := new(big.Float).SetString(token.UsdValue)
		usdValueFormatted = strings.Replace(formatFloat(usdValueAsFloat, 2), ".", "\\.", 1)
		priceFormatted = strings.Replace(displayDecimal(token.Price, 4), ".", "\\.", 1)
		if position.Trader != 0 || position.TokenAddress != "" {
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
		}
	}
	marketCapFormatted := strings.Replace(displayDecimal(token.MarketCap, 2), ".", "\\.", 1)
	replacer := strings.NewReplacer(".", "\\.", "-", "\\-", "+", "\\+")
	interval30MinFormatted := replacer.Replace(formatPnl(displayDecimal(token.Intervals.Interval30Min.PriceChange, 2)))
	interval1HourFormatted := replacer.Replace(formatPnl(displayDecimal(token.Intervals.Interval1Hour.PriceChange, 2)))
	interval4HourFormatted := replacer.Replace(formatPnl(displayDecimal(token.Intervals.Interval4Hour.PriceChange, 2)))
	interval24HourFormatted := replacer.Replace(formatPnl(displayDecimal(token.Intervals.Interval24Hour.PriceChange, 2)))
	inlineText := fmt.Sprintf("%d\\. [*%s*](https://t.me/Monad_BlockBot?start=position_%d)\nPnL: *%s%% / %s MON*\nValue: *$%s / %s MON*\nMcap: *$%s*\nPrice: *$%s*\n30m: *%s%%*, 1h: *%s%%*, 4h: *%s%%*, 24h: *%s%%*\n\n", serial, escapeMarkdown(token.Symbol), serial-1, pnlPercentFormatted, pnlFormatted, usdValueFormatted, monValueFormatted, marketCapFormatted, priceFormatted, interval30MinFormatted, interval1HourFormatted, interval4HourFormatted, interval24HourFormatted)
	builder.WriteString(inlineText)
}
