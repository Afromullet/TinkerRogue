# Dependency Boundary Interfaces

Reference for all interfaces and patterns used to break circular import dependencies between packages.

---

## Overview

Go forbids circular imports. When package A needs behavior from package B and B also needs something from A, a boundary interface breaks the cycle. TinkerRogue uses three patterns:

1. **Structural typing boundaries** — Package A defines an interface. A type in package B satisfies it without B ever importing A. The caller passes B's concrete type through A's interface.

2. **Registry/plugin pattern** — A central package defines an interface and a `Register*()` function. Implementations in other packages call `Register*()` from `init()`, so the central package never imports them.

3. **Dependency injection structs** — A `*Deps` struct bundles narrow interfaces so that downstream packages receive only what they need, avoiding broad imports.

---

## Combat Pipeline Boundaries

The largest group of boundary interfaces. These separate the combat start/exit pipeline from the domain-specific encounter logic.

### `tactical/combat/combat_contracts.go`

| Interface | Methods | Satisfied By | Cycle Broken |
|-----------|---------|-------------|--------------|
| `CombatStarter` | `Prepare(manager) (*CombatSetup, error)` | `OverworldCombatStarter` (mind/encounter), `GarrisonDefenseStarter` (mind/encounter), `RaidCombatStarter` (mind/raid) | `combat` / `combatlifecycle` ↔ `encounter` / `raid` |
| `CombatTransitioner` | `TransitionToCombat(setup) error` | `EncounterService` (mind/encounter) | `combatlifecycle` ↔ `encounter` |
| `CombatStartRollback` | `Rollback()` | Optional — starters that need cleanup if `TransitionToCombat` fails after `Prepare` succeeds | — |
| `CombatCleaner` | `CleanupCombat(enemySquadIDs)` | `CombatService` (tactical/combatservices) | `combat` ↔ `combatservices` |
| `EncounterCallbacks` | `ExitCombat(reason, result, cleaner)`, `GetRosterOwnerID()`, `GetCurrentEncounterID()` | `EncounterService` (mind/encounter) | `gui/*` ↔ `mind/encounter` |

**How it works:** `ExecuteCombatStart` (in `mind/combatlifecycle/starter.go`) accepts a `CombatTransitioner` and a `CombatStarter`. Callers pass the `EncounterService` as the transitioner and a domain-specific starter (overworld, garrison, raid). Neither the pipeline package nor the combat types package imports `encounter` or `raid`.

### `mind/combatlifecycle/pipeline.go`

| Interface | Methods | Satisfied By | Cycle Broken |
|-----------|---------|-------------|--------------|
| `CombatResolver` | `Resolve(manager) *ResolutionPlan` | `OverworldCombatResolver` (mind/encounter), `GarrisonDefenseResolver` (mind/encounter), `RaidRoomResolver` / `RaidDefeatResolver` (mind/raid) | Multiple resolver domains ↔ `combatlifecycle` |

**How it works:** `ExecuteResolution` accepts a `CombatResolver` and calls `Resolve()` → `Grant()`. Each domain implements its own resolver with domain-specific logic (threat damage, room clearing, etc.) without the pipeline importing those packages.

---

## Encounter ↔ GUI Boundary

### `mind/encounter/types.go`

| Interface | Methods | Satisfied By | Cycle Broken |
|-----------|---------|-------------|--------------|
| `CombatTransitionHandler` | `SetPostCombatReturnMode(mode)`, `SetTriggeredEncounterID(id)`, `ResetTacticalState()`, `EnterCombatMode() error`, `GetPlayerEntityID()`, `GetPlayerPosition()` | `GameModeCoordinator` (gui/framework) | `mind/encounter` ↔ `gui/framework` |

**How it works:** `EncounterService` needs to trigger GUI mode transitions (enter combat mode, reset tactical state) but cannot import `gui/framework`. Instead, it holds a `CombatTransitionHandler` interface. `GameModeCoordinator` satisfies it via structural typing. The concrete coordinator is passed to `NewEncounterService()` in `game_main/setup.go`.

---

## GUI Dependency Injection Structs

GUI packages need encounter-related callbacks but must not import `mind/encounter` directly. Instead, each GUI subsystem defines a `*Deps` struct that holds `combat.EncounterCallbacks` (defined in `tactical/combat`, which both sides already import).

### `gui/guicombat/combatdeps.go` — `CombatModeDeps`

```go
type CombatModeDeps struct {
    BattleState   *framework.TacticalState
    CombatService *combatservices.CombatService
    Queries       *framework.GUIQueries
    ModeManager   *framework.UIModeManager
    Encounter     combat.EncounterCallbacks   // EncounterService passed here
}
```

### `gui/guispells/spell_deps.go` — `SpellCastingDeps`

