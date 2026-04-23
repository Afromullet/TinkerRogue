# Entity Cleanup Audit & Recommendations

**Date:** 2026-03-12
**Scope:** All ECS entity types in TinkerRogue, reviewed against `ENTITY_REFERENCE.md`

---

## ECS Cleanup Rules

The bytearena/ecs library provides `World.DisposeEntities(entity)` to remove an entity and all its components. TinkerRogue adds `GlobalPositionSystem` (spatial grid), so the correct patterns are:

| Situation | Method | Location |
|-----------|--------|----------|
| Entity WITH position | `manager.CleanDisposeEntity(entity, pos)` | `common/ecsutil.go:193` |
| Entity WITHOUT position | `manager.World.DisposeEntities(entity)` | bytearena/ecs |
| Strip position but keep entity | `manager.UnregisterEntityPosition(entity)` | `common/ecsutil.go:176` |

---

## Entity Inventory

### Properly Cleaned Up (No Action Needed)

| Entity | Has Position | Cleanup Method | Code Location |
|--------|-------------|----------------|---------------|
| Squad Entity | Yes | `CleanDisposeEntity` via `DisposeSquadAndUnits` | `squads/squadcreation.go:473` |
| Unit Entity | Yes | `CleanDisposeEntity` via `DisposeDeadUnitsInSquad` / `disposeEnemyUnits` | `squads/squadcreation.go:451`, `combatservices/combat_service.go:420` |
| Overworld Node | Yes | `DestroyNode` → `CleanDisposeEntity` | `overworld/node/system.go:108` |
| Combat Faction | No | `disposeEntitiesByTag` → `World.DisposeEntities` | `combatservices/combat_service.go:399` |
| Turn State | No | `disposeEntitiesByTag` → `World.DisposeEntities` | `combatservices/combat_service.go:399` |
| Action State | No | `disposeEntitiesByTag` → `World.DisposeEntities` | `combatservices/combat_service.go:399` |

### Game-Session Singletons (No Cleanup Needed)

These live for the entire game session and are never disposed — this is correct.

| Entity | Notes |
|--------|-------|
| Player Entity | One per game, persists always |
| Tick State Entity | Singleton counter |
| Victory State Entity | Singleton |
| Overworld Turn State Entity | Singleton |

### Session-Persistent Entities (Conditional Cleanup)

| Entity | Has Position | Current Cleanup | Notes |
|--------|-------------|-----------------|-------|
| Commander Entity | Yes | **None** | If a commander dies mid-game, no disposal code exists. Position would leak in `GlobalPositionSystem`. Currently commanders cannot die, so this is defensive. |
| Overworld Faction Entity | No | **None** | If a faction is eliminated, entity persists as orphan. Low severity since no position component. |

---

## Missing Cleanup (Bugs)

### Priority 1: Encounter Entity Leak

**Severity:** Medium
**Impact:** One entity leaked per combat encounter. Accumulates over a game session.

**Creation:** `mind/encounter/encounter_trigger.go:57` (`createOverworldEncounter`) and `:136` (`TriggerGarrisonDefense`)
**Expected disposal:** After combat ends in `EncounterService.ExitCombat` (`mind/encounter/encounter_service.go:218`)
**Current behavior:** Encounter entity is marked `IsDefeated` and sprite is hidden, but entity is **never disposed** from the ECS world.

**Recommended fix:**
After `teardown.TeardownCombat(enemySquadIDs)` in `ExitCombat`, dispose the encounter entity:

```go
// In ExitCombat, after Step 3 (combat cleanup):
encounterEntity := es.manager.FindEntityByID(es.activeEncounter.EncounterID)
if encounterEntity != nil {
    es.manager.World.DisposeEntities(encounterEntity) // No position component
}
```

**Consideration:** Check if any code queries encounter entities after combat (e.g., history, statistics). If encounter history needs to persist, store the relevant data in a non-ECS structure before disposing.

---

### Priority 2: Raid Infrastructure Entity Leaks

