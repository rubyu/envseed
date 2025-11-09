package parser_test

import (
	"testing"

	"envseed/internal/ast"
	"envseed/internal/parser"
	"envseed/internal/testgen"
	"envseed/internal/testsupport"
)

// [EVT-MGC-2]
func TestParserGeneratorCoverageGate_MinimalAxes(t *testing.T) {
	t.Helper()

	plans := []struct {
		seed  int64
		iters uint32
	}{
		{seed: 3141592653, iters: 256},
		{seed: -2718281828, iters: 256},
	}
	gate := testsupport.NewCoverageGate()
	prof := &testgen.ParserSyntaxProfile{}
	for _, p := range plans {
		testgen.RunIterations(t, p.seed, p.iters, prof, func(t *testing.T, _ testgen.Meta, c testgen.Case) {
			elems, err := parser.Parse(c.Template)
			if err != nil {
				t.Fatalf("parse generated template: %v\n%s", err, c.Template)
			}
			gate.ObserveElements(elems)
		})
	}
	wantContexts := []ast.ValueContext{ast.ContextBare, ast.ContextDoubleQuoted, ast.ContextSingleQuoted, ast.ContextCommandSubstitution, ast.ContextBacktick}
	wantMods := []string{"allow_tab", "allow_newline", "base64", "dangerously_bypass_escape"}
	gate.AssertAtLeast(t, wantContexts, true, wantMods)
}
