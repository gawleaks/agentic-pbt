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
	if contract.Applicability == "not_applicable" {
		result := verificationResult(state, task, VerificationNotApplicable, "verification_not_applicable")
		state.VerificationState = result.VerificationState
		if err := SaveState(config.ProjectRoot, state); err != nil {
			return Result{}, err
		}
		return result, nil
	}

	before, err := Fingerprint(config.ProjectRoot, config, contractPath)
	if err != nil {
		return Result{}, err
	}
	runDirectory, err := os.MkdirTemp(filepath.Join(config.ProjectRoot, ".pbt", "runs"), task+"-")
	if err != nil {
		return Result{}, err
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
		return service.persistVerification(config, state, task, runDirectory, "pytest_timed_out")
	}
	if collection.ExitCode != 0 || collection.Error != "" {
		return service.persistVerification(config, state, task, runDirectory, "pytest_collection_failed")
	}
	if !collectedRequiredProperties(collection.Stdout, contract.Properties) {
		return service.persistVerification(config, state, task, runDirectory, "required_test_not_collected")
	}

	nodeResultsPath := filepath.Join(runDirectory, "node-results.jsonl")
	junitPath := filepath.Join(runDirectory, "junit.xml")
	env := replaceEnvironment(os.Environ(), "PYTHONPATH", prependPath(runDirectory, os.Getenv("PYTHONPATH")))
	env = replaceEnvironment(env, "PBT_NODE_RESULTS", nodeResultsPath)
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

	after, err := Fingerprint(config.ProjectRoot, config, contractPath)
	if err != nil {
		return Result{}, err
	}
	if before != after {
		return service.persistVerification(config, state, task, runDirectory, "stale")
	}
	if execution.TimedOut {
		return service.persistVerification(config, state, task, runDirectory, "pytest_timed_out")
	}
	if execution.ExitCode != 0 || execution.Error != "" {
		return service.persistVerification(config, state, task, runDirectory, "pytest_failed")
	}
	if err := validateJUnit(junitPath); err != nil {
		return service.persistVerification(config, state, task, runDirectory, "junit_invalid")
	}
	nodeResults, err := parseNodeResults(nodeResultsPath)
	if err != nil {
		return service.persistVerification(config, state, task, runDirectory, "node_results_invalid")
	}
	for _, property := range contract.Properties {
		outcome, found := nodeResults[property.Test]
		if !found {
			return service.persistVerification(config, state, task, runDirectory, "required_test_missing_result")
		}
		if outcome == "passed" {
			continue
		}
		if outcome == "skipped" {
			return service.persistVerification(config, state, task, runDirectory, "required_test_skipped")
		}
		return service.persistVerification(config, state, task, runDirectory, "required_test_failed")
	}
	return service.persistVerification(config, state, task, runDirectory, "verification_passed")
}

func (service *Service) persistVerification(config *Config, state TaskState, task, runDirectory, reasonCode string) (Result, error) {
	verificationState := VerificationFailed
	if reasonCode == "verification_passed" {
		verificationState = VerificationPassed
	}
	if reasonCode == "stale" {
		verificationState = VerificationStale
	}
	result := verificationResult(state, task, verificationState, reasonCode)
	if err := writeJSONAtomic(filepath.Join(runDirectory, "result.json"), result); err != nil {
		return Result{}, err
	}
	fingerprint, err := Fingerprint(config.ProjectRoot, config, filepath.Join(config.ProjectRoot, ".pbt", "contracts", task+".json"))
	if err != nil {
		return Result{}, err
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
