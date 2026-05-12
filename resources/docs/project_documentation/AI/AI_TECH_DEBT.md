# Technical Debt Analysis — AI Subsystem (`mind/ai/`, `mind/evaluation/`)

**Scope:** ~1,278 LOC across `mind/ai/` (721 LOC) and `mind/evaluation/` (557 LOC including test).
**Date:** 2026-05-12
**Companion document:** `BEHAVIOR_TECH_DEBT.md` covers `mind/behavior/`. This document covers the *rest* of the AI stack: the controller, the action evaluator, and the shared power-evaluation module. Cross-package issues are flagged below.

**External callers of `mind/ai/`:** `setup/gamesetup/moderegistry.go`, `tactical/combat/combatservices/combat_service.go`, `gui/guicombat/combat_turn_flow.go`.
**External callers of `mind/evaluation/`:** `mind/spawning/`, `mind/behavior/dangerlevel.go`, `campaign/raid/deployment.go`.

---

## 1. Debt Inventory

### 🔴 HIGH — Performance: Per-tile re-computation of position-invariant data

**Location:** `mind/ai/action_evaluator.go:190–212` (`scoreMovementPosition`) → `:245–288` (`scoreApproachEnemy`) → `:291–319` (`findNearestEnemy`), `:385–405` (`getMaxAttackRange`).

For every candidate movement tile generated in `evaluateMovement` (up to `(2·moveRange+1)² − 1` tiles per squad turn), `scoreMovementPosition` is called. Inside:

- `scoreApproachEnemy` calls `findNearestEnemy()` — which iterates **all factions × all squads × position lookup** to find the nearest enemy. **The result depends only on `CurrentPos`, not on `pos`.**
- `scoreApproachEnemy` then calls `getMaxAttackRange()` — which iterates **every unit in the squad** to find the max range. **The result is invariant within a turn.**

For `moveRange = 3` (49 tiles), 4 factions × 5 squads, that is ~49 × 20 = **980 redundant faction/squad scans per squad turn**, plus 49 redundant unit-component scans. Both values should be computed once per `ActionContext` and reused.

**Impact:** O(T × F × S) work where O(F × S + T) suffices. On a typical encounter (~5 enemy squads, moveRange 3), each squad turn does roughly 20–50× more `GetSquadMapPosition` calls than necessary. Compounded across all AI squads per turn, this is the second-biggest AI hot-path waste after the behavior-package full-grid sweeps (see `BEHAVIOR_TECH_DEBT.md` item 1).

**Fix (low effort, ~1 hr):**
- Cache `nearestEnemy`, `nearestDistance`, `maxAttackRange` as fields on `ActionEvaluator` (or `ActionContext`), populated in `NewActionEvaluator`.
- `scoreApproachEnemy` reads from the cache instead of recomputing.
- ZoC scan in `scoreZoCRisk` already only checks 9 tiles around `pos` so it actually does need per-tile evaluation — leave alone.

### 🔴 HIGH — Unconditional debug logging in AI hot path

**Location:** `mind/ai/action_evaluator.go:66–77`.

```go
fmt.Printf("[AI] %s attacked %s\n", attackerName, defenderName)
if result.TargetDestroyed {
    fmt.Printf("[AI] %s was destroyed!\n", defenderName)
}
...
fmt.Printf("[AI] Attack failed: %s\n", result.ErrorReason)
```

These `fmt.Printf` calls run on every AI attack in release builds. Per the project's memory note (`MEMORY.md` → "Testing package"), debug-only code is expected to be gated by `config.DEBUG_MODE`. The AI package has **no such gating anywhere** (`grep -n DEBUG_MODE mind/**/*.go` → no matches).

The same pattern is endemic across `mind/encounter/` and `mind/combatlifecycle/` (~14 unconditional `fmt.Printf`s) but those are out of scope for this doc — flagged here so the AI fixes don't ship without the conventions matching the rest of the codebase.

**Impact:** Console spam in production; minor stdout-flush latency per attack; non-compliance with the project's debug-gating convention.

