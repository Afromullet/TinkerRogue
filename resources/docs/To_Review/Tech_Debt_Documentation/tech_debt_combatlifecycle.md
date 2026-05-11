# Technical Debt Analysis: `mind/combatlifecycle`

**Date:** 2026-05-10
**Scope:** `mind/combatlifecycle/` (7 source files, 668 LOC) and its callers

---

## Package Profile

| File | LOC | Role |
|---|---|---|
| `contracts.go` | 160 | Interfaces, enums, `CombatSetup`, `EncounterOutcome` |
| `reward.go` | 165 | `Reward`, `Grant`, per-currency grant helpers |
| `enrollment.go` | 98 | `CreateFactionPair`, `EnrollSquadInFaction`, `EnrollSquadsAtPositions`, `EnsureUnitPositions` |
| `cleanup.go` | 69 | `ApplyHPRecovery`, `StripCombatComponents` |
| `pipeline.go` | 46 | `ExecuteResolution`, `ResolutionPlan`, `ResolutionResult` |
| `casualties.go` | 40 | `GetLivingUnitIDs`, `CountDeadUnits` |
| `starter.go` | 25 | `ExecuteCombatStart` |
| `resolution_test.go` | 65 | Tests for `Reward.Scale` only |
| **Total** | **668** | |

Dependents: 16 files across `mind/encounter`, `campaign/raid`, `tactical/combat/combatservices`, `gui/guicombat`, `gui/guioverworld`, `setup/gamesetup`. The package is **on the critical path for every combat encounter**.

---

## 1. Debt Inventory

### A. Code Debt

#### A1. Fragmented unit-iteration helpers (Medium)
The "iterate squads → iterate units → filter by `Attributes.CurrentHealth`" pattern is implemented at least **four times** across two packages with subtly different shapes:

| Function | Location | Returns |
|---|---|---|
| `GetLivingUnitIDs` | `casualties.go:11` | `[]ecs.EntityID` |
| `CountDeadUnits` | `casualties.go:25` | `int` |
| `squadcore.CountLivingUnitsInSquad` | `squadqueries.go:14` | `int` |
| `squadcore.IsSquadDestroyed` | `squadqueries.go:117` | `bool` |
| `squadcore.GetSquadHealthPercent` | `squadqueries.go:461` | `float64` |

Even within `casualties.go` the two functions are inconsistent: `GetLivingUnitIDs` uses `GetComponentTypeByID`, while `CountDeadUnits` uses `FindEntityByID`+`GetComponentType`. Same data, two access patterns.

**Cost:** Each new aggregation copies ~5 lines and risks divergence (e.g., what does "alive" mean if `Attributes` is missing?).

#### A2. `fmt.Printf` as logging mechanism (Low)
5 `fmt.Printf` calls in `cleanup.go:67`, `reward.go:99,121,144,162` emit unconditional trace output. No `DEBUG_MODE` gate, no logger abstraction. Cannot be silenced in tests, pollutes stderr in saves/load cycles.

#### A3. Per-call RNG seeding (Low–Medium)
`reward.go:116`: `rng := rand.New(rand.NewSource(time.Now().UnixNano()))` allocates a fresh RNG **inside `grantExperience`**. Downstream `unitprogression.AwardExperience` already accepts `*rand.Rand`, so the dependency exists — the seam stops one layer too high. Tests cannot inject deterministic XP rolls.

#### A4. Dead exported API (Low)
`EnrollSquadInFaction` (enrollment.go:22) is exported and documented but the only caller is its sibling `EnrollSquadsAtPositions` in the same file. No external use. `COMBAT_PIPELINES.md:266` still documents it as public API — doc drift.

#### A5. Boolean parameter smell (Low)
`EnrollSquadInFaction(..., markDeployed bool)` and `EnrollSquadsAtPositions(..., markDeployed bool)` propagate a flag that callers always pass as a constant (true for garrison defenders, false for overworld attackers). The flag is policy that belongs to the caller, not infrastructure.

### B. Architecture Debt

#### B1. Two parallel post-combat dispatch paths (High)
- **Path 1:** `CombatSetup.BuildResolver` closure → `EncounterService.ExitCombat` → `ExecuteResolution` (overworld, garrison)
- **Path 2:** `EncounterService.SetPostCombatCallback` → `RaidRunner.ResolveEncounter` → `ExecuteResolution` (raid)

