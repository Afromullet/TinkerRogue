# Package Boundaries: `combatlifecycle` / `encounter` / `spawning`

**Last Updated:** 2026-04-22

Analysis and recommended cleanup for the three `mind/` packages that implement combat lifecycle, encounter flow, and enemy squad generation.

---

## Current Dependency Graph

```
spawning (leaf — zero mind/ imports)
   ↑
encounter ──→ combatlifecycle (zero mind/ imports)
```

- **No cycles.** The flow is strictly one-directional.
- `combatlifecycle` and `spawning` are independent of each other.
- Only `encounter` depends on both.

The packages are **not inter-dependent** today. The concern is responsibility clarity and preventing future drift, not untangling existing coupling.

---

## One-Sentence Responsibility Per Package

| Package | Responsibility |
| --- | --- |
| `mind/combatlifecycle/` | Combat lifecycle vocabulary (contracts + orchestrators) **plus** the shared ECS mutation helpers (enrollment, casualties, cleanup, reward distribution) that every combat path needs. |
| `mind/encounter/` | The overworld/garrison flavor of those contracts plus service-layer state (active encounter, history, post-combat callbacks, mode transitions). |
| `mind/spawning/` | Pure enemy-squad and position generator from a power budget. No lifecycle awareness. |

`spawning` is crisp. `encounter` is crisp. `combatlifecycle`'s self-description at `contracts.go:1-11` ("only defines the shared contracts") is factually wrong — `reward.go`, `cleanup.go`, `casualties.go`, `enrollment.go` contradict it.

---

## Misplaced Symbols

### 1. `EncounterCallbacks` — belongs in `encounter`, not `combatlifecycle`
- **Location:** `mind/combatlifecycle/contracts.go:153-159`
- **Why it's misplaced:** Only `EncounterService` satisfies it. Only the GUI (`gui/guicombat`, `gui/guispells`, `gui/guiartifacts`) consumes it. It is pure encounter-layer glue.
- **Destination:** `mind/encounter/types.go`
- **Catch:** The interface references `combatlifecycle.CombatExitReason`, `combatlifecycle.EncounterOutcome`, `combatlifecycle.CombatCleaner`. `encounter/types.go` already imports `combatlifecycle`, so the move is clean. GUI callers must swap the import alias: `combatlifecycle.EncounterCallbacks` → `encounter.EncounterCallbacks`.

### 2. Two `*EncounterService` methods in the wrong file
- **Location:** `mind/encounter/resolvers.go:212-233`
- **Symbols:** `getAllPlayerSquadIDs`, `returnGarrisonSquadsToNode`
- **Why it's misplaced:** Both are methods on `*EncounterService`, used by the service's own `ExitCombat`, never by any resolver.
- **Destination:** `mind/encounter/encounter_service.go`

Nothing in `spawning` is misplaced.

---

## Is `combatlifecycle` Too Broad?

Mildly. It holds both the vocabulary (contracts, orchestrators) and the shared ECS helpers (enrollment, cleanup, casualties, reward). Splitting into `combatlifecycle` (contracts) + `combatops` (helpers) would be textbook-clean but moves ~400 LOC for little gain — every caller is already a combat caller.

**Recommendation:** Keep it. Fix the doc comment. Revisit only if:
- A non-combat subsystem (scripted cutscene, tutorial) starts calling `Grant` or `EnrollSquadInFaction`, OR
- The package exceeds ~800 LOC.

---

## Is `encounter` Too Broad?

No. All eight files are overworld/garrison/threat specific. The "multiple concerns" view is an illusion — it's one concern (overworld encounter flow) expressed in layers:

- triggering (threat → encounter entity)
- service state (active encounter, history, callbacks)
- starters (prepare combat)
- resolvers (post-combat outcome)
- setup (spawn orchestration)
- reward calc (intensity math)

One readability nit: `EncounterService.ExitCombat` at `encounter_service.go:137-198` is a 60-line method with six numbered `// Step N:` comments. Extract each step into a named private helper.

---

## Where Boundaries Are Thinnest (Future Drift Risk)

