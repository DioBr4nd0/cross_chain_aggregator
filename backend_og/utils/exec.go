package utils

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// RunWASMDCommand executes a wasmd CLI command and returns its output or error.
func RunWASMDCommand(args ...string) (string, error) {
	cmd := exec.Command("wasmd", args...) // Assumes wasmd is in PATH

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	fmt.Printf("Executing: wasmd %s\n", strings.Join(args, " ")) // Log the command

	err := cmd.Run()
	if err != nil {
		return stdout.String(), fmt.Errorf("wasmd command failed: %w. Stderr: %s. Stdout: %s", err, stderr.String(), stdout.String())
	}

	return stdout.String(), nil
}
