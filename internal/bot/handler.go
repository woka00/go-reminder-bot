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
	"github.com/wokaxd/reminder-bot/internal/parser"
	"github.com/wokaxd/reminder-bot/internal/service"
)

const reschedulePromptPrefix = "Новое время для #"

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
	if update.CallbackQuery != nil {
		h.handleCallback(ctx, update.CallbackQuery)
		return
	}
	if update.Message == nil {
		return
	}
	text := strings.TrimSpace(update.Message.Text)
	if text == "" {
		return
	}
	replyChatID := update.Message.Chat.ID

	if id, ok := h.matchReschedulePrompt(update.Message.ReplyToMessage); ok {
		h.handleCustomReschedule(ctx, replyChatID, update.Message.ReplyToMessage, id, text)
		return
	}

	cmd, args := splitCommand(text)
	switch {
	case cmd == "/start":
		h.handleStart(replyChatID)
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

func (h *Handler) handleStart(chatID int64) {
	msg := tgbotapi.NewMessage(chatID, "Привет. Пиши задачи естественным языком, например: «завтра в 14:00 позвонить маме».\nКоманды: /list, /history.")
	msg.ReplyMarkup = persistentKeyboard()
	if _, err := h.bot.Send(msg); err != nil {
		h.log.Error("send start", "err", err)
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
	msg := tgbotapi.NewMessage(replyChatID, formatCreated(r, h.loc))
	msg.ReplyMarkup = createdKeyboard(r.ID)
	if _, err := h.bot.Send(msg); err != nil {
		h.log.Error("send created", "err", err)
	}
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
	msg := tgbotapi.NewMessage(replyChatID, formatActiveList(items, h.loc))
	msg.ReplyMarkup = listInlineKeyboard(items)
	if _, err := h.bot.Send(msg); err != nil {
		h.log.Error("send list", "err", err)
	}
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

func (h *Handler) handleCallback(ctx context.Context, cb *tgbotapi.CallbackQuery) {
	parts := strings.Split(cb.Data, ":")
	if len(parts) < 2 {
		h.ackCallback(cb.ID, "")
		return
	}
	action := parts[0]
	id, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil || id <= 0 {
		h.ackCallback(cb.ID, "")
		return
	}

	switch action {
	case cbDone:
		h.cbComplete(ctx, cb, id, false)
	case cbListClose:
		h.cbComplete(ctx, cb, id, true)
	case cbCancel:
		h.cbCancel(ctx, cb, id)
	case cbSnooze:
		if len(parts) < 3 {
			h.ackCallback(cb.ID, "")
			return
		}
		mins, err := strconv.Atoi(parts[2])
		if err != nil || mins <= 0 {
			h.ackCallback(cb.ID, "")
			return
		}
		h.cbSnooze(ctx, cb, id, time.Duration(mins)*time.Minute)
	case cbTomorrow:
		h.cbTomorrow(ctx, cb, id)
	case cbCustom:
		h.cbAskCustom(cb, id)
	case cbReschedule:
		h.cbSwapMarkup(cb, rescheduleMenuKeyboard(id))
	case cbBack:
		h.cbSwapMarkup(cb, notificationKeyboard(id))
	default:
		h.ackCallback(cb.ID, "")
	}
}

func (h *Handler) ackCallback(cbID, text string) {
	cfg := tgbotapi.NewCallback(cbID, text)
	if _, err := h.bot.Request(cfg); err != nil {
		h.log.Error("ack callback", "err", err)
	}
}

func (h *Handler) cbComplete(ctx context.Context, cb *tgbotapi.CallbackQuery, id int64, fromList bool) {
	if err := h.service.Complete(ctx, id); err != nil {
		if errors.Is(err, service.ErrNotFound) {
			h.ackCallback(cb.ID, fmt.Sprintf("Задача #%d не найдена", id))
			return
		}
		h.log.Error("cb complete", "id", id, "err", err)
		h.ackCallback(cb.ID, "Ошибка")
		return
	}
	if fromList {
		h.refreshList(ctx, cb)
	} else {
		h.editMessageAppend(cb.Message, "Закрыто.")
	}
	h.ackCallback(cb.ID, fmt.Sprintf("Закрыто #%d", id))
}

func (h *Handler) cbCancel(ctx context.Context, cb *tgbotapi.CallbackQuery, id int64) {
	if err := h.service.Cancel(ctx, id); err != nil {
		if errors.Is(err, service.ErrNotFound) {
			h.ackCallback(cb.ID, fmt.Sprintf("Задача #%d не найдена", id))
			return
		}
		h.log.Error("cb cancel", "id", id, "err", err)
		h.ackCallback(cb.ID, "Ошибка")
		return
	}
	h.editMessageAppend(cb.Message, "Отменено.")
	h.ackCallback(cb.ID, fmt.Sprintf("Отменено #%d", id))
}

func (h *Handler) cbSnooze(ctx context.Context, cb *tgbotapi.CallbackQuery, id int64, d time.Duration) {
	newAt := time.Now().In(h.loc).Add(d).Truncate(time.Minute)
	if err := h.service.Reschedule(ctx, id, newAt); err != nil {
		if errors.Is(err, service.ErrNotFound) {
			h.ackCallback(cb.ID, fmt.Sprintf("Задача #%d не найдена", id))
			return
		}
		h.log.Error("cb snooze", "id", id, "err", err)
		h.ackCallback(cb.ID, "Ошибка")
		return
	}
	h.editMessageAppend(cb.Message, fmt.Sprintf("Отложено до %s.", formatHM(newAt, h.loc)))
	h.ackCallback(cb.ID, "Отложено")
}

func (h *Handler) cbTomorrow(ctx context.Context, cb *tgbotapi.CallbackQuery, id int64) {
	now := time.Now().In(h.loc)
	target := time.Date(now.Year(), now.Month(), now.Day(), 9, 0, 0, 0, h.loc).AddDate(0, 0, 1)
	if err := h.service.Reschedule(ctx, id, target); err != nil {
		if errors.Is(err, service.ErrNotFound) {
			h.ackCallback(cb.ID, fmt.Sprintf("Задача #%d не найдена", id))
			return
		}
		h.log.Error("cb tomorrow", "id", id, "err", err)
		h.ackCallback(cb.ID, "Ошибка")
		return
	}
	h.editMessageAppend(cb.Message, fmt.Sprintf("Перенесено на %s, %s.", formatDayMonth(target, h.loc), formatHM(target, h.loc)))
	h.ackCallback(cb.ID, "Перенесено")
}

func (h *Handler) cbSwapMarkup(cb *tgbotapi.CallbackQuery, kb tgbotapi.InlineKeyboardMarkup) {
	edit := tgbotapi.NewEditMessageReplyMarkup(cb.Message.Chat.ID, cb.Message.MessageID, kb)
	if _, err := h.bot.Request(edit); err != nil {
		h.log.Error("swap markup", "err", err)
	}
	h.ackCallback(cb.ID, "")
}

func (h *Handler) editMessageAppend(msg *tgbotapi.Message, suffix string) {
	text := strings.TrimRight(msg.Text, " \n") + "\n" + suffix
	edit := tgbotapi.NewEditMessageTextAndMarkup(msg.Chat.ID, msg.MessageID, text, emptyInlineMarkup)
	if _, err := h.bot.Send(edit); err != nil {
		h.log.Error("edit message append", "err", err)
	}
}

func (h *Handler) cbAskCustom(cb *tgbotapi.CallbackQuery, id int64) {
	prompt := fmt.Sprintf(
		"%s%d. Напиши новое время, например: «завтра в 14:00», «13 мая в 10:00», «послезавтра в 18:00».",
		reschedulePromptPrefix, id,
	)
	msg := tgbotapi.NewMessage(cb.Message.Chat.ID, prompt)
	msg.ReplyMarkup = tgbotapi.ForceReply{ForceReply: true, InputFieldPlaceholder: "завтра в 14:00"}
	if _, err := h.bot.Send(msg); err != nil {
		h.log.Error("ask custom", "err", err)
	}
	h.ackCallback(cb.ID, "")
}

func (h *Handler) matchReschedulePrompt(reply *tgbotapi.Message) (int64, bool) {
	if reply == nil || reply.From == nil {
		return 0, false
	}
	if reply.From.ID != h.bot.Self.ID {
		return 0, false
	}
	if !strings.HasPrefix(reply.Text, reschedulePromptPrefix) {
		return 0, false
	}
	rest := reply.Text[len(reschedulePromptPrefix):]
	end := strings.IndexAny(rest, ". \n")
	if end < 0 {
		end = len(rest)
	}
	id, err := strconv.ParseInt(rest[:end], 10, 64)
	if err != nil || id <= 0 {
		return 0, false
	}
	return id, true
}

func (h *Handler) handleCustomReschedule(ctx context.Context, replyChatID int64, prompt *tgbotapi.Message, id int64, text string) {
	newAt, err := parser.ParseDateTime(text, time.Now(), h.loc)
	if err != nil {
		h.reply(replyChatID, "Не понял время. Формат: «завтра в 14:00» или «13 мая в 10:00».")
		return
	}
	if err := h.service.Reschedule(ctx, id, newAt); err != nil {
		if errors.Is(err, service.ErrNotFound) {
			h.reply(replyChatID, fmt.Sprintf("Задача #%d не найдена.", id))
			return
		}
		h.log.Error("custom reschedule", "id", id, "err", err)
		h.reply(replyChatID, "Не удалось перенести.")
		return
	}
	confirmation := fmt.Sprintf("#%d — перенесено на %s, %s.", id, formatDayMonth(newAt, h.loc), formatHM(newAt, h.loc))
	edit := tgbotapi.NewEditMessageText(prompt.Chat.ID, prompt.MessageID, confirmation)
	if _, err := h.bot.Send(edit); err != nil {
		h.log.Error("edit reschedule prompt", "err", err)
		h.reply(replyChatID, confirmation)
	}
}

func (h *Handler) refreshList(ctx context.Context, cb *tgbotapi.CallbackQuery) {
	items, err := h.service.ListActive(ctx, h.storageChatID)
	if err != nil {
		h.log.Error("refresh list", "err", err)
		return
	}
	if len(items) == 0 {
		edit := tgbotapi.NewEditMessageTextAndMarkup(cb.Message.Chat.ID, cb.Message.MessageID, "Активных задач нет.", emptyInlineMarkup)
		if _, err := h.bot.Send(edit); err != nil {
			h.log.Error("edit empty list", "err", err)
		}
		return
	}
	edit := tgbotapi.NewEditMessageTextAndMarkup(cb.Message.Chat.ID, cb.Message.MessageID, formatActiveList(items, h.loc), listInlineKeyboard(items))
	if _, err := h.bot.Send(edit); err != nil {
		h.log.Error("edit list", "err", err)
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
		if r.NotifiedAt != nil && !r.Done && !r.Cancelled {
			b.WriteString(" · ждёт подтверждения")
		}
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
	return fmt.Sprintf("Напоминание #%d — %s.", r.ID, r.Task)
}
