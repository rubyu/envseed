//go:generate go run ../../cmd/docgen-errors

package envseed

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// Exit codes defined by the specification.
// Note: Exit code 1 is reserved for "differences exist" in diff.
// All non-success, non-diff failures begin at 101.
const (
	ExitOK              = 0
	ExitInvalidInput    = 101
	ExitTemplateRead    = 102
	ExitTemplateParse   = 103
	ExitResolverFailure = 104
	ExitRenderError     = 105
	ExitOutputFailure   = 106
	ExitTargetParse     = 107
	ExitDiffFailure     = 108
	ExitInternalError   = 199
)

// ErrorDetail describes a specific failure variant within a coarse exit code.
type ErrorDetail struct {
	Exit    int
	Message string
	Detail  string
	DocSlug string
}

// ErrorDetailEntry pairs a detail code with its metadata.
type ErrorDetailEntry struct {
	Code   string
	Detail ErrorDetail
}

// Note: Presentation order is determined dynamically in ErrorDetails().

var errorRegistry = map[string]ErrorDetail{
	// 101 CLI / Input & Path Resolution (sorted by subcode)
	"EVE-101-1":   {Exit: ExitInvalidInput, Message: "no command specified", Detail: "No subcommand was provided. Specify a valid subcommand: `sync`, `diff`, `validate`, or `version`. See `envseed --help` for an overview and `envseed <command> --help` for usage.", DocSlug: "docs/errors.md#eve-101-1"},
	"EVE-101-2":   {Exit: ExitInvalidInput, Message: "unknown command %q", Detail: "An unsupported subcommand was provided. Use a valid subcommand: `sync`, `diff`, `validate`, or `version`. See `envseed --help` and `envseed <command> --help` for usage.", DocSlug: "docs/errors.md#eve-101-2"},
	"EVE-101-3":   {Exit: ExitInvalidInput, Message: "unsupported flag combination", Detail: "The provided flags conflict or are not supported together. Remove the conflicting flags. See `envseed <command> --help` for supported combinations.", DocSlug: "docs/errors.md#eve-101-3"},
	"EVE-101-4":   {Exit: ExitInvalidInput, Message: "version command does not accept flags or arguments", Detail: "Flags or arguments were provided to `version`. Run `envseed version` with no flags or arguments. See `envseed version --help` for details.", DocSlug: "docs/errors.md#eve-101-4"},
	"EVE-101-5":   {Exit: ExitInvalidInput, Message: "unknown or invalid flag %q", Detail: "An unknown or invalid flag was provided. Remove or correct the flag. See `envseed <command> --help` for supported options.", DocSlug: "docs/errors.md#eve-101-5"},
	"EVE-101-6":   {Exit: ExitInvalidInput, Message: "unexpected positional arguments", Detail: "Too many positional arguments were provided. Provide at most one optional INPUT_FILE.", DocSlug: "docs/errors.md#eve-101-6"},
	"EVE-101-101": {Exit: ExitInvalidInput, Message: "stdin is not supported", Detail: "This command intentionally does not accept stdin for templates for safety and reproducibility. Provide a readable file path instead of stdin. See `envseed <command> --help` for argument usage.", DocSlug: "docs/errors.md#eve-101-101"},
	"EVE-101-201": {Exit: ExitInvalidInput, Message: "input file %q must contain `envseed` when `--output` is omitted", Detail: "Omitting `--output` requires the template filename to contain `envseed`. Include `envseed` in the template filename or supply `--output`. See `envseed <command> --help` for usage.", DocSlug: "docs/errors.md#eve-101-201"},
	"EVE-101-301": {Exit: ExitInvalidInput, Message: "output path %q is a directory", Detail: "The output path resolves to a directory. Choose a path that resolves to a regular file. Specify the output file explicitly with `--output` when needed.", DocSlug: "docs/errors.md#eve-101-301"},

	// 102 Template Read (I/O)
	"EVE-102-1":   {Exit: ExitTemplateRead, Message: "selected input %q not found", Detail: "The selected input does not exist (ENOENT). Verify the path or create the file.", DocSlug: "docs/errors.md#eve-102-1"},
	"EVE-102-2":   {Exit: ExitTemplateRead, Message: "selected input %q is a directory", Detail: "The selected input resolves to a directory (EISDIR). Provide a readable regular file.", DocSlug: "docs/errors.md#eve-102-2"},
	"EVE-102-3":   {Exit: ExitTemplateRead, Message: "selected input %q path component is not a directory", Detail: "One or more path components are not directories (ENOTDIR). Fix the path structure.", DocSlug: "docs/errors.md#eve-102-3"},
	"EVE-102-4":   {Exit: ExitTemplateRead, Message: "selected input %q symlink loop detected", Detail: "A symbolic link loop prevents resolving the selected input (ELOOP). Fix the links.", DocSlug: "docs/errors.md#eve-102-4"},
	"EVE-102-5":   {Exit: ExitTemplateRead, Message: "selected input %q name too long", Detail: "The path or filename exceeds the length limit (ENAMETOOLONG). Shorten the path or filename.", DocSlug: "docs/errors.md#eve-102-5"},
	"EVE-102-101": {Exit: ExitTemplateRead, Message: "permission denied reading %q", Detail: "Reading the file was denied by the operating system. Check file ownership and read permissions, or run with sufficient privileges.", DocSlug: "docs/errors.md#eve-102-101"},
	"EVE-102-201": {Exit: ExitTemplateRead, Message: "open failed for selected input %q", Detail: "Opening the selected input failed due to resource exhaustion or other OS-level errors.", DocSlug: "docs/errors.md#eve-102-201"},
	"EVE-102-202": {Exit: ExitTemplateRead, Message: "I/O error reading selected input %q", Detail: "Reading the selected input failed (e.g., EIO). Try again or check the media.", DocSlug: "docs/errors.md#eve-102-202"},
	"EVE-102-203": {Exit: ExitTemplateRead, Message: "failed to read template file %q", Detail: "An unspecified I/O error occurred while accessing the selected input.", DocSlug: "docs/errors.md#eve-102-203"},

	// 103 Parsing (Parser → AST) — B0 illustrative details per Section 4.5
	"EVE-103-1":   {Exit: ExitTemplateParse, Message: "non-ASCII whitespace around placeholder separators or before `>`", Detail: "Non‑ASCII whitespace was detected around `|`, `,`, or before `>`. Use ASCII SPACE or TAB only. For example: NG: `<pass:api_key | base64>`. OK: `<pass:api_key|base64>`.", DocSlug: "docs/errors.md#eve-103-1"},
	"EVE-103-2":   {Exit: ExitTemplateParse, Message: "non-ASCII whitespace adjacent to placeholder PATH", Detail: "Non‑ASCII whitespace was detected adjacent to the placeholder path. Use ASCII SPACE or TAB only when trimming around the placeholder path. For example: NG uses U+00A0 between `:` and `api_key`: `<pass:api_key|...>`. OK: `<pass:api_key|...>`.", DocSlug: "docs/errors.md#eve-103-2"},
	"EVE-103-3":   {Exit: ExitTemplateParse, Message: "non-ASCII whitespace at start of line", Detail: "Leading whitespace contains non‑ASCII characters. Use ASCII SPACE or TAB only and avoid Unicode spaces such as U+00A0.", DocSlug: "docs/errors.md#eve-103-3"},
	"EVE-103-4":   {Exit: ExitTemplateParse, Message: "whitespace between `pass` and `:`", Detail: "Whitespace was inserted between `pass` and `:`. Do not add whitespace there. For example: NG: `<pass :path|...>`. OK: `<pass:path|...>`.", DocSlug: "docs/errors.md#eve-103-4"},
	"EVE-103-101": {Exit: ExitTemplateParse, Message: "invalid assignment name", Detail: "The assignment name is invalid. Use ASCII letters, digits, or underscore, and do not leave the name empty. For example: valid `FOO_1`. Invalid `1FOO`.", DocSlug: "docs/errors.md#eve-103-101"},
	"EVE-103-102": {Exit: ExitTemplateParse, Message: "missing '=' in assignment", Detail: "The assignment is missing `=` or `+=` between name and value. Ensure the operator is present. For example: NG: `NAME value`. OK: `NAME=value`.", DocSlug: "docs/errors.md#eve-103-102"},
	"EVE-103-103": {Exit: ExitTemplateParse, Message: "unexpected line; expected an assignment", Detail: "A non‑blank line is neither an assignment nor a comment. Each non‑blank line must be an assignment or a comment; blank lines are allowed.", DocSlug: "docs/errors.md#eve-103-103"},
	"EVE-103-201": {Exit: ExitTemplateParse, Message: "empty placeholder path", Detail: "The placeholder path is empty. Provide a non‑empty path inside `<pass:...>`. For example: NG: `<pass:|...>`. OK: `<pass:secret/path|...>`.", DocSlug: "docs/errors.md#eve-103-201"},
	"EVE-103-202": {Exit: ExitTemplateParse, Message: "unterminated placeholder", Detail: "The placeholder is unterminated. Close placeholders with `>` and ensure all modifiers are complete. For example: NG: `<pass:api_key|allow_newline`. OK: `<pass:api_key|allow_newline>`.", DocSlug: "docs/errors.md#eve-103-202"},
	"EVE-103-203": {Exit: ExitTemplateParse, Message: "placeholder path contains NUL byte", Detail: "The placeholder path contains a NUL byte. Remove NUL bytes U+0000 from the path.", DocSlug: "docs/errors.md#eve-103-203"},
	"EVE-103-205": {Exit: ExitTemplateParse, Message: "template contains NUL byte", Detail: "The template contains a NUL byte (U+0000). Remove NUL bytes from the input.", DocSlug: "docs/errors.md#eve-103-205"},
	"EVE-103-204": {Exit: ExitTemplateParse, Message: "placeholder path contains non-ASCII whitespace", Detail: "Non‑ASCII whitespace was found around the placeholder path. Use ASCII SPACE or TAB only.", DocSlug: "docs/errors.md#eve-103-204"},
	"EVE-103-301": {Exit: ExitTemplateParse, Message: "missing placeholder modifiers after '|'", Detail: "The `|` separator was present but no modifiers were provided after it. List at least one modifier after `|`. For example: `<pass:api_key|allow_newline>`.", DocSlug: "docs/errors.md#eve-103-301"},
	"EVE-103-302": {Exit: ExitTemplateParse, Message: "unknown placeholder modifier %q", Detail: "An unknown placeholder modifier was provided. Use only supported modifiers: `allow_newline`, `allow_tab`, `base64`, `strip`, `strip_left`, `strip_right`, `dangerously_bypass_escape`.", DocSlug: "docs/errors.md#eve-103-302"},
	"EVE-103-303": {Exit: ExitTemplateParse, Message: "duplicate placeholder modifier %q", Detail: "A placeholder modifier was repeated. Specify each modifier at most once. For example: NG: `<pass:api_key|strip,strip>`. OK: `<pass:api_key|strip>`.", DocSlug: "docs/errors.md#eve-103-303"},
	"EVE-103-304": {Exit: ExitTemplateParse, Message: "empty placeholder modifier", Detail: "An empty placeholder modifier was found. Remove empty entries between commas. For example: NG: `<pass:api_key|strip,,base64>`.", DocSlug: "docs/errors.md#eve-103-304"},
	"EVE-103-305": {Exit: ExitTemplateParse, Message: "invalid whitespace or NUL byte in placeholder modifiers", Detail: "Invalid whitespace or NUL bytes were found in placeholder modifiers. Use ASCII SPACE or TAB only and remove NUL bytes.", DocSlug: "docs/errors.md#eve-103-305"},
	"EVE-103-401": {Exit: ExitTemplateParse, Message: "unterminated double quote", Detail: "A double‑quoted string is unterminated. Close the string before the line ends. For example: NG: `NAME=\"value`.", DocSlug: "docs/errors.md#eve-103-401"},
	"EVE-103-402": {Exit: ExitTemplateParse, Message: "unterminated single quote", Detail: "A single‑quoted string is unterminated. Close the string before the line ends. For example: NG: `NAME='value`.", DocSlug: "docs/errors.md#eve-103-402"},
	"EVE-103-403": {Exit: ExitTemplateParse, Message: "unterminated backtick substitution", Detail: "A backtick command substitution is unterminated. Close the substitution. For example: NG: `` NAME=`cmd `.", DocSlug: "docs/errors.md#eve-103-403"},
	"EVE-103-404": {Exit: ExitTemplateParse, Message: "unterminated command substitution", Detail: "A `$()` command substitution is unterminated. Ensure the opening and closing parentheses match. For example: NG: `NAME=$(cmd`.", DocSlug: "docs/errors.md#eve-103-404"},
	"EVE-103-501": {Exit: ExitTemplateParse, Message: "mismatched brackets in assignment name", Detail: "Brackets in the assignment name are mismatched. Balance `[` and `]`. For example: NG: `ARR[0=value`.", DocSlug: "docs/errors.md#eve-103-501"},
	"EVE-103-502": {Exit: ExitTemplateParse, Message: "unexpected `]` in assignment", Detail: "An unexpected `]` was found in the assignment name. Check bracket usage. For example: NG: `ARR]0=value`.", DocSlug: "docs/errors.md#eve-103-502"},

	// 104 Resolver (pass)
	"EVE-104-1":   {Exit: ExitResolverFailure, Message: "pass command not found", Detail: "The `pass` CLI is not available. Install `pass` and ensure it is available in `PATH`.", DocSlug: "docs/errors.md#eve-104-1"},
	"EVE-104-101": {Exit: ExitResolverFailure, Message: "pass show %q failed", Detail: "The `pass` command returned an error for the requested entry. Run `pass show <PATH>` to see the underlying cause and resolve the issue such as a missing entry or a permission error.", DocSlug: "docs/errors.md#eve-104-101"},
	"EVE-104-201": {Exit: ExitResolverFailure, Message: "pass entry %q not found", Detail: "The requested `pass` entry was not found. Create the entry or correct the placeholder path. For example: `pass insert <PATH>`.", DocSlug: "docs/errors.md#eve-104-201"},
	"EVE-104-301": {Exit: ExitResolverFailure, Message: "pass entry %q contains NUL byte", Detail: "The `pass` entry value contains a NUL byte. Remove NUL characters U+0000 from the value.", DocSlug: "docs/errors.md#eve-104-301"},

	// 105 Rendering + Re-parse Validation
	"EVE-105-1": {Exit: ExitRenderError, Message: "rendering failed due to placeholder constraints", Detail: "The secret cannot be represented in the chosen placeholder context without violating constraints. Adjust quoting or add the required modifiers such as `allow_newline` or `allow_tab`, or choose a different quoting context.", DocSlug: "docs/errors.md#eve-105-1"},
	// B1: Single-quoted context
	"EVE-105-101": {Exit: ExitRenderError, Message: "single-quoted placeholder cannot contain a single quote (`'`)", Detail: "Single quotes cannot contain `'`. Switch to double quotes; escaping is applied automatically in double‑quoted context.", DocSlug: "docs/errors.md#eve-105-101"},
	"EVE-105-102": {Exit: ExitRenderError, Message: "newline not permitted in single-quoted placeholder", Detail: "Newlines are not permitted in single‑quoted placeholders. Switch to double quotes and add the `allow_newline` modifier.", DocSlug: "docs/errors.md#eve-105-102"},
	"EVE-105-103": {Exit: ExitRenderError, Message: "TAB not permitted in single-quoted placeholder", Detail: "TAB is not permitted in single‑quoted placeholders. Add the `allow_tab` modifier or switch to double quotes with `allow_tab`.", DocSlug: "docs/errors.md#eve-105-103"},
	"EVE-105-104": {Exit: ExitRenderError, Message: "control character U+%04X not permitted in single-quoted placeholder", Detail: "Control characters are not supported in single‑quoted placeholders. Switch to double quotes or adjust the value.", DocSlug: "docs/errors.md#eve-105-104"},
	"EVE-105-105": {Exit: ExitRenderError, Message: "allow_newline modifier is not supported in single-quoted context", Detail: "The `allow_newline` modifier is not supported in single‑quoted context. Switch to double quotes and add the `allow_newline` modifier.", DocSlug: "docs/errors.md#eve-105-105"},
	// B2: Double-quoted context
	"EVE-105-201": {Exit: ExitRenderError, Message: "newline not permitted in double-quoted placeholder", Detail: "Newlines are not permitted in double‑quoted placeholders by default. Add the `allow_newline` modifier.", DocSlug: "docs/errors.md#eve-105-201"},
	"EVE-105-202": {Exit: ExitRenderError, Message: "TAB not permitted in double-quoted placeholder", Detail: "TAB is not permitted in double‑quoted placeholders by default. Add the `allow_tab` modifier.", DocSlug: "docs/errors.md#eve-105-202"},
	"EVE-105-203": {Exit: ExitRenderError, Message: "control character U+%04X not permitted in double-quoted placeholder", Detail: "Control characters are not supported in double‑quoted placeholders. Adjust the value or encoding.", DocSlug: "docs/errors.md#eve-105-203"},
	// B3: Command substitution
	"EVE-105-301": {Exit: ExitRenderError, Message: "newline not permitted in command substitution placeholder", Detail: "Newlines are not permitted in command substitution placeholders by default. Add the `allow_newline` modifier.", DocSlug: "docs/errors.md#eve-105-301"},
	"EVE-105-302": {Exit: ExitRenderError, Message: "TAB not permitted in command substitution placeholder", Detail: "TAB is not permitted in command substitution placeholders by default. Add the `allow_tab` modifier.", DocSlug: "docs/errors.md#eve-105-302"},
	"EVE-105-303": {Exit: ExitRenderError, Message: "control character U+%04X not permitted in command substitution placeholder", Detail: "Control characters are not supported in command substitution placeholders. Adjust the value.", DocSlug: "docs/errors.md#eve-105-303"},
	// B4: Backtick context
	"EVE-105-401": {Exit: ExitRenderError, Message: "newline not permitted in backtick placeholder", Detail: "Newlines are not permitted in backtick placeholders. Replace backticks with `$()` and add the `allow_newline` modifier if needed.", DocSlug: "docs/errors.md#eve-105-401"},
	"EVE-105-402": {Exit: ExitRenderError, Message: "TAB not permitted in backtick placeholder", Detail: "TAB is not permitted in backtick placeholders. Add the `allow_tab` modifier or switch to a different quoting context.", DocSlug: "docs/errors.md#eve-105-402"},
	"EVE-105-403": {Exit: ExitRenderError, Message: "control character U+%04X not permitted in backtick placeholder", Detail: "Control characters are not supported in backtick placeholders. Adjust the value.", DocSlug: "docs/errors.md#eve-105-403"},
	"EVE-105-404": {Exit: ExitRenderError, Message: "allow_newline modifier is not supported in backtick context", Detail: "The `allow_newline` modifier is not supported in backtick context. Replace backticks with `$()` and use `allow_newline`.", DocSlug: "docs/errors.md#eve-105-404"},
	// B5: Bare context
	"EVE-105-501": {Exit: ExitRenderError, Message: "newline not permitted in bare placeholder", Detail: "Newlines are not permitted in bare placeholders. Switch to double quotes and add the `allow_newline` modifier.", DocSlug: "docs/errors.md#eve-105-501"},
	"EVE-105-502": {Exit: ExitRenderError, Message: "TAB not permitted in bare placeholder", Detail: "TAB is not permitted in bare placeholders. Switch to double quotes or add the `allow_tab` modifier.", DocSlug: "docs/errors.md#eve-105-502"},
	"EVE-105-503": {Exit: ExitRenderError, Message: "control character U+%04X not permitted in bare placeholder", Detail: "Control characters are not supported in bare placeholders. Quote or encode the value to avoid emitting unsupported control characters.", DocSlug: "docs/errors.md#eve-105-503"},
	"EVE-105-504": {Exit: ExitRenderError, Message: "allow_newline modifier is not supported in bare context", Detail: "The `allow_newline` modifier is not supported in bare context. Switch to double quotes and add the `allow_newline` modifier.", DocSlug: "docs/errors.md#eve-105-504"},
	// B6: Invalid modifier combination
	"EVE-105-601": {Exit: ExitRenderError, Message: "invalid placeholder modifier combination", Detail: "The `base64` modifier cannot be combined with any other modifier. Remove the other modifiers, including the strip family and `dangerously_bypass_escape`.", DocSlug: "docs/errors.md#eve-105-601"},
	// B7: Post-render re-parse validation failure
	"EVE-105-701": {Exit: ExitRenderError, Message: "rendered output is syntactically invalid", Detail: "The rendered output is syntactically invalid. Ensure the rendered assignments are syntactically valid and review any use of `dangerously_bypass_escape`.", DocSlug: "docs/errors.md#eve-105-701"},

	// 106 Output (sync write: I/O)
	"EVE-106-1":   {Exit: ExitOutputFailure, Message: "output directory %q does not exist", Detail: "The output directory does not exist. Create the directory before running `envseed`.", DocSlug: "docs/errors.md#eve-106-1"},
	"EVE-106-2":   {Exit: ExitOutputFailure, Message: "failed to access output directory %q", Detail: "The output directory could not be accessed. Check directory permissions and ensure `envseed` can access the target directory.", DocSlug: "docs/errors.md#eve-106-2"},
	"EVE-106-3":   {Exit: ExitOutputFailure, Message: "output path parent %q is not a directory", Detail: "The parent of the output path is not a directory. Select an output path whose parent is a directory.", DocSlug: "docs/errors.md#eve-106-3"},
	"EVE-106-4":   {Exit: ExitOutputFailure, Message: "failed to stat output file %q", Detail: "The output path could not be inspected. Investigate filesystem issues that prevent `envseed` from statting the path.", DocSlug: "docs/errors.md#eve-106-4"},
	"EVE-106-101": {Exit: ExitOutputFailure, Message: "output file %q already exists", Detail: "The output file already exists. Use `--force` when you intend to replace the existing file.", DocSlug: "docs/errors.md#eve-106-101"},
	"EVE-106-102": {Exit: ExitOutputFailure, Message: "failed to read output file %q", Detail: "Reading the existing output file failed. Resolve permission or locking problems before reading or writing.", DocSlug: "docs/errors.md#eve-106-102"},
	"EVE-106-103": {Exit: ExitOutputFailure, Message: "failed to set file mode on output file %q", Detail: "Setting the file mode on the output file failed. Ensure `envseed` has permission to change the mode to `0600`.", DocSlug: "docs/errors.md#eve-106-103"},
	"EVE-106-201": {Exit: ExitOutputFailure, Message: "failed to create temporary output file in %q", Detail: "Creating a temporary output file failed. Check directory permissions and available disk space.", DocSlug: "docs/errors.md#eve-106-201"},
	"EVE-106-202": {Exit: ExitOutputFailure, Message: "failed to set file mode on temporary output file %q", Detail: "Setting the file mode on the temporary output file failed. Ensure the filesystem permits mode `0600` for temporary files.", DocSlug: "docs/errors.md#eve-106-202"},
	"EVE-106-203": {Exit: ExitOutputFailure, Message: "failed to write temporary output file %q", Detail: "Writing the temporary output file failed. Resolve disk or permission issues that prevent writing the rendered content.", DocSlug: "docs/errors.md#eve-106-203"},
	"EVE-106-204": {Exit: ExitOutputFailure, Message: "failed to close temporary output file %q", Detail: "Closing the temporary output file failed. Investigate filesystem issues causing failures on file close.", DocSlug: "docs/errors.md#eve-106-204"},
	"EVE-106-301": {Exit: ExitOutputFailure, Message: "failed to replace %q with %q atomically", Detail: "Atomic replacement failed during rename. Fix rename failures, which are often due to cross‑filesystem moves or permissions.", DocSlug: "docs/errors.md#eve-106-301"},
	"EVE-106-302": {Exit: ExitOutputFailure, Message: "failed to set permissions on %q", Detail: "Setting permissions on the output file failed. Ensure `envseed` can `chmod` the file to `0600`.", DocSlug: "docs/errors.md#eve-106-302"},
	"EVE-106-401": {Exit: ExitOutputFailure, Message: "failed to write dry-run output", Detail: "Writing dry‑run output to stdout failed. Resolve stdout write failures when running with `--dry-run`.", DocSlug: "docs/errors.md#eve-106-401"},

	// 107 Target .env parsing (A/B)
	"EVE-107-1":   {Exit: ExitTargetParse, Message: "unsupported line in target .env", Detail: "An unsupported line was found in the target `.env`. Only assignments, comments, and blank lines are allowed.", DocSlug: "docs/errors.md#eve-107-1"},
	"EVE-107-101": {Exit: ExitTargetParse, Message: "target .env contains non-ASCII whitespace at the beginning of the line before the assignment", Detail: "Non‑ASCII whitespace was found at the beginning of an assignment line (indentation). Use ASCII SPACE or TAB only in that position. For example: NG uses U+00A0 before `APP_PORT=8080`: `\\u00A0APP_PORT=8080`. OK: ` APP_PORT=8080` or `\\tAPP_PORT=8080`.", DocSlug: "docs/errors.md#eve-107-101"},
	"EVE-107-102": {Exit: ExitTargetParse, Message: "target .env contains NUL byte", Detail: "A NUL byte was found in the target `.env`. Remove NUL bytes U+0000 from the file.", DocSlug: "docs/errors.md#eve-107-102"},
	"EVE-107-201": {Exit: ExitTargetParse, Message: "unterminated double quote in target .env", Detail: "A double‑quoted string is unterminated in the target `.env`. Close the string before the line ends. For example: NG: `NAME=\"value`.", DocSlug: "docs/errors.md#eve-107-201"},
	"EVE-107-202": {Exit: ExitTargetParse, Message: "unterminated single quote in target .env", Detail: "A single‑quoted string is unterminated in the target `.env`. Close the string before the line ends. For example: NG: `NAME='value`.", DocSlug: "docs/errors.md#eve-107-202"},
	"EVE-107-203": {Exit: ExitTargetParse, Message: "unterminated backtick substitution in target .env", Detail: "A backtick command substitution is unterminated in the target `.env`. Close the substitution. For example: NG: `` NAME=`cmd `.", DocSlug: "docs/errors.md#eve-107-203"},
	"EVE-107-204": {Exit: ExitTargetParse, Message: "unterminated command substitution in target .env", Detail: "A `$()` command substitution is unterminated in the target `.env`. Ensure the opening and closing parentheses match. For example: NG: `NAME=$(cmd`.", DocSlug: "docs/errors.md#eve-107-204"},
	"EVE-107-205": {Exit: ExitTargetParse, Message: "invalid syntax in target .env", Detail: "The target `.env` contains invalid syntax. Ensure it follows the same grammar as the template, allowing assignments, comments, and blank lines only.", DocSlug: "docs/errors.md#eve-107-205"},
	"EVE-107-301": {Exit: ExitTargetParse, Message: "placeholders are not allowed in target .env", Detail: "Placeholders are not allowed in the target `.env`. Remove constructs such as `<pass:...>`.", DocSlug: "docs/errors.md#eve-107-301"},

	// 108 Diff (comparison) — densified in B0
	"EVE-108-1": {Exit: ExitDiffFailure, Message: "diff target %q exceeds 10 MiB size limit", Detail: "The target file exceeds the 10 MiB diff size limit. Reduce the file size or split the environment file before running `envseed diff`.", DocSlug: "docs/errors.md#eve-108-1"},
	"EVE-108-2": {Exit: ExitDiffFailure, Message: "failed to build diff for %q", Detail: "Building the diff failed. Inspect filesystem permissions and retry the diff operation.", DocSlug: "docs/errors.md#eve-108-2"},
	"EVE-108-3": {Exit: ExitDiffFailure, Message: "failed to write diff for %q", Detail: "Writing the diff failed. Ensure stdout accepts diff output and rerun `envseed diff`.", DocSlug: "docs/errors.md#eve-108-3"},

	// 199 Internal exceptions
	"EVE-199-1": {Exit: ExitInternalError, Message: "redaction failed (internal error)", Detail: "Redaction rendering failed due to an internal error. Please report this bug and include reproducible steps.", DocSlug: "docs/errors.md#eve-199-1"},
	"EVE-199-2": {Exit: ExitInternalError, Message: "resolver used after close", Detail: "The resolver was used after it was closed. Please report this bug and include reproducible steps.", DocSlug: "docs/errors.md#eve-199-2"},
}

