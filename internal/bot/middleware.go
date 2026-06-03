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
		chatID := chatIDOf(update)
		if chatID == 0 {
			return
		}
		if _, ok := allowed[chatID]; !ok {
			log.Warn("ignored update from foreign chat", "chat_id", chatID)
			return
		}
		next(ctx, update)
	}
}

func chatIDOf(u tgbotapi.Update) int64 {
	if u.Message != nil && u.Message.Chat != nil {
		return u.Message.Chat.ID
	}
	if u.CallbackQuery != nil && u.CallbackQuery.Message != nil && u.CallbackQuery.Message.Chat != nil {
		return u.CallbackQuery.Message.Chat.ID
	}
	return 0
}
