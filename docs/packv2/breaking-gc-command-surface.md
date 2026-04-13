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

## Implemented Command-Surface Changes

| Change | 14.0 / V1 behavior | Current implemented behavior | Required migration action | Validation seam |
|---|---|---|---|---|
| Imported pack command namespace | Pack commands were surfaced by pack name | Imported pack commands are surfaced under the import binding: `gc <binding> <command ...>` | Update scripts and docs to call pack commands through the binding namespace | `cmd/gc/cmd_commands.go`, `cmd/gc/testdata/pack-commands-doctor.txtar` |
| Command word mapping | `[[commands]]` and `command = [...]` could decouple CLI words from on-disk shape | Directory structure defines command words; nested directories define nested words | Move multi-word commands into nested directories under `commands/` | `internal/config/command_discovery.go`, `internal/config/command_discovery_test.go` |
| Runnable parent nodes | A discovered runnable node stopped descent, so `foo` and `foo bar` could not both exist by convention | A command node may be both runnable and a parent | Keep `commands/foo/run.sh` and `commands/foo/bar/run.sh` together when both are real commands | `internal/config/command_discovery_test.go`, `cmd/gc/cmd_commands_test.go` |
| Help on parent namespaces | Help text only attached to discovered runnable leaves | `help.md` is valid on any command node, including a non-runnable parent | Move namespace help to the directory that owns that command node | `internal/config/command_discovery.go`, `cmd/gc/cmd_commands.go` |
| `command.toml` forward surface | Manifest could carry `command`, `description`, and `run` | `command.toml` is reduced to `description` and `run`; `command = [...]` is no longer part of the forward surface | Remove `command = [...]`; rename directories instead | `internal/config/command_discovery.go`, `internal/config/command_discovery_test.go` |
| Entry-point convention | Pack command scripts were commonly modeled as arbitrary script paths from TOML | `run.sh` is the preferred default entrypoint; `run` is only the local filename override | Prefer `run.sh`; use `run = "name.sh"` only when needed | `internal/config/command_discovery.go`, `docs/guides/migrating-to-pack-vnext.md` |
| Core-name collision rule | Collision handling was under-specified | A binding that actually contributes commands may not shadow a built-in top-level `gc` command or alias | Rename the import binding if it exports commands under a reserved top-level name | `cmd/gc/cmd_pack_commands.go`, `cmd/gc/cmd_commands_test.go` |
| Doctor visible identity | Pack doctor checks were displayed as `<packName>:<name>` | Pack doctor checks are displayed as `<binding>:<name>` when a binding is present | Update scripts, docs, and expectations that grep or assert doctor check names | `cmd/gc/cmd_doctor.go`, `cmd/gc/cmd_doctor_test.go`, `cmd/gc/testdata/pack-commands-doctor.txtar` |
| Doctor layout | Doctor behavior was still described as parallel design work | Doctor is now a flat discovered surface: `doctor/<name>/` with `run.sh`, optional `help.md`, and optional minimal `doctor.toml` | Keep doctor checks flat; do not model nested doctor hierarchies | `internal/config/doctor_discovery.go`, `internal/config/doctor_discovery_test.go` |

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
