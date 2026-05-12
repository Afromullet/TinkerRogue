# Technical Debt Analysis — `mind/behavior/`

**Scope:** ~1,379 LOC across 11 files implementing the AI threat-layer evaluation pipeline.
**Date:** 2026-05-12
**External callers:** `mind/ai/ai_controller.go`, `mind/ai/action_evaluator.go`, `tactical/combat/combatservices/ai_interfaces.go`, `gui/guicombat/combatvisualization/threat.go`.

Companion to `BEHAVIOR_THREAT_LAYERS.md`. That document describes how the package works; this one catalogues what needs fixing.

---

## 1. Debt Inventory

### 🔴 HIGH — Performance: Triple full-grid sweep on every AI turn

**Location:** `mind/behavior/threat_positional.go:188–234` (`computeEngagementPressure`, `computeRetreatQuality`) + `:167–184` (`computeIsolationRisk`).

Every call to `PositionalRiskLayer.Compute()` walks the entire `width × height` dungeon grid three times via `IterateMapGrid` (`mind/behavior/threat_gridutils.go:10`). On a 100×60 dungeon that is **18,000 callback invocations per turn**, each allocating one `LogicalPosition`. In `computeRetreatQuality`, every tile performs **8 map lookups × 2 maps = 16 lookups per tile**. For most of the map both threat maps are zero — pure wasted work.

`computeFlankingRisk` is sparse (only paints around enemies), but `engagementPressure` and `retreatQuality` densely populate maps even where `meleeThreat == 0 && rangedThreat == 0`, bloating the maps and slowing future lookups in `GetTotalRiskAt`.

**Impact:**
- 3× full-grid scans + 1 painted scan per evaluator update.
- `CompositeThreatEvaluator.Update()` is called per AI turn AND for visualization (`mind/behavior/threat_composite.go:48–57` says "and at turn-end for visualization").
- Empty positions populate the result map, so `clear()` next turn still costs O(populated entries).

**Fix (medium effort, ~4–6 hr):**
- `engagementPressure`/`retreatQuality` should iterate only positions present in `combatLayer.meleeThreatByPos ∪ rangedPressureByPos` (typically <10% of grid).
- `isolationRisk` should iterate only positions within `isolationMaxDistance` of an ally (cheaper than full sweep when allies are clustered).
- `retreatQuality`: skip writes when `retreatScore == checkedDirs` (saturated default value is redundant; consumer reads zero from missing key the same as "no retreat data").

### 🟠 MEDIUM — Duplication: Role-weight fallback table mirrors JSON

**Location:** `mind/behavior/threat_constants.go:116–145`.

The `switch role` fallback in `GetRoleBehaviorWeights` hard-codes role weights that should mirror `aiconfig.json` — but `aiconfig.json` is already the source of truth and is loaded at startup. The fallback only triggers if the template lookup fails (lines 105–114), which in practice means the config file was missing or malformed. This is **defensive code for a scenario that should panic at startup**, not silently use stale constants that may diverge from the JSON over time.

The architecture note in CLAUDE.md (`roleBehaviors` vs `roleMultipliers`) confirms `roleBehaviors` is the source of truth — the in-code fallback table is a redundant copy that will rot.

**Fix (low effort, ~1 hr):** Delete the role-specific fallback switch. Fail-fast if config missing, or use one neutral fallback `RoleThreatWeights{0.5, rangedWeight, 0.5, positionalWeight}`. Drops ~30 lines.

### 🟠 MEDIUM — Dead/under-utilized API surface

**Locations:** `mind/behavior/threat_constants.go`, `mind/behavior/threat_support.go`, `mind/behavior/dangerlevel.go`.

