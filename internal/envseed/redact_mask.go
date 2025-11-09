package envseed

// Redaction masking and target parsing (Section 6.3): MaskEnv, ParseTarget.
// This file focuses on masking; diff reconstruction lives in redact_diff.go.
import (
	"errors"
	"strings"
	"unicode"
	"unicode/utf8"

	"envseed/internal/ast"
	"envseed/internal/parser"
)

// ParseTarget parses a rendered .env text using the existing template parser,
// then enforces .env-specific constraints:
// - Placeholders MUST NOT appear
// - Parse failures are mapped into EVE-107-* detail codes
func ParseTarget(text string) ([]ast.Element, error) {
	// Early forbidden characters
	if containsNUL(text) {
		return nil, NewExitError("EVE-107-102")
	}

	elems, err := parser.Parse(text)
	if err != nil {
		// Map parser errors (103) into 107-series for target parsing
		var perr *parser.ParseError
		if errors.As(err, &perr) {
			// Non-ASCII whitespace in target context
			if containsNonASCIIWhitespace(text) {
				return nil, NewExitError("EVE-107-101").WithErr(err)
			}
			// Unterminated constructs mapping (103-401..404 → 107-201..204)
			switch perr.DetailCode {
			case "EVE-103-401":
				return nil, NewExitError("EVE-107-201").WithErr(err)
			case "EVE-103-402":
				return nil, NewExitError("EVE-107-202").WithErr(err)
			case "EVE-103-403":
				return nil, NewExitError("EVE-107-203").WithErr(err)
			case "EVE-103-404":
				return nil, NewExitError("EVE-107-204").WithErr(err)
			}
			// Generic parse failure
			return nil, NewExitError("EVE-107-205").WithErr(err)
		}
		return nil, NewExitError("EVE-107-205").WithErr(err)
	}
	// Reject placeholders in target .env
	for _, el := range elems {
		if el.Type != ast.ElementAssignment || el.Assignment == nil {
			continue
		}
		for _, tok := range el.Assignment.ValueTokens {
			if tok.Kind == ast.ValuePlaceholder {
				return nil, NewExitError("EVE-107-301")
			}
		}
	}
	return elems, nil
}

// MaskEnv returns a masked representation of a rendered .env text using the
// Section 6.3 policy with length-bounded head/tail reveal and newline
// preservation. Delimiting tokens are preserved; escape backslashes inside
// string segments are elided and MUST NOT appear in masked outputs.
func MaskEnv(text string) (string, error) {
	elems, err := ParseTarget(text)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	for _, el := range elems {
		switch el.Type {
		case ast.ElementBlank:
			b.WriteString(el.Text)
		case ast.ElementComment:
			b.WriteString(el.Text)
			if el.HasTrailingNewline {
				b.WriteString("\n")
			}
		case ast.ElementAssignment:
			as := el.Assignment
			// Leading whitespace + name + operator
			b.WriteString(as.LeadingWhitespace)
			b.WriteString(as.Name)
			if as.Operator == ast.OperatorAppend {
				b.WriteString("+=")
			} else {
				b.WriteString("=")
			}
			// Masked value
			rawVal := valueText(as.ValueTokens)
			b.WriteString(maskValueString(rawVal))
			// Trailing comment
			if as.TrailingComment != "" {
				b.WriteString(as.TrailingComment)
			}
			if as.HasTrailingNewline {
				b.WriteString("\n")
			}
		}
	}
	return b.String(), nil
}

func valueText(tokens []ast.ValueToken) string {
	if len(tokens) == 0 {
		return ""
	}
	var b strings.Builder
	for _, t := range tokens {
		b.WriteString(t.Text)
	}
	return b.String()
}

