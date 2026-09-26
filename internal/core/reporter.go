package core

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

type nodeResult struct {
	NodeID  string `json:"nodeid"`
	Outcome string `json:"outcome"`
}

func parseNodeResults(path string) (map[string]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	results := map[string]string{}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		var result nodeResult
		if err := json.Unmarshal(scanner.Bytes(), &result); err != nil {
			return nil, err
		}
		if strings.TrimSpace(result.NodeID) == "" || strings.TrimSpace(result.Outcome) == "" {
			return nil, fmt.Errorf("node result must include nodeid and outcome")
		}
		if _, exists := results[result.NodeID]; exists {
			return nil, fmt.Errorf("duplicate node result for %q", result.NodeID)
		}
		results[result.NodeID] = result.Outcome
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return results, nil
}
