# Powers Subsystem Tech Debt — Spells, Effects, Powercore, Progression, GUI, and Cross-Cutting

**Date:** 2026-05-16
**Scope:** `tactical/powers/{spells,effects,powercore,progression}` + `gui/{guispells,guiartifacts,guiprogression}` + cross-cutting themes across all `tactical/powers/*`.
**Reads in conjunction with:** `tech_debt_perks.md`, `tech_debt_artifacts.md` (this doc references but does not duplicate them).

---

## Verdict at a Glance

The newest power subsystems — spells (added alongside progression) and the GUI panels for spells/artifacts/progression — are the **least covered** part of the powers ecosystem, both in tests and in remediated debt. The biggest issues are:

1. **Testing gaps**: spells has zero tests; all three power-related GUI panels have zero tests.
2. **Logging inconsistency**: `fmt.Printf` in 6 power files bypasses the `powercore.PowerLogger` interface introduced to centralize this.
3. **Duplicated effect-application** between spells and artifacts (two functions, near-identical body).
4. **GUI panel copy-paste** between `spell_panel.go` and `artifact_panel.go`, despite `guiprogression` already having the right factory pattern.

The good news: `progression/library.go` is an exemplary deduplication pattern that should be applied elsewhere, `powercore.PowerContext` is shaped correctly, and the `effects` package itself is well-tested.

---

## 1. Debt Inventory

### A. Testing Debt

#### A1. Spells package has zero tests (**HIGH**)
- Location: `tactical/powers/spells/` — 4 files (`components.go`, `init.go`, `queries.go`, `system.go`), 0 `*_test.go` files.
- Untested code paths:
  - `ExecuteSpellCast` (system.go:95–148): mana validation, spellbook gating, error branches, mana deduction
  - `applyDamageSpell` (system.go:151–203): damage formula (`spell.Damage − magicDefense`, min 1), squad destruction, map removal
  - `applyBuffDebuffSpell` (system.go:206–240): stat parsing, effect construction, application
  - `filterSpellsByCommanderLibrary` (system.go:67–83): progression-gated spell access
- **Impact:** spells are the only major power subsystem without test coverage. Perks have ~1021 LOC of tests; artifacts have 5 test files. Mana, damage, and unlock-gating regressions ship blind.

#### A2. Three power GUI panels have zero tests (**MEDIUM**)
- Location: `gui/guispells/`, `gui/guiartifacts/`, `gui/guiprogression/` — 0 `*_test.go` in each.
- Uncovered: panel construct/show/hide/toggle, item selection, mode toggling, refresh-after-cast, validation guards (empty spellbook, insufficient mana, no equipped artifacts).
- **Impact:** the ebitenui input-layer gotcha documented in `MEMORY.md` (hidden containers still create input layers) is exactly the class of bug a smoke test catches. Combat-mode panel interactions are the only place this manifests in production.

#### A3. No test for commander-library gating in spell casting (**LOW-MEDIUM**)
- `spells/system.go:67–83` intersects spell IDs with the commander's unlocked spells; spells not unlocked are silently filtered out of the spellbook.
- **Impact:** a regression in `progression.IsSpellUnlocked` would silently grant all spells to all commanders, with no test failing.

### B. Logging Inconsistency (Cross-Cutting)

#### B1. `fmt.Printf` bypasses `PowerLogger` in 6 power files (**MEDIUM**)
- Sites verified:
  - `tactical/powers/spells/system.go:196, 201, 218, 238` — spell warnings and confirmations
  - `tactical/powers/effects/system.go:42, 92` — effect-application warnings
  - `tactical/powers/artifacts/system.go:186` (invalid stat modifier) and `artifacts/balanceconfig.go` (3 config-load sites — also flagged as A8 in `tech_debt_artifacts.md`)
  - `tactical/powers/perks/balanceconfig.go` and `perks/registry.go` (10+ sites)
