package cppanalysis

import (
	"errors"
	"os/exec"
)

// CommandRunner abstracts process lookup and execution so tests can mock
// clang-format without requiring the binary on PATH.
type CommandRunner interface {
	LookPath(file string) (string, error)
	// Run executes name with args and returns combined stdout/stderr, the
	// process exit code (0 on success), and an error only when the process
	// could not be started. A non-zero exit code is not an error.
	Run(name string, args []string) (output []byte, exitCode int, err error)
}

// ExecRunner is the production CommandRunner backed by os/exec.
type ExecRunner struct{}

func (ExecRunner) LookPath(file string) (string, error) {
	return exec.LookPath(file)
}

func (ExecRunner) Run(name string, args []string) ([]byte, int, error) {
	cmd := exec.Command(name, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return output, exitErr.ExitCode(), nil
		}
		return output, -1, err
	}
	return output, 0, nil
}
