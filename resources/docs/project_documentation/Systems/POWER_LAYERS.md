# Power Layers — Cross-Reference

**Last Updated:** 2026-05-17
**Scope:** `tactical/powers/{artifacts,perks,effects,spells,powercore,progression}`
**Related:**
[ARTIFACT_SYSTEM.md](ARTIFACT_SYSTEM.md), [PERK_SYSTEM.md](PERK_SYSTEM.md), [SPELLS_DOCUMENTATION.md](SPELLS_DOCUMENTATION.md), [PROGRESSION.md](PROGRESSION.md)

---

## Purpose

The combat-side power systems are split across four packages that grew at
different times and adopted distinct vocabularies (`Behavior`, `Hook`,
`Effect`) for what is conceptually one layer — "things that fire when
something happens in combat." This page is a single-stop cross-reference so a
contributor reading any one package can locate the analogue in the others
without re-learning the dispatch model each time.

It is **not** a how-to for adding new power content; see the per-system docs
linked above for that.

---

## The Four Layers

| Layer | Trigger | Cost / Resource | Per-Combat State |
|---|---|---|---|
| **Perks** | Automatic — fires on combat events (attack, move, turn) | None — equipped passively | `PerkRoundState` |
| **Artifacts** | Passive stat mods + optional player-triggered activations | Limited charges per battle / round | `ArtifactChargeTracker` |
| **Spells** | Explicit player cast action | Mana pool, deducted per cast | None — mana is the squad-level resource |
| **Effects** | Applied by other layers (spell, artifact) | None — effects ARE the result | `ActiveEffectsData` on each unit |

`Effects` is the substrate the other three sit on top of: it owns the
"timed stat modifier on a unit" component (`ActiveEffectsData`), the apply /
tick / remove API, and the `EffectSource` enum that lets callers filter by
origin.

---

## Vocabulary Comparison

The three "dispatched on combat events" layers each have a separate name for
the same three concepts. The differences are historical, not semantic.

| Concept | Artifacts | Perks | Effects |
|---|---|---|---|
| Per-item logic interface | `ArtifactBehavior` | `PerkBehavior` | *(none — `ApplyEffect` is the only entry point)* |
| Dispatcher type | `ArtifactDispatcher` | `SquadPerkDispatcher` | *(none — caller invokes `ApplyEffect`/`TickEffects` directly)* |
| Per-dispatch context | `BehaviorContext` (embeds `*PowerContext`) | `HookContext` (embeds `PowerContext` by value) | *(none — functions take `*EntityManager` directly)* |
| Registration | `RegisterArtifactBehavior` in `init()` | `RegisterPerkBehavior` in `init()` | *(no registry — effects are anonymous values)* |
| Validation at startup | `ValidateBehaviorCoverage` (called from `setup/gamesetup/bootstrap.go:41`) | `ValidateHookCoverage` (defined but currently uncalled — see `tech_debt_perks.md` D3) | *(none — stat names are validated lazily by `ParseStatType`)* |

`Behaviors` (artifacts) and `Hooks` (perks) are the same idea: a typed
struct registered against an ID, with methods invoked by the dispatcher at
known lifecycle points. Effects has no dispatcher because effects are pure
data — they are produced by perks/artifacts/spells and tick down on their
own.

---

## Shared Foundation: `powercore`

All three dispatched layers share the `tactical/powers/powercore` package:

| Type | Purpose | Defined in |
|---|---|---|
| `PowerContext` | Bundled deps (manager, cache, round number, logger) — embedded by `BehaviorContext` and `HookContext` | `tactical/powers/powercore/context.go` |
| `PowerLogger` interface | Single seam for activation / warning logs across all power layers | `tactical/powers/powercore/logger.go:13` |
| `LoggerFunc` adapter | Function-to-interface adapter so combat can pass a closure | `tactical/powers/powercore/logger.go:17` |
| `PowerPipeline` | Ordered event subscriber list owned by `CombatService` (PostReset, AttackComplete, TurnEnd, MoveComplete) | `tactical/powers/powercore/pipeline.go` |

