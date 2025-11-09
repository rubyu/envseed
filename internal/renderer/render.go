package renderer

import (
	"encoding/base64"
	"fmt"
	"strings"
	"unicode"

	"envseed/internal/ast"
	"envseed/internal/parser"
)

// Resolver resolves placeholder paths to secret values.
type Resolver interface {
	Resolve(path string) (string, error)
}

type renderObserver interface {
	RecordRendered(path, rendered string)
}

// RenderElements renders parsed elements using the provided resolver.
func RenderElements(elements []ast.Element, resolver Resolver) (string, error) {
	var out strings.Builder
	dangerousBypassUsed := false
	for _, elem := range elements {
		switch elem.Type {
		case ast.ElementBlank:
			out.WriteString(elem.Text)
		case ast.ElementComment:
			out.WriteString(elem.Text)
			if elem.HasTrailingNewline {
				out.WriteString("\n")
			}
		case ast.ElementAssignment:
			rendered, dangerous, err := renderAssignment(elem.Assignment, resolver)
			if err != nil {
				return "", err
			}
			if dangerous {
				dangerousBypassUsed = true
			}
			out.WriteString(rendered)
			if elem.Assignment.HasTrailingNewline {
				out.WriteString("\n")
			}
		default:
			return "", fmt.Errorf("unknown element type %v", elem.Type)
		}
	}

	result := out.String()
	if !dangerousBypassUsed {
		if _, err := parser.Parse(result); err != nil {
			return "", &OutputValidationError{Err: err}
		}
	}
	return result, nil
}

// RenderString parses the input, renders it, and validates the output.
func RenderString(input string, resolver Resolver) (string, error) {
	elems, err := parser.Parse(input)
	if err != nil {
		return "", err
	}
	return RenderElements(elems, resolver)
}

// PlaceholderError captures a rendering violation tied to a specific placeholder.
type PlaceholderError struct {
	line       int
	column     int
	path       string
	detailCode string
	detailArgs []any
	message    string
}

func newPlaceholderError(line, column int, path, detailCode, message string, args ...any) *PlaceholderError {
	return &PlaceholderError{
		line:       line,
		column:     column,
		path:       path,
		detailCode: detailCode,
		detailArgs: args,
		message:    fmt.Sprintf(message, args...),
	}
}

// Error implements the error interface.
func (e *PlaceholderError) Error() string {
	if e == nil {
		return ""
	}
	location := fmt.Sprintf("line %d", e.line)
	if e.column > 0 {
		location = fmt.Sprintf("%s, column %d", location, e.column)
	}
	return fmt.Sprintf("%s: placeholder %q: %s", location, e.path, e.message)
}

// DetailCode reports the specification detail code for this error.
func (e *PlaceholderError) DetailCode() string {
	if e == nil {
		return ""
	}
	return e.detailCode
}

// DetailArgs exposes arguments used to format the CLI message.
func (e *PlaceholderError) DetailArgs() []any {
	if e == nil {
		return nil
	}
	return e.detailArgs
}

func renderAssignment(assign *ast.Assignment, resolver Resolver) (string, bool, error) {
	var b strings.Builder
	dangerousBypassUsed := false
	b.WriteString(assign.LeadingWhitespace)
	b.WriteString(assign.Name)
	switch assign.Operator {
	case ast.OperatorAssign:
		b.WriteString("=")
	case ast.OperatorAppend:
		b.WriteString("+=")
	default:
		return "", false, fmt.Errorf("line %d: unsupported operator", assign.Line)
	}

	for _, tok := range assign.ValueTokens {
		switch tok.Kind {
		case ast.ValueLiteral:
			b.WriteString(tok.Text)
		case ast.ValuePlaceholder:
			secret, err := resolver.Resolve(tok.Path)
			if err != nil {
				return "", false, fmt.Errorf("line %d: resolve %q: %w", assign.Line, tok.Path, err)
			}
			rendered, dangerous, err := renderSecret(assign, tok, resolver, secret)
			if err != nil {
				return "", false, err
			}
			if dangerous {
				dangerousBypassUsed = true
			}
			b.WriteString(rendered)
		default:
			return "", false, fmt.Errorf("line %d: unknown token kind", assign.Line)
		}
	}

	if assign.TrailingComment != "" {
		b.WriteString(assign.TrailingComment)
	}
	return b.String(), dangerousBypassUsed, nil
}

