package core

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFingerprintChangesForUntrackedConfiguredInput(t *testing.T) {
	root, configPath := stateTestRepository(t)
	before := mustFingerprint(t, root, configPath)

	writeStateTestFile(t, filepath.Join(root, "src", "new.py"), "x = 1\n")
	after := mustFingerprint(t, root, configPath)

	if before == after {
		t.Fatal("fingerprint ignored untracked configured input")
	}
}

func TestFingerprintChangesForDeletedConfiguredInput(t *testing.T) {
	root, configPath := stateTestRepository(t)
	before := mustFingerprint(t, root, configPath)

	if err := os.Remove(filepath.Join(root, "src", "value.py")); err != nil {
		t.Fatal(err)
	}
	after := mustFingerprint(t, root, configPath)

	if before == after {
		t.Fatal("fingerprint ignored deleted configured input")
	}
}

func TestFingerprintIncludesConfigAndContract(t *testing.T) {
	root, configPath := stateTestRepository(t)
	contractPath := filepath.Join(root, ".pbt", "contracts", "discount.json")
	writeStateTestFile(t, contractPath, `{"schemaVersion":1,"task":"discount"}`)
	before := mustFingerprint(t, root, configPath)

	writeStateTestFile(t, configPath, `{
		"schemaVersion": 1,
		"projectRoot": ".",
		"inputs": ["src/**"],
		"python": ["python"],
		"command": ["pytest"],
		"timeoutSeconds": 60
	}`)
	afterConfig := mustFingerprint(t, root, configPath)
	if before == afterConfig {
		t.Fatal("fingerprint ignored configuration change")
	}

	writeStateTestFile(t, contractPath, `{"schemaVersion":1,"task":"discount","changed":true}`)
	afterContract := mustFingerprint(t, root, configPath)
	if afterConfig == afterContract {
		t.Fatal("fingerprint ignored contract change")
	}
}

func TestFingerprintExcludesLocalStateAndRuns(t *testing.T) {
	root, configPath := stateTestRepository(t)
	before := mustFingerprint(t, root, configPath)

	writeStateTestFile(t, filepath.Join(root, ".pbt", "state", "discount.json"), `{"task":"discount"}`)
	writeStateTestFile(t, filepath.Join(root, ".pbt", "runs", "run-1", "result.json"), `{"status":"passed"}`)
	after := mustFingerprint(t, root, configPath)

	if before != after {
		t.Fatal("fingerprint included local state or run artifacts")
	}
}

func mustFingerprint(t *testing.T, root, configPath string) string {
	t.Helper()

	config, err := LoadConfig(configPath)
	if err != nil {
		t.Fatal(err)
	}
	fingerprint, err := Fingerprint(root, config, filepath.Join(root, ".pbt", "contracts", "discount.json"))
	if err != nil {
		t.Fatal(err)
	}
	return fingerprint
}
