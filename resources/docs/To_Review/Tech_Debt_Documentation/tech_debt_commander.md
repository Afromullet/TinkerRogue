# Technical Debt Report — `tactical/commander` Package

**Date:** 2026-05-16
**Scope:** 7 files, 483 LOC, 31 external consumers. No test files exist.

---

## 1. Debt Inventory

### Critical

**C1 — Save/load asymmetry: lost components on load**
- Files: `setup/savesystem/chunks/commander_chunk.go:113-142` vs `tactical/commander/system.go:38-46`
- `CreateCommander` attaches `RenderableComponent` and `ProgressionComponent`. `CommanderChunk.Load` rebuilds the entity without either. After a save→load cycle, commanders are invisible on the overworld and their unlocked perks/spells are gone. `progression_chunk.go` may compensate for progression, but rendering definitely won't be restored — the input handler/renderer assume the component exists.
- **Risk:** Data loss on every save/load. Probably masked because saving isn't exercised in normal play yet.
- **Impact:** Will surface as a P0 the moment save/load ships.

**C2 — No tests in package**
- Directory: `tactical/commander/`
- Zero `_test.go` files. The package owns turn state mutation (`EndTurn`, `StartNewTurn`), movement validation (`MoveCommander`, `CanMoveTo`), and roster invariants (`AddCommander` rejects duplicates / over-cap). All untested. Combat sibling has ~10 test files; commander has 0.
- **Impact:** Every commander-touching change is unverified. Refactoring is high-risk.

### High

**H1 — Movement system duplicates combat movement**
- Files: `tactical/commander/movement.go:75-132` vs `tactical/combat/combatcore/combatmovementsystem.go:42-155`
- `CanMoveTo`, `GetValidMovementTiles`, and the Chebyshev flood-fill loop are 80% structurally identical to `CombatMovementSystem`. Each has its own `posSystem`, its own action-state lookup, its own movement-cost decrement. Differences are real but small:
  - commander: 1 occupant-per-tile + walkability.
  - combat: squad-vs-squad + ZoC + post-move hook.
- **Cost:** Any tweak (diagonal cost, terrain cost, pathfinding upgrade) must be done twice. ~80 LOC duplicated.
- **Suggested fix:** Extract a `MovementCalculator{distanceFn, canEnterFn}` in `core/common` that both systems wrap with their domain rules.

**H2 — Stored EntityID inside its own component**
- File: `tactical/commander/components.go:24, 31`
```go
type CommanderData struct { CommanderID ecs.EntityID; ... }
type CommanderActionStateData struct { CommanderID ecs.EntityID; ... }
```
- Both fields are written by `CreateCommander` and **never read anywhere in the codebase** (grep confirmed: no `.CommanderID` reads on these structs — the matches are on `raidState.CommanderID`). Same anti-pattern exists in combat's `ActionStateData.SquadID`. Per CLAUDE.md ECS rules, the entity already knows its own ID via `entity.GetID()`.
- **Cost:** Dead fields, dead writes, misleading API surface, redundant remapping risk on save/load.

**H3 — Turn-state singleton via O(N) query**
- File: `tactical/commander/turnstate.go:22-28`
```go
results := manager.World.Query(OverworldTurnTag)
if len(results) == 0 { return nil }
return common.GetComponentType[...](results[0].Entity, ...)
```
- Singleton lookup runs a full tag query on every call. Called from `EndTurn`, renderer, action handler, GUI mode update — likely several times per frame. Combat's equivalent (`TurnStateData`) has the same pattern but is also unfixed. The package already uses `CommanderView` for the commander tag — same pattern would work here.
- **Suggested fix:** Cache the singleton entity ID at creation (or use a package-level `*ecs.View` like `CommanderView`).

### Medium

**M1 — Magic-number default in `StartNewTurn`**
- File: `tactical/commander/turnstate.go:55-58`
```go
if attr != nil {
    actionState.MovementRemaining = attr.GetMovementSpeed()
} else {
    actionState.MovementRemaining = 3 // Default
}
```
- The "3" is an undeclared invariant. Any commander without `AttributeComponent` silently gets 3 movement. `CreateCommander` always adds `AttributeComponent`, so this branch is unreachable in practice — making it either dead defensive code or a hidden bug-magnet if a future code path bypasses `CreateCommander` (the save/load already does — see C1).
- **Fix:** Either delete the fallback (trust the invariant) or move "3" to `templates.GameConfig.Commander` as `DefaultMovementSpeed`.

**M2 — Dead query functions**
- File: `tactical/commander/queries.go:24, 38`
- `GetCommanderEntity` — defined, no external callers.
- `IsCommander` — defined, no external callers.
- Pure dead code. Delete or document why kept.

**M3 — `MovementSpeed` set twice in `CreateCommander`**
- File: `tactical/commander/system.go:32-44`
- The constructor sets `CommanderActionStateData.MovementRemaining = movementSpeed` AND `Attributes.MovementSpeed = movementSpeed`. After the first `EndTurn → StartNewTurn`, the action state is overwritten from `Attributes.MovementSpeed`. So the initial `MovementRemaining` argument is correct, but `MovementSpeed` is the source of truth — having both as a constructor parameter is misleading. Should be one input, not "movementSpeed" passed twice.

**M4 — Roster API split across types**
- Files: `tactical/commander/roster.go` + `queries.go:43`
- `AddCommander`/`RemoveCommander` are methods on `CommanderRosterData`, but `GetAllCommanders(playerID, manager)` is a package function that hides the roster lookup. Inconsistent — callers sometimes go through the roster, sometimes through the package. Pick one pattern (recommend: package functions taking `playerID`, since callers always have the player).

