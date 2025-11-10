package main

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"envseed/internal/envseed"
)

func captureOutput(t *testing.T, fn func()) (string, string) {
	t.Helper()
	oldStdout := os.Stdout
	oldStderr := os.Stderr
	rOut, wOut, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe stdout: %v", err)
	}
	rErr, wErr, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe stderr: %v", err)
	}
	os.Stdout = wOut
	os.Stderr = wErr
	stdoutCh := make(chan string)
	stderrCh := make(chan string)
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, rOut)
		stdoutCh <- buf.String()
	}()
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, rErr)
		stderrCh <- buf.String()
	}()
	fn()
	_ = wOut.Close()
	_ = wErr.Close()
	os.Stdout = oldStdout
	os.Stderr = oldStderr
	stdout := <-stdoutCh
	stderr := <-stderrCh
	return stdout, stderr
}

// [EVT-BCU-10][EVT-BIU-6]
func TestRunSyncDefaultMissingReturnsTemplateRead102(t *testing.T) {
	dir := t.TempDir()
	oldwd, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(oldwd) })
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	err := runSync(context.Background(), []string{"--dry-run"})
	var exitErr *envseed.ExitError
	if !errors.As(err, &exitErr) || exitErr.Code != envseed.ExitTemplateRead {
		t.Fatalf("expected ExitTemplateRead(102), got: %v", err)
	}
}

// [EVT-BCU-2]
func TestRunSyncDryRunSuccess(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "simple.envseed")
	if err := os.WriteFile(input, []byte("APP_NAME=staging\n"), 0o600); err != nil {
		t.Fatalf("write template: %v", err)
	}
	stdout, _ := captureOutput(t, func() {
		if err := runSync(context.Background(), []string{"--dry-run", input}); err != nil {
			t.Fatalf("runSync error: %v", err)
		}
	})
	if !strings.Contains(stdout, "target:") {
		t.Fatalf("stdout missing target line: %q", stdout)
	}
	// stderr no longer emits a dry-run notice; only stdout is used for metadata
}

// [EVT-BCU-2]
func TestRunSyncQuietSuppressesInfo(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "quiet.envseed")
	if err := os.WriteFile(input, []byte("APP_NAME=staging\n"), 0o600); err != nil {
		t.Fatalf("write template: %v", err)
	}
	_, stderr := captureOutput(t, func() {
		if err := runSync(context.Background(), []string{"--dry-run", "--quiet", input}); err != nil {
			t.Fatalf("runSync error: %v", err)
		}
	})
	if strings.Contains(stderr, "dry-run") {
		t.Fatalf("quiet mode should suppress informational stderr, got %q", stderr)
	}
}

// [EVT-BCU-1][EVT-BCU-6]
func TestRunDiffExitCodes(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "diff.envseed")
	if err := os.WriteFile(input, []byte("APP_NAME=staging\n"), 0o600); err != nil {
		t.Fatalf("write template: %v", err)
	}
	output := filepath.Join(dir, "diff.env")
	if err := os.WriteFile(output, []byte("APP_NAME=production\n"), 0o600); err != nil {
		t.Fatalf("write output: %v", err)
	}
	stdout, _ := captureOutput(t, func() {
		err := runDiff(context.Background(), []string{input})
		var req exitRequest
		if !errors.As(err, &req) || req.code != 1 {
			t.Fatalf("expected exit code 1, got %v", err)
		}
	})
	if !strings.Contains(stdout, "---") {
		t.Fatalf("expected unified diff in stdout, got %q", stdout)
	}
	if err := os.WriteFile(output, []byte("APP_NAME=staging\n"), 0o600); err != nil {
		t.Fatalf("rewrite output: %v", err)
	}
	if err := runDiff(context.Background(), []string{input}); err != nil {
		t.Fatalf("runDiff with matching files returned error: %v", err)
	}
}

// [EVT-BCU-10][EVT-BDU-1]
func TestRunValidateDefaultMissingIs102(t *testing.T) {
	dir := t.TempDir()
	oldwd, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(oldwd) })
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	err := runValidate(context.Background(), nil)
	var exitErr *envseed.ExitError
	if !errors.As(err, &exitErr) || exitErr.Code != envseed.ExitTemplateRead {
		t.Fatalf("expected ExitTemplateRead(102), got: %v", err)
	}
	var req exitRequest
	if !errors.As(runValidate(context.Background(), []string{"-h"}), &req) || req.code != envseed.ExitOK {
		t.Fatalf("expected help to return exit OK, got %v", req)
	}
}

// [EVT-BCU-5]
func TestRunValidateSuccess(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "validate.envseed")
	if err := os.WriteFile(input, []byte("APP_NAME=staging\n"), 0o600); err != nil {
		t.Fatalf("write template: %v", err)
	}
	if err := runValidate(context.Background(), []string{input}); err != nil {
		t.Fatalf("runValidate error: %v", err)
	}
}

// removed local errorAs helper; use errors.As directly
