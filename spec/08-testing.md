## 8. Conformance and Testing

### 8.1 Deterministic Unit Tests
- CLI default input handling: Suites MUST include cases where `[INPUT_FILE]` is omitted and the default `./.envseed` is used (success) and missing (exit code 102). Suites MUST also include cases where the selected input name does not contain `envseed` without `--output` (exit code 101) and where `--output` is provided.
- Parser rules: MUST cover assignment names, arrays, `+=`, comments (whole-line and trailing), and placeholder syntax (unknown/duplicate/empty modifiers, invalid PATH, presence of newline/NUL).
- Parser rules (sigil strictness): MUST reject any whitespace between `pass` and `:` in `<pass:...>`; diagnostics include line and column (see Sections 4.3 and 4.5).
- Target parsing: MUST verify that target `.env` files (A/B) follow the same grammar (assignments/comments/blank lines). Lines outside this grammar MUST raise parse errors with line and column. No heuristic masking is permitted on parse failure.
- Renderer context rules: MUST cover success/failure, escaping, and interactions of `base64`/`dangerously_bypass_escape`/`allow_*` across contexts (bare/double/single/command/backtick).
- Single-quoted: MUST cover acceptance (no `'`, no newline, TAB only with the `allow_tab` modifier) and rejection (`'` present, newline present, control characters, TAB without permission).
- Error details: MUST verify that context violations and invalid modifier combinations are reported with unique subcodes per Section 5.4, including message/guidance and source position when available. CLI diagnostics for render-time failures MUST include the source line and MUST include the column when tracked (see Section 7.11).

### 8.2 Round-Trip and Re-parse
- On success, `parse -> render -> parse` MUST succeed.
- Idempotence: `parse -> render -> parse -> render` MUST be byte-identical.
- When `dangerously_bypass_escape` is absent, re-parse the output for validation; failures MUST be reported with a dedicated unique subcode.
- Conditional shell validation: When `bash` is available on PATH and: (a) the template does not contain backtick contexts; or (b) the sandbox policy explicitly permits validation, implementations MUST validate with `bash -n`. When these conditions are not met, implementations MUST skip this validation and emit an informational log explaining the reason (e.g., bash unavailable or policy constraints).
- For diff, target `.env` parsing for A/B and masked text construction MUST succeed; otherwise processing MUST terminate with a parsing error (no fallback masking).

### 8.3 Property-Based Testing
- Generation (templates): MUST combine line kinds (blank/comment/assignment), operators (`=`/`+=`), value contexts (bare/double/single/command/backtick), nesting (recommended max depth: 3), and boundaries (adjacent placeholders, surrounding literals, line head/tail, immediately before comments).
- Generation (secrets): MUST explore the cross-product of ASCII-allowed sets/space/TAB/CR/LF/CRLF/other control/non-ASCII (multilingual/emoji/zero-width) x modifiers (`allow_tab`/`allow_newline`/`base64`/`dangerously_bypass_escape`), MUST include the C.4.U minimal representative set, and MUST also sample beyond the minimal across the variation axes defined in C.4.U (Property).
- Primary properties: MUST verify successful re-parse, AST equivalence, `bash -n` success (when possible), conformance of render/parse unique subcodes, and one-time resolution per PATH (cache effectiveness).
- Trailing newlines: MUST validate behavior across CR/LF/CRLF per Section 5.1 (default EOF newline normalization) and Section 4 (parser preservation).

### 8.4 Fuzz Corpus
- Failing inputs MUST be minimized and stored under the package-relative path `internal/<package>/testdata/fuzz/<FuzzName>/` in `go test fuzz v1` format. "Package" refers to the Go module path segment that contains the fuzz test (e.g., `internal/parser`, `internal/renderer`).
- Suites MUST maintain fixed seed/iteration baselines to ensure regressions are reproducible.
- For coverage dimensions, implementations MUST consult Appendix C and cover at least the minimal representative set defined there.

### 8.5 Sandbox Execution
- Where tools like bubblewrap are available, implementations MUST evaluate environment variable values and verify they match expectations. When the necessary tools or namespaces are unavailable, this step MUST be skipped with an informational log stating the unavailability and reason.

### 8.6 Resolver Doubles & Failure Injection
- Suites MUST provide doubles that simulate paths such as missing `pass`, `pass show` failure, missing entry, and presence of NUL.
- Suites MUST verify that each PATH is resolved at most once (cache effectiveness).

### 8.7 Test Identifiers in Tests
- Every test MUST include its EVT identifier in the canonical form `[EVT-<S><F><M>-<n>]` within an adjacent comment, to ensure searchability and traceability to this specification (see Section C.2).
- The EVT identifier MUST appear at the end of the comment immediately preceding the test function.

Examples:
- Single: [EVT-101-1]
- Multiple: [EVT-101-1][EVT-101-2]
