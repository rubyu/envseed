package renderer_test

import (
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"testing"

	"envseed/internal/parser"
	"envseed/internal/renderer"
	"envseed/internal/sandbox"
	"envseed/internal/testsupport"
)

// Shared resolver for external renderer tests is defined in helpers_external_test.go

// [EVT-MEU-1]
func TestRender_BareSimple(t *testing.T) {
	input := "EXAMPLE_VAR=<pass:example>\n"
	got, err := renderer.RenderString(input, externalResolver{"example": "abc123"})
	if err != nil {
		t.Fatalf("RenderString error: %v", err)
	}
	if got != "EXAMPLE_VAR=abc123\n" {
		t.Fatalf("rendered output = %q", got)
	}
}

// [EVT-MEU-1]
func TestRender_BareAutoEscapes(t *testing.T) {
	input := "EXAMPLE_VAR=<pass:example>\n"
	got, err := renderer.RenderString(input, externalResolver{"example": "hello world"})
	if err != nil {
		t.Fatalf("RenderString error: %v", err)
	}
	want := "EXAMPLE_VAR=hello\\ world\n"
	if got != want {
		t.Fatalf("rendered output = %q, want %q", got, want)
	}
}

// [EVT-MEU-1]
func TestRender_BareEscapesSpecialCharacters(t *testing.T) {
	input := "SPECIAL=<pass:example>\n"
	secret := "$value#(test)\\"
	got, err := renderer.RenderString(input, externalResolver{"example": secret})
	if err != nil {
		t.Fatalf("RenderString error: %v", err)
	}
	want := "SPECIAL=\\$value\\#\\(test\\)\\\\\n"
	if got != want {
		t.Fatalf("rendered output = %q, want %q", got, want)
	}
}

// [EVT-MEU-1]
func TestRender_DoubleQuotedEscaping(t *testing.T) {
	input := "EXAMPLE_VAR=\"pre <pass:example> post\"\n"
	got, err := renderer.RenderString(input, externalResolver{"example": `he"llo$`})
	if err != nil {
		t.Fatalf("RenderString error: %v", err)
	}
	want := "EXAMPLE_VAR=\"pre he\\\"llo\\$ post\"\n"
	if got != want {
		t.Fatalf("rendered output = %q, want %q", got, want)
	}
}

// [EVT-MWP-4]
func TestRender_DoubleQuotedAllowNewline(t *testing.T) {
	input := "EXAMPLE_VAR=\"<pass:example|allow_newline>\"\n"
	got, err := renderer.RenderString(input, externalResolver{"example": "line1\nline2"})
	if err != nil {
		t.Fatalf("RenderString error: %v", err)
	}
	want := "EXAMPLE_VAR=\"line1\nline2\"\n"
	if got != want {
		t.Fatalf("rendered output = %q, want %q", got, want)
	}
}

// [EVT-MEU-1]
func TestRenderStringExternal(t *testing.T) {
	out, err := renderer.RenderString("FOO=<pass:foo>\n", externalResolver{"foo": "bar"})
	if err != nil {
		t.Fatalf("RenderString error: %v", err)
	}
	if out != "FOO=bar\n" {
		t.Fatalf("rendered = %q, want FOO=bar", out)
	}
	elems, err := parser.Parse(out)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if len(elems) != 1 || elems[0].Assignment == nil || elems[0].Assignment.Name != "FOO" {
		t.Fatalf("unexpected parse result: %#v", elems)
	}
}

// [EVT-MEU-1]
func TestRender_CommandSubstitutionEscapes(t *testing.T) {
	input := "CMD=$(echo <pass:val|allow_newline>)\n"
	got, err := renderer.RenderString(input, externalResolver{"val": "value)\nnext"})
	if err != nil {
		t.Fatalf("RenderString error: %v", err)
	}
	want := "CMD=$(echo value\\)\nnext)\n"
	if got != want {
		t.Fatalf("rendered output = %q, want %q", got, want)
	}
}

