package parser_test

import (
	"math/rand"
	"testing"

	"envseed/internal/parser"
	"envseed/internal/testgen"
	"envseed/internal/testsupport"
)

// [EVT-MGF-1][EVT-MGF-4]
func FuzzParserASTStability(f *testing.F) {
	// Replay minimized failures from package-relative corpus.
	if seeds, err := testsupport.LoadCorpusSeeds("FuzzParserASTStability"); err == nil {
		for _, s := range seeds {
			f.Add(s.Seed, s.Iteration)
		}
	}
	// Hand-picked diverse seeds for baseline exploration.
	base := []struct {
		seed int64
		iter uint32
	}{
		{seed: 4242424242, iter: 0},
		{seed: -1123581321, iter: 15},
		{seed: 662607015, iter: 47},
	}
	for _, b := range base {
		f.Add(b.seed, b.iter)
	}

	prof := &testgen.ParserSyntaxProfile{}
	f.Fuzz(func(t *testing.T, seed int64, iteration uint32) {
		iterSeed := deriveIterationSeed(seed, iteration)
		_ = rand.New(rand.NewSource(iterSeed))
		// Minimal property: parse/render/parse idempotence for the last case in sequence
		var elems []Element
		testgen.RunIterations(t, iterSeed, iteration+1, prof, func(t *testing.T, _ testgen.Meta, c testgen.Case) {
			var err error
			elems, err = parser.Parse(c.Template)
			if err != nil {
				t.Fatalf("Parse error for generated template: %v\n%s", err, c.Template)
			}
		})
		out, err := renderElementsText(elems)
		if err != nil {
			t.Fatalf("renderElementsText: %v", err)
		}
		back, err := parser.Parse(out)
		if err != nil {
			t.Fatalf("Parse(rendered): %v\nrendered=%q", err, out)
		}
		if diff := compareElements(elems, back); diff != "" {
			t.Fatalf("AST mismatch: %s", diff)
		}
	})
}
