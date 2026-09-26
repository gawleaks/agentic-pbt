package core

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

const (
	processOutputLimit = 64 * 1024
	pythonVersionQuery = "import json, hypothesis, pytest\nprint(json.dumps({\"hypothesis\": hypothesis.__version__, \"pytest\": pytest.__version__}))"
)

// Service provides read-only task inspection and status operations.
type Service struct {
	Root string
	Run  RunFunc
}

// NewService creates a service for the worktree containing pbt.config.json.
func NewService(root string) *Service {
	return &Service{Root: root, Run: RunProcess}
}

// Inspect reports the configured command and installed Python test-library versions.
func (service *Service) Inspect(ctx context.Context, task string) (InspectResult, error) {
	if err := validateTask(task); err != nil {
		return InspectResult{}, err
	}
	if err := ctx.Err(); err != nil {
		return InspectResult{}, err
	}
	config, err := service.loadConfig()
	if err != nil {
		return InspectResult{}, err
	}

	result := InspectResult{
		SchemaVersion: schemaVersion,
		Task:          task,
		ProjectRoot:   config.ProjectRoot,
		Command:       slices.Clone(config.Command),
		ContractPath:  filepath.Join(config.ProjectRoot, ".pbt", "contracts", task+".json"),
	}
	argv := append(slices.Clone(config.Python), "-c", pythonVersionQuery)
	process := service.run(ctx, argv, config.ProjectRoot, os.Environ(), processOutputLimit)
	if process.ExitCode != 0 || process.Error != "" {
		result.ReasonCode = "python_version_query_failed"
		return result, fmt.Errorf("python version query failed: %s", processError(process))
	}

	var versions struct {
		Hypothesis string `json:"hypothesis"`
		Pytest     string `json:"pytest"`
	}
	if err := decodeJSONBytes([]byte(process.Stdout), &versions); err != nil || strings.TrimSpace(versions.Hypothesis) == "" || strings.TrimSpace(versions.Pytest) == "" {
		result.ReasonCode = "python_version_query_failed"
		if err != nil {
			return result, fmt.Errorf("python version query failed: %w", err)
		}
		return result, fmt.Errorf("python version query failed: missing package version")
	}
	result.HypothesisVersion = versions.Hypothesis
	result.PytestVersion = versions.Pytest
	return result, nil
}

// Status loads a named task or the active task and recomputes freshness.
func (service *Service) Status(ctx context.Context, task string) (Result, error) {
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}
	config, err := service.loadConfig()
	if err != nil {
		return Result{}, err
	}
	if task == "" {
		task, err = loadActiveTask(config.ProjectRoot)
		if err != nil {
			return Result{}, err
		}
	} else if err := validateTask(task); err != nil {
		return Result{}, err
	}

	state, err := LoadState(config.ProjectRoot, task)
	if err != nil {
		return Result{}, err
	}
	verificationState := state.VerificationState
	if state.VerificationState == VerificationPassed {
		fingerprint, err := Fingerprint(config.ProjectRoot, config, filepath.Join(config.ProjectRoot, ".pbt", "contracts", task+".json"))
		if err != nil {
			return Result{}, err
		}
		if fingerprint != state.CurrentFingerprint {
			verificationState = VerificationStale
		}
	}

	return Result{
		SchemaVersion:     schemaVersion,
		Task:              task,
		VerificationState: verificationState,
		WorkflowState:     state.WorkflowState,
		ReasonCode:        statusReason(verificationState),
		NextAction:        statusAction(verificationState),
	}, nil
}

func (service *Service) loadConfig() (*Config, error) {
	if strings.TrimSpace(service.Root) == "" {
		return nil, fmt.Errorf("service root must not be empty")
	}
	return LoadConfig(filepath.Join(service.Root, "pbt.config.json"))
}

func (service *Service) run(ctx context.Context, argv []string, dir string, env []string, limit int) ProcessResult {
	if service.Run == nil {
		return RunProcess(ctx, argv, dir, env, limit)
	}
	return service.Run(ctx, argv, dir, env, limit)
}

func loadActiveTask(root string) (string, error) {
	var active activeTask
	if err := decodeJSON(filepath.Join(root, ".pbt", "state", "active.json"), &active); err != nil {
		return "", err
	}
	if err := validateTask(active.Task); err != nil {
		return "", err
	}
	return active.Task, nil
}

func processError(result ProcessResult) string {
	if result.Error != "" {
		return result.Error
	}
	if result.Stderr != "" {
		return strings.TrimSpace(result.Stderr)
	}
	return fmt.Sprintf("exit code %d", result.ExitCode)
}

func statusReason(state VerificationState) string {
	return "verification_" + string(state)
}

func statusAction(state VerificationState) string {
	switch state {
	case VerificationPassed, VerificationNotApplicable:
		return "none"
	case VerificationStale:
		return "verify_again"
	default:
		return "verify"
	}
}
