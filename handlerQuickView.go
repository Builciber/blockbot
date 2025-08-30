package main

import (
	"context"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

func (cfg *apiConfig) quickViewCallback(ctx context.Context, b *bot.Bot, update *models.Update) {
	b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
		CallbackQueryID: update.CallbackQuery.ID,
		ShowAlert:       false,
	})
	switch update.CallbackQuery.Data {
	case "quckView_home":
		cfg.handlerHomeButton(ctx, b, update)
	case "quickView_close":
		cfg.handlerCloseButton(ctx, b, update)
	case "quickView_buyLeft":
		cfg.handlerBuyCommandBuyLeft(ctx, b, update)
	case "quickView_buyRight":
		cfg.handlerBuyCommandBuyRight(ctx, b, update)
	case "quickView_buyX":
		cfg.handlerBuyCommandBuyX(ctx, b, update)
	case "quickView_sellLeft":
		cfg.handlerQuickViewSellLeft(ctx, b, update)
	case "quickView_sellRight":
		cfg.handlerQuickViewSellRight(ctx, b, update)
	case "quickView_sellX":
		cfg.handlerQuickViewSellX(ctx, b, update)
	case "quickView_refresh":
		cfg.handlerQuickViewRefresh(ctx, b, update)
	}
}
