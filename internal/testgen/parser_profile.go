package testgen

import (
	"fmt"
	"math/rand"
	"strings"
)

// ParserSyntaxProfile generates simple, valid templates that exercise
// assignments, comments, blanks, operators, and value contexts enough to
// stress parser round-trip without focusing on rendering rules.
type ParserSyntaxProfile struct {
	varIndex int
}

func randName(r *rand.Rand, idx int) string {
	return fmt.Sprintf("VAR%d", idx)
}

func randComment(r *rand.Rand) string {
	texts := []string{"# comment", "# another", "# こんにちは"}
	return texts[r.Intn(len(texts))]
}

func randPlaceholder(r *rand.Rand) string {
	paths := []string{"alpha/key", "service/token", "日本語/鍵"}
	mods := [][]string{{}, {"allow_tab"}, {"allow_newline"}, {"base64"}, {"dangerously_bypass_escape"}}
	path := paths[r.Intn(len(paths))]
	m := mods[r.Intn(len(mods))]
	if len(m) == 0 {
		return fmt.Sprintf("<pass:%s>", path)
	}
	return fmt.Sprintf("<pass:%s|%s>", path, strings.Join(m, ","))
}

func randValueBare(r *rand.Rand, idx int) string {
	opts := []string{
		"literal",
		randPlaceholder(r),
		fmt.Sprintf("pre%s", randPlaceholder(r)),
		fmt.Sprintf("%spost", randPlaceholder(r)),
	}
	return opts[r.Intn(len(opts))]
}

func randValueDouble(r *rand.Rand, idx int) string {
	opts := []string{
		fmt.Sprintf("\"%s\"", randPlaceholder(r)),
		"\"text\"",
	}
	return opts[r.Intn(len(opts))]
}

func randValueSingle(r *rand.Rand, idx int) string {
	opts := []string{
		"'text'",
		fmt.Sprintf("'%s'", randPlaceholder(r)), // remains literal for parser purposes
	}
	return opts[r.Intn(len(opts))]
}

func randValueCommand(r *rand.Rand, idx int) string {
	opts := []string{
		fmt.Sprintf("$(echo %s)", randPlaceholder(r)),
		fmt.Sprintf("$(printf %s %s)", "%s", randPlaceholder(r)),
	}
	return opts[r.Intn(len(opts))]
}

func randValueBacktick(r *rand.Rand, idx int) string {
	opts := []string{
		fmt.Sprintf("`echo %s`", randPlaceholder(r)),
		"`echo literal`",
	}
	return opts[r.Intn(len(opts))]
}

// Generate implements Profile.
func (p *ParserSyntaxProfile) Generate(r *rand.Rand, _ uint32) Case {
	lines := 1 + r.Intn(3)
	var b strings.Builder
	for i := 0; i < lines; i++ {
		switch r.Intn(5) {
		case 0:
			b.WriteString("\n")
		case 1:
			b.WriteString(randComment(r))
			b.WriteString("\n")
		default:
			name := randName(r, p.varIndex)
			p.varIndex++
			op := "="
			if r.Intn(4) == 0 {
				op = "+="
			}
			// choose a value context
			var value string
			switch r.Intn(5) {
			case 0:
				value = randValueBare(r, i)
			case 1:
				value = randValueDouble(r, i)
			case 2:
				value = randValueSingle(r, i)
			case 3:
				value = randValueCommand(r, i)
			default:
				value = randValueBacktick(r, i)
			}
			b.WriteString(name)
			b.WriteString(op)
			b.WriteString(value)
			b.WriteString("\n")
		}
	}
	return Case{Template: b.String(), Resolver: map[string]string{}, Expect: Expectation{}}
}
