# EnvSeed
```
 _____             ____                _
| ____|_ ____   __/ ___|  ___  ___  __| |
|  _| | '_ \ \ / /\___ \ / _ \/ _ \/ _` |
| |___| | | \ V /  ___) |  __/  __/ (_| |
|_____|_| |_|\_/  |____/ \___|\___|\__,_|
🌱   EnvSeed - grow your .env from seeds
```

> “They call it `.env`—a cursed thing.  
> Everyone owns one, everyone depends on it, every moment of their lives.  
> Yet they toss it aside, never thinking what would happen…  
> until it’s gone—and the nightmare begins.”

---
EnvSeed is a small command-line utility that reconstructs `.env` files from `.envseed*` templates. It helps you generate `.env` files safely and reproducibly, while keeping the seed templates securely stored. Placeholders in the templates are resolved via the `pass` command, and context-aware escaping ensures secrets are written safely.

## Demo

```sh
cat > .envseed <<EOF
PASSWORD='<pass:www.example.com/password>'
EOF

envseed sync .envseed
cat .env
```

```sh
PASSWORD='vP9%cQ$m*Nqk'
```

## Why EnvSeed
- `.env` files should never be committed or shared, yet teams still need a reliable way to restore them.  
- Manually copying secrets into `.env` is error-prone and, frankly, a waste of time.  
- EnvSeed securely handles secrets while providing a safe and elegant way to manage escaping, quoting, and newlines or tabs, with context- and syntax-aware processing.

## Features
- Create `.env` files from template files.  
- Retrieve secrets stored in `pass`.
- Safely preview changes with a dry run.
- Perform syntax validation using a compact Bash-subset parser.  
- Show a diff and confirm changes before applying them to `.env` files.

## How It Works
1. Create a template file (e.g., `.envseed`).  
2. Write its contents with placeholders such as `<pass:service/api-token>`.  
3. Run the command `envseed sync .envseed`.
4. Enter the password for your GPG key when prompted.
5. EnvSeed then:
   - parses the template,  
   - retrieves secrets using `pass show <PATH>`,  
   - applies context-aware escaping, and  
   - writes the resulting `.env` file.

## Requirements
- Dependency: `pass` (Password Store) — https://www.passwordstore.org/
- Supported OS: Linux or macOS

## Installation
There are several ways to install EnvSeed.

1. Releases via command (recommended)

   Fetch the latest release for your OS/ARCH and install into `~/.local/bin` (or `/usr/local/bin`).

   ```bash
   curl -fsSL https://raw.githubusercontent.com/rubyu/envseed/main/scripts/install.sh | bash
   ```

   Pin a version and choose a destination (example: install v0.1.0 to /usr/local/bin):

   ```bash
   curl -fsSL https://raw.githubusercontent.com/rubyu/envseed/main/scripts/install.sh \
     | bash -s -- -v v0.1.0 -b /usr/local/bin
   ```

2. Go install

   ```bash
   go install github.com/rubyu/envseed/cmd/envseed@latest
   ```

3. Build from source (Make)

   ```bash
   make build
   ```

   The binary is written to `dist/envseed`.

## Quick Start
#### Preview (dry-run)
Shows a masked preview on stdout without writing any files. Safe for a first run.
```bash
envseed sync --dry-run .envseed
```

#### Write to an implicit file
Replaces the first "envseed" in the input filename with "env" and writes the resulting `.env*` next to the template. Example: `.envseed.testing` → `.env.testing`.
```bash
envseed sync .envseed
```

#### Write to an explicit file
Writes to the exact file you specify. Use `--force` if the file already exists.
```bash
envseed sync -o ./config/.env .envseed
```

#### Write to a directory
When given a directory, writes under it using a derived filename (`envseed` → `env`). Example: `./build/.env.testing`.
```bash
envseed sync -o ./build .envseed.testing
```

#### Diff against an explicit target
Renders in memory and prints a masked unified diff.
```bash
envseed diff -o ./config/.env .envseed
```

#### Validate (parse only)
Parses the template and reports syntax/lexing errors. Does not contact `pass`.
```bash
envseed validate .envseed
```

Notes
- When `--output` is omitted, the template filename must contain `envseed` (validate is exempt).
- In non–dry-run, rendered content is not printed to stdout. Writes are atomic and final permissions are `0600`.
- Diff and dry-run display masked content only.

## CI Status
| Branch  | Status |
|--------|--------|
| main   | [![CI - main](https://github.com/rubyu/envseed/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/rubyu/envseed/actions/workflows/ci.yml?query=branch%3Amain) |
| develop| [![CI - develop](https://github.com/rubyu/envseed/actions/workflows/ci.yml/badge.svg?branch=develop)](https://github.com/rubyu/envseed/actions/workflows/ci.yml?query=branch%3Adevelop) |

## CLI Reference
For the full CLI reference (commands, flags, path resolution, exit codes, template language, and security), see:

- docs/cli.md

## Development
- Build: `make build`
- Docs (generate error docs): `make docs`
- Run tests: `make test`
- Run fuzz Tests: `make test-fuzz`
- Lint/vet and formatting check: `make check`
- Pre‑commit tasks: `make pre-commit`

For conformance guidance, parser/renderer details, and the full error taxonomy, see the developer specification:
- spec/README.md

For a clear, reproducible approach to property‑based tests and fuzzing (deterministic baselines, corpus replay, and optional exploratory fuzzing), see:
- docs/testing/fuzz.md

## License
MIT. See `LICENSE` for details.
