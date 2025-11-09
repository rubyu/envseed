package parser

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

import "envseed/internal/ast"

type ParseError struct {
	Line       int
	Column     int
	Msg        string
	DetailCode string
	DetailArgs []any
}

func (e *ParseError) Error() string {
	return fmt.Sprintf("line %d, column %d: %s", e.Line, e.Column, e.Msg)
}

func newParseError(line, column int, detailCode, message string, args ...any) *ParseError {
	return &ParseError{
		Line:       line,
		Column:     column,
		Msg:        message,
		DetailCode: detailCode,
		DetailArgs: args,
	}
}

type parseIssue struct {
	detailCode string
	message    string
	detailArgs []any
}

func newParseIssue(code, message string, args ...any) *parseIssue {
	return &parseIssue{
		detailCode: code,
		message:    message,
		detailArgs: args,
	}
}

type parser struct {
	src  string
	pos  int
	line int
	col  int
}

type scanner struct {
	src  string
	pos  int
	line int
	col  int
}

func Parse(input string) ([]ast.Element, error) {
	p := parser{src: input, line: 1, col: 1}
	var elems []ast.Element
	for {
		// Detect non-ASCII whitespace at line start (spec 4.5 / EVE-103-3)
		if p.col == 1 && !p.eof() {
			s := p.newScanner()
			r, _ := s.peek()
			if r != '\r' && r != '\n' {
				if unicode.IsSpace(r) && r != ' ' && r != '\t' {
					return nil, newParseError(p.line, p.col, "EVE-103-3", "non-ASCII whitespace at line start")
				}
			}
		}
		if p.eof() {
			break
		}
		if elem, ok := p.consumeBlank(); ok {
			elems = append(elems, elem)
			continue
		}
		if elem, ok := p.consumeComment(); ok {
			elems = append(elems, elem)
			continue
		}
		elem, err := p.consumeAssignment()
		if err != nil {
			return nil, err
		}
		elems = append(elems, elem)
	}
	return elems, nil
}

func (p *parser) eof() bool {
	return p.pos >= len(p.src)
}

func (p *parser) newScanner() scanner {
	return scanner{src: p.src, pos: p.pos, line: p.line, col: p.col}
}

func (p *parser) commit(s scanner) {
	p.pos = s.pos
	p.line = s.line
	p.col = s.col
}

func (p *parser) consumeBlank() (ast.Element, bool) {
	if p.col != 1 {
		return ast.Element{}, false
	}
	s := p.newScanner()
	startLine := s.line
	startPos := s.pos

	for !s.eof() {
		r, size := s.peek()
		if r == '\r' {
			s.advance(size)
			continue
		}
		if r == ' ' || r == '\t' {
			s.advance(size)
			continue
		}
		if r == '\n' {
			s.advance(size)
			p.commit(s)
			text := p.src[startPos:p.pos]
			return ast.Element{
				Type:               ast.ElementBlank,
				Text:               text,
				Line:               startLine,
				ColumnStart:        1,
				HasTrailingNewline: strings.HasSuffix(text, "\n"),
			}, true
		}
		break
	}
	return ast.Element{}, false
}

func (p *parser) consumeComment() (ast.Element, bool) {
	if p.col != 1 {
		return ast.Element{}, false
	}
	s := p.newScanner()
	startLine := s.line
	startPos := s.pos

	for !s.eof() {
		r, size := s.peek()
		if r == '\r' {
			s.advance(size)
			continue
		}
		if r == ' ' || r == '\t' {
			s.advance(size)
			continue
		}
		if r == '#' {
			s.advance(size)
			for !s.eof() {
				r2, size2 := s.peek()
				if r2 == '\n' {
					break
				}
				s.advance(size2)
			}
			text := p.src[startPos:s.pos]
			hasNewline := false
			if !s.eof() {
				r2, size2 := s.peek()
				if r2 == '\n' {
					s.advance(size2)
					hasNewline = true
				}
			}
			p.commit(s)
			return ast.Element{
				Type:               ast.ElementComment,
				Text:               text,
				Line:               startLine,
				ColumnStart:        1,
				HasTrailingNewline: hasNewline,
			}, true
		}
		break
	}
	return ast.Element{}, false
}

