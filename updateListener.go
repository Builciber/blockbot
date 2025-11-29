package main

import (
	"context"
	"log"

	"github.com/go-telegram/bot"
)

func (cfg *apiConfig) updatesListener(ctx context.Context, b *bot.Bot) {
	for {
		select {
		case <-ctx.Done():
			log.Println("Killed update listener")
			return
		case updates := <-cfg.telegramUpdatesChan:
			go b.ProcessUpdate(ctx, updates)
		}
	}
}
