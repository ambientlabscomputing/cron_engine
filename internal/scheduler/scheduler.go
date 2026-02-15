package scheduler

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/robfig/cron/v3"
)

type Scheduler struct {
	logger *slog.Logger
	parser cron.Parser
	jobs   map[string]chan bool
}

func NewScheduler(logger *slog.Logger) *Scheduler {
	return &Scheduler{
		logger: logger,
		parser: cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor),
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
	schedule, err := s.parser.Parse(expression)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid cron expression '%s': %w", expression, err)
	}
	return schedule.Next(time.Now()), nil
}

func (s *Scheduler) nextOccurrence(expression string, after time.Time) time.Time {
	schedule, err := s.parser.Parse(expression)
	if err != nil {
		s.logger.Warn("failed to parse cron expression, using 1 minute fallback", "expression", expression, "error", err)
		return after.Add(1 * time.Minute)
	}
	return schedule.Next(after)
}
