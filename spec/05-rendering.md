## 5. Rendering Specification
### 5.1 Rendering Pipeline
- Implementations MUST process the element sequence obtained in Section 4 in order.
  - `Blank` and `Comment` MUST be emitted as preserved, according to stored text and trailing-newline flags.
  - `Assignment` MUST write literal tokens verbatim and process placeholder tokens as follows.
- Placeholder token processing consists of the following stages:
  1) Resolve: obtain the value using the Resolver (see Section 6.2 for single-resolution and caching requirements). The resolver MUST return the raw value (including any trailing CR/LF/CRLF) and MUST NOT modify it.
  2) EOF newline normalization: If the resolved value ends with a logical newline (LF or CRLF), implementations MUST remove exactly one. If multiple consecutive newlines are present at EOF, implementations MUST remove exactly one and preserve the remainder. Internal newlines are unaffected.
  3) Preprocess: apply modifier preprocessing (Section 5.2). If `strip`/`strip_left`/`strip_right` are present, remove whitespace (Space/TAB/CR/LF) accordingly. If `base64` is present, perform encoding here per the combination rules.
  4) Context rules: apply allowance/forbiddance/escaping per the occurrence context (bare/double/single/command/backtick) as defined in Section 5.3. If disallowed, rendering MUST fail.
  5) Assemble: concatenate tokens and write to the output honoring the assignment operator, trailing comment, and trailing-newline flag.

### 5.2 Modifier Semantics
- This section defines the semantics of modifiers. The syntax and allowed modifier set follow Section 4.
- Applicability: All allowances and prohibitions in this section MUST be interpreted in conjunction with the per-context rules in Section 5.3.
- `dangerously_bypass_escape`
  - MUST bypass escaping and post-render re-parse validation (Section 5.4).
  - MUST NOT be combined with any other modifier. If combined, this is a render-time failure; see Section 7.10 for exit categorization and `docs/errors.md` for subcode mapping.
- `base64`
  - MUST encode the value using standard Base64.
  - MUST NOT be combined with any other modifier. If combined, this is a render-time failure; see Section 7.10 for exit categorization and `docs/errors.md` for subcode mapping.
  - The encoding alphabet MUST be `[A-Za-z0-9+/]` (standard Base64); `=` is used only as padding.
  - MUST NOT insert line breaks (no wrapping).
- `allow_newline` / `allow_tab`
  - Control allowance of newline (LF/CR/CRLF) and TAB per context. Specific allowances per context MUST follow Section 5.3.
- `strip` / `strip_left` / `strip_right`
  - The character set MUST be Space (U+0020), TAB (U+0009), LF (U+000A), and CR (U+000D).
  - `strip` MUST remove all leading and trailing runs of the above whitespace; `strip_left` MUST remove leading only; `strip_right` MUST remove trailing only.
  - MUST be allowed in combination with `allow_*`. MUST NOT be combined with `dangerously_bypass_escape` or `base64`.
- Application order
  - Implementations MUST apply a default EOF newline normalization before any modifier processing (see Section 5.1). Then implementations MUST apply modifiers after fetching the raw value and before context validation, in the following order: strip-family, then base64, then context validation/escaping.
  - Note: This ordering applies only to modifier sets that are permitted to coexist. `base64` is single-only; when present with any other modifier, it constitutes an invalid combination (see above) rather than an application-order case.

