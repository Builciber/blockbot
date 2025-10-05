package main

import (
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strconv"
	"strings"

	"github.com/Builciber/blockbot/internal/database"
	"github.com/go-telegram/bot/models"
	"github.com/jackc/pgx/v5/pgtype"
)

type MonUsdPrice struct {
	Price string `json:"price"`
}

func stringToPGNumericRetired(str string) (pgtype.Numeric, error) {
	var exp int32
	var intAmount int64
	var err error
	if strings.Contains(str, ".") {
		parts := strings.Split(str, ".")
		fractionalPart := parts[1]
		integralPart := parts[0]
		if val, _ := strconv.ParseInt(fractionalPart, 10, 64); val == 0 {
			exp = 0
			intAmount, _ = strconv.ParseInt(integralPart, 10, 64)
		}
		exp = int32(len(fractionalPart)) * -1
		intAmount, _ = strconv.ParseInt(strings.Join(parts, ""), 10, 64)
		return pgtype.Numeric{Int: big.NewInt(intAmount), Exp: exp, Valid: true}, nil
	}
	intAmount, err = strconv.ParseInt(str, 10, 64)
	if err != nil {
		return pgtype.Numeric{}, err
	}
	return pgtype.Numeric{Int: big.NewInt(intAmount), Exp: exp, Valid: true}, nil
}

func pgNumericToString(numeric pgtype.Numeric) string {
	b, _ := numeric.MarshalJSON()
	return string(b)
}

func stringToPGNumeric(str string) (pgtype.Numeric, error) {
	numeric := pgtype.Numeric{}
	err := numeric.UnmarshalJSON([]byte(str))
	if err != nil {
		return pgtype.Numeric{}, err
	}
	return numeric, nil
}

func getMONUSDPrice() (string, error) {
	url := "https://testnet-api.monorail.xyz/v1/symbol/MONUSD"
	res, err := http.Get(url)
	if err != nil {
		return "", err
	}
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	if res.StatusCode > 299 {
		return "", err
	}
	monPrice := MonUsdPrice{}
	err = json.Unmarshal(body, &monPrice)
	if err != nil {
		return "", err
	}
	return monPrice.Price, nil
}

func getTokenUSDPrice(monPerToken *big.Float) (*big.Float, error) {
	price, err := getMONUSDPrice()
	if err != nil {
		return nil, err
	}
	tokenPrice, ok := new(big.Float).SetString(price)
	if !ok {
		return nil, fmt.Errorf("mon price unavailable")
	}
	tokenPrice.Mul(tokenPrice, monPerToken)
	return tokenPrice, nil
}

func (cfg *apiConfig) findToken(tokenAddress, walletAddress string) (monorailBalancesResp, error) {
	url := fmt.Sprintf("https://testnet-api.monorail.xyz/v1/tokens?find=%s&address=%s", tokenAddress, walletAddress)
	res, err := http.Get(url)
	if err != nil {
		return monorailBalancesResp{}, err
	}
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return monorailBalancesResp{}, err
	}
	res.Body.Close()
	if res.StatusCode > 299 {
		return monorailBalancesResp{}, fmt.Errorf("non 2xx status code returned")
	}
	monorailRespBody := []monorailBalancesResp{}
	err = json.Unmarshal(body, &monorailRespBody)
	if err != nil {
		return monorailBalancesResp{}, err
	}
	return monorailRespBody[0], nil
}

func retrieveRefCode(link string) string {
	var refStartIndex int
	for i := range link {
		if link[i] == 'r' && (i+1 < len(link) && link[i+1] == '_') {
			refStartIndex = i + 2
			break
		}
	}
	if refStartIndex >= len(link) {
		return ""
	}
	refEndIndex := refStartIndex
	for i := refStartIndex + 1; i < len(link); i++ {
		if link[i] != '_' {
			refEndIndex++
		} else {
			break
		}
	}
	return link[refStartIndex : refEndIndex+1]
}

