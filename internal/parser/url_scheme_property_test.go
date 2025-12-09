package parser_test

import (
	"strings"
	"testing"

	parser "envseed/internal/parser"
	"envseed/internal/testsupport"
)

// [EVT-MPF-2]
func TestParser_NearURLSchemeVariants(t *testing.T) {
	t.Helper()

	type tc struct {
		name     string
		input    string
		wantErr  bool
		detail   string
		contains string
	}

	cases := []tc{
		{
	        name:    "ValidURLScheme",
			input:   "VAR=<pass://ok>\n",
			wantErr: false,
		},
		{
			name:    "LegacyColon",
			input:   "VAR=<pass:legacy>\n",
			wantErr: true,
			detail:  "EVE-103-4",
		},
		{
			name:    "SlashSingle",
			input:   "VAR=<pass/path>\n",
			wantErr: true,
			detail:  "EVE-103-4",
		},
		{
			name:    "AlnumAfterPass",
			input:   "VAR=<passx>\n",
			wantErr: true,
			detail:  "EVE-103-4",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			elems, err := parser.Parse(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected parse error, got success; input=%q", tc.input)
				}
				perr := testsupport.ExpectErrorAs[*parser.ParseError](t, err, tc.detail, func(pe *parser.ParseError) string {
					return pe.DetailCode
				})
				if perr.DetailCode != tc.detail {
					t.Fatalf("detail code = %s, want %s", perr.DetailCode, tc.detail)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected parse error: %v; input=%q", err, tc.input)
			}
			if len(elems) != 1 || elems[0].Assignment == nil {
				t.Fatalf("unexpected AST for input %q: %#v", tc.input, elems)
			}
	            // When successful, ensure placeholder text contains the canonical URL scheme prefix.
	            foundPlaceholder := false
	            for _, tok := range elems[0].Assignment.ValueTokens {
	                if strings.Contains(tok.Text, "<pass://") {
	                    foundPlaceholder = true
	                    break
	                }
	            }
	            if !foundPlaceholder {
	                t.Fatalf("expected placeholder token with '<pass://', input=%q AST=%#v", tc.input, elems)
	            }
		})
	}
}

