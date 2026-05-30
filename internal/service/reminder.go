package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/wokaxd/reminder-bot/internal/models"
	"github.com/wokaxd/reminder-bot/internal/parser"
	"github.com/wokaxd/reminder-bot/internal/storage"
)

var (
	ErrInvalidInput = errors.New("invalid input")
	ErrNotFound     = errors.New("reminder not found")
)

const historyLimit = 10

type ReminderService interface {
	CreateFromText(ctx context.Context, chatID int64, text string) (*models.Reminder, error)
	Complete(ctx context.Context, id int64) error
	Cancel(ctx context.Context, id int64) error
	ListActive(ctx context.Context, chatID int64) ([]*models.Reminder, error)
	ListHistory(ctx context.Context, chatID int64) ([]*models.Reminder, error)
	ProcessDue(ctx context.Context) error
}

type Sender interface {
	SendReminder(ctx context.Context, r *models.Reminder) error
}

type Service struct {
	store  storage.ReminderStorage
	sender Sender
	loc    *time.Location
	now    func() time.Time
	log    *slog.Logger
}

func New(store storage.ReminderStorage, sender Sender, loc *time.Location, log *slog.Logger) *Service {
	return &Service{
		store:  store,
		sender: sender,
		loc:    loc,
		now:    time.Now,
		log:    log,
	}
}

func (s *Service) CreateFromText(ctx context.Context, chatID int64, text string) (*models.Reminder, error) {
	parsed, err := parser.Parse(text, s.now(), s.loc)
	if err != nil {
		if errors.Is(err, parser.ErrEmptyTask) {
			return nil, ErrInvalidInput
		}
		return nil, fmt.Errorf("parse: %w", err)
	}

	r := &models.Reminder{
		ChatID:        chatID,
		Task:          parsed.Task,
		RemindAt:      parsed.RemindAt,
		Recurrence:    parsed.Recurrence,
		RecurrenceDay: parsed.RecurrenceDay,
	}
	if _, err := s.store.Create(ctx, r); err != nil {
		return nil, fmt.Errorf("create: %w", err)
	}
	return r, nil
}

func (s *Service) Complete(ctx context.Context, id int64) error {
	if err := s.store.MarkDone(ctx, id); err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return ErrNotFound
		}
		return fmt.Errorf("mark done: %w", err)
	}
	return nil
}

func (s *Service) Cancel(ctx context.Context, id int64) error {
	if err := s.store.MarkCancelled(ctx, id); err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return ErrNotFound
		}
		return fmt.Errorf("mark cancelled: %w", err)
	}
	return nil
}

func (s *Service) ListActive(ctx context.Context, chatID int64) ([]*models.Reminder, error) {
	out, err := s.store.ListActive(ctx, chatID)
	if err != nil {
		return nil, fmt.Errorf("list active: %w", err)
	}
	return out, nil
}

func (s *Service) ListHistory(ctx context.Context, chatID int64) ([]*models.Reminder, error) {
	out, err := s.store.ListHistory(ctx, chatID, historyLimit)
	if err != nil {
		return nil, fmt.Errorf("list history: %w", err)
	}
	return out, nil
}

func (s *Service) ProcessDue(ctx context.Context) error {
	due, err := s.store.ListDue(ctx, s.now())
	if err != nil {
		return fmt.Errorf("list due: %w", err)
	}
	for _, r := range due {
		if err := s.sender.SendReminder(ctx, r); err != nil {
			s.log.Error("send reminder", "id", r.ID, "err", err)
			continue
		}
		if err := s.store.MarkDone(ctx, r.ID); err != nil {
			s.log.Error("mark done after send", "id", r.ID, "err", err)
			continue
		}
		if r.Recurrence != nil {
			if err := s.scheduleNext(ctx, r); err != nil {
				s.log.Error("schedule next recurrence", "id", r.ID, "err", err)
			}
		}
	}
	return nil
}

func (s *Service) scheduleNext(ctx context.Context, prev *models.Reminder) error {
	nextAt := nextRecurrence(prev.RemindAt, *prev.Recurrence)

	parentID := prev.ID
	if prev.ParentID != nil {
		parentID = *prev.ParentID
	}

	next := &models.Reminder{
		ChatID:        prev.ChatID,
		Task:          prev.Task,
		RemindAt:      nextAt,
		Recurrence:    prev.Recurrence,
		RecurrenceDay: prev.RecurrenceDay,
		ParentID:      &parentID,
	}
	if _, err := s.store.Create(ctx, next); err != nil {
		return fmt.Errorf("create next: %w", err)
	}
	return nil
}

func nextRecurrence(prev time.Time, rec string) time.Time {
	switch rec {
	case "daily":
		return prev.AddDate(0, 0, 1)
	case "weekly":
		return prev.AddDate(0, 0, 7)
	}
	return prev.AddDate(0, 0, 1)
}
