package scheduler

import (
	"context"
	"log/slog"
	"time"

	"github.com/wokaxd/reminder-bot/internal/service"
)

const defaultInterval = 30 * time.Second

type Scheduler struct {
	svc      service.ReminderService
	interval time.Duration
	log      *slog.Logger
}

func New(svc service.ReminderService, log *slog.Logger) *Scheduler {
	return &Scheduler{svc: svc, interval: defaultInterval, log: log}
}

func (s *Scheduler) Run(ctx context.Context) {
	t := time.NewTicker(s.interval)
	defer t.Stop()

	s.log.Info("scheduler started", "interval", s.interval)
	for {
		select {
		case <-ctx.Done():
			s.log.Info("scheduler stopped")
			return
		case <-t.C:
			if err := s.svc.ProcessDue(ctx); err != nil {
				s.log.Error("process due", "err", err)
			}
		}
	}
}
