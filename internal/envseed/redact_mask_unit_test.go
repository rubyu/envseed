// [EVT-MSU-1][EVT-MSU-2]
package envseed

import (
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
