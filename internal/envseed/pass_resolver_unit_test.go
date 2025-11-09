package envseed

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"envseed/internal/parser"
	"envseed/internal/renderer"
)

// [EVT-MZU-1][EVT-MZU-3][EVT-MZU-4][EVT-MZU-5]
func TestPassResolverCachesDuplicatePaths(t *testing.T) {
	template := strings.Join([]string{
		"ONE=<pass:shared|strip_right>",
		"TWO=<pass:shared|strip_right>",
		"THREE=<pass:shared|strip_right><pass:shared|strip_right>",
		"FOUR=<pass:unique|strip_right>",
	}, "\n") + "\n"
	pass := &fakePass{values: map[string]string{
		"shared": "shared-value",
		"unique": "unique-value",
	}}
	elems, err := parser.Parse(template)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	resolver := newPassResolver(context.Background(), pass)
	defer resolver.Close()
	if _, err := renderer.RenderElements(elems, resolver); err != nil {
		t.Fatalf("RenderElements error: %v", err)
	}
	if got := pass.calls["shared"]; got != 1 {
		t.Fatalf("shared path resolved %d times, want 1", got)
	}
	if got := pass.calls["unique"]; got != 1 {
		t.Fatalf("unique path resolved %d times, want 1", got)
	}
}

// [EVT-BZU-1][EVT-BZP-1]
func TestPassResolverCacheScopedPerInstance(t *testing.T) {
	pass := &fakePass{values: map[string]string{"shared": "value"}}
	template := "VAR=<pass:shared|strip_right>\n"
	elems, err := parser.Parse(template)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	for i := 0; i < 2; i++ {
		resolver := newPassResolver(context.Background(), pass)
		if _, err := renderer.RenderElements(elems, resolver); err != nil {
			resolver.Close()
			t.Fatalf("RenderElements iteration %d error: %v", i, err)
		}
		resolver.Close()
	}
	if got := pass.calls["shared"]; got != 2 {
		t.Fatalf("shared path resolved %d times, want 2 across resolvers", got)
	}
}

// [EVT-MZU-2][EVT-MUU-1]
func TestPassResolverRejectsNULSecrets(t *testing.T) {
	template := "VAR=<pass:bad>\n"
	elems, err := parser.Parse(template)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	pass := &fakePass{values: map[string]string{"bad": "nul\x00value"}}
	resolver := newPassResolver(context.Background(), pass)
	defer resolver.Close()
	_, err = renderer.RenderElements(elems, resolver)
	if err == nil {
		t.Fatal("expected render error for NUL secret")
	}
	wrapped := wrapRenderError(err)
	var exitErr *ExitError
	if !errors.As(wrapped, &exitErr) {
		t.Fatalf("expected ExitError, got %v", wrapped)
	}
	if exitErr.DetailCode != "EVE-104-301" {
		t.Fatalf("detail code = %s, want EVE-104-301", exitErr.DetailCode)
	}
}

// [EVT-MWP-4]
func TestWrapRenderErrorSingleQuotedNewline(t *testing.T) {
	expectRenderError(t,
		"VAR='<pass:secret>'\n",
		map[string]string{"secret": "line1\nline2\n"},
		"EVE-105-102",
		"newline not permitted in single-quoted placeholder",
		"Newlines are not permitted in single‑quoted placeholders. Switch to double quotes and add the `allow_newline` modifier.",
	)
}

// [EVT-MGU-6][EVT-BDU-1]
func TestWrapRenderErrorReportsColumn(t *testing.T) {
	err := func() error {
		elems, parseErr := parser.Parse("VAR=<pass:secret>\n")
		if parseErr != nil {
			return parseErr
		}
		resolver := newPassResolver(context.Background(), &staticPassClient{values: map[string]string{"secret": "line1\nline2\n"}})
		defer resolver.Close()
		_, renderErr := renderer.RenderElements(elems, resolver)
		return renderErr
	}()
	if err == nil {
		t.Fatal("expected render error")
	}
	wrapped := wrapRenderError(err)
	var exitErr *ExitError
	if !errors.As(wrapped, &exitErr) {
		t.Fatalf("expected ExitError, got %v", wrapped)
	}
	msg := exitErr.Error()
	if !strings.Contains(msg, "column 5") {
		t.Fatalf("expected column information in error, got %q", msg)
	}
}

