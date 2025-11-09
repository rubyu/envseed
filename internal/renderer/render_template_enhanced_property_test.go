// Test overview:
//  1. Enhanced plans and corpus cases reuse property fuzz infrastructure but swap in buildEnhancedTemplate to emphasize edge scenarios.
//  2. buildEnhancedTemplate stitches together line generators that purposefully mix modifier matrices, quoting contexts, and error cases.
//  3. enhancedIteration mirrors round-trip validation: render, expect-or-accept errors, parse, bash validate, and optional sandbox execution.
//  4. TestPlaceholderModifierMatrix supplements randomness with deterministic checks to ensure each modifier rule is exercised explicitly.
package renderer_test

import (
	"encoding/base64"
	"errors"
	"fmt"
	"math/rand"
	"path/filepath"
	"strings"
	"testing"

	"envseed/internal/parser"
	renderer "envseed/internal/renderer"
	sbox "envseed/internal/sandbox"
	"envseed/internal/testsupport"
)

// Local helpers mirroring renderer strip behavior for external tests
func applyStrip(secret string, mods map[string]bool) string {
	if !(mods["strip"] || mods["strip_left"] || mods["strip_right"]) {
		return secret
	}
	isWS := func(r rune) bool { return r == ' ' || r == '\t' || r == '\n' || r == '\r' }
	if mods["strip"] || mods["strip_left"] {
		secret = strings.TrimLeftFunc(secret, isWS)
	}
	if mods["strip"] || mods["strip_right"] {
		secret = strings.TrimRightFunc(secret, isWS)
	}
	return secret
}

type enhancedMode int

const (
	modeEnhancedTemplates enhancedMode = iota
	modeGrammarFocus
)

var enhancedTemplatePlans = []propertyLoopPlan{
	{Seed: 982451653, Iterations: 192},
	{Seed: -271828183, Iterations: 192},
}

var enhancedGrammarPlans = []propertyLoopPlan{
	{Seed: 577215664, Iterations: 160},
	{Seed: -141592653, Iterations: 160},
}

// [EVT-MEP-1][EVT-MEP-3]
func TestRenderTemplateEnhanced(t *testing.T) {
	sandbox := newSandboxTracker(t)
	runEnhancedPlans(t, "enhanced/property", enhancedTemplatePlans, modeEnhancedTemplates, sandbox)
	runEnhancedCorpusCases(t, "FuzzRenderTemplateEnhanced", modeEnhancedTemplates, sandbox)
}

// [EVT-MGF-3]
func TestRenderGrammarFuzzedTemplates(t *testing.T) {
	sandbox := newSandboxTracker(t)
	runEnhancedPlans(t, "enhanced/grammar", enhancedGrammarPlans, modeGrammarFocus, sandbox)
	runEnhancedCorpusCases(t, "FuzzRenderTemplateEnhanced", modeGrammarFocus, sandbox)
}

// Fuzz moved to render_template_enhanced_fuzz_test.go

