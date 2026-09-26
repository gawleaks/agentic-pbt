package core

import (
	"context"
	"fmt"
	"path/filepath"
	"time"
)

const repairBudget = 3

// Gate decides whether a task has current verification evidence or needs repair.
func (service *Service) Gate(ctx context.Context, task string) (Result, error) {
	if err := validateTask(task); err != nil {
		return gateSetupResult(task), err
	}
	if err := ctx.Err(); err != nil {
		return gateSetupResult(task), err
	}
	config, err := service.loadConfig()
	if err != nil {
		return gateSetupResult(task), err
	}
	unlock, err := lockGate(ctx, config.ProjectRoot)
	if err != nil {
		return gateSetupResult(task), err
	}
	defer unlock()

	state, err := LoadState(config.ProjectRoot, task)
	if err != nil {
		return gateSetupResult(task), err
	}
	if state.VerificationState == VerificationPassed {
		fingerprint, err := Fingerprint(config.ProjectRoot, config, filepath.Join(config.ProjectRoot, ".pbt", "contracts", task+".json"))
		if err != nil {
			return gateSetupResult(task), err
		}
		if fingerprint != state.CurrentFingerprint {
			state.VerificationState = VerificationStale
			if state.WorkflowState != WorkflowBlocked && state.WorkflowState != WorkflowCancelled {
				state.WorkflowState = WorkflowActive
			}
			if err := SaveState(config.ProjectRoot, state); err != nil {
				return gateSetupResult(task), err
			}
			return gateResult(state), nil
		}
	}
	if state.WorkflowState == WorkflowBlocked || state.WorkflowState == WorkflowCancelled {
		return gateResult(state), nil
	}
	if state.VerificationState == VerificationPassed {
		state.WorkflowState = WorkflowComplete
		if err := SaveState(config.ProjectRoot, state); err != nil {
			return gateSetupResult(task), err
		}
		return gateResult(state), nil
	}
	if state.VerificationState == VerificationNotApplicable {
		state.WorkflowState = WorkflowComplete
		if err := SaveState(config.ProjectRoot, state); err != nil {
			return gateSetupResult(task), err
		}
		return gateResult(state), nil
	}
	if state.WorkflowState != WorkflowActive || state.LastResultID == "" || state.LastGateRunID == state.LastResultID {
		return gateResult(state), nil
	}
	if state.RepairCount >= repairBudget {
		state.LastGateRunID = state.LastResultID
		state.WorkflowState = WorkflowBlocked
		if err := SaveState(config.ProjectRoot, state); err != nil {
			return gateSetupResult(task), err
		}
		return gateResult(state), nil
	}

	state.RepairCount++
	state.LastGateRunID = state.LastResultID
	if err := SaveState(config.ProjectRoot, state); err != nil {
		return gateSetupResult(task), err
	}
	result := gateResult(state)
	result.FollowUp = fmt.Sprintf("%s: inspect %s, then pbt verify --task %s", result.ReasonCode, state.RunDirectory, task)
	return result, nil
}

// Cancel records task cancellation without discarding verification evidence.
func (service *Service) Cancel(ctx context.Context, task string) (Result, error) {
	if err := validateTask(task); err != nil {
		return Result{}, err
	}
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}
	config, err := service.loadConfig()
	if err != nil {
		return Result{}, err
	}
	unlock, err := lockGate(ctx, config.ProjectRoot)
	if err != nil {
		return Result{}, err
	}
	defer unlock()

	state, err := LoadState(config.ProjectRoot, task)
	if err != nil {
		return Result{}, err
	}
	state.WorkflowState = WorkflowCancelled
	if err := SaveState(config.ProjectRoot, state); err != nil {
		return Result{}, err
	}
	return gateResult(state), nil
}

// Resume reactivates a blocked or cancelled task without discarding its evidence.
func (service *Service) Resume(ctx context.Context, task string) (Result, error) {
	if err := validateTask(task); err != nil {
		return Result{}, err
	}
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}
	config, err := service.loadConfig()
	if err != nil {
		return Result{}, err
	}
	unlock, err := lockGate(ctx, config.ProjectRoot)
	if err != nil {
		return Result{}, err
	}
	defer unlock()

	state, err := LoadState(config.ProjectRoot, task)
	if err != nil {
		return Result{}, err
	}
	if state.WorkflowState != WorkflowBlocked && state.WorkflowState != WorkflowCancelled {
		return Result{}, fmt.Errorf("task is not blocked or cancelled")
	}

	state.WorkflowState = WorkflowActive
	state.RepairCount = 0
	state.ResumedAt = time.Now().UTC()
	if err := SaveState(config.ProjectRoot, state); err != nil {
		return Result{}, err
	}
	return gateResult(state), nil
}

func gateSetupResult(task string) Result {
	return Result{
		SchemaVersion: schemaVersion,
		Task:          task,
		ReasonCode:    "setup_error",
		NextAction:    "none",
	}
}

func gateResult(state TaskState) Result {
	reasonCode := gateReason(state)
	nextAction := "verify"
	switch reasonCode {
	case "satisfied", "not_applicable", "verification_blocked":
		nextAction = "none"
	case "verification_failed":
		nextAction = "repair_and_verify"
	case "verification_stale":
		nextAction = "verify_again"
	}
	if state.WorkflowState == WorkflowCancelled {
		nextAction = "none"
	}
	return Result{
		SchemaVersion:     schemaVersion,
		Task:              state.Task,
		VerificationState: state.VerificationState,
		WorkflowState:     state.WorkflowState,
		ReasonCode:        reasonCode,
		NextAction:        nextAction,
	}
}

func gateReason(state TaskState) string {
	if state.WorkflowState == WorkflowBlocked {
		return "verification_blocked"
	}
	switch state.VerificationState {
	case VerificationPassed:
		return "satisfied"
	case VerificationNotApplicable:
		return "not_applicable"
	case VerificationFailed:
		return "verification_failed"
	case VerificationStale:
		return "verification_stale"
	default:
		return "verification_missing"
	}
}
