package cron

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/ambientlabscomputing/cron_engine/internal/syscall"
)

type Orchestrator struct {
	syscallClient *syscall.Client
	logger        *slog.Logger
	jobs          map[string]*Job
}

type Job struct {
	ID         string                 `json:"id"`
	Expression string                 `json:"expression"`
	Command    string                 `json:"command"`
	Timeout    string                 `json:"timeout"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt  time.Time              `json:"created_at"`
	UpdatedAt  time.Time              `json:"updated_at"`
}

func NewOrchestrator(syscallClient *syscall.Client, logger *slog.Logger) *Orchestrator {
	return &Orchestrator{
		syscallClient: syscallClient,
		logger:        logger,
		jobs:          make(map[string]*Job),
	}
}

func (o *Orchestrator) CreateJob(ctx context.Context, id, expression, command, timeout string, metadata map[string]interface{}) error {
	job := &Job{
		ID:         id,
		Expression: expression,
		Command:    command,
		Timeout:    timeout,
		Metadata:   metadata,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	o.jobs[id] = job

	data, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("failed to marshal job: %w", err)
	}

	if err := o.syscallClient.SaveCronState(ctx, id, data); err != nil {
		return fmt.Errorf("failed to save cron state: %w", err)
	}

	o.logger.Info("cron job created", "id", id, "expression", expression)
	return nil
}

func (o *Orchestrator) GetJob(ctx context.Context, id string) (*Job, error) {
	if job, ok := o.jobs[id]; ok {
		return job, nil
	}

	data, err := o.syscallClient.GetCronState(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get cron state: %w", err)
	}

	var job Job
	if err := json.Unmarshal(data, &job); err != nil {
		return nil, fmt.Errorf("failed to unmarshal job: %w", err)
	}

	o.jobs[id] = &job
	return &job, nil
}

func (o *Orchestrator) DeleteJob(ctx context.Context, id string) error {
	delete(o.jobs, id)

	if err := o.syscallClient.EmitCronEvent(ctx, "cron.deleted", id, map[string]interface{}{}); err != nil {
		o.logger.Warn("failed to emit cron deleted event", "id", id, "error", err)
	}

	o.logger.Info("cron job deleted", "id", id)
	return nil
}

func (o *Orchestrator) ListJobs() map[string]*Job {
	return o.jobs
}
