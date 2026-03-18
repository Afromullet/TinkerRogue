# TinkerRogue Dependency Boundaries & Coupling Analysis

Analysis of architectural boundaries, coupling patterns, and risk areas across the codebase.

---

## 1. Architectural Layers

The codebase has 6 clear layers, ordered from foundation to surface. Each layer depends only on layers below it — no upward imports exist.

```
Layer 6 — Bootstrap/Entry
  gamesetup, game_main
    │
Layer 5 — Presentation
  gui/guicombat, gui/guioverworld, gui/guisquads, gui/guiraid, ...
  gui/framework, gui/builders, gui/specs, gui/widgets, gui/widgetresources
    │
Layer 4 — AI & Orchestration
  mind/ai, mind/behavior, mind/evaluation
  mind/encounter, mind/raid, mind/combatlifecycle
    │
Layer 3 — Game Systems
  tactical/squads, tactical/combat, tactical/effects, tactical/spells
  tactical/commander, tactical/combatservices, tactical/squadcommands
  gear, world/worldmap
  overworld/core, overworld/faction, overworld/garrison, overworld/tick, ...
    │
Layer 2 — Core Infrastructure
  common, visual/graphics, visual/rendering, templates
    │
Layer 1 — Primitives
  world/coords
    │
Layer 0 — Config
  config
```

### Layer 0 — Config (zero dependencies)
- `config/` — Pure leaf package with no internal imports. Used by ~11 packages.

### Layer 1 — Primitives (depend only on config)
- `world/coords/` → config only. Clean coordinate primitive used by ~21 packages.

### Layer 2 — Core Infrastructure (depend on Layer 0-1)
- `common/` → config, world/coords. The ECS foundation, imported by ~37 packages.
- `visual/graphics/` → common, config, world/coords
- `visual/rendering/` → common, visual/graphics, world/coords, world/worldmap
- `templates/` → common, config, world/coords, world/worldmap

### Layer 3 — Game Systems (depend on Layer 0-2)
- `tactical/effects/` → common (cleanest tactical package)
- `tactical/squads/` → common, config, tactical/effects, templates, world/coords
- `tactical/combat/` → common, config, tactical/combat/battlelog, tactical/effects, tactical/squads, testing, world/coords
- `tactical/spells/` → common, tactical/combat, tactical/effects, tactical/squads, templates
- `tactical/commander/` → common, overworld/core, overworld/tick, tactical/spells, tactical/squads, world/coords
- `tactical/squadcommands/` → common, tactical/combat, tactical/squads, tactical/squadservices, world/coords
- `tactical/squadservices/` → common, tactical/squads, world/coords
- `tactical/combatservices/` → common, gear, mind/combatlifecycle, tactical/combat, tactical/combat/battlelog, tactical/effects, tactical/squads, world/coords (8 imports — orchestrator)
- `gear/` → common, config, tactical/combat, tactical/effects, tactical/squads, templates
- `world/worldmap/` → common, config, visual/graphics, world/coords
- Overworld subsystems: `core/` is the local foundation; faction, garrison, influence, node, threat, tick, victory all depend on core + common. See Section 2F for details.

### Layer 4 — AI & Orchestration (depend on Layer 3)
- `mind/evaluation/` → common, tactical/squads, templates (clean)
- `mind/behavior/` → common, mind/evaluation, tactical/combat, tactical/squads, templates, world/coords
- `mind/combatlifecycle/` → common, tactical/combat, tactical/spells, tactical/squads
- `mind/ai/` → common, mind/behavior, tactical/combat, tactical/combatservices, tactical/squadcommands, tactical/squads, world/coords
- `mind/encounter/` → 10 internal imports spanning overworld + tactical (heaviest bridge)
- `mind/raid/` → 8 internal imports including encounter, evaluation, worldmap

### Layer 5 — Presentation (depend on everything below)
- `gui/specs/`, `gui/widgets/` — zero internal imports (pure UI data/components)
- `gui/widgetresources/` → config only
- `gui/builders/` → common, gui/specs, gui/widgetresources, gui/widgets, tactical/squads
- `gui/framework/` → 10 internal imports (local foundation for all GUI modes)
- `gui/guicombat/` → **25 internal imports** (highest fan-out in codebase)
- `gui/guioverworld/` → 17 internal imports (second-highest)
- `gui/guisquads/` → 16 internal imports (third-highest)
- Other modes (guiraid, guiexploration, guinodeplacement, etc.) are well-scoped at 5-10 imports each

