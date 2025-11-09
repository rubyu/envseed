## Appendix A. Examples as Fixtures (Informative)

This appendix provides ready-to-use fixtures grouped by context and purpose. Unless stated otherwise:
- pass output is shown with comments like `# pass show path => value`.
- Each example indicates expected exit code where applicable (see Section 7.10) and references the normative section.

### A.1 Conventions
- Success examples indicate that `parse -> render -> parse` succeeds; when possible, `bash -n` also succeeds.
 - Failure examples indicate the error category (classified with a unique subcode) or a parse error; guidance refers to Section 5.4.1.

### A.2 Bare Context (`VAR=<pass:...>`)
Status: Success (exit code 0)
```sh
# pass show service/api-token => tokenValue
API_TOKEN=<pass:service/api-token>

# Base64 stays in the bare-allowed alphabet
TOKEN_B64=<pass:service/token|base64>

# Space and other special chars are backslash-escaped automatically per Section 5.3.3
# pass show example => hello world
VALUE_WITH_SPACE=<pass:example>    # renderer emits hello\ world (reparsable)
```
- Style: values containing spaces are easier to read and maintain when quoted; prefer `VAR="<pass:...>"` for readability.

Status: Failure (exit code 105)
```sh
# pass show example => a\tb
VAR=<pass:example>                  # rendering error: TAB without allow_tab (bare; see Section 5.3.4)

# pass show example => line1\nline2
VAR=<pass:example>                  # rendering error: newline unsupported (bare; see Section 5.3.4)

# pass show example => ping\x01
VAR=<pass:example>                  # rendering error: control character (bare; see Section 5.3.4)
```

### A.3 Double-Quoted (`VAR="..."`)
Status: Success (exit code 0)
```sh
# Escaping of " \ $ ` is applied automatically
AUTH_HEADER="Bearer <pass:auth/token>"

# Newline allowed when requested
SCRIPT="$(printf %s \"<pass:deploy/script|allow_newline>\")"

# TAB allowed when requested
MESSAGE="<pass:alert/body|allow_tab>"
```

Status: Failure (exit code 105)
```sh
# pass show val => line1\nline2
VAL="<pass:val>"                # rendering error: newline without allow_newline (double-quoted; see Section 5.3.5)

# pass show val => hello\tworld
VAL="<pass:val>"                # rendering error: TAB without allow_tab (double-quoted; see Section 5.3.5)

# pass show val => ping\x01
VAL="<pass:val>"                # rendering error: control character (double-quoted; see Section 5.3.5)
```

### A.4 Single-Quoted (`VAR='...'`)
Status: Success (exit code 0)
```sh
# pass show example => abcDEF123
SINGLE_OK='<pass:example>'

# pass show example_tab => hello\tworld
SINGLE_OK_TAB='<pass:example_tab|allow_tab>'
```

Status: Failure (exit code 105)
```sh
# pass show example_quote => O'Connor
SINGLE_FAIL_QUOTE='<pass:example_quote>'    # rendering error: single quote not allowed (single-quoted; see Section 5.3.6)

# pass show example_nl => line1\nline2
SINGLE_FAIL_NL='<pass:example_nl>'          # rendering error: newline unsupported (single-quoted; see Section 5.3.6)

# pass show example_tab => hello\tworld
SINGLE_FAIL_TAB='<pass:example_tab>'        # rendering error: TAB without allow_tab (single-quoted; see Section 5.3.6)

# pass show example_ctrl => ping\x07
SINGLE_FAIL_CTRL='<pass:example_ctrl>'      # rendering error: control character (single-quoted; see Section 5.3.6)

SINGLE_FAIL_MOD='<pass:x|allow_newline>'    # rendering error: allow_newline unsupported (single-quoted; see Section 5.3.6)
```

### A.5 Command Substitution (`VAR=$(...)`)
Status: Success (exit code 0)
```sh
# pass show val => value)tail
CMD=$(printf %s <pass:val>)      # closing ')' is escaped as \)

# pass show val => a\tb
CMD=$(printf %s <pass:val|allow_tab>)

# pass show val => line1\nline2
CMD=$(printf %s <pass:val|allow_newline>)
 

Status: Failure (exit code 105)
```sh
# pass show val => line1\nline2
CMD=$(printf %s <pass:val>)      # rendering error: newline without allow_newline (command_subst; see Section 5.3.7)

# pass show val => hello\tworld
CMD=$(printf %s <pass:val>)      # rendering error: TAB without allow_tab (command_subst; see Section 5.3.7)

# pass show val => ping\x02
CMD=$(printf %s <pass:val>)      # rendering error: control character (command_subst; see Section 5.3.7)
```

