package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/gawleaks/agentic-pbt/internal/core"
)

// Run executes the PBT command and returns its exit code.
func Run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 1 && args[0] == "help" {
		fmt.Fprintln(stdout, "Usage: pbt help")
		return 0
	}
	if len(args) > 0 && args[0] == "begin" {
		return runBegin(args[1:], stdout, stderr)
	}
	if len(args) > 0 && args[0] == "inspect" {
		return runInspect(args[1:], stdout, stderr)
	}
	if len(args) > 0 && args[0] == "status" {
		return runStatus(args[1:], stdout, stderr)
	}
	if len(args) > 0 && args[0] == "verify" {
		return runVerify(args[1:], stdout, stderr)
	}

	fmt.Fprintln(stderr, "unknown command")
	return 3
}

func runInspect(args []string, stdout, stderr io.Writer) int {
	task, jsonOutput, ok := parseTaskArgs(args, "usage: pbt inspect --task <id> [--json]", true, stderr)
	if !ok {
		return 3
	}
	root, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 3
	}
	result, err := core.NewService(root).Inspect(context.Background(), task)
	if err != nil {
		if jsonOutput && result.SchemaVersion != 0 {
			writeJSON(stdout, stderr, result)
		}
		fmt.Fprintln(stderr, err)
		return 3
	}
	if jsonOutput {
		return writeJSON(stdout, stderr, result)
	}
	fmt.Fprintf(stdout, "Hypothesis %s; pytest %s.\n", result.HypothesisVersion, result.PytestVersion)
	return 0
}

func runStatus(args []string, stdout, stderr io.Writer) int {
	task, jsonOutput, ok := parseTaskArgs(args, "usage: pbt status [--task <id>] [--json]", false, stderr)
	if !ok {
		return 3
	}
	root, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 3
	}
	result, err := core.NewService(root).Status(context.Background(), task)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 3
	}
	if jsonOutput {
		return writeJSON(stdout, stderr, result)
	}
	fmt.Fprintf(stdout, "Task %q is %s.\n", result.Task, result.VerificationState)
	return 0
}

func runVerify(args []string, stdout, stderr io.Writer) int {
	task, jsonOutput, ok := parseTaskArgs(args, "usage: pbt verify --task <id> [--json]", true, stderr)
	if !ok {
		return 3
	}
	root, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 3
	}
	result, err := core.NewService(root).Verify(context.Background(), task)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 3
	}
	if jsonOutput {
		if code := writeJSON(stdout, stderr, result); code != 0 {
			return code
		}
	} else {
		fmt.Fprintf(stdout, "Task %q is %s.\n", result.Task, result.VerificationState)
	}
	if result.VerificationState == core.VerificationPassed || result.VerificationState == core.VerificationNotApplicable {
		return 0
	}
	return 1
}

func parseTaskArgs(args []string, usage string, required bool, stderr io.Writer) (string, bool, bool) {
	var task string
	var jsonOutput bool
	for index := 0; index < len(args); index++ {
		switch args[index] {
		case "--task":
			index++
			if index == len(args) || task != "" {
				fmt.Fprintln(stderr, usage)
				return "", false, false
			}
			task = args[index]
		case "--json":
			jsonOutput = true
		default:
			fmt.Fprintln(stderr, usage)
			return "", false, false
		}
	}
	if required && task == "" {
		fmt.Fprintln(stderr, usage)
		return "", false, false
	}
	return task, jsonOutput, true
}

func writeJSON(stdout, stderr io.Writer, value any) int {
	if err := json.NewEncoder(stdout).Encode(value); err != nil {
		fmt.Fprintln(stderr, err)
		return 3
	}
	return 0
}

func runBegin(args []string, stdout, stderr io.Writer) int {
	var task string
	var jsonOutput bool
	for index := 0; index < len(args); index++ {
		switch args[index] {
		case "--task":
			index++
			if index == len(args) {
				fmt.Fprintln(stderr, "usage: pbt begin --task <id> [--json]")
				return 3
			}
			task = args[index]
		case "--json":
			jsonOutput = true
		default:
			fmt.Fprintln(stderr, "usage: pbt begin --task <id> [--json]")
			return 3
		}
	}
	if task == "" {
		fmt.Fprintln(stderr, "usage: pbt begin --task <id> [--json]")
		return 3
	}

	root, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 3
	}
	result, err := core.Begin(root, task)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 3
	}
	if jsonOutput {
		return writeJSON(stdout, stderr, result)
	}
	fmt.Fprintf(stdout, "Started task %q; verification is pending.\n", result.Task)
	return 0
}
