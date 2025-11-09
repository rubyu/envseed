## 7. CLI Contract
### 7.1 Global CLI Behavior
- In non-dry-run execution, see Section 6.4 for stream and logging policy. The CLI MUST adhere to those rules.
- Informational messages and diagnostics MUST be printed to stderr. With `--quiet`, informational messages are suppressed; errors are not.
- Version semantics: see Section 10.4 for the printed form and rules. In particular, `envseed version` rejects extra args/flags (exit code 101); the global `--version` prints the version string to stdout and exits (exit code 0), ignoring other inputs.

Note: See Section 5.2 for `dangerously_bypass_escape` semantics; security considerations are discussed in Section 6.6.
### 7.2 Commands
- `sync`: render a template and write to the resolved output path.
- `diff`: render in memory and compare against the resolved output file, printing a redacted unified diff.
- `validate`: parse the template and report lexical/syntax errors.
- `version`: print the EnvSeed version string and exit.
- Unknown or missing commands MUST return exit code 101.

### 7.3 Arguments
```
envseed <command> [flags] <INPUT_FILE>
```

#### 7.3.1 Input File Requirements
- `<INPUT_FILE>` MUST be provided for `sync`, `diff`, and `validate`. The `version` subcommand MUST NOT accept an input file.
- `<INPUT_FILE>` MUST be a path to a file. The file MUST be readable, and MUST be formatted in the template format (see Section 4).
- File name rules: see Section 7.5 (derivation and directory semantics). This requirement does not apply to `validate`.
- Reading from stdin MUST NOT be supported.

### 7.4 Common Options
- `--output`, `-o` `<PATH>` (sync, diff): specify write/compare destination. If omitted, replace the first occurrence of `envseed` in `<INPUT_FILE>` with `env` to derive the path.
- Option combinations: unless a subcommand explicitly lists an unsupported combination, options MUST be combinable. Unsupported combinations MUST return exit code 101 with an explanatory message.
- `--version` (global): see Section 10.4.

### 7.5 Path Resolution
1) Choose candidate: use `--output` when provided; otherwise replace the first `envseed` in `<INPUT_FILE>` with `env` to form the candidate path.
2) Interpret directories: if the candidate ends with a path separator or resolves to an existing directory, join the derived file name (`envseed` -> `env`).
3) Validate path: a missing parent directory MUST return exit code 106; a candidate that resolves to an existing directory MUST return exit code 101. Non-existent files are valid targets.

Implementations MUST compute absolute paths when emitting paths (e.g., the `target:` header and unified diff headers), using OS-level absolutization without resolving symbolic links.


Diagnostics: see Section 7.11 for the canonical display format. The exit codes for the cases below remain as specified.

#### 7.5.1 Path Resolution Properties
- Across combinations of: whether the input name contains `envseed`; presence or absence of `--output`; whether the candidate ends as a directory, a file, or is non-existent; and presence or absence of a trailing path separator, the following MUST be validated.
  - Without `--output`: derive the candidate path by replacing the first occurrence of `envseed` in the input name with `env`. If the parent directory does not exist, implementations MUST return exit code 106. If attempting to treat an existing directory as a file, implementations MUST return exit code 101.
  - With `--output`: when the path ends with a separator or resolves to an existing directory, append the derived file name (`envseed` -> `env`); otherwise use the provided path as-is. Any inconsistency MUST be classified under exit code 106 or 101. Subcode assignment follows `docs/errors.md`.

### 7.6 Target .env Parsing Requirements
Apply newline and whitespace definitions from Section 1.2 and Appendix D.1.
Parsing MUST follow Appendix D.1–D.3 (same as the template grammar). Target `.env` files do not contain placeholders; only assignments, comments, and blank lines are valid. Non-ASCII whitespace where grammar-level whitespace is expected (Space/Tab-only; see Appendix D.1) MUST cause a parse error (exit code 107). See `docs/errors.md` for subcode mapping.
Lines that are not assignments, comments, or blank lines (e.g., shell commands such as `export VAR=...`) MUST be rejected as parse errors.
- If a target `.env` file (A or B) cannot be parsed according to this grammar, processing MUST terminate with a parsing error (see Section 7.10; exit code 107). Fallback or heuristic masking MUST NOT be used.

