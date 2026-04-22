# TinkerRogue Technical Documentation

**Version:** 5.1 (2026 Edition)
**Last Updated:** 2026-04-22
**Project Type:** Turn-based tactical roguelike with squad combat
**Language:** Go 1.x
**ECS Library:** bytearena/ecs
**UI Framework:** Ebitengine v2 + ebitenui

---

## Executive Summary

TinkerRogue is a turn-based tactical roguelike implemented in Go using a pure Entity Component System (ECS) architecture. The game offers two modes: **Overworld Mode**, a strategic layer where player-controlled commanders traverse a procedurally generated map, manage factions, garrison nodes, and trigger tactical encounters against growing threats; and **Roguelike Mode**, a pure tactical dungeon crawl on cavern maps. Both modes share the same squad-based tactical combat system where players control formations of units in a 3x3 grid layout, casting spells and using artifacts to defeat enemy squads on procedurally generated battle maps.

### Key Architectural Decisions

1. **Pure ECS Architecture** — All game logic follows strict ECS patterns with zero logic in components, EntityID-only relationships, and query-based data access.

2. **Global Coordinate Manager** — A single global `core/coords.CoordManager` instance handles all coordinate conversions, preventing index-out-of-bounds errors.

3. **O(1) Spatial Queries** — The `core/common.GlobalPositionSystem` provides constant-time position lookups using value-based map keys, replacing O(n) linear searches (~50x improvement over pointer keys).

4. **Mode-Based GUI** — UI uses a mode manager pattern with two context states (tactical / overworld) and a global `PanelRegistry` that builds per-mode panels declaratively.

5. **Unified Powers Pipeline** — Spells, active effects, artifacts, and perks share a single `tactical/powers/powercore` pipeline (context, logger, dispatcher), making hook and callback behavior uniform across power sources.

6. **Campaign / Tactical Split** — Strategic systems (overworld, raid) live under `campaign/`; in-battle systems live under `tactical/`. The bridge runs through `mind/encounter/` and `mind/combatlifecycle/`.

7. **Template-Based Entity Creation** — Entity spawning uses JSON templates loaded from files, separating data from code.

### Technical Metrics

- **Packages**: 40+ packages across `core/`, `tactical/`, `campaign/`, `mind/`, `world/`, `visual/`, `gui/`, `templates/`, `setup/`
- **Entity Types**: Units, Squads, Commanders, Items, Artifacts, Tiles, Factions, Combat State, Overworld Nodes, Encounters, Raids
- **Performance**:
  - Value-based map keys provide 50x improvement over pointer-based keys
  - Cached list widgets reduce CPU usage by ~90%
  - Batched rendering for tiles and sprites
  - O(1) spatial queries via `GlobalPositionSystem`
  - Cached combat queries (`combatstate/combatqueriescache.go`) and squad info cache (`gui/framework/squadinfo_cache.go`)

---

## Reading Paths

