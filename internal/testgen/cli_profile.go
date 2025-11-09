package testgen

// CLISemanticsProfile generates small templates suitable for black‑box CLI
// contract tests (e.g., sync/diff path derivation, quiet mode). The template
// itself is simple; CLI tests control flags/paths outside of the generator.

import (
	"fmt"
	"math/rand"
	"strings"
)

type CLISemanticsProfile struct {
	varIndex    int
	secretIndex int
}

func (p *CLISemanticsProfile) nextVar() string {
	v := fmt.Sprintf("CLI_VAR_%d", p.varIndex)
	p.varIndex++
	return v
}

func (p *CLISemanticsProfile) nextSecret() (path, value string) {
	path = fmt.Sprintf("cli/%d", p.secretIndex)
	samples := []string{"alpha", "bravo", "charlie"}
	value = samples[p.secretIndex%len(samples)]
	p.secretIndex++
	return
}

// Generate implements Profile. It prefers constructs that will render without
// special environment constraints so that CLI tests can run portably.
func (p *CLISemanticsProfile) Generate(r *rand.Rand, _ uint32) Case {
	var b strings.Builder
	resolver := map[string]string{}
	lines := 1 + r.Intn(2)
	for i := 0; i < lines; i++ {
		if i == 1 && r.Intn(2) == 0 {
			b.WriteString("# header comment\n")
		}
		name := p.nextVar()
		op := "="
		path, value := p.nextSecret()
		resolver[path] = value
		// keep to bare/double to avoid sandbox or bash nuances
		if r.Intn(2) == 0 {
			b.WriteString(name + op + "<pass:" + path + ">\n")
		} else {
			b.WriteString(name + op + "\"<pass:" + path + ">\"\n")
		}
	}
	return Case{Template: b.String(), Resolver: resolver}
}
