# Technical Debt Analysis — `tactical/squads/`

**Date:** 2026-05-15
**Scope:** 6 sub-packages, ~5,815 LOC (3,279 production / 2,536 test ≈ 44% test-to-prod ratio by line count)
**Sub-packages reviewed:** `squadcore`, `squadcommands`, `squadservices`, `roster`, `unitdefs`, `unitprogression`

---

## 1. Debt Inventory

### 1.1 Code Debt — Duplication (HIGH severity)

| # | Location | Description | LOC affected |
|---|----------|-------------|--------------|
| D1 | `squadcore/squadqueries.go` ↔ `squadcore/squadcache.go` | **Twin APIs** for `GetSquadEntity`, `GetUnitIDsInSquad`, `GetLeaderID`, `GetSquadName`. The cached version exists for GUI hot paths, but the canonical/uncached version still uses `World.Query()` even though a package-level `squadMemberView` already exists in `squadmanager.go:14`. Two near-identical loop bodies per query. | ~120 lines duplicated |
| D2 | `squadcore/squadcreation.go:74-127`, `:149-207`, `:294-418` | Three creation paths (`AddUnitToSquad`, `PlaceUnitInSquad`, `CreateSquadFromTemplate`) each repeat: grid-bounds validation, occupancy check, `SquadMemberData` add, `GridPositionData` write, `Renderable.Visible=false`, leader auto-promote. | ~150 lines |
| D3 | `squadcreation.go:81,151,255` + `units.go:21-27` + `templates.go:46-52` | Grid bounds `0..2` / size `1..3` / `+Height>3` checks repeated 5 times. Magic numbers `3` (grid size) and `9` (MaxUnits) hardcoded across 6+ files. | 5 sites |
| D4 | `squadcreation.go:116-119, 192-195, 380-383` | "Hide unit renderable" three-line snippet copy-pasted 3×. | 3 sites |
| D5 | `unitroster.go:120-130` and `:133-151` | `MarkUnitInSquad` / `MarkUnitAvailable` both perform a nested O(n²) walk over the `Units` map and `UnitEntities` slice to find a single ID. The `UnitsInSquads` map is keyed correctly, but the lookup re-traverses every entry. | 2 sites |
| D6 | `unitroster.go:182-198` and `:161-173` | Same iteration pattern ("entries → entity IDs → check `!HasComponent(SquadMemberComponent)`") in `GetAvailableUnitDetails` and `GetUnitEntityForTemplate`. | 2 sites |
| D7 | `add_unit_command.go:71-96` ↔ `remove_unit_command.go:59-83` | Add/Remove are mirror operations but do not share a transactional helper (place + `roster.MarkInSquad` / unassign + `roster.MarkAvailable`). | ~40 lines |
| D8 | `experience.go:73-90` | Six manually unrolled `if rng.Intn(100) < GrowthChance(growthData.X) { attr.X++ }` blocks. Adding a new growable stat requires editing 3 places (`StatGrowthData`, `templates.go` mapping, this loop). Replace with a slice/loop or `[]struct{Grade, *int}`. | 18 lines |

### 1.2 Code Debt — Long Methods / Complexity (MEDIUM)

| # | Location | Issue |
|---|----------|-------|
| C1 | `squadcreation.go:CreateSquadFromTemplate` | **125 lines, cyclomatic ~14**. Mixes squad creation, per-unit dimension defaulting, bounds validation, occupancy tracking via stringly-keyed map (`fmt.Sprintf("%d,%d")`), entity creation, name generation, renderable hiding, position re-binding, and leader assignment. Should decompose to `createSquadEntity`, `placeTemplateUnit`, `validateTemplatePlacement`. |
| C2 | `squadcreation.go:MoveUnitInSquad` | Nested loop with double `GetUnitIDsAtGridPosition` call inside — O(units²) for a single move. Each `GetUnitIDsAtGridPosition` itself runs `World.Query(SquadMemberTag)`. |
| C3 | `unit_purchase_service.go:PurchaseUnit` | Manual 3-step transaction with rollback at each step. Pattern is correct but unencapsulated; a `transactionStep` / defer-rollback helper would prevent future maintainers from forgetting a rollback branch. |

### 1.3 Architecture Debt

