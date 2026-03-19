# Combat Pipeline Simplification Analysis

**Reviewer:** Tactical Simplifier (Go game architecture specialist)
**Date:** 2026-03-18
**Scope:** Combat start/end pipelines, resolvers, starters, encounter service, GUI integration

---

## Executive Assessment

The combat pipeline has a solid architectural foundation. The `CombatStarter` / `CombatResolver` interface pattern is clean, the shared `ExecuteCombatStart` funnel is well-designed, and the dependency boundaries (GUI never imports `mind/encounter`) are correctly maintained. The codebase is in good shape overall.

That said, there are concrete simplification opportunities. The issues are not structural failures but accumulated friction: duplicated positioning logic, a `CombatSetup` struct that uses boolean flags instead of polymorphism, an `ExitCombat` method with too many responsibilities, and scattered squad-to-faction assignment patterns that could be unified. None of these require architectural rewrites. They are medium-sized, incremental improvements.

---

## 1. Mental Complexity Analysis

### 1.1 What is hard to understand and why

**The ExitCombat method in EncounterService (encounter_service.go:220-269)**

This is the single highest cognitive load point in the pipeline. A developer reading this method must simultaneously track:

- The difference between raid and non-raid combat (line 237: `!es.activeEncounter.IsRaidCombat`)
- The three-way switch on `CombatExitReason` (Victory/Defeat vs Flee)
- That `EndEncounter` is called for non-raid victory/defeat, which internally dispatches to ANOTHER switch between garrison defense and overworld resolvers
- That `RecordEncounterCompletion` clears `activeEncounter`, so fields must be captured beforehand (lines 230-232)
- That garrison defense victory has special squad-return logic AFTER recording but BEFORE cleanup
- That `PostCombatCallback` fires last (for RaidRunner)

The method is only 45 lines, but it requires understanding 6 different state transitions in sequence, with ordering dependencies between them. The temporal coupling (must capture fields before `RecordEncounterCompletion` clears them) is a latent bug magnet.

**The EndEncounter method (encounter_service.go:138-194)**

This method has a second dispatch layer hidden inside it. `ExitCombat` dispatches by exit reason, then `EndEncounter` dispatches again by combat type (garrison vs overworld). This two-level dispatch is not immediately obvious from reading either method in isolation. A developer debugging a garrison defense resolution must trace through: `CombatMode.Exit` -> `ExitCombat` -> `EndEncounter` -> `GarrisonDefenseResolver.Resolve` -> `ExecuteResolution` -> `Grant`. That is 6 call levels before reaching the actual game logic.

**CombatSetup as a bag of flags (combat_contracts.go:20-35)**

The flags `IsGarrisonDefense`, `IsRaidCombat`, and `PostCombatReturnMode` are consumed in different places:
- `IsRaidCombat` is checked in `ExitCombat` to skip overworld resolution
- `IsGarrisonDefense` is checked in `ExitCombat` for garrison squad return
- `PostCombatReturnMode` is checked in `TransitionToCombat` and again in `CombatTurnFlow.getPostCombatReturnMode`

Every consumer of `CombatSetup` must know which flags are relevant to their context. Adding a new combat type means adding a new flag, then hunting through every consumer to add handling.

### 1.2 What is easy to understand

- `ExecuteCombatStart` in `starter.go` -- 20 lines, crystal clear. Prepare, transition, rollback on failure.
- `ExecuteResolution` in `pipeline.go` -- 15 lines, equally clear. Resolve, grant rewards if any.
- The `Reward` / `GrantTarget` / `Grant` system in `reward.go` -- clean value types, obvious flow.
- Individual resolver structs (`RaidRoomResolver`, `FleeResolver`, etc.) -- small, focused, easy to test.
- The `CombatModeDeps` pattern in `combatdeps.go` -- good consolidation of dependencies.

---

## 2. Separation of Concerns

### 2.1 Where boundaries are blurred

**EncounterService does too many things in ExitCombat**

