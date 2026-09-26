package core

import (
	"context"
	"encoding/xml"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const discountNode = "tests/property/test_discount.py::test_discount_total"

func TestVerifyPassesEveryRequiredNode(t *testing.T) {
	service, root := verifyTestService(t, discountNode+"\n", `{"nodeid":"`+discountNode+`","outcome":"passed"}`+"\n", ProcessResult{})

	result, err := service.Verify(context.Background(), "discount")
	if err != nil {
		t.Fatal(err)
	}
	if result.VerificationState != VerificationPassed || result.ReasonCode != "verification_passed" {
		t.Fatalf("result = %#v", result)
	}

	state, err := LoadState(root, "discount")
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"junit.xml", "node-results.jsonl", "stdout.txt", "stderr.txt", "result.json", "pbt_pytest_reporter.py"} {
		if _, err := os.Stat(filepath.Join(state.RunDirectory, name)); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
	}
	junit, err := os.ReadFile(filepath.Join(state.RunDirectory, "junit.xml"))
	if err != nil {
		t.Fatal(err)
	}
	var document struct{ XMLName xml.Name }
	if err := xml.Unmarshal(junit, &document); err != nil {
		t.Fatalf("JUnit XML = %q: %v", junit, err)
	}
}

func TestVerifyFailsWhenRequiredNodeWasCollectedButSkipped(t *testing.T) {
	service, _ := verifyTestService(t, discountNode+"\n", `{"nodeid":"`+discountNode+`","outcome":"skipped"}`+"\n", ProcessResult{})

	result, err := service.Verify(context.Background(), "discount")
	if err != nil {
		t.Fatal(err)
	}
	if result.VerificationState != VerificationFailed || result.ReasonCode != "required_test_skipped" {
		t.Fatalf("result = %#v", result)
	}
}

func TestVerifyFailsWhenRequiredNodeWasNotCollected(t *testing.T) {
	service, _ := verifyTestService(t, "", "", ProcessResult{})

	result, err := service.Verify(context.Background(), "discount")
	if err != nil {
		t.Fatal(err)
	}
	if result.VerificationState != VerificationFailed || result.ReasonCode != "required_test_not_collected" {
		t.Fatalf("result = %#v", result)
	}
}

func TestVerifyFailsForMalformedNodeResults(t *testing.T) {
	service, _ := verifyTestService(t, discountNode+"\n", "not json\n", ProcessResult{})

	result, err := service.Verify(context.Background(), "discount")
	if err != nil {
		t.Fatal(err)
	}
	if result.VerificationState != VerificationFailed || result.ReasonCode != "node_results_invalid" {
		t.Fatalf("result = %#v", result)
	}
}

func TestVerifyFailsForAssertionFailure(t *testing.T) {
	service, _ := verifyTestService(t, discountNode+"\n", `{"nodeid":"`+discountNode+`","outcome":"failed"}`+"\n", ProcessResult{})

	result, err := service.Verify(context.Background(), "discount")
	if err != nil {
		t.Fatal(err)
	}
	if result.VerificationState != VerificationFailed || result.ReasonCode != "required_test_failed" {
		t.Fatalf("result = %#v", result)
	}
}

func TestVerifyFailsForNonzeroPytestExit(t *testing.T) {
	service, _ := verifyTestService(t, discountNode+"\n", `{"nodeid":"`+discountNode+`","outcome":"failed"}`+"\n", ProcessResult{ExitCode: 1})

	result, err := service.Verify(context.Background(), "discount")
	if err != nil {
		t.Fatal(err)
	}
	if result.VerificationState != VerificationFailed || result.ReasonCode != "pytest_failed" {
		t.Fatalf("result = %#v", result)
	}
}

func TestVerifyFailsForPytestTimeout(t *testing.T) {
	service, _ := verifyTestService(t, discountNode+"\n", "", ProcessResult{TimedOut: true})

	result, err := service.Verify(context.Background(), "discount")
	if err != nil {
		t.Fatal(err)
	}
	if result.VerificationState != VerificationFailed || result.ReasonCode != "pytest_timed_out" {
		t.Fatalf("result = %#v", result)
	}
}

func TestVerifyMarksChangedFingerprintStale(t *testing.T) {
	service, root := verifyTestService(t, discountNode+"\n", `{"nodeid":"`+discountNode+`","outcome":"passed"}`+"\n", ProcessResult{})
	originalRun := service.Run
	service.Run = func(ctx context.Context, argv []string, dir string, env []string, limit int) ProcessResult {
		result := originalRun(ctx, argv, dir, env, limit)
		if !hasArgument(argv, "--collect-only") {
			writeStateTestFile(t, filepath.Join(root, "src", "value.py"), "value = 2\n")
		}
		return result
	}

	result, err := service.Verify(context.Background(), "discount")
	if err != nil {
		t.Fatal(err)
	}
	if result.VerificationState != VerificationStale || result.ReasonCode != "stale" {
		t.Fatalf("result = %#v", result)
	}
}

