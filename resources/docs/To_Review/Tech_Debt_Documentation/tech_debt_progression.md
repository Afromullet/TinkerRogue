# Progression System — Technical Debt Report

**Date:** 2026-05-16
**Scope reviewed:** `tactical/powers/progression/` (3 files + 1 test), `gui/guiprogression/` (4 files), and the integration sites that read/write progression state (`mind/combatlifecycle/reward.go`, `tactical/powers/spells/system.go`, `tactical/commander/init.go`, `gui/guisquads/squadeditor_perks.go`, `setup/savesystem/chunks/progression_chunk.go`).

**Headline:** the *core* progression library is small (~140 LOC), well-factored, and already deduplicates perks/spells through the `library` value pattern. The debt is almost entirely **around** it, not in it — stale documentation, doc/code identity drift (Player vs. Commander), missing integration tests, and one duplicated formatter. Total remediation is small (one focused day) for outsized clarity wins.

---

## 1. Debt Inventory

### D1 — Critical: Documentation/code drift (Player vs. Commander scope)
**Type:** Documentation debt, with risk of architectural confusion.

The doc `resources/docs/project_documentation/Systems/PROGRESSION.md` still describes a *Player-entity-scoped* progression model, but the code was migrated to *Commander-scoped*:
- `tactical/commander/system.go:46` attaches `ProgressionComponent` to each commander entity.
- `progression_chunk.go:23` documents the migration: *"Format bumped to v2 when progression moved from Player-scoped to Commander-scoped."*
- Yet `PROGRESSION.md:104`, `:276`, `:295`, `:335`, `:394` still talk about *Player entity* / `playerinit.go`.
- `ARCHITECTURE_LAYERS.md:128` and `ENTITY_REFERENCE.md:203,908` both still list **Player** as the owner.

Also fictional in the docs:
- `tactical/powers/progression/defaults.go` — does **not** exist.
- `NewProgressionData()`, `StartingUnlockedPerks()`, `StartingUnlockedSpells()` — referenced repeatedly in PROGRESSION.md but **none** exist in code. Starters come from `templates.GameConfig.Commander.StartingPerks/StartingSpells` via `commander.SeedStarters` (`commander/init.go:42`).

**Risk:** High. Any new contributor reading `PROGRESSION.md` will write code against a model that no longer exists. The dispatch in `reward.go:78-96` already chose Commander-scoping; future work will silently diverge.

**Effort to fix:** ~1 hour. Rewrite the affected sections; delete the `defaults.go` section; update the entity-reference table.

---

### D2 — High: Duplicated perk-detail formatter
**Type:** Code duplication.

Two near-identical functions render the same perk fields:

| Function | File | Renders |
|---|---|---|
| `formatPerkDetail` | `gui/guisquads/squadeditor_perks.go:194-228` | Name, Tier, Category, Roles, Description, ExclusiveWith |
| `formatPerkBody` | `gui/guiprogression/progression_controller.go:215-228` | Tier, Category, Roles, Description |

The progression-side version is a strict subset of the squad-editor version. When the perk schema changes (e.g., add `Synergy` or `Conflicts`), both have to be updated, in lockstep, in two packages.

**Risk:** Medium. Low blast radius right now (only string output), but the file-rot rate is real — `def.ExclusiveWith` already only appears in one of the two.

**Fix:** Move a single `perks.FormatPerkDetail(def, opts)` into the `perks` package (closer to the data it describes) and have both UIs call it. Same applies to spell detail if you want to be thorough — `formatSpellBody` could live in `templates`.

**Effort:** ~30 min.

---

### D3 — High: No tests for integration sites
**Type:** Test coverage gap.

`library_test.go` covers the core API well (idempotency, insufficient points, add-points sign guard). What is **not** tested:

| Surface | Risk if it breaks |
|---|---|
| `mind/combatlifecycle/reward.go:78-96` (Grant routes points to first squad's commander) | Silent — players get no points from combat |
| `mind/combatlifecycle/reward.go:165` (`grantProgressionPoints` nil-guard skips if no `ProgressionComponent`) | Silent — enemy commanders defined later may swallow rewards |
| `tactical/powers/spells/system.go:64-83` (`filterSpellsByCommanderLibrary`) | Spell loadouts silently shrink to empty if data shape changes |
| `setup/savesystem/chunks/progression_chunk.go` (round-trip save/load + ID remap) | Save corruption / lost unlocks across sessions |
| `commander.SeedStarters` after `CreateCommander` | New commanders ship with no starters |

`grep` found zero references to any of those symbols in `_test.go` files outside `library_test.go`.

**Risk:** High. The most player-visible bug class — *"my unlocks vanished after save/load"* or *"my commander earned no skill points"* — is exactly what's untested.

**Effort:** ~3 hours. One table-driven test per integration site is sufficient; the existing `testfx.NewTestEntityManager` already does the heavy lifting (see `library_test.go:13`).

---

### D4 — Medium: "Library only routes points to first squad" is an unwritten invariant
**Type:** Hidden assumption.

`reward.go:78-83`:
```go
// Progression points go to the commander of the first squad in the target.
// An encounter is owned by one commander, so all squads in SquadIDs share
// a commander; resolving from the first is sufficient.
commanderID := ecs.EntityID(0)
if len(target.SquadIDs) > 0 {
    commanderID = commander.FindCommanderForSquad(target.SquadIDs[0], manager)
}
```

The comment explains the assumption but nothing **enforces** it. The day someone adds a multi-commander encounter (raids? events?), the second commander silently earns nothing. The same `GrantTarget` cleanly distributes **gold** across one player, and **XP/mana** across all squads, but progression is sneakily single-commander.

**Risk:** Medium. Easy to miss in code review, hard to notice in play-testing because the symptom is *"some progression feels slow"*.

**Fix options:** (a) iterate `target.SquadIDs`, dedupe commanders, split rewards; (b) make the invariant explicit by adding `target.CommanderIDs []ecs.EntityID` and removing the *"derive from first squad"* hack; (c) at minimum, log/`panic` in debug mode if `SquadIDs` resolve to multiple commanders.

**Effort:** ~1 hour for (b).

---

### D5 — Medium: `PerkSlotData` comment lies
**Type:** Stale comment.

`tactical/powers/perks/components.go:8-10`:
```go
// PerkSlotData stores equipped perks on a squad entity.
// Number of available slots scales with squad progression.
```

This is incorrect — per the design decisions captured in memory (`project_progression_system.md` decision #1), **max perk slots stays at 3** and never scales. The squad editor uses `perks.MaxPerkSlots` (`squadeditor_perks.go:131,166`) as a constant. The comment will mislead anyone designing a *"+1 perk slot"* perk or item.

**Risk:** Low–Medium. Cheap to fix, expensive to discover.

**Effort:** 1 minute.

---

### D6 — Low: `library.unlocked` returns `*[]string`
**Type:** Mildly unusual API.

`library.go:25-28`:
```go
type library struct {
    unlocked     func(*ProgressionData) *[]string
    currency     func(*ProgressionData) *int
    ...
}
```

Returning pointers to slice headers (rather than letting callers `append` to a slice and re-assign) is the only mechanism that makes `lib.unlock` write back to `ProgressionData` through a generic abstraction. It works, but: (a) every caller within the file must remember the dereference (`*list = append(*list, …)`); (b) the API leaks "this slice lives in a struct, mutate in place" semantics that would be a code-smell in a public API. It's fine as a package-internal helper, but tag it as such — a future contributor moving anything *out* of this package will trip over it.

**Risk:** Low.

**Fix:** No code change needed; add one comment near `library` explaining *why* the pointers exist (it's the cost of avoiding a `library[T]` generic on Go's current API surface).

**Effort:** 5 minutes.

---

### D7 — Low: Reward type grows with every new currency
**Type:** Premature warning, not yet debt.

`Reward` (`reward.go:28-34`) is a flat struct: every new currency requires editing `Reward`, `Scale`, the two `if r.XPts > 0 { ... }` blocks in `Grant`, and `grantProgressionPoints` becomes `grantXPoints`. Today there are 5 fields — fine. At 8+, this becomes a god struct. The same `library` deduplication used for unlocks could deduplicate granting (`type currency struct { name string; add func(...) }`).

**Risk:** Low today; pre-emptive flag.

**Fix:** Don't act now. Note in `CLAUDE.md` or `PROGRESSION.md` that *adding a 4th currency* should trigger the refactor.

---

### D8 — Low: Debug grant buttons live in production UI
**Type:** Mild infrastructure debt.

`progression_panels_registry.go:90-97` adds **"+1 Skill"** and **"+1 Arcana"** buttons unconditionally. There's no `if config.DEBUG_MODE { ... }` gate. Player-facing builds will ship with infinite-currency buttons.

**Risk:** Medium *only when shipping a build*; Low while in development.

**Fix:** Wrap both `result.Container.AddChild` calls in `if config.DEBUG_MODE`. Matches the gating pattern noted in `MEMORY.md` (`config.DEBUG_MODE` for test fixtures).

**Effort:** 5 minutes.

---

## 2. Impact Summary

| Item | Player-visible failure mode | Likelihood | Effort |
|---|---|---|---|
| D1 docs | New contributor builds against fictional API | High | 1h |
| D2 dup formatter | Two-place edits, eventually inconsistent UIs | Medium | 30m |
| D3 no integration tests | Lost unlocks on save/load; missing points | High | 3h |
| D4 first-squad routing | Multi-commander encounter gives zero points | Low | 1h |
| D5 stale comment | Designs a non-existent slot-scaling system | Low | 1m |
| D6 pointer slices | Friction for future maintainers | Low | 5m |
| D7 Reward growth | Boilerplate when adding 4th currency | Future | — |
| D8 debug buttons | Ships infinite currency in release | High (on ship) | 5m |

**Total focused effort to clear D1–D6, D8: ~5 hours** (well under one day).

---

## 3. Prioritized Roadmap

### Quick wins (this sprint — ~1 hour total)
1. **D5** — Fix the `PerkSlotData` comment (1m).
2. **D8** — Gate the debug grant buttons behind `config.DEBUG_MODE` (5m).
3. **D6** — Add a one-line `// Pointer return is required so unlock() can append in place.` to `library.unlocked` (5m).
4. **D1** — Rewrite the affected sections of `PROGRESSION.md`, `ARCHITECTURE_LAYERS.md:128`, and `ENTITY_REFERENCE.md:203,908` to say *Commander entity*. Delete every reference to `defaults.go`, `NewProgressionData`, `StartingUnlockedPerks`, `StartingUnlockedSpells`; replace with `commander.SeedStarters` + `templates.GameConfig.Commander.StartingPerks/StartingSpells` (~1h).

### Short-term (next sprint — ~4 hours)
5. **D2** — Extract `perks.FormatPerkDetail` and `templates.FormatSpellDetail`; have both UI sites call them (30m).
6. **D3** — Add four integration tests:
   - `reward_test.go`: Grant routes ArcanaPts/SkillPts to the squad's commander; no-op when squad has no commander or commander has no `ProgressionData`.
   - `spells_filter_test.go`: `filterSpellsByCommanderLibrary` returns input unchanged when no commander, intersects with library otherwise.
   - `progression_chunk_test.go`: Save → Load → RemapIDs round-trip preserves currencies and slices.
   - `commander_test.go`: `CreateCommander` + `SeedStarters` populates starters from `GameConfig`.
   - All can reuse the `testfx.NewTestEntityManager` pattern from `library_test.go:13` (3h).

### Medium-term (when D4 surfaces or a multi-commander encounter is designed)
7. **D4** — Either iterate-and-dedupe `target.SquadIDs → commander`, or add `GrantTarget.CommanderIDs` explicitly (1h).

### Watch list (no action yet)
8. **D7** — Re-evaluate `Reward` if a 4th currency is proposed.

---

## 4. Prevention

- Add a short *"if you change the progression scope (Player vs. Commander), update these doc files"* note at the top of `progression/components.go` to keep docs and code aligned next time.
- Make D3's integration tests a CI gate going forward — they take milliseconds and cover the silent-failure surfaces.
- When adding a new exported function to `tactical/powers/progression/`, require it appears either in `library.go` (so the `library` factoring stays the canonical place to add things) or in a new file with a clear reason — prevents back-sliding to the per-currency parallel code paths the current design already eliminated.

---

## 5. What's already healthy (don't touch)

- The `library` value pattern in `library.go:24-41` is exactly the right factoring — collapsed two parallel code paths (perks vs. spells) into one. Don't generify it.
- `librarySource` in `progression_controller.go:29-39` mirrors the same pattern on the UI side and drives both panels from one controller class. Same shape, same wins.
- The `errors.Is(err, ErrNotEnoughPoints)` sentinel + `fmt.Errorf("%w: ...", ...)` wrapping in `library.go:79` is the idiomatic Go pattern and is correctly tested at `library_test.go:70`.
- `ProgressionData` is pure data (no methods); the test seeds it directly via component `AddComponent` — matches the project's ECS guidelines exactly.

The core package is one of the cleaner small subsystems in the repo. The debt is at its **boundaries** (docs, integration tests, the perk-detail formatter shared with the squad editor) rather than inside it.
