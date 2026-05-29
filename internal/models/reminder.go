package models

import "time"

type Reminder struct {
	ID            int64
	ChatID        int64
	Task          string
	RemindAt      time.Time
	Done          bool
	DoneAt        *time.Time
	Cancelled     bool
	CancelledAt   *time.Time
	Recurrence    *string
	RecurrenceDay *string
	ParentID      *int64
	CreatedAt     time.Time
}