**Fix (~30 min):** Wrap each `fmt.Printf` in `if config.DEBUG_MODE { ... }`, or route through a `debug.Log("[AI] ...")` helper. The `defenderName == "Unknown"` fallback at lines 60–65 should also fall away — if `result.Success == true`, `result.CombatLog` is non-nil; the nil-check is defensive code for a state that doesn't happen.

### 🔴 HIGH — Asymmetric power calculation: entity vs. template

**Location:** `mind/evaluation/power.go:97–105` (`CalculateUtilityPower`) vs. `:359–365` (heal block in `EstimateUnitPowerFromTemplate`).

The entity-based utility path:
```go
func CalculateUtilityPower(entity, attr, roleData, config) float64 {
    return calculateRoleValue(roleData) + calculateAbilityValue(entity) + calculateCoverValue(entity)
}
```

The template-based path:
```go
healValue := 0.0
if unit.AttackType == unitdefs.AttackTypeHeal {
    healValue = float64(attr.GetHealingAmount()) * 1.5
}
utilityPower := roleValue + abilityValue + coverValue + healValue   // <-- heal included
```

**Same unit, two scores.** The encounter generator's power budget includes heal contribution; the AI's threat assessment of that same heal-capable squad does not. This means:

1. Encounter generation creates squads at "1000 power" that the AI then thinks are worth ~850 power for threat calculations.
2. Veterancy/perk changes that bump healing amounts will silently mis-calibrate AI behavior while encounter generation stays correct.

Worse, `EstimateUnitPowerFromTemplate` also **inlines** offensive, defensive, and utility math instead of calling `CalculateOffensivePower` / `CalculateDefensivePower` / `CalculateUtilityPower` (it only calls `CalculateOffensivePower`). Both copies will drift independently.

**Fix (medium effort, ~2 hr):**
- Add a `healValue` branch to `calculateAbilityValue` or a separate `calculateHealValue(entity)` helper so the entity path includes heal contribution.
- Refactor `EstimateUnitPowerFromTemplate` to call `CalculateDefensivePower` directly. Templates can synthesize a transient `*common.Attributes` with `CurrentHealth = MaxHealth`, eliminating the duplicated defensive block (lines 327–340).
- Add a regression test asserting `EstimateUnitPowerFromTemplate(tmpl) == calculateUnitPower(spawn(tmpl))` for a heal unit, a leader, and a cover provider.

### 🟠 MEDIUM — Dead `config` parameter on three public functions

**Location:** `mind/evaluation/power.go:57`, `:73`, `:97`.

```go
func CalculateOffensivePower(attr *common.Attributes, config *PowerConfig) float64 { ... }
func CalculateDefensivePower(attr *common.Attributes, config *PowerConfig) float64 { ... }
func CalculateUtilityPower(entity, attr, roleData, config *PowerConfig) float64 { ... }
```

**None of these three functions read from `config`.** Internal call sites pass it through; external callers (if any existed) would assume the parameter influences behavior. It does not. This is dead API surface and a future-bug magnet: any developer wiring a new profile-aware tweak ("scale offense by config.OffensiveWeight inside CalculateOffensivePower") will introduce double-counting because the weight is *already* applied in `calculateUnitPower` (lines 48–50).

**Fix (~15 min):** Drop the parameter from all three signatures. They become pure utility functions. Callers in `calculateUnitPower` and `EstimateUnitPowerFromTemplate` stop passing it.

### 🟠 MEDIUM — JSON-fallback duplication (same anti-pattern as behavior package)

**Location:** `mind/evaluation/roles.go:32–41`, `:54–67`, `:79–89`.

Three accessor functions (`GetRoleMultiplierFromConfig`, `GetAbilityPowerValue`, `GetCompositionBonusFromConfig`) all follow the same pattern:

```go
for _, rm := range templates.PowerConfigTemplate.RoleMultipliers {
    if rm.Role == roleStr { return rm.Multiplier }
}
// Fallback to default values
switch role {
case unitdefs.RoleTank:    return 1.2
case unitdefs.RoleDPS:     return 1.5
case unitdefs.RoleSupport: return 1.0
default:                   return 1.0
}
```

