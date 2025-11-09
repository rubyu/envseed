package testsupport

import (
	"errors"
	"testing"
)

// ExpectErrorAs asserts that err can be cast to T via errors.As and that the
// provided extractor returns the expected code. The matched error is returned.
func ExpectErrorAs[T any](t testing.TB, err error, code string, extractor func(T) string) T {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error %s, got nil", code)
	}
	var target T
	if !errors.As(err, &target) {
		t.Fatalf("expected %T, got %T (%v)", target, err, err)
	}
	if got := extractor(target); got != code {
		t.Fatalf("detail code = %s, want %s", got, code)
	}
	return target
}
