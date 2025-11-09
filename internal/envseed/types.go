package envseed

import (
	"context"
	"io"
)

// SyncOptions configure the sync subcommand.
type SyncOptions struct {
	InputPath  string
	OutputPath string
	Force      bool
	DryRun     bool
	Quiet      bool

	PassClient PassClient
	Stdout     io.Writer
	Stderr     io.Writer
}

// DiffOptions configure the diff subcommand.
type DiffOptions struct {
	InputPath  string
	OutputPath string

	PassClient PassClient
	Stdout     io.Writer
	Stderr     io.Writer
}

// DiffResult reports whether differences were detected.
type DiffResult struct {
	Changed bool
}

// ValidateOptions configure the validate subcommand.
type ValidateOptions struct {
	InputPath string
}

// PassClient retrieves secrets from pass.
type PassClient interface {
	Show(ctx context.Context, path string) (string, error)
}
