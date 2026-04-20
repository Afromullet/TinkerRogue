# TinkerRogue — Architectural Layers

**Last Updated:** 2026-04-10

A high-level guide to the layered architecture of TinkerRogue. Each layer depends only on layers below it. There are no upward imports and no import cycles across the entire codebase.

---

## Layer Map

```
Layer 6 — Bootstrap/Entry          setup/gamesetup, game_main
              │
Layer 5 — Presentation             gui/guicombat, gui/guioverworld, gui/guisquads,
                                   gui/guiraid, gui/framework, gui/builders,
                                   gui/specs, gui/widgets, gui/widgetresources, ...
              │
Layer 4 — AI & Orchestration       mind/ai, mind/behavior, mind/evaluation
                                   mind/encounter, mind/raid, mind/combatlifecycle
              │
Layer 3 — Game Systems             tactical/squads/*, tactical/combat/*,
                                   tactical/powers/*, tactical/commander,
                                   world/worldmapcore, world/worldgen,
                                   world/garrisongen,
                                   overworld/core, overworld/faction, overworld/garrison,
                                   overworld/influence, overworld/node, overworld/threat,
                                   overworld/tick, overworld/victory, overworld/overworldlog
              │
Layer 2 — Core Infrastructure      common, visual/graphics, visual/rendering,
                                   templates, input
              │
Layer 1 — Primitives               world/coords
              │
Layer 0 — Config                   setup/config
```

---

## Layer 0 — Config

**Packages:** `setup/config`

The absolute foundation. `setup/config` is a pure leaf with no internal imports of its own. It holds game-wide constants and feature flags (including `DEBUG_MODE`) that every other layer may need to read. Because nothing in the project imports it circularly, it is the safest possible place to put values that must be visible everywhere.

Roughly 11 packages import `setup/config` directly. All of them sit at Layer 2 or above.

---

## Layer 1 — Primitives

**Packages:** `world/coords`

A single package that owns the canonical representation of positions in the game world. It defines `LogicalPosition`, the `CoordinateManager` singleton (which handles conversions between logical tile coordinates and flat array indices), and the rules for how space is addressed throughout the engine.

Because coordinate math is used by almost every system that touches the map — ECS, rendering, AI pathfinding, worldmap generation — keeping it in its own layer with no game-logic dependencies prevents any circular dependency from forming. `world/coords` imports only `setup/config` and is used by roughly 21 packages.

---

## Layer 2 — Core Infrastructure

**Packages:** `common`, `visual/graphics`, `visual/rendering`, `templates`, `input`

This layer provides the shared infrastructure that all game systems are built on.

### common

The ECS foundation. `common` provides the `EntityManager`, component registration helpers (`RegisterSubsystem`, `InitializeSubsystems`), the `GlobalPositionSystem` spatial grid, and the component access utilities (`GetComponentType`, `GetComponentTypeByID`) that every package in the codebase uses. It depends only on `setup/config` and `world/coords`, which is why it can safely be imported by all ~37 packages above it.

`common` is also where universally shared components live — `PositionComponent`, `PlayerData`, and the random number generator.

### visual/graphics and visual/rendering

These two packages own the game's drawing pipeline. `visual/graphics` defines the low-level sprite and batch-drawing abstractions. `visual/rendering` sits on top of `visual/graphics` and adds knowledge of world geometry (via `world/worldmapcore`) to drive tile-by-tile rendering. Neither package knows anything about game logic, factions, units, or combat — they render geometry and sprites, nothing more.

### templates

A read-only data library. Templates store the static definitions for unit archetypes, abilities, and similar content. Other systems read from templates to instantiate entities, which keeps authored content separated from runtime state.

### input

Camera controller infrastructure. Currently contains only the camera controller — a minimal package that handles viewport panning and zoom without knowledge of game logic.

---

## Layer 3 — Game Systems

**Packages:** `tactical/*`, `world/worldmapcore`, `world/worldgen`, `world/garrisongen`, `overworld/*`

