package main

import (
	"context"
	"database/sql"
	"internal/database"
	"log"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

type apiConfig struct {
	bwsApiKey string
	DB        *database.Queries
	intSeqMap map[chatID]*interactionSequence
	mu        *sync.RWMutex
}

type interactionSequence struct {
	funcSlice   []interactionHandler
	retValues   []string
	createdAt   time.Time
	nextFuncIdx uint8
}

type interactionHandler func(context.Context, *bot.Bot, *models.Message)

type chatID int64

func main() {
	godotenv.Load()
	dbURL := os.Getenv("CONN")
	apiKey := os.Getenv("BWS_API_KEY")
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatal(err.Error())
	}
	dbQueries := database.New(db)
	intSeqMap := make(map[chatID]*interactionSequence)
	mu := &sync.RWMutex{}
	cfg := &apiConfig{
		bwsApiKey: apiKey,
		DB:        dbQueries,
		intSeqMap: intSeqMap,
		mu:        mu,
	}
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	opts := []bot.Option{
		bot.WithDefaultHandler(cfg.handlerDefault),
		bot.WithWorkers(10),
	}

	b, err := bot.New("7744933903:AAG2BOVyU5xdx3laRCcK21DPSfgvUTQajlE", opts...)
	if err != nil {
		log.Fatal(err.Error())
	}

	b.RegisterHandler(bot.HandlerTypeMessageText, "/start", bot.MatchTypePrefix, cfg.handlerStart)
	b.RegisterHandler(bot.HandlerTypeMessageText, "/home", bot.MatchTypeExact, cfg.handlerHome)
	b.RegisterHandler(bot.HandlerTypeMessageText, "/wallet", bot.MatchTypeExact, cfg.handlerWalletCommand)
	b.RegisterHandler(bot.HandlerTypeMessageText, "/referrals", bot.MatchTypeExact, cfg.handlerReferralCommand)
	b.RegisterHandler(bot.HandlerTypeCallbackQueryData, "home_", bot.MatchTypePrefix, cfg.handlerHomeCallback)
	b.RegisterHandler(bot.HandlerTypeCallbackQueryData, "wallet_", bot.MatchTypePrefix, cfg.walletViewCallback)

	go cleaner(ctx, 3*time.Hour, cfg.intSeqMap, cfg.mu)

	b.Start(ctx)
}
