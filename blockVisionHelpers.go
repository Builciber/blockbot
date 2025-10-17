package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
)

type blockVisionWalletTokensAPIResp struct {
	Code    int    `json:"code"`
	Reason  string `json:"reason"`
	Message string `json:"message"`
	Result  struct {
		Data []Token `json:"data"`
	} `json:"result"`
	Total    int     `json:"total"`
	UsdValue float64 `json:"usdValue"`
}

type Token struct {
	ContractAddress string `json:"contractAddress"`
	Name            string `json:"name"`
	Symbol          string `json:"symbol"`
	Price           string `json:"price"`
	Decimals        int    `json:"decimal"`
	Balance         string `json:"balance"`
	UsdValue        string `json:"usdValue"`
	MarketCap       string `json:"marketCap"`
	Liquidity       string `json:"liquidityInUsd"`
	Intervals       struct {
		Interval30Min  intervalData `json:"m30"`
		Interval1Hour  intervalData `json:"hour1"`
		Interval4Hour  intervalData `json:"hour4"`
		Interval24Hour intervalData `json:"hour24"`
	} `json:"market"`
	Tag string `json:"tag"`
}

type blockVisionTokenMarketDataResp struct {
	Code    int             `json:"code"`
	Reason  string          `json:"reason"`
	Message string          `json:"message"`
	Result  tokenMarketData `json:"result"`
}

type tokenMarketData struct {
	Price     string `json:"priceInUsd"`
	MarketCap string `json:"marketCap"`
	Liquidity string `json:"liquidityInUsd"`
	Intervals struct {
		Interval30Min  intervalData `json:"m30"`
		Interval1Hour  intervalData `json:"hour1"`
		Interval4Hour  intervalData `json:"hour4"`
		Interval24Hour intervalData `json:"hour24"`
	} `json:"market"`
	Tag string `json:"tag"`
}

type intervalData struct {
	PriceChange string `json:"priceChange"`
}

func (cfg *apiConfig) getWalletTokens(walletAddress string) ([]Token, float64, error) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", fmt.Sprintf("https://api.blockvision.org/v2/monad/account/tokens?address=%s", walletAddress), nil)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("x-api-key", cfg.blockVisionApiKey)
	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, 0, err
	}
	if resp.StatusCode > 299 {
		return nil, 0, fmt.Errorf("failed to fetch wallet tokens")
	}
	respBody := blockVisionWalletTokensAPIResp{}
	err = json.Unmarshal(body, &respBody)
	if err != nil {
		return nil, 0, err
	}
	if respBody.Code != 0 {
		return nil, 0, fmt.Errorf(fmt.Sprintf("nonzero code returned: %d", respBody.Code))
	}
	tokens := respBody.Result.Data
	portfolioValue := 0.0
	for i := range tokens {
		value, err := strconv.ParseFloat(tokens[i].UsdValue, 64)
		if err != nil {
			continue
		}
		portfolioValue += value
	}
	return respBody.Result.Data, portfolioValue, nil
}

func (cfg *apiConfig) getTokenMarketDataBlockVision(contractAddress string) (tokenMarketData, error) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", fmt.Sprintf("https://api.blockvision.org/v2/monad/token/market/data?token=%s", contractAddress), nil)
	if err != nil {
		return tokenMarketData{}, err
	}
	req.Header.Set("x-api-key", cfg.blockVisionApiKey)
	resp, err := client.Do(req)
	if err != nil {
		return tokenMarketData{}, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return tokenMarketData{}, err
	}
	if resp.StatusCode > 299 {
		return tokenMarketData{}, fmt.Errorf("failed to fetch token market data")
	}
	respBody := blockVisionTokenMarketDataResp{}
	err = json.Unmarshal(body, &respBody)
	if err != nil {
		return tokenMarketData{}, err
	}
	if respBody.Code != 0 {
		return tokenMarketData{}, fmt.Errorf(fmt.Sprintf("nonzero code returned: %d", respBody.Code))
	}
	return respBody.Result, nil
}