The fallback `switch` mirrors `powerconfig.json` line-for-line. This is the **same anti-pattern** flagged in `BEHAVIOR_TECH_DEBT.md` item 2 for `GetRoleBehaviorWeights`. The reasoning there applies verbatim: the JSON is the source of truth, the fallback duplicates that truth in Go, and the duplicates will rot when designers tune the JSON without recompiling.

The validation hook at `templates/validation.go:207` already requires the `"Balanced"` profile to exist at startup, so the fallback is dead in practice — config load failures already panic before any of these functions execute.

**Fix (~1 hr):** Delete all three `switch` fallbacks. If a value is missing from the JSON, return `0.0` and let `templates/validation.go` enforce completeness at startup. Drops ~40 LOC across the three functions.

### 🟠 MEDIUM — `aiController` pointer leaked through SquadAction interface

**Location:** `mind/ai/action_evaluator.go:48–81` (`AttackAction`), `:181–198` (`ActionContext`).

`AttackAction` carries an `aiController *AIController` field for the sole purpose of calling `QueueAttack(attackerID, defenderID)` after a successful attack. The pointer is plumbed in via `ActionContext.AIController` (`ai_controller.go:193`) → `evaluateAttacks` (`action_evaluator.go:335`) → `AttackAction{aiController: ...}`.

This is a leaky abstraction:
- `SquadAction.Execute(manager, movementSystem, combatActSystem, cache) bool` is the abstract contract.
- `AttackAction` smuggles a fifth dependency in via a constructor field, bypassing the interface.
- `ActionContext` is mostly read-only context (squad ID, role, position, threat eval), but `AIController` is a write-side dependency that doesn't fit.

The animation queue is the *only* reason this pointer exists. There's a cleaner shape: `Execute` returns `(success bool, queuedAttacks []QueuedAttack)`, or `SquadAction` has a separate `PostEffects() []QueuedAttack` method, or the queue is passed as a small interface (`AttackQueueSink`) rather than the whole AIController.

**Fix (medium effort, ~1.5 hr):** Replace the `*AIController` field on `AttackAction` with a `queue interface{ Push(QueuedAttack) }` minimal interface. Pass it via `ActionContext.AttackQueue`. Removes the circular-feeling reference from action data back to the controller that created it.

### 🟠 MEDIUM — `findNearestEnemy` reimplements `behavior.ThreatLayerBase.getEnemyFactions`

**Location:** `mind/ai/action_evaluator.go:291–319`, `mind/behavior/threat_layers.go:33–44`.

Both functions walk `combatstate.GetAllFactions` and filter out the viewing faction. `findNearestEnemy` then additionally iterates squads and computes distance — but the first half is duplicated logic, and the AI controller already holds a `CompositeThreatEvaluator` per faction (via `ActionContext.ThreatEval`) that *could* expose this.

The simpler win: extract `GetEnemySquads(factionID, manager) []ecs.EntityID` into `combatstate` (where `GetAllFactions` and `GetActiveSquadsForFaction` already live). Both `findNearestEnemy` and `getAttackableTargets` need it, and so does every threat-layer in `behavior/`.

**Fix (~30 min):** Add `combatstate.GetEnemySquadsForFaction(factionID, manager)`. Use it from `findNearestEnemy`, `getAttackableTargets`, and `ThreatLayerBase.getEnemyFactions`. Removes ~25 LOC of triple-nested faction loops.

### 🟠 MEDIUM — Profile system over-engineered for a single profile

**Location:** `resources/assets/gamedata/powerconfig.json:2–10`, `mind/evaluation/power_config.go:38–57`, `mind/spawning/types.go:35`.

The `profiles` array in `powerconfig.json` contains exactly one entry (`"Balanced"`). `GetPowerConfigByProfile` does a linear scan over an array of length 1, gated by a const `DefaultPowerProfile = "Balanced"` in a different package. `templates/validation.go:207` enforces `"Balanced"` exists at startup.

