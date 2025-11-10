package envseed

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"envseed/internal/parser"
	"envseed/internal/renderer"
)

// Sync executes the envseed sync workflow.
func Sync(ctx context.Context, opts SyncOptions) error {
	// NOTE: input selection is handled by CLI (0/1 args). I/O classification follows EVE-102 bands.

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

	// stat-first classification (B0/B1), then open/read (B2)
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
		// open time I/O failure
		return NewExitError("EVE-102-201", opts.InputPath).WithErr(err)
	}
	data, rerr := io.ReadAll(f)
	_ = f.Close()
	if rerr != nil {
		return NewExitError("EVE-102-202", opts.InputPath).WithErr(rerr)
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
