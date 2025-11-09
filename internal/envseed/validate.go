package envseed

import (
	"context"
	"errors"
	"os"

	"envseed/internal/parser"
)

// Validate parses the template and reports syntax errors.
func Validate(_ context.Context, opts ValidateOptions) error {
	if opts.InputPath == "" {
		return NewExitError("EVE-101-201")
	}

	data, err := os.ReadFile(opts.InputPath)
	if err != nil {
		if os.IsNotExist(err) {
			return NewExitError("EVE-101-202", opts.InputPath).WithErr(err)
		}
		if errors.Is(err, os.ErrPermission) {
			return NewExitError("EVE-102-101", opts.InputPath).WithErr(err)
		}
		if info, statErr := os.Stat(opts.InputPath); statErr == nil && info.IsDir() {
			return NewExitError("EVE-101-302", opts.InputPath).WithErr(err)
		}
		return NewExitError("EVE-102-1", opts.InputPath).WithErr(err)
	}

	if _, err := parser.Parse(string(data)); err != nil {
		return wrapParseError(err)
	}
	return nil
}
