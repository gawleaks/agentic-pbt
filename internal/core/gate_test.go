package core

import (
	"context"
	"fmt"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
)

func TestGateBlocksAfterThreeDistinctRepairDecisions(t *testing.T) {
	service, root := gateTestService(t)
	seedFailedGateState(t, root, "run-1")

	for index := 0; index < 3; index++ {
		got := mustGate(t, service, "discount")
		if got.FollowUp == "" || got.WorkflowState != WorkflowActive {
			t.Fatalf("gate %d = %#v", index, got)
		}
		seedFailedGateState(t, root, fmt.Sprintf("run-%d", index+2))
	}

	got := mustGate(t, service, "discount")
	if got.WorkflowState != WorkflowBlocked || got.FollowUp != "" || got.ReasonCode != "verification_blocked" {
		t.Fatalf("blocked gate = %#v", got)
	}
}

func TestGateConcurrentCallsIssueOneFollowUpAndSpendOneRepair(t *testing.T) {
	service, root := gateTestService(t)
	seedFailedGateState(t, root, "run-1")

	const calls = 16
	start := make(chan struct{})
	results := make(chan Result, calls)
	errors := make(chan error, calls)
	var group sync.WaitGroup
	for range calls {
		group.Add(1)
		go func() {
			defer group.Done()
			<-start
			result, err := service.Gate(context.Background(), "discount")
			if err != nil {
				errors <- err
				return
			}
			results <- result
		}()
	}
	close(start)
	group.Wait()
	close(results)
	close(errors)

	for err := range errors {
		t.Fatal(err)
	}
	followUps := 0
	for result := range results {
		if result.FollowUp != "" {
			followUps++
		}
	}
	if followUps != 1 {
		t.Fatalf("follow-ups = %d, want 1", followUps)
	}
	state := gateState(t, root)
	if state.RepairCount != 1 {
		t.Fatalf("repair count = %d, want 1", state.RepairCount)
	}
}

func TestGateDoesNotSpendBudgetTwiceForSameRun(t *testing.T) {
	service, root := gateTestService(t)
	seedFailedGateState(t, root, "run-1")

	first := mustGate(t, service, "discount")
	second := mustGate(t, service, "discount")
	if first.FollowUp == "" || second.FollowUp != "" {
		t.Fatalf("first=%#v second=%#v", first, second)
	}
	state, err := LoadState(root, "discount")
	if err != nil {
		t.Fatal(err)
	}
	if state.RepairCount != 1 {
		t.Fatalf("repair count = %d, want 1", state.RepairCount)
	}
}

func TestGatePromptsOnceForPendingVerification(t *testing.T) {
	service, root := gateTestService(t)

	first := mustGate(t, service, "discount")
	second := mustGate(t, service, "discount")
	if first.ReasonCode != "verification_missing" || first.FollowUp == "" {
		t.Fatalf("first gate = %#v", first)
	}
	if second.FollowUp != "" {
		t.Fatalf("second gate = %#v", second)
	}
	if state := gateState(t, root); state.RepairCount != 1 {
		t.Fatalf("repair count = %d, want 1", state.RepairCount)
	}
}

func TestGateCompletesCurrentPassedVerification(t *testing.T) {
	service, root := gateTestService(t)
	state := gateState(t, root)
	state.VerificationState = VerificationPassed
	state.CurrentFingerprint = gateFingerprint(t, root)
	setGateState(t, root, state)

	got := mustGate(t, service, "discount")
	if got.VerificationState != VerificationPassed || got.WorkflowState != WorkflowComplete || got.ReasonCode != "satisfied" || got.FollowUp != "" {
		t.Fatalf("gate = %#v", got)
	}
}

func TestGateMarksBlockedPassedEvidenceStale(t *testing.T) {
	service, root := gateTestService(t)
	state := gateState(t, root)
	state.VerificationState = VerificationPassed
	state.WorkflowState = WorkflowBlocked
	state.CurrentFingerprint = gateFingerprint(t, root)
	setGateState(t, root, state)
	writeStateTestFile(t, filepath.Join(root, "src", "value.py"), "value = 2\n")

	got := mustGate(t, service, "discount")
	if got.VerificationState != VerificationStale || got.WorkflowState != WorkflowBlocked {
		t.Fatalf("gate = %#v", got)
	}
	persisted := gateState(t, root)
	if persisted.VerificationState != VerificationStale || persisted.WorkflowState != WorkflowBlocked {
		t.Fatalf("state = %#v", persisted)
	}
}