func (p *parser) consumeAssignment() (ast.Element, error) {
	s := p.newScanner()
	startPos := s.pos
	startLine := s.line
	startCol := s.col

	for !s.eof() {
		r, size := s.peek()
		if r == ' ' || r == '\t' || r == '\r' {
			s.advance(size)
			continue
		}
		break
	}
	leadingWhitespace := ""
	if s.pos > startPos {
		leadingWhitespace = p.src[startPos:s.pos]
	}

	name, op, err := scanAssignmentName(&s)
	if err != nil {
		return ast.Element{}, err
	}
	if name == "" {
		return ast.Element{}, newParseError(startLine, startCol, "EVE-103-101", "empty assignment name")
	}

	tokens, trailingComment, hasNewline, valueErr := scanValue(&s)
	if valueErr != nil {
		return ast.Element{}, valueErr
	}

	endPos := s.pos
	raw := p.src[startPos:endPos]

	p.commit(s)

	assign := &ast.Assignment{
		Name:               name,
		Operator:           op,
		LeadingWhitespace:  leadingWhitespace,
		Raw:                raw,
		ValueTokens:        tokens,
		TrailingComment:    trailingComment,
		Line:               startLine,
		Column:             startCol,
		HasTrailingNewline: hasNewline,
	}

	return ast.Element{
		Type:               ast.ElementAssignment,
		Assignment:         assign,
		Line:               startLine,
		ColumnStart:        startCol,
		HasTrailingNewline: hasNewline,
	}, nil
}

func scanAssignmentName(s *scanner) (string, ast.AssignmentOperator, error) {
	startPos := s.pos
	startLine := s.line
	startCol := s.col
	if s.eof() {
		return "", ast.OperatorAssign, newParseError(startLine, startCol, "EVE-103-103", "expected assignment")
	}

	r, size := s.peek()
	if !isNameStart(r) {
		return "", ast.OperatorAssign, newParseError(startLine, startCol, "EVE-103-101", "invalid assignment name")
	}
	s.advance(size)

	bracketDepth := 0
	for !s.eof() {
		r, size := s.peek()
		if r == '=' && bracketDepth == 0 {
			name := s.src[startPos:s.pos]
			op := ast.OperatorAssign
			if strings.HasSuffix(name, "+") {
				op = ast.OperatorAppend
				name = name[:len(name)-1]
				if name == "" {
					return "", ast.OperatorAssign, newParseError(startLine, startCol, "EVE-103-101", "invalid assignment name")
				}
			}
			s.advance(size)
			return name, op, nil
		}
		if r == '\n' {
			if bracketDepth > 0 {
				return "", ast.OperatorAssign, newParseError(startLine, startCol, "EVE-103-501", "mismatched brackets in assignment name")
			}
			return "", ast.OperatorAssign, newParseError(startLine, startCol, "EVE-103-102", "assignment missing '='")
		}
		if bracketDepth == 0 {
			if r == '[' {
				bracketDepth++
				s.advance(size)
				continue
			}
			if r == '+' {
				nextRune, _ := s.peekAhead(size)
				if nextRune != '=' {
					return "", ast.OperatorAssign, newParseError(s.line, s.col, "EVE-103-101", "invalid character '+' in assignment name")
				}
				s.advance(size)
				continue
			}
			if !isNameChar(r) {
				return "", ast.OperatorAssign, newParseError(s.line, s.col, "EVE-103-101", fmt.Sprintf("invalid character %q in assignment name", r), r)
			}
			s.advance(size)
			continue
		}
		if r == '[' {
			bracketDepth++
			s.advance(size)
			continue
		}
		if r == ']' {
			if bracketDepth == 0 {
				return "", ast.OperatorAssign, newParseError(s.line, s.col, "EVE-103-502", "unexpected ']'")
			}
			bracketDepth--
			s.advance(size)
			continue
		}
		if r == '\n' {
			if bracketDepth > 0 {
				return "", ast.OperatorAssign, newParseError(startLine, startCol, "EVE-103-501", "mismatched brackets in assignment name")
			}
			return "", ast.OperatorAssign, newParseError(startLine, startCol, "EVE-103-102", "assignment missing '='")
		}
		s.advance(size)
	}
	return "", ast.OperatorAssign, newParseError(startLine, startCol, "EVE-103-102", "assignment missing '='")
}

