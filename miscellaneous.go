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
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

type MonUsdPrice struct {
	Price string `json:"price"`
}

type NadfunTokenPrice struct {
	Price string `json:"price"`
}

type monorailBalancesResp struct {
	Address     string   `json:"address"`
	Balance     string   `json:"balance"`
	Categories  []string `json:"categories"`
	Decimals    int      `json:"decimals"`
	MonPerToken string   `json:"mon_per_token"`
	MonValue    string   `json:"mon_value"`
	Name        string   `json:"name"`
	Pconf       string   `json:"pconf"`
	Symbol      string   `json:"symbol"`
	UsdPerToken string   `json:"usd_per_token"`
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

/*func getMONUSDPrice() (string, error) {
	url := "https://api.monorail.xyz/v2/symbol/MONUSD"
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
		return "", fmt.Errorf("failed to fetch MONUSD price: %d status code returned", res.StatusCode)
	}
	monPrice := MonUsdPrice{}
	err = json.Unmarshal(body, &monPrice)
	if err != nil {
		return "", err
	}
	return monPrice.Price, nil
}*/

func (cfg *apiConfig) getMONUSDPrice() (string, error) {
	wrappedMonad := "0x3bd359C1119dA7Da1D913D1C4D2B7c461115433A"
	marketData, err := cfg.getTokenMarketDataBlockVision(wrappedMonad)
	if err != nil {
		return "", err
	}
	return marketData.Price, nil
}

func (cfg *apiConfig) findToken(tokenAddress, walletAddress string) (monorailBalancesResp, error) {
	url := fmt.Sprintf("https://api.monorail.xyz/v2/tokens?find=%s&address=%s", tokenAddress, walletAddress)
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
		return monorailBalancesResp{}, fmt.Errorf("failed to find token %s: %d status code returned", tokenAddress, res.StatusCode)
	}
	monorailRespBody := []monorailBalancesResp{}
	err = json.Unmarshal(body, &monorailRespBody)
	if err != nil {
		return monorailBalancesResp{}, err
	}
	if len(monorailRespBody) == 0 {
		return monorailBalancesResp{}, fmt.Errorf("token not found")
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

func (cfg *apiConfig) getNadfunTokenPrice(tokenAddress string) (string, error) {
	url := fmt.Sprintf("%s/trade/market/%s", cfg.nadfunApiOrigin, tokenAddress)
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
		if res.StatusCode == 500 {
			/*errorResp := ErrorResp{}
			err = json.Unmarshal(body, &errorResp)
			if err != nil {
				return "", err
			}
			if strings.Contains(errorResp.Error, "no rows") {
				return "", nil
			}*/
			return "", nil
		}
		return "", fmt.Errorf("failed to fetch Nadfun token price for %s: %d status code returned", tokenAddress, res.StatusCode)
	}
	marketInfo := nadFunMarketInfo{}
	err = json.Unmarshal(body, &marketInfo)
	if err != nil {
		return "", err
	}
	return marketInfo.MarketInfo.Price, nil
}

type nadFunMarketInfo struct {
	MarketInfo struct {
		Price string `json:"token_price"`
	} `json:"market_info"`
}
type tokenPrice struct {
	token string
	price string
}

func (cfg *apiConfig) getNadfunTokenPrices(tokenAddresses []string) (map[string]string, error) {
	priceChan := make(chan tokenPrice, len(tokenAddresses))
	tokenAddressToPrices := make(map[string]string)
	errChan := make(chan error)
	for _, tokenAddress := range tokenAddresses {
		go func(tokenAddress string, resultChan chan<- tokenPrice, errChan chan<- error) {
			price, err := cfg.getNadfunTokenPrice(tokenAddress)
			if err != nil {
				errChan <- err
				return
			}
			resultChan <- tokenPrice{
				token: tokenAddress,
				price: price,
			}
		}(tokenAddress, priceChan, errChan)
	}
	for i := 0; i < len(tokenAddresses); i++ {
		select {
		case result := <-priceChan:
			tokenAddressToPrices[result.token] = result.price
		case err := <-errChan:
			return nil, err
		}
	}
	return tokenAddressToPrices, nil
}

func (cfg *apiConfig) fillMissingPriceData(tokens []Token, currentPortValue float64) ([]Token, string, error) {
	tokenAddresses := []string{}
	addressToPrice := make(map[string]string)
	for _, token := range tokens {
		if token.Price == "0" || token.Price == "" {
			tokenAddresses = append(tokenAddresses, token.ContractAddress)
			addressToPrice[token.ContractAddress] = ""
		}
	}
	portfolioValue := big.NewFloat(currentPortValue)
	if len(tokenAddresses) == 0 {
		return tokens, portfolioValue.Text(byte('f'), 2), nil
	}
	addressToPrice, err := cfg.getNadfunTokenPrices(tokenAddresses)
	if err != nil {
		return nil, "", err
	}
	err = cfg.fillOnchainPrice(addressToPrice)
	if err != nil {
		return nil, "", err
	}
	for i := range tokens {
		if price := addressToPrice[tokens[i].ContractAddress]; price != "" {
			tokenUsdPrice, ok := new(big.Float).SetString(price)
			if !ok {
				return nil, "", fmt.Errorf("fillMissingPriceData: failed to parse token USD price as *big.Float: %s", price)
			}
			tokenUsdValue := new(big.Float)
			balance, ok := new(big.Float).SetString(tokens[i].Balance)
			if !ok {
				return nil, "", fmt.Errorf("fillMissingPriceData: failed to parse token balance as *big.Float: %s", price)
			}
			tokenUsdValue.Mul(balance, tokenUsdPrice)
			tokens[i].Price = tokenUsdPrice.Text(byte('f'), -1)
			tokens[i].UsdValue = tokenUsdValue.Text(byte('f'), -1)
			portfolioValue.Add(portfolioValue, tokenUsdValue)
		}
	}
	return tokens, portfolioValue.Text(byte('f'), 2), nil
}

func displayDecimal(decimal string, precision int) string {
	float, ok := new(big.Float).SetString(decimal)
	if !ok {
		return "0"
	}
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
	return abbreviateDecimal(roundedDecimal)
}

func formatFloat(float *big.Float, precision int) string {
	if float == nil {
		return ""
	}
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
	return abbreviateDecimal(roundedDecimal)
}

type nadfunTokenPriceChange struct {
	TimeFrame          string `json:"timeframe"`
	PriceChangePercent string `json:"price_change_percent"`
	//CurrentPrice       string `json:"current_price"`
}

type nadfunTokenPriceChangeResp struct {
	Metrics []struct {
		Timeframe    string  `json:"timeframe"`
		Percent      float64 `json:"percent"`
		Transactions struct {
			Buy   int `json:"buy"`
			Sell  int `json:"sell"`
			Total int `json:"total"`
		} `json:"transactions"`
		Volume struct {
			Buy   string `json:"buy"`
			Sell  string `json:"sell"`
			Total string `json:"total"`
		} `json:"volume"`
		Makers struct {
			Buy   int `json:"buy"`
			Sell  int `json:"sell"`
			Total int `json:"total"`
		} `json:"makers"`
	} `json:"metrics"`
}

func (cfg *apiConfig) getNadfunTokenPriceChange(tokenAddress, interval string) (nadfunTokenPriceChange, error) {
	url := fmt.Sprintf("%s/trade/metrics/%s?timeframes=%s", cfg.nadfunApiOrigin, tokenAddress, interval)
	res, err := http.Get(url)
	if err != nil {
		return nadfunTokenPriceChange{}, err
	}
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nadfunTokenPriceChange{}, err
	}
	res.Body.Close()
	if res.StatusCode > 299 {
		return nadfunTokenPriceChange{}, fmt.Errorf("failed to fetch Nadfun token prices changes for %s: %d status code returned", tokenAddress, res.StatusCode)
	}
	priceChange := nadfunTokenPriceChangeResp{}
	err = json.Unmarshal(body, &priceChange)
	if err != nil {
		return nadfunTokenPriceChange{}, err
	}
	percentAsString := strconv.FormatFloat(priceChange.Metrics[0].Percent, byte('f'), 2, 64)
	return nadfunTokenPriceChange{
		TimeFrame:          priceChange.Metrics[0].Timeframe,
		PriceChangePercent: percentAsString,
	}, nil
}

