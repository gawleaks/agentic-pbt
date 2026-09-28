package core

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gawleaks/agentic-pbt/internal/pytest"
)

// Verify collects and runs every required pytest property for task.
func (service *Service) Verify(ctx context.Context, task string) (Result, error) {
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
	unlock, locked, err := tryLockGate(config.ProjectRoot)
	if err != nil {
		return Result{}, err
	}
	if !locked {
		state, err := LoadState(config.ProjectRoot, task)
		if err != nil {
			return Result{}, err
		}
		return verificationResult(state, task, VerificationRunning, "verification_running"), nil
	}
	defer unlock()

	contractPath := filepath.Join(config.ProjectRoot, ".pbt", "contracts", task+".json")
	contract, err := LoadContract(contractPath)
	if err != nil {
		return Result{}, err
	}
	if contract.Task != task {
		return Result{}, fmt.Errorf("contract task %q does not match %q", contract.Task, task)
	}
	state, err := LoadState(config.ProjectRoot, task)
	if err != nil {
		return Result{}, err
	}
	state.VerificationState = VerificationRunning
	if err := SaveState(config.ProjectRoot, state); err != nil {
		return Result{}, err
	}

	before, err := Fingerprint(config.ProjectRoot, config, contractPath)
	if err != nil {
		return Result{}, err
	}
	runDirectory, err := os.MkdirTemp(filepath.Join(config.ProjectRoot, ".pbt", "runs"), task+"-")
	if err != nil {
		return Result{}, err
	}
	if contract.Applicability != "not_applicable" {
		if err := service.checkPython(ctx, config); err != nil {
			return service.persistVerification(config, state, task, runDirectory, "python_version_query_failed", before)
		}
	}
	if state.SetupError {
		state.SetupError = false
		if state.RepairCount < repairBudget {
			state.WorkflowState = WorkflowActive
		}
		if err := SaveState(config.ProjectRoot, state); err != nil {
			return Result{}, err
		}
	}
	if contract.Applicability == "not_applicable" {
		return service.persistVerification(config, state, task, runDirectory, "verification_not_applicable", before)
	}
	if err := os.WriteFile(filepath.Join(runDirectory, "pbt_pytest_reporter.py"), []byte(pytest.Reporter), 0o600); err != nil {
		return Result{}, err
	}

	collectionContext, cancelCollection := context.WithTimeout(ctx, time.Duration(config.TimeoutSeconds)*time.Second)
	collection := service.run(
		collectionContext,
		append(append([]string{}, config.Command...), "--collect-only", "-q"),
		config.ProjectRoot,
		os.Environ(),
		processOutputLimit,
	)
	cancelCollection()
	if err := writeRunOutput(runDirectory, collection); err != nil {
		return Result{}, err
	}
	if collection.TimedOut {
		return service.persistVerification(config, state, task, runDirectory, "pytest_timed_out", "")
	}
	if collection.ExitCode != 0 || collection.Error != "" {
		return service.persistVerification(config, state, task, runDirectory, "pytest_collection_failed", "")
	}
	if !collectedRequiredProperties(collection.Stdout, contract.Properties) {
		return service.persistVerification(config, state, task, runDirectory, "required_test_not_collected", "")
	}

	nodeResultsPath := filepath.Join(runDirectory, "node-results.jsonl")
	junitPath := filepath.Join(runDirectory, "junit.xml")
	artifactLimitMarkerPath := filepath.Join(runDirectory, "artifact-limit-exceeded")
	env := replaceEnvironment(os.Environ(), "PYTHONPATH", prependPath(runDirectory, os.Getenv("PYTHONPATH")))
	env = replaceEnvironment(env, "PBT_NODE_RESULTS", nodeResultsPath)
	env = replaceEnvironment(env, "PBT_RUN_ARTIFACT_LIMIT", fmt.Sprint(runArtifactLimit))
	env = replaceEnvironment(env, "PBT_ARTIFACT_LIMIT_MARKER", artifactLimitMarkerPath)
	executionContext, cancelExecution := context.WithTimeout(ctx, time.Duration(config.TimeoutSeconds)*time.Second)
	execution := service.run(
		executionContext,
		append(append([]string{}, config.Command...), "--junitxml", junitPath, "-p", "pbt_pytest_reporter"),
		config.ProjectRoot,
		env,
		processOutputLimit,
	)
	cancelExecution()
	if err := writeRunOutput(runDirectory, execution); err != nil {
		return Result{}, err
	}

	artifactLimitExceeded, err := enforceArtifactLimit(junitPath, nodeResultsPath)
	if err != nil {
		return Result{}, err
	}
	if _, err := os.Stat(artifactLimitMarkerPath); err == nil {
		artifactLimitExceeded = true
	} else if !os.IsNotExist(err) {
		return Result{}, err
	}
	after, err := Fingerprint(config.ProjectRoot, config, contractPath)
	if err != nil {
		return Result{}, err
	}
	if before != after {
		return service.persistVerification(config, state, task, runDirectory, "stale", after)
	}
	if artifactLimitExceeded {
		return service.persistVerification(config, state, task, runDirectory, "artifact_limit_exceeded", after)
	}
	if execution.TimedOut {
		return service.persistVerification(config, state, task, runDirectory, "pytest_timed_out", after)
	}
	if execution.ExitCode != 0 || execution.Error != "" {
		return service.persistVerification(config, state, task, runDirectory, "pytest_failed", after)
	}
	if err := validateJUnit(junitPath); err != nil {
		return service.persistVerification(config, state, task, runDirectory, "junit_invalid", after)
	}
	nodeResults, err := parseNodeResults(nodeResultsPath)
	if err != nil {
		return service.persistVerification(config, state, task, runDirectory, "node_results_invalid", after)
	}
	for _, property := range contract.Properties {
		outcome, found := nodeResults[property.Test]
		if !found {
			return service.persistVerification(config, state, task, runDirectory, "required_test_missing_result", after)
		}
		if outcome == "passed" {
			continue
		}
		if outcome == "skipped" {
			return service.persistVerification(config, state, task, runDirectory, "required_test_skipped", after)
		}
		return service.persistVerification(config, state, task, runDirectory, "required_test_failed", after)
	}
	return service.persistVerification(config, state, task, runDirectory, "verification_passed", after)
}