Layer 3 is where the actual game rules live. It is divided into three broad sub-domains: the **tactical** domain (squad-based combat, powers, and artifacts), the **world generation** domain, and the **overworld** domain (strategic map).

### Tactical sub-domain

The tactical packages implement everything that happens on the battle map. The domain is organized into three major package groups: **combat**, **squads**, and **powers**.

#### Combat packages (`tactical/combat/`)

| Package | Role |
|---|---|
| `tactical/combat/combatcore` | Combat rules: turn order, attack resolution, movement validation. Depends on squads to read unit state. Contains battle logging for structured combat narration. |
| `tactical/combat/combatservices` | The combat orchestration hub. Coordinates artifacts, combat, effects, and the AI lifecycle in a single service layer. Uses interface injection to accept an AI controller and a threat provider without importing those packages directly. |

`tactical/combat/combatservices` is intentionally the highest-coupling package in this domain. It acts as the single point of coordination for a combat encounter — managing state, applying effects, triggering artifacts, and notifying listeners. GUI packages register callbacks here rather than calling combat internals directly.

#### Squad packages (`tactical/squads/`)

| Package | Role |
|---|---|
| `tactical/squads/squadcore` | Squad and unit entity definitions, formation data, action points, roles. The most depended-upon game system (~22 fan-in). It is the shared data contract for almost everything above it. |
| `tactical/squads/squadcommands` | Command objects that translate player or AI intent into squad and combat mutations. |
| `tactical/squads/squadservices` | Stateless helpers for squad deployment and unit purchasing. |
| `tactical/squads/unitdefs` | Unit template definitions — the static data that describes unit archetypes. |
| `tactical/squads/unitprogression` | Unit progression mechanics — experience gain, leveling, stat growth. |
| `tactical/squads/roster` | Squad and unit roster management — the persistent collection of available squads and units outside of combat. |

#### Powers packages (`tactical/powers/`)

| Package | Role |
|---|---|
| `tactical/powers/effects` | Status effect definitions (buffs, debuffs, damage over time). Pure data, depends only on `common`. The cleanest tactical package. |
| `tactical/powers/spells` | Spell definitions and resolution. Bridges effects, combat, and squads. |
| `tactical/powers/artifacts` | The artifact item system. Artifacts can activate behaviors during combat, which requires importing both `tactical/combat` and `tactical/squads` to execute queries against live combat state. Contains inventory management, charge tracking, and both passive and activated behaviors. |
| `tactical/powers/artifacts/effects` | Artifact-specific effects — a dedicated effects sub-package scoped to artifact behaviors, separate from the general status effects in `tactical/powers/effects`. |

#### Commander

| Package | Role |
|---|---|
| `tactical/commander` | Manages the player's commander entity and its interaction with squads and overworld. |

### World generation

The world generation domain is split into three packages:

- `world/worldmapcore` — The shared data contract for map generation. Defines the `MapGenerator` interface, `GenerationResult`, `Tile`, `Biome`, and map utility types. This is what `visual/rendering` and other consumers import to work with generated maps. It depends only on `world/coords`.
- `world/worldgen` — The generation algorithm implementations (cavern, rooms & corridors, overworld) and the generator registry. Generators register themselves via `init()`, so new algorithms can be added without modifying any central wiring file. Depends on `worldmapcore` for shared types.
- `world/garrisongen` — Garrison-specific map generation. Builds multi-floor garrison layouts using DAG-based room connectivity and terrain placement. Depends on `worldmapcore`, `worldgen`, and `common`.

This three-package split separates the stable interface (`worldmapcore`) from the algorithm implementations (`worldgen`, `garrisongen`), allowing consumers like `visual/rendering` and `mind/raid` to depend on the core types without pulling in generation code.

### Overworld sub-domain

The overworld models the strategic layer of the game: nodes on a campaign map, faction control, garrisons, resource influence, and victory conditions.