**M5 — `EndTurn` couples package to `tick.AdvanceTick`**
- File: `tactical/commander/turnstate.go:64-81`
- The commander package imports `campaign/overworld/tick` and `common.PlayerData`. Turn advancement is currently "commander does the tick" which inverts the natural dependency (overworld should drive the turn loop, not commander). This wedges raid/event handling into the commander package's responsibility.
- **Cost:** `commander` can't be reused outside of overworld context (e.g., a tutorial or test harness needs a `tick` system).

### Low

**L1 — Inconsistent receiver naming**
- File: `tactical/commander/movement.go`
- Constructor uses `cms`, methods use `cms` AND `ms` for the same struct (`CommanderMovementSystem`). Cosmetic, but `go fmt`/`go vet` allow this so it stays.

**L2 — `fmt.Printf` in production initial-commander path**
- File: `setup/gamesetup/initial_commanders.go:62`
- Not strictly in the commander package, but the only caller of `CreateCommander` prints to stdout. Should be a structured log.

---

## 2. Impact Assessment

| ID | Category | Effort | Bug Risk | Priority |
|----|----------|--------|----------|----------|
| C1 | Save/load asymmetry | 2h | Will break save/load on ship | P0 |
| C2 | No tests | 8h initial + ongoing | Every change unverified | P0 |
| H1 | Movement duplication | 6h extract, 2h migrate | Drift; tweaks happen 2x | P1 |
| H2 | Dead EntityID fields | 30min | Misleading API; save/load complexity | P1 |
| H3 | O(N) singleton query | 30min | Perf — called every frame | P1 |
| M1 | Magic-number default | 15min | Hidden bug surface | P2 |
| M2 | Dead queries | 5min | Code rot | P2 |
| M3 | Double-set movement speed | 20min | Constructor confusion | P2 |
| M4 | Roster API split | 1h | Inconsistency | P3 |
| M5 | tick coupling in commander | 3h | Reuse blocked; layering inverted | P3 |

---

## 3. Quick Wins (this sprint, ≤2 hours total)

1. **Delete `CommanderID` field from `CommanderData` and `CommanderActionStateData`** (H2) — confirmed zero readers. Removes a misleading invariant and shrinks save format.
2. **Delete `GetCommanderEntity` and `IsCommander`** (M2) — unused. 5 minutes.
3. **Cache `OverworldTurnState` entity ID at creation** (H3) — return the ID from `CreateOverworldTurnState`, store as package var, replace the query in `GetOverworldTurnState`.
4. **Remove or extract the magic `3` default** (M1) — either delete the dead else-branch or move to config.

---

## 4. Sprint 1-2: Fix Save/Load + Add Tests

**Save/load fix (C1, ~2h):**
- Restore `RenderableComponent` on load (re-load `config.PlayerImagePath`, attach with `Visible: true`).
- Restore `ProgressionComponent` if `progression_chunk.go` doesn't already (verify).
- Add an integration test: create commander → save → load → assert all components present.

**Test scaffolding (C2, ~8h):**
- `roster_test.go`: AddCommander over cap, AddCommander duplicate, RemoveCommander missing, RemoveCommander present.
- `turnstate_test.go`: StartNewTurn resets all action states, EndTurn increments counter (mock `tick`).
- `movement_test.go`: MoveCommander insufficient movement, occupied tile, valid tiles flood-fill bounds, exact MovementRemaining=cost edge case.
- `queries_test.go`: FindCommanderForSquad happy + not-found, GetCommanderAt with stacked entities.

---

## 5. Sprint 3+: Structural Cleanup

**Extract movement helper (H1):**
Create `core/common/movement_grid.go`:
```go
type GridMovement struct {
    PositionSystem *PositionSystem
    CanEnter       func(entityID ecs.EntityID, pos coords.LogicalPosition) bool
    DistanceFn     func(a, b coords.LogicalPosition) int
}
func (g *GridMovement) ValidTilesInRange(from coords.LogicalPosition, rng int) []coords.LogicalPosition
```
Migrate both `CommanderMovementSystem` and `CombatMovementSystem` to delegate the flood-fill.

**Decouple `EndTurn` from `tick` (M5):**
Move `EndTurn` out of `commander` into `campaign/overworld/overworldturn.go`. Commander package keeps `StartNewTurn` (pure commander state reset) and exposes it; overworld orchestrates the sequence.

---

## 6. Prevention

- **Mandatory tests for new components:** any new `*Component` requires a `_test.go` covering create/access/dispose.
- **Save/load symmetry checklist:** when `CreateX` adds component Y, the chunk's `Load` must add Y or document why not.
- **ECS lint rule (manual for now):** components must not store the EntityID of the entity they're attached to.

---

## 7. ROI Summary

| Investment | Return |
|-----------|--------|
| Sprint 1: 10h (quick wins + save/load + initial tests) | Eliminates P0 bug, establishes test baseline |
| Sprint 2-3: 15h (movement extract + decouple tick) | Halves future movement-tweak cost; unblocks commander reuse |
| **Total: ~25h** | Removes top 3 risk items; package becomes maintainable |

The package is small (483 LOC) and recently written — debt is shallow and cheap to fix now. The save/load asymmetry (C1) is the only item that **must** be fixed before save/load ships.
