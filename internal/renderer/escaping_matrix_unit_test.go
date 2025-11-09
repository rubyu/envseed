package renderer_test

import (
	"strings"
	"testing"

	"envseed/internal/parser"
	renderer "envseed/internal/renderer"
)

// [EVT-MEU-1][EVT-MEP-1]
func TestRender_ContextEscapingMatrix(t *testing.T) {
	t.Helper()
	cases := []struct {
		name     string
		template string
		secret   string
		expect   []string
		forbid   []string
	}{
		{
			name:     "BareEscapesSpacesAndHashes",
			template: "VAL=<pass:secret>\n",
			secret:   "value #$",
			expect:   []string{`value\ `, `\#`, `\$`},
			forbid:   []string{"value #"},
		},
		{
			name:     "DoubleQuotedEscapesMinimal",
			template: `VAL="<pass:secret>"` + "\n",
			secret:   "# \" \\ $ `",
			expect:   []string{`"# `, `\"`, `\\`, `\$`, "\\`"},
			forbid:   []string{`\#`},
		},
		{
			name:     "CommandSubstitutionEscapesClosingParen",
			template: "VAL=$(echo <pass:secret>)\n",
			secret:   "value)$",
			expect:   []string{`\)`, `\$`},
			forbid:   []string{"value)"},
		},
		{
			name:     "BacktickContextEscapesTickAndBackslash",
			template: "VAL=`echo <pass:secret>`\n",
			secret:   "`value\\",
			expect:   []string{"\\`", `\\`},
			forbid:   nil,
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got, err := renderer.RenderString(tc.template, externalResolver{"secret": tc.secret})
			if err != nil {
				t.Fatalf("RenderString error: %v", err)
			}
			for _, want := range tc.expect {
				if !strings.Contains(got, want) {
					t.Fatalf("rendered output %q missing %q", got, want)
				}
			}
			for _, forbidden := range tc.forbid {
				if forbidden != "" && strings.Contains(got, forbidden) {
					t.Fatalf("rendered output %q unexpectedly contains %q", got, forbidden)
				}
			}
		})
	}
}

// [EVT-MEU-2][EVT-MEP-2]
func TestRender_BareHashDoesNotBecomeComment(t *testing.T) {
	t.Helper()
	cases := []struct {
		name     string
		secret   string
		expected string
	}{
		{
			name:     "PlainHash",
			secret:   "value#comment",
			expected: "value\\#comment",
		},
		{
			name:     "PreEscapedHash",
			secret:   `value\#comment`,
			expected: "value\\\\\\#comment",
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			out, err := renderer.RenderString("VAL=<pass:secret>\n", externalResolver{"secret": tc.secret})
			if err != nil {
				t.Fatalf("RenderString error: %v", err)
			}
			elems, err := parser.Parse(out)
			if err != nil {
				t.Fatalf("parser.Parse(rendered) error: %v", err)
			}
			if len(elems) != 1 {
				t.Fatalf("expected 1 element, got %d", len(elems))
			}
			assign := elems[0].Assignment
			if assign.TrailingComment != "" {
				t.Fatalf("unexpected trailing comment: %q", assign.TrailingComment)
			}
			if len(assign.ValueTokens) != 1 {
				t.Fatalf("expected single value token, got %#v", assign.ValueTokens)
			}
			if assign.ValueTokens[0].Text != tc.expected {
				t.Fatalf("value token text = %q, want %q", assign.ValueTokens[0].Text, tc.expected)
			}
		})
	}
}

