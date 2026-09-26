package core

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func TestBeginRecordsDirtyBaselineAndMakesOneTaskActive(t *testing.T) {
	root, _ := stateTestRepository(t)
	writeStateTestFile(t, filepath.Join(root, "src", "value.py"), "value = 2\n")

	result, err := Begin(root, "discount")
	if err != nil {
		t.Fatal(err)
	}
	if result.VerificationState != VerificationPending || result.WorkflowState != WorkflowActive {
		t.Fatalf("result = %#v", result)
	}

	state, err := LoadState(root, "discount")
	if err != nil {
		t.Fatal(err)
	}
	if len(state.Baseline.Modified) != 1 || state.Baseline.Modified[0] != "src/value.py" {
		t.Fatalf("baseline = %#v", state.Baseline)
	}
	if state.CurrentFingerprint == "" {
		t.Fatal("state did not record a fingerprint")
	}
	if _, err := Begin(root, "other"); err == nil {
		t.Fatal("Begin accepted a second active task")
	}

	exclude := readStateTestFile(t, filepath.Join(root, ".git", "info", "exclude"))
	for _, rule := range []string{".pbt/state/", ".pbt/runs/"} {
		if strings.Count(exclude, rule) != 1 {
			t.Fatalf("exclude = %q, want one %q rule", exclude, rule)
		}
	}
}

func TestBeginRejectsActiveTaskName(t *testing.T) {
	root, _ := stateTestRepository(t)

	if _, err := Begin(root, "active"); err == nil {
		t.Fatal("Begin accepted the reserved active task name")
	}
}

func TestBeginUsesResolvedProjectRootForStateAndBaseline(t *testing.T) {
	root, configPath := stateTestRepository(t)
	projectRoot := filepath.Join(root, "app")
	writeStateTestFile(t, configPath, `{
		"schemaVersion": 1,
		"projectRoot": "app",
		"inputs": ["src/**"],
		"python": ["python"],
		"command": ["pytest"],
		"timeoutSeconds": 30
	}`)
	writeStateTestFile(t, filepath.Join(projectRoot, "src", "value.py"), "value = 1\n")
	runStateTestGit(t, root, "add", "pbt.config.json", "app")
	runStateTestGit(t, root, "-c", "user.name=test", "-c", "user.email=test@example.com", "commit", "-m", "add app")
	writeStateTestFile(t, filepath.Join(projectRoot, "src", "value.py"), "value = 2\n")

	if _, err := Begin(root, "discount"); err != nil {
		t.Fatal(err)
	}
	state, err := LoadState(projectRoot, "discount")
	if err != nil {
		t.Fatal(err)
	}
	if len(state.Baseline.Modified) != 1 || state.Baseline.Modified[0] != "src/value.py" {
		t.Fatalf("baseline = %#v", state.Baseline)
	}
	if _, err := os.Stat(filepath.Join(projectRoot, ".pbt", "state", "active.json")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, ".pbt", "state", "active.json")); !os.IsNotExist(err) {
		t.Fatalf("config directory unexpectedly contains active state: %v", err)
	}
}

func TestBeginExcludesNestedLocalArtifacts(t *testing.T) {
	root, configPath := stateTestRepository(t)
	projectRoot := filepath.Join(root, "app")
	writeStateTestFile(t, configPath, `{
		"schemaVersion": 1,
		"projectRoot": "app",
		"inputs": ["**"],
		"python": ["python"],
		"command": ["pytest"],
		"timeoutSeconds": 30
	}`)
	writeStateTestFile(t, filepath.Join(projectRoot, "src", "value.py"), "value = 1\n")
	runStateTestGit(t, root, "add", "pbt.config.json", "app")
	runStateTestGit(t, root, "-c", "user.name=test", "-c", "user.email=test@example.com", "commit", "-m", "add app")

	if _, err := Begin(root, "discount"); err != nil {
		t.Fatal(err)
	}
	if _, err := runGit("-C", root, "check-ignore", "app/.pbt/state/active.json"); err != nil {
		t.Fatalf("nested active state is not excluded: %v", err)
	}
}

func TestBeginAllowsOnlyOneConcurrentTask(t *testing.T) {
	root, _ := stateTestRepository(t)
	const attempts = 16

	start := make(chan struct{})
	results := make(chan error, attempts)
	var group sync.WaitGroup
	for index := range attempts {
		group.Add(1)
		go func() {
			defer group.Done()
			<-start
			_, err := Begin(root, fmt.Sprintf("task-%d", index))
			results <- err
		}()
	}
	close(start)
	group.Wait()
	close(results)

	successes := 0
	for err := range results {
		if err == nil {
			successes++
		}
	}
	if successes != 1 {
		t.Fatalf("successful Begin calls = %d, want 1", successes)
	}
}

func TestSaveStateReplacesTaskState(t *testing.T) {
	root, _ := stateTestRepository(t)
	state := TaskState{
		Task:              "discount",
		VerificationState: VerificationPending,
		WorkflowState:     WorkflowActive,
	}

	if err := SaveState(root, state); err != nil {
		t.Fatal(err)
	}
	state.RepairCount = 1
	if err := SaveState(root, state); err != nil {
		t.Fatal(err)
	}

	loaded, err := LoadState(root, "discount")
	if err != nil {
		t.Fatal(err)
	}
	if loaded.RepairCount != 1 {
		t.Fatalf("RepairCount = %d, want 1", loaded.RepairCount)
	}
}

func stateTestRepository(t *testing.T) (string, string) {
	t.Helper()

	root := t.TempDir()
	runStateTestGit(t, root, "init")
	writeStateTestFile(t, filepath.Join(root, "pbt.config.json"), `{
		"schemaVersion": 1,
		"projectRoot": ".",
		"inputs": ["src/**"],
		"python": ["python"],
		"command": ["pytest"],
		"timeoutSeconds": 30
	}`)
	writeStateTestFile(t, filepath.Join(root, "src", "value.py"), "value = 1\n")
	runStateTestGit(t, root, "add", ".")
	runStateTestGit(t, root, "-c", "user.name=test", "-c", "user.email=test@example.com", "commit", "-m", "initial")

	return root, filepath.Join(root, "pbt.config.json")
}

func runStateTestGit(t *testing.T, root string, args ...string) {
	t.Helper()

	command := append([]string{"-C", root}, args...)
	output, err := runGit(command...)
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, output)
	}
}

func writeStateTestFile(t *testing.T, path, content string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func readStateTestFile(t *testing.T, path string) string {
	t.Helper()

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(content)
}