`ExitCombat` currently handles:
1. Resolver dispatch (game logic)
2. History recording (analytics)
3. Player position restoration (spatial state)
4. Garrison squad return (combat type-specific cleanup)
5. Entity cleanup via CombatCleaner (ECS lifecycle)
6. PostCombatCallback notification (event system)

This violates single responsibility. The method is an orchestrator, but it also contains conditional game logic (the garrison defense special case on line 259). The ordering is fragile: if someone moves the `RecordEncounterCompletion` call before the garrison return, squads get stripped incorrectly.

**EndEncounter mixes resolution dispatch with encounter entity mutation**

`EndEncounter` (lines 138-194) does three unrelated things:
1. Dispatches to the correct resolver (game logic)
2. Marks `encounterData.IsDefeated = true` (ECS state mutation)
3. Hides the encounter sprite (rendering concern)

The sprite hiding is already done by `OverworldCombatStarter.Prepare()` during combat start. The version in `EndEncounter` is for permanent hiding after victory. This dual-location sprite management is confusing -- `Prepare()` hides temporarily, `EndEncounter` hides permanently, and `RestoreEncounterSprite` restores on flee. The sprite lifecycle is spread across three methods in two different files.

**CombatMode.Exit knows about battle log export**

Lines 477-489 of `combatmode.go` handle battle log export. This is a debug/analytics concern mixed into the mode transition. It works fine but adds cognitive load to the Exit method.

### 2.2 Where boundaries are clean

- `CombatStarter` / `CombatResolver` interfaces cleanly separate type-specific logic from shared infrastructure.
- `CombatTransitioner` / `CombatCleaner` / `EncounterCallbacks` interfaces correctly break the import cycle between GUI and encounter packages.
- `combatlifecycle` package is a clean shared layer with no dependencies on specific combat types.
- `CombatService` is properly focused on combat mechanics (turn management, faction queries, victory conditions).

---

## 3. Duplication Between Combat Types

### 3.1 Squad positioning and faction assignment

This is the most significant duplication. Every starter performs the same sequence:

1. Create `CombatQueryCache` and `CombatFactionManager`
2. Call `CreateFactionWithPlayer` twice (player and enemy)
3. Loop over squads, calling `AddSquadToFaction` + `EnsureUnitPositions` + `CreateActionStateForSquad`
4. Optionally mark `IsDeployed = true`

This pattern appears in:
- `encounter_setup.go:SpawnCombatEntities` (lines 52-76)
- `encounter_setup.go:spawnGarrisonEncounter` (lines 86-116)
- `encounter/starters.go:GarrisonDefenseStarter.Prepare` (lines 135-155)
- `raid/raidencounter.go:SetupRaidFactions` (lines 48-95)

Four places with the same 3-step squad assignment loop. The differences are:
- Position calculation strategy (arc-based vs offset-based)
- Whether `IsDeployed` is set
- Whether squads come from a roster, a garrison, or are pre-generated

**Quantified impact:** ~120 lines of duplicated faction/squad/position setup across 4 locations. Could be reduced to ~40 lines with a shared helper, a 67% reduction in this specific pattern.

### 3.2 Power calculation and difficulty modifiers

Both `GarrisonDefenseStarter.Prepare` and `GenerateEncounterSpec` perform the same power calculation:

```
powerConfig := evaluation.GetPowerConfigByProfile(DefaultPowerProfile)
totalPower := sum of CalculateSquadPower for each squad
avgPower := totalPower / count
difficultyMod := getDifficultyModifier(level)
targetEnemyPower := avgPower * difficultyMod.PowerMultiplier
// clamp to min/max
```

This appears in `encounter/starters.go:158-169` and `encounter_generator.go:48-70`. The garrison version uses garrison squads as the reference; the overworld version uses player squads. The actual calculation is identical.

**Quantified impact:** ~25 lines duplicated. Could be extracted to a single `CalculateTargetPower(squadIDs, level, manager) float64` function.