// (no static order index)

// ExitError represents an error that carries a specification-defined exit code.
type ExitError struct {
	Code       int
	Err        error
	Msg        string
	DetailCode string
	DetailText string
	DocSlug    string
}

// Error implements the error interface.
func (e *ExitError) Error() string {
	prefix := "envseed ERROR"
	if e.DetailCode != "" {
		prefix = fmt.Sprintf("%s [%s]", prefix, e.DetailCode)
	}

	body := e.Msg
	if body == "" {
		body = fmt.Sprintf("unexpected error (code %d)", e.Code)
	}
	if e.Err != nil {
		body = fmt.Sprintf("%s: %v", body, e.Err)
	}

	out := fmt.Sprintf("%s: %s", prefix, body)
	if e.DetailText != "" {
		out += "\nDetail: " + e.DetailText
	}
	if e.DocSlug != "" {
		out += "\nReference: " + e.DocSlug
	}
	return out
}

// Unwrap allows errors.Is / errors.As to observe the underlying error.
func (e *ExitError) Unwrap() error {
	return e.Err
}

// WithErr returns a shallow copy of the error that wraps an additional error.
func (e *ExitError) WithErr(err error) *ExitError {
	if e == nil {
		return nil
	}
	clone := *e
	clone.Err = err
	return &clone
}