`overworld/core` is the local foundation for this sub-domain — it defines the fundamental node and resource types. All other overworld packages (`faction`, `garrison`, `influence`, `node`, `threat`, `victory`) build on `core` plus `common`. `overworld/tick` acts as the overworld's internal orchestrator, stepping faction, influence, and threat updates each game tick. `overworld/overworldlog` provides structured recording, export, and summarization of overworld events for debugging and game history.

The only place the overworld domain reaches into the tactical domain is `overworld/garrison`, which imports `tactical/squads` to station squads at nodes. This single crossing point makes the overworld the best-isolated domain in the codebase.

---

## Layer 4 — AI and Orchestration

**Packages:** `mind/evaluation`, `mind/behavior`, `mind/ai`, `mind/combatlifecycle`, `mind/encounter`, `mind/raid`

Layer 4 sits above all game rules and is responsible for making decisions about them. It has two distinct roles: running enemy AI during tactical combat, and orchestrating how encounters and raids are constructed.

### AI pipeline

The three AI packages form a clean internal stack:

- `mind/evaluation` — Scores units and squads using power formulas. Reads squad data and templates, produces numeric assessments. No combat awareness.
- `mind/behavior` — Builds on evaluation to produce behavioral decisions: positioning weights, target selection, action priorities. Reads combat state to make context-aware choices.
- `mind/ai` — The top of the stack. Consumes behavior outputs and issues commands by calling into `tactical/combat/combatservices` and `tactical/squads/squadcommands`. It does not implement combat logic — it drives it.

`mind/ai` does not import `tactical/combat/combatservices` directly through its type; instead, `combatservices` exposes an `AITurnController` interface that `mind/ai` satisfies. This breaks what would otherwise be a mutual dependency.

### Combat lifecycle

`mind/combatlifecycle` is a thin coordinator that handles the setup and teardown sequence for a combat encounter — initializing turn order, applying pre-combat spells, and cleaning up after resolution.

### Encounter and raid bridge

`mind/encounter` is the primary bridge between the overworld domain and the tactical domain. When a player triggers a combat encounter from the overworld map, `encounter` reads overworld state (node ownership, garrison composition, threat level) and constructs the corresponding tactical setup (squads, positions, combat initialization). It holds 10 internal imports spanning both domains, making it the heaviest cross-domain package in the codebase.

`mind/raid` extends encounter by chaining multiple encounters into a raid sequence, adding worldmap generation and evaluation-based scaling on top of encounter's foundation.

---

## Layer 5 — Presentation

**Packages:** `gui/specs`, `gui/widgets`, `gui/widgetresources`, `gui/builders`, `gui/framework`, `gui/guicombat`, `gui/guioverworld`, `gui/guisquads`, `gui/guiraid`, `gui/guiexploration`, `gui/guinodeplacement`, `gui/guiartifacts`, `gui/guispells`, `gui/guiinspect`, `gui/guiunitview`, `gui/guistartmenu`

The presentation layer translates game state into a user interface and translates user input into game commands. It reads broadly from layers below but does not own any game logic.

### GUI primitives (no internal imports)

- `gui/specs` — Data structures describing UI layout and configuration. Zero internal imports.
- `gui/widgets` — Reusable widget components. Zero internal imports.
- `gui/widgetresources` — Asset references and resource constants. Depends only on `setup/config`.

These three packages form the vocabulary used by all GUI mode packages.

### GUI infrastructure

- `gui/builders` — Factory functions for assembling common UI constructs (unit lists, panel layouts). Imports `tactical/squads` directly for typed list entries.
- `gui/framework` — The local foundation for all GUI modes. Provides the mode manager, context switching, shared state containers (`BattleMapState`, `OverworldState`), and the input coordinator bridge. All GUI mode packages import `gui/framework`.

### GUI mode packages

Each mode package drives one screen or sub-screen of the game. They are leaves in the dependency graph — nothing imports them except `setup/gamesetup`.

