package cli

import (
	"bytes"
	"encoding/json"
	"io"
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

func TestInspectJSONWritesFailedVersionResult(t *testing.T) {
	root := t.TempDir()
	if output, err := exec.Command("git", "init", root).CombinedOutput(); err != nil {
		t.Fatalf("git init: %v\n%s", err, output)
	}
	config := `{
		"schemaVersion":1,"projectRoot":".","inputs":["src/**"],
		"python":["false"],"command":["pytest"],"timeoutSeconds":30
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
	code := Run([]string{"inspect", "--task", "discount", "--json"}, &stdout, &stderr)
	if code != 3 {
		t.Fatalf("Run code = %d, want 3", code)
	}
	if !strings.Contains(stderr.String(), "python version query failed") {
		t.Fatalf("stderr = %q", stderr.String())
	}

	decoder := json.NewDecoder(&stdout)
	var result core.InspectResult
	if err := decoder.Decode(&result); err != nil {
		t.Fatalf("stdout = %q: %v", stdout.String(), err)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		t.Fatalf("stdout contains more than one result: %q", stdout.String())
	}
	if result.SchemaVersion != 1 || result.ReasonCode != "python_version_query_failed" {
		t.Fatalf("result = %#v", result)
	}
}

func TestVerifyJSONWritesVerificationFailure(t *testing.T) {
	root := t.TempDir()
	if output, err := exec.Command("git", "init", root).CombinedOutput(); err != nil {
		t.Fatalf("git init: %v\n%s", err, output)
	}
	config := `{
		"schemaVersion":1,"projectRoot":".","inputs":["src/**"],
		"python":["python"],"command":["false"],"timeoutSeconds":30
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
	contract := `{
		"schemaVersion":1,"task":"discount","applicability":"required","behavior":"discount",
		"properties":[{"id":"discount","statement":"total","generator":"amounts","oracle":"expected","test":"tests/test_discount.py::test_total"}]
	}`
	if err := os.MkdirAll(filepath.Join(root, ".pbt", "contracts"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".pbt", "contracts", "discount.json"), []byte(contract), 0o600); err != nil {
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

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"verify", "--task", "discount", "--json"}, &stdout, &stderr)
	var result core.Result
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("stdout = %q: %v", stdout.String(), err)
	}
	if code != 1 || result.ReasonCode != "pytest_collection_failed" || stderr.Len() != 0 {
		t.Fatalf("code=%d result=%#v stderr=%q", code, result, stderr.String())
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

func TestGateExitCodes(t *testing.T) {
	tests := []struct {
		name         string
		verification core.VerificationState
		workflow     core.WorkflowState
		want         int
	}{
		{name: "complete", verification: core.VerificationPassed, workflow: core.WorkflowActive, want: 0},
		{name: "failed", verification: core.VerificationFailed, workflow: core.WorkflowActive, want: 1},
		{name: "pending", verification: core.VerificationPending, workflow: core.WorkflowActive, want: 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := gateCLIRoot(t, tt.verification, tt.workflow)
			withCLIRoot(t, root)

			var stdout bytes.Buffer
			var stderr bytes.Buffer
			if got := Run([]string{"gate", "--task", "discount", "--json"}, &stdout, &stderr); got != tt.want {
				t.Fatalf("Run code = %d, want %d; stdout=%q stderr=%q", got, tt.want, stdout.String(), stderr.String())
			}
		})
	}

	t.Run("setup error", func(t *testing.T) {
		withCLIRoot(t, t.TempDir())
		var stdout bytes.Buffer
		var stderr bytes.Buffer
		if got := Run([]string{"gate", "--task", "discount", "--json"}, &stdout, &stderr); got != 3 {
			t.Fatalf("Run code = %d, want 3; stdout=%q stderr=%q", got, stdout.String(), stderr.String())
		}
		var result core.Result
		if err := json.Unmarshal(stdout.Bytes(), &result); err != nil || result.ReasonCode != "setup_error" {
			t.Fatalf("stdout = %q, result = %#v, err = %v", stdout.String(), result, err)
		}
	})
}

func TestResumeCommandReactivatesBlockedTask(t *testing.T) {
	root := gateCLIRoot(t, core.VerificationFailed, core.WorkflowBlocked)
	withCLIRoot(t, root)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"resume", "--task", "discount", "--json"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run code = %d, stderr = %q", code, stderr.String())
	}
	var result core.Result
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("stdout = %q: %v", stdout.String(), err)
	}
	if result.WorkflowState != core.WorkflowActive {
		t.Fatalf("result = %#v", result)
	}
}

func gateCLIRoot(t *testing.T, verification core.VerificationState, workflow core.WorkflowState) string {
	t.Helper()

	root := t.TempDir()
	resolvedRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}
	root = resolvedRoot
	if output, err := exec.Command("git", "init", root).CombinedOutput(); err != nil {
		t.Fatalf("git init: %v\n%s", err, output)
	}
	config := `{
		"schemaVersion":1,"projectRoot":".","inputs":["src/**"],
		"python":["python"],"command":["pytest"],"timeoutSeconds":30
	}`
	contract := `{
		"schemaVersion":1,"task":"discount","applicability":"required","behavior":"discount",
		"properties":[{"id":"discount","statement":"total","generator":"amounts","oracle":"expected","test":"tests/test_discount.py::test_total"}]
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
	if err := os.MkdirAll(filepath.Join(root, ".pbt", "contracts"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".pbt", "contracts", "discount.json"), []byte(contract), 0o600); err != nil {
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
	state, err := core.LoadState(root, "discount")
	if err != nil {
		t.Fatal(err)
	}
	state.VerificationState = verification
	state.WorkflowState = workflow
	state.LastResultID = "run-1"
	state.RunDirectory = filepath.Join(root, ".pbt", "runs", "run-1")
	if verification == core.VerificationPassed {
		config, err := core.LoadConfig(filepath.Join(root, "pbt.config.json"))
		if err != nil {
			t.Fatal(err)
		}
		state.CurrentFingerprint, err = core.Fingerprint(config.ProjectRoot, config, filepath.Join(config.ProjectRoot, ".pbt", "contracts", "discount.json"))
		if err != nil {
			t.Fatal(err)
		}
	}
	if err := core.SaveState(root, state); err != nil {
		t.Fatal(err)
	}
	return root
}

func withCLIRoot(t *testing.T, root string) {
	t.Helper()

	original, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(original) })
}