| # | Location | Issue |
|---|----------|-------|
| A1 | `squadcache.go` vs `squadqueries.go` | **Dual public APIs for the same operations** create a "which one do I use?" decision at every call site. Doc says "prefer cached when you have a `SquadQueryCache`," but with `squadMemberView` already package-global, the canonical functions could (and should) use Views transparently. The cached struct adds value only for `SquadView`/`LeaderView`. Net effect: 4 redundant methods + cognitive overhead. |
| A2 | `squadcore/squadcreation.go:282-291` | `FormationPreset` / `FormationPosition` types are **defined but never used** anywhere in the package or repo. Dead architecture. |
| A3 | `squadqueries.go:GetSquadEntity` | Comment admits "non-cached version (O(n))" and tells callers to "prefer SquadQueryCache." This is a smell — the cache should be transparent (e.g., a package-level singleton initialized in `init()` like `squadMemberView`). |
| A4 | `squadcommands` package | Every command duplicates ~30 lines of struct boilerplate (constructor + receiver methods). Go doesn't support generics on receivers cleanly, but a `BaseCommand` embedded struct could absorb `manager`, `squadID`, name capture, and standardized `descriptionf(...)`. |
| A5 | `squadqueries.go:282-296` and `squadcomponents.go:GridPositionData` | `coords.LogicalPosition` (world space) and `GridPositionData` (3×3 squad-local) live side by side with no namespacing. `Width`/`Height` on `GridPositionData` collide visually with renderable dimensions. Consider renaming to `CellWidth`/`CellHeight` or wrapping in a `SquadGridCell` value type. |
| A6 | `unit_purchase_service.go:GetUnitCost` | TODO + cost computed from a hash of `UnitType` rune codes. This is design debt: cost should live in JSON / `UnitTemplate`. The hash makes balancing impossible without renaming units. |
| A7 | `unitdefs/enums.go:GetAttackType` | "Backward compatibility" fallback by `attackRange` is documented but the JSON loader appears to always supply `attackType` — verify and remove if so, since this dual codepath obscures the contract. |

### 1.4 Data / Component Debt (MEDIUM)

| # | Location | Issue |
|---|----------|-------|
| DC1 | `squadcomponents.go:AbilitySlotData` | Hard-coded `[4]AbilitySlot` plus parallel `[4]int` cooldowns in `CooldownTrackerData`. Adding/removing slots requires touching both arrays + every loop that iterates `0..3`. Consider `Slots []AbilitySlot` and a single `Cooldowns []int` with paired length invariants. |
| DC2 | `squadcomponents.go:GetAbilityParams` | Hardcoded switch that hides 4 ability balance constants in code rather than data (templates JSON). Same anti-pattern as A6. |
| DC3 | `squadcomponents.go:50` (`SquadData.SquadLevel`) and `LeaderData.Experience` (`squadcomponents.go:182`) | Both fields exist but no system writes to them in the squads package. Dead fields → confusion / garbage data risk. |
| DC4 | `squadcomponents.go:50` (`GarrisonedAtNodeID`) | Squad data carries a campaign-domain field (`campaign/overworld` knowledge) — leaks higher-level concept into core squad component. Should be a separate `GarrisonComponent`. |

### 1.5 Testing Debt (LOW–MEDIUM)

| # | Location | Issue |
|---|----------|-------|
| T1 | `squadcommands/` | **No tests for any command** (8 command files, 0 `_test.go` files). Undo paths are particularly under-tested — easy place for regressions when entity lifecycle changes. |
| T2 | `squadservices/` | **Zero tests** for `UnitPurchaseService` (rollback paths) or `SquadDeploymentService`. |
| T3 | `roster/squadroster.go` | `unitroster_test.go` exists but no tests for `SquadRoster` (Add/Remove/Reorder/`filterSquadsByDeployment`). |
| T4 | `squadcore/squadcreation.go:CreateSquadFromTemplate` | 125-line function with multi-cell + occupancy logic — partially covered by `squads_test.go` but no explicit tests for the multi-cell occupancy guard or the `fmt.Sprintf("%d,%d")` keying. |

### 1.6 Documentation / Naming Debt (LOW)

