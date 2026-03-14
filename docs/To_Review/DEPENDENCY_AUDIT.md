# TinkerRogue Dependency Audit

**Date:** 2026-03-14

Deep audit of package dependencies performed by two independent agents (codebase explorer + refactoring specialist). Findings synthesized below.

---

## Dependency Statistics

| Metric | Value |
|--------|-------|
| Most-imported package | `common` (177 files) |
| Highest fan-out (single pkg) | `gui/guicombat` (24 deps), `gamesetup` (27 deps) |
| GUI layer total imports | 61 unique internal packages |
| Architectural violations | 6 critical/high cross-layer issues |
| Circular dependencies | 0 found |
| Testing imports in prod code | 2 locations |

### Most-Imported Packages (Fan-In)

| Package | Import Count | Role |
|---------|-------------|------|
| `common` | 177 | ECS core, position system, components, RNG, resources |
| `world/coords` | 102 | Coordinate types and conversions |
| `tactical/squads` | 97 | Squad/unit data and queries |
| `templates` | 45 | Entity templates, JSON data |
| `gui/framework` | 44 | GUI mode management, queries |
| `overworld/core` | 38 | Overworld components and data |
| `tactical/combat` | 28 | Combat state, factions, actions |
| `world/worldmap` | 27 | Map generation, tile data |
| `gui/builders` | 23 | Widget construction helpers |
| `config` | 21 | Constants and flags |

### Highest Fan-Out Packages (Most Dependencies)

| Package | Unique Deps | Concern |
|---------|------------|---------|
| `gamesetup` | 27 | Bootstrap — expected |
| `gui/guicombat` | 24 | Combat UI — too high |
| `game_main` | 18 | Entry point — acceptable |
| `gui/guioverworld` | 17 | Overworld UI — high but functional |
| `gui/guisquads` | 16 | Squad UI — high |

---

## Critical Issues


---

### 3. `templates` → `visual/rendering` (Data layer depends on rendering)

**Severity: HIGH**

**File:** `templates/entity_factory.go`

**Problem:** The templates package is supposed to be a data/definition layer but directly imports `visual/rendering` and `world/worldmap`:

```go
entity.AddComponent(rendering.RenderableComponent, &rendering.Renderable{
    Image:   commanderImage,
    Visible: config.Visible,
})
```

**Impact:**
- Cannot create entities without graphics subsystem running
- Harder to test data loading independently
- Data definitions coupled to dynamic rendering

**Fix Options:**
- Extract entity creation with rendering components to a separate factory in a higher layer
- Accept a callback for adding rendering components instead of doing it directly
- Split templates into `templates/definitions` (pure data) and move creation to `gamesetup` or `common/factory`

---

### 4. `tactical/` packages → `visual/rendering` (game logic imports presentation)

**Severity: CRITICAL**

**Files:**
- `tactical/commander/system.go` imports `visual/rendering`
- `tactical/squads/squadcreation.go` imports `visual/rendering`
- `tactical/squads/units.go` imports `visual/rendering`

**Problem:** Multiple tactical packages import `visual/rendering` to create `Renderable` components during entity creation. Game logic should never depend on the presentation layer.

**Impact:**
- Tactical packages cannot be tested without graphics infrastructure
- Changes to rendering require recompilation of tactical code
- Same root cause as issue #3 (templates → rendering)

**Fix:** Use a factory pattern — tactical code creates entities with game-logic components only, then a presentation-layer factory (in `visual/` or `gamesetup/`) adds visual components. This is the same pattern needed for issue #3 and can be solved together.

---

### 5. `mind/encounter` → `visual/rendering` (AI logic imports presentation)

**Severity: CRITICAL**

**Files:**
- `mind/encounter/encounter_service.go` imports `visual/rendering`
- `mind/encounter/starters.go` imports `visual/rendering`

**Problem:** The encounter system imports `visual/rendering` to hide/show sprites during combat transitions (creating `hiddenRenderable` components). AI/encounter logic should not manipulate rendering directly.

**Impact:**
- Encounter system cannot be tested without graphics infrastructure
- Combat transition visuals are tangled into encounter logic
- Same root cause as issues #3 and #4

**Fix:** Encounter service should emit events (or set a flag component); a visual system listens and handles sprite visibility changes. Alternatively, the same presentation-layer factory pattern from issues #3/#4 can handle this.

---

## High Priority Issues

### 6. GUI → `mind/combatpipeline` (GUI imports game logic)

**Severity: HIGH**

**Files:**
- `gui/guioverworld/overworld_action_handler.go` imports `mind/combatpipeline`
- `gui/guiraid/deploy_panel.go` imports `mind/combatpipeline`

**Problem:** GUI packages directly import `mind/combatpipeline` to trigger combat setup and cleanup. The GUI layer should not reach into the mind layer.

