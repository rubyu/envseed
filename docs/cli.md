# EnvSeed CLI Reference
```
 _____             ____                _
| ____|_ ____   __/ ___|  ___  ___  __| |
|  _| | '_ \ \ / /\___ \ / _ \/ _ \/ _` |
| |___| | | \ V /  ___) |  __/  __/ (_| |
|_____|_| |_|\_/  |____/ \___|\___|\__,_|
🌱   EnvSeed - grow your .env from seeds
```

This document describes the envseed command line interface in detail: commands, flags, path resolution rules, output behavior, and exit codes. For installation and a short introduction, see the project README.

## Overview
### Commands
- `sync` — Render a template and write the target `.env` file.
- `diff` — Render in memory and print a redacted unified diff.
- `validate` — Parse the template and report syntax/lexing errors.
- `version` — Print the EnvSeed version string.

### General Rules
- `sync`/`diff`/`validate` can omit `[INPUT_FILE]`. If omitted, envseed uses `./.envseed` as the input file. `version` accepts no input file. Stdin is not supported.
- When `--output` is omitted, the input file name must contain `envseed` (validate is exempt).
- Rendered secrets are never printed to stdout. Informational messages go to stderr and can be suppressed with `--quiet`.
- `--version` (global): Recognized at any position. Prints exactly one line containing the version string to stdout and exits `0`; ignores other flags/args and does not write to stderr.
- Unknown or missing commands return exit `101`.
- Unsupported option combinations return exit `101`.

## Path Resolution
1) Choose candidate: use `--output` when provided; otherwise replace the first `envseed` in the input path (the explicit `INPUT_FILE`, or `./.envseed` when omitted) with `env` to form the candidate path.
2) Interpret directories: if the candidate ends with a path separator or resolves to an existing directory, join the derived file name (`envseed` → `env`).
3) Validate path: a missing parent directory returns exit `106`. If the candidate resolves to an existing directory (treating a directory as a file), return exit `101`. Non‑existent files are valid targets.

## Commands
### sync

```
╔════════════════════════════════════════════════════╗
║ envseed  sync  [flags]  [INPUT_FILE]               ║
║          ────                                      ║
╚════════════════════════════════════════════════════╝
```

Render the template and write the output file.

#### Flags
- `--output`, `-o <PATH>` — Override destination path.
- `--force`, `-f` — Allow overwrite of an existing file.
- `--dry-run` — Do not write; print a redacted preview instead.
- `--quiet`, `-q` — Suppress informational messages (errors are not suppressed).

#### Behavior
- Writes are atomic (temporary file + rename). Final permissions are `0600`.
- If content changes: `wrote <path> (mode 0600)` is printed to stderr.
- If content is unchanged: `wrote <path> (unchanged)` is printed to stderr.
- In non‑dry‑run, rendered content is not printed to stdout.
- Dry‑run details: The first line is `target: <absolute output path>`. The path is computed by OS‑level absolutization without resolving symbolic links. Stdout contains only this header and redacted content; informational logs go to stderr (suppressed by `--quiet`). The target path resolves exactly as a real write would (including `--output`).

### diff

```
╔════════════════════════════════════════════════════╗
║ envseed  diff  [flags]  [INPUT_FILE]               ║
║          ────                                      ║
╚════════════════════════════════════════════════════╝
```

Render in memory and print a redacted unified diff.

#### Flags
- `--output`, `-o <PATH>` — Select the comparison target without changing the template read path.

#### Behavior
- If the target does not exist, compare against empty content (all additions).
- When `--output` names a directory, envseed derives the comparison file inside that directory by replacing the first `envseed` in the input path (the explicit `INPUT_FILE`, or `./.envseed` when omitted) with `env`; otherwise it compares against the exact path provided.
- Unified diff with `---`, `+++`, and `@@` hunk markers and context lines. Unified diff headers use the first two lines `--- <path>` and `+++ <path>`, where each `<path>` is the absolute output path and both paths are byte‑identical. EnvSeed does not add prefixes or annotations. No differences → stdout/stderr remain silent.
- Comparisons larger than 10 MiB are rejected (exit `108`, e.g., `EVE-108-1`).
- Redaction: diff output is reconstructed from masked A′/B′ per spec/06-security.md §6.3 (Redaction Policy & Algorithm).

#### Exit Codes
- `0` when files match; `1` when differences exist.

### validate
```
╔════════════════════════════════════════════════════╗
║ envseed  validate  [flags]  [INPUT_FILE]           ║
║          ────────                                  ║
╚════════════════════════════════════════════════════╝
```

Parse the template and report errors. Does not contact `pass`.

#### Behavior
- No additional flags are accepted. Unexpected flags return exit `101`.
- Success is silent. Errors are printed to stderr with exit `103`.

### version

```
╔════════════════════════════════════════════════════╗
║ envseed  version                                   ║
║          ───────                                   ║
╚════════════════════════════════════════════════════╝
```

Print the EnvSeed version string.

#### Behavior
- Accepts no additional positional arguments or flags; unexpected options/args return exit `101`.
- Prints exactly one line containing the version string to stdout and does not write to stderr.
- Exit code is always `0`.

## Template Language
Placeholders have the form `<pass:PATH>` or `<pass:PATH|modifier[, modifier...]>`.
Placeholders appear on the right-hand side of assignment lines and may be placed inside any of the supported contexts.
EnvSeed detects the surrounding context and renders the secret with minimal, context-appropriate escaping before writing the `.env` file.

```sh
PASSWORD='<pass:www.example.com/password>'
```

For detailed rules on placeholders and modifiers, see spec/05-rendering.md and spec/04-parsing.md.

### Contexts
- Bare: Appears outside of quotes or command forms.
- Double-quoted (`"..."`): Standard double-quoted string.
- Single-quoted (`'...'`): Standard single-quoted string.
- Command substitution (`$(...)`): Shell-style command substitution context.
- Backticks (`` `...` ``): Legacy backtick command substitution context.

### Modifiers
- `allow_newline` — Permit newline characters (double‑quoted or command substitution only).
- `allow_tab` — Permits literal TAB characters (`U+0009`) in contexts that otherwise reject them. It is required to retain TAB inside single-quoted or backtick placeholders; other control characters remain unsupported.
 - `base64` — Base64‑encode the secret (must appear alone; cannot be combined with other modifiers).
- `dangerously_bypass_escape` — Insert the secret verbatim (disables validation; use with caution).
- `strip` — Remove leading and trailing runs of SPACE/TAB/CR/LF.
- `strip_left` — Remove leading runs of SPACE/TAB/CR/LF.
- `strip_right` — Remove trailing runs of SPACE/TAB/CR/LF.

### Combination rules
- `base64` is single‑only; combining `base64` with any other modifier is invalid.
- `dangerously_bypass_escape` cannot be combined with other modifiers.

## Exit Codes
The CLI uses the following exit codes:
- `0` success; `1` differences exist (diff only)
- `101` invalid input
- `102` template read failure
- `103` template parsing failure
- `104` resolver failures
- `105` rendering failures
- `106` output failures
- `107` target parsing failure
- `108` diff failures
- `199` unexpected internal exception

In addition, the CLI outputs the corresponding detailed error code follows the form `EVE-<exit>-<sub>`. 
CLI errors are prefixed as: `envseed ERROR [EVE-<exit>-<sub>]: <message>` and optional subsequent lines may include additional details.
For complete definitions including sub-band allocation within each exit category, see spec/07-cli.md §7.

## Security Note
Secrets are never printed to stdout. Dry‑run and diff apply redaction per spec/06-security.md §6.3 with a length‑bounded head/tail reveal: newline positions are preserved; only interior characters within string segments are replaced with `*`, while variable names, operators, and syntactic delimiters (quotes, $(), backticks) are preserved.

## Authority Note
This document is a concise, non‑normative reference. The normative CLI contract is defined in spec/07-cli.md, and the redaction policy in spec/06-security.md (§6.3). Subcode mapping (numbers/messages/guidance) is generated from internal/envseed/errors.go to docs/errors.md. In case of any discrepancy, the spec prevails.
