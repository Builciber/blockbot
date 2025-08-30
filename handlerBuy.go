package main

import (
	"context"
	"fmt"
	"time"

	"github.com/Builciber/blockbot/internal/database"
	"github.com/jackc/pgx/v5/pgtype"
)

type buyReqBody struct {
	TokenDecimals         uint8  `json:"token_decimals"`
	TelegramID            int64  `json:"telegram_id"`
	Slippage              int    `json:"slippage"`
	MaxPriceImpact        int    `json:"max_price_impact"`
	ReferrerFeePercent    uint16 `json:"referrer_fee_percent"`
	ReferrerWalletAddress string `json:"referrer_wallet_address"`
	TokenAddress          string `json:"token_address"`
	Amount                string `json:"amount"`
	PriorityFee           string `json:"priority_fee"`
}

type buyRespBody struct {
	Success          bool   `json:"success"`
	BuyTimestamp     int    `json:"buy_timestamp"`
	WalletAddress    string `json:"wallet_address"`
	TxHash           string `json:"tx_hash"`
	BoughtAmount     string `json:"bought_amount"`
	BuyPrice         string `json:"buy_price"`
	SpentMon         string `json:"spent_mon"`
	ReferrerEarnings string `json:"referrer_earnings"`
}

func (cfg *apiConfig) handlerBuy(ctx context.Context, telegramID int64, amount string, tokenAddress string, tokenDecimals uint8) (buyRespBody, error) {
	buyData, err := cfg.DB.GetTradeData(ctx, telegramID)
	if err != nil {
		return buyRespBody{}, err
	}
	refAddress, ok := buyData.Referreraddress.(string)
	if !ok {
		return buyRespBody{}, fmt.Errorf("failed to cast referrer address as string")
	}
	priorityFee, ok := buyData.Priorityfee.(string)
	if !ok {
		return buyRespBody{}, fmt.Errorf("failed to cast priority fee as string")
	}
	reqBody := buyReqBody{
		TelegramID:            telegramID,
		TokenDecimals:         tokenDecimals,
		Slippage:              int(buyData.Buyslippage.Int16),
		MaxPriceImpact:        int(buyData.Maxpriceimpact.Int16),
		ReferrerFeePercent:    uint16(buyData.Referrerfeepercent.Int16),
		ReferrerWalletAddress: refAddress,
		TokenAddress:          tokenAddress,
		Amount:                amount,
		PriorityFee:           priorityFee,
	}
	buyrespBody := buyRespBody{}
	err = WalletServiceCall("POST", fmt.Sprintf("%s/v1/buy", cfg.bwsOrigin), cfg.bwsApiKey, reqBody, &buyrespBody)
	if err != nil {
		return buyRespBody{}, err
	}
	fromAmount, err := stringToPGNumeric(buyrespBody.SpentMon)
	if err != nil {
		return buyRespBody{}, err
	}
	toAmount, err := stringToPGNumeric(buyrespBody.BoughtAmount)
	if err != nil {
		return buyRespBody{}, err
	}
	createdAt := time.Now()
	insertBuyTxParams := database.InsertTransactionParams{
		Trader:             pgtype.Int8{Int64: telegramID, Valid: true},
		WalletAddress:      buyrespBody.WalletAddress,
		FromToken:          "0x0000000000000000000000000000000000000000",
		ToToken:            tokenAddress,
		FromAmount:         fromAmount,
		ToAmount:           toAmount,
		TxHash:             buyrespBody.TxHash,
		TradeUnixTimestamp: pgtype.Timestamp{Time: time.Unix(int64(buyrespBody.BuyTimestamp), 0), Valid: true},
		WebhookEventID:     pgtype.Text{String: "polling", Valid: false},
		CreatedAt:          pgtype.Timestamp{Time: createdAt, Valid: true},
	}
	mutatePositionFuncParams := database.MutatePositionParams{
		Traderid:     telegramID,
		Tokenaddress: tokenAddress,
		MonCost:      fromAmount,
		TokenAmount:  toAmount,
	}
	referrerEarnings, err := stringToPGNumeric(buyrespBody.ReferrerEarnings)
	if err != nil {
		return buyRespBody{}, err
	}
	updateReferrerEarningsParams := database.UpdateReferrerEarningsParams{
		Telegramid:       telegramID,
		Referrerearnings: referrerEarnings,
	}
	err = cfg.createBuyTxTx(ctx, insertBuyTxParams, mutatePositionFuncParams, updateReferrerEarningsParams)
	if err != nil {
		return buyRespBody{}, err
	}
	return buyrespBody, nil
}

func (cfg *apiConfig) createBuyTxTx(ctx context.Context, insertBuyTxParams database.InsertTransactionParams, mutatePositionFuncParams database.MutatePositionParams, updateReferrerEarningsParams database.UpdateReferrerEarningsParams) error {
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
	err = qtx.MutatePosition(ctx, mutatePositionFuncParams)
	if err != nil {
		return err
	}
	err = qtx.UpdateReferrerEarnings(ctx, updateReferrerEarningsParams)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}
