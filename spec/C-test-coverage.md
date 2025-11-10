## Appendix C. Test Coverage Checklist

### C.1 Overview (Informative)
Appendix C provides a complete testing coverage map for this specification. Coverage is organized by scope (module-level vs. broader-scope), then by test family (G/P/E/W/R/U/N/C/I/S/D/Z), and finally by method (Unit, Property, Fuzz). Each test item refers to the relevant normative sections instead of restating details inline.

This appendix is normative overall. Explanatory background may be explicitly labeled Informative. Before using the per-family lists in C.4 and C.5, read C.2 (Identifiers & Conventions) to understand the identifier format, cross-cutting rules, and reporting conventions. Property-based and fuzz tests are subject to implementation guidelines, and the requirement to consult and follow these guidelines is specified in C.2.5.

Structure:
- C.2 Identifiers & Conventions
- C.3 Taxonomy and Notation (Informative)
- C.4 Module-Level Tests
- C.5 Broader-Scope Tests

### C.2 Identifiers & Conventions
#### C.2.1 Format and Codes
This specification assigns a stable, semantic identifier to every test item in Appendix C using the format `EVT-<S><F><M>-<n>`.

- Scope code `<S>`:
  - `M` = Module-Level tests (Section C.4)
  - `B` = Broader-Scope tests (Section C.5)

- Family code `<F>`:
  - `G` (Grammar and AST), `P` (Placeholders and Modifiers), `E` (Context and Escaping), `W` (Whitespace and Newlines), `R` (Round-Trip and Idempotence), `U` (Unicode), `N` (Nesting and Boundaries), `C` (CLI and UX), `I` (I/O and Path), `S` (Security and Redaction), `D` (Diagnostics and Error Mapping), `Z` (Resolver and Caching)

- Method code `<M>`:
  - `U` (Unit), `P` (Property), `F` (Fuzz)

- Counter `<n>`:
  - For each unique triple `<S,F,M>`, numbering starts at 1 and increments by 1 in document order.

#### C.2.2 Stability & Migration
- Identifiers MUST NOT be reused. Minor wording edits or reordering items MUST NOT change identifiers.
- Moving an item across scope or family REQUIRES minting a new identifier. The previous identifier is retired and MUST remain unused thereafter.
- Editorial and change-control rule: do not renumber existing items. Append new items at the end of the relevant `<S,F,M>` group and allocate the next counter value.

#### C.2.3 Usage Requirements
- Specification: Each item in Appendix C MUST start with its EVT identifier in the canonical form:
  `EVT-<S><F><M>-<n> - <concise item title>`
- Tests: Implementations MUST include the corresponding EVT identifier in an adjacent comment so failures can be searched and traced to this specification.
- For compact formatting, implementers MAY wrap identifiers as "[EVT-…]" (see Section C.2.3.1). Both forms are equivalent for conformance.
- Each item SHOULD include a "Refs:" line naming the relevant normative sections. Appendices MAY be included as informative references. Example: "Refs: Sections 4, 4.1, 4.5; Appendix B (Informative)".
- Item text MUST follow a simple template where applicable:
  - Success: expected successful behavior
  - Failure: expected exit code and classification (band). Subcode assignment and display labels follow Section 7.11 and `docs/errors.md`.
  - Notes: assumptions and scope boundaries.

##### C.2.3.1 Compact Item Format
Implementations MAY write each Appendix C test item using the following compact, machine- and human-readable format. The first line begins with a list dash.
- [EVT-<S><F><M>-<n>] <concise item title>
  <single-sentence detail> (optional)
  [Refs: Sections x.y, x.y; Appendix B] (optional)
  [See also: EVT-XXXX - Short title, EVT-YYYY] (optional)

Refs rules:
- Refs list normative sections from this specification. Appendices MAY be included as informative references. Do not include EVT identifiers.

See also rules:
- See also lists related test items from Appendix C by EVT identifier (optionally with a short title). Do not list specification sections. Include at most three entries.

#### C.2.4 Examples
- C.4.G (Module-Level / Grammar and AST), Unit, first item -> `EVT-MGU-1`.
- C.4.E (Module-Level / Context and Escaping), Fuzz, third item -> `EVT-MEF-3`.
- C.5.C (Broader-Scope / CLI and UX), Unit, second item -> `EVT-BCU-2`.

Compact format example:
- [EVT-MGU-1] Parser/Lexer
  [Refs: Sections 4.1, 4.5]
  [See also: EVT-MGF-1]

#### C.2.5 Property-Based and Fuzz Test Requirements
Property-based and fuzz tests MUST follow the implementation guidelines detailed in `docs/testing/fuzz.md`. At a minimum, suites MUST:
- ensure determinism under CI through fixed seeds and bounded iterations;
- replay persisted corpora using the `go test fuzz v1` format in regular tests;
- keep exploratory, long-running fuzzing out of CI and promote minimized reproducers to the persisted corpus;
- where applicable, apply `bash -n` validation and sandbox gating per Sections 8.2 and 8.5; see also Appendix F (Informative).

### C.3 Taxonomy and Notation (Informative)
This section defines the scope and intent of each test family. Code definitions (S/F/M) are centralized in C.2; they are not repeated here.

