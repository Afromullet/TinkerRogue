# TinkerRogue — Architectural Layers

**Last Updated:** 2026-04-21

A high-level guide to the layered architecture of TinkerRogue. Each layer depends only on layers below it. There are no upward imports and no import cycles across the entire codebase.

---

## Layer Map

```
Layer 6 — Bootstrap/Entry          setup/gamesetup, setup/savesystem, game_main
              │
Layer 5 — Presentation             gui/guicombat, gui/guioverworld, gui/guisquads,
                                   gui/guiraid, gui/guiprogression, gui/guiexploration,
                                   gui/guinodeplacement, gui/guiartifacts, gui/guispells,
                                   gui/guiinspect, gui/guiunitview, gui/guistartmenu,
                                   gui/framework, gui/builders, gui/specs,
                                   gui/widgets, gui/widgetresources
              │
Layer 4 — AI & Orchestration       mind/ai, mind/behavior, mind/evaluation,
                                   mind/encounter, mind/spawning, mind/combatlifecycle
              │
Layer 3 — Game Systems             tactical/squads/*, tactical/commander,
                                   tactical/combat/*, tactical/powers/*,
                                   campaign/overworld/*, campaign/raid,
                                   world/worldmapcore, world/worldgen,
                                   world/garrisongen,
                                   visual/rendering, visual/maprender,
                                   visual/combatrender, visual/vfx
              │
Layer 2 — Core Infrastructure      core/common, visual/graphics, templates, input
              │
Layer 1 — Primitives               core/coords
              │
Layer 0 — Config                   core/config
```

---

## Layer 0 — Config

**Packages:** `core/config`

The absolute foundation. `core/config` is a pure leaf with no internal imports of its own. It holds game-wide constants, asset-path helpers, and feature flags (including `DEBUG_MODE`) that every other layer may need to read. Because nothing in the project imports it circularly, it is the safest possible place to put values that must be visible everywhere.

---

## Layer 1 — Primitives

**Packages:** `core/coords`

A single package that owns the canonical representation of positions in the game world. It defines `LogicalPosition`, the `CoordinateManager` singleton (which handles conversions between logical tile coordinates and flat array indices), and the rules for how space is addressed throughout the engine.

Because coordinate math is used by almost every system that touches the map — ECS, rendering, AI pathfinding, worldmap generation — keeping it in its own layer with no game-logic dependencies prevents any circular dependency from forming. `core/coords` imports only `core/config`.

---

## Layer 2 — Core Infrastructure

**Packages:** `core/common`, `visual/graphics`, `templates`, `input`

This layer provides the shared infrastructure that all game systems are built on.

### core/common

The ECS foundation. `core/common` provides the `EntityManager`, component registration helpers (`RegisterSubsystem`, `InitializeSubsystems`), the `GlobalPositionSystem` spatial grid, and the component access utilities (`GetComponentType`, `GetComponentTypeByID`) that every package in the codebase uses. It depends only on `core/config` and `core/coords`, which is why it can be imported by every package above it.

`core/common` is also where universally shared components live — `PositionComponent`, `NameComponent`, `AttributeComponent`, `RenderableComponent`, `PlayerData`, `ResourceStockpile`, and the random number generator. The `Renderable` and `RenderablesTag` types live here (not in `visual/rendering`) so that non-visual packages can read and set sprites without pulling in the rendering stack.

### visual/graphics

Low-level sprite and batch-drawing abstractions. Knows nothing about game logic, factions, units, or combat — it renders geometry and sprites, nothing more.

### templates

A read-only data library. Templates store the static definitions for unit archetypes, spells, artifacts, encounters, overworld nodes, and tuning configs, loaded from the JSON files in `assets/gamedata/`. Other systems read from templates to instantiate entities, which keeps authored content separated from runtime state. `templates` also owns the name generator and the `GameConfig` global.

### input

Camera controller infrastructure. Currently contains only the camera controller — a minimal package that handles viewport panning and zoom without knowledge of game logic.

---

## Layer 3 — Game Systems