// [EVT-MWP-3][EVT-MWP-4][EVT-MWP-6][EVT-MPU-7]
func TestRender_AllowNewlineMatrix(t *testing.T) {
	t.Helper()
	cases := []struct {
		name       string
		template   string
		secret     string
		wantErr    bool
		code       string
		expectSub  string
		forbidSubs []string
	}{
		{
			name:      "DoubleQuotedWithModifier",
			template:  "VAL=\"<pass:secret|allow_newline>\"\n",
			secret:    "line1\nline2",
			expectSub: "line1\nline2",
		},
		{
			name:     "DoubleQuotedWithoutModifier",
			template: "VAL=\"<pass:secret>\"\n",
			secret:   "line1\nline2",
			wantErr:  true,
			code:     "EVE-105-201",
		},
		{
			name:      "BareWithAllowTab",
			template:  "VAL=<pass:secret|allow_tab>\n",
			secret:    "a\tb",
			expectSub: "a\\\tb",
		},
		{
			name:     "BareWithNewline",
			template: "VAL=<pass:secret>\n",
			secret:   "line1\nline2",
			wantErr:  true,
			code:     "EVE-105-501",
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got, err := renderer.RenderString(tc.template, externalResolver{"secret": tc.secret})
			if tc.wantErr {
				expectPlaceholderError(t, err, tc.code)
				return
			}
			if err != nil {
				t.Fatalf("RenderString error: %v", err)
			}
			if tc.expectSub != "" && !strings.Contains(got, tc.expectSub) {
				t.Fatalf("rendered output %q missing %q", got, tc.expectSub)
			}
		})
	}
}

// [EVT-MEF-1][EVT-MUU-1]
func TestRender_ControlCharacterRejectionMatrix(t *testing.T) {
	t.Helper()
	ctrlSecret := "bad\x01value"
	cases := []struct {
		name     string
		template string
		secret   string
		code     string
	}{
		{
			name:     "BareContext",
			template: "VAL=<pass:secret>\n",
			secret:   ctrlSecret,
			code:     "EVE-105-503",
		},
		{
			name:     "DoubleQuotedContext",
			template: "VAL=\"<pass:secret>\"\n",
			secret:   ctrlSecret,
			code:     "EVE-105-203",
		},
		{
			name:     "CommandSubstitutionContext",
			template: "VAL=$(echo <pass:secret>)\n",
			secret:   ctrlSecret,
			code:     "EVE-105-303",
		},
		{
			name:     "BacktickContext",
			template: "VAL=`echo <pass:secret>`\n",
			secret:   ctrlSecret,
			code:     "EVE-105-403",
		},
		{
			name:     "SingleQuotedContext",
			template: "VAL='<pass:secret>'\n",
			secret:   ctrlSecret,
			code:     "EVE-105-104",
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			_, err := renderer.RenderString(tc.template, externalResolver{"secret": tc.secret})
			expectPlaceholderError(t, err, tc.code)
		})
	}
}

// [EVT-MUP-1][EVT-MUP-2][EVT-MUP-3][EVT-MUF-1]
func TestRender_UnicodeAcceptanceMatrix(t *testing.T) {
	t.Helper()
	cases := []struct {
		name     string
		template string
		secret   string
	}{
		{
			name:     "BareNonASCII",
			template: "VAL=<pass:secret>\n",
			secret:   "値段アルファ",
		},
		{
			name:     "DoubleQuotedEmoji",
			template: "VAL=\"<pass:secret>\"\n",
			secret:   "café ☕",
		},
		{
			name:     "CommandSubstitutionCyrillic",
			template: "VAL=$(echo <pass:secret>)\n",
			secret:   "данные",
		},
		{
			name:     "SingleQuotedArabic",
			template: "VAL='<pass:secret>'\n",
			secret:   "مرحبا",
		},
		{
			name:     "BacktickHebrew",
			template: "VAL=`echo <pass:secret>`\n",
			secret:   "עברית",
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got, err := renderer.RenderString(tc.template, externalResolver{"secret": tc.secret})
			if err != nil {
				t.Fatalf("RenderString error: %v", err)
			}
			if !strings.Contains(got, tc.secret) {
				t.Fatalf("rendered output %q missing secret %q", got, tc.secret)
			}
		})
	}
}
