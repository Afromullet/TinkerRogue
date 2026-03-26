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

## Remaining Items

### 3.2 `encounter.SpawnCombatEntities` Does Too Much
- **File:** `mind/encounter/encounter_setup.go` lines 29-76
- **Problem:** Mixes encounter spec generation with combat infrastructure setup (factions, action states). Could delegate combat entity creation to a combat-layer function.
- **Fix:** Split so encounter produces the spec and a `combat`/`combatlifecycle` function sets up factions. Best done after 2.1 clarifies combat package boundaries.
- **Found by:** Tactical-Simplifier

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

## Completed Items (2026-03-26)

| Item | Description | How Resolved |
|------|-------------|-------------|
| 1.1 | Name-based enemy detection bug | Replaced `Name[0] != 'P'` with faction-aware `GetSquadFaction` query in `combatabilities.go` |
| 1.2 | Test fixtures in production binary | Already resolved (file removed in prior refactoring) |
| 1.3 | Dead code `calculateCounterattackDamage` | Already resolved (deleted in prior refactoring) |
| 3.1 | `gui/builders` importing `tactical/squads` | Moved `CreateSquadList`/`CreateUnitList` to `gui/guisquads/squadlists.go` |
| 3.3 | Direct field access in `threat_combat.go` | Added `GetSquadThreatLevel` accessor to `FactionThreatLevelManager` |
| 3.4 | `BehaviorContext` created 3 times | Extracted `makeBehaviorContext` closure in `setupBehaviorDispatch` |
| CP-3 | Duplicate power calculation pattern | Extracted `calculateTargetPower` helper in `encounter_config.go` |
| CP-5 | `Resolve` method too long (107 lines) | Split into `resolveVictory` and `resolveDefeat` private helpers |

### Combat Pipeline Review (2026-03-20)

| # | Proposal | How Resolved |
|---|----------|-------------|
| 1 | CombatType enum (replace boolean flags) | `CombatType` enum in `combat_contracts.go` with 4 types |
| 2 | AssignSquadsToFaction helper | 3 helpers in `combatlifecycle/enrollment.go` |
| 3 | CalculateTargetPower helper | `calculateTargetPower` in `encounter_config.go` (2026-03-26) |
| 4 | Flatten ExitCombat dispatch | Single unified exit with snapshot pattern |
| 5 | Split OverworldCombatResolver.Resolve | `resolveVictory`/`resolveDefeat` helpers (2026-03-26) |
| 6 | Battle log export in CombatMode.Exit | Properly isolated with config guard |