### 3.3 Position generation approaches

Three different positioning strategies exist:
- `generatePositionsAroundPoint` in `encounter_setup.go` -- arc-based, used by overworld and garrison defense
- `generatePlayerSquadPositions` in `encounter_setup.go` -- wrapper around the above
- Inline offset calculation in `raid/raidencounter.go:58-89` -- simple X/Y offsets

The raid positioning is intentionally simpler (rooms are small, positions are deterministic), so this is not pure duplication. However, all three share the concept of "place N squads relative to a center point" and could share a common positioning abstraction.

---

## 4. CombatSetup: Bag of Flags vs Polymorphism

### Current state

```go
type CombatSetup struct {
    // Core fields (used by all types)
    PlayerFactionID ecs.EntityID
    EnemyFactionID  ecs.EntityID
    EnemySquadIDs   []ecs.EntityID
    CombatPosition  coords.LogicalPosition
    EncounterID     ecs.EntityID
    ThreatName      string
    RosterOwnerID   ecs.EntityID

    // Type-discriminator fields (only relevant to specific types)
    ThreatID             ecs.EntityID      // overworld only
    IsGarrisonDefense    bool              // garrison only
    DefendedNodeID       ecs.EntityID      // garrison only
    IsRaidCombat         bool              // raid only
    PostCombatReturnMode string            // raid only
}
```

### Problem

The boolean flags create implicit type discrimination. Every consumer must check the right combination of flags. Adding "arena combat" or "scripted boss encounter" means adding more flags, and every `if/switch` that checks `IsGarrisonDefense` or `IsRaidCombat` must be updated.

### Assessment

This is a moderate concern, not critical. There are currently only 3 combat types and the flags are checked in exactly 2 places (`ExitCombat` and `TransitionToCombat`). The flag pattern works at this scale. It becomes problematic at 5+ types.

### Recommendation: CombatType enum (low-risk, moderate gain)

Replace the boolean flags with a typed enum:

```go
type CombatType int

const (
    CombatTypeOverworld CombatType = iota
    CombatTypeGarrisonDefense
    CombatTypeRaid
)

type CombatSetup struct {
    // Core fields (unchanged)
    PlayerFactionID ecs.EntityID
    EnemyFactionID  ecs.EntityID
    EnemySquadIDs   []ecs.EntityID
    CombatPosition  coords.LogicalPosition
    EncounterID     ecs.EntityID
    ThreatName      string
    RosterOwnerID   ecs.EntityID

    // Type identification (replaces IsGarrisonDefense + IsRaidCombat)
    CombatType CombatType

    // Type-specific context (only populated for relevant types)
    ThreatID             ecs.EntityID       // overworld
    DefendedNodeID       ecs.EntityID       // garrison defense
    PostCombatReturnMode string             // raid
}
```

Benefits:
- Adding a new type is one line in the enum, not a new boolean + hunting for if-checks
- `switch setup.CombatType` is exhaustive and the compiler can warn on missing cases (with linters)
- Self-documenting: `CombatTypeGarrisonDefense` is clearer than `IsGarrisonDefense: true`
- No structural change to consumers, just replace `if setup.IsGarrisonDefense` with `if setup.CombatType == CombatTypeGarrisonDefense`

Risk: Very low. Mechanical refactor, no behavior change.

---

## 5. The ExitCombat Switch/If-Chain

### Current code (encounter_service.go:220-269)

