package envseed

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// [EVT-BIU-3][EVT-BIU-5][EVT-BIP-1]
func TestResolveOutputPathDerivation(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "config.envseed.local")
	got, err := resolveOutputPath(input, "")
	if err != nil {
		t.Fatalf("resolveOutputPath error: %v", err)
	}
	want := filepath.Join(dir, "config.env.local")
	if got != want {
		t.Fatalf("output path = %q, want %q", got, want)
	}
	inputMulti := filepath.Join(dir, "multi.envseed.envseed")
	got, err = resolveOutputPath(inputMulti, "")
	if err != nil {
		t.Fatalf("resolveOutputPath error: %v", err)
	}
	want = filepath.Join(dir, "multi.env.envseed")
	if got != want {
		t.Fatalf("first occurrence replacement failed: %q", got)
	}
}

// [EVT-BIU-4]
func TestResolveOutputPathRequiresEnvseed(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "config.env")
	_, err := resolveOutputPath(input, "")
	if err == nil {
		t.Fatal("expected error for input lacking 'envseed'")
	}
	var exitErr *ExitError
	if !errors.As(err, &exitErr) || exitErr.DetailCode != "EVE-101-203" {
		t.Fatalf("unexpected error: %v", err)
	}
}

// [EVT-BIU-3]
func TestResolveOutputPathDirectoryTrailingSeparator(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "app.envseed")
	subdir := filepath.Join(dir, "outputs")
	if err := os.Mkdir(subdir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	explicit := subdir + string(os.PathSeparator)
	got, err := resolveOutputPath(input, explicit)
	if err != nil {
		t.Fatalf("resolveOutputPath error: %v", err)
	}
	want := filepath.Join(subdir, "app.env")
	if got != want {
		t.Fatalf("output path = %q, want %q", got, want)
	}
}

// [EVT-BIF-2]
func TestValidateOutputPathRejectsDirectory(t *testing.T) {
	dir := t.TempDir()
	targetDir := filepath.Join(dir, "existing")
	if err := os.Mkdir(targetDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := validateOutputPath(targetDir); err == nil {
		t.Fatal("expected error for directory path")
	} else {
		var exitErr *ExitError
		if !errors.As(err, &exitErr) || exitErr.DetailCode != "EVE-101-301" {
			t.Fatalf("unexpected error: %v", err)
		}
	}
}
