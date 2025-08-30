package main

import (
	"context"
	"fmt"
	"time"

	"github.com/Builciber/blockbot/internal/database"
	"github.com/jackc/pgx/v5/pgtype"
)

type sellReqBody struct {
	TokenDecimals         uint8  `json:"token_decimals"`
	ReferrerFeePercent    int    `json:"referrer_fee_percent"`
	TelegramID            int64  `json:"telegram_id"`
	Slippage              int    `json:"slippage"`
	MaxPriceImpact        int    `json:"max_price_impact"`
	SellPercent           int    `json:"sell_percent"`
	ReferrerWalletAddress string `json:"referrer_wallet_address"`
	TokenAddress          string `json:"token_address"`
	PriorityFee           string `json:"priority_fee"`
}

type sellRespBody struct {
	Success          bool   `json:"success"`
	SaleTimestamp    int    `json:"sale_timestamp"`
	WalletAddress    string `json:"wallet_address"`
	TxHash           string `json:"tx_hash"`
	SalePrice        string `json:"sale_price"`
	SoldAmount       string `json:"sold_amount"`
	ReceivedMon      string `json:"received_mon"`
	ReferrerEarnings string `json:"referrer_earnings"`
}

func (cfg *apiConfig) handlerSell(ctx context.Context, telegramID int64, sellPercent int, tokenAddress string, tokenDecimals uint8) (sellRespBody, error) {
	sellData, err := cfg.DB.GetTradeData(ctx, telegramID)
	if err != nil {
		return sellRespBody{}, err
	}
	refAddress, ok := sellData.Referreraddress.(string)
	if !ok {
		return sellRespBody{}, fmt.Errorf("failed to cast referrer address as string")
	}
	priorityFee, ok := sellData.Priorityfee.(string)
	if !ok {
		return sellRespBody{}, fmt.Errorf("failed to cast priority fee as string")
	}
	reqBody := sellReqBody{
		TokenDecimals:         tokenDecimals,
		ReferrerFeePercent:    int(sellData.Referrerfeepercent.Int16),
		TelegramID:            telegramID,
		Slippage:              int(sellData.Sellslippage.Int16),
		MaxPriceImpact:        int(sellData.Maxpriceimpact.Int16),
		SellPercent:           sellPercent,
		ReferrerWalletAddress: refAddress,
		TokenAddress:          tokenAddress,
		PriorityFee:           priorityFee,
	}
	sellrespBody := sellRespBody{}
	err = WalletServiceCall("POST", fmt.Sprintf("%s/v1/sell", cfg.bwsOrigin), cfg.bwsApiKey, reqBody, &sellrespBody)
	if err != nil {
		return sellRespBody{}, err
	}
	fromAmount, err := stringToPGNumeric(sellrespBody.SoldAmount)
	if err != nil {
		return sellRespBody{}, err
	}
	toAmount, err := stringToPGNumeric(sellrespBody.ReceivedMon)
	if err != nil {
		return sellRespBody{}, err
	}
	createdAt := time.Now()
	insertSellTxParams := database.InsertTransactionParams{
		Trader:             pgtype.Int8{Int64: telegramID, Valid: true},
		WalletAddress:      sellrespBody.WalletAddress,
		FromToken:          tokenAddress,
		ToToken:            "0x0000000000000000000000000000000000000000",
		FromAmount:         fromAmount,
		ToAmount:           toAmount,
		TxHash:             sellrespBody.TxHash,
		TradeUnixTimestamp: pgtype.Timestamp{Time: time.Unix(int64(sellrespBody.SaleTimestamp), 0), Valid: true},
		WebhookEventID:     pgtype.Text{String: "polling", Valid: false},
		CreatedAt:          pgtype.Timestamp{Time: createdAt, Valid: true},
	}
	mutatePositionSellFuncParams := database.MutatePositionSellParams{
		Traderid:     telegramID,
		Tokenaddress: tokenAddress,
		TokenAmount:  fromAmount,
	}
	referrerEarnings, err := stringToPGNumeric(sellrespBody.ReferrerEarnings)
	if err != nil {
		return sellRespBody{}, err
	}
	updateReferrerEarningsParams := database.UpdateReferrerEarningsParams{
		Telegramid:       telegramID,
		Referrerearnings: referrerEarnings,
	}
	err = cfg.createSellTxTx(ctx, insertSellTxParams, mutatePositionSellFuncParams, updateReferrerEarningsParams)
	if err != nil {
		return sellRespBody{}, err
	}
	return sellrespBody, nil
}

func (cfg *apiConfig) createSellTxTx(ctx context.Context, insertBuyTxParams database.InsertTransactionParams, mutatePositionSellFuncParams database.MutatePositionSellParams, updateReferrerEarningsParams database.UpdateReferrerEarningsParams) error {
	tx, err := cfg.dbConn.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	qtx := cfg.DB.WithTx(tx)
	err = qtx.InsertTransaction(ctx, insertBuyTxParams)
	if err != nil {
		return err
	}
	err = qtx.MutatePositionSell(ctx, mutatePositionSellFuncParams)
	if err != nil {
		return err
	}
	err = qtx.UpdateReferrerEarnings(ctx, updateReferrerEarningsParams)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}
