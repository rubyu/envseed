package e2e

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"
)

var (
	buildOnce sync.Once
	buildErr  error
	binPath   string
)

// buildEnvseed builds the CLI once per package and returns the binary path.
func buildEnvseed(t *testing.T) string {
	t.Helper()
	buildOnce.Do(func() {
		dir, err := os.Getwd()
		if err != nil {
			buildErr = err
			return
		}
		repoRoot := filepath.Clean(filepath.Join(dir, "..", ".."))
		tempDir, err := os.MkdirTemp("", "envseed-bin-*")
		if err != nil {
			buildErr = err
			return
		}
		binPath = filepath.Join(tempDir, "envseed")
		cmd := exec.Command("go", "build", "-o", binPath, "./cmd/envseed")
		cmd.Dir = repoRoot
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		buildErr = cmd.Run()
	})
	if buildErr != nil {
		t.Fatalf("build envseed: %v", buildErr)
	}
	return binPath
}

// runEnvseed runs the CLI with args and returns stdout, stderr, and error.
func runEnvseed(t *testing.T, args ...string) (string, string, error) {
	t.Helper()
	bin := buildEnvseed(t)
	cmd := exec.Command(bin, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}

// writeTemplate writes a template file with the given content and returns its absolute path.
func writeTemplate(t *testing.T, dir, name, content string) string {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir dir: %v", err)
	}
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write template: %v", err)
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		t.Fatalf("abs path: %v", err)
	}
	return abs
}
