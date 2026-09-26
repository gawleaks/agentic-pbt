package core

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseNodeResultsRejectsDuplicateNodeIDs(t *testing.T) {
	path := filepath.Join(t.TempDir(), "node-results.jsonl")
	content := "{\"nodeid\":\"tests/test_x.py::test_x\",\"outcome\":\"passed\"}\n" +
		"{\"nodeid\":\"tests/test_x.py::test_x\",\"outcome\":\"failed\"}\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := parseNodeResults(path); err == nil {
		t.Fatal("parseNodeResults accepted duplicate node IDs")
	}
}

func TestParseNodeResultsRejectsMalformedJSONL(t *testing.T) {
	path := filepath.Join(t.TempDir(), "node-results.jsonl")
	if err := os.WriteFile(path, []byte("not json\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := parseNodeResults(path); err == nil {
		t.Fatal("parseNodeResults accepted malformed JSONL")
	}
}
