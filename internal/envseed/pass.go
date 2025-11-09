package envseed

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"strings"
)

// PassCommand implements PassClient using the pass CLI.
type PassCommand struct{}

// Show retrieves PATH through pass.
func (p *PassCommand) Show(ctx context.Context, path string) (string, error) {
	cmd := exec.CommandContext(ctx, "pass", "show", path)
	// Connect stdin so that interactive pinentry can receive user input.
	cmd.Stdin = os.Stdin
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return "", NewExitError("EVE-104-1").WithErr(err)
		}
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			errMsg := stderr.String()
			if passEntryNotFound(errMsg) {
				return "", NewExitError("EVE-104-201", path).WithErr(err)
			}
			return "", NewExitError("EVE-104-101", path).WithErr(err)
		}
		return "", NewExitError("EVE-104-101", path).WithErr(err)
	}
	return string(out), nil
}

// passEntryNotFound tries to recognize common 'not found' diagnostics from pass.
func passEntryNotFound(msg string) bool {
	m := strings.ToLower(msg)
	patterns := []string{
		"is not in the password store",
		"not in the password store",
		"no such password",
		"not found",
	}
	for _, p := range patterns {
		if strings.Contains(m, p) {
			return true
		}
	}
	return false
}
