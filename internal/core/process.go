package core

import (
	"bytes"
	"context"
	"errors"
	"os/exec"
	"sync"
	"syscall"
	"time"
)

// RunFunc runs a command without invoking a shell.
type RunFunc func(context.Context, []string, string, []string, int) ProcessResult

func replaceEnvironment(env []string, key, value string) []string {
	prefix := key + "="
	updated := make([]string, 0, len(env)+1)
	for _, entry := range env {
		if len(entry) >= len(prefix) && entry[:len(prefix)] == prefix {
			continue
		}
		updated = append(updated, entry)
	}
	return append(updated, prefix+value)
}

// RunProcess executes argv with capped stdout and stderr capture.
func RunProcess(ctx context.Context, argv []string, dir string, env []string, limit int) ProcessResult {
	started := time.Now()
	result := ProcessResult{ExitCode: -1}
	if len(argv) == 0 {
		result.Error = "argv must not be empty"
		result.Duration = time.Since(started)
		return result
	}

	stdout := newCappedWriter(limit)
	stderr := newCappedWriter(limit)
	command := exec.Command(argv[0], argv[1:]...)
	command.Dir = dir
	command.Env = env
	command.Stdout = stdout
	command.Stderr = stderr
	command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := command.Start(); err != nil {
		result.Error = err.Error()
		result.Duration = time.Since(started)
		result.TimedOut = errors.Is(ctx.Err(), context.DeadlineExceeded)
		result.Cancelled = errors.Is(ctx.Err(), context.Canceled)
		return result
	}

	done := make(chan struct{})
	go func(pid int) {
		select {
		case <-ctx.Done():
			_ = syscall.Kill(-pid, syscall.SIGKILL)
		case <-done:
		}
	}(command.Process.Pid)
	err := command.Wait()
	close(done)

	result.Stdout, result.StdoutTruncated = stdout.String()
	result.Stderr, result.StderrTruncated = stderr.String()
	result.Duration = time.Since(started)
	if err == nil {
		result.ExitCode = 0
	} else {
		var exitError *exec.ExitError
		if errors.As(err, &exitError) {
			result.ExitCode = exitError.ExitCode()
		} else {
			result.Error = err.Error()
		}
	}
	result.TimedOut = errors.Is(ctx.Err(), context.DeadlineExceeded)
	result.Cancelled = errors.Is(ctx.Err(), context.Canceled)
	return result
}

type cappedWriter struct {
	buffer    bytes.Buffer
	limit     int
	truncated bool
	mutex     sync.Mutex
}

func newCappedWriter(limit int) *cappedWriter {
	if limit < 0 {
		limit = 0
	}
	return &cappedWriter{limit: limit}
}

func (writer *cappedWriter) Write(data []byte) (int, error) {
	writer.mutex.Lock()
	defer writer.mutex.Unlock()

	remaining := writer.limit - writer.buffer.Len()
	if remaining <= 0 {
		if len(data) > 0 {
			writer.truncated = true
		}
		return len(data), nil
	}
	if len(data) > remaining {
		_, _ = writer.buffer.Write(data[:remaining])
		writer.truncated = true
		return len(data), nil
	}
	_, err := writer.buffer.Write(data)
	return len(data), err
}

func (writer *cappedWriter) String() (string, bool) {
	writer.mutex.Lock()
	defer writer.mutex.Unlock()
	return writer.buffer.String(), writer.truncated
}