### Layer 6 — Bootstrap/Entry
- `gamesetup/` → 28 imports (composition root — expected)
- `game_main/` → 18 imports (entry point)

---

## 2. Coupling Assessment by Domain

### A. Foundation Layer (config, coords, common) — LOOSE COUPLING

- `config/` is a true leaf: zero internal imports, used by ~11 packages
- `world/coords/` depends only on config — clean primitive
- `common/` depends only on config and coords — clean ECS infrastructure

**Knowledge required**: Packages only need to know ECS primitives (EntityID, component access patterns). No internal details leak.

**Verdict**: Well-isolated foundation. Changes here ripple widely but the API surface is stable.

### B. Tactical Core (squads, combat, effects, spells) — MODERATE COUPLING

- `tactical/effects/` is the cleanest: depends only on `common`, pure data definitions
- `tactical/squads/` has moderate fan-out (5 imports) but very high fan-in (~22 packages import it) — it's the most depended-upon game system
- `tactical/combat/` depends on squads (must read squad data for turn order, movement validation). Also imports `testing` package directly.
- `tactical/spells/` bridges effects, combat, and squads — moderate coupling but justified by domain
- `tactical/combat/battlelog/` is a clean sub-package with only squads as internal dependency

**Knowledge required**: Combat must understand squad data structures (SquadData, unit roles, action points). Spells must understand both effect definitions and combat targeting. These packages share ECS component definitions extensively.

**Concern**: `squads` is imported by virtually everything. Any change to SquadData or squad component structure cascades widely.

**Verdict**: Acceptable coupling given the domain. The squad package is the heaviest shared contract in the codebase.

### C. Combat Services Hub — HIGH COUPLING (by design)

- `tactical/combatservices/` imports 8 internal packages including `gear`, `mind/combatlifecycle`, `combat`, `effects`, `squads`
- It's an explicit orchestration layer — encapsulates all combat game logic
- Uses interface injection to avoid import cycles (AITurnController interface instead of importing mind/ai directly)
- Uses callback registration for GUI integration (OnAttackComplete hooks)

**Knowledge required**: Deep. CombatService knows combat state machine internals, gear charge tracking, effect application, faction management, and lifecycle hooks.

**Verdict**: Intentional God Service for combat. The interface-based injection pattern (SetAIController, SetThreatProvider) shows awareness of the coupling problem. The coupling is contained but the package is a single point of change for any combat-related modification.

### D. Gear System — MODERATE-HIGH COUPLING

- `gear/` imports `tactical/combat` and `tactical/squads` directly (6 total imports)
- `BehaviorContext` struct wraps `combat.CombatQueryCache` — gear behaviors execute combat queries directly
- Artifact activated behaviors call into combat and squad query functions

**Knowledge required**: Gear must understand combat's query cache, squad faction lookups, and action point systems.

**Concern**: Artifact behaviors are essentially combat logic living in the gear package. The BehaviorContext creates a tight contract between gear and combat internals.

**Verdict**: This is the tightest cross-domain coupling in the codebase. If combat query APIs change, gear behaviors break.

### E. Mind Layer (AI, behavior, evaluation, encounter) — LAYERED BUT CROSS-CUTTING

- `mind/evaluation/` is clean: only imports common + squads + templates (power calculation)
- `mind/behavior/` adds combat awareness on top of evaluation
- `mind/ai/` is the consumer: reads behavior outputs, issues commands via combatservices + squadcommands
- `mind/combatlifecycle/` is a thin coordinator for combat setup/teardown
- `mind/encounter/` is the heaviest: bridges overworld (core, garrison, threat) with tactical (combat, squads) — 10 internal imports
- `mind/raid/` chains encounters and adds worldmap + evaluation dependencies (8 imports)

**Knowledge required**: AI needs to understand squad abilities, combat state, and movement rules. Encounter needs to understand both overworld node/threat state AND tactical squad/combat setup. Raid adds map generation awareness.

**Concern**: `encounter` is the primary bridge between the overworld and tactical domains. It must understand internal details of both. Changes to either overworld threat system or combat setup propagate through encounter.

**Verdict**: The mind layer is where the two game domains (overworld strategy + tactical combat) meet. Coupling here is inherent to the game design but encounter is carrying a lot of cross-domain knowledge.

### F. Overworld Subsystems — WELL-STRUCTURED INTERNAL COUPLING

