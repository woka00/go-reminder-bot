package bot

import (
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/wokaxd/reminder-bot/internal/models"
)

const (
	cbDone       = "done"
	cbCancel     = "cancel"
	cbSnooze     = "snz"
	cbReschedule = "rsch"
	cbBack       = "back"
	cbTomorrow   = "tmrw"
	cbListClose  = "lc"
)

const listButtonsPerRow = 4

var emptyInlineMarkup = tgbotapi.InlineKeyboardMarkup{
	InlineKeyboard: [][]tgbotapi.InlineKeyboardButton{},
}

func notificationKeyboard(id int64) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Готово", fmt.Sprintf("%s:%d", cbDone, id)),
			tgbotapi.NewInlineKeyboardButtonData("+15 мин", fmt.Sprintf("%s:%d:15", cbSnooze, id)),
			tgbotapi.NewInlineKeyboardButtonData("Перенести", fmt.Sprintf("%s:%d", cbReschedule, id)),
		),
	)
}

func rescheduleMenuKeyboard(id int64) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("+30 мин", fmt.Sprintf("%s:%d:30", cbSnooze, id)),
			tgbotapi.NewInlineKeyboardButtonData("+1 ч", fmt.Sprintf("%s:%d:60", cbSnooze, id)),
			tgbotapi.NewInlineKeyboardButtonData("+3 ч", fmt.Sprintf("%s:%d:180", cbSnooze, id)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Завтра 09:00", fmt.Sprintf("%s:%d", cbTomorrow, id)),
			tgbotapi.NewInlineKeyboardButtonData("← Назад", fmt.Sprintf("%s:%d", cbBack, id)),
		),
	)
}

func createdKeyboard(id int64) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Отменить", fmt.Sprintf("%s:%d", cbCancel, id)),
		),
	)
}

func listInlineKeyboard(items []*models.Reminder) tgbotapi.InlineKeyboardMarkup {
	var rows [][]tgbotapi.InlineKeyboardButton
	var row []tgbotapi.InlineKeyboardButton
	for i, r := range items {
		row = append(row, tgbotapi.NewInlineKeyboardButtonData(
			fmt.Sprintf("#%d", r.ID),
			fmt.Sprintf("%s:%d", cbListClose, r.ID),
		))
		if (i+1)%listButtonsPerRow == 0 {
			rows = append(rows, row)
			row = nil
		}
	}
	if len(row) > 0 {
		rows = append(rows, row)
	}
	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

func persistentKeyboard() tgbotapi.ReplyKeyboardMarkup {
	kb := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("/list"),
		),
	)
	kb.ResizeKeyboard = true
	return kb
}
