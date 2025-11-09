package envseed

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// [EVT-MSU-1][EVT-BCU-6][EVT-BSP-1][EVT-BCU-8]
func TestDiffDetectsChangesAndMasksSecrets(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	input := filepath.Join(dir, ".envseed")
	if err := os.WriteFile(input, []byte("TOKEN=<pass:token|strip_right>\n"), 0o600); err != nil {
		t.Fatalf("write template: %v", err)
	}
	output := filepath.Join(dir, ".env")
	if err := os.WriteFile(output, []byte("TOKEN=oldvalue\n"), 0o600); err != nil {
		t.Fatalf("write existing: %v", err)
	}

	pass := &fakePass{values: map[string]string{"token": "abcdefghi"}}
	var stdout bytes.Buffer
	res, err := Diff(context.Background(), DiffOptions{
		InputPath:  input,
		PassClient: pass,
		Stdout:     &stdout,
	})
	if err != nil {
		t.Fatalf("Diff() error = %v", err)
	}
	if !res.Changed {
		t.Fatalf("expected changes")
	}
	diff := stdout.String()
	if !strings.Contains(diff, "a*******i") {
		t.Fatalf("masked secret missing from diff: %s", diff)
	}
	if strings.Contains(diff, "abcdefghi") {
		t.Fatalf("diff leaked secret: %s", diff)
	}
	if !strings.Contains(diff, "--- "+output) || !strings.Contains(diff, "+++ "+output) {
		t.Fatalf("diff missing file headers: %s", diff)
	}
}

// [EVT-BCU-6]
func TestDiffNoChanges(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	input := filepath.Join(dir, ".envseed")
	if err := os.WriteFile(input, []byte("TOKEN=<pass:token|strip_right>\n"), 0o600); err != nil {
		t.Fatalf("write template: %v", err)
	}
	output := filepath.Join(dir, ".env")
	if err := os.WriteFile(output, []byte("TOKEN=value\n"), 0o600); err != nil {
		t.Fatalf("write existing: %v", err)
	}

	pass := &fakePass{values: map[string]string{"token": "value"}}
	var stdout bytes.Buffer
	res, err := Diff(context.Background(), DiffOptions{
		InputPath:  input,
		PassClient: pass,
		Stdout:     &stdout,
	})
	if err != nil {
		t.Fatalf("Diff() error = %v", err)
	}
	if res.Changed {
		t.Fatalf("expected no changes")
	}
	if stdout.Len() != 0 {
		t.Fatalf("expected no diff output, got %s", stdout.String())
	}
}

// [EVT-BCU-6][EVT-BIF-1]
func TestDiffHandlesMissingTarget(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	input := filepath.Join(dir, "missing.envseed")
	if err := os.WriteFile(input, []byte("VALUE=data\n"), 0o600); err != nil {
		t.Fatalf("write template: %v", err)
	}
	var stdout bytes.Buffer
	res, err := Diff(context.Background(), DiffOptions{
		InputPath: input,
		Stdout:    &stdout,
	})
	if err != nil {
		t.Fatalf("Diff error: %v", err)
	}
	if !res.Changed {
		t.Fatal("expected diff to report changes when target missing")
	}
	if !strings.Contains(stdout.String(), "--- ") {
		t.Fatalf("expected diff header, got %q", stdout.String())
	}
}

// [EVT-BIF-1]
func TestDiffAllowsBoundarySize(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	input := filepath.Join(dir, "boundary.envseed")
	valueLen := diffSizeLimit - len("VALUE=")
	template := "VALUE=" + strings.Repeat("A", valueLen)
	if err := os.WriteFile(input, []byte(template), 0o600); err != nil {
		t.Fatalf("write template: %v", err)
	}
	output := filepath.Join(dir, "boundary.env")
	if err := os.WriteFile(output, []byte("VALUE="+strings.Repeat("B", valueLen)), 0o600); err != nil {
		t.Fatalf("write output: %v", err)
	}
	var stdout bytes.Buffer
	res, err := Diff(context.Background(), DiffOptions{
		InputPath:  input,
		OutputPath: output,
		Stdout:     &stdout,
	})
	if err != nil {
		t.Fatalf("Diff error: %v", err)
	}
	if !res.Changed {
		t.Fatalf("expected differences at boundary size")
	}
}

// [EVT-BIU-2]
func TestDiffRejectsLargeOutputs(t *testing.T) {
	t.Parallel()

	secret := strings.Repeat("a", diffSizeLimit+1)

	dir := t.TempDir()
	input := filepath.Join(dir, ".envseed")
	if err := os.WriteFile(input, []byte("TOKEN=<pass:token|strip_right>\n"), 0o600); err != nil {
		t.Fatalf("write template: %v", err)
	}

	pass := &fakePass{values: map[string]string{"token": secret}}
	_, err := Diff(context.Background(), DiffOptions{
		InputPath:  input,
		PassClient: pass,
		Stdout:     io.Discard,
	})
	var exitErr *ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("expected ExitError, got %v", err)
	}
	if exitErr.DetailCode != "EVE-108-1" {
		t.Fatalf("detail code = %s, want EVE-108-1", exitErr.DetailCode)
	}
}