**Severity:** Medium (collectively high — many entities per raid)
**Impact:** Each raid creates: 1 RaidState + N FloorStates + N AlertDatas + many RoomDatas + 1 Deployment. None are disposed when the raid ends.

#### Raid State Entity

**Creation:** `mind/raid/garrison.go:19` (`GenerateGarrison`)
**Expected disposal:** When raid finishes in `RaidRunner.finishRaid` (`mind/raid/raidrunner.go:349`)
**Current behavior:** `finishRaid` clears the callback and zeroes `raidEntityID`, but does **not** dispose the entity.

**Recommended fix:**
```go
func (rr *RaidRunner) finishRaid(status RaidStatus) {
    // Dispose raid entity and all child entities (floors, rooms, alerts, deployment)
    disposeRaidEntities(rr.manager, rr.raidEntityID)

    rr.encounterService.PostCombatCallback = nil
    rr.raidEntityID = 0
}
```

#### Floor State Entity

**Creation:** `mind/raid/garrison.go:94` (`generateFloor`)
**Expected disposal:** When raid finishes
**Current behavior:** Never disposed. Floor IDs are stored in `RaidStateData.FloorIDs`.

**Recommended fix:** Dispose as part of raid cleanup. The RaidStateData contains `FloorIDs` — iterate and dispose each.

#### Alert Data Entity

**Creation:** `mind/raid/garrison.go:47` (`generateFloor`)
**Expected disposal:** When raid finishes
**Current behavior:** Never disposed. Alert entity ID stored in `FloorStateData.AlertEntityID`.

**Recommended fix:** Dispose as part of floor cleanup. Each floor's `AlertEntityID` points to its alert entity.

#### Room Data Entity

**Creation:** `mind/raid/floorgraph.go` (via `buildFloorGraph`)
**Expected disposal:** When raid finishes
**Current behavior:** Never disposed. Room entity IDs stored in `FloorStateData.RoomEntityIDs`.

**Recommended fix:** Dispose as part of floor cleanup. Each floor's `RoomEntityIDs` contains all room entity IDs.

#### Deployment Entity

**Creation:** `mind/raid/deployment.go:35-44` (`SetDeployment` — creates or reuses via query)
**Expected disposal:** When raid finishes
**Current behavior:** Reused across encounters within a raid, but never disposed after.

**Recommended fix:** Dispose as part of raid cleanup via `DeploymentTag` query.

#### Consolidated Raid Cleanup Function

Create a single `disposeRaidEntities` function in `mind/raid/`:

```go
// disposeRaidEntities disposes the raid state entity and all its child entities
// (floors, rooms, alerts, deployment). Call this when a raid ends.
func disposeRaidEntities(manager *common.EntityManager, raidEntityID ecs.EntityID) {
    raidEntity := manager.FindEntityByID(raidEntityID)
    if raidEntity == nil {
        return
    }
    raidData := common.GetComponentType[*RaidStateData](raidEntity, RaidStateComponent)
    if raidData == nil {
        return
    }

    // Dispose each floor and its children
    for _, floorID := range raidData.FloorIDs {
        floorEntity := manager.FindEntityByID(floorID)
        if floorEntity == nil {
            continue
        }
        floorData := common.GetComponentType[*FloorStateData](floorEntity, FloorStateComponent)
        if floorData != nil {
            // Dispose alert entity
            if floorData.AlertEntityID != 0 {
                if alertEntity := manager.FindEntityByID(floorData.AlertEntityID); alertEntity != nil {
                    manager.World.DisposeEntities(alertEntity)
                }
            }
            // Dispose room entities
            for _, roomID := range floorData.RoomEntityIDs {
                if roomEntity := manager.FindEntityByID(roomID); roomEntity != nil {
                    manager.World.DisposeEntities(roomEntity)
                }
            }
        }
        manager.World.DisposeEntities(floorEntity)
    }

    // Dispose deployment entity (query-based, since it's a singleton)
    for _, result := range manager.World.Query(DeploymentTag) {
        manager.World.DisposeEntities(result.Entity)
    }

    // Dispose the raid state entity itself
    manager.World.DisposeEntities(raidEntity)
}
```

