package envseed

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

// [EVT-BIF-2]
func TestPassCommandShowMissingBinary(t *testing.T) {
	emptyBin := filepath.Join(t.TempDir(), "bin")
	if err := os.Mkdir(emptyBin, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	t.Setenv("PATH", emptyBin)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := (&PassCommand{}).Show(ctx, "any/path")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var exitErr *ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("expected ExitError, got %v", err)
	}
	if exitErr.DetailCode != "EVE-104-1" {
		t.Fatalf("detail code = %s, want EVE-104-1", exitErr.DetailCode)
	}
	if !errors.Is(err, exec.ErrNotFound) {
		t.Fatalf("error does not wrap exec.ErrNotFound: %v", err)
	}
}

// [EVT-BIF-2]
func TestPassCommandShowFailureExit(t *testing.T) {
	dir := t.TempDir()
	passPath := filepath.Join(dir, "pass")
	script := "#!/bin/sh\n\necho \"fatal: bad things\" >&2\nexit 9\n"
	if err := os.WriteFile(passPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write script: %v", err)
	}
	t.Setenv("PATH", dir)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := (&PassCommand{}).Show(ctx, "any/path")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var exitErr *ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("expected ExitError, got %v", err)
	}
	if exitErr.DetailCode != "EVE-104-101" {
		t.Fatalf("detail code = %s, want EVE-104-101", exitErr.DetailCode)
	}
}
