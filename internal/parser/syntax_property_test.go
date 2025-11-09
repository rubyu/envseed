package parser_test

import (
	"fmt"
	"strings"
	"testing"

	"envseed/internal/ast"
	parser "envseed/internal/parser"
	"envseed/internal/testgen"
)

// [EVT-MGP-2]
func TestParser_SyntaxProfile_RoundTrip(t *testing.T) {
	prof := &testgen.ParserSyntaxProfile{}
	plans := []struct {
		seed  int64
		iters uint32
	}{
		{seed: 3141592653, iters: 128},
		{seed: -2718281828, iters: 128},
	}
	for _, p := range plans {
		testgen.RunIterations(t, p.seed, p.iters, prof, func(t *testing.T, _ testgen.Meta, c testgen.Case) {
			elems, err := parser.Parse(c.Template)
			if err != nil {
				t.Fatalf("Parse generated template: %v\n%s", err, c.Template)
			}
			out, err := RenderElementsForTest(elems)
			if err != nil {
				t.Fatalf("RenderElementsForTest: %v", err)
			}
			back, err := parser.Parse(out)
			if err != nil {
				t.Fatalf("Parse(rendered) failed: %v\nrendered=%q", err, out)
			}
			if len(back) != len(elems) {
				t.Fatalf("element count changed: got %d want %d", len(back), len(elems))
			}
		})
	}
}

// RenderElementsForTest is a minimal renderer for testing parser round-trips.
// It writes back the textual form from AST without invoking secret resolution.
func RenderElementsForTest(elems []ast.Element) (string, error) {
	var b strings.Builder
	for _, e := range elems {
		switch e.Type {
		case ast.ElementBlank:
			b.WriteString(e.Text)
		case ast.ElementComment:
			b.WriteString(e.Text)
			if e.HasTrailingNewline {
				b.WriteString("\n")
			}
		case ast.ElementAssignment:
			a := e.Assignment
			b.WriteString(a.LeadingWhitespace)
			b.WriteString(a.Name)
			if a.Operator == ast.OperatorAppend {
				b.WriteString("+=")
			} else {
				b.WriteString("=")
			}
			for _, tok := range a.ValueTokens {
				b.WriteString(tok.Text)
			}
			b.WriteString(a.TrailingComment)
			if a.HasTrailingNewline {
				b.WriteString("\n")
			}
		default:
			return "", fmt.Errorf("unknown element type %v", e.Type)
		}
	}
	return b.String(), nil
}