**Impact:**
- GUI tests require the full combat pipeline stack
- Tight coupling between UI and combat orchestration

**Fix:** GUI should call through a coordinator or service interface rather than importing `mind/combatpipeline` directly. Define an interface in `gui/framework` or `tactical/combatservices` and wire the concrete implementation in `gamesetup`.

---

### 7. `tactical/combatservices` → `mind/ai` + `mind/behavior` (tactical layer reaches into AI)

**Severity: HIGH**

**Problem:** `CombatService` directly creates and manages `ai.AIController`, `behavior.FactionThreatLevelManager`, and `behavior.CompositeThreatEvaluator`. The type aliases (`AIController = ai.AIController`, `QueuedAttack = ai.QueuedAttack`) are dependency laundering — they exist so `gui/guicombat` can use AI types without importing `mind/ai` directly, but don't actually decouple anything.

**Impact:**
- Testing combat service requires the entire AI stack
- Tactical layer has upward dependency into mind layer
- `CombatService` becomes a mega-object owning systems from two conceptual layers

**Fix:** Define a `CombatAI` interface in `tactical/combat`:

```go
type CombatAI interface {
    ProcessAITurn(factionID ecs.EntityID) []QueuedAction
}
```

`CombatService` accepts this interface instead of creating `ai.AIController` directly. Wire concrete implementations in `gamesetup`.

---

### 8. `gui/guicombat` → `mind/encounter` (GUI imports encounter system)

**Severity: HIGH**

**Files:** `gui/guicombat/` imports `mind/encounter` for `encounter.EncounterService`, `encounter.ExitDefeat`, `encounter.ExitVictory`, `encounter.ExitFlee`, `encounter.CombatResult`

**Problem:** The combat UI directly references the encounter system for exit reason constants and result types. Conceptually wrong — the combat UI should report results through shared types, not directly reference the encounter system.

**Fix:** Move `CombatResult`, `ExitReason` constants to `tactical/combat` (already imported by both sides). This drops `gui/guicombat`'s `mind/encounter` dependency entirely with ~20 lines of type relocation.

---

### 9. `common` is a God Package (177 imports)

**Severity: HIGH**

The `common` package (1,028 lines, 7 files) contains six distinct responsibilities:

| Responsibility | File | Lines |
|---------------|------|-------|
| ECS Infrastructure | `ecsutil.go` | 264 |
| Position System | `positionsystem.go` | 210 |
| Game Components | `commoncomponents.go` | 165 |
| Resources | `resources.go` | 80 |
| RNG | `randnumgen.go` | 50 |
| Player Data | `playerdata.go` | 42 |

**Most concerning:** The `Attributes` struct has 12+ derived-stat methods (`GetPhysicalDamage`, `GetMaxHealth`, `GetHitRate`, etc.) embedding game balance formulas in a foundation package.

**Fix:** Extract derived-stat methods to `tactical/combat` or a new `tactical/stats` package as free functions: `func GetPhysicalDamage(attr *common.Attributes) int`. Keep the `Attributes` struct itself in `common` as pure data.

---

## Medium Priority Issues

### 10. `gui/framework` → `tactical/combat`, `tactical/squads`, `tactical/squadcommands`

**Problem:** `GUIQueries` in `gui/framework` directly creates and wraps `combat.CombatQueryCache`, `combat.CombatFactionManager`, and `squads.SquadQueryCache`. `CommandHistory` wraps `squadcommands.CommandExecutor`. The GUI framework has hard dependencies on three tactical packages, and is imported by 44 files.

**Fix:** Define query interfaces in `gui/framework` and have tactical packages implement them. Wire concrete implementations in `gamesetup`.

**Risk:** Medium-high due to 44 downstream files. Interface design must be stable.

---

### 11. `savesystem/chunks` Hub Pattern

**Problem:** The serialization package imports 10+ domain-specific packages (`gear`, `mind/raid`, `tactical/commander`, `tactical/spells`, `tactical/squads`, `visual/graphics`, `world/coords`, `world/worldmap`). Adding new systems requires modifying chunks.

**Fix:** Create a plugin/registry pattern where each domain registers its own serialization logic. `savesystem/chunks` depends only on `common` and a plugin interface. Move domain-specific chunk logic to `tactical/chunks`, `mind/chunks`, `gear/chunks`.

---

### 12. `input` → `gui/framework` (Inverted dependency)

**Problem:** Input is supposed to be a low-level system, but it imports `gui/framework`. The input layer is aware of GUI infrastructure.

**Fix:** Move `InputState` definition to the `input` package. Have GUI framework depend on input, not vice versa.

---

### 13. Testing Package in Production Code

