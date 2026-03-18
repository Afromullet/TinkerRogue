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

These interfaces separate the combat start/exit pipeline from domain-specific encounter logic.

### `tactical/combat/combat_contracts.go`

| Interface | Methods | Satisfied By | Cycle Broken |
|-----------|---------|-------------|--------------|
| `CombatStarter` | `Prepare(manager) (*CombatSetup, error)` | `OverworldCombatStarter`, `GarrisonDefenseStarter` (mind/encounter), `RaidCombatStarter` (mind/raid) | `combatlifecycle` ↔ `encounter` / `raid` |
| `CombatTransitioner` | `TransitionToCombat(setup) error` | `EncounterService` (mind/encounter) | `combatlifecycle` ↔ `encounter` |
| `CombatStartRollback` | `Rollback()` | Optional — starters that need cleanup if `TransitionToCombat` fails after `Prepare` succeeds | — |
| `CombatCleaner` | `CleanupCombat(enemySquadIDs)` | `CombatService` (tactical/combatservices) | `tactical/combat` ↔ `combatservices` |
| `EncounterCallbacks` | `ExitCombat(reason, result, cleaner)`, `GetRosterOwnerID()`, `GetCurrentEncounterID()` | `EncounterService` (mind/encounter) | `gui/*` ↔ `mind/encounter` |

`ExecuteCombatStart` (in `mind/combatlifecycle/starter.go`) accepts a `CombatTransitioner` and a `CombatStarter`. Callers pass `EncounterService` as the transitioner and a domain-specific starter (overworld, garrison, raid). Neither `combatlifecycle` nor `tactical/combat` imports `encounter` or `raid`.

### `mind/combatlifecycle/pipeline.go`

| Interface | Methods | Satisfied By | Cycle Broken |
|-----------|---------|-------------|--------------|
| `CombatResolver` | `Resolve(manager) *ResolutionPlan` | `OverworldCombatResolver`, `GarrisonDefenseResolver` (mind/encounter), `RaidRoomResolver`, `RaidDefeatResolver` (mind/raid) | Multiple resolver domains ↔ `combatlifecycle` |

`ExecuteResolution` accepts a `CombatResolver` and calls `Resolve()` then `Grant()`. Each domain implements its own resolver with domain-specific logic (threat damage, room clearing, etc.) without `combatlifecycle` importing those packages.

---

## Encounter ↔ GUI Boundary

### `mind/encounter/types.go`

| Interface | Methods | Satisfied By | Cycle Broken |
|-----------|---------|-------------|--------------|
| `CombatTransitionHandler` | `SetPostCombatReturnMode(mode)`, `SetTriggeredEncounterID(id)`, `ResetTacticalState()`, `EnterCombatMode() error`, `GetPlayerEntityID()`, `GetPlayerPosition()` | `GameModeCoordinator` (gui/framework) | `mind/encounter` ↔ `gui/framework` |

`EncounterService` needs to trigger GUI mode transitions (enter combat mode, reset tactical state) but cannot import `gui/framework`. It holds a `CombatTransitionHandler` interface instead. `GameModeCoordinator` satisfies it via structural typing. The concrete coordinator is passed to `encounter.NewEncounterService()` in `game_main/setup.go`.

---

## GUI Dependency Injection Structs

GUI packages need encounter-related callbacks but must not import `mind/encounter` directly. Each GUI subsystem defines a `*Deps` struct holding `combat.EncounterCallbacks` (defined in `tactical/combat`, which both sides already import). This means GUI packages import only `tactical/combat` for the interface type, never `mind/encounter` for the concrete type.

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

---

## AI and Threat System Boundary

### `tactical/combatservices/ai_interfaces.go`

