package renderer

// NOTE: Renderer-scoped test helpers
// Keep only renderer-specific convenience wrappers in this file. Generic
// testing utilities (error assertions, string comparisons, etc.) live under
// internal/testsupport and should be reused from there to avoid duplication.

import (
	"errors"
	"testing"

	"envseed/internal/testsupport"
)

type mapResolver map[string]string

func (m mapResolver) Resolve(path string) (string, error) {
	v, ok := m[path]
	if !ok {
		return "", errors.New("missing secret")
	}
	return v, nil
}

func expectPlaceholderError(t *testing.T, err error, code string) *PlaceholderError {
	return testsupport.ExpectErrorAs[*PlaceholderError](t, err, code, func(pe *PlaceholderError) string {
		return pe.DetailCode()
	})
}