func (cfg *apiConfig) getNadfunTokenPriceChanges(tokenAddress string) (map[string]nadfunTokenPriceChange, error) {
	intervals := []string{"30", "60", "240", "1D"}
	priceChangeChan := make(chan nadfunTokenPriceChange, len(intervals))
	errChan := make(chan error)
	intervalToPriceChange := make(map[string]nadfunTokenPriceChange)
	for _, interval := range intervals {
		go func(tokenAddress string, resultChan chan<- nadfunTokenPriceChange, errChan chan<- error) {
			priceChange, err := cfg.getNadfunTokenPriceChange(tokenAddress, interval)
			if err != nil {
				errChan <- err
				return
			}
			resultChan <- priceChange
		}(tokenAddress, priceChangeChan, errChan)
	}
	for i := 0; i < len(intervals); i++ {
		select {
		case result := <-priceChangeChan:
			intervalToPriceChange[result.TimeFrame] = result
		case err := <-errChan:
			return nil, err
		}
	}
	return intervalToPriceChange, nil
}

type getOnchainDataReq struct {
	TokenAddress string `json:"token_address"`
	UserAddress  string `json:"user_address"`
}

type getOnchainDataResp struct {
	TokenAddress       string `json:"token_address"`
	TokenSupply        string `json:"token_supply"`
	TokenBalance       string `json:"token_balance"`
	MonBalance         string `json:"mon_balance"`
	TokenName          string `json:"token_name"`
	TokenSymbol        string `json:"token_symbol"`
	TokenDecimals      int    `json:"token_decimals"`
	AvailableBuyTokens string `json:"available_buy_tokens"`
	Price              string `json:"price"`
	Tag                string `json:"tag"`
}

