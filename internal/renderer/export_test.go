// Test-only exports to support external tests without leaking internal details
// in the production API.

package renderer

// Test-only exports to support external tests without leaking internal details
// in the production API.

import (
	"fmt"
	"strings"

	"envseed/internal/sandbox"
	"envseed/internal/testsupport"
)

// ExportSandboxAvailable reports whether a sandboxed execution environment is available.
func ExportSandboxAvailable() (bool, error) { return sandbox.Available() }

// ExportSandboxCapture executes the rendered script in a sandbox and returns
// environment variable values for the given names via `declare -p` parsing.
func ExportSandboxCapture(rendered string, names []string) (map[string]string, error) {
	if len(names) == 0 {
		return map[string]string{}, nil
	}
	var b strings.Builder
	b.WriteString("set -eo pipefail\n")
	b.WriteString("set -a\n")
	b.WriteString(rendered)
	b.WriteString("\nset +a\n")
	for _, name := range names {
		b.WriteString("declare -p ")
		b.WriteString(name)
		b.WriteString("\n")
	}
	out, err := sandbox.Run(b.String())
	if err != nil {
		return nil, err
	}
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) != len(names) {
		return nil, fmt.Errorf("unexpected declare output: got %d lines, want %d", len(lines), len(names))
	}
	result := make(map[string]string, len(names))
	for _, line := range lines {
		name, value, err := testsupport.ParseBashDeclareLine(line)
		if err != nil {
			return nil, err
		}
		result[name] = value
	}
	return result, nil
}

// Export wrapper for context-local renderers and modifier processing.
func ExportModifierSet(mods []string) map[string]bool { return modifierSet(mods) }
func ExportRenderSingleQuoted(secret string, mods map[string]bool, line, column int, path string) (string, error) {
	return renderSingleQuoted(secret, mods, line, column, path)
}
func ExportRenderBare(secret string, mods map[string]bool, line, column int, path string) (string, error) {
	return renderBare(secret, mods, line, column, path)
}
func ExportRenderDoubleQuoted(secret string, mods map[string]bool, line, column int, path string) (string, error) {
	return renderDoubleQuoted(secret, mods, line, column, path)
}
func ExportRenderCommandSubst(secret string, mods map[string]bool, line, column int, path string) (string, error) {
	return renderCommandSubst(secret, mods, line, column, path)
}
func ExportRenderBacktick(secret string, mods map[string]bool, line, column int, path string) (string, error) {
	return renderBacktick(secret, mods, line, column, path)
}