// [EVT-MUP-1][EVT-MUP-2][EVT-MUP-3]
func TestPlaceholderModifierMatrix(t *testing.T) {
	type checkFn func(*testing.T, string)

	requireContains := func(substr string) checkFn {
		return func(t *testing.T, rendered string) {
			t.Helper()
			if !strings.Contains(rendered, substr) {
				t.Fatalf("rendered output %q does not contain %q", rendered, substr)
			}
		}
	}

	checkBase64Chars := func(chars ...string) checkFn {
		return func(t *testing.T, rendered string) {
			t.Helper()
			for _, ch := range chars {
				if !strings.Contains(rendered, ch) {
					t.Fatalf("base64 output %q missing %q", rendered, ch)
				}
			}
		}
	}

	cases := []struct {
		name       string
		template   string
		resolver   externalResolver
		expectErr  bool
		checks     []checkFn
		expectCode string
	}{
		{
			name:     "BareBase64AllowsSpecialCharacters",
			template: "TOKEN=<pass:secret|base64>\n",
			resolver: externalResolver{
				"secret": string([]byte{0xfb, 0xfe}),
			},
			checks: []checkFn{
				checkBase64Chars("+", "/", "="),
			},
		},
		{
			name:       "BareBase64InvalidComboAllowTab",
			template:   "TOKEN=<pass:secret|base64,allow_tab>\n",
			resolver:   externalResolver{"secret": string([]byte{0xfb, 0xfe})},
			expectErr:  true,
			expectCode: "EVE-105-601",
		},
		{
			name:     "BareTabWithModifier",
			template: "VALUE=<pass:secret|allow_tab>\n",
			resolver: externalResolver{"secret": "tab\tvalue"},
			checks: []checkFn{
				requireContains("VALUE=tab\\\tvalue"),
			},
		},
		{
			name:       "BareTabWithoutModifierFails",
			template:   "VALUE=<pass:secret>\n",
			resolver:   externalResolver{"secret": "tab\tvalue"},
			expectErr:  true,
			expectCode: "EVE-105-502",
		},
		{
			name:     "BareIndexAssignmentWithBase64",
			template: "ARRAY[1]+=<pass:secret|base64>\n",
			resolver: externalResolver{"secret": string([]byte{0xfb})},
			checks: []checkFn{
				checkBase64Chars("+", "="),
			},
		},
		{
			name:     "DoubleQuotedNewlineAllowed",
			template: "MESSAGE=\"prefix <pass:secret|allow_newline> suffix\"\n",
			resolver: externalResolver{"secret": "line1\nline2"},
			checks: []checkFn{
				requireContains("line1\nline2"),
			},
		},
		{
			name:       "DoubleQuotedNewlineMissingModifier",
			template:   "MESSAGE=\"<pass:secret>\"\n",
			resolver:   externalResolver{"secret": "line1\nline2"},
			expectErr:  true,
			expectCode: "EVE-105-201",
		},
		{
			name:     "CommandSubstitutionAllowTab",
			template: "SCRIPT=$(printf %s <pass:secret|allow_tab>)\n",
			resolver: externalResolver{"secret": "tab\tcmd"},
			checks: []checkFn{
				requireContains("tab\tcmd"),
			},
		},
		{
			name:       "CommandSubstitutionMissingModifier",
			template:   "SCRIPT=$(printf %s <pass:secret>)\n",
			resolver:   externalResolver{"secret": "line1\nline2"},
			expectErr:  true,
			expectCode: "EVE-105-301",
		},
		{
			name:       "BacktickNewlineDisallowed",
			template:   "LEGACY=`echo <pass:secret|allow_newline>`\n",
			resolver:   externalResolver{"secret": "line1\nline2"},
			expectErr:  true,
			expectCode: "EVE-105-404",
		},
		{
			name:       "DangerouslyCannotCombineModifiers",
			template:   "RAW=<pass:secret|dangerously_bypass_escape,allow_tab>\n",
			resolver:   externalResolver{"secret": "raw"},
			expectErr:  true,
			expectCode: "EVE-105-601",
		},
		{
			name:     "DoubleQuotedBase64",
			template: "TOKEN=\"value:<pass:secret|base64>\"\n",
			resolver: externalResolver{"secret": "abc"},
			checks: []checkFn{
				requireContains("TOKEN=\"value:YWJj\""),
			},
		},
		{
			name:       "DoubleQuotedBase64InvalidCombo",
			template:   "TOKEN=\"<pass:secret|base64,allow_tab,allow_newline>\"\n",
			resolver:   externalResolver{"secret": "value"},
			expectErr:  true,
			expectCode: "EVE-105-601",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			rendered, err := renderer.RenderString(tc.template, tc.resolver)
			if tc.expectErr {
				if err == nil {
					t.Fatalf("expected error but got success: rendered=%q", rendered)
				}
				if tc.expectCode != "" {
					if perr, ok := err.(*renderer.PlaceholderError); ok {
						if perr.DetailCode() != tc.expectCode {
							t.Fatalf("detail code = %s (msg=%q), want %s", perr.DetailCode(), err.Error(), tc.expectCode)
						}
					} else {
						t.Fatalf("expected *PlaceholderError for code check, got %T (%v)", err, err)
					}
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			for _, check := range tc.checks {
				check(t, rendered)
			}
			if _, err := parser.Parse(rendered); err != nil {
				t.Fatalf("re-parse failed: %v\nrendered=%q", err, rendered)
			}
		})
	}
}

func runEnhancedPlans(t *testing.T, label string, plans []propertyLoopPlan, mode enhancedMode, sandbox *sandboxTracker) {
	t.Helper()
	for idx, plan := range plans {
		plan := plan
		name := fmt.Sprintf("%s/seed_%d", label, plan.Seed)
		if len(plans) > 1 {
			name = fmt.Sprintf("%s/%02d_seed_%d", label, idx, plan.Seed)
		}
		t.Run(name, func(t *testing.T) {
			runEnhancedTemplateIterations(t, plan.Seed, plan.Iterations, mode, sandbox)
		})
	}
}

func runEnhancedCorpusCases(t *testing.T, fuzzName string, mode enhancedMode, sandbox *sandboxTracker) {
	t.Helper()
	seeds, err := testsupport.LoadCorpusSeeds(fuzzName)
	if err != nil {
		t.Fatalf("load corpus %s: %v", fuzzName, err)
	}
	for _, seed := range seeds {
		seed := seed
		label := seed.File
		pkgDir := filepath.Clean(filepath.Join("testdata", "fuzz", fuzzName))
		if dir := filepath.Clean(seed.Dir); dir != pkgDir {
			label = filepath.Join(filepath.Base(dir), seed.File)
		}
		t.Run(fmt.Sprintf("corpus/%s", label), func(t *testing.T) {
			iterations := seed.Iteration + 1
			if iterations == 0 {
				t.Fatalf("iteration overflow for seed %d in %s", seed.Seed, seed.File)
			}
			runEnhancedTemplateIterations(t, seed.Seed, iterations, mode, sandbox)
		})
	}
}

func runEnhancedTemplateIterations(t *testing.T, seed int64, iterations uint32, mode enhancedMode, sandbox *sandboxTracker) {
	t.Helper()
	rnd := rand.New(rand.NewSource(seed))
	state := &enhancedState{}
	for i := uint32(0); i < iterations; i++ {
		tpl := buildEnhancedTemplate(rnd, state, mode)
		meta := propertyMeta{Seed: seed, Iteration: i}
		enhancedIteration(t, sandbox, meta, tpl)
	}
}

func generateEnhancedCase(t *testing.T, seed int64, iteration uint32, mode enhancedMode) (propertyMeta, propertyTemplate) {
	t.Helper()
	rnd := rand.New(rand.NewSource(seed))
	state := &enhancedState{}
	var tpl propertyTemplate
	for i := uint32(0); i <= iteration; i++ {
		tpl = buildEnhancedTemplate(rnd, state, mode)
	}
	return propertyMeta{Seed: seed, Iteration: iteration}, tpl
}

func enhancedIteration(t *testing.T, sandbox *sandboxTracker, meta propertyMeta, tpl propertyTemplate) {
	t.Helper()
	rendered, err := renderer.RenderString(tpl.Template, tpl.Resolver)
	if checkTemplateExpectation(t, meta, tpl, rendered, err) {
		return
	}
	if tpl.SkipBash || strings.Contains(tpl.Template, "dangerously_bypass_escape") {
		return
	}
	if _, err := parser.Parse(rendered); err != nil {
		t.Fatalf("seed=%d iteration=%d reparse error: %v\nrendered=%q", meta.Seed, meta.Iteration, err, rendered)
	}
	if err := testsupport.BashValidate(rendered); err != nil {
		t.Fatalf("seed=%d iteration=%d bash validation failed: %v\nrendered=%q", meta.Seed, meta.Iteration, err, rendered)
	}
	if sandbox != nil && sandbox.shouldRun(meta, tpl, rendered) {
		var script strings.Builder
		script.WriteString("set -eo pipefail\n")
		script.WriteString("set -a\n")
		script.WriteString(rendered)
		script.WriteString("\nset +a\n")
		if _, err := sandboxRun(script.String()); err != nil {
			if errors.Is(err, sbox.ErrUnsupported) {
				sandbox.supported = false
				t.Logf("sandbox became unsupported: %v", err)
				return
			}
			t.Fatalf("seed=%d iteration=%d sandbox execution failed: %v\nrendered=%q", meta.Seed, meta.Iteration, err, rendered)
		}
	}
}

type enhancedState struct {
	varIndex    int
	secretIndex int
}

type enhancedLine struct {
	text        string
	resolver    externalResolver
	expectErr   bool
	skipBash    bool
	skipSandbox bool
}

func buildEnhancedTemplate(r *rand.Rand, state *enhancedState, mode enhancedMode) propertyTemplate {
	var b strings.Builder
	resolver := make(externalResolver)
	var expect propertyExpectation
	skipBash := false
	skipSandbox := false

	lines := 1 + r.Intn(4)
	for i := 0; i < lines; i++ {
		line := buildEnhancedLine(r, state, mode)
		b.WriteString(line.text)
		for k, v := range line.resolver {
			resolver[k] = v
		}
		skipBash = skipBash || line.skipBash
		skipSandbox = skipSandbox || line.skipSandbox
	}

	template := b.String()
	if strings.Contains(template, "dangerously_bypass_escape") {
		skipBash = true
		skipSandbox = true
	}

	if elems, err := parser.Parse(template); err != nil {
		expect = propertyExpectation{ShouldErr: true, Phase: failurePhaseParse}
		var perr *parser.ParseError
		if errors.As(err, &perr) {
			expect.DetailCode = perr.DetailCode
		}
	} else {
	outer:
		for _, elem := range elems {
			if elem.Type != renderer.ElementAssignment || elem.Assignment == nil {
				continue
			}
			for _, tok := range elem.Assignment.ValueTokens {
				if tok.Kind != renderer.ValuePlaceholder {
					continue
				}
				modsSet := renderer.ExportModifierSet(tok.Modifiers)
				// Detect invalid modifier combinations consistent with renderer rules
				if modsSet["base64"] {
					count := 0
					for range modsSet {
						count++
					}
					if count > 1 {
						expect = propertyExpectation{ShouldErr: true, Phase: failurePhaseRender, DetailCode: "EVE-105-601"}
						break outer
					}
				}
				if modsSet["dangerously_bypass_escape"] {
					continue
				}
				secret, ok := resolver[tok.Path]
				if !ok {
					continue
				}
				secret = applyStrip(secret, modsSet)
				if modsSet["base64"] {
					secret = base64.StdEncoding.EncodeToString([]byte(secret))
				}
				var ctxErr error
				switch tok.Context {
				case renderer.ContextSingleQuoted:
					_, ctxErr = renderer.ExportRenderSingleQuoted(secret, modsSet, tok.Line, tok.Column, tok.Path)
				case renderer.ContextBare:
					_, ctxErr = renderer.ExportRenderBare(secret, modsSet, tok.Line, tok.Column, tok.Path)
				case renderer.ContextDoubleQuoted:
					_, ctxErr = renderer.ExportRenderDoubleQuoted(secret, modsSet, tok.Line, tok.Column, tok.Path)
				case renderer.ContextCommandSubstitution:
					_, ctxErr = renderer.ExportRenderCommandSubst(secret, modsSet, tok.Line, tok.Column, tok.Path)
				case renderer.ContextBacktick:
					_, ctxErr = renderer.ExportRenderBacktick(secret, modsSet, tok.Line, tok.Column, tok.Path)
				}
				if ctxErr != nil {
					expect = propertyExpectation{ShouldErr: true, Phase: failurePhaseRender}
					var placeholderErr *renderer.PlaceholderError
					if errors.As(ctxErr, &placeholderErr) {
						expect.DetailCode = placeholderErr.DetailCode()
					}
					break outer
				}
			}
		}
	}

	// Expectation is determined purely by static parse/context checks above.

	return propertyTemplate{
		Template:    template,
		Resolver:    resolver,
		Expect:      expect,
		SkipBash:    skipBash,
		SkipSandbox: skipSandbox,
	}
}

func buildEnhancedLine(r *rand.Rand, state *enhancedState, mode enhancedMode) enhancedLine {
	generators := enhancedLineGenerators
	if mode == modeGrammarFocus {
		generators = enhancedGrammarGenerators
	}
	gen := generators[r.Intn(len(generators))]
	return gen(r, state, mode)
}

type enhancedLineGenerator func(*rand.Rand, *enhancedState, enhancedMode) enhancedLine

var enhancedLineGenerators = []enhancedLineGenerator{
	lineEnhancedBase64Bare,
	lineEnhancedBase64InvalidCombo,
	lineEnhancedBareMissingModifier,
	lineEnhancedSingleQuoted,
	lineEnhancedSingleQuotedInvalid,
	lineEnhancedDoubleQuoted,
	lineEnhancedDoubleQuotedMissingModifier,
	lineEnhancedCommandSubstitution,
	lineEnhancedCommandMissingModifier,
	lineEnhancedBacktickInvalid,
	lineEnhancedDangerouslyBypass,
	lineEnhancedDanglingPlaceholder,
	lineEnhancedWhitespaceVariant,
}

var enhancedGrammarGenerators = []enhancedLineGenerator{
	lineEnhancedBase64Bare,
	lineEnhancedBareMissingModifier,
	lineEnhancedSingleQuoted,
	lineEnhancedDoubleQuoted,
	lineEnhancedCommandSubstitution,
	lineEnhancedComment,
	lineEnhancedBlank,
	lineEnhancedDangerouslyBypass,
	lineEnhancedDoubleQuotedMissingModifier,
	lineEnhancedCommandMissingModifier,
	lineEnhancedWhitespaceVariant,
	lineEnhancedSingleQuotedInvalid,
	lineEnhancedDanglingPlaceholder,
}

func lineEnhancedBase64Bare(r *rand.Rand, state *enhancedState, mode enhancedMode) enhancedLine {
	var b strings.Builder
	b.WriteString(strings.Repeat(" ", r.Intn(3)))
	name := nextEnhancedVarName(state)
	head := buildEnhancedAssignmentHead(r, name, mode == modeGrammarFocus || r.Intn(2) == 0)
	b.WriteString(head)
	path := nextEnhancedSecretPath(state)
	secret := enhancedBase64Secret(r)
	b.WriteString("<pass:")
	b.WriteString(path)
	b.WriteString("|base64>")
	if mode == modeGrammarFocus && r.Intn(2) == 0 {
		b.WriteString(" # base64 bare context")
	}
	b.WriteString("\n")
	return enhancedLine{
		text:        b.String(),
		resolver:    externalResolver{path: secret},
		expectErr:   false,
		skipSandbox: true,
	}
}

func lineEnhancedBase64InvalidCombo(r *rand.Rand, state *enhancedState, mode enhancedMode) enhancedLine {
	var b strings.Builder
	b.WriteString(strings.Repeat(" ", r.Intn(2)))
	name := nextEnhancedVarName(state)
	head := buildEnhancedAssignmentHead(r, name, mode == modeGrammarFocus || r.Intn(2) == 0)
	b.WriteString(head)
	path := nextEnhancedSecretPath(state)
	b.WriteString("<pass:")
	b.WriteString(path)
	b.WriteString("|base64,allow_tab>")
	b.WriteString("\n")
	// Invalid combination: base64 MUST be single-only (no other modifiers).
	return enhancedLine{
		text:      b.String(),
		resolver:  externalResolver{path: "x"},
		expectErr: true,
	}
}

func lineEnhancedBareMissingModifier(r *rand.Rand, state *enhancedState, mode enhancedMode) enhancedLine {
	var b strings.Builder
	b.WriteString(strings.Repeat(" ", r.Intn(3)))
	name := nextEnhancedVarName(state)
	head := buildEnhancedAssignmentHead(r, name, mode == modeGrammarFocus || r.Intn(2) == 0)
	b.WriteString(head)
	path := nextEnhancedSecretPath(state)
	secret := "value with space"
	expectErr := false
	if r.Intn(2) == 0 {
		secret = "line1\nline2"
		expectErr = true
	}
	b.WriteString(buildEnhancedPlaceholder(r, path, nil))
	b.WriteString("\n")
	return enhancedLine{
		text:      b.String(),
		resolver:  externalResolver{path: secret},
		expectErr: expectErr,
	}
}

func lineEnhancedSingleQuoted(r *rand.Rand, state *enhancedState, mode enhancedMode) enhancedLine {
	var b strings.Builder
	b.WriteString(strings.Repeat(" ", r.Intn(3)))
	name := nextEnhancedVarName(state)
	head := buildEnhancedAssignmentHead(r, name, true)
	b.WriteString(head)
	path := nextEnhancedSecretPath(state)
	mods, secret := pickSingleQuotedValid(r)
	b.WriteString("'prefix ")
	b.WriteString(buildEnhancedPlaceholder(r, path, mods))
	if mode == modeGrammarFocus && r.Intn(2) == 0 {
		b.WriteString(" suffix")
	}
	b.WriteString("'")
	if mode == modeGrammarFocus && r.Intn(3) == 0 {
		b.WriteString(" # single quoted ok")
	}
	b.WriteString("\n")
	return enhancedLine{
		text:     b.String(),
		resolver: externalResolver{path: secret},
	}
}

func lineEnhancedSingleQuotedInvalid(r *rand.Rand, state *enhancedState, mode enhancedMode) enhancedLine {
	var b strings.Builder
	b.WriteString(strings.Repeat(" ", r.Intn(2)))
	name := nextEnhancedVarName(state)
	head := buildEnhancedAssignmentHead(r, name, mode == modeGrammarFocus || r.Intn(2) == 0)
	b.WriteString(head)
	path := nextEnhancedSecretPath(state)
	mods, secret := pickSingleQuotedInvalid(r)
	b.WriteString("'value ")
	b.WriteString(buildEnhancedPlaceholder(r, path, mods))
	b.WriteString("'")
	if mode == modeGrammarFocus && r.Intn(2) == 0 {
		b.WriteString(" # single quoted invalid")
	}
	b.WriteString("\n")
	return enhancedLine{
		text:      b.String(),
		resolver:  externalResolver{path: secret},
		expectErr: true,
	}
}

func lineEnhancedDoubleQuoted(r *rand.Rand, state *enhancedState, mode enhancedMode) enhancedLine {
	var b strings.Builder
	b.WriteString(strings.Repeat(" ", r.Intn(3)))
	name := nextEnhancedVarName(state)
	head := buildEnhancedAssignmentHead(r, name, true)
	b.WriteString(head)
	path := nextEnhancedSecretPath(state)
	modifiers, secret := pickDoubleQuotedSecret(r)
	b.WriteString(`"prefix `)
	b.WriteString(buildEnhancedPlaceholder(r, path, modifiers))
	if mode == modeGrammarFocus && r.Intn(2) == 0 {
		b.WriteString(` suffix $(printf %s sample)`)
	} else {
		b.WriteString(` suffix`)
	}
	b.WriteString(`"`)
	if mode == modeGrammarFocus && r.Intn(3) == 0 {
		b.WriteString(" # dq with modifiers")
	}
	b.WriteString("\n")
	return enhancedLine{
		text:        b.String(),
		resolver:    externalResolver{path: secret},
		skipSandbox: true,
	}
}

func lineEnhancedDoubleQuotedMissingModifier(r *rand.Rand, state *enhancedState, mode enhancedMode) enhancedLine {
	var b strings.Builder
	b.WriteString(strings.Repeat(" ", r.Intn(2)))
	name := nextEnhancedVarName(state)
	head := buildEnhancedAssignmentHead(r, name, mode == modeGrammarFocus || r.Intn(2) == 0)
	b.WriteString(head)
	path := nextEnhancedSecretPath(state)
	secret := "line1\nline2"
	b.WriteString(`"data `)
	b.WriteString(buildEnhancedPlaceholder(r, path, nil))
	b.WriteString(` tail"`)
	b.WriteString("\n")
	return enhancedLine{
		text:      b.String(),
		resolver:  externalResolver{path: secret},
		expectErr: true,
	}
}

func lineEnhancedCommandSubstitution(r *rand.Rand, state *enhancedState, mode enhancedMode) enhancedLine {
	var b strings.Builder
	b.WriteString(strings.Repeat(" ", r.Intn(3)))
	name := nextEnhancedVarName(state)
	head := buildEnhancedAssignmentHead(r, name, true)
	b.WriteString(head)
	path := nextEnhancedSecretPath(state)
	modifiers, secret := pickCommandSecret(r)
	resolver := externalResolver{path: secret}
	b.WriteString("$(printf %s ")
	if r.Intn(2) == 0 {
		b.WriteString(`"`)
		b.WriteString(buildEnhancedPlaceholder(r, path, modifiers))
		b.WriteString(`"`)
	} else {
		b.WriteString(buildEnhancedPlaceholder(r, path, modifiers))
	}
	if mode == modeGrammarFocus && r.Intn(3) == 0 {
		nestedPath := nextEnhancedSecretPath(state)
		b.WriteString(" && printf %s ")
		b.WriteString(buildEnhancedPlaceholder(r, nestedPath, []string{"allow_newline"}))
		resolver[nestedPath] = "multi line\ncontent"
	}
	b.WriteString(")")
	if mode == modeGrammarFocus && r.Intn(3) == 0 {
		b.WriteString(" # command substitution")
	}
	b.WriteString("\n")
	return enhancedLine{
		text:        b.String(),
		resolver:    resolver,
		skipSandbox: true,
	}
}

func lineEnhancedCommandMissingModifier(r *rand.Rand, state *enhancedState, mode enhancedMode) enhancedLine {
	var b strings.Builder
	b.WriteString(strings.Repeat(" ", r.Intn(2)))
	name := nextEnhancedVarName(state)
	head := buildEnhancedAssignmentHead(r, name, mode == modeGrammarFocus || r.Intn(2) == 0)
	b.WriteString(head)
	path := nextEnhancedSecretPath(state)
	secret := "line1\nline2"
	b.WriteString("$(echo ")
	b.WriteString(buildEnhancedPlaceholder(r, path, nil))
	b.WriteString(")")
	b.WriteString("\n")
	return enhancedLine{
		text:      b.String(),
		resolver:  externalResolver{path: secret},
		expectErr: true,
	}
}

func lineEnhancedBacktickInvalid(r *rand.Rand, state *enhancedState, mode enhancedMode) enhancedLine {
	var b strings.Builder
	b.WriteString(strings.Repeat(" ", r.Intn(2)))
	name := nextEnhancedVarName(state)
	head := buildEnhancedAssignmentHead(r, name, mode == modeGrammarFocus || r.Intn(2) == 0)
	b.WriteString(head)
	path := nextEnhancedSecretPath(state)
	secret := "line1\nline2"
	b.WriteString("`echo ")
	b.WriteString(buildEnhancedPlaceholder(r, path, []string{"allow_newline"}))
	b.WriteString("`")
	b.WriteString("\n")
	return enhancedLine{
		text:        b.String(),
		resolver:    externalResolver{path: secret},
		expectErr:   true,
		skipSandbox: true,
	}
}

func lineEnhancedDangerouslyBypass(r *rand.Rand, state *enhancedState, mode enhancedMode) enhancedLine {
	var b strings.Builder
	if mode == modeGrammarFocus {
		b.WriteString(strings.Repeat("\t", r.Intn(2)))
	} else {
		b.WriteString(strings.Repeat(" ", r.Intn(2)))
	}
	name := nextEnhancedVarName(state)
	head := buildEnhancedAssignmentHead(r, name, true)
	b.WriteString(head)
	path := nextEnhancedSecretPath(state)
	secret := "rm -rf /tmp/should_not_run"
	b.WriteString(buildEnhancedPlaceholder(r, path, []string{"dangerously_bypass_escape"}))
	b.WriteString("\n")
	return enhancedLine{
		text:        b.String(),
		resolver:    externalResolver{path: secret},
		skipBash:    true,
		skipSandbox: true,
	}
}

func lineEnhancedDanglingPlaceholder(r *rand.Rand, state *enhancedState, mode enhancedMode) enhancedLine {
	var b strings.Builder
	b.WriteString(strings.Repeat(" ", r.Intn(2)))
	name := nextEnhancedVarName(state)
	head := buildEnhancedAssignmentHead(r, name, mode == modeGrammarFocus || r.Intn(2) == 0)
	b.WriteString(head)
	b.WriteString("<pass:unterminated")
	b.WriteString("\n")
	return enhancedLine{
		text:      b.String(),
		expectErr: true,
	}
}

func lineEnhancedWhitespaceVariant(r *rand.Rand, state *enhancedState, mode enhancedMode) enhancedLine {
	var b strings.Builder
	spaces := " "
	if mode == modeGrammarFocus && r.Intn(2) == 0 {
		spaces = "\t"
	}
	b.WriteString(spaces)
	name := nextEnhancedVarName(state)
	head := buildEnhancedAssignmentHead(r, name, true)
	b.WriteString(head)
	path := nextEnhancedSecretPath(state)
	secret := enhancedBareSecretWithTab(r)
	modifiers := []string{"allow_tab"}
	b.WriteString(buildEnhancedPlaceholder(r, path, modifiers))
	if mode == modeGrammarFocus && r.Intn(2) == 0 {
		b.WriteString(" # trailing comment")
	}
	b.WriteString("\n")
	return enhancedLine{
		text:        b.String(),
		resolver:    externalResolver{path: secret},
		skipSandbox: true,
	}
}

func lineEnhancedComment(r *rand.Rand, state *enhancedState, mode enhancedMode) enhancedLine {
	words := []string{"comment", "note", "todo"}
	word := words[r.Intn(len(words))]
	return enhancedLine{
		text: "#" + word + "\n",
	}
}

func lineEnhancedBlank(r *rand.Rand, state *enhancedState, mode enhancedMode) enhancedLine {
	if r.Intn(2) == 0 {
		return enhancedLine{text: "\n"}
	}
	return enhancedLine{text: "   \n"}
}

func nextEnhancedVarName(state *enhancedState) string {
	name := fmt.Sprintf("ENH_VAR_%d", state.varIndex)
	state.varIndex++
	return name
}

func nextEnhancedSecretPath(state *enhancedState) string {
	path := fmt.Sprintf("enhanced_secret_%d", state.secretIndex)
	state.secretIndex++
	return path
}

func buildEnhancedAssignmentHead(r *rand.Rand, base string, allowAdvanced bool) string {
	var b strings.Builder
	if allowAdvanced {
		b.WriteString(strings.Repeat(" ", r.Intn(3)))
	} else {
		b.WriteString(strings.Repeat(" ", r.Intn(2)))
	}
	target := base
	if allowAdvanced && r.Intn(3) == 0 {
		target = fmt.Sprintf("%s[%d]", base, r.Intn(4))
	}
	b.WriteString(target)
	appendOp := allowAdvanced && r.Intn(3) == 0
	if appendOp {
		b.WriteString("+=")
	} else {
		b.WriteString("=")
	}
	return b.String()
}

func buildEnhancedPlaceholder(r *rand.Rand, path string, modifiers []string) string {
	if len(modifiers) == 0 {
		return fmt.Sprintf("<pass:%s>", path)
	}
	var b strings.Builder
	b.WriteString("<pass:")
	b.WriteString(path)
	b.WriteString("|")
	for i, mod := range modifiers {
		if i > 0 {
			if r.Intn(2) == 0 {
				b.WriteString(" ")
			}
			b.WriteString(",")
			if r.Intn(2) == 0 {
				b.WriteString(" ")
			}
		}
		if r.Intn(2) == 0 {
			b.WriteString(" ")
		}
		b.WriteString(mod)
		if r.Intn(2) == 0 {
			b.WriteString(" ")
		}
	}
	b.WriteString(">")
	return b.String()
}

func enhancedBase64Secret(r *rand.Rand) string {
	predefined := [][]byte{
		{0xfb},
		{0xfe},
		{0xfb, 0xfe},
		{0xfb, 0xff, 0x7f},
	}
	if r.Intn(3) == 0 {
		length := 3 + r.Intn(21)
		if length%3 == 0 {
			length++
		}
		buf := make([]byte, length)
		if _, err := r.Read(buf); err == nil {
			encoded := base64.StdEncoding.EncodeToString(buf)
			if strings.ContainsAny(encoded, "+/") && strings.Contains(encoded, "=") {
				return string(buf)
			}
		}
	}
	choice := predefined[r.Intn(len(predefined))]
	return string(choice)
}

func pickDoubleQuotedSecret(r *rand.Rand) ([]string, string) {
	switch r.Intn(5) {
	case 0:
		nl := enhancedNewlineSequence(r)
		return []string{"allow_newline"}, "line1" + nl + "line2"
	case 1:
		return []string{"allow_tab"}, "tab\tvalue"
	case 2:
		nl := enhancedNewlineSequence(r)
		return []string{"allow_newline", "allow_tab"}, "mix" + nl + "value\t"
	case 3:
		return nil, "plain$text"
	default:
		return []string{"allow_newline"}, "line" + enhancedNewlineSequence(r) + "trail"
	}
}

func pickCommandSecret(r *rand.Rand) ([]string, string) {
	switch r.Intn(5) {
	case 0:
		return []string{"allow_newline"}, "line1" + enhancedNewlineSequence(r) + "line2"
	case 1:
		return []string{"allow_tab"}, "tab\tcommand"
	case 2:
		return []string{"allow_newline", "allow_tab"}, "mix" + enhancedNewlineSequence(r) + "value\tcommand"
	case 3:
		return nil, "simple$cmd"
	default:
		nl := enhancedNewlineSequence(r)
		return []string{"allow_newline"}, "cmd" + nl + "suffix"
	}
}

func pickSingleQuotedValid(r *rand.Rand) ([]string, string) {
	options := []struct {
		mods   []string
		secret string
	}{
		{mods: nil, secret: "simpleValue"},
		{mods: []string{"allow_tab"}, secret: "tab\tvalue"},
	}
	choice := options[r.Intn(len(options))]
	return choice.mods, choice.secret
}

func pickSingleQuotedInvalid(r *rand.Rand) ([]string, string) {
	options := []struct {
		mods   []string
		secret string
	}{
		{mods: nil, secret: "line1\nline2"},
		{mods: nil, secret: "line1\rline2"},
		{mods: nil, secret: "O'Connor"},
		{mods: nil, secret: "tab\tvalue"},
		{mods: []string{"allow_newline"}, secret: "plain"},
		{mods: nil, secret: "control\x07"},
		{mods: nil, secret: "control\x7f"},
	}
	choice := options[r.Intn(len(options))]
	return choice.mods, choice.secret
}

func enhancedBareSecretWithTab(r *rand.Rand) string {
	if r.Intn(2) == 0 {
		return "bare\tvalue"
	}
	return "bare_value"
}

func enhancedNewlineSequence(r *rand.Rand) string {
	switch r.Intn(3) {
	case 0:
		return "\n"
	case 1:
		return "\r"
	default:
		return "\r\n"
	}
}
