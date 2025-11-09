package envseed

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"

	"envseed/internal/parser"
	"envseed/internal/renderer"
)

const diffSizeLimit = 10 * (1 << 20)

// Diff executes the envseed diff workflow.
func Diff(ctx context.Context, opts DiffOptions) (DiffResult, error) {
	if opts.InputPath == "" {
		return DiffResult{}, NewExitError("EVE-101-201")
	}

	passClient := opts.PassClient
	if passClient == nil {
		passClient = &PassCommand{}
	}

	stdout := opts.Stdout
	if stdout == nil {
		stdout = os.Stdout
	}

	data, err := os.ReadFile(opts.InputPath)
	if err != nil {
		if os.IsNotExist(err) {
			return DiffResult{}, NewExitError("EVE-101-202", opts.InputPath).WithErr(err)
		}
		if errors.Is(err, os.ErrPermission) {
			return DiffResult{}, NewExitError("EVE-102-101", opts.InputPath).WithErr(err)
		}
		if info, statErr := os.Stat(opts.InputPath); statErr == nil && info.IsDir() {
			return DiffResult{}, NewExitError("EVE-101-302", opts.InputPath).WithErr(err)
		}
		return DiffResult{}, NewExitError("EVE-102-1", opts.InputPath).WithErr(err)
	}

	targetPath, err := resolveOutputPath(opts.InputPath, opts.OutputPath)
	if err != nil {
		return DiffResult{}, err
	}

	source := string(data)
	elements, err := parser.Parse(source)
	if err != nil {
		return DiffResult{}, wrapParseError(err)
	}

	resolver := newPassResolver(ctx, passClient)
	defer resolver.Close()

	rendered, err := renderer.RenderElements(elements, resolver)
	if err != nil {
		return DiffResult{}, wrapRenderError(err)
	}

	renderedBytes := []byte(rendered)
	// Masked redacted output (B')
	redactedOutput, err := MaskEnv(rendered)
	if err != nil {
		return DiffResult{}, err
	}

	existing, err := readFileIfExists(targetPath)
	if err != nil {
		return DiffResult{}, err
	}

	if len(existing) > diffSizeLimit || len(renderedBytes) > diffSizeLimit {
		return DiffResult{}, NewExitError("EVE-108-1", targetPath)
	}

	if bytes.Equal(existing, renderedBytes) {
		return DiffResult{Changed: false}, nil
	}

	// Masked existing (A')
	redactedExisting, err := MaskEnv(string(existing))
	if err != nil {
		return DiffResult{}, err
	}

	// Build raw diff and reconstruct its content using masked A′/B′ so that
	// small masked segments still appear as changes.
	rawDiff, err := unifiedDiff(targetPath, string(existing), rendered)
	if err != nil {
		return DiffResult{}, err
	}
	// Reconstruct hunk/body using masked A′/B′; preserve headers from rawDiff.
	diffText := reconstructMaskedDiff(rawDiff, redactedExisting, redactedOutput)

	if diffText != "" {
		if _, err := io.WriteString(stdout, diffText); err != nil {
			return DiffResult{}, NewExitError("EVE-108-3", targetPath).WithErr(err)
		}
	}

	return DiffResult{Changed: true}, nil
}