- G: Grammar and AST - Parsing and lexing rules, assignment forms and operators, AST element structure, and diagnostic source positions.
- P: Placeholders and Modifiers - Placeholder syntax, modifier order and semantics, valid/invalid combinations, and bypass behavior.
- E: Context and Escaping - Minimal required escaping per context, prohibitions, and quoting invariants across contexts.
- W: Whitespace and Newlines - Leading/trailing whitespace behavior, CR/LF/CRLF handling, and allow_tab/allow_newline interactions.
- R: Round-Trip and Idempotence - Stability across render -> parse -> render and re-canonicalization after bounded mutations.
- U: Unicode - Normalization forms, combining marks and zero-width characters, bidi controls, emoji sequences, and plane coverage.
- N: Nesting and Boundaries - $(...) nesting, closer handling, bare boundaries, and adjacency scenarios.
- C: CLI and UX - Commands and flags, stream and logging policies, diff format, and user-visible behavior.
- I: I/O and Path - Path derivation and resolution, overwrite policy, permissions, and size limits.
- S: Security and Redaction - Masking invariants, exposure negatives, and policy windows.
- D: Diagnostics and Error Mapping - Exit codes, unique subcodes, message format, and documentation links.
- Z: Resolver and Caching - Resolver behaviors, failure mapping, and cache semantics across scopes and processes.

Headings follow `C.<scope>.<family>`. Within each family, items are grouped by method in the fixed order: Unit, Property, Fuzz.


### C.4 Module-Level Tests
#### C.4.G Grammar and AST
##### Unit
- [EVT-MGU-1] Parser/Lexer (Sections 4, 4.1, 4.5): assignment forms (names, arrays, +=), missing =, unterminated quotes/$(...)/backticks; element ordering and trailing-newline flag recorded; CR/LF/CRLF preserved (no implicit trimming). See C.4.W for newline-flag stability across transforms.
- [EVT-MGU-2] Parse error diagnostics (Section 4.5): report line and column; corruption families map to expected messages/positions.
- [EVT-MGU-3] Additive and indexed assignments (Sections 4.1, 5.1): NAME+=value and NAME[INDEX]=value reparsable; adjacency and trailing comments preserved.
- [EVT-MGU-4] Trailing comment attachment (Sections 3, 4.1, 4.3): trailing comments belong to Assignment and survive round-trip intact.
- [EVT-MGU-5] Operators × adjacency (Sections 4.1, 5.1): =/+=/[INDEX]= cross-product with literal/placeholder adjacency. For trailing-newline flag behavior and stability, see C.4.W.
- [EVT-MGU-6] Render-time error source position (Sections 3, 5.4): failures anchored to the placeholder token’s line/column.
##### Property
- [EVT-MGP-1] Parse preservation (Sections 4, 5.1): element order remains stable across parse -> render -> parse. For whitespace and trailing-newline stability, see C.4.W.
- [EVT-MGP-2] Parser-AST mutation invariants (Sections 4, 5.1): targeted corruptions yield the intended error category and source position. For re-canonicalization and byte identity guarantees, see C.4.R.
##### Fuzz
- [EVT-MGF-1] Syntax fuzz (Sections 4, 4.1): unterminated quotes/$(...)/backticks, arrays, +=, missing =.
- [EVT-MGF-2] Corruption families (Sections 4.1, 4.5): missing closer, invalid name chars, unmatched ], swapped $( / ).
- [EVT-MGF-3] Grammar-focused exploration (Sections 4, 5): boundary/error bias; success -> re-parse succeeds; failure -> specific error + position.
- [EVT-MGF-4] Failure assertion detail codes (Sections 4.5, 7.11): fuzz failures MUST assert DetailCode and position; tests MUST NOT rely on message substrings for pass/fail.

#### C.4.P Placeholders and Modifiers
##### Unit
- [EVT-MPU-1] Placeholder syntax and whitespace (Section 4.3): PATH trimming; tolerances around `|`, `,`, and immediately before `>`; unknown/duplicate/empty modifiers; newline/NUL in body.
- [EVT-MPU-2] Sigil strictness (Section 4.3): no whitespace between pass and : (e.g., `<pass :` is a parse error) and diagnostics MUST include line and column.
- [EVT-MPU-3] Case sensitivity (Section 4.3): modifiers are case-sensitive; Allow_Tab is unknown.
- [EVT-MPU-4] Modifier semantics (Section 5.2): strip-family -> base64 -> context checks; invalid combinations (unique subcodes); dangerously_bypass_escape behavior.
- [EVT-MPU-5] Base64 fundamentals (Section 5.2): [A-Za-z0-9+/=], no wrapping; empty and varied lengths including non-ASCII sources.
- [EVT-MPU-6] Strip family specifics (Section 5.2): Space/TAB/CR/LF trimming; repeated application idempotence; boundary to empty.
- [EVT-MPU-7] Valid strip × allow_* (Section 5.2): normalize before context checks (strip first).
- Post-render re-parse validation: see Section 5.4 and C.2; failures occur under exit code 105 when bypass is not used. Display labels follow Section 7.11; subcodes per `docs/errors.md`.
##### Property
- [EVT-MPP-1] Modifier ordering and closure (Section 5.2): strip-family then base64 then context checks; idempotence under repetition.
- [EVT-MPP-2] Dangerous bypass path (Sections 5.2, 5.4, 6.6): bypass skips re-parse validation; no secret exposure on streams/logs; downstream parse errors handled.
- [EVT-MPP-3] Base64 in bare context (Sections 5.2, 5.3.3): `+`, `/`, `=` MUST NOT be escaped; rendered output MUST remain reparsable.
##### Fuzz
- [EVT-MPF-1] Placeholder body fuzz (Section 4.3): PATH/modifier spacing, unknown/duplicate/empty modifiers, newline/NUL in body.
- [EVT-MPF-2] Near-sigil fuzz (Section 4.3): generate `<pass` followed by one of { `:`, SPACE, TAB, CR, LF, ALNUM, PUNCT } to assert that any whitespace prior to `:` yields a parse error with source position.
- [EVT-MPF-3] Invalid modifier combinations (Section 5.2): render-time errors with unique subcodes.
- [EVT-MPF-4] Base64 variety (Section 5.2): alphabet coverage; no line wrapping.
- [EVT-MPF-5] Non-ASCII whitespace around PATH/modifiers (Sections 4.3, 4.5): trimming and list whitespace MUST reject non-ASCII whitespace with the appropriate DetailCode (`EVE-103-204`, `EVE-103-305`).

