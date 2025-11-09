package testgen

import (
	"fmt"
	"math/rand"
	"strings"
)

// RendererRoundTripProfile generates small valid templates intended to pass
// rendering and round-trip parsing in the renderer suite. It avoids dangerous
// combinations by default so tests can exercise bash validation and sandbox.
type RendererRoundTripProfile struct {
	varIndex    int
	secretIndex int
}

func (p *RendererRoundTripProfile) nextVar() string {
	v := fmt.Sprintf("VAR%d", p.varIndex)
	p.varIndex++
	return v
}

func (p *RendererRoundTripProfile) nextSecret() (path, value string) {
	path = fmt.Sprintf("secret/%d", p.secretIndex)
	p.secretIndex++
	// keep values simple to allow bash -n success and reparse
	samples := []string{"alpha", "hello world", "hash#value", "tick$val", "paren)close"}
	value = samples[p.secretIndex%len(samples)]
	return
}

func pickOne[T any](r *rand.Rand, xs []T) T { return xs[r.Intn(len(xs))] }

type rtLine struct {
	text     string
	resolver map[string]string
	ctx      string
	mods     []string
	secret   string
}

func (p *RendererRoundTripProfile) buildLine(r *rand.Rand) rtLine {
	name := p.nextVar()
	op := "="
	if r.Intn(4) == 0 {
		op = "+="
	}
	path, value := p.nextSecret()
	resolver := map[string]string{path: value}
	// choose a context
	ctx := pickOne(r, []string{"bare", "double", "cmd", "backtick", "single"})
	var rhs string
	var mods []string
	switch ctx {
	case "bare":
		rhs = fmt.Sprintf("<pass:%s>", path)
	case "double":
		rhs = fmt.Sprintf("\"<pass:%s>\"", path)
	case "cmd":
		rhs = fmt.Sprintf("$(echo <pass:%s>)", path)
	case "backtick":
		rhs = fmt.Sprintf("`echo <pass:%s>`", path)
	case "single":
		// keep single quoted simple and rely on renderer rules to leave it literal
		mods = []string{"allow_tab"}
		rhs = fmt.Sprintf("'<pass:%s|allow_tab>'", path)
	}
	return rtLine{
		text:     strings.Join([]string{name, op, rhs}, ""),
		resolver: resolver,
		ctx:      ctx,
		mods:     mods,
		secret:   value,
	}
}

// Generate implements Profile.
func (p *RendererRoundTripProfile) Generate(r *rand.Rand, _ uint32) Case {
	lines := 1 + r.Intn(3)
	var b strings.Builder
	resolver := map[string]string{}
	var expect Expectation
	var skipBash, skipSandbox bool
	for i := 0; i < lines; i++ {
		switch r.Intn(5) {
		case 0:
			b.WriteString("\n")
		case 1:
			b.WriteString("# random comment\n")
		default:
			ln := p.buildLine(r)
			b.WriteString(ln.text)
			b.WriteString("\n")
			for k, v := range ln.resolver {
				resolver[k] = v
			}
			e, sbash, ssbox := evaluateExpectation(ln.ctx, ln.mods, ln.secret)
			// First failure wins
			if !expect.ShouldErr && e.ShouldErr {
				expect = e
			}
			skipBash = skipBash || sbash
			skipSandbox = skipSandbox || ssbox || ln.ctx == "backtick"
		}
	}
	return Case{Template: b.String(), Resolver: resolver, Expect: expect, SkipBash: skipBash, SkipSandbox: skipSandbox}
}