| Interface | Methods | Satisfied By | Cycle Broken |
|-----------|---------|-------------|--------------|
| `AITurnController` | `DecideFactionTurn(factionID)`, `HasQueuedAttacks()`, `GetQueuedAttacks()`, `ClearAttackQueue()` | `AIController` (mind/ai) | `combatservices` ↔ `mind/ai` |
| `ThreatProvider` | `AddFaction(factionID)`, `UpdateAllFactions()`, `GetSquadThreatAtRange(factionID, squadID, distance)` | `FactionThreatLevelManager` (mind/behavior) | `combatservices` + `gui/guicombat` ↔ `mind/behavior` |
| `ThreatLayerEvaluator` | `Update(round)`, `MarkDirty()`, `GetMeleeThreatAt(pos)`, `GetRangedPressureAt(pos)`, `GetSupportValueAt(pos)`, `GetFlankingRiskAt(pos)`, `GetIsolationRiskAt(pos)`, `GetEngagementPressureAt(pos)`, `GetRetreatQuality(pos)` | `CompositeThreatEvaluator` (mind/behavior) | `combatservices` + `gui/guicombat` ↔ `mind/behavior` |

`mind/ai` imports `combatservices` for `QueuedAttack` and the interface types, so `combatservices` cannot import `mind/ai` or `mind/behavior` back. `CombatService` holds the AI controller, threat provider, and evaluator factory as interfaces with setter methods for each. All concrete types are created by `ai.SetupCombatAI()` and injected during `CombatMode.Initialize()`.

---

## Rendering Boundaries

### `visual/rendering/renderdata.go`

| Interface | Methods | Satisfied By | Cycle Broken |
|-----------|---------|-------------|--------------|
| `SquadInfoProvider` | `GetAllSquadIDs()`, `GetSquadRenderInfo(squadID) *SquadRenderInfo` | `GUIQueries` adapter (gui/framework) | `visual/rendering` ↔ `gui` |
| `UnitInfoProvider` | `GetUnitIDsInSquad(squadID)`, `GetUnitRenderInfo(unitID) *UnitRenderInfo` | `GUIQueries` adapter (gui/framework) | `visual/rendering` ↔ `gui` |

The rendering system needs squad/unit data for drawing but cannot import GUI packages. `rendering` defines provider interfaces with minimal data structs (`SquadRenderInfo`, `UnitRenderInfo`). `GUIQueries` in `gui/framework` implements adapter methods that satisfy these interfaces by querying ECS components and translating them into render-friendly structs.

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
    OnPostReset(ctx *BehaviorContext, factionID ecs.EntityID, squadIDs []ecs.EntityID)
    OnAttackComplete(ctx *BehaviorContext, attackerID, defenderID ecs.EntityID, result *squads.CombatResult)
    OnTurnEnd(ctx *BehaviorContext, round int)
    IsPlayerActivated() bool
    Activate(ctx *BehaviorContext, targetSquadID ecs.EntityID) error
}
```

- Registry: `var behaviorRegistry = map[string]ArtifactBehavior{}`
- Registration: `RegisterBehavior(b ArtifactBehavior)` called from `init()` in each behavior file
- Lookup: `GetBehavior(key string) ArtifactBehavior`
- `BaseBehavior` provides no-op defaults for embedding; concrete behaviors override only the hooks they need

### Save System Chunks — `savesystem/savesystem.go`

```go
type SaveChunk interface {
    ChunkID() string
    ChunkVersion() int
    Save(em *common.EntityManager) (json.RawMessage, error)
    Load(em *common.EntityManager, data json.RawMessage, idMap *EntityIDMap) error
    RemapIDs(em *common.EntityManager, idMap *EntityIDMap) error
}