#### C.4.E Context and Escaping
##### Unit
- [EVT-MEU-1] Per-context escaping minimality (Section 5.3.3): double-quoted (`\"`, `\\`, `$`, and `` ` `` only); `$(...)` (`\\`, `$`, and placeholder-originated `)` only); backtick (`` ` ``, `\\`, and `$` only).
- [EVT-MEU-2] Bare literalization (Sections 4.1, 5.3.3, 5.3.4): backslash preserves literal meaning for SPACE, #, $, quotes, backslash, brackets; top-level # odd/even rule.
- [EVT-MEU-4] Renderer/Bare: minimal escape set compliance (Sections 5.3.3–5.3.4)
  Verify that the renderer, in Bare context, escapes exactly the characters mandated by Section 5.3.3 (Always‑escape matrix) and does not introduce escapes for other characters (except when modifiers or context rules require). Construct synthetic secrets covering:
  - Always‑escape set: SPACE, #, $, ", ', `, \\, (, ), {, }, [, ], |, &, ;, <, >
  - Non‑escape exemplars: ASCII letters/digits/underscore/dot/dash; selected punctuation not listed in the matrix
  Assertions:
  - Characters in the Always‑escape set MUST be escaped.
  - Characters outside the set MUST remain unescaped (unless the chosen modifier/context requires otherwise).
  [Refs: Sections 5.3.3, 5.3.4; Appendix A.2; Appendix D.4]
- [EVT-MEU-5] Bare leading tilde (Sections 5.3.3–5.3.4): when the first emitted code point of the RHS is `~`, the renderer MUST escape it as `\~` to prevent tilde expansion by shells that `source` `.env`.
- [EVT-MEU-6] Bare non-leading tilde (Sections 5.3.3–5.3.4): a `~` that is not the first emitted code point of the RHS MUST remain unescaped.
- [EVT-MEU-7] Bare leading TAB then tilde with `allow_tab` (Sections 5.2, 5.3.2, 5.3.4): with `allow_tab` present, a leading TAB is emitted as-is; a subsequent `~` is not the first emitted code point and MUST therefore remain unescaped.
- [EVT-MEU-8] Bare start-of-word tracking across tokens (Sections 5.3.3–5.3.4): if leading tokens render an empty string (e.g., empty literal/placeholder after strip), and the next token’s first code point is `~`, the renderer MUST treat it as the first emitted code point and escape it as `\\~`.
##### Property
- [EVT-MEP-1] Escaping closure (Section 5.3.3): neither over- nor under-escaping across contexts.
- [EVT-MEP-2] Comment detection stability (Section 4.1): top-level # odd/even backslashes; quoted/$(...)/backtick interiors unaffected.
- [EVT-MEP-3] Re-parse on rendered output (Sections 5.1, 5.4): when `dangerously_bypass_escape` is absent, the rendered output MUST pass parser validation.
- [EVT-MEP-4] Bare conformance property (Sections 5.3.3–5.3.4): for random secrets over {ASCII graph, spaces, tabs, `|&;<>`, quotes, parens, brackets, braces, backticks, `$`, backslashes, `~`} with/without `allow_tab`, escaping MUST equal matrix ∪ conditional; `parse -> render -> parse` MUST be stable.

