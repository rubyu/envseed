package parser_test

import (
	"strings"
	"testing"

	"envseed/internal/ast"
	parser "envseed/internal/parser"
	"envseed/internal/testsupport"
)

// [EVT-MGU-1]
func TestParse_AssignmentWithPlaceholders(t *testing.T) {
	input := "  URL=http://<pass:host>/v1/<pass:key|dangerously_bypass_escape>\n"
	elems, err := parser.Parse(input)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if len(elems) != 1 {
		t.Fatalf("expected 1 element, got %d", len(elems))
	}
	elem := elems[0]
	if elem.Type != ast.ElementAssignment {
		t.Fatalf("expected assignment element, got %v", elem.Type)
	}
	assign := elem.Assignment
	if assign.Name != "URL" {
		t.Fatalf("assignment name = %s, want URL", assign.Name)
	}
	if assign.LeadingWhitespace != "  " {
		t.Fatalf("leading whitespace = %q, want two spaces", assign.LeadingWhitespace)
	}
	tokens := assign.ValueTokens
	if len(tokens) != 4 {
		t.Fatalf("expected 4 tokens, got %d", len(tokens))
	}
	if tokens[0].Kind != ast.ValueLiteral || tokens[0].Text != "http://" {
		t.Fatalf("unexpected first token: %#v", tokens[0])
	}
	if tokens[1].Kind != ast.ValuePlaceholder || tokens[1].Path != "host" || tokens[1].Context != ast.ContextBare {
		t.Fatalf("unexpected second token: %#v", tokens[1])
	}
	if len(tokens[1].Modifiers) != 0 {
		t.Fatalf("unexpected modifiers for host: %#v", tokens[1].Modifiers)
	}
	if tokens[2].Kind != ast.ValueLiteral || tokens[2].Text != "/v1/" {
		t.Fatalf("unexpected third token: %#v", tokens[2])
	}
	if tokens[3].Kind != ast.ValuePlaceholder || tokens[3].Path != "key" {
		t.Fatalf("unexpected fourth token: %#v", tokens[3])
	}
	if want := []string{"dangerously_bypass_escape"}; !testsupport.EqualStrings(tokens[3].Modifiers, want) {
		t.Fatalf("modifiers = %#v, want %#v", tokens[3].Modifiers, want)
	}
	if !assign.HasTrailingNewline {
		t.Fatalf("assignment should report trailing newline")
	}
}

// [EVT-MPU-1]
func TestParse_ModifiersWithWhitespace(t *testing.T) {
	input := "VAL=<pass:path | allow_tab , allow_newline>\n"
	elems, err := parser.Parse(input)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	mods := elems[0].Assignment.ValueTokens[0].Modifiers
	want := []string{"allow_tab", "allow_newline"}
	if !testsupport.EqualStrings(mods, want) {
		t.Fatalf("modifiers = %#v, want %#v", mods, want)
	}
}

// [EVT-MGU-1]
func TestParse_DoubleQuotedPlaceholderContext(t *testing.T) {
	input := "GREETING=\"Hello <pass:name>!\"\n"
	elems, err := parser.Parse(input)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	placeholder := findFirstPlaceholder(elems[0].Assignment.ValueTokens)
	if placeholder == nil {
		t.Fatalf("expected placeholder token")
	}
	if placeholder.Context != ast.ContextDoubleQuoted {
		t.Fatalf("placeholder context = %v, want double quoted", placeholder.Context)
	}
}

// [EVT-MGU-1]
func TestParse_CommandSubstitutionPlaceholderContext(t *testing.T) {
	input := "CMD=$(echo <pass:secret>)\n"
	elems, err := parser.Parse(input)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	placeholder := findFirstPlaceholder(elems[0].Assignment.ValueTokens)
	if placeholder == nil {
		t.Fatalf("expected placeholder token")
	}
	if placeholder.Context != ast.ContextCommandSubstitution {
		t.Fatalf("placeholder context = %v, want command substitution", placeholder.Context)
	}
}

// [EVT-MGU-1]
func TestParse_BacktickPlaceholderContext(t *testing.T) {
	input := "STAMP=`echo <pass:token>`\n"
	elems, err := parser.Parse(input)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	placeholder := findFirstPlaceholder(elems[0].Assignment.ValueTokens)
	if placeholder == nil {
		t.Fatalf("expected placeholder token")
	}
	if placeholder.Context != ast.ContextBacktick {
		t.Fatalf("placeholder context = %v, want backtick", placeholder.Context)
	}
}

