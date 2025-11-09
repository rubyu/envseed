//go:build linux && sandbox
// +build linux,sandbox

package sandbox

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

var state struct {
	once sync.Once
	ok   bool
	err  error
}

// Available checks whether a bubblewrap-based sandbox can run on this host.
func Available() (bool, error) {
	state.once.Do(func() {
		_, err := Run("true")
		if err != nil {
			if errors.Is(err, ErrUnsupported) {
				state.ok = false
				state.err = err
				return
			}
			state.ok = false
			state.err = err
			return
		}
		state.ok = true
	})
	return state.ok, state.err
}

// Run executes a bash script inside a restricted bubblewrap sandbox and
// returns stdout.
func Run(script string) (string, error) {
	// Locate bwrap and bash
	if _, err := exec.LookPath("bwrap"); err != nil {
		return "", fmt.Errorf("%w: bwrap not found", ErrUnsupported)
	}
	bashPath, err := exec.LookPath("bash")
	if err != nil {
		return "", fmt.Errorf("bash not found: %w", err)
	}

	// Discover shared library dependencies for bash via ldd.
	libs, err := detectBashLibs(bashPath)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrUnsupported, err)
	}

	// Base sandbox args and minimal filesystem
	args := []string{
		"--unshare-all",
		"--die-with-parent",
		"--dev", "/dev",
		"--proc", "/proc",
		"--tmpfs", "/tmp",
		"--dir", "/bin",
		"--ro-bind", bashPath, "/bin/bash",
		"--clearenv",
		"--setenv", "PATH", "",
		"--setenv", "HOME", "/tmp",
		"--setenv", "LC_ALL", "C",
		"--chdir", "/tmp",
	}

	// Ensure parent directories exist for libraries, then ro-bind libraries.
	dirSet := make(map[string]struct{})
	addDir := func(d string) {
		d = filepath.Clean(d)
		if d == "/" || d == "." || d == "" {
			return
		}
		// add all ancestors: /lib, /lib/x86_64-linux-gnu, ...
		parts := strings.Split(d, "/")
		if len(parts) > 0 && parts[0] == "" {
			parts = parts[1:]
		}
		cur := "/"
		for _, p := range parts {
			if p == "" {
				continue
			}
			if cur == "/" {
				cur = "/" + p
			} else {
				cur = cur + "/" + p
			}
			dirSet[cur] = struct{}{}
		}
	}
	for _, lib := range libs {
		addDir(filepath.Dir(lib))
	}
	// Create dirs in stable order
	var ordered []string
	for d := range dirSet {
		ordered = append(ordered, d)
	}
	// Shallow to deep ordering by path length to avoid bwrap complaints
	// about binding into non-existent parents.
	sort.Slice(ordered, func(i, j int) bool { return len(ordered[i]) < len(ordered[j]) })
	for _, d := range ordered {
		args = append(args, "--dir", d)
	}
	for _, lib := range libs {
		args = append(args, "--ro-bind", lib, lib)
	}

	// Execute script via bash
	args = append(args, "/bin/bash", "-c", script)

	cmd := exec.Command("bwrap", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err = cmd.Run()
	if err != nil {
		stderrMsg := strings.TrimSpace(stderr.String())
		if strings.Contains(stderrMsg, "No permissions to create new namespace") {
			return stdout.String(), fmt.Errorf("%w: %s", ErrUnsupported, stderrMsg)
		}
		if strings.Contains(stderrMsg, "bwrap:") && strings.Contains(stderrMsg, "Permission denied") {
			return stdout.String(), fmt.Errorf("%w: %s", ErrUnsupported, stderrMsg)
		}
		return stdout.String(), fmt.Errorf("bwrap failed: %w (stderr: %s)", err, stderrMsg)
	}
	return stdout.String(), nil
}

// detectBashLibs runs `ldd` on the given bash binary and extracts absolute
// library paths to bind into the sandbox. It verifies each path exists.
func detectBashLibs(bashPath string) ([]string, error) {
	lddPath, err := exec.LookPath("ldd")
	if err != nil {
		return nil, fmt.Errorf("ldd not found")
	}
	cmd := exec.Command(lddPath, bashPath)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("ldd failed: %v (%s)", err, strings.TrimSpace(out.String()))
	}
	var libs []string
	for _, line := range strings.Split(out.String(), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.Contains(line, "not found") {
			// Incomplete runtime; better to refuse than to bind missing paths.
			return nil, fmt.Errorf("dependency missing: %s", line)
		}
		// Patterns:
		//   libtinfo.so.6 => /lib/x86_64-linux-gnu/libtinfo.so.6 (0x...)
		//   /lib64/ld-linux-x86-64.so.2 (0x...)
		var candidate string
		if i := strings.Index(line, "=>"); i >= 0 {
			right := strings.TrimSpace(line[i+2:])
			if strings.HasPrefix(right, "/") {
				candidate = strings.Fields(right)[0]
			}
		} else if strings.HasPrefix(line, "/") {
			candidate = strings.Fields(line)[0]
		}
		if candidate == "" {
			continue
		}
		if _, err := os.Stat(candidate); err == nil {
			libs = append(libs, candidate)
		}
	}
	if len(libs) == 0 {
		return nil, fmt.Errorf("no libraries discovered from ldd output")
	}
	return libs, nil
}
