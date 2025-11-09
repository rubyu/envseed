//go:build integration
// +build integration

package envseed_integration

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	envseedpkg "envseed/internal/envseed"
)

// [EVT-BIF-2]
func TestPassCommandShowMissingEntryIntegration(t *testing.T) {
	_ = newPassTestEnv(t)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := (&envseedpkg.PassCommand{}).Show(ctx, "missing/entry")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var exitErr *envseedpkg.ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("expected ExitError, got %v", err)
	}
	if exitErr.DetailCode != "EVE-104-201" {
		t.Fatalf("detail code = %s, want EVE-104-201", exitErr.DetailCode)
	}
}

// [EVT-BZU-1]
func TestPassCommandShowSuccessIntegration(t *testing.T) {
	env := newPassTestEnv(t)
	env.insertSecret(t, "service/token", "super-secret-value")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	out, err := (&envseedpkg.PassCommand{}).Show(ctx, "service/token")
	if err != nil {
		t.Fatalf("Show() error = %v", err)
	}
	if out != "super-secret-value\n" {
		t.Fatalf("unexpected output = %q, want super-secret-value\\n", out)
	}

	// Call Show twice to ensure repeated retrieval succeeds against the store.
	out2, err := (&envseedpkg.PassCommand{}).Show(ctx, "service/token")
	if err != nil {
		t.Fatalf("Show() second call error = %v", err)
	}
	if out2 != "super-secret-value\n" {
		t.Fatalf("unexpected output on second call = %q, want super-secret-value\\n", out2)
	}
}

type passTestEnv struct {
	gpgHome  string
	storeDir string
	homeDir  string
}

func newPassTestEnv(t *testing.T) *passTestEnv {
	t.Helper()

	requireCommand(t, "pass")
	requireCommand(t, "gpg")

	base := t.TempDir()
	gpgHome := filepath.Join(base, "gnupg")
	if err := os.Mkdir(gpgHome, 0o700); err != nil {
		t.Fatalf("mkdir gnupg: %v", err)
	}
	storeDir := filepath.Join(base, "password-store")
	if err := os.Mkdir(storeDir, 0o700); err != nil {
		t.Fatalf("mkdir password-store: %v", err)
	}

	if err := os.WriteFile(filepath.Join(gpgHome, "gpg.conf"), []byte("pinentry-mode loopback\n"), 0o600); err != nil {
		t.Fatalf("write gpg.conf: %v", err)
	}
	if err := os.WriteFile(filepath.Join(gpgHome, "gpg-agent.conf"), []byte("allow-loopback-pinentry\n"), 0o600); err != nil {
		t.Fatalf("write gpg-agent.conf: %v", err)
	}

	t.Setenv("GNUPGHOME", gpgHome)
	t.Setenv("PASSWORD_STORE_DIR", storeDir)
	t.Setenv("PASSWORD_STORE_GPG_OPTS", "--pinentry-mode loopback")
	t.Setenv("HOME", base)

	genSpec := bytes.NewBufferString(strings.TrimSpace(`
Key-Type: RSA
Key-Length: 2048
Subkey-Type: RSA
Subkey-Length: 2048
Name-Real: Envseed Test
Name-Email: envseed-test@example.com
Expire-Date: 1d
%no-protection
%commit
`))

	genCmd := exec.Command("gpg", "--batch", "--pinentry-mode", "loopback", "--homedir", gpgHome, "--gen-key")
	genCmd.Stdin = genSpec
	if out, err := genCmd.CombinedOutput(); err != nil {
		t.Skipf("skipping pass integration tests: gpg --gen-key failed: %v\n%s", err, out)
	}

	fprCmd := exec.Command("gpg", "--batch", "--homedir", gpgHome, "--with-colons", "--list-secret-keys")
	fprOut, err := fprCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("gpg --list-secret-keys failed: %v\n%s", err, fprOut)
	}
	fingerprint := extractFingerprint(string(fprOut))
	if fingerprint == "" {
		t.Fatalf("failed to extract fingerprint from gpg output:\n%s", fprOut)
	}

	initCmd := exec.Command("pass", "init", fingerprint)
	initCmd.Env = append(os.Environ(),
		"GNUPGHOME="+gpgHome,
		"PASSWORD_STORE_DIR="+storeDir,
		"PASSWORD_STORE_GPG_OPTS=--pinentry-mode loopback",
		"HOME="+base,
	)
	if out, err := initCmd.CombinedOutput(); err != nil {
		t.Skipf("skipping pass integration tests: pass init failed: %v\n%s", err, out)
	}

	return &passTestEnv{
		gpgHome:  gpgHome,
		storeDir: storeDir,
		homeDir:  base,
	}
}

func (env *passTestEnv) insertSecret(t *testing.T, path, secret string) {
	t.Helper()

	if !strings.HasSuffix(secret, "\n") {
		secret += "\n"
	}

	cmd := exec.Command("pass", "insert", "-m", path)
	cmd.Env = append(os.Environ(),
		"GNUPGHOME="+env.gpgHome,
		"PASSWORD_STORE_DIR="+env.storeDir,
		"PASSWORD_STORE_GPG_OPTS=--pinentry-mode loopback",
		"HOME="+env.homeDir,
	)
	cmd.Stdin = strings.NewReader(secret)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("pass insert failed: %v\n%s", err, out)
	}
}

func requireCommand(t *testing.T, name string) {
	t.Helper()
	if _, err := exec.LookPath(name); err != nil {
		t.Skipf("%s not available: %v", name, err)
	}
}

func extractFingerprint(listOutput string) string {
	lines := strings.Split(listOutput, "\n")
	for _, line := range lines {
		if !strings.HasPrefix(line, "fpr:") {
			continue
		}
		parts := strings.Split(line, ":")
		if len(parts) > 9 && parts[9] != "" {
			return parts[9]
		}
		if len(parts) >= 2 {
			return parts[len(parts)-2]
		}
	}
	return ""
}
