package envseed

import (
	"testing"

	"envseed/internal/ast"
	"envseed/internal/parser"
	"envseed/internal/testgen"
	"envseed/internal/testsupport"
)

// [EVT-BGC-1]
// Basic generator coverage gate applied in the CLI/core package to ensure
// that our standard template profiles exercise a minimal set of axes. While
// the CLI semantics themselves are validated in E2E tests, this gate keeps
// the underlying template space healthy from the perspective of contexts and
// modifiers used by Sync/Validate flows.
// [EVT-BGC-1]
func TestCLIGeneratorCoverageGate_MinimalAxes(t *testing.T) {
	t.Helper()
	plans := []struct {
		seed  int64
		iters uint32
	}{
		{seed: 123456789, iters: 128},
		{seed: -987654321, iters: 128},
	}
	gate := testsupport.NewCoverageGate()
	prof := &testgen.CLISemanticsProfile{}
	for _, p := range plans {
		testgen.RunIterations(t, p.seed, p.iters, prof, func(t *testing.T, _ testgen.Meta, c testgen.Case) {
			elems, err := parser.Parse(c.Template)
			if err != nil {
				t.Fatalf("parse generated template: %v\n%s", err, c.Template)
			}
			gate.ObserveElements(elems)
		})
	}
	wantContexts := []ast.ValueContext{ast.ContextBare, ast.ContextDoubleQuoted}
	gate.AssertAtLeast(t, wantContexts, false, nil)
}
