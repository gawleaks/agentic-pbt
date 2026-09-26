package core

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const schemaVersion = 1

type contractDocument struct {
	SchemaVersion int             `json:"schemaVersion"`
	Task          string          `json:"task"`
	Applicability string          `json:"applicability"`
	Behavior      *string         `json:"behavior"`
	Reason        *string         `json:"reason"`
	Properties    json.RawMessage `json:"properties"`
}

// LoadConfig reads and validates a PBT configuration file.
func LoadConfig(path string) (*Config, error) {
	var config Config
	if err := decodeJSON(path, &config); err != nil {
		var typeErr *json.UnmarshalTypeError
		if errors.As(err, &typeErr) && typeErr.Field == "python" {
			return nil, fmt.Errorf("python must be an argument array")
		}
		return nil, err
	}

	if config.SchemaVersion != schemaVersion {
		return nil, fmt.Errorf("unsupported schemaVersion %d", config.SchemaVersion)
	}
	if filepath.IsAbs(config.ProjectRoot) {
		return nil, fmt.Errorf("projectRoot must be relative")
	}
	if len(config.Inputs) == 0 {
		return nil, fmt.Errorf("inputs must not be empty")
	}
	if err := validateArguments("python", config.Python); err != nil {
		return nil, err
	}
	if err := validateArguments("command", config.Command); err != nil {
		return nil, err
	}
	if config.TimeoutSeconds <= 0 {
		return nil, fmt.Errorf("timeoutSeconds must be positive")
	}

	root, err := resolveProjectRoot(path, config.ProjectRoot)
	if err != nil {
		return nil, err
	}
	config.ProjectRoot = root
	return &config, nil
}

// LoadContract reads and validates a PBT contract file.
func LoadContract(path string) (*Contract, error) {
	var document contractDocument
	if err := decodeJSON(path, &document); err != nil {
		return nil, err
	}

	contract := Contract{
		SchemaVersion: document.SchemaVersion,
		Task:          document.Task,
		Applicability: document.Applicability,
		Properties:    []RequiredProperty{},
	}
	if document.Behavior != nil {
		contract.Behavior = *document.Behavior
	}
	if document.Reason != nil {
		contract.Reason = *document.Reason
	}
	if document.Properties != nil {
		if err := decodeJSONBytes(document.Properties, &contract.Properties); err != nil {
			return nil, err
		}
	}

	if contract.SchemaVersion != schemaVersion {
		return nil, fmt.Errorf("unsupported schemaVersion %d", contract.SchemaVersion)
	}
	if strings.TrimSpace(contract.Task) == "" {
		return nil, fmt.Errorf("task is required")
	}

	switch contract.Applicability {
	case "required":
		if document.Behavior == nil || strings.TrimSpace(contract.Behavior) == "" {
			return nil, fmt.Errorf("behavior is required")
		}
		if document.Reason != nil {
			return nil, fmt.Errorf("reason is only valid for not_applicable contracts")
		}
		if document.Properties == nil || len(contract.Properties) == 0 {
			return nil, fmt.Errorf("properties must not be empty")
		}
		for _, property := range contract.Properties {
			if err := validateProperty(property); err != nil {
				return nil, err
			}
		}
	case "not_applicable":
		if document.Reason == nil || strings.TrimSpace(contract.Reason) == "" {
			return nil, fmt.Errorf("reason is required")
		}
		if document.Behavior != nil {
			return nil, fmt.Errorf("behavior is only valid for required contracts")
		}
		if document.Properties != nil {
			return nil, fmt.Errorf("properties are not allowed for not_applicable contracts")
		}
	default:
		return nil, fmt.Errorf("applicability must be required or not_applicable")
	}

	return &contract, nil
}

func decodeJSON[T any](path string, value *T) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	return decodeJSONReader(file, value)
}

func decodeJSONBytes[T any](data []byte, value *T) error {
	return decodeJSONReader(bytes.NewReader(data), value)
}

func decodeJSONReader[T any](reader io.Reader, value *T) error {
	decoder := json.NewDecoder(reader)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(value); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			return fmt.Errorf("invalid JSON: multiple values")
		}
		return err
	}
	return nil
}

func validateArguments(name string, arguments []string) error {
	if len(arguments) == 0 {
		return fmt.Errorf("%s must be a non-empty argument array", name)
	}
	for _, argument := range arguments {
		if strings.TrimSpace(argument) == "" {
			return fmt.Errorf("%s must be a non-empty argument array", name)
		}
	}
	return nil
}

func validateProperty(property RequiredProperty) error {
	fields := []struct {
		name  string
		value string
	}{
		{name: "id", value: property.ID},
		{name: "statement", value: property.Statement},
		{name: "generator", value: property.Generator},
		{name: "oracle", value: property.Oracle},
		{name: "test", value: property.Test},
	}
	for _, field := range fields {
		if strings.TrimSpace(field.value) == "" {
			return fmt.Errorf("property %s is required", field.name)
		}
	}
	return nil
}

func resolveProjectRoot(configPath, projectRoot string) (string, error) {
	configDirectory, err := filepath.Abs(filepath.Dir(configPath))
	if err != nil {
		return "", err
	}
	repositoryRoot, err := findRepositoryRoot(configDirectory)
	if err != nil {
		return "", err
	}
	resolvedRoot := filepath.Clean(filepath.Join(configDirectory, projectRoot))
	relativeRoot, err := filepath.Rel(repositoryRoot, resolvedRoot)
	if err != nil {
		return "", err
	}
	if relativeRoot == ".." || strings.HasPrefix(relativeRoot, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("projectRoot must be inside its containing repository")
	}
	return resolvedRoot, nil
}

func findRepositoryRoot(directory string) (string, error) {
	for {
		if _, err := os.Stat(filepath.Join(directory, ".git")); err == nil {
			return directory, nil
		}
		parent := filepath.Dir(directory)
		if parent == directory {
			return "", fmt.Errorf("config file is not inside a repository")
		}
		directory = parent
	}
}

