---
title: "Migrating to Gas City 0.13.6"
description: How to move an existing Gas City 0.13.5 city or pack to the Gas City 0.13.6 pack/city schema and directory conventions.
---

This guide is the practical migration companion to the Gas City 0.13.6
package and city file and directory structure.

Gas City 0.13.6 separates a city into three layers:

- **Definition**
  - portable pack definition in `pack.toml` plus well-known top-level directories
- **Deployment**
  - team-shared city deployment in `city.toml`
- **Binding and runtime state**
  - machine-local state in `.gc/`

The user-facing migration work is mostly in the first two layers.

## Before you start

The important mental shift is:

- **Gas City 0.13.5** centers on `city.toml` and a lot of explicit path wiring
- **Gas City 0.13.6** centers on `pack.toml`, named imports, and convention-based directories

The clean target shape for a city is:

```text
my-city/
├── pack.toml
├── city.toml
├── agents/
├── formulas/
├── orders/
├── commands/
├── doctor/
├── overlays/
├── skills/
├── mcp/
├── template-fragments/
├── assets/
└── .gc/
```

The clean target shape for a reusable pack is the same, just without
`city.toml` and `.gc/`.

The broad file-structure rule is simple: use `pack.toml` for pack-wide
metadata and policy, use `city.toml` for deployment choices (e.g., rigs, ports), use
well-known definition directories for things like agents and formulas,
and use `assets/` for everything else the pack carries.

## First: split `city.toml` and `pack.toml`

This is the most important migration step. Everything else hangs off it.

In Gas City 0.13.6, a city is a deployed pack. That means the root city
directory has its own `pack.toml`, and the old "everything lives in
`city.toml`" model gets broken apart.

### What belongs in `pack.toml`

`pack.toml` is the home for portable definition:

- pack identity and compatibility metadata
- imports
- providers
- pack-wide agent defaults
- named sessions
- pack-level patches
- other pack-wide declarative policy

The important change from 0.13.5 is that most definitions are done based on directory and file convention, not in TOML. In 0.13.6, much of the old TOML inventory moves into named files
and directories:

- `[[agent]]` definitions move to `agents/<name>/`
- `[formulas]` directory wiring gives way to `formulas/<name>.formula.toml`
- order definitions move to `orders/<name>.order.toml`
- `[[commands]]` definitions move to `commands/<name>/`
- `[[doctor]]` definitions move to `doctor/<name>/`

So `pack.toml` gets narrower even as the pack definition gets richer.

`city.toml`, by contrast, is the home for deployment:

- rigs
- rig-specific composition and patches
- substrate choices
- API and daemon behavior
- capacity and scheduling policy

Rigs are the main thing that remain in `city.toml`.

## First concrete step: move includes to imports

For most existing cities, the first change you will actually make is
composition.

In Gas City 0.13.5, composition is include-based. In Gas City 0.13.6,
composition is import-based.

Use the city pack's `pack.toml` for city-wide imports. Use rig-scoped
imports in `city.toml` when a pack should compose only into one rig.
If you used `workspace.default_rig_includes`, that maps to
`[defaults.rig.imports.*]` in the root `pack.toml`.

### Smallest city-wide example

Before:

```toml
# city.toml
[workspace]
name = "my-city"
includes = ["maintenance"]
```

After:

```toml
# pack.toml
[pack]
name = "my-city"
schema = 2

[imports.maintenance]
source = "https://github.com/gastownhall/gascity-packs/maintenance"
```
Remote imports are expected to resolve into local materialized state,
not require live internet access on every load. Fetching, updating, and
re-materializing missing imports is the online step. The lock file is
the authoritative statement of desired installed state, and normal load
should use local materialized content. If that content is missing, Gas
City should repair from local cache first and then from the remote
source if needed.