- `overworld/core/` is the local foundation (like common is globally) — 5 imports, 11 fan-in
- Other overworld packages (faction, garrison, influence, node, threat, victory) all depend on core + common
- `overworld/tick/` is the orchestrator: calls faction → influence → threat updates
- `overworld/garrison/` is the one cross-domain link: it imports `tactical/squads` to station squads at nodes
- `overworld/overworldlog/` has zero internal imports — pure logging

**Knowledge required**: Overworld packages mostly only need to know `overworld/core` node/resource types. Garrison is the exception — it must understand squad entity structure.

**Verdict**: Best-structured domain in the codebase. Clean internal layering with minimal external coupling. Only garrison crosses into tactical territory.

### G. GUI Layer — HIGH FAN-OUT, MODERATE KNOWLEDGE DEPTH

- `gui/specs/` and `gui/widgets/` have zero internal imports — pure UI primitives
- `gui/framework/` is the local foundation — all gui modes import it (10 imports, 11 fan-in)
- `gui/builders/` imports `tactical/squads` directly for list formatting — leaks domain types into generic UI infrastructure

**Mode packages (grouped)**:
- **guicombat** (25 imports): The outlier. Imports from every layer — AI, evaluation, combat, spells, artifacts, squad commands, battlelog, plus 7 gui sub-packages. This is the most complex GUI mode by far.
- **guioverworld** (17 imports): Second-heaviest. Bridges overworld state (core, garrison, threat, tick), encounter triggering, and commander management.
- **guisquads** (16 imports): Third-heaviest. Manages gear, squad services, commander, inspection, and unit viewing.
- **Other modes** (guiraid, guiexploration, guinodeplacement, guiartifacts, guispells, guiinspect, guiunitview, guistartmenu): Well-scoped at 2-10 imports each, handling single concerns.

**Knowledge required**: GUI modes need to read game state (squad data, combat state, overworld nodes) for display and translate user input into game commands. They don't modify game state directly — they call into service layers.

**Concern**: `gui/builders/` imports `tactical/squads` directly for list formatting — this leaks domain types into generic UI infrastructure.

**Verdict**: Fan-out is high but mostly read-only. The callback/hook pattern in combatservices helps keep GUI from reaching too deep. guicombat's 25-import count is exceptional and may warrant sub-decomposition over time.

### H. Visual Layer — CLEAN SEPARATION

- `visual/graphics/` depends on common + config + coords (3 imports)
- `visual/rendering/` depends on graphics + common + coords + worldmap (4 imports)
- Neither package imports any tactical, mind, or GUI packages

**Verdict**: Excellent isolation. Rendering knows about world geometry but nothing about game logic.

### I. Save System — WIDE BUT SHALLOW COUPLING

- `savesystem/` itself only imports `common` (1 import)
- `savesystem/chunks/` imports 10 packages (squads, commander, spells, gear, raid, worldmap, etc.) but only accesses their component data for serialization

**Knowledge required**: Chunks must know component data structures for serialization, but doesn't invoke any game logic.

**Verdict**: The coupling is structural (data shapes) not behavioral (logic). This is the nature of serialization. Well-contained.

### J. Gamesetup & Entry — EXPECTED HIGH COUPLING

- `gamesetup/` imports 28 packages — it's the wiring/composition root
- Registers all GUI modes, initializes overworld, tactical, and template systems
- `game_main/` imports 18 packages as the entry point

**Verdict**: This is the composition root. High coupling here is expected and correct — it's the one place that knows about everything so other packages don't have to.

---

## 3. Key Coupling Patterns

### Interface Injection (good)
- `combatservices.AITurnController` interface avoids ai→combatservices import cycle
- `combatservices.ThreatProvider` interface decouples threat evaluation from combat
- `encounter.CombatTransitionHandler` interface decouples GUI mode switching from encounter logic

These interfaces let high-level packages (AI, GUI) plug into lower-level orchestration without creating import cycles.

### Callback Registration (good)
- `combatservices` exposes OnAttackComplete/OnMoveComplete/OnTurnEnd hooks
- GUI registers callbacks at initialization time, reacts to combat events without combatservices knowing about GUI

This pattern keeps the dependency arrow pointing downward (GUI → combatservices, never reversed).

