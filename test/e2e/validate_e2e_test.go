package e2e

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// [EVT-BCU-5]
func TestValidateE2E_SimpleTemplate(t *testing.T) {
	t.Helper()
	tmp := t.TempDir()
	input := writeTemplate(t, tmp, "validate.envseed", "APP_NAME=staging\n")
	if _, stderr, err := runEnvseed(t, "validate", input); err != nil {
		t.Fatalf("validate failed: %v (stderr=%s)", err, stderr)
	}
}

// [EVT-BCU-5]
func TestValidateE2E_InvalidTemplateFails(t *testing.T) {
	t.Helper()
	tmp := t.TempDir()
	bad := writeTemplate(t, tmp, "bad.envseed", "APP_NAME # missing equals\n")
	if _, _, err := runEnvseed(t, "validate", bad); err == nil {
		t.Fatalf("validate expected failure, got success")
	}
}

// [EVT-BIU-7]
func TestValidateE2E_ENOTDIR_Classification(t *testing.T) {
	t.Helper()
	tmp := t.TempDir()
	// Create a regular file and then attempt to address a child entry -> ENOTDIR.
	file := filepath.Join(tmp, "notadir")
	if err := os.WriteFile(file, []byte("X=1\n"), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}
	bad := file + string(os.PathSeparator) + "child"
	stdout, stderr, err := runEnvseed(t, "validate", bad)
	if err == nil {
		t.Fatalf("expected failure for ENOTDIR path, got success: stdout=%s stderr=%s", stdout, stderr)
	}
	s := stdout + stderr
	if !strings.Contains(s, "EVE-102-3") {
		t.Fatalf("expected EVE-102-3 in diagnostics, got: %s", s)
	}
}

// [EVT-BIU-8]
func TestValidateE2E_ELOOP_Classification(t *testing.T) {
	t.Helper()
	tmp := t.TempDir()
	// Create a self-referential symlink: loop -> loop.
	loop := filepath.Join(tmp, "loop")
	if err := os.Symlink("loop", loop); err != nil {
		t.Fatalf("symlink self-loop: %v", err)
	}
	// Accessing loop/child should encounter a symlink loop during resolution.
	bad := filepath.Join(loop, "child")
	stdout, stderr, err := runEnvseed(t, "validate", bad)
	if err == nil {
		t.Fatalf("expected failure for ELOOP path, got success: stdout=%s stderr=%s", stdout, stderr)
	}
	s := stdout + stderr
	if !strings.Contains(s, "EVE-102-4") {
		t.Fatalf("expected EVE-102-4 in diagnostics, got: %s", s)
	}
}

// [EVT-BIU-9]
func TestValidateE2E_ENAMETOOLONG_Classification(t *testing.T) {
	t.Helper()
	tmp := t.TempDir()
	// Construct a single component exceeding typical NAME_MAX (e.g., 300).
	long := strings.Repeat("a", 300)
	bad := filepath.Join(tmp, long)
	stdout, stderr, err := runEnvseed(t, "validate", bad)
	if err == nil {
		t.Fatalf("expected failure for ENAMETOOLONG, got success: stdout=%s stderr=%s", stdout, stderr)
	}
	s := stdout + stderr
	if !strings.Contains(s, "EVE-102-5") {
		t.Fatalf("expected EVE-102-5 in diagnostics, got: %s", s)
	}
}