```go
func (es *EncounterService) ExitCombat(...) {
    // Capture before RecordEncounterCompletion clears activeEncounter
    enemySquadIDs := es.activeEncounter.EnemySquadIDs
    isGarrisonDefense := es.activeEncounter.IsGarrisonDefense
    defendedNodeID := es.activeEncounter.DefendedNodeID

    // Step 1: Resolve
    switch reason {
    case combat.ExitVictory, combat.ExitDefeat:
        if !es.activeEncounter.IsRaidCombat {
            es.EndEncounter(...)
        }
    case combat.ExitFlee:
        es.RestoreEncounterSprite()
        // Flee resolver inline
    }

    // Step 2: Record history
    es.RecordEncounterCompletion(...)

    // Step 3: Cleanup (with garrison special case)
    if combatCleaner != nil {
        if isGarrisonDefense && result.IsPlayerVictory {
            es.returnGarrisonSquadsToNode(defendedNodeID)
        }
        combatCleaner.CleanupCombat(enemySquadIDs)
    }

    // Step 4: Notify listeners
    if es.PostCombatCallback != nil {
        es.PostCombatCallback(reason, result)
    }
}
```

### Problems

1. **Temporal coupling**: Fields must be captured before `RecordEncounterCompletion` clears `activeEncounter`. This is a comment-documented invariant, not a compiler-enforced one. A future refactor could break it silently.

2. **Inline flee resolution**: The flee case creates a resolver inline (lines 244-248), while victory/defeat delegate to `EndEncounter`. Inconsistent dispatch.

3. **Garrison special case in cleanup**: The `returnGarrisonSquadsToNode` call on line 260 is a combat-type-specific concern embedded in the shared exit path. This is exactly the kind of logic that should be in a resolver.

4. **EndEncounter has a second dispatch**: Victory/defeat go through `EndEncounter`, which then dispatches again based on `IsGarrisonDefense` vs `ThreatNodeID != 0`. Two-level dispatch for what should be a single polymorphic call.

### Recommendation: Flatten to single resolver dispatch

The exit path should have ONE dispatch point, not two. Currently it is:

```
ExitCombat -> switch(reason) -> EndEncounter -> if(garrison) / else if(overworld)
```

It should be:

```
ExitCombat -> resolve(resolver) -> record -> cleanup -> notify
```

Where the resolver is chosen at combat START time and stored on `ActiveEncounter`, or determined at exit time from the `CombatType` enum. The resolver handles ALL type-specific logic including sprite management, garrison squad return, and reward calculation.

Concrete change:

```go
func (es *EncounterService) ExitCombat(
    reason combat.CombatExitReason,
    result *combat.EncounterOutcome,
    combatCleaner combat.CombatCleaner,
) {
    if es.activeEncounter == nil {
        return
    }

    // Snapshot mutable state before any mutations
    snapshot := es.snapshotActiveEncounter()

    // Step 1: Resolve (single dispatch point)
    resolver := es.buildResolver(reason, result, snapshot)
    if resolver != nil {
        combatlifecycle.ExecuteResolution(es.manager, resolver)
    }

    // Step 2: Handle encounter entity state (defeated, sprite visibility)
    es.finalizeEncounterEntity(reason, result.IsPlayerVictory, snapshot)

    // Step 3: Record history + restore player position
    es.RecordEncounterCompletion(reason, result.VictorFaction,
        result.VictorName, result.RoundsCompleted)

    // Step 4: Type-specific pre-cleanup (garrison return, etc.)
    es.preCleanup(snapshot, result)

    // Step 5: Entity cleanup
    if combatCleaner != nil {
        combatCleaner.CleanupCombat(snapshot.EnemySquadIDs)
    }

    // Step 6: Notify listeners
    if es.PostCombatCallback != nil {
        es.PostCombatCallback(reason, result)
    }
}
```

Where `buildResolver` is:

