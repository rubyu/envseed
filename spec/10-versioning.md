## 10. Versioning & Release Semantics

### 10.1 Version String Forms
- Stable release: `v<MAJOR>.<MINOR>.<PATCH>+<DATE>.<SHA>`
- Development snapshot: `v<MAJOR>.<MINOR>.<PATCH>-dev+<DATE>.<SHA>[.dirty]`

Constraints:
- `MAJOR`, `MINOR`, `PATCH` are non-negative integers.
- `DATE` is UTC in the form `YYYYMMDD`.
- The `SHA` output MUST be a lowercase hexadecimal commit identifier normalized to exactly 12 characters.
- `dirty` MUST be appended when the working tree contains uncommitted changes at build time. Release tags MUST be created from a clean tree; stable release tags MUST NOT include `.dirty`.
- Stable releases MUST NOT include a pre-release segment and MUST include build metadata `+<DATE>.<SHA>` (no `.dirty`).
- Development snapshots MUST use the `-dev` pre-release segment and include build metadata `+<DATE>.<SHA>`.

### 10.2 Branch and Tag Rules
- main branch:
  - Tagged commit: the tag denotes a stable release. The tag name MUST be `v<MAJOR>.<MINOR>.<PATCH>` (no build metadata) and MUST NOT include a pre-release segment. Tags MUST be created from a clean tree (no `.dirty`).
  - Untagged commit: emit a snapshot version derived from the latest stable tag `vM.m.p[+...]` by incrementing the patch: `vM.m.(p+1)-dev+<DATE>.<SHA>[.dirty]`.
- develop branch:
  - Always snapshot: derive from the latest stable tag `vM.m.p[+...]` by incrementing the minor: `vM.(m+1).0-dev+<DATE>.<SHA>[.dirty]`.
- Bootstrap when no stable tag exists:
  - Baseline is `v0.0.0`. On main, use `v0.0.1-dev+<DATE>.<SHA>[.dirty]`; on develop, use `v0.1.0-dev+<DATE>.<SHA>[.dirty]`.

Prohibitions:
- Teams MUST NOT create stable (pre-release-free) tags on the `develop` branch.

### 10.3 Version Resolution and Embedding
- The version string printed by `envseed --version` and `envseed version` MUST be embedded at build time (e.g., via Go `-ldflags -X internal/version.Version=<STRING>`), and MUST conform to the output rules in Section 10.4 and the branch/tag rules in Section 10.2.
- If no embedded version string is provided, the command MUST emit the fallback string `v0.0.0-dev+<DATE>.unknown` where `<DATE>` is UTC `YYYYMMDD`. In this fallback, `.dirty` MUST NOT be appended.
- Runtime VCS probing (e.g., invoking Git to compute branch, tag, or SHA) MUST NOT be performed.

### 10.4 Output Semantics
- `envseed --version` and `envseed version` MUST print exactly one line containing the version string to stdout and exit code 0.
- For a tagged stable release on main, the printed version string MUST be `v<MAJOR>.<MINOR>.<PATCH>+<DATE>.<SHA>` and MUST NOT include `.dirty`.
- Examples (Informative):
  - main (tagged): `v1.2.3+20241103.abcdef123456`
  - main (untagged): `v1.2.4-dev+20241103.abcdef123456`
  - develop: `v1.3.0-dev+20241103.abcdef123456`
