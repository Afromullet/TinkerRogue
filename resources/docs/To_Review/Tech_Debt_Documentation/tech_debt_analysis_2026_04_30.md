# TinkerRogue — Technical Debt Analysis

**Date:** 2026-04-30
**Files scanned:** 401 Go files, ~134k lines

---

## Executive Summary

The codebase is architecturally healthy — clean ECS discipline, no circular imports, only 6 TODO comments, well-named symbols. The dominant debt is **test coverage** (56 packages with zero tests, including all of `combatmath` and all GUI packages) followed by **size/responsibility concentration** in the `guicombat` package. Neither is critical today, but the test gap creates invisible risk as combat math and overworld logic grow.

**Debt score: Medium.** No emergency, but the test gap is growing linearly with feature additions.

---

## 1. Debt Inventory

### 1A. Testing Debt — CRITICAL PRIORITY

**56 packages have zero test files.** This is the single largest debt item.

| Subsystem | Untested Packages | Risk |
|---|---|---|
| `gui/` | 16 packages (entire subsystem) | Low — UI logic is hard to unit test |
| `tactical/combat/` | `combatmath`, `combatstate`, `battlelog`, `combattypes` | **HIGH** — core math, no coverage |
| `tactical/powers/` | `powercore`, `spells` | **HIGH** — pipeline + spell casting |
| `tactical/squads/` | `squadcommands`, `squadservices`, `unitdefs` | Medium |
| `campaign/overworld/` | All 8 sub-packages | Medium |
| `mind/` | `ai`, `encounter`, `spawning` | Medium |
| `world/` | `worldgen`, `garrisongen`, `worldmapcore` | Medium |
| `setup/gamesetup/` | All 10 files | Low |

**The highest-risk untested area is `tactical/combat/combatmath/`** — it contains the core damage/healing/targeting math (`CalculateDamage`, `SelectTargetUnits`, `GetCoverProvidersFor`, `CalculateSquadStatus`) and has zero tests despite being the numerical foundation that the heavily-tested `combatexecution_test.go` (1,641 lines) depends on indirectly.

Similarly, `tactical/powers/powercore/` (the shared power pipeline that everything flows through) and `tactical/powers/spells/` have no direct tests.

### 1B. Size / Responsibility Debt — MEDIUM PRIORITY

**`gui/guicombat/` package** is doing too much in too few files:

| File | Lines | Exported Functions | Concerns |
|---|---|---|---|
| `combat_input_handler.go` | 625 | 21 | Input for spells, artifacts, movement, selection, debug kill mode |
| `combat_animation_mode.go` | 587 | 23 | Animation state + 2 types |
| `combatmode.go` | 549 | 20 | Mode orchestration |

Three files, 64 exported functions. The `CombatInputHandler` specifically conflates spell input, artifact input, movement input, double-click tracking, and debug mode — concepts that evolve at different rates.

Other large files that are borderline but not currently concerning:
- `world/worldgen/gen_cavern.go` (713 lines) — map generation algorithm; size is expected
- `tactical/combat/combatservices/combat_service.go` (512 lines) — service coordinator; pipeline wiring is verbose but intentional

### 1C. Code Pattern Debt — LOW PRIORITY

**Repeated nil-guard closures** in `combat_service.go` (lines 104–139):

```go
// This exact pattern repeats 6 times for perkDispatcher
cs.powerPipeline.OnXxx(func(...) {
    if cs.perkDispatcher != nil {
        cs.perkDispatcher.DispatchXxx(...)
    }
})
```

Each pipeline subscription wraps in its own anonymous func purely for the nil check. Readable as-is, but a small helper would reduce noise.

**Validation spread across 3 locations:** `templates/validation.go`, `campaign/overworld/node/validation.go`, `mind/encounter/validators.go`. These serve different scopes (template loading, node placement, encounter entities) so the split is justified — not actual duplication.

### 1D. TODO Debt — NEGLIGIBLE

Only 6 TODOs in the entire codebase:

| File | Comment | Action |
|---|---|---|
| `mind/ai/ai_controller.go:84` | `AI spell casting - enemy commanders don't cast spells yet` | Feature backlog |
| `tactical/squads/squadservices/unit_purchase_service.go:59` | `Add cost field to UnitTemplate or JSON data` | Small data schema change |
| `world/worldmapcore/dungeongen.go:233` | `Change this to check for WALL, not blocked` | **Bug-risk: could cause wrong pathfinding** |
| `tactical/combat/combatcore/turnmanager.go:62` | `Abilities not yet implemented` | Feature backlog |
| `campaign/overworld/faction/system.go:137` | `Consider using interface for intent` | Design consideration |
| `core/coords/cordmanager.go:45` | Empty TODO | Delete it |

**One item deserves immediate attention:** `dungeongen.go:233` — checking `blocked` instead of `WALL` is a semantic bug risk in map generation.

### 1E. Architecture Debt — LOW PRIORITY