```go
func (es *EncounterService) buildResolver(
    reason combat.CombatExitReason,
    result *combat.EncounterOutcome,
    snapshot encounterSnapshot,
) combatlifecycle.CombatResolver {
    if snapshot.IsRaidCombat {
        return nil // Raid handles resolution via PostCombatCallback
    }

    switch reason {
    case combat.ExitFlee:
        if snapshot.ThreatNodeID != 0 {
            return &FleeResolver{ThreatNodeID: snapshot.ThreatNodeID}
        }
        return nil

    case combat.ExitVictory, combat.ExitDefeat:
        if snapshot.IsGarrisonDefense {
            return &GarrisonDefenseResolver{
                PlayerVictory:        result.IsPlayerVictory,
                DefendedNodeID:       snapshot.DefendedNodeID,
                AttackingFactionType: snapshot.AttackingFactionType,
            }
        }
        if snapshot.ThreatNodeID != 0 {
            return &OverworldCombatResolver{
                ThreatNodeID:   snapshot.ThreatNodeID,
                PlayerVictory:  result.IsPlayerVictory,
                PlayerEntityID: snapshot.PlayerEntityID,
                PlayerSquadIDs: es.getAllPlayerSquadIDs(),
                EnemySquadIDs:  snapshot.EnemySquadIDs,
            }
        }
        return nil
    }
    return nil
}
```

Benefits:
- Single dispatch point instead of two
- `snapshotActiveEncounter` eliminates temporal coupling -- snapshot is immutable
- Each step is a named method with clear responsibility
- Adding a new combat type = adding a case to `buildResolver`, nothing else

Risk: Medium. Requires touching `ExitCombat`, `EndEncounter`, and moving sprite/defeat logic into `finalizeEncounterEntity`. Must be done carefully with tests verifying all 3 combat types x 3 exit reasons = 9 scenarios.

---

## 6. Resolver Pattern Assessment

### Current state: Clean, with one wrinkle

The resolver pattern itself is well-designed:
- Small interface: `Resolve(manager) *ResolutionPlan`
- Stateless structs with all context in fields
- `ExecuteResolution` is the single entry point
- `ResolutionPlan` cleanly separates "what to grant" from "how to grant it"

### The wrinkle: OverworldCombatResolver is too large

`OverworldCombatResolver.Resolve` in `encounter/resolvers.go` is 65 lines with three branches (destroy threat, weaken threat, player defeat). Each branch has event logging, reward calculation, and node mutation. This is the only resolver that feels heavy.

### Recommendation

Split into private helper methods on the resolver:

```go
func (r *OverworldCombatResolver) Resolve(manager *common.EntityManager) *combatlifecycle.ResolutionPlan {
    threatEntity := manager.FindEntityByID(r.ThreatNodeID)
    if threatEntity == nil {
        return nil
    }
    nodeData := common.GetComponentType[*core.OverworldNodeData](threatEntity, core.OverworldNodeComponent)
    if nodeData == nil {
        return nil
    }

    if r.PlayerVictory {
        return r.resolveVictory(manager, threatEntity, nodeData)
    }
    return r.resolveDefeat(manager, threatEntity, nodeData)
}
```

Where `resolveVictory` further splits into `destroyThreat` and `weakenThreat` based on whether intensity hits zero. This is a straightforward extract-method refactor with no risk.

---

## 7. Squad Positioning Logic Duplication

### The pattern that repeats

Every starter must:
1. Generate positions for N squads around a center point
2. For each squad: `AddSquadToFaction` + `EnsureUnitPositions` + `CreateActionStateForSquad`
3. Optionally set `IsDeployed = true`

### Where it appears

| File | Function | Lines |
|------|----------|-------|
| `encounter/encounter_setup.go` | `SpawnCombatEntities` | 63-70 |
| `encounter/encounter_setup.go` | `spawnGarrisonEncounter` | 98-110 |
| `encounter/encounter_setup.go` | `assignPlayerSquadsToFaction` | 166-179 |
| `encounter/starters.go` | `GarrisonDefenseStarter.Prepare` | 142-155 |
| `raid/raidencounter.go` | `SetupRaidFactions` | 58-90 |

### Recommendation: Extract `AssignSquadsToFaction` helper

