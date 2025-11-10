//go:build sandbox
// +build sandbox

package renderer_test

import (
	"fmt"
	"strings"
	"testing"

	"envseed/internal/renderer"
	"envseed/internal/sandbox"
)

// fakeResolver is provided in another test file within this package.

// [EVT-BEU-1] Bare/minimal escape set via sandboxed bash (source)
func TestBareMinimalEscape_Source_Succeeds(t *testing.T) {
	t.Parallel()
	if ok, err := sandbox.Available(); !ok || err != nil {
		t.Skipf("sandbox unavailable: %v", err)
	}

	// Always-escape candidates + a few investigative metas.
	candidates := []rune{' ', '#', '$', '"', '\'', '`', '\\', '(', ')', '{', '}', '[', ']', '|', '&', ';', '<', '>'}

	// Build a template with Bare placeholders, one per candidate.
	var tpl strings.Builder
	fr := fakeResolver{m: make(map[string]string)}
	varNames := make([]string, 0, len(candidates))
	for i, r := range candidates {
		name := varNameFor(i)
		path := name // use name as fake path key
		fr.m[path] = string(r)
		tpl.WriteString(name)
		tpl.WriteString("=<pass:")
		tpl.WriteString(path)
		tpl.WriteString(">\n")
		varNames = append(varNames, name)
	}

	rendered, err := renderer.RenderString(tpl.String(), fr)
	if err != nil {
		t.Fatalf("render error: %v", err)
	}

	// Run in sandboxed bash: source and declare -p each var (no external utilities)
	var sb strings.Builder
	sb.WriteString("set -euo pipefail\n")
	sb.WriteString("printf %s ")
	sb.WriteString(bashDollars(rendered))
	sb.WriteString(" > .env\nset -a\n. ./.env\n")
	for _, v := range varNames {
		sb.WriteString("if [ \"$")
		sb.WriteString(v)
		sb.WriteString("\" = ")
		sb.WriteString(bashDollars(fr.m[v]))
		sb.WriteString(" ]; then echo OK:")
		sb.WriteString(v)
		sb.WriteString("; else echo NG:")
		sb.WriteString(v)
		sb.WriteString("; fi\n")
	}

	out, err := sandbox.Run(sb.String())
	if err != nil {
		t.Fatalf("sandbox run failed: %v\nstdout=%s", err, out)
	}

	// Verify all lines report OK
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		if !strings.HasPrefix(strings.TrimSpace(line), "OK:") {
			t.Fatalf("unexpected result line: %q\nstdout=%s", line, out)
		}
	}
}

func varNameFor(idx int) string {
	// Produce simple, valid bash variable names.
	return fmt.Sprintf("VAR_%03d", idx)
}

// bashDollars is provided in another test file within this package.
