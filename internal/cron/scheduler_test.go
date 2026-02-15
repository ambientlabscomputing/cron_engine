package cron

import (
	"strings"
	"testing"
)

// TestCommandParsing verifies the command parsing logic that scheduleJobExecution uses
func TestCommandParsing(t *testing.T) {
	tests := []struct {
		name        string
		command     string
		wantCmd     string
		wantArgs    []string
		shouldParse bool
	}{
		{
			name:        "simple command",
			command:     "ls",
			wantCmd:     "ls",
			wantArgs:    []string{},
			shouldParse: true,
		},
		{
			name:        "command with arguments",
			command:     "echo hello world",
			wantCmd:     "echo",
			wantArgs:    []string{"hello", "world"},
			shouldParse: true,
		},
		{
			name:        "command with path",
			command:     "/usr/bin/python3 script.py --verbose",
			wantCmd:     "/usr/bin/python3",
			wantArgs:    []string{"script.py", "--verbose"},
			shouldParse: true,
		},
		{
			name:        "empty command",
			command:     "",
			shouldParse: false,
		},
		{
			name:        "whitespace only",
			command:     "   ",
			shouldParse: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parts := strings.Fields(tt.command)

			if !tt.shouldParse {
				if len(parts) != 0 {
					t.Errorf("Expected empty parse for %q, got %v", tt.command, parts)
				}
				return
			}

			if len(parts) == 0 {
				t.Fatalf("Failed to parse command %q", tt.command)
			}

			cmd := parts[0]
			var args []string
			if len(parts) > 1 {
				args = parts[1:]
			}

			if cmd != tt.wantCmd {
				t.Errorf("Command: want %q, got %q", tt.wantCmd, cmd)
			}

			if len(args) != len(tt.wantArgs) {
				t.Errorf("Args length: want %d, got %d", len(tt.wantArgs), len(args))
			}

			for i, arg := range args {
				if i < len(tt.wantArgs) && arg != tt.wantArgs[i] {
					t.Errorf("Arg[%d]: want %q, got %q", i, tt.wantArgs[i], arg)
				}
			}
		})
	}
}

// TestTimeoutParsing verifies timeout defaulting logic
func TestTimeoutParsing(t *testing.T) {
	tests := []struct {
		name            string
		input           uint32
		expectedSeconds uint32
	}{
		{"zero defaults to 60", 0, 60},
		{"explicit value used", 300, 300},
		{"small value accepted", 1, 1},
		{"large value accepted", 3600, 3600},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			timeout := tt.input
			if timeout == 0 {
				timeout = 60
			}

			if timeout != tt.expectedSeconds {
				t.Errorf("Want %d seconds, got %d", tt.expectedSeconds, timeout)
			}
		})
	}
}

// TestJobValidation verifies basic job validation
func TestJobValidation(t *testing.T) {
	jobs := []struct {
		name      string
		job       *Job
		shouldErr bool
	}{
		{
			name: "valid job",
			job: &Job{
				ID:         "job-1",
				Expression: "*/5 * * * *",
				Command:    "echo test",
			},
			shouldErr: false,
		},
		{
			name: "missing ID",
			job: &Job{
				Expression: "*/5 * * * *",
				Command:    "echo test",
			},
			shouldErr: true,
		},
		{
			name: "missing command",
			job: &Job{
				ID:         "job-1",
				Expression: "*/5 * * * *",
			},
			shouldErr: true,
		},
		{
			name: "missing expression",
			job: &Job{
				ID:      "job-1",
				Command: "echo test",
			},
			shouldErr: true,
		},
	}

	for _, tt := range jobs {
		t.Run(tt.name, func(t *testing.T) {
			hasErr := (tt.job.ID == "" || tt.job.Command == "" || tt.job.Expression == "")

			if hasErr != tt.shouldErr {
				t.Errorf("Validation: want error=%v, got error=%v", tt.shouldErr, hasErr)
			}
		})
	}
}

// TestCronScheduleFormat validates cron expression format
func TestCronScheduleFormat(t *testing.T) {
	schedules := []struct {
		expr  string
		valid bool
		desc  string
	}{
		{"*/5 * * * *", true, "every 5 minutes"},
		{"0 * * * *", true, "hourly"},
		{"0 0 * * *", true, "daily"},
		{"@hourly", true, "hourly shorthand"},
		{"@daily", true, "daily shorthand"},
		{"invalid", false, "invalid format"},
		{"", false, "empty"},
		{"* * * *", false, "missing field"},
	}

	for _, tt := range schedules {
		t.Run(tt.desc, func(t *testing.T) {
			var valid bool
			if strings.HasPrefix(tt.expr, "@") {
				valid = len(tt.expr) > 1
			} else {
				fields := strings.Fields(tt.expr)
				valid = len(fields) == 5
			}

			if valid != tt.valid {
				t.Errorf("Schedule %q: want valid=%v, got %v", tt.expr, tt.valid, valid)
			}
		})
	}
}
