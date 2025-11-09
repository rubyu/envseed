package testsupport

import (
	"bufio"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// CorpusSeed represents a single fuzz corpus entry.
type CorpusSeed struct {
	Dir       string
	File      string
	Seed      int64
	Iteration uint32
}

var (
	reInt64    = regexp.MustCompile(`^\s*int64\(\s*([+-]?\d+)\s*\)\s*$`)
	reUint32   = regexp.MustCompile(`^\s*uint32\(\s*([+-]?\d+)\s*\)\s*$`)
	reByteLike = regexp.MustCompile(`^\s*(byte|uint8)\(\s*(.+?)\s*\)\s*$`)
)

// LoadCorpusSeeds loads seeds from the package‑local corpus directory only
// (internal/<package>/testdata/fuzz/<FuzzName>/). Missing directories are
// skipped quietly so packages without a corpus do not fail.
func LoadCorpusSeeds(fuzzName string) ([]CorpusSeed, error) {
	// Package-local corpus only: internal/<package>/testdata/fuzz/<FuzzName>/
	dirs := []string{filepath.Join("testdata", "fuzz", fuzzName)}

	seenDirs := make(map[string]struct{})
	var seeds []CorpusSeed
	for _, dir := range dirs {
		dir = filepath.Clean(dir)
		if dir == "" {
			continue
		}
		if _, ok := seenDirs[dir]; ok {
			continue
		}
		seenDirs[dir] = struct{}{}
		entries, err := os.ReadDir(dir)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("read corpus directory %s: %w", dir, err)
		}
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			path := filepath.Join(dir, entry.Name())
			seed, iter, err := ReadCorpusInt64Uint32(path)
			if err != nil {
				return nil, fmt.Errorf("parse corpus %s: %w", path, err)
			}
			seeds = append(seeds, CorpusSeed{
				Dir:       dir,
				File:      entry.Name(),
				Seed:      seed,
				Iteration: iter,
			})
		}
	}
	sort.Slice(seeds, func(i, j int) bool {
		if seeds[i].Dir == seeds[j].Dir {
			return seeds[i].File < seeds[j].File
		}
		return seeds[i].Dir < seeds[j].Dir
	})
	return seeds, nil
}

// ReadCorpusInt64Uint32 reads a corpus file storing an int64 seed and a uint32
// iteration number in the "go test fuzz v1" format.
func ReadCorpusInt64Uint32(path string) (int64, uint32, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, 0, err
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 1024), 1024*1024)

	if !sc.Scan() {
		return 0, 0, errors.New("empty corpus file")
	}
	header := strings.TrimSpace(sc.Text())
	if header != "go test fuzz v1" {
		return 0, 0, fmt.Errorf("unexpected header: %q", header)
	}

	if !sc.Scan() {
		return 0, 0, errors.New("missing int64 line")
	}
	seed, err := parseInt64Line(sc.Text())
	if err != nil {
		return 0, 0, fmt.Errorf("parse int64: %w", err)
	}

	if !sc.Scan() {
		return 0, 0, errors.New("missing uint32 line")
	}
	iteration, err := parseUint32Line(sc.Text())
	if err != nil {
		return 0, 0, fmt.Errorf("parse uint32: %w", err)
	}

	if sc.Scan() {
		return 0, 0, fmt.Errorf("unexpected extra line: %q", sc.Text())
	}
	if err := sc.Err(); err != nil {
		return 0, 0, err
	}

	return seed, iteration, nil
}

func parseInt64Line(line string) (int64, error) {
	m := reInt64.FindStringSubmatch(line)
	if m == nil {
		return 0, fmt.Errorf("line %q not in int64(...) form", line)
	}
	return strconv.ParseInt(m[1], 10, 64)
}

func parseUint32Line(line string) (uint32, error) {
	if m := reUint32.FindStringSubmatch(line); m != nil {
		if strings.HasPrefix(m[1], "-") {
			return 0, fmt.Errorf("uint32 cannot be negative in %q", line)
		}
		v, err := strconv.ParseUint(m[1], 10, 32)
		if err != nil {
			return 0, err
		}
		return uint32(v), nil
	}
	if m := reByteLike.FindStringSubmatch(line); m != nil {
		v, err := parseByteLikeValue(m[2])
		if err != nil {
			return 0, err
		}
		return uint32(v), nil
	}
	return 0, fmt.Errorf("line %q not in uint32(...) form", line)
}

func parseByteLikeValue(inner string) (uint8, error) {
	if len(inner) >= 2 && inner[0] == '\'' && inner[len(inner)-1] == '\'' {
		s, err := strconv.Unquote(inner)
		if err != nil {
			return 0, fmt.Errorf("unquote %q: %w", inner, err)
		}
		runes := []rune(s)
		if len(runes) != 1 {
			return 0, fmt.Errorf("expected one rune, got %d", len(runes))
		}
		return uint8(runes[0]), nil
	}
	if strings.HasPrefix(inner, "0x") || strings.HasPrefix(inner, "0X") {
		v, err := strconv.ParseUint(inner[2:], 16, 8)
		if err != nil {
			return 0, err
		}
		return uint8(v), nil
	}
	v, err := strconv.ParseUint(inner, 10, 8)
	if err != nil {
		return 0, err
	}
	return uint8(v), nil
}
