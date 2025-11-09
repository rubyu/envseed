package e2e

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// [EVT-BCU-2]
func TestSyncE2E_DryRunBasic(t *testing.T) {
	t.Helper()
	tmp := t.TempDir()
	input := writeTemplate(t, tmp, "sync.envseed", "FOO=bar\n")

	stdout, stderr, err := runEnvseed(t, "sync", "--dry-run", input)
	if err != nil {
		t.Fatalf("sync --dry-run failed: %v (stderr=%s)", err, stderr)
	}
	if !strings.Contains(stdout, "target:") {
		t.Fatalf("stdout missing target header: %q", stdout)
	}
}

// [EVT-BIU-5]
func TestSyncFirstOccurrenceRule(t *testing.T) {
	t.Helper()
	tmp := t.TempDir()
	in := writeTemplate(t, tmp, "foo.envseed.envseed", "X=1\n")
	stdout, stderr, err := runEnvseed(t, "sync", "--dry-run", in)
	if err != nil {
		t.Fatalf("sync --dry-run failed: %v (stderr=%s)", err, stderr)
	}
	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	if len(lines) == 0 || !strings.HasPrefix(lines[0], "target: ") {
		t.Fatalf("missing target header: %q", stdout)
	}
	got := strings.TrimPrefix(lines[0], "target: ")
	want := filepath.Join(tmp, "foo.env.envseed")
	if got != want {
		t.Fatalf("first-occurrence mapping mismatch: got %q want %q", got, want)
	}
}

// [EVT-BCU-2]
func TestSyncE2E_WritesExplicitOutputFile(t *testing.T) {
	t.Helper()
	tmp := t.TempDir()

	input := writeTemplate(t, tmp, "sync.envseed", "FOO=bar\n")
	outPath := filepath.Join(tmp, "result.env")
	if _, stderr, err := runEnvseed(t, "sync", "--output", outPath, input); err != nil {
		t.Fatalf("sync failed: %v (stderr=%s)", err, stderr)
	}
	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}
	if string(data) != "FOO=bar\n" {
		t.Fatalf("output mismatch: %q", string(data))
	}
}

// [EVT-BCU-2]
func TestSyncE2E_WritesDerivedOutputInDirectory(t *testing.T) {
	t.Helper()
	tmp := t.TempDir()

	input := writeTemplate(t, tmp, "service.envseed", "FOO=bar\n")

	// Use --output with a directory path; output filename should be derived
	// from the input (envseed -> env).
	outDir := filepath.Join(tmp, "outdir")
	if err := os.MkdirAll(outDir, 0o700); err != nil {
		t.Fatalf("mkdir outdir: %v", err)
	}
	if _, stderr, err := runEnvseed(t, "sync", "--output", outDir+string(os.PathSeparator), input); err != nil {
		t.Fatalf("sync failed: %v (stderr=%s)", err, stderr)
	}
	derived := filepath.Join(outDir, "service.env")
	data, err := os.ReadFile(derived)
	if err != nil {
		t.Fatalf("read derived output: %v", err)
	}
	if string(data) != "FOO=bar\n" {
		t.Fatalf("derived output mismatch: %q", string(data))
	}
}

// [EVT-BCU-2]
func TestSyncE2E_DryRunQuietSuppressesInfo(t *testing.T) {
	t.Helper()
	tmp := t.TempDir()
	input := writeTemplate(t, tmp, "quiet.envseed", "FOO=bar\n")

	stdout, stderr, err := runEnvseed(t, "sync", "--dry-run", "--quiet", input)
	if err != nil {
		t.Fatalf("sync --dry-run --quiet failed: %v (stderr=%s)", err, stderr)
	}
	// Quiet mode should suppress informational text; ensure 'dry-run' is absent.
	if strings.Contains(stdout, "dry-run") || strings.Contains(stderr, "dry-run") {
		t.Fatalf("quiet mode should suppress informational text, got stdout=%q stderr=%q", stdout, stderr)
	}
}

// [EVT-BCU-2]
func TestSyncE2E_DryRun_NoStderr(t *testing.T) {
	t.Helper()
	tmp := t.TempDir()
	input := writeTemplate(t, tmp, "dryrun.envseed", "FOO=bar\n")

	stdout, stderr, err := runEnvseed(t, "sync", "--dry-run", input)
	if err != nil {
		t.Fatalf("sync --dry-run failed: %v (stdout=%s, stderr=%s)", err, stdout, stderr)
	}
	if !strings.HasPrefix(stdout, "target:") {
		t.Fatalf("stdout should start with 'target:' header, got %q", stdout)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("stderr should be empty for dry-run, got %q", stderr)
	}
}