type frameKind int

const (
	frameBare frameKind = iota
	frameDouble
	frameSingle
	frameBacktick
	frameCommand
)

type frame struct {
	kind       frameKind
	parenDepth int
}

func scanValue(s *scanner) ([]ast.ValueToken, string, bool, error) {
	stack := []frame{{kind: frameBare}}
	var tokens []ast.ValueToken
	var literal strings.Builder
	var literalCtx ast.ValueContext
	hasLiteral := false
	escaped := false
	var trailingComment string
	hasTrailingNewline := false

	flushLiteral := func() {
		if !hasLiteral {
			return
		}
		tokens = append(tokens, ast.ValueToken{Kind: ast.ValueLiteral, Text: literal.String(), Context: literalCtx})
		literal.Reset()
		hasLiteral = false
	}

	appendLiteral := func(r rune, ctx ast.ValueContext) {
		if !hasLiteral {
			literalCtx = ctx
			hasLiteral = true
		} else if literalCtx != ctx {
			flushLiteral()
			literalCtx = ctx
			hasLiteral = true
		}
		literal.WriteRune(r)
	}

	for !s.eof() {
		r, size := s.peek()
		if r == '\r' {
			s.advance(size)
			continue
		}

		ctx := currentContext(stack)

		if r == '\n' {
			if escaped {
				appendLiteral(r, ctx)
				s.advance(size)
				escaped = false
				continue
			}
			if len(stack) == 1 {
				s.advance(size)
				hasTrailingNewline = true
				break
			}
			appendLiteral(r, ctx)
			s.advance(size)
			continue
		}

		if !escaped && ctx != ast.ContextSingleQuoted && len(stack) == 1 && r == '#' {
			flushLiteral()
			commentStart := s.pos
			for !s.eof() {
				r2, size2 := s.peek()
				if r2 == '\n' {
					break
				}
				s.advance(size2)
			}
			trailingComment = s.src[commentStart:s.pos]
			if !s.eof() {
				r2, size2 := s.peek()
				if r2 == '\n' {
					s.advance(size2)
					hasTrailingNewline = true
				}
			}
			return tokens, trailingComment, hasTrailingNewline, nil
		}

		if !escaped && ctx != ast.ContextSingleQuoted && ctx != ast.ContextBacktick && r == '$' {
			nextRune, nextSize := s.peekAhead(size)
			if nextRune == '(' {
				appendLiteral(r, ctx)
				s.advance(size)
				ctx = currentContext(stack)
				appendLiteral('(', ctx)
				s.advance(nextSize)
				stack = append(stack, frame{kind: frameCommand, parenDepth: 1})
				escaped = false
				continue
			}
		}

		if !escaped {
			if strings.HasPrefix(s.src[s.pos:], "<pass") {
				check := *s
				check.advance(len("<pass"))
				if !check.eof() {
					if r2, _ := check.peek(); r2 == ' ' || r2 == '\t' || r2 == '\n' || r2 == '\r' {
						return nil, "", false, newParseError(check.line, check.col, "EVE-103-4", "sigil violation: whitespace between 'pass' and ':'")
					}
				}
			}
			placeholderLine := s.line
			placeholderCol := s.col
			path, modifiers, length, ok, issue := scanPlaceholderLiteral(s.src, s.pos)
			if issue != nil {
				return nil, "", false, newParseError(s.line, s.col, issue.detailCode, issue.message, issue.detailArgs...)
			}
			if ok {
				flushLiteral()
				raw := s.src[s.pos : s.pos+length]
				tokens = append(tokens, ast.ValueToken{
					Kind:      ast.ValuePlaceholder,
					Text:      raw,
					Path:      path,
					Modifiers: modifiers,
					Context:   ctx,
					Line:      placeholderLine,
					Column:    placeholderCol,
				})
				s.advance(length)
				escaped = false
				continue
			}
		}

		appendLiteral(r, ctx)
		s.advance(size)

		if escaped {
			escaped = false
			continue
		}

		switch r {
		case '\\':
			if ctx != ast.ContextSingleQuoted {
				escaped = true
			}
		case '"':
			if topKind(stack) == frameDouble {
				stack = stack[:len(stack)-1]
			} else if topKind(stack) != frameSingle && topKind(stack) != frameBacktick {
				stack = append(stack, frame{kind: frameDouble})
			}
		case '\'':
			if topKind(stack) == frameSingle {
				stack = stack[:len(stack)-1]
			} else if topKind(stack) != frameDouble && topKind(stack) != frameBacktick {
				stack = append(stack, frame{kind: frameSingle})
			}
		case '`':
			if topKind(stack) == frameBacktick {
				stack = stack[:len(stack)-1]
			} else if topKind(stack) != frameSingle {
				stack = append(stack, frame{kind: frameBacktick})
			}
		case '(':
			if topKind(stack) == frameCommand {
				stack[len(stack)-1].parenDepth++
			}
		case ')':
			if topKind(stack) == frameCommand {
				stack[len(stack)-1].parenDepth--
				if stack[len(stack)-1].parenDepth <= 0 {
					stack = stack[:len(stack)-1]
				}
			}
		}
	}

	if len(stack) > 1 {
		kind := topKind(stack)
		msg := "unterminated construct"
		code := "EVE-103-401"
		switch kind {
		case frameDouble:
			msg = "unterminated double quote"
			code = "EVE-103-401"
		case frameSingle:
			msg = "unterminated single quote"
			code = "EVE-103-402"
		case frameBacktick:
			msg = "unterminated backtick substitution"
			code = "EVE-103-403"
		case frameCommand:
			msg = "unterminated command substitution"
			code = "EVE-103-404"
		}
		return nil, "", false, newParseError(s.line, s.col, code, msg)
	}

	flushLiteral()
	return tokens, trailingComment, hasTrailingNewline, nil
}

