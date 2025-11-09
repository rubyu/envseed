package envseed

import (
	"strings"

	"github.com/pmezard/go-difflib/difflib"
)

// unifiedDiff builds a unified diff on raw existing(A) and rendered(B).
func unifiedDiff(path, existing, rendered string) (string, error) {
	ud := difflib.UnifiedDiff{
		A:        difflib.SplitLines(existing),
		B:        difflib.SplitLines(rendered),
		FromFile: path,
		ToFile:   path,
		Context:  3,
	}
	diffText, err := difflib.GetUnifiedDiffString(ud)
	if err != nil {
		return "", NewExitError("EVE-108-2", path).WithErr(err)
	}
	return diffText, nil
}

// reconstructMaskedDiff takes a unified diff computed on raw A/B and rebuilds
// its content lines using masked A′/B′. Headers (---/+++) and hunk markers are
// preserved as-is.
func reconstructMaskedDiff(rawDiff, maskedA, maskedB string) string {
	aLines := strings.SplitAfter(maskedA, "\n")
	bLines := strings.SplitAfter(maskedB, "\n")
	ai, bi := 0, 0
	var out strings.Builder
	lines := strings.Split(rawDiff, "\n")
	for i, line := range lines {
		var emit string
		switch {
		case strings.HasPrefix(line, "--- ") || strings.HasPrefix(line, "+++ ") || strings.HasPrefix(line, "@@") || line == "":
			// headers and hunk markers: preserve as-is
			emit = line
		case strings.HasPrefix(line, " "):
			if ai < len(aLines) {
				emit = " " + strings.TrimSuffix(aLines[ai], "\n")
				ai++
				bi++
			} else {
				emit = line
			}
		case strings.HasPrefix(line, "-"):
			if ai < len(aLines) {
				emit = "-" + strings.TrimSuffix(aLines[ai], "\n")
				ai++
			} else {
				emit = line
			}
		case strings.HasPrefix(line, "+"):
			if bi < len(bLines) {
				emit = "+" + strings.TrimSuffix(bLines[bi], "\n")
				bi++
			} else {
				emit = line
			}
		default:
			emit = line
		}
		out.WriteString(emit)
		if i < len(lines)-1 || strings.HasSuffix(rawDiff, "\n") {
			out.WriteString("\n")
		}
	}
	return out.String()
}
