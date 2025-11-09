package envseed

import (
	"errors"

	"envseed/internal/parser"
	"envseed/internal/renderer"
)

func wrapParseError(err error) error {
	var perr *parser.ParseError
	if errors.As(err, &perr) {
		code := perr.DetailCode
		if code == "" {
			code = "EVE-103-1"
		}
		return NewExitError(code, perr.DetailArgs...).WithErr(err)
	}
	return NewExitError("EVE-103-1").WithErr(err)
}

func wrapRenderError(err error) error {
	var invalid *renderer.OutputValidationError
	if errors.As(err, &invalid) {
		return NewExitError("EVE-105-701").WithErr(err)
	}

	var placeholderErr *renderer.PlaceholderError
	if errors.As(err, &placeholderErr) {
		return NewExitError(placeholderErr.DetailCode(), placeholderErr.DetailArgs()...).WithErr(err)
	}

	var exitErr *ExitError
	if errors.As(err, &exitErr) {
		clone := *exitErr
		if clone.Err == nil {
			clone.Err = err
		}
		return &clone
	}
	return NewExitError("EVE-105-1").WithErr(err)
}