| # | Location | Issue |
|---|----------|-------|
| N1 | `squadcore/squadabilities.go:11` | Stub doc comment `// EquipAbilityToLeader -` is a leftover. |
| N2 | `unitprogression/components.go:31` | Doc says `XPToNextLevel int // XP required to level up (fixed 100)` — but `awardExperience` reads from the field. If it really is fixed 100, make it a constant; otherwise the comment is misleading. |
| N3 | `squadqueries.go:96-97`, `:114-115`, `:344-345` | Three near-identical "NOTE: This is the non-cached version" doc blocks signaling unresolved design tension — see A1. |
| N4 | `squadcomponents.go` | Package doc says "Package squads" but the package is `squadcore`. Stale after rename. |

---

## 2. Impact Assessment

| Item | Velocity Impact | Risk | Priority |
|------|-----------------|------|----------|
| D1/A1/A3 (twin query APIs) | Every new query feature must be added in 2 places; mistakes are silent (results match). ~30 min/feature × ~6 features/quarter = **2 h/qtr drag** | LOW correctness, MEDIUM cognitive | **Quick win** |
| D2/D4 (creation duplication) | Bug fixes (e.g., the renderable-hiding logic) need editing in 3 spots. | MEDIUM (entity lifecycle bugs) | **High** |
| D3 (magic numbers `3`/`9`) | Any "expand grid to 4×4" experiment requires touching 6+ files. | LOW (no current need), but **architecture-blocking** | Medium |
| D7 (add/remove asymmetry) | Roster + squad state can desync if one half throws. | HIGH — `RemoveUnit` returns bool, callers don't always check | **High** |
| C1 (`CreateSquadFromTemplate`) | 125-line function is the #1 onboarding hurdle in this package. | MEDIUM | High |
| DC1 (fixed `[4]` ability arrays) | Trivial today, lock-in cost when adding a 5th slot. | LOW now | Defer |
| DC4 (`GarrisonedAtNodeID` leak) | Cross-package coupling between `squadcore` and `campaign/overworld`. | MEDIUM (boundary violation) | Medium |
| T1/T2 (missing command/service tests) | Each command is ~100 lines of validation/execute/undo; no safety net for refactors. | HIGH (undo correctness) | **High** |
| A6 (unit cost hash) | Game balance literally cannot be tuned today without renaming units. | HIGH for a tactical RPG | **High** |
| A2/DC3 (dead types/fields) | No active cost; misleads readers. | LOW | Cleanup |

---

## 3. Quick Wins (sprint 1, < 8 h each)

1. **Delete `FormationPreset` / `FormationPosition`** (`squadcreation.go:282-291`) — unused, ~10 lines. *15 min.*
2. **Delete `SquadLevel` and `LeaderData.Experience`** if confirmed unused, or wire `unitprogression.ExperienceData` into Leader. *30 min for confirmation grep + delete.*
3. **Fix stale package doc comment** (`squadcomponents.go:1-7` references "Package squads"; package is `squadcore`). *5 min.*
4. **Extract `hideUnitRenderable(entity)` helper** — collapses D4. *15 min.*
5. **Extract `validateGridBounds(row, col, w, h int)` helper** in `squadcore` — collapses D3 (5 sites). *30 min.*
6. **Move grid limits to constants** (`SquadGridSize = 3`, `SquadMaxUnits = 9`) in `squadcomponents.go`. *20 min.*
7. **Add `RemoveUnit` callers' bool-return checks** (audit roster usage). *1 h.*
8. **Replace `experience.go:73-90` unrolled if-chain** with a tabular loop driven by `[]struct{Grade GrowthGrade; Stat *int}`. *45 min.*

**Estimated quick-win savings:** ~120 lines deleted, future-stat-add cost drops from "edit 3 files" to "edit 1 struct."

---

## 4. Medium-Term Roadmap (1–3 sprints)

### Sprint 1–2 — Consolidate query APIs (resolves A1/A3/D1)

- Make `squadMemberView`, `squadView`, `leaderView` package-level vars, all initialized in `init()` (already exists for `squadMemberView`).
- Rewrite the canonical functions in `squadqueries.go` to use the views directly; **delete `SquadQueryCache`** (or keep only as a typed handle).
- Remove "NOTE: non-cached version" doc blocks.
- **Effort:** ~6 h. **Win:** −120 LOC, no caller changes (public API identical).

