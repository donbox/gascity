---
title: "Migrating to Pack/City v.next"
description: How to move an existing Gas City city or pack to the new pack/city schema and directory conventions.
---

This guide walks through the current direction of the Pack/City v.next
design and how to migrate existing Gas City products toward it.

It is intentionally practical. The goal is to help you understand:

- what changed
- what moves where
- how to restructure a city or pack
- how commands, doctor checks, assets, and orders fit the new model

## Before you start

Pack/City v.next is a design direction that changes both schema and
filesystem structure. The important mental shift is:

- **V1** centers `city.toml` and uses explicit path wiring
- **V2** centers `pack.toml`, convention-based directories, and named imports

The new direction separates three concerns:

- **pack definition**
  - portable, shareable definition of agents, formulas, commands, doctor
    checks, overlays, and related assets
- **city deployment**
  - team-shared decisions about rigs, capacity, and runtime policy
- **site binding**
  - machine-local state and runtime data in `.gc/`

## What changed

At a high level, V2 does four things:

1. **A city becomes a pack.**
   - The root definition lives in `pack.toml`.
   - `city.toml` becomes deployment-only.

2. **Imports replace includes.**
   - Packs compose through named bindings under `[imports.*]`.
   - The binding name becomes the durable consumption identity.

3. **Convention replaces path wiring.**
   - Standard directories such as `agents/`, `formulas/`, `commands/`,
     `doctor/`, `overlays/`, and `assets/` define structure by existing.

4. **The pack root becomes more intentional.**
   - Standard top-level directories are recognized.
   - Arbitrary pack-owned files live under `assets/`.

## The new shape

### City

```text
my-city/
├── pack.toml
├── city.toml
├── .gc/
├── agents/
├── formulas/
├── orders/
├── commands/
├── doctor/
├── overlays/
├── skills/
├── mcp/
├── template-fragments/
└── assets/
```

### Pack

```text
my-pack/
├── pack.toml
├── agents/
├── formulas/
├── orders/
├── commands/
├── doctor/
├── overlays/
├── skills/
├── mcp/
├── template-fragments/
└── assets/
```

The difference is that only a city has:

- `city.toml`
- `.gc/`

Everything else should work the same way for the root city pack and for
ordinary imported packs.

## Migration checklist

If you want the short version first, use this checklist:

1. Create or promote a root `pack.toml`.
2. Move portable definition out of `city.toml` and into pack-owned files.
3. Leave only deployment decisions in `city.toml`.
4. Replace pack includes with `[imports.*]`.
5. Move agent definitions toward `agents/<name>/`.
6. Move command and doctor entrypoints into `commands/` and `doctor/`.
7. Move arbitrary helper files into `assets/`.
8. Move orders toward top-level `orders/` flat files.
9. Check every path-valued field and make sure it resolves inside the pack.
10. Treat `.gc/` as site binding and runtime state, not portable definition.

## Migrating a city

### Step 1: separate definition from deployment

In V1, `city.toml` tends to do too much. In V2:

- `pack.toml` answers: "what is this city?"
- `city.toml` answers: "how is this city deployed?"

Move pack-owned definition into `pack.toml` and pack directories:

- imports
- providers
- agent defaults
- named sessions
- patches
- pack-wide assets

Leave deployment in `city.toml`:

- rigs
- capacity
- deployment-specific policy

Move machine-local state to `.gc/`:

- path bindings
- prefixes
- caches
- runtime state

### Step 2: create standard directories

Start giving the root city pack the same structure as an imported pack.

At minimum, create the standard directories you actually use:

- `agents/`
- `formulas/`
- `orders/`
- `commands/`
- `doctor/`
- `overlays/`
- `assets/`

Do not create `scripts/` as a new long-term top-level convention. If a
file is just an opaque helper, it belongs in `assets/` unless it lives
next to the thing that uses it.

### Step 3: replace includes with imports

V1 composition flattens packs by include order. V2 makes composition
explicit and named.

Instead of wiring packs only by include lists, move toward:

```toml
[imports.gastown]
source = "https://github.com/example/gastown"
version = "^1.2"
```

The important effect is that the binding name, here `gastown`, becomes
the durable local identity for imported content.

## Migrating a pack

### Step 1: narrow `pack.toml`

`pack.toml` should become more declarative and less like a file index.

It should contain pack-wide configuration such as:

- `[pack]` metadata
- `[imports.*]`
- `[providers.*]`
- `[agents]` defaults
- `[[named_session]]`
- pack-level patches

It should trend away from holding:

- individual agent definitions
- prompt file paths
- overlay directory wiring
- script directory wiring
- simple command inventories
- simple doctor inventories

### Step 2: move agent definitions to directories

Instead of defining every agent inline in `pack.toml`, move toward:

```text
agents/
├── mayor/
│   ├── prompt.md
│   └── agent.toml
└── polecat/
    └── prompt.md
```

Use `agent.toml` only when an agent needs overrides beyond pack-level
defaults.

### Step 3: move opaque helpers into `assets/`

The positive rule in V2 is:

- if it is not a standard discovered surface, it belongs in `assets/`

Examples:

- helper scripts
- static data files
- test fixtures
- imported pack payloads carried inside another pack

Any field that accepts a path may point anywhere inside the same pack,
including `assets/`.

## Migrating commands

### The direction

Commands are moving toward convention-first entry directories.

Simple case:

```text
commands/
└── status/
    └── run.sh
```

This should be enough for a default single-word command.

When you need more shape or metadata:

```text
commands/
└── repo-sync/
    ├── command.toml
    ├── run.sh
    └── help.md
```

Use `command.toml` when you need to make things explicit, for example:

- multi-word command placement
- extension-root placement
- description or richer metadata
- non-default entrypoint