func currentContext(stack []frame) ast.ValueContext {
	for i := len(stack) - 1; i >= 0; i-- {
		switch stack[i].kind {
		case frameDouble:
			return ast.ContextDoubleQuoted
		case frameSingle:
			return ast.ContextSingleQuoted
		case frameBacktick:
			return ast.ContextBacktick
		}
	}
	for i := len(stack) - 1; i >= 0; i-- {
		if stack[i].kind == frameCommand {
			return ast.ContextCommandSubstitution
		}
	}
	return ast.ContextBare
}

func topKind(stack []frame) frameKind {
	if len(stack) == 0 {
		return frameBare
	}
	return stack[len(stack)-1].kind
}

var validModifiers = map[string]struct{}{
	"dangerously_bypass_escape": {},
	"allow_newline":             {},
	"allow_tab":                 {},
	"base64":                    {},
	"strip":                     {},
	"strip_left":                {},
	"strip_right":               {},
}

func trimASCIIWhitespace(segment, subject, detailCode string) (string, *parseIssue) {
	start := 0
	end := len(segment)
	for start < end {
		r, size := utf8.DecodeRuneInString(segment[start:])
		if r == ' ' || r == '\t' {
			start += size
			continue
		}
		if unicode.IsSpace(r) {
			return "", newParseIssue(detailCode, fmt.Sprintf("%s contains non-ASCII whitespace", subject))
		}
		break
	}
	for end > start {
		r, size := utf8.DecodeLastRuneInString(segment[:end])
		if r == ' ' || r == '\t' {
			end -= size
			continue
		}
		if unicode.IsSpace(r) {
			return "", newParseIssue(detailCode, fmt.Sprintf("%s contains non-ASCII whitespace", subject))
		}
		break
	}
	return segment[start:end], nil
}

