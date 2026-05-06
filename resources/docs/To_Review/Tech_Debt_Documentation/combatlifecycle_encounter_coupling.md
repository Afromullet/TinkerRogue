# Tech Debt: `mind/combatlifecycle` & `mind/encounter` Coupling

**Date:** 2026-04-30  
**Severity:** High (2 items), Medium (1 item), Low (1 item)  
**Estimated Fix Time:** 6–10 hours for high-priority items

---

## Diagnosis

The dependency direction is correct — `encounter` imports `combatlifecycle`, never the reverse. The coupling feeling comes from four specific structural problems, not circular imports.

```
combatlifecycle   ← defines contracts, types, entry points (platform-agnostic)
     ↑ implements
encounter         ← overworld + garrison concrete implementations
     ↑ implements
raid              ← raid concrete implementations (does NOT import encounter)
```

---

## Debt #1 — `ExitCombat` Is a Dispatch Monolith

**Priority: HIGH**

`encounter.EncounterService.ExitCombat` does six different things in one function:

1. Chooses which resolver to instantiate (type-switch on `CombatType`)
2. Calls `combatlifecycle.ExecuteResolution` with that resolver
3. Calls `teardown.TeardownCombat`
4. Calls `combatlifecycle.StripCombatComponents`
5. Restores garrison squads if garrison type
6. Records history, fires callbacks, resets state

The type-switch is the sharpest problem:

```go
// Current pattern in ExitCombat
if encounter.Type == combatlifecycle.CombatTypeGarrisonDefense {
    resolver = &GarrisonDefenseResolver{DefendedNodeID: ..., ...}
} else if reason == Flee {
    resolver = &FleeResolver{ThreatNodeID: ...}
} else {
    resolver = &OverworldCombatResolver{ThreatNodeID: ..., ...}
}
```

**Shotgun surgery:** Adding any new encounter-based combat type requires editing `combatlifecycle` (add const) + `encounter/ExitCombat` (add dispatch case) + a new resolver file. Three files across two packages for one concept.

### Fix: Store the Resolver Factory on `ActiveEncounter`

The starter already knows which resolver to create. That knowledge should live there, not in `ExitCombat`.

**Step 1 — Add a factory field to `ActiveEncounter`:**

```go
// encounter/types.go
type ActiveEncounter struct {
    // existing fields unchanged ...

    // Set by TransitionToCombat; called by ExitCombat to get the right resolver.
    BuildResolver func(playerVictory bool) combatlifecycle.CombatResolver
}
```

**Step 2 — Move resolver construction into each starter's `Prepare()`:**

```go
// encounter/starters.go — OverworldCombatStarter.Prepare()
setup.BuildResolverFn = func(playerVictory bool) combatlifecycle.CombatResolver {
    return &OverworldCombatResolver{
        ThreatNodeID:   threatNodeID,
        PlayerVictory:  playerVictory,
        PlayerEntityID: playerEntityID,
        PlayerSquadIDs: playerSquadIDs,
        EnemySquadIDs:  setup.EnemySquadIDs,
    }
}
```

**Step 3 — Simplify `ExitCombat`:**

```go
resolver := encounter.BuildResolver(result.IsPlayerVictory) // no type switch
resolutionResult := combatlifecycle.ExecuteResolution(es.manager, resolver)
// teardown, strip, history, callback — identical for all types
```

**Result:** Adding a new encounter type only requires a new starter. `ExitCombat` never changes.

---

## Debt #2 — `CombatSetup` and `ActiveEncounter` Are Data Mirrors

**Priority: HIGH**

After `Prepare()` returns a `CombatSetup`, encounter copies the relevant fields into an `ActiveEncounter`. These structs carry nearly identical data:

| Field | `CombatSetup` | `ActiveEncounter` |
|-------|:---:|:---:|
| EncounterID | ✓ | ✓ |
| ThreatID | ✓ | ✓ |
| ThreatName | ✓ | ✓ |
| EnemySquadIDs | ✓ | ✓ |
| RosterOwnerID | ✓ | ✓ |
| Type (CombatType) | ✓ | ✓ |
| DefendedNodeID | ✓ | ✓ |
| PlayerEntityID | — | ✓ |
| PlayerPosition | — | ✓ |
| StartTime | — | ✓ |

