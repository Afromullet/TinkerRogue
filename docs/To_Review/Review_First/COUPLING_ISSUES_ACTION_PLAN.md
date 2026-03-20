# TinkerRogue Coupling Issues: Consolidated Action Plan

**Date:** 2026-03-18
**Baseline:** DEPENDENCY_BOUNDARIES_v2.md
**Reviewers:** Karen (reality-check), Tactical-Simplifier (design analysis), Refactoring-Pro (duplication hunting)

---

## How This Document Was Created

Three independent automated reviewers analyzed the actual codebase against the claims in DEPENDENCY_BOUNDARIES_v2.md. Each reviewer had a different focus:

- **Karen** validated whether each "Area to Watch" is a real problem or theoretical concern, rating each as MUST FIX, SHOULD FIX, NICE TO HAVE, or LEAVE ALONE
- **Tactical-Simplifier** analyzed design and separation of concerns in the tactical and mind layers, looking for misplaced responsibilities and accidental complexity
- **Refactoring-Pro** searched for concrete code duplication, dead code, and redundant abstractions with file paths and line numbers

Their findings were deduplicated and consolidated into this prioritized action plan.

---

## Tier 1: MUST FIX (Correctness + Clear Violations)

### 1.1 BUG: Name-Based Enemy Detection in `squads/squadabilities.go`
- **File:** `tactical/squads/squadabilities.go` ~line 106
- **Problem:** `countEnemySquads()` identifies enemies by checking `squadData.Name[0] != 'P'`. Any squad whose name doesn't start with 'P' is counted as enemy. Player squads with custom names are misidentified. This exists because `squads` can't import `combat` (which owns faction data) without creating a cycle.
- **Fix:** Pass a faction-aware enemy counter function into `CheckAndTriggerAbilities` as a parameter: `enemyCounter func() int`. The caller in `combat`/`combatservices` has faction context and can provide the correct count.
- **Effort:** Low (~15 min)
- **Found by:** Tactical-Simplifier

### 1.2 Test Fixtures Compiled Into Production Binary
- **File:** `tactical/combat/combat_testing_2.go`
- **Problem:** This file exports `CreateTestCombatManager`, `CreateTestSquad`, etc. It imports `game_main/testing` but is NOT a `_test.go` file -- no build tags, no DEBUG_MODE gating. It compiles into the production binary unconditionally. This violates the project's own MEMORY.md rule about test-only functions.
- **Why it exists:** Test files in `mind/behavior/` need to call these helpers, but `_test.go` symbols aren't exported across packages. This is a known Go limitation.
- **Fix:** Create `tactical/combat/combattest/helpers.go` as an accepted Go test-helper export package. Move the 5 helper functions there. Update the 2-3 import sites (`combat_test.go`, `threat_layers_test.go`). The production `tactical/combat` package no longer imports `game_main/testing`.
- **Effort:** Low (~20 min)
- **Found by:** Karen, Refactoring-Pro

### 1.3 Dead Code: `calculateCounterattackDamage`
- **File:** `tactical/squads/squadcombat.go` ~lines 266-273
- **Problem:** Zero callers anywhere in the codebase. Superseded by `ProcessCounterattackOnTargets`.
- **Fix:** Delete it.
- **Effort:** 2 min
- **Found by:** Refactoring-Pro

---

## Tier 2: SHOULD FIX (Duplication + Coupling Reduction)

### 2.1 Combat Logic Living in `tactical/squads/` (Biggest Structural Issue)
- **Files:** `tactical/squads/squadcombat.go` (~700 lines), `tactical/squads/combatevents.go` (~160 lines)
- **Problem:** The squads package has ~22 fan-in because it contains full combat execution logic alongside squad data: damage calculation pipeline, target selection, hit/dodge/crit rolls, HP mutation, cover calculation, combat logging/snapshotting. Every package needing squad *data* also compile-depends on combat logic.
- **Evidence of duplication:** `combat.CombatActionSystem.ExecuteAttackAction()` (combatactionsystem.go:38-184) duplicates the flow from `squads.executeCombatPhase()` (squadcombat.go:41-87): initialize log, snapshot units, iterate attackers, process attacks/heals, apply damage, finalize.
- **Fix:** Extract combat calculation types and functions into `tactical/combat/`:
  1. Move `CombatResult`, `CombatLog`, `AttackEvent`, `HealEvent`, `CoverBreakdown` and event types to `tactical/combat/`
  2. Move `calculateDamage`, `processAttackWithModifiers`, `rollHit`/`rollCrit`/`rollDodge`, `SelectTargetUnits`, cover system to `tactical/combat/`
  3. Keep only data queries in `squads/` (unit lists, grid positions, health, roles)
  4. `ExecuteAttackAction` becomes the single combat execution path