func renderSecret(assign *ast.Assignment, tok ast.ValueToken, resolver Resolver, secret string) (string, bool, error) {
	mods := modifierSet(tok.Modifiers)
	var observer renderObserver
	if rec, ok := resolver.(renderObserver); ok {
		observer = rec
	}
	if mods["dangerously_bypass_escape"] && len(mods) > 1 {
		return "", false, newPlaceholderError(assign.Line, tok.Column, tok.Path, "EVE-105-601", "invalid placeholder modifier combination")
	}
	if mods["base64"] {
		for mod := range mods {
			if mod == "base64" {
				continue
			}
			return "", false, newPlaceholderError(assign.Line, tok.Column, tok.Path, "EVE-105-601", "invalid placeholder modifier combination")
		}
	}
	if mods["dangerously_bypass_escape"] {
		if observer != nil {
			observer.RecordRendered(tok.Path, secret)
		}
		return secret, true, nil
	}

	// Default EOF newline normalization (Section 5.1):
	// Remove exactly one trailing logical newline (LF or CRLF) at EOF, if present.
	// Internal newlines and additional trailing newlines beyond the last one are preserved.
	secret = normalizeEOFNewline(secret)

	// Apply strip-family modifiers (Section 5.2) after normalization.
	secret = applyStripModifiers(secret, mods)

	if mods["base64"] {
		secret = base64.StdEncoding.EncodeToString([]byte(secret))
	}

	switch tok.Context {
	case ast.ContextSingleQuoted:
		rendered, err := renderSingleQuoted(secret, mods, assign.Line, tok.Column, tok.Path)
		if err != nil {
			return "", false, err
		}
		if observer != nil {
			observer.RecordRendered(tok.Path, rendered)
		}
		return rendered, false, nil
	case ast.ContextBare:
		rendered, err := renderBare(secret, mods, assign.Line, tok.Column, tok.Path)
		if err != nil {
			return "", false, err
		}
		if observer != nil {
			observer.RecordRendered(tok.Path, rendered)
		}
		return rendered, false, nil
	case ast.ContextDoubleQuoted:
		rendered, err := renderDoubleQuoted(secret, mods, assign.Line, tok.Column, tok.Path)
		if err != nil {
			return "", false, err
		}
		if observer != nil {
			observer.RecordRendered(tok.Path, rendered)
		}
		return rendered, false, nil
	case ast.ContextCommandSubstitution:
		rendered, err := renderCommandSubst(secret, mods, assign.Line, tok.Column, tok.Path)
		if err != nil {
			return "", false, err
		}
		if observer != nil {
			observer.RecordRendered(tok.Path, rendered)
		}
		return rendered, false, nil
	case ast.ContextBacktick:
		rendered, err := renderBacktick(secret, mods, assign.Line, tok.Column, tok.Path)
		if err != nil {
			return "", false, err
		}
		if observer != nil {
			observer.RecordRendered(tok.Path, rendered)
		}
		return rendered, false, nil
	default:
		return "", false, fmt.Errorf("line %d: unsupported placeholder context", assign.Line)
	}
}

// normalizeEOFNewline removes exactly one trailing logical newline at EOF.
// A logical newline is either LF ("\n") or CRLF ("\r\n"). A standalone CR is not
// treated as a logical newline for normalization purposes.
func normalizeEOFNewline(s string) string {
	if len(s) == 0 {
		return s
	}
	// Prefer CRLF match first.
	if strings.HasSuffix(s, "\r\n") {
		return s[:len(s)-2]
	}
	if strings.HasSuffix(s, "\n") {
		return s[:len(s)-1]
	}
	return s
}

