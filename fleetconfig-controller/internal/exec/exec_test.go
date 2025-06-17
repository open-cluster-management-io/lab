package exec

import (
	"context"
	"os/exec"
	"strings"
	"testing"
	"time"
)

func TestCmdWithLogs(t *testing.T) {
	tests := []struct {
		name    string
		cmdFunc func() *exec.Cmd
		timeout time.Duration
		wantErr bool
		wantOut string
	}{
		{
			name: "successful command",
			cmdFunc: func() *exec.Cmd {
				return exec.Command("echo", "hello world")
			},
			timeout: 2 * time.Second,
			wantErr: false,
			wantOut: "hello world",
		},
		{
			name: "failing command",
			cmdFunc: func() *exec.Cmd {
				return exec.Command("ls", "/nonexistent/path")
			},
			timeout: 2 * time.Second,
			wantErr: true,
			wantOut: "No such file",
		},
		{
			name: "context timeout",
			cmdFunc: func() *exec.Cmd {
				return exec.Command("sleep", "10")
			},
			timeout: 7 * time.Second, // ensure timeout is longer than logging interval
			wantErr: true,
			wantOut: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), tt.timeout)
			defer cancel()

			out, err := CmdWithLogs(ctx, tt.cmdFunc(), "waiting for command...")

			if (err != nil) != tt.wantErr {
				t.Errorf("CmdWithLogs() error = %v, wantErr %v", err, tt.wantErr)
			}

			outStr := strings.TrimSpace(string(out))
			if tt.wantOut != "" && !strings.Contains(outStr, tt.wantOut) {
				t.Errorf("output = %q, want to contain %q", outStr, tt.wantOut)
			}
		})
	}
}
