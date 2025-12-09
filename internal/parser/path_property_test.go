package parser_test

import (
	"testing"

	parser "envseed/internal/parser"
	"envseed/internal/testgen"
	"envseed/internal/testsupport"
)

// [EVT-MPF-6]
func TestParser_PlaceholderPathIRIProfile(t *testing.T) {
	t.Helper()

	prof := &testgen.PlaceholderPathProfile{}
	plans := []struct {
		seed  int64
		iters uint32
	}{
		{seed: 2023120301, iters: 64},
		{seed: -2023120301, iters: 64},
	}

	for _, p := range plans {
		p := p
		t.Run(testPathPlanName(p.seed), func(t *testing.T) {
			testgen.RunIterations(t, p.seed, p.iters, prof, func(t *testing.T, meta testgen.Meta, c testgen.Case) {
				elems, err := parser.Parse(c.Template)
				if c.Expect.ShouldErr {
					if err == nil {
						t.Fatalf("seed=%d iter=%d expected error but got success; template=%q", meta.Seed, meta.Iteration, c.Template)
					}
					perr := testsupport.ExpectErrorAs[*parser.ParseError](t, err, c.Expect.DetailCode, func(pe *parser.ParseError) string {
						return pe.DetailCode
					})
					if perr.DetailCode != c.Expect.DetailCode {
						t.Fatalf("seed=%d iter=%d detail code = %s, want %s", meta.Seed, meta.Iteration, perr.DetailCode, c.Expect.DetailCode)
					}
					return
				}
				if err != nil {
					t.Fatalf("seed=%d iter=%d unexpected parse error: %v; template=%q", meta.Seed, meta.Iteration, err, c.Template)
				}
				if len(elems) != 1 || elems[0].Assignment == nil {
					t.Fatalf("seed=%d iter=%d unexpected AST shape: %#v", meta.Seed, meta.Iteration, elems)
				}
			})
		})
	}
}

func testPathPlanName(seed int64) string {
	if seed >= 0 {
		return "seed_pos"
	}
	return "seed_neg"
}
