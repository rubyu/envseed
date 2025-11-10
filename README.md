# EnvSeed
```
 _____             ____                _
| ____|_ ____   __/ ___|  ___  ___  __| |
|  _| | '_ \ \ / /\___ \ / _ \/ _ \/ _` |
| |___| | | \ V /  ___) |  __/  __/ (_| |
|_____|_| |_|\_/  |____/ \___|\___|\__,_|
🌱   EnvSeed - grow your .env from seeds
```

EnvSeed is a tiny CLI that rebuilds your `.env` from safe templates. 
Templates live in Git; secrets stay in `pass`.

> “Never babysit your .env again.”

## Demo

```sh
cat > .envseed <<EOF
PASSWORD='<pass:www.example.com/password>'
EOF

envseed sync
cat .env
```

```sh
PASSWORD='vP9%cQ$m*Nqk'
```

## Features
- Create `.env` files from template files.  
- Retrieve secrets stored in `pass`.
- Safely preview changes with a dry run.
- Perform syntax validation using a compact Bash-subset parser.  
- Show a diff and confirm changes before applying them to `.env` files.

## Safety
- 🔐 Secrets live in `pass` (GPG) — not in Git.
- 🙈 Masked output for dry‑run/diff.
- 🧪 Parser validation before touching `pass`.
- ✍️ Atomic writes + `0600` perms.
- 🧯 Non–dry‑run does not echo rendered secrets.

## Why EnvSeed
- `.env` files should never be committed or shared, yet teams still need a reliable way to restore them.  
- Manually copying secrets into `.env` is error-prone and, frankly, a waste of time.  
- EnvSeed securely handles secrets while providing a safe and elegant way to manage escaping, quoting, and newlines or tabs, with context- and syntax-aware processing.

## How It Works
1. Parse `.envseed` with a compact Bash‑subset parser (syntax validated).
2. For each `<pass:…>` placeholder, run `pass show <PATH>`.
3. Apply context‑aware escaping (quotes/newlines/tabs).
4. Atomically write target `.env*` with `0600` permissions.
5. Optionally show a masked unified diff before applying.

Result: predictable, reproducible `.env` without leaking secrets. 

## Requirements
- Dependency: `pass` (Password Store, backed by GPG) — https://www.passwordstore.org/
- Supported OS: Linux or macOS

## Installation

```bash
curl -fsSL https://raw.githubusercontent.com/rubyu/envseed/main/scripts/install.sh | bash
```

## Quick Start
#### Sync
```
╔════════════════════════════════════════════════════╗
║ envseed  sync  [flags]  [INPUT_FILE]               ║
║          ────                                      ║
╚════════════════════════════════════════════════════╝
```

Render the template and write the output file.

```bash
envseed sync
```

Note: You can omit `[INPUT_FILE]`. If omitted, envseed uses `.envseed` in the current directory as the input file.

#### Sync (override the input file)
```bash
envseed sync .envseed.testing
```

#### Sync (safe preview with --dry-run)
```bash
envseed sync --dry-run
```

#### Diff
```
╔════════════════════════════════════════════════════╗
║ envseed  diff  [flags]  [INPUT_FILE]               ║
║          ────                                      ║
╚════════════════════════════════════════════════════╝
```

Render in memory and print a redacted unified diff.

```bash
envseed diff
```

#### Validate
```
╔════════════════════════════════════════════════════╗
║ envseed  validate  [flags]  [INPUT_FILE]           ║
║          ────────                                  ║
╚════════════════════════════════════════════════════╝
```

Parse the template and report errors.

```bash
envseed validate
```

---

## Anothor Installation Methods
### Pin a version / custom bin dir:

```bash
curl -fsSL https://raw.githubusercontent.com/rubyu/envseed/main/scripts/install.sh \
  | bash -s -- -v v0.1.0 -b /usr/local/bin
```

### Go users:

```bash
go install github.com/rubyu/envseed/cmd/envseed@latest
```

### Build from source (optional):

```bash
make build
```

The binary is written to `dist/envseed`.

## CI Status
| Branch  | Status |
|--------|--------|
| main   | [![CI - main](https://github.com/rubyu/envseed/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/rubyu/envseed/actions/workflows/ci.yml?query=branch%3Amain) |
| develop| [![CI - develop](https://github.com/rubyu/envseed/actions/workflows/ci.yml/badge.svg?branch=develop)](https://github.com/rubyu/envseed/actions/workflows/ci.yml?query=branch%3Adevelop) |

## CLI Reference
For the full CLI reference (commands, flags, default input behavior, path resolution, exit codes, template language, and security), see:

- [CLI Reference](docs/cli.md)

## Development
- Build: `make build`
- Docs (generate error docs): `make docs`
- Run tests: `make test`
- Run fuzz Tests: `make test-fuzz`
- Lint/vet and formatting check: `make check`
- Pre‑commit tasks: `make pre-commit`

For conformance guidance, parser/renderer details, and the full error taxonomy, see the developer specification:
- [Developer Specification](spec/README.md)

For contribution guidelines, development workflow, and the project's testing strategy, see:
- [Contributing Guide](CONTRIBUTING.md)

## License
MIT. See [LICENSE](LICENSE) for details.