### What changes from V1

V1:

```toml
[[commands]]
name = "status"
description = "Show status"
script = "commands/status.sh"
```

V2 simple case:

```text
commands/status/run.sh
```

V2 richer case:

```toml
# commands/repo-sync/command.toml
command = ["repo", "sync"]
description = "Sync repository state"
```

with:

```text
commands/repo-sync/run.sh
commands/repo-sync/help.md
```

## Migrating doctor checks

Doctor checks are moving in parallel with commands.

Simple case:

```text
doctor/
└── binaries/
    └── run.sh
```

Richer case:

```text
doctor/
└── git-clean/
    ├── doctor.toml
    ├── run.sh
    └── help.md
```

Use local TOML only when the default directory-name mapping is not
enough.

The important shift is the same as commands:

- keep the entrypoint local to the thing it belongs to
- do not depend on a special top-level `scripts/` directory

## Migrating paths and assets

### The new rule

Path-valued fields should be able to point anywhere inside the same
pack.

That includes:

- files under standard directories
- files under `assets/`
- relative paths that use `..`

The hard constraint is:

- after normalization, the path must still stay inside the pack root

### Good examples

```toml
run = "./run.sh"
help = "./help.md"
run = "../shared/run.sh"
source = "./assets/imports/maintenance"
```

if they stay inside the pack.

### Bad examples

```toml
run = "../../../outside.sh"
run = "/tmp/outside-pack.sh"
```

if they escape the pack boundary.

## Migrating orders

Yes: the current local design direction is that orders should move to
look more like formulas.

The strongest current guidance is:

- move orders out of `formulas/orders/`
- standardize on top-level `orders/`
- use flat files such as `orders/<name>.order.toml`

This is a better fit for what orders are:

- orders are not formulas
- orders reference formulas
- orders are scheduled or gated dispatch definitions
- formulas define workflow structure

### Old shape

```text
formulas/
└── orders/
    └── nightly-sync/
        └── order.toml
```

### New direction

```text
orders/
└── nightly-sync.order.toml
```

This gives orders a file model that better matches formulas:

- `formulas/<name>.formula.toml`
- `orders/<name>.order.toml`

It also removes the awkwardness of treating an order as a directory that
contains only one file.

## Common migration gotchas

### "I still have a lot in `city.toml`"

That usually means definition and deployment are still mixed together.

Ask:

- is this portable definition?
- is this team-shared deployment?
- is this machine-local state?

Then move it to:

- `pack.toml` and pack directories
- `city.toml`
- `.gc/`

respectively.

### "I used to rely on `scripts/`"

Do not recreate `scripts/` as a top-level convention just because V1 had
it.

Instead:

- put entrypoint scripts next to the command/doctor entry that uses them
- put general helpers under `assets/`

### "My pack depends on another pack I carry locally"

Put the embedded pack under `assets/` and import it explicitly by
relative path.

Example:

```toml
[imports.maintenance]
source = "./assets/imports/maintenance"
```

### "Do I need TOML everywhere?"

No. That is one of the design tests for the new model.

Simple cases should work by convention:

- `agents/<name>/prompt.md`
- `commands/<name>/run.sh`
- `doctor/<name>/run.sh`

TOML should appear when it is actually needed for:

- defaults
- overrides
- metadata
- explicit placement
- compatibility or policy

## Reference: current to new

This section is the quick lookup table for converting existing content.

### Root files

| Current | New direction |
|---|---|
| `city.toml` holds almost everything | Split into `pack.toml` + `city.toml` + `.gc/` |
| `pack.toml` acts as both metadata and inventory | `pack.toml` becomes pack-wide declarative policy |

### TOML elements

| Current element | New direction |
|---|---|
| `[[agent]]` in `pack.toml` or `city.toml` | `agents/<name>/` with optional `agent.toml` |
| `prompt_template = "..."` | `agents/<name>/prompt.md` |
| `overlay_dir = "..."` | `overlays/` or `agents/<name>/overlay/` |
| `scripts_dir = "scripts"` | no standard `scripts/`; use colocated scripts or `assets/` |
| `[[commands]]` | `commands/<name>/run.sh` by default, optional `command.toml` |
| `[[doctor]]` | `doctor/<name>/run.sh` by default, optional `doctor.toml` |
| `workspace.includes` / `rig.includes` | `[imports.*]` model |
| `[formulas].dir` | fixed `formulas/` convention |
| `formulas/orders/<name>/order.toml` | `orders/<name>.order.toml` |

### Directory structure

| Current | New direction |
|---|---|
| `prompts/` as a global bucket | prompts live with the agent that owns them |
| `scripts/` as a global bucket | use colocated entrypoint scripts or `assets/` |
| `formulas/` | stays, but as a stronger fixed convention |
| `formulas/orders/` | move to top-level `orders/` |
| `packs/` as a root habit | use `assets/` for opaque local payloads |
| root-level loose files | put opaque pack-owned files under `assets/` |

### `pack.toml` contents

| Keep in `pack.toml` | Move out of `pack.toml` |
|---|---|
| `[pack]` metadata | individual agent definitions |
| `[imports.*]` | prompt paths |
| `[providers.*]` | overlay path wiring |
| `[agents]` defaults | script directory wiring |
| `[[named_session]]` | simple command inventory |
| patches and pack-wide policy | simple doctor inventory |

## Next step

If you are migrating a real city or pack, the simplest practical order is:

1. create `pack.toml`
2. split deployment into `city.toml`
3. move agents into `agents/`
4. move commands and doctor checks into `commands/` and `doctor/`
5. move helpers into `assets/`
6. move orders toward top-level flat files
7. replace includes with imports

That gets you most of the conceptual migration before you worry about
polishing every edge case.