This is YAGNI infrastructure. Three options:

1. **Commit to multiple profiles.** Add at least one alternate (e.g., `"Aggressive"`, `"Defensive"`) and a way for designers/encounters to pick. The infrastructure exists, but nothing uses it.
2. **Flatten.** Move offensive/defensive/utility weights to top-level keys in `powerconfig.json` and drop the `profiles` array, the `GetPowerConfigByProfile` function, and the `"Balanced"` magic string. Saves ~20 LOC.
3. **Status quo with a comment.** Document the profile system as "designed for future tuning experiments" so it doesn't get ripped out.

Recommended: option 2 unless there's a concrete near-term plan to use profiles.

### 🟠 MEDIUM — `WaitAction` mutates action state as control-flow signal

**Location:** `mind/ai/action_evaluator.go:88–100`, `mind/ai/ai_controller.go:101–128`.

The AI turn loop:
```go
for {
    actionState := ... // get squad's action state
    if actionState.HasMoved && actionState.HasActed { break }
    ctx := NewActionContext(squadID, aic)
    if !aic.executeSquadAction(ctx) { break }
}
```

`executeSquadAction` *always* finds an action — at minimum, `WaitAction` is appended with score `0.0` (line 127–130). For the loop to terminate when no real action is appealing, `WaitAction.Execute` forces termination by **mutating both `HasMoved` and `HasActed` to true**:

```go
actionState.HasMoved = true
actionState.HasActed = true
return true
```

This is a control-flow hack. The contract "selected action triggered turn-end" is implicit and easy to break — e.g., adding a "Defend" action with a low score but `actionState`-preserving `Execute` would silently infinite-loop. The hack works but it's the kind of subtle correctness contract that doesn't survive refactoring.

**Fix (low effort, ~30 min):** Make `executeSquadAction` return `(executed bool, terminate bool)` or a tri-state enum. The loop terminates on `terminate == true` regardless of how the action mutates state. `WaitAction.Execute` returns `(true, true)`. Movement/attack return `(true, false)`. No state-mutation indirection.

### 🟡 LOW — No tests for `mind/ai/`

**Location:** Entire `mind/ai/` package.

```
mind/ai/action_evaluator.go   450 LOC
mind/ai/ai_controller.go      271 LOC
mind/ai/*_test.go             0 LOC
```

The action-scoring functions contain numerous magic thresholds that materially shape AI behavior:
- Attack base score = 100, must exceed movement base score = 50 (commented at line 420–422 as a CRITICAL invariant — but no test enforces it).
- Approach multipliers: Tank 15, DPS 8, Support −5.
- In-range bonuses: +20 if `dist ≤ maxRange`, +10 if `dist ≤ maxRange+2`.
- ZoC penalties: Tank 0, DPS −5, Support −15, default −3.
- Focus-fire bonus: `(1 − healthPct) × 20`.
- Healer-priority bonus: +10.
- Role-counter bonuses: +10 for DPS-vs-Support and Tank-vs-DPS.

Each is a number that could be twiddled to "tune" behavior and silently break the relative ordering that makes the AI play sensibly. The most fragile invariant — "attack always beats movement when in range" — is a single line that would break the moment movement base climbs to 101 or attack base drops to 49.

