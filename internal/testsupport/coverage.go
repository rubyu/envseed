package testsupport

// Shared coverage helpers for property-based/generative tests.
// These utilities focus on specification coverage axes (contexts, operators,
// modifiers) rather than implementation line/branch coverage.

import (
	"testing"

	"envseed/internal/ast"
)

// CoverageGate observes parser AST elements and records which axes were seen.
// After observation, AssertAtLeast can be used to ensure a minimal set of axes
// was exercised.
type CoverageGate struct {
	seenContexts  map[ast.ValueContext]bool
	seenModifiers map[string]bool
	seenPlusEqual bool
}

// NewCoverageGate creates a new, empty coverage gate.
func NewCoverageGate() *CoverageGate {
	return &CoverageGate{
		seenContexts:  make(map[ast.ValueContext]bool, 8),
		seenModifiers: make(map[string]bool, 16),
	}
}

// ObserveElements scans AST elements, recording operators, value contexts,
// and modifiers for later assertions.
func (g *CoverageGate) ObserveElements(elems []ast.Element) {
	for _, e := range elems {
		if e.Assignment == nil {
			continue
		}
		if e.Assignment.Operator == ast.OperatorAppend {
			g.seenPlusEqual = true
		}
		for _, tok := range e.Assignment.ValueTokens {
			g.seenContexts[tok.Context] = true
			for _, m := range tok.Modifiers {
				g.seenModifiers[m] = true
			}
		}
	}
}

// AssertAtLeast verifies that the observed coverage includes all required
// contexts and modifiers, and optionally the "+=" operator.
func (g *CoverageGate) AssertAtLeast(t testing.TB, wantContexts []ast.ValueContext, wantPlusEqual bool, wantModifiers []string) {
	t.Helper()
	// contexts
	for _, c := range wantContexts {
		if !g.seenContexts[c] {
			t.Fatalf("context coverage incomplete: missing %v", c)
		}
	}
	// operator
	if wantPlusEqual && !g.seenPlusEqual {
		t.Fatalf("operator '+=' not observed in generated templates")
	}
	// modifiers
	for _, m := range wantModifiers {
		if !g.seenModifiers[m] {
			t.Fatalf("modifier coverage incomplete: missing %s", m)
		}
	}
}