// [EVT-MEU-1]
func TestRender_BacktickError(t *testing.T) {
	input := "STAMP=`echo <pass:token>`\n"
	// With only a trailing newline, normalization removes it and no error occurs.
	out, err := renderer.RenderString(input, externalResolver{"token": "line1\n"})
	if err != nil {
		t.Fatalf("RenderString unexpected error: %v", err)
	}
	if out != "STAMP=`echo line1`\n" {
		t.Fatalf("rendered output = %q", out)
	}
	// With an internal newline, backtick context MUST reject.
	_, err = renderer.RenderString(input, externalResolver{"token": "line1\nline2\n"})
	expectPlaceholderError(t, err, "EVE-105-401")
}

// [EVT-MEU-1]
func TestRender_SingleQuotedAllowsSimpleValue(t *testing.T) {
	input := "RAW='<pass:secret>'\n"
	out, err := renderer.RenderString(input, externalResolver{"secret": "value"})
	if err != nil {
		t.Fatalf("RenderString error: %v", err)
	}
	if out != "RAW='value'\n" {
		t.Fatalf("rendered output = %q", out)
	}
}

// [EVT-MEU-1]
func TestRender_SingleQuotedRejectsQuote(t *testing.T) {
	input := "RAW='<pass:secret>'\n"
	_, err := renderer.RenderString(input, externalResolver{"secret": "O'Connor"})
	expectPlaceholderError(t, err, "EVE-105-101")
}

// [EVT-MWP-4]
func TestRender_SingleQuotedRejectsNewline(t *testing.T) {
	input := "RAW='<pass:secret>'\n"
	_, err := renderer.RenderString(input, externalResolver{"secret": "line1\nline2"})
	expectPlaceholderError(t, err, "EVE-105-102")
}

// [EVT-MWP-3]
func TestRender_SingleQuotedTabHandling(t *testing.T) {
	base := "RAW='<pass:secret%s>'\n"
	secret := "hello\tworld"

	_, err := renderer.RenderString(fmt.Sprintf(base, ""), externalResolver{"secret": secret})
	expectPlaceholderError(t, err, "EVE-105-103")

	out, err := renderer.RenderString(fmt.Sprintf(base, "|allow_tab"), externalResolver{"secret": secret})
	if err != nil {
		t.Fatalf("RenderString with allow_tab error: %v", err)
	}
	if out != fmt.Sprintf("RAW='%s'\n", secret) {
		t.Fatalf("rendered output = %q", out)
	}
}

// [EVT-MUU-1]
func TestRender_SingleQuotedRejectsControlCharacter(t *testing.T) {
	input := "RAW='<pass:secret>'\n"
	_, err := renderer.RenderString(input, externalResolver{"secret": "ping\x07"})
	expectPlaceholderError(t, err, "EVE-105-104")
}

// [EVT-MWP-4]
func TestRender_SingleQuotedRejectsAllowNewlineModifier(t *testing.T) {
	input := "RAW='<pass:secret|allow_newline>'\n"
	_, err := renderer.RenderString(input, externalResolver{"secret": "value"})
	expectPlaceholderError(t, err, "EVE-105-105")
}

// [EVT-MWU-2] EOF newline normalization (Sections 5.1, 5.2)
func TestEOFNormalization_Unit(t *testing.T) {
	// Cases that, after normalization, are newline-free can render in bare context.
	for _, tc := range []struct {
		name   string
		secret string
		want   string
	}{
		{"lf_single", "abc\n", "abc"},
		{"crlf_single", "abc\r\n", "abc"},
		{"no_trailing", "abc", "abc"},
	} {
		t.Run("bare/"+tc.name, func(t *testing.T) {
			input := "VAL=<pass:pv>\n"
			out, err := renderer.RenderString(input, externalResolver{"pv": tc.secret})
			if err != nil {
				t.Fatalf("render: %v", err)
			}
			want := "VAL=" + tc.want + "\n"
			if out != want {
				t.Fatalf("got %q, want %q", out, want)
			}
		})
	}

	// Cases that leave a newline after normalization require allow_newline in contexts that accept newlines.
	for _, tc := range []struct {
		name   string
		secret string
		want   string
	}{
		{"lf_double", "abc\n\n", "abc\n"},
		{"crlf_double", "abc\r\n\r\n", "abc\r\n"},
		{"internal_then_trailing", "a\nb\n", "a\nb"},
		{"mixed_crlf_lf", "x\r\n\n", "x\r\n"},
		{"mixed_lf_crlf", "x\n\r\n", "x\n"},
	} {
		t.Run("double-allow-newline/"+tc.name, func(t *testing.T) {
			input := "VAL=\"<pass:pv|allow_newline>\"\n"
			out, err := renderer.RenderString(input, externalResolver{"pv": tc.secret})
			if err != nil {
				t.Fatalf("render: %v", err)
			}
			want := "VAL=\"" + tc.want + "\"\n"
			if out != want {
				t.Fatalf("got %q, want %q", out, want)
			}
		})
	}
}

