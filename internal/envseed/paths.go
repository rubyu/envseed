package envseed

// Path utilities for output derivation and validation.
// This file holds:
//  - resolveOutputPath / deriveOutputFilename: derive target path from input
//  - validateOutputPath: ensure the output path points to a regular file
// If responsibilities grow, consider splitting into paths_derive.go and
// paths_validate.go to keep concerns clear.

import (
	"os"
	"path/filepath"
	"strings"
)

func resolveOutputPath(input, explicit string) (string, error) {
	var candidate string
	if explicit == "" {
		if !strings.Contains(input, "envseed") {
			return "", NewExitError("EVE-101-203", input)
		}
		candidate = strings.Replace(input, "envseed", "env", 1)
	} else {
		if strings.HasSuffix(explicit, string(os.PathSeparator)) {
			dir := strings.TrimSuffix(explicit, string(os.PathSeparator))
			if dir == "" {
				dir = explicit
			}
			info, err := os.Stat(dir)
			if err != nil {
				if os.IsNotExist(err) {
					return "", NewExitError("EVE-106-1", dir).WithErr(err)
				}
				return "", NewExitError("EVE-106-2", dir).WithErr(err)
			}
			if !info.IsDir() {
				return "", NewExitError("EVE-106-3", dir)
			}
			candidate = filepath.Join(dir, deriveOutputFilename(input))
		} else {
			info, err := os.Stat(explicit)
			switch {
			case err == nil && info.IsDir():
				candidate = filepath.Join(explicit, deriveOutputFilename(input))
			case err == nil:
				candidate = explicit
			case os.IsNotExist(err):
				candidate = explicit
			default:
				return "", NewExitError("EVE-106-4", explicit).WithErr(err)
			}
		}
	}
	abs, err := filepath.Abs(candidate)
	if err != nil {
		return "", NewExitError("EVE-106-4", candidate).WithErr(err)
	}
	return abs, nil
}

func deriveOutputFilename(input string) string {
	name := filepath.Base(input)
	if strings.Contains(name, "envseed") {
		return strings.Replace(name, "envseed", "env", 1)
	}
	return name
}

func validateOutputPath(path string) error {
	dir := filepath.Dir(path)
	info, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return NewExitError("EVE-106-1", dir).WithErr(err)
		}
		return NewExitError("EVE-106-2", dir).WithErr(err)
	}
	if !info.IsDir() {
		return NewExitError("EVE-106-3", dir)
	}
	if finfo, ferr := os.Stat(path); ferr == nil && finfo.IsDir() {
		return NewExitError("EVE-101-301", path)
	} else if ferr != nil && !os.IsNotExist(ferr) {
		return NewExitError("EVE-106-4", path).WithErr(ferr)
	}
	return nil
}
