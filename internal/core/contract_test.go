package core

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadRequiredContractRequiresEveryPropertyField(t *testing.T) {
	_, err := LoadContract(writeContract(t, `{
      "schemaVersion":1,"task":"discount","applicability":"required",
      "behavior":"discount","properties":[{"id":"total","test":"tests/test_x.py::test_x"}]
    }`))
	if err == nil || !strings.Contains(err.Error(), "generator") {
		t.Fatalf("LoadContract error = %v", err)
	}
}

func TestLoadContractValidation(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantErr string
	}{
		{
			name: "unknown schema version",
			content: `{
				"schemaVersion":2,"task":"discount","applicability":"not_applicable","reason":"not relevant"
			}`,
			wantErr: "unsupported schemaVersion",
		},
		{
			name: "not applicable requires reason",
			content: `{
				"schemaVersion":1,"task":"discount","applicability":"not_applicable","reason":""
			}`,
			wantErr: "reason",
		},
		{
			name: "not applicable forbids properties",
			content: `{
				"schemaVersion":1,"task":"discount","applicability":"not_applicable","reason":"not relevant",
				"properties":[{"id":"total","statement":"total","generator":"gen","oracle":"oracle","test":"tests/test_x.py::test_x"}]
			}`,
			wantErr: "properties",
		},
		{
			name: "unknown field",
			content: `{
				"schemaVersion":1,"task":"discount","applicability":"not_applicable","reason":"not relevant","extra":true
			}`,
			wantErr: "unknown field",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := LoadContract(writeContract(t, tt.content))
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("LoadContract error = %v, want containing %q", err, tt.wantErr)
			}
		})
	}
}

func TestLoadContractAcceptsBothForms(t *testing.T) {
	tests := []struct {
		name    string
		content string
	}{
		{
			name: "required",
			content: `{
				"schemaVersion":1,"task":"discount","applicability":"required","behavior":"discount",
				"properties":[{"id":"total","statement":"total is correct","generator":"generated cart","oracle":"expected total","test":"tests/test_x.py::test_x"}]
			}`,
		},
		{
			name: "not applicable",
			content: `{
				"schemaVersion":1,"task":"discount","applicability":"not_applicable","reason":"no discount behavior"
			}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := LoadContract(writeContract(t, tt.content)); err != nil {
				t.Fatalf("LoadContract error = %v", err)
			}
		})
	}
}

func writeContract(t *testing.T, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "contract.json")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}