- Array index notation and invalid cases
  - Valid: `ARR[0]=x` succeeds syntactically and at runtime (`${ARR[0]}` is `x`).
  - Invalid: `ARR[=x` yields `unexpected EOF while looking for matching ']'` under `bash -n` (syntax error).

- Treatment of `+` in names (Bash syntax facts)
  - `X+Y=1` is not an `assignment_word` in Bash but a normal command word; `bash -n` succeeds and execution yields `command not found`. Implementations MUST treat this as a parse error for “invalid assignment name”. This specification is stricter than Bash.

- Handling of a missing `=` (Bash syntax facts)
  - `VAR value` is not an assignment but a command sequence (`VAR` is interpreted as a command name). `bash -n` succeeds and execution yields `command not found`. Implementations MUST raise a parse error “assignment missing '='”. This specification is stricter than Bash.

### 7.7 sync
```
envseed sync [flags] <INPUT_FILE>
```
Behavior:
- Read input; resolve output per Section 7.5; fetch secrets via `pass`; render and write the `.env` file.

Options:
- `--force`, `-f`: allow overwriting an existing output file.
- `--dry-run`: render without writing; report target path and redacted content.
- `--quiet`, `-q`: suppress informational logs; errors remain visible.

Output and streams:
- Output file permissions are always `0600`. Writing is atomic: data is written to a temporary file and renamed.
- When content is unchanged, the CLI MUST emit `wrote <path> (unchanged)` to stderr (unless `--quiet`).
- If content changes and write succeeds, emit `wrote <path> (mode 0600)` to stderr (suppressed by `--quiet`).
- See Section 7.1 for output and stream requirements.

Dry-run details:
- Never write files. Always resolve the effective output path (per Section 7.5) and include it in the report, regardless of whether `--output` is provided. The path MUST be absolute (see Section 7.5 for the definition of absolute path).
- The first line of the dry-run report MUST be `target: <path>`, where `<path>` is the absolute effective output path resolved per Section 7.5. Implementations MUST NOT add prefixes, quotes, or annotations to `<path>`.
- Print a redacted content summary to stdout using the rules in Section 6.3 (masked texts A′/B′). Real secrets MUST NOT be printed.

Example (informative; mask uses `*`):
```
target: /abs/path/to/.env
FOO=****
BAR=**********
# Example (informative): with B rendered as `PASSWORD="vP9%cQ\$m*Nqk"`,
# the masked B′ is `PASSWORD="************"` (no backslashes appear in the masked value).
```

Exit codes:
- Refusing to overwrite without `--force` MUST use exit code 106. Other failures follow Section 7.10.

### 7.8 diff
```
envseed diff [flags] <INPUT_FILE>
```
Behavior:
- Render the same content as `sync` in memory. Compute a unified diff on raw A (current file) and raw B (newly rendered), and emit a reconstructed masked unified diff on stdout using A′/B′ from Section 6.3.

Options:
- `--output`, `-o`: select the comparison target without affecting the template read path.

Limits:
- Comparisons larger than 10 MiB MUST be rejected to constrain memory usage (10 MiB = 10 × 1,048,576 bytes). Exceeding the limit returns exit code 108.
  Informative: Exit code 108 here denotes diff-specific resource/size constraints; it does not imply a generic filesystem error.

Details:
- If the target file is missing, compare against empty content, resulting in an all-additions diff.
- The unified diff headers MUST be the first two lines `--- <path>` and `+++ <path>`. Each `<path>` MUST be the absolute output path (see Section 7.5). The two `<path>` values MUST be byte-identical. Implementations MUST NOT add prefixes or annotations to these header paths. Rationale (Informative): enforcing identical header paths improves interoperability and machine readability of unified diffs.
- The unified diff body MUST select context and deletion lines from A′ and addition lines from B′ using the hunk line numbers from the raw diff. Preserve hunk ordering and metadata. No secret values may appear in the final output.
- When there are no differences, both stdout and stderr MUST remain silent unless errors occur.
- Path handling for `--output` MUST follow Section 7.5 (derivation and directory semantics).

Exit codes:
- `0` when files match; `1` when differences exist. All other failures use the error code ranges defined in Section 7.10 (101+) and MUST NOT use `1`.

### 7.9 validate
```
envseed validate [flags] <INPUT_FILE>
```
Behavior:
- Perform parsing only. Do not call `pass`. Do not read or write files other than `<INPUT_FILE>`.

Options:
- No additional flags are accepted. Unexpected options MUST return `101`.

