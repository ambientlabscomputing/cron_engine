package scheduler

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

type Scheduler struct {
	logger *slog.Logger
	jobs   map[string]chan bool
}

func NewScheduler(logger *slog.Logger) *Scheduler {
	return &Scheduler{
		logger: logger,
		jobs:   make(map[string]chan bool),
	}
}

func (s *Scheduler) ScheduleJob(ctx context.Context, id, expression string, callback func(context.Context) error) error {
	nextRun, err := s.parseExpression(expression)
	if err != nil {
		return fmt.Errorf("failed to parse cron expression: %w", err)
	}

	stopChan := make(chan bool)
	s.jobs[id] = stopChan

	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()

		for {
			select {
			case <-stopChan:
				s.logger.Info("cron job stopped", "id", id)
				return

			case <-ticker.C:
				now := time.Now()
				if now.After(nextRun) {
					if err := callback(ctx); err != nil {
						s.logger.Error("cron job execution failed", "id", id, "error", err)
					} else {
						s.logger.Info("cron job executed", "id", id)
					}
					nextRun = s.nextOccurrence(expression, now)
				}
			}
		}
	}()

	s.logger.Info("cron job scheduled", "id", id, "expression", expression, "next_run", nextRun)
	return nil
}

func (s *Scheduler) StopJob(id string) {
	if stopChan, ok := s.jobs[id]; ok {
		close(stopChan)
		delete(s.jobs, id)
	}
}

func (s *Scheduler) parseExpression(expression string) (time.Time, error) {
	return time.Now().Add(1 * time.Minute), nil
}

func (s *Scheduler) nextOccurrence(expression string, after time.Time) time.Time {
	return after.Add(1 * time.Minute)
}
