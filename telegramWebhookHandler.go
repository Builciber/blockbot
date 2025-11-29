package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"

	"github.com/go-telegram/bot/models"
)

func (cfg *apiConfig) telegramWebhookHandler() http.HandlerFunc {
	return func(_ http.ResponseWriter, req *http.Request) {
		if cfg.telegramWebhookSecret != "" && req.Header.Get("X-Telegram-Bot-Api-Secret-Token") != cfg.telegramWebhookSecret {
			log.Println("invalid webhook secret token received from update")
			return
		}

		body, errReadBody := io.ReadAll(req.Body)
		if errReadBody != nil {
			log.Printf("error read request body, %s\n", errReadBody.Error())
			return
		}

		update := &models.Update{}

		errDecode := json.Unmarshal(body, update)
		if errDecode != nil {
			log.Printf("error decode request body, %s, %s\n", body, errDecode.Error())
			return
		}

		select {
		case <-req.Context().Done():
			log.Println("some updates lost, ctx done")
			return
		default:
		}

		select {
		case cfg.telegramUpdatesChan <- update:
		case <-req.Context().Done():
			log.Println("failed to send update, ctx done")
		}
	}
}