// evaluateExpectation estimates renderer outcomes for a single placeholder
// using spec rules reflected in internal/renderer (no imports to avoid cycles).
func evaluateExpectation(ctx string, mods []string, secret string) (Expectation, bool, bool) {
	// Return default (no error, no skips)
	out := Expectation{}
	skipBash := false
	skipSandbox := false

	has := func(mod string) bool {
		for _, m := range mods {
			if m == mod {
				return true
			}
		}
		return false
	}

	// dangerously_bypass_escape: single-only; when present with anything else → error.
	if has("dangerously_bypass_escape") {
		skipBash, skipSandbox = true, true
		if len(mods) > 1 {
			return Expectation{ShouldErr: true, DetailCode: "EVE-105-601", Phase: "render"}, skipBash, skipSandbox
		}
		return out, skipBash, skipSandbox
	}

	// Helpers to inspect content
	containsNewline := strings.ContainsAny(secret, "\n\r")
	containsTab := strings.Contains(secret, "\t")
	hasCtrl := false
	for _, r := range secret {
		if (r >= 0 && r < 0x20 && r != '\n' && r != '\t') || r == 0x7f {
			hasCtrl = true
			break
		}
	}

	switch ctx {
	case "single":
		if containsNewline {
			return Expectation{ShouldErr: true, DetailCode: "EVE-105-102", Phase: "render"}, skipBash, skipSandbox
		}
		if containsTab && !has("allow_tab") {
			return Expectation{ShouldErr: true, DetailCode: "EVE-105-103", Phase: "render"}, skipBash, skipSandbox
		}
		if hasCtrl {
			return Expectation{ShouldErr: true, DetailCode: "EVE-105-104", Phase: "render"}, skipBash, skipSandbox
		}
	case "bare":
		if has("allow_newline") {
			return Expectation{ShouldErr: true, DetailCode: "EVE-105-504", Phase: "render"}, skipBash, skipSandbox
		}
		if containsNewline {
			return Expectation{ShouldErr: true, DetailCode: "EVE-105-501", Phase: "render"}, skipBash, skipSandbox
		}
		if containsTab && !has("allow_tab") {
			return Expectation{ShouldErr: true, DetailCode: "EVE-105-502", Phase: "render"}, skipBash, skipSandbox
		}
		if hasCtrl {
			return Expectation{ShouldErr: true, DetailCode: "EVE-105-503", Phase: "render"}, skipBash, skipSandbox
		}
	case "double":
		if containsNewline && !has("allow_newline") {
			return Expectation{ShouldErr: true, DetailCode: "EVE-105-201", Phase: "render"}, skipBash, skipSandbox
		}
		if containsTab && !has("allow_tab") {
			return Expectation{ShouldErr: true, DetailCode: "EVE-105-202", Phase: "render"}, skipBash, skipSandbox
		}
		if hasCtrl {
			return Expectation{ShouldErr: true, DetailCode: "EVE-105-203", Phase: "render"}, skipBash, skipSandbox
		}
	case "cmd":
		if containsNewline && !has("allow_newline") {
			return Expectation{ShouldErr: true, DetailCode: "EVE-105-301", Phase: "render"}, skipBash, skipSandbox
		}
		if containsTab && !has("allow_tab") {
			return Expectation{ShouldErr: true, DetailCode: "EVE-105-302", Phase: "render"}, skipBash, skipSandbox
		}
		if hasCtrl {
			return Expectation{ShouldErr: true, DetailCode: "EVE-105-303", Phase: "render"}, skipBash, skipSandbox
		}
	case "backtick":
		// For sandbox, never execute backticks
		skipSandbox = true
		if has("allow_newline") {
			return Expectation{ShouldErr: true, DetailCode: "EVE-105-404", Phase: "render"}, skipBash, skipSandbox
		}
		if containsNewline {
			return Expectation{ShouldErr: true, DetailCode: "EVE-105-401", Phase: "render"}, skipBash, skipSandbox
		}
		if containsTab && !has("allow_tab") {
			return Expectation{ShouldErr: true, DetailCode: "EVE-105-402", Phase: "render"}, skipBash, skipSandbox
		}
		if hasCtrl {
			return Expectation{ShouldErr: true, DetailCode: "EVE-105-403", Phase: "render"}, skipBash, skipSandbox
		}
	}
	return out, skipBash, skipSandbox
}
