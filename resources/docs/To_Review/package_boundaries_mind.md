# Package Boundaries Cleanup: `combatlifecycle` / `encounter` / `spawning`

**Last Updated:** 2026-04-22

Action list. Apply in order.

---

## 1. Rewrite the `combatlifecycle` package doc

**File:** `mind/combatlifecycle/contracts.go:1-11`

Replace the current "only defines the shared contracts" comment with one that describes the actual contents: contracts + orchestration entry points + shared ECS utilities (enrollment, casualties, cleanup, reward distribution).

---

## 2. Rename `mind/encounter/rewards.go` → `mind/encounter/overworld_rewards.go`

No symbol changes. Filename only. Signals domain-specific reward math vs. the generic `Reward`/`Grant` distribution in `mind/combatlifecycle/reward.go`.

---

## 3. Move two orphan methods from `resolvers.go` to `encounter_service.go`

**From:** `mind/encounter/resolvers.go:212-233`
**To:** `mind/encounter/encounter_service.go`

- `(es *EncounterService) getAllPlayerSquadIDs()`
- `(es *EncounterService) returnGarrisonSquadsToNode(nodeID ecs.EntityID)`

Both are `*EncounterService` methods used only by the service itself, not by any resolver.

---

## 4. ~~Move `EncounterCallbacks` to `encounter`~~ — **DONE (2026-04-23)**

Moved to `mind/encounter/types.go` and renamed `EncounterCallbacks` → `EncounterController` to stop the naming collision with `EncounterService.postCombatCallback` (which is an actual callback). Also renamed the adjacent `CombatCleaner` interface + its `CleanupCombat` method to `CombatTeardown` + `TeardownCombat` so the orchestrator (`EncounterService.ExitCombat`) and the step it invokes no longer read as duplicates of each other.

---

## Verification

1. `go build ./...`
2. `go vet ./...`
3. `go test ./...`
4. Runtime smoke test — trigger one overworld encounter, one garrison defense, and one flee. All three must resolve and exit cleanly with rewards granted.
5. Import discipline grep — both must return empty:
   - `grep -r "mind/encounter" mind/combatlifecycle mind/spawning`
   - `grep -r "mind/combatlifecycle\|mind/encounter" mind/spawning`

---

## Ongoing Boundary Discipline

- `mind/spawning` stays a leaf. New "scale by X" features take `X` as a parameter; they do not import `X`'s package.
- `mind/combatlifecycle` stays zero-mind-imports. New combat types implement existing interfaces from their own domain package.
- Interfaces that only bridge GUI ↔ encounter belong in `mind/encounter/types.go`, not `mind/combatlifecycle/contracts.go`.
- Keep `ExecuteResolution` (`mind/combatlifecycle/pipeline.go:30`) a strict dispatcher. Universal post-combat side effects go behind a new resolver-side hook interface, not inline.
