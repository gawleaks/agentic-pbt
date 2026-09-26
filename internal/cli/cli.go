package cli

import (
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

	fmt.Fprintln(stderr, "unknown command")
	return 3
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
		if err := json.NewEncoder(stdout).Encode(result); err != nil {
			fmt.Fprintln(stderr, err)
			return 3
		}
		return 0
	}
	fmt.Fprintf(stdout, "Started task %q; verification is pending.\n", result.Task)
	return 0
}
