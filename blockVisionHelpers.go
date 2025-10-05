package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type blockVisionWalletTokensAPIResp struct {
	Code    int    `json:"code"`
	Reason  string `json:"reason"`
	Message string `json:"message"`
	Result  struct {
		Data []walletToken `json:"data"`
	} `json:"result"`
	Total    int     `json:"total"`
	UsdValue float64 `json:"usdValue"`
}

type walletToken struct {
	ContractAddress string `json:"contractAddress"`
	Name            string `json:"name"`
	Symbol          string `json:"symbol"`
	Price           string `json:"price"`
	Decimal         int    `json:"decimal"`
	Balance         string `json:"balance"`
}

func (cfg *apiConfig) getWalletTokens(walletAddress string) ([]walletToken, float64, error) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", fmt.Sprintf("https://api.blockvision.org/v2/monad/account/tokens?address=%s", walletAddress), nil)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Content-Type", "application/json")
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
	return respBody.Result.Data, respBody.UsdValue, nil
}
