// Algorithm overview:
//  1. Iteration plans (baseline seeds plus fuzz corpus) drive random template/resolver synthesis via runRandomTemplateIterations.
//  2. propertyIterationRoundTrip renders templates, asserts expected failures, and when successful performs parse validation,
//     bash safety checks, and optional sandbox execution to catch runtime hazards.
//  3. propertyIterationSimple shares the generator but limits assertions to rendering success and reparsability for lower-cost coverage.
//  4. Corpus files produced by fuzzing are replayed deterministically by readCorpusSeeds to guarantee regression visibility.
package renderer_test

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"envseed/internal/parser"
	renderer "envseed/internal/renderer"
	sbox "envseed/internal/sandbox"
	"envseed/internal/testgen"
	"envseed/internal/testsupport"
)

const (
	propertySandboxIterationLimit uint32 = 256
	propertyFuzzIterationLimit    uint32 = 2048
)

var propertyRoundTripBaselines = []propertyLoopPlan{
	{Seed: 314159265358979323, Iterations: 512},
	{Seed: -271828182845904523, Iterations: 512},
}

var propertySimpleBaselines = []propertyLoopPlan{
	{Seed: 161803398874989485, Iterations: 512},
	{Seed: -141421356237309504, Iterations: 512},
}

type propertyLoopPlan struct {
	Seed       int64
	Iterations uint32
}

type propertyMeta struct {
	Seed      int64
	Iteration uint32
}

type failurePhase string

const (
	failurePhaseNone   failurePhase = ""
	failurePhaseParse  failurePhase = "parse"
	failurePhaseRender failurePhase = "render"
)

type propertyExpectation struct {
	ShouldErr  bool
	DetailCode string
	Phase      failurePhase
}

func mergeExpectation(cur propertyExpectation, next propertyExpectation) propertyExpectation {
	if !cur.ShouldErr && next.ShouldErr {
		return next
	}
	return cur
}

type propertyTemplate struct {
	Template    string
	Resolver    externalResolver
	Expect      propertyExpectation
	SkipBash    bool
	SkipSandbox bool
}

type propertyIterationFunc func(t *testing.T, meta propertyMeta, tpl propertyTemplate)

// [EVT-MEP-3][EVT-MWP-6]
func TestRenderTemplateRoundTrip(t *testing.T) {
	sandbox := newSandboxTracker(t)
	runPropertyPlans(t, "roundtrip/baseline", propertyRoundTripBaselines, func(t *testing.T, meta propertyMeta, tpl propertyTemplate) {
		propertyIterationRoundTrip(t, sandbox, meta, tpl)
	})
	runPropertyCorpusCases(t, "FuzzRenderTemplateRoundTrip", func(t *testing.T, meta propertyMeta, tpl propertyTemplate) {
		propertyIterationRoundTrip(t, sandbox, meta, tpl)
	})
}

// [EVT-MEP-3]
func TestRenderTemplateSimple(t *testing.T) {
	runPropertyPlans(t, "simple/baseline", propertySimpleBaselines, propertyIterationSimple)
	runPropertyCorpusCases(t, "FuzzRenderTemplateSimple", propertyIterationSimple)
}

// [EVT-MEU-1][EVT-MWP-3]
func TestRender_SingleQuotedCorpusCases(t *testing.T) {
	cases := []struct {
		name string
		prop propertyTemplate
	}{
		{
			name: "ok_allow_tab",
			prop: propertyTemplate{
				Template: "VAR='<pass:secret|allow_tab>'\n",
				Resolver: externalResolver{"secret": "hello\tworld"},
			},
		},
		{
			name: "fail_newline",
			prop: propertyTemplate{
				Template: "VAR='<pass:secret>'\n",
				Resolver: externalResolver{"secret": "line1\nline2"},
				Expect:   propertyExpectation{ShouldErr: true, DetailCode: "EVE-105-102", Phase: failurePhaseRender},
			},
		},
		{
			name: "fail_contains_quote",
			prop: propertyTemplate{
				Template: "VAR='<pass:secret>'\n",
				Resolver: externalResolver{"secret": "O'Connor"},
				Expect:   propertyExpectation{ShouldErr: true, DetailCode: "EVE-105-101", Phase: failurePhaseRender},
			},
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			meta := propertyMeta{Seed: 0, Iteration: 0}
			propertyIterationSimple(t, meta, tc.prop)
		})
	}
}

