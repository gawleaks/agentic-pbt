package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gawleaks/agentic-pbt/internal/core"
)

func TestRunHelp(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"help"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run code = %d, want 0", code)
	}
	if !strings.Contains(stdout.String(), "Usage: pbt") {
		t.Fatalf("stdout = %q", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}

func TestRunBeginPrintsJSONResult(t *testing.T) {
	root := t.TempDir()
	if output, err := exec.Command("git", "init", root).CombinedOutput(); err != nil {
		t.Fatalf("git init: %v\n%s", err, output)
	}
	config := `{
		"schemaVersion":1,"projectRoot":".","inputs":["src/**"],
		"python":["python"],"command":["pytest"],"timeoutSeconds":30
	}`
	if err := os.WriteFile(filepath.Join(root, "pbt.config.json"), []byte(config), 0o600); err != nil {
		t.Fatal(err)
	}

	original, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(original) })

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"begin", "--task", "discount", "--json"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run code = %d, stderr = %q", code, stderr.String())
	}
	var result core.Result
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("stdout = %q: %v", stdout.String(), err)
	}
	isPending := result.VerificationState == core.VerificationPending
	isActive := result.WorkflowState == core.WorkflowActive
	if result.Task != "discount" || !isPending || !isActive {
		t.Fatalf("result = %#v", result)
	}
}

func TestCLIJSONWritesOneResultToStdout(t *testing.T) {
	root := t.TempDir()
	if output, err := exec.Command("git", "init", root).CombinedOutput(); err != nil {
		t.Fatalf("git init: %v\n%s", err, output)
	}
	config := `{
		"schemaVersion":1,"projectRoot":".","inputs":["src/**"],
		"python":["python"],"command":["pytest"],"timeoutSeconds":30
	}`
	if err := os.WriteFile(filepath.Join(root, "pbt.config.json"), []byte(config), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "src"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "src", "value.py"), []byte("value = 1\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if output, err := exec.Command("git", "-C", root, "add", ".").CombinedOutput(); err != nil {
		t.Fatalf("git add: %v\n%s", err, output)
	}
	if output, err := exec.Command("git", "-C", root, "-c", "user.name=test", "-c", "user.email=test@example.com", "commit", "-m", "initial").CombinedOutput(); err != nil {
		t.Fatalf("git commit: %v\n%s", err, output)
	}
	if _, err := core.Begin(root, "discount"); err != nil {
		t.Fatal(err)
	}

	original, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(original) })

	stdout, stderr := new(bytes.Buffer), new(bytes.Buffer)
	code := Run([]string{"status", "--task", "discount", "--json"}, stdout, stderr)
	if code != 0 || !json.Valid(stdout.Bytes()) || strings.Contains(stdout.String(), "\n{") {
		t.Fatalf("stdout=%q stderr=%q code=%d", stdout.String(), stderr.String(), code)
	}
}

func TestRunRejectsUnknownCommand(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"unknown"}, &stdout, &stderr)
	if code != 3 {
		t.Fatalf("Run code = %d, want 3", code)
	}
	if !strings.Contains(stderr.String(), "unknown command") {
		t.Fatalf("stderr = %q", stderr.String())
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
}