// [EVT-MGU-1]
func TestParse_SingleQuotedLiteralPlaceholder(t *testing.T) {
	input := "RAW='<pass:secret>'\n"
	elems, err := parser.Parse(input)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	tokens := elems[0].Assignment.ValueTokens
	if len(tokens) != 3 {
		t.Fatalf("expected three tokens (quote, placeholder, quote), got %#v", tokens)
	}
	if tokens[1].Kind != ast.ValuePlaceholder || tokens[1].Context != ast.ContextSingleQuoted {
		t.Fatalf("expected placeholder token in single quoted context, got %#v", tokens[1])
	}
}

// [EVT-MGU-1][EVT-MGU-4]
func TestParse_TrailingCommentPreserved(t *testing.T) {
	input := "KEY=value # trailing\n"
	elems, err := parser.Parse(input)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	assign := elems[0].Assignment
	if assign.TrailingComment != "# trailing" {
		t.Fatalf("trailing comment = %q, want %q", assign.TrailingComment, "# trailing")
	}
	tokens := assign.ValueTokens
	if len(tokens) != 1 || tokens[0].Text != "value " {
		t.Fatalf("unexpected value tokens: %#v", tokens)
	}
}

// [EVT-MGU-1]
func TestParse_BlankAndCommentElements(t *testing.T) {
	input := "# comment\n\nEXAMPLE_VAR=bar\n"
	elems, err := parser.Parse(input)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if len(elems) != 3 {
		t.Fatalf("expected 3 elements, got %d", len(elems))
	}
	if elems[0].Type != ast.ElementComment || !elems[0].HasTrailingNewline {
		t.Fatalf("first element should be comment with newline")
	}
	if elems[1].Type != ast.ElementBlank {
		t.Fatalf("second element type = %v, want blank", elems[1].Type)
	}
	if elems[2].Type != ast.ElementAssignment {
		t.Fatalf("third element type = %v, want assignment", elems[2].Type)
	}
}

// [EVT-MGU-1]
func TestParse_MultilineDoubleQuote(t *testing.T) {
	input := "NOTE=\"line1\nline2\"\nNEXT=ok\n"
	elems, err := parser.Parse(input)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if len(elems) != 2 {
		t.Fatalf("expected 2 elements, got %d", len(elems))
	}
	text := joinTokenText(elems[0].Assignment.ValueTokens)
	if !strings.Contains(text, "line2") {
		t.Fatalf("multiline content missing line2: %q", text)
	}
}

// [EVT-MGU-1]
func TestParse_InvalidName(t *testing.T) {
	input := "1INVALID=value\n"
	_, err := parser.Parse(input)
	expectParseError(t, err, "EVE-103-101")
}

// [EVT-MGU-1]
func TestParse_UnterminatedDoubleQuote(t *testing.T) {
	input := "BROKEN=\"value\n"
	_, err := parser.Parse(input)
	expectParseError(t, err, "EVE-103-401")
}

// [EVT-MGU-1]
func TestParse_UnterminatedCommandSubstitution(t *testing.T) {
	input := "BROKEN=$(echo value\n"
	_, err := parser.Parse(input)
	expectParseError(t, err, "EVE-103-404")
}

// [EVT-MPU-4]
func TestParse_AllowsModifierCombination(t *testing.T) {
	t.Helper()
	cases := []struct {
		name  string
		input string
		want  []string
	}{
		{
			name:  "dangerously_with_allow_tab",
			input: "VAL=<pass:path|dangerously_bypass_escape,allow_tab>\n",
			want:  []string{"dangerously_bypass_escape", "allow_tab"},
		},
		{
			name:  "base64_with_allow_tab",
			input: "VAL=<pass:path|base64,allow_tab>\n",
			want:  []string{"base64", "allow_tab"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			elems, err := parser.Parse(tc.input)
			if err != nil {
				t.Fatalf("Parse error: %v", err)
			}
			mods := elems[0].Assignment.ValueTokens[0].Modifiers
			if !testsupport.EqualStrings(mods, tc.want) {
				t.Fatalf("modifiers = %#v, want %#v", mods, tc.want)
			}
		})
	}
}

