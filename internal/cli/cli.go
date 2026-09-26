package cli

import (
	"fmt"
	"io"
)

// Run executes the PBT command and returns its exit code.
func Run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 1 && args[0] == "help" {
		fmt.Fprintln(stdout, "Usage: pbt help")
		return 0
	}

	fmt.Fprintln(stderr, "unknown command")
	return 3
}