Both terminate at the same function. The split is mediated by `CombatSetup.SkipServiceResolution bool` (contracts.go:76). Every new combat type forces a decision about which mechanism to use, and the rules ("if your starter owns its own state, use the callback") are encoded only in code comments.

#### B2. `CombatSetup` is a discriminated union flattened into 13 fields (Medium)
Several fields are conditionally meaningful by `Type`:

| Field | Used by |
|---|---|
| `RosterOwnerID` | overworld, raid (garrison sets 0) |
| `DefendedNodeID` | garrison only |
| `PostCombatReturnMode` | raid only |
| `SkipServiceResolution` | raid only |
| `BuildResolver` | overworld + garrison; raid leaves nil |

Constructing an invalid `CombatSetup` (e.g., `Type=Raid` with a `BuildResolver`) is a compile-time success, runtime ambiguity. The `CombatType` enum exists (good, replaced two bools per its own comment) but the struct doesn't enforce per-type field validity.

#### B3. `BuildResolver` closure mixes preparation and factory responsibilities (Medium)
Starters return a closure invoked at exit time:
```go
BuildResolver: func(playerVictory bool, _ ecs.EntityID, _ []ecs.EntityID) combatlifecycle.CombatResolver {
    return &GarrisonDefenseResolver{...}
}
```
The garrison closure (`starters.go:145`) discards both `playerEntityID` and `playerSquadIDs` — the signature is wider than necessary because overworld needs them. This is signature-driven coupling: every resolver now nominally needs the player context regardless of use.

#### B4. `StripCombatComponents` is a shotgun-surgery hot spot (Medium)
`cleanup.go:31` knows about:
- `combatstate.FactionMembershipComponent`
- `common.PositionComponent` (via `UnregisterEntityPosition`)
- `perks.PerkRoundStateComponent`
- `squadcore.SquadData.IsDeployed`

Every new combat-only component (e.g., a future `MoraleComponent`, `ComboMeterComponent`) requires editing this function. There is no registration pattern ("register a cleanup hook on combat exit"). A new component added without touching this function silently leaks combat state into out-of-combat squads.

#### B5. `CombatTeardown.TeardownCombat` returns `[]ecs.EntityID` to dodge an import cycle (Low)
Contract documented at `contracts.go:151–160` and `COMBAT_PIPELINES.md:637`. The return value exists only because `tactical/combat/combatservices` cannot import `mind/combatlifecycle` to call `StripCombatComponents` directly. Now permanently encoded into the public contract.

### C. Testing Debt (Highest impact)

| Function | Tested? |
|---|---|
| `Reward.Scale` | yes |
| `ExecuteCombatStart` | no |
| `ExecuteResolution` | no |
| `Grant` / `grantGold` / `grantExperience` / `grantManaToSquads` / `grantProgressionPoints` | no |
| `EnrollSquadInFaction` / `EnrollSquadsAtPositions` / `EnsureUnitPositions` / `CreateFactionPair` | no |
| `ApplyHPRecovery` | no |
| `StripCombatComponents` | no |
| `GetLivingUnitIDs` / `CountDeadUnits` | no |
| `DetermineExitReason` | no |
| `CombatType.String` / `CombatExitReason.String` | no |

**1 of 14 public functions has tests (~7%).** This is the package's largest single risk: every reward path and every cleanup path is unverified, and the fixtures already exist (`testing/` package, see CLAUDE.md memory note about `DEBUG_MODE` gating).

### D. Documentation Debt (Low)

- All exported functions have at least one godoc line — good baseline.
- `COMBAT_PIPELINES.md` is comprehensive but documents `EnrollSquadInFaction` as a public helper that has no external callers.
- No diagram showing the two-track resolver/callback dispatch — it's only described in prose.

---

## 2. Impact Assessment

