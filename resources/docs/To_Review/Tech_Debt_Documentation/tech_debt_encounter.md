# Technical Debt Analysis: `mind/encounter/`

**Date:** 2026-05-10
**Scope:** 8 files, 1,008 LOC, 0 tests
**Context:** Recent commit `f1d9993` ("Cleaned up encounter package") means structural bones are decent — this is debt at the seams, not foundational rot.

---

## File Sizes

| File | Lines |
|---|---|
| `encounter_service.go` | 272 |
| `resolvers.go` | 228 |
| `starters.go` | 153 |
| `encounter_setup.go` | 127 |
| `encounter_trigger.go` | 94 |
| `types.go` | 81 |
| `validators.go` | 33 |
| `rewards.go` | 20 |
| **Total** | **1,008** |

---

## 1. Debt Inventory

### A. Testing Debt — CRITICAL

| Metric | Value |
|---|---|
| Test files | **0** |
| Lines covered | **0 / 1,008** |
| Public surface tested | **0%** |
| External callers | 5 (raid, guicombat, guioverworld, guiartifacts, guispells) |

`EncounterService.ExitCombat` (89 lines, 6 sequenced steps with snapshot/cleanup/callback) is the most consequential untested method in the module. `OverworldCombatResolver.resolveVictory/resolveDefeat`, `CalculateIntensityReward`, and the trio of `Trigger*` factories are also untested. Every regression here only surfaces as a runtime bug during gameplay.

**Risk:** High. Combat resolution is silent on error (returns `nil` from `Resolve` on missing nodes — 2 sites in `resolvers.go:33,39`); a future refactor could break victory rewards or threat propagation without any signal.

### B. Logging Debt — `fmt.Printf` scattered throughout

11 `fmt.Printf` call sites mixed into business logic:

- `resolvers.go:33,39` — `WARNING:` for missing node/component
- `resolvers.go:79,103,139,162,172,174,200` — gameplay events (rewards, defeat, capture, retreat)
- `encounter_service.go:124` — `WARNING:` for missing encounter entity

All also redundantly emit `core.LogEvent` for the same gameplay events (`resolvers.go:70, 93, 131, 159, 191`). So you have two parallel log streams: structured (`LogEvent`) and unstructured (`Printf`). The `Printf` is dev console noise that has shipped past the cleanup pass.

**Cost:** ~30 min to replace; immediate clarity win in console output.

### C. Architecture Debt

#### C1. Triple "skip resolution" encoding

The same intent — "this combat type owns its own resolution" — is encoded three different ways:

1. `SkipServiceResolution = true` (raid uses this)
2. `BuildResolver` returns `nil` (debug encounters use this — `starters.go:65`)
3. `ThreatNodeID == 0` sentinel (random encounters; the trigger comment at `encounter_trigger.go:37` calls this out explicitly)

`EncounterService.ExitCombat:122` has to check `!enc.SkipServiceResolution && enc.BuildResolver != nil` to handle two of them, while the third (`ThreatNodeID=0`) leaks into resolver construction. Three signals for one decision is exactly the kind of subtle invariant that breaks during the next refactor.

**Fix:** Collapse to one — make `BuildResolver == nil` the canonical "skip" signal and delete `SkipServiceResolution`. The raid path can return `nil` from its `BuildResolver` now that callbacks own resolution.

#### C2. Dead/redundant `IsGarrisonDefense` field

`OverworldEncounterData.IsGarrisonDefense` (set in `encounter_trigger.go:89`) is now redundant with `CombatSetup.Type == CombatTypeGarrisonDefense`. Per `contracts.go:29`, the enum was introduced specifically to "replace the IsGarrisonDefense/IsRaidCombat bool flags." The bool was kept on the data component but no read site uses it for control flow inside the encounter package — pure dead weight.

#### C3. Single-subscriber callback pattern is fragile

`postCombatCallback` (`encounter_service.go:38`) is a single-slot, last-write-wins field. Only `RaidRunner` uses it. The `RaidRunner` registers in its constructor (`raidrunner.go:55`) and has to embed a defensive guard (`raidEntityID != 0 && raidState.Status == RaidActive`) to filter out non-raid combats that flow through the same channel.

This is OK now (1 subscriber) but won't scale to a 2nd. Either:
- Document it as "single-listener by design, do not add" (pin invariant), or
- Promote to a small `[]listener` slice (5 LOC change, removes the guard burden from each subscriber).

#### C4. Magic numbers in `rewards.go`

```go
baseGold := 100 + (intensity * 50)    // rewards.go:8
baseXP := 50 + (intensity * 25)        // rewards.go:9
basePoints := 1 + intensity            // rewards.go:10
typeMultiplier := 1.0 + (float64(intensity) * 0.1)  // rewards.go:12
```

