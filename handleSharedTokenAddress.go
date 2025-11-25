package main

import (
	"context"
	"fmt"
	"log"
	"math/big"
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
	buySellButtons, err := cfg.DB.GetBuySellButtons(ctx, telegramID)
	if err != nil {
		return
	}
	compoundImpact, balance, err := cfg.getCompoundImpactAndMonBalance(telegramID, pgNumericToString(buySellButtons.BuyButtonRight), inputs.tokenAddress)
	if err != nil {
		return
	}
	token, err := cfg.getToken(inputs.tokenAddress, walletAddress)
	if err != nil {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatId,
			Text:   "Something went wrong, please try again",
		})
		log.Println(err.Error())
		return
	}
	if token.Balance != "0" {
		inlineText, err := cfg.handlerShowBoughttokenPM(ctx, telegramID, token, balance)
		if err != nil {
			return
		}
		keyboard := genQuickViewKeyboard(buySellButtons, token.Symbol)
		b.SendMessage(ctx, &bot.SendMessageParams{
			ParseMode:   models.ParseModeMarkdown,
			Text:        inlineText,
			ChatID:      chatId,
			ReplyMarkup: keyboard,
		})
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
	compoundImpactAsFloat, impactPresent := new(big.Float).SetString(compoundImpact)
	compoundImpactFormatted := "Unknown"
	if impactPresent {
		compoundImpactFormatted = strings.Replace(formatFloat(compoundImpactAsFloat, 3), ".", "\\.", 1)
	}
	balanceFormatted := strings.Replace(formatFloat(balanceAsFloat, 3), ".", "\\.", 1)
	if token.Price == "0" || token.Price == "" {
		keyboard = &models.InlineKeyboardMarkup{
			InlineKeyboard: [][]models.InlineKeyboardButton{
				{
					{Text: "Close", CallbackData: "buy_close"},
				},
			},
		}
		inlineText := fmt.Sprintf("*%s* \\| *%s* \\| *`%s`*\n\nPrice: *$0\\.00*\nPrice Impact \\(%s MON\\): *Unknown*\n\nWallet Balance: *%s MON*\n\n[View Token on Explorer](https://monadvision.com/token/%s)", token.Name, token.Symbol, token.ContractAddress, buyButtonRight, balanceFormatted, token.ContractAddress)
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:      chatId,
			ParseMode:   models.ParseModeMarkdown,
			Text:        inlineText,
			ReplyMarkup: keyboard,
		})
		return
	}
	priceFormatted := strings.Replace(displayDecimal(token.Price, 4), ".", "\\.", 1)
	marketCapFormatted := strings.Replace(displayDecimal(token.MarketCap, 2), ".", "\\.", 1)
	liquidityFormatted := strings.Replace(displayDecimal(token.Liquidity, 4), ".", "\\.", 1)
	liquidityFieldKey := "Liquidity"
	liquidityFieldValue := "$" + liquidityFormatted
	launchpad := "None"
	if token.Tag == "Nadfun" {
		liquidityFieldKey = "TokensInBondingCurve"
		liquidityFieldValue = fmt.Sprintf("%s %s", liquidityFormatted, token.Symbol)
		launchpad = "Nadfun"
	}
	inlineText := fmt.Sprintf("*%s* \\| *%s* \\| *`%s`*\n\nPrice: *$%s*\nMarket Cap: *$%s*\n%s: *%s*\nLaunchpad: *%s*\nPrice Impact \\(%s MON\\): *%s%%*\n\nWallet Balance: *%s MON*\n\n[View Token on Explorer](https://monadvision.com/token/%s) \\| [*Share Token*](https://t.me/Monad_BlockBot?start=st_%s)", bot.EscapeMarkdown(token.Name), bot.EscapeMarkdown(token.Symbol), token.ContractAddress, priceFormatted, marketCapFormatted, liquidityFieldKey, liquidityFieldValue, launchpad, buyButtonRight, compoundImpactFormatted, balanceFormatted, token.ContractAddress, token.ContractAddress)
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      chatId,
		ParseMode:   models.ParseModeMarkdown,
		Text:        inlineText,
		ReplyMarkup: keyboard,
	})
}
