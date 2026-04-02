# Hook & Callback System — Refactoring Analysis

**Date:** 2026-04-01
**Scope:** `tactical/perks/`, `tactical/powers/artifacts/`, `tactical/combat/combatcore/`, `tactical/combat/combatservices/`

---

## Current Architecture Summary

The codebase has two parallel hook/callback pipelines for combat-reactive systems:

| Dimension | Perks | Artifacts |
|-----------|-------|-----------|
| Pattern | Hook table (struct of 9 optional function fields) | Interface + registry (6 methods, BaseBehavior no-ops) |
| Registry | `map[string]*PerkHooks` in `hooks.go` | `map[string]ArtifactBehavior` in `artifactbehavior.go` |
| Context | `HookContext` (parameter object: entity IDs + round state) | `BehaviorContext` (service locator: cache + charge tracker) |
| Dispatch | Squad-scoped: iterates only equipped perks on relevant squad | Broadcast: iterates ALL registered behaviors on every event |
| Per-combat state | `PerkRoundState` ECS component (17+ fields, manual reset) | `ArtifactChargeTracker` (generic string-keyed charges) |
| Import bridge | `PerkCallbacks` in combatcore (7 function type aliases) | None needed (no circular import) |
| Hook into combat | 9 inline hooks (damage calc, targeting, death override) + 4 CombatService events | 3 CombatService events + player activation |

Both systems register in `init()`, both are dispatched from `CombatService` callbacks, both use context structs.

---

## What's Working Well

- **Perk hook-slot pattern** is correct for passive combat modifiers — squad-scoped dispatch, nil-skip for unused hooks, clean separation of attacker/defender sides
- **Artifact interface + BaseBehavior** is clean for optional method implementation
- **CombatService multi-subscriber events** (`[]func` slices) allow perks, artifacts, and GUI to coexist
- **Bridge layer** (`PerkCallbacks`) is the minimal solution for Go's circular import constraint — compiler-enforced signature sync
- **Adding a new perk or artifact already requires only 3 files** (JSON + behavior code + registration)

---

## Problems Identified



### P2: Artifact Broadcast Dispatch (All 3 agents flagged)

`setupBehaviorDispatch` calls `AllBehaviors()` on every combat event and iterates ALL registered behaviors. Each behavior must self-check ownership (`HasArtifactBehavior(squadID, key)`). The perk system does this correctly — runners iterate only the equipped perks on the relevant squad. With 20+ artifacts, missed self-guards become bugs.

### P3: RunXxx Boilerplate (2 of 3 agents flagged)

9 runner functions in `queries.go` (~200 lines) follow near-identical patterns. The attacker/defender pairs (`RunAttackerDamageModHooks`/`RunDefenderDamageModHooks`, `RunAttackerPostDamageHooks`/`RunDefenderPostDamageHooks`) differ only in which squad is iterated and which hook field is read.

### P4: behaviors.go Scale (2 of 3 agents flagged)

All 24 perk implementations + the `init()` block live in a single 600-line file. At 50+ perks, this becomes 1500+ lines — painful to navigate, merge-conflict-prone.

---

## What Should NOT Be Unified

All three agents agreed on these:

- **HookContext vs BehaviorContext** — Different design intents. HookContext is a per-invocation parameter bag. BehaviorContext is a session-level service locator. Merging them adds unused fields to both.
- **PerkHooks struct vs ArtifactBehavior interface** — Perks modify damage inline during calculation. Artifacts react to events and include player-activated abilities. A combined interface would have 15+ methods where each system ignores half.
- **Registry patterns** — Each is ~10-25 lines with unique validation needs. Generic `Registry[T]` would be a leaky abstraction.
- **Dispatch wiring** (`setupPerkDispatch` vs `setupBehaviorDispatch`) — Structurally parallel but intentionally different iteration strategies. Unifying saves ~30 lines but obscures the different per-squad vs broadcast patterns.

---

## Refactoring Options (Ranked by Impact/Effort)

### Tier 1: High Impact, Low-Medium Effort