func renderSingleQuoted(secret string, mods map[string]bool, line, column int, path string) (string, error) {
	if mods["allow_newline"] {
		return "", newPlaceholderError(line, column, path, "EVE-105-105", "allow_newline modifier is not supported in single-quoted context")
	}
	allowTab := mods["allow_tab"]
	for _, r := range secret {
		switch r {
		case '\'':
			return "", newPlaceholderError(line, column, path, "EVE-105-101", "single-quoted placeholder cannot contain a single quote (`'`)")
		case '\n', '\r':
			return "", newPlaceholderError(line, column, path, "EVE-105-102", "newline not permitted in single-quoted placeholder")
		case '\t':
			if !allowTab {
				return "", newPlaceholderError(line, column, path, "EVE-105-103", "TAB not permitted in single-quoted placeholder")
			}
		default:
			if isControlRune(r) {
				return "", newPlaceholderError(line, column, path, "EVE-105-104", "control character U+%04X not permitted in single-quoted placeholder", r)
			}
		}
	}
	return secret, nil
}
func renderBare(secret string, mods map[string]bool, line, column int, path string) (string, error) {
	if mods["allow_newline"] {
		return "", newPlaceholderError(line, column, path, "EVE-105-504", "allow_newline modifier is not supported in bare context")
	}
	allowTab := mods["allow_tab"]
	base64Mod := mods["base64"]
	var b strings.Builder
	for _, r := range secret {
		switch {
		case r == '\n' || r == '\r':
			return "", newPlaceholderError(line, column, path, "EVE-105-501", "newline not permitted in bare placeholder")
		case r == '\t':
			if !allowTab {
				return "", newPlaceholderError(line, column, path, "EVE-105-502", "TAB not permitted in bare placeholder")
			}
			b.WriteByte('\\')
			b.WriteRune(r)
		default:
			if isControlRune(r) {
				return "", newPlaceholderError(line, column, path, "EVE-105-503", "control character U+%04X not permitted in bare placeholder", r)
			}
			if shouldEscapeBare(r, base64Mod) {
				b.WriteByte('\\')
			}
			b.WriteRune(r)
		}
	}
	return b.String(), nil
}

func renderDoubleQuoted(secret string, mods map[string]bool, line, column int, path string) (string, error) {
	allowNewline := mods["allow_newline"]
	allowTab := mods["allow_tab"]
	var b strings.Builder
	for _, r := range secret {
		switch r {
		case '"', '\\', '$', '`':
			b.WriteByte('\\')
			b.WriteRune(r)
		case '\n':
			if !allowNewline {
				return "", newPlaceholderError(line, column, path, "EVE-105-201", "newline not permitted in double-quoted placeholder")
			}
			b.WriteRune(r)
		case '\r':
			if !allowNewline {
				return "", newPlaceholderError(line, column, path, "EVE-105-201", "newline not permitted in double-quoted placeholder")
			}
			b.WriteRune(r)
		case '\t':
			if !allowTab {
				return "", newPlaceholderError(line, column, path, "EVE-105-202", "TAB not permitted in double-quoted placeholder")
			}
			b.WriteRune(r)
		default:
			if isControlRune(r) {
				return "", newPlaceholderError(line, column, path, "EVE-105-203", "control character U+%04X not permitted in double-quoted placeholder", r)
			}
			b.WriteRune(r)
		}
	}
	return b.String(), nil
}