All 6 constants hardcoded, no config hook. Per `PROGRESSION.md:299`, ArcanaPts and SkillPts always scale identically because `basePoints` is shared — a balancing decision frozen in code. For a tactical RPG mid-balance-iteration, these belong in `templates/gameconfig` alongside other tunables.

### D. Code Smells (minor)

- **`encounter_service.go:101–186` `ExitCombat` is 89 lines, 6 numbered steps.** Each step is short and labeled, so it reads OK, but it mixes resolution dispatch, sprite restoration, history snapshot, teardown, and notification. Splitting into `resolveExit`, `recordHistory`, `notifyListeners` would shrink the function and make each step independently testable.
- **`encounter_service.go:50` `make([]*CompletedEncounter, 0, 10)` then `maxHistory: 10` literal.** Cap and length use the same `10` but aren't tied — the literal in `make` is decorative.
- **3 nearly-identical `Trigger*` functions** (`encounter_trigger.go:38, 50, 77`) all do `createEncounterEntity(manager, &core.OverworldEncounterData{...})` with different field sets. Could be one builder, but the 3-function form is more readable than a `TriggerOptions` struct — judgment call, **leave it**.
- **EncounterController interface (`types.go:77`)** is a 3-method subset of `EncounterService` for the GUI. Maintained in two places. This is intentional decoupling, but worth noting as a small surface to keep in sync.

---

## 2. Impact Assessment

| Item | Dev cost (current) | Risk |
|---|---|---|
| Zero tests on `ExitCombat` | Every refactor = manual playthrough. ~2h wasted/refactor | **High** — silent failures |
| `fmt.Printf` noise | Console clutter slows debugging | Low |
| Triple skip-resolution encoding | Next combat type adds a 4th flavor | **Medium** — invariant rot |
| Dead `IsGarrisonDefense` bool | Reader confusion, save-format bloat | Low |
| Magic numbers in rewards | Balance tweak = recompile | Medium during balancing phase |
| Single-slot callback | Adding a 2nd listener = redesign | Low (only matters if you add one) |

---

## 3. Prioritized Roadmap

### Quick Wins (< 1 day total)

1. **Delete `fmt.Printf` calls in `resolvers.go` + `encounter_service.go`.** They duplicate `core.LogEvent`. ~30 min.
2. **Move reward constants to config.** Lift the 4 magic numbers in `rewards.go` to `templates/gameconfig` (or a new `encounter_config.go`). ~45 min.
3. **Remove `OverworldEncounterData.IsGarrisonDefense`.** Replace its 1 write site with reliance on `CombatType`. Verify no save-load reads it. ~30 min.

### Medium-Term (1–2 sprints)

4. **Add `encounter_service_test.go`.** Target `ExitCombat` first — fixture: build an `ActiveEncounter`, mock `ModeCoordinator` + `CombatTeardown`, assert history recording, callback firing, sprite restoration, and the snapshot-before-clear ordering. Then `OverworldCombatResolver.resolveVictory/resolveDefeat`. ~6h. **Highest ROI item.**
5. **Collapse skip-resolution signals.** Make `BuildResolver == nil` the single skip signal; delete `SkipServiceResolution`; update `raid/starters.go:43`. ~2h including raid retest.
6. **Split `ExitCombat` into 3 helpers** (`resolveExit`, `recordHistory`, `notify`). ~1h once tests exist.

### Long-Term / Watch-Don't-Fix

7. `postCombatCallback` single-slot — fine until a 2nd subscriber exists. Add a comment pinning the invariant; promote to slice only if a real 2nd consumer appears.
8. `EncounterController` 3-method interface duplication — accept as a decoupling tax.

---

## 4. Prevention

- **Add `encounter` to a coverage gate** once the first test exists. CI fail < 40% for the package.
- **Linting rule:** ban new `fmt.Printf` in `mind/encounter/**` (use `core.LogEvent` or a structured logger).
- **One-flag-per-decision rule** when adding new `CombatSetup` fields — block PRs that add a 2nd "skip X" signal.

---

## Files Touched in Roadmap

| File | Quick wins | Medium-term |
|---|---|---|
| `mind/encounter/resolvers.go` | Delete 9 Printf | Add tests |
| `mind/encounter/encounter_service.go` | Delete 1 Printf | Split `ExitCombat`, add tests |
| `mind/encounter/rewards.go` | Constants to config | — |
| `mind/encounter/encounter_trigger.go` | Drop `IsGarrisonDefense` write | — |
| `campaign/overworld/core/components.go` | Drop `IsGarrisonDefense` field | — |
| `mind/combatlifecycle/contracts.go` | — | Drop `SkipServiceResolution` |
| `campaign/raid/starters.go` | — | Drop `SkipServiceResolution: true` |
| `mind/encounter/encounter_service_test.go` (new) | — | Test fixture + 5–8 cases |

**Net effort estimate:** ~1.5 days for quick wins + medium-term items. ROI dominated by item #4 (testing) — every other refactor in this package becomes safer once it lands.
