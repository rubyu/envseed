package e2e

import (
	"testing"
)

// [EVT-BCU-5]
func TestValidateE2E_SimpleTemplate(t *testing.T) {
	t.Helper()
	tmp := t.TempDir()
	input := writeTemplate(t, tmp, "validate.envseed", "FOO=bar\n")
	if _, stderr, err := runEnvseed(t, "validate", input); err != nil {
		t.Fatalf("validate failed: %v (stderr=%s)", err, stderr)
	}
}

// [EVT-BCU-5]
func TestValidateE2E_InvalidTemplateFails(t *testing.T) {
	t.Helper()
	tmp := t.TempDir()
	bad := writeTemplate(t, tmp, "bad.envseed", "FOO # missing equals\n")
	if _, _, err := runEnvseed(t, "validate", bad); err == nil {
		t.Fatalf("validate expected failure, got success")
	}
}