Streams and exit codes:
- Success is silent by default. Errors are printed to stderr.
- Exit code 0 on success; exit code 103 on parse/lex errors.
- Example error: unterminated double-quote (exit code 103).

### 7.10 Exit Codes
The exit status space is partitioned as follows:

```
0    Success
1    Differences exist (diff only)
101+ All other errors (see categories below)
```

Error categories (101+):

```
101 Invalid arguments or unknown/missing command
102 Template read failure (I/O)
103 Template parsing failure (.envseed -> AST)
104 Resolver failures (missing `pass` binary; `pass show` I/O failure; entry not found; value contains NUL)
105 Rendering failures (context/modifier issues) and post-render re-parse failure
106 Output failures (sync write I/O: path preconditions, tmp write, rename, chmod, dry-run write)
107 Target parsing failure (.env for A/B)
108 Diff failures (size limits and diff I/O)
199 Unexpected internal exception
```

Rationale:
- Exit codes 1-99 are reserved for future cross-tool alignment. In particular, exit code 1 is reserved for "differences exist" to align with common diff semantics.
- All non-success, non-diff failures start at 101 to avoid collisions with other tools that may standardize 1-99.
- Exit code 100 is not used.
 - Note: The reservation above applies to exit codes. Subcodes use bands where B0 is 1..99 within each exit category; this is independent of exit-code reservations.

Diagnostic label scheme:
- The canonical display format and label structure are defined in Section 7.11 (CLI Diagnostics). This section defines exit codes and band classification only.


- `diff`: differences return exit code 1; matches return exit code 0. Exit code 1 MUST NOT be used for any other condition.
- Errors MUST be assigned a unique subcode, and include a clear message, optionally followed by guidance and a documentation link. The canonical subcode mapping (numbering, messages, guidance) is generated from `internal/envseed/errors.go` to `docs/errors.md`. The diagnostic display format is defined in Section 7.11.
- Render-time failures (exit code 105) MUST include the source position in CLI diagnostics when available: the line number MUST be included; the column MUST be included when tracked. The position MUST be anchored to the offending placeholder token (Section 7.11). Diagnostics MUST NOT reveal secrets.
 - Refusing to overwrite an existing file without `--force` is classified under exit code 106 (Output failure).

### 7.10.1 Subcode Bands
This section defines the band allocation for subcodes within each exit category. Band allocation is Normative. The canonical mapping of individual subcodes (numbers, messages, guidance) is generated from `internal/envseed/errors.go` to `docs/errors.md` (Informative).

- 101 CLI / Input & Path Resolution
  - EVE-101-B0 (1..99) — Command/flag validation (missing/unknown/extra/unsupported combination)
  - EVE-101-B1 (101..199) — Input channel (stdin unsupported)
  - EVE-101-B2 (201..299) — Input file validation (missing/not specified/name requirements)
  - EVE-101-B3 (301..399) — Path resolution logic (using directory as file, etc.)

- 102 Template Read (I/O)
  - EVE-102-B0 (1..99) — open/read failures
  - EVE-102-B1 (101..199) — Permission errors
  - EVE-102-B2 (201..299) — Input path is a directory, etc.

- 103 Parsing (Parser -> AST)
  - EVE-103-B0 (1..99) — Lexical & sigil constraints (non-ASCII whitespace around placeholder separators `|`, `,`, before `>`, trimming around PATH; whitespace between `pass` and `:`; non-ASCII leading whitespace at line start)
  - EVE-103-B1 (101..199) — Assignment structure (name/operator/= / non-assignment input)
  - EVE-103-B2 (201..299) — Placeholder body/sigil (empty PATH/newline/NUL)
  - EVE-103-B3 (301..399) — Modifiers (missing/unknown/empty/duplicate/non-ASCII whitespace/NUL)
  - EVE-103-B4 (401..499) — Unterminated quotes/substitutions (double/single/backtick/`$(...)`)
  - EVE-103-B5 (501..599) — Indexing (mismatched brackets, etc.)

- 104 Resolver (pass)
  - EVE-104-B0 (1..99) — `pass` not installed
  - EVE-104-B1 (101..199) — `pass show` failure (non-missing entry)
  - EVE-104-B2 (201..299) — Missing `pass` entry
  - EVE-104-B3 (301..399) — Value contains unsupported characters (e.g., NUL)

