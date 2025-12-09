package testsupport

import (
	"os"
	"strings"
	"testing"

	"envseed/internal/sandbox"
)

// RequireSandbox enforces availability of the bubblewrap-based sandbox.
//   - If ENVSEED_SANDBOX_DISABLE=1 is set, it skips with a clear reason.
//   - Otherwise, it performs a smoke check and fails the test with actionable
//     hints if the sandbox is unavailable.
func RequireSandbox(t *testing.T) {
	t.Helper()

	if os.Getenv("ENVSEED_SANDBOX_DISABLE") == "1" {
		t.Skipf("sandbox disabled via ENVSEED_SANDBOX_DISABLE=1 (developer override)")
		return
	}

	ok, err := sandbox.Available()
	if ok && err == nil {
		return
	}

	// Build a compact, actionable failure message based on structured reasons.
	var hints []string
	switch {
	case err == nil:
		// Shouldn't happen, but guard for safety.
		hints = append(hints, "- Unexpected availability state; please report.")
	case sandbox.Is(err, sandbox.ErrNotInstalled):
		hints = append(hints, "- Install bubblewrap (e.g., apt-get install bubblewrap) or ensure it is in PATH.")
	case sandbox.Is(err, sandbox.ErrNamespaceDenied):
		fallthrough
	case sandbox.Is(err, sandbox.ErrNetlinkRouteDenied):
		hints = append(hints,
			"- Host or outer sandbox restricts user/net namespaces.",
			"- If using an external sandbox (e.g., Codex), disable it and re-run.",
			"- As a temporary workaround, set ENVSEED_SANDBOX_DISABLE=1 to skip sandbox tests.")
	case sandbox.Is(err, sandbox.ErrDepsMissing):
		hints = append(hints, "- Ensure ldd is available and bash runtime libraries are present.")
	case sandbox.Is(err, sandbox.ErrBashMissing):
		hints = append(hints, "- Ensure bash is installed and accessible in PATH.")
	default:
		// Generic unavailable or unknown reason
		hints = append(hints,
			"- If running under an external sandbox/CI, try disabling it.",
			"- As a last resort, set ENVSEED_SANDBOX_DISABLE=1 to skip sandbox tests.")
	}

	// Include raw error for context; tests remain concise but helpful.
	// Fixed format string for go vet
	base := "sandbox required but unavailable"
	if err != nil {
		base = base + ": " + err.Error()
	}
	if len(hints) > 0 {
		base = base + "\nHints:\n  " + strings.Join(hints, "\n  ")
	}
	t.Fatalf("%s", base)
}