- **Why it's debt:** `powercore.PowerLogger` was introduced as the unified log sink; it's plumbed through `PowerContext` and used in behavior dispatch. Config-loading and effect-application paths predate it and were never migrated.
- **Impact:** logs are unsuppressable (noisy test runs), uncapturable (GUI cannot route to its console), and ungreppable (no structured fields/levels).

### C. Code Duplication

#### C1. Spell buff/debuff and artifact stat-effect application are near-identical (**MEDIUM**)
- `tactical/powers/spells/system.go:206–240` (`applyBuffDebuffSpell`) and `tactical/powers/artifacts/system.go:170–200` (`ApplyArtifactStatEffects`) share the exact shape:
  1. Iterate squads → `squadcore.GetUnitIDsInSquad`
  2. For each stat modifier: `effects.ParseStatType` → log on error → build `effects.ActiveEffect` → `effects.ApplyEffectToUnits`
- Differences: `Source` (`SourceSpell` vs `SourceItem`), `RemainingTurns` (`spell.Duration` vs `-1`), and the log message phrasing.
- **Fix:** extract `effects.ApplyStatModifiers(unitIDs, name string, mods []templates.StatModifier, source EffectSource, duration int, logger PowerLogger, manager *common.EntityManager)` and call it from both sites.
- **Impact:** any future change to effect application (new stat type, stack-policy field, logger migration) must be applied in two places. The next power layer (abilities? consumables?) will likely produce a third near-copy.

#### C2. Spell and artifact GUI panels are copy-paste siblings (**MEDIUM**)
- `gui/guispells/spell_panel.go` (162 LOC) and `gui/guiartifacts/artifact_panel.go` (146 LOC) share the same lifecycle: `SetWidgets`, `OnItemSelected`, `UpdateDetailPanel`, `Refresh`, `Show`, `Hide`, `Toggle`; identical Deps-injection; identical list → detail → button layout.
- **Reference (the correct pattern):** `gui/guiprogression/progression_panels_registry.go:108–181` already uses a `buildLibraryPanel(librarySource, ...)` factory to share construction across perk- and spell-library panels.
- **Fix:** extract a shared selection-panel builder in `gui/builders/` (or `gui/widgetresources/`) modeled on `buildLibraryPanel`. Migrate the two combat panels.
- **Impact:** lifecycle changes (focus handling, refresh-after-cast, error display) require synchronized edits across two files. Drift is already visible in small naming differences.

#### C3. Spell damage calculation is reimplemented inline (**LOW**)
- `spells/system.go:172–177`: `damage := spell.Damage − magicDefense; if damage < 1 { damage = 1 }`. The combat package has its own damage pipeline; magic defense, resistances, and damage floors live there.
- **Impact:** the design note at `spells/system.go:90–94` already calls out future work to add `OnSpellCast` perk hooks. That work will fork the damage calc further unless this is consolidated first.

### D. Architecture / Design Debt

