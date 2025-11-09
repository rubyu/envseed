# EnvSeed Contributing Guide

## Scope
- This document explains how to build, test, and contribute changes with consistent naming, test visibility, EVT tagging, and corpus management.

## Spec-first Workflow

- Change control and tests are driven by the specification under `spec/`.
- When proposing or implementing a change:
  1) Update the relevant spec documents first (e.g., `spec/07-cli.md`, `spec/05-rendering.md`, `spec/08-testing.md`).
  2) Mint new test items in `spec/C-test-coverage.md` (Section C.2) and allocate new EVT identifiers; do not reuse existing identifiers.
  3) Implement the product code change.
  4) Add tests and include the EVT identifier(s) in an adjacent comment above each `Test*` or `Fuzz*` function.
  5) For new error codes, follow `spec/07-cli.md` and `spec/B-error-map.md`, register the detail(s) in `internal/envseed/errors.go`, and run `go generate ./...` to refresh `docs/errors.md`.

## Prerequisites
- Go toolchain on PATH.
- Optional tools used by some test targets:
  - bash (validation in tests)
  - bubblewrap + ldd (Linux-only, for sandbox-tagged tests)
  - pass + gpg (for integration-tagged tests)

## Make Targets
- Build: `make build`
- All tests: `make test`
- Integration tests: `make test-integration` (uses `-tags=integration`; may skip when tools are missing)
- Sandbox tests: `make test-sandbox` (uses `-tags=sandbox`; Linux + bubblewrap only; must skip cleanly when unavailable)
- Fuzz (on-demand, not required in CI): `make test-fuzz`
- Static checks: `make check` (format and vet; internally calls the EVT check)
- EVT-only check: `make check-evt`
- Pre-commit (comprehensive): `make pre-commit`
  - Runs: docs → check → check-evt → test → test-sandbox → test-integration

## Generated docs
- Regenerate `docs/errors.md` via `go generate ./...`. This file is authoritative for numbers/messages/guidance.

## Test Layout & Visibility
- Default visibility: prefer external tests (`package <mod>_test`).
- Internal access only when strictly necessary. Use `export_test.go` to expose a minimal surface for tests.
- Naming convention:
  - `*_unit_test.go` — unit tests
  - `*_property_test.go` — property-based tests (deterministic)
  - `*_fuzz_test.go` — fuzz entry points (exploratory)
  - `*_integration_test.go` — tests requiring external tools or build tags
- Directory layout:
  - Package-local testdata under `internal/<pkg>/testdata/`
  - E2E tests under `test/e2e/`
  - Integration tests under `internal/<pkg>/integration/` when package-scoped

## EVT Tags
- Every `Test*` and `Fuzz*` function MUST have an adjacent EVT tag comment immediately above the function per `spec/C-test-coverage.md` (Section C.2).
- Format examples:
  - `// [EVT-BCU-2]`
  - `// [EVT-MEP-3][EVT-MWP-6]`
- CI checks via `scripts/check-evt.sh` (see `make check-evt`). If no matching EVT exists yet, update `spec/C-test-coverage.md` before landing the test.

## Fuzz & Property Testing
- Follow `docs/testing/fuzz.md` for all policies and patterns (deterministic baselines, corpus replay, exploratory fuzzing). Avoid duplicating those rules here.
- Use `internal/testgen` and `internal/testsupport` as documented in `docs/testing/fuzz.md` and inline examples.
- Corpus placement and replay behavior are specified in `spec/08-testing.md` §8.4; use `testsupport.LoadCorpusSeeds` for standard search order.
- Keep exploratory fuzz out of CI; always replay minimized reproductions in regular tests.

## Corpus Placement
- See `spec/08-testing.md` §8.4 (normative) for corpus locations and the package-relative MUST rule.
- When multiple packages share cases, use `testdata/<domain>/<FuzzName>/` as a shared mirror, and document consumers.

## Sandbox & Integration
- Sandbox-based tests MUST skip cleanly when the environment is unsupported (non-Linux, missing bubblewrap/namespaces).
- `pass`/`gpg` integration tests MUST skip when dependencies are unavailable.

## Coding Style
- Keep changes minimal and focused. Avoid mixing unrelated refactors.
- Use clear names; avoid single-letter identifiers in non-trivial scopes.

## Submitting Changes
- Run `make pre-commit` and ensure all checks pass before pushing.
- Summarize the change and reference relevant EVT items if you introduce or extend coverage.
- For fuzz regressions, add minimized corpus files to the appropriate directory per `spec/08-testing.md` §8.4.