func (cfg *apiConfig) getNadfunTokenMarketData(tokenAddress string) (tokenMarketData, error) {
	tokenPrice, err := cfg.getNadfunTokenPrice(tokenAddress)
	if err != nil {
		return tokenMarketData{}, err
	}
	onchainData := getOnchainDataResp{}
	zeroAddress := "0x0000000000000000000000000000000000000000"
	err = WalletServiceCall("GET", fmt.Sprintf("%s/v1/onchain_data", cfg.bwsOrigin), cfg.bwsApiKey, getOnchainDataReq{tokenAddress, zeroAddress}, &onchainData)
	if err != nil {
		return tokenMarketData{}, err
	}
	intervalToPriceChange, err := cfg.getNadfunTokenPriceChanges(tokenAddress)
	if err != nil {
		return tokenMarketData{}, err
	}
	tokenUsdPrice, ok := new(big.Float).SetString(tokenPrice)
	if !ok {
		return tokenMarketData{}, fmt.Errorf("getNadfunTokenMarketData: failed to parse token USD price *big.Float: %s", tokenPrice)
	}
	marketCap, ok := new(big.Float).SetString(onchainData.TokenSupply)
	if !ok {
		return tokenMarketData{}, fmt.Errorf("getNadfunTokenMarketData: failed to parse token supply as *big.Float: %s", onchainData.TokenSupply)
	}
	marketCap.Mul(tokenUsdPrice, marketCap)
	intervals := struct {
		Interval30Min  intervalData `json:"m30"`
		Interval1Hour  intervalData `json:"hour1"`
		Interval4Hour  intervalData `json:"hour4"`
		Interval24Hour intervalData `json:"hour24"`
	}{
		Interval30Min: intervalData{
			PriceChange: intervalToPriceChange["30"].PriceChangePercent,
		},
		Interval1Hour: intervalData{
			PriceChange: intervalToPriceChange["60"].PriceChangePercent,
		},
		Interval4Hour: intervalData{
			PriceChange: intervalToPriceChange["240"].PriceChangePercent,
		},
		Interval24Hour: intervalData{
			PriceChange: intervalToPriceChange["1D"].PriceChangePercent,
		},
	}
	marketData := tokenMarketData{
		Price:     tokenUsdPrice.Text(byte('f'), -1),
		MarketCap: marketCap.Text(byte('f'), -1),
		Intervals: intervals,
		Liquidity: onchainData.AvailableBuyTokens,
		Tag:       "Nadfun",
	}
	return marketData, nil
}

