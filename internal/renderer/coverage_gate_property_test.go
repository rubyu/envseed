package renderer_test

import (
	"testing"

	"envseed/internal/parser"
	renderer "envseed/internal/renderer"
	"envseed/internal/testgen"
	"envseed/internal/testsupport"
)

// [EVT-MGC-1]
func TestGeneratorCoverageGate_MinimalAxes(t *testing.T) {
	t.Helper()

	// Sample plans: a couple of diverse seeds with bounded iterations
	plans := []struct {
		seed  int64
		iters uint32
	}{
		{seed: 4242424242, iters: 256},
		{seed: -1123581321, iters: 256},
	}

	gate := testsupport.NewCoverageGate()
	// Combine coverage from round-trip and enhanced profiles
	rt := &testgen.RendererRoundTripProfile{}
	eh := &testgen.RendererEnhancedProfile{}
	for _, p := range plans {
		testgen.RunIterations(t, p.seed, p.iters, rt, func(t *testing.T, _ testgen.Meta, c testgen.Case) {
			elems, err := parser.Parse(c.Template)
			if err != nil {
				t.Fatalf("parse generated template (roundtrip): %v\n%s", err, c.Template)
			}
			gate.ObserveElements(elems)
		})
		testgen.RunIterations(t, p.seed, p.iters, eh, func(t *testing.T, _ testgen.Meta, c testgen.Case) {
			elems, err := parser.Parse(c.Template)
			if err != nil {
				t.Fatalf("parse generated template (enhanced): %v\n%s", err, c.Template)
			}
			gate.ObserveElements(elems)
		})
	}

	// Assert minimal renderer axes coverage
	wantContexts := []renderer.ValueContext{renderer.ContextBare, renderer.ContextDoubleQuoted, renderer.ContextSingleQuoted, renderer.ContextCommandSubstitution, renderer.ContextBacktick}
	wantModifiers := []string{"allow_tab", "allow_newline", "base64", "dangerously_bypass_escape"}
	gate.AssertAtLeast(t, wantContexts, true, wantModifiers)
}
