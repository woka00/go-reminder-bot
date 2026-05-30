package bot

import (
	"context"
	"log/slog"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type HandlerFunc func(ctx context.Context, update tgbotapi.Update)

func ChatFilter(allowedChatID int64, log *slog.Logger, next HandlerFunc) HandlerFunc {
	return func(ctx context.Context, update tgbotapi.Update) {
		if update.Message == nil || update.Message.Chat == nil {
			return
		}
		if update.Message.Chat.ID != allowedChatID {
			log.Warn("ignored message from foreign chat",
				"chat_id", update.Message.Chat.ID,
				"user_id", update.Message.From.ID,
			)
			return
		}
		next(ctx, update)
	}
}