func (cfg *apiConfig) getSomethingTokenMarketData(tokenAddress string) (tokenMarketData, error) {
	onchainData := getOnchainDataResp{}
	zeroAddress := "0x0000000000000000000000000000000000000000"
	err := WalletServiceCall("GET", fmt.Sprintf("%s/v1/onchain_data", cfg.bwsOrigin), cfg.bwsApiKey, getOnchainDataReq{tokenAddress, zeroAddress}, &onchainData)
	if err != nil {
		return tokenMarketData{}, err
	}
	monUsdprice, err := cfg.getMONUSDPrice()
	if err != nil {
		return tokenMarketData{}, nil
	}
	monUsdPriceAsFloat, ok := new(big.Float).SetString(monUsdprice)
	if !ok {
		return tokenMarketData{}, fmt.Errorf("getMarketData: failed to parse MONUSD price *big.Float: %s", monUsdprice)
	}
	tokenUsdPrice, ok := new(big.Float).SetString(onchainData.Price)
	if !ok {
		return tokenMarketData{}, fmt.Errorf("getMarketData: failed to parse token price *big.Float: %s", tokenUsdPrice)
	}
	tokenUsdPrice.Mul(tokenUsdPrice, monUsdPriceAsFloat)
	marketCap, ok := new(big.Float).SetString(onchainData.TokenSupply)
	if !ok {
		return tokenMarketData{}, fmt.Errorf("getNadfunTokenMarketData: failed to parse token supply as *big.Float: %s", onchainData.TokenSupply)
	}
	marketCap.Mul(tokenUsdPrice, marketCap)
	intervals := struct {
		Interval30Min  intervalData `json:"m30"`
		Interval1Hour  intervalData `json:"hour1"`
		Interval4Hour  intervalData `json:"hour4"`
		Interval24Hour intervalData `json:"hour24"`
	}{
		Interval30Min: intervalData{
			PriceChange: "0",
		},
		Interval1Hour: intervalData{
			PriceChange: "0",
		},
		Interval4Hour: intervalData{
			PriceChange: "0",
		},
		Interval24Hour: intervalData{
			PriceChange: "0",
		},
	}
	marketData := tokenMarketData{
		Price:     tokenUsdPrice.Text(byte('f'), -1),
		MarketCap: marketCap.Text(byte('f'), -1),
		Intervals: intervals,
		Tag:       onchainData.Tag,
	}
	return marketData, nil
}

