package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/big"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/Builciber/blockbot/internal/database"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

func (cfg *apiConfig) handlerBuyCommand(ctx context.Context, b *bot.Bot, msg *models.Message) {
	telegramID := msg.From.ID
	tokenIdentifier := msg.Text
	params, err := cfg.DB.GetBuyCommandParams(ctx, telegramID)
	if err != nil {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: msg.Chat.ID,
			Text:   "Something went wrong, please try again",
		})
		log.Println(err.Error())
		return
	}
	walletAddress, ok := params.Walletaddress.(string)
	if !ok {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: msg.Chat.ID,
			Text:   "Something went wrong, please try again",
		})
		log.Println("Failed to parse wallet address as string")
		return
	}
	url := fmt.Sprintf("https://testnet-api.monorail.xyz/v1/tokens?find=%s&address=%s", tokenIdentifier, walletAddress)
	res, err := http.Get(url)
	if err != nil {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: msg.Chat.ID,
			Text:   "Something went wrong, please try again",
		})
		log.Println(err.Error())
		return
	}
	body, err := io.ReadAll(res.Body)
	if err != nil {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: msg.Chat.ID,
			Text:   "Something went wrong, please try again",
		})
		log.Println(err.Error())
		return
	}
	res.Body.Close()
	if res.StatusCode > 299 {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: msg.Chat.ID,
			Text:   "Something went wrong, please try again",
		})
		return
	}
	monorailRespBody := []monorailBalancesResp{}
	err = json.Unmarshal(body, &monorailRespBody)
	if err != nil {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: msg.Chat.ID,
			Text:   "Something went wrong, please try again",
		})
		log.Println(err.Error())
		log.Println(1)
		return
	}
	if len(monorailRespBody) == 0 {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: msg.Chat.ID,
			Text:   fmt.Sprintf("Token not found. Ensure that the address or ticker, %s is correct. To buy a token, send it's ticker or token address in chat", msg.Text),
			ReplyParameters: &models.ReplyParameters{
				MessageID:                msg.ID,
				AllowSendingWithoutReply: true,
			},
		})
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
		cfg.displayBoughtToken(ctx, b, msg, alreadyBoughtToken, walletAddress)
		return
	}
	filled, err := cfg.fillMissingPriceData(monorailRespBody[0:1])
	if err != nil {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: msg.Chat.ID,
			Text:   "Something went wrong, please try again",
		})
		log.Println(err.Error())
		return
	}
	token := filled[0]
	// Auto buy if enabled
	if ok, _ := regexp.MatchString(`^0x[0-9a-fA-F]{40}$`, tokenIdentifier); ok && params.Autobuyenabled.Bool {
		cfg.handlerAutoBuy(ctx, b, msg, params, token)
		return
	}
	compoundImpact, balance, err := cfg.getCompoundImpactAndMonBalance(telegramID, pgNumericToString(params.BuyButtonRight), token.Address)
	if err != nil {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: msg.Chat.ID,
			Text:   "Something went wrong, please try again",
		})
		log.Println(err.Error())
		return
	}
	buyButtonRight := strings.Replace(pgNumericToString(params.BuyButtonRight), ".", "\\.", 1)
	keyboard := &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{
				{Text: "Close ❌", CallbackData: "buy_close"},
			}, {
				{Text: token.Symbol, CallbackData: "buy_symbol"},
			}, {
				{Text: fmt.Sprintf("Buy %v MON", pgNumericToString(params.BuyButtonLeft)), CallbackData: "buy_buyLeft"},
				{Text: fmt.Sprintf("Buy %v MON", pgNumericToString(params.BuyButtonRight)), CallbackData: "buy_buyRight"},
				{Text: "Buy X MON", CallbackData: "buy_buyX"},
			}, {
				{Text: "Refresh ⟳", CallbackData: "buy_refresh"},
			},
		},
	}
	balanceAsFloat, _ := new(big.Float).SetString(balance)
	tokenPriceAsFloat, pricePresent := new(big.Float).SetString(token.MonPerToken)
	compoundImpactAsFloat, impactPresent := new(big.Float).SetString(compoundImpact)
	compoundImpactFormatted := "Unknown"
	if impactPresent {
		compoundImpactFormatted = strings.Replace(formatFloat(compoundImpactAsFloat, 3), ".", "\\.", 1)
	}
	balanceFormatted := strings.Replace(formatFloat(balanceAsFloat, 3), ".", "\\.", 1)
	if !pricePresent {
		keyboard = &models.InlineKeyboardMarkup{
			InlineKeyboard: [][]models.InlineKeyboardButton{
				{
					{Text: "Close", CallbackData: "buy_close"},
				},
			},
		}
		inlineText := fmt.Sprintf("*%s* \\| *%s* \\| *`%s`*\n\nPrice: *0\\.00 MON*\nPrice Impact \\(%s MON\\): *%s*\n\nWallet Balance: *%s MON*\n\n[View Token on Explorer](https://testnet.monadexplorer.com/token/%s)", token.Name, token.Symbol, token.Address, compoundImpactFormatted, buyButtonRight, balanceFormatted, token.Address)
		if ok, _ := regexp.MatchString(`^0x[0-9a-fA-F]{40}$`, tokenIdentifier); !ok {
			inlineText = inlineText + "\n\n*Proceed with caution: Multiple tokens can have the same names and symbols\\.*"
		}
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:      msg.Chat.ID,
			ParseMode:   models.ParseModeMarkdown,
			Text:        inlineText,
			ReplyMarkup: keyboard,
		})
		return
	}
	monPriceFormatted := strings.Replace(formatFloat(tokenPriceAsFloat, 4), ".", "\\.", 1)
	inlineText := fmt.Sprintf("*%s* \\| *%s* \\| *`%s`*\n\nPrice: *%s MON*\nPrice Impact \\(%s MON\\): *%s%%*\n\nWallet Balance: *%s MON*\n\n[View Token on Explorer](https://testnet.monadexplorer.com/token/%s)", token.Name, token.Symbol, token.Address, monPriceFormatted, buyButtonRight, compoundImpactFormatted, balanceFormatted, token.Address)
	if ok, _ := regexp.MatchString(`^0x[0-9a-fA-F]{40}$`, tokenIdentifier); !ok {
		inlineText = inlineText + "\n\n*Proceed with caution: Multiple tokens can have the same names and symbols\\.*"
	}
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      msg.Chat.ID,
		ParseMode:   models.ParseModeMarkdown,
		Text:        inlineText,
		ReplyMarkup: keyboard,
	})
}

