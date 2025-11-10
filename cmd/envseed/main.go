package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	"envseed/internal/envseed"
	"envseed/internal/version"
)

func main() {
	args := os.Args[1:]
	if containsVersionFlag(args) {
		fmt.Fprintln(os.Stdout, version.String())
		return
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	if len(args) < 1 {
		handleError(envseed.NewExitError("EVE-101-1"))
		return
	}

	cmd := args[0]
	subArgs := args[1:]

	switch cmd {
	case "sync":
		handleError(runSync(ctx, subArgs))
	case "diff":
		handleError(runDiff(ctx, subArgs))
	case "validate":
		handleError(runValidate(ctx, subArgs))
	case "version":
		handleError(runVersion(subArgs))
	case "-h", "--help", "help":
		printUsage(os.Stdout)
		os.Exit(envseed.ExitOK)
	default:
		handleError(envseed.NewExitError("EVE-101-2", cmd))
	}
}

func runSync(ctx context.Context, args []string) error {
	var outputPath string
	var force bool
	var dryRun bool
	var quiet bool

	fs := flag.NewFlagSet("sync", flag.ContinueOnError)
	fs.StringVar(&outputPath, "output", "", "override the destination path")
	fs.StringVar(&outputPath, "o", "", "override the destination path (shorthand)")
	fs.BoolVar(&force, "force", false, "overwrite existing output files")
	fs.BoolVar(&force, "f", false, "overwrite existing output files (shorthand)")
	fs.BoolVar(&dryRun, "dry-run", false, "print redacted result instead of writing files")
	fs.BoolVar(&quiet, "quiet", false, "suppress informational output")
	fs.BoolVar(&quiet, "q", false, "suppress informational output (shorthand)")
	fs.SetOutput(io.Discard)
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: envseed sync [flags] [INPUT_FILE]\n\nFlags:\n")
		fs.SetOutput(os.Stderr)
		fs.PrintDefaults()
		fs.SetOutput(io.Discard)
	}

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			fs.Usage()
			return exitRequest{code: envseed.ExitOK}
		}
		return envseed.NewExitError("EVE-101-5", err.Error())
	}

	if fs.NArg() > 1 {
		return envseed.NewExitError("EVE-101-6")
	}

	inputPath := ".envseed"
	if fs.NArg() == 1 {
		inputPath = fs.Arg(0)
	}
	if inputPath == "-" {
		return envseed.NewExitError("EVE-101-101")
	}

	return envseed.Sync(ctx, envseed.SyncOptions{
		InputPath:  inputPath,
		OutputPath: outputPath,
		Force:      force,
		DryRun:     dryRun,
		Quiet:      quiet,
		Stdout:     os.Stdout,
		Stderr:     os.Stderr,
	})
}

func runDiff(ctx context.Context, args []string) error {
	var outputPath string

	fs := flag.NewFlagSet("diff", flag.ContinueOnError)
	fs.StringVar(&outputPath, "output", "", "override the destination path")
	fs.StringVar(&outputPath, "o", "", "override the destination path (shorthand)")
	fs.SetOutput(io.Discard)
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: envseed diff [flags] [INPUT_FILE]\n\nFlags:\n")
		fs.SetOutput(os.Stderr)
		fs.PrintDefaults()
		fs.SetOutput(io.Discard)
	}

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			fs.Usage()
			return exitRequest{code: envseed.ExitOK}
		}
		return envseed.NewExitError("EVE-101-5", err.Error())
	}

	if fs.NArg() > 1 {
		return envseed.NewExitError("EVE-101-6")
	}

	inputPath := ".envseed"
	if fs.NArg() == 1 {
		inputPath = fs.Arg(0)
	}
	if inputPath == "-" {
		return envseed.NewExitError("EVE-101-101")
	}
	result, err := envseed.Diff(ctx, envseed.DiffOptions{
		InputPath:  inputPath,
		OutputPath: outputPath,
		Stdout:     os.Stdout,
		Stderr:     os.Stderr,
	})
	if err != nil {
		return err
	}
	if result.Changed {
		// Differences exist: reserved exit code 1
		return exitRequest{code: 1}
	}
	return nil
}

func runValidate(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("validate", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: envseed validate [INPUT_FILE]\n")
	}

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			fs.Usage()
			return exitRequest{code: envseed.ExitOK}
		}
		return envseed.NewExitError("EVE-101-5", err.Error())
	}

	if fs.NArg() > 1 {
		return envseed.NewExitError("EVE-101-6")
	}

	inputPath := ".envseed"
	if fs.NArg() == 1 {
		inputPath = fs.Arg(0)
	}
	if inputPath == "-" {
		return envseed.NewExitError("EVE-101-101")
	}
	return envseed.Validate(ctx, envseed.ValidateOptions{
		InputPath: inputPath,
	})
}

func runVersion(args []string) error {
	if len(args) > 0 {
		return envseed.NewExitError("EVE-101-4")
	}
	fmt.Fprintln(os.Stdout, version.String())
	return nil
}

func handleError(err error) {
	if err == nil {
		return
	}

	var req exitRequest
	if errors.As(err, &req) {
		os.Exit(req.code)
	}

	var exitErr *envseed.ExitError
	if errors.As(err, &exitErr) {
		fmt.Fprintln(os.Stderr, exitErr.Error())
		os.Exit(exitErr.Code)
	}

	fmt.Fprintf(os.Stderr, "envseed: unexpected error: %v\n", err)
	os.Exit(envseed.ExitInternalError)
}

func containsVersionFlag(args []string) bool {
	for _, arg := range args {
		if arg == "--version" {
			return true
		}
	}
	return false
}

func printUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage: envseed <command> [flags] [INPUT_FILE]")
	fmt.Fprintln(w, "\nCommands:")
	fmt.Fprintln(w, "  sync      Render a template into its .env target")
	fmt.Fprintln(w, "  diff      Compare the current .env file with regenerated output")
	fmt.Fprintln(w, "  validate  Parse the template and report syntax errors")
	fmt.Fprintln(w, "  version   Print the EnvSeed version string")
	fmt.Fprintln(w, "\nGlobal Options:")
	fmt.Fprintln(w, "  --version  Print the EnvSeed version string and exit")
}

type exitRequest struct {
	code int
}

func (e exitRequest) Error() string {
	return ""
}
