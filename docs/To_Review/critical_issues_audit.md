# Critical Issues Audit — 2026-03-12

Comprehensive assessment of all issues from `critical_patterns_and_gotchas.md` plus new findings from a deep codebase audit covering entity lifecycle, nil safety, combat/AI/effects systems, and ECS pattern compliance.

---



### NEW-2. Destroyed Factions Retain Turn Order Slots (MEDIUM)
**File:** `tactical/combat/turnmanager.go:144-170`

`EndTurn()` increments `CurrentTurnIndex` through `TurnOrder` without checking if the next faction still exists or has any living squads:

```go
func (tm *TurnManager) EndTurn() error {
    turnState.CurrentTurnIndex++
    if turnState.CurrentTurnIndex >= len(turnState.TurnOrder) {
        turnState.CurrentTurnIndex = 0
        turnState.CurrentRound++
    }
    newFactionID := turnState.TurnOrder[turnState.CurrentTurnIndex]
    if err := tm.ResetSquadActions(newFactionID); err != nil {
        return fmt.Errorf("failed to reset squad actions: %w", err)
    }
    // ...
}
```

**Impact:** When a faction is eliminated (all squads destroyed), its slot remains in `TurnOrder`. Each round, the destroyed faction gets a wasted turn cycle — `ResetSquadActions` runs for a faction with no squads (finding nothing to reset), and then the next `EndTurn` call advances past it. No crash, but:
- Wasted processing each round
- AI/UI flow receives a turn-end event for a non-existent faction
- If `onTurnEnd` callback logic assumes the current faction is alive, it could misbehave

**Fix:** Skip destroyed factions in `EndTurn`:
```go
func (tm *TurnManager) EndTurn() error {
    // ... advance index ...
    // Skip destroyed factions
    for attempts := 0; attempts < len(turnState.TurnOrder); attempts++ {
        newFactionID := turnState.TurnOrder[turnState.CurrentTurnIndex]
        if tm.factionHasLivingSquads(newFactionID) {
            break
        }
        turnState.CurrentTurnIndex++
        if turnState.CurrentTurnIndex >= len(turnState.TurnOrder) {
            turnState.CurrentTurnIndex = 0
            turnState.CurrentRound++
        }
    }
    // ... rest unchanged
}
```

Or prune `TurnOrder` when a faction is destroyed.

---


### NEW-4. Manual Coordinate Indexing in Map Generators (LOW-MEDIUM)
**Severity:** Low-Medium (24 instances across 4 files)

**Files affected:**
- `world/worldmap/gen_overworld.go` — 2 instances
- `world/worldmap/gen_military_base.go` — 8 instances
- `world/worldmap/gen_cavern.go` — 8 instances
- `world/worldmap/gen_helpers.go` — 6 instances

All use `y*width + x` (via `positionToIndex()` or inline) instead of `coords.CoordManager.LogicalToIndex()`.

**Why lower severity than expected:** These generators operate on local arrays sized to `width*height` parameters, and the width comes from the same source as `CoordManager.dungeonWidth`. The math is equivalent as long as widths match. Some generators even do bounds checking.

**Risk:** Fragile implicit coupling. If a generator is ever called with a width that differs from CoordManager, indices silently mismatch. CLAUDE.md explicitly warns against this pattern.

**Recommendation:** Migrate to `coords.CoordManager.LogicalToIndex()` when touching these files. Not urgent.

---

### NEW-5. Stored `*ecs.Entity` Pointers in Influence System (LOW-MEDIUM)
**File:** `overworld/influence/queries.go:12-24`

```go
type NodePair struct {
    EntityA  *ecs.Entity  // Should be EntityID
    EntityB  *ecs.Entity  // Should be EntityID
    Distance int
}

type overworldNode struct {
    Entity   *ecs.Entity  // Should be EntityID
    EntityID ecs.EntityID
    Pos      coords.LogicalPosition
    Radius   int
}
```

Violates ECS best practice: "EntityID Only — Never store `*ecs.Entity` pointers." If an entity is disposed while a `NodePair` is held, the pointer becomes dangling.

**Mitigating factor:** These are short-lived local structs created and consumed within `FindOverlappingNodes()`. No long-term storage. The `overworldNode` struct even redundantly stores both the pointer and the ID.

**Recommendation:** Refactor to use `ecs.EntityID` when next modifying this package.

---

### NEW-6. Missing Nil Checks on Tag-Guaranteed Components in squadqueries.go (LOW)
**File:** `tactical/squads/squadqueries.go` — lines 19-20, 48-50, 67-69, 83-85

Functions like `GetUnitIDsAtGridPosition`, `GetUnitIDsInSquad`, `GetSquadEntity`, and `GetLeaderID` dereference `GetComponentType` results without nil checks:

