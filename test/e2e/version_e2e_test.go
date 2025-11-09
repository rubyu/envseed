package e2e

import (
	"strings"
	"testing"
)

// [EVT-BCU-7]
func TestVersionCommandsE2E(t *testing.T) {
	t.Helper()

	stdout, stderr, err := runEnvseed(t, "--version")
	if err != nil {
		t.Fatalf("--version failed: %v (stderr=%s)", err, stderr)
	}
	if strings.TrimSpace(stdout) == "" {
		t.Fatalf("empty version output for --version")
	}

	stdout2, stderr2, err := runEnvseed(t, "version")
	if err != nil {
		t.Fatalf("version subcommand failed: %v (stderr=%s)", err, stderr2)
	}
	if strings.TrimSpace(stdout2) == "" {
		t.Fatalf("empty version output for version subcommand")
	}
}
