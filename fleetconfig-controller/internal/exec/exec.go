// Package exec provides utility functions for executing commands.
package exec

import (
	"bytes"
	"context"
	"os/exec"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/log"
)

const logInterval = 5 * time.Second

// CmdWithLogs executes the passed in command in a goroutine while the main thread waits for the command to complete.
// The main thread logs a message at regular intervals until the command completes.
// Returns stdout, stderr, and error separately.
func CmdWithLogs(ctx context.Context, cmd *exec.Cmd, message string) ([]byte, []byte, error) {
	logger := log.FromContext(ctx)

	resultCh := make(chan struct {
		stdout []byte
		stderr []byte
		err    error
	}, 1)

	go func() {
		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		err := cmd.Run()
		resultCh <- struct {
			stdout []byte
			stderr []byte
			err    error
		}{stdout: stdout.Bytes(), stderr: stderr.Bytes(), err: err}
	}()

	ticker := time.NewTicker(logInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			_ = cmd.Process.Kill()
			return nil, nil, ctx.Err()
		case res := <-resultCh:
			return res.stdout, res.stderr, res.err
		case <-ticker.C:
			logger.V(1).Info(message)
		}
	}
}
