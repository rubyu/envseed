// [EVT-MSU-1][EVT-MSU-2]
package envseed

import (
	"errors"
	"testing"
)

// [EVT-MSU-1][EVT-MSU-2] Basic sanity for escape-pair elision in masking.
func TestMaskEnv_ElidesBackslashesInSegments(t *testing.T) {
	cases := []struct {
		name            string
		in              string
		wantNoBackslash bool
	}{
		{"double_dollar", "KEY=\"abc\\$xyz\"\n", true},
		{"double_quote", "KEY=\"abc\\\"xyz\"\n", true},
		{"backtick_dollar", "KEY=`echo \\$x`\n", true},
		{"command_subst_close", "KEY=$(echo \\)x)\n", true},
		{"bare_space", "KEY=abc\\ xyz\n", true},
		{"top_level_hash", "KEY=abc\\#xyz\n", true},
		{"single_quote_literal", "KEY='a\\$b'\n", false}, // backslash is literal in single quotes
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out, err := MaskEnv(tc.in)
			if err != nil {
				t.Fatalf("MaskEnv error: %v", err)
			}
			if tc.wantNoBackslash && containsRune(out, '\\') {
				t.Fatalf("masked output contains backslash: %q", out)
			}
		})
	}
}

func containsRune(s string, r rune) bool {
	for _, x := range s {
		if x == r {
			return true
		}
	}
	return false
}

// [EVT-MWU-3] Target .env grammar-level whitespace: value-internal Unicode is accepted
func TestParseTarget_AllowsUnicodeWhitespaceInsideValue(t *testing.T) {
	// NBSP inside a double-quoted value must be accepted
	text := "VAL=\"a\u00A0b\"\n"
	if _, err := ParseTarget(text); err != nil {
		t.Fatalf("ParseTarget unexpected error: %v", err)
	}
}

// [EVT-MWU-3] Target .env grammar-level whitespace: leading non-ASCII whitespace rejected
func TestParseTarget_RejectsNonASCIILeadingWhitespace(t *testing.T) {
	// NBSP before a whole-line comment: should be grammar-level whitespace violation (EVE-107-101)
	text := "\u00A0# comment\n"
	_, err := ParseTarget(text)
	var exitErr *ExitError
	if err == nil || !errors.As(err, &exitErr) || exitErr.DetailCode != "EVE-107-101" {
		t.Fatalf("expected EVE-107-101, got %v", err)
	}
}