### Sprint 2 — Refactor `CreateSquadFromTemplate` (C1, D2)

- Split into:
  - `createSquadEntity(name, formation, pos)`
  - `placeTemplateUnit(squadID, template, occupied)` (returns error, mutates `occupied`)
  - `markCellsOccupied(occupied, anchor, w, h)`
- Replace `map[string]bool` keyed by `fmt.Sprintf("%d,%d")` with `[3][3]bool` (zero-alloc, type-safe).
- Add tests for multi-cell occupancy collisions.
- **Effort:** ~8 h. **Win:** main function drops to ~30 lines; eliminates string formatting in hot path.

### Sprint 2–3 — Roster transactional helpers (D7)

- Add `roster.AssignUnitToSquad(unitID, squadID, manager)` and `roster.UnassignFromSquad(unitID)` that combine `MarkUnitInSquad` + `PlaceUnitInSquad` (and inverse) with rollback.
- Refactor `AddUnitCommand`, `RemoveUnitCommand`, `ChangeLeaderCommand` to use these helpers.
- Add tests for rollback paths.
- **Effort:** ~8 h. **Win:** eliminates state-desync class of bugs.

### Sprint 3 — Move balance data to JSON (A6, DC2)

- Add `cost`, `abilities[]` to monster JSON / a new abilities JSON file.
- Replace `GetUnitCost` hash and `GetAbilityParams` switch with table reads.
- **Effort:** ~10 h (incl. content-side updates). **Win:** unblocks balancing iteration.

---

## 5. Long-Term (quarter)

### Boundary cleanup (DC4)

- Move `GarrisonedAtNodeID` from `SquadData` to a new `GarrisonComponent` owned by `campaign/overworld/garrison`.
- Adds `HasComponent(GarrisonComponent)` checks in 2-3 GUI/AI sites but removes a domain leak.
- **Effort:** ~6 h.

### Test coverage push (T1–T3)

- `squadcommands/` end-to-end tests: validate → execute → undo → re-execute for each command.
- `squadservices/` rollback-path tests (force ECS errors, assert state restored).
- `SquadRoster` reorder edge cases (already covered by `ReorderSquadsCommand` validation but no roster-level tests).
- **Target:** +15% line coverage on `squadcommands` and `squadservices`. **Effort:** ~16 h.

---

## 6. Prevention

- **Lint rule (custom or convention doc):** "If you find yourself writing `World.Query(SquadXXX)`, use the `squadXXXView` instead." Add to CLAUDE.md once A1 is resolved.
- **Constants over magic numbers:** Adopt `SquadGridSize`, `SquadMaxUnits` immediately so future grid resizing is a one-line change.
- **No new "balance constants in code":** All ability/cost/cooldown numbers go in JSON.
- **PR checklist addition:** "Does this command have tests for execute + undo?" — block command merges without them.

---

## 7. ROI Summary

| Investment tier | Effort | Annual maintenance saved | Risk reduction |
|---|---|---|---|
| Quick wins (1 sprint, ~5 h) | 5 h | ~10 h/yr | Cleanup; eliminates dead code paths |
| Query consolidation (A1) | 6 h | ~8 h/yr | Removes a documented "preferred API" footgun |
| Creation refactor + roster txns | 16 h | ~20 h/yr | Eliminates state-desync class of bugs (T1 not catching them today) |
| Cost / ability data move | 10 h | Unblocks design iteration | Major design velocity |
| Test coverage push | 16 h | Hard to quantify; major regression-defense ROI on undo paths | High |

**Recommended next step:** Knock out quick wins #1–6 in one ~3 h batch, then plan Sprint 2 around D7 + C1 (the two highest-risk items).

---

## 8. Top 3 Takeaways

1. **Twin query API** (`squadqueries.go` vs `squadcache.go`) — ~120 lines of duplication maintained side-by-side; consolidate by lifting all three Views to package-level (mirroring the existing `squadMemberView`).
2. **Three squad-creation paths** in `squadcreation.go` repeat the same hide-renderable / bounds-check / leader-promote sequence — extract helpers; fold magic `3`/`9` into named constants.
3. **`squadcommands/` and `squadservices/` have zero tests** despite carrying the undo + transaction-rollback logic — this is the biggest correctness risk.
