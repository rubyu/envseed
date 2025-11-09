package parser_test

import (
	"fmt"
	"testing"

	"envseed/internal/parser"
	"envseed/internal/testgen"
)

// [EVT-MGP-1][EVT-MGP-2]
func TestParserASTRoundTrip(t *testing.T) {
	plans := []struct {
		Seed       int64
		Iterations uint32
	}{
		{Seed: 686303773, Iterations: 128},
		{Seed: -2147483647, Iterations: 128},
	}
	prof := &testgen.ParserSyntaxProfile{}
	for _, p := range plans {
		p := p
		t.Run(fmt.Sprintf("seed_%d", p.Seed), func(t *testing.T) {
			testgen.RunIterations(t, p.Seed, p.Iterations, prof, func(t *testing.T, _ testgen.Meta, c testgen.Case) {
				// parse → render(text) → parse → render: idempotence
				elems, err := parser.Parse(c.Template)
				if err != nil {
					t.Fatalf("Parse generated template: %v\n%s", err, c.Template)
				}
				out, err := renderElementsText(elems)
				if err != nil {
					t.Fatalf("renderElementsText: %v", err)
				}
				back, err := parser.Parse(out)
				if err != nil {
					t.Fatalf("Parse(rendered) failed: %v\nrendered=%q", err, out)
				}
				if diff := compareElements(elems, back); diff != "" {
					t.Fatalf("AST mismatch after round trip: %s", diff)
				}
				out2, err := renderElementsText(back)
				if err != nil {
					t.Fatalf("renderElementsText second: %v", err)
				}
				if out != out2 {
					t.Fatalf("rendering not idempotent\nfirst:  %q\nsecond: %q", out, out2)
				}
			})
		})
	}
}