| Debt item | Risk | Why it matters |
|---|---|---|
| **C. Untested reward / cleanup paths** | **High** | Silent reward duplication or loss is unobservable until a player notices. `StripCombatComponents` skipping a new component leaks combat state into save files. |
| **B1. Two dispatch paths** | High | Adding a 4th combat type means picking between resolver-closure and callback. Easy to wire half a path. |
| **B4. Shotgun-surgery cleanup** | Medium | Each combat-state component added elsewhere requires editing this file. Non-local change. |
| **A1. Iteration helper sprawl** | Medium | New aggregations keep being added (`unitsLost` counted manually in raidrunner.go:222 instead of via a helper). |
| **B2. Bloated `CombatSetup`** | Medium | Refactor cost grows with each field. Already 13 fields with type-specific subsets. |
| **B3. Resolver closure** | Low–Medium | Mostly cosmetic; works correctly today. |
| **A2. `fmt.Printf` logging** | Low | Cosmetic + test noise. |
| **A3. Per-call RNG** | Low | Test determinism; not a player-visible bug. |
| **A4. Dead `EnrollSquadInFaction`** | Low | Just unexport it. |
| **A5. `markDeployed bool`** | Low | Fix when refactoring B2. |
| **D. Doc drift** | Low | Update during the same PR as A4. |

---

## 3. Prioritized Roadmap

### Quick Wins — this week

| # | Action | Effort | Files | Value |
|---|---|---|---|---|
| 1 | **Unexport `EnrollSquadInFaction` → `enrollSquadInFaction`**; update `COMBAT_PIPELINES.md` | 15 min | enrollment.go, COMBAT_PIPELINES.md | Removes dead surface |
| 2 | **Inject RNG into `grantExperience`** via `Grant` parameter (or package-level seeded `*rand.Rand`) | 30 min | reward.go | Test determinism |
| 3 | **Add `_test.go` for `DetermineExitReason` and `CombatType.String`** (pure functions, no fixtures) | 30 min | new test file | Trivial coverage win |
| 4 | **Replace `fmt.Printf` with `config.DEBUG_MODE`-gated trace** following the project's existing pattern | 30 min | reward.go, cleanup.go | Silences test noise |

Total: ~2 hours.

### Sprint 1 (1–2 weeks) — Cover the critical paths

| # | Action | Effort |
|---|---|---|
| 5 | **Write tests for `Grant`** with a fake `EntityManager` (use `testing/` fixtures): zero rewards, partial rewards, missing commander, missing stockpile | 4 h |
| 6 | **Write tests for `StripCombatComponents`**: leaves non-combat components, removes faction/position/perk-state, resets `IsDeployed`, handles missing entities | 3 h |
| 7 | **Write tests for `ExecuteResolution`**: nil plan, zero-reward plan, all-currency plan; verify `RewardText` shape | 2 h |
| 8 | **Write tests for `EnrollSquadsAtPositions`**: mismatched lengths, partial failure error wrapping, position propagation to units | 3 h |
| 9 | **Extract a `ForEachAliveUnit` iterator** in `casualties.go`; rewrite `GetLivingUnitIDs`, `CountDeadUnits`, `squadcore.CountLivingUnitsInSquad`, `squadcore.IsSquadDestroyed` on top of it | 2 h |

Total: ~14 hours. Outcome: package goes from ~7% to ~70% test coverage on logic paths, and the iteration pattern is consolidated.

### Sprint 2–3 (3–6 weeks) — Architecture cleanup

| # | Action | Effort |
|---|---|---|
| 10 | **Replace `CombatSetup.SkipServiceResolution` + `BuildResolver` + callback split with a single `CombatResolver` field on the setup**, owned by the starter. Raid's starter would attach a resolver that does what `RaidRunner.ResolveEncounter` does today; the post-combat callback survives as a notification hook only (not a dispatch path). | 8 h |
| 11 | **Replace `markDeployed bool`** with two thin wrappers: `EnrollDefenders` (markDeployed=true) and `EnrollAttackers` (markDeployed=false). Or push the `IsDeployed=true` write to the caller of `EnrollSquadsAtPositions`. | 1 h |
| 12 | **Per-type setup constructors:** replace direct `&CombatSetup{...}` literals with `NewOverworldSetup(...)`, `NewGarrisonSetup(...)`, `NewRaidSetup(...)` so unused fields can't be set for the wrong type. | 3 h |
| 13 | **Cleanup-hook registry:** in `cleanup.go`, expose `RegisterCombatComponentCleanup(func(*ecs.Entity))`. Faction/position/perk packages register their own removers in `init()`. `StripCombatComponents` becomes a loop over registrations. | 4 h |

Total: ~16 hours.

### Long-term (Q3+) — Optional polish

