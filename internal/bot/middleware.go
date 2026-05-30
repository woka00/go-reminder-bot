package bot

import (
	"context"
	"log/slog"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type HandlerFunc func(ctx context.Context, update tgbotapi.Update)

func ChatFilter(allowedChatIDs []int64, log *slog.Logger, next HandlerFunc) HandlerFunc {
	allowed := make(map[int64]struct{}, len(allowedChatIDs))
	for _, id := range allowedChatIDs {
		allowed[id] = struct{}{}
	}
	return func(ctx context.Context, update tgbotapi.Update) {
		if update.Message == nil || update.Message.Chat == nil {
			return
		}
		if _, ok := allowed[update.Message.Chat.ID]; !ok {
			log.Warn("ignored message from foreign chat",
				"chat_id", update.Message.Chat.ID,
				"user_id", update.Message.From.ID,
			)
			return
		}
		next(ctx, update)
	}
}