// [EVT-MPF-1][EVT-MPF-4]
// Fuzz functions moved to render_template_fuzz_test.go

func runPropertyPlans(t *testing.T, label string, plans []propertyLoopPlan, fn propertyIterationFunc) {
	t.Helper()
	for idx, plan := range plans {
		plan := plan
		name := fmt.Sprintf("%s/seed_%d", label, plan.Seed)
		if len(plans) > 1 {
			name = fmt.Sprintf("%s/%02d_seed_%d", label, idx, plan.Seed)
		}
		t.Run(name, func(t *testing.T) {
			runRandomTemplateIterations(t, plan.Seed, plan.Iterations, fn)
		})
	}
}

func runPropertyCorpusCases(t *testing.T, fuzzName string, fn propertyIterationFunc) {
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
			runRandomTemplateIterations(t, seed.Seed, iterations, func(t *testing.T, meta propertyMeta, tpl propertyTemplate) {
				if meta.Iteration != seed.Iteration {
					return
				}
				fn(t, meta, tpl)
			})
		})
	}
}

func checkTemplateExpectation(t *testing.T, meta propertyMeta, tpl propertyTemplate, rendered string, err error) bool {
	t.Helper()
	if tpl.Expect.ShouldErr {
		if err == nil {
			t.Fatalf("seed=%d iteration=%d expected error but succeeded; template=%q rendered=%q", meta.Seed, meta.Iteration, tpl.Template, rendered)
		}
		if tpl.Expect.DetailCode != "" {
			switch tpl.Expect.Phase {
			case failurePhaseParse:
				var perr *parser.ParseError
				if !errors.As(err, &perr) {
					t.Fatalf("seed=%d iteration=%d expected parse detail %s, got %T (%v)", meta.Seed, meta.Iteration, tpl.Expect.DetailCode, err, err)
				}
				if perr.DetailCode != tpl.Expect.DetailCode {
					t.Fatalf("seed=%d iteration=%d parse detail code = %s, want %s", meta.Seed, meta.Iteration, perr.DetailCode, tpl.Expect.DetailCode)
				}
			case failurePhaseRender:
				var placeholderErr *renderer.PlaceholderError
				if errors.As(err, &placeholderErr) {
					if placeholderErr.DetailCode() != tpl.Expect.DetailCode {
						t.Fatalf("seed=%d iteration=%d render detail code = %s, want %s", meta.Seed, meta.Iteration, placeholderErr.DetailCode(), tpl.Expect.DetailCode)
					}
				} else {
					var outputErr *renderer.OutputValidationError
					if errors.As(err, &outputErr) {
						var innerParse *parser.ParseError
						if !errors.As(outputErr.Err, &innerParse) {
							t.Fatalf("seed=%d iteration=%d expected output parse detail %s, got %T (%v)", meta.Seed, meta.Iteration, tpl.Expect.DetailCode, outputErr.Err, outputErr.Err)
						}
						if innerParse.DetailCode != tpl.Expect.DetailCode {
							t.Fatalf("seed=%d iteration=%d output parse detail code = %s, want %s", meta.Seed, meta.Iteration, innerParse.DetailCode, tpl.Expect.DetailCode)
						}
					} else {
						t.Fatalf("seed=%d iteration=%d expected render detail %s, got %T (%v)", meta.Seed, meta.Iteration, tpl.Expect.DetailCode, err, err)
					}
				}
			}
		}
		return true
	}
	if err != nil {
		t.Fatalf("seed=%d iteration=%d unexpected error: %v\ntemplate=%q", meta.Seed, meta.Iteration, err, tpl.Template)
	}
	return false
}

