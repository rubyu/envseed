//go:build !linux || !sandbox
// +build !linux !sandbox

package sandbox

// Available reports whether a sandboxed execution environment is available.
func Available() (bool, error) { return false, ErrUnsupported }

// Run executes the given bash script in a sandboxed environment (if available)
// and returns stdout or an error.
func Run(script string) (string, error) { return "", ErrUnsupported }