#### D1. `PowerContext` couples power subsystems to combat (**MEDIUM**)
- `tactical/powers/powercore/context.go:22` embeds `*combatstate.CombatQueryCache` directly. Every artifact behavior and perk hook receives the cache.
- **Tension:** ECS guidance per `CLAUDE.md` favors query-based access over cached relationships. The cache exists because squad lookups during dispatch are hot — that's a legitimate optimization, but it now binds every power consumer to combat.
- **Impact:** spells, effects, and any future ability system can't reuse `PowerContext` outside combat. Test fixtures need a fully-wired `CombatQueryCache` (see `artifacts/gear_test.go` setup, called out as D1 in `tech_debt_artifacts.md`).
- **Options (don't act on this today):**
  1. Inject the cache via a narrower `SquadLocator` interface so non-combat callers can pass a query-based stub.
  2. Split `CombatPowerContext` from base `PowerContext`.
  - Treat as architectural debt to revisit when the second non-combat consumer materializes.

#### D2. Two `EffectSource` values are declared but never instantiated (**LOW**)
- `tactical/powers/effects/components.go:25–32`: enum defines `SourceSpell, SourceAbility, SourcePerk, SourceItem`. Only `SourceSpell` and `SourceItem` are ever set in production code.
- **Impact:** misleading API surface. A new contributor might tag a perk-driven effect with `SourcePerk` thinking it does something. Either remove them or add a `// reserved for future ability/perk effects` comment.

#### D3. Spell-vs-perk interaction is intentionally absent — but the design note is hidden (**LOW**)
- `spells/system.go:88–94` explains that spells bypass the perk hook chain by design (no `OnSpellCast`); the comment cites the four-layer power system rationale.
- **Impact:** the next contributor who reads `powercore.PowerContext` won't see this; the comment is on `ExecuteSpellCast`. Replicate the cross-reference on `PowerContext`'s doc-comment so the design intent is discoverable at the layer that would naturally host the missing hook.

#### D4. Behavior-vs-Hook vocabulary drift across the three power subsystems (**LOW**)
- Artifacts: `Behaviors` + `Dispatcher` + `BehaviorContext`.
- Perks: `Hooks` + `Dispatcher` + `HookContext`.
- Effects: `ApplyEffect*` functions, no dispatcher.
- All three sit on top of `PowerContext` but use distinct vocabulary for what is conceptually one layer ("things that fire on combat events").
- **Impact:** newcomers must learn three abstractions for one conceptual area. Not a refactor priority, but a top-level `docs/project_documentation/Systems/POWER_LAYERS.md` cross-referencing the three vocabularies would lower the on-ramp cost.

### E. Cross-Cutting Pattern Worth Standardizing

#### E1. Apply the `progression/library.go` pattern to other parallel paths (**REF**)
- `tactical/powers/progression/library.go:24–40` cleanly deduplicates the perk-vs-spell parallel paths via a `library` struct value with function-pointer accessors (`unlocked`, `currency`, `currencyName`). Result: one implementation of unlock/check/spend, parameterized by axis.
- The same shape applies to:
  - **Registry loading** — perks and spells (and artifact templates) each load JSON, validate, and register. Validation paths drift between them (perks have `ValidateHookCoverage`, artifacts have `ValidateBehaviorCoverage`, spells have neither).
  - **GUI selection panels** — see C2.
- Treat `library.go` as the project's reference example for "two parallel ID-driven feature axes."

---

## 2. Impact Assessment

| # | Item | Severity | Practical impact |
|---|---|---|---|
| A1 | Zero spell tests | **High** | Spell mechanics ship blind; damage/mana/destruction regressions invisible |
| A2 | Zero GUI panel tests | **Medium-High** | ebitenui input-layer class bugs (per MEMORY) ship undetected |
| B1 | `fmt.Printf` in 6 power files | **Medium** | Polluted logs, unsuppressable in tests, blocks GUI log capture |
| C1 | Effect-application duplication | **Medium** | Two places to update; next power consumer makes it three |
| C2 | Panel copy-paste | **Medium** | Synchronized maintenance, drift risk; progression already shows the fix |
| D1 | `PowerContext` combat coupling | **Medium** | Blocks non-combat reuse; complicates test setup |
| A3 | Library integration untested | **Low-Medium** | Silent unlock-gating regression risk |
| C3 | Spell damage calc inline | **Low** | Forks combat damage pipeline; future modifier work pays this off |
| D2 | Unused `EffectSource` values | **Low** | Misleading API surface |
| D3 | Spell-bypasses-perks doc hidden | **Low** | Design choice risks being silently undone |
| D4 | Vocabulary drift | **Low** | On-ramp friction, no behavioral risk |

---

## 3. Prioritized Roadmap

### Quick Wins (this week, ~1 day)

1. **[A1, ~2 hr]** Add `tactical/powers/spells/system_test.go`. Minimum coverage:
   - `ExecuteSpellCast` error branches: no active combat, unknown spell, no mana component, insufficient mana, spell not in book
   - `applyDamageSpell`: damage formula (subtracts magic defense, enforces min 1), squad destruction → map removal, multi-target accumulation in `result.TotalDamageDealt`
   - `applyBuffDebuffSpell`: stat-modifier parse error skipped, effect appears on each alive unit, mana deducted exactly once
2. **[B1 partial, ~30 min]** Replace `fmt.Printf` in `spells/system.go` (4 sites) and `effects/system.go` (2 sites) with `powercore.PowerLogger` calls. This demonstrates the migration pattern for the perks/artifacts config-load sites in B1 remainder.
3. **[D2, ~10 min]** Remove `SourceAbility` and `SourcePerk` from `effects/components.go:25–32` (or add a one-line `// reserved` doc comment if planned).
4. **[D3, ~5 min]** Mirror the "spells bypass perks" design note onto `powercore.PowerContext` doc-comment so the design intent surfaces at the layer that would host the missing hook.
5. **[A3, ~30 min]** Add `TestExecuteSpellCast_RespectsCommanderLibrary` — caster commander hasn't unlocked spell X, attempt to cast → `ErrorReason` set, no mana deducted.

### Medium-Term (1–2 weeks)

6. **[C1, ~2 hr]** Extract `effects.ApplyStatModifiers(unitIDs, name, mods, source, duration, logger, manager)` and migrate both call sites. Single source of truth for "iterate mods → parse → build ActiveEffect → apply → log on error."
7. **[C2, ~1 day]** Extract a shared selection-panel builder in `gui/builders/` modeled on `gui/guiprogression/progression_panels_registry.go:buildLibraryPanel`. Migrate `gui/guispells/spell_panel.go` and `gui/guiartifacts/artifact_panel.go` to use it. Verify with the A2 smoke tests below.
8. **[A2 partial, ~1 day]** Add smoke tests for spell and artifact panels: panel constructs without panic, `Toggle` alternates visibility, `Refresh` after cast does not leak input layers (per the ebitenui gotcha in MEMORY). Use the existing `testing/` fixtures.
9. **[B1 remainder, ~1 hr]** Migrate remaining `fmt.Printf` sites in `perks/balanceconfig.go`, `perks/registry.go`, `artifacts/balanceconfig.go`, `artifacts/system.go:186` to `PowerLogger` or a config-load logger sink (whichever fits — config loads run before any `PowerContext` exists, so they may need a package-level logger or stdlib `log/slog`).

### Long-Term (next quarter, only if scope grows)

10. **[D1]** Split `CombatPowerContext` from base `PowerContext` once a second non-combat consumer (e.g., exploration abilities, world-map powers) appears. Premature today.
11. **[C3]** Consolidate spell damage calculation with the combat damage pipeline when the first cross-cutting modifier (spell-affecting perk, "spell resistance" stat) arrives. Aligns with the `OnSpellCast` hook recommendation already documented in `spells/system.go:90–94`.
12. **[D4]** Add `docs/project_documentation/Systems/POWER_LAYERS.md` cross-referencing artifact `Behaviors`, perk `Hooks`, and effect application as one conceptual layer with three vocabularies. One page, one cross-reference table.

---

## 4. Recommended PR Sequence

```
PR 1 (Quick wins):     A1 + B1-partial + D2 + D3 + A3   — ~1 day, all isolated
PR 2 (Effect dedup):   C1                                — single helper, two call sites, no GUI impact
PR 3 (Panel factory):  C2 + A2-partial                   — GUI refactor with smoke-test coverage
PR 4 (Logging finish): B1-remainder                      — config-load paths, cosmetic but completes B1
```

Each is independently revertible. PR 1 alone closes the highest-impact testing gap (A1) and demonstrates the `PowerLogger` migration pattern that PR 4 finishes.

---

## 5. Prevention Plan

- **CI gate**: `go test ./tactical/powers/...` already runs in CI; once spells gain a test file (A1), the gate covers it automatically. No infrastructure work needed.
- **Lint rule**: ban `fmt.Print*` in `tactical/powers/**/*.go` via `golangci-lint`'s `forbidigo`. New code is forced through `PowerLogger`. Apply only after B1 is complete to avoid spurious failures.
- **Panel template**: once C2 lands, document the shared factory in `gui/builders/` as the required pattern for new selection panels. Add a one-line note to the GUI section in `CLAUDE.md`.
- **`library.go` as reference**: link `tactical/powers/progression/library.go` from any future "registry of X" feature design doc — it is the project's exemplar for parallel-axis deduplication.

---

## 6. What This Doc Does NOT Cover (and why)

- **Perks** — see `tech_debt_perks.md`. 52% test coverage, mature subsystem, 5 prioritized items already enumerated.
- **Artifacts** — see `tech_debt_artifacts.md`. 2 bugs + 5 test gaps shippable in ~4 hours via PR 1 there.
- **Templates package** — see `tech_debt_templates.md`. Spell definitions live in templates; spell-definition concerns belong in that doc.
- **Combat damage pipeline** — out of scope. Surfaces in C3 only as a future-coupling note.

---

## 7. Positive Signals

- `tactical/powers/progression/library.go:24–40`: the deduplication pattern is exemplary. Function-pointer accessors with two value instances eliminate parallel code paths.
- `tactical/powers/powercore/context.go`: `PowerContext` design is correct in shape — bundled deps, embedded in package-specific contexts, nil-safe logger documented at the constructor.
- `tactical/powers/effects/`: well-tested (`effects_test.go` at 312 LOC) and small (~200 LOC production).
- `artifacts.ValidateBehaviorCoverage` **is** called at startup from `setup/gamesetup/bootstrap.go:41` — startup validation pattern is in place. Worth replicating for perks (`ValidateHookCoverage` exists but is uncalled — see `tech_debt_perks.md` D3) and adding for spells.

---

## Verification

Findings cross-checked against the live repo at time of writing:

| Claim | Verified via |
|---|---|
| Spells has 0 test files | `Glob tactical/powers/spells/*_test.go` → empty |
| `guispells`/`guiartifacts`/`guiprogression` have 0 test files | `Glob gui/{guispells,guiartifacts,guiprogression}/*_test.go` → empty |
| `fmt.Printf` lives in 6 power files | `Grep fmt\.Printf` over `tactical/powers/` → 6 files |
| `ValidateBehaviorCoverage` IS called at startup | `Grep ValidateBehaviorCoverage` → `setup/gamesetup/bootstrap.go:41` |
| Effect-application duplication confirmed | Read `spells/system.go:206–240` and `artifacts/system.go:170–200` |
| `progression/library.go` factory pattern confirmed | Read `progression/library.go:24–40` |
| `PowerContext` embeds `CombatQueryCache` | Read `powercore/context.go:20–25` |
| GUI mode flags are in `framework/contextstate.go` (UI state — correct) | Read `gui/framework/contextstate.go:29–34, 57–64, 84–91` |

---

## Critical Files Referenced

- `tactical/powers/spells/system.go` (240 LOC, the entire spell pipeline)
- `tactical/powers/effects/system.go`, `effects/components.go`
- `tactical/powers/powercore/context.go`, `powercore/logger.go`
- `tactical/powers/progression/library.go` (positive reference pattern)
- `tactical/powers/artifacts/system.go:170–200` (duplication site)
- `gui/guispells/spell_panel.go`, `gui/guiartifacts/artifact_panel.go` (duplication site)
- `gui/guiprogression/progression_panels_registry.go:108–181` (positive reference pattern)
- `setup/gamesetup/bootstrap.go:41` (validation-at-startup pattern)
- `gui/framework/contextstate.go:29–34` (UI mode flag definitions)
