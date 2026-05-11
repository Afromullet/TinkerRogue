# Technical Debt Analysis: `tactical/powers/perks/`

**Date:** 2026-05-10
**Scope:** 12 files, 2992 LOC (1971 production / 1021 test)
**Status:** Mature subsystem; recent commit (`f44e81c Perks cleanup`) shows ongoing care.

---

## Verdict at a Glance

The perks package is in **noticeably good shape** for a stateful gameplay subsystem of this size. The core abstractions (`BasePerkBehavior`, `run`/`combatCtx`/`HookContext`, `GetPerkState`/`GetBattleState`) eliminate the most damaging shapes of duplication. Tests are dense (1021 LOC for 1971 production LOC ≈ 52%). What remains is **medium-grade debt**: a handful of ergonomic issues, light coupling, and validation gaps — no critical or high-severity items.

---

## 1. Debt Inventory

### A. Code Debt — Duplication

#### A1. Per-perk balance-config plumbing (medium)
`balanceconfig.go:11-28` declares 17 typed sub-structs that always pair 1:1 with a perk. Each new perk requires:

1. A `XBalance` struct
2. A field on `PerkBalanceConfig`
3. An entry in `validatePerkBalance` (`balanceconfig.go:137-203`)
4. A JSON key
5. A default in `setupTestBalance` (`perks_test.go:38-61`)

That's five touch-points spread across two files for one tunable. ~70 lines of `validatePerkBalance` are mechanical "must be positive" checks that could be table-driven with `(name string, value float64)` tuples.

#### A2. Repeated alive-and-role unit filtering (medium)
`unithelpers.go:23-122`. Four functions (`FindLowestHPUnit`, `FindHighestDexUnitByRole`, `CountTanksInRow`, `FindFirstTankInSquad`) all run the pattern: `iterate squad units → check alive → fetch entity → fetch role component`. `FindHighestDexUnitByRole` and `FindFirstTankInSquad` are essentially the same loop with different reducers.

A single `forEachAliveUnit(squadID, manager, fn)` helper and a `forEachAliveUnitWithRole(squadID, role, manager, fn)` would shrink this file ~30%.

#### A3. Repeated `ChebyshevDistance` proximity checks (low)
Two perks (`IsolatedPredator` `behaviors.go:120-146`, `GuardianProtocol` `behaviors.go:286-310`) implement the exact same pattern: get squad position, iterate friendlies, check `ChebyshevDistance` ≤ N. A helper `ForEachFriendlySquadWithinRange(squadID, range, manager, fn)` would deduplicate.

### B. Code Debt — Complexity

#### B1. `ResoluteBehavior.TurnStart` is O(units) per turn even when nobody dies (low)
`behaviors.go:565-578` snapshots HP for every unit every turn; the snapshot is only consulted in `DeathOverride`. For squads with the perk this is unavoidable, but it could be rewritten lazily (snapshot on first damage taken that round, or on a cheaper signal).

#### B2. `behaviors.go` is 642 LOC, 17 behaviors (medium)
Not a god-class — each behavior is 10-30 lines and well-isolated by section comments. But the file is becoming a navigational liability. Splitting into `behaviors_stateless.go` / `behaviors_per_round.go` / `behaviors_per_battle.go` would mirror the existing taxonomy comment and matches the package's naming convention elsewhere.

#### B3. `LastLine.AttackerDamageMod` subtracts from `HitModifier` — sign convention bug magnet (HIGH risk)
`behaviors.go:208`: `modifiers.HitModifier -= PerkBalance.LastLine.HitBonus`

This is actually a *bonus* (`HitBonus: 10` in the test config). Same inversion in `DeadshotsPatience` `behaviors.go:474`. The convention is implicit and easy to invert by accident.

**Why it's debt:** the sign convention isn't documented on the field or hook. Either rename to `HitPenalty` or add a doc comment on `DamageModifiers.HitModifier`.

### C. Architecture Debt

