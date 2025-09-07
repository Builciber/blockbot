package main

import (
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strconv"
	"strings"

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

func getTokenUSDPrice(monPerToken *big.Float) (*big.Float, error) {
	url := "https://testnet-api.monorail.xyz/v1/symbol/MONUSD"
	res, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode > 299 {
		return nil, err
	}
	monPrice := MonUsdPrice{}
	err = json.Unmarshal(body, &monPrice)
	if err != nil {
		return nil, err
	}
	tokenPrice, ok := new(big.Float).SetString(monPrice.Price)
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
