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
	"github.com/jackc/pgx/v5/pgconn"
)

func (cfg *apiConfig) handlerBuyCommand(ctx context.Context, b *bot.Bot, msg *models.Message) {
	telegramID := msg.From.ID
	tokenIdentifier := msg.Text
	walletAddress, err := cfg.DB.GetWalletAddress(ctx, telegramID)
	if err != nil {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: msg.Chat.ID,
			Text:   "Something went wrong, please try again",
		})
		log.Println(err.Error())
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
	token := monorailRespBody[0]
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
	monPriceFormatted := strings.Replace(tokenPriceAsFloat.Text('f', 6), ".", "\\.", 1)
	compoundImpactFormatted := strings.Replace(compoundImpactAsFloat.Text('f', 3), ".", "\\.", 1)
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
	telegramId := msg.From.ID
	url := fmt.Sprintf("https://testnet-api.monorail.xyz/v1/portfolio/%s/value", walletAddress)
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
	totalPortfolioValue := monorailTotalPortfolioResp{}
	err = json.Unmarshal(body, &totalPortfolioValue)
	if err != nil {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: msg.Chat.ID,
			Text:   "Something went wrong, please try again",
		})
		log.Println(err.Error())
		return
	}
	getBalanceResp := getBalanceRespBody{}
	err = WalletServiceCall("GET", fmt.Sprintf("%s/v1/balance", cfg.bwsOrigin), cfg.bwsApiKey, ReqBody{TelegramID: telegramId}, &getBalanceResp)
	if err != nil {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: msg.Chat.ID,
			Text:   "Something went wrong, please try again",
		})
		log.Println(err.Error())
		return
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
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: msg.Chat.ID,
				Text:   "Something went wrong, please try again",
			})
			log.Println(err.Error())
			return
		}
	}
	monValueFormatted := "N/A"
	usdValueFormatted := "N/A"
	tokenAmount, _ := new(big.Float).SetString(token.Balance)
	if token.MonPerToken != "" {
		currPricePerToken, _ := new(big.Float).SetString(token.MonPerToken)
		usdPerToken, err := getTokenUSDPrice(currPricePerToken)
		if err != nil {
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: msg.Chat.ID,
				Text:   "Something went wrong, please try again",
			})
			log.Println(err.Error())
			return
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
	/*tokenAmount, _ := new(big.Float).SetString(token.Balance)
	if token.UsdPerToken != "" {
		if val, _ := strconv.ParseFloat(token.UsdPerToken, 64); val != 0 {
			usdPerToken, _ := new(big.Float).SetString(token.UsdPerToken)
			usdValue := usdPerToken.Mul(tokenAmount, usdPerToken)
			usdValueFormatted = strings.Replace(usdValue.Text(byte('f'), 2), ".", "\\.", 1)
		}
	}*/
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
			ChatID: msg.Chat.ID,
			Text:   "Something went wrong, please try again",
		})
		log.Println(err.Error())
		return
	}
	keyboard := &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{
				{Text: "Home 🏠︎", CallbackData: "quickView_home"},
				{Text: "Close ❌", CallbackData: "quickView_close"},
			}, {
				{Text: token.Symbol, CallbackData: "quickView_symbol"},
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
		ChatID:      msg.Chat.ID,
		ReplyMarkup: keyboard,
	})
	/*cfg.userBalancesMu.Lock()
	userBalances, ok := cfg.usersBalances[telegramID(telegramId)]
	if !ok {
		cfg.userBalancesMu.Unlock()
		return
	}
	userBalances.balances[userBalances.currBalanceIdx] = token
	userBalances.totalPortFolioValue = totalPortfolioValueFormatted
	userBalances.monBalance = monBalanceFormatted
	cfg.usersBalances[telegramID(telegramId)] = userBalances
	cfg.userBalancesMu.Unlock()*/
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
	balanceFormatted := strings.Replace(balanceAsFloat.Text('f', 6), ".", "\\.", 1)
	if !pricePresent || !impactPresent {
		keyboard = &models.InlineKeyboardMarkup{
			InlineKeyboard: [][]models.InlineKeyboardButton{
				{
					{Text: "Close ❌", CallbackData: "buy_close"},
				}, {
					{Text: token.Symbol, CallbackData: "buy_symbol"},
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
	monPriceFormatted := strings.Replace(tokenPriceAsFloat.Text('f', 6), ".", "\\.", 1)
	compoundImpactFormatted := strings.Replace(compoundImpactAsFloat.Text('f', 3), ".", "\\.", 1)
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