- **Impact:** Reduces squads fan-in; makes squads a true data+query package
- **Effort:** High -- mechanical refactor touching many files, but low risk
- **Found by:** Tactical-Simplifier, Refactoring-Pro (both independently identified this)

### 2.2 `ComputeGenericAttackPattern` Duplicates `ComputeGenericPatternFiltered`
- **File:** `tactical/squads/squadqueries.go` lines 325-414
- **Problem:** ~40 lines of identical code. The only difference is a `healOnly bool` filter on attack type.
- **Fix:** Delete `ComputeGenericAttackPattern`, have its single caller use `ComputeGenericPatternFiltered` with appropriate parameters.
- **Effort:** 15 min
- **Found by:** Tactical-Simplifier, Refactoring-Pro

### 2.3 `rollHit`/`rollCrit`/`rollDodge` Are Identical Functions
- **File:** `tactical/squads/squadcombat.go` lines 618-638
- **Problem:** All three roll d100 and compare to threshold. Only parameter names differ.
- **Fix:** Replace with single `rollD100Check(threshold int) (roll int, passed bool)`.
- **Effort:** 5 min
- **Found by:** Refactoring-Pro

### 2.4 `GetDeployedSquads`/`GetReserveSquads` Differ by One Boolean
- **File:** `tactical/squads/squadroster.go` lines 69-91
- **Problem:** Identical loops, one checks `IsDeployed`, the other `!IsDeployed`.
- **Fix:** Extract `filterSquadsByDeployment(deployed bool)` private method.
- **Effort:** 10 min
- **Found by:** Refactoring-Pro

### 2.5 Speed Default-Fallback Pattern Duplicated 3 Times
- **Files:** `gear/artifactbehavior.go:43`, `tactical/combat/combatmovementsystem.go:39`, `tactical/combat/combatqueries.go:118`
- **Problem:** All three do: `speed := GetSquadMovementSpeed(); if speed == 0 { speed = config.DefaultMovementSpeed }`
- **Fix:** Add `GetSquadMovementSpeedOrDefault()` to `squads` package.
- **Effort:** 10 min
- **Found by:** Refactoring-Pro

### 2.6 `CanUndoMove`/`CanRedoMove` Identical Guard Logic
- **File:** `gui/guicombat/combat_action_handler.go` lines 196-229
- **Problem:** The "has selected squad acted?" check is copy-pasted between both methods.
- **Fix:** Extract `isSelectedSquadActed() bool` private method.
- **Effort:** 5 min
- **Found by:** Refactoring-Pro

### 2.7 `unitfilters.go` Has 4 Identical Filter-Loop Functions
- **File:** `tactical/squads/unitfilters.go` (entire file, 46 lines)
- **Problem:** Four functions with the same iterate-filter-append structure, differing only by predicate.
- **Fix:** Create `FilterUnits(predicate func(UnitTemplate) bool) []UnitTemplate`. Each existing function becomes a one-liner.
- **Effort:** 15 min
- **Found by:** Refactoring-Pro

### 2.8 `guicombat` Imports `mind/ai` for Bootstrapping (Not GUI Logic)
- **File:** `gui/guicombat/combatmode.go` ~line 89
- **Problem:** `CombatMode.Enter()` calls `ai.SetupCombatAI(...)` then immediately injects results into `combatService` via setters. This is wiring/initialization, not GUI logic. It adds `mind/ai` to guicombat's 25-import count unnecessarily.
- **Fix:** Move the `SetupCombatAI` call to wherever `CombatService` is created (e.g., `combatservices.NewCombatService()` or `combatlifecycle`). The GUI should not know AI exists.
- **Effort:** Medium (~30 min)
- **Found by:** Karen