##### Fuzz
- [EVT-MEF-1] Prohibited controls stress (Section 5.3.1): inject controls and NUL per the acceptance rules in C.4.U; focus validation on context-local escaping minimality and comment detection invariants.
- [EVT-MEF-2] Bare extended set fuzz (Sections 5.3.3–5.3.4): generate combinations over `{SPACE, TAB (with/without allow_tab), #, $, ", ', `, \\, (, ), {, }, [, ], |, &, ;, <, >, ~}` with adjacency to literals/placeholders; assert matrix ∪ conditional escapes only; stable re-parse and (when applicable) `bash -n` success.

#### C.4.W Whitespace and Newlines
##### Unit
- [EVT-MWU-1] Trailing whitespace preservation (Sections 4.4, 7.2): end-of-line spaces/TABs before comments/newline preserved across round-trip.
- [EVT-MWU-2] EOF newline normalization (Sections 5.1, 5.2): For resolver-sourced values, implementations MUST remove exactly one trailing logical newline at EOF when present. "Logical newline" here means LF or CRLF. Internal newlines are unaffected. Cases to cover:
  - "abc\n" -> "abc"
  - "abc\r\n" -> "abc"
  - "abc\n\n" -> "abc\n"
  - "abc\r\n\r\n" -> "abc\r\n"
  - "abc" (no trailing newline) -> "abc"
  - "a\nb\n" -> "a\nb"
  - Context independence: results above MUST hold regardless of placeholder context; acceptance of internal newlines remains governed by Section 5.3 and the presence of allow_newline.
##### Property
- [EVT-MWP-1] CRLF as a single logical newline (Sections 1.1, 4): parity with LF in representable contexts; rejection elsewhere.
- [EVT-MWP-2] Leading whitespace preservation (Sections 3, 4.1, 4.4, 7.2): leading spaces/TABs for assignments and whole-line comments preserved.
- [EVT-MWU-3] Target .env grammar-level whitespace (Sections 1.2, 7.6): non-ASCII whitespace MUST be rejected only where grammar-level whitespace is expected (leading indentation for blank/comment/assignment). Unicode whitespace within value string segments MUST be accepted and MUST NOT raise `EVE-107-101`.
  [Refs: Sections 1.2, 7.6; Appendix D.1]
- [EVT-MWP-3] Strip-family behavior reference (Section 5.2): trimming semantics and idempotence are covered under C.4.P. This family verifies only interaction with newline/TAB allowances and representation constraints.
- [EVT-MWP-4] Newline with allow_newline (Sections 5.3.2, 5.3.4-5.3.8): acceptance/denial by context; positions and mixed runs; trailing-newline flag stability.
- [EVT-MWP-5] TAB with allow_tab (Section 5.3.2): acceptance/denial across contexts; adjacency with spaces; placeholder vs literal origin.
- [EVT-MWP-6] Minimal matrix coverage: Suites MUST, at minimum, exercise the cross-product below to verify newline handling with CR/LF/CRLF and `allow_newline` across contexts.
  - Accepting contexts: double-quoted, $(...). For each newline kind {LF, CR, CRLF}, for `allow_newline` ∈ {present, absent}, positions {beginning, middle, end}, and runs {single, consecutive (≥2), mixed sequences (e.g., LF then CRLF)}, verify:
    - With `allow_newline` present: acceptance with correct rendering and re-parse stability (byte-identity where applicable), without over/under-escaping (per Section 5.3.3).
    - With `allow_newline` absent: render-time error with a unique subcode and guidance per docs/errors.md.
  - Prohibited contexts: bare, single-quoted, backtick. For newline kinds {LF, CR, CRLF}, regardless of `allow_newline`, for positions {beginning, middle, end} and runs {single, consecutive}, verify render-time errors with unique subcodes and appropriate guidance per docs/errors.md.
- [EVT-MWP-7] Normalization × strip-family/order (Sections 5.1, 5.2): Default EOF newline normalization MUST occur before strip-family modifiers, then base64, then context checks. Suites MUST include examples demonstrating order-dependent outcomes:
  - Value "X\n " (LF then SPACE): after normalization -> "X "; with strip_right present -> "X".
  - Value "X\r\n\t" (CRLF then TAB): after normalization -> "X\t"; with strip_right present -> "X".
  - Value "X \n" (SPACE then LF): after normalization -> "X "; with strip_right present -> "X".
  - Value "A\nB\n" (internal newline): normalization removes only the trailing newline -> "A\nB"; subsequent strip_right MUST NOT remove the internal newline.
  - Idempotence: applying normalization exactly once per resolution pass MUST be sufficient; subsequent steps MUST preserve determinism across render -> parse -> render.
- [EVT-MWP-8] Normalization × contexts (Sections 5.1, 5.3): Normalization is context-independent, but context acceptance of interior newlines is not. Suites MUST verify:
  - Double-quoted without allow_newline:
    - Value "T\n" -> after normalization "T" (allowed; no allow_newline needed).
    - Value "T\nU" (internal newline) -> normalization does not remove the interior newline; render MUST fail without allow_newline.
  - Bare/single-quoted/backtick: normalization may remove a trailing newline, but any remaining internal newline MUST still be prohibited per Section 5.3, independent of normalization.
  - Command substitution: same acceptance rules as double-quoted; trailing-only newline removal MUST not require allow_newline, interior newlines still require allow_newline.
- Trailing-newline flag stability: Suites MUST verify that the template line’s trailing-newline flag remains stable across render -> parse, and that placeholder value trailing newlines (LF/CR/CRLF) affect the rendered artifact and re-parse result consistently per Sections 4 and 7.2.
  - Parity note: A standalone CR and an LF are treated as distinct newline code points; a CRLF sequence MUST be treated as a single logical newline for allowance decisions (Section 1.2).
##### Fuzz
(none specific beyond E/P fuzz; see also C.4.P and C.4.E)

#### C.4.R Round-Trip and Idempotence
##### Property
- [EVT-MRP-1] Render -> Parse -> Render idempotence (Sections 4, 5.1): first and second renders are byte-identical.
- [EVT-MRP-2] Parser-AST mutation closure (Sections 4, 5.1): re-canonicalization under bounded mutations.
- [EVT-MRP-3] Round-trip (lightweight) (Sections 4, 5.1): render then re-parse succeeds; for byte-identity guarantees, see Render -> Parse -> Render idempotence.
- [EVT-MRP-4] Literal token verbatim preservation (Sections 4.4, 5.1, 7.2): literal segments, including backslash sequences, remain byte-identical after render -> parse -> render.
##### Fuzz
- [EVT-MRF-1] Composite templates (Sections 4, 5): mixed contexts, boundaries/adjacency, comments/whitespace/trailing-newline flags across whole file.

#### C.4.U Unicode
##### Unit
- [EVT-MUU-1] Acceptance filter (Sections 5.3.1, 5.3.2): reject NUL and prohibited controls; others allowed subject to context rules.
##### Property
- [EVT-MUP-1] Comprehensive matrix (Section 5.3): NFC/NFD/NFKC/NFKD, combining chains, ZWJ/ZWNJ, bidi controls, emoji ZWJ/skin tones, BMP/supplementary planes; reject prohibited controls and NUL.
- [EVT-MUP-2] Minimal representative set: Suites MUST, at minimum, exercise the following representative code points and sequences across contexts (bare/double/single/command_subst/backtick), verifying acceptance rules, context-local escaping, re-parse stability, and that CLI artifacts contain no secret values where applicable (Sections 6.1 and 6.3).
  - Normalization pairs:
    - U+00E9 (é) vs U+0065 U+0301
    - U+00C5 (Å) vs U+0041 U+030A
    - U+AC00 (가) vs U+1100 U+1161
  - Combining sequences:
    - U+0061 U+0301 U+0327 (a + combining acute + combining cedilla)
  - Zero-width and join controls:
    - U+200D (ZWJ), U+200C (ZWNJ), U+200B (ZERO WIDTH SPACE), U+2060 (WORD JOINER)
  - Bidirectional controls:
    - U+200E (LRM), U+200F (RLM)
    - U+202A (LRE), U+202B (RLE), U+202D (LRO), U+202E (RLO), U+202C (PDF)
    - U+2066 (LRI), U+2067 (RLI), U+2068 (FSI), U+2069 (PDI)
  - Emoji and variants:
    - U+1F600 (GRINNING FACE)
    - U+1F469 U+200D U+1F4BB (woman technologist, ZWJ)
    - U+1F44B U+1F3FB (waving hand + skin tone)
    - U+2764 U+FE0F (heart + VS16)
  - Planes and PUA:
    - U+4E2D (中), U+1F9D1 (🧑), U+20000 (𠀀), U+E000 (PUA)
- The following negative controls MUST NOT be included:
    - U+0000 (NUL)
    - C0/C1 controls except TAB U+0009 and newline LF/CR (e.g., U+0001, U+000B, U+001F, U+007F)
- [EVT-MUP-3] Expanded coverage beyond minimal: Beyond the minimal representative set, suites MUST explore a broader matrix by sampling variations across normalization forms {NFC/NFD/NFKC/NFKD}, sequence lengths {1, 2-3, 4-7, 8-15, 16+}, combining mark chains {≥3, ≥8}, zero-width counts {1, 2, ≥3} and placements (ASCII adjacency, Unicode adjacency), bidi isolates/overrides with nesting depth {≥2}, multiple emoji ZWJ clusters {≥2} including skin tones and VS16, and planes {BMP/SMP/SIP/PUA}. Coverage MUST span all contexts and, where applicable, both presence/absence of `allow_tab`/`allow_newline`. Failures MUST be reproducible via fixed seeds and recorded cases.
##### Fuzz
- [EVT-MUF-1] Large sequences (Sections 5.3, 7.3): combining/zero-width/bidi/emoji across contexts; only context-local escaping; stable re-parse.

#### C.4.N Nesting and Boundaries
##### Unit
- [EVT-MNU-1] $(...) closer boundary (Section 5.3.7): placeholder-originated ) escaped as \); syntactic closers unescaped; re-parse succeeds.
- [EVT-MNU-2] Bare adjacency with surrounding literals (Sections 4.1, 5.1, 5.3.3-5.3.4): when a bare placeholder is immediately adjacent to literal text on either side (prefix and/or suffix), and the resolved value contains any character outside the bare-safe set `[A-Za-z0-9_.-]` (subject to TAB/newline/control prohibitions and base64 allowances), the renderer MUST preserve lexical boundaries by emitting a backslash before each such character per Section 5.3.3 (Bare). Rendering MUST succeed and the result MUST re-parse identically. Example: Input `EXAMPLE_VAR=prefix<pass:a>suffix`, `<pass:a>` -> ` value` (leading space), expectation: `EXAMPLE_VAR=prefix\ valuesuffix`.
- [EVT-MNU-3] Bare adjacency (positive complement) (Sections 4.1, 5.1, 5.3.3-5.3.4): when the resolved value consists solely of the bare-safe set (or permitted base64 characters when `|base64>` is present: `[A-Za-z0-9+/=]`), adjacency to surrounding literals MUST render successfully and re-parse identically. Example (positive): Input `EXAMPLE_VAR=prefix<pass:a>suffix`, `<pass:a>` -> `safe123`, expectation: `EXAMPLE_VAR=prefixsafe123suffix`.
- [EVT-MNU-4] Bare adjacency with extended escapes (Sections 5.3.3-5.3.4): when adjacency exists and the resolved value contains any of `|`, `&`, `;`, `<`, `>`, renderer MUST backslash-escape each such character; result MUST re-parse identically; with `allow_tab`, TAB MUST be emitted as-is.
  [Refs: Sections 5.3.3, 5.3.4]
##### Property
- [EVT-MNP-1] $(...) nesting (Section 5.3.7): depths 1-3; placeholder-originated ) escaped; syntactic closers unescaped.
- [EVT-MNP-2] Bare boundaries (Sections 4.1, 5.3.3-5.3.4): line start/end, pre-comment, adjacency to placeholders.
- [EVT-MNP-3] Consecutive placeholders and adjacency (Sections 4.3, 5.1): <pass:a><pass:b> reparsable with unchanged quoting; see C.4.R for byte-identity guarantees.
  - See C.5.C for sandboxed execution semantics involving command substitution and multi-level evaluation.
##### Fuzz
- [EVT-MNF-1] Deep $(...) nesting with multiple closing parentheses (Section 5.3.7): depths 1-3; only placeholder-originated ) escaped; re-parse succeeds.

#### C.4.S Security and Redaction
##### Unit
- [EVT-MSU-1] Redaction core (Sections 6.1, 6.3): no secret exposure to stdout/stderr/logs. Mask shape/length/determinism are implementation-defined.
- [EVT-MSU-2] Post-diff reconstruction (Section 6.3): verify that masked outputs never include raw secrets across contexts and value varieties (ASCII, non-ASCII, whitespace, CR/LF/CRLF, negative controls). See C.5.C for diff header and hunk structure requirements.
##### Property
- [EVT-MSP-1] See C.5.S for end-to-end exposure-negative checks across CLI paths. This family focuses on module-level invariants.

#### C.4.Z Resolver and Caching
##### Unit
- [EVT-MZU-1] Resolver caching and single-resolution policy (Sections 6.2, 7.6): resolve each PATH at most once per render pass; handle NUL-containing values and use-after-close.
- [EVT-MZU-2] Resolver failures: expectations follow Appendix B / `docs/errors.md` (exit code 104; display labels per Section 7.11).
- [EVT-MZU-3] Multi-appearance reuse: repeated PATHs across lines/adjacent placeholders MUST hit the in-process cache (no duplicate resolutions).
- [EVT-MZU-4] Call-count constraint: For a template containing the same PATH multiple times (across lines and adjacency), the underlying resolver/Pass client MUST be invoked at most once per unique PATH per render pass. Suites MUST use an instrumented PassClient that counts Show calls to assert this.
- [EVT-MZU-5] Adjacency and cross-line cases: Suites MUST include both `<pass:dup><pass:dup>` adjacency and duplicates across separate lines to verify cache hits in distinct parse positions.
  - [EVT-MZU-6] Resolver raw vs renderer normalization (Sections 5.1, 6.2): The resolver MUST return the raw value unchanged, and the renderer MUST perform the default EOF newline normalization. Suites MUST:
    - Use an instrumented PassClient double to return "abc\n" (and "abc\r\n") and assert that the renderer outputs "abc".
    - Assert single-resolution policy (Show called at most once per PATH per render pass) and separation of concerns: the resolver output includes trailing newline(s); the renderer output reflects exactly-one trailing newline removal.

#### C.4.I I/O and Path
##### Unit
- [EVT-MIU-1] Stat errno classifier mapping (Unix/mac)
  Map os.Lstat errors to EVE-102-B0: ENOENT -> EVE-102-1; ENOTDIR -> EVE-102-3; ELOOP -> EVE-102-4; ENAMETOOLONG -> EVE-102-5; others -> EVE-102-203 fallback. Tested under Unix/mac; non-Unix builds provide fallback behavior.
  [Refs: Sections 7.10.1, 7.3; docs/errors.md#eve-102-1, #eve-102-3, #eve-102-4, #eve-102-5, #eve-102-203]
    - See also: EVT-MZU-1 (caching policy), EVT-MWP-7 (ordering with modifiers).

### C.5 Broader‑Scope Tests
#### C.5.E Context and Escaping
##### Unit
- [EVT-BEU-1] Bare/minimal escape set via sandboxed bash (source)
  Using a sandboxed bash (when available), validate that `.env` produced by the renderer for Bare context survives `set -a; . <env>` and preserves values. Build a matrix of representative single‑character secrets and short strings:
  - Always‑escape candidates: SPACE, #, $, ", ', `, \\, (, ), {, }, [, ]
  - Additional shell meta probes (for investigation): |, &, ;, <, >
  Procedure:
  1) Render assignments `VAR=<pass:...>` for each candidate (resolver double returns the candidate).
  2) In sandboxed bash: `set -a; . <env>; declare -p VAR` and decode the value.
  3) Assert: sourcing succeeds and decoded values equal the renderer’s intended values.
  Environment: print `bash --version` (first line) and `shopt -p` for diagnostics; skip with a clear reason when sandbox/bash are unavailable.
  Skip with an informational log when sandbox/bash are unavailable.
  [Refs: Sections 5.3.3–5.3.4, 8.2, 8.5; Appendix F]

