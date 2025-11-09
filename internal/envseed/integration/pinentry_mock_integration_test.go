//go:build integration
// +build integration

package envseed_integration

import (
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

// [EVT-BIF-4] pinentry-mock success path
func TestPassCommandShow_PinentryMock_Success(t *testing.T) {
	env := newPinentryMockEnv(t)

	// Build pinentry-mock
	mockPath := filepath.Join(env.base, "pinentry-mock")
	build := exec.Command("go", "build", "-o", mockPath, "./cmd/pinentry-mock")
	build.Env = os.Environ()
	if out, err := build.CombinedOutput(); err != nil {
		t.Skipf("skipping: build pinentry-mock failed: %v\n%s", err, out)
	}

	// Configure agent to use mock
	if err := os.WriteFile(filepath.Join(env.gpgHome, "gpg-agent.conf"), []byte("pinentry-program "+mockPath+"\n"), 0o600); err != nil {
		t.Fatalf("write gpg-agent.conf: %v", err)
	}
	_ = exec.Command("gpgconf", "--kill", "gpg-agent").Run()

	// Generate a key; pinentry-mock will provide passphrase
	name := "Envseed Test (mock) <mock@example.com>"
	gen := exec.Command("gpg", "--batch", "--quick-gen-key", name, "default", "default", "1d")
	gen.Env = append(os.Environ(),
		"GNUPGHOME="+env.gpgHome,
		"PINENTRY_MOCK_PASSPHRASE=testpass",
	)
	if out, err := gen.CombinedOutput(); err != nil {
		t.Skipf("skipping: gpg --quick-gen-key failed: %v\n%s", err, out)
	}

	// Extract fingerprint
	fprCmd := exec.Command("gpg", "--batch", "--homedir", env.gpgHome, "--with-colons", "--list-secret-keys")
	fprOut, err := fprCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("gpg --list-secret-keys failed: %v\n%s", err, fprOut)
	}
	fingerprint := extractFingerprint(string(fprOut))
	if fingerprint == "" {
		t.Fatalf("failed to extract fingerprint from gpg output:\n%s", fprOut)
	}

	// pass init and insert a secret
	initCmd := exec.Command("pass", "init", fingerprint)
	initCmd.Env = append(os.Environ(),
		"GNUPGHOME="+env.gpgHome,
		"PASSWORD_STORE_DIR="+env.storeDir,
		"HOME="+env.base,
	)
	if out, err := initCmd.CombinedOutput(); err != nil {
		t.Skipf("skipping: pass init failed: %v\n%s", err, out)
	}

	insert := exec.Command("pass", "insert", "-m", "demo/token")
	insert.Env = append(os.Environ(),
		"GNUPGHOME="+env.gpgHome,
		"PASSWORD_STORE_DIR="+env.storeDir,
		"HOME="+env.base,
	)
	insert.Stdin = strings.NewReader("secret-value\n")
	if out, err := insert.CombinedOutput(); err != nil {
		t.Fatalf("pass insert failed: %v\n%s", err, out)
	}

	// Resolve via PassCommand using pinentry-mock
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Ensure the agent sees mock env
	t.Setenv("GNUPGHOME", env.gpgHome)
	t.Setenv("PASSWORD_STORE_DIR", env.storeDir)
	t.Setenv("HOME", env.base)
	t.Setenv("PINENTRY_MOCK_PASSPHRASE", "testpass")

	out, err := (&envseedpkg.PassCommand{}).Show(ctx, "demo/token")
	if err != nil {
		t.Fatalf("Show() error = %v", err)
	}
	if strings.TrimSpace(out) != "secret-value" {
		t.Fatalf("unexpected output = %q", out)
	}
}

// [EVT-BIF-5] pinentry-mock generic failure path (wrong/cancel)
func TestPassCommandShow_PinentryMock_GenericFailure(t *testing.T) {
	env := newPinentryMockEnv(t)

	// Build pinentry-mock
	mockPath := filepath.Join(env.base, "pinentry-mock")
	build := exec.Command("go", "build", "-o", mockPath, "./cmd/pinentry-mock")
	if out, err := build.CombinedOutput(); err != nil {
		t.Skipf("skipping: build pinentry-mock failed: %v\n%s", err, out)
	}

	// Configure agent to use mock
	if err := os.WriteFile(filepath.Join(env.gpgHome, "gpg-agent.conf"), []byte("pinentry-program "+mockPath+"\n"), 0o600); err != nil {
		t.Fatalf("write gpg-agent.conf: %v", err)
	}
	_ = exec.Command("gpgconf", "--kill", "gpg-agent").Run()

	// Create a key (passphrase via mock)
	name := "Envseed Test (mock) <mock@example.com>"
	gen := exec.Command("gpg", "--batch", "--quick-gen-key", name, "default", "default", "1d")
	gen.Env = append(os.Environ(),
		"GNUPGHOME="+env.gpgHome,
		"PINENTRY_MOCK_PASSPHRASE=testpass",
	)
	if out, err := gen.CombinedOutput(); err != nil {
		t.Skipf("skipping: gpg --quick-gen-key failed: %v\n%s", err, out)
	}

	fprCmd := exec.Command("gpg", "--batch", "--homedir", env.gpgHome, "--with-colons", "--list-secret-keys")
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
		"GNUPGHOME="+env.gpgHome,
		"PASSWORD_STORE_DIR="+env.storeDir,
		"HOME="+env.base,
	)
	if out, err := initCmd.CombinedOutput(); err != nil {
		t.Skipf("skipping: pass init failed: %v\n%s", err, out)
	}

	insert := exec.Command("pass", "insert", "-m", "demo/token")
	insert.Env = append(os.Environ(),
		"GNUPGHOME="+env.gpgHome,
		"PASSWORD_STORE_DIR="+env.storeDir,
		"HOME="+env.base,
	)
	insert.Stdin = strings.NewReader("secret-value\n")
	if out, err := insert.CombinedOutput(); err != nil {
		t.Fatalf("pass insert failed: %v\n%s", err, out)
	}

	// Wrong passphrase should cause a generic resolver failure (EVE-104-101)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	t.Setenv("GNUPGHOME", env.gpgHome)
	t.Setenv("PASSWORD_STORE_DIR", env.storeDir)
	t.Setenv("HOME", env.base)

	// Kill agent to force a fresh prompt, then set wrong passphrase
	_ = exec.Command("gpgconf", "--kill", "gpg-agent").Run()
	t.Setenv("PINENTRY_MOCK_PASSPHRASE", "wrong")

	_, err = (&envseedpkg.PassCommand{}).Show(ctx, "demo/token")
	if err == nil {
		t.Fatal("expected error, got nil for wrong passphrase")
	}
	var exitErr *envseedpkg.ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("expected ExitError, got %v", err)
	}
	if exitErr.DetailCode != "EVE-104-101" {
		t.Fatalf("detail code = %s, want EVE-104-101", exitErr.DetailCode)
	}
}

type pinentryMockEnv struct {
	base     string
	gpgHome  string
	storeDir string
}

func newPinentryMockEnv(t *testing.T) *pinentryMockEnv {
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
	return &pinentryMockEnv{base: base, gpgHome: gpgHome, storeDir: storeDir}
}
