package e2e

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// [EVT-BCU-6]
func TestDiffE2E_ExitCodesAndOutput(t *testing.T) {
	t.Helper()

	tmp := t.TempDir()
	input := writeTemplate(t, tmp, "diff.envseed", "APP_NAME=staging\n")
	output := filepath.Join(tmp, "diff.env")

	// Case 1: different output should yield exit code 1 and show a diff.
	if err := os.WriteFile(output, []byte("APP_NAME=production\n"), 0o600); err != nil {
		t.Fatalf("write output (changed): %v", err)
	}
	stdout1, _, err1 := runEnvseed(t, "diff", "-o", output, input)
	if err1 == nil {
		t.Fatalf("expected non-zero exit code for changed files; output=%q", stdout1)
	}
	if !strings.Contains(stdout1, "--- ") {
		t.Fatalf("expected unified diff in output, got %q", stdout1)
	}

	// Case 2: matching output should yield exit code 0 and no diff.
	if err := os.WriteFile(output, []byte("APP_NAME=staging\n"), 0o600); err != nil {
		t.Fatalf("write output (matching): %v", err)
	}
	stdout2, _, err2 := runEnvseed(t, "diff", "-o", output, input)
	if err2 != nil {
		t.Fatalf("expected zero exit code for matching files, got error: %v\n%s", err2, stdout2)
	}
	if strings.TrimSpace(stdout2) != "" {
		t.Fatalf("expected empty diff for matching files, got %q", stdout2)
	}
}

// [EVT-BCU-1]
func TestDiffE2E_InvalidInputPath(t *testing.T) {
	t.Helper()
	// '-' is not allowed as input for diff per CLI contract
	if _, _, err := runEnvseed(t, "diff", "-"); err == nil {
		t.Fatalf("diff expected failure for '-' but got success")
	}
}

// [EVT-BCU-9]
func TestDiffE2E_ErrorLabelFormat(t *testing.T) {
	t.Helper()
	// Use a known-invalid invocation to trigger an error
	stdout, stderr, err := runEnvseed(t, "diff", "-")
	if err == nil {
		t.Fatalf("expected failure for invalid input, got success: stdout=%s stderr=%s", stdout, stderr)
	}
	s := stdout + stderr
	if !strings.Contains(s, "envseed ERROR [EVE-") {
		t.Fatalf("missing canonical error label prefix in output: %q", s)
	}
}

// [EVT-BCU-8]
func TestDiffUsesDerivedTargetHeadersWhenOutputIsDirectory(t *testing.T) {
	t.Helper()
	tmp := t.TempDir()
	in := writeTemplate(t, tmp, "service.envseed.prod", "NAME=new\n")
	outDir := filepath.Join(tmp, "out")
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		t.Fatalf("mkdir outdir: %v", err)
	}
	// Existing target file with different content to ensure a diff
	derived := filepath.Join(outDir, "service.env.prod")
	if err := os.WriteFile(derived, []byte("NAME=old\n"), 0o600); err != nil {
		t.Fatalf("write existing target: %v", err)
	}

	stdout, _, err := runEnvseed(t, "diff", "--output", outDir, in)
	if err == nil {
		t.Fatalf("expected non-zero exit for changed diff")
	}
	if !strings.Contains(stdout, "--- "+derived) || !strings.Contains(stdout, "+++ "+derived) {
		t.Fatalf("unified diff headers not absolute/derived as expected: %q", stdout)
	}
}

// [EVT-BIU-5]
func TestDiffFirstOccurrenceRule(t *testing.T) {
	t.Helper()
	tmp := t.TempDir()
	in := writeTemplate(t, tmp, "service.envseed.envseed", "NAME=new\n")
	derived := filepath.Join(tmp, "service.env.envseed")
	if err := os.WriteFile(derived, []byte("NAME=old\n"), 0o600); err != nil {
		t.Fatalf("write target: %v", err)
	}
	stdout, _, err := runEnvseed(t, "diff", in)
	if err == nil {
		t.Fatalf("expected non-zero exit for changed diff")
	}
	if !strings.Contains(stdout, "--- "+derived) || !strings.Contains(stdout, "+++ "+derived) {
		t.Fatalf("unified diff headers not equal to derived path: %q", stdout)
	}
}