```go
// Line 19-20
memberData := common.GetComponentType[*SquadMemberData](unitEntity, SquadMemberComponent)
if memberData.SquadID != squadID {  // Dereference without nil check
```

**Why this is LOW:** The tag query (`SquadMemberTag`) guarantees the component exists — that's how tags work in this ECS. A nil result here would indicate a fundamental ECS invariant violation, not a realistic runtime scenario.

**Recommendation:** Defensive nil checks would be nice for robustness but are not urgent. The tag system is the guard.

---

### NEW-7. disposeEntitiesByTag Skips Position Cleanup (LOW)
**File:** `tactical/combatservices/combat_service.go:400-407`

```go
func (cs *CombatService) disposeEntitiesByTag(tag ecs.Tag, name string) {
    for _, result := range cs.EntityManager.World.Query(tag) {
        cs.EntityManager.World.DisposeEntities(result.Entity)  // Direct dispose
    }
}
```

Uses `World.DisposeEntities()` directly instead of `CleanDisposeEntity()`. Called for `FactionTag`, `ActionStateTag`, and `TurnStateTag` entities.

**Mitigating factor:** These entity types are combat-only metadata that do NOT have position components. The code is safe for current usage. `disposeEnemySquads()` (line 410-414) correctly uses `CleanDisposeEntity` for positioned entities.

**Recommendation:** Add a defensive position check for future-proofing:
```go
pos := common.GetComponentType[*coords.LogicalPosition](result.Entity, common.PositionComponent)
if pos != nil {
    cs.EntityManager.CleanDisposeEntity(result.Entity, pos)
} else {
    cs.EntityManager.World.DisposeEntities(result.Entity)
}
```

---

## Part 3: False Positives Investigated and Excluded

These were flagged during the audit but verified as NOT actual issues:

| Claim | Verdict | Reason |
|-------|---------|--------|
| Double-disposal of enemy units in combat cleanup | **FALSE** | `disposeEnemySquads` disposes squad entities, `disposeEnemyUnits` disposes unit entities. Different entity types, no overlap. |
| `GetSquadHealthPercent` division by zero | **FALSE** | Already guards against 0 units (`len(unitIDs) == 0`) and 0 alive units. |
| Save system misses active effects | **Not a bug** | Game saves between combats, not during. Effects are combat-scoped and naturally cleared. |
| Healing can exceed max HP | **FALSE** | Already clamped at `squadcombat.go:991-993`. |

---

## Summary Table

| # | Issue | Status | Severity | Category |
|---|-------|--------|----------|----------|
| 1 | GetComponentType swallows panics | **RESOLVED** | — | Error handling |
| 2 | MaxHealth cached field desync | **RESOLVED** | — | Data integrity |
| 3 | Position dual-state invariant | **RESOLVED** | — | Entity lifecycle |
| 4 | ParseStatType silent fallback | **RESOLVED** | — | Error handling |
| 5 | Unit component composition | **RESOLVED** | — | ECS patterns |
| 6 | Boot sequence order | Documented | Info | Architecture |
| 7 | Blank imports for registration | Documented | Info | Architecture |
| 8 | IsOpaque() no bounds check | **STILL OPEN** | Medium | Crash risk |
| 9 | UnassignUnitFromSquad orphans leader | **STILL OPEN** | Medium | Logic bug |
| 10 | JSON requirement levels | Documented | Info | Configuration |
| 11 | DEBUG_MODE hardcoded true | Open (by design) | Low | Release prep |
| NEW-1 | Effect system no stat floor clamping | **NEW** | **High** | Game logic |
| NEW-2 | Destroyed factions retain turn slots | **NEW** | Medium | Combat flow |
| NEW-3 | Spell casting no combat-context check | **NEW** | Medium | Defense in depth |
| NEW-4 | Manual coordinate indexing in generators | **NEW** | Low-Medium | Fragile coupling |
| NEW-5 | Entity pointers in influence system | **NEW** | Low-Medium | ECS anti-pattern |
| NEW-6 | Missing nil checks in squadqueries | **NEW** | Low | Defensive coding |
| NEW-7 | disposeEntitiesByTag skips position cleanup | **NEW** | Low | Defensive coding |

---

## Priority Fix Order

1. **NEW-1** (High) — Stat floor clamping. Negative stats cascade into broken derived values. Quick fix.
2. **#8** (Medium) — IsOpaque bounds check. One-line fix prevents potential panic.
3. **#9** (Medium) — Leader orphaning. Add leader component check before unassign.
4. **NEW-2** (Medium) — Turn order pruning. Prevents wasted cycles and potential AI confusion.
5. **NEW-3** (Medium) — Spell combat-context validation. Defense in depth.
6. **NEW-4 through NEW-7** (Low) — Address opportunistically when modifying affected files.