#### Option 1: Per-Perk State Isolation
**Problem:** PerkRoundState grows a new field for every stateful perk.
**Solution:** Add `CustomState map[string]interface{}` to PerkRoundState. New perks store their state under their perk ID key. Gradually migrate existing single-perk fields (CounterpunchReady, DeadshotReady, etc.). Keep genuinely shared fields (MovedThisTurn, AttackedThisTurn) on the struct.
**Trade-off:** Type assertions on read. Mitigated by typed helper functions per perk.
**Files:** `components.go`, affected behavior functions in `behaviors.go`
**Effort:** Medium (mechanical migration)

#### Option 2: Squad-Aware Artifact Dispatch
**Problem:** Artifacts broadcast to all behaviors; each must self-guard ownership.
**Solution:** Iterate only artifacts equipped on the relevant squads, matching the perk dispatch pattern. Use existing `HasArtifactBehavior`/`GetEquipmentData` queries.
**Trade-off:** Global artifacts (if any future artifact affects all squads) would need a separate path.
**Files:** `combat_service.go` (setupBehaviorDispatch), possibly `artifactbehavior.go` (add query helper)
**Effort:** Low-medium

#### Option 3: Collapse RunXxx Attacker/Defender Pairs
**Problem:** 4 functions that differ only in squad selection and hook field.
**Solution:** Extract `runDamageModHooks(ownerSquadID, hookSelector)` and `runPostDamageHooks(ownerSquadID, hookSelector)` helpers. 4 functions become 2 helpers + 4 one-liner wrappers.
**Trade-off:** Slightly more indirection.
**Files:** `queries.go`
**Effort:** Low (~30 minutes)

### Tier 2: Medium Impact, Low Effort

#### Option 4: Split behaviors.go by Tier
**Problem:** Single 600-line file will grow to 1500+.
**Solution:** `behaviors_tier1.go` and `behaviors_tier2.go`, each with their own `init()`.
**Trade-off:** Registration scattered across files (mitigated by `validateHookCoverage`).
**Files:** `behaviors.go` split into 2 files
**Effort:** Very low

#### Option 5: Extract setupBehaviorDispatch to Own File
**Problem:** Artifact dispatch wiring is buried in `combat_service.go`.
**Solution:** Create `behavior_dispatch.go` mirroring `perk_dispatch.go` structure.
**Trade-off:** None.
**Files:** `combat_service.go` -> `behavior_dispatch.go`
**Effort:** Very low (~10 minutes)

### Tier 3: Medium Impact, Medium Effort

#### Option 6: Explicit Hook Priority/Ordering
**Problem:** Multiple perks on the same hook fire in equipment order. Undocumented, potentially surprising.
**Solution:** Add `Priority int` field to PerkHooks. Sort hooks by priority before dispatch.
**Trade-off:** One extra sort per dispatch. Negligible cost.
**Files:** `hooks.go`, runner functions in `queries.go`
**Effort:** Medium

#### Option 7: Combat Event Log (Replace fmt.Printf)
**Problem:** Both systems use `fmt.Printf` for gameplay events. No way to surface these to players.
**Solution:** Define `CombatEvent` struct + collector. Hooks emit events; GUI layer renders them.
**Trade-off:** Threading through all behaviors. Benefits: combat log UI, replay, debugging.
**Files:** New `combatcore/combatevent.go`, all behavior functions
**Effort:** Medium-high

---

## Recommended Execution Order

1. **Option 3** (collapse RunXxx pairs) — quick win, reduces boilerplate before anything else
2. **Option 5** (extract behavior_dispatch.go) — trivial reorg, clarifies structure
3. **Option 2** (squad-aware artifact dispatch) — fixes the most likely source of future bugs
4. **Option 1** (per-perk state isolation) — biggest architectural improvement, prevents PerkRoundState from becoming unmaintainable
5. **Option 4** (split behaviors.go) — do when perk count reaches ~30

Options 6 and 7 are quality-of-life improvements to pursue after the structural work is done.