### A.6 Backtick (`` `...` ``)
Status: Success (exit code 0)
```sh
# pass show tok => a\tb
STAMP=`echo <pass:tok|allow_tab>`
```
Note:
- In backtick context (`backtick`), `$`, `` ` ``, and `\\` MUST be escaped automatically with a preceding backslash per Section 5.3.3.

Additional examples
```sh
# pass show tok_dollar => cost$100
STAMP=`echo <pass:tok_dollar>`     # renderer emits cost\$100

# pass show tok_bt => a`b
STAMP=`echo <pass:tok_bt>`         # renderer emits a\`b

# pass show tok_bs => C:\path
STAMP=`echo <pass:tok_bs>`         # renderer emits C:\\path
```

Status: Failure (exit code 105)
```sh
# pass show tok => line1\nline2
STAMP=`echo <pass:tok>`          # rendering error: newline unsupported (backtick; see Section 5.3.8)

STAMP=`echo <pass:tok|allow_newline>`   # rendering error: allow_newline unsupported (backtick; see Section 5.3.8)

# pass show tok => ping\x03
STAMP=`echo <pass:tok>`          # rendering error: control character (backtick; see Section 5.3.8)
```

### A.7 Placeholder Syntax
Accepted spacing
```sh
KEY=<pass:path | allow_tab , allow_newline >
```

Parse errors (classified with unique subcodes)
```sh
KEY=<pass:>                            # empty PATH
KEY=<pass:path|>                       # empty modifier
KEY=<pass:path|unknown_mod>            # unknown modifier
KEY=<pass:path|allow_tab,allow_tab>    # duplicate modifier
KEY=<pass:path\n|allow_tab>            # newline inside placeholder
```

### A.8 Modifiers
Base64
Status: Success (exit code 0)
```sh
# Bare
TOKEN=<pass:secret|base64>

# Double-quoted
TOKEN_D="<pass:secret|base64>"
```
 

Invalid combinations -> render-time error (invalid modifier combination)
Status: Failure (exit code 105)
```sh
VAL=<pass:secret|base64,allow_tab>              # rendering error: invalid modifier combination (assigned unique subcode)
RAW=<pass:secret|dangerously_bypass_escape,allow_tab>   # rendering error: invalid modifier combination (assigned unique subcode)
```
 

Dangerously bypass escape
Status: Success (exit code 0)
```sh
# Caller-controlled; skips re-parse validation
RAW_CONFIG=<pass:service/config|dangerously_bypass_escape>
```
 

Strip family (normalizing whitespace)
Status: Success (exit code 0)
```sh
# Example: drop trailing newline(s) explicitly
VAL="<pass:secret|strip_right>"
```
 

### A.9 Boundary / Edge Cases
CRLF-terminated secret
```sh
VAL="<pass:secret_crlf|allow_newline>"  # pass show => line1\r\nline2\r\n
```
Unicode / Emoji / Zero-width
```sh
GREETING="<pass:greet>"  # pass show => こんにちは👋\u200b
```
Adjacent placeholders
```sh
TOKEN=<pass:a><pass:b>
```
Closing parenthesis inside command substitution
```sh
CMD=$(printf %s "<pass:val|allow_newline>")  # re-parse remains possible even if val contains ")"
```

### A.10 Realistic Combined Fixture
Status: Success (exit code 0)
```sh
# Example service configuration (mixed contexts and modifiers)
API_ENDPOINT="https://<pass:host>/v1"
AUTH_HEADER="Bearer <pass:key>"
SCRIPT=$(printf "%s" "<pass:script|allow_newline>")
MESSAGE="<pass:message|allow_tab>"
```

### A.11 Unified Diff Example
Minimal example (string segments redacted per Section 6.3; `-` denotes removal, `+` denotes addition, space indicates context):

```
--- /abs/path/to/.env
+++ /abs/path/to/.env
@@
-API_TOKEN=******
+API_TOKEN=******
 DB_HOST=localhost
```

Note:
- Redaction replaces only string segments with `*` while preserving syntactic delimiters and line boundaries.
- Expected: `envseed diff` returns exit 1 on differences (see Section 7.10, Exit Codes).

 - The following illustrates expected diff shape. For normative header and body rules, see Section 7.8.