// [EVT-MPU-4]
func TestWrapRenderErrorInvalidModifierCombination(t *testing.T) {
	elems, err := parser.Parse("VAL=<pass:secret|base64,strip>\n")
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	_, err = renderer.RenderElements(elems, renderMapResolver{
		"secret": "value",
	})
	if err == nil {
		t.Fatalf("expected render error but got nil")
	}
	wrapped := wrapRenderError(err)
	var exitErr *ExitError
	if !errors.As(wrapped, &exitErr) {
		t.Fatalf("expected ExitError, got %v", wrapped)
	}
	if exitErr.DetailCode != "EVE-105-601" {
		t.Fatalf("detail code = %s, want EVE-105-601", exitErr.DetailCode)
	}
	expectedDetail := "The `base64` modifier cannot be combined with any other modifier. Remove the other modifiers, including the strip family and `dangerously_bypass_escape`."
	if exitErr.DetailText != expectedDetail {
		t.Fatalf("detail text = %q, want %q", exitErr.DetailText, expectedDetail)
	}
}

// [EVT-MEF-1][EVT-MUU-1]
func TestWrapRenderErrorContextDetailCodes(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name       string
		template   string
		values     map[string]string
		wantCode   string
		wantMsg    string
		wantDetail string
	}{
		{
			name:       "SingleQuotedContainsQuote",
			template:   "VAR='<pass:secret>'\n",
			values:     map[string]string{"secret": "bad'value\n"},
			wantCode:   "EVE-105-101",
			wantMsg:    "single-quoted placeholder cannot contain a single quote (`'`)",
			wantDetail: "Single quotes cannot contain `'`. Switch to double quotes; escaping is applied automatically in double‑quoted context.",
		},
		{
			name:       "SingleQuotedTabMissingAllow",
			template:   "VAR='<pass:secret>'\n",
			values:     map[string]string{"secret": "tab\tvalue\n"},
			wantCode:   "EVE-105-103",
			wantMsg:    "TAB not permitted in single-quoted placeholder",
			wantDetail: "TAB is not permitted in single‑quoted placeholders. Add the `allow_tab` modifier or switch to double quotes with `allow_tab`.",
		},
		{
			name:       "CommandSubstControl",
			template:   "VAR=$(printf %s <pass:secret>)\n",
			values:     map[string]string{"secret": "control\x02\n"},
			wantCode:   "EVE-105-303",
			wantMsg:    "control character U+0002 not permitted in command substitution placeholder",
			wantDetail: "Control characters are not supported in command substitution placeholders. Adjust the value.",
		},
		{
			name:       "BacktickNewlineUnsupported",
			template:   "VAR=`echo <pass:secret>`\n",
			values:     map[string]string{"secret": "line1\nline2\n"},
			wantCode:   "EVE-105-401",
			wantMsg:    "newline not permitted in backtick placeholder",
			wantDetail: "Newlines are not permitted in backtick placeholders. Replace backticks with `$()` and add the `allow_newline` modifier if needed.",
		},
		{
			name:       "BacktickTabMissingModifier",
			template:   "VAR=`echo <pass:secret>`\n",
			values:     map[string]string{"secret": "tab\tvalue\n"},
			wantCode:   "EVE-105-402",
			wantMsg:    "TAB not permitted in backtick placeholder",
			wantDetail: "TAB is not permitted in backtick placeholders. Add the `allow_tab` modifier or switch to a different quoting context.",
		},
		{
			name:       "BacktickControl",
			template:   "VAR=`echo <pass:secret>`\n",
			values:     map[string]string{"secret": "control\x03\n"},
			wantCode:   "EVE-105-403",
			wantMsg:    "control character U+0003 not permitted in backtick placeholder",
			wantDetail: "Control characters are not supported in backtick placeholders. Adjust the value.",
		},
		{
			name:       "BacktickAllowNewlineUnsupportedModifier",
			template:   "VAR=`echo <pass:secret|allow_newline>`\n",
			values:     map[string]string{"secret": "value\n"},
			wantCode:   "EVE-105-404",
			wantMsg:    "allow_newline modifier is not supported in backtick context",
			wantDetail: "The `allow_newline` modifier is not supported in backtick context. Replace backticks with `$()` and use `allow_newline`.",
		},
		{
			name:       "BareNewlineUnsupported",
			template:   "VAR=<pass:secret>\n",
			values:     map[string]string{"secret": "line1\nline2\n"},
			wantCode:   "EVE-105-501",
			wantMsg:    "newline not permitted in bare placeholder",
			wantDetail: "Newlines are not permitted in bare placeholders. Switch to double quotes and add the `allow_newline` modifier.",
		},
		{
			name:       "BareTabMissingModifier",
			template:   "VAR=<pass:secret>\n",
			values:     map[string]string{"secret": "tab\tvalue\n"},
			wantCode:   "EVE-105-502",
			wantMsg:    "TAB not permitted in bare placeholder",
			wantDetail: "TAB is not permitted in bare placeholders. Switch to double quotes or add the `allow_tab` modifier.",
		},
		{
			name:       "BareControl",
			template:   "VAR=<pass:secret>\n",
			values:     map[string]string{"secret": "control\x01\n"},
			wantCode:   "EVE-105-503",
			wantMsg:    "control character U+0001 not permitted in bare placeholder",
			wantDetail: "Control characters are not supported in bare placeholders. Quote or encode the value to avoid emitting unsupported control characters.",
		},
		{
			name:       "BareAllowNewlineUnsupportedModifier",
			template:   "VAR=<pass:secret|allow_newline>\n",
			values:     map[string]string{"secret": "value\n"},
			wantCode:   "EVE-105-504",
			wantMsg:    "allow_newline modifier is not supported in bare context",
			wantDetail: "The `allow_newline` modifier is not supported in bare context. Switch to double quotes and add the `allow_newline` modifier.",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			expectRenderError(t, tc.template, tc.values, tc.wantCode, tc.wantMsg, tc.wantDetail)
		})
	}
}

