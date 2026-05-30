package bot

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/wokaxd/reminder-bot/internal/models"
	"github.com/wokaxd/reminder-bot/internal/service"
)

const unknownInputMessage = "Не распознал задачу. Формат: «завтра в 14:00 сделать дз» или «13 мая закончить доклад»."

type Handler struct {
	bot          *tgbotapi.BotAPI
	service      service.ReminderService
	storageChatID int64
	loc          *time.Location
	log          *slog.Logger
}

func NewHandler(bot *tgbotapi.BotAPI, svc service.ReminderService, storageChatID int64, loc *time.Location, log *slog.Logger) *Handler {
	return &Handler{bot: bot, service: svc, storageChatID: storageChatID, loc: loc, log: log}
}

func (h *Handler) Handle(ctx context.Context, update tgbotapi.Update) {
	if update.Message == nil {
		return
	}
	text := strings.TrimSpace(update.Message.Text)
	if text == "" {
		return
	}
	replyChatID := update.Message.Chat.ID

	cmd, args := splitCommand(text)
	switch {
	case cmd == "/list":
		h.handleList(ctx, replyChatID)
	case cmd == "/history":
		h.handleHistory(ctx, replyChatID)
	case cmd == "/cancel":
		h.handleCancel(ctx, replyChatID, args)
	case strings.EqualFold(cmd, "выполнил"):
		h.handleComplete(ctx, replyChatID, args)
	default:
		h.handleCreate(ctx, replyChatID, text)
	}
}

func splitCommand(text string) (string, string) {
	parts := strings.SplitN(text, " ", 2)
	cmd := parts[0]
	if at := strings.Index(cmd, "@"); at >= 0 {
		cmd = cmd[:at]
	}
	args := ""
	if len(parts) == 2 {
		args = strings.TrimSpace(parts[1])
	}
	return cmd, args
}

func (h *Handler) handleCreate(ctx context.Context, replyChatID int64, text string) {
	r, err := h.service.CreateFromText(ctx, h.storageChatID, text)
	if err != nil {
		if errors.Is(err, service.ErrInvalidInput) {
			h.reply(replyChatID, unknownInputMessage)
			return
		}
		h.log.Error("create from text", "err", err, "text", text)
		h.reply(replyChatID, unknownInputMessage)
		return
	}
	h.reply(replyChatID, formatCreated(r, h.loc))
}

func (h *Handler) handleList(ctx context.Context, replyChatID int64) {
	items, err := h.service.ListActive(ctx, h.storageChatID)
	if err != nil {
		h.log.Error("list active", "err", err)
		h.reply(replyChatID, "Не удалось получить список.")
		return
	}
	if len(items) == 0 {
		h.reply(replyChatID, "Активных задач нет.")
		return
	}
	h.reply(replyChatID, formatActiveList(items, h.loc))
}

func (h *Handler) handleHistory(ctx context.Context, replyChatID int64) {
	items, err := h.service.ListHistory(ctx, h.storageChatID)
	if err != nil {
		h.log.Error("list history", "err", err)
		h.reply(replyChatID, "Не удалось получить историю.")
		return
	}
	if len(items) == 0 {
		h.reply(replyChatID, "История пуста.")
		return
	}
	h.reply(replyChatID, formatHistory(items, h.loc))
}

func (h *Handler) handleCancel(ctx context.Context, chatID int64, args string) {
	id, ok := parseID(args)
	if !ok {
		h.reply(chatID, "Укажи номер задачи: /cancel 5")
		return
	}
	if err := h.service.Cancel(ctx, id); err != nil {
		if errors.Is(err, service.ErrNotFound) {
			h.reply(chatID, fmt.Sprintf("Задача #%d не найдена.", id))
			return
		}
		h.log.Error("cancel", "id", id, "err", err)
		h.reply(chatID, "Не удалось отменить задачу.")
		return
	}
	h.reply(chatID, fmt.Sprintf("Задача #%d отменена.", id))
}

