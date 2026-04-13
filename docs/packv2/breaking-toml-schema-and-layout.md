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

## Implemented Schema And Layout Changes

| Change | 14.0 / V1 behavior | Current implemented behavior | Required migration action | Validation seam |
|---|---|---|---|---|
| Root model split | `city.toml` carried definition, deployment, and machine-local concerns together | A city now composes `pack.toml` plus `city.toml`, with `.gc/` as runtime/site state | Move portable definition into `pack.toml`; keep deployment in `city.toml`; keep machine-local state out of authored files | `internal/config/compose.go`, `docs/guides/migrating-to-pack-vnext.md` |
| Pack composition | V1 composition centered on `includes` / `packs` | Pack composition now centers on `[imports.<binding>]`; rig-scoped composition uses `[rigs.imports.<binding>]` | Rewrite composition to imports and bindings | `internal/config/pack.go`, `internal/config/compose.go`, `cmd/gc/testdata/pack-v2-imports.txtar` |
| Agent authoring shape | Agents were primarily declared as `[[agent]]` rows in TOML | Agents are convention-discovered from `agents/<name>/` | Move owned agent configuration under `agents/<name>/` | `internal/config/agent_discovery.go`, `docs/guides/migrating-to-pack-vnext.md` |
| Prompt file semantics | Prompt wiring depended on explicit TOML paths, and plain `.md` often implied templating in practice | `prompt.md` is inert markdown; `prompt.template.md` is the templated form | Rename templated prompts to `prompt.template.md`; keep plain prompts as `prompt.md` | `internal/config/agent_discovery.go`, `cmd/gc/prompt.go` |
| Overlay and namepool paths | Overlay and namepool paths were commonly wired from TOML | Overlay and namepool are now convention surfaces: `agents/<name>/overlay/` and `agents/<name>/namepool.txt` | Move owned assets into the agent directory | `internal/config/agent_discovery.go` |
| Template fragment layout | Fragment injection depended on legacy fragment fields | Template fragments now live in `template-fragments/` or `agents/<name>/template-fragments/` | Move reusable template fragments into convention directories | `cmd/gc/prompt.go`, `docs/guides/migrating-to-pack-vnext.md` |
| Agent defaults placement | Defaults were workspace-shaped and not clearly portable | `[agent_defaults]` is legal in both `pack.toml` and `city.toml`, with city winning on merge; current runtime inheritance remains limited to implemented fields | Move forward defaults into `[agent_defaults]`, but check the migration guide and skew analysis for the currently inherited subset | `internal/config/compose.go`, `internal/config/config.go`, `docs/packv2/skew-analysis.md` |
| Formula filename truth | Formula layout was still being normalized during the rollout | Current formula truth is `formulas/<name>.formula.toml` | Rename formula files to the flat `.formula.toml` convention used on this branch | `cmd/gc/system_formulas.go`, `internal/citylayout/layout.go` |
| Order filename truth | Order layout was still being normalized during the rollout | Current order truth is `orders/<name>.order.toml` | Rename order files to the flat `.order.toml` convention used on this branch | `internal/orders/discovery.go`, `cmd/gc/cmd_order.go` |
| Command and doctor directories | Commands and doctor checks were mixed between TOML-declared inventory and ad hoc script paths | Commands now live under `commands/`; doctor checks live under `doctor/`, each with local entry directories and minimal manifests | Move operational entrypoints under `commands/` and `doctor/` and follow the local `run.sh` / `help.md` convention | `internal/config/command_discovery.go`, `internal/config/doctor_discovery.go`, `docs/packv2/breaking-gc-command-surface.md` |

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