- 105 Rendering + Re-parse Validation
  - EVE-105-B0 (1..99) — General placeholder-constraint failure
  - EVE-105-B1 (101..199) — Single-quoted context
  - EVE-105-B2 (201..299) — Double-quoted context
  - EVE-105-B3 (301..399) — Command substitution `$(...)`
  - EVE-105-B4 (401..499) — Backtick context
  - EVE-105-B5 (501..599) — Bare context
  - EVE-105-B6 (601..699) — Invalid modifier combination
  - EVE-105-B7 (701..799) — Post-render re-parse validation failure (when bypass is not used)

- 106 Output (sync write: I/O)
  - EVE-106-B0 (1..99) — Preconditions/path (missing parent/inaccessible/not a directory/stat failure)
  - EVE-106-B1 (101..199) — Existing output handling (`--force` missing/read existing/chmod failure)
  - EVE-106-B2 (201..299) — Temporary file (create/chmod/write/close)
  - EVE-106-B3 (301..399) — Finalize (rename/final chmod)
  - EVE-106-B4 (401..499) — Reporting (dry-run write failure)

- 107 Target .env Parsing (A/B)
  - EVE-107-B0 (1..99) — Unexpected line (not assignment/comment/blank)
  - EVE-107-B1 (101..199) — Non-ASCII whitespace
  - EVE-107-B2 (201..299) — Parse failure (generic)

- 108 Diff (comparison)
  - EVE-108-B0 (1..99) — Size limit exceeded (10 MiB); Diff generation failure; Diff output write failure
  - Note: Differences themselves are reported with exit code 1 (not an error)

- 199 Diagnostics/Internal Exceptions (cross-cutting)
  - EVE-199-B0 (1..99) — Internal invariant violations (e.g., redaction rendering failure, resolver used after close)
  - EVE-199-B1 (101..199) — Runtime panic/unexpected state bridging (as needed)

### 7.10.2 Subcode Assignment Principles
- Uniqueness: every distinct failure condition is assigned a unique subcode under its exit category and band.
- Band density: within a band, subcodes are allocated densely starting at the band start; reserves and gaps should be avoided. When categories are reorganized prior to release, subcodes may be renumbered to restore density.
- Ownership and synchronization: the canonical mapping (numbers, messages, guidance) is maintained in the generator source (`internal/envseed/errors.go`) and emitted to `docs/errors.md` via `go generate`.
- User-facing exposure: bands are not shown in CLI diagnostics. User-facing labels use concrete codes only (see Section 7.11).

### 7.10.3 Assignment Flow (Informative)
1) Classify the failure into an exit category.
2) Select the most specific band within that exit.
3) Allocate the next available subcode within the band.
4) Update the generator (`internal/envseed/errors.go`).
5) Run `go generate` to refresh `docs/errors.md`.

### 7.10.4 Examples (Informative)
These examples illustrate band selection and concrete-code assignment. Specific numbers are illustrative only; the canonical mapping is in `docs/errors.md`.
- Non-ASCII whitespace around a placeholder separator -> classify under `EVE-103-B0` -> concrete code e.g., `EVE-103-12`.
- Missing parent directory for output -> classify under `EVE-106-B0` -> concrete code e.g., `EVE-106-3`.
- Post-render re-parse failure (bypass not used) -> classify under `EVE-105-B7` -> concrete code e.g., `EVE-105-702`.

### 7.11 Diagnostics (CLI Diagnostic Display)
- The canonical label format MUST be `envseed ERROR [EVE-<exit>-<subcode>]: <message>`.
- Render-time failures (exit code 105) MUST include source position in CLI diagnostics when available from the renderer/parser. The position is anchored to the offending placeholder token (Sections 3, 4) and MUST include the line number. The column MUST be included when tracked.
- Implementations MUST propagate available source position from the render layer to the CLI error output. Do not fabricate positions when unavailable.
- Diagnostics MUST NOT reveal secrets (Sections 6.1, 6.4). Paths and messages MUST be free of secret values.
- Examples (Informative):
  - `envseed ERROR [EVE-105-<subcode>]: <message>: line N[, column M]: placeholder <PATH>`
  - `envseed ERROR [EVE-105-<subcode>]: <message>\nAt: line N[, column M], placeholder <PATH>`
  - When including placeholder strings in diagnostics (e.g., `<pass:...>`), the placeholder string MUST be masked and MUST NOT be emitted verbatim.