func (cfg *apiConfig) getPrintrTokenMarketData(tokenAddress string) (tokenMarketData, error) {
	onchainData := getOnchainDataResp{}
	zeroAddress := "0x0000000000000000000000000000000000000000"
	err := WalletServiceCall("GET", fmt.Sprintf("%s/v1/onchain_data", cfg.bwsOrigin), cfg.bwsApiKey, getOnchainDataReq{tokenAddress, zeroAddress}, &onchainData)
	if err != nil {
		return tokenMarketData{}, err
	}
	monUsdprice, err := cfg.getMONUSDPrice()
	if err != nil {
		return tokenMarketData{}, nil
	}
	monUsdPriceAsFloat, ok := new(big.Float).SetString(monUsdprice)
	if !ok {
		return tokenMarketData{}, fmt.Errorf("getMarketData: failed to parse MONUSD price *big.Float: %s", monUsdprice)
	}
	tokenUsdPrice, ok := new(big.Float).SetString(onchainData.Price)
	if !ok {
		return tokenMarketData{}, fmt.Errorf("getMarketData: failed to parse token price *big.Float: %s", tokenUsdPrice)
	}
	tokenUsdPrice.Mul(tokenUsdPrice, monUsdPriceAsFloat)
	marketCap, ok := new(big.Float).SetString(onchainData.TokenSupply)
	if !ok {
		return tokenMarketData{}, fmt.Errorf("getNadfunTokenMarketData: failed to parse token supply as *big.Float: %s", onchainData.TokenSupply)
	}
	marketCap.Mul(tokenUsdPrice, marketCap)
	intervals := struct {
		Interval30Min  intervalData `json:"m30"`
		Interval1Hour  intervalData `json:"hour1"`
		Interval4Hour  intervalData `json:"hour4"`
		Interval24Hour intervalData `json:"hour24"`
	}{
		Interval30Min: intervalData{
			PriceChange: "0",
		},
		Interval1Hour: intervalData{
			PriceChange: "0",
		},
		Interval4Hour: intervalData{
			PriceChange: "0",
		},
		Interval24Hour: intervalData{
			PriceChange: "0",
		},
	}
	marketData := tokenMarketData{
		Price:     tokenUsdPrice.Text(byte('f'), -1),
		MarketCap: marketCap.Text(byte('f'), -1),
		Intervals: intervals,
		Tag:       onchainData.Tag,
	}
	return marketData, nil
}

func (cfg *apiConfig) getMarketData(tokenAddress string) (tokenMarketData, error) {
	onchainData := getOnchainDataResp{}
	zeroAddress := "0x0000000000000000000000000000000000000000"
	err := WalletServiceCall("GET", fmt.Sprintf("%s/v1/onchain_data", cfg.bwsOrigin), cfg.bwsApiKey, getOnchainDataReq{tokenAddress, zeroAddress}, &onchainData)
	if err != nil {
		return tokenMarketData{}, err
	}
	if onchainData.Tag == "Nadfun" {
		marketData, err := cfg.getNadfunTokenMarketData(tokenAddress)
		if err != nil {
			return tokenMarketData{}, err
		}
		return marketData, nil
	}
	if onchainData.Tag == "Something" {
		marketData, err := cfg.getSomethingTokenMarketData(tokenAddress)
		if err != nil {
			return tokenMarketData{}, err
		}
		return marketData, nil
	}
	/*if onchainData.Tag == "Printr" {
		marketData, err := cfg.getPrintrTokenMarketData(tokenAddress)
		if err != nil {
			return tokenMarketData{}, err
		}
		return marketData, nil
	}*/
	marketData, err := cfg.getTokenMarketDataBlockVision(tokenAddress)
	if err != nil {
		return tokenMarketData{}, err
	}
	return marketData, nil
}

func (cfg *apiConfig) getToken(tokenAddress, walletAddress string) (Token, error) {
	onchainData := getOnchainDataResp{}
	err := WalletServiceCall("GET", fmt.Sprintf("%s/v1/onchain_data", cfg.bwsOrigin), cfg.bwsApiKey, getOnchainDataReq{tokenAddress, walletAddress}, &onchainData)
	if err != nil {
		return Token{}, err
	}
	var marketData tokenMarketData

	switch onchainData.Tag {
	case "Nadfun":
		marketData, err = cfg.getNadfunTokenMarketData(onchainData.TokenAddress)
		if err != nil {
			return Token{}, err
		}
	case "Something":
		marketData, err = cfg.getSomethingTokenMarketData(onchainData.TokenAddress)
		if err != nil {
			return Token{}, err
		}
	/*case "Printr":
	marketData, err = cfg.getPrintrTokenMarketData(onchainData.TokenAddress)
	if err != nil {
		return Token{}, err
	}*/
	default:
		marketData, err = cfg.getTokenMarketDataBlockVision(onchainData.TokenAddress)
		if err != nil {
			return Token{}, err
		}
	}

	tokenBalance, ok := new(big.Float).SetString(onchainData.TokenBalance)
	if !ok {
		return Token{}, fmt.Errorf("getToken: failed to parse token balance *big.Float: %s", onchainData.TokenBalance)
	}
	usdValue, ok := new(big.Float).SetString(marketData.Price)
	if !ok {
		return Token{}, fmt.Errorf("getToken: failed to parse token price: %s", marketData.Price)
	}
	usdValue.Mul(tokenBalance, usdValue)
	token := Token{
		ContractAddress: onchainData.TokenAddress,
		Name:            onchainData.TokenName,
		Symbol:          onchainData.TokenSymbol,
		Price:           marketData.Price,
		Decimals:        onchainData.TokenDecimals,
		Balance:         onchainData.TokenBalance,
		UsdValue:        usdValue.Text(byte('f'), -1),
		MarketCap:       marketData.MarketCap,
		Liquidity:       marketData.Liquidity,
		Intervals:       marketData.Intervals,
		Tag:             onchainData.Tag,
	}
	return token, nil
}