#### C1. Logger plumbed through every HookContext (low)
Every dispatch builds a fresh `HookContext` and copies `Logger` + `Manager` from the dispatcher (`dispatcher.go:51`, `system.go:144-148`). This is the right shape for testability, but the same pattern is duplicated in `RunTurnStartHooks` outside the dispatcher. Consolidating turn-start dispatch into `SquadPerkDispatcher.run` would remove that drift surface.

#### C2. `getSquadIDForUnit` is private to perks but the same lookup exists in many places
`queries.go:32-39`. This is a project-wide pattern, not a perks-specific debt — but the function being lower-case here means callers outside perks duplicate it. Consider promoting to `squadcore.GetSquadIDForUnit`.

#### C3. Two dispatch flavors (low)
Damage-pipeline hooks go through `SquadPerkDispatcher.run` which fills `PowerContext + RoundState`. `RunTurnStartHooks` (`system.go:139`) builds the same structure inline. Not severe — they are consistent today — but they will drift.

### D. Validation / Correctness Gaps

#### D1. `validatePerkBalance` only checks "> 0"
Doesn't catch nonsensical values (e.g., `DamageMult: 0.001` would silently nerf a perk to nothing; `Fortify.MaxStationaryTurns: 1000` would never cap). Range bounds (e.g., `DamageMult ∈ [0.5, 5.0]`) would catch typos in JSON.

#### D2. No validation that JSON `Roles` strings exist (warning only)
`registry.go:147` warns on invalid roles but **still inserts the perk into the registry**. That means a perk with all-invalid roles silently never appears in role-filtered UI. Fail-loud or skip-and-log-clearly.

#### D3. `ValidateHookCoverage` is defined but never called from production startup
`registry.go:183-196` exists; `Grep` shows no callers outside tests. A behavior/JSON drift would only be noticed by the test suite, not at runtime — but in a dev-mode boot it should panic.

### E. Testing Debt

#### E1. `setupTestPerkRegistry` hardcodes 13 perks; full registry has 22
`perks_test.go:65-80`. New perks won't be covered by exclusion tests until someone remembers to add them. A loop seeded from `PerkRegistry` would scale automatically.

#### E2. No test for `LoadPerkDefinitions` JSON parsing (low)
Malformed JSON only `WARNING:`s and silently leaves an empty registry — combat would just have zero perks. A small fixture-based test would prevent silent failure.

#### E3. No fuzz / property test for the state lifecycle
`ResetPerkRoundStateTurn` snapshots/clears across multiple flags; a property test asserting "after N random turn cycles, snapshots equal prior turn state" would catch invariant violations the existing 4 unit tests miss.

### F. Documentation Debt

#### F1. `PerkRoundState.PerkBattleState` comment claims "Cleared entirely by ResetPerRound"
`components.go:53`. It's not — `ResetPerkRoundStateRound` only clears `PerkState`, not `PerkBattleState` (correctly). The comment contradicts the code.

#### F2. Mutation rules of `HookContext` between hooks aren't documented
Several perks mutate state via stored pointers (e.g., `RecklessAssaultBehavior` stores `&RecklessAssaultState{}` and later mutates `state.Vulnerable`). This is fine but unstated; future perks may expect `GetPerkState` to return a copy.

---

## 2. Impact Assessment

| Item | Severity | Velocity Impact | Risk |
|---|---|---|---|
| A1 Balance plumbing | Medium | ~30 min / new perk × 22 perks → already paid; adds ~30min to each future perk | Low (loud failures) |
| A2 Unit-helper duplication | Medium | Bug fixes touch 4 functions instead of 1 | Low |
| A3 Proximity duplication | Low | <1 hr / similar perk | Low |
| B2 behaviors.go size | Medium | Slows code review and navigation | Low |
| **B3 HitModifier sign** | **High** | **Easy bug source — silent wrong-direction effect** | **High (gameplay bug)** |
| D1/D2/D3 Validation gaps | Medium | Silent misconfiguration | Medium (live bugs in JSON edits) |
| E1 Hardcoded test registry | Low-Medium | New perks ship without exclusion coverage | Medium |
| F1 Stale comment | Low | Misleads future readers | Low |