The planned command surface for checking and repairing this state is
`gc import check`. That missing feature is tracked in
[#575](https://github.com/gastownhall/gascity/issues/575).

That is the core change:

- `includes` moves out of `city.toml`
- the imported pack gets a local name
- the import points at the pack's stable repository URL, not at an ambient `packs/` directory

### Add another imported pack

Before:

```toml
# city.toml
[workspace]
name = "my-city"
includes = ["maintenance", "gastown"]
```

After:

```toml
# pack.toml
[pack]
name = "my-city"
schema = 2

[imports.maintenance]
source = "https://github.com/gastownhall/gascity-packs/maintenance"

[imports.gastown]
source = "https://github.com/gastownhall/gascity-packs/gastown"
```

The maintenance, gastown, and dog packs are now in the `gascity-packs`
repository and you can import them directly by source. If those were
the only packs in your old `packs/` directory, you can delete that
directory after the migration.

If you have a local custom pack, the preferred migration is to move it
to its own git repository and refer to it by URL:

```toml
# pack.toml
[imports.my-helper]
source = "https://github.com/my-org/my-helper-pack"
```

If you want the pack contents be local to your city and move around with it, keep the
pack under `assets/` and import it explicitly.

Before:

```text
my-city/
├── city.toml
└── packs/
    └── my-helper/
        └── pack.toml
```

After:

```text
my-city/
├── pack.toml
└── assets/
    └── local-packs/
        └── my-helper/
            └── pack.toml
```

```toml
# pack.toml
[imports.my-helper]
source = "./assets/local-packs/my-helper"
```

// I'm a little confused. I could swear that we had both per-rig import as well as "default rig imports". Am i misremembering? Either way can you check and make sure all is correct

### Change rig-specific composition from include to import

Before:

```toml
# city.toml
[[rigs]]
name = "api-server"
path = "/srv/api"
includes = ["gastown"]
```

After:

```toml
# city.toml
[[rigs]]
name = "api-server"

[rigs.imports.gastown]
source = "https://github.com/gastownhall/gascity-packs/gastown"
```

Rig-scoped imports are the direct replacement for old `rigs.includes`.

### Move default rig includes to default rig imports

Before:

```toml
# city.toml
[workspace]
default_rig_includes = ["gastown"]
```

After:

```toml
# pack.toml
[defaults.rig.imports.gastown]
source = "https://github.com/gastownhall/gascity-packs/gastown"
```

This is different from `rigs.imports.*`:

- `rigs.imports.*` applies to one specific rig already declared in `city.toml`
- `[defaults.rig.imports.*]` is the default import set for newly created rigs

As you migrate, the usual pattern is:

- move portable definition into `pack.toml` and named definition directories
- leave rigs and other deployment choices in `city.toml`

### Imports create named bindings

Gas City 0.13.6 does not just rename `includes` to `imports`. In Gas
City 0.13.5, includes mainly flattened pack content into a city or rig.
In Gas City 0.13.6, imports create named bindings.

Example:

```toml
# pack.toml
[imports.gastown]
source = "https://github.com/gastownhall/gascity-packs/gastown"
```

Here, `gastown` is not just a label for where the pack came from. It is
the local name for that imported pack inside the city pack. That is the
main semantic change behind the migration from includes to imports.

You use that local name anywhere the composed system needs to refer to
an imported definition explicitly.

For example, if you import two packs that both define an agent called
`mayor`:

```toml
# pack.toml
[imports.civic]
source = "https://github.com/gastownhall/gascity-packs/civic"

[imports.ops]
source = "https://github.com/gastownhall/gascity-packs/ops"
```

then the local names are what let you qualify them as `civic.mayor` and
`ops.mayor` instead of relying on load order or accidental flattening.

## Then: migrate area by area

Once the root split is in place, the rest of the work gets much more
mechanical.

## Agents

What changed:

- agents moved out of inline TOML inventories into named directories under `agents/`
- prompt content now lives with the agent that owns it
- agent-local assets such as overlays can live with the agent too

Why:

- each agent becomes a self-contained unit
- prompt files are easier to find and edit
- `pack.toml` no longer has to inventory every agent path

### Smallest agent migration

Before:

```toml
# city.toml or pack.toml
[[agent]]
name = "mayor"
prompt_template = "prompts/mayor.md"
```

After:

```text
agents/
└── mayor/
    └── prompt.md
```

### Add agent-specific configuration

Before:

```toml
# city.toml or pack.toml
[[agent]]
name = "mayor"
prompt_template = "prompts/mayor.md"
wake_mode = "fresh"
```

After:

```text
agents/
└── mayor/
    ├── prompt.md
    └── agent.toml
```

```toml
# agents/mayor/agent.toml
wake_mode = "fresh"
```

### Move the default provider into `[agents]`

> **Pending potential change**: [#580: We need to scrub field names..](https://github.com/gastownhall/gascity/issues/580). Something seems fishy here.


In Gas City 0.13.5, the default provider lived on `[workspace]`. In Gas
City 0.13.6, that default belongs on `[agents]` in `pack.toml`.



Before:

```toml
# city.toml
[workspace]
provider = "claude"

[[agent]]
name = "mayor"
prompt_template = "prompts/mayor.md"

[[agent]]
name = "deacon"
prompt_template = "prompts/deacon.md"

[[agent]]
name = "scribe"
prompt_template = "prompts/scribe.md"
provider = "codex"
```

After:

```toml
# pack.toml
[agents]
provider = "claude"
```

```text
agents/
├── mayor/
│   └── prompt.md
├── deacon/
│   └── prompt.md
└── scribe/
    ├── prompt.md
    └── agent.toml
```

```toml
# agents/scribe/agent.toml
provider = "codex"
```

Agents that use the common default no longer need to repeat it. Agents
that differ can still override it locally.

### Add other shared defaults

> **Pending potential change**: [#580: We need to scrub field names and default behavior..](https://github.com/gastownhall/gascity/issues/580). Something seems fishy here.


Provider is not the only thing that can move into `[agents]`.

Before:

```toml
# city.toml or pack.toml
[[agent]]
name = "mayor"
prompt_template = "prompts/mayor.md"
wake_mode = "fresh"

[[agent]]
name = "deacon"
prompt_template = "prompts/deacon.md"
wake_mode = "fresh"
```

After:

```toml
# pack.toml
[agents]
wake_mode = "fresh"
```

```text
agents/
├── mayor/
│   └── prompt.md
└── deacon/
    └── prompt.md
```

The same pattern applies to other agent defaults that belong at pack
scope, such as `model`, `default_sling_formula`, `allow_overlay`, and
`allow_env_override`.

In 0.13.6, the supported pack-wide defaults under `[agents]` are:

- `provider`
- `model`
- `wake_mode`
- `default_sling_formula`
- `allow_overlay`
- `allow_env_override`

### Add agent-local overlay content

Overlays are files copied into an agent's working environment at
startup. In 0.13.5 they were usually referenced by path. In 0.13.6 they
can live directly with the agent that uses them.

Before:

```toml
# city.toml or pack.toml
[[agent]]
name = "mayor"
prompt_template = "prompts/mayor.md"
overlay_dir = "overlays/default"
```

After:

```text
agents/
└── mayor/
    ├── prompt.md
    ├── agent.toml
    └── overlay/
```

If you are migrating a city, city-local agents are still just agents in
the root city pack.

> **Pending potential  change**: [#582: Do we require the .tmpl extension to trigger template processing](https://github.com/gastownhall/gascity/issues/582)

Prompt processing now follows an explicit file-extension rule:

- `prompt.md` is plain Markdown
- `prompt.md.tmpl` is rendered through the template engine

So if your old prompt relied on template expansion, rename it to
`prompt.md.tmpl` as part of the migration. If it is plain prompt
content, keep it as `prompt.md`.
> **Pending potential change**: The .tmpl issue again.

## Formulas

Formulas stay as named files, but the convention becomes stricter.

### Smallest formula migration

Before:

```toml
# city.toml or pack.toml
[formulas]
dir = "formulas"
```

```text
formulas/
└── build-review.formula.toml
```

After:

```text
formulas/
└── build-review.formula.toml
```

> **Pending potential change**: [#581: Do we need the infix for this and orders.](https://github.com/gastownhall/gascity/issues/581)

The file shape stays familiar. What changes is that the directory is now
the convention instead of something you wire up in TOML.  If your city.toml file pointed formulas to any other directory, you'll need to move them to the `formulas/` directory.

## Orders

Orders have moved to look like formulas.

The new rule is:

- orders live at top-level `orders/`
- each order is one flat file
- orders no longer live under `formulas/orders/`

### Smallest order migration

Before:

```text
formulas/
└── orders/
    └── nightly-sync/
        └── order.toml
```

After:

```text
orders/
└── nightly-sync.order.toml
```

This gives a consistent pair:

- `formulas/<name>.formula.toml`
- `orders/<name>.order.toml`

The extra noun in the filename is intentional. Gas City 0.13.6 uses the
`name.noun.toml` pattern for file-based definitions so the file is
self-describing even outside its directory context.

## Commands

Commands have moved toward convention-first entry directories.

### Smallest command migration

Before:

```toml
# pack.toml
[[commands]]
name = "status"
script = "commands/status.sh"
```

After:

```text
commands/
└── status/
    └── run.sh
```

### Add command metadata

Before:

```toml
# pack.toml
[[commands]]
name = "status"
description = "Show status"
script = "commands/status.sh"
```

After:

```text
commands/
└── status/
    ├── command.toml
    └── run.sh
```

```toml
# commands/status/command.toml
description = "Show status"
```

### Add local help

After:

```text
commands/
└── status/
    ├── command.toml
    ├── run.sh
    └── help.md
```

Use `command.toml` only when the default `commands/<name>/run.sh`
mapping is not enough or when you need local metadata.

## Doctor checks

Doctor checks now follow the same on-disk pattern as commands.

### Smallest doctor migration

Before:

```toml
# pack.toml
[[doctor]]
name = "binaries"
script = "doctor/check-binaries.sh"
```

After:

```text
doctor/
└── binaries/
    └── run.sh
```

### Add doctor metadata

Before:

```toml
# pack.toml
[[doctor]]
name = "binaries"
description = "Verify required tools are installed"
script = "doctor/check-binaries.sh"
```

After:

```text
doctor/
└── binaries/
    ├── doctor.toml
    └── run.sh
```

```toml
# doctor/binaries/doctor.toml
description = "Verify required tools are installed"
```

### Add local help

After:

```text
doctor/
└── binaries/
    ├── doctor.toml
    ├── run.sh
    └── help.md
```

Use `doctor.toml` only when the default `doctor/<name>/run.sh` mapping
is not enough or when you need local metadata.

## Assets and paths

Gas City 0.13.5 was fairly loose about how a city or pack directory was
structured, and put a lot of weight on TOML to wire things up.

Gas City 0.13.6 moves to a more explicit model where the directory
structure carries much more of the definition. To make that work, the
top-level directory structure is controlled for any given version of Gas
City.

To allow pack authors to carry arbitrary files, Gas City reserves one
top-level directory, `assets/`, that is treated as opaque data.

### Smallest assets migration

Before:

```text
my-pack/
├── pack.toml
└── helper-data/
```

After:

```text
my-pack/
├── pack.toml
└── assets/
    └── helper-data/
```

### Path-valued fields

Any field that accepts a path may point to any file inside the same
pack.

That includes:

- files under standard directories
- files under `assets/`
- relative paths that use `..`

The hard constraint is:

- after normalization, the path must still stay inside the pack root

### Examples

```toml
run = "./run.sh"
help = "./help.md"
run = "../shared/run.sh"
source = "./assets/imports/maintenance"
```

## Overlays

Overlays now have a clearer split between pack-wide and agent-local
content.

### Smallest overlay migration

Before:

```toml
# city.toml or pack.toml
[[agent]]
name = "mayor"
overlay_dir = "overlays/default"
```

After:

```text
agents/
└── mayor/
    └── overlay/
```

This is the smallest agent-local overlay case. The overlay content moves
out of a shared directory path and into the agent definition itself.

### Move shared overlay content

After:

```text
overlays/
```

When overlay content is shared across the pack rather than attached to a
single agent, keep it in top-level `overlays/`.

Use:

- `overlays/` for pack-wide overlay material
- `agents/<name>/overlay/` for agent-local overlay material

## Template fragments

Template fragments already existed, but in 0.13.6 they sit more cleanly
inside the overall pack structure and can also exist as agent-local
definitions.

Before:

```text
template-fragments/
└── review.md.tmpl
```

After:

```text
template-fragments/
└── review.md.tmpl
```

They can also have agent-local counterparts:

```text
agents/
└── mayor/
    └── template-fragments/
```

## Skills and MCP

Gas City 0.13.6 adds explicit support for skills and MCP servers. This
is a new feature, but you may have used other techniques to achieve the
same end.

They can be defined per-pack and apply to all agents:

```text
skills/
└── my-skill/
    └── SKILL.md
mcp/
└── my-server.toml
```

or individually to a specific agent:

```text
agents/
└── mayor/
    ├── skills/
    │   └── my-skill/
    │       └── SKILL.md
    └── mcp/
        └── my-server.toml
```

## Common migration gotchas

### "I still have a lot in `city.toml`"

That usually means definition and deployment are still mixed together.

Ask:

- is this portable definition?
- is this deployment?

Then move it to:

- `pack.toml` and discovered pack directories
- `city.toml`

respectively.

### "I used to rely on `scripts/`"

Do not recreate `scripts/` as a standard top-level convention just
because 0.13.5 had it.

Instead:

- put entrypoint scripts next to the command or doctor entry that uses them
- put general helper scripts you want to share across multiple commands or doctor checks under `assets/`

For example, this old pattern:

```text
scripts/
└── setup.sh
```

plus:

```toml
session_setup_script = "scripts/setup.sh"
```

becomes either:

```text
commands/status/run.sh
```

or:

```text
assets/scripts/setup.sh
```

depending on whether the script is entry-local or a general helper.

### "Do I need TOML everywhere?"

No.

Simple cases should work by convention:

- `agents/<name>/prompt.md`
- `commands/<name>/run.sh`
- `doctor/<name>/run.sh`

Use TOML when you actually need:

- defaults
- overrides
- metadata
- explicit placement

### "Will `gc doctor` catch structural mistakes?"

That is the expectation. Gas City 0.13.6 leans much more heavily on
convention, so `gc doctor` should be the place that validates common
pack and city layout mistakes.

Examples include:

- unrecognized top-level directories
- missing expected files in recognized definition directories
- malformed command or doctor entries
- invalid path references

> **Pending potential change**: [#575: Import cache and materialization validation is a separate concern. That
work should land under `gc import check`, not under `gc doctor`.](https://github.com/gastownhall/gascity/issues/575).

## Reference: Gas City 0.13.5 `city.toml` elements to 0.13.6

This is the exhaustive top-level lookup table for the old `city.toml`
schema, plus the qualified rows that matter most during migration.

| 0.13.5 element | What it did | New home or action |
|---|---|---|
| `include` | Merged extra config fragments into `city.toml` before load | Remove as part of migration. Move real composition to imports and move remaining config to `pack.toml`, `city.toml`, or well-known definition directories. |
| `[workspace]` | Held city metadata and pack composition in one place | Split across the root `pack.toml`, `city.toml`, and `.gc/`. |
| `workspace.name` | Workspace identity | Managed site binding, not portable definition. Do not model this as pack content. |
| `workspace.includes` | City-level pack composition | Move to `[imports.*]` in the root city `pack.toml`. |
| `[providers.*]` | Named provider presets | Usually move to `[providers.*]` in the root city `pack.toml`, unless the setting is truly deployment-only. |
| `[packs.*]` |Named remote pack sources used by includes | Collapse into `[imports.*]` entries. There should no longer be a separate `[packs.*]` registry in `city.toml`. |
| `[[agent]]` | Inline agent definitions | Move to `agents/<name>/`, with optional `agent.toml`. |
| `agent.prompt_template` | Path to agent prompt | Move to `agents/<name>/prompt.md`. |
| `agent.overlay_dir` | Path to overlay content | Move content to `agents/<name>/overlay/` or pack-wide `overlays/`. |
| `agent.session_setup_script` | Path to setup script | Keep as a path-valued field, but point at a pack-local file, usually in the agent's directory, or `assets/` if it's a shared script. |
| `agent.namepool` | Path to names file | Move toward agent-local content such as `agents/<name>/names.txt` if retained. |
| `[[named_session]]` | Named reusable sessions | Move to `[[named_session]]` in the root city `pack.toml`. |
| `[[rigs]]` | Rig deployment entries | Keep in `city.toml`. |
| `rigs.path` | Machine-local project binding | Managed site binding, not portable pack definition. |
| `rigs.prefix` | Derived rig prefix | Managed site binding, not portable pack definition. |
| `rigs.suspended` | Operational toggle | Managed site binding, not portable pack definition. |
| `rigs.includes` | Rig-scoped pack composition | Move to rig-scoped imports in `city.toml`. |
| `workspace.default_rig_includes` | Default pack composition for newly created rigs | Move to `[defaults.rig.imports.*]` in the root city `pack.toml`. |
| `rigs.overrides` | Rig-specific customization of imported agents | Keep as rig-level deployment customization in `city.toml`. |
| `[patches]` | Post-merge modifications | Move pack-definition patches to `pack.toml`. Keep rig-specific patches with the rig in `city.toml`. |
| `[beads]` | Bead store backend choice | Keep in `city.toml`. |
| `[session]` | Session substrate config | Keep in `city.toml`, except site-local bindings. |
| `[mail]` | Mail substrate config | Keep in `city.toml`. |
| `[events]` | Events substrate config | Keep in `city.toml`. |
| `[dolt]` | Dolt connection defaults | Keep in `city.toml`. |
| `[formulas]` | Formula directory config | Remove. Formula location is now the fixed top-level `formulas/` convention. |
| `formulas.dir` | Formula directory path | Replace with the fixed top-level `formulas/` convention. |
| `[daemon]` | Controller daemon behavior | Keep in `city.toml`. |
| `[orders]` | Order runtime policy such as skip lists and timeouts | Keep in `city.toml`. |
| `[api]` | API server deployment config | Keep in `city.toml`, except machine-local bind details. |
| `[chat_sessions]` | Chat session runtime policy | Keep in `city.toml`. |
| `[session_sleep]` | Sleep policy defaults | Keep in `city.toml`. |
| `[convergence]` | Convergence limits | Keep in `city.toml`. |
| `[[service]]` | Workspace-owned service declarations | Keep in `city.toml` if they are deployment-owned services. |
| `[agent_defaults]` | Defaults applied to agents in this city | Move to `[agents]` in the root city `pack.toml`. |

## Reference: Gas City 0.13.5 top-level directories to 0.13.6

This is the filesystem companion to the `city.toml` table above.

| Old directory or pattern | What it meant in 0.13.5 | New home or action |
|---|---|---|
| `prompts/` | Shared bucket of prompt templates addressed by path | Move prompt content into `agents/<name>/prompt.md`. |
| `scripts/` | Shared bucket of helper and entrypoint scripts | Do not preserve as a standard top-level directory. Put entrypoint scripts next to what uses them, and put general helpers under `assets/`. |
| `formulas/` | Formula directory, sometimes path-wired via TOML | Keep as the fixed top-level `formulas/` convention. |
| `formulas/orders/` | Nested order definitions under formulas | Move to top-level `orders/` using flat `*.order.toml` files. |
| `orders/` | Top-level order directory in some cities | Standardize on this location, but use flat `orders/<name>.order.toml` files. |
| `overlays/` | Pack-wide overlay bucket | Keep as top-level `overlays/`. |
| `overlay/` | Singular overlay directory seen in some older packs | Remove or migrate to `overlays/` or `agents/<name>/overlay/`. |
| `namepools/` | Shared bucket of agent name pools | Move toward agent-local files if retained. |
| `commands/` with ad hoc scripts | Command helper directory plus TOML wiring | Keep `commands/`, but organize as entry directories such as `commands/<name>/run.sh`. |
| `doctor/` with ad hoc scripts | Doctor helper directory plus TOML wiring | Keep `doctor/`, but organize as entry directories such as `doctor/<name>/run.sh`. |
| `skills/` | Pack-wide skills directory in newer layouts | Keep as top-level `skills/`. |
| `mcp/` | Pack-wide MCP directory in newer layouts | Keep as top-level `mcp/`. |
| `template-fragments/` | Shared prompt-fragment directory in newer layouts | Keep as top-level `template-fragments/`. |
| `packs/` | Local vendored packs or bootstrap imports | Do not treat as a standard top-level directory. If you need opaque embedded packs, place them under `assets/` and import them explicitly. |
| loose helper files at pack root | Arbitrary files mixed into controlled surface area | Move them under `assets/`. |

## Reference: Gas City 0.13.5 `pack.toml` elements to 0.13.6

This is the compact lookup table for migrating old shareable packs.

| 0.13.5 element | What it did | New home or action |
|---|---|---|
| `[pack]` | Pack metadata | Keep in `pack.toml`. |
| `pack.name` | Pack identity | Keep in `[pack]`. |
| `pack.version` | Pack version | Keep in `[pack]`. |
| `pack.schema` | Pack schema version | Update to `schema = 2` in 0.13.6. |
| `pack.requires_gc` | Minimum supported gc version | Keep in `[pack]`. |
| `pack.includes` | Pack-to-pack composition | Replace with `[imports.*]` in `pack.toml`. |
| `[imports.*]` | Named imports in transitional configs | Keep in `pack.toml`. This is the new composition surface. |
| `[[agent]]` | Inline pack agent definitions | Move to `agents/<name>/`, with optional `agent.toml`. |
| `agent.prompt_template` | Agent prompt file path | Move to `agents/<name>/prompt.md`. |
| `agent.overlay_dir` | Agent overlay path | Move content to `agents/<name>/overlay/` or `overlays/`. |
| `agent.session_setup_script` | Agent setup script path | Keep as a path-valued field pointing at a pack-local file. |
| `[[named_session]]` | Pack-defined named sessions | Keep in `pack.toml`. |
| `[providers.*]` | Provider presets used by the pack | Keep in `pack.toml`. |
| `[formulas]` | Formula directory config | Remove. Formula location is now the fixed top-level `formulas/` convention. |
| `formulas.dir` | Formula directory path | Replace with the fixed top-level `formulas/` convention. |
| `[patches]` | Pack-level patching rules | Keep in `pack.toml`. |
| `[[doctor]]` | Pack doctor inventory | Move toward `doctor/<name>/run.sh` by default, with optional `doctor.toml` when needed. |
| `doctor.script` | Path to doctor entrypoint | Keep as a pack-local path, usually `doctor/<name>/run.sh`. |
| `[[commands]]` | Pack command inventory | Move toward `commands/<name>/run.sh` by default, with optional `command.toml` when needed. |
| `commands.script` | Path to command entrypoint | Keep as a pack-local path, usually `commands/<name>/run.sh`. |
| `[agents]` | Pack-wide agent defaults in transitional configs | Keep in `pack.toml`. |

## Suggested migration order

For a real city or pack, the most practical order is:

1. add a root `pack.toml`
2. move `workspace.includes` and `rigs.includes` to imports
3. move agent definitions into `agents/`
4. move orders to top-level flat files
5. move commands and doctor checks into `commands/` and `doctor/`
6. move opaque helpers into `assets/`
7. clean up whatever remains in `city.toml` and `pack.toml` using the reference tables above

That gets the big structural changes done before you spend time on the
smaller cleanup work.