**Packages:** `tactical/*`, `campaign/overworld/*`, `campaign/raid`, `world/worldmapcore`, `world/worldgen`, `world/garrisongen`, `visual/rendering`, `visual/maprender`, `visual/combatrender`, `visual/vfx`

Layer 3 is where the actual game rules live. It is divided into four broad sub-domains: the **tactical** domain (squad-based combat, powers, artifacts, perks, progression), the **campaign** domain (overworld strategic map and raid dungeons), the **world generation** domain, and the **visual** rendering domain.

### Tactical sub-domain

The tactical packages implement everything that happens on the battle map. The domain is organized into four major package groups: **squads**, **combat**, **powers**, and **commander**.

#### Squad packages (`tactical/squads/`)

| Package | Role |
|---|---|
| `tactical/squads/squadcore` | Squad and unit entity definitions, formation data, grid positions, leader abilities, and the squad/unit component registry. The most depended-upon tactical package — the shared data contract for almost everything above it. |
| `tactical/squads/squadcommands` | Command objects that translate player or AI intent into squad and combat mutations (move, add unit, remove unit, change leader, rename). |
| `tactical/squads/squadservices` | Stateless helpers for squad deployment and unit purchasing. |
| `tactical/squads/unitdefs` | Unit template definitions, role enums, attack type enums, and filter helpers — the static data that describes unit archetypes. |
| `tactical/squads/unitprogression` | Unit experience, leveling, and per-stat growth (Fire Emblem-style growth grades). |
| `tactical/squads/roster` | Unit and squad roster management — the persistent collections of available units (attached to the Player entity) and squads (attached to Commander entities) outside of combat. |

#### Combat packages (`tactical/combat/`)

| Package | Role |
|---|---|
| `tactical/combat/combatcore` | Combat rules: turn order, attack resolution, movement validation, action state tracking. Depends on squads to read unit state. Owns the combat balance config. |
| `tactical/combat/combatstate` | Lightweight state queries (e.g., `IsCombatActive`, `RemoveSquadFromMap`) shared across the combat subsystem. |
| `tactical/combat/combatmath` | Pure math helpers for combat calculations. |
| `tactical/combat/combattypes` | Shared combat enums and small types. |
| `tactical/combat/battlelog` | Structured combat narration / battlelog for the GUI. |
| `tactical/combat/combatservices` | The combat orchestration hub. Coordinates artifacts, combat, effects, perks, and the AI lifecycle in a single service layer. Uses interface injection to accept an AI controller and a threat provider without importing those packages directly. |

`tactical/combat/combatservices` is intentionally the highest-coupling package in this domain. It acts as the single point of coordination for a combat encounter — managing state, applying effects, triggering artifacts and perks, and notifying listeners. GUI packages register callbacks here rather than calling combat internals directly.

#### Powers packages (`tactical/powers/`)