**Fix (medium effort, ~3 hr):** Add tests covering:
- `scoreAttackTarget` for a representative target returns > max `scoreMovementPosition` for any candidate tile (the attack-beats-movement invariant).
- `scoreApproachEnemy` returns positive value for Tank moving closer, negative for Support moving closer.
- `scoreZoCRisk` returns 0 for Tank, negative for Support, when an enemy is adjacent.
- `evaluateAttacks` returns no actions when no enemies in range.
- `SelectBestAction` returns highest-score action, tie-breaks deterministically (or document it doesn't).

### 🟡 LOW — `ActionContext` allocated per-action-iteration

**Location:** `mind/ai/ai_controller.go:101–127`.

Inside the per-squad action loop, `NewActionContext(squadID, aic)` is called every iteration. It re-derives:
- `factionID` (lookup by squad ID — but the outer loop already knows `factionID`).
- `actionState` (cache lookup — already done at the top of the iteration on line 105 to test loop termination, then re-fetched here).
- `SquadRole` (component scan).
- `CurrentPos` (component scan via `GetSquadMapPosition`).

Of these, only `CurrentPos` can change between iterations (after a `MoveAction`). The others are stable for the duration of one squad's turn. Re-doing four ECS lookups per inner-loop iteration is wasted, though not on the same magnitude as the per-tile redundancy in HIGH item 1.

**Fix (~30 min):** Build `ActionContext` once per squad outside the loop. Update `CurrentPos` after `MoveAction.Execute` returns true. Move `actionState` reads to use the cached pointer (the cache returns a pointer, so mutations stay visible).

### 🟡 LOW — Stale "moved elsewhere" comment

**Location:** `mind/ai/ai_controller.go:269–271`.

```go
// NOTE: getSquadPrimaryRole and calculateSquadHealthPercent have been moved to
// squads.GetSquadPrimaryRole() and squads.GetSquadHealthPercent() respectively.
// These centralized functions eliminate code duplication across ai and behavior packages.
```

This is a migration breadcrumb. Per CLAUDE.md code-style guidance: comments should not reference removals or history (that's commit-message territory). Same category as `BEHAVIOR_TECH_DEBT.md` item 6.

**Fix (~5 min):** Delete the comment block.

### 🟡 LOW — `TurnManager` and `combatCache` fields on `AIController` unused

**Location:** `mind/ai/ai_controller.go:18–24`.

`AIController` stores `turnManager *combatcore.TurnManager` (set at construction, never read) and `combatCache *combatstate.CombatQueryCache` (passed through to `NewActionContext` via `aic.combatCache.FindActionStateBySquadID` — used, but ambiguously: the cache is also accessible via the controllers it's wired into). A quick `grep` on `aic\.turnManager` returns zero matches.

**Fix (~5 min):** Remove `turnManager` from the struct and from `SetupCombatAI`. Caller in `setup/gamesetup/moderegistry.go` can stop wiring it.

### 🟡 LOW — TODO without owner or date

**Location:** `mind/ai/ai_controller.go:84–86`.

```go
// TODO: AI spell casting - enemy commanders don't cast spells yet.
// When implemented: check if faction has a commander with mana/spells,
// evaluate spell value vs saving mana, pick target, call spells.ExecuteSpellCast.
```

Per CLAUDE.md comment guidance: TODOs should include context (effort estimate, owner, or tracking issue). This one's been here since the spell system was built and is essentially a feature stub. Either:
- Link to a tracking issue / progression-system doc, or
- Implement it (the spell system already exists, see `tactical/powers/spells/`), or
- Delete the comment and let the absence of the feature speak for itself.

---

## 2. Prioritized Remediation Plan

| # | Item | Effort | Impact | ROI tier |
|---|------|--------|--------|----------|
| 1 | Cache `findNearestEnemy` + `getMaxAttackRange` per ActionEvaluator | 1 hr | 20–50× reduction in faction scans per squad turn | Quick win |
| 2 | Gate AI debug `fmt.Printf`s behind `config.DEBUG_MODE` | 30 min | Production hygiene, convention compliance | Quick win |
| 3 | Drop dead `config` parameter from 3 public `Calculate*Power` functions | 15 min | API surface cleanup, prevents future double-weight bug | Quick win |
| 4 | Delete JSON-fallback `switch` in `roles.go` (3 functions) | 1 hr | -40 LOC, removes silent-divergence risk | Quick win |
| 5 | Add `combatstate.GetEnemySquadsForFaction`, replace open-coded loops | 30 min | -25 LOC across `ai/` and `behavior/` | Quick win |
| 6 | Remove stale "moved elsewhere" comment + unused `turnManager` field | 10 min | Hygiene | Quick win |
| 7 | Flatten or commit single PowerConfig profile system | 1 hr | -20 LOC, removes magic string `"Balanced"` | Quick win |
| 8 | Fix entity-vs-template heal asymmetry; have template path call `CalculateDefensivePower` | 2 hr | Correctness — eliminates silent power-score drift | Medium-term |
| 9 | Replace `WaitAction` state-mutation hack with explicit terminate signal | 30 min | Robustness — prevents accidental infinite loops on future actions | Medium-term |
| 10 | Replace `*AIController` field on `AttackAction` with `AttackQueueSink` interface | 1.5 hr | Cleaner separation, smaller test surface | Medium-term |
| 11 | Cache `ActionContext` outside inner loop; only refresh `CurrentPos` after moves | 30 min | Minor perf, signal of intentional design | Medium-term |
| 12 | Add `mind/ai/` tests for score thresholds and attack-beats-movement invariant | 3 hr | Regression safety for tuning changes | Medium-term |
| 13 | Resolve or delete AI spell casting TODO | varies | Either a real feature or honest absence | Medium-term |

**Recommended first sprint (items 1–7):** ~5 hours, removes ~85 LOC, fixes one real perf issue (item 1) and one production-hygiene issue (item 2). All items are low-risk: items 3–6 are pure deletions, item 7 is a small refactor inside one package, and items 1–2 don't change observable behavior.

**Recommended second sprint (items 8–12):** ~7.5 hours, fixes the heal-asymmetry correctness bug, hardens the action loop, and adds regression coverage.

---

## 3. Prevention

- **Unused-parameter lint.** `staticcheck` (or `go vet -shadow`) catches the kind of dead-parameter pattern at the heart of item 3. Adding a CI gate would have flagged `CalculateOffensivePower(attr, config)` the day `config` stopped being used.
- **Config-fallback convention.** Adopt the rule from the behavior tech-debt doc: AI/balance numbers live in JSON, not in Go fallback constants. Validation at startup enforces presence; missing values panic rather than silently using stale defaults. Documented in CLAUDE.md.
- **Cross-path power-score parity test.** A unit test that constructs a heal unit, computes both `EstimateUnitPowerFromTemplate(template)` and `calculateUnitPower(spawnedEntity, ...)`, and asserts equality (or documents the intentional delta). Catches future drift between encounter-gen and AI-threat code paths.
- **Debug-log gate as a project-wide convention.** Add a `make lint` check (or pre-commit hook) that flags unguarded `fmt.Printf("[AI]")`-style logging in non-test files within `mind/`.

---

## 4. Cross-Package Observations

Reading `BEHAVIOR_TECH_DEBT.md` together with this document reveals a consistent shape across the entire AI subsystem:

1. **JSON-fallback duplication is endemic** — flagged once per package (behavior, evaluation). Both should be fixed together; the convention should be documented in CLAUDE.md so it doesn't recur.
2. **Faction-iteration logic is repeated** in `mind/behavior/threat_layers.go`, `mind/ai/action_evaluator.go`, and inside multiple threat-layer constructors. A single `combatstate` helper (`GetEnemySquadsForFaction`) would consolidate all three.
3. **Per-turn computation is repeated per-tile / per-candidate** in both packages: `mind/behavior/threat_positional.go` walks the full grid for sparse data (HIGH item 1 in companion doc); `mind/ai/action_evaluator.go` walks all factions per tile for position-invariant data (HIGH item 1 in this doc). Different root causes but same anti-pattern — "compute once per turn, read N times."
4. **Test coverage is thin where the magic numbers live.** `mind/behavior/` has 193 LOC of tests covering ~1,379 LOC of source. `mind/ai/` has zero. `mind/evaluation/` has 36 LOC covering 521 LOC. Tuning changes happen frequently in this subsystem (per the existence of `aiconfig.json` / `powerconfig.json` / `difficultyconfig.json`) — silent regressions on score-ordering invariants are the most likely class of bug.

The combined first-sprint scope across both documents (~12 hours) delivers measurable performance wins, removes ~155 LOC, and aligns three packages with the data-driven design philosophy already established in the rest of the codebase.
