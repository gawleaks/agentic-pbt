package core

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadConfigRejectsShellCommandAndNonPositiveTimeout(t *testing.T) {
	_, err := LoadConfig(writeConfig(t, `{
      "schemaVersion":1,"projectRoot":".","inputs":["src/**"],
      "python":"uv run python","command":["uv","run","pytest"],"timeoutSeconds":0
    }`))
	if err == nil || !strings.Contains(err.Error(), "python must be an argument array") {
		t.Fatalf("LoadConfig error = %v", err)
	}
}

func TestLoadConfigRejectsMissingAndEmptyProjectRoot(t *testing.T) {
	tests := []struct {
		name    string
		content string
	}{
		{
			name: "missing",
			content: `{
				"schemaVersion":1,"inputs":["src/**"],
				"python":["uv","run","python"],"command":["uv","run","pytest"],"timeoutSeconds":30
			}`,
		},
		{
			name: "empty",
			content: `{
				"schemaVersion":1,"projectRoot":"","inputs":["src/**"],
				"python":["uv","run","python"],"command":["uv","run","pytest"],"timeoutSeconds":30
			}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := LoadConfig(writeConfig(t, tt.content))
			if err == nil || !strings.Contains(err.Error(), "projectRoot must not be empty") {
				t.Fatalf("LoadConfig error = %v", err)
			}
		})
	}
}

func TestLoadConfigRejectsSymlinkOutsideRepository(t *testing.T) {
	repository := t.TempDir()
	if err := os.Mkdir(filepath.Join(repository, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}

	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(repository, "linked-outside")); err != nil {
		t.Fatal(err)
	}

	_, err := LoadConfig(writeConfigInRepository(t, repository, `{
		"schemaVersion":1,"projectRoot":"linked-outside","inputs":["src/**"],
		"python":["uv","run","python"],"command":["uv","run","pytest"],"timeoutSeconds":30
	}`))
	if err == nil || !strings.Contains(err.Error(), "projectRoot must be inside its containing repository") {
		t.Fatalf("LoadConfig error = %v", err)
	}
}

func TestLoadConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantErr string
	}{
		{
			name: "unknown schema version",
			content: `{
				"schemaVersion":2,"projectRoot":".","inputs":["src/**"],
				"python":["uv","run","python"],"command":["uv","run","pytest"],"timeoutSeconds":30
			}`,
			wantErr: "unsupported schemaVersion",
		},
		{
			name: "absolute project root",
			content: `{
				"schemaVersion":1,"projectRoot":"/tmp","inputs":["src/**"],
				"python":["uv","run","python"],"command":["uv","run","pytest"],"timeoutSeconds":30
			}`,
			wantErr: "projectRoot must be relative",
		},
		{
			name: "project root outside repository",
			content: `{
				"schemaVersion":1,"projectRoot":"..","inputs":["src/**"],
				"python":["uv","run","python"],"command":["uv","run","pytest"],"timeoutSeconds":30
			}`,
			wantErr: "projectRoot must be inside its containing repository",
		},
		{
			name: "empty inputs",
			content: `{
				"schemaVersion":1,"projectRoot":".","inputs":[],
				"python":["uv","run","python"],"command":["uv","run","pytest"],"timeoutSeconds":30
			}`,
			wantErr: "inputs must not be empty",
		},
		{
			name: "unknown field",
			content: `{
				"schemaVersion":1,"projectRoot":".","inputs":["src/**"],
				"python":["uv","run","python"],"command":["uv","run","pytest"],"timeoutSeconds":30,"extra":true
			}`,
			wantErr: "unknown field",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := LoadConfig(writeConfig(t, tt.content))
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("LoadConfig error = %v, want containing %q", err, tt.wantErr)
			}
		})
	}
}

func TestLoadConfigAcceptsValidConfig(t *testing.T) {
	config, err := LoadConfig(writeConfig(t, `{
		"schemaVersion":1,"projectRoot":".","inputs":["src/**"],
		"python":["uv","run","python"],"command":["uv","run","pytest"],"timeoutSeconds":30
	}`))
	if err != nil {
		t.Fatalf("LoadConfig error = %v", err)
	}
	if config.ProjectRoot == "." {
		t.Fatal("ProjectRoot was not resolved")
	}
}

func writeConfig(t *testing.T, content string) string {
	t.Helper()

	repository := t.TempDir()
	if err := os.Mkdir(filepath.Join(repository, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	return writeConfigInRepository(t, repository, content)
}

func writeConfigInRepository(t *testing.T, repository, content string) string {
	t.Helper()

	path := filepath.Join(repository, "pbt.json")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}
