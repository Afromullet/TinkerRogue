# Complexity Analysis — Karen's Reality Check

**Reviewed:** 2026-04-23
**Source document:** `resources/docs/complexity_analysis_combined.md`
**Branch examined:** Encounter-Cleanup (recent commits include encounter cleanup and commander-level perk/spell unlocking)

---

## Bottom Line Up Front

The combined document is mostly accurate and executable as written. The metrics are real, the file:line references are correct within a line or two, and the HIGH list is genuinely high-priority rather than aesthetic cleanup dressed up as urgency. Two items need calling out before anyone executes: the Leader Abilities deletion claim is substantially correct but undersells the true scope of the deletion (the component is added to every leader at creation time, is serialized unconditionally, and the evaluator actively uses it — this is not just dead code sitting in a corner), and the "save-file version bump" description of the migration risk is a hand-wave that understates what actually needs to happen. The document is worth acting on, but the Leader Abilities section needs one more paragraph of specifics before anyone touches it.

---

## Claims Verified / Busted

### Claim 1: `EquipAbilityToLeader` has zero production callers — VERIFIED

Grep result is conclusive:

```
tactical/squads/squadcore/squadabilities.go:10  // definition comment
tactical/squads/squadcore/squadabilities.go:12  func EquipAbilityToLeader(  // definition
```

Two hits. Both are the definition line and its comment. No call site exists anywhere in production code. The document's claim holds.

### Claim 2: `CheckAndTriggerAbilities` runs "unconditionally every turn" — PARTIALLY CORRECT, BUT THE FUNCTION HAS EARLY EXITS THE DOCUMENT DOES NOT MENTION

The document says "wasted per-turn work." That framing is accurate but incomplete. Reading `combatabilities.go:14-63`:

The function exits immediately if:
1. The squad has no leader (`leaderID == 0`) — line 16
2. The leader entity is not found — line 22
3. The leader entity lacks `AbilitySlotComponent` — line 25
4. The leader entity lacks `CooldownTrackerComponent` — line 29

So in the current codebase where no abilities are ever equipped via `EquipAbilityToLeader`, every call to `CheckAndTriggerAbilities` terminates at line 25 or 29 after three map lookups. The "wasted per-turn work" is real but it is cheap — not the full loop at lines 40-56. The document implies the ability evaluation loop runs every turn. It does not. This does not change the verdict (delete it), but the "performance" framing is overstated. The actual cost is 3-4 nil/HasComponent checks per squad per trigger site, not 4-slot iteration.

Call sites confirmed at:
- `combatactionsystem.go:205, 209` — post-attack for surviving squads
- `turnmanager.go:70` — combat start for all squads
- `turnmanager.go:105` — start of each faction's turn (in `ResetSquadActions`)

That is 4 distinct call paths, not 3. The document says "3 sites." `turnmanager.go` contributes two separate call paths: one in the initialization loop and one in `ResetSquadActions`. The distinction matters because `ResetSquadActions` is called once per faction turn, not once per combat. This means the call count scales with the number of factions and turns, not just squads. Still cheap given the early exits, but the document undercounts the sites.

### Claim 3: AbilitySlotData serialized at `squad_chunk.go:275-291, 418+` — VERIFIED

Lines 275-291 serialize `AbilitySlotData` conditionally (if the component exists). Lines 417-428 deserialize it conditionally. Both confirmed. The `CooldownTrackerData` is also serialized at lines 287-291 and deserialized at 430-434.

The document does not mention that `AbilitySlotComponent` and `CooldownTrackerComponent` are added to every leader entity at creation time unconditionally in `squadcreation.go:25-32` (via `AddLeaderComponents`). Since no abilities are ever equipped, every saved leader will have an `AbilitySlotData` with all slots at default zero/false values. Deleting the subsystem means saved games that loaded this data would fail to find the component — but since the data is all-zero defaults, it is safe to silently drop on load. The document's "save-file version bump" recommendation is correct but the reasoning should be: bump the version so old saves that contain ability data fields in JSON are handled gracefully, not because schema migration is complex. The fields will simply be ignored if the component is no longer added on load. No migration logic is needed — just a version gate and a note that old saves round-trip cleanly because the data was never meaningful.

### Claim 4: `evaluator hook at mind/evaluation/power.go:121` — VERIFIED, BUT THE DOCUMENT UNDERSTATES THE DEPENDENCY

The document references `power.go:121` as a call site to clean up. Reading the actual code:

- `power.go:104`: `calculateRoleValue(roleData) + calculateAbilityValue(entity) + calculateCoverValue(entity)`
- `power.go:113-134`: `calculateAbilityValue` reads `AbilitySlotComponent`, iterates equipped slots, calls `GetAbilityPowerValue`
- `roles.go:44-46`: `GetAbilityPowerValue` defined separately