// [EVT-MWP-7] Normalization × strip-family/order (Sections 5.1, 5.2)
func TestNormalizationOrder_WithStrip(t *testing.T) {
	// Value "X\n " should normalize to "X " then strip_right → "X".
	input := "VAL=<pass:pv|strip_right>\n"
	out, err := renderer.RenderString(input, externalResolver{"pv": "X\n "})
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if want := "VAL=X\n"; out != want {
		t.Fatalf("got %q, want %q", out, want)
	}

	// CRLF then TAB: normalize CRLF → leave TAB for strip_right.
	input = "VAL=<pass:pv|strip_right>\n"
	out, err = renderer.RenderString(input, externalResolver{"pv": "X\r\n\t"})
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if want := "VAL=X\n"; out != want { // TAB is stripped; final line has trailing newline from assignment
		t.Fatalf("got %q, want %q", out, want)
	}

	// Internal newline must not be removed by normalization; strip_right does not remove it.
	input = "VAL=\"<pass:pv|strip_right,allow_newline>\"\n"
	out, err = renderer.RenderString(input, externalResolver{"pv": "A\nB\n"})
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if want := "VAL=\"A\nB\"\n"; out != want {
		t.Fatalf("got %q, want %q", out, want)
	}
}

// [EVT-MWP-8] Normalization × contexts (Sections 5.1, 5.3)
func TestNormalization_ContextIndependence_DoubleQuoted(t *testing.T) {
	// Trailing-only newline is removed without needing allow_newline.
	input := "VAL=\"<pass:pv>\"\n"
	out, err := renderer.RenderString(input, externalResolver{"pv": "T\n"})
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if want := "VAL=\"T\"\n"; out != want {
		t.Fatalf("got %q, want %q", out, want)
	}

	// Internal newline remains and requires allow_newline.
	input = "VAL=\"<pass:pv>\"\n"
	_, err = renderer.RenderString(input, externalResolver{"pv": "T\nU"})
	if err == nil {
		t.Fatalf("want error for internal newline without allow_newline")
	}
}

// [EVT-MWP-3]
func TestRender_StripRightRemovesTrailingNewline(t *testing.T) {
	input := "TOKEN=<pass:secret|strip_right>\n"
	got, err := renderer.RenderString(input, externalResolver{"secret": "value\n"})
	if err != nil {
		t.Fatalf("RenderString error: %v", err)
	}
	want := "TOKEN=value\n"
	if got != want {
		t.Fatalf("rendered output = %q, want %q", got, want)
	}
}

// [EVT-MWP-3]
func TestRender_StripLeftRemovesLeadingWhitespace(t *testing.T) {
	input := "VAL=\"<pass:secret|strip_left>\"\n"
	got, err := renderer.RenderString(input, externalResolver{"secret": "\t\n trimmed"})
	if err != nil {
		t.Fatalf("RenderString error: %v", err)
	}
	want := "VAL=\"trimmed\"\n"
	if got != want {
		t.Fatalf("rendered output = %q, want %q", got, want)
	}
}

// [EVT-MWP-3]
func TestRender_StripRemovesBothSides(t *testing.T) {
	input := "VAL=\"<pass:secret|strip>\"\n"
	got, err := renderer.RenderString(input, externalResolver{"secret": "\r\n spaced value \n"})
	if err != nil {
		t.Fatalf("RenderString error: %v", err)
	}
	want := "VAL=\"spaced value\"\n"
	if got != want {
		t.Fatalf("rendered output = %q, want %q", got, want)
	}
}