No circular imports. Dependency layering is mostly clean with one pattern to watch:

- `gui/guiartifacts/artifact_deps.go` and `gui/guicombat/combatdeps.go` import from `mind/encounter` — GUI reaching into the AI/encounter layer. This is likely for encounter context at combat start. It's not wrong, but if it grows it could create upward coupling in the dependency graph.

---

## 2. Prioritized Remediation Roadmap

### Quick Wins (1–3 hours each)

**1. Add tests to `combatmath/` — highest ROI in the codebase**

These functions have clear inputs and outputs and are currently tested only indirectly:

```
combatcalculation.go: CalculateDamage, CalculateHealing, CalculateSquadStatus
combattargeting.go:   SelectTargetUnits, CanUnitAttack, SelectHealTargets
combatcover.go:       GetCoverProvidersFor, CalculateCoverBreakdown
```

Direct unit tests here would catch regressions in balance-critical code without needing a full combat simulation.

**2. Delete the empty TODO in `core/coords/cordmanager.go:45`**

One line. Zero risk.

**3. Investigate `dungeongen.go:233` — the WALL vs blocked check**

Read the context, determine if this is an active bug or a deferred design note, and either fix it or convert to a proper comment explaining why `blocked` is currently correct.

**4. Add tests to `powercore/` — pipeline event ordering**

The `PowerPipeline` controls artifacts → perks → GUI execution order. A single test that verifies subscriber call order would give high confidence in a brittle area.

### Medium Term (1–2 days each)

**5. Add tests to `combatstate/` — `CombatQueryCache`**

The cache is used everywhere in combat; a test that verifies it invalidates correctly on state changes would prevent subtle stale-data bugs.

**6. Add tests to `spells/` spell casting**

Spells flow through `powercore` which has no tests either. Cover the happy path and one edge case (e.g., casting with no valid targets).

**7. Split `CombatInputHandler` by input domain**

The 625-line handler has 4 distinct concerns. A natural split:
- `SpellInputHandler` — spell selection and targeting
- `ArtifactInputHandler` — artifact activation
- `MovementInputHandler` — squad movement commands
- `CombatInputHandler` — orchestrator that delegates to the above

This doesn't need to happen now, but every new input feature added to the current file makes the split more expensive later.

### Long Term (ongoing discipline)

**8. Establish a test-as-you-go rule for `tactical/` and `campaign/`**

The `gui/` test gap is acceptable — ebitenui widget logic is notoriously hard to unit test. But new logic added to `tactical/`, `campaign/`, and `world/` packages should ship with at least one happy-path test. The currently-untested packages (`combatstate`, `combatmath`, `powercore`, `encounter`, `spawning`) represent ~15 files of logic that only exists in integration tests today.

---

## 3. Debt Metrics Snapshot

```
Total .go files:           401
Files over 300 lines:       45   (11%)
TODO/FIXME comments:         6   (excellent)
Packages with zero tests:   56   (out of ~90 total — 62%)
Circular imports:            0   (clean)
Deprecated patterns:         0   (clean)
```

**Packages with tests that matter most** (already covered — keep them healthy):

| Package | Test Files | Notes |
|---|---|---|
| `tactical/combat/combatcore/` | 3 test files | 1,641 + 643 lines of coverage |
| `tactical/powers/perks/` | 1 test file | 880 lines |
| `tactical/squads/squadcore/` | 3 test files | 767 lines |
| `tactical/powers/artifacts/` | 3 test files | 686 lines |

---

## 4. Prevention Habits

**Before adding logic to an untested package:**
```bash
go test -cover ./tactical/combat/combatmath/...
```
If the output is `no test files`, add one as part of the same change.

**Periodically sweep TODOs:**
```bash
grep -rn "TODO\|FIXME\|HACK\|XXX" --include="*.go" .
```
Current count is 6 — trivial to review. Keep it that way.

**Flag files approaching 400 lines:**
The 300-line threshold in CLAUDE.md is right. The three `guicombat` files (549–625 lines) are already past it. The next feature added to `combat_input_handler.go` is the right time to split it.

---

## 5. Priority Matrix

| Item | Priority | Effort | Risk if Ignored |
|---|---|---|---|
| Tests for `combatmath/` | **High** | 3–4 hrs | Silent balance regressions |
| Tests for `powercore/` | **High** | 2–3 hrs | Pipeline ordering bugs |
| `dungeongen.go:233` TODO investigation | Medium | 30 min | Incorrect map generation |
| Split `CombatInputHandler` | Medium | 4–6 hrs | Increasingly painful to extend |
| Tests for `combatstate/` | Medium | 2 hrs | Cache coherence bugs |
| Tests for `spells/` | Medium | 2–3 hrs | Spell casting regressions |
| Delete empty TODO in `cordmanager.go` | Low | 5 min | Noise |
| Reduce nil-guard duplication in `combat_service.go` | Low | 1 hr | Minor readability |
