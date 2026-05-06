# Perks Package — Technical Debt Analysis

**Package:** `tactical/powers/perks`
**Date:** 2026-04-30

---

## Overall Assessment

The package is well-structured for its current size. The core architecture (hook interface, two-tier state maps, dispatcher) is solid with no serious design flaws. Debt is concentrated in validation ergonomics, test coverage gaps, and one naming ambiguity that carries a real bug risk. None of this is emergency work, but items 1 and 2 should be addressed before the next round of perk additions.

---

## Debt Inventory

### 1. CRITICAL — Silent zero-damage bug if JSON fails to load

**Files:** `balanceconfig.go`, `perks_test.go`

`validatePerkBalance` prints warnings to stdout and returns nothing. If `LoadPerkBalanceConfig` fails (missing file, bad JSON), all `PerkBalance` fields stay at zero. A `DamageMultiplier *= 0.0` produces zero damage — the behavior compiles, ships, and is only caught in play.

`TestPerkBalanceConfig_ZeroFieldsAreDetectable` in the test file explicitly documents this failure mode without fixing it. Compare to `ValidateHookCoverage()` in `registry.go`, which correctly returns `[]error`.

**Fix:** Change `validatePerkBalance` to return `[]error`. Have `LoadPerkBalanceConfig` log or fatal on non-empty results.
**Effort:** ~1 hour

---

### 2. HIGH — `DispatchAttackTracking` parameter names imply wrong ID type

**File:** `dispatcher.go:203-212`

```go
func (d *SquadPerkDispatcher) DispatchAttackTracking(attackerID, defenderID ecs.EntityID, ...) {
    attackerState := GetRoundState(attackerID, manager)  // PerkRoundStateComponent is on squads
    defenderState := GetRoundState(defenderID, manager)
```

`GetRoundState` looks for `PerkRoundStateComponent`, which is attached to **squads**, not units. The parameter names `attackerID`/`defenderID` suggest unit IDs. If the caller passes unit IDs, `GetRoundState` returns nil and tracking is silently dropped — `AttackedThisTurn` never gets set, breaking `Counterpunch` and `Deadshot's Patience`.

**Fix:** Rename parameters to `attackerSquadID, defenderSquadID` to match every other method in the dispatcher. Add a comment stating squad IDs are required.
**Effort:** ~15 minutes

---

### 3. HIGH — Role strings hardcoded in `LoadPerkDefinitions`

**File:** `registry.go:146-148`

```go
if role != "Tank" && role != "DPS" && role != "Support" {
```

Hardcodes role name strings instead of using constants from `unitdefs`. If `unitdefs.RoleTank.String()` ever changes spelling, this validation silently stops catching bad data. The same concept is represented as both a `unitdefs.UnitRole` enum and a bare string with no compile-time link between them.

**Fix:** Replace string literals with `unitdefs.RoleTank.String()`, etc., or build the valid-role set from unitdefs constants programmatically.
**Effort:** ~30 minutes

---

### 4. MEDIUM — `setupTestBalance()` is incomplete; future behavior tests get zero values

**File:** `perks_test.go:33-45`

`setupTestBalance()` only populates balance values for stateful perks. Stateless perks — `BraceForImpact`, `ExecutionersInstinct`, `ShieldwallDiscipline`, `IsolatedPredator`, `FieldMedic`, `LastLine`, `Cleave`, `GuardianProtocol` — are absent. Any future test that exercises a stateless perk behavior will read zero-valued balance fields, producing silent wrong results (zero multipliers, not errors). Debt item #1 already shows this produces zero damage without panicking.

**Fix:** Complete `setupTestBalance()` to include all perks that read from `PerkBalance`.
**Effort:** ~20 minutes

---

### 5. MEDIUM — `PerkRoundState` comment table will drift

**File:** `components.go:25-32`

