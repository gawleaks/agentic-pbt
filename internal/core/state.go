package core

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

type activeTask struct {
	Task string `json:"task"`
}

// Begin records the current relevant changes and makes task active.
func Begin(root, task string) (Result, error) {
	if err := validateTask(task); err != nil {
		return Result{}, err
	}

	root, err := filepath.Abs(root)
	if err != nil {
		return Result{}, err
	}
	config, err := LoadConfig(filepath.Join(root, "pbt.config.json"))
	if err != nil {
		return Result{}, err
	}
	projectRoot := config.ProjectRoot
	if err := addLocalExcludes(projectRoot); err != nil {
		return Result{}, err
	}
	if err := os.MkdirAll(filepath.Join(projectRoot, ".pbt", "runs"), 0o755); err != nil {
		return Result{}, err
	}
	if err := createActiveTask(projectRoot, task); err != nil {
		return Result{}, err
	}
	keepActiveTask := false
	defer func() {
		if !keepActiveTask {
			_ = os.Remove(filepath.Join(projectRoot, ".pbt", "state", "active.json"))
		}
	}()

	baseline, err := dirtyBaseline(projectRoot, config.Inputs)
	if err != nil {
		return Result{}, err
	}
	fingerprint, err := Fingerprint(projectRoot, config, filepath.Join(projectRoot, ".pbt", "contracts", task+".json"))
	if err != nil {
		return Result{}, err
	}
	state := TaskState{
		SchemaVersion:      schemaVersion,
		Task:               task,
		Baseline:           baseline,
		CurrentFingerprint: fingerprint,
		RepairCount:        0,
		LastResultID:       "",
		VerificationState:  VerificationPending,
		WorkflowState:      WorkflowActive,
		RunDirectory:       "",
	}
	if err := SaveState(projectRoot, state); err != nil {
		return Result{}, err
	}
	keepActiveTask = true

	return Result{
		SchemaVersion:     schemaVersion,
		Task:              task,
		VerificationState: VerificationPending,
		WorkflowState:     WorkflowActive,
		ReasonCode:        "verification_pending",
		NextAction:        "create_contract_and_verify",
	}, nil
}

// LoadState reads the named task's local state.
func LoadState(root, task string) (TaskState, error) {
	if err := validateTask(task); err != nil {
		return TaskState{}, err
	}

	var state TaskState
	if err := decodeJSON(filepath.Join(root, ".pbt", "state", task+".json"), &state); err != nil {
		return TaskState{}, err
	}
	return state, nil
}

// SaveState atomically replaces a task's local state.
func SaveState(root string, state TaskState) error {
	if err := validateTask(state.Task); err != nil {
		return err
	}
	return writeJSONAtomic(filepath.Join(root, ".pbt", "state", state.Task+".json"), state)
}

func createActiveTask(root, task string) error {
	path := filepath.Join(root, ".pbt", "state", "active.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if os.IsExist(err) {
		return fmt.Errorf("an active task already exists")
	}
	if err != nil {
		return err
	}
	if err := json.NewEncoder(file).Encode(activeTask{Task: task}); err != nil {
		_ = file.Close()
		_ = os.Remove(path)
		return err
	}
	if err := file.Close(); err != nil {
		_ = os.Remove(path)
		return err
	}
	return nil
}

func addLocalExcludes(root string) error {
	output, err := runGit("-C", root, "rev-parse", "--git-path", "info/exclude")
	if err != nil {
		return err
	}
	excludePath := strings.TrimSpace(output)
	if !filepath.IsAbs(excludePath) {
		excludePath = filepath.Join(root, excludePath)
	}
	if err := os.MkdirAll(filepath.Dir(excludePath), 0o755); err != nil {
		return err
	}

	content, err := os.ReadFile(excludePath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	repositoryRoot, err := gitRepositoryRoot(root)
	if err != nil {
		return err
	}
	projectPath, err := filepath.Rel(repositoryRoot, root)
	if err != nil {
		return err
	}
	prefix := ""
	if projectPath != "." {
		prefix = filepath.ToSlash(projectPath) + "/"
	}

	lines := strings.Split(string(content), "\n")
	rules := []string{prefix + ".pbt/state/", prefix + ".pbt/runs/"}
	for _, rule := range rules {
		found := false
		for _, line := range lines {
			if line == rule {
				found = true
				break
			}
		}
		if found {
			continue
		}
		if len(content) > 0 && !strings.HasSuffix(string(content), "\n") {
			content = append(content, '\n')
		}
		content = append(content, rule+"\n"...)
		lines = append(lines, rule)
	}
	return os.WriteFile(excludePath, content, 0o600)
}

func dirtyBaseline(root string, inputs []string) (DirtyBaseline, error) {
	output, err := runGit("-C", root, "status", "--porcelain=v1", "-z", "--untracked-files=all")
	if err != nil {
		return DirtyBaseline{}, err
	}
	repositoryRoot, err := gitRepositoryRoot(root)
	if err != nil {
		return DirtyBaseline{}, err
	}
	baseline := DirtyBaseline{
		Modified:  []string{},
		Untracked: []string{},
		Deleted:   []string{},
	}
	entries := strings.Split(output, "\x00")
	for index := 0; index < len(entries); index++ {
		entry := entries[index]
		if len(entry) < 4 {
			continue
		}
		status := entry[:2]
		name := filepath.ToSlash(entry[3:])
		if status[0] == 'R' || status[0] == 'C' {
			index++
		}
		name, insideProjectRoot, err := projectRelativePath(repositoryRoot, root, name)
		if err != nil {
			return DirtyBaseline{}, err
		}
		if !insideProjectRoot {
			continue
		}
		matches, err := matchesInputs(name, inputs)
		if err != nil {
			return DirtyBaseline{}, err
		}
		if !matches {
			continue
		}

		switch {
		case status == "??":
			baseline.Untracked = append(baseline.Untracked, name)
		case status[0] == 'D' || status[1] == 'D':
			baseline.Deleted = append(baseline.Deleted, name)
		default:
			baseline.Modified = append(baseline.Modified, name)
		}
	}
	sort.Strings(baseline.Modified)
	sort.Strings(baseline.Untracked)
	sort.Strings(baseline.Deleted)
	return baseline, nil
}

func validateTask(task string) error {
	if strings.TrimSpace(task) == "" || task == "." || task == "active" || strings.ContainsAny(task, `/\\`) {
		return fmt.Errorf("task must be a non-empty file name")
	}
	return nil
}

func writeJSONAtomic(path string, value any) error {
	content, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	content = append(content, '\n')
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	temporary, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+"-*")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if err := temporary.Chmod(0o600); err != nil {
		temporary.Close()
		return err
	}
	if _, err := temporary.Write(content); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	return os.Rename(temporaryPath, path)
}

func runGit(args ...string) (string, error) {
	output, err := exec.Command("git", args...).CombinedOutput()
	if err != nil {
		return string(output), fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}
	return string(output), nil
}