// [EVT-MEP-3][EVT-MWP-6]
func TestRender_CRLFAllowance(t *testing.T) {
	t.Helper()
	cr := "a\rb"
	lf := "a\nb"
	crlf := "a\r\nb"
	cases := []struct {
		name        string
		template    string
		secret      string
		expectErr   bool
		code        string
		verifyParse bool
	}{
		{
			name:        "DoubleQuotedAllowCRLF",
			template:    "VAL=\"<pass:secret|allow_newline>\"\n",
			secret:      crlf,
			verifyParse: true,
		},
		{
			name:        "DoubleQuotedAllowCR",
			template:    "VAL=\"<pass:secret|allow_newline>\"\n",
			secret:      cr,
			verifyParse: true,
		},
		{
			name:      "DoubleQuotedRejectCRLF",
			template:  "VAL=\"<pass:secret>\"\n",
			secret:    crlf,
			expectErr: true,
			code:      "EVE-105-201",
		},
		{
			name:      "DoubleQuotedRejectCR",
			template:  "VAL=\"<pass:secret>\"\n",
			secret:    cr,
			expectErr: true,
			code:      "EVE-105-201",
		},
		{
			name:        "CommandSubstitutionAllowCRLF",
			template:    "VAL=$(echo <pass:secret|allow_newline>)\n",
			secret:      crlf,
			verifyParse: true,
		},
		{
			name:        "CommandSubstitutionAllowCR",
			template:    "VAL=$(echo <pass:secret|allow_newline>)\n",
			secret:      cr,
			verifyParse: true,
		},
		{
			name:      "CommandSubstitutionRejectCRLF",
			template:  "VAL=$(echo <pass:secret>)\n",
			secret:    crlf,
			expectErr: true,
			code:      "EVE-105-301",
		},
		{
			name:      "CommandSubstitutionRejectCR",
			template:  "VAL=$(echo <pass:secret>)\n",
			secret:    cr,
			expectErr: true,
			code:      "EVE-105-301",
		},
		{
			name:      "BareRejectsAllowNewlineModifier",
			template:  "VAL=<pass:secret|allow_newline>\n",
			secret:    crlf,
			expectErr: true,
			code:      "EVE-105-504",
		},
		{
			name:      "SingleQuotedRejectsNewlines",
			template:  "VAL='<pass:secret>'\n",
			secret:    crlf,
			expectErr: true,
			code:      "EVE-105-102",
		},
		{
			name:      "BacktickRejectsAllowNewlineModifier",
			template:  "VAL=`echo <pass:secret|allow_newline>`\n",
			secret:    crlf,
			expectErr: true,
			code:      "EVE-105-404",
		},
		{
			name:      "BareRejectsLF",
			template:  "VAL=<pass:secret|allow_newline>\n",
			secret:    lf,
			expectErr: true,
			code:      "EVE-105-504",
		},
		{
			name:      "CommandSubstitutionRejectLFWithoutModifier",
			template:  "VAL=$(echo <pass:secret>)\n",
			secret:    lf,
			expectErr: true,
			code:      "EVE-105-301",
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			mapper := externalResolver{"secret": tc.secret}
			rendered, err := renderer.RenderString(tc.template, mapper)
			if tc.expectErr {
				expectPlaceholderError(t, err, tc.code)
				return
			}
			if err != nil {
				t.Fatalf("RenderString error: %v", err)
			}
			if tc.verifyParse {
				if _, err := parser.Parse(rendered); err != nil {
					t.Fatalf("parser.Parse(rendered) error: %v", err)
				}
			}
		})
	}
}

// [EVT-MUP-1]
func TestRender_InvalidModifierCombination(t *testing.T) {
	inputs := []string{
		"VAL=<pass:secret|base64,strip>\n",
		"VAL=<pass:secret|base64,strip_left>\n",
		"VAL=<pass:secret|base64,strip_right>\n",
	}
	for _, input := range inputs {
		_, err := renderer.RenderString(input, externalResolver{"secret": "value"})
		expectPlaceholderError(t, err, "EVE-105-601")
	}
}

