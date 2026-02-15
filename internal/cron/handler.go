package cron

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/ambientlabscomputing/cron_engine/internal/scheduler"
	"github.com/ambientlabscomputing/cron_engine/internal/syscall"
)

type Orchestrator struct {
	syscallClient *syscall.Client
	scheduler     *scheduler.Scheduler
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
		scheduler:     scheduler.NewScheduler(logger),
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

	// Schedule the job for execution
	if err := o.scheduleJobExecution(ctx, job); err != nil {
		return fmt.Errorf("failed to schedule job: %w", err)
	}

	o.logger.Info("cron job created and scheduled", "id", id, "expression", expression)
	return nil
}

// scheduleJobExecution schedules a job with the scheduler and sets up command execution
func (o *Orchestrator) scheduleJobExecution(ctx context.Context, job *Job) error {
	callback := func(execCtx context.Context) error {
		// Parse command into command and args
		parts := strings.Fields(job.Command)
		if len(parts) == 0 {
			return fmt.Errorf("empty command")
		}
		command := parts[0]
		args := []string{}
		if len(parts) > 1 {
			args = parts[1:]
		}

		// Parse timeout (default to 60 seconds if not set or invalid)
		timeoutSeconds := uint32(60)
		if job.Timeout != "" {
			if parsed, err := strconv.ParseUint(job.Timeout, 10, 32); err == nil {
				timeoutSeconds = uint32(parsed)
			}
		}

		// Execute command via kernel ExecService
		resp, err := o.syscallClient.RunProcess(execCtx, command, args, nil, "", timeoutSeconds)
		if err != nil {
			// Emit failure event
			o.syscallClient.EmitCronEvent(execCtx, "cron.failed", job.ID, map[string]interface{}{
				"error": err.Error(),
			})
			return fmt.Errorf("command execution failed: %w", err)
		}

		// Emit completion event with exit code
		o.syscallClient.EmitCronEvent(execCtx, "cron.completed", job.ID, map[string]interface{}{
			"exit_code":   resp.ExitCode,
			"duration_ms": resp.DurationMs,
		})

		if resp.ExitCode != 0 {
			return fmt.Errorf("command exited with code %d: %s", resp.ExitCode, resp.Stderr)
		}

		return nil
	}

	return o.scheduler.ScheduleJob(ctx, job.ID, job.Expression, callback)
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
	// Stop the scheduled job
	o.scheduler.StopJob(id)

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