- [EVT-BEU-2] Bare/negative probes for Always‑escape set (source)
  Craft `.env` lines that intentionally remove escapes for a small representative subset of Always‑escape characters (e.g., SPACE, #, $, ", ', `, \\, {, }) and try `set -a; . <env>`.
  Expectation:
  - Sourcing fails (syntax error), or succeeds with an incorrect value (demonstrating why the character MUST be escaped).
  Notes:
  - Run only under sandbox and stable bash settings; collect `bash --version` and `shopt -p` for diagnostics.
  [Refs: Sections 5.3.3–5.3.4, 8.5; Appendix F]

- [EVT-BEU-3] Bare/allow_tab behavior under source
  For TAB in secrets, verify:
  - Without `allow_tab`: rendering fails with the appropriate render‑time error (EVE‑105‑502).
  - With `allow_tab`: renderer emits TAB as‑is; `set -a; . <env>` succeeds and `VAR` contains a literal TAB.
  [Refs: Sections 5.2, 5.3.4; Appendix A.2]

##### Property
- [EVT-BEP-1] Cross‑check: renderer vs measured minimal escape set (source)
  Compute an empirical minimal escape set by probing single characters and short pairs under `set -a; . <env>` and comparing pre/post values. Include adjacency positions {line start, middle, end} with ASCII alnum prefix/suffix where applicable. Assert that the renderer’s Always‑escape behavior matches the measured set:
  - If the renderer under‑escapes (measured set minus renderer set ≠ ∅): FAIL (dangerous).
  - If the renderer over‑escapes (renderer set minus measured set ≠ ∅): report as over‑escaping; treat as FAIL unless the specification explicitly mandates the larger set.
  Skip in environments without sandbox/bash; log the reason.
  [Refs: Sections 5.3.3–5.3.4, 8.2, 8.5; Appendix F]

#### C.5.C CLI and UX
##### Unit
- [EVT-BCU-1] CLI commands (Sections 7.2-7.4, 7.9): argument/flag validation for sync/diff/validate; unknown/unsupported combinations; optional `[INPUT_FILE]` with default `./.envseed` when omitted.
- [EVT-BCU-2] Streams and logging (Sections 6.4, 7.1): no stdout in non-dry-run; quiet suppresses info; diagnostics to stderr only; redaction in dry-run/diff.
- [EVT-BCU-3] Sync message format (Sections 7.7, 7.1): on change wrote <path> (mode 0600); on no change wrote <path> (unchanged); --quiet suppresses info but never errors.
- [EVT-BCU-4] Stdin not supported (Section 7.3): reading from stdin rejected with diagnostics; expected exit code 101. Display labels follow Section 7.11.
- [EVT-BCU-5] validate specifics (Section 7.9): parse-only; input name need not contain envseed; for option validation semantics, see CLI commands above.
- [EVT-BCU-6] diff behaviors (Section 7.8): missing target -> diff against empty (all additions, exit 1); identical files -> silent stdout/stderr and exit 0.
- [EVT-BCU-7] sync dry-run output (Sections 6.3, 7.7): always include the absolute resolved output path (per Section 7.5) in the report, regardless of `--output`.
- [EVT-BCU-8] Unified diff headers (Section 7.8): first two lines are `--- <path>` and `+++ <path>` where both `<path>` values are byte-identical absolute resolved output paths (per Section 7.5); no prefixes or annotations. Body uses `@@` hunks per Section 7.8. See also C.5.S for redaction requirements.
- [EVT-BCU-9] Render-time error display (Sections 7.11, 7.10): CLI diagnostics MUST include source line and MUST include column when tracked; formatting is stable and secrets are never revealed.
- [EVT-BCU-10] Default input (Sections 7.3, 7.7–7.9): when `[INPUT_FILE]` is omitted and `./.envseed` exists, `sync`/`diff`/`validate` succeed using the default file.
##### Property
- [EVT-BCP-1] Bash validation and sandbox gating (Sections 8.2, 8.5): When conditions in Section 8.2 are satisfied, suites MUST perform `bash -n` validation; otherwise suites MUST skip with an explicit reason (e.g., backticks present, missing bwrap, unsupported namespaces).
- [EVT-BCP-2] Sandboxed execution: When a non-network, process-isolated sandbox is available, suites MUST execute rendered artifacts and capture observable state (e.g., selected environment variables) to validate end-to-end semantics. Execution MUST be gated by environment checks and MUST be skipped with an explicit reason when prerequisites are absent. Suites MUST ensure no secret exposure on stdout/stderr during execution.
- [EVT-BCP-3] Execution determinism: For a given template/resolver pair, captured values MUST be deterministic across runs. Suites MUST fix seeds and bound iterations to keep execution stable. Backticks MUST NOT be executed unless explicitly permitted by the sandbox policy.
- [EVT-BCP-4] Multi-level evaluation semantics: For templates that mix placeholder substitution with command substitution `$(...)` at depths 1-3, the final runtime values MUST match the semantic composition implied by the template, including adjacency, quoting boundaries, and command-substitution escaping. Only placeholder-originated `)` MUST be escaped; syntactic closers MUST remain unescaped.
- [EVT-BCP-5] Command-substitution boundary results: Within `$(...)`, suites MUST assert that placeholder-originated `)` are escaped during rendering and that the runtime output equals the expected unescaped structure when executed in the sandbox.
##### Fuzz
- [EVT-BCF-1] End-to-end CLI flows (Sections 6, 7.2-7.10): inject resolver/I/O/size/flag failures; verify exit codes, unique subcodes, and guidance text per docs/errors.md.
- [EVT-BCF-2] CLI diagnostics under failure injection (Sections 7.2-7.10): unknown/invalid flags, resolver errors, I/O failures, size overruns; see also CLI commands above for validation semantics.
- [EVT-BCF-3] Corpus management (Sections 8.3, 8.4): minimize failing inputs, store in package-relative `internal/<package>/testdata/fuzz/<FuzzName>/` (go test fuzz v1); use fixed seeds/iterations; optional bash -n validation when feasible.

#### C.5.I I/O and Path
##### Unit
- [EVT-BIU-1] I/O and permissions (Sections 6.5, 7.7): atomic writes via rename; mode 0600; skip unchanged writes; refusing to overwrite without --force returns 106.
- [EVT-BIU-2] Diff limits (Sections 7.8, 7.10): reject comparisons > 10 MiB; expected exit code 108.
- [EVT-BIU-3] Path resolution logic (Section 7.5): --output handling; trailing separators; existing dir/file; missing parent; expected exit codes `101` / `106`.
- [EVT-BIU-4] Name requirement (Sections 7.3, 7.5): for sync/diff without `--output`, the selected input name must contain `envseed`.
- [EVT-BIU-5] First-occurrence rule (Section 7.5): without `--output`, replace only the first `envseed` in the selected input path.
- [EVT-BIU-6] Default missing (Sections 7.3, 7.10): when `[INPUT_FILE]` is omitted and `./.envseed` is missing, return exit 102 with DetailCode EVE-102-1 (ENOENT). If the default path fails for other stat-phase reasons, classify per EVE-102-B0 (EISDIR/EVE-102-2; ENOTDIR/EVE-102-3; ELOOP/EVE-102-4; ENAMETOOLONG/EVE-102-5).
- [EVT-BIU-7] ENOTDIR classification (EVE-102-3)
  When a mid-path component is not a directory (e.g., <file>/subpath), sync/diff/validate return exit 102 with DetailCode EVE-102-3 (“selected input %q path component is not a directory”). No secret exposure in diagnostics.
  [Refs: Sections 7.3, 7.7–7.9, 7.10.1, 7.11; docs/errors.md#eve-102-3]
- [EVT-BIU-8] ELOOP classification (EVE-102-4)
  When a symbolic link loop prevents resolution, sync/diff/validate return exit 102 with DetailCode EVE-102-4 (“selected input %q symlink loop detected”). No secret exposure in diagnostics.
  [Refs: Sections 7.3, 7.7–7.9, 7.10.1, 7.11; docs/errors.md#eve-102-4]
- [EVT-BIU-9] ENAMETOOLONG classification (EVE-102-5)
  When a path (or a single component) exceeds the length limit, sync/diff/validate return exit 102 with DetailCode EVE-102-5 (“selected input %q name too long”). No secret exposure in diagnostics.
  [Refs: Sections 7.3, 7.7–7.9, 7.10.1, 7.11; docs/errors.md#eve-102-5]
- [EVT-BIU-10] Open vs Read failures (Section 7.3): injecting failures at open MUST return `EVE-102-201`; injecting failures at read MUST return `EVE-102-202`; permission MUST return `EVE-102-101`. Applies to `sync`, `diff`, and `validate`.
  [Refs: Sections 7.3, 7.10.1; docs/errors.md#eve-102-201, #eve-102-202, #eve-102-101]
##### Property
- [EVT-BIP-1] Path resolution branches (Section 7.5): cross-product over --output, directory/file/non-existent, trailing separator; expected exit codes `101` / `106`.
- [EVT-BIP-2] Cross-command invariance for EVE-102-B0
  For each stat-phase condition (ENOENT/EISDIR/ENOTDIR/ELOOP/ENAMETOOLONG), sync/diff/validate MUST produce exit 102 with the same DetailCode and stable diagnostic formatting. Test with both relative and absolute paths.
  [Refs: Sections 7.3, 7.7–7.10, 7.10.1, 7.11; docs/errors.md#eve-102-1, #eve-102-2, #eve-102-3, #eve-102-4, #eve-102-5]
- [EVT-BIP-3] Open/read separation (Section 7.3): across commands, induced open vs read failures MUST map consistently to `EVE-102-201` vs `EVE-102-202`; diagnostics MUST be stable per Section 7.11.
  [Refs: Sections 7.3, 7.10.1, 7.11; docs/errors.md]
##### Fuzz
- [EVT-BIF-1] Diff size limit boundary (Section 7.8): at/over 10 MiB threshold; exit 108 stability.
- [EVT-BIF-2] I/O failure conditions: inject filesystem/permission and size-limit failures; verify exit `106/108`, CLI message, and guidance text. Subcodes follow `docs/errors.md` (no specific ranges listed here).
- [EVT-BIF-3] Stat-phase errno discovery fuzz
  Generate path constructs that trigger ENOTDIR/ELOOP/ENAMETOOLONG at the stat phase (without destructive operations). Assert exit 102 with EVE-102-3/4/5 respectively. Skip when filesystem semantics are unsupported (document reason).
  [Refs: Sections 7.8, 7.10.1; docs/errors.md#eve-102-3, #eve-102-4, #eve-102-5; docs/testing/fuzz.md]

#### C.5.S Security and Redaction
##### Property
- [EVT-BSP-1] Secret exposure negatives (Sections 6.1, 6.3, 7.1): scan stdout/stderr/logs in dry-run/diff and non-dry-run paths. See also C.4.S.

#### C.5.D Diagnostics and Error Mapping
##### Unit
- [EVT-BDU-1] Parse-error diagnostics (Section 4.5): exit code 103 for template parsing failures; display format follows Section 7.11.
- [EVT-BDU-2] EVE-102-B0 diagnostics mapping
  For ENOTDIR/ELOOP/ENAMETOOLONG, CLI diagnostics MUST include the exact message text defined in docs/errors.md and the documentation link slug. Formatting MUST match Section 7.11 and MUST NOT reveal secrets.
  [Refs: Sections 7.10.1, 7.11; docs/errors.md#eve-102-3, #eve-102-4, #eve-102-5]
- [EVT-BDU-3] Target .env unexpected line mapping (Sections 7.6, 7.11): parser `EVE-103-103` MUST map to target parsing `EVE-107-1` with CLI diagnostics per Section 7.11 (no secret exposure; correct reference slug).
  [Refs: Sections 7.6, 7.11; docs/errors.md#eve-107-1]
##### Fuzz
- [EVT-BDF-1] Internal exception mapping (Section 7.10): induce unexpected exception; exit code 199; diagnostics formatting stability.
- [EVT-BDF-2] Documentation link presence (Section 7.10): presence/format and consistency with docs/errors.md.
- [EVT-BDF-3] DetailCode assertions in fuzz/property (Sections 4.5, 7.11): failure-inducing inputs MUST assert DetailCode equality and report line/column; message substrings MUST NOT be used for pass/fail.

#### C.5.Z Resolver and Caching
##### Unit
- [EVT-BZU-1] Process-bound resolver cache (Section 6.2): across separate executions, identical PATHs are re-resolved (no cross-process persistence).
##### Property
- [EVT-BZP-1] Cross-process persistence negative: Across two separate CLI executions with identical inputs, suites MUST observe independent resolver invocations (no cache reuse). Implement via an instrumented Pass wrapper (e.g., counting/logging Show calls to a tempfile) and assert that the call count increases across runs.