Adding any new context field (e.g., a boss flag, difficulty scalar) means updating both structs and the copy step — even if only `ActiveEncounter` ever reads it.

### Fix: Embed `CombatSetup` in `ActiveEncounter`

```go
// encounter/types.go
type ActiveEncounter struct {
    combatlifecycle.CombatSetup              // embed — fields promoted
    OriginalPlayerPosition *coords.LogicalPosition
    StartTime              time.Time
    PlayerEntityID         ecs.EntityID
    BuildResolver          func(bool) combatlifecycle.CombatResolver
}
```

`TransitionToCombat` stores `setup` directly instead of field-copying. All existing readers of `encounter.Type`, `encounter.ThreatID`, etc. continue to work via promoted fields. New fields added to `CombatSetup` are immediately available on `ActiveEncounter`.

**Minor trade-off:** `ActiveEncounter` publicly exposes setup-phase-only fields like `PostCombatReturnMode` and `SkipServiceResolution`. Document that these are set-once at combat start and not meaningful after `TransitionToCombat` returns.

---

## Debt #3 — `EncounterController` Bleeds `combatlifecycle` Types Into the GUI Boundary

**Priority: MEDIUM**

```go
// encounter package
type EncounterController interface {
    ExitCombat(
        reason   combatlifecycle.CombatExitReason,
        result   combatlifecycle.EncounterOutcome,
        teardown combatlifecycle.CombatTeardown,
    )
    GetRosterOwnerID() ecs.EntityID
    GetCurrentEncounterID() ecs.EntityID
}
```

GUI packages that only want to signal "combat ended, here's the result" are forced to import `combatlifecycle` just for the parameter types. The GUI has no business knowing about the combat lifecycle contract system.

### Fix: Define an `encounter`-Owned Exit Request Struct

```go
// encounter/types.go
type CombatExitRequest struct {
    Reason           string // "victory", "defeat", "flee"
    IsPlayerVictory  bool
    VictorFaction    string
    VictorName       string
    RoundsCompleted  int
    DefeatedFactions []string
}

type EncounterController interface {
    ExitCombat(req CombatExitRequest, teardown combatlifecycle.CombatTeardown)
    GetRosterOwnerID() ecs.EntityID
    GetCurrentEncounterID() ecs.EntityID
}
```

`CombatTeardown` stays typed because it is a real behavioral dependency, not just data. Internally `EncounterService.ExitCombat` converts `CombatExitRequest` to whatever `combatlifecycle` needs. GUI callers only import `encounter`.

---

## Debt #4 — Reward Calculation Is Split Across Package Boundaries

**Priority: LOW**

`encounter.CalculateIntensityReward` produces a `combatlifecycle.Reward`. `combatlifecycle.Grant` distributes it. Creation and distribution live in different packages with no natural home for reward policy (e.g., "double gold on hard mode"). Currently harmless, but the policy gap will surface if reward modifiers are ever added.

**Fix when touching both files for another reason:** Move `CalculateIntensityReward` to `combatlifecycle` as `RewardForIntensity`, making reward policy fully owned by the lifecycle package. Or keep it in `encounter` and accept that encounter owns reward creation while combatlifecycle owns distribution.

---

## What Not To Fix

The one-directional dependency (`encounter` → `combatlifecycle`, never the reverse) is correct. Do not try to merge the packages or invert the dependency. The contract/implementation split is intentional — it's what allows `raid` to bypass `encounter` entirely by implementing the same contracts independently.

---

## Prevention Rules

- **New encounter type:** instantiate its resolver in the starter's `Prepare()`. If `ExitCombat` needs a new branch, the factory pattern wasn't applied.
- **New setup context field:** add to `CombatSetup` only. After Debt #2 is fixed, `ActiveEncounter` gets it for free.
- **New GUI caller of encounter methods:** check whether the signature forces a `combatlifecycle` import. If it does, add an `encounter`-owned wrapper type.