// NewExitError constructs an ExitError using a registered detail code.
func NewExitError(detailCode string, args ...any) *ExitError {
	detail, ok := errorRegistry[detailCode]
	if !ok {
		panic(fmt.Sprintf("envseed: unknown error detail code %q", detailCode))
	}

	msg := detail.Message
	if len(args) > 0 {
		msg = fmt.Sprintf(detail.Message, args...)
	}

	return &ExitError{
		Code:       detail.Exit,
		Msg:        msg,
		DetailCode: detailCode,
		DetailText: detail.Detail,
		DocSlug:    detail.DocSlug,
	}
}

// LookupErrorDetail returns metadata for a detail code.
func LookupErrorDetail(code string) (ErrorDetail, bool) {
	detail, ok := errorRegistry[code]
	return detail, ok
}

// ErrorDetails returns all registered error details in a stable order.
func ErrorDetails() []ErrorDetailEntry {
	// Collect and sort codes by numeric Exit then numeric Subcode ascending.
	keys := make([]string, 0, len(errorRegistry))
	for code := range errorRegistry {
		keys = append(keys, code)
	}
	parse := func(code string) (exit, sub int) {
		// code format: EVE-<exit>-<sub>
		parts := strings.Split(code, "-")
		if len(parts) != 3 {
			return 0, 0
		}
		exit, _ = strconv.Atoi(parts[1])
		sub, _ = strconv.Atoi(parts[2])
		return exit, sub
	}
	sort.Slice(keys, func(i, j int) bool {
		ei, si := parse(keys[i])
		ej, sj := parse(keys[j])
		if ei != ej {
			return ei < ej
		}
		if si != sj {
			return si < sj
		}
		return keys[i] < keys[j]
	})

	entries := make([]ErrorDetailEntry, 0, len(keys))
	for _, code := range keys {
		entries = append(entries, ErrorDetailEntry{
			Code:   code,
			Detail: errorRegistry[code],
		})
	}
	return entries
}