```go
// In combatlifecycle or encounter package

type SquadPlacement struct {
    SquadID    ecs.EntityID
    Position   coords.LogicalPosition
    MarkDeploy bool // Set IsDeployed = true
}

// AssignSquadsToFaction adds squads to a faction with positions and action states.
// This is the single source of truth for the squad-to-faction assignment sequence.
func AssignSquadsToFaction(
    fm *combat.CombatFactionManager,
    manager *common.EntityManager,
    factionID ecs.EntityID,
    placements []SquadPlacement,
) error {
    for _, p := range placements {
        if err := fm.AddSquadToFaction(factionID, p.SquadID, p.Position); err != nil {
            return fmt.Errorf("failed to add squad %d: %w", p.SquadID, err)
        }
        EnsureUnitPositions(manager, p.SquadID, p.Position)
        combat.CreateActionStateForSquad(manager, p.SquadID)

        if p.MarkDeploy {
            squadData := common.GetComponentTypeByID[*squads.SquadData](manager, p.SquadID, squads.SquadComponent)
            if squadData != nil {
                squadData.IsDeployed = true
            }
        }
    }
    return nil
}
```

Each starter builds a `[]SquadPlacement` using its own positioning strategy, then calls this single function. The positioning logic remains type-specific (it should be -- arc-based and offset-based serve different purposes), but the assignment loop is unified.

**Impact:** Eliminates ~80 lines of duplicated loop code across 5 call sites. Each call site becomes ~5 lines (build placements, call helper) instead of ~15 lines (manual loop with error handling).

Risk: Low. Pure extraction, no behavior change.

---

## 8. Specific Simplification Proposals

### Proposal 1: CombatType enum (Priority: Low, Effort: 30 min)

**Files:** `tactical/combat/combat_contracts.go`, `encounter/starters.go`, `raid/starters.go`, `encounter/encounter_service.go`

Replace `IsGarrisonDefense bool` + `IsRaidCombat bool` with `CombatType CombatType`. Mechanical find-and-replace. No behavior change.

**Lines affected:** ~15 lines changed across 4 files.

### Proposal 2: AssignSquadsToFaction helper (Priority: High, Effort: 1 hour)

**Files:** New function in `encounter/encounter_setup.go` (or `combatlifecycle/`). Refactor callers in `encounter/encounter_setup.go`, `encounter/starters.go`, `raid/raidencounter.go`.

Extract the repeated squad assignment loop into a single helper.

**Lines reduced:** ~80 lines of duplication eliminated.

### Proposal 3: CalculateTargetPower helper (Priority: Medium, Effort: 30 min)

**Files:** `encounter/encounter_generator.go`, `encounter/starters.go`

Extract power calculation into:

```go
func CalculateTargetPower(
    manager *common.EntityManager,
    referenceSquadIDs []ecs.EntityID,
    level int,
) (targetPower float64, difficultyMod templates.JSONEncounterDifficulty)
```

**Lines reduced:** ~25 lines of duplication.

### Proposal 4: Flatten ExitCombat dispatch (Priority: High, Effort: 2-3 hours)

**Files:** `encounter/encounter_service.go` (primary), `encounter/resolvers.go` (minor)

Restructure `ExitCombat` to use snapshot + single resolver dispatch as described in Section 5. Remove `EndEncounter` as a separate method -- inline its non-resolver responsibilities into named helper methods.

**Lines reduced:** Net reduction ~20 lines, but the real gain is cognitive -- single dispatch path instead of two-level dispatch.

**Risk mitigation:** Write a test table covering all 9 scenarios (3 combat types x 3 exit reasons) before refactoring. Verify each scenario produces the same resolver type and same side effects.

### Proposal 5: Split OverworldCombatResolver.Resolve (Priority: Low, Effort: 20 min)

**File:** `encounter/resolvers.go`

Extract `resolveVictory` and `resolveDefeat` private methods. Pure readability improvement.

**Lines reduced:** 0 (same total, better organized).

### Proposal 6: Move battle log export out of CombatMode.Exit (Priority: Low, Effort: 30 min)

**File:** `gui/guicombat/combatmode.go`

Register a callback on `PostCombatCallback` or use the existing `RegisterOnTurnEnd` pattern to handle battle log export outside the mode transition. Alternatively, extract to a private method on `CombatMode` to at least name the concern.

