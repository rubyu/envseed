package main

import (
	"os"
	"path/filepath"
	"testing"
)

// [EVT-BDU-1] Basic unit tests for helper functions used by the generator.

// [EVT-BDU-1]
func TestParseCode(t *testing.T) {
	exit, sub := parseCode("EVE-103-501")
	if exit != 103 || sub != 501 {
		t.Fatalf("parseCode mismatch: got (%d,%d) want (103,501)", exit, sub)
	}
	exit, sub = parseCode("bad-format")
	if exit != 0 || sub != 0 {
		t.Fatalf("parseCode invalid should return zeros, got (%d,%d)", exit, sub)
	}
}

// [EVT-BDU-1]
func TestBandOfAndStart(t *testing.T) {
	if b, ok := bandOf(1); !ok || b != 0 {
		t.Fatalf("bandOf(1) = (%d,%v), want (0,true)", b, ok)
	}
	if b, ok := bandOf(101); !ok || b != 1 {
		t.Fatalf("bandOf(101) = (%d,%v), want (1,true)", b, ok)
	}
	if b, ok := bandOf(599); !ok || b != 5 {
		t.Fatalf("bandOf(599) = (%d,%v), want (5,true)", b, ok)
	}
	if _, ok := bandOf(600); ok {
		t.Fatalf("bandOf(600) should be invalid (false)")
	}
	if s := bandStart(0); s != 1 {
		t.Fatalf("bandStart(0) = %d, want 1", s)
	}
	if s := bandStart(5); s != 501 {
		t.Fatalf("bandStart(5) = %d, want 501", s)
	}
}

// [EVT-BDU-1]
func TestFindModuleRoot(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(wd) })

	// Case 1: finds nearest go.mod
	tmp := t.TempDir()
	// create nested dirs
	nested := filepath.Join(tmp, "a", "b")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("mkdir nested: %v", err)
	}
	// place go.mod at tmp
	if err := os.WriteFile(filepath.Join(tmp, "go.mod"), []byte("module example.com/tmp\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	if err := os.Chdir(nested); err != nil {
		t.Fatalf("chdir nested: %v", err)
	}
	if root := findModuleRoot(); root != tmp {
		t.Fatalf("findModuleRoot() = %q, want %q", root, tmp)
	}

	// Case 2: no go.mod anywhere → fallback to CWD
	tmp2 := t.TempDir()
	if err := os.Chdir(tmp2); err != nil {
		t.Fatalf("chdir tmp2: %v", err)
	}
	if root := findModuleRoot(); root != tmp2 {
		t.Fatalf("findModuleRoot() no-go.mod = %q, want %q", root, tmp2)
	}
}
