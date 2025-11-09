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
  - A placeholder MUST be either `<pass:PATH>` or `<pass:PATH|modifier[, modifier...]>`.
  
  Note: The rules in this section apply to the placeholder body only and do not affect the lexical preservation policy for template text outside placeholders (see Section 4.4).
- Sigil strictness
  - The leading token MUST be exactly `<pass:`.
  - Whitespace between `pass` and `:` (e.g., `<pass :`) MUST NOT occur; encountering it MUST be reported as a parse error with source position (see Section 4.5). For this rule, `whitespace` means ASCII SPACE (U+0020) and TAB (U+0009) only. Line terminators (CR U+000D, LF U+000A) are also prohibited within the sigil.
- Whitespace handling
  - Trimming and separator-adjacent whitespace MUST follow Appendix D.5 (Space/Tab only; newlines prohibited). Violations are parse errors (exit code 103); see Section 4.5.
- Grammar for placeholders (sigil strictness, Space/Tab only around separators and PATH trimming, modifier list) is defined in Appendix D.5. PATH MAY contain non-ASCII Unicode except NUL/line terminators/separators; see Appendix D.5 notes.
- Recognized modifiers (case-sensitive)
  - `allow_newline`
  - `allow_tab`
  - `base64`
  - `dangerously_bypass_escape`
  - `strip`
  - `strip_left`
  - `strip_right`
- Parse-time validation
  - Unknown, duplicate, or empty modifiers MUST be reported as parse errors.
  - The placeholder body MUST NOT contain newlines (LF/CR/CRLF) or NUL. Input that crosses lines before reaching `>` MUST be reported as a parse error.
- Relation to context (reference)
  - A placeholder MUST record the occurrence context (bare/double/single/command/backtick). Per-context allowance/forbiddance/escaping rules MUST follow Section 5.3.
- Accepted examples (valid)
  - `<pass:path>`
  - `<pass: path >`  (Space/Tab-only trimming is permitted; see Appendix D.5)
  - `<pass:path|allow_tab>`
  - `<pass:path | allow_tab , allow_newline >`
  - `<pass:path|strip>`
  - `<pass:path|strip_left,allow_tab>`
  - `<pass:path | strip_right , allow_newline >`
- Rejected examples (invalid)
  - `<pass : path>` (whitespace inside sigil)
  - `<pass:>` (empty PATH)
  - `<pass:path|>` (empty modifier)
  - `<pass:path|strip,>` (trailing empty modifier)
  - `<pass:path|strip,strip>` (duplicate modifier)
  - `<pass:path\n|strip>` (contains newline)
  - `<pass:path|base64,strip>` (valid syntax; invalid combination at render time — see Section 5.2)
  - `<pass:path|dangerously_bypass_escape,strip>` (valid syntax; invalid combination at render time — see Section 5.2)

### 4.4 Determinism and Preservation
- Lexical preservation: implementations MUST preserve whitespace, escape sequences, and original fragments as written, except where explicitly specified otherwise.
- Comments: per Section 4.1, preserve whole-line comments and trailing comments in their respective forms.
- Newlines: each Element retains the newline flag to allow reconstructing the original line structure.

### 4.5 Parse Errors and Diagnostics
- Parse error subjects: invalid assignment names; unterminated quotes/command substitutions/backticks; placeholder syntax errors (empty/unknown/duplicate modifiers, invalid PATH, presence of newline or NUL, etc.).
- Sigil violation: if the parser encounters the sequence `<pass` followed by any whitespace prior to `:`, it MUST raise a parse error at the position of the offending character. The diagnostic MUST identify the sigil violation (whitespace between `pass` and `:`) and include source position (line and column).
- Space/Tab-only whitespace violations: whitespace recognized by this specification is limited to ASCII Space (U+0020) and Tab (U+0009). The following MUST be classified as parse errors (exit code 103). See Section 7.10 and `docs/errors.md` for subcode assignment.
  - Use of any other Unicode whitespace in leading whitespace at line start.
  - Use of any other Unicode whitespace around placeholder separators (`|`, `,`, `>`), or adjacent to `PATH` for trimming.
- Band classification (Informative): See Section 7.10.1 for bands (e.g., `EVE-103-B0`). Canonical mapping (numbers, messages, guidance) is maintained in `docs/errors.md`.
- Source location: all parse errors MUST include line and column.
- Exit code: these failures MUST use exit code 103 (Template parsing failure). Diagnostic label format follows Section 7.11 (CLI Diagnostics).
### 4.6 Bash Behavior Validation (Informative)
Informative Bash observations and minimal reproductions have been moved to Appendix F. See Appendix F (Bash Behavior Validation) for examples that motivate where the parser aligns with Bash syntax.