**Highest-leverage single fix:** B3 (HitModifier sign convention). Two perks already use it backward-feeling; the third one to be written is a coin-flip bug.

---

## 3. Prioritized Roadmap

### Quick Wins (≤ 1 day total)

1. **Fix `PerkBattleState` doc comment** (`components.go:53`) — 5 min.
2. **Document `HitModifier` sign convention** on `combattypes.DamageModifiers.HitModifier` and add a passing-direction test for `LastLine` and `DeadshotsPatience` — 30 min.
3. **Call `ValidateHookCoverage` at startup in `DEBUG_MODE`** — 15 min. Mirrors `LoadPerkBalanceConfig`'s panic pattern.
4. **Make `LoadPerkDefinitions` skip invalid-role perks instead of warning-and-inserting** (or fail loud in DEBUG) — 20 min.
5. **Loop-driven `setupTestPerkRegistry`** seeded from `PerkRegistry` — 20 min. Prevents future drift.

### Medium-Term (1–3 days)

6. **Extract unit-iteration helpers** (`forEachAliveUnit`, `forEachAliveUnitWithRole`) and refactor the four functions in `unithelpers.go` — half day. Direct LOC reduction; opens path for more efficient role-indexed lookups later.
7. **Split `behaviors.go`** into `behaviors_stateless.go` / `behaviors_per_round.go` / `behaviors_per_battle.go` — 1 hour. Mechanical move; existing `init()` registration stays in one of them or is split too.
8. **Add `ForEachFriendlySquadWithinRange`** helper in `queries.go` and migrate IsolatedPredator + GuardianProtocol — 1 hour.
9. **Range-bound validation** in `validatePerkBalance` (e.g., `DamageMult ∈ [0.1, 10.0]`) — 1–2 hours.

### Long-Term (only if it pays off)

10. **Table-driven balance validation.** Could shrink `validatePerkBalance` from ~70 LOC to ~15. But the current code is readable and rarely changes — defer until a 25th perk arrives.
11. **Consider a `PerkBalance` field tag-driven validator** (struct tags like `validate:"gt=0,lt=10"`). Brings in a dep or a tiny custom reflector. Probably not worth it for 17 fields.

---

## 4. Prevention

- **Pre-merge gate:** require `ValidateHookCoverage()` to return empty in `go test ./tactical/powers/perks/...` (already true today via the test suite indirectly — make it explicit via a `TestNoOrphanPerks` test).
- **PerkID-balance struct mapping test:** assert every `PerkID` constant has a corresponding `PerkBalance.<X>` field accessible (reflection-based) — catches missing balance configs at PR time.
- **Naming convention for sign-laden modifiers:** if `HitPenalty` (subtracted from accuracy) and `HitBonus` (added) are distinct fields, the bug class disappears.

---

## 5. What I Did NOT Find (positive signals)

- **No god-classes.** Largest type method count is `SquadPerkDispatcher` with 12 methods, all 5–15 LOC.
- **No circular dependencies.** Perks import combat/squads/powercore; nothing in those imports perks.
- **No `*ecs.Entity` storage** — all interfaces use `ecs.EntityID`. ECS hygiene is clean.
- **No legacy dead code.** Every exported symbol I spot-checked has callers.
- **Test coverage is solid** for state lifecycle, exclusion, and multi-perk interactions.

The package is a strong reference example of how `tactical/powers/*` should be structured. Most of the debt above is incremental polish rather than structural risk.

---

## 6. File-by-File LOC Reference

```
balanceconfig.go     203
behaviors.go         642  ← largest file
components.go        186
dispatcher.go        223
hooks.go              98
init.go               19
perkids.go            30
perks_test.go       1021
queries.go            79
registry.go          196
system.go            153
unithelpers.go       142
─────────────────────────
Total               2992
Production          1971
Test                1021 (52%)
```
