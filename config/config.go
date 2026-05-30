package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	BotToken           string
	AllowedChatID      int64
	NotifyChatID       int64
	DatabaseURL        string
	Timezone           string
	Location           *time.Location
	TelegramAPIBaseURL string
}

func Load() (*Config, error) {
	cfg := &Config{
		BotToken:    os.Getenv("BOT_TOKEN"),
		DatabaseURL: os.Getenv("DATABASE_URL"),
		Timezone:    os.Getenv("TIMEZONE"),
	}

	if cfg.BotToken == "" {
		return nil, fmt.Errorf("BOT_TOKEN is required")
	}
	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}
	if cfg.Timezone == "" {
		cfg.Timezone = "UTC"
	}

	allowedRaw := os.Getenv("ALLOWED_CHAT_ID")
	if allowedRaw == "" {
		return nil, fmt.Errorf("ALLOWED_CHAT_ID is required")
	}
	allowed, err := strconv.ParseInt(allowedRaw, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid ALLOWED_CHAT_ID: %w", err)
	}
	cfg.AllowedChatID = allowed

	notifyRaw := os.Getenv("NOTIFY_CHAT_ID")
	if notifyRaw == "" {
		cfg.NotifyChatID = allowed
	} else {
		notify, err := strconv.ParseInt(notifyRaw, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid NOTIFY_CHAT_ID: %w", err)
		}
		cfg.NotifyChatID = notify
	}

	loc, err := time.LoadLocation(cfg.Timezone)
	if err != nil {
		return nil, fmt.Errorf("invalid TIMEZONE %q: %w", cfg.Timezone, err)
	}
	cfg.Location = loc

	if base := strings.TrimRight(os.Getenv("TELEGRAM_API_BASE_URL"), "/"); base != "" {
		cfg.TelegramAPIBaseURL = base
	}

	return cfg, nil
}
