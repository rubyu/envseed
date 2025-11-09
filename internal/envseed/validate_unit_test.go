package envseed

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// [EVT-BDU-1]
func TestValidateParseError(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	input := filepath.Join(dir, ".envseed")
	if err := os.WriteFile(input, []byte("NAME=\"unterminated\n"), 0o600); err != nil {
		t.Fatalf("write template: %v", err)
	}

	err := Validate(context.Background(), ValidateOptions{
		InputPath: input,
	})
	var exitErr *ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("expected ExitError, got %v", err)
	}
	if exitErr.Code != ExitTemplateParse {
		t.Fatalf("exit code = %d, want %d", exitErr.Code, ExitTemplateParse)
	}
}
