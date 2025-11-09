package envseed

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func writeOutput(path string, content []byte, quiet bool, force bool, stderr io.Writer) error {
	dir := filepath.Dir(path)
	info, err := os.Stat(path)
	exists := err == nil
	if err != nil && !os.IsNotExist(err) {
		return NewExitError("EVE-106-4", path).WithErr(err)
	}

	if exists && info.IsDir() {
		return NewExitError("EVE-101-301", path)
	}

	if exists {
		existing, rerr := os.ReadFile(path)
		if rerr != nil {
			return NewExitError("EVE-106-102", path).WithErr(rerr)
		}
		if bytes.Equal(existing, content) {
			if err := os.Chmod(path, 0o600); err != nil {
				return NewExitError("EVE-106-103", path).WithErr(err)
			}
			if !quiet {
				fmt.Fprintf(stderr, "wrote %s (unchanged)\n", path)
			}
			if info.Mode().Perm() != 0o600 && !quiet {
				fmt.Fprintf(stderr, "chmod %s -> 0600\n", path)
			}
			return nil
		}
		if !force {
			return NewExitError("EVE-106-101", path)
		}
	}

	tmp, err := os.CreateTemp(dir, ".envseed-*")
	if err != nil {
		return NewExitError("EVE-106-201", dir).WithErr(err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)

	if err := tmp.Chmod(0o600); err != nil {
		tmp.Close()
		return NewExitError("EVE-106-202", tmpName).WithErr(err)
	}

	if _, err := tmp.Write(content); err != nil {
		tmp.Close()
		return NewExitError("EVE-106-203", tmpName).WithErr(err)
	}

	if err := tmp.Close(); err != nil {
		return NewExitError("EVE-106-204", tmpName).WithErr(err)
	}

	if err := os.Rename(tmpName, path); err != nil {
		return NewExitError("EVE-106-301", tmpName, path).WithErr(err)
	}

	if err := os.Chmod(path, 0o600); err != nil {
		return NewExitError("EVE-106-302", path).WithErr(err)
	}

	if !quiet {
		fmt.Fprintf(stderr, "wrote %s (mode 0600)\n", path)
	}
	if exists && info.Mode().Perm() != 0o600 && !quiet {
		fmt.Fprintf(stderr, "chmod %s -> 0600\n", path)
	}
	return nil
}

func readFileIfExists(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, NewExitError("EVE-106-102", path).WithErr(err)
	}
	return data, nil
}

// unifiedDiff moved to diff_util.go
