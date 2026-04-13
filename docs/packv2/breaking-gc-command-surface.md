# Breaking Changes: `gc` Command Surface

This note records the implemented command-surface breakage in the current
Pack/City v2 rollout.

It is intentionally narrower than the full design docs:

- include implemented or release-gated behavior only
- exclude target-state-only ideas that are still deferred
- point every row at a validation seam when possible

## Format And Generation Contract

Future breaking-change notes in this directory should follow this same
shape:

1. scope statement
2. breaking-change table
3. compatibility / deferred notes
4. generation notes

The target end state is for docs like this to be machine-generated from:

- `docs/reference/cli.md`
- `docs/packv2/doc-conformance-matrix.md`
- release-gated testscript and unit seams

Until then, keep these notes branch-truth only. Do not list behavior that
is merely desired.

When a change is concrete enough to show succinctly, the breaking-change
table should include:

- `Category`
- `Before`
- `After`

using small, abstract examples rather than full product-specific prose.

## Implemented Command-Surface Changes

| Category | Change | Before | After | Required migration action | Validation seam |
|---|---|---|---|---|---|
| Namespace | Imported pack command namespace | `gc ops status` where `ops` is the pack name | `gc crew status` where `crew` is the import binding | Update scripts and docs to call pack commands through the binding namespace | `cmd/gc/cmd_commands.go`, `cmd/gc/testdata/pack-commands-doctor.txtar` |
| Naming | Command word mapping | `commands/repo-sync/command.toml` with `command = ["repo", "sync"]` | `commands/repo/sync/run.sh` | Move multi-word commands into nested directories under `commands/` | `internal/config/command_discovery.go`, `internal/config/command_discovery_test.go` |
| Tree shape | Runnable parent nodes | `commands/foo/run.sh` blocked `commands/foo/bar/run.sh` | `commands/foo/run.sh` and `commands/foo/bar/run.sh` both work | Keep `commands/foo/run.sh` and `commands/foo/bar/run.sh` together when both are real commands | `internal/config/command_discovery_test.go`, `cmd/gc/cmd_commands_test.go` |
| Help | Help on parent namespaces | `commands/foo/help.md` only mattered if `foo` was runnable | `commands/foo/help.md` may document a non-runnable parent node | Move namespace help to the directory that owns that command node | `internal/config/command_discovery.go`, `cmd/gc/cmd_commands.go` |
| Manifest | `command.toml` forward surface | `command = [...]`, `description`, `run` | `description`, `run` only | Remove `command = [...]`; rename directories instead | `internal/config/command_discovery.go`, `internal/config/command_discovery_test.go` |
| Entrypoint | Entry-point convention | `run = "./scripts/entry.sh"` as a common pattern | `run.sh` by default, or `run = "sync.sh"` locally | Prefer `run.sh`; use `run = "name.sh"` only when needed | `internal/config/command_discovery.go`, `docs/guides/migrating-to-pack-vnext.md` |
| Collision | Core-name collision rule | `[imports.start]` plus commands was under-specified | a binding that exports commands may not shadow a core top-level command | Rename the import binding if it exports commands under a reserved top-level name | `cmd/gc/cmd_pack_commands.go`, `cmd/gc/cmd_commands_test.go` |
| Identity | Doctor visible identity | `maintenance:tooling` | `crew:tooling` | Update scripts, docs, and expectations that grep or assert doctor check names | `cmd/gc/cmd_doctor.go`, `cmd/gc/cmd_doctor_test.go`, `cmd/gc/testdata/pack-commands-doctor.txtar` |
| Layout | Doctor layout | doctor shape was still a design-forward sibling of commands | `doctor/tooling/run.sh` with optional `help.md` and minimal `doctor.toml` | Keep doctor checks flat; do not model nested doctor hierarchies | `internal/config/doctor_discovery.go`, `internal/config/doctor_discovery_test.go` |

## Compatibility And Deferred Notes

- Legacy `[[commands]]` and `[[doctor]]` pack TOML entries still compose for migration compatibility.
- This note does not freeze any broader extension-root command model, alias model, or future command-product expansion.
- This note does not cover patch surfaces, `skills`, `mcp`, or `.gc/site.toml`.

## Generation Notes

If this document becomes generated, the minimum useful inputs are:

- built-in CLI/reference data from `docs/reference/cli.md`
- release-gated assertions from `docs/packv2/doc-conformance-matrix.md`
- explicit proof seams from:
  - `cmd/gc/testdata/pack-commands-doctor.txtar`
  - `cmd/gc/cmd_commands_test.go`
  - `cmd/gc/cmd_doctor_test.go`
  - `internal/config/command_discovery_test.go`
  - `internal/config/doctor_discovery_test.go`
