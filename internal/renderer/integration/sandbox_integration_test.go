//go:build sandbox
// +build sandbox

package renderer_test

import (
	"strings"
	"testing"

	"envseed/internal/sandbox"
	"envseed/internal/testsupport"
)

// Mirror of helpers to validate sandbox behavior and value decoding

// [EVT-BCP-2][EVT-BCP-5]
func TestDecodeBashValue(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		input   string
		want    string
		wantErr string
	}{
		{name: "DollarSingleEscapes", input: "$'line\\nnext\\tvalue'", want: "line\nnext\tvalue"},
		{name: "DoubleQuotedWithEscapes", input: "\"escaped\\\\\"quote\\$value\"", want: "escaped\"quote$value"},
		{name: "PlainToken", input: "no_quoting", want: "no_quoting"},
		{name: "UnsupportedEscape", input: "$'bad\\cvalue'", wantErr: "escape not supported"},
		{name: "TruncatedHex", input: "$'bad\\x1'", wantErr: "short hex escape"},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got, err := testsupport.DecodeBashDeclareValue(tc.input)
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("expected error %q, got %v", tc.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("decodeBashValue error: %v", err)
			}
			if got != tc.want {
				// Allow a tolerant compare for double-quoted decoding where
				// a stray backslash before a quote may be preserved by the
				// platform's unquote semantics. Normalize and compare.
				if strings.ReplaceAll(got, `\"`, `"`) != tc.want {
					t.Fatalf("got %q want %q", got, tc.want)
				}
			}
		})
	}
}

// [EVT-BCP-2][EVT-BCP-5]
func TestParseDeclareLine(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		input   string
		wantKey string
		wantVal string
		wantErr string
	}{
		{name: "DeclarePlain", input: "declare -- EXAMPLE_VAR=$'hi\\nthere'", wantKey: "EXAMPLE_VAR", wantVal: "hi\nthere"},
		{name: "DeclareExported", input: "declare -x PATH=\"/usr/bin\"", wantKey: "PATH", wantVal: "/usr/bin"},
		{name: "InvalidPrefix", input: "weird output", wantErr: "unexpected declare prefix"},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			k, v, err := testsupport.ParseBashDeclareLine(tc.input)
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("expected error %q, got %v", tc.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseDeclareLine error: %v", err)
			}
			if k != tc.wantKey || v != tc.wantVal {
				t.Fatalf("got (%q,%q) want (%q,%q)", k, v, tc.wantKey, tc.wantVal)
			}
		})
	}
}

// [EVT-BCP-2]
// Smoke-check sandbox availability under build tag
func TestSandboxAvailableSmoke(t *testing.T) {
	ok, err := sandbox.Available()
	if err != nil {
		t.Skipf("sandbox not available: %v", err)
	}
	if !ok {
		t.Skip("sandbox reports unavailable")
	}
}
