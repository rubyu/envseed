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

// [EVT-BEP-1] Cross‑check: renderer vs measured minimal escape set (source)
// This test ensures no under‑escape: if sourcing an unescaped value fails or
// changes the value (measurement says escape is needed), then the renderer's
// Bare output MUST escape that character.
func TestBareEscape_CrossCheck_NoUnderEscape(t *testing.T) {
	t.Parallel()
	if ok, err := sandbox.Available(); !ok || err != nil {
		t.Skipf("sandbox unavailable: %v", err)
	}

	// Probe set: Always-escape + investigative metas
	candidates := []rune{' ', '#', '$', '"', '\'', '`', '\\', '(', ')', '{', '}', '[', ']', '|', '&', ';', '<', '>'}

	// Measure: for each candidate, determine if unescaped RHS in `.env` survives
	// `set -a; . .env` and preserves the value. Emit NEED: or FREE: lines.
	var script strings.Builder
	script.WriteString("set -e\n")
	script.WriteString("printf 'BASH_VERSION=%s\\n' \"$BASH_VERSION\"\nshopt -p\n")
	for i, r := range candidates {
		name := fmt.Sprintf("MVAR_%03d", i)
		script.WriteString("printf \"%s=%s\\n\" ")
		script.WriteString(name)
		script.WriteString(" ")
		script.WriteString(bashDollars(string(r)))
		script.WriteString(" > .env\n")
		script.WriteString("set +e\nset -a; . ./.env >/dev/null 2>&1\nrc=$?\nset -e\n")
		script.WriteString("if [ $rc -ne 0 ]; then echo NEED:")
		script.WriteString(name)
		script.WriteString("; else if [ \"$")
		script.WriteString(name)
		script.WriteString("\" = ")
		script.WriteString(bashDollars(string(r)))
		script.WriteString(" ]; then echo FREE:")
		script.WriteString(name)
		script.WriteString("; else echo NEED:")
		script.WriteString(name)
		script.WriteString("; fi; fi\n")
	}
	out, err := sandbox.Run(script.String())
	if err != nil {
		t.Fatalf("sandbox run (measure) failed: %v\nstdout=%s", err, out)
	}
	need := make(map[int]bool)
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "NEED:MVAR_") {
			var idx int
			fmt.Sscanf(line, "NEED:MVAR_%03d", &idx)
			need[idx] = true
		}
	}

	// Render Bare placeholders and check whether renderer escapes as needed.
	var tpl strings.Builder
	paths := make([]string, 0, len(candidates))
	values := make(map[string]string)
	for i, r := range candidates {
		name := fmt.Sprintf("RVAR_%03d", i)
		path := fmt.Sprintf("PATH_%03d", i)
		values[path] = string(r)
		paths = append(paths, name)
		tpl.WriteString(name)
		tpl.WriteString("=<pass:")
		tpl.WriteString(path)
		tpl.WriteString(">\n")
	}
	rendered, err := renderer.RenderString(tpl.String(), fakeResolver{m: values})
	if err != nil {
		t.Fatalf("render error: %v", err)
	}
	// Build a map of renderer escaping: for our simple case (single rune), RHS
	// should be either "\\<r>" (escaped) or "<r>" (unescaped).
	esc := make(map[int]bool)
	lines := strings.Split(strings.TrimSpace(rendered), "\n")
	for i, line := range lines {
		eq := strings.IndexByte(line, '=')
		if eq <= 0 {
			t.Fatalf("unexpected rendered line: %q", line)
		}
		rhs := line[eq+1:]
		r := string(candidates[i])
		if len(rhs) == 2 && rhs[0] == '\\' && string(rhs[1]) == r {
			esc[i] = true
		} else if rhs == r {
			esc[i] = false
		} else {
			// Some characters (e.g., backslash) may render as \\\\ (two backslashes)
			if r == "\\" && rhs == "\\\\" {
				esc[i] = true
			} else {
				t.Fatalf("cannot classify RHS for %q: %q", r, rhs)
			}
		}
	}

	// Assert: no under-escape (if measurement NEED -> renderer must escape)
	for i := range candidates {
		if need[i] && !esc[i] {
			t.Fatalf("UNDER-ESCAPE: char %q requires escaping by measurement but renderer did not escape", candidates[i])
		}
	}
}

// bashDollars encodes s as a $'...' literal suitable for bash printf %s.
func bashDollars(s string) string {
	var b strings.Builder
	b.WriteString("$")
	b.WriteByte('\'')
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch c {
		case '\\':
			b.WriteString("\\\\")
		case '\'':
			b.WriteString("\\'")
		case '\n':
			b.WriteString("\\n")
		case '\r':
			b.WriteString("\\r")
		case '\t':
			b.WriteString("\\t")
		default:
			if c < 0x20 || c == 0x7f {
				b.WriteString(fmt.Sprintf("\\x%02x", c))
			} else {
				b.WriteByte(c)
			}
		}
	}
	b.WriteByte('\'')
	return b.String()
}

// fakeResolver for this package
type fakeResolver struct{ m map[string]string }

func (f fakeResolver) Resolve(path string) (string, error) { return f.m[path], nil }