func (cfg *apiConfig) displayBoughtToken(ctx context.Context, b *bot.Bot, msg *models.Message, token monorailBalancesResp, walletAddress string) {
	filled, err := cfg.fillMissingPriceData([]monorailBalancesResp{token})
	if err != nil {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: msg.Chat.ID,
			Text:   "Something went wrong, please try again",
		})
		log.Println(err.Error())
		return
	}
	token = filled[0]
	telegramId := msg.From.ID
	inlineText, err := cfg.showBoughtToken(ctx, telegramId, token, walletAddress)
	if err != nil {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: msg.Chat.ID,
			Text:   "Something went wrong, please try again",
		})
		log.Println(err.Error())
		return
	}
	buySellButtons, err := cfg.DB.GetBuySellButtons(ctx, telegramId)
	if err != nil {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: msg.Chat.ID,
			Text:   "Something went wrong, please try again",
		})
		log.Println(err.Error())
		return
	}
	keyboard := genQuickViewKeyboard(buySellButtons, token.Symbol)
	b.SendMessage(ctx, &bot.SendMessageParams{
		ParseMode:   models.ParseModeMarkdown,
		Text:        inlineText,
		ChatID:      msg.Chat.ID,
		ReplyMarkup: &keyboard,
	})
}

type monorailPathFinderResp struct {
	CompoundImpact string `json:"compound_impact"`
}

func (cfg *apiConfig) getCompoundImpactAndMonBalance(telegramId int64, forAmount, tokenAddress string) (string, string, error) {
	fromAddress := "0x0000000000000000000000000000000000000000"
	url := fmt.Sprintf("https://testnet-pathfinder-v2.monorail.xyz/v4/quote?amount=%s&from=%s&to=%s&source=%s", forAmount, fromAddress, tokenAddress, cfg.monorailAppId)
	res, err := http.Get(url)
	if err != nil {
		return "", "", err
	}
	body, err := io.ReadAll(res.Body)
	res.Body.Close()
	if res.StatusCode > 299 {
		return "", "", err
	}
	if err != nil {
		return "", "", err
	}
	monorailRespBody := monorailPathFinderResp{}
	err = json.Unmarshal(body, &monorailRespBody)
	if err != nil {
		return "", "", err
	}
	getBalanceResp := getBalanceRespBody{}
	err = WalletServiceCall("GET", fmt.Sprintf("%s/v1/balance", cfg.bwsOrigin), cfg.bwsApiKey, ReqBody{TelegramID: telegramId}, &getBalanceResp)
	if err != nil {
		return "", "", err
	}
	return monorailRespBody.CompoundImpact, getBalanceResp.Balance, nil
}