### Direct Type References (mixed)
- `gear.BehaviorContext` wrapping `combat.CombatQueryCache` — tight but contained to a single struct
- `gui/builders` importing `tactical/squads` for list entry types — leaks domain types into generic UI infrastructure, should be parameterized with generics or interfaces

### Shared Component Contracts (unavoidable)
- `SquadData`, `SquadMemberData`, combat components are shared across ~22 packages
- This is inherent to ECS — components are the shared language
- The ECS pattern makes this manageable: packages depend on data shapes, not behavior

### Self-Registration via init() (good)
- Worldmap generators register themselves in `init()`
- ECS subsystems use `common.RegisterSubsystem()` for component initialization
- Avoids manual wiring in gamesetup for new subsystems

---

## 4. Highest-Risk Change Points

Ranked by blast radius and probability of breakage:

### 1. `tactical/squads/` component changes
**Fan-in: ~22 packages**. Any field change in SquadData or SquadMemberData affects combat, AI, GUI, gear, save system, encounter, raid, overworld/garrison, battlelog, and more. This is the single highest-risk package in the codebase.

### 2. `tactical/combat/` query API changes
**Fan-in: ~12 packages**. Breaks gear behaviors (BehaviorContext), AI decision-making, GUI combat visualization, encounter setup, and combatlifecycle. The CombatQueryCache is a particularly tight contract.

### 3. `common/` ECS utility changes
**Fan-in: ~37 packages**. Foundation shift affects everything. However, the API surface (GetComponentType, EntityID patterns) is stable and well-documented, reducing practical risk.

### 4. `mind/encounter/` interface changes
**Fan-in: 3 packages, but bridges 2 domains**. Breaks the overworld↔tactical bridge, affects GUI overworld mode and raid system. Changes here require coordinating both game domains.

### 5. `overworld/core/` data structure changes
**Fan-in: ~11 packages**. All overworld subsystems depend on core node/resource types. Clean internal structure limits blast radius to the overworld domain.

---

## 5. Summary Scorecard

| Domain | Internal Coupling | Cross-Domain Coupling | Knowledge Depth | Fan-In Risk |
|--------|------------------|-----------------------|-----------------|-------------|
| Foundation (config, coords, common) | Low | N/A (provides to all) | Shallow — stable APIs | High (37) but stable |
| Tactical Core (squads, combat, effects, spells) | Moderate | Moderate (gear, AI consume) | Moderate — shared ECS components | Very High (22 for squads) |
| Combat Services | N/A (single package) | High (8 imports, orchestrator) | Deep — knows combat internals | Low (3) |
| Gear | Low internal | High (combat + squads) | Deep — executes combat queries | Low (4) |
| Mind/AI | Layered (eval→behavior→ai) | High (encounter bridges domains) | Deep for encounter, moderate for others | Low-Moderate |
| Overworld | Well-layered (core→subsystems→tick) | Low (only garrison→squads) | Shallow — mostly self-contained | Moderate (11 for core) |
| GUI | Low between modes | High fan-out, shallow depth | Read-mostly — displays state | Low (modes are leaves) |
| Visual | Low | Low | Shallow — geometry only | Moderate (7 for graphics) |
| Save System | Low | Wide but shallow (data shapes) | Data shapes only | Low (3) |
| Gamesetup | N/A (composition root) | Intentionally high (28 imports) | Wiring only | Lowest (1) |

---

## 6. Architectural Strengths

1. **No import cycles** — The layered structure is acyclic. `go build ./...` confirms this.
2. **Interface injection at cycle boundaries** — Where natural cycles would form (combatservices↔ai, encounter↔gui), interfaces break the dependency.
3. **Overworld is exemplary** — Clean internal layering with core as foundation, minimal external coupling.
4. **Visual layer is fully decoupled** — Knows geometry, not game logic.
5. **ECS as shared language** — Component data structures provide a stable, well-understood contract between packages.

## 7. Areas to Watch

1. **gui/guicombat fan-out (25)** — May benefit from further decomposition as features grow.
2. **gui/builders → tactical/squads** — Generic UI infrastructure shouldn't depend on domain types. Could be parameterized.
3. **gear.BehaviorContext → combat.CombatQueryCache** — Tight cross-domain coupling. If combat queries evolve, gear must follow.
4. **tactical/combat → testing** — Production package importing test utilities (gated by DEBUG_MODE, but still a compile-time dependency).
5. **mind/encounter breadth** — 10 imports spanning two game domains. Natural bridge point but carries significant cross-domain knowledge.
