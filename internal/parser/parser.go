package parser

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	ErrEmptyTask  = errors.New("empty task")
	ErrNoDateTime = errors.New("no date or time")
)

type ParseResult struct {
	Task          string
	RemindAt      time.Time
	Recurrence    *string
	RecurrenceDay *string
}

func Parse(input string, now time.Time, loc *time.Location) (*ParseResult, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return nil, ErrEmptyTask
	}

	now = now.In(loc)

	rest, rec := extractRecurrence(input)

	rest, tr, err := extractTime(rest)
	if err != nil {
		return nil, fmt.Errorf("parse time: %w", err)
	}

	rest, dr, err := extractDate(rest, now)
	if err != nil {
		return nil, fmt.Errorf("parse date: %w", err)
	}

	remindAt, err := buildRemindAt(rec, dr, tr, now, loc)
	if err != nil {
		return nil, fmt.Errorf("build remind_at: %w", err)
	}

	task := strings.Trim(collapseSpaces(rest), " ,.;:-")
	if task == "" {
		return nil, ErrEmptyTask
	}

	return &ParseResult{
		Task:          task,
		RemindAt:      remindAt,
		Recurrence:    rec.Recurrence,
		RecurrenceDay: rec.RecurrenceDay,
	}, nil
}

func ParseDateTime(input string, now time.Time, loc *time.Location) (time.Time, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return time.Time{}, ErrNoDateTime
	}
	now = now.In(loc)

	rest, tr, err := extractTime(input)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse time: %w", err)
	}
	_, dr, err := extractDate(rest, now)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse date: %w", err)
	}
	if !tr.Found && !dr.Found {
		return time.Time{}, ErrNoDateTime
	}
	if dr.Found {
		return time.Date(dr.Year, dr.Month, dr.Day, tr.Hour, tr.Minute, 0, 0, loc), nil
	}
	t := time.Date(now.Year(), now.Month(), now.Day(), tr.Hour, tr.Minute, 0, 0, loc)
	if !t.After(now) {
		t = t.AddDate(0, 0, 1)
	}
	return t, nil
}

func buildRemindAt(rec recurrenceResult, dr dateResult, tr timeResult, now time.Time, loc *time.Location) (time.Time, error) {
	if dr.Found {
		return time.Date(dr.Year, dr.Month, dr.Day, tr.Hour, tr.Minute, 0, 0, loc), nil
	}

	if rec.Recurrence != nil {
		return computeNextRecurrence(*rec.Recurrence, rec.RecurrenceDay, tr.Hour, tr.Minute, now, loc), nil
	}

	t := time.Date(now.Year(), now.Month(), now.Day(), tr.Hour, tr.Minute, 0, 0, loc)
	if !t.After(now) {
		t = t.AddDate(0, 0, 1)
	}
	return t, nil
}

func computeNextRecurrence(rec string, day *string, hour, minute int, now time.Time, loc *time.Location) time.Time {
	base := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, loc)

	switch rec {
	case "daily":
		if !base.After(now) {
			return base.AddDate(0, 0, 1)
		}
		return base

	case "weekly":
		if day == nil {
			if !base.After(now) {
				return base.AddDate(0, 0, 7)
			}
			return base
		}
		target := weekdayByName(*day)
		offset := (int(target) - int(now.Weekday()) + 7) % 7
		if offset == 0 && !base.After(now) {
			offset = 7
		}
		return base.AddDate(0, 0, offset)
	}

	return base
}

func weekdayByName(s string) time.Weekday {
	switch s {
	case "sunday":
		return time.Sunday
	case "monday":
		return time.Monday
	case "tuesday":
		return time.Tuesday
	case "wednesday":
		return time.Wednesday
	case "thursday":
		return time.Thursday
	case "friday":
		return time.Friday
	case "saturday":
		return time.Saturday
	}
	return time.Sunday
}

func collapseSpaces(s string) string {
	return strings.Join(strings.Fields(s), " ")
}