// maskValueString masks string segments while preserving syntactic delimiters
// (quotes, $(), backticks) and backslashes, with newline preservation.
func maskValueString(s string) string {
	type frameKind int
	const (
		frameBare frameKind = iota
		frameDouble
		frameSingle
		frameBacktick
		frameCommand
	)
	type frame struct {
		kind  frameKind
		paren int
	}

	var out strings.Builder
	var seg strings.Builder
	stack := []frame{{kind: frameBare}}
	escaped := false

	flush := func() {
		if seg.Len() == 0 {
			return
		}
		out.WriteString(maskWithRevealPreservingNewlines(seg.String()))
		seg.Reset()
	}

	// Helper: write delimiter and update stack as needed
	write := func(r rune) {
		out.WriteRune(r)
	}

	for i := 0; i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])
		if r == '\n' || r == '\r' {
			flush()
			write(r)
			i += size
			escaped = false
			continue
		}

		top := stack[len(stack)-1].kind

		// Trailing comment at top-level (bare) begins with # (not escaped)
		if !escaped && top == frameBare && r == '#' {
			flush()
			// Copy the rest verbatim
			out.WriteString(s[i:])
			return out.String()
		}

		if !escaped {
			// Command substitution start: $(
			if r == '$' && top != frameSingle && top != frameBacktick {
				// Look ahead for '('
				if nr, nsize := utf8.DecodeRuneInString(s[i+size:]); nr == '(' {
					flush()
					write(r)
					write(nr)
					stack = append(stack, frame{kind: frameCommand, paren: 1})
					i += size + nsize
					continue
				}
			}
			switch r {
			case '"':
				flush()
				write(r)
				if top == frameDouble {
					stack = stack[:len(stack)-1]
				} else if top != frameSingle && top != frameBacktick {
					stack = append(stack, frame{kind: frameDouble})
				}
				i += size
				continue
			case '\'':
				flush()
				write(r)
				if top == frameSingle {
					stack = stack[:len(stack)-1]
				} else if top != frameDouble && top != frameBacktick {
					stack = append(stack, frame{kind: frameSingle})
				}
				i += size
				continue
			case '`':
				flush()
				write(r)
				if top == frameBacktick {
					stack = stack[:len(stack)-1]
				} else if top != frameSingle {
					stack = append(stack, frame{kind: frameBacktick})
				}
				i += size
				continue
			case ')':
				if top == frameCommand {
					flush()
					write(r)
					// decrement paren depth
					fr := stack[len(stack)-1]
					fr.paren--
					if fr.paren <= 0 {
						stack = stack[:len(stack)-1]
					} else {
						stack[len(stack)-1] = fr
					}
					i += size
					continue
				}
			}
		}

		// Backslash handling (elide escape backslashes inside string segments):
		// - Outside single quotes, treat a backslash followed by a non-newline code point
		//   as an escape pair: elide the backslash and mask the following code point
		//   by appending '*' to the current segment.
		// - If followed by CR/LF, do not form an escape pair: elide the backslash and
		//   let the newline branch handle line preservation in the next iteration.
		if r == '\\' && top != frameSingle {
			// lookahead next rune (if any)
			if i+size >= len(s) {
				// trailing backslash at end: elide
				i += size
				continue
			}
			nr, nsize := utf8.DecodeRuneInString(s[i+size:])
			if nr == '\n' || nr == '\r' {
				// elide backslash; newline handled on next loop
				i += size
				continue
			}
			// escape pair: elide backslash and mask following code point
			seg.WriteByte('*')
			i += size + nsize
			// ensure escaped state does not leak
			if escaped {
				escaped = false
			}
			continue
		}

		// Regular character within string segment
		seg.WriteRune(r)
		i += size
		if escaped {
			escaped = false
		}
	}

	flush()
	return out.String()
}

// maskWithRevealPreservingNewlines applies head/tail reveal per line,
// preserving CR/LF characters.
func maskWithRevealPreservingNewlines(s string) string {
	if s == "" {
		return ""
	}
	// Fast path: no newline
	if !strings.ContainsAny(s, "\n\r") {
		return maskRevealUnit(s)
	}
	var b strings.Builder
	start := 0
	for start < len(s) {
		// find next newline (either CR or LF)
		i := start
		for i < len(s) {
			r, size := utf8.DecodeRuneInString(s[i:])
			if r == '\n' || r == '\r' {
				break
			}
			i += size
		}
		unit := s[start:i]
		b.WriteString(maskRevealUnit(unit))
		if i < len(s) {
			r, size := utf8.DecodeRuneInString(s[i:])
			b.WriteRune(r)
			i += size
		}
		start = i
	}
	return b.String()
}

func maskRevealUnit(s string) string {
	// Count Unicode scalar values (code points); CR/LF are split earlier.
	runes := []rune(s)
	n := len(runes)
	if n <= 0 {
		return ""
	}
	if n < 8 {
		return strings.Repeat("*", n)
	}
	if n < 16 {
		return string(runes[:1]) + strings.Repeat("*", n-2) + string(runes[n-1:])
	}
	return string(runes[:2]) + strings.Repeat("*", n-4) + string(runes[n-2:])
}

func containsNonASCIIWhitespace(s string) bool {
	for _, r := range s {
		if r == ' ' || r == '\t' || r == '\n' || r == '\r' {
			continue
		}
		if unicode.IsSpace(r) {
			return true
		}
	}
	return false
}

func containsNUL(s string) bool {
	return strings.IndexByte(s, 0) >= 0
}
