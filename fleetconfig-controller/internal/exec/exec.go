// Package exec provides utility functions for executing commands.
package exec

import (
	"context"
	"os/exec"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/log"
)

const logInterval = 5 * time.Second

// CmdWithLogs executes the passed in command in a goroutine while the main thread waits for the command to complete.
// The main thread logs a message at regular intervals until the command completes.
func CmdWithLogs(ctx context.Context, cmd *exec.Cmd, message string) ([]byte, error) {
	logger := log.FromContext(ctx)

	resultCh := make(chan struct {
		out []byte
		err error
	}, 1)

	go func() {
		out, err := cmd.CombinedOutput()
		resultCh <- struct {
			out []byte
			err error
		}{out: out, err: err}
	}()

	ticker := time.NewTicker(logInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			_ = cmd.Process.Kill()
			return nil, ctx.Err()
		case res := <-resultCh:
			return res.out, res.err
		case <-ticker.C:
			logger.V(1).Info(message)
		}
	}
}