### 5.3 Context Rules and Escaping
Summary (Informative):
- Bare — Prohibit: NUL, disallowed control characters, newline (unrepresentable); Allow with modifier: TAB (`allow_tab`) [emitted as‑is]; Always escape: SPACE, #, $, ", ', `, \\, (, ), {, }, [, ], |, &, ;, <, >. Note: A leading `~` in the RHS MUST be escaped (to prevent tilde expansion) or placed into a quoted context.
- Double-quoted — Prohibit: NUL, disallowed controls; Allow with modifier: TAB (`allow_tab`), newline (`allow_newline`); Always escape: ", \\, $, `.
- Single-quoted — Prohibit: NUL, controls (other than TAB via modifier), newline, the single quote `'`.
- Command substitution ($(...)) — Prohibit: NUL, disallowed controls; Allow with modifier: TAB/newline; Always escape: \\, $, placeholder‑originated `)`.
- Backtick — Prohibit: NUL, disallowed controls, newline; Allow with modifier: TAB; Always escape: `, \\, $.
This section classifies, for each context, which characters cannot be emitted, which characters are permitted only when a modifier is present, and which characters are always permitted. The renderer MUST NOT change the quoting context chosen by the template author. The renderer MUST apply only context-local escaping within the original context. For clarity, `context-local escaping` means: in bare, adding a preceding backslash for characters that would alter lexical interpretation; in double-quoted, escaping " \\ $ and `; in command substitution ($(...)), escaping \\ $ and placeholder-originated ) only; in backticks, escaping ` \\ and $; in single-quoted, no escape exists for ' (prohibited).

For tokenization boundaries and structural forms by context, see Appendix D.4. Exact acceptance and escaping rules are defined by the normative lists in Sections 5.3.1-5.3.6 and by Section 4.1 for top-level comment detection.

Grammar scope note (Informative): Appendix D.4 specifies structural boundaries only. Where this section prohibits emitting certain code points (e.g., NUL or control characters), those prohibitions take precedence over any broader ranges that the ABNF may appear to admit.

#### 5.3.1 Common Prohibitions
- The NUL character U+0000 MUST NOT be emitted in any context.
- Control characters other than TAB and newline (C0/C1 ranges: U+0001–U+0008, U+000B, U+000C, U+000E–U+001F, U+007F, U+0080–U+009F) MUST NOT be emitted in any context.

#### 5.3.2 Modifier-Controlled Allowances
- TAB (U+0009) MUST NOT be emitted unless the `allow_tab` modifier is present; when `allow_tab` is present and the context permits TAB, the TAB in the secret value MUST be emitted as-is.
- Newline (LF/CR/CRLF) MUST NOT be emitted unless the `allow_newline` modifier is present; when `allow_newline` is present and the context permits newline, newlines in the secret value MUST be emitted as-is. When a context cannot represent newlines (e.g., bare, single_quoted, backtick), the prohibition takes precedence and newlines MUST NOT be emitted regardless of modifiers.

#### 5.3.3 Escaping Rules by Context
The renderer MUST apply the following escaping rules whenever the corresponding characters occur in the given context:
- Bare: characters that would otherwise change lexical interpretation in a bare assignment (as exhaustively defined by the unconditional matrix plus conditional rules) MUST be emitted with a preceding backslash. Top-level `#` escaping for trailing comment detection is defined in Section 4.1.
- Double-quoted: `"`, `\\`, `$`, and `` ` `` MUST be escaped with a preceding backslash.
- Single-quoted: no escape mechanism exists for the single quote U+0027; see Section 5.3.6. Other characters are emitted verbatim.
- command_subst (`$(...)`): `\\` and `$` MUST be emitted with a preceding backslash. Any `)` originating from placeholder content MUST be escaped as `\\)`. The syntactic closing parenthesis of the substitution MUST NOT be escaped.
- Backtick (`` `...` ``): `` ` ``, `\\`, and `$` MUST be emitted with a preceding backslash to preserve literal meaning.

Required escaping (unconditional) matrix.

This table lists the unconditional required escapes. Conditional rules are specified in Sections 4.1 and 5.3; together they deterministically define escaping.

Implementations MUST NOT introduce escapes beyond (a) the unconditional set in this matrix and (b) the conditional cases explicitly defined in this specification. For any input, the decision is deterministic.

Decision procedure: (1) Prohibitions (e.g., NUL or disallowed control characters) — reject; (2) Unconditional escapes (this matrix) — always escape; (3) Conditional escapes (e.g., placeholder‑originated ')' in $(...), top‑level '#' per Section 4.1, and leading `~` in Bare per Section 5.3.4). The combination yields a unique result.

```
Context        Characters that MUST be escaped per Section 5.3.3
-------------- --------------------------------
Bare           SPACE, #, $, ", ', `, \, (, ), {, }, [, ], |, &, ;, <, >
Double-quoted  ", \, $, `
Single-quoted  
$(...)         \, $, )
Backtick       `, \, $
```

- Note 1: Single-quoted has no escape mechanism; ' (U+0027) is prohibited.
- Note 2: In $(...), only placeholder-originated ) MUST be escaped; the syntactic closing parenthesis MUST NOT be escaped.
- Note 3: In Bare, a leading `~` at the beginning of the RHS MUST be escaped (to prevent tilde expansion) or placed into a quoted context. This is a conditional (position‑dependent) rule, not part of the unconditional matrix.

#### 5.3.4 Bare Context (`VAR=<pass:...>`)
- The following characters MUST NOT be emitted: NUL; control characters other than TAB/newline; newline (unrepresentable in bare even when `allow_newline` is present).
- TAB MUST NOT be emitted unless the `allow_tab` modifier is present. When `allow_tab` is present, TAB in the secret value MUST be emitted as‑is (no preceding backslash added by the renderer).
- All other Unicode code points MUST be accepted. Characters requiring protection to preserve literal meaning in a bare assignment MUST be backslash-escaped as specified in Section 5.3.3 (Bare). Top-level `#` MUST be treated per Section 4.1 (an odd number of preceding backslashes yields a literal `#`; otherwise it begins a trailing comment).
- A leading `~` at the beginning of the RHS MUST be escaped (conditional rule) or placed into a quoted context to prevent tilde expansion. This conditional rule complements the unconditional matrix; see Note 3 above for scope and rationale.