// [EVT-MPU-1]
func TestParse_DuplicateOrUnknownModifier(t *testing.T) {
	cases := []struct {
		input string
		code  string
	}{
		{input: "KEY=<pass:path|allow_tab,allow_tab>\n", code: "EVE-103-303"},
		{input: "KEY=<pass:path|unknown_mod>\n", code: "EVE-103-302"},
		{input: "KEY=<pass:path|>\n", code: "EVE-103-301"},
	}
	for _, tc := range cases {
		_, err := parser.Parse(tc.input)
		expectParseError(t, err, tc.code)
	}
}

// [EVT-MPU-1]
func TestParse_AllowModifiersSpacing(t *testing.T) {
	input := "KEY=<pass:path| allow_tab ,allow_newline >\n"
	elems, err := parser.Parse(input)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	mods := elems[0].Assignment.ValueTokens[0].Modifiers
	want := []string{"allow_tab", "allow_newline"}
	if !testsupport.EqualStrings(mods, want) {
		t.Fatalf("modifiers = %#v, want %#v", mods, want)
	}
}

// [EVT-MPU-1]
func TestParse_PlaceholderDisallowsNewlineInBody(t *testing.T) {
	input := "SECRET=<pass:path\nNEXT=value\n"
	_, err := parser.Parse(input)
	perr := expectParseError(t, err, "EVE-103-202")
	if perr.Line != 1 || perr.Column != 8 {
		t.Fatalf("error position = (%d, %d), want (1, 8)", perr.Line, perr.Column)
	}
}

// [EVT-MPU-1][EVT-MUU-1]
func TestParse_PlaceholderDisallowsNULPath(t *testing.T) {
	input := "SECRET=<pass:pa\x00th>\n"
	_, err := parser.Parse(input)
	perr := expectParseError(t, err, "EVE-103-203")
	if perr.Line != 1 || perr.Column != 8 {
		t.Fatalf("error position = (%d, %d), want (1, 8)", perr.Line, perr.Column)
	}
}

// [EVT-MPU-3]
func TestParse_ModifierNamesAreCaseSensitive(t *testing.T) {
	input := "SECRET=<pass:path|Allow_tab>\n"
	_, err := parser.Parse(input)
	perr := expectParseError(t, err, "EVE-103-302")
	if perr.Line != 1 || perr.Column != 8 {
		t.Fatalf("error position = (%d, %d), want (1, 8)", perr.Line, perr.Column)
	}
}

// [EVT-MGU-1][EVT-MWP-1]
func TestParse_AssignmentNewlineVariants(t *testing.T) {
	t.Helper()
	input := strings.Join([]string{
		"A=value\n",
		"B=value\r\n",
		"C=value",
	}, "")
	elems, err := parser.Parse(input)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if len(elems) != 3 {
		t.Fatalf("expected 3 elements, got %d", len(elems))
	}
	check := func(t *testing.T, elem ast.Element, wantNewline bool, contains string) {
		t.Helper()
		assign := elem.Assignment
		if assign == nil {
			t.Fatalf("expected assignment element")
		}
		if assign.HasTrailingNewline != wantNewline {
			t.Fatalf("assignment HasTrailingNewline = %v, want %v", assign.HasTrailingNewline, wantNewline)
		}
		if elem.HasTrailingNewline != wantNewline {
			t.Fatalf("element HasTrailingNewline = %v, want %v", elem.HasTrailingNewline, wantNewline)
		}
		if contains != "" && !strings.Contains(assign.Raw, contains) {
			t.Fatalf("raw assignment %q missing %q", assign.Raw, contains)
		}
	}
	t.Run("LF", func(t *testing.T) { check(t, elems[0], true, "\n") })
	t.Run("CRLF", func(t *testing.T) { check(t, elems[1], true, "\r\n") })
	t.Run("NoNewline", func(t *testing.T) { check(t, elems[2], false, "") })
}

