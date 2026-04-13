# Breaking Changes: TOML Schema, Directories, And File Formats

This note records the implemented schema, directory, and file-format
breakage in the current Pack/City v2 rollout.

It is a release-facing summary, not a design memo:

- include current branch truth only
- keep deferred work out of the main table
- point each row at a validation seam or authoritative doc

## Format And Generation Contract

Future breaking-change notes in this directory should use the same
structure:

1. scope statement
2. breaking-change table
3. compatibility / deferred notes
4. generation notes

The long-term goal is to generate this class of doc from the same inputs
that already drive the reference schema docs:

- `docs/reference/config.md`
- release-gated pack/city conformance rows
- named testscript and unit seams

Until then, keep the wording tied to implemented branch truth.

When a change is concrete enough to show succinctly, the breaking-change
table should include:

- `Category`
- `Before`
- `After`

using short, abstract examples instead of long prose.

## Implemented Schema And Layout Changes

| Category | Change | Before | After | Required migration action | Validation seam |
|---|---|---|---|---|---|
| Root model | Root model split | one `city.toml` carried everything | `pack.toml` + `city.toml` + `.gc/` | Move portable definition into `pack.toml`; keep deployment in `city.toml`; keep machine-local state out of authored files | `internal/config/compose.go`, `docs/guides/migrating-to-pack-vnext.md` |
| Composition | Pack composition | `includes = ["../gastown"]` | `[imports.gastown] source = "../gastown"` | Rewrite composition to imports and bindings | `internal/config/pack.go`, `internal/config/compose.go`, `cmd/gc/testdata/pack-v2-imports.txtar` |
| Agents | Agent authoring shape | `[[agent]] name = "mayor"` | `agents/mayor/agent.toml` | Move owned agent configuration under `agents/<name>/` | `internal/config/agent_discovery.go`, `docs/guides/migrating-to-pack-vnext.md` |
| Prompts | Prompt file semantics | `prompt_template = "prompts/mayor.md"` | `agents/mayor/prompt.template.md` | Rename templated prompts to `prompt.template.md`; keep plain prompts as `prompt.md` | `internal/config/agent_discovery.go`, `cmd/gc/prompt.go` |
| Agent assets | Overlay and namepool paths | `overlay_dir = "overlays/mayor"` and `namepool = "pools/mayor.txt"` | `agents/mayor/overlay/` and `agents/mayor/namepool.txt` | Move owned assets into the agent directory | `internal/config/agent_discovery.go` |
| Template fragments | Template fragment layout | `global_fragments = ["ops"]` | `template-fragments/ops.md` or `agents/mayor/template-fragments/ops.md` | Move reusable template fragments into convention directories | `cmd/gc/prompt.go`, `docs/guides/migrating-to-pack-vnext.md` |
| Defaults | Agent defaults placement | `workspace.provider = "claude"` as the obvious defaults bucket | `[agent_defaults]` in `pack.toml` or `city.toml` | Move forward defaults into `[agent_defaults]`, but check the migration guide and skew analysis for the currently inherited subset | `internal/config/compose.go`, `internal/config/config.go`, `docs/packv2/skew-analysis.md` |
| Formulas | Formula filename truth | mixed or nested formula locations during rollout | `formulas/<name>.formula.toml` | Rename formula files to the flat `.formula.toml` convention used on this branch | `cmd/gc/system_formulas.go`, `internal/citylayout/layout.go` |
| Orders | Order filename truth | mixed or nested order locations during rollout | `orders/<name>.order.toml` | Rename order files to the flat `.order.toml` convention used on this branch | `internal/orders/discovery.go`, `cmd/gc/cmd_order.go` |
| Operational entries | Command and doctor directories | `[[commands]]`, `[[doctor]]`, or ad hoc script paths | `commands/...` and `doctor/...` convention directories | Move operational entrypoints under `commands/` and `doctor/` and follow the local `run.sh` / `help.md` convention | `internal/config/command_discovery.go`, `internal/config/doctor_discovery.go`, `docs/packv2/breaking-gc-command-surface.md` |

## Compatibility And Deferred Notes

- Legacy `[[agent]]`, `[[commands]]`, and `[[doctor]]` TOML surfaces still exist for migration compatibility on this branch, but they are no longer the forward authoring surface.
- This note intentionally reflects the current branch truth for formulas and orders; the separate infix-removal cleanup lane may change those filenames again.
- This note does not claim that `skills`, `mcp`, `.gc/site.toml`, or loader-discovered patch directories are implemented.

## Generation Notes

If this document becomes generated, the minimum useful inputs are:

- `docs/reference/config.md` for as-built field inventory
- `docs/packv2/skew-analysis.md` for release-gated truth
- `docs/packv2/doc-conformance-matrix.md` for current validation scope
- release proof seams such as:
  - `cmd/gc/testdata/migrate-v2.txtar`
  - `cmd/gc/testdata/pack-v2-imports.txtar`
  - `internal/config/compose.go`
  - `internal/config/pack.go`
