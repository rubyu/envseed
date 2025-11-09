## 3. Data Model
Input templates are parsed into an ordered sequence of Elements:
- Assignment: leading whitespace; name; operator (`=` or `+=`); an ordered list of value tokens; an optional trailing comment; a trailing-newline flag.
  - Example: `DB_USER="alice"`
- Comment: a whole-line comment whose first non-whitespace character is `#` (leading whitespace is allowed and preserved); trailing-newline flag.
  - Example: `# Deploy credentials`; `  # Indented`
- Blank: an empty line (possibly containing only whitespace).

Trailing comments that appear after an assignment's value on the same line are part of the Assignment element (not a separate Comment element). See Section 4.1 and Appendix D.2.

Value tokens record:
- Kind: `Literal` or `Placeholder`.
- Context: one of `bare`, `double_quoted`, `single_quoted`, `command_subst`, or `backtick`.
- Text: verbatim literal text (for `Literal`) or raw placeholder text (for `Placeholder`). For placeholders, the raw text MUST include the surrounding angle brackets ("<" and ">") exactly as it appears in the template.
- Path and Modifiers: parsed from `<pass:PATH|modifier[, modifier...]>`.
- Source position: line and column for diagnostics.
