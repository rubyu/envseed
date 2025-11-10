## Appendix D. ABNF Grammar

This appendix centralizes the grammar used throughout the specification. ABNF follows RFC 5234. It is ASCII-oriented; Unicode acceptance is defined in the notes.

### D.1 Core Tokens & Whitespace
```
SP       = %x20
HTAB     = %x09
CR       = %x0D
LF       = %x0A
CRLF     = CR LF
EOL      = CRLF / LF / CR

; Grammar-level whitespace is Space (U+0020) and Tab (U+0009) only.
WSP      = *( SP / HTAB )

; Octet classes (ASCII-oriented ABNF). Non-ASCII is permitted unless restricted
; by context; see notes.
NON-EOL  = %x01-09 / %x0B-0C / %x0E-7F
```
Notes:
- "Whitespace" in this specification means Space (U+0020) and Tab (U+0009) only. Any other Unicode whitespace where whitespace is expected is a parse error (exit code 103 — Template parsing failure; subcode assignment follows `docs/errors.md`; see Section 4.5).
- Non-ASCII Unicode is allowed in values/PATH except where restricted (NUL/line terminators/separators). ABNF is ASCII-oriented; Unicode acceptance is defined by these notes.
- NUL (U+0000) is prohibited. Template inputs containing NUL MUST be rejected as parse errors (exit code 103 — Template parsing failure). Resolver-provided values containing NUL are runtime errors (exit code 104 — Resolver failures) per Section 6.2.
- ALPHA, DIGIT, and related core terminals are those defined by RFC 5234 (ABNF core rules).

### D.2 File & Elements
```
file        = *( element )
element     = assignment / comment / blank
blank       = WSP EOL
comment     = WSP "#" *( NON-EOL ) [ EOL ]
assignment  = WSP name operator value [ *WSP trailing_comment ] [ EOL ]
trailing_comment = "#" *( NON-EOL )
```
Trailing comment: A top-level `#` begins a trailing comment only if the number of immediately preceding backslashes is even (including zero). If odd, `#` is literal and MUST NOT start a comment.

### D.3 Name / Operator / Index
```
name        = name-head *( name-tail ) [ index ]
name-head   = "_" / ALPHA
name-tail   = "_" / ALPHA / DIGIT
operator    = ( "+=" / "=" )

index       = *( "[" index-body "]" )
index-body  = *( index-char / index )
index-char  = %x01-5A / %x5C / %x5E-7F ; ']' and newline excluded
```
Notes: INDEX content is not evaluated; only bracket balance is verified (unclosed/extra `]` is a parse error).

### D.4 Value & Tokenization
```
value       = *( token )
token       = bare-seg / dq / sq / cmdsub / bt / placeholder

bare-seg    = 1*( bare-char / escape )
bare-char   = %x01-09 / %x0B-0C / %x0E-1F / %x21-23 / %x25-26 /
              %x2A-2B / %x2D-2E / %x30-39 / %x3A-3B / %x3D /
              %x3F-5B / %x5D-60 / %x7B-7E
escape      = "\\" %x01-7F

dq          = %x22 *( dq-char / escape / placeholder ) %x22
dq-char     = %x01-21 / %x23-5B / %x5D-7F

sq          = %x27 *( sq-char / placeholder ) %x27
sq-char     = %x01-26 / %x28-7F

cmdsub      = %x24 "(" cmd-body ")"
cmd-body    = *( cmd-char / escape / dq / sq / bt / cmdsub / placeholder )
cmd-char    = %x01-27 / %x2A-7F ; ')' excluded

bt          = %x60 *( bt-char / escape / placeholder ) %x60
bt-char     = %x01-23 / %x25-5B / %x5D-7F ; '`', '\\', '$' escaped
```
Note (Informative): This ABNF captures structural boundaries only. Emission/escaping/allowances are governed by Section 5.3.

### D.5 Placeholder
```
placeholder = "<pass:" path [ *WSP "|" *WSP modifiers ] *WSP ">"
path        = 1*( path-char )
modifiers   = modifier *( *WSP "," *WSP modifier )
modifier    = "allow_newline"
            / "allow_tab"
            / "base64"
            / "dangerously_bypass_escape"
            / "strip"
            / "strip_left"
            / "strip_right"
; path-char excludes NUL, CR, LF, '|' and '>' (separators)
; ASCII non-separators (exclude '>' %x3E and '|' %x7C)
ASCII-NONSEP = %x01-09 / %x0B-0C / %x0E-3D / %x3F-7B / %x7D-7F
; UTF-8 multi-byte sequences (RFC 3629) excluding surrogates and out-of-range values
UTF8-2     = %xC2-DF %x80-BF
UTF8-3     = %xE0 %xA0-BF %x80-BF
           / %xE1-EC %x80-BF %x80-BF
           / %xED %x80-9F %x80-BF
           / %xEE-EF %x80-BF %x80-BF
UTF8-4     = %xF0 %x90-BF %x80-BF %x80-BF
           / %xF1-F3 %x80-BF %x80-BF %x80-BF
           / %xF4 %x80-8F %x80-BF %x80-BF
path-char  = ASCII-NONSEP / UTF8-2 / UTF8-3 / UTF8-4
```
Notes:
- PATH MAY contain non-ASCII Unicode (UTF-8). Accept any code point except NUL/line terminators; separators `|`, `>` are forbidden within PATH. Trimming/around-separators whitespace is Space (U+0020) and Tab (U+0009) only.
- Sigil strictness: `<pass` MUST be followed immediately by `:` with no whitespace; violations are parse errors with source position (see Section 4.5).
- The ABNF above admits UTF-8 code points in PATH (excluding NUL/line terminators and the separators `|`, `>`). Implementations MUST reject any Unicode whitespace other than Space (U+0020) and Tab (U+0009) where trimming or around-separator whitespace is expected (see Sections 4.3 and 4.5).
