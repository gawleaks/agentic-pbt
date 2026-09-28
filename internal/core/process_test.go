package core

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestRunProcessCapturesBoundedOutput(t *testing.T) {
	result := RunProcess(context.Background(), []string{os.Args[0], "-test.run=TestProcessHelper", "--", "output"}, "", nil, 8)

	if result.ExitCode != 0 {
		t.Fatalf("exit code = %d, stderr = %q", result.ExitCode, result.Stderr)
	}
	if result.Stdout != "12345678" || !result.StdoutTruncated {
		t.Fatalf("stdout = %q, truncated = %t", result.Stdout, result.StdoutTruncated)
	}
	if result.Stderr != "abcdefgh" || !result.StderrTruncated {
		t.Fatalf("stderr = %q, truncated = %t", result.Stderr, result.StderrTruncated)
	}
}

func TestRunProcessRejectsEmptyArguments(t *testing.T) {
	result := RunProcess(context.Background(), []string{}, "", nil, 8)

	if result.Error == "" || !strings.Contains(result.Error, "argv") {
		t.Fatalf("error = %q", result.Error)
	}
}

func TestRunProcessKillsTimedOutProcessGroup(t *testing.T) {
	pidPath := filepath.Join(t.TempDir(), "child.pid")
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	result := RunProcess(ctx, []string{os.Args[0], "-test.run=TestProcessHelper", "--", "spawn", pidPath}, "", nil, 1024)
	if !result.TimedOut {
		t.Fatalf("result = %#v", result)
	}
	content, err := os.ReadFile(pidPath)
	if err != nil {
		t.Fatal(err)
	}
	var pid int
	if _, err := fmt.Sscanf(string(content), "%d", &pid); err != nil {
		t.Fatal(err)
	}
	if err := syscall.Kill(pid, 0); err == nil {
		_ = syscall.Kill(pid, syscall.SIGKILL)
		t.Fatalf("child process %d survived timeout", pid)
	}
}

func TestProcessHelper(t *testing.T) {
	if !strings.HasSuffix(os.Args[0], ".test") {
		return
	}
	separator := -1
	for index, argument := range os.Args {
		if argument == "--" {
			separator = index
			break
		}
	}
	if separator == -1 || separator+1 == len(os.Args) {
		return
	}
	switch os.Args[separator+1] {
	case "output":
		_, _ = os.Stdout.WriteString("123456789")
		_, _ = os.Stderr.WriteString("abcdefghi")
		os.Exit(0)
	case "hold":
		time.Sleep(10 * time.Second)
	default:
		if separator+2 == len(os.Args) {
			os.Exit(1)
		}
		pidPath := os.Args[separator+2]
		child := exec.Command(os.Args[0], "-test.run=TestProcessHelper", "--", "hold")
		child.Stdout = os.Stdout
		child.Stderr = os.Stderr
		if err := child.Start(); err != nil {
			os.Exit(1)
		}
		_ = os.WriteFile(pidPath, []byte(fmt.Sprintf("%d", child.Process.Pid)), 0o600)
		time.Sleep(10 * time.Second)
	}
}