func runRandomTemplateIterations(t *testing.T, seed int64, iterations uint32, fn propertyIterationFunc) {
	t.Helper()
	prof := &testgen.RendererRoundTripProfile{}
	testgen.RunIterations(t, seed, iterations, prof, func(t *testing.T, m testgen.Meta, c testgen.Case) {
		// cast to unified externalResolver
		r := externalResolver{}
		for k, v := range c.Resolver {
			r[k] = v
		}
		tpl := propertyTemplate{Template: c.Template, Resolver: r}
		fn(t, propertyMeta{Seed: m.Seed, Iteration: m.Iteration}, tpl)
	})
}

func propertyIterationRoundTrip(t *testing.T, sandbox *sandboxTracker, meta propertyMeta, tpl propertyTemplate) {
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

func propertyIterationSimple(t *testing.T, meta propertyMeta, tpl propertyTemplate) {
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
}

type sandboxTracker struct {
	t         *testing.T
	supported bool
}

func newSandboxTracker(t *testing.T) *sandboxTracker {
	t.Helper()
	supported, err := sbox.Available()
	if err != nil && !errors.Is(err, sbox.ErrUnsupported) && !supported {
		t.Logf("sandbox check failed, continuing without runtime validation: %v", err)
	}
	if errors.Is(err, sbox.ErrUnsupported) {
		t.Logf("bubblewrap sandbox unsupported: %v", err)
	}
	return &sandboxTracker{t: t, supported: supported}
}

func (s *sandboxTracker) shouldRun(meta propertyMeta, tpl propertyTemplate, rendered string) bool {
	if s == nil {
		return false
	}
	if !s.supported {
		s.t.Logf("sandbox skip: unsupported environment")
		return false
	}
	if tpl.SkipSandbox || tpl.SkipBash {
		s.t.Logf("sandbox skip: template flagged skip (skipSandbox=%v skipBash=%v)", tpl.SkipSandbox, tpl.SkipBash)
		return false
	}
	if meta.Iteration >= propertySandboxIterationLimit {
		s.t.Logf("sandbox skip: iteration %d exceeds limit %d", meta.Iteration, propertySandboxIterationLimit)
		return false
	}
	if strings.Contains(rendered, "`") {
		s.t.Logf("sandbox skip: rendered output contains backtick")
		return false
	}
	return true
}

func clampFuzzIteration(v uint32) uint32 {
	if propertyFuzzIterationLimit == 0 {
		return v
	}
	return v % propertyFuzzIterationLimit
}

// thin wrapper to call sandbox.Run irrespective of build tags
func sandboxRun(script string) (string, error) { return sbox.Run(script) }

func generatePropertyCase(t *testing.T, seed int64, iteration uint32) (propertyMeta, propertyTemplate) {
	t.Helper()
	prof := &testgen.RendererRoundTripProfile{}
	var last propertyTemplate
	testgen.RunIterations(t, seed, iteration+1, prof, func(t *testing.T, m testgen.Meta, c testgen.Case) {
		r := externalResolver{}
		for k, v := range c.Resolver {
			r[k] = v
		}
		last = propertyTemplate{
			Template:    c.Template,
			Resolver:    r,
			Expect:      propertyExpectation{ShouldErr: c.Expect.ShouldErr, DetailCode: c.Expect.DetailCode, Phase: failurePhase(c.Expect.Phase)},
			SkipBash:    c.SkipBash,
			SkipSandbox: c.SkipSandbox,
		}
	})
	return propertyMeta{Seed: seed, Iteration: iteration}, last
}

// [EVT-MEU-1][EVT-MWP-3]