Grepping the codebase shows these public methods/functions have no external callers (or only test callers):
- `FactionThreatLevelManager.UpdateFaction(factionID)` — `mind/behavior/dangerlevel.go:35` — only `UpdateAllFactions` is used.
- `PaintThreatToMap(..., trackPositions=true)` — `mind/behavior/threat_painting.go:24` — the `trackPositions` parameter is never set to `true` by any caller (every call site passes `false`). The whole `paintedPositions` return slice is dead.
- `SupportValueLayer.allyProximity` map — computed every turn at `mind/behavior/threat_support.go:78–85` (a 2N² nested loop per squad), used only at `mind/ai/action_evaluator.go:200` with a flat `× 3.0` weight. Real use, but the cost is hidden — every squad in the faction paints a `(2r+1)²` square map every turn even when no AI is querying support proximity.

**Fix (low effort, ~30 min):**
- Delete `UpdateFaction` (or document why it's part of the public API if cron-style updates are planned).
- Delete the `trackPositions` parameter from `PaintThreatToMap`. Callers don't care, the slice is wasted allocation.

### 🟠 MEDIUM — Magic numbers and behavior split across constants vs config

**Location:** `mind/behavior/threat_constants.go:9–18`, `:57`, `:71`, `:86`, `:157`.

The package mixes three configuration tiers without a clear principle:
1. **Hardcoded constants in Go**: `isolationMaxDistance = 8`, `engagementPressureMax = 200`, `defaultSharedRangedWeight = 0.5`, default `flanking range = 3`, default `isolation threshold = 3`, default `retreatSafeThreatThreshold = 10`, default `healRadius = 3`.
2. **JSON-driven values** via `AIConfigTemplate.ThreatCalculation.*` and `AIConfigTemplate.SupportLayer.*`.
3. **Difficulty offsets** layered on top via `templates.GlobalDifficulty.AI().*`.

`isolationMaxDistance` and `engagementPressureMax` (a "cosmetic" normalizer per the comment) are the only two values **not** tunable via JSON, yet they materially shape AI behavior:
- `engagementPressureMax = 200` decides where threat saturates to 1.0. If squads ever produce >200 combined threat, all positions become equally bad (clamped at 1.0) and the layer becomes useless.
- `isolationMaxDistance = 8` is a hard cap on the isolation gradient.

**Fix (low effort, ~1 hr):** Move both to `aiconfig.json` under `ThreatCalculation` for consistency with the rest of the package. Document the saturation behavior of `engagementPressureMax` so designers know what raising/lowering it does.

### 🟡 LOW — `getDirection` reinvents `math.Atan2` for 8-way directions

**Location:** `mind/behavior/threat_positional.go:127–145`.

A 19-line if-else chain that classifies `(dx, dy)` into 8 directions. Hot path: called inside the `(2r+1)²` loop in `computeFlankingRisk`.

**Fix (~15 min):** Replace with a table lookup `dirTable[sign(dx)+1][sign(dy)+1]` (3×3 array), or inline as a 2-bit encoding. The current code is correct but ugly and verbose; would shrink to ~5 lines.

### 🟡 LOW — Outdated comments

**Location:** `mind/behavior/threat_combat.go:14–17`, `mind/behavior/threat_composite.go:13`.

Comments reference the historic split that was already collapsed:
> "This is a unified layer that replaces the separate MeleeThreatLayer and RangedThreatLayer."
> "// Note: RoleThreatWeights struct and GetRoleBehaviorWeights are defined in threat_constants.go"

These are migration breadcrumbs that have outlived their purpose. Per CLAUDE.md style guidance: comments should not reference history or sibling locations — both belong in commit messages, not source.

### 🟡 LOW — `GetTotalRiskAt` averages incommensurable units

**Location:** `mind/behavior/threat_positional.go:259–265`.

```go
return (flanking + isolation + pressure + retreatPenalty) * 0.25
```

`flanking` ∈ {0, 0.5, 1.0} (3 buckets), `isolation` ∈ [0,1] (linear gradient), `pressure` ∈ [0,1] (saturated at `engagementPressureMax`), `retreatPenalty` ∈ [0,1] (continuous, density of low-threat neighbors). Averaging four values with **different shapes and noise characteristics** with equal weight is a code smell — the more granular continuous values (isolation, pressure, retreatQuality) get drowned out by binary flanking buckets.

This method is exposed but **not used anywhere outside the package** (no callers in grep). So it is both: low-quality math AND dead. Either delete or restructure into a weighted combination tunable from `aiconfig.json`.

### 🟡 LOW — Test coverage gaps for hot code

**Location:** `mind/behavior/threat_layers_test.go` covers: role weights, enemy faction detection, optimal-position-with-empty-or-singleton-list, and "does Update panic". Missing:
- No tests for `PaintThreatToMap` falloff math (the foundation of every layer).
- No tests for `computeFlankingRisk` direction-counting.
- No tests for `computeIsolationRisk` linear gradient behavior.
- No tests for `engagementPressureMax` saturation.
- No tests for `GetRetreatQuality` boundary conditions (edges of map).

**Risk:** silent regressions when refactoring the hot path identified in HIGH above.

---

## 2. Prioritized Remediation Plan

| # | Item | Effort | Impact | ROI tier |
|---|------|--------|--------|----------|
| 1 | Sparse iteration for `engagementPressure` / `retreatQuality` (skip empty positions) | 3 hr | 5–10× faster `PositionalRiskLayer.Compute()` | Quick win |
| 2 | Delete role-weight fallback switch in `GetRoleBehaviorWeights` | 1 hr | -30 LOC, removes silent-divergence risk | Quick win |
| 3 | Delete `trackPositions` parameter from `PaintThreatToMap` (always false in callers) | 30 min | -10 LOC, removes dead slice alloc | Quick win |
| 4 | Move `isolationMaxDistance` + `engagementPressureMax` to aiconfig.json | 1 hr | Designer tunability, consistency | Quick win |
| 5 | Replace `getDirection` 19-line if-else with 3×3 lookup table | 15 min | Readability, perf in flanking loop | Quick win |
| 6 | Strip outdated "unified replaces split" / "defined elsewhere" comments | 10 min | Hygiene | Quick win |
| 7 | Delete unused `UpdateFaction` and unused `GetTotalRiskAt` (or document) | 30 min | -20 LOC, smaller API surface | Quick win |
| 8 | Add tests for `PaintThreatToMap` falloff, `computeIsolationRisk` gradient, `engagementPressureMax` saturation, edge-of-map retreat | 4 hr | Regression safety for #1 | Medium-term |
| 9 | Replace `GetTotalRiskAt` flat-average with config-driven weighted combination (or delete) | 1 hr | Tunable AI judgment, no silent muting | Medium-term |
| 10 | Profile `Compute()` end-to-end on representative encounter; verify #1 actually helps | 1 hr | Validate ROI assumption | Medium-term |

**Recommended first sprint (items 1–7):** ~7 hours, removes ~70 LOC, delivers ~5–10× speedup on the most expensive recurring AI computation. No external API changes — `behavior` is consumed via two interfaces in `combatservices` (`ThreatProvider`, `ThreatLayerEvaluator`) which are untouched by any of these.

---

## 3. Prevention

- **Pre-commit lint:** add a `go vet`-style check (or `staticcheck`) for unused exported methods — would have caught `UpdateFaction` and `GetTotalRiskAt`.
- **Benchmark gate:** introduce `BenchmarkCompositeThreatEvaluator_Update` to lock in the post-fix-1 performance and catch regressions when future layers are added.
- **Config-vs-constant convention:** any AI-behavior-shaping number lives in `aiconfig.json`, not in Go constants. Constants reserved for math/algorithmic invariants (e.g., 8 cardinal directions, π).

---

## 4. Why this matters

`mind/behavior/` is on the **hot path of every AI turn** and is also re-run for end-of-turn visualization. Items 1, 3, and 7 are pure removals (negative LOC) with measurable performance wins. Items 2 and 4 align the package with the existing data-driven design philosophy already established in the rest of the codebase. None require API breaks.
