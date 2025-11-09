package envseed

import (
	"context"
	"errors"
	"fmt"

	"envseed/internal/parser"
	"envseed/internal/renderer"
)

// Shared test helpers extracted from core_test.go for reuse across split files.

type fakePass struct {
	values map[string]string
	errs   map[string]error
	calls  map[string]int
}

func (f *fakePass) Show(ctx context.Context, path string) (string, error) {
	if f.calls == nil {
		f.calls = make(map[string]int)
	}
	f.calls[path]++
	if err, ok := f.errs[path]; ok {
		return "", err
	}
	if v, ok := f.values[path]; ok {
		if len(v) > 0 && v[len(v)-1] == '\n' {
			return v, nil
		}
		return v + "\n", nil
	}
	return "", NewExitError("EVE-105-1", path)
}

type staticPassClient struct {
	values map[string]string
}

func (s *staticPassClient) Show(ctx context.Context, path string) (string, error) {
	if v, ok := s.values[path]; ok {
		return v, nil
	}
	return "", fmt.Errorf("missing pass entry %q", path)
}

type renderMapResolver map[string]string

func (m renderMapResolver) Resolve(path string) (string, error) {
	v, ok := m[path]
	if !ok {
		return "", fmt.Errorf("missing secret %q", path)
	}
	return v, nil
}

func renderResultWithPass(t TestingT, template string, values map[string]string) (string, error) {
	t.Helper()
	elems, err := parser.Parse(template)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	resolver := newPassResolver(context.Background(), &staticPassClient{values: values})
	defer resolver.Close()
	return renderer.RenderElements(elems, resolver)
}

func expectRenderError(t TestingT, template string, values map[string]string, wantCode, wantMsg, wantDetail string) {
	t.Helper()
	_, err := renderResultWithPass(t, template, values)
	if err == nil {
		t.Fatalf("expected render error for template %q", template)
	}
	wrapped := wrapRenderError(err)
	var exitErr *ExitError
	if !errors.As(wrapped, &exitErr) {
		t.Fatalf("expected ExitError, got %v", wrapped)
	}
	if exitErr.DetailCode != wantCode {
		t.Fatalf("detail code = %s, want %s", exitErr.DetailCode, wantCode)
	}
	if exitErr.Msg != wantMsg {
		t.Fatalf("message = %q, want %q", exitErr.Msg, wantMsg)
	}
	if exitErr.DetailText != wantDetail {
		t.Fatalf("detail text = %q, want %q", exitErr.DetailText, wantDetail)
	}
}

func renderMustSucceed(t TestingT, template string, values map[string]string) string {
	t.Helper()
	rendered, err := renderResultWithPass(t, template, values)
	if err != nil {
		t.Fatalf("unexpected render error: %v", wrapRenderError(err))
	}
	return rendered
}

// Minimal testing interface to decouple helpers from *testing.T
type TestingT interface {
	Helper()
	Fatal(args ...any)
	Fatalf(format string, args ...any)
}

// errors.As used directly; no local wrapper required.