func renderCommandSubst(secret string, mods map[string]bool, line, column int, path string) (string, error) {
	allowNewline := mods["allow_newline"]
	allowTab := mods["allow_tab"]
	var b strings.Builder
	for _, r := range secret {
		switch r {
		case '\\', '$':
			b.WriteByte('\\')
			b.WriteRune(r)
		case ')':
			b.WriteString("\\)")
		case '\n':
			if !allowNewline {
				return "", newPlaceholderError(line, column, path, "EVE-105-301", "newline not permitted in command substitution placeholder")
			}
			b.WriteRune(r)
		case '\r':
			if !allowNewline {
				return "", newPlaceholderError(line, column, path, "EVE-105-301", "newline not permitted in command substitution placeholder")
			}
			b.WriteRune(r)
		case '\t':
			if !allowTab {
				return "", newPlaceholderError(line, column, path, "EVE-105-302", "TAB not permitted in command substitution placeholder")
			}
			b.WriteRune(r)
		default:
			if isControlRune(r) {
				return "", newPlaceholderError(line, column, path, "EVE-105-303", "control character U+%04X not permitted in command substitution placeholder", r)
			}
			b.WriteRune(r)
		}
	}
	return b.String(), nil
}

func renderBacktick(secret string, mods map[string]bool, line, column int, path string) (string, error) {
	if mods["allow_newline"] {
		return "", newPlaceholderError(line, column, path, "EVE-105-404", "allow_newline modifier is not supported in backtick context")
	}
	allowTab := mods["allow_tab"]
	var b strings.Builder
	for _, r := range secret {
		switch r {
		case '\n', '\r':
			return "", newPlaceholderError(line, column, path, "EVE-105-401", "newline not permitted in backtick placeholder")
		case '`', '\\', '$':
			b.WriteByte('\\')
			b.WriteRune(r)
		case '\t':
			if !allowTab {
				return "", newPlaceholderError(line, column, path, "EVE-105-402", "TAB not permitted in backtick placeholder")
			}
			b.WriteRune(r)
		default:
			if isControlRune(r) {
				return "", newPlaceholderError(line, column, path, "EVE-105-403", "control character U+%04X not permitted in backtick placeholder", r)
			}
			b.WriteRune(r)
		}
	}
	return b.String(), nil
}

func modifierSet(mods []string) map[string]bool {
	set := make(map[string]bool, len(mods))
	for _, m := range mods {
		set[m] = true
	}
	return set
}

func applyStripModifiers(secret string, mods map[string]bool) string {
	if !(mods["strip"] || mods["strip_left"] || mods["strip_right"]) {
		return secret
	}
	if mods["strip"] || mods["strip_left"] {
		secret = strings.TrimLeftFunc(secret, isStripWhitespace)
	}
	if mods["strip"] || mods["strip_right"] {
		secret = strings.TrimRightFunc(secret, isStripWhitespace)
	}
	return secret
}

func isStripWhitespace(r rune) bool {
	switch r {
	case ' ', '\t', '\n', '\r':
		return true
	default:
		return false
	}
}

func shouldEscapeBare(r rune, base64Mod bool) bool {
	if base64Mod {
		switch r {
		case '+', '/', '=':
			return false
		}
	}
	if isBareRune(r) {
		return false
	}
	switch r {
	case ' ', '\t', '#', '$', '"', '\'', '`', '\\', '(', ')', '{', '}', '[', ']', '|', '&', ';', '<', '>', '*', '?', '!', '~':
		return true
	}
	return false
}

func isBareRune(r rune) bool {
	if r == '_' || r == '.' || r == '-' {
		return true
	}
	if r >= '0' && r <= '9' {
		return true
	}
	if unicode.IsLetter(r) {
		return true
	}
	return false
}

func isControlRune(r rune) bool {
	return (r >= 0 && r < 0x20 && r != '\n' && r != '\t') || r == 0x7f
}

// OutputValidationError reports that the rendered output failed the parser check.
type OutputValidationError struct {
	Err error
}

func (e *OutputValidationError) Error() string {
	if e == nil || e.Err == nil {
		return "rendered output failed validation"
	}
	return fmt.Sprintf("rendered output failed validation: %v", e.Err)
}

func (e *OutputValidationError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}