The comment block at the top of `PerkRoundState` is a manually maintained table mapping each shared field to which hook writes it, reads it, and resets it. It is already slightly misleading: `Fortify.TurnStart` resets `TurnsStationary` to 0 on movement, but the table attributes this to `OnMoveComplete`, collapsing two distinct code sites.

This is documentation debt — the table will diverge from reality as perks change.

**Fix:** No immediate code change. As a discipline: when adding a new shared field, annotate the write site in the behavior with a comment instead of updating the central table. Consider removing the table and relying on field-level comments on `PerkRoundState` itself.

---

### 6. LOW — Stateless perks with non-trivial logic have no unit tests

**File:** `perks_test.go`

All per-round and per-battle stateful perks are well tested. Stateless perks are entirely untested, including two with non-trivial logic:

- **`CleaveBehavior`** — modifies both the target list (`TargetOverride`) and applies a damage penalty (`AttackerDamageMod`). The only perk that touches two hook types simultaneously.
- **`GuardianProtocolBehavior`** — iterates friendly squads, checks Chebyshev adjacency, finds a tank, and redirects a fraction of damage via the early-exit `DamageRedirect` hook.

Simple stateless perks (`Vigilance`, `BraceForImpact`) are safe to leave untested — their logic is a single-line field assignment.

**Fix:** Add unit tests for `Cleave` and `GuardianProtocol` specifically.
**Effort:** ~1 hour

---

### 7. LOW — `behaviors.go` is 643 lines and growing

**File:** `behaviors.go`

All 21 perks live in one file, grouped by lifecycle tier with comment headers. At 21 perks this is navigable. The risk is that it becomes the default place to add new perks without question, and splitting becomes more disruptive the longer it waits.

**Fix when triggered (around 35 perks):** Split into `behaviors_stateless.go`, `behaviors_round.go`, `behaviors_battle.go`. The `init()` registration block moves to `behaviors_init.go` or stays in `behaviors.go`.
**Effort:** ~1 hour when triggered

---

### 8. LOW — `TestValidatePerkBalance_ZeroMultipliersWarn` is a non-test

**File:** `perks_test.go:827-834`

```go
func TestValidatePerkBalance_ZeroMultipliersWarn(t *testing.T) {
    cfg := &PerkBalanceConfig{}
    // Should not panic
    validatePerkBalance(cfg)
}
```

This only verifies the function doesn't panic. It cannot assert warnings were produced because `validatePerkBalance` returns nothing. Fixing debt item #1 (making it return `[]error`) makes this test upgradeable to actually assert validation failures.

---

### 9. LOW — `GetUnitsInRow` always allocates a dedup map

**File:** `unithelpers.go:127`

```go
seen := make(map[ecs.EntityID]bool)
```

Allocated on every call to prevent a unit from appearing twice if it occupies multiple grid cells. At squad sizes of 6–9 units this is negligible, but `GetUnitsInRow` is called per-attack for `Cleave`. Not worth fixing until allocation profiling points here.

---

## Priority Ranking

| # | Item | Severity | Effort |
|---|------|----------|--------|
| 1 | `validatePerkBalance` returns void — silent zero-damage bug | Critical | ~1 hr |
| 2 | `DispatchAttackTracking` parameter names imply wrong ID type | High | 15 min |
| 3 | Role strings hardcoded in `LoadPerkDefinitions` | High | 30 min |
| 4 | `setupTestBalance()` incomplete — future tests get zero values | Medium | 20 min |
| 5 | Tests for `Cleave` and `GuardianProtocol` | Medium | ~1 hr |
| 6 | `PerkRoundState` comment table will drift | Medium | Ongoing discipline |
| 7 | `behaviors.go` size (revisit at ~35 perks) | Low | 1 hr when triggered |
| 8 | `TestValidatePerkBalance_ZeroMultipliersWarn` non-test | Low | Follows fix #1 |
| 9 | `GetUnitsInRow` dedup map allocation | Low | Negligible now |

Items 1–4 are a ~2 hour block that eliminates all real bug risks. Items 5–9 are quality improvements with no urgency.