// [EVT-MUP-1]
func TestRender_Base64InvalidCombinations(t *testing.T) {
	t.Helper()
	cases := []struct {
		name     string
		template string
		secret   string
		code     string
	}{
		{
			name:     "AllowTab",
			template: "VAL=<pass:secret|base64,allow_tab>\n",
			secret:   "value",
			code:     "EVE-105-601",
		},
		{
			name:     "AllowNewline",
			template: "VAL=\"<pass:secret|base64,allow_newline>\"\n",
			secret:   "value",
			code:     "EVE-105-601",
		},
		{
			name:     "BothAllow",
			template: "VAL=\"<pass:secret|base64,allow_tab,allow_newline>\"\n",
			secret:   "value",
			code:     "EVE-105-601",
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

// [EVT-MPP-3]
func TestRender_BareBase64SpecialCharacters(t *testing.T) {
	t.Helper()
	resolver := externalResolver{"secret": string([]byte{0xfb, 0xfe})}
	out, err := renderer.RenderString("VAL=<pass:secret|base64>\n", resolver)
	if err != nil {
		t.Fatalf("RenderString error: %v", err)
	}
	if !strings.Contains(out, "+/4=") {
		t.Fatalf("base64 output missing expected characters: %q", out)
	}
	if strings.Contains(out, `\+`) || strings.Contains(out, `\/`) || strings.Contains(out, `\=`) {
		t.Fatalf("base64 output unexpectedly escaped specials: %q", out)
	}
	if _, err := parser.Parse(out); err != nil {
		t.Fatalf("parser.Parse(rendered) error: %v", err)
	}
}

// [EVT-MPP-2]
func TestRender_DangerouslyBypass(t *testing.T) {
	input := "EXAMPLE_VAR=<pass:example|dangerously_bypass_escape>\n"
	got, err := renderer.RenderString(input, externalResolver{"example": "a b$c"})
	if err != nil {
		t.Fatalf("RenderString error: %v", err)
	}
	want := "EXAMPLE_VAR=a b$c\n"
	if got != want {
		t.Fatalf("rendered output = %q, want %q", got, want)
	}
}

// [EVT-MPP-2]
func TestRender_DangerouslyBypassSkipsReparse(t *testing.T) {
	input := "VAR=<pass:secret|dangerously_bypass_escape>\n"
	secret := "\n=oops"
	got, err := renderer.RenderString(input, externalResolver{"secret": secret})
	if err != nil {
		t.Fatalf("RenderString error: %v", err)
	}
	want := "VAR=\n=oops\n"
	if got != want {
		t.Fatalf("rendered output = %q, want %q", got, want)
	}
	if _, err := parser.Parse(got); err == nil {
		t.Fatalf("expected parser.Parse to fail for dangerously_bypass_escape output")
	}
}

// [EVT-MEP-1]
func TestRender_MultiLevelEvaluation(t *testing.T) {
	supported, checkErr := renderer.ExportSandboxAvailable()
	if !supported {
		if errors.Is(checkErr, sandbox.ErrUnsupported) {
			t.Skipf("bubblewrap sandbox unsupported: %v", checkErr)
		}
		if checkErr != nil {
			t.Skipf("sandbox availability check failed: %v", checkErr)
		}
		t.Skip("bubblewrap sandbox unavailable")
	}

	input := strings.ReplaceAll(`OUT=$(printf "%s|%s|%s" "$(printf %s "<pass:alpha>")" "$(printf %s "$(printf %s "<pass:beta>")")" "{{BT}}printf %s "<pass:gamma>"{{BT}}")`+"\n", "{{BT}}", "`")

	resolver := externalResolver{
		"alpha": "LEVEL-A",
		"beta":  "LEVEL-B",
		"gamma": "LEVEL-C",
	}

	rendered, err := renderer.RenderString(input, resolver)
	if err != nil {
		t.Fatalf("RenderString error: %v", err)
	}
	values, err := renderer.ExportSandboxCapture(rendered, []string{"OUT"})
	if err != nil {
		t.Fatalf("sandbox execution failed: %v", err)
	}
	if got := values["OUT"]; got != "LEVEL-A|LEVEL-B|LEVEL-C" {
		t.Fatalf("OUT value = %q, want %q", got, "LEVEL-A|LEVEL-B|LEVEL-C")
	}
}

// [EVT-MWP-3]
func TestRender_AllowTabBare(t *testing.T) {
	input := "EXAMPLE_VAR=<pass:example|allow_tab>\n"
	got, err := renderer.RenderString(input, externalResolver{"example": "a\tb"})
	if err != nil {
		t.Fatalf("RenderString error: %v", err)
	}
	want := "EXAMPLE_VAR=a\\\tb\n"
	if got != want {
		t.Fatalf("rendered output = %q, want %q", got, want)
	}
}

// [EVT-MEP-3]
func TestRender_RoundTripAndBash(t *testing.T) {
	input := "# header\nEXAMPLE_VAR=\"value <pass:example|allow_tab>\"\nBAR=$(echo <pass:bar|allow_newline>)\n"
	rendered, err := renderer.RenderString(input, externalResolver{
		"example": "a\tb",
		"bar":     "line1\nline2",
	})
	if err != nil {
		t.Fatalf("RenderString error: %v", err)
	}
	if _, err := parser.Parse(rendered); err != nil {
		t.Fatalf("re-parse failure: %v", err)
	}
	if err := testsupport.BashValidate(rendered); err != nil {
		t.Fatalf("bash -n validation failed: %v", err)
	}
}

// [EVT-MPP-3]
func TestRender_Base64Bare(t *testing.T) {
	input := "TOKEN=<pass:secret|base64>\n"
	secret := "\xff"
	rendered, err := renderer.RenderString(input, externalResolver{"secret": secret})
	if err != nil {
		t.Fatalf("RenderString error: %v", err)
	}
	wantEncoded := base64.StdEncoding.EncodeToString([]byte(secret))
	want := "TOKEN=" + wantEncoded + "\n"
	if rendered != want {
		t.Fatalf("rendered output = %q, want %q", rendered, want)
	}
}

// [EVT-MPP-3]
func TestRender_Base64BareAllowsPaddingAndSlash(t *testing.T) {
	input := "TOKEN=<pass:secret|base64>\n"
	secret := "\xff\xef"
	rendered, err := renderer.RenderString(input, externalResolver{"secret": secret})
	if err != nil {
		t.Fatalf("RenderString error: %v", err)
	}
	wantEncoded := base64.StdEncoding.EncodeToString([]byte(secret))
	if !strings.Contains(wantEncoded, "/") || !strings.HasSuffix(wantEncoded, "=") {
		t.Fatalf("test secret did not produce expected base64 characters, got %q", wantEncoded)
	}
	want := "TOKEN=" + wantEncoded + "\n"
	if rendered != want {
		t.Fatalf("rendered output = %q, want %q", rendered, want)
	}
}

// [EVT-MPP-3]
func TestRender_Base64DoubleQuoted(t *testing.T) {
	input := `TOKEN="<pass:secret|base64>"` + "\n"
	secret := "raw value"
	rendered, err := renderer.RenderString(input, externalResolver{"secret": secret})
	if err != nil {
		t.Fatalf("RenderString error: %v", err)
	}
	wantEncoded := base64.StdEncoding.EncodeToString([]byte(secret))
	want := `TOKEN="` + wantEncoded + `"` + "\n"
	if rendered != want {
		t.Fatalf("rendered output = %q, want %q", rendered, want)
	}
}

// [EVT-MEP-1]
func TestRender_BareMultiplePlaceholders(t *testing.T) {
	input := "EXAMPLE_VAR=<pass:a><pass:b>\n"
	rendered, err := renderer.RenderString(input, externalResolver{"a": "one", "b": "two"})
	if err != nil {
		t.Fatalf("RenderString error: %v", err)
	}
	if rendered != "EXAMPLE_VAR=onetwo\n" {
		t.Fatalf("rendered = %q", rendered)
	}
}

// [EVT-MEP-1]
func TestRender_BareMultiplePlaceholdersSanitized(t *testing.T) {
	input := "EXAMPLE_VAR=<pass:a><pass:b>\n"
	rendered, err := renderer.RenderString(input, externalResolver{"a": "ok", "b": ";rm -rf"})
	if err != nil {
		t.Fatalf("RenderString error: %v", err)
	}
	want := "EXAMPLE_VAR=ok\\;rm\\ -rf\n"
	if rendered != want {
		t.Fatalf("rendered output = %q, want %q", rendered, want)
	}
}

// [EVT-MEP-1]
func TestRender_PrefixSuffixInjectionGuard(t *testing.T) {
	input := "EXAMPLE_VAR=prefix<pass:a>suffix\n"
	rendered, err := renderer.RenderString(input, externalResolver{"a": " value"})
	if err != nil {
		t.Fatalf("RenderString error: %v", err)
	}
	want := "EXAMPLE_VAR=prefix\\ valuesuffix\n"
	if rendered != want {
		t.Fatalf("rendered output = %q, want %q", rendered, want)
	}
}

// [EVT-MEU-1]
func TestRender_CommandBoundaryEscaping(t *testing.T) {
	input := "CMD=$(printf %s <pass:a|allow_newline>)\n"
	rendered, err := renderer.RenderString(input, externalResolver{"a": "middle)more"})
	if err != nil {
		t.Fatalf("RenderString error: %v", err)
	}
	if !strings.Contains(rendered, "middle\\)more") {
		t.Fatalf("expected escaped closing paren, got %q", rendered)
	}
}

// [EVT-MEU-1]
func TestRender_BoundaryMatrix(t *testing.T) {
	cases := []struct {
		name     string
		input    string
		resolver externalResolver
		want     string
		verify   bool
	}{
		{
			name:  "BareAdjacentPlaceholders",
			input: "EXAMPLE_VAR=<pass:a><pass:b>\n",
			resolver: externalResolver{
				"a": "one",
				"b": "two",
			},
			want: "EXAMPLE_VAR=onetwo\n",
		},
		{
			name:     "BareSecretAutoEscapes",
			input:    "EXAMPLE_VAR=<pass:a>\n",
			resolver: externalResolver{"a": "needs space"},
			want:     "EXAMPLE_VAR=needs\\ space\n",
		},
		{
			name:  "DoubleQuotedBoundary",
			input: "EXAMPLE_VAR=\"[<pass:a>]<pass:b>\"\n",
			resolver: externalResolver{
				"a": "left]",
				"b": "right",
			},
			want: "EXAMPLE_VAR=\"[left]]right\"\n",
		},
		{
			name:     "CommandClosingParen",
			input:    "CMD=$(printf %s <pass:a>)\n",
			resolver: externalResolver{"a": "value)tail"},
			want:     "CMD=$(printf %s value\\)tail)\n",
		},
		{
			name:     "BarePlaceholderBeforeComment",
			input:    "VALUE=<pass:a> # comment\n",
			resolver: externalResolver{"a": "ok"},
			want:     "VALUE=ok # comment\n",
			verify:   true,
		},
		{
			name:  "DoubleQuotedAdjacentPlaceholders",
			input: "VAL=\"<pass:a><pass:b>\"\n",
			resolver: externalResolver{
				"a": "one",
				"b": "two",
			},
			want:   "VAL=\"onetwo\"\n",
			verify: true,
		},
		{
			name:  "CommandSubstitutionAdjacentPlaceholders",
			input: "CMD=$(printf %s <pass:a><pass:b>)\n",
			resolver: externalResolver{
				"a": "left",
				"b": "right",
			},
			want:   "CMD=$(printf %s leftright)\n",
			verify: true,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			rendered, err := renderer.RenderString(tc.input, tc.resolver)
			if err != nil {
				t.Fatalf("RenderString error: %v", err)
			}
			if rendered != tc.want {
				t.Fatalf("rendered output = %q, want %q", rendered, tc.want)
			}
			if tc.verify {
				if _, err := parser.Parse(rendered); err != nil {
					t.Fatalf("parser.Parse(rendered) error: %v", err)
				}
			}
		})
	}
}

// [EVT-MUP-1]
func TestRender_ModifierErrors(t *testing.T) {
	cases := []struct {
		name     string
		input    string
		resolver externalResolver
		code     string
	}{
		{
			name:  "BareAllowNewlineUnsupported",
			input: "VAL=<pass:secret|allow_newline>\n",
			resolver: externalResolver{
				"secret": "line1\nline2",
			},
			code: "EVE-105-504",
		},
		{
			name:  "BacktickAllowNewlineUnsupported",
			input: "VAL=`printf %s <pass:secret|allow_newline>`\n",
			resolver: externalResolver{
				"secret": "line1\nline2",
			},
			code: "EVE-105-404",
		},
		{
			name:  "DoubleQuotedTabMissingModifier",
			input: "VAL=\"<pass:secret>\"\n",
			resolver: externalResolver{
				"secret": "tab\tvalue",
			},
			code: "EVE-105-202",
		},
		{
			name:  "CommandSubstitutionTabMissingModifier",
			input: "CMD=$(printf %s <pass:secret>)\n",
			resolver: externalResolver{
				"secret": "tab\tcommand",
			},
			code: "EVE-105-302",
		},
		{
			name:  "CommandSubstitutionNewlineMissingModifier",
			input: "CMD=$(printf %s <pass:secret>)\n",
			resolver: externalResolver{
				"secret": "line1\nline2",
			},
			code: "EVE-105-301",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			_, err := renderer.RenderString(tc.input, tc.resolver)
			expectPlaceholderError(t, err, tc.code)
		})
	}
}

// [EVT-MEP-3]
func TestRender_RealisticScenario(t *testing.T) {
	input := `# Example service configuration
API_ENDPOINT="https://<pass:host>/v1"
AUTH_HEADER="Bearer <pass:key>"
SCRIPT=$(printf "%s" "<pass:script|allow_newline>")
MESSAGE="<pass:message|allow_tab>"
`
	resolver := externalResolver{
		"host":    "internal.example.org",
		"key":     "s3cr3t$token",
		"script":  "deploy\nnext",
		"message": "tab\tok",
	}
	rendered, err := renderer.RenderString(input, resolver)
	if err != nil {
		t.Fatalf("RenderString error: %v", err)
	}
	if _, err := parser.Parse(rendered); err != nil {
		t.Fatalf("re-parse failure: %v", err)
	}
	if err := testsupport.BashValidate(rendered); err != nil {
		t.Fatalf("bash -n validation failed: %v", err)
	}
	if supported, checkErr := renderer.ExportSandboxAvailable(); supported {
		values, err := renderer.ExportSandboxCapture(rendered, []string{"API_ENDPOINT", "AUTH_HEADER", "SCRIPT", "MESSAGE"})
		if err != nil {
			if !errors.Is(err, sandbox.ErrUnsupported) {
				t.Fatalf("sandbox capture failed: %v", err)
			}
		} else {
			want := map[string]string{
				"API_ENDPOINT": "https://internal.example.org/v1",
				"AUTH_HEADER":  "Bearer s3cr3t$token",
				"SCRIPT":       "deploy\nnext",
				"MESSAGE":      "tab\tok",
			}
			if diff := compareStringMaps(values, want); diff != "" {
				t.Fatalf("sandbox values mismatch:\n%s", diff)
			}
		}
	} else if checkErr != nil && !errors.Is(checkErr, sandbox.ErrUnsupported) {
		t.Fatalf("sandbox availability check failed: %v", checkErr)
	}
}

func compareStringMaps(got, want map[string]string) string {
	var b strings.Builder
	for key, wantVal := range want {
		gotVal, ok := got[key]
		if !ok {
			b.WriteString(fmt.Sprintf("- missing key %s (want %q)\n", key, wantVal))
			continue
		}
		if gotVal != wantVal {
			b.WriteString(fmt.Sprintf("- %s mismatch: got %q want %q\n", key, gotVal, wantVal))
		}
	}
	for key := range got {
		if _, ok := want[key]; !ok {
			b.WriteString(fmt.Sprintf("- unexpected key %s (got %q)\n", key, got[key]))
		}
	}
	return b.String()
}
