package testgen

// RendererEnhancedProfile generates templates that purposefully exercise a
// broader set of contexts and modifier combinations than the round‑trip
// profile, including newlines/tabs and base64. It stays mostly within
// renderer's success domain so downstream tests can validate bash and sandbox
// when appropriate. Expectation metadata is attached to capture known
// invalid combinations per specification (e.g., TAB/newline without allow_*,
// dangerously_bypass_escape with any other modifier).

import (
	"encoding/base64"
	"fmt"
	"math/rand"
	"strings"
)

// RendererEnhancedProfile controls generation of lines and secrets.
type RendererEnhancedProfile struct {
	varIndex    int
	secretIndex int
}

func (p *RendererEnhancedProfile) nextVar() string {
	v := fmt.Sprintf("VAR%d", p.varIndex)
	p.varIndex++
	return v
}

func (p *RendererEnhancedProfile) nextSecret() (path, value string) {
	path = fmt.Sprintf("enhanced/%d", p.secretIndex)
	// Alternate several values to cover escapes and whitespace.
	samples := []string{
		"alpha",
		"hello world",
		"line1\nline2",
		"tab\tvalue",
		"$dollar#hash(paren)\\backslash`tick`",
	}
	value = samples[p.secretIndex%len(samples)]
	p.secretIndex++
	return
}

// Generate implements Profile.
func (p *RendererEnhancedProfile) Generate(r *rand.Rand, _ uint32) Case {
	lines := 1 + r.Intn(3)
	var b strings.Builder
	resolver := map[string]string{}
	var expect Expectation
	var skipBash, skipSandbox bool
	for i := 0; i < lines; i++ {
		// occasional comment/blank
		if i > 0 && r.Intn(5) == 0 {
			b.WriteString("# comment\n")
		}
		if i > 0 && r.Intn(5) == 0 {
			b.WriteString("\n")
		}
		name := p.nextVar()
		op := "="
		if r.Intn(4) == 0 {
			op = "+="
		}
		path, secret := p.nextSecret()
		resolver[path] = secret
		// choose context + modifiers
		type combo struct {
			ctx  string
			mods []string
		}
		choices := []combo{
			{ctx: "bare", mods: nil},
			{ctx: "bare", mods: []string{"allow_tab"}},
			{ctx: "double", mods: []string{"allow_newline"}},
			{ctx: "double", mods: nil},
			{ctx: "cmd", mods: nil},
			{ctx: "backtick", mods: nil},
			{ctx: "single", mods: []string{"allow_tab"}},
		}
		c := choices[r.Intn(len(choices))]
		// occasionally combine with base64
		if r.Intn(4) == 0 {
			c.mods = append(c.mods, "base64")
			secret = base64.StdEncoding.EncodeToString([]byte(secret))
			resolver[path] = secret
		}
		// rarely include dangerously_bypass_escape to exercise parser/modifier axes
		if r.Intn(10) == 0 {
			c.mods = append(c.mods, "dangerously_bypass_escape")
		}
		// prohibit obviously dangerous combos; leave exploration to fuzz
		// format placeholder
		var ph strings.Builder
		ph.WriteString("<pass:")
		ph.WriteString(path)
		if len(c.mods) > 0 {
			ph.WriteString("|")
			ph.WriteString(strings.Join(c.mods, ","))
		}
		ph.WriteString(">")
		var rhs string
		switch c.ctx {
		case "bare":
			rhs = ph.String()
		case "double":
			rhs = fmt.Sprintf("\"%s\"", ph.String())
		case "cmd":
			rhs = fmt.Sprintf("$(echo %s)", ph.String())
		case "backtick":
			rhs = fmt.Sprintf("`echo %s`", ph.String())
		case "single":
			rhs = fmt.Sprintf("'%s'", ph.String())
		}
		b.WriteString(name)
		b.WriteString(op)
		b.WriteString(rhs)
		b.WriteString("\n")
		// Attach expectation for this line and accumulate first failure.
		e, sbash, ssbox := evaluateExpectation(c.ctx, c.mods, secret)
		if !expect.ShouldErr && e.ShouldErr {
			expect = e
		}
		skipBash = skipBash || sbash
		skipSandbox = skipSandbox || ssbox || c.ctx == "backtick"
	}
	return Case{Template: b.String(), Resolver: resolver, Expect: expect, SkipBash: skipBash, SkipSandbox: skipSandbox}
}