```go
type SpellCastingDeps struct {
    BattleState *framework.TacticalState
    ECSManager  *common.EntityManager
    GameMap     *worldmap.GameMap
    PlayerPos   *coords.LogicalPosition
    Queries     *framework.GUIQueries
    Encounter   combat.EncounterCallbacks   // EncounterService passed here
}
```

### `gui/guiartifacts/artifact_deps.go` — `ArtifactActivationDeps`

```go
type ArtifactActivationDeps struct {
    BattleState   *framework.TacticalState
    CombatService *combatservices.CombatService
    Queries       *framework.GUIQueries
    Encounter     combat.EncounterCallbacks   // EncounterService passed here
}
```

**Pattern:** The `EncounterService` (from `mind/encounter`) structurally satisfies `combat.EncounterCallbacks` (from `tactical/combat`). GUI packages import only `tactical/combat` for the interface type, never `mind/encounter` for the concrete type.

---

## Threat Visualization Boundaries

### `gui/guicombat/threatvisualizer.go`

| Interface | Methods | Satisfied By | Cycle Broken |
|-----------|---------|-------------|--------------|
| `ThreatDataProvider` | `AddFaction(factionID)`, `UpdateAllFactions()`, `GetSquadThreatAtRange(factionID, squadID, distance) (float64, bool)` | `FactionThreatLevelManager` (mind/behavior) | `gui/guicombat` ↔ `mind/behavior` |
| `LayerDataProvider` | `Update(currentRound)`, `MarkDirty()`, `GetMeleeThreatAt(pos)`, `GetRangedPressureAt(pos)`, `GetSupportValueAt(pos)`, `GetFlankingRiskAt(pos)`, `GetIsolationRiskAt(pos)`, `GetEngagementPressureAt(pos)`, `GetRetreatQuality(pos)` | `CompositeThreatEvaluator` (mind/behavior) | `gui/guicombat` ↔ `mind/behavior` |

**How it works:** `ThreatVisualizer` renders threat overlays on the map. It needs data from the AI evaluation layer (`mind/behavior`) but cannot import it. Both interfaces are defined in `gui/guicombat` and satisfied by behavior types via structural typing. The concrete implementations are injected when `CombatMode` initializes the visualizer.

---

## Rendering Boundaries

### `visual/rendering/renderdata.go`

| Interface | Methods | Satisfied By | Cycle Broken |
|-----------|---------|-------------|--------------|
| `SquadInfoProvider` | `GetAllSquadIDs()`, `GetSquadRenderInfo(squadID) *SquadRenderInfo` | `GUIQueries` adapter (gui/framework) | `visual/rendering` ↔ `gui` |
| `UnitInfoProvider` | `GetUnitIDsInSquad(squadID)`, `GetUnitRenderInfo(unitID) *UnitRenderInfo` | `GUIQueries` adapter (gui/framework) | `visual/rendering` ↔ `gui` |

**How it works:** The rendering system needs squad/unit data for drawing but cannot import GUI packages. `rendering` defines provider interfaces with minimal data structs (`SquadRenderInfo`, `UnitRenderInfo`). `GUIQueries` in `gui/framework` implements adapter methods that satisfy these interfaces by querying ECS components and translating them into the render-friendly structs.

---

## Registry/Plugin Pattern Boundaries

Self-registering interfaces where implementations call `Register*()` from `init()`. The central package never imports the implementations.

### Map Generators — `world/worldmap/generator.go`

```go
type MapGenerator interface {
    Generate(width, height int, images TileImageSet) GenerationResult
    Name() string
    Description() string
}
```

- Registry: `var generators = map[string]MapGenerator{}`
- Registration: `RegisterGenerator(gen MapGenerator)` called from `init()` in each `gen_*.go` file
- Lookup: `GetGeneratorOrDefault(name string) MapGenerator`

### Artifact Behaviors — `gear/artifactbehavior.go`

```go
type ArtifactBehavior interface {
    BehaviorKey() string
    TargetType() int
    OnPostReset(ctx, factionID, squadIDs)
    OnAttackComplete(ctx, attackerID, defenderID, result)
    OnTurnEnd(ctx, round)
    IsPlayerActivated() bool
    Activate(ctx, targetSquadID) error
}
```

- Registry: `var behaviorRegistry = map[string]ArtifactBehavior{}`
- Registration: `RegisterBehavior(b ArtifactBehavior)` called from `init()` in each behavior file
- Lookup: `GetBehavior(key string) ArtifactBehavior`
- Provides `BaseBehavior` struct with no-op defaults for embedding

### Save System Chunks — `savesystem/savesystem.go`

```go
type SaveChunk interface {
    ChunkID() string
    ChunkVersion() int
    Save(em) (json.RawMessage, error)
    Load(em, data, idMap) error
    RemapIDs(em, idMap) error
}

type Validatable interface {   // Optional, checked via type assertion
    Validate(em) error
}
```

