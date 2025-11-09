package parser_test

import (
	"fmt"
	"strings"

	"envseed/internal/ast"
)

// Local aliases for brevity in parser external tests.
type Element = ast.Element
type Assignment = ast.Assignment
type ValueToken = ast.ValueToken

const (
	ElementAssignment = ast.ElementAssignment
	ElementComment    = ast.ElementComment
	ElementBlank      = ast.ElementBlank
)

// deriveIterationSeed mixes iteration into the seed to generate a per-iteration
// seed with simple arithmetic. Stable and deterministic.
func deriveIterationSeed(seed int64, iteration uint32) int64 {
	const mix = int64(6364136223846793005)
	return seed ^ (int64(iteration)+1)*mix
}

// renderElementsText writes the textual form of AST elements without invoking
// any rendering rules beyond structure re-emission.
func renderElementsText(elems []ast.Element) (string, error) {
	var b strings.Builder
	for _, e := range elems {
		switch e.Type {
		case ast.ElementBlank:
			b.WriteString(e.Text)
		case ast.ElementComment:
			b.WriteString(e.Text)
			if e.HasTrailingNewline {
				b.WriteString("\n")
			}
		case ast.ElementAssignment:
			a := e.Assignment
			if a == nil {
				continue
			}
			b.WriteString(a.LeadingWhitespace)
			b.WriteString(a.Name)
			if a.Operator == ast.OperatorAppend {
				b.WriteString("+=")
			} else {
				b.WriteString("=")
			}
			for _, tok := range a.ValueTokens {
				b.WriteString(tok.Text)
			}
			b.WriteString(a.TrailingComment)
			if a.HasTrailingNewline {
				b.WriteString("\n")
			}
		default:
			return "", fmt.Errorf("unknown element type %v", e.Type)
		}
	}
	return b.String(), nil
}

func compareElements(a, b []ast.Element) string {
	if len(a) != len(b) {
		return fmt.Sprintf("element count mismatch: %d vs %d", len(a), len(b))
	}
	for i := range a {
		ai := a[i]
		bi := b[i]
		if ai.Type != bi.Type {
			return fmt.Sprintf("element %d type mismatch: %v vs %v", i, ai.Type, bi.Type)
		}
		if ai.HasTrailingNewline != bi.HasTrailingNewline {
			return fmt.Sprintf("element %d newline mismatch: %v vs %v", i, ai.HasTrailingNewline, bi.HasTrailingNewline)
		}
		switch ai.Type {
		case ast.ElementBlank, ast.ElementComment:
			if ai.Text != bi.Text {
				return fmt.Sprintf("element %d text mismatch: %q vs %q", i, ai.Text, bi.Text)
			}
		case ast.ElementAssignment:
			if ai.Assignment == nil || bi.Assignment == nil {
				return fmt.Sprintf("element %d assignment nil mismatch", i)
			}
			a1 := ai.Assignment
			a2 := bi.Assignment
			if a1.LeadingWhitespace != a2.LeadingWhitespace || a1.Name != a2.Name || a1.Operator != a2.Operator || a1.TrailingComment != a2.TrailingComment {
				return fmt.Sprintf("element %d assignment fields differ", i)
			}
			if len(a1.ValueTokens) != len(a2.ValueTokens) {
				return fmt.Sprintf("element %d token count mismatch: %d vs %d", i, len(a1.ValueTokens), len(a2.ValueTokens))
			}
			for j := range a1.ValueTokens {
				if a1.ValueTokens[j].Text != a2.ValueTokens[j].Text {
					return fmt.Sprintf("element %d token %d text mismatch: %q vs %q", i, j, a1.ValueTokens[j].Text, a2.ValueTokens[j].Text)
				}
			}
		}
	}
	return ""
}