### 2.9 `DirtyCache` in Wrong Package
- **File:** Currently in `mind/evaluation/`
- **Problem:** `DirtyCache` is a generic round-based cache invalidation utility with zero domain knowledge. It's imported by `gui/guicombat/threatvisualizer.go`, which adds `mind/evaluation` to guicombat's imports unnecessarily.
- **Fix:** Move `DirtyCache` to `common/` or `common/cache/`. It's a generic utility.
- **Effort:** Low (~15 min)
- **Found by:** Karen

### 2.10 `BehaviorContext.Cache` Exposes Concrete Combat Type
- **File:** `gear/artifactbehavior.go`
- **Problem:** `Cache *combat.CombatQueryCache` is a public field. Artifact behaviors directly call `ctx.Cache.FindActionStateBySquadID()` in 6 locations. The context already has facade methods for some operations but the Cache field leaks through for action state manipulation.
- **Fix:** Add facade methods: `GetActionState(squadID)`, `SetSquadLocked(squadID)`, `ResetSquadActions(squadID, speed)`. Then unexport the `Cache` field. Narrows the contract without new interfaces.
- **Effort:** Low (~20 min)
- **Found by:** Tactical-Simplifier

---

## Tier 3: NICE TO HAVE (Low Impact, Opportunistic)

### 3.1 `gui/builders` Importing `tactical/squads`
- **File:** `gui/builders/lists.go`
- **Problem:** Generic UI infrastructure imports domain types (`squads.GetSquadName`, `squads.UnitIdentity`). The squad-specific functions are not actually generic.
- **Fix:** Move `CreateSquadList`/`CreateUnitList` to `gui/guisquads/`. The rest of `gui/builders/` is already clean.
- **Impact:** Cosmetic. Zero practical harm currently.
- **Found by:** Karen

### 3.2 `encounter.SpawnCombatEntities` Does Too Much
- **File:** `mind/encounter/encounter_setup.go` lines 29-76
- **Problem:** Mixes encounter spec generation with combat infrastructure setup (factions, action states). Could delegate combat entity creation to a combat-layer function.
- **Fix:** Split so encounter produces the spec and a `combat`/`combatlifecycle` function sets up factions. Best done after 2.1 clarifies combat package boundaries.
- **Found by:** Tactical-Simplifier

### 3.3 `behavior/threat_combat.go` Direct Field Access
- **File:** `mind/behavior/threat_combat.go` ~line 63
- **Problem:** Directly accesses `ctl.baseThreatMgr.factions[enemyFactionID]` internal maps instead of using accessor methods on `FactionThreatLevelManager`.
- **Fix:** Add accessor methods to `FactionThreatLevelManager`.
- **Found by:** Tactical-Simplifier

### 3.4 `setupBehaviorDispatch` Creates `BehaviorContext` 3 Times
- **File:** `tactical/combatservices/combat_service.go` lines 298-322
- **Fix:** Extract `makeBehaviorContext()` method.
- **Found by:** Refactoring-Pro

### 3.5 `CheckVictoryCondition` Could Be a Pure Function
- **File:** `tactical/combatservices/combat_service.go`
- **Fix:** Extract as `CheckVictory(factions, queryCache, manager) *VictoryCheckResult` for testability.
- **Found by:** Tactical-Simplifier

---

## Issues Reviewed and Determined Safe to LEAVE ALONE

### gear.BehaviorContext -> combat.CombatQueryCache (overall coupling)
The DEPENDENCY_BOUNDARIES_v2.md called this "the tightest cross-domain coupling." In reality, the coupling surface is narrow: 1 method on 1 struct, field mutations on 1 data type. Artifacts that skip turns and grant bonus attacks MUST modify `ActionStateData` -- this is correct by design. The facade fix (2.10 above) narrows the contract sufficiently without requiring full interface extraction. Adding an interface still exposes `*combat.ActionStateData` in the return type, so you haven't actually removed the dependency -- you've only hidden the cache implementation. The actual tight coupling is to `ActionStateData` fields, not to `CombatQueryCache` as a struct.

### mind/encounter breadth (10 imports)
Each import is used in a specific file for a specific reason. The package is already well-decomposed internally (encounter_generator.go, encounter_setup.go, encounter_service.go, resolvers.go, starters.go each import only what they need). It's the legitimate bridge between overworld and tactical domains. The coupling is inherent to the game design -- you cannot separate it further without creating a third package that imports both, which achieves nothing except moving the coupling.