func (h *Handler) handleComplete(ctx context.Context, chatID int64, args string) {
	id, ok := parseID(args)
	if !ok {
		h.reply(chatID, "Укажи номер задачи: выполнил 5")
		return
	}
	if err := h.service.Complete(ctx, id); err != nil {
		if errors.Is(err, service.ErrNotFound) {
			h.reply(chatID, fmt.Sprintf("Задача #%d не найдена.", id))
			return
		}
		h.log.Error("complete", "id", id, "err", err)
		h.reply(chatID, "Не удалось закрыть задачу.")
		return
	}
	h.reply(chatID, fmt.Sprintf("Задача #%d закрыта.", id))
}

func (h *Handler) reply(chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	if _, err := h.bot.Send(msg); err != nil {
		h.log.Error("send reply", "chat_id", chatID, "err", err)
	}
}

func parseID(s string) (int64, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, false
	}
	id, err := strconv.ParseInt(s, 10, 64)
	if err != nil || id <= 0 {
		return 0, false
	}
	return id, true
}

var monthsGenitive = [...]string{
	"января", "февраля", "марта", "апреля", "мая", "июня",
	"июля", "августа", "сентября", "октября", "ноября", "декабря",
}

var weekdayPhrase = map[string]string{
	"monday":    "каждый понедельник",
	"tuesday":   "каждый вторник",
	"wednesday": "каждую среду",
	"thursday":  "каждый четверг",
	"friday":    "каждую пятницу",
	"saturday":  "каждую субботу",
	"sunday":    "каждое воскресенье",
}

func formatDayMonth(t time.Time, loc *time.Location) string {
	t = t.In(loc)
	return fmt.Sprintf("%d %s", t.Day(), monthsGenitive[t.Month()-1])
}

func formatHM(t time.Time, loc *time.Location) string {
	t = t.In(loc)
	return fmt.Sprintf("%02d:%02d", t.Hour(), t.Minute())
}

func formatWhen(r *models.Reminder, loc *time.Location) string {
	hm := formatHM(r.RemindAt, loc)
	if r.Recurrence == nil {
		return fmt.Sprintf("%s, %s", formatDayMonth(r.RemindAt, loc), hm)
	}
	switch *r.Recurrence {
	case "daily":
		return fmt.Sprintf("каждый день, %s", hm)
	case "weekly":
		if r.RecurrenceDay != nil {
			if phrase, ok := weekdayPhrase[*r.RecurrenceDay]; ok {
				return fmt.Sprintf("%s, %s", phrase, hm)
			}
		}
		return fmt.Sprintf("каждую неделю, %s", hm)
	}
	return fmt.Sprintf("%s, %s", formatDayMonth(r.RemindAt, loc), hm)
}

func formatCreated(r *models.Reminder, loc *time.Location) string {
	if r.Recurrence != nil {
		return fmt.Sprintf("Принято. #%d — %s, %s.", r.ID, r.Task, formatWhen(r, loc))
	}
	return fmt.Sprintf("Принято. #%d — %s, %s в %s.",
		r.ID, r.Task, formatDayMonth(r.RemindAt, loc), formatHM(r.RemindAt, loc))
}

func formatActiveList(items []*models.Reminder, loc *time.Location) string {
	var b strings.Builder
	b.WriteString("Активные задачи:\n\n")
	for i, r := range items {
		if i > 0 {
			b.WriteByte('\n')
		}
		fmt.Fprintf(&b, "#%d · %s — %s", r.ID, formatWhen(r, loc), r.Task)
	}
	return b.String()
}

func formatHistory(items []*models.Reminder, loc *time.Location) string {
	var b strings.Builder
	b.WriteString("История (последние 10):\n\n")
	for i, r := range items {
		if i > 0 {
			b.WriteByte('\n')
		}
		status := "выполнена"
		when := r.DoneAt
		if r.Cancelled {
			status = "отменена"
			when = r.CancelledAt
		}
		dateStr := formatDayMonth(r.RemindAt, loc)
		if when != nil {
			dateStr = formatDayMonth(*when, loc)
		}
		fmt.Fprintf(&b, "#%d · %s · %s — %s", r.ID, status, dateStr, r.Task)
	}
	return b.String()
}

func formatReminderNotification(r *models.Reminder) string {
	return fmt.Sprintf("Напоминание #%d — %s.\nЕсли выполнено, закрой задачу: выполнил %d", r.ID, r.Task, r.ID)
}