// [EVT-MGU-3]
func TestParse_AdditiveAndIndexedAssignments(t *testing.T) {
	input := strings.Join([]string{
		`TOTAL+=<pass:sum>`,
		`ARRAY[1]="value"`,
	}, "\n") + "\n"
	elems, err := parser.Parse(input)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if len(elems) != 2 {
		t.Fatalf("expected 2 assignments, got %d", len(elems))
	}
	total := elems[0].Assignment
	if total.Operator != ast.OperatorAppend {
		t.Fatalf("TOTAL operator = %v, want ast.OperatorAppend", total.Operator)
	}
	if total.Name != "TOTAL" {
		t.Fatalf("TOTAL name = %q, want TOTAL", total.Name)
	}
	if len(total.ValueTokens) != 1 || total.ValueTokens[0].Kind != ast.ValuePlaceholder {
		t.Fatalf("TOTAL tokens = %#v, want single placeholder", total.ValueTokens)
	}
	array := elems[1].Assignment
	if array.Operator != ast.OperatorAssign {
		t.Fatalf("ARRAY operator = %v, want ast.OperatorAssign", array.Operator)
	}
	if array.Name != "ARRAY[1]" {
		t.Fatalf("ARRAY name = %q, want ARRAY[1]", array.Name)
	}
	if got := joinTokenText(array.ValueTokens); got != `"value"` {
		t.Fatalf("ARRAY tokens combined = %q, want %q", got, `"value"`)
	}
}

// [EVT-MGU-5]
func TestParse_AssignmentOperatorAdjacency(t *testing.T) {
	t.Helper()
	cases := []struct {
		name     string
		input    string
		wantOps  ast.AssignmentOperator
		wantText []string
	}{
		{
			name:     "AssignPlaceholderLiteral",
			input:    `VAL=<pass:a>suffix`,
			wantOps:  ast.OperatorAssign,
			wantText: []string{"<pass:a>", "suffix"},
		},
		{
			name:     "AppendLiteralPlaceholder",
			input:    `LOG+=prefix<pass:entry>`,
			wantOps:  ast.OperatorAppend,
			wantText: []string{"prefix", "<pass:entry>"},
		},
		{
			name:     "IndexedPlaceholderAdjacency",
			input:    `MAP[0]=<pass:key><pass:value>`,
			wantOps:  ast.OperatorAssign,
			wantText: []string{"<pass:key>", "<pass:value>"},
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			elems, err := parser.Parse(tc.input + "\n")
			if err != nil {
				t.Fatalf("Parse error: %v", err)
			}
			if len(elems) != 1 {
				t.Fatalf("expected single assignment, got %d", len(elems))
			}
			assign := elems[0].Assignment
			if assign.Operator != tc.wantOps {
				t.Fatalf("operator = %v, want %v", assign.Operator, tc.wantOps)
			}
			var got []string
			for _, tok := range assign.ValueTokens {
				got = append(got, tok.Text)
			}
			if !testsupport.EqualStrings(got, tc.wantText) {
				t.Fatalf("token text = %#v, want %#v", got, tc.wantText)
			}
		})
	}
}

// [EVT-MGU-2]
func TestParse_ErrorReportsLineColumn(t *testing.T) {
	t.Helper()
	input := strings.Join([]string{
		"OK=value",
		"BAD NAME=value",
		"NEXT=\"unterminated",
	}, "\n") + "\n"
	_, err := parser.Parse(input)
	perr := expectParseError(t, err, "EVE-103-101")
	if perr.Line != 2 {
		t.Fatalf("error line = %d, want 2", perr.Line)
	}
	if perr.Column != 4 {
		t.Fatalf("error column = %d, want 4", perr.Column)
	}
}

// [EVT-MPU-2][EVT-MPF-2]
func TestParse_SigilViolationWhitespace(t *testing.T) {
	inputs := []string{
		"VAR=<pass :secret>\n",
		"VAR=<pass\t:secret>\n",
	}
	for _, in := range inputs {
		_, err := parser.Parse(in)
		perr := expectParseError(t, err, "EVE-103-4")
		if perr.Column <= 6 {
			t.Fatalf("error column = %d, want to point at whitespace after 'pass'", perr.Column)
		}
	}
}

// [EVT-MPU-5]
func TestParse_Base64Modifier(t *testing.T) {
	input := "TOKEN=<pass:secret|base64>\n"
	elems, err := parser.Parse(input)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	mods := elems[0].Assignment.ValueTokens[0].Modifiers
	want := []string{"base64"}
	if !testsupport.EqualStrings(mods, want) {
		t.Fatalf("modifiers = %#v, want %#v", mods, want)
	}
}

