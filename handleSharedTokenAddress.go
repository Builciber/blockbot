package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strings"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

type sharedTokenAddressFuncInputs struct {
	telegramId   int64
	chatId       int64
	tokenAddress string
}

func (cfg *apiConfig) handleSharedTokenAddress(ctx context.Context, b *bot.Bot, inputs sharedTokenAddressFuncInputs) {
	telegramID := inputs.telegramId
	chatId := inputs.chatId
	walletAddress, err := cfg.DB.GetWalletAddress(ctx, telegramID)
	if err != nil {
		return
	}
	url := fmt.Sprintf("https://testnet-api.monorail.xyz/v1/tokens?find=%s&address=%s", inputs.tokenAddress, walletAddress)
	res, err := http.Get(url)
	if err != nil {
		return
	}
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return
	}
	res.Body.Close()
	if res.StatusCode > 299 {
		return
	}
	monorailRespBody := []monorailBalancesResp{}
	err = json.Unmarshal(body, &monorailRespBody)
	if err != nil {
		return
	}
	if len(monorailRespBody) == 0 {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatId,
			Text:   fmt.Sprintf("Token not found. Ensure that the address, %s is correct. To buy a token, send it's ticker or token address in chat", inputs.tokenAddress),
		})
		return
	}
	buySellButtons, err := cfg.DB.GetBuySellButtons(ctx, telegramID)
	if err != nil {
		return
	}
	alreadyBoughtToken := monorailBalancesResp{}
	isBoughtToken := false
	for _, token := range monorailRespBody {
		if token.Balance != "" {
			tokenBalance, _ := new(big.Float).SetString(token.Balance)
			if tokenBalance.Cmp(big.NewFloat(0)) == 1 {
				alreadyBoughtToken = token
				isBoughtToken = true
				break
			}
		}
	}
	if isBoughtToken {
		inlineText, err := cfg.showBoughtToken(ctx, telegramID, alreadyBoughtToken, walletAddress)
		if err != nil {
			return
		}
		keyboard := &models.InlineKeyboardMarkup{
			InlineKeyboard: [][]models.InlineKeyboardButton{
				{
					{Text: "Home 🏠︎", CallbackData: "quickView_home"},
					{Text: "Close ❌", CallbackData: "quickView_close"},
				}, {
					{Text: alreadyBoughtToken.Symbol, CallbackData: "quickView_symbol"},
				}, {
					{Text: fmt.Sprintf("Buy %v MON", pgNumericToString(buySellButtons.BuyButtonLeft)), CallbackData: "quickView_buyLeft"},
					{Text: fmt.Sprintf("Buy %v MON", pgNumericToString(buySellButtons.BuyButtonRight)), CallbackData: "quickView_buyRight"},
					{Text: "Buy X MON", CallbackData: "quickView_buyX"},
				}, {
					{Text: fmt.Sprintf("Sell %v%%", buySellButtons.SellButtonLeft), CallbackData: "quickView_sellLeft"},
					{Text: fmt.Sprintf("Sell %v%%", buySellButtons.SellButtonRight), CallbackData: "quickView_sellRight"},
					{Text: "Sell X %", CallbackData: "quickView_sellX"},
				}, {
					{Text: "Refresh ⟳", CallbackData: "quickView_refresh"},
				},
			},
		}
		b.SendMessage(ctx, &bot.SendMessageParams{
			ParseMode:   models.ParseModeMarkdown,
			Text:        inlineText,
			ChatID:      chatId,
			ReplyMarkup: keyboard,
		})
		return
	}
	token := monorailRespBody[0]
	compoundImpact, balance, err := cfg.getCompoundImpactAndMonBalance(telegramID, pgNumericToString(buySellButtons.BuyButtonRight), token.Address)
	if err != nil {
		return
	}
	buyButtonRight := strings.Replace(pgNumericToString(buySellButtons.BuyButtonRight), ".", "\\.", 1)
	keyboard := &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{
				{Text: "Close ❌", CallbackData: "buy_close"},
			}, {
				{Text: token.Symbol, CallbackData: "buy_symbol"},
			}, {
				{Text: fmt.Sprintf("Buy %v MON", pgNumericToString(buySellButtons.BuyButtonLeft)), CallbackData: "buy_buyLeft"},
				{Text: fmt.Sprintf("Buy %v MON", pgNumericToString(buySellButtons.BuyButtonRight)), CallbackData: "buy_buyRight"},
				{Text: "Buy X MON", CallbackData: "buy_buyX"},
			}, {
				{Text: "Refresh ⟳", CallbackData: "buy_refresh"},
			},
		},
	}
	balanceAsFloat, _ := new(big.Float).SetString(balance)
	tokenPriceAsFloat, pricePresent := new(big.Float).SetString(token.MonPerToken)
	compoundImpactAsFloat, impactPresent := new(big.Float).SetString(compoundImpact)
	balanceFormatted := strings.Replace(balanceAsFloat.Text('f', 6), ".", "\\.", 1)
	if !pricePresent || !impactPresent {
		keyboard = &models.InlineKeyboardMarkup{
			InlineKeyboard: [][]models.InlineKeyboardButton{
				{
					{Text: "Close", CallbackData: "buy_close"},
				},
			},
		}
		inlineText := fmt.Sprintf("*%s* \\| *%s* \\| *`%s`*\n\nPrice: *0\\.00 MON*\nPrice Impact \\(%s MON\\): *Unknown*\n\nWallet Balance: *%s MON*\n\n[View Token on Explorer](https://testnet.monadexplorer.com/token/%s)", token.Name, token.Symbol, token.Address, buyButtonRight, balanceFormatted, token.Address)
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:      chatId,
			ParseMode:   models.ParseModeMarkdown,
			Text:        inlineText,
			ReplyMarkup: keyboard,
		})
		return
	}
	monPriceFormatted := strings.Replace(tokenPriceAsFloat.Text('f', 6), ".", "\\.", 1)
	compoundImpactFormatted := strings.Replace(compoundImpactAsFloat.Text('f', 3), ".", "\\.", 1)
	inlineText := fmt.Sprintf("*%s* \\| *%s* \\| *`%s`*\n\nPrice: *%s MON*\nPrice Impact \\(%s MON\\): *%s%%*\n\nWallet Balance: *%s MON*\n\n[View Token on Explorer](https://testnet.monadexplorer.com/token/%s)", token.Name, token.Symbol, token.Address, monPriceFormatted, buyButtonRight, compoundImpactFormatted, balanceFormatted, token.Address)
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      chatId,
		ParseMode:   models.ParseModeMarkdown,
		Text:        inlineText,
		ReplyMarkup: keyboard,
	})
}