func TestVerifyCollectionFailurePersistsPostCollectionDigest(t *testing.T) {
	service, _ := verifyTestService(t, discountNode+"\n", `{"nodeid":"`+discountNode+`","outcome":"passed"}`+"\n", ProcessResult{})
	config, err := service.loadConfig()
	if err != nil {
		t.Fatal(err)
	}
	contractPath := filepath.Join(config.ProjectRoot, ".pbt", "contracts", "discount.json")
	initial, err := Fingerprint(config.ProjectRoot, config, contractPath)
	if err != nil {
		t.Fatal(err)
	}
	service.Run = func(_ context.Context, argv []string, _ string, _ []string, _ int) ProcessResult {
		if hasArgument(argv, "--collect-only") {
			writeStateTestFile(t, filepath.Join(config.ProjectRoot, "src", "value.py"), "value = 2\n")
			return ProcessResult{ExitCode: 1}
		}
		t.Fatal("execution must not run after collection failure")
		return ProcessResult{}
	}

	result, err := service.Verify(context.Background(), "discount")
	if err != nil {
		t.Fatal(err)
	}
	if result.ReasonCode != "pytest_collection_failed" {
		t.Fatalf("result = %#v", result)
	}
	want, err := Fingerprint(config.ProjectRoot, config, contractPath)
	if err != nil {
		t.Fatal(err)
	}
	state, err := LoadState(config.ProjectRoot, "discount")
	if err != nil {
		t.Fatal(err)
	}
	if state.CurrentFingerprint != want {
		t.Fatalf("current fingerprint = %q, want %q", state.CurrentFingerprint, want)
	}
	if initial == want {
		t.Fatal("test mutation did not change the fingerprint")
	}
}

