## 1. Foundations and Conventions

### 1.1 Scope and Normativity
- Audience: implementers, maintainers, and test authors.
- Normativity: Normative by default. Non-normative content is explicitly labeled `Informative`.
- Labeling policy: Chapter-level (##) headings add "(Informative)" only for non-normative sections. Normative sections are unlabeled by default. Subsections (###) MAY clarify normativity inline when needed.
- Sources of truth:
  - This specification is authoritative for behavior, algorithms, and failure conditions.
  - Error subcodes (numbering, messages, guidance) are canonical in `docs/errors.md` (generated from `internal/envseed/errors.go`). This specification does not enumerate specific numbers here; see Section 7.10 for exit categories and assignment principles, and Section 7.11 for diagnostic display format.

### 1.2 Vocabulary Style
- Identifiers (e.g., contexts `bare`, `double_quoted`, `single_quoted`, `command_subst`, `backtick`; modifiers `allow_newline`, etc.) MUST be written in lowercase and displayed in monospace format.
- Headings MAY contain human-readable names followed by the corresponding identifier in parentheses, e.g., "Backtick (`backtick`)".
- Smart quotation marks (“ ” ‘ ’) MUST NOT be used for defined terms. Authors MUST use monospace format by surrounding the term with backticks (\`) instead.
- Unicode arrows (→, ⇒, etc.) MUST NOT be used. ASCII arrows (A -> B) SHOULD be used in their place.
- The term "whitespace" in this specification refers to "Space (U+0020) and Tab (U+0009)" and may be abbreviated as "Space/Tab".
- When referencing failures, the exact exit code MUST be included in the form "exit code NNN". Do not enclose the numeric code in backticks in prose. Within code blocks and command-line interface examples, authors MAY use the format "exit NNN".
- Prose MUST reference sections and appendices using the forms "See Section X.Y" and "See Appendix D.5". Abbreviations such as "Sec." or "§" MUST NOT be used.
- Authors SHOULD employ plain English and MUST avoid idiomatic expressions, metaphors, or colloquial language. Latin abbreviations (for example, "e.g.," and "i.e.,") MAY be used with a trailing comma.
- Unicode code points MUST be expressed in the form "U+XXXX". Binary size units MUST use "KiB" and "MiB".
- When a parenthetical reference occurs at the end of a sentence, the period MUST appear after the closing parenthesis.
- The standard prefixes for notes are "Note:" and "Notes:". The trailing colon MUST always be included.

### 1.3 Terminology and Definitions
This document uses "EnvSeed" for the system name and "envseed" for the CLI command and file names.

Core entities:
- AST: Abstract Syntax Tree; the hierarchical representation of the input.
- Element: one of `Assignment`, `Comment`, or `Blank` in the AST (see Section 3, Data Model).
- Assignment: `NAME=value` (or `NAME+=value`, `NAME[INDEX]=value`), with preserved leading whitespace, name, operator, an ordered list of value tokens, optional trailing comment, and a trailing-newline flag.
- ValueToken: either `Literal` or `Placeholder`, including its context and source position.
- Resolver: component that retrieves secrets. The default implementation uses `pass`.
- Redaction: masking of secrets so that real secret values never appear on stdout, stderr, or logs. See Section 6.3 for policy and algorithm.

Syntax and tokens:
- Placeholder: an embedded token in the form `<pass:PATH|modifier[, modifier...]>`.
- Context: one of `bare`, `double_quoted`, `single_quoted`, `command_subst`, `backtick`.
- Modifier: one of `allow_newline`, `allow_tab`, `base64`, `dangerously_bypass_escape`, `strip`, `strip_left`, `strip_right`.
- String segment: The content of a value on the right-hand side of an assignment excluding syntactic delimiters and operators. It excludes variable names, assignment operators, syntactic delimiters, and the top-level trailing comment introducer (`#`). For redaction semantics regarding string segments and escape pairs, see Section 6.3.
- Syntactic delimiter: Characters that form quoting or substitution boundaries in a value: `"`, `'`, `` ` ``, `$`, `(`, `)`. Backslashes are not delimiters. Redaction-specific handling of delimiters and escape backslashes is defined in Section 6.3.
- Required backslashes: Backslashes necessary to preserve literal meaning in a given quoting/substitution context. For parsing they are not part of the parse-time string segment. For redaction behavior pertaining to escape pairs and newline treatment, see Section 6.3.

Character classes: 
- See Appendix D.1 for normative definitions of whitespace and newline tokens. Control character handling at render time is governed by Section 5.3.

CLI path terms:
- Selected input path: The path string chosen by the CLI as the input. It is either the explicit `INPUT_FILE` argument when provided or the default `./.envseed` when `INPUT_FILE` is omitted. Selection does not imply existence or readability; validation is performed separately.
- Selected input name: The last path component (file name) of the selected input path. Name-based rules (for example, the `envseed` -> `env` derivation) refer to this value.
- Input file: The file at the selected input path. For `sync`, `diff`, and `validate`, the input file MUST exist as a readable regular file (see Section 7.3). The `version` subcommand MUST NOT accept an input file.
- Resolved output path: The final destination path computed from `--output` (when provided) or by replacing the first occurrence of `envseed` in the selected input path with `env` (see Section 7.5). When `--output` ends with a path separator or points to an existing directory, the derived file name (`envseed` -> `env`) is appended. Paths emitted in diagnostics (for example, dry-run header, diff headers) MUST be absolute.
