package storage

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/wokaxd/reminder-bot/internal/models"
)

var ErrNotFound = errors.New("reminder not found")

type ReminderStorage interface {
	Create(ctx context.Context, r *models.Reminder) (int64, error)
	GetByID(ctx context.Context, id int64) (*models.Reminder, error)
	ListActive(ctx context.Context, chatID int64) ([]*models.Reminder, error)
	ListDue(ctx context.Context, now time.Time) ([]*models.Reminder, error)
	MarkDone(ctx context.Context, id int64) error
	MarkCancelled(ctx context.Context, id int64) error
	ListHistory(ctx context.Context, chatID int64, limit int) ([]*models.Reminder, error)
}

type PostgresReminderStorage struct {
	pool *pgxpool.Pool
}

func NewPostgresReminderStorage(pool *pgxpool.Pool) *PostgresReminderStorage {
	return &PostgresReminderStorage{pool: pool}
}

const reminderColumns = `id, chat_id, task, remind_at, done, done_at, cancelled, cancelled_at,
    recurrence, recurrence_day, parent_id, created_at`

func (s *PostgresReminderStorage) Create(ctx context.Context, r *models.Reminder) (int64, error) {
	const q = `
        INSERT INTO reminders (chat_id, task, remind_at, recurrence, recurrence_day, parent_id)
        VALUES ($1, $2, $3, $4, $5, $6)
        RETURNING id, created_at
    `
	var id int64
	var createdAt time.Time
	err := s.pool.QueryRow(ctx, q,
		r.ChatID, r.Task, r.RemindAt, r.Recurrence, r.RecurrenceDay, r.ParentID,
	).Scan(&id, &createdAt)
	if err != nil {
		return 0, fmt.Errorf("insert reminder: %w", err)
	}
	r.ID = id
	r.CreatedAt = createdAt
	return id, nil
}

func (s *PostgresReminderStorage) GetByID(ctx context.Context, id int64) (*models.Reminder, error) {
	q := `SELECT ` + reminderColumns + ` FROM reminders WHERE id = $1`
	row := s.pool.QueryRow(ctx, q, id)
	r, err := scanReminder(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get by id: %w", err)
	}
	return r, nil
}

func (s *PostgresReminderStorage) ListActive(ctx context.Context, chatID int64) ([]*models.Reminder, error) {
	q := `SELECT ` + reminderColumns + `
        FROM reminders
        WHERE chat_id = $1 AND done = FALSE AND cancelled = FALSE
        ORDER BY remind_at ASC`
	rows, err := s.pool.Query(ctx, q, chatID)
	if err != nil {
		return nil, fmt.Errorf("list active: %w", err)
	}
	defer rows.Close()
	return scanReminders(rows)
}

func (s *PostgresReminderStorage) ListDue(ctx context.Context, now time.Time) ([]*models.Reminder, error) {
	q := `SELECT ` + reminderColumns + `
        FROM reminders
        WHERE done = FALSE AND cancelled = FALSE AND remind_at <= $1
        ORDER BY remind_at ASC`
	rows, err := s.pool.Query(ctx, q, now)
	if err != nil {
		return nil, fmt.Errorf("list due: %w", err)
	}
	defer rows.Close()
	return scanReminders(rows)
}

func (s *PostgresReminderStorage) MarkDone(ctx context.Context, id int64) error {
	const q = `UPDATE reminders SET done = TRUE, done_at = NOW()
        WHERE id = $1 AND done = FALSE AND cancelled = FALSE`
	tag, err := s.pool.Exec(ctx, q, id)
	if err != nil {
		return fmt.Errorf("mark done: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *PostgresReminderStorage) MarkCancelled(ctx context.Context, id int64) error {
	const q = `UPDATE reminders SET cancelled = TRUE, cancelled_at = NOW()
        WHERE id = $1 AND done = FALSE AND cancelled = FALSE`
	tag, err := s.pool.Exec(ctx, q, id)
	if err != nil {
		return fmt.Errorf("mark cancelled: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *PostgresReminderStorage) ListHistory(ctx context.Context, chatID int64, limit int) ([]*models.Reminder, error) {
	q := `SELECT ` + reminderColumns + `
        FROM reminders
        WHERE chat_id = $1 AND (done = TRUE OR cancelled = TRUE)
        ORDER BY COALESCE(done_at, cancelled_at) DESC
        LIMIT $2`
	rows, err := s.pool.Query(ctx, q, chatID, limit)
	if err != nil {
		return nil, fmt.Errorf("list history: %w", err)
	}
	defer rows.Close()
	return scanReminders(rows)
}

type scanner interface {
	Scan(dest ...any) error
}

func scanReminder(row scanner) (*models.Reminder, error) {
	var r models.Reminder
	err := row.Scan(
		&r.ID, &r.ChatID, &r.Task, &r.RemindAt,
		&r.Done, &r.DoneAt, &r.Cancelled, &r.CancelledAt,
		&r.Recurrence, &r.RecurrenceDay, &r.ParentID, &r.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &r, nil
}

func scanReminders(rows pgx.Rows) ([]*models.Reminder, error) {
	var out []*models.Reminder
	for rows.Next() {
		r, err := scanReminder(rows)
		if err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows: %w", err)
	}
	return out, nil
}
