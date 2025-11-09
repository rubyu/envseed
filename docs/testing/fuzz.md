# Fuzz Testing Guidelines: Reproducible Tests and Exploratory Fuzzing

This guide explains how to introduce fuzz testing so that regular tests always replay the fuzz corpus, while exploratory fuzzing remains optional. It is self‑contained and does not rely on any existing files. The code templates are minimal and easy to adapt.

Note: The example below uses a very simple `Sum(a, b int64) int64` function to keep the concepts clear. Replace it with your own target function as needed. All code comments are written in English.

## Goals
- Regular tests always run two layers: deterministic baselines and corpus replay. This catches regressions reliably.
- Exploratory fuzzing is run on demand. When it finds a failure, minimize the input and add it to the corpus.
- From then on, regular tests reproduce the failure deterministically.

## Core Principles
- Deterministic baselines: run property checks over fixed seeds and iteration counts to cover broad behavior quickly.
- Corpus replay guarantee: regular tests read package‑local corpora under `internal/<package>/testdata/fuzz/<FuzzName>/` (in the `go test fuzz v1` format) and always execute those cases.
- Separation of concerns: `go test -fuzz` is optional (local/nightly). CI remains stable by running only baselines and corpus replay.
- Stable runtime: cap work via an iteration limit (clamp) and keep the core property lightweight (for example, render → parse only).

## Naming and Layout
- Fuzz function name: `Fuzz<Subject>` (for example, `FuzzSum`).
- Corpus location: `internal/<package>/testdata/fuzz/<FuzzName>/` (files in `go test fuzz v1` format).
- Subtest naming: `corpus/<filename>` so a single case is easy to target.

## Template 1: Deterministic baselines + property (Sum)
```go
package pkg_test

import (
    "math"
    "math/rand"
    "testing"
)

// Deterministic loop config
type loopPlan struct {
	Seed       int64
	Iterations uint32
}

type meta struct {
    Seed      int64
    Iteration uint32
}

// Subject under test (replace with your own)
func Sum(a, b int64) int64 { return a + b }

// Detect overflow for int64 addition
func wouldOverflowInt64Add(a, b int64) bool {
    if b > 0 && a > math.MaxInt64-b { return true }
    if b < 0 && a < math.MinInt64-b { return true }
    return false
}

// Generate a random pair in a modest range
func randomPair(r *rand.Rand) (int64, int64) {
    const lim = int64(1_000_000_000)
    a := r.Int63n(2*lim+1) - lim
    b := r.Int63n(2*lim+1) - lim
    return a, b
}

// Properties for Sum:
//  - Commutativity: Sum(a, b) == Sum(b, a)
//  - Identity:     Sum(a, 0) == a and Sum(0, b) == b
//  - Inverse:      Sum(a, -a) == 0 (when it does not overflow)
func checkSumProperties(t *testing.T, m meta, a, b int64) {
    if wouldOverflowInt64Add(a, b) {
        return // skip cases that would overflow
    }
    ab := Sum(a, b)
    ba := Sum(b, a)
    if ab != ba {
        t.Fatalf("seed=%d iter=%d commutativity failed: %d+%d=%d vs %d+%d=%d", m.Seed, m.Iteration, a, b, ab, b, a, ba)
    }
    if Sum(a, 0) != a {
        t.Fatalf("seed=%d iter=%d identity failed: a+0 != a (a=%d)", m.Seed, m.Iteration, a)
    }
    if Sum(0, b) != b {
        t.Fatalf("seed=%d iter=%d identity failed: 0+b != b (b=%d)", m.Seed, m.Iteration, b)
    }
    if !wouldOverflowInt64Add(a, -a) && Sum(a, -a) != 0 {
        t.Fatalf("seed=%d iter=%d inverse failed: a+(-a) != 0 (a=%d)", m.Seed, m.Iteration, a)
    }
}

func Test_Sum_Baseline(t *testing.T) {
    plans := []loopPlan{
        {Seed: 3141592653, Iterations: 256},
        {Seed: -2718281828, Iterations: 256},
    }
    for i, p := range plans {
		p := p
		name := "baseline"
		if len(plans) > 1 {
			name = name + "/" + string(rune('A'+i))
		}
		t.Run(name, func(t *testing.T) {
            r := rand.New(rand.NewSource(p.Seed))
            for it := uint32(0); it < p.Iterations; it++ {
                a, b := randomPair(r)
                checkSumProperties(t, meta{Seed: p.Seed, Iteration: it}, a, b)
            }
        })
    }
}
```