// isNonASCIISpace reports Unicode whitespace other than ASCII SPACE/TAB.
func isNonASCIISpace(r rune) bool {
	return unicode.IsSpace(r) && r != ' ' && r != '\t'
}

func scanPlaceholderLiteral(src string, start int) (string, []string, int, bool, *parseIssue) {
	if start >= len(src) {
		return "", nil, 0, false, nil
	}
	// Sigil detection: "<pass" followed by whitespace before ':' is a violation (EVE-103-4)
	if strings.HasPrefix(src[start:], "<pass") && !strings.HasPrefix(src[start:], "<pass:") {
		j := start + len("<pass")
		if j < len(src) {
			r := rune(src[j])
			if unicode.IsSpace(r) {
				return "", nil, 0, false, newParseIssue("EVE-103-4", "whitespace between 'pass' and ':' (sigil violation)")
			}
		}
	}
	if !strings.HasPrefix(src[start:], "<pass:") {
		return "", nil, 0, false, nil
	}
	i := start + len("<pass:")
	if i >= len(src) {
		return "", nil, 0, false, newParseIssue("EVE-103-202", "unterminated placeholder")
	}
	pathStart := i
	for i < len(src) {
		c := src[i]
		switch c {
		case '>':
			if i == pathStart {
				return "", nil, 0, false, newParseIssue("EVE-103-201", "empty placeholder path")
			}
			segment := src[pathStart:i]
			path, issue := trimASCIIWhitespace(segment, "placeholder path", "EVE-103-204")
			if issue != nil {
				return "", nil, 0, false, issue
			}
			if path == "" {
				return "", nil, 0, false, newParseIssue("EVE-103-201", "empty placeholder path")
			}
			if strings.IndexByte(path, 0) >= 0 {
				return "", nil, 0, false, newParseIssue("EVE-103-203", "placeholder path contains NUL")
			}
			return path, nil, i - start + 1, true, nil
		case '|':
			if i == pathStart {
				return "", nil, 0, false, newParseIssue("EVE-103-201", "empty placeholder path")
			}
			segment := src[pathStart:i]
			path, issue := trimASCIIWhitespace(segment, "placeholder path", "EVE-103-204")
			if issue != nil {
				return "", nil, 0, false, issue
			}
			if path == "" {
				return "", nil, 0, false, newParseIssue("EVE-103-201", "empty placeholder path")
			}
			if strings.IndexByte(path, 0) >= 0 {
				return "", nil, 0, false, newParseIssue("EVE-103-203", "placeholder path contains NUL")
			}
			modStart := i + 1
			j := modStart
			for j < len(src) && src[j] != '>' && src[j] != '\n' {
				j++
			}
			if j >= len(src) || src[j] != '>' {
				return "", nil, 0, false, newParseIssue("EVE-103-202", "unterminated placeholder")
			}
			// Separator-adjacent non-ASCII whitespace checks (EVE-103-1)
			//  - immediately after '|'
			if modStart < len(src) {
				r, _ := utf8.DecodeRuneInString(src[modStart:])
				if isNonASCIISpace(r) {
					return "", nil, 0, false, newParseIssue("EVE-103-1", "non-ASCII whitespace around placeholder separators or before >")
				}
			}
			//  - around commas inside modifiers
			{
				k := modStart
				for k < j {
					r, size := utf8.DecodeRuneInString(src[k:])
					if r == ',' {
						// check previous
						if k > modStart {
							pr, _ := utf8.DecodeLastRuneInString(src[modStart:k])
							if isNonASCIISpace(pr) {
								return "", nil, 0, false, newParseIssue("EVE-103-1", "non-ASCII whitespace around placeholder separators or before >")
							}
						}
						// check next
						if k+size < j {
							nr, _ := utf8.DecodeRuneInString(src[k+size:])
							if isNonASCIISpace(nr) {
								return "", nil, 0, false, newParseIssue("EVE-103-1", "non-ASCII whitespace around placeholder separators or before >")
							}
						}
					}
					k += size
				}
			}
			//  - immediately before '>'
			if j > modStart {
				pr, _ := utf8.DecodeLastRuneInString(src[modStart:j])
				if isNonASCIISpace(pr) {
					return "", nil, 0, false, newParseIssue("EVE-103-1", "non-ASCII whitespace around placeholder separators or before >")
				}
			}
			modifiers, issue := parseModifiers(src[modStart:j])
			if issue != nil {
				return "", nil, 0, false, issue
			}
			if len(modifiers) == 0 {
				return "", nil, 0, false, newParseIssue("EVE-103-301", "placeholder modifiers missing")
			}
			return path, modifiers, j - start + 1, true, nil
		case '\n':
			return "", nil, 0, false, newParseIssue("EVE-103-202", "unterminated placeholder")
		default:
			i++
		}
	}
	return "", nil, 0, false, newParseIssue("EVE-103-202", "unterminated placeholder")
}