func genQuickViewKeyboard(buySellButtons database.GetBuySellButtonsRow, tokenSymbol string) models.InlineKeyboardMarkup {
	return models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{
				{Text: "Home 🏠︎", CallbackData: "quickView_home"},
				{Text: "Close ❌", CallbackData: "quickView_close"},
			}, {
				{Text: tokenSymbol, CallbackData: "quickView_symbol"},
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
}

func (cfg *apiConfig) fillMissingPriceData(tokensData []monorailBalancesResp) ([]monorailBalancesResp, error) {
	tokenAddresses := []string{}
	for _, data := range tokensData {
		pconf, _ := strconv.ParseInt(data.Pconf, 10, 64)
		if pconf < 90 {
			tokenAddresses = append(tokenAddresses, data.Address)
		}
	}
	if len(tokenAddresses) == 0 {
		return tokensData, nil
	}
	priceData := []getPricesResp{}
	err := WalletServiceCall("GET", fmt.Sprintf("%s/v1/prices", cfg.bwsOrigin), cfg.bwsApiKey, getPricesReq{TokenAddresses: tokenAddresses}, &priceData)
	if err != nil {
		return nil, err
	}
	addressToPriceData := make(map[string]getPricesResp)
	for _, data := range priceData {
		addressToPriceData[data.Address] = data
	}
	monPrice, err := getMONUSDPrice()
	if err != nil {
		return nil, err
	}
	for i := range tokensData {
		if val, ok := addressToPriceData[tokensData[i].Address]; ok && val.MonPerToken != "" {
			monPriceAsFloat, _ := new(big.Float).SetString(monPrice)
			tokenUsdPrice, _ := new(big.Float).SetString(val.MonPerToken)
			tokenUsdPrice.Mul(monPriceAsFloat, tokenUsdPrice)
			tokenMonValue, _ := new(big.Float).SetString(val.MonPerToken)
			balance, _ := new(big.Float).SetString(tokensData[i].Balance)
			tokenMonValue.Mul(balance, tokenMonValue)
			tokensData[i].MonPerToken = val.MonPerToken
			tokensData[i].MonValue = tokenMonValue.Text(byte('f'), -1)
			tokensData[i].UsdPerToken = tokenUsdPrice.Text(byte('f'), -1)
		}
	}
	return tokensData, nil
}

func getPortfolioWorth(walletAddress string) (string, error) {
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
		return "", fmt.Errorf("non 2XX status code returned")
	}
	totalPortfolioValue := monorailTotalPortfolioResp{}
	err = json.Unmarshal(body, &totalPortfolioValue)
	if err != nil {
		return "", err
	}
	return totalPortfolioValue.Value, nil
}

func getWalletTokenBalance(walletAddress string) ([]monorailBalancesResp, error) {
	url := fmt.Sprintf("https://testnet-api.monorail.xyz/v1/wallet/%s/balances", walletAddress)
	res, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	res.Body.Close()
	if res.StatusCode > 299 {
		return nil, fmt.Errorf("non 2XX status code received")
	}
	monorailRespBody := []monorailBalancesResp{}
	err = json.Unmarshal(body, &monorailRespBody)
	if err != nil {
		return nil, err
	}
	return monorailRespBody, nil
}

func displayDecimal(decimal string, precision int) (string, error) {
	float, ok := new(big.Float).SetString(decimal)
	if !ok {
		return "", fmt.Errorf("failed to parse decimal as big.Float")
	}
	scientificNotation := float.Text(byte('e'), precision)
	roundedDecimal := float.Text(byte('f'), precision)
	split := strings.Split(scientificNotation, "e")
	exponentString := split[1]
	exponent, err := strconv.ParseInt(exponentString, 10, 64)
	if err != nil {
		return "", nil
	}
	if exponent < 0 && exponent*-1 > 1 {
		var builder strings.Builder
		subscript := int(exponent)*-1 - 1
		mantissa := split[0]
		hex := fmt.Sprintf("208%d", subscript)
		hexAsInt, _ := strconv.ParseInt(hex, 16, 64)
		builder.WriteString("0.0")
		builder.WriteRune(rune(hexAsInt))
		for _, char := range mantissa {
			if char != '.' {
				builder.WriteRune(char)
			}
		}
		return builder.String(), nil
	}
	return roundedDecimal, nil
}

func formatFloat(float *big.Float, precision int) string {
	scientificNotation := float.Text(byte('e'), precision)
	roundedDecimal := float.Text(byte('f'), precision)
	split := strings.Split(scientificNotation, "e")
	exponentString := split[1]
	exponent, _ := strconv.ParseInt(exponentString, 10, 64)
	if exponent < 0 && exponent*-1-1 > 1 {
		var builder strings.Builder
		subscript := int(exponent)*-1 - 1
		mantissa := split[0]
		if mantissa[0] == '-' {
			builder.WriteString("-")
		}
		builder.WriteString("0.0")
		if subscript > 9 {
			str := strconv.Itoa(subscript)
			hex := fmt.Sprintf("208%s", string(str[0]))
			hexAsInt, _ := strconv.ParseInt(hex, 16, 64)
			builder.WriteRune(rune(hexAsInt))
			hex = fmt.Sprintf("208%s", string(str[1]))
			hexAsInt, _ = strconv.ParseInt(hex, 16, 64)
			builder.WriteRune(rune(hexAsInt))
		} else {
			hex := fmt.Sprintf("208%d", subscript)
			hexAsInt, _ := strconv.ParseInt(hex, 16, 64)
			builder.WriteRune(rune(hexAsInt))
		}
		for _, char := range mantissa {
			if char != '.' && char != '-' {
				builder.WriteRune(char)
			}
		}
		return builder.String()
	}
	return roundedDecimal
}