func (service *Service) checkPython(ctx context.Context, config *Config) error {
	argv := append(append([]string{}, config.Python...), "-c", pythonVersionQuery)
	process := service.run(ctx, argv, config.ProjectRoot, os.Environ(), processOutputLimit)
	if process.ExitCode != 0 || process.Error != "" {
		return fmt.Errorf("python version query failed: %s", processError(process))
	}
	var versions struct {
		Hypothesis string `json:"hypothesis"`
		Pytest     string `json:"pytest"`
	}
	if err := decodeJSONBytes([]byte(process.Stdout), &versions); err != nil || strings.TrimSpace(versions.Hypothesis) == "" || strings.TrimSpace(versions.Pytest) == "" {
		if err != nil {
			return fmt.Errorf("python version query failed: %w", err)
		}
		return fmt.Errorf("python version query failed: missing package version")
	}
	return nil
}

func (service *Service) persistVerification(config *Config, state TaskState, task, runDirectory, reasonCode, fingerprint string) (Result, error) {
	verificationState := VerificationFailed
	switch reasonCode {
	case "verification_passed":
		verificationState = VerificationPassed
	case "stale":
		verificationState = VerificationStale
	case "verification_not_applicable":
		verificationState = VerificationNotApplicable
	}
	if reasonCode == "python_version_query_failed" {
		state.WorkflowState = WorkflowBlocked
		state.SetupError = true
	}
	result := verificationResult(state, task, verificationState, reasonCode)
	if err := writeJSONAtomic(filepath.Join(runDirectory, "result.json"), result); err != nil {
		return Result{}, err
	}
	if fingerprint == "" {
		var err error
		fingerprint, err = Fingerprint(config.ProjectRoot, config, filepath.Join(config.ProjectRoot, ".pbt", "contracts", task+".json"))
		if err != nil {
			return Result{}, err
		}
	}
	state.CurrentFingerprint = fingerprint
	state.LastResultID = filepath.Base(runDirectory)
	state.RunDirectory = runDirectory
	state.VerificationState = verificationState
	if err := SaveState(config.ProjectRoot, state); err != nil {
		return Result{}, err
	}
	return result, nil
}

func enforceArtifactLimit(paths ...string) (bool, error) {
	exceeded := false
	for _, path := range paths {
		info, err := os.Stat(path)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return false, err
		}
		if info.Size() <= runArtifactLimit {
			continue
		}
		if err := os.Truncate(path, runArtifactLimit); err != nil {
			return false, err
		}
		exceeded = true
	}
	return exceeded, nil
}

func verificationResult(state TaskState, task string, verificationState VerificationState, reasonCode string) Result {
	nextAction := "repair_and_verify"
	if verificationState == VerificationPassed || verificationState == VerificationNotApplicable {
		nextAction = "none"
	}
	if verificationState == VerificationStale {
		nextAction = "verify_again"
	}
	return Result{
		SchemaVersion:     schemaVersion,
		Task:              task,
		VerificationState: verificationState,
		WorkflowState:     state.WorkflowState,
		ReasonCode:        reasonCode,
		NextAction:        nextAction,
	}
}

func collectedRequiredProperties(output string, properties []RequiredProperty) bool {
	collected := map[string]bool{}
	for _, line := range strings.Split(output, "\n") {
		collected[strings.TrimSuffix(line, "\r")] = true
	}
	for _, property := range properties {
		if !collected[property.Test] {
			return false
		}
	}
	return true
}

func prependPath(path, existing string) string {
	if existing == "" {
		return path
	}
	return path + string(os.PathListSeparator) + existing
}

func writeRunOutput(runDirectory string, process ProcessResult) error {
	if err := os.WriteFile(filepath.Join(runDirectory, "stdout.txt"), []byte(process.Stdout), 0o600); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(runDirectory, "stderr.txt"), []byte(process.Stderr), 0o600)
}

func validateJUnit(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	decoder := xml.NewDecoder(file)
	hasElement := false
	for {
		token, err := decoder.Token()
		if err != nil {
			if err == io.EOF && hasElement {
				return nil
			}
			if err == io.EOF {
				return fmt.Errorf("empty JUnit XML")
			}
			return err
		}
		if _, ok := token.(xml.StartElement); ok {
			hasElement = true
		}
	}
}