func TestGateMarksChangedPassedVerificationStale(t *testing.T) {
	service, root := gateTestService(t)
	state := gateState(t, root)
	state.VerificationState = VerificationPassed
	state.CurrentFingerprint = gateFingerprint(t, root)
	setGateState(t, root, state)
	writeStateTestFile(t, filepath.Join(root, "src", "value.py"), "value = 2\n")

	got := mustGate(t, service, "discount")
	if got.VerificationState != VerificationStale || got.ReasonCode != "verification_stale" {
		t.Fatalf("gate = %#v", got)
	}
	if state := gateState(t, root); state.VerificationState != VerificationStale {
		t.Fatalf("state = %#v", state)
	}
}

func TestGatePromptsImmediatelyForStalePassedEvidence(t *testing.T) {
	service, root := gateTestService(t)
	state := gateState(t, root)
	state.VerificationState = VerificationPassed
	state.CurrentFingerprint = gateFingerprint(t, root)
	state.LastResultID = "run-1"
	state.RunDirectory = filepath.Join(root, ".pbt", "runs", "run-1")
	setGateState(t, root, state)
	writeStateTestFile(t, filepath.Join(root, "src", "value.py"), "value = 2\n")

	got := mustGate(t, service, "discount")
	if got.VerificationState != VerificationStale || got.FollowUp == "" {
		t.Fatalf("gate = %#v", got)
	}
}

func TestGateRevalidatesNotApplicableEvidence(t *testing.T) {
	service, root := gateTestService(t)
	contractPath := filepath.Join(root, ".pbt", "contracts", "discount.json")
	writeStateTestFile(t, contractPath, `{"schemaVersion":1,"task":"discount","applicability":"not_applicable","reason":"docs only"}`)
	if _, err := service.Verify(context.Background(), "discount"); err != nil {
		t.Fatal(err)
	}
	writeStateTestFile(t, contractPath, `{
		"schemaVersion":1,"task":"discount","applicability":"required","behavior":"discount",
		"properties":[{"id":"discount","statement":"total","generator":"amounts","oracle":"expected","test":"tests/test_discount.py::test_total"}]
	}`)

	got := mustGate(t, service, "discount")
	if got.VerificationState != VerificationStale || got.FollowUp == "" {
		t.Fatalf("gate = %#v", got)
	}
}

func TestGateCompletesNotApplicableVerification(t *testing.T) {
	service, root := gateTestService(t)
	state := gateState(t, root)
	state.VerificationState = VerificationNotApplicable
	setGateState(t, root, state)

	got := mustGate(t, service, "discount")
	if got.VerificationState != VerificationNotApplicable || got.WorkflowState != WorkflowComplete || got.ReasonCode != "not_applicable" {
		t.Fatalf("gate = %#v", got)
	}
}

func TestCancelPreservesEvidenceAndSuppressesFollowUps(t *testing.T) {
	service, root := gateTestService(t)
	seedFailedGateState(t, root, "run-1")
	before := gateState(t, root)

	got, err := service.Cancel(context.Background(), "discount")
	if err != nil {
		t.Fatal(err)
	}
	if got.WorkflowState != WorkflowCancelled {
		t.Fatalf("cancel = %#v", got)
	}
	cancelled := gateState(t, root)
	if cancelled.VerificationState != before.VerificationState || cancelled.CurrentFingerprint != before.CurrentFingerprint || cancelled.LastResultID != before.LastResultID || cancelled.RunDirectory != before.RunDirectory || !reflect.DeepEqual(cancelled.Baseline, before.Baseline) {
		t.Fatalf("cancel changed evidence: got %#v, want %#v", cancelled, before)
	}
	gated := mustGate(t, service, "discount")
	if gated.WorkflowState != WorkflowCancelled || gated.FollowUp != "" {
		t.Fatalf("gate after cancel = %#v", gated)
	}
}