This is not just a call site cleanup. `calculateAbilityValue` is wired into the power evaluation pipeline that the AI uses. Deleting it means power scores for leaders will drop by whatever `GetAbilityPowerValue` would have returned for equipped abilities. Since no abilities are ever equipped, this is always returning 0.0. The evaluator is doing correct arithmetic on data that is always zero. Deleting it is safe — but the document should say explicitly that the power evaluation result is unchanged because the addend is always 0.

### Claim 5: `applyFireballEffect` directly mutates `common.Attributes.CurrentHealth` — VERIFIED, IS A REAL ECS VIOLATION

`combatabilities.go:228`: `attr.CurrentHealth -= params.BaseDamage`. This bypasses the combat math pipeline entirely — no hit calculation, no cover, no perk hooks, no death event recording. The fireball kills units without any of the normal combat systems knowing about it. The document correctly flags this as a minor ECS boundary violation. It is also a correctness bug: Resolute/Bloodlust/DeathOverride perks would not fire, killed units would not be recorded in `UnitsKilled`, and the battle log would not reflect the kills. This is academic since nothing equips Fireball, but it confirms the system was never finished and justifies deletion.

Also worth noting: all four `applyXEffect` functions use bare `fmt.Printf` debug statements that would print to stdout in production. This is additional evidence the subsystem was never made production-ready.

### Claim 6: `processAttack` death-override block at lines `88-116` with nestif 13 — VERIFIED AT CORRECT LINES

The block starts at line 88 (`if event.WasKilled && dispatcher != nil`) and ends at line 116. The raw metrics confirm nestif 13 for `combatprocessing.go:88`. The extraction to `applyDeathOverride` is straightforward — the block accesses `result`, `manager`, `event`, `defenderID`, `defenderMember`, `defSquadID`, `attr` — all of which can be passed as parameters. The comment at lines 86-87 explaining ordering is load-bearing and must move with the extracted function, not stay at the call site.

### Claim 7: `GenerateThreatSummary` cog=52 with repeated int/float64 JSON unmarshaling — VERIFIED

The pattern appears at lines 68-79 (intensity int + float64), 102-112 (new_intensity int + float64). The document's description of "three near-identical int/float64 JSON-unmarshaling blocks" is accurate. The function is 63 lines total (`funlen` confirms this). The `incrementThreatTypeStat` helper extraction is genuinely trivial.

### Claim 8: Panel registry `init()` sizes — VERIFIED

Raw metrics confirm:
- `combat_panels_registry.go:53` — cog 16, funlen 341 lines (confirmed)
- `squadeditor_panels_registry.go:49` — cog 15, funlen 384 lines (confirmed)

These numbers are real.

---

## Priority Reality Check

**HIGH item 1 — Split `HandleInput` (cyc 47, cog 76):** Fully justified. The function is 99 statements (`funlen` confirms). It has three distinct sub-mode dispatchers (spell, artifact, inspect) followed by normal-mode logic. The split is mechanical and low-risk. A seasoned engineer would not push back.

**HIGH item 2 — De-duplicate `initial_squads.go`:** Justified, but only medium priority in practice. This is test-only code (`testing/bootstrap/`). The duplication is real — `createRangedSquad:168` and `createCavalrySquad:324` share identical structure. A seasoned engineer might reasonably defer this until another reason to touch the file arises, since it affects zero production behavior. The document calls it HIGH unanimously; "HIGH for test code" is a slightly inflated label.

**HIGH item 3 — Extract `applyDeathOverride`:** Justified. The nestif-13 block is genuinely complex and self-contained. Effort estimate of under 1 hour is accurate.

**HIGH item 4 — Delete Leader Abilities subsystem:** Justified with caveats. The functional case is correct (zero production callers, always-zero power contribution, debug-printf code, unfinished ECS boundary). The "Risk: low" rating is also correct. But the effort description "medium" is vague. A more honest breakdown: deleting `combatabilities.go` is trivial, removing the 4 call sites of `CheckAndTriggerAbilities` is trivial, removing the component registration in `squadmanager.go` requires care (other components shift), removing serialization from `squad_chunk.go` is straightforward, removing `calculateAbilityValue` from `power.go` is one line, removing `GetAbilityPowerValue` from `roles.go` is trivial. The component definitions in `squadcomponents.go:185-289` are about 105 lines. Total actual work is probably 1.5-2 hours including a test run, not "medium" (which typically implies 2-3 hours of complexity, not LOC removal).

