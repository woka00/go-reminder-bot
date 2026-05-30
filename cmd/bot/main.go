package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/wokaxd/reminder-bot/config"
	"github.com/wokaxd/reminder-bot/internal/bot"
	"github.com/wokaxd/reminder-bot/internal/scheduler"
	"github.com/wokaxd/reminder-bot/internal/service"
	"github.com/wokaxd/reminder-bot/internal/storage"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(log)

	if err := run(log); err != nil {
		log.Error("fatal", "err", err)
		os.Exit(1)
	}
}

func run(log *slog.Logger) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	migrationsPath := os.Getenv("MIGRATIONS_PATH")
	if migrationsPath == "" {
		migrationsPath = "migrations"
	}
	if err := storage.RunMigrations(cfg.DatabaseURL, migrationsPath); err != nil {
		return err
	}
	log.Info("migrations applied")

	rootCtx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	pool, err := storage.NewPool(rootCtx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()
	log.Info("postgres connected")

	store := storage.NewPostgresReminderStorage(pool)

	tgBot, err := newTelegramBot(cfg.BotToken, cfg.TelegramAPIBaseURL)
	if err != nil {
		return err
	}
	log.Info("telegram bot authorized",
		"username", tgBot.Self.UserName,
		"proxy", cfg.TelegramAPIBaseURL != "",
	)

	sender := bot.NewSender(tgBot, cfg.NotifyChatID)
	svc := service.New(store, sender, cfg.Location, log)

	handler := bot.NewHandler(tgBot, svc, cfg.NotifyChatID, cfg.Location, log)
	allowedChats := []int64{cfg.AllowedChatID}
	if cfg.NotifyChatID != cfg.AllowedChatID {
		allowedChats = append(allowedChats, cfg.NotifyChatID)
	}
	handle := bot.ChatFilter(allowedChats, log, handler.Handle)

	sch := scheduler.New(svc, log)

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		sch.Run(rootCtx)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		runUpdates(rootCtx, tgBot, handle, log)
	}()

	<-rootCtx.Done()
	log.Info("shutdown signal received")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()
	select {
	case <-done:
		log.Info("goroutines stopped")
	case <-shutdownCtx.Done():
		log.Warn("shutdown timeout, exiting")
	}
	return nil
}

func newTelegramBot(token, baseURL string) (*tgbotapi.BotAPI, error) {
	if baseURL == "" {
		return tgbotapi.NewBotAPI(token)
	}
	return tgbotapi.NewBotAPIWithAPIEndpoint(token, baseURL+"/bot%s/%s")
}

func runUpdates(ctx context.Context, tgBot *tgbotapi.BotAPI, handle bot.HandlerFunc, log *slog.Logger) {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 30
	updates := tgBot.GetUpdatesChan(u)

	log.Info("update loop started")
	for {
		select {
		case <-ctx.Done():
			tgBot.StopReceivingUpdates()
			log.Info("update loop stopped")
			return
		case update, ok := <-updates:
			if !ok {
				log.Info("updates channel closed")
				return
			}
			handle(ctx, update)
		}
	}
}