**Note:** None of these raid entities have position components, so `World.DisposeEntities` is sufficient (no need for `CleanDisposeEntity`).

---

### Priority 3: Garrison Squad Edge Case

**Severity:** Medium
**Impact:** Garrison squads that survive combat (returned to nodes via `returnGarrisonSquadsToNode`) may not be cleaned when the node is eventually destroyed.

**Current flow:**
1. Garrison squads created: `mind/raid/garrison.go:108` (`InstantiateGarrisonSquad`)
2. Enemy garrison squads in combat: properly disposed via `TeardownCombat` → `disposeEnemySquads`
3. Surviving garrison squads returned to node: `encounter_service.go:259` (`returnGarrisonSquadsToNode`)
4. Node destroyed later: `DestroyNode` (`overworld/node/system.go:108`) — **only disposes the node entity itself**, not associated garrison squads

**Recommended fix:**
In `DestroyNode` (or a wrapper), also dispose any garrison squads associated with the node. This requires knowing which squads belong to a node — check if `NodeData` or similar tracks garrison squad IDs.

**Investigation needed:** Verify whether `NodeData` stores garrison squad references and whether `DestroyNode` is the right place for this cleanup.

---

### Priority 4: Creature Entity (Standalone)

**Severity:** Low
**Impact:** Standalone creatures (not in squads) have no disposal path. Currently unused or rare.

**Recommended fix:** If standalone creatures are used, add disposal similar to unit disposal. If not used, no action needed — just note for future implementation.

---

## Code Smell: `disposeEntitiesByTag` Safety

**Location:** `combatservices/combat_service.go:399-407`

```go
func (cs *CombatService) disposeEntitiesByTag(tag ecs.Tag) {
    for _, result := range cs.Manager.World.Query(tag) {
        cs.Manager.World.DisposeEntities(result.Entity)
    }
}
```

**Issue:** Uses `World.DisposeEntities` directly instead of checking for position components first. Currently safe because Faction, ActionState, and TurnState entities never have positions, but violates the safety pattern.

**Recommended fix (defensive):**
```go
func (cs *CombatService) disposeEntitiesByTag(tag ecs.Tag) {
    for _, result := range cs.Manager.World.Query(tag) {
        pos := common.GetComponentType[*coords.LogicalPosition](result.Entity, common.PositionComponent)
        if pos != nil {
            cs.Manager.CleanDisposeEntity(result.Entity, pos)
        } else {
            cs.Manager.World.DisposeEntities(result.Entity)
        }
    }
}
```

**Priority:** Low. This is a defensive improvement, not a bug fix.

---

## Implementation Order

1. **Raid cleanup function** (Priority 2) — Highest entity count per incident. Create `disposeRaidEntities` and call from `finishRaid`.
2. **Encounter disposal** (Priority 1) — Simple fix in `ExitCombat`. Verify no post-combat queries first.
3. **Garrison squad node cleanup** (Priority 3) — Investigate `NodeData` structure, then implement.
4. **`disposeEntitiesByTag` safety** (Code smell) — Defensive, low risk either way.
5. **Commander/Faction orphans** (Conditional) — Only if these entities can actually be destroyed in gameplay.
6. **Creature disposal** (Priority 4) — Only if standalone creatures are implemented.

---

## Verification Checklist

After implementing fixes:
- [ ] Run `go test ./...` — all tests pass
- [ ] Play through a full raid (multiple floors) and verify no orphan entities remain
- [ ] Trigger 3+ overworld encounters and verify encounter entities are disposed
- [ ] Check that garrison squads surviving combat don't leak when nodes are destroyed
- [ ] Verify save/load still works (raid entities may be expected to persist for save state)

---

## Save/Load Consideration

**Important:** Before disposing raid entities in `finishRaid`, verify that save/load does not depend on these entities persisting after a raid ends. The `RestoreFromSave` function (`raidrunner.go:344`) sets `raidEntityID` from a saved value, implying raid entities should persist while a raid is in progress but can be disposed once the raid completes. Ensure `finishRaid` is only called after the raid is truly complete, not during save checkpoints.