type Validatable interface {   // Optional, checked via type assertion
    Validate(em *common.EntityManager) error
}
```

- Registry: `var registeredChunks []SaveChunk`
- Registration: `RegisterChunk(chunk SaveChunk)` called from `init()` in each `*_savechunk.go` file
- Lookup: `GetChunk(chunkID string) SaveChunk`
- Three-phase load: `Load` (Phase 1: create entities) → `RemapIDs` (Phase 2: fix cross-references) → `Validate` (Phase 3: optional)

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

`UIModeManager` (in `gui/framework/modemanager.go`) operates entirely through this interface and never imports concrete mode packages (`guicombat`, `guiexploration`, etc.). Modes are registered via `RegisterMode(mode UIMode)` from the setup layer.

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

Checked via type assertion during input resolution. Modes that use semantic input binding implement this; `UIModeManager` automatically resolves actions each frame.

---

## Simplified Data Structs

Sometimes a minimal data struct avoids a circular import where a full interface would be overkill.

| Struct | File | Why |
|--------|------|-----|
| `VictoryInfo` | `tactical/combat/battlelog/battle_recorder.go` | `BattleRecorder.Finalize()` needs victory data but cannot import `combatservices` (which owns `VictoryConditionResult`). `VictoryInfo` is a minimal struct with only `RoundsCompleted`, `VictorFaction`, and `VictorName`. |

---

## Wiring: Where Concrete Implementations Are Injected

The "glue" code that connects interfaces to implementations lives in the setup layer.

### `game_main/setup.go`

- `setupUICore()` creates `GameModeCoordinator` and passes it to `encounter.NewEncounterService()` as the `CombatTransitionHandler`.
- The returned `EncounterService` is passed through mode registration so GUI code receives it as `combat.EncounterCallbacks`.

### `gamesetup/moderegistry.go`

- `RegisterTacticalModes()`, `RegisterOverworldModes()`, and `RegisterRoguelikeTacticalModes()` register concrete mode implementations with the coordinator.
- `RegisterOverworldModes()` creates a `startCombat` closure that calls `combatlifecycle.ExecuteCombatStart(encounterService, ecsManager, starter)`, passing `encounterService` as the `CombatTransitioner`.
- `EncounterService` is passed to `guicombat.NewCombatMode()` where it becomes the `combat.EncounterCallbacks` in `CombatModeDeps`.

### `mind/raid/raidrunner.go`

- `NewRaidRunner()` sets `encounterService.PostCombatCallback` — a function closure that calls `rr.ResolveEncounter()` when combat ends. This is callback injection rather than an interface, but serves the same decoupling purpose: `encounter` does not import `raid`.
- `TriggerRaidEncounter()` calls `combatlifecycle.ExecuteCombatStart(rr.encounterService, rr.manager, starter)`, passing the encounter service as `CombatTransitioner` and a `RaidCombatStarter` as `CombatStarter`.

---

## Import Direction Summary

```
game_main/setup.go          (wiring layer — imports everything)
gamesetup/moderegistry.go   (wiring layer — imports everything)
    │
    ├── gui/framework        ← defines UIMode; satisfies CombatTransitionHandler
    ├── gui/guicombat        ← imports mind/ai for wiring only (SetupCombatAI); uses combatservices interfaces for threat
    ├── gui/guispells        ← defines SpellCastingDeps
    ├── gui/guiartifacts     ← defines ArtifactActivationDeps
    │
    ├── tactical/combat      ← defines CombatStarter, CombatTransitioner, EncounterCallbacks, CombatCleaner
    ├── mind/combatlifecycle ← defines CombatResolver; calls interfaces from tactical/combat
    │
    ├── mind/encounter       ← defines CombatTransitionHandler; satisfies CombatTransitioner, EncounterCallbacks
    ├── mind/raid            ← satisfies CombatStarter, CombatResolver
    │
    ├── visual/rendering     ← defines SquadInfoProvider, UnitInfoProvider
    ├── world/worldmap       ← defines MapGenerator
    ├── gear                 ← defines ArtifactBehavior
    └── savesystem           ← defines SaveChunk, Validatable
```

Arrows flow downward (toward fewer dependencies). The wiring layer at the top is the only code that sees both sides of each boundary.
