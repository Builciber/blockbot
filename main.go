package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"sync"
	"time"

	_ "net/http/pprof"

	"github.com/Builciber/blockbot/internal/database"
	"github.com/go-chi/chi/v5"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

type apiConfig struct {
	bwsApiKey      string
	botToken       string
	bwsOrigin      string
	monorailAppId  string
	DB             *database.Queries
	dbConn         *pgxpool.Pool
	intSeqMap      map[chatID]*interactionSequence
	usersBalances  map[telegramID]*userBalances
	mu             *sync.RWMutex
	userBalancesMu *sync.RWMutex
}

type interactionSequence struct {
	funcSlice   []interactionHandler
	retValues   []string
	createdAt   time.Time
	nextFuncIdx uint8
}

type interactionHandler func(context.Context, *bot.Bot, *models.Message)

type chatID int64

type telegramID int64

type userBalances struct {
	balances            []monorailBalancesResp
	currBalanceIdx      int
	totalPortFolioValue string
	monBalance          string
}

func main() {
	godotenv.Load()
	dbURL := os.Getenv("CONN")
	botToken := os.Getenv("BOT_TOKEN")
	apiKey := os.Getenv("BWS_API_KEY")
	bwsOrigin := os.Getenv("BWS_ORIGIN")
	monorailAppId := os.Getenv("MONORAIL_APP_ID")
	tgWebhookSecret := os.Getenv("TELEGRAM_WEBHOOK_SECRET")
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	db, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatal(err.Error())
	}
	dbQueries := database.New(db)
	intSeqMap := make(map[chatID]*interactionSequence)
	usersBalances := make(map[telegramID]*userBalances)

	mu := &sync.RWMutex{}
	usersBalancesMu := &sync.RWMutex{}
	cfg := &apiConfig{
		bwsApiKey:      apiKey,
		botToken:       botToken,
		bwsOrigin:      bwsOrigin,
		monorailAppId:  monorailAppId,
		DB:             dbQueries,
		dbConn:         db,
		intSeqMap:      intSeqMap,
		mu:             mu,
		usersBalances:  usersBalances,
		userBalancesMu: usersBalancesMu,
	}

	exists, err := cfg.DB.PrivateBetaTestersExists(ctx)
	if err != nil {
		log.Fatal("failed to insert beta testers into DB")
	}
	if !exists {
		err = cfg.insertTestersUsernames(ctx)
		if err != nil {
			log.Fatal("failed to insert beta testers into DB")
		}
	}

	opts := []bot.Option{
		bot.WithDefaultHandler(cfg.handlerDefault),
		bot.WithWebhookSecretToken(tgWebhookSecret),
	}

	b, err := bot.New(cfg.botToken, opts...)
	if err != nil {
		log.Fatal(err.Error())
	}

	b.RegisterHandler(bot.HandlerTypeMessageText, "/start", bot.MatchTypePrefix, cfg.handlerStartParamCallback)
	b.RegisterHandler(bot.HandlerTypeMessageText, "/home", bot.MatchTypeExact, cfg.handlerHome)
	b.RegisterHandler(bot.HandlerTypeMessageText, "/wallet", bot.MatchTypeExact, cfg.handlerWalletCommand)
	b.RegisterHandler(bot.HandlerTypeMessageText, "/referrals", bot.MatchTypeExact, cfg.handlerReferralCommand)
	b.RegisterHandler(bot.HandlerTypeMessageText, "/portfolio", bot.MatchTypeExact, cfg.handlerPortfolioCommand)
	b.RegisterHandler(bot.HandlerTypeMessageText, "/settings", bot.MatchTypeExact, cfg.handlerSettingsCommand)
	b.RegisterHandler(bot.HandlerTypeMessageText, "/changemode", bot.MatchTypeExact, cfg.handlerChangeMode)
	b.RegisterHandler(bot.HandlerTypeCallbackQueryData, "home_", bot.MatchTypePrefix, cfg.handlerHomeCallback)
	b.RegisterHandler(bot.HandlerTypeCallbackQueryData, "wallet_", bot.MatchTypePrefix, cfg.walletViewCallback)
	b.RegisterHandler(bot.HandlerTypeCallbackQueryData, "settings_", bot.MatchTypePrefix, cfg.settingsViewCallback)
	b.RegisterHandler(bot.HandlerTypeCallbackQueryData, "buy_", bot.MatchTypePrefix, cfg.buyCommandViewCallback)
	b.RegisterHandler(bot.HandlerTypeCallbackQueryData, "positions_", bot.MatchTypePrefix, cfg.positionsViewCallback)
	b.RegisterHandler(bot.HandlerTypeCallbackQueryData, "quickView_", bot.MatchTypePrefix, cfg.quickViewCallback)
	b.RegisterHandler(bot.HandlerTypeCallbackQueryData, "mode_", bot.MatchTypePrefix, cfg.modeViewCallback)
	b.RegisterHandler(bot.HandlerTypeCallbackQueryData, "change_", bot.MatchTypePrefix, cfg.changeModeViewCallback)

	go cleaner(ctx, 3*time.Hour, cfg.intSeqMap, cfg.mu)
	mux := chi.NewRouter()
	mux.Post("/webhooks/telegram", b.WebhookHandler())

	b.Start(ctx)

	/*server := http.Server{
		Addr:    "0.0.0.0:8080",
		Handler: mux,
	}
	log.Println("Started server on localhost at port 8080")
	server.ListenAndServe()*/
}
