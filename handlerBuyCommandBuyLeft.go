package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

func (cfg *apiConfig) handlerBuyCommandBuyLeft(ctx context.Context, b *bot.Bot, update *models.Update) {
	processingMsg, _ := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.CallbackQuery.Message.Message.Chat.ID,
		Text:   "Processing request...",
	})
	telegramId := update.CallbackQuery.From.ID
	msg := update.CallbackQuery.Message.Message
	splits := strings.Split(msg.Text, "|")
	withTokenAddress := strings.TrimPrefix(splits[2], " ")
	tokenAddress := withTokenAddress[0:42]
	tokenSymbol := splits[0]
	buySellButtons, err := cfg.DB.GetBuySellButtons(ctx, telegramId)
	if err != nil {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.CallbackQuery.Message.Message.Chat.ID,
			Text:   "Something went wrong, please try again",
		})
		log.Println(err.Error())
		return
	}
	amount := pgNumericToString(buySellButtons.BuyButtonLeft)
	executingMsg, _ := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.CallbackQuery.Message.Message.Chat.ID,
		Text:   "Executing purchase...",
	})
	tokenDecimals, err := cfg.getTokenDecimals(tokenAddress)
	if err != nil {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: msg.Chat.ID,
			Text:   "Something went wrong, please try again",
		})
		log.Println(err.Error())
		return
	}
	buyResult, err := cfg.handlerBuy(ctx, telegramId, amount, tokenAddress, tokenDecimals)
	if err != nil {
		errorMessage, found := strings.CutPrefix(err.Error(), "display to user: ")
		if found {
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: update.CallbackQuery.Message.Message.Chat.ID,
				Text:   errorMessage,
			})
			return
		}
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.CallbackQuery.Message.Message.Chat.ID,
			Text:   "Something went wrong, please try again",
		})
		log.Println(err.Error())
		return
	}
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    update.CallbackQuery.Message.Message.Chat.ID,
		ParseMode: models.ParseModeMarkdown,
		Text:      fmt.Sprintf("Purchase successful: Bought *%v %s* for *%v MON*\n[View on the explorer](https://monadvision.com/tx/%s)", strings.Replace(displayDecimal(buyResult.BoughtAmount, 3), ".", "\\.", 1), escapeMarkdown(tokenSymbol), strings.Replace(displayDecimal(amount, 3), ".", "\\.", 1), buyResult.TxHash),
	})
	b.DeleteMessages(ctx, &bot.DeleteMessagesParams{
		ChatID:     update.CallbackQuery.Message.Message.Chat.ID,
		MessageIDs: []int{processingMsg.ID, executingMsg.ID},
	})
	walletAddress, err := cfg.DB.GetWalletAddress(ctx, telegramId)
	if err != nil {
		return
	}
	inlineText, err := cfg.showBoughtToken(ctx, telegramId, tokenAddress, walletAddress)
	if err != nil {
		return
	}
	keyboard := genQuickViewKeyboard(buySellButtons, tokenSymbol)
	b.SendMessage(ctx, &bot.SendMessageParams{
		ParseMode:   models.ParseModeMarkdown,
		Text:        inlineText,
		ChatID:      msg.Chat.ID,
		ReplyMarkup: &keyboard,
	})
}