// Renderer smoke tests retained locally for non-dup coverage

// [EVT-MWP-1][EVT-MWP-3]
func TestRenderSingleQuotedTrimsTrailingNewlines(t *testing.T) {
	t.Parallel()

	template := "VAR='<pass:secret|strip_right>'\n"
	cases := []struct {
		name string
		raw  string
		want string
	}{
		{name: "LF", raw: "value\n", want: "value"},
		{name: "CR", raw: "value\r", want: "value"},
		{name: "CRLF", raw: "value\r\n", want: "value"},
		{name: "NonASCII", raw: "こんにちは\n", want: "こんにちは"},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			rendered := renderMustSucceed(t, template, map[string]string{"secret": tc.raw})
			wantRendered := fmt.Sprintf("VAR='%s'\n", tc.want)
			if rendered != wantRendered {
				t.Fatalf("rendered = %q, want %q", rendered, wantRendered)
			}
		})
	}
}

// [EVT-MWP-5]
func TestRenderSingleQuotedAllowTabSuccess(t *testing.T) {
	t.Parallel()

	template := "VAR='<pass:secret|allow_tab>'\n"
	rendered := renderMustSucceed(t, template, map[string]string{"secret": "a\tb"})
	const want = "VAR='a\tb'\n"
	if rendered != want {
		t.Fatalf("rendered = %q, want %q", rendered, want)
	}
}

// [EVT-MPP-2]
func TestRenderSingleQuotedDangerouslyBypassEscape(t *testing.T) {
	t.Parallel()

	template := "VAR='<pass:secret|dangerously_bypass_escape>'\n"
	secret := "bad'value\nline\n"
	rendered := renderMustSucceed(t, template, map[string]string{"secret": secret})
	if !strings.Contains(rendered, "bad'value") || !strings.Contains(rendered, "line") {
		t.Fatalf("rendered output %q does not contain expected secret", rendered)
	}
}