- Registry: `var registeredChunks []SaveChunk`
- Registration: `RegisterChunk(chunk SaveChunk)` called from `init()` in each `*_savechunk.go` file
- Lookup: `GetChunk(chunkID string) SaveChunk`
- Two-phase load: `Load` (Phase 1: create entities) → `RemapIDs` (Phase 2: fix cross-references) → `Validate` (Phase 3: optional)

---

## UI Mode Boundaries

### `gui/framework/uimode.go` — `UIMode`

```go
type UIMode interface {
    Initialize(ctx *UIContext) error
    Enter(fromMode UIMode) error
    Exit(toMode UIMode) error
    Update(deltaTime float64) error
    Render(screen *ebiten.Image)
    HandleInput(inputState *InputState) bool
    GetEbitenUI() *ebitenui.UI
    GetModeName() string
}
```

`UIModeManager` (in `gui/framework/modemanager.go`) operates entirely through this interface. It never imports concrete mode packages (`guicombat`, `guiexploration`, etc.). Modes are registered via `RegisterMode(mode UIMode)` from the setup layer.

### `gui/framework/modemanager.go` — `OverlayRenderer` (optional)

```go
type OverlayRenderer interface {
    RenderOverlay(screen *ebiten.Image)
}
```

Checked via type assertion after `UI.Draw()`. Modes that need to draw custom content on top of ebitenui widgets implement this.

### `gui/framework/actionmap.go` — `ActionMapProvider` (optional)

```go
type ActionMapProvider interface {
    GetActionMap() *ActionMap
}
```

Checked via type assertion during input resolution. Modes that use semantic input binding implement this; the `UIModeManager` automatically resolves actions each frame.

---

## Simplified Data Structs

Sometimes a simple data struct avoids a circular import where a full interface would be overkill.

| Struct | File | Why |
|--------|------|-----|
| `VictoryInfo` | `tactical/combat/battlelog/battle_recorder.go:161` | `BattleRecorder.Finalize()` needs victory data but cannot import `combatservices` (which owns `VictoryConditionResult`). `VictoryInfo` is a minimal 3-field struct with just `RoundsCompleted`, `VictorFaction`, and `VictorName`. |

---

## Wiring: Where Concrete Implementations Are Injected

The "glue" code that connects interfaces to implementations lives in the setup layer.

### `game_main/setup.go`

- `setupUICore()` creates `GameModeCoordinator` and passes it to `encounter.NewEncounterService()` as the `CombatTransitionHandler`.
- The returned `EncounterService` is then passed through mode registration so GUI code receives it as `combat.EncounterCallbacks`.

### `gamesetup/moderegistry.go`

- `RegisterTacticalModes()` / `RegisterOverworldModes()` / `RegisterRoguelikeTacticalModes()` register concrete mode implementations with the coordinator.
- `RegisterOverworldModes()` creates a `startCombat` closure that calls `combatlifecycle.ExecuteCombatStart(encounterService, ecsManager, starter)`, passing `encounterService` as the `CombatTransitioner`.
- `EncounterService` is passed to `guicombat.NewCombatMode()` where it becomes the `combat.EncounterCallbacks` in `CombatModeDeps`.

### `mind/raid/raidrunner.go`

- `NewRaidRunner()` sets `encounterService.PostCombatCallback` — a function closure that calls `rr.ResolveEncounter()` when combat ends. This is callback injection rather than an interface, but serves the same decoupling purpose: the encounter service doesn't import the raid package.
- `TriggerRaidEncounter()` calls `combatlifecycle.ExecuteCombatStart(rr.encounterService, rr.manager, starter)`, passing the encounter service as `CombatTransitioner` and a `RaidCombatStarter` as `CombatStarter`.

---

## Import Direction Summary

```
game_main/setup.go          (wiring layer — imports everything)
gamesetup/moderegistry.go   (wiring layer — imports everything)
    │
    ├── gui/framework        ← defines UIMode, CombatTransitionHandler (via encounter/types.go)
    ├── gui/guicombat        ← defines ThreatDataProvider, LayerDataProvider
    ├── gui/guispells        ← defines SpellCastingDeps
    ├── gui/guiartifacts     ← defines ArtifactActivationDeps
    │
    ├── tactical/combat      ← defines CombatStarter, CombatTransitioner, EncounterCallbacks, CombatCleaner
    ├── mind/combatlifecycle  ← defines CombatResolver; calls interfaces from tactical/combat
    │
    ├── mind/encounter       ← defines CombatTransitionHandler; satisfies CombatTransitioner, EncounterCallbacks
    ├── mind/raid            ← satisfies CombatStarter, CombatResolver
    │
    ├── visual/rendering     ← defines SquadInfoProvider, UnitInfoProvider
    ├── world/worldmap       ← defines MapGenerator
    ├── gear                 ← defines ArtifactBehavior
    └── savesystem           ← defines SaveChunk, Validatable
```

Arrows flow **downward** (toward fewer dependencies). The wiring layer at the top is the only code that sees both sides of each boundary.