| Mode package | Scope |
|---|---|
| `gui/guicombat` | The battle map screen. 25 internal imports — the highest fan-out in the codebase. Covers combat animation, attack and move input, spell casting, artifact activation, and battlelog display. |
| `gui/guioverworld` | The strategic map screen. 17 imports. Bridges overworld node state, encounter triggering, and commander management. |
| `gui/guisquads` | Squad editing and unit purchasing. 16 imports. Covers artifact assignment, squad services, commander interaction, and unit inspection. |
| `gui/guiraid` | Raid setup and progression. 5-10 imports. |
| `gui/guiexploration` | Exploration mode UI. Well-scoped. |
| `gui/guinodeplacement` | Node placement on the campaign map. Single-concern. |
| `gui/guiartifacts`, `gui/guispells`, `gui/guiinspect`, `gui/guiunitview`, `gui/guistartmenu` | Focused sub-screens, each handling a single UI concern at 2-10 imports. |

GUI modes do not call game logic directly. They call into service packages (`tactical/combat/combatservices`, `tactical/squads/squadcommands`, `mind/encounter`) and register callbacks to receive state-change notifications. The dependency arrows always point downward.

---

## Layer 6 — Bootstrap and Entry

**Packages:** `setup/gamesetup`, `setup/savesystem`, `game_main`

The topmost layer exists to assemble everything else. No game logic belongs here — only wiring.

`setup/gamesetup` is the composition root. It initializes the `EntityManager`, calls `common.InitializeSubsystems()` to trigger all `init()`-registered ECS subsystems, wires GUI modes to the framework, seeds overworld and tactical state, and hands the assembled game off to the Ebiten loop. It imports ~28 packages because it must know about everything. That breadth is intentional and correct — concentrated knowledge of wiring here means no other package needs to know about siblings.

`setup/savesystem` handles game persistence — saving and loading game state. It sits at the bootstrap layer because it must know about all game systems in order to serialize and deserialize their state.

`game_main` is the executable entry point. It owns the Ebiten window configuration and calls into `setup/gamesetup`. Its 18 imports reflect the minimal set needed to start the process.

---

## Cross-Cutting Patterns

Three patterns allow the strict layering to remain acyclic even where natural bidirectional relationships exist.

**Interface injection.** When a lower-layer package needs to call into a higher-layer package, an interface is defined in the lower layer instead. `tactical/combat/combatservices` defines `AITurnController` and `ThreatProvider` interfaces; `mind/ai` and the overworld threat system satisfy them at runtime. `mind/encounter` defines a `CombatTransitionHandler` interface for GUI mode switching. No upward import is needed.

**Callback registration.** `tactical/combat/combatservices` exposes event hooks (`OnAttackComplete`, `OnMoveComplete`, `OnTurnEnd`). GUI modes register closures at initialization time and react to combat events without `combatservices` ever importing GUI packages.

**Self-registration via `init()`.** Worldmap generators and ECS subsystems register themselves by calling `worldgen.RegisterGenerator` or `common.RegisterSubsystem` inside `init()` functions. `setup/gamesetup` triggers these registrations simply by importing the relevant packages. New generators or subsystems can be added without modifying any central coordinator.

---

## Quick Reference: Package Fan-In

The table below highlights the most depended-upon packages. High fan-in means a change to that package's API ripples widely.

| Package | Approximate fan-in | Why so high |
|---|---|---|
| `common` | ~37 | ECS foundation — every package needs component access |
| `world/coords` | ~21 | Coordinate math is needed wherever tiles are addressed |
| `tactical/squads/squadcore` | ~22 | Shared data contract for units across combat, AI, GUI, artifacts, save |
| `tactical/combat/combatcore` | ~12 | Combat state read by AI, GUI, artifacts, encounter |
| `overworld/core` | ~11 | Local foundation for all overworld sub-packages |
| `gui/framework` | ~11 | Local foundation for all GUI mode packages |