func parseModifiers(section string) ([]string, *parseIssue) {
	trimmed, issue := trimASCIIWhitespace(section, "placeholder modifiers", "EVE-103-305")
	if issue != nil {
		return nil, issue
	}
	if strings.IndexByte(trimmed, 0) >= 0 {
		return nil, newParseIssue("EVE-103-305", "placeholder modifiers contain NUL")
	}
	if trimmed == "" {
		return nil, newParseIssue("EVE-103-301", "placeholder modifiers missing")
	}
	parts := strings.Split(trimmed, ",")
	var modifiers []string
	seen := make(map[string]struct{})
	for _, part := range parts {
		mod, issue := trimASCIIWhitespace(part, "placeholder modifiers", "EVE-103-305")
		if issue != nil {
			return nil, issue
		}
		if mod == "" {
			return nil, newParseIssue("EVE-103-304", "invalid empty modifier")
		}
		if _, ok := validModifiers[mod]; !ok {
			return nil, newParseIssue("EVE-103-302", fmt.Sprintf("unknown placeholder modifier %q", mod), mod)
		}
		if _, exists := seen[mod]; exists {
			return nil, newParseIssue("EVE-103-303", fmt.Sprintf("duplicate placeholder modifier %q", mod), mod)
		}
		modifiers = append(modifiers, mod)
		seen[mod] = struct{}{}
	}
	return modifiers, nil
}

func (s *scanner) eof() bool {
	return s.pos >= len(s.src)
}

func (s *scanner) peek() (rune, int) {
	if s.eof() {
		return 0, 0
	}
	r, size := utf8.DecodeRuneInString(s.src[s.pos:])
	return r, size
}

func (s *scanner) peekAhead(offset int) (rune, int) {
	idx := s.pos + offset
	if idx >= len(s.src) {
		return 0, 0
	}
	r, size := utf8.DecodeRuneInString(s.src[idx:])
	return r, size
}

func (s *scanner) advance(n int) {
	for n > 0 && s.pos < len(s.src) {
		r, size := utf8.DecodeRuneInString(s.src[s.pos:])
		s.pos += size
		n -= size
		if r == '\n' {
			s.line++
			s.col = 1
		} else {
			s.col++
		}
	}
}

func isNameStart(r rune) bool {
	return r == '_' || unicode.IsLetter(r)
}

func isNameChar(r rune) bool {
	return r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r)
}
