# Critical Issues Audit — 2026-03-12

Comprehensive assessment of all issues from `critical_patterns_and_gotchas.md` plus new findings from a deep codebase audit covering entity lifecycle, nil safety, combat/AI/effects systems, and ECS pattern compliance.

---

## Part 1: Status of Previously Documented Issues

### 8. IsOpaque() Has No Bounds Check — STILL OPEN
**File:** `world/worldmap/dungeongen.go:249-252`
**Severity:** Medium

```go
func (gameMap GameMap) IsOpaque(x, y int) bool {
    logicalPos := coords.LogicalPosition{X: x, Y: y}
    idx := coords.CoordManager.LogicalToIndex(logicalPos)
    return gameMap.Tiles[idx].TileType == WALL  // No bounds check!
}
```

Compare with `GetBiomeAt()` (line 240-244) which correctly checks `idx < 0 || idx >= len(gm.BiomeMap)`. If FOV or line-of-sight calculations pass edge coordinates, this will panic with index out of range.

**Fix:** Add the same bounds guard:
```go
func (gameMap GameMap) IsOpaque(x, y int) bool {
    logicalPos := coords.LogicalPosition{X: x, Y: y}
    idx := coords.CoordManager.LogicalToIndex(logicalPos)
    if idx < 0 || idx >= len(gameMap.Tiles) {
        return true // Treat out-of-bounds as opaque (blocks vision)
    }
    return gameMap.Tiles[idx].TileType == WALL
}
```

### 9. UnassignUnitFromSquad Orphans Leader Components — STILL OPEN
**File:** `tactical/squads/squadcreation.go:211-232`
**Severity:** Medium

`UnassignUnitFromSquad()` removes `SquadMemberComponent` and resets grid position, but does NOT check if the unit is the squad leader. If the leader is unassigned, it retains `LeaderComponent`, `AbilitySlotComponent`, and `CooldownTrackerComponent`. Reassigning this unit to another squad could create dual-leader bugs.

**Fix:** Check for leader status and strip leader components before removal:
```go
func UnassignUnitFromSquad(unitEntityID ecs.EntityID, manager *common.EntityManager) error {
    // ... existing validation ...

    // If this unit is the leader, strip leader components first
    if unitEntity.HasComponent(LeaderComponent) {
        RemoveLeaderComponents(unitEntity)
    }

    unitEntity.RemoveComponent(SquadMemberComponent)
    // ... rest unchanged ...
}
```

### 10. Soft vs Hard JSON Requirements — DOCUMENTED, NO CODE CHANGE NEEDED
Still accurate. Only `mapgenconfig.json` is a soft requirement.

### 11. DEBUG_MODE and ENABLE_BENCHMARKING Hardcoded True — OPEN (by design)
**File:** `config/config.go`

Both flags are still `true`. Expected during development — must be flipped for release builds.

---

## Part 2: New Issues Found

### NEW-1. Effect System Has No Stat Floor Clamping (HIGH)
**File:** `tactical/effects/system.go:136-154`

`applyModifierToStat()` adds modifiers directly to stat fields with no minimum value enforcement:

```go
func applyModifierToStat(attr *common.Attributes, stat StatType, modifier int) {
    switch stat {
    case StatStrength:
        attr.Strength += modifier   // Can go negative!
    case StatDexterity:
        attr.Dexterity += modifier  // Can go negative!
    // ... all stats unclamped
    }
}
```

**Impact — negative stats cascade through derived calculations:**

| Derived Stat | Formula | Effect of Negative Base |
|---|---|---|
| `GetMaxHealth()` | `20 + (Strength * 2)` | Strength=-5 → MaxHealth=10. If current HP > new max, unit has "extra" HP that won't regenerate. If debuff is extreme, MaxHealth could go below 0 (Strength < -10). |
| `GetPhysicalDamage()` | `(Strength / 2) + (Weapon * 2)` | Negative damage values — integer division truncates toward zero so Strength=-1 gives 0, but Strength=-2 gives -1. Negative damage would heal enemies or cause underflow depending on callers. |
| `GetPhysicalResistance()` | `(Strength / 4) + (Armor * 3 / 2)` | Negative resistance could amplify incoming damage. |
| `GetMagicDamage()` | `Magic * 3` | Negative magic damage. |
| `GetMovementSpeed()` | Guarded: `if a.MovementSpeed <= 0 { return default }` | **Safe** — but silently ignores debuff. |
| `GetAttackRange()` | Guarded: `if a.AttackRange <= 0 { return default }` | **Safe** — but silently ignores debuff. |

**Concrete scenario:** A debuff spell applies StatStrength modifier of -5 to a unit with Strength=3. Result: Strength=-2, MaxHealth=16 (down from 26), PhysicalDamage=-1, PhysicalResistance is reduced. If a second effect applies more negative strength, MaxHealth could reach single digits or below.

**Fix:** Clamp stats to a minimum of 0 (or 1 for stats that are divisors):
```go
func applyModifierToStat(attr *common.Attributes, stat StatType, modifier int) {
    switch stat {
    case StatStrength:
        attr.Strength = max(0, attr.Strength + modifier)
    case StatDexterity:
        attr.Dexterity = max(0, attr.Dexterity + modifier)
    // ... etc for all stats
    }
}
```

Alternatively, clamp in the derived stat methods themselves (defensive in depth):
```go
func (a *Attributes) GetPhysicalDamage() int {
    dmg := (a.Strength / 2) + (a.Weapon * 2)
    return max(0, dmg)
}
```

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

### NEW-3. Spell Casting Has No Combat-Context Validation (MEDIUM)
**File:** `tactical/spells/system.go:17-63`

`ExecuteSpellCast()` validates mana, spellbook, and targets, but never checks whether the caster is actually in an active combat context. There is no guard like:
- Is there an active combat? (TurnState entity exists)
- Is it the caster's faction's turn?
- Has the caster already acted this turn?

```go
func ExecuteSpellCast(casterEntityID ecs.EntityID, spellID string,
    targetSquadIDs []ecs.EntityID, manager *common.EntityManager) *SpellCastResult {
    // Validates: spell exists, mana sufficient, spell in spellbook
    // Does NOT validate: combat active, faction turn, action state
}
```

**Impact:** The GUI prevents misuse (spell buttons only appear during combat on the player's turn), but the ECS-level function has no guards. Any code path that calls `ExecuteSpellCast` outside combat will succeed — deducting mana and applying damage/effects to targets even if combat isn't active.

**Severity justification:** Medium rather than High because the GUI layer currently prevents this, but it's a defense-in-depth gap. If AI code or future features call this function, there's no safety net.

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