**Lines reduced:** 0 (organizational only).

---

## 9. Recommended Implementation Order

1. **Proposal 2: AssignSquadsToFaction** -- Highest impact, lowest risk. Reduces the most duplication with a pure extraction. Do this first to clean up the starters before touching ExitCombat.

2. **Proposal 3: CalculateTargetPower** -- Quick win that removes duplication between garrison defense and overworld encounter generation.

3. **Proposal 1: CombatType enum** -- Quick mechanical change that makes Proposal 4 cleaner (switch on enum instead of boolean combinations).

4. **Proposal 4: Flatten ExitCombat** -- Largest change, but builds on the cleaner code from Proposals 1-3. Write tests first.

5. **Proposal 5: Split OverworldCombatResolver** -- Minor cleanup, do whenever convenient.

6. **Proposal 6: Battle log export** -- Low priority, cosmetic.

---

## 10. What NOT to Change

### Things that are working well

- **The CombatStarter / CombatResolver interface pair.** Clean, extensible, testable. Do not over-abstract these.

- **ExecuteCombatStart and ExecuteResolution.** These are the right level of abstraction. They are small, focused, and do exactly one thing each.

- **The dependency boundary via EncounterCallbacks interface.** GUI correctly never imports `mind/encounter`. Do not collapse this boundary.

- **CombatModeDeps pattern.** Good consolidation of constructor parameters. No change needed.

- **The Reward/GrantTarget/Grant system.** Clean value types with clear flow. The `Scale` method is a nice touch.

- **PostCombatCallback for RaidRunner.** Simple, effective event notification. Do not replace with a complex event bus.

- **CombatService.CleanupCombat.** Well-structured cleanup sequence with clear phases.

### Temptations to avoid

- **Do not create a CombatType interface hierarchy.** Three combat types with boolean flags is a code smell, but an interface hierarchy with `OverworldCombat`, `GarrisonCombat`, `RaidCombat` types would be over-engineering at this scale. The enum is the right level.

- **Do not merge encounter and raid packages.** They serve different domains (overworld encounters vs dungeon raids). The current package split is correct.

- **Do not add a generic event system.** The `PostCombatCallback` is simple and works. A publish-subscribe system would add complexity for no gain with only one listener.

---

## Key File Reference

| File | Role | Simplification Target |
|------|------|----------------------|
| `tactical/combat/combat_contracts.go` | Interface definitions + CombatSetup | Proposal 1 (CombatType enum) |
| `mind/combatlifecycle/starter.go` | Shared start pipeline | No changes needed |
| `mind/combatlifecycle/pipeline.go` | Shared resolution pipeline | No changes needed |
| `mind/combatlifecycle/cleanup.go` | StripCombatComponents | No changes needed |
| `mind/combatlifecycle/reward.go` | Reward granting | No changes needed |
| `mind/encounter/encounter_service.go` | ExitCombat orchestration | Proposal 4 (flatten dispatch) |
| `mind/encounter/starters.go` | Overworld + Garrison starters | Proposals 2, 3 (extract helpers) |
| `mind/encounter/encounter_setup.go` | Squad setup helpers | Proposal 2 (AssignSquadsToFaction) |
| `mind/encounter/encounter_generator.go` | Power-based generation | Proposal 3 (CalculateTargetPower) |
| `mind/encounter/resolvers.go` | All encounter resolvers | Proposal 5 (split large resolver) |
| `mind/raid/raidencounter.go` | Raid faction setup | Proposal 2 (use shared helper) |
| `mind/raid/starters.go` | Raid combat starter | Proposal 1 (CombatType enum) |
| `gui/guicombat/combatmode.go` | Combat GUI mode | Proposal 6 (battle log extraction) |
| `gui/guicombat/combat_turn_flow.go` | Turn lifecycle | No changes needed |
| `gui/guicombat/combatdeps.go` | Dependency container | No changes needed |