// [EVT-MPU-5][EVT-MPU-7]
func TestParse_Base64CombinationAllowed(t *testing.T) {
	inputs := []struct {
		name  string
		input string
		want  []string
	}{
		{
			name:  "Base64ThenAllowTab",
			input: "SECRET=<pass:path|base64,allow_tab>\n",
			want:  []string{"base64", "allow_tab"},
		},
		{
			name:  "AllowNewlineThenBase64",
			input: "SECRET=<pass:path|allow_newline,base64>\n",
			want:  []string{"allow_newline", "base64"},
		},
	}
	for _, tc := range inputs {
		t.Run(tc.name, func(t *testing.T) {
			elems, err := parser.Parse(tc.input)
			if err != nil {
				t.Fatalf("Parse error: %v", err)
			}
			mods := elems[0].Assignment.ValueTokens[0].Modifiers
			if !testsupport.EqualStrings(mods, tc.want) {
				t.Fatalf("modifiers = %#v, want %#v", mods, tc.want)
			}
		})
	}
}

// [EVT-MWU-1]
func TestParse_PreservesTrailingWhitespace(t *testing.T) {
	input := "NOTE=value  \n"
	elems, err := parser.Parse(input)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	assign := elems[0].Assignment
	if len(assign.ValueTokens) != 1 {
		t.Fatalf("expected single value token, got %#v", assign.ValueTokens)
	}
	if assign.ValueTokens[0].Text != "value  " {
		t.Fatalf("trailing whitespace lost, got %q", assign.ValueTokens[0].Text)
	}
	if !assign.HasTrailingNewline {
		t.Fatalf("expected trailing newline flag to be true")
	}
}

// [EVT-MPU-1][EVT-MUU-1]
func TestParse_PlaceholderDisallowsNULInModifiers(t *testing.T) {
	input := "VAL=<pass:path|allo\x00w_tab>\n"
	_, err := parser.Parse(input)
	expectParseError(t, err, "EVE-103-305")
}

// [EVT-MPU-1]
func TestParse_PlaceholderRejectsNonASCIIWhitespaceAroundPath(t *testing.T) {
	input := "VAL=<pass:\u00A0secret>\n"
	_, err := parser.Parse(input)
	expectParseError(t, err, "EVE-103-204")
}

// [EVT-MPU-1]
func TestParse_PlaceholderRejectsNonASCIIWhitespaceAroundSeparators_AfterPipe(t *testing.T) {
	input := "VAL=<pass:path|\u00A0allow_tab>\n"
	_, err := parser.Parse(input)
	expectParseError(t, err, "EVE-103-1")
}

// [EVT-MPU-1]
func TestParse_PlaceholderRejectsNonASCIIWhitespaceAroundSeparators_CommaAdjacency(t *testing.T) {
	input := "VAL=<pass:path|allow_tab,\u00A0allow_newline>\n"
	_, err := parser.Parse(input)
	expectParseError(t, err, "EVE-103-1")
}

// [EVT-MPU-1]
func TestParse_PlaceholderRejectsNonASCIIWhitespaceBeforeClose(t *testing.T) {
	input := "VAL=<pass:path|allow_tab\u00A0>\n"
	_, err := parser.Parse(input)
	expectParseError(t, err, "EVE-103-1")
}

// [EVT-MGU-1]
func TestParse_AssignmentNameMismatchedBrackets(t *testing.T) {
	input := "FOO[0=bar\n"
	_, err := parser.Parse(input)
	expectParseError(t, err, "EVE-103-501")
}

func findFirstPlaceholder(tokens []ast.ValueToken) *ast.ValueToken {
	for i := range tokens {
		if tokens[i].Kind == ast.ValuePlaceholder {
			return &tokens[i]
		}
	}
	return nil
}

func joinTokenText(tokens []ast.ValueToken) string {
	var b strings.Builder
	for _, tok := range tokens {
		b.WriteString(tok.Text)
	}
	return b.String()
}

func expectParseError(t *testing.T, err error, code string) *parser.ParseError {
	return testsupport.ExpectErrorAs[*parser.ParseError](t, err, code, func(pe *parser.ParseError) string {
		return pe.DetailCode
	})
}
