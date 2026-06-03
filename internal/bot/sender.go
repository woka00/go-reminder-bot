package bot

import (
	"context"
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/wokaxd/reminder-bot/internal/models"
)

type TelegramSender struct {
	bot          *tgbotapi.BotAPI
	notifyChatID int64
}

func NewSender(bot *tgbotapi.BotAPI, notifyChatID int64) *TelegramSender {
	return &TelegramSender{bot: bot, notifyChatID: notifyChatID}
}

func (s *TelegramSender) SendReminder(_ context.Context, r *models.Reminder) error {
	msg := tgbotapi.NewMessage(s.notifyChatID, formatReminderNotification(r))
	msg.ReplyMarkup = notificationKeyboard(r.ID)
	if _, err := s.bot.Send(msg); err != nil {
		return fmt.Errorf("send reminder %d: %w", r.ID, err)
	}
	return nil
}
