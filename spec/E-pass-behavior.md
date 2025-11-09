## Appendix E. Pass Behavior: pass show semantics and whitespace (Informative)

### E.1 Scope and Sources
- Scope. This appendix documents the behavior of the upstream password-store implementation (pass) relevant to EnvSeed's renderer and resolver. This appendix is informative; normative requirements remain in Sections 5.1 and 6.2.
- Sources. Upstream repository zx2c4/password-store (commit 3ca13cd8882cae4083c1c478858adbf2e82dd037), pass(1) man page in that repository.

### E.2 pass show (stdout)
- pass show prints the decrypted file bytes exactly as stored. It does not trim or add whitespace; trailing newlines and spaces are preserved.
- Rationale. The implementation base64-encodes the decrypted stream into a shell variable and decodes back to stdout, preserving bytes including trailing newline(s).

### E.3 pass show --clip/--qrcode
- When --clip or --qrcode is used, pass selects a single line and omits its trailing newline; spaces within the line are preserved.
- Implementation detail. A tail/head pipeline selects the requested line; command substitution strips the line-ending newline, and the clipboard/QR emission uses echo -n, preventing a newline from being added.

### E.4 Entry creation and whitespace
- pass insert (single-line) uses a Bash read pipeline before encryption, which trims leading/trailing IFS whitespace (space/tab/newline) before storing, then writes a final newline at EOF.
- pass insert -m (multiline) and pass edit preserve whitespace exactly as typed/saved; whether the file ends with a newline depends on user input/editor behavior (many editors add one by default).
- pass generate writes the generated password with a trailing newline at EOF.

### E.5 Examples
- 'abc\n' -> pass show prints with trailing LF.
- 'abc\r\n\r\n' -> pass show prints with two trailing CRLF sequences.
- 'a\nb\n' -> pass show prints with an internal newline and a trailing newline.
- With --clip/--qrcode on a line that ends with a newline, the newline is not included in the copied/encoded value; spaces in the line are preserved.

### E.6 Interactions with EnvSeed
- Normative references:
  - Resolver and rendering behaviors follow Sections 6.2 and 5.1 (normative). Context rules are defined in Section 5.3.
- Implication. Values retrieved by EnvSeed may include trailing newline(s) depending on how the entry was created. EnvSeed's renderer normalizes exactly one trailing logical newline by default; further trimming (e.g., spaces/TAB/newlines) is controlled by modifiers.
