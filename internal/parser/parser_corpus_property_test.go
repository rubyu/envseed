package parser_test

import (
	"fmt"
	"path/filepath"
	"testing"

	"envseed/internal/parser"
	"envseed/internal/testgen"
	"envseed/internal/testsupport"
)

// [EVT-MGP-2]
func TestParser_CorpusReplay(t *testing.T) {
	t.Helper()
	const fuzzName = "FuzzParserASTStability"
	seeds, err := testsupport.LoadCorpusSeeds(fuzzName)
	if err != nil {
		t.Fatalf("load corpus %s: %v", fuzzName, err)
	}
	prof := &testgen.ParserSyntaxProfile{}
	for _, seed := range seeds {
		seed := seed
		label := seed.File
		pkgDir := filepath.Clean(filepath.Join("testdata", "fuzz", fuzzName))
		if dir := filepath.Clean(seed.Dir); dir != pkgDir {
			label = filepath.Join(filepath.Base(dir), seed.File)
		}
		t.Run(fmt.Sprintf("corpus/%s", label), func(t *testing.T) {
			iterSeed := deriveIterationSeed(seed.Seed, seed.Iteration)
			var elems []Element
			testgen.RunIterations(t, iterSeed, seed.Iteration+1, prof, func(t *testing.T, _ testgen.Meta, c testgen.Case) {
				var err error
				elems, err = parser.Parse(c.Template)
				if err != nil {
					t.Fatalf("Parse generated template: %v\n%s", err, c.Template)
				}
			})
			out, err := renderElementsText(elems)
			if err != nil {
				t.Fatalf("renderElementsText: %v", err)
			}
			back, err := parser.Parse(out)
			if err != nil {
				t.Fatalf("Parse(rendered) failed: %v\nrendered=%q", err, out)
			}
			if diff := compareElements(elems, back); diff != "" {
				t.Fatalf("AST mismatch after replay: %s", diff)
			}
		})
	}
}