## Template 2: Exploratory fuzzing (optional, Sum)
```go
package pkg_test

import (
    "math"
    "math/rand"
    "testing"
)

// Bound exploration work to keep runs predictable
func clamp(v, limit uint32) uint32 {
	if limit == 0 {
		return v
	}
	return v % limit
}

func FuzzSum(f *testing.F) {
    // Seed with a few diverse starting points
    f.Add(int64(0), int64(0))
    f.Add(int64(1), int64(-1))
    f.Add(math.MaxInt64, int64(-1))
    f.Add(math.MinInt64, int64(1))

    f.Fuzz(func(t *testing.T, a int64, b int64) {
        if wouldOverflowInt64Add(a, b) {
            return
        }
        checkSumProperties(t, meta{Seed: 0, Iteration: 0}, a, b)
    })
}
```

## Template 3: Corpus replay in regular tests (Sum)
```go
package pkg_test

import (
    "bufio"
    "errors"
    "os"
    "path/filepath"
    "regexp"
    "sort"
    "strconv"
    "strings"
    "testing"
)

// Minimal reader for "go test fuzz v1" corpus with two int64 arguments
var reInt64 = regexp.MustCompile(`^\s*int64\(\s*([+-]?\d+)\s*\)\s*$`)

func readCorpusV1Int64Int64(path string) (int64, int64, error) {
    f, err := os.Open(path)
    if err != nil {
        return 0, 0, err
    }
    defer f.Close()
    sc := bufio.NewScanner(f)
    if !sc.Scan() {
        return 0, 0, errors.New("empty corpus")
    }
    if strings.TrimSpace(sc.Text()) != "go test fuzz v1" {
        return 0, 0, errors.New("bad header")
    }
    if !sc.Scan() {
        return 0, 0, errors.New("missing first int64 line")
    }
    m := reInt64.FindStringSubmatch(sc.Text())
    if m == nil {
        return 0, 0, errors.New("bad int64 line (first)")
    }
    a, err := strconv.ParseInt(m[1], 10, 64)
    if err != nil {
        return 0, 0, err
    }
    if !sc.Scan() {
        return 0, 0, errors.New("missing second int64 line")
    }
    m2 := reInt64.FindStringSubmatch(sc.Text())
    if m2 == nil {
        return 0, 0, errors.New("bad int64 line (second)")
    }
    b, err := strconv.ParseInt(m2[1], 10, 64)
    if err != nil {
        return 0, 0, err
    }
    if sc.Scan() {
        return 0, 0, errors.New("extra lines")
    }
    return a, b, nil
}

func Test_Sum_Corpus(t *testing.T) {
    dir := filepath.Join("testdata", "fuzz", "FuzzSum")
    ents, err := os.ReadDir(dir)
    if err != nil && errors.Is(err, os.ErrNotExist) {
        t.Skip("no corpus yet")
    }
    if err != nil {
        t.Fatalf("read corpus: %v", err)
	}
	var files []string
	for _, e := range ents {
		if !e.IsDir() {
			files = append(files, e.Name())
		}
	}
	sort.Strings(files)
	for _, name := range files {
        name := name
        t.Run("corpus/"+name, func(t *testing.T) {
            a, b, err := readCorpusV1Int64Int64(filepath.Join(dir, name))
            if err != nil {
                t.Fatalf("parse corpus: %v", err)
            }
            checkSumProperties(t, meta{Seed: 0, Iteration: 0}, a, b)
        })
    }
}
```

## How to run
- Regular tests (always run baselines and corpus replay):
  - `go test ./...`
- Exploratory fuzzing (optional):
  - Single fuzz target: `go test -run=^$ -fuzz=FuzzSum -fuzztime=1m .`
  - Multiple targets: script it and control total time via `-fuzztime`.

## Minimal policy
- When fuzzing finds a failure, minimize it and store it under `internal/<package>/testdata/fuzz/<FuzzName>/`.
- Regular tests must always replay the corpus to catch regressions.
- Keep the corpus small and non‑duplicative so test time stays predictable.
