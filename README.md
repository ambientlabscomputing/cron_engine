# Cron Engine UMC

Cron Engine is a User Mode Component (UMC) that provides distributed, reliable cron job scheduling for the Underleaf platform. It manages scheduled task definitions, execution tracking, and distributed coordination through the UA kernel.

## Architecture

The Cron Engine follows the standard UMC pattern:

```
┌─────────────────────────────────────────┐
│         Cron Engine Process             │
├─────────────────────────────────────────┤
│ HTTP API Server (:8080)                 │
│  - /health        (GET)                 │
│  - /crons         (POST, GET, DELETE)   │
│  - /crons/{id}    (GET, PATCH)          │
├─────────────────────────────────────────┤
│ Kernel Syscall Client                   │
│  - Emit cron events (execution, error)  │
│  - Store cron state in cluster KV       │
│  - Subscribe to executions              │
├─────────────────────────────────────────┤
│ Cron Scheduler/Executor                 │
│  - Parse cron expressions               │
│  - Schedule job executions              │
│  - Track execution history              │
│  - Handle retries and backoffs          │
└─────────────────────────────────────────┘
         ↓ (Unix Domain Socket)
┌─────────────────────────────────────────┐
│      UA Kernel Syscall Server           │
│  - EventService (publish events)        │
│  - ClusterService (KV store)            │
│  - ExecService (command execution)      │
└─────────────────────────────────────────┘
```

## Features

- **Distributed Scheduling**: Cron jobs coordinated across cluster via kernel KV
- **Event Publishing**: Cron execution events published to event stream
- **State Management**: Cron state persisted in cluster key-value store
- **HTTP API**: RESTful interface for cron job management
- **Graceful Shutdown**: Resource cleanup on termination
- **Health Checks**: Built-in health monitoring endpoint

## Getting Started

### Build

```bash
make build
```

### Run

```bash
make run
```

The cron engine will:
1. Connect to kernel syscall server at `/tmp/ua_kernel.sock`
2. Start HTTP API server on `:8080`
3. Wait for cron job definitions from API or cluster state

### Configuration

Environment variables:

- `KERNEL_SOCKET` - Path to kernel syscall socket (default: `/tmp/ua_kernel.sock`)
- `LOG_LEVEL` - Logging level: debug, info, warn, error (default: `info`)

### Syscalls Used

The cron engine uses the following kernel syscalls:

**EventService:**
- `EmitEvent` - Publish cron execution events
  - `cron.scheduled` - Cron job scheduled
  - `cron.started` - Cron execution started
  - `cron.completed` - Cron execution completed successfully
  - `cron.failed` - Cron execution failed
  - `cron.skipped` - Cron execution skipped (e.g., due to concurrency limit)

**ClusterService:**
- `GetKV` - Retrieve cron state from cluster
- `PutKV` - Store cron state in cluster
- `ListKV` - List all cron jobs in cluster

**ExecService:**
- `Execute` - Execute cron job command (via kernel)

## Package Structure

- `cmd/serve/` - Main entry point with lifecycle management
- `internal/syscall/` - Kernel syscall API wrapper
- `internal/api/` - HTTP request handlers
- `internal/cron/` - Cron job orchestration logic
- `internal/scheduler/` - Cron expression parsing and scheduling

## Integration with UA Agent

The UA agent (underleaf_client) manages the cron_engine process:

1. **Discovery**: Locates binary at `~/.underleaf/bin/cron-engine-serve`
2. **Launch**: Spawns cron engine at startup with `KERNEL_SOCKET` environment variable
3. **Shutdown**: Sends SIGTERM and waits up to 5 seconds for graceful exit
4. **Restart**: Automatically restarts if crash detected

## HTTP API Examples

### Health Check

```bash
curl -X GET http://localhost:8080/health
# Response: {"status":"healthy"}
```

### Create Cron Job

```bash
curl -X POST http://localhost:8080/crons \
  -H "Content-Type: application/json" \
  -d '{
    "id": "backup-daily",
    "expression": "0 2 * * *",
    "command": "backup-database",
    "timeout": "1h"
  }'
```

### Get Cron Job

```bash
curl -X GET http://localhost:8080/crons/backup-daily
```

### List All Crons

```bash
curl -X GET http://localhost:8080/crons
```

### Delete Cron Job

```bash
curl -X DELETE http://localhost:8080/crons/backup-daily
```

## Event Flow

```
Client Request
    ↓
HTTP API Handler
    ↓
Cron Orchestrator
    ↓
Kernel Syscall Client
    ↓
EventService (publish event)
ClusterService (persist state)
    ↓
Event Stream (subscribers notified)
```

## Testing

```bash
make test
```

## Dependencies

- `umc_sdk` - UMC SDK with lifecycle and logging
- `google.golang.org/grpc` - gRPC client for kernel communication
- `lmittmann/tint` - Console formatting for logs

## Future Enhancements

- [ ] Cron job retry policies
- [ ] Execution history tracking
- [ ] Job dependency chains
- [ ] Time zone support
- [ ] Webhook notifications on job completion
- [ ] Resource limits per job
- [ ] Job deduplication across cluster