type monorailGetTokenResp struct {
	Address     string   `json:"address"`
	Name        string   `json:"name"`
	Symbol      string   `json:"symbol"`
	Decimals    int      `json:"decimals"`
	Categories  []string `json:"categories"`
	MonPerToken string   `json:"mon_per_token"`
	Pconf       string   `json:"pconf"`
	UsdPerToken string   `json:"usd_per_token"`
}

func (cfg *apiConfig) handlerBuyViewRefresh(ctx context.Context, b *bot.Bot, update *models.Update) {
	telegramID := update.CallbackQuery.From.ID
	if update.CallbackQuery.Message.Message == nil {
		return
	}
	msg := update.CallbackQuery.Message.Message
	splits := strings.Split(msg.Text, "|")
	withTokenAddress := strings.TrimPrefix(splits[2], " ")
	tokenAddress := withTokenAddress[0:42]
	url := fmt.Sprintf("https://testnet-api.monorail.xyz/v1/token/%s", tokenAddress)
	res, err := http.Get(url)
	if err != nil {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: msg.Chat.ID,
			Text:   "Something went wrong, please try again",
		})
		log.Println(err.Error())
		return
	}
	body, err := io.ReadAll(res.Body)
	if err != nil {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: msg.Chat.ID,
			Text:   "Something went wrong, please try again",
		})
		log.Println(err.Error())
		return
	}
	res.Body.Close()
	if res.StatusCode > 299 {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: msg.Chat.ID,
			Text:   "Something went wrong, please try again",
		})
		log.Println("non 2xx status code received: ", res.StatusCode)
		return
	}
	token := monorailGetTokenResp{}
	err = json.Unmarshal(body, &token)
	if err != nil {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: msg.Chat.ID,
			Text:   "Something went wrong, please try again",
		})
		log.Println(err.Error())
		return
	}
	buySellButtons, err := cfg.DB.GetBuySellButtons(ctx, telegramID)
	if err != nil {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: msg.Chat.ID,
			Text:   "Something went wrong, please try again",
		})
		log.Println(err.Error())
		return
	}
	compoundImpact, balance, err := cfg.getCompoundImpactAndMonBalance(telegramID, pgNumericToString(buySellButtons.BuyButtonRight), token.Address)
	if err != nil {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: msg.Chat.ID,
			Text:   "Something went wrong, please try again",
		})
		log.Println(err.Error())
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
	balanceFormatted := strings.Replace(formatFloat(balanceAsFloat, 4), ".", "\\.", 1)
	compoundImpactFormatted := "Unknown"
	if impactPresent {
		compoundImpactFormatted = strings.Replace(formatFloat(compoundImpactAsFloat, 3), ".", "\\.", 1)
	}
	if !pricePresent {
		keyboard = &models.InlineKeyboardMarkup{
			InlineKeyboard: [][]models.InlineKeyboardButton{
				{
					{Text: "Close ❌", CallbackData: "buy_close"},
				},
			},
		}
		inlineText := fmt.Sprintf("*%s* \\| *%s* \\| *`%s`*\n\nPrice: *0\\.00 MON*\nPrice Impact \\(%s MON\\): *Unknown*\n\nWallet Balance: *%s MON*\n\n[View Token on Explorer](https://testnet.monadexplorer.com/token/%s)", token.Name, token.Symbol, token.Address, buyButtonRight, balanceFormatted, token.Address)
		b.EditMessageText(ctx, &bot.EditMessageTextParams{
			MessageID:   msg.ID,
			ChatID:      msg.Chat.ID,
			ParseMode:   models.ParseModeMarkdown,
			Text:        inlineText,
			ReplyMarkup: keyboard,
		})
		return
	}
	monPriceFormatted := strings.Replace(formatFloat(tokenPriceAsFloat, 4), ".", "\\.", 1)
	inlineText := fmt.Sprintf("*%s* \\| *%s* \\| *`%s`*\n\nPrice: *%s MON*\nPrice Impact \\(%s MON\\): *%s%%*\n\nWallet Balance: *%s MON*\n\n[View Token on Explorer](https://testnet.monadexplorer.com/token/%s)", token.Name, token.Symbol, token.Address, monPriceFormatted, buyButtonRight, compoundImpactFormatted, balanceFormatted, token.Address)
	b.EditMessageText(ctx, &bot.EditMessageTextParams{
		MessageID:   msg.ID,
		ChatID:      msg.Chat.ID,
		ParseMode:   models.ParseModeMarkdown,
		Text:        inlineText,
		ReplyMarkup: keyboard,
	})
}

