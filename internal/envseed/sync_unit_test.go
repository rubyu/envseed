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

	"envseed/internal/parser"
	"envseed/internal/renderer"
)

// [EVT-BIU-1][EVT-BCU-2]
func TestSyncSimpleReplacement(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	input := filepath.Join(dir, ".envseed")
	template := strings.Join([]string{
		`# sample`,
		`API_TOKEN=<pass:service/api-token|strip_right>`,
		`MESSAGE="hello <pass:user|strip_right>"`,
		`LOG_LEVEL=info`,
	}, "\n") + "\n"
	if err := os.WriteFile(input, []byte(template), 0o600); err != nil {
		t.Fatalf("write template: %v", err)
	}

	pass := &fakePass{
		values: map[string]string{
			"service/api-token": "tokenValue",
			"user":              `world$`,
		},
	}

	var stderr bytes.Buffer
	if err := Sync(context.Background(), SyncOptions{
		InputPath:  input,
		PassClient: pass,
		Stderr:     &stderr,
	}); err != nil {
		t.Fatalf("Sync() error = %v", err)
	}

	output := filepath.Join(dir, ".env")
	data, err := os.ReadFile(output)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}
	got := string(data)
	want := strings.Join([]string{
		`# sample`,
		`API_TOKEN=tokenValue`,
		`MESSAGE="hello world\$"`,
		`LOG_LEVEL=info`,
	}, "\n") + "\n"
	if got != want {
		t.Fatalf("output mismatch\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}

	if stderr.Len() == 0 {
		t.Fatalf("expected informational stderr output")
	}
	if pass.calls["service/api-token"] != 1 || pass.calls["user"] != 1 {
		t.Fatalf("unexpected pass calls: %#v", pass.calls)
	}

	info, err := os.Stat(output)
	if err != nil {
		t.Fatalf("stat output: %v", err)
	}
	if perms := info.Mode().Perm(); perms != 0o600 {
		t.Fatalf("permissions = %o, want 0600", perms)
	}
}

// [EVT-BIU-1]
func TestSyncRefusesOverwriteWithoutForce(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	input := filepath.Join(dir, ".envseed")
	if err := os.WriteFile(input, []byte("TOKEN=<pass:path|strip_right>\n"), 0o600); err != nil {
		t.Fatalf("write template: %v", err)
	}
	output := filepath.Join(dir, ".env")
	if err := os.WriteFile(output, []byte("TOKEN=old\n"), 0o600); err != nil {
		t.Fatalf("write output: %v", err)
	}

	pass := &fakePass{values: map[string]string{"path": "newsecret"}}
	err := Sync(context.Background(), SyncOptions{
		InputPath:  input,
		PassClient: pass,
		Stderr:     io.Discard,
		Stdout:     io.Discard,
	})
	var exitErr *ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("expected ExitError, got %v", err)
	}
	if exitErr.DetailCode != "EVE-106-101" {
		t.Fatalf("detail code = %s, want EVE-106-101", exitErr.DetailCode)
	}
}

// [EVT-MSU-1][EVT-BSP-1][EVT-BCU-2]
func TestSyncDryRunRedaction(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	input := filepath.Join(dir, ".envseed")
	template := strings.Join([]string{
		`SHORT=<pass:short|strip_right>`,
		`MEDIUM=<pass:medium|strip_right>`,
		`LONG=<pass:long|strip_right>`,
	}, "\n") + "\n"
	if err := os.WriteFile(input, []byte(template), 0o600); err != nil {
		t.Fatalf("write template: %v", err)
	}

	pass := &fakePass{
		values: map[string]string{
			"short":  "abcde",
			"medium": "abcdefghij",
			"long":   "abcdefghijklmnopqr",
		},
	}

	var stdout, stderr bytes.Buffer
	if err := Sync(context.Background(), SyncOptions{
		InputPath:  input,
		PassClient: pass,
		DryRun:     true,
		Stdout:     &stdout,
		Stderr:     &stderr,
	}); err != nil {
		t.Fatalf("Sync() error = %v", err)
	}

	out := stdout.String()
	if !strings.Contains(out, "target: "+filepath.Join(dir, ".env")) {
		t.Fatalf("stdout missing target path: %s", out)
	}
	if strings.Contains(out, "abcde") || strings.Contains(out, "abcdefghij") || strings.Contains(out, "abcdefghijklmnopqr") {
		t.Fatalf("stdout leaked secret content: %s", out)
	}
	if !strings.Contains(out, "SHORT=*****") {
		t.Fatalf("short secret not fully masked: %s", out)
	}
	if !strings.Contains(out, "MEDIUM=a********j") {
		t.Fatalf("medium secret not masked correctly: %s", out)
	}
	if !strings.Contains(out, "LONG=ab**************qr") {
		t.Fatalf("long secret not masked correctly: %s", out)
	}
	if _, err := os.Stat(filepath.Join(dir, ".env")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("dry-run wrote output file unexpectedly")
	}
}