func abbreviateDecimal(decimal string) string {
	integralPart := strings.Split(decimal, ".")[0]
	absoluteValue := integralPart
	if integralPart[0] == '-' {
		absoluteValue = integralPart[1:]
	}
	if absoluteValue == "0" {
		return decimal
	}
	if len(absoluteValue) < 4 {
		return decimal
	}
	units := map[int]string{
		3:  "K",
		6:  "M",
		9:  "B",
		12: "T",
		15: "Qa",
		18: "Qi",
	}
	powerOfTen := len(absoluteValue) - 1
	rem := powerOfTen % 3
	unit := units[powerOfTen-rem]
	float, ok := new(big.Float).SetString(integralPart)
	if !ok {
		return "0"
	}
	divisor, _ := new(big.Float).SetString(fmt.Sprintf("1e+%d", powerOfTen-rem))
	float.Quo(float, divisor)
	return float.Text(byte('f'), 2) + unit
}

func escapeMarkdown(str string) string {
	return bot.EscapeMarkdown(str)
}

type marketData struct {
	data         tokenMarketData
	tokenAddress string
}

func (cfg *apiConfig) getMarketDataMulti(tokenAddresses []string) (map[string]tokenMarketData, error) {
	marketDataChan := make(chan marketData, len(tokenAddresses))
	tokenAddressToMarketData := make(map[string]tokenMarketData)
	errChan := make(chan error)
	for _, tokenAddress := range tokenAddresses {
		go func(_tokenAddress string, resultChan chan<- marketData, errChan chan<- error) {
			data, err := cfg.getMarketData(_tokenAddress)
			if err != nil {
				errChan <- err
				return
			}
			resultChan <- marketData{
				data:         data,
				tokenAddress: _tokenAddress,
			}
		}(tokenAddress, marketDataChan, errChan)
	}
	for i := 0; i < len(tokenAddresses); i++ {
		select {
		case result := <-marketDataChan:
			tokenAddressToMarketData[result.tokenAddress] = result.data
		case err := <-errChan:
			return nil, err
		}
	}
	return tokenAddressToMarketData, nil
}

type position struct {
	data         database.CallGetPositionFuncRow
	tokenAddress string
}

func (cfg *apiConfig) CallGetPositionFuncMulti(ctx context.Context, tokenAddresses []string, telegramId int64) (map[string]database.CallGetPositionFuncRow, error) {
	positionsChan := make(chan position, len(tokenAddresses))
	tokenAddressToPosition := make(map[string]database.CallGetPositionFuncRow)
	errChan := make(chan error)
	for _, tokenAddress := range tokenAddresses {
		go func(_tokenAddress string, resultChan chan<- position, errChan chan<- error) {
			pos, err := cfg.DB.CallGetPositionFunc(ctx, database.CallGetPositionFuncParams{Traderid: telegramId, Tokenaddress: _tokenAddress})
			if err != nil {
				if pgErr, ok := err.(*pgconn.PgError); !(ok && pgErr.Code == "P0002") { // if not a `no_data_found` PL/pgsql error
					errChan <- err
					return
				}
			}
			resultChan <- position{
				data:         pos,
				tokenAddress: _tokenAddress,
			}
		}(tokenAddress, positionsChan, errChan)
	}
	for range tokenAddresses {
		select {
		case result := <-positionsChan:
			tokenAddressToPosition[result.tokenAddress] = result.data
		case err := <-errChan:
			return nil, err
		}
	}
	return tokenAddressToPosition, nil
}