**Files:**
- `gamesetup/helpers.go` — imports `testing` and calls `testing.CreateTestItems(gm)` without compile-time gating
- `tactical/combat/combat_testing_2.go` — production file with confusing test-like name

**Fix:**
- Add build tags or compile-time gating for testing package imports
- Rename `combat_testing_2.go` to something clearer (e.g., `combat_setup_fixtures.go`)

---

## Low Priority Issues

### 14. `tactical/commander` → `overworld/core` + `overworld/tick`

**Problem:** Commander straddles tactical and overworld layers. Not necessarily wrong — documented and intentional — but tests for commander need overworld infrastructure.

**Assessment:** Acceptable if intentional. Consider extracting a `CommanderContext` interface for walkability checks if this becomes painful.

---

### 15. `gui/builders` → `tactical/squads`

**Problem:** Only used for `squads.UnitIdentity` type (a simple struct) and `squads.GetSquadName`. Widget builder layer directly references squad data types.

**Fix:** Move `UnitIdentity` to `common` or define it locally. Small effort, low risk.

---

### 16. Empty Root `combatservices/` Directory

An empty directory at project root, likely left over from a refactor. Delete it.

---

### 17. Small Package Consolidation Candidates

| Package | Lines | Merge Into |
|---------|-------|------------|
| `overworld/tick` | 84 | `overworld/core` (check for import cycles first) |
| `overworld/victory` | 191 | `overworld/core` |
| `gui/guiinspect` | 182 | `gui/guiunitview` (195 lines) |

---

## Service Layer Assessment

### `tactical/combatservices` — Partially Effective

**Good:** Encapsulates combat lifecycle, manages event callbacks for GUI decoupling, handles artifact behavior dispatch.

**Bad:** Reaches upward into `mind/ai` and `mind/behavior`. Type aliases are dependency laundering, not real decoupling.

### `tactical/squadservices` — Well Designed

Clean separation of purchase and deployment logic. Pure business operations with result types. Proper rollback semantics. Minimal dependencies. This is the model to follow.

---

## Positive Findings

- **No circular dependencies detected**
- **World layer properly isolated** — `world/coords` and `world/worldmap` have no internal dependencies beyond config
- **Visual layer properly separated** — graphics and rendering are cleanly distinct
- **Combat system has reasonable cohesion**
- **Documented dependency structure in PACKAGE_DEPENDENCIES.md is mostly accurate**

---

## Root Causes

Many of the architectural violations share a single root cause: **entity creation requires `visual/rendering` imports for `Renderable` components**. This pattern appears in:

- `templates/entity_factory.go` (issue #3)
- `tactical/commander/system.go` (issue #4)
- `tactical/squads/squadcreation.go` and `units.go` (issue #4)
- `mind/encounter/encounter_service.go` and `starters.go` (issue #5)

In each case, game logic creates an entity and immediately attaches a `Renderable` component, forcing an import of `visual/rendering`. A **centralized presentation-layer entity factory** would fix all four issues at once. The pattern:

1. Game logic creates entities with only game-logic components (position, stats, squad membership, etc.)
2. A presentation factory (in `visual/` or `gamesetup/`) observes entity creation and attaches `Renderable` components based on entity type
3. Game logic packages no longer import `visual/rendering`

This single architectural change would eliminate 4 of the 6 critical/high cross-layer violations.

---

## Recommended Action Order

| Step | Change | Effort | Impact |
|------|--------|--------|--------|
| 1 | Move `ThreatVisualizer` out of `mind/behavior` | Small | Breaks AI → Graphics dependency |
| 2 | Create presentation-layer entity factory for `Renderable` | Medium | Fixes templates, tactical, and encounter → rendering (issues #3, #4, #5) |
| 3 | Move combat exit/result types to `tactical/combat` | Small | Breaks GUI → mind/encounter |
| 4 | Define `CombatAI` interface, break combatservices → mind | Medium | Breaks tactical → mind dependency |
| 5 | Break gui/guioverworld and gui/guiraid → mind/combatpipeline via interface | Small | Breaks GUI → mind dependency |
| 6 | Extract `Attributes` stat methods from `common` | Small | Cleaner foundation package |
| 7 | Decouple `gui/framework` from tactical types | Med-Large | GUI framework independence |
| 8 | Refactor `savesystem/chunks` to plugin pattern | Medium | Eliminates hub dependencies |
| 9 | Move `InputState` to `input` package | Small | Correct dependency direction |
| 10 | Delete empty `combatservices/` directory | Trivial | Housekeeping |
| 11 | Merge small packages (tick, victory, inspect) | Small | Less package sprawl |

Steps 1-2 together eliminate the most widespread cross-layer dependencies — the entity factory alone fixes 4 violations. Steps 3-5 are small, incremental changes that each stand alone as safe, testable fixes.
