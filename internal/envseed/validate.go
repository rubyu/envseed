package envseed

import (
	"context"
	"errors"
	"io"
	"os"

	"envseed/internal/parser"
)

// Validate parses the template and reports syntax errors.
func Validate(_ context.Context, opts ValidateOptions) error {
	if opts.InputPath == "" {
		// CLI handles input selection; treat empty as internal misuse
		return NewExitError("EVE-102-203", "<empty>")
	}

	if info, statErr := os.Lstat(opts.InputPath); statErr != nil {
		code := classifyStatDetail(statErr)
		return NewExitError(code, opts.InputPath).WithErr(statErr)
	} else if info.IsDir() {
		return NewExitError("EVE-102-2", opts.InputPath)
	}
	f, err := os.Open(opts.InputPath)
	if err != nil {
		if errors.Is(err, os.ErrPermission) {
			return NewExitError("EVE-102-101", opts.InputPath).WithErr(err)
		}
		return NewExitError("EVE-102-201", opts.InputPath).WithErr(err)
	}
	data, rerr := io.ReadAll(f)
	_ = f.Close()
	if rerr != nil {
		return NewExitError("EVE-102-202", opts.InputPath).WithErr(rerr)
	}

	if _, err := parser.Parse(string(data)); err != nil {
		return wrapParseError(err)
	}
	return nil
}
