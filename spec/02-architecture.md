## 2. Architecture Overview (Informative)
EnvSeed consists of:
- Parser: reads `.envseed*` templates into an AST (a sequence of Elements) while preserving order, whitespace, and comments.
- Renderer: walks the AST, writes literal tokens verbatim, resolves placeholders via a Resolver, and applies context-aware escaping.
- Resolver: retrieves secrets using `pass show <PATH>` with in-process single-resolution caching (see Section 6.2).
- CLI: exposes `sync` (write), `diff` (compare), `validate` (parse-only), and `version` (print version string).

Data flow: `template -> parser -> AST -> renderer(resolver) -> output|compare|validate`.