### 1. `combatlifecycle` → `encounter`
Risk: a "universal post-combat side effect" lands inside `ExecuteResolution` (`pipeline.go:30`) instead of behind a new resolver-side hook interface. **Mitigation:** Keep `ExecuteResolution` a strict dispatcher. If something universal is needed, add a new resolver-side interface — do not inline behavior in the orchestrator.

### 2. `spawning` growing inward
Risk: features like "spawn scaled to faction control" or "spawn based on encounter history" will tempt someone to import `encounter` into `spawning`. **Mitigation:** Block at the function signature. Push context *into* `GenerateAttackerSquads` parameters (or into `OverworldEncounterData`). `spawning` must never *reach* for state.

### 3. `EncounterCallbacks` accreting methods
Already at 3 methods. If it grows past ~6 it becomes a mirror of `EncounterService`'s public API trapped inside `combatlifecycle`. **Mitigation:** Move it to `encounter/types.go` now (see Misplaced Symbols #1).

---

## Recommended Cleanup (Low-Risk, ~2 Hours Total)

Apply in order, smallest first:

1. **Rewrite package doc** at `mind/combatlifecycle/contracts.go:1-11` to honestly describe "contracts + orchestration entry points + shared ECS utilities."

2. **Rename** `mind/encounter/rewards.go` → `mind/encounter/overworld_rewards.go`. Signals domain-specific reward math vs. generic `Reward`/`Grant` distribution in `mind/combatlifecycle/reward.go`. No symbol changes.

3. **Move `EncounterCallbacks`** from `mind/combatlifecycle/contracts.go:153-159` → `mind/encounter/types.go`. Update GUI imports in `gui/guicombat/combatdeps.go`, `gui/guicombat/combatmode.go`, `gui/guispells/spell_deps.go`, `gui/guiartifacts/artifact_deps.go`.

4. **Relocate two orphan service methods** from `mind/encounter/resolvers.go:212-233` → `mind/encounter/encounter_service.go`:
   - `getAllPlayerSquadIDs`
   - `returnGarrisonSquadsToNode`

5. **Extract `ExitCombat` helpers** from `mind/encounter/encounter_service.go:137-198`. Each numbered step becomes a private method on `*EncounterService`:
   - `resolveCombatOutcome(enc, reason, result)` — step 1
   - `markDefeatedOnVictory(enc, result)` — step 2
   - `restorePlayerPosition(enc)` — step 3
   - (step 4 is already one call — leave it)
   - `cleanupCombatEntities(enc, result, cleaner)` — step 5
   - (step 6 is already one call — leave it)

---

## Changes Explicitly NOT Recommended

- **Do not split `combatlifecycle`** into `combatlifecycle` + `combatops`. Textbook-clean but ~400 LOC of churn for zero caller benefit today.
- **Do not touch `spawning`.** It is correctly shaped. Only ongoing discipline: never let it import `encounter` or `combatlifecycle`.
- **Do not modify `ExecuteResolution`** (`pipeline.go:30`). It is a strict dispatcher. Add hook interfaces rather than inlining behavior.

---

## Verification Checklist

After applying the cleanup:

1. `go build ./...` — compiles.
2. `go vet ./...` — no new warnings.
3. `go test ./mind/...` — existing `mind/combatlifecycle/resolution_test.go` still passes.
4. `go test ./...` — full suite passes.
5. Runtime smoke test — trigger one overworld encounter, one garrison defense, and one flee. Each must reach combat, resolve, and exit cleanly with rewards granted (exercises both `EncounterCallbacks` paths and the refactored `ExitCombat`).
6. **Import discipline grep:**
   - `grep -r "mind/encounter" mind/combatlifecycle mind/spawning` → empty
   - `grep -r "mind/combatlifecycle\|mind/encounter" mind/spawning` → empty

---

## Ongoing Boundary Discipline

- `mind/spawning` must remain a leaf. New "scale by X" features take `X` as a parameter; they do not import `X`'s package.
- `mind/combatlifecycle` must remain zero-mind-imports. New combat types add their implementations in their own domain package and register via the existing interfaces.
- New interfaces that only bridge GUI ↔ encounter belong in `mind/encounter/types.go`, not `mind/combatlifecycle/contracts.go`.