func (cfg *apiConfig) fetchOverviewData(tokens []Token) error {
	listSize := min(len(tokens), 10)
	tokenAddresses := make([]string, listSize)
	for i := range listSize {
		tokenAddresses[i] = tokens[i].ContractAddress
	}
	tokensMarketData, err := cfg.getMarketDataMulti(tokenAddresses)
	if err != nil {
		return err
	}
	for i := range listSize {
		tokens[i].MarketCap = tokensMarketData[tokenAddresses[i]].MarketCap
		tokens[i].Liquidity = tokensMarketData[tokenAddresses[i]].Liquidity
		tokens[i].Intervals = tokensMarketData[tokenAddresses[i]].Intervals
		tokens[i].Tag = tokensMarketData[tokenAddresses[i]].Tag
	}
	return nil
}

func (cfg *apiConfig) constructOverviewString(ctx context.Context, tokens []Token, telegramId int64, startIndex int, netWorth string, monBalance string, portfolioSize int) (string, error) {
	listSize := min(len(tokens), 10)
	tokenAddresses := make([]string, listSize)
	for i := range listSize {
		tokenAddresses[i] = tokens[i].ContractAddress
	}
	positions, err := cfg.CallGetPositionFuncMulti(ctx, tokenAddresses, telegramId)
	if err != nil {
		return "", err
	}
	monUsdPrice, err := cfg.getMONUSDPrice()
	if err != nil {
		return "", err
	}
	builder := new(strings.Builder)
	builder.WriteString("*Positions Overview:*\n\n")
	for i := range listSize {
		position := positions[tokenAddresses[i]]
		serial := startIndex + i + 1
		err := cfg.handlerShowBoughtTokenBrief(&tokens[i], position, monUsdPrice, serial, builder)
		if err != nil {
			return "", err
		}
	}
	builder.WriteString(strings.ReplaceAll(fmt.Sprintf("Wallet Balance: *%s MON*\nPortfolio Size: *%d tokens*\nTotal Portfolio Value: *$%s*", displayDecimal(monBalance, 4), portfolioSize, displayDecimal(netWorth, 2)), ".", "\\."))
	return builder.String(), nil
}

func (cfg *apiConfig) fillOnchainPrice(tokenAddressToPrice map[string]string) error {
	onchainDataChan := make(chan getOnchainDataResp, len(tokenAddressToPrice))
	errChan := make(chan error)
	zeroAddress := "0x0000000000000000000000000000000000000000"
	tokenAddresses := make([]string, 0)
	for tokenAddress, price := range tokenAddressToPrice {
		if price == "" {
			tokenAddresses = append(tokenAddresses, tokenAddress)
		}
	}
	for _, tokenAddress := range tokenAddresses {
		go func(_tokenAddress string, resultChan chan<- getOnchainDataResp, errChan chan<- error) {
			onchainData := getOnchainDataResp{}
			err := WalletServiceCall("GET", fmt.Sprintf("%s/v1/onchain_data", cfg.bwsOrigin), cfg.bwsApiKey, getOnchainDataReq{tokenAddress, zeroAddress}, &onchainData)
			if err != nil {
				errChan <- err
				return
			}
			resultChan <- onchainData
		}(tokenAddress, onchainDataChan, errChan)
	}
	for i := 0; i < len(tokenAddresses); i++ {
		select {
		case result := <-onchainDataChan:
			tokenAddressToPrice[result.TokenAddress] = result.Price
		case err := <-errChan:
			return err
		}
	}
	return nil
}
