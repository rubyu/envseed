package testgen

import (
	"math/rand"
	"testing"
)

// Expectation describes whether a generated case should error and how.
type Expectation struct {
	ShouldErr  bool
	DetailCode string
	Phase      string // "parse" or "render" (open set for future use)
}

// Case represents one generated test instance.
type Case struct {
	Template    string
	Resolver    map[string]string
	Expect      Expectation
	SkipBash    bool
	SkipSandbox bool
}

// Meta carries seed/iteration used to generate a Case.
type Meta struct {
	Seed      int64
	Iteration uint32
}

// Profile generates cases for a given domain (parser/renderer/cli).
type Profile interface {
	Generate(r *rand.Rand, iteration uint32) Case
}

// IterationFunc is invoked for each generated case.
type IterationFunc func(t *testing.T, meta Meta, c Case)

// RunIterations drives a profile across iterations and invokes fn for each.
func RunIterations(t *testing.T, seed int64, iterations uint32, p Profile, fn IterationFunc) {
	t.Helper()
	r := rand.New(rand.NewSource(seed))
	for i := uint32(0); i < iterations; i++ {
		c := p.Generate(r, i)
		fn(t, Meta{Seed: seed, Iteration: i}, c)
	}
}
