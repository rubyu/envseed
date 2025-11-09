## 6. Security Model

This chapter defines the security invariants EnvSeed MUST satisfy. RFC 2119 terminology applies.

### 6.1 Security Invariants
- Real secrets MUST NOT be emitted to stdout or stderr, including informational logs, diagnostics, and error reports.
- When `dangerously_bypass_escape` is not present, rendered output MUST be re-validated by the parser. Revalidation requirements and failure reporting follow Section 5.4.
- Text produced by `sync --dry-run` and `diff` MUST be redacted per Section 6.3 and follow the CLI stream/output policies (see Sections 7.7 and 7.8).
- Secrets MUST be kept in-process only, and the cache MUST be cleared by process termination.
- When content changes, output files MUST be written atomically. File permissions MUST be `0600`.

### 6.2 Resolver & Secret Lifecycle
- Secret retrieval is limited to in-process resolution. Calls to `pass show <PATH>` MUST be limited to one per PATH during execution using an in-process cache.
- The resolver MUST NOT modify the output of `pass`. EOF newline normalization is defined in Section 5.1; any additional adjustments to trailing whitespace or newlines are controlled by modifiers (`strip`/`strip_right`/`strip_left`).
- Values that contain NUL bytes are invalid. See Appendix D.1 for template-time prohibition and Section 7.10 for resolver-time exit categorization.
- The resolver MUST NOT be used after it is closed. Violations are internal errors and are assigned unique subcodes.
- The cache is limited to the lifetime of the process and is cleared on process termination (see Section 6.1).

Resolver interaction (Normative)
- EnvSeed launches `pass show <PATH>` and connects the child’s stdin so that interactive pinentry can prompt the user.
- EnvSeed MUST NOT set or override `GPG_TTY`.
- EnvSeed MUST NOT change pinentry/gpg-agent environment or configuration, and MUST NOT force non-interactive loopback or read passphrases from stdin.
- Operators are responsible for configuring TTY pinentry when needed (e.g., `export GPG_TTY=$(tty)`).

### 6.3 Redaction Policy & Algorithm
Secret-free outputs.
- Implementations MUST ensure that no secret values appear on stdout, stderr, or in logs under any circumstances.

Line-boundary invariants.
- Redaction MUST NOT insert or remove newline characters. The number and positions of line breaks in redacted texts MUST remain identical to the pre-redaction texts.

Masked text construction.
- Let A be the current target file content and B be the newly rendered content. Implementations MUST construct masked variants A′ and B′ as follows:
  1) Parse A and B as .env files using the same grammar as the template parser for assignments, comments, and blank lines. Placeholders do not appear in target `.env` files; only lexical boundaries common to both grammars are considered. Any parse failure MUST terminate processing with an error. Fallback masking MUST NOT be used.
  2) For each assignment line, traverse the value using a context stack (bare, double_quoted, single_quoted, command_subst, backtick) and replace only string segments with mask characters while preserving syntactic delimiters.
     Note: In this context, `required backslashes` are those necessary to preserve literal meaning in the given quoting/substitution context (see Section 1.2).
  3) Masking rule: Replace all code points of string segments with the ASCII asterisk `*` (U+002A) while preserving newline positions. Backslashes that form an escape pair inside string segments MUST NOT be emitted as backslashes; instead, elide the backslash and mask the escaped code point. Implementations MAY apply a length‑bounded head/tail reveal for readability, but MUST NOT reveal any code point that belongs to an escape pair; if a reveal window would expose an escape pair, shift or shrink the window, or fully mask the unit. Delimiting tokens (quotes, `$(`, `)`, backticks, and a `$` that starts a substitution), variable names, assignment operators, and top‑level trailing comments (beginning with `#`) MUST be preserved unmodified.
  4) Nested quoting and substitutions MUST be handled correctly. The delimiters for `"..."`, `'...'`, `$(...)`, and `` `...` `` MUST be preserved; only their interior string segments are masked per the rule above.

Diff reconstruction using masked texts.
- Implementations MUST compute the unified diff on raw A and B (not emitted), then reconstruct the emitted diff using A′ and B′ as defined in Section 7.8.

Note:
- Reveal thresholds are defined in terms of Unicode scalar values (code points), not bytes.

### 6.4 Streams & Logging Policy
- In non-dry-run execution, rendered content MUST NOT be written to stdout. Artifacts MUST be written only to files.
- In `sync --dry-run` and `diff`, stdout MUST NOT contain unredacted template content. Only redacted content and non-secret metadata lines are allowed. Formatting and the `target:` header follow Sections 7.7 and 7.8.
- Exceptions, diagnostics, and informational logs MUST NOT include secrets. Apply masking as needed.

### 6.5 Output Artifacts & Permissions
- When content changes, writing MUST be atomic: write to a temporary file and replace with `rename(2)`.
- Output file permissions MUST be `0600`. When content is unchanged, implementations MUST NOT perform write/replace.

### 6.6 Dangerous Mode Considerations
- For placeholders that specify `dangerously_bypass_escape`, implementations MUST NOT perform context-aware escaping or post-render re-parse validation.
- Use of this modifier may render output invalid for shells. Other requirements in this chapter (no secret exposure on streams/logs, file permissions, cache clearing, etc.) MUST continue to apply.
  Note (Informative): This modifier is powerful and shifts all responsibility to the template author. Misuse undermines safety.
  Informative examples: Unescaped quotes or closing delimiters may lead to outputs that cannot be re-parsed or that fail `bash -n`, and may change lexical boundaries at runtime. Teams SHOULD avoid this modifier in production or require explicit peer review and targeted tests when its use is unavoidable.