func (cfg *apiConfig) buyCommandViewCallback(ctx context.Context, b *bot.Bot, update *models.Update) {
	b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
		CallbackQueryID: update.CallbackQuery.ID,
		ShowAlert:       false,
	})
	switch update.CallbackQuery.Data {
	case "buy_close":
		cfg.handlerCloseButton(ctx, b, update)
	case "buy_buyLeft":
		cfg.handlerBuyCommandBuyLeft(ctx, b, update)
	case "buy_buyRight":
		cfg.handlerBuyCommandBuyRight(ctx, b, update)
	case "buy_buyX":
		cfg.handlerBuyCommandBuyX(ctx, b, update)
	case "buy_refresh":
		cfg.handlerBuyViewRefresh(ctx, b, update)
	}
}

func (cfg *apiConfig) getTokenDecimals(tokenAddress string) (uint8, error) {
	url := fmt.Sprintf("https://testnet-api.monorail.xyz/v1/token/%s", tokenAddress)
	res, err := http.Get(url)
	if err != nil {
		return 0, err
	}
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return 0, err
	}
	res.Body.Close()
	if res.StatusCode > 299 {
		return 0, err
	}
	token := monorailGetTokenResp{}
	err = json.Unmarshal(body, &token)
	if err != nil {
		return 0, err
	}
	return uint8(token.Decimals), nil
}

func (cfg *apiConfig) handlerAutoBuy(ctx context.Context, b *bot.Bot, msg *models.Message, params database.GetBuyCommandParamsRow, token monorailBalancesResp) {
	processingMsg, _ := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: msg.Chat.ID,
		Text:   "Processing request...",
	})
	telegramID := msg.From.ID
	tokenAddress := msg.Text
	walletAddress, ok := params.Walletaddress.(string)
	if !ok {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: msg.Chat.ID,
			Text:   "Something went wrong, please try again",
		})
		log.Println("Failed to parse wallet address as string")
		return
	}
	decimals, err := strconv.ParseUint(token.Decimals, 10, 64)
	if err != nil {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: msg.Chat.ID,
			Text:   "Something went wrong, please try again",
		})
		log.Println(err.Error())
		return
	}
	executingMsg, _ := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: msg.Chat.ID,
		Text:   "Executing purchase...",
	})
	buyResult, err := cfg.handlerBuy(ctx, telegramID, pgNumericToString(params.Autobuyamount), tokenAddress, uint8(decimals))
	if err != nil {
		errorMessage, found := strings.CutPrefix(err.Error(), "display to user: ")
		if found {
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: msg.Chat.ID,
				Text:   errorMessage,
			})
			return
		}
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: msg.Chat.ID,
			Text:   "Something went wrong, please try again",
		})
		log.Println(err.Error())
		return
	}
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    msg.Chat.ID,
		ParseMode: models.ParseModeMarkdown,
		Text:      fmt.Sprintf("Purchase successful: Bought *%v %s* for *%v MON*\n[View on the explorer](https://testnet.monadexplorer.com/tx/%s)", strings.Replace(buyResult.BoughtAmount, ".", "\\.", 1), token.Symbol, strings.Replace(pgNumericToString(params.Autobuyamount), ".", "\\.", 1), buyResult.TxHash),
	})
	inlineText, err := cfg.showBoughtToken(ctx, telegramID, token, walletAddress)
	if err != nil {
		return
	}
	buySellButtons := database.GetBuySellButtonsRow{
		BuyButtonLeft:   params.BuyButtonLeft,
		BuyButtonRight:  params.BuyButtonRight,
		SellButtonLeft:  params.SellButtonLeft.Int16,
		SellButtonRight: params.SellButtonRight.Int16,
	}
	keyboard := genQuickViewKeyboard(buySellButtons, token.Symbol)
	b.DeleteMessages(ctx, &bot.DeleteMessagesParams{
		ChatID:     msg.Chat.ID,
		MessageIDs: []int{processingMsg.ID, executingMsg.ID},
	})
	b.SendMessage(ctx, &bot.SendMessageParams{
		ParseMode:   models.ParseModeMarkdown,
		Text:        inlineText,
		ChatID:      msg.Chat.ID,
		ReplyMarkup: &keyboard,
	})
}
