package sandbox

import (
	"errors"
	"fmt"
)

// Sentinel errors to classify sandbox unavailability causes.
var (
	// Generic parent (for backwards compatibility with callers that only
	// check availability without inspecting a specific reason).
	ErrUnavailable = errors.New("sandbox unavailable")

	// Specific reasons (use errors.Is(err, <Reason>) to branch):
	ErrNotInstalled       = errors.New("sandbox bwrap not installed")
	ErrNamespaceDenied    = errors.New("sandbox namespace denied")
	ErrNetlinkRouteDenied = errors.New("sandbox netlink route denied")
	ErrDepsMissing        = errors.New("sandbox dependencies missing")
	ErrBashMissing        = errors.New("sandbox bash not found")
)

// Backwards-compat alias for existing callers.
var ErrUnsupported = ErrUnavailable

// UnavailableError carries a structured reason for sandbox unavailability.
// Use errors.Is(err, ErrUnavailable) or a specific reason (e.g., ErrNamespaceDenied)
// to match.
type UnavailableError struct {
	// Reason is one of the sentinel errors above.
	Reason error
	// Stage indicates where the error occurred (e.g., lookup_bwrap, ldd, run).
	Stage string
	// Stderr optionally carries tool stderr (e.g., from bwrap/ldd) for diagnosis.
	Stderr string
	// Cause optionally chains the underlying error.
	Cause error
}

func (e *UnavailableError) Error() string {
	if e == nil {
		return "sandbox unavailable"
	}
	msg := "sandbox unavailable"
	if e.Reason != nil {
		msg = fmt.Sprintf("%s: %s", msg, e.Reason.Error())
	}
	if e.Stage != "" {
		msg = fmt.Sprintf("%s (stage=%s)", msg, e.Stage)
	}
	if e.Stderr != "" {
		msg = fmt.Sprintf("%s: %s", msg, e.Stderr)
	}
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", msg, e.Cause)
	}
	return msg
}

// Unwrap makes errors.Is(err, e.Reason) succeed and also allows callers to
// match ErrUnavailable via Is (below).
func (e *UnavailableError) Unwrap() error { return e.Reason }

// Is allows matching both ErrUnavailable and the specific Reason sentinel.
func (e *UnavailableError) Is(target error) bool {
	if target == nil || e == nil {
		return false
	}
	return target == ErrUnavailable || target == e.Reason
}

// Is reports whether err matches target, unwrapping UnavailableError if needed.
func Is(err, target error) bool {
	if err == nil || target == nil {
		return false
	}
	if ue, ok := err.(*UnavailableError); ok {
		if target == ErrUnavailable || target == ue.Reason {
			return true
		}
		// also allow chaining via errors.Is on Reason
		return errors.Is(ue.Reason, target)
	}
	return errors.Is(err, target)
}