| Package | Role |
|---|---|
| `tactical/powers/powercore` | Shared pipeline infrastructure for the power system: `PowerContext`, the `Pipeline` executor, and a structured `Logger`. All power types (spells, artifacts, perks) flow through this to coordinate damage modifiers, hit/cover/crit adjustments, and battle logging. |
| `tactical/powers/effects` | Status effect definitions (buffs, debuffs, damage-over-time). Pure data, depends only on `core/common`. |
| `tactical/powers/spells` | Spell definitions and resolution. Bridges effects, combat, and squads. `ManaComponent` and `SpellBookComponent` are attached to **squad** entities (scoped to the squad leader's unit type) and filtered against the player's unlocked-spell library. |
| `tactical/powers/artifacts` | The artifact item system: inventory management, charge tracking, passive and activated behaviors, pending-effects queue, and the behavior dispatcher. Owns the artifact balance config. |
| `tactical/powers/perks` | Squad-level perk system. Perks are equipped into `PerkSlotComponent` on the squad entity; `PerkRoundStateComponent` is added for the duration of a combat. Includes the perk registry, balance config, dispatcher, and the hook tables that plug into combat events. |
| `tactical/powers/progression` | Player-level progression state: arcana/skill points and unlocked perk/spell libraries. `ProgressionComponent` is attached to the Player entity and consulted when generating spell lists and offering perks. |

#### Commander

| Package | Role |
|---|---|
| `tactical/commander` | Field-commander entity, overworld movement, per-turn action state, and the overworld turn-state singleton. Owns the `CommanderRosterComponent` that lives on the Player entity. |

### Campaign sub-domain

The campaign layer models long-running play outside of individual combats: the strategic overworld map, and garrison-raid dungeons.

#### Overworld packages (`campaign/overworld/`)

| Package | Role |
|---|---|
| `campaign/overworld/core` | Local foundation for the overworld. Defines node/faction/interaction/influence/victory/tick/encounter components, overworld event types, walkability rules, resource accessors, and the node-definition registry. All other overworld packages build on `core` plus `core/common`. |
| `campaign/overworld/node` | Node creation, queries, and validation. |
| `campaign/overworld/faction` | AI-controlled factions, including the scoring/archetype logic that decides when they expand, raid, or fortify. |
| `campaign/overworld/garrison` | Squad garrisoning at player-owned nodes; the bridge into the tactical domain. |
| `campaign/overworld/influence` | Propagation of node influence, synergy/competition/suppression interactions between nodes. |
| `campaign/overworld/threat` | Threat-node behaviors and intensity tracking. |
| `campaign/overworld/victory` | Victory/defeat condition evaluation. |
| `campaign/overworld/tick` | The overworld's internal orchestrator, stepping faction, influence, threat, and victory updates each game tick. |
| `campaign/overworld/overworldlog` | Structured recording, export, and summarization of overworld events for debugging and game history. |

The only place the overworld domain reaches into the tactical domain is `campaign/overworld/garrison`, which imports `tactical/squads` to station squads at nodes.

#### Raid package (`campaign/raid`)

A single-package raid system that drives multi-floor garrison dungeons: floor-graph DAG construction, archetype-driven garrison composition, alert-level progression, deployment/reserve management, resolvers, recovery, and rewards. `campaign/raid` owns its own JSON loaders (`LoadRaidConfig`, `LoadArchetypeData`) that are called from bootstrap.

### World generation

The world generation domain is split into three packages:

- `world/worldmapcore` — The shared data contract for map generation. Defines the `MapGenerator` interface, `GenerationResult`, `Tile`, `Biome`, and map utility types. This is what the visual layer and other consumers import to work with generated maps. It depends only on `core/coords`.
- `world/worldgen` — The generation algorithm implementations (cavern, rooms & corridors, overworld) and the generator registry. Generators register themselves via `init()`, so new algorithms can be added without modifying any central wiring file.
- `world/garrisongen` — Garrison-specific map generation. Builds multi-floor garrison layouts using DAG-based room connectivity and terrain placement. Depends on `worldmapcore`, `worldgen`, and `core/common`.

This three-package split separates the stable interface (`worldmapcore`) from the algorithm implementations (`worldgen`, `garrisongen`), allowing consumers like `visual/maprender` and `campaign/raid` to depend on the core types without pulling in generation code.

### Visual domain

The rendering stack sits at Layer 3 because it needs to read game state (squad, unit, tile, combat) but it does not mutate game rules.

| Package | Role |
|---|---|
| `visual/rendering` | The generic renderable pipeline: viewport-aware sprite batching, quad batching, and rendering caches. Builds on `visual/graphics`, `core/common` (for `RenderableComponent`), and `world/worldmapcore`. |
| `visual/maprender` | Map- and tile-specific rendering on top of `visual/rendering`. |
| `visual/combatrender` | Squad/unit renderers, combat overlays, and targeting highlights specific to the battle map. |
| `visual/vfx` | Visual effects pipeline — `vx` handler, factory, animators, and renderers used by spell casts and artifact activations. |

---

## Layer 4 — AI and Orchestration

**Packages:** `mind/evaluation`, `mind/behavior`, `mind/ai`, `mind/combatlifecycle`, `mind/encounter`, `mind/spawning`

Layer 4 sits above all game rules and is responsible for making decisions about them. It has three distinct roles: running enemy AI during tactical combat, orchestrating how encounters and squads are constructed, and handling the combat lifecycle.

### AI pipeline

The three AI packages form a clean internal stack:

- `mind/evaluation` — Scores units and squads using power formulas. Reads squad data and templates, produces numeric assessments. No combat awareness.
- `mind/behavior` — Builds on evaluation to produce behavioral decisions: positioning weights, target selection, action priorities, threat layers. Reads combat state to make context-aware choices.
- `mind/ai` — The top of the stack. Consumes behavior outputs and issues commands by calling into `tactical/combat/combatservices` and `tactical/squads/squadcommands`. It does not implement combat logic — it drives it.

`mind/ai` does not import `tactical/combat/combatservices` through a concrete type; instead, `combatservices` exposes an `AITurnController` interface that `mind/ai` satisfies. This breaks what would otherwise be a mutual dependency.

### Combat lifecycle

`mind/combatlifecycle` is a coordinator for the setup and teardown sequence around a combat encounter — enrollment, pre-combat spell pipeline, casualty processing, reward distribution, and post-combat cleanup (including tearing down `PerkRoundStateComponent`). It defines its own contract interfaces so that `combatservices` and `encounter` can plug in without cycles.

### Encounter and spawning

- `mind/encounter` — The primary bridge between the campaign and tactical domains. When a player triggers a combat encounter from the overworld map, `encounter` reads overworld state (node ownership, garrison composition, threat level) and constructs the corresponding tactical setup (squads, positions, combat initialization). Owns encounter resolvers, rewards, starters, and validators.
- `mind/spawning` — Squad creation and composition helpers: given an archetype, difficulty, and faction, produce a concrete squad of units ready to be added to combat.

`campaign/raid` extends encounter by chaining multiple encounters into a raid sequence, adding garrison map generation and evaluation-based scaling on top of encounter's foundation.

---

## Layer 5 — Presentation

**Packages:** `gui/specs`, `gui/widgets`, `gui/widgetresources`, `gui/builders`, `gui/framework`, `gui/guicombat`, `gui/guioverworld`, `gui/guisquads`, `gui/guiraid`, `gui/guiprogression`, `gui/guiexploration`, `gui/guinodeplacement`, `gui/guiartifacts`, `gui/guispells`, `gui/guiinspect`, `gui/guiunitview`, `gui/guistartmenu`

The presentation layer translates game state into a user interface and translates user input into game commands. It reads broadly from layers below but does not own any game logic.

### GUI primitives

- `gui/specs` — Data structures describing UI layout and configuration.
- `gui/widgets` — Reusable widget components (cached text areas, cached lists, text displays).
- `gui/widgetresources` — Asset references, font/background caches, and resource constants.

These three packages form the vocabulary used by all GUI mode packages.

### GUI infrastructure

- `gui/builders` — Factory functions for assembling common UI constructs (unit lists, panel layouts, dialogs). Imports `tactical/squads` directly for typed list entries.
- `gui/framework` — The local foundation for all GUI modes. Provides the mode manager, panel registry, context switching, shared state containers (`BattleMapState`, `OverworldState`), `ActionMap` semantic input bindings, `GUIQueries*` state readers, and the command-history infrastructure. All GUI mode packages import `gui/framework`.

### GUI mode packages

Each mode package drives one screen or sub-screen of the game. They are leaves in the dependency graph — nothing imports them except `setup/gamesetup`.

| Mode package | Scope |
|---|---|
| `gui/guicombat` | The battle map screen: combat animation, attack/move input, spell casting, artifact activation, threat visualization, and the battlelog display. |
| `gui/guioverworld` | The strategic map screen: overworld rendering, action/input handlers, formatters, and panel registry. |
| `gui/guisquads` | Squad editing, unit purchasing, squad deployment, artifact assignment, and perk equipping. Contains `squadeditor_*` modes covering grid editing, roster, perks, and refresh. |
| `gui/guiraid` | Raid setup and progression: deploy panel, floor-map panel and renderer, raid state, summary panel. |
| `gui/guiprogression` | Progression mode — panel registry, controller, and refresh logic for spending arcana/skill points. |
| `gui/guiexploration` | Exploration mode UI. |
| `gui/guinodeplacement` | Node placement on the campaign map. |
| `gui/guiartifacts`, `gui/guispells`, `gui/guiinspect`, `gui/guiunitview`, `gui/guistartmenu` | Focused sub-screens, each handling a single UI concern. |

GUI modes do not call game logic directly. They call into service packages (`tactical/combat/combatservices`, `tactical/squads/squadcommands`, `mind/encounter`) and register callbacks to receive state-change notifications. The dependency arrows always point downward.

---

## Layer 6 — Bootstrap and Entry

**Packages:** `setup/gamesetup`, `setup/savesystem`, `game_main`

The topmost layer exists to assemble everything else. No game logic belongs here — only wiring.

`setup/gamesetup` is the composition root. It initializes the `EntityManager`, calls `common.InitializeSubsystems()` to trigger all `init()`-registered ECS subsystems, loads JSON game data via `templates.ReadGameData()` (and raid/perk/artifact balance loaders), wires GUI modes to the framework, seeds overworld and tactical state, and hands the assembled game off to the Ebiten loop. It imports broadly because it must know about everything — that breadth is intentional and correct, so no other package needs to know about siblings.

`setup/savesystem` handles game persistence. It owns a per-subsystem chunks/ directory where each chunk knows how to serialize and deserialize one slice of state (player, squads, progression, overworld, etc.). It sits at the bootstrap layer because it must know about all game systems.

`game_main` is the executable entry point. It owns the Ebiten window configuration and calls into `setup/gamesetup`. It also loads the raid-config and raid-archetype JSON files before the main game loop starts.

---

## Cross-Cutting Patterns

Three patterns allow the strict layering to remain acyclic even where natural bidirectional relationships exist.

**Interface injection.** When a lower-layer package needs to call into a higher-layer package, an interface is defined in the lower layer instead. `tactical/combat/combatservices` defines `AITurnController` and `ThreatProvider` interfaces; `mind/ai` and the overworld threat system satisfy them at runtime. `mind/encounter` defines a `CombatTransitionHandler` interface for GUI mode switching. `mind/combatlifecycle` defines its own contracts for the pieces it drives. No upward import is needed.

**Callback registration.** `tactical/combat/combatservices` exposes event hooks (`OnAttackComplete`, `OnMoveComplete`, `OnTurnEnd`). GUI modes register closures at initialization time and react to combat events without `combatservices` ever importing GUI packages. The perk dispatcher uses the same hooks to drive combat-reactive perks.

**Self-registration via `init()`.** Worldmap generators, perks, and ECS subsystems register themselves by calling `worldgen.RegisterGenerator`, `perks.RegisterBehavior`, or `common.RegisterSubsystem` inside `init()` functions. `setup/gamesetup` triggers these registrations simply by importing the relevant packages. New generators, perks, or subsystems can be added without modifying any central coordinator.

---

## Quick Reference: Package Fan-In

The table below highlights the most depended-upon packages. High fan-in means a change to that package's API ripples widely.

| Package | Why fan-in is high |
|---|---|
| `core/common` | ECS foundation — every package needs component access, attributes, position, renderable |
| `core/coords` | Coordinate math is needed wherever tiles are addressed |
| `tactical/squads/squadcore` | Shared data contract for units across combat, AI, GUI, artifacts, perks, save |
| `tactical/combat/combatcore` | Combat state read by AI, GUI, artifacts, perks, encounter |
| `campaign/overworld/core` | Local foundation for all overworld sub-packages |
| `gui/framework` | Local foundation for all GUI mode packages |
| `templates` | Every JSON-defined entity (units, spells, artifacts, encounters) is instantiated through it |
