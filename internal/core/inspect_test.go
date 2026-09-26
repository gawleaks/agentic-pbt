package core

import (
	"context"
	"path/filepath"
	"slices"
	"testing"
)

const versionQuery = "import json, hypothesis, pytest\nprint(json.dumps({\"hypothesis\": hypothesis.__version__, \"pytest\": pytest.__version__}))"

func TestInspectUsesConfiguredPythonArray(t *testing.T) {
	root, configPath := stateTestRepository(t)
	writeStateTestFile(t, configPath, `{
		"schemaVersion":1,"projectRoot":".","inputs":["src/**"],
		"python":["uv","run","python"],"command":["pytest"],"timeoutSeconds":30
	}`)
	service := NewService(root)
	service.Run = func(_ context.Context, argv []string, _ string, _ []string, _ int) ProcessResult {
		want := []string{"uv", "run", "python", "-c", versionQuery}
		if !slices.Equal(want, argv) {
			t.Fatalf("argv = %#v, want %#v", argv, want)
		}
		return ProcessResult{Stdout: `{"pytest":"8.3.0","hypothesis":"6.100.0"}`}
	}

	got, err := service.Inspect(context.Background(), "discount")
	if err != nil || got.PytestVersion != "8.3.0" || got.HypothesisVersion != "6.100.0" {
		t.Fatalf("inspect = %#v, %v", got, err)
	}
	expectedRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}
	if got.ContractPath != filepath.Join(expectedRoot, ".pbt", "contracts", "discount.json") {
		t.Fatalf("contract path = %q", got.ContractPath)
	}
}

func TestInspectReportsInvalidVersionOutput(t *testing.T) {
	root, _ := stateTestRepository(t)
	service := NewService(root)
	service.Run = func(context.Context, []string, string, []string, int) ProcessResult {
		return ProcessResult{Stdout: "not JSON"}
	}

	got, err := service.Inspect(context.Background(), "discount")
	if err == nil || got.ReasonCode != "python_version_query_failed" {
		t.Fatalf("inspect = %#v, error = %v", got, err)
	}
}

func TestStatusMarksChangedPassedStateStale(t *testing.T) {
	root, _ := stateTestRepository(t)
	if _, err := Begin(root, "discount"); err != nil {
		t.Fatal(err)
	}
	state, err := LoadState(root, "discount")
	if err != nil {
		t.Fatal(err)
	}
	state.VerificationState = VerificationPassed
	if err := SaveState(root, state); err != nil {
		t.Fatal(err)
	}
	writeStateTestFile(t, filepath.Join(root, "src", "value.py"), "value = 2\n")

	got, err := NewService(root).Status(context.Background(), "")
	if err != nil {
		t.Fatal(err)
	}
	if got.Task != "discount" || got.VerificationState != VerificationStale || got.ReasonCode != "verification_stale" {
		t.Fatalf("status = %#v", got)
	}
}
