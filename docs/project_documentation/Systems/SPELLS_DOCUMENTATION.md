# TinkerRogue Spells System — Technical Documentation

**Last Updated:** 2026-03-25
**Packages Covered:** `tactical/powers/spells`, `tactical/powers/effects`, `tactical/powers/artifacts/effects`, `templates` (spell definitions), `gui/guispells`

---

## Table of Contents

1. [Executive Summary](#1-executive-summary)
2. [Package Structure and Relationships](#2-package-structure-and-relationships)
3. [Data Structures and Components](#3-data-structures-and-components)
   - 3.1 [Spell Definitions (`templates`)](#31-spell-definitions-templates)
   - 3.2 [Commander ECS Components (`spells`)](#32-commander-ecs-components-spells)
   - 3.3 [Effects ECS Components (`effects`)](#33-effects-ecs-components-effects)
4. [Spell Lifecycle: Definition to Resolution](#4-spell-lifecycle-definition-to-resolution)
   - 4.1 [Phase 1 — Data Loading at Startup](#41-phase-1--data-loading-at-startup)
   - 4.2 [Phase 2 — Commander Creation](#42-phase-2--commander-creation)
   - 4.3 [Phase 3 — GUI Spell Panel Interaction](#43-phase-3--gui-spell-panel-interaction)
   - 4.4 [Phase 4 — Targeting](#44-phase-4--targeting)
   - 4.5 [Phase 5 — Execution (`ExecuteSpellCast`)](#45-phase-5--execution-executespellcast)
   - 4.6 [Phase 6 — Effect Application](#46-phase-6--effect-application)
   - 4.7 [Phase 7 — Turn Ticking and Expiry](#47-phase-7--turn-ticking-and-expiry)
5. [The Effects System](#5-the-effects-system)
   - 5.1 [Design Philosophy](#51-design-philosophy)
   - 5.2 [StatType Enum](#52-stattype-enum)
   - 5.3 [EffectSource Enum](#53-effectsource-enum)
   - 5.4 [ActiveEffect Struct](#54-activeeffect-struct)
   - 5.5 [Applying, Ticking, and Removing Effects](#55-applying-ticking-and-removing-effects)
   - 5.6 [Permanent Effects (RemainingTurns = -1)](#56-permanent-effects-remainingturns---1)
6. [Integration with the ECS Architecture](#6-integration-with-the-ecs-architecture)
   - 6.1 [Subsystem Registration](#61-subsystem-registration)
   - 6.2 [Component Access Patterns Used](#62-component-access-patterns-used)
   - 6.3 [Entity Relationships](#63-entity-relationships)
7. [Key Functions Reference](#7-key-functions-reference)
   - 7.1 [spells Package](#71-spells-package)
   - 7.2 [effects Package](#72-effects-package)
   - 7.3 [templates Package (spell registry)](#73-templates-package-spell-registry)
8. [Spell Configuration and JSON Schema](#8-spell-configuration-and-json-schema)
   - 8.1 [File Location](#81-file-location)
   - 8.2 [SpellDefinition Fields](#82-spelldefinition-fields)
   - 8.3 [Target Shape Configurations](#83-target-shape-configurations)
   - 8.4 [Complete Spell Roster](#84-complete-spell-roster)
9. [Artifacts and the Shared Effects System](#9-artifacts-and-the-shared-effects-system)
   - 9.1 [Why Two `effects` Packages Exist](#91-why-two-effects-packages-exist)
   - 9.2 [Minor Artifact Effects vs Spell Effects](#92-minor-artifact-effects-vs-spell-effects)
   - 9.3 [Artifact Effect Application Timing](#93-artifact-effect-application-timing)
10. [GUI Layer Integration (`guispells`)](#10-gui-layer-integration-guispells)
    - 10.1 [SpellCastingHandler](#101-spellcastinghandler)
    - 10.2 [SpellPanelController](#102-spellpanelcontroller)
    - 10.3 [TacticalState Spell Fields](#103-tacticalstate-spell-fields)
    - 10.4 [AoE Shape Targeting Flow](#104-aoe-shape-targeting-flow)
11. [Visual Effects (VX) Integration](#11-visual-effects-vx-integration)
12. [Turn Management and Effect Ticking](#12-turn-management-and-effect-ticking)
13. [Mana Economy and Strategic Design](#13-mana-economy-and-strategic-design)
14. [Adding New Spells — Developer Guide](#14-adding-new-spells--developer-guide)
15. [Common Pitfalls and Gotchas](#15-common-pitfalls-and-gotchas)
16. [Appendix: Full Data Structures](#16-appendix-full-data-structures)

---

## 1. Executive Summary

The spell system in TinkerRogue enables commanders (player-controlled heroes on the overworld) to cast magical abilities during tactical combat. Spells are a **commander-level power**: only entities holding both a `ManaComponent` and a `SpellBookComponent` can cast them.

The system is split across three distinct layers:

- **Data layer** (`templates`): Static spell blueprints loaded from JSON. The global `SpellRegistry` map stores all `SpellDefinition` structs keyed by spell ID string.
- **Logic layer** (`tactical/powers/spells`, `tactical/powers/effects`): ECS components for mana and spellbooks, plus the execution functions that apply damage or stat modifiers to target squads.
- **GUI layer** (`gui/guispells`): The targeting workflow — the panel that lists spells, the AoE shape overlay that follows the cursor, and the click handler that fires the execution function.

The effects system (`tactical/powers/effects`) is shared infrastructure used by both spells and artifacts. When a buff or debuff spell is cast, it creates `ActiveEffect` structs that are attached to unit entities as an `ActiveEffectsData` component. The effect immediately modifies the unit's `Attributes` in-place, and the modification is reversed when the effect expires. This makes stat reads simple: callers always see the current (modified) value without needing to query active effects.

**Key architectural decision:** Spell casting is intentionally **limited to one spell per commander turn** (`HasCastSpell` flag in `TacticalState`). Mana persists across battles, making mana allocation a strategic overworld-level decision rather than a per-combat tactical resource.

---

## 2. Package Structure and Relationships

```
game_main/
├── assets/gamedata/
│   └── spelldata.json                  ← Spell blueprint definitions (JSON)
│
├── templates/
│   ├── spelldefinitions.go             ← SpellDefinition struct, SpellRegistry map,
│   │                                      LoadSpellDefinitions(), helper methods
│   └── registry.go                     ← ReadGameData() calls LoadSpellDefinitions()
│
├── tactical/powers/
│   ├── spells/
│   │   ├── components.go               ← ManaData, SpellBookData, SpellCastResult structs;
│   │   │                                  ManaComponent, SpellBookComponent ECS vars
│   │   ├── init.go                     ← RegisterSubsystem: creates ECS components+tags
│   │   ├── queries.go                  ← GetManaData, GetSpellBook, HasEnoughMana,
│   │   │                                  GetCastableSpells, HasSpellInBook, GetAllSpells
│   │   └── system.go                   ← ExecuteSpellCast (main entry point),
│   │                                      applyDamageSpell, applyBuffDebuffSpell
│   │
│   ├── effects/
│   │   ├── components.go               ← StatType, EffectSource, ActiveEffect,
│   │   │                                  ActiveEffectsData; ParseStatType()
│   │   ├── init.go                     ← RegisterSubsystem: creates ECS component+tag
│   │   ├── queries.go                  ← GetActiveEffects, HasActiveEffects
│   │   └── system.go                   ← ApplyEffect, ApplyEffectToUnits,
│   │                                      TickEffects, TickEffectsForUnits,
│   │                                      RemoveAllEffects
│   │
│   └── artifacts/
│       ├── effects/                    ← Duplicate of tactical/powers/effects/
│       │   └── (same API, separate package)
│       └── system.go                   ← ApplyArtifactStatEffects (uses effects package)
│
├── tactical/commander/
│   └── system.go                       ← CreateCommander() adds ManaComponent +
│                                          SpellBookComponent to new commanders
│
├── tactical/combat/combatcore/
│   └── turnmanager.go                  ← ResetSquadActions() calls effects.TickEffectsForUnits()
│
├── tactical/combat/combatservices/
│   └── combat_service.go               ← InitializeCombat() calls
│                                          artifacts.ApplyArtifactStatEffects()
│
└── gui/guispells/
    ├── spell_deps.go                   ← SpellCastingDeps (dependency injection struct)
    ├── spell_handler.go                ← SpellCastingHandler: SelectSpell, targeting,
    │                                      executeSpellOnTargets, VX triggering
    └── spell_panel.go                  ← SpellPanelController: list UI, detail panel,
                                           cast/cancel button callbacks
```

**Dependency flow (no cycles):**

```
gui/guispells
    → tactical/powers/spells
        → templates (SpellDefinition)
        → tactical/powers/effects
        → tactical/combat/combatcore (RemoveSquadFromMap)
        → tactical/squads/squadcore (GetUnitIDsInSquad)
    → templates
    → visual/graphics (TileBasedShape, VX)
    → world/coords (CoordManager)
```

---

## 3. Data Structures and Components

### 3.1 Spell Definitions (`templates`)

Spell definitions are **static blueprints** loaded once at startup and stored in a global registry. They are never modified at runtime.

**File:** `templates/spelldefinitions.go`

```go
type SpellDefinition struct {
    ID            string              // Unique key, e.g., "fireball"
    Name          string              // Display name, e.g., "Fireball"
    Description   string              // Flavor/tooltip text
    ManaCost      int                 // Mana deducted from the caster on cast
    Damage        int                 // Base damage for EffectDamage spells (0 for buffs/debuffs)
    TargetType    SpellTargetType     // "single" or "aoe"
    EffectType    SpellEffectType     // "damage", "buff", or "debuff"
    Shape         *JSONTargetArea     // AoE shape definition (nil for single-target)
    VXType        string              // Visual effect type: "fire", "electricity", "ice", "cloud"
    VXDuration    int                 // VX animation duration in frames
    Duration      int                 // Buff/debuff duration in turns (0 for damage spells)
    StatModifiers []SpellStatModifier // Stat changes for buff/debuff spells
}

type SpellStatModifier struct {
    Stat     string // "strength", "dexterity", "magic", etc.
    Modifier int    // Positive for buff, negative for debuff
}
```

**Target types:**

| Constant | JSON Value | Meaning |
|---|---|---|
| `TargetSingleSquad` | `"single"` | Targets one squad by clicking on it |
| `TargetAoETile` | `"aoe"` | Targets a tile area; all squads within the shape are hit |

**Effect types:**

| Constant | JSON Value | Behavior in `ExecuteSpellCast` |
|---|---|---|
| `EffectDamage` | `"damage"` | Calls `applyDamageSpell` — deals `Damage - MagicDefense` HP to each unit |
| `EffectBuff` | `"buff"` | Calls `applyBuffDebuffSpell` — creates `ActiveEffect` with positive modifiers |
| `EffectDebuff` | `"debuff"` | Calls `applyBuffDebuffSpell` — creates `ActiveEffect` with negative modifiers |

**Global registry:**

```go
var SpellRegistry = make(map[string]*SpellDefinition)

func GetSpellDefinition(id string) *SpellDefinition { return SpellRegistry[id] }
func GetAllSpellIDs() []string { ... }
```

Helper methods on `SpellDefinition` for cleaner conditional logic: `IsSingleTarget()`, `IsAoE()`, `IsDamage()`, `IsBuff()`, `IsDebuff()`.

---

### 3.2 Commander ECS Components (`spells`)

**File:** `tactical/powers/spells/components.go`

Commanders (the player's hero units) carry two ECS components from the spells package:

```go
// ManaData tracks a commander's mana pool.
// CurrentMana is deducted on cast; persists across battles.
type ManaData struct {
    CurrentMana int
    MaxMana     int
}

// SpellBookData holds the IDs of spells this commander can cast.
// SpellIDs are keys into the global SpellRegistry.
type SpellBookData struct {
    SpellIDs []string
}
```

**ECS variables (initialized by `init()`):**

```go
var (
    ManaComponent      *ecs.Component
    SpellBookComponent *ecs.Component

    ManaTag      ecs.Tag
    SpellBookTag ecs.Tag
)
```

These are registered in the ECS world's `WorldTags` map as `"mana"` and `"spellbook"` respectively, enabling tag-based queries elsewhere.

**SpellCastResult** is the return value of `ExecuteSpellCast`, used by the GUI to update caches and trigger VX:

```go
type SpellCastResult struct {
    Success          bool
    ErrorReason      string         // Non-empty on failure
    TotalDamageDealt int            // Sum of all damage across all targets
    AffectedSquadIDs []ecs.EntityID // Squads that were hit/buffed/debuffed
    SquadsDestroyed  []ecs.EntityID // Squads with no surviving units
}
```

---

### 3.3 Effects ECS Components (`effects`)

**File:** `tactical/powers/effects/components.go`

The effects system is the runtime counterpart to the static spell definitions. When a buff or debuff is applied, an `ActiveEffect` is created and attached to each affected unit entity.

```go
// StatType identifies which Attributes field an effect modifies.
type StatType int
const (
    StatStrength StatType = iota
    StatDexterity
    StatMagic
    StatLeadership
    StatArmor
    StatWeapon
    StatMovementSpeed
    StatAttackRange
)

// EffectSource identifies the origin system (for future filtering/removal).
type EffectSource int
const (
    SourceSpell   EffectSource = iota  // Created by a spell cast
    SourceAbility                       // Created by a unit ability
    SourceItem                          // Created by an artifact (permanent)
)

// ActiveEffect is a single stat modifier with a remaining duration.
type ActiveEffect struct {
    Name           string       // Display name (typically the spell name)
    Source         EffectSource // Where this effect came from
    Stat           StatType     // Which attribute to modify
    Modifier       int          // Amount; positive = buff, negative = debuff
    RemainingTurns int          // -1 = permanent; 0 = expired; >0 = turns left
}

// ActiveEffectsData is the ECS component attached to entities with active effects.
type ActiveEffectsData struct {
    Effects []ActiveEffect
}
```

**ECS variables:**

```go
var (
    ActiveEffectsComponent *ecs.Component
    ActiveEffectsTag       ecs.Tag
)
```

The `ActiveEffectsComponent` is created lazily: `ApplyEffect` creates it on the entity the first time an effect is applied. Entities with no active effects have no component attached, which is more memory-efficient than attaching empty components to all units.

**ParseStatType** converts the JSON stat name strings in `SpellStatModifier` to `StatType` values. It is case-insensitive. Valid inputs: `"strength"`, `"dexterity"`, `"magic"`, `"leadership"`, `"armor"`, `"weapon"`, `"movementspeed"`, `"attackrange"`. Returns an error for unrecognized names, which `applyBuffDebuffSpell` logs as a warning and skips.

---

## 4. Spell Lifecycle: Definition to Resolution

This section traces a complete spell cast from game startup through the moment the effect is reversed on expiry.

### 4.1 Phase 1 — Data Loading at Startup

`templates.ReadGameData()` is called once during game initialization (`setup/gamesetup`). It calls `LoadSpellDefinitions()` near the end of its sequence:

```go
// templates/registry.go
func ReadGameData() {
    ReadGameConfig()
    ReadDifficultyConfig()
    // ... (other data)
    LoadSpellDefinitions()       // ← spells loaded here
    LoadArtifactDefinitions()
}
```

`LoadSpellDefinitions()` reads `assets/gamedata/spelldata.json`, unmarshals it into `[]SpellDefinition`, and inserts each entry into `SpellRegistry` keyed by `SpellDefinition.ID`. The registry is a plain `map[string]*SpellDefinition` — there is no lock because it is written once at startup and read-only thereafter.

After loading, the registry contains 22 spells (as of the current data file): 18 damage spells and 4 buff/debuff spells.

---

### 4.2 Phase 2 — Commander Creation

When a commander entity is created (typically during overworld setup), `tactical/commander/system.go:CreateCommander()` attaches the spell-related components:

```go
entity.
    AddComponent(spells.ManaComponent, &spells.ManaData{
        CurrentMana: startingMana,
        MaxMana:     maxMana,
    }).
    AddComponent(spells.SpellBookComponent, &spells.SpellBookData{
        SpellIDs: initialSpells,
    })
```

The `startingMana` and `maxMana` values come from `templates.GameConfig.Commander.StartingMana` and `templates.GameConfig.Commander.MaxMana`, which are read from `assets/gamedata/gameconfig.json`. The `initialSpells` slice (a list of spell ID strings) is passed in by the calling code and determines which spells the commander starts with.

A commander who has no `ManaComponent` or `SpellBookComponent` cannot cast spells — `ExecuteSpellCast` returns early with a descriptive error. This means NPCs and squad entities (which never receive these components) are structurally unable to cast spells.

---

### 4.3 Phase 3 — GUI Spell Panel Interaction

When the player clicks the "Spell" button during their turn, `SpellPanelController.Toggle()` is called:

```
Toggle()
  → if already in spell mode: CancelSpellMode() + Hide()
  → else: Show()
       → checks HasCastSpell (one spell per turn limit)
       → calls Handler.EnterSpellMode()     → sets BattleState.InSpellMode = true
       → calls Refresh()
            → GetAllSpells(commanderID) — all spells in spellbook (regardless of mana)
            → GetCommanderMana(commanderID) — updates mana label
            → populates spell list widget
       → calls ShowSubmenu()               → displays the spell panel
```

The spell list shows all spells in the spellbook. The detail panel, updated by `OnSpellSelected()`, checks whether `currentMana >= spell.ManaCost` and disables the Cast button if the player cannot afford the spell. Mana-checking happens in two places: in the UI (to gray out the button) and inside `ExecuteSpellCast` (to guard against race conditions or direct API calls).

---

### 4.4 Phase 4 — Targeting

After the player selects a spell and clicks "Cast", `SpellPanelController.OnCastClicked()` calls `Handler.SelectSpell(spellID)`:

```go
func (h *SpellCastingHandler) SelectSpell(spellID string) {
    // 1. Validate mana
    if !spells.HasEnoughMana(commanderID, spellID, h.deps.ECSManager) { return }

    // 2. Store selected spell ID
    h.deps.BattleState.SelectedSpellID = spellID

    // 3. For AoE spells: create the shape overlay
    if !spell.IsSingleTarget() {
        h.activeShape = createAoEShape(spell)
    }
    // Single-target: no overlay needed; next click on a squad fires the cast
}
```

**Single-target flow:** The combat input handler detects `IsInSpellMode() && !IsAoETargeting()` and routes the next click to `HandleSingleTargetClick()`. This converts the mouse position to a `LogicalPosition`, looks up any squad at that tile using `combatcore.GetSquadAtPosition()`, validates that the clicked squad is an enemy in the current encounter, then calls `executeSpellOnTargets([clickedSquadID], nil)`.

**AoE flow:** On each frame the input handler calls `HandleAoETargetingFrame(mouseX, mouseY)`, which moves the shape overlay on the game map and applies a purple color matrix to the covered tiles. When the player left-clicks, `HandleAoEConfirmClick()` gathers all enemy squads in the covered tile set (deduplicating with a map) and calls `executeSpellOnTargets(targetIDs, &logicalPos)`.

**Shape rotation:** For directional AoE shapes (e.g., `Cone`, `Line`), the player can rotate the shape using Q/E keys, which call `RotateShapeLeft()` and `RotateShapeRight()` on the handler. These mutate the `Direction` field on the underlying `*graphics.BaseShape`.

---

### 4.5 Phase 5 — Execution (`ExecuteSpellCast`)

**File:** `tactical/powers/spells/system.go`

`ExecuteSpellCast` is the public entry point for the logic layer. The GUI calls this after targeting is complete:

```go
func ExecuteSpellCast(
    casterEntityID ecs.EntityID,
    spellID string,
    targetSquadIDs []ecs.EntityID,
    manager *common.EntityManager,
) *SpellCastResult
```

Validation sequence (returns early with `Success = false` on any failure):

1. Spell ID exists in `SpellRegistry`
2. Caster has a `ManaComponent`
3. Caster has enough `CurrentMana`
4. Spell is in the caster's `SpellBookData.SpellIDs` list

On success: deducts `spell.ManaCost` from `mana.CurrentMana`, then dispatches to the appropriate handler based on `spell.EffectType`:

```go
switch spell.EffectType {
case templates.EffectDamage:
    applyDamageSpell(spell, targetSquadIDs, result, manager)
case templates.EffectBuff, templates.EffectDebuff:
    applyBuffDebuffSpell(spell, targetSquadIDs, result, manager)
}
result.Success = true
```

---

### 4.6 Phase 6 — Effect Application

**Damage spells (`applyDamageSpell`):**

Iterates over each target squad, then each unit in the squad:
1. Gets the unit entity and its `Attributes` component.
2. Skips dead units (`CurrentHealth <= 0`).
3. Calculates `damage = spell.Damage - attr.GetMagicDefense()`, floored at 1. This means magic defense (`Magic/2 + BaseMagicResist`) reduces damage but never fully negates it.
4. Subtracts `damage` from `attr.CurrentHealth`, clamped at 0.
5. Accumulates `result.TotalDamageDealt`.

After processing all units in a squad, if `squadcore.IsSquadDestroyed(squadID, manager)` is true (all units dead), the squad is removed from the map via `combatcore.RemoveSquadFromMap(squadID, manager)` and the squad ID is added to `result.SquadsDestroyed`.

**Buff/debuff spells (`applyBuffDebuffSpell`):**

For each target squad and each `StatModifier` in the spell definition:
1. Parses the stat name with `effects.ParseStatType(mod.Stat)`.
2. Constructs an `effects.ActiveEffect` with:
   - `Name`: the spell's display name
   - `Source`: `effects.SourceSpell`
   - `Stat`: the parsed `StatType`
   - `Modifier`: `mod.Modifier` (positive for buff, negative for debuff)
   - `RemainingTurns`: `spell.Duration`
3. Calls `effects.ApplyEffectToUnits(unitIDs, effect, manager)`.

The `ApplyEffect` function (in the effects package) immediately applies the modifier to the live `Attributes` struct and stores the `ActiveEffect` in the entity's `ActiveEffectsData` component. This means buff/debuff spells take effect instantly on the turn they are cast.

---

### 4.7 Phase 7 — Turn Ticking and Expiry

**File:** `tactical/combat/combatcore/turnmanager.go`, `ResetSquadActions()`

At the start of each faction's turn, `ResetSquadActions()` is called. For every squad belonging to the newly-active faction, it calls:

```go
unitIDs := squadcore.GetUnitIDsInSquad(squadID, tm.manager)
effects.TickEffectsForUnits(unitIDs, tm.manager)
```

`TickEffects` (per unit) decrements `RemainingTurns` on each non-permanent effect. When `RemainingTurns` reaches 0, the modifier is reversed by calling `applyModifierToStat(attr, stat, -modifier)` (i.e., adding the negative), and the `ActiveEffect` is removed from the slice. This restores the unit to its pre-spell stat values.

The in-place slice filter pattern used in `TickEffects` avoids heap allocation on every tick:

```go
kept := effectsData.Effects[:0]  // reuse the slice's backing array
for i := range effectsData.Effects {
    e := &effectsData.Effects[i]
    if e.RemainingTurns == -1 { kept = append(kept, *e); continue }
    e.RemainingTurns--
    if e.RemainingTurns <= 0 {
        reverseModifierFromStat(attr, e.Stat, e.Modifier)
    } else {
        kept = append(kept, *e)
    }
}
effectsData.Effects = kept
```

---

## 5. The Effects System

### 5.1 Design Philosophy

The effects system uses an **apply-immediately, reverse-on-expiry** model rather than storing base stats and computing modified values on demand. This choice has several consequences:

- **Reads are O(1) and simple.** Anything reading `attr.Strength` always sees the current value. There is no "calculate effective stat" function to call.
- **Stacking is additive.** Multiple effects on the same stat accumulate. Two +4 Strength buffs produce +8 Strength. When one expires, Strength drops by 4.
- **Reversal must be precise.** `TickEffects` stores the original modifier value and subtracts it exactly. This works correctly as long as no other system rounds or clamps stats mid-effect. The only clamping in the system is on `CurrentHealth` (clamped at 0 on damage) and derived stats that have configuration caps.
- **Dead units are skipped.** `ApplyEffect` checks `attr.CurrentHealth <= 0` and returns without applying if the unit is dead. This prevents wasted effects and avoids a reversal later that could drive stats below zero for dead units.

### 5.2 StatType Enum

The `StatType` enum covers all eight attributes from `common.Attributes` that can be meaningfully modified by a temporary effect:

| StatType | Targets field | Affects |
|---|---|---|
| `StatStrength` | `Attributes.Strength` | Physical damage, resistance, max HP |
| `StatDexterity` | `Attributes.Dexterity` | Hit rate, crit, dodge |
| `StatMagic` | `Attributes.Magic` | Magic damage, magic defense, healing |
| `StatLeadership` | `Attributes.Leadership` | Squad unit capacity |
| `StatArmor` | `Attributes.Armor` | Physical damage reduction modifier |
| `StatWeapon` | `Attributes.Weapon` | Physical damage increase modifier |
| `StatMovementSpeed` | `Attributes.MovementSpeed` | Tiles per turn |
| `StatAttackRange` | `Attributes.AttackRange` | Attack distance in tiles |

`CurrentHealth` and `CanAct` are intentionally excluded: health changes are not stat modifiers, and action-state manipulation is handled directly by the artifact behavior system (not the effects system).

### 5.3 EffectSource Enum

| EffectSource | Value | Currently used by |
|---|---|---|
| `SourceSpell` | `0` | All spell-created effects in `applyBuffDebuffSpell` |
| `SourceAbility` | `1` | Permanent effects from unit abilities (e.g., the effects test uses this for "Permanent Armor") |
| `SourceItem` | `2` | Artifact stat modifier effects applied at battle start |

The source enum is stored on each `ActiveEffect` but is not currently used for filtering or removal. It exists to support future features such as "remove all spell effects" without affecting equipment bonuses.

### 5.4 ActiveEffect Struct

```go
type ActiveEffect struct {
    Name           string
    Source         EffectSource
    Stat           StatType
    Modifier       int   // positive = buff, negative = debuff
    RemainingTurns int   // -1 = permanent, 0 = expired (removed), >0 = active
}
```

One `ActiveEffect` represents a single stat modifier from a single source. A spell with two `StatModifiers` (e.g., `frost_slow` modifies both Dexterity and MovementSpeed) creates two separate `ActiveEffect` entries on each unit.

### 5.5 Applying, Ticking, and Removing Effects

**Apply:** `ApplyEffect(entityID, effect, manager)` — applies to a single unit entity.

`ApplyEffectToUnits(unitIDs, effect, manager)` — convenience wrapper for squad-level application.

**Tick:** `TickEffects(entityID, manager)` — decrements and expires for a single unit.

`TickEffectsForUnits(unitIDs, manager)` — convenience wrapper called by `TurnManager.ResetSquadActions()`.

**Remove all:** `RemoveAllEffects(entityID, manager)` — reverses all active effects and clears the list. Used for cleanup scenarios (e.g., on combat exit, to ensure units return to baseline). Not currently called automatically; callers must invoke explicitly when needed.

### 5.6 Permanent Effects (RemainingTurns = -1)

Effects with `RemainingTurns == -1` are never decremented or expired by `TickEffects`. They survive for the entire battle (or until explicitly removed by `RemoveAllEffects`). This is used by the artifact system: when `ApplyArtifactStatEffects` applies equipment bonuses at battle start, it sets `RemainingTurns: -1` so the bonus persists for the whole fight regardless of how many turns elapse.

---

## 6. Integration with the ECS Architecture

### 6.1 Subsystem Registration

Both `spells` and `effects` packages register themselves via `init()`:

```go
// tactical/powers/spells/init.go
func init() {
    common.RegisterSubsystem(func(em *common.EntityManager) {
        ManaComponent      = em.World.NewComponent()
        SpellBookComponent = em.World.NewComponent()
        ManaTag      = ecs.BuildTag(ManaComponent)
        SpellBookTag = ecs.BuildTag(SpellBookComponent)
        em.WorldTags["mana"]      = ManaTag
        em.WorldTags["spellbook"] = SpellBookTag
    })
}

// tactical/powers/effects/init.go
func init() {
    common.RegisterSubsystem(func(em *common.EntityManager) {
        ActiveEffectsComponent = em.World.NewComponent()
        ActiveEffectsTag       = ecs.BuildTag(ActiveEffectsComponent)
    })
}
```

This self-registration pattern means these subsystems are automatically initialized when `common.InitializeSubsystems(manager)` is called in main. The `init()` function simply registers a closure; the actual component creation happens in that closure when the `EntityManager` is ready. No manual coordination in `main.go` or `gamesetup` is needed.

### 6.2 Component Access Patterns Used

The spells and effects packages follow the project's standard ECS access patterns:

**By entity ID (most common — used throughout spell execution):**

```go
// spells/queries.go
func GetManaData(entityID ecs.EntityID, manager *common.EntityManager) *ManaData {
    return common.GetComponentTypeByID[*ManaData](manager, entityID, ManaComponent)
}
```

**By entity pointer (used in ApplyEffect when entity is already needed):**

```go
entity := manager.FindEntityByID(entityID)
attr := common.GetComponentType[*common.Attributes](entity, common.AttributeComponent)
```

**Lazy component creation (used in ApplyEffect for ActiveEffectsData):**

```go
if entity.HasComponent(ActiveEffectsComponent) {
    effectsData = common.GetComponentType[*ActiveEffectsData](entity, ActiveEffectsComponent)
} else {
    effectsData = &ActiveEffectsData{}
    entity.AddComponent(ActiveEffectsComponent, effectsData)
}
```

This pattern avoids attaching an empty `ActiveEffectsData` to every unit entity at creation time.

### 6.3 Entity Relationships

The spell system involves three entity types:

```
Commander entity
├── CommanderComponent (commander/components.go)
├── ManaComponent (spells/components.go)          ← ManaData {CurrentMana, MaxMana}
├── SpellBookComponent (spells/components.go)     ← SpellBookData {SpellIDs []}
├── AttributeComponent (common)
├── PositionComponent (common)
├── RenderableComponent (common)
├── SquadRosterComponent (squads/roster)
└── CommanderActionStateComponent (commander/components.go)

Squad entity
└── (no spell-related components; squads are spell targets, not casters)

Unit entity (child of squad)
├── AttributeComponent (common)                  ← Attributes {Strength, Dexterity, ...}
└── ActiveEffectsComponent (effects/components.go) ← Added lazily when first effect applied
    └── ActiveEffectsData {Effects [ActiveEffect, ...]}
```

Commander entities are always on the overworld map (they have `PositionComponent`). They are not deployed to the tactical combat map; they cast spells from the GUI panel. Squad entities and unit entities are on the tactical map and are the targets of spells.

---

## 7. Key Functions Reference

### 7.1 spells Package

| Function | Signature | Purpose |
|---|---|---|
| `GetManaData` | `(entityID, manager) → *ManaData` | Gets mana component; nil if not found |
| `GetSpellBook` | `(entityID, manager) → *SpellBookData` | Gets spellbook component; nil if not found |
| `HasEnoughMana` | `(entityID, spellID, manager) → bool` | Checks mana AND that spell exists in registry |
| `GetCastableSpells` | `(entityID, manager) → []*SpellDefinition` | Returns spells the entity can currently afford |
| `HasSpellInBook` | `(entityID, spellID, manager) → bool` | Checks whether spell is in the spellbook |
| `GetAllSpells` | `(entityID, manager) → []*SpellDefinition` | Returns all spells in book (regardless of mana) |
| `ExecuteSpellCast` | `(casterID, spellID, targetSquadIDs, manager) → *SpellCastResult` | Main execution entry point; validates, deducts mana, applies effects |

### 7.2 effects Package

| Function | Signature | Purpose |
|---|---|---|
| `ParseStatType` | `(string) → (StatType, error)` | Converts JSON stat name to enum value; case-insensitive |
| `GetActiveEffects` | `(entityID, manager) → *ActiveEffectsData` | Gets effects component; nil if none |
| `HasActiveEffects` | `(entityID, manager) → bool` | True if entity has any active effects |
| `ApplyEffect` | `(entityID, effect, manager)` | Applies one effect to one entity; creates component if needed; skips dead units |
| `ApplyEffectToUnits` | `(unitIDs, effect, manager)` | Calls `ApplyEffect` for each ID in the slice |
| `TickEffects` | `(entityID, manager)` | Decrements duration, reverses and removes expired effects |
| `TickEffectsForUnits` | `(unitIDs, manager)` | Calls `TickEffects` for each ID in the slice |
| `RemoveAllEffects` | `(entityID, manager)` | Reverses all active effects and clears the list |

### 7.3 templates Package (spell registry)

| Function | Signature | Purpose |
|---|---|---|
| `LoadSpellDefinitions` | `()` | Reads `spelldata.json`, populates `SpellRegistry` |
| `GetSpellDefinition` | `(id string) → *SpellDefinition` | Lookup by ID; nil if not registered |
| `GetAllSpellIDs` | `() → []string` | Returns all registered spell IDs |
| `(sd) IsSingleTarget` | `() → bool` | True if `TargetType == "single"` |
| `(sd) IsAoE` | `() → bool` | True if `TargetType == "aoe"` |
| `(sd) IsDamage` | `() → bool` | True if `EffectType == "damage"` |
| `(sd) IsBuff` | `() → bool` | True if `EffectType == "buff"` |
| `(sd) IsDebuff` | `() → bool` | True if `EffectType == "debuff"` |

---

## 8. Spell Configuration and JSON Schema

### 8.1 File Location

```
assets/gamedata/spelldata.json
```

The path constant is defined as:
```go
// templates/registry.go
const SpellDataPath = "gamedata/spelldata.json"
```

`LoadSpellDefinitions` uses `AssetPath(SpellDataPath)` to resolve the full path relative to the binary's working directory.

### 8.2 SpellDefinition Fields

The JSON object for each spell maps directly to `SpellDefinition` via Go's `encoding/json`:

| JSON Field | Go Field | Type | Required | Notes |
|---|---|---|---|---|
| `id` | `ID` | string | Yes | Must be unique across all spells |
| `name` | `Name` | string | Yes | Display name shown in UI |
| `description` | `Description` | string | Yes | Tooltip text |
| `manaCost` | `ManaCost` | int | Yes | Deducted from commander on cast |
| `damage` | `Damage` | int | No (default 0) | Base damage for damage spells; ignored for buff/debuff |
| `targetType` | `TargetType` | string | Yes | `"single"` or `"aoe"` |
| `effectType` | `EffectType` | string | Yes | `"damage"`, `"buff"`, or `"debuff"` |
| `shape` | `Shape` | object | No | Required for AoE spells; omit for single-target |
| `vxType` | `VXType` | string | Yes | `"fire"`, `"electricity"`, `"ice"`, or `"cloud"` |
| `vxDuration` | `VXDuration` | int | Yes | Visual effect duration in frames |
| `duration` | `Duration` | int | No | Required for buff/debuff; ignored for damage |
| `statModifiers` | `StatModifiers` | array | No | Required for buff/debuff; ignored for damage |

**Shape object (`JSONTargetArea`):**

| JSON Field | Type | Used by Shape Types |
|---|---|---|
| `type` | string | `"Circle"`, `"Square"`, `"Rectangle"`, `"Line"`, `"Cone"` |
| `size` | int | Circle (radius), Square (half-side) |
| `length` | int | Line, Cone |
| `width` | int | Rectangle |
| `height` | int | Rectangle |
| `radius` | int | Circle (alternative to size) |

**StatModifier object:**

| JSON Field | Type | Notes |
|---|---|---|
| `stat` | string | One of: `"strength"`, `"dexterity"`, `"magic"`, `"leadership"`, `"armor"`, `"weapon"`, `"movementspeed"`, `"attackrange"` |
| `modifier` | int | Positive for buff; negative for debuff |

### 8.3 Target Shape Configurations

The shape type string maps to a `graphics.TileBasedShape` via `graphics.CreateShapeFromConfig()`. Shape behavior:

- **`Circle` with `size`**: Circular area. Size 1 = small, 2 = medium, 3 = large. Used by Fireball, Immolation, Firestorm.
- **`Square` with `size`**: Square area. Size 1 = 3x3 tiles, 2 = 5x5, 3 = 7x7. Used by Blizzard, Frost Snap, Absolute Zero.
- **`Rectangle` with `width` and `height`**: Rectangular area. Width controls horizontal extent; height controls vertical. Used by Miasma (5×2), Fog of Ruin (3×4).
- **`Line` with `length`**: Linear strip of tiles. Rotatable. Used by Chain Lightning, Ice Lance, Wall of Flame.
- **`Cone` with `length`**: Fan-shaped area. Rotatable. Used by Scalding Gust, Thunder Cone, Noxious Eruption.

Directional shapes (Line and Cone) support rotation via Q/E during targeting, which mutates the `Direction` field on the `*graphics.BaseShape`.

### 8.4 Complete Spell Roster

As of the current data file, 22 spells are defined:

**Damage Spells (18):**

| ID | Name | Mana | Damage | Target | Shape |
|---|---|---|---|---|---|
| `spark` | Spark | 5 | 15 | single | — |
| `singe` | Singe | 6 | 18 | single | — |
| `frost_snap` | Frost Snap | 8 | 12 | aoe | Square 1 |
| `lightning_bolt` | Lightning Bolt | 10 | 30 | single | — |
| `ice_lance` | Ice Lance | 12 | 25 | aoe | Line 4 |
| `chain_lightning` | Chain Lightning | 14 | 22 | aoe | Line 3 |
| `miasma` | Miasma | 14 | 10 | aoe | Rect 5×2 |
| `fireball` | Fireball | 15 | 20 | aoe | Circle 1 |
| `scalding_gust` | Scalding Gust | 16 | 16 | aoe | Cone 3 |
| `wall_of_flame` | Wall of Flame | 18 | 18 | aoe | Line 5 |
| `fog_of_ruin` | Fog of Ruin | 20 | 12 | aoe | Rect 3×4 |
| `thunder_cone` | Thunder Cone | 20 | 20 | aoe | Cone 4 |
| `noxious_eruption` | Noxious Eruption | 25 | 18 | aoe | Cone 5 |
| `blizzard` | Blizzard | 25 | 15 | aoe | Square 2 |
| `obliterate` | Obliterate | 30 | 55 | single | — |
| `immolation` | Immolation | 30 | 25 | aoe | Circle 2 |
| `firestorm` | Firestorm | 35 | 22 | aoe | Circle 3 |
| `absolute_zero` | Absolute Zero | 40 | 20 | aoe | Square 3 |

**Buff Spells (2):**

| ID | Name | Mana | Duration | Effect |
|---|---|---|---|---|
| `war_cry` | War Cry | 10 | 3 turns | Strength +4 |
| `arcane_shield` | Arcane Shield | 12 | 3 turns | Armor +3 |

**Debuff Spells (2):**

| ID | Name | Mana | Duration | Effect |
|---|---|---|---|---|
| `weaken` | Weaken | 8 | 2 turns | Strength -3 |
| `frost_slow` | Frost Slow | 10 | 2 turns | Dexterity -2, MovementSpeed -1 |

Note that `frost_slow` applies two separate `ActiveEffect` entries (one for Dexterity, one for MovementSpeed) to each unit, since each `SpellStatModifier` becomes one effect.

---

## 9. Artifacts and the Shared Effects System

### 9.1 Why Two `effects` Packages Exist

There are two packages that contain essentially identical code:

- `tactical/powers/effects/` — used by the spells package
- `tactical/powers/artifacts/effects/` — used by the artifacts package

Both declare the same `StatType`, `EffectSource`, `ActiveEffect`, and `ActiveEffectsData` types along with the same `ApplyEffect`/`TickEffects`/`RemoveAllEffects` functions. This is a code duplication that exists to avoid a circular import: if artifacts imported `tactical/powers/effects`, and effects imported something that artifacts also needed, a cycle would form.

From a developer perspective, when working on either system, be aware that changes to one effects package may need to be mirrored in the other. If the duplication causes maintenance issues, the resolution would be extracting a shared `effectscore` package that both can import.

The `TurnManager` in `combatcore` imports `tactical/powers/effects` (not the artifact version), so ticking is handled by the same implementation that spells use.

### 9.2 Minor Artifact Effects vs Spell Effects

| Dimension | Spell Effects | Artifact Stat Effects |
|---|---|---|
| Source | `SourceSpell` | `SourceItem` |
| `RemainingTurns` | Set from `spell.Duration` (positive int) | Always `-1` (permanent for battle) |
| When applied | At cast time (mid-battle) | At battle start, before turn init |
| Applied to | Enemy or friendly squads (depending on spell type) | Player squads only |
| Can expire | Yes | No (until `RemoveAllEffects` is called) |

### 9.3 Artifact Effect Application Timing

`CombatService.InitializeCombat()` calls `artifacts.ApplyArtifactStatEffects(factionSquads, manager)` **before** calling `TurnManager.InitializeCombat()`. This sequencing ensures artifact bonuses are in place before the first faction's turn begins, so the turn manager sees accurate movement speeds when setting `MovementRemaining`.

The call in `combat_service.go` iterates all factions, so if an enemy faction somehow had equipped artifacts, those would also be applied (though the current game only equips player squads).

---

## 10. GUI Layer Integration (`guispells`)

### 10.1 SpellCastingHandler

**File:** `gui/guispells/spell_handler.go`

`SpellCastingHandler` is the authoritative controller for all spell mode transitions. It owns:
- `deps *SpellCastingDeps` — all injected dependencies
- `activeShape graphics.TileBasedShape` — the current AoE shape (nil between casts)
- `prevIndices []int` — tile indices covered in the previous frame, for clearing overlays

**Key responsibilities:**
1. Gating entry into spell mode (`EnterSpellMode`, `CancelSpellMode`)
2. Validating mana before entering targeting (`SelectSpell`)
3. Managing the AoE shape overlay during the targeting frame loop
4. Routing clicks to the appropriate handler (single vs AoE)
5. Calling `spells.ExecuteSpellCast` and handling the result
6. Triggering visual effects via `triggerSpellVX`
7. Invalidating the GUI query cache for affected squads

**Dependencies (via `SpellCastingDeps`):**

```go
type SpellCastingDeps struct {
    BattleState *framework.TacticalState   // For mode flags and spell selection
    ECSManager  *common.EntityManager      // For ECS queries
    GameMap     *worldmap.GameMap          // For tile color matrix (overlay)
    PlayerPos   *coords.LogicalPosition    // For coordinate conversion
    Queries     *framework.GUIQueries      // For squad info and cache invalidation
    Encounter   combatcore.EncounterCallbacks // For commander and encounter IDs
}
```

### 10.2 SpellPanelController

**File:** `gui/guispells/spell_panel.go`

`SpellPanelController` owns the spell selection UI. It bridges between the ebitenui widget events and the `SpellCastingHandler` logic:

- `Refresh()` — repopulates the spell list and mana label whenever the panel is shown
- `OnSpellSelected(spell)` — updates the detail area and enables/disables the Cast button based on current mana
- `OnCastClicked()` — calls `Handler.SelectSpell()` and hides the panel to enter targeting mode
- `OnCancelClicked()` — calls `Handler.CancelSpellMode()` and hides the panel
- `Toggle()` — the entry point wired to the Spell button; cancels if already in spell mode, shows if not

The controller checks `BattleState.HasCastSpell` in `Show()` and returns early if the commander has already cast this turn. This enforces the one-spell-per-turn rule before the player even sees the panel.

### 10.3 TacticalState Spell Fields

**File:** `gui/framework/contextstate.go`

Three fields in `TacticalState` govern spell behavior:

```go
InSpellMode     bool    // True while the player is in any spell-related mode
                        // (panel open OR targeting active)
SelectedSpellID string  // The spell ID currently being targeted; "" between casts
HasCastSpell    bool    // True after a successful cast this turn; reset at turn end
```

`HasCastSpell` is set to `true` by `executeSpellOnTargets` after a successful cast. It is reset to `false` by `CombatMode` when the turn changes (in the turn end logic). `TacticalState.Reset()` clears all three fields for new battles.

The `InSpellMode` flag is used by the input handler to route clicks during targeting. The `SelectedSpellID` field stores which spell is being targeted so `executeSpellOnTargets` can look it up without re-querying the panel.

### 10.4 AoE Shape Targeting Flow

```
Frame loop (combat_input_handler.go):
  if IsInSpellMode() && IsAoETargeting():
    HandleAoETargetingFrame(mouseX, mouseY)
      → CoordManager.LogicalToPixel(logicalPos)
      → activeShape.UpdatePosition(pixelX, pixelY)
      → activeShape.GetIndices() → []int
      → GameMap.ApplyColorMatrix(prevIndices, emptyMatrix)  ← clear previous frame
      → for each idx in indices:
           GameMap.ApplyColorMatrixToIndex(idx, purple_overlay)
      → prevIndices = indices

  if left click:
    HandleAoEConfirmClick(mouseX, mouseY)
      → same position logic as above
      → for each covered index:
           CoordManager.IndexToLogical(idx)
           combatcore.GetSquadAtPosition(logicalPos, manager)
           filter: IsEnemySquadInEncounter(squadID, encounterID)
      → executeSpellOnTargets(targetIDs, &logicalPos)
```

The color matrix overlay uses pixel-coordinate-aligned tile indices from the `CoordManager`. Using `coords.CoordManager.LogicalToPixel()` followed by shape positioning (rather than directly computing from mouse coordinates) ensures the shape snaps to tile boundaries, which prevents visual artifacts from sub-pixel positions.

---

## 11. Visual Effects (VX) Integration

After a spell cast succeeds, `triggerSpellVX()` creates visual effects that are rendered over the game map for `spell.VXDuration` frames:

**AoE spells:** One `VisualEffectArea` is created at the clicked position using the same shape configuration. The area is added to a global VX queue via `graphics.AddVXArea(area)`.

**Single-target spells:** One `VisualEffect` is created per target squad, positioned at the squad's screen coordinates. Each is added via `graphics.AddVX(vx)`.

VX types map to visual effect presets: `"fire"` (orange/red), `"electricity"` (blue/white flash), `"ice"` (blue crystalline), `"cloud"` (gray/green). The specific effect behavior is defined in `visual/graphics`.

VX creation uses `coords.CoordManager.LogicalToScreen(pos, playerPos)` for single-target spells, which correctly applies the camera offset relative to the player's position. This is different from `LogicalToPixel`, which gives absolute coordinates without camera offset.

---

## 12. Turn Management and Effect Ticking

Effect ticking is wired into the turn system at a single point: `TurnManager.ResetSquadActions()`.

```
TurnManager.EndTurn()
  → increments CurrentTurnIndex (wraps to next round if needed)
  → ResetSquadActions(newFactionID)
       → for each squad in the faction:
            reset action state flags (HasMoved, HasActed, BonusAttackActive = false)
            set MovementRemaining from squad speed
            effects.TickEffectsForUnits(unitIDs, manager)  ← *** tick here ***
            CheckAndTriggerAbilities(squadID, manager)
       → postResetHook(factionID, factionSquads)  ← artifacts fire here
```

This means effects tick at the **start of the owning faction's turn** rather than at the end of every faction's turn. The practical consequence:

- A buff with `Duration: 2` cast on a player squad during the player's turn will be active for the player's next two turns (ticked down at the start of each).
- If the enemy has two factions, the effect is ticked twice per round but only once between player turns.

**Current behavior note:** Effects tick for the faction that is about to move, not the faction that owns the units being affected. When player squad effects tick via `ResetSquadActions(playerFactionID)`, it ticks effects on player squads. When the enemy turn begins and `ResetSquadActions(enemyFactionID)` is called, it ticks effects on enemy squads (from debuffs). This is the intended behavior — debuffs cast by the player on enemy squads expire at the start of each enemy turn.

---

## 13. Mana Economy and Strategic Design

Mana is a **cross-battle persistent resource** attached to commanders, not a per-combat resource. This is a deliberate design decision:

- Commanders start a battle with whatever mana they had at the end of the last battle.
- Spells consume mana permanently unless mana is replenished through game progression mechanics.
- This creates strategic tension: spending all mana in one difficult battle leaves the commander unable to cast in subsequent encounters.

The `gameconfig.json` `commander` section defines starting values:
- `startingMana`: Mana when a commander is first created.
- `maxMana`: The cap that `CurrentMana` cannot exceed.

There is no in-battle mana regeneration mechanic in the current implementation. Mana recovery would need to be implemented at the overworld level (e.g., resting at a node, purchasing mana potions).

The one-spell-per-turn limit (`HasCastSpell`) prevents the mana economy from being trivially exploited through turn manipulation. Even if a commander has enough mana for multiple spells, they can only cast one per turn.

---

## 14. Adding New Spells — Developer Guide

### Step 1: Add the JSON definition

Open `assets/gamedata/spelldata.json` and add a new entry to the `"spells"` array. Choose a unique `id` string.

**Minimum viable damage spell:**

```json
{
  "id": "my_new_spell",
  "name": "My New Spell",
  "description": "Does something interesting",
  "manaCost": 12,
  "damage": 25,
  "targetType": "single",
  "effectType": "damage",
  "vxType": "fire",
  "vxDuration": 2
}
```

**Buff/debuff spell (must include `duration` and `statModifiers`):**

```json
{
  "id": "battle_trance",
  "name": "Battle Trance",
  "description": "Heightens reflexes for several turns",
  "manaCost": 15,
  "targetType": "single",
  "effectType": "buff",
  "duration": 3,
  "statModifiers": [
    { "stat": "dexterity", "modifier": 5 },
    { "stat": "attackrange", "modifier": 1 }
  ],
  "vxType": "electricity",
  "vxDuration": 2
}
```

**AoE spell (must include `shape`):**

```json
{
  "id": "ring_of_fire",
  "name": "Ring of Fire",
  "description": "Encircles an area in flame",
  "manaCost": 22,
  "damage": 20,
  "targetType": "aoe",
  "effectType": "damage",
  "shape": { "type": "Circle", "size": 2 },
  "vxType": "fire",
  "vxDuration": 3
}
```

No Go code changes are needed for data-only additions. The spell will be automatically available to any commander whose `initialSpells` list includes the new ID.

### Step 2: Grant the spell to a commander

The `initialSpells` parameter is passed to `CreateCommander()`. The calling code (typically game setup or commander upgrade systems) must include the new spell ID in that slice:

```go
commanderID := commander.CreateCommander(
    manager,
    "Arcturus",
    startPos,
    movementSpeed,
    maxSquads,
    commanderImage,
    startingMana,
    maxMana,
    []string{"fireball", "lightning_bolt", "ring_of_fire"}, // ← new spell here
)
```

Alternatively, a spell can be added to an existing commander's spellbook at runtime:

```go
book := spells.GetSpellBook(commanderID, manager)
if book != nil {
    book.SpellIDs = append(book.SpellIDs, "ring_of_fire")
}
```

### Step 3: Validate

Run `go test ./...` to catch any issues. The `effects_test.go` suite verifies the effects system independently. If you added a buff/debuff spell with a new stat name, confirm it is one of the eight recognized by `ParseStatType` or add a new case there (and update the documentation table in Section 5.2).

### Adding a new effect type (beyond damage/buff/debuff)

If a new spell needs a behavior not covered by the three existing `EffectType` values (e.g., summoning, terrain modification, teleportation), you would need to:

1. Add a new `SpellEffectType` constant in `templates/spelldefinitions.go`.
2. Add a new case to the `switch spell.EffectType` block in `spells/system.go`.
3. Implement the handler function (e.g., `applyTeleportSpell`).
4. Update `gui/guispells/spell_panel.go:UpdateDetailPanel()` if the new type needs different display logic.

---

## 15. Common Pitfalls and Gotchas

**1. Forgetting the spellbook check in `ExecuteSpellCast`**

`ExecuteSpellCast` validates that the spell is in the caster's spellbook even after mana is checked. This is intentional: mana is a shared pool and could theoretically be validated externally, but the spellbook check prevents casting spells the commander hasn't learned. Do not remove this check when calling `ExecuteSpellCast` programmatically.

**2. Buff/debuff spells need both `duration` and `statModifiers`**

If a buff/debuff spell is missing `duration` (defaults to 0) or `statModifiers` (defaults to empty slice), `applyBuffDebuffSpell` will create zero effects or create effects that expire immediately. The spell will appear to succeed (returns `Success = true` after mana deduction) but have no observable outcome.

**3. ParseStatType is case-insensitive but whitespace-sensitive**

`"MovementSpeed"` works. `"movement speed"` (with a space) does not. The JSON data file uses `"movementspeed"` (all lowercase, no spaces). Do not use `"MovementSpeed"` or `"movement_speed"` in JSON data.

**4. Effect modifiers are not clamped**

`applyModifierToStat` directly adds the modifier to the attribute field without clamping. A large negative debuff on Strength could drive it negative. The only protection is that damage calculations use derived methods (`GetPhysicalDamage`, etc.) that compute from the raw attribute values — negative Strength produces negative physical damage, which is not specifically handled. Design spells so that modifiers combined with likely base values stay in reasonable ranges.

**5. `RemoveAllEffects` is not called automatically on combat exit**

The current system does not automatically reverse active effects when combat ends. If a unit has an active `SourceSpell` effect with several turns remaining when combat exits, their attributes remain modified. This matters for any stats that persist meaningfully across combat (which, at present, is all of them). If this is a concern, call `RemoveAllEffects` on all units during the post-combat cleanup phase.

**6. The two `effects` packages have identical APIs but separate ECS components**

`tactical/powers/effects.ActiveEffectsComponent` and `tactical/powers/artifacts/effects.ActiveEffectsComponent` are different ECS components. A unit can theoretically have one from each package. The `TurnManager` calls `tactical/powers/effects.TickEffectsForUnits`, so only effects applied via the spells-package effects system are ticked. Artifact effects (permanent, source `SourceItem`) are not affected by ticking, so this duplication is currently harmless.

**7. AoE targeting uses screen coordinates for shape positioning**

`HandleAoETargetingFrame` converts mouse position to logical coordinates then back to pixel using `CoordManager.LogicalToPixel`. This snaps the shape to tile boundaries. Do not use raw mouse pixel coordinates for shape positioning — they don't snap to tiles and cause the shape to drift off-grid.

---

## 16. Appendix: Full Data Structures

### SpellDefinition (templates/spelldefinitions.go)

```go
type SpellDefinition struct {
    ID            string
    Name          string
    Description   string
    ManaCost      int
    Damage        int
    TargetType    SpellTargetType    // "single" | "aoe"
    EffectType    SpellEffectType    // "damage" | "buff" | "debuff"
    Shape         *JSONTargetArea    // nil for single-target spells
    VXType        string             // "fire" | "electricity" | "ice" | "cloud"
    VXDuration    int
    Duration      int                // turns; 0 for damage spells
    StatModifiers []SpellStatModifier
}

type SpellStatModifier struct {
    Stat     string
    Modifier int
}

type JSONTargetArea struct {
    Type   string
    Size   int
    Length int
    Width  int
    Height int
    Radius int
}
```

### ManaData, SpellBookData (tactical/powers/spells/components.go)

```go
type ManaData struct {
    CurrentMana int
    MaxMana     int
}

type SpellBookData struct {
    SpellIDs []string
}

type SpellCastResult struct {
    Success          bool
    ErrorReason      string
    TotalDamageDealt int
    AffectedSquadIDs []ecs.EntityID
    SquadsDestroyed  []ecs.EntityID
}
```

### ActiveEffect, ActiveEffectsData (tactical/powers/effects/components.go)

```go
type StatType int   // StatStrength, StatDexterity, ... StatAttackRange

type EffectSource int  // SourceSpell, SourceAbility, SourceItem

type ActiveEffect struct {
    Name           string
    Source         EffectSource
    Stat           StatType
    Modifier       int
    RemainingTurns int   // -1 = permanent
}

type ActiveEffectsData struct {
    Effects []ActiveEffect
}
```

### TacticalState spell fields (gui/framework/contextstate.go)

```go
type TacticalState struct {
    // ... (other fields)
    InSpellMode     bool
    SelectedSpellID string
    HasCastSpell    bool
    // ...
}
```

---

*For ECS architecture fundamentals, see `docs/project_documentation/ECS_BEST_PRACTICES.md`.*
*For artifact behavior documentation, see the artifacts package source at `tactical/powers/artifacts/`.*
