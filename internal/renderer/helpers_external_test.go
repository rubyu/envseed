package renderer_test

import (
	"errors"
	"testing"

	renderer "envseed/internal/renderer"
	"envseed/internal/testsupport"
)

// Minimal resolver for external (renderer_test) package-based tests.
// Unified name across external tests for clarity.
type externalResolver map[string]string

func (m externalResolver) Resolve(path string) (string, error) {
	v, ok := m[path]
	if !ok {
		return "", errors.New("missing secret")
	}
	return v, nil
}

// Shared placeholder error expectation helper for external tests
func expectPlaceholderError(t testing.TB, err error, code string) *renderer.PlaceholderError {
	return testsupport.ExpectErrorAs[*renderer.PlaceholderError](t, err, code, func(pe *renderer.PlaceholderError) string {
		return pe.DetailCode()
	})
}
