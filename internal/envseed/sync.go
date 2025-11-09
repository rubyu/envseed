package envseed

import (
	"context"
	"errors"
	"fmt"
	"os"

	"envseed/internal/parser"
	"envseed/internal/renderer"
)

// Sync executes the envseed sync workflow.
func Sync(ctx context.Context, opts SyncOptions) error {
	if opts.InputPath == "" {
		return NewExitError("EVE-101-201")
	}

	passClient := opts.PassClient
	if passClient == nil {
		passClient = &PassCommand{}
	}

	stdout := opts.Stdout
	if stdout == nil {
		stdout = os.Stdout
	}
	stderr := opts.Stderr
	if stderr == nil {
		stderr = os.Stderr
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

	targetPath, err := resolveOutputPath(opts.InputPath, opts.OutputPath)
	if err != nil {
		return err
	}

	source := string(data)
	elements, err := parser.Parse(source)
	if err != nil {
		return wrapParseError(err)
	}

	resolver := newPassResolver(ctx, passClient)
	defer resolver.Close()

	rendered, err := renderer.RenderElements(elements, resolver)
	if err != nil {
		return wrapRenderError(err)
	}

	// Build masked preview from the rendered output per redaction policy.
	redacted, err := MaskEnv(rendered)
	if err != nil {
		return err
	}

	if opts.DryRun {
		if _, err := fmt.Fprintf(stdout, "target: %s\n%s", targetPath, redacted); err != nil {
			return NewExitError("EVE-106-401").WithErr(err)
		}
		return nil
	}

	if err := validateOutputPath(targetPath); err != nil {
		return err
	}

	if err := writeOutput(targetPath, []byte(rendered), opts.Quiet, opts.Force, stderr); err != nil {
		return err
	}

	return nil
}