func TestGateDoesNotFollowUpCancelledTask(t *testing.T) {
	service, root := gateTestService(t)
	seedFailedGateState(t, root, "run-1")
	state := gateState(t, root)
	state.WorkflowState = WorkflowCancelled
	setGateState(t, root, state)

	got := mustGate(t, service, "discount")
	if got.WorkflowState != WorkflowCancelled || got.FollowUp != "" {
		t.Fatalf("gate = %#v", got)
	}
}

func TestResumeResetsOnlyRepairBudget(t *testing.T) {
	service, root := gateTestService(t)
	seedFailedGateState(t, root, "run-4")
	state := gateState(t, root)
	state.Baseline.Modified = []string{"src/original.py"}
	state.CurrentFingerprint = "original-fingerprint"
	state.LastGateRunID = "run-3"
	state.RepairCount = 3
	state.WorkflowState = WorkflowBlocked
	setGateState(t, root, state)

	got, err := service.Resume(context.Background(), "discount")
	if err != nil {
		t.Fatal(err)
	}
	if got.WorkflowState != WorkflowActive {
		t.Fatalf("resume = %#v", got)
	}
	resumed := gateState(t, root)
	if resumed.RepairCount != 0 || resumed.ResumedAt.IsZero() {
		t.Fatalf("resumed state = %#v", resumed)
	}
	state.RepairCount = 0
	if !reflect.DeepEqual(resumed.Baseline, state.Baseline) || resumed.CurrentFingerprint != state.CurrentFingerprint || resumed.LastResultID != state.LastResultID || resumed.LastGateRunID != state.LastGateRunID || resumed.VerificationState != state.VerificationState {
		t.Fatalf("resume changed evidence: got %#v, want %#v", resumed, state)
	}
}

func TestResumeRejectsActiveTask(t *testing.T) {
	service, _ := gateTestService(t)

	if _, err := service.Resume(context.Background(), "discount"); err == nil {
		t.Fatal("Resume accepted active task")
	}
}

func gateTestService(t *testing.T) (*Service, string) {
	t.Helper()

	root, _ := stateTestRepository(t)
	writeStateTestFile(t, filepath.Join(root, ".pbt", "contracts", "discount.json"), `{
		"schemaVersion":1,"task":"discount","applicability":"required","behavior":"discount",
		"properties":[{"id":"discount","statement":"total","generator":"amounts","oracle":"expected","test":"tests/test_discount.py::test_total"}]
	}`)
	if _, err := Begin(root, "discount"); err != nil {
		t.Fatal(err)
	}
	return NewService(root), root
}

func mustGate(t *testing.T, service *Service, task string) Result {
	t.Helper()

	got, err := service.Gate(context.Background(), task)
	if err != nil {
		t.Fatal(err)
	}
	return got
}

func seedFailedGateState(t *testing.T, root, runID string) {
	t.Helper()

	state := gateState(t, root)
	state.LastResultID = runID
	state.RunDirectory = filepath.Join(root, ".pbt", "runs", runID)
	state.VerificationState = VerificationFailed
	state.WorkflowState = WorkflowActive
	setGateState(t, root, state)
}

func gateState(t *testing.T, root string) TaskState {
	t.Helper()

	state, err := LoadState(root, "discount")
	if err != nil {
		t.Fatal(err)
	}
	return state
}

func setGateState(t *testing.T, root string, state TaskState) {
	t.Helper()

	if err := SaveState(root, state); err != nil {
		t.Fatal(err)
	}
}

func gateFingerprint(t *testing.T, root string) string {
	t.Helper()

	config, err := LoadConfig(filepath.Join(root, "pbt.config.json"))
	if err != nil {
		t.Fatal(err)
	}
	fingerprint, err := Fingerprint(config.ProjectRoot, config, filepath.Join(config.ProjectRoot, ".pbt", "contracts", "discount.json"))
	if err != nil {
		t.Fatal(err)
	}
	return fingerprint
}

func TestGateFollowUpNamesReasonRunAndCommand(t *testing.T) {
	service, root := gateTestService(t)
	seedFailedGateState(t, root, "run-1")

	got := mustGate(t, service, "discount")
	for _, value := range []string{"verification_failed", filepath.Join(root, ".pbt", "runs", "run-1"), "pbt verify --task discount"} {
		if !strings.Contains(got.FollowUp, value) {
			t.Fatalf("follow-up = %q, want %q", got.FollowUp, value)
		}
	}
}