- **Reward pipeline as data:** make `Reward` extensible (map of currency → amount) so adding a new currency doesn't touch `Grant`.
- **Property tests for resolution dispatch:** generate `(CombatType, ExitReason, Victory)` triples and assert which side effects fire.

---

## 4. Implementation Strategy

### Cleanup-hook registry sketch (item 13)

```go
// cleanup.go
var combatComponentCleaners []func(*common.EntityManager, *ecs.Entity)

func RegisterCombatComponentCleaner(fn func(*common.EntityManager, *ecs.Entity)) {
    combatComponentCleaners = append(combatComponentCleaners, fn)
}

func StripCombatComponents(manager *common.EntityManager, squadIDs []ecs.EntityID) {
    for _, squadID := range squadIDs {
        entity := manager.FindEntityByID(squadID)
        if entity == nil { continue }
        for _, fn := range combatComponentCleaners { fn(manager, entity) }
        // Position + IsDeployed remain here (always relevant)
        manager.UnregisterEntityPosition(entity)
        for _, unitID := range squadcore.GetUnitIDsInSquad(squadID, manager) {
            if u := manager.FindEntityByID(unitID); u != nil {
                manager.UnregisterEntityPosition(u)
            }
        }
        if d := common.GetComponentType[*squadcore.SquadData](entity, squadcore.SquadComponent); d != nil {
            d.IsDeployed = false
        }
    }
}

// combatstate/init.go
func init() {
    combatlifecycle.RegisterCombatComponentCleaner(func(_ *common.EntityManager, e *ecs.Entity) {
        if e.HasComponent(FactionMembershipComponent) {
            e.RemoveComponent(FactionMembershipComponent)
        }
    })
}
```

Each combat-only component owns its own cleanup. Adding a `MoraleComponent` later means writing one `init()` block, not editing `combatlifecycle`.

### Resolver-on-setup sketch (item 10)

```go
// contracts.go
type CombatSetup struct {
    // ... static fields ...
    Resolver CombatResolver // built by Prepare(), nil for no-op
}

// raid/starters.go — raid attaches its own resolver up front, no callback needed
func (s *RaidCombatStarter) Prepare(...) (*CombatSetup, error) {
    return &CombatSetup{
        ...,
        Resolver: &RaidRoomResolver{...}, // built with starter's captured state
    }, nil
}
```

The post-combat callback survives but only as a notification ("raid encounter ended"), not as a resolution branch.

---

## 5. Prevention Plan

| Gate | How |
|---|---|
| **No exported helpers without external callers** | Add a `make staticcheck` step running `staticcheck -checks U1000` before merge |
| **All new public functions in `combatlifecycle` need a test** | PR template checkbox; coverage gate `go test -coverpkg=game_main/mind/combatlifecycle ./...` ≥ 70% |
| **No new fields on `CombatSetup` without per-type validation** | After item 12, the per-type constructors enforce this by construction |
| **No new direct `combatstate.*Component` removals outside `combatlifecycle`** | Hook registry (item 13) makes the alternative obvious |

---

## 6. ROI Projection

Assume internal cost of $50/hr and one combat-related defect every two months currently costs ~6 hours (investigate + fix + retest, no production exposure but lost dev time):

| Investment | Effort | Expected return |
|---|---|---|
| Quick wins (items 1–4) | 2 h ($100) | Test noise eliminated, dead API removed; payback on first new test that no longer needs to filter Printf output |
| Sprint 1 (items 5–9) | 14 h ($700) | At ~70% coverage of reward + cleanup, expect ≥1 prevented production-class bug per quarter (~6 h saved) → break-even after 2–3 prevented bugs (~6 months) |
| Sprint 2–3 (items 10–13) | 16 h ($800) | Each new combat type (in roadmap) gets ~4 h cheaper to add. With 2+ types planned, payback inside 12 months |

**Recommended sequence:** Quick wins now → Sprint 1 next (the testing gap is the biggest concrete risk) → Sprint 2 only if a new combat type is on the near-term roadmap.

---

## TL;DR

The package is small (668 LOC), well-documented, and the contracts are sensible. **The single biggest debt item is testing — only `Reward.Scale` is covered, and every reward and cleanup path is unverified.** The structural debts (two dispatch paths, flat `CombatSetup` discriminated union, shotgun-surgery `StripCombatComponents`) are worth fixing but only become urgent when adding a new combat type. Priority: write tests, then refactor.