**MEDIUM items 5-12:** Correctly categorized. The cover bitmask refactor (item 5) is the most substantive — it touches per-attack math and should be done with the existing combat tests running green before and after. The `map[string]bool{"row,col"}` replacement (item 12) is genuinely trivial — one declaration and a few access-pattern changes.

---

## Missing Specifics

### No test coverage gap acknowledgment

The document recommends extracting `applyDeathOverride` (HIGH, "Risk: medium — combat-simulator regression test required") but does not state whether that test currently exists. It does. `tactical/combat/combatcore/combatexecution_test.go` has an execution test harness. `tactical/combat/combatservices/combat_service_test.go` exists. These should be run before and after the extraction. The document should have said "run `go test ./tactical/combat/...`" explicitly rather than vaguely gesturing at "combat-simulator regression test" — the combat simulator is an offline tool in `tools/`, not the unit tests.

Similarly for the bitmask refactor (MEDIUM item 5), the recommendation says "damage-calculation test coverage required" without noting whether it exists. `combatmath` package has no `_test.go` file at all. This is the real risk. If an engineer executes item 5 tomorrow, they will discover there are no combatmath unit tests, and they will need to write them first. The document does not flag this.

### Save-file version bump is underdefined

The document says deleting Leader Abilities requires a "save-file version bump." It does not say:
- Which version to bump to (currently `CurrentSaveVersion = 1` in `savesystem.go:51`)
- Whether migration logic is needed (it is not, because all ability data is default-zero)
- Whether existing saves will fail to load or just silently ignore the unknown JSON fields (they will silently ignore them since Go's `json.Unmarshal` into structs ignores unknown fields — but only if the save format removes the fields from the struct; if `savedAbilitySlots` struct still exists in the load path, it will still deserialize fine even without the components)

The correct answer is: bump `CurrentSaveVersion` to 2, add a comment explaining the change, verify the existing save test in `savesystem_test.go` is updated. No migration function needed. An engineer who reads only the combined document would not know this.

### `AbilitySlotComponent` is added to every leader, not just equipped leaders

The document says abilities are dormant because `EquipAbilityToLeader` is never called. This is accurate. What it does not say is that `AddLeaderComponents` (called from `squadcreation.go:25`) always adds `AbilitySlotComponent` with empty slots to every leader entity. This means `CheckAndTriggerAbilities` exits at line 25 (`HasComponent(AbilitySlotComponent)` returns true) and proceeds to line 33 to get the data — it does not exit at line 25. It exits at line 43 on the first iteration of the slot loop because `!slot.IsEquipped` is true for all slots. The document's claim that the function does "wasted per-turn work" is correct, but the early-exit analysis is off by one check.

### No mention of `fmt.Printf` debug pollution

The `combatabilities.go` file has four bare `fmt.Printf` calls that would print "[ABILITY] ..." to stdout in production if abilities were ever triggered. The document does not mention this. It is an additional argument for deletion, not merely refactoring.

### No mention of `mind/evaluation/roles.go` as a deletion target

The document lists `mind/evaluation/power.go:121` as a cleanup site. It does not mention that `GetAbilityPowerValue` in `roles.go:44-46` also needs deletion. An engineer executing the HIGH list would discover this during the deletion pass.

---

## Hand-Waves and Contradictions

### Contradiction: C2 dissent on panel-registry `init()` is unresolved

The document says refactoring-pro argues panel registries are "declarative tables" in Part C (Leave Alone), then the same document promotes them to MEDIUM in the action list. The recommendation text says "Treat as MEDIUM. Extraction helps grep-ability... accept that the file count stays constant." This is fine as a resolution, but the document should explicitly say refactoring-pro's Part C verdict was overridden by majority vote, not just note a "dissent." An engineer reading Part C will see "Leave Alone" and the action list will say "do it," with no clear authority on which wins.

### Hand-wave: "Effort: small (1-2 hr)" for `HandleInput` split

The `HandleInput` function at 99 statements is straightforward to split. But the document does not address the fact that the extracted sub-handlers (`handleSpellModeInput`, etc.) will need access to `cih` fields currently accessed inline. The split is a method extraction on the same receiver, so this is not actually a problem — but an engineer might hesitate when they see how many fields the spell mode block touches. The estimate is probably correct; it would be more confidence-inspiring if the document said "method extraction on `*CombatInputHandler` receiver, no new parameter passing needed."

### "Risk: low (save-file version bump)" for Leader Abilities is accurate but implies version bumping is a trivial task

In this codebase it is trivial (change one constant, update one test). But the document treats "save-file version bump" as a parenthetical, which could lead an engineer to skip it and produce a save format that silently differs from version 1 without being detectable. The version system at `savesystem.go:163-164` only rejects saves that are *newer* than the current version, not older ones with schema changes. This means bumping is not strictly required for correctness but is correct practice.

### The document claims `processAttack` has cog=57 "accounting for the bulk" from the death-override block

This is accurate in direction but imprecise. The nestif-13 death-override block contributes significantly to cog=57, but `processAttack` also has a per-target loop iterating over `targetIDs`, six `dispatcher != nil` guards, and a `DamageRedirect` branch. Extracting `applyDeathOverride` will reduce the function's cognitive load meaningfully but will not bring it close to a "clean" number. The document implies one extraction fixes the function; the real result will be a cog reduction to something in the 35-45 range, still elevated. This is fine — the document correctly says "extract `applyDeathOverride` only; keep pipeline intact" — but the framing of "accounts for the bulk" oversells the outcome.

---

## Executability Gaps

If an engineer sat down tomorrow to execute the HIGH list in order:

**Item 1 (Split `HandleInput`):** Executable immediately. No blockers. The sub-mode structure is already visible in the function (spell block at :120, artifact block at :157, inspect block at :177, normal block at :195+). Method extraction on `*CombatInputHandler`. Verify with `go build` and manual combat test — no automated GUI tests exist.

**Item 2 (De-duplicate `initial_squads.go`):** Executable immediately. Pure test code. One function signature to define, five callers to replace. Run `go test ./testing/...` to verify.

**Item 3 (Extract `applyDeathOverride`):** Executable immediately. Run `go test ./tactical/combat/...` before and after. The existing tests in `combatexecution_test.go` and `combat_test.go` cover the execution path. What the engineer needs to know that the document does not say: `applyDeathOverride` needs to take `result *combattypes.CombatResult`, `manager *common.EntityManager`, `defenderID ecs.EntityID`, `defenderSquadID ecs.EntityID`, `event *combattypes.AttackEvent`, and return nothing (it mutates in place). The comment at lines 86-87 must move into the extracted function's doc comment.

**Item 4 (Delete Leader Abilities):** Requires answering questions the document does not answer:
- Does `squadmanager.go` need updating when `AbilitySlotComponent` is removed from registration? Yes — it is registered at `squadmanager.go:37`. The registration line must go.
- Does `squadcreation.go`'s `AddLeaderComponents` (which adds `AbilitySlotComponent`) need updating? Yes.
- Does `squadcreation.go`'s `RemoveLeaderComponents` (which removes `AbilitySlotComponent`) need updating? Yes.
- Does `squads_test.go:112` (which adds leader components in tests) need updating? Yes.
- Is there a test for `CheckAndTriggerAbilities` that will break? No — `combatabilities.go` has no associated test file.
- Does the evaluation test at `mind/evaluation/evaluation_test.go` test `calculateAbilityValue`? Needs checking before deletion.

An engineer executing item 4 without this checklist will discover each of these in compilation errors and fix them reactively, which works — but it will take longer than "medium" implies because there is no single obvious deletion boundary.

---

## Recommended Next Action

**Act on it, with two pre-conditions:**

1. Before executing HIGH item 4 (Leader Abilities deletion), document the full deletion checklist in the ticket/commit message: `combatabilities.go` (entire file), `squadabilities.go` (entire file), `squadcomponents.go:185-289` (component definitions), `squadmanager.go` (component registration), `squadcreation.go` (AddLeaderComponents/RemoveLeaderComponents cleanup), `squad_chunk.go:275-291, 417-434` (save/load), `power.go:104, 113-134` (`calculateAbilityValue` and its call), `roles.go:44+` (`GetAbilityPowerValue`), `turnmanager.go:70, 105` (two call sites), `combatactionsystem.go:205, 209` (two call sites), `squads_test.go:112` (test fixture cleanup). Bump `savesystem.go:CurrentSaveVersion` to 2. That is the complete list. The document gives about 60% of it.

2. Before executing MEDIUM item 5 (bitmask refactor of cover calculation), note that `tactical/combat/combatmath` has no test file. Write at minimum two table-driven tests covering `GetCoverProvidersFor` and `CalculateCoverBreakdown` before touching the implementation. The document says "damage-calculation test coverage required" without flagging that this coverage does not currently exist.

The rest of the action list is executable as described. The SKIP list is correctly populated — no legitimate high-complexity item is being abandoned, and no low-complexity item is being inflated into work. The Anti-Recommendations section is particularly solid and should be treated as binding.

The document is worth acting on. The HIGH list represents roughly 6-8 hours of real work, not 4, but the work is correctly identified.