**For New Developers:**
1. Start with [Architecture Overview](#architecture-overview)
2. Read [ECS Best Practices](project_documentation/Process/ECS_BEST_PRACTICES.md)
3. Study [Squad & Combat System](project_documentation/Systems/SQUAD_COMBAT_SYSTEM.md) as reference implementation

**For Gameplay Programmers:**
1. [Squad & Combat System](project_documentation/Systems/SQUAD_COMBAT_SYSTEM.md) — Formation-based combat, turn management
2. [Combat Pipelines](project_documentation/Architecture/COMBAT_PIPELINES.md) — Setup, turn, resolution pipelines
3. [Spells](project_documentation/Systems/SPELLS_DOCUMENTATION.md), [Artifacts](project_documentation/Systems/ARTIFACT_SYSTEM.md), [Perks](project_documentation/Systems/PERK_SYSTEM.md)
4. [Hooks and Callbacks](project_documentation/Architecture/HOOKS_AND_CALLBACKS.md) — How powers subscribe to combat events

**For UI Developers:**
1. [UI Design Principles](project_documentation/Process/UI_DESIGN_PRINCIPLES.md)
2. [GUI Architecture](project_documentation/UI/GUI_DOCUMENTATION.md) — Mode management, PanelRegistry, GUIQueries
3. [Data Flow Patterns](project_documentation/Architecture/DATA_FLOW_PATTERNS.md) — UI ↔ ECS integration
4. [Input Reference](project_documentation/UI/INPUT_REFERENCE.md) — ActionMap and bindings

**For Systems Programmers:**
1. [Project Layout](PROJECT_LAYOUT.md) — Full package tree
2. [Architecture Layers](project_documentation/Architecture/ARCHITECTURE_LAYERS.md) — Dependency rules
3. [Performance Guide](project_documentation/Process/PERFORMANCE_GUIDE.md)
4. [Coords Package](project_documentation/Systems/COORDS_PACKAGE.md)

**For Overworld / Strategic Developers:**
1. [Overworld Architecture](project_documentation/Systems/OVERWORLD_ARCHITECTURE.md) — Tick system, factions, influence
2. [Raids](project_documentation/Systems/RAIDS.md) — Garrison defense encounters
3. [Encounter System](project_documentation/AI/ENCOUNTER_SYSTEM.md) — Triggers, setup, resolution
4. [Config Tuning Guide](project_documentation/AI/CONFIG_TUNING_GUIDE.md) — Balancing overworld parameters

**For AI Developers:**
1. [AI Controller](project_documentation/AI/AI_CONTROLLER.md) — Entry point and orchestration
2. [AI Algorithm Architecture](project_documentation/AI/AI_ALGORITHM_ARCHITECTURE.md) — Utility AI, action evaluation
3. [Behavior & Threat Layers](project_documentation/AI/BEHAVIOR_THREAT_LAYERS.md) — Threat painting, danger maps
4. [Power Evaluation](project_documentation/AI/POWER_EVALUATION.md) — Power scoring, role evaluation
5. [Caching Overview](project_documentation/Architecture/CACHING_OVERVIEW.md)

**For Data / Config Developers:**
1. [Game Data Overview](project_documentation/Architecture/GAMEDATA_OVERVIEW.md) — JSON data files, templates
2. [AI Configuration](project_documentation/AI/AI_CONFIGURATION.md)
3. [Config Tuning Guide](project_documentation/AI/CONFIG_TUNING_GUIDE.md)

**For Progression / Commander Developers:**
1. [Progression](project_documentation/Systems/PROGRESSION.md) — Commander veterancy, perk/spell library
2. [Save System](project_documentation/Systems/SAVE_SYSTEM.md) — Chunk-based serialization

---

## Architecture Overview

### System Diagram

```
┌──────────────────────────────────────────────────────────────────────┐
│                           Start Menu                                  │
│  gui/guistartmenu/ → Overworld Mode OR Roguelike Mode                │
└───────────────┬──────────────────────────────────────────────────────┘
                │
┌───────────────▼──────────────────────────────────────────────────────┐
│                    Game Main Loop (game_main/)                        │
│  game_main/setup.go + setup/gamesetup/bootstrap.go                    │
└───────────────┬──────────────────────────────────────────────────────┘
                │
                ├─ Core / Global Infrastructure (Initialized Once)
                │  ├─ core/coords.CoordManager     (coordinate conversions)
                │  ├─ core/common.GlobalPositionSystem (O(1) spatial queries)
                │  ├─ core/common.EntityManager    (ECS world + tag registry)
                │  └─ core/config                  (game constants, debug flags)
                │
                ├─ Tactical Systems (In-Battle)
                │  ├─ tactical/squads/             (squadcore, squadservices, squadcommands,
                │  │                                roster, unitdefs, unitprogression)
                │  ├─ tactical/combat/             (combatcore, combatstate, combatmath,
                │  │                                combatservices, combattypes, battlelog)
                │  ├─ tactical/commander/          (commander entities, roster, turn state)
                │  └─ tactical/powers/
                │     ├─ powercore/                (shared pipeline, context, logger)
                │     ├─ spells/                   (spell casting)
                │     ├─ effects/                  (active buffs/debuffs)
                │     ├─ artifacts/                (inventory, charges, behaviors, dispatcher)
                │     ├─ perks/                    (perk hooks, dispatcher, behaviors)
                │     └─ progression/              (commander-level perk/spell library)
                │
                ├─ Campaign Systems (Strategic)
                │  ├─ campaign/overworld/
                │  │  ├─ core/       (components, types, node registry, walkability)
                │  │  ├─ tick/       (strategic turn clock)
                │  │  ├─ faction/    (NPC faction AI, archetypes, scoring)
                │  │  ├─ threat/     (threat node growth)
                │  │  ├─ influence/  (influence radius interactions)
                │  │  ├─ node/       (unified node management)
                │  │  ├─ garrison/   (garrison defense / raid detection)
                │  │  ├─ victory/    (win/loss conditions)
                │  │  └─ overworldlog/ (event recording & export)
                │  └─ campaign/raid/ (raid encounters, floor graph, deployment, recovery)
                │
                ├─ Mind Systems (AI + Encounter Bridge)
                │  ├─ mind/ai/              (utility AI action evaluation)
                │  ├─ mind/behavior/        (threat layers, danger maps)
                │  ├─ mind/evaluation/      (power scoring, role evaluation)
                │  ├─ mind/encounter/       (encounter triggers, setup, resolution)
                │  ├─ mind/combatlifecycle/ (combat setup, cleanup, rewards, casualties)
                │  └─ mind/spawning/        (automatic squad creation / composition)
                │
                ├─ World Systems (Map Generation)
                │  ├─ world/worldmapcore/   (tile types, biomes, generator interface)
                │  ├─ world/worldgen/       (map generator algorithms and registry)
                │  └─ world/garrisongen/    (garrison-specific multi-floor pipelines)
                │
                ├─ Visual / Rendering Layer
                │  ├─ visual/graphics/      (primitives)
                │  ├─ visual/rendering/     (batch rendering, viewport, cache)
                │  ├─ visual/maprender/     (tile rendering)
                │  ├─ visual/combatrender/  (squad renderer, overlays, highlights)
                │  └─ visual/vfx/           (animators, effect renderers)
                │
                ├─ GUI Layer (GameModeCoordinator + PanelRegistry)
                │  ├─ gui/framework/        (UIMode, ModeManager, coordinator, contextstate,
                │  │                         PanelRegistry, GUIQueries, ActionMap, submenus)
                │  ├─ gui/builders/         (panels, dialogs, lists, layout, widgets)
                │  ├─ gui/widgets/          (cached list, cached text area, text display)
                │  ├─ gui/specs/            (responsive layout specs)
                │  ├─ gui/widgetresources/  (fonts, images, cached backgrounds)
                │  ├─ gui/guicombat/        (combat + combat animation modes)
                │  ├─ gui/guioverworld/     (overworld mode)
                │  ├─ gui/guiexploration/   (exploration mode)
                │  ├─ gui/guinodeplacement/ (node placement mode)
                │  ├─ gui/guisquads/        (squad editor, purchase, deployment, artifact modes)
                │  ├─ gui/guiraid/          (raid mode: deploy, floor map, summary)
                │  ├─ gui/guiprogression/   (commander progression mode)
                │  ├─ gui/guispells/        (spell panel)
                │  ├─ gui/guiartifacts/     (artifact panel)
                │  ├─ gui/guiinspect/       (unit inspection)
                │  ├─ gui/guiunitview/      (unit detail view mode)
                │  └─ gui/guistartmenu/     (start menu)
                │
                └─ Input Layer
                   └─ input/cameracontroller.go (WASD movement, diagonals, map scroll)
```

### Dependency Flow

```
game_main  (entry point)
    ↓
setup/gamesetup + setup/savesystem
    ↓
core/common, core/coords, core/config   ← ALL other packages depend on core/
    ↓
world/worldmapcore, world/worldgen, world/garrisongen
    ↓
tactical/squads, tactical/combat, tactical/commander, tactical/powers/*
    ↓
campaign/overworld/* , campaign/raid
    ↓
mind/ai, mind/behavior, mind/evaluation, mind/encounter,
mind/combatlifecycle, mind/spawning          ← bridges tactical ↔ campaign
    ↓
visual/*                ← reads ECS state for rendering
    ↓
gui/framework → gui/<mode>                    ← mode-based presentation layer
    ↓
input/                  ← feeds ActionMap, modifies camera/selection
```

**Key Principles**:
- Dependencies flow downward. GUI, input, and visual layers depend on lower layers, never the other way around.
- `core/` is leaf (pure infrastructure) and imported by everything.
- Tactical and campaign layers are siblings; they communicate only through `mind/encounter/` and `mind/combatlifecycle/`.
- `tactical/powers/*` all share `powercore/` — spells, effects, artifacts, and perks route through the same pipeline and dispatcher patterns.
- GUI state (`gui/framework/contextstate.go`) holds **only** UI concerns (selection, mode flags). Game state lives in ECS components.

**Cross-layer bridge points to know:**
- `mind/encounter/` — overworld threats trigger → build enemy squads → hand off to tactical combat → resolve → write rewards back to campaign.
- `mind/combatlifecycle/` — orchestrates combat setup, per-turn processing, and cleanup (casualties, rewards) around the tactical core.
- `tactical/commander/movement.go` — commander movement consumes the walkable grid built in `campaign/overworld/core/walkability.go`.
- `gui/framework/guiqueries*.go` — the only layer GUI code uses to read ECS state. No mode reaches into ECS components directly.

### Package Structure

For the full annotated package tree with file-level descriptions, see [PROJECT_LAYOUT.md](PROJECT_LAYOUT.md).

---

## Documentation Index

### Systems
- [Squad & Combat System](project_documentation/Systems/SQUAD_COMBAT_SYSTEM.md) — Formation-based combat, turn management, damage resolution
- [Overworld Architecture](project_documentation/Systems/OVERWORLD_ARCHITECTURE.md) — Tick system, factions, influence, strategic layer
- [Raids](project_documentation/Systems/RAIDS.md) — Garrison defense encounters, floor graphs, deployment
- [Artifact System](project_documentation/Systems/ARTIFACT_SYSTEM.md) — Items, equipment, charges, behaviors
- [Spells](project_documentation/Systems/SPELLS_DOCUMENTATION.md) — Spell casting, spellbooks, effect application
- [Perk System](project_documentation/Systems/PERK_SYSTEM.md) — Hooks, dispatcher, unit/commander perks
- [Progression](project_documentation/Systems/PROGRESSION.md) — Commander veterancy, perk/spell library, split currency
- [World Generation](project_documentation/Systems/WORLD_GENERATION.md) — Generator registry, map algorithms, tile/biome system
- [Coords Package](project_documentation/Systems/COORDS_PACKAGE.md) — Coordinate manager, logical vs pixel, index conversion
- [Visual Packages](project_documentation/Systems/VISUAL_PACKAGES.md) — Graphics, rendering, maprender, combatrender, vfx
- [Save System](project_documentation/Systems/SAVE_SYSTEM.md) — Chunk-based serialization

### AI
- [AI Controller](project_documentation/AI/AI_CONTROLLER.md) — AI entry point and orchestration
- [AI Algorithm Architecture](project_documentation/AI/AI_ALGORITHM_ARCHITECTURE.md) — Utility AI, action evaluation
- [AI Configuration](project_documentation/AI/AI_CONFIGURATION.md) — JSON config files for AI behavior
- [Behavior & Threat Layers](project_documentation/AI/BEHAVIOR_THREAT_LAYERS.md) — Threat painting, danger maps
- [Power Evaluation](project_documentation/AI/POWER_EVALUATION.md) — Power scoring, role evaluation
- [Encounter System](project_documentation/AI/ENCOUNTER_SYSTEM.md) — Encounter triggers, setup, resolution
- [Config Tuning Guide](project_documentation/AI/CONFIG_TUNING_GUIDE.md) — Tunable parameters and balancing

### Architecture
- [Architecture Layers](project_documentation/Architecture/ARCHITECTURE_LAYERS.md) — Dependency rules and layer boundaries
- [Combat Pipelines](project_documentation/Architecture/COMBAT_PIPELINES.md) — Setup, turn, resolution pipelines
- [Hooks and Callbacks](project_documentation/Architecture/HOOKS_AND_CALLBACKS.md) — Powers pipeline event hooks
- [Data Flow Patterns](project_documentation/Architecture/DATA_FLOW_PATTERNS.md) — How data moves through the system
- [Caching Overview](project_documentation/Architecture/CACHING_OVERVIEW.md) — Evaluation and query caching strategies
- [Game Data Overview](project_documentation/Architecture/GAMEDATA_OVERVIEW.md) — JSON data files, templates
- [Entity Reference](project_documentation/Architecture/ENTITY_REFERENCE.md) — Entity types and components

### UI
- [GUI Documentation](project_documentation/UI/GUI_DOCUMENTATION.md) — Mode management, PanelRegistry, GUIQueries
- [Input Reference](project_documentation/UI/INPUT_REFERENCE.md) — ActionMap bindings and input flow

### Process
- [ECS Best Practices](project_documentation/Process/ECS_BEST_PRACTICES.md) — Core ECS patterns and component access
- [UI Design Principles](project_documentation/Process/UI_DESIGN_PRINCIPLES.md) — Nine UI principles enforced across modes
- [Performance Guide](project_documentation/Process/PERFORMANCE_GUIDE.md) — Optimization strategies

### Navigation
- [Project Layout](PROJECT_LAYOUT.md) — Full annotated package tree

---

**Document Version**: 5.1 (2026 Edition)
**Last Updated**: 2026-04-22
**Purpose**: Architecture overview and navigation hub. For critical developer warnings (coord manager, entity lifecycle, GUI state separation), see `CLAUDE.md`. For detailed subsystem documentation, follow the links above.