### combatservices as "God Service"
The 7 responsibilities are appropriate for a game orchestrator. Interface injection is already in place (`AITurnController`, `ThreatProvider`). The callback registration pattern correctly keeps dependency arrows pointing downward. Minor cleanups (3.4, 3.5) are sufficient -- no structural split needed.

### gui/guicombat fan-out (25 imports)
22 of the 25 imports are legitimate -- a combat GUI mode must read combat state, call services, and display panels. The package is already well-decomposed into distinct files (combatmode.go, combat_turn_flow.go, combat_action_handler.go, combat_animation_mode.go, combatvisualization.go, threatvisualizer.go). The 2 illegitimate imports (`mind/ai` in 2.8, `mind/evaluation` via DirtyCache in 2.9) are addressed above. No further decomposition needed.

### common/ ECS utility changes (fan-in ~37)
Despite the highest fan-in, the API surface (`GetComponentType`, `GetComponentTypeByID`, `EntityID` patterns) is small and extremely stable. Low practical risk despite the theoretical blast radius.

### overworld/core/ data structure changes (fan-in ~11)
Well-structured internal layering with minimal external coupling. Changes stay within the overworld domain. The overworld subsystem is the best-structured domain in the codebase.

---

## Recommended Execution Order

| Phase | Items | Effort | Rationale |
|-------|-------|--------|-----------|
| 1 - Quick Wins | 1.1, 1.2, 1.3, 2.2, 2.3, 2.4, 2.5, 2.6, 2.7 | ~80 min | Bug fix + dead code + all small dedup items |
| 2 - Coupling Reduction | 2.8, 2.9, 2.10 | ~65 min | Remove bad imports from guicombat, narrow gear contract |
| 3 - Big Structural | 2.1 | 2-3 hrs | Move combat logic out of squads (the biggest win) |
| 4 - Opportunistic | 3.1-3.5 | ~60 min | Do alongside related work |

**Estimated total line reduction from Tier 1+2: ~200-250 lines**

---

## Verification After Each Phase

1. `go build ./...` -- confirms no import cycles or compilation errors
2. `go test ./...` -- confirms no behavioral regressions
3. `go vet ./...` -- confirms no new issues
4. For Phase 3 (structural move): verify fan-in of `tactical/squads/` has decreased by checking imports

---

## Combat Pipeline Review (2026-03-20)

**Context:** The Tactical-Simplifier reviewed the combat pipeline (`mind/combatpipeline`, `mind/encounter`, `tactical/combatlifecycle`) on 2026-03-18 and proposed 6 simplifications. Commits `4d89a7b`, `916a2e1`, and `d6344ef` addressed most of them.

### Completed (4 of 6)

| # | Proposal | How Resolved |
|---|----------|-------------|
| 1 | CombatType enum (replace boolean flags) | `CombatType` enum in `combat_contracts.go` with 4 types |
| 2 | AssignSquadsToFaction helper | 3 helpers in `combatlifecycle/enrollment.go` |
| 4 | Flatten ExitCombat dispatch | Single unified exit with snapshot pattern |
| 6 | Battle log export in CombatMode.Exit | Properly isolated with config guard |

### Remaining (2 of 6)

#### CP-3: CalculateTargetPower Helper

- **Files:** `mind/encounter/encounter_generator.go` lines 48-62, `mind/encounter/starters.go` lines 121-130
- **Problem:** Identical calculate-average-power-and-scale pattern duplicated. Both iterate squad IDs, sum power via `evaluation.CalculateSquadPower`, average, then clamp with `combatlifecycle.ClampPowerTarget`. Only difference is variable names and squad source.
- **Fix:** Extract `CalculateTargetPower(manager, squadIDs, level) (float64, templates.JSONEncounterDifficulty)` in `combatlifecycle/helpers.go` or `encounter/`.
- **Effort:** ~30 min, ~25 lines dedup

#### CP-5: Split OverworldCombatResolver.Resolve

- **File:** `mind/encounter/resolvers.go` lines 26-132
- **Problem:** Method grew to 107 lines (was 65 at review time). Handles casualty counting, threat node validation, victory with full destruction, victory with weakening, and player defeat. The victory path alone is 58 lines with duplicated logging/reward logic between branches.
- **Fix:** Extract `resolveVictory(manager, threatEntity, nodeData)` and `resolveDefeat(manager, threatEntity, nodeData)` private helpers. Pure readability improvement, no behavior change.
- **Effort:** ~20 min, 0 net line change