func TestVerifyFailsWhenJUnitArtifactExceedsLimit(t *testing.T) {
	service, root := verifyTestService(t, discountNode+"\n", `{"nodeid":"`+discountNode+`","outcome":"passed"}`+"\n", ProcessResult{})
	originalRun := service.Run
	service.Run = func(ctx context.Context, argv []string, dir string, env []string, limit int) ProcessResult {
		result := originalRun(ctx, argv, dir, env, limit)
		if hasArgument(argv, "--collect-only") {
			return result
		}
		junitPath := argumentAfter(argv, "--junitxml")
		content := []byte("<testsuites><testsuite/></testsuites>" + strings.Repeat(" ", 64*1024))
		if err := os.WriteFile(junitPath, content, 0o600); err != nil {
			t.Fatal(err)
		}
		return result
	}

	result, err := service.Verify(context.Background(), "discount")
	if err != nil {
		t.Fatal(err)
	}
	if result.VerificationState != VerificationFailed || result.ReasonCode != "artifact_limit_exceeded" {
		t.Fatalf("result = %#v", result)
	}
	state, err := LoadState(root, "discount")
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(filepath.Join(state.RunDirectory, "junit.xml"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Size() != 64*1024 {
		t.Fatalf("JUnit size = %d, want %d", info.Size(), 64*1024)
	}
}

func TestVerifyFailsWhenNodeResultsArtifactExceedsLimit(t *testing.T) {
	service, root := verifyTestService(t, discountNode+"\n", `{"nodeid":"`+discountNode+`","outcome":"passed"}`+"\n", ProcessResult{})
	originalRun := service.Run
	service.Run = func(ctx context.Context, argv []string, dir string, env []string, limit int) ProcessResult {
		result := originalRun(ctx, argv, dir, env, limit)
		if hasArgument(argv, "--collect-only") {
			return result
		}
		nodeResultsPath := environmentValue(env, "PBT_NODE_RESULTS")
		content := `{"nodeid":"` + strings.Repeat("x", 64*1024) + `","outcome":"passed"}` + "\n"
		file, err := os.OpenFile(nodeResultsPath, os.O_APPEND|os.O_WRONLY, 0o600)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := file.WriteString(content); err != nil {
			t.Fatal(err)
		}
		if err := file.Close(); err != nil {
			t.Fatal(err)
		}
		return result
	}

	result, err := service.Verify(context.Background(), "discount")
	if err != nil {
		t.Fatal(err)
	}
	if result.VerificationState != VerificationFailed || result.ReasonCode != "artifact_limit_exceeded" {
		t.Fatalf("result = %#v", result)
	}
	state, err := LoadState(root, "discount")
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(filepath.Join(state.RunDirectory, "node-results.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Size() != 64*1024 {
		t.Fatalf("node results size = %d, want %d", info.Size(), 64*1024)
	}
}

func TestPersistVerificationStoresComparedPostRunDigest(t *testing.T) {
	service, root := verifyTestService(t, discountNode+"\n", `{"nodeid":"`+discountNode+`","outcome":"passed"}`+"\n", ProcessResult{})
	config, err := service.loadConfig()
	if err != nil {
		t.Fatal(err)
	}
	state, err := LoadState(root, "discount")
	if err != nil {
		t.Fatal(err)
	}
	contractPath := filepath.Join(root, ".pbt", "contracts", "discount.json")
	postRunDigest, err := Fingerprint(root, config, contractPath)
	if err != nil {
		t.Fatal(err)
	}
	writeStateTestFile(t, filepath.Join(root, "src", "value.py"), "value = 2\n")

	if _, err := service.persistVerification(config, state, "discount", t.TempDir(), "verification_passed", postRunDigest); err != nil {
		t.Fatal(err)
	}
	persisted, err := LoadState(root, "discount")
	if err != nil {
		t.Fatal(err)
	}
	if persisted.CurrentFingerprint != postRunDigest {
		t.Fatalf("current fingerprint = %q, want compared post-run digest %q", persisted.CurrentFingerprint, postRunDigest)
	}
}

func TestVerifyPreservesPythonPathAndUsesBundledReporter(t *testing.T) {
	t.Setenv("PYTHONPATH", "existing")
	service, _ := verifyTestService(t, discountNode+"\n", `{"nodeid":"`+discountNode+`","outcome":"passed"}`+"\n", ProcessResult{})
	originalRun := service.Run
	service.Run = func(ctx context.Context, argv []string, dir string, env []string, limit int) ProcessResult {
		if !hasArgument(argv, "--collect-only") {
			if argumentAfter(argv, "-p") != "pbt_pytest_reporter" {
				t.Fatalf("argv = %#v", argv)
			}
			if !strings.HasSuffix(environmentValue(env, "PYTHONPATH"), string(os.PathListSeparator)+"existing") {
				t.Fatalf("PYTHONPATH = %q", environmentValue(env, "PYTHONPATH"))
			}
		}
		return originalRun(ctx, argv, dir, env, limit)
	}

	if _, err := service.Verify(context.Background(), "discount"); err != nil {
		t.Fatal(err)
	}
}

func verifyTestService(t *testing.T, collected, nodeResults string, execution ProcessResult) (*Service, string) {
	t.Helper()

	root, _ := stateTestRepository(t)
	contractPath := filepath.Join(root, ".pbt", "contracts", "discount.json")
	writeStateTestFile(t, contractPath, `{
		"schemaVersion":1,"task":"discount","applicability":"required","behavior":"discount",
		"properties":[{"id":"discount","statement":"total is discounted","generator":"amounts","oracle":"expected total","test":"`+discountNode+`"}]
	}`)
	if _, err := Begin(root, "discount"); err != nil {
		t.Fatal(err)
	}

	service := NewService(root)
	service.Run = func(_ context.Context, argv []string, _ string, env []string, _ int) ProcessResult {
		if hasArgument(argv, "--collect-only") {
			return ProcessResult{Stdout: collected}
		}
		for _, value := range env {
			const prefix = "PBT_NODE_RESULTS="
			if path, ok := cutPrefix(value, prefix); ok {
				if err := os.WriteFile(path, []byte(nodeResults), 0o600); err != nil {
					t.Fatal(err)
				}
			}
		}
		junitPath := argumentAfter(argv, "--junitxml")
		if err := os.WriteFile(junitPath, []byte("<testsuites><testsuite name=\"different-names\"/></testsuites>"), 0o600); err != nil {
			t.Fatal(err)
		}
		return execution
	}
	return service, root
}

func hasArgument(argv []string, target string) bool {
	for _, argument := range argv {
		if argument == target {
			return true
		}
	}
	return false
}

func argumentAfter(argv []string, target string) string {
	for index, argument := range argv {
		if argument == target && index+1 < len(argv) {
			return argv[index+1]
		}
	}
	return ""
}

func environmentValue(env []string, key string) string {
	for _, value := range env {
		if result, ok := cutPrefix(value, key+"="); ok {
			return result
		}
	}
	return ""
}

func cutPrefix(value, prefix string) (string, bool) {
	if len(value) < len(prefix) || value[:len(prefix)] != prefix {
		return "", false
	}
	return value[len(prefix):], true
}