The same `PowerLogger` instance flows through `artifacts.ArtifactDispatcher.SetLogger`
and `perks.SquadPerkDispatcher.SetLogger`, both wired by
`PowerOrchestrator.InstallLogger` (`tactical/combat/combatservices/power_orchestrator.go`).
Source-tag prefixes (`[GEAR]`, `[PERK]`, `[COMBAT]`) are assigned by
`classifyLogPrefix` in
`tactical/combat/combatservices/combat_power_dispatch.go`.

Spell and effect narration uses stdlib `log.Printf("[SPELL] ...")` /
`log.Printf("[EFFECT] ...")` directly — the developer-console stream is the
only consumer today, so threading them through `PowerLogger` was overhead
without a payoff. If a GUI panel ever needs to capture them, add a
`PowerLogger` argument to `ExecuteSpellCast`, `effects.ApplyStatModifiers`,
and `effects.TickEffectsForUnits` at the three entry points (the CombatService
caller already has the logger).

### Design note: spells bypass the hook chain

Spells deliberately do **not** flow through `PowerContext`'s artifact/perk
hook chain. Spells operate in a separate mana-gated layer. The rationale is
written on `ExecuteSpellCast` in `tactical/powers/spells/system.go` (search
for "Design decision: spells intentionally bypass perk hooks"). The
`PowerContext` doc comment cross-references it so anyone tempted to add an
`OnSpellCast` hook at that layer reads the rationale first.

---

## Library Pattern (Progression)

`tactical/powers/progression/library.go` is the project's reference example
for **two near-identical, ID-driven feature axes**. Perks-vs-spells
unlocking, currency tracking, and validation collapse into one
`library` struct with function-pointer accessors:

```go
type library struct {
    unlocked     func(*ProgressionData) *[]string
    currency     func(*ProgressionData) *int
    currencyName string
}

var (
    perkLib  = library{ ... UnlockedPerkIDs  / SkillPoints  / "skill" }
    spellLib = library{ ... UnlockedSpellIDs / ArcanaPoints / "arcana" }
)
```

`IsPerkUnlocked` / `IsSpellUnlocked` / `UnlockPerk` / `UnlockSpell` are thin
typed wrappers over shared `lib.isUnlocked` / `lib.unlock` methods. The
result: one body of unlock-and-spend logic, parameterised by axis.

Apply the same shape to any new "registry of X" with two parallel loaders.
Candidates today (per the cross-cutting tech-debt notes):

- **Validation paths** — perks have `ValidateHookCoverage`, artifacts have
  `ValidateBehaviorCoverage`, spells have none. A future
  `ValidateRegistryCoverage(perksLib | artifactsLib | spellsLib)` would fold
  these into the same pattern.
- **GUI selection panels** — the spell and artifact panels share a builder
  (`gui/builders/selection_panel.go`) and a `SelectionPanelCore` struct
  (`gui/widgets/selection_panel_core.go`); the original copy-paste was the
  motivating example.

---

## Effect Application — One Helper, Two Callers

`effects.ApplyStatModifiers(unitIDs, name, mods, source, duration, manager)`
is the single entry point for "iterate stat modifiers → parse → build
`ActiveEffect` → apply to units → log on error." Both spell buff/debuff
casts and artifact stat effects use it:

| Caller | File | Source | Duration |
|---|---|---|---|
| `applyBuffDebuffSpell` | `tactical/powers/spells/system.go` | `SourceSpell` | `spell.Duration` |
| `ApplyArtifactStatEffects` | `tactical/powers/artifacts/system.go` | `SourceItem` | `-1` (permanent) |

Future power layers (abilities, consumables, environmental hazards — see
`resources/docs/To_Review/Features_To_Add/environmental_hazards.md`) should
reuse the same helper rather than re-implementing the loop.

---

## Quick Navigation

| Want to ... | Go to ... |
|---|---|
| Add a new perk | [PERK_SYSTEM.md §7](PERK_SYSTEM.md) |
| Add a new artifact | [ARTIFACT_SYSTEM.md](ARTIFACT_SYSTEM.md) |
| Add a new spell | [SPELLS_DOCUMENTATION.md](SPELLS_DOCUMENTATION.md) |
| Wire commander progression | [PROGRESSION.md](PROGRESSION.md) |
| Apply a timed stat modifier | `effects.ApplyStatModifiers` or `effects.ApplyEffectToUnits` |
| Capture combat log output | Set a `powercore.PowerLogger` on the orchestrator (`PowerOrchestrator.InstallLogger`) |
