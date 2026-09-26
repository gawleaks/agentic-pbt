package core

import (
	"context"
	"os"
	"strings"
	"testing"
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

func TestProcessHelper(t *testing.T) {
	if !strings.HasSuffix(os.Args[0], ".test") || len(os.Args) < 3 || os.Args[len(os.Args)-1] != "output" {
		return
	}
	_, _ = os.Stdout.WriteString("123456789")
	_, _ = os.Stderr.WriteString("abcdefghi")
	os.Exit(0)
}
