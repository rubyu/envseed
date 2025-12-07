## 4. Parsing Specification
### 4.1 Bash-Compatible Assignment and Lexical Rules
- Recognized assignments: `NAME=value`, `NAME+=value`, and `NAME[INDEX]=value` (aligned with Bash `assignment_word`).
- Assignment name: MUST begin with a letter or `_`; subsequent characters MUST be alphanumeric or `_`. Bracket notation in `NAME[INDEX]` is accepted, but an unclosed bracket or an extra `]` is a parse error.
- `+=`: Interpreted as additive assignment. A `+` that does not form `+=` in the operator position is a parse error (invalid as part of the assignment name).
- Missing `=`: If `=` does not appear by end of line, implementations MUST raise a parse error (exit code 103, Template parsing failure). See Section 7.10 and `docs/errors.md` for exit categorization and canonical messages.
- Quoting/substitution recognition: double quotes `"..."`, single quotes `'...'`, command substitution `$(...)`, and backticks `` `...` `` are recognized for structure only and are not evaluated. Unterminated constructs are parse errors.
- Comments:
  - Whole-line comment: a line whose first non-whitespace character is `#` (leading whitespace allowed) is treated as a Comment.
  - Trailing comment: top-level `#` detection (odd/even backslashes) follows Appendix D.2. A `#` inside quotes or inside `$(...)`/backticks is literal and does not begin a comment.
- Command substitution parentheses: track nesting depth of `$(...)`; unterminated constructs are parse errors.
- Preservation policy: implementations MUST preserve literal whitespace and escape sequences as written, except where this specification explicitly defines normalization.

- Whitespace tokens are defined in Appendix D.1 (Space/Tab-only). The grammar for files/elements/assignments/trailing comments is defined in Appendix D.2–D.3.

### 4.2 Elements and Tokens (AST)
- The parser produces `Element` and `Assignment`/`ValueToken` as defined in Section 3 (Data Model).
- Each element preserves order and records whether the line ends with a newline (trailing-newline flag).
- An Assignment records leading whitespace, name, operator, value token sequence, trailing comment, source line/column, and trailing-newline flag.
- A ValueToken records kind (`Literal`/`Placeholder`), text, context, and for `Placeholder` its PATH and modifiers, plus source line/column.

### 4.3 Placeholder Syntax (EnvSeed Extension)
- Form
  - A placeholder MUST be either "<pass://PATH>" or "<pass://PATH|modifier[, modifier...]>".
  
  Note: The rules in this section apply to the placeholder body only and do not affect the lexical preservation policy for template text outside placeholders (see Section 4.4).
- Sigil strictness
  - A placeholder sigil MUST start with the literal sequence "<pass://" with no intervening whitespace. Implementations MUST NOT treat any other `<pass...>` sequence as a valid placeholder.
  - Placeholder-like sequences that violate these sigil rules (for example, `<pass` not followed by `://`, or whitespace inserted between `pass` and `://`) MUST be reported as parse errors with source position; see Section 4.5 for error classification and diagnostics.
- Whitespace and PATH rules
  - Detailed rules for PATH characters, whitespace trimming around PATH and separators, and the full placeholder grammar (including modifiers) are defined in Appendix D.5. Implementations MUST follow that grammar.
- Recognized modifiers (case-sensitive)
  - `allow_newline`
  - `allow_tab`
  - `base64`
  - `dangerously_bypass_escape`
  - `strip`
  - `strip_left`
  - `strip_right`
- Parse-time validation
  - Placeholder syntax errors (for example, empty/unknown/duplicate modifiers, invalid PATH, or the presence of newlines or NUL inside the placeholder body) MUST be reported as parse errors with exit code 103. The precise grammar and character-level rules are defined in Appendix D.5.
- Relation to context (reference)
  - A placeholder MUST record the occurrence context (bare/double/single/command/backtick). Per-context allowance/forbiddance/escaping rules MUST follow Section 5.3.
- Examples
  - Example (valid): `VAR=<pass://service/api-token>`.
  - Example (invalid): `VAR=<pass://>` (empty PATH; see Appendix A.7 for placeholder syntax error examples).

### 4.4 Determinism and Preservation
- Lexical preservation: implementations MUST preserve whitespace, escape sequences, and original fragments as written, except where explicitly specified otherwise.
- Comments: per Section 4.1, preserve whole-line comments and trailing comments in their respective forms.
- Newlines: each Element retains the newline flag to allow reconstructing the original line structure.

### 4.5 Parse Errors and Diagnostics
- Parse error subjects: invalid assignment names; unterminated quotes/command substitutions/backticks; placeholder syntax errors (empty/unknown/duplicate modifiers, invalid PATH, presence of newline or NUL, etc.).
- Sigil violation: placeholder sigil errors (for example, `<pass` not followed by `://` or whitespace inserted between `pass` and `://`) MUST be reported as parse errors with source position (line and column). Exact sigil rules are defined in Section 4.3 and Appendix D.5, and diagnostics SHOULD recommend rewriting the token using the `<pass://PATH>` placeholder form (see Section 7.11).
- Space/Tab-only whitespace violations: whitespace recognized by this specification is limited to ASCII Space (U+0020) and Tab (U+0009). The following MUST be classified as parse errors (exit code 103). See Section 7.10 and `docs/errors.md` for subcode assignment.
  - Use of any other Unicode whitespace in leading whitespace at line start.
  - Use of any other Unicode whitespace around placeholder separators (`|`, `,`, `>`), or adjacent to `PATH` for trimming.
- Band classification (Informative): See Section 7.10.1 for bands (e.g., `EVE-103-B0`). Canonical mapping (numbers, messages, guidance) is maintained in `docs/errors.md`.
- Source location: all parse errors MUST include line and column.
- Exit code: these failures MUST use exit code 103 (Template parsing failure). Diagnostic label format follows Section 7.11 (CLI Diagnostics).
### 4.6 Bash Behavior Validation (Informative)
Informative Bash observations and minimal reproductions have been moved to Appendix F. See Appendix F (Bash Behavior Validation) for examples that motivate where the parser aligns with Bash syntax.