// [EVT-MSU-1][EVT-BSP-1]
func TestSyncDryRunRedactsAcrossContexts(t *testing.T) {
	t.Parallel()

	template := strings.Join([]string{
		"BARE1=<pass:shared|strip_right>",
		"BARE2=<pass:shared|strip_right>",
		"DOUBLE1=\"<pass:shared|strip_right>\"",
		"DOUBLE2=\"<pass:shared|strip_right>\"",
		"CMD1=$(printf %s <pass:shared|strip_right>)",
		"CMD2=$(printf %s <pass:shared|strip_right>)",
		"BACKTICK1=`echo <pass:shared|strip_right>`",
		"BACKTICK2=`echo <pass:shared|strip_right>`",
		"SINGLE1='<pass:shared|strip_right>'",
		"SINGLE2='<pass:shared|strip_right>'",
	}, "\n") + "\n"

	dir := t.TempDir()
	input := filepath.Join(dir, "matrix.envseed")
	if err := os.WriteFile(input, []byte(template), 0o600); err != nil {
		t.Fatalf("write template: %v", err)
	}

	pass := &staticPassClient{values: map[string]string{"shared": "SensitiveValue!"}}
	var stdout, stderr bytes.Buffer
	if err := Sync(context.Background(), SyncOptions{
		InputPath:  input,
		PassClient: pass,
		DryRun:     true,
		Stdout:     &stdout,
		Stderr:     &stderr,
	}); err != nil {
		t.Fatalf("Sync() error = %v", err)
	}

	if strings.Contains(stdout.String(), "SensitiveValue!") {
		t.Fatalf("stdout leaked secret: %q", stdout.String())
	}

	parts := strings.SplitN(stdout.String(), "\n", 2)
	if len(parts) != 2 {
		t.Fatalf("unexpected stdout format: %q", stdout.String())
	}
	redacted := parts[1]
	elems, err := parser.Parse(redacted)
	if err != nil {
		t.Fatalf("Parse(redacted) error: %v", err)
	}
	values := make(map[string]string)
	for _, elem := range elems {
		if elem.Type != renderer.ElementAssignment {
			continue
		}
		assign := elem.Assignment
		var b strings.Builder
		for _, tok := range assign.ValueTokens {
			b.WriteString(tok.Text)
		}
		values[assign.Name] = b.String()
	}
	checkPairs := [][2]string{{"BARE1", "BARE2"}, {"DOUBLE1", "DOUBLE2"}, {"CMD1", "CMD2"}, {"BACKTICK1", "BACKTICK2"}, {"SINGLE1", "SINGLE2"}}
	for _, pair := range checkPairs {
		v1, ok1 := values[pair[0]]
		v2, ok2 := values[pair[1]]
		if !ok1 || !ok2 {
			t.Fatalf("missing assignments %s/%s in redacted output: %#v", pair[0], pair[1], values)
		}
		if v1 != v2 {
			t.Fatalf("redacted values for %s and %s differ: %q vs %q", pair[0], pair[1], v1, v2)
		}
		if !strings.Contains(v1, "*") {
			t.Fatalf("redacted value for %s lacks masking: %q", pair[0], v1)
		}
	}
}

// [EVT-BIU-3]
func TestSyncWritesToOutputDirectory(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	input := filepath.Join(dir, "config.envseed.local")
	outDir := filepath.Join(dir, "outputs")
	if err := os.Mkdir(outDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(input, []byte("KEY=<pass:key|strip_right>\n"), 0o600); err != nil {
		t.Fatalf("write template: %v", err)
	}

	pass := &fakePass{values: map[string]string{"key": "value"}}
	if err := Sync(context.Background(), SyncOptions{
		InputPath:  input,
		OutputPath: outDir,
		PassClient: pass,
		Quiet:      true,
		Stdout:     io.Discard,
		Stderr:     io.Discard,
	}); err != nil {
		t.Fatalf("Sync() error = %v", err)
	}

	target := filepath.Join(outDir, "config.env.local")
	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}
	if got := string(data); got != "KEY=value\n" {
		t.Fatalf("output = %q, want KEY=value", got)
	}
}

// [EVT-BCU-3][EVT-BIU-1]
func TestSyncMessageFormat(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	input := filepath.Join(dir, "format.envseed")
	if err := os.WriteFile(input, []byte("NAME=value\n"), 0o600); err != nil {
		t.Fatalf("write template: %v", err)
	}
	output := filepath.Join(dir, "format.env")
	var stderr bytes.Buffer
	if err := Sync(context.Background(), SyncOptions{
		InputPath: input,
		Stdout:    io.Discard,
		Stderr:    &stderr,
	}); err != nil {
		t.Fatalf("Sync error: %v", err)
	}
	msg := stderr.String()
	if !strings.Contains(msg, "wrote "+output+" (mode 0600)") {
		t.Fatalf("unexpected write message: %q", msg)
	}
	stderr.Reset()
	if err := Sync(context.Background(), SyncOptions{
		InputPath: input,
		Stdout:    io.Discard,
		Stderr:    &stderr,
	}); err != nil {
		t.Fatalf("Sync (unchanged) error: %v", err)
	}
	if got := stderr.String(); !strings.Contains(got, "wrote "+output+" (unchanged)") {
		t.Fatalf("expected unchanged message, got %q", got)
	}
}