#### 5.3.5 Double-Quoted Context (`"..."`)
- The following characters MUST NOT be emitted: NUL; control characters other than TAB/newline.
- TAB MUST NOT be emitted unless `allow_tab` is present; newline MUST NOT be emitted unless `allow_newline` is present.
- All other Unicode code points MUST be accepted, with escaping applied per Section 5.3.3 (Double-quoted).

#### 5.3.6 Single-Quoted Context (`'...'`)
- The following characters MUST NOT be emitted: NUL; control characters other than TAB; newline; the single quote U+0027 (no escaping is available in this context).
- TAB MUST NOT be emitted unless `allow_tab` is present.
- All other Unicode code points MUST be accepted (emitted verbatim).

#### 5.3.7 Command Substitution Context (`$(...)`)
- The following characters MUST NOT be emitted: NUL; control characters other than TAB/newline.
- TAB MUST NOT be emitted unless `allow_tab` is present; newline MUST NOT be emitted unless `allow_newline` is present.
- All other Unicode code points MUST be accepted, with escaping applied per Section 5.3.3 (command_subst).

#### 5.3.8 Backtick Context (`` `...` ``)
- The following characters MUST NOT be emitted: NUL; control characters other than TAB; newline.
- TAB MUST NOT be emitted unless `allow_tab` is present.
- All other Unicode code points MUST be accepted, with escaping applied per Section 5.3.3 (Backtick). Note: Escaping rules for backticks vary across shells; authors SHOULD prefer `$(...)` where portability is a concern (Informative).

### 5.4 Error Semantics, Post-render Re-parse, and Subcodes
- This section defines the principles for classifying render-time failures.
  - Failures defined by this specification MUST be classified under exit code 105 with unique subcodes. See Section 7.10 for exit categories and `docs/errors.md` for canonical mapping.
  - Context violations and invalid modifier combinations are render-time failures (exit code 105) with unique subcodes.
  - When `dangerously_bypass_escape` is not used, a post-render re-parse failure MUST be assigned a dedicated unique subcode under exit code 105 (see Section 7.10).

#### 5.4.1 Context-specific failures
Implementations MUST surface the following context-specific failures under exit code 105 and assign unique subcodes. Implementations MUST choose the most specific matching category; subcodes are defined in `docs/errors.md`.

- Single-quoted
  - contains single quote (')
    - Guidance: Switch to a different quoting context that can represent `'` (e.g., double quotes with escaping).
  - contains newline (unrepresentable in single-quoted)
    - Guidance: Switch to double quotes and add the allow_newline modifier.
  - contains TAB without the `allow_tab` modifier
    - Guidance: Add the allow_tab modifier in single quotes, or switch to double quotes and add the allow_tab modifier.
  - contains control character (other than TAB/newline)
    - Guidance: Such control characters are unsupported in single quotes; adjust the value or encoding.

- Double-quoted
  - contains newline without the `allow_newline` modifier
    - Guidance: Add the allow_newline modifier.
  - contains TAB without the `allow_tab` modifier
    - Guidance: Add the allow_tab modifier.
  - contains control character
    - Guidance: Such control characters are unsupported; adjust the value or encoding.

- Command substitution (`command_subst`)
  - contains newline without the `allow_newline` modifier
    - Guidance: Add the allow_newline modifier.
  - contains TAB without the `allow_tab` modifier
    - Guidance: Add the allow_tab modifier.
  - contains control character
    - Guidance: Such control characters are unsupported.

- Backtick (`backtick`)
  - contains newline (unrepresentable in backtick)
    - Guidance: Consider replacing backticks with $(), or switch to double quotes and add the allow_newline modifier.
  - contains TAB without the `allow_tab` modifier
    - Guidance: Add the allow_tab modifier or switch to a different quoting context.
  - contains control character
    - Guidance: Such control characters are unsupported.

- Bare
  - contains newline (unrepresentable in bare)
    - Guidance: Switch to double quotes and add the allow_newline modifier.
  - contains TAB without the `allow_tab` modifier
    - Guidance: Switch to double quotes or add the allow_tab modifier.
  - contains control character
    - Guidance: Quote or encode the value.
  - contains non-bare characters (backslash-escape to preserve lexical boundaries)
    - Guidance: The renderer MUST preserve lexical boundaries by inserting backslashes before characters that would change bare interpretation (per Section 5.3.3). Quoting by the template author is optional for readability; the renderer does not change the chosen quoting context, and auto-escaping suffices. When appropriate (e.g., the secret contains many control characters or mixed newlines that the chosen context cannot represent cleanly), consider using `<pass:PATH|base64>` and place the placeholder in a context that can represent it.

- Modifiers (common)
  - invalid modifier combination
    - Guidance: Do not combine base64 with other modifiers; do not combine dangerously_bypass_escape with any modifier.
