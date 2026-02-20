# Perk System: Table-Driven Hook Functions (Approach A)

Implementation plan for TinkerRogue's perk/trait system using table-driven hook functions.
Cross-references the ~50 perks from `docs/perk_system_brainstorm.md` against existing codebase systems.

**Goal:** Minimize changes to existing code, leverage existing patterns, follow Go idioms.

---

## Table of Contents

1. [Perk Tier Classification](#1-perk-tier-classification)
2. [Foundation](#2-foundation)
3. [Hook Types](#3-hook-types)
4. [Hook Registry](#4-hook-registry)
5. [Hook Runner Functions](#5-hook-runner-functions)
6. [Perk ID Collection](#6-perk-id-collection)
7. [Example Perk Behaviors](#7-example-perk-behaviors)
8. [Existing Code Changes](#8-existing-code-changes)
9. [Combat Integration Strategy](#9-combat-integration-strategy)
10. [Minimal Invasion Accounting](#10-minimal-invasion-accounting)
11. [Data-Driven JSON Schema](#11-data-driven-json-schema)
12. [Implementation Order](#12-implementation-order)
13. [Go Design Pattern Analysis](#13-go-design-pattern-analysis)
14. [Implementation Landmines](#14-implementation-landmines)
15. [Stacking Rules](#15-stacking-rules)
16. [Testing Strategy](#16-testing-strategy)

---

## 1. Perk Tier Classification

Every perk from the brainstorm document classified by implementation complexity.

### Tier 1: Pure Stat Modifications (13 perks) -- Trivial

These use the existing `ActiveEffect` system (`tactical/effects/`) with `RemainingTurns = -1` (permanent). **Zero new infrastructure needed.** All derived combat stats (physical damage, hit rate, crit chance, dodge) are computed on-the-fly from `Attributes`, so modifying base attributes via effects automatically flows through all calculations and AI threat evaluation.

| Perk | Level | Effect | Implementation |
|------|-------|--------|----------------|
| Iron Constitution | Unit | +20% max HP | `ActiveEffect{Stat: StatStrength, Modifier: str*0.2}` |
| Quick Reflexes | Unit | +15% dodge | `ActiveEffect{Stat: StatDexterity, Modifier: dex*0.15}` |
| Precise Strikes | Unit | +10% hit, +10% crit | Two effects: StatDexterity |
| Hardened Armor | Unit | +4 armor | `ActiveEffect{Stat: StatArmor, Modifier: 4}` |
| Arcane Attunement | Unit | +5 magic | `ActiveEffect{Stat: StatMagic, Modifier: 5}` |
| Fleet Footed | Unit | +1 movement | `ActiveEffect{Stat: StatMovementSpeed, Modifier: 1}` |
| Forced March | Squad | +1 squad movement | Apply to all units: StatMovementSpeed |
| Efficient Casting | Commander | -20% mana cost | Modify mana cost before deduction |
| Spell Mastery | Commander | +25% spell damage | Modify damage before application |
| Overcharge | Commander | +50% mana, +75% dmg | Modify both values |
| Lingering Magic | Commander | +2 turn duration | Modify effect duration |
| Potent Enchantment | Commander | +50% buff/debuff values | Modify stat modifiers |
| Mana Regeneration | Commander | +10% mana per battle end | Post-battle recovery |

### Tier 2: Modify Existing Calculations (14 perks) -- Moderate

These modify values that already flow through `calculateDamage()` or related functions. They need hook points but each hook is a specific, identifiable line of code.

| Perk | Level | What Changes | Hook Location |
|------|-------|-------------|---------------|
| Glass Cannon | Squad | +35% dmg dealt, +20% taken | `DamageModifiers.DamageMultiplier` (both sides) |
| Ambush Doctrine | Squad | +50% first attack | `DamageModifiers.DamageMultiplier` (turn 1 check) |
| Wolf Pack | Squad | +5% dmg per DPS | `DamageModifiers.DamageMultiplier` (count-based) |
| Berserker | Unit | +30% dmg below 50% HP | `DamageModifiers.DamageMultiplier` (HP check) |
| Sharpshooter | Unit | +1 range, +15% crit | Stat effect + range component |
| Armor Piercing | Unit | Ignore 50% armor | `DamageModifiers.ArmorReduction` (new field) |
| Executioner | Unit | +50% vs <30% HP | `DamageModifiers.DamageMultiplier` (target HP check) |
| Reckless Assault | Unit | +30% dmg, no counter | `DamageModifiers` + skip counter flag |
| Vengeful Strike | Unit | Counter +20% crit | Modify crit threshold in counter path |
| Suppressing Fire | Unit | Apply -15% hit debuff | `PostDamageHook` applies `ActiveEffect` |
| Shock and Awe | Squad | Turn 1: +25% dmg, +15% hit | `DamageModifiers` (turn check) |
| Adrenaline | Unit | +5% per turn stacking | `TurnStartHook` increments effect |
| Dig In | Squad | +5% def per stationary turn | `TurnStartHook` checks movement |
| Fortified Position | Squad | +0.1 cover when stationary | Cover mod (movement check) |

### Tier 3: Change Behavior (13 perks) -- Complex

These alter targeting, damage routing, or combat flow. They need explicit code hooks that modify how combat resolves.

| Perk | Level | Behavior Change | Integration Point |
|------|-------|----------------|-------------------|
| Cleave | Unit | Hit target row + row behind | `TargetOverrideHook` in `SelectTargetUnits` |
| Impale | Unit | MeleeColumn ignores cover | `CoverModHook` zeroes cover |
| Focus Fire | Unit | Single target at 2x damage | `TargetOverrideHook` + `DamageModHook` |
| Scatter Shot | Unit | Hit target row + adjacent at 70% | `TargetOverrideHook` + split damage |
| Chain Cast | Unit | Magic bounces +1 cell | `TargetOverrideHook` for magic type |
| Riposte | Unit | Counter at 100% damage | `CounterModHook` overrides multiplier |
| Stone Wall | Unit | No counter, -30% dmg taken | `CounterModHook` + `DamageModHook` |
| Lifesteal | Unit | Heal 25% of dmg dealt | `PostDamageHook` modifies HP |
| Guardian | Unit | Takes 50% of adjacent ally dmg | Damage redirect before recording |
| Inspiration | Unit | On kill: +2 str allies 2 turns | `PostDamageHook` (checks wasKill) |
| Last Stand | Unit | Last alive: +50% stats | `DamageModHook` (checks unit count) |
| Retribution | Unit | On ally death: +50% next atk | Death event tracking + temp buff |
| Double Strike | Unit | 25% chance attack twice | `PostDamageHook` schedules extra attack |

### Tier 4: New Mechanics (10 perks) -- Very Complex

These introduce mechanics that don't exist yet: formation swapping, turn order manipulation, conditional state machines.

| Perk | Level | New Mechanic | Complexity Notes |
|------|-------|-------------|------------------|
| Preemptive Strike | Unit | Damage BEFORE attacker resolves | Reverse attack order in `ExecuteAttackAction` |
| Overwatch | Unit | Skip attack, 150% counter next turn | New state tracking across turns |
| Vanguard | Unit | Always targeted first (taunt) | Target selection override |
| Bulwark | Unit | +50% cover, can't deal damage | Cover mod + attack suppression |
| War Medic | Unit | Heal 3 HP to lowest ally/turn | `TurnStartHook` (straightforward implementation) |
| Combined Arms | Squad | +10% dmg/def if 3 roles present | Role-counting condition + multi-stat effect |
| Flexible Doctrine | Squad | Switch formation once per battle | New action type in combat UI |
| Reserves | Squad | +2 capacity, -5% stats | Squad creation modifier |
| Rally Commander | Commander | Reset one squad's action | New action in combat action system |
| Battle Sense | Commander | See enemy compositions | GUI/info layer change |

---

## 2. Foundation

### 2.1 Package Structure

```
tactical/perks/
    init.go              -- Subsystem registration (RegisterSubsystem pattern)
    components.go        -- ECS component definitions (pure data)
    perkdefinition.go    -- PerkDefinition struct, JSON loading
    registry.go          -- PerkRegistry global map, LoadPerkDefinitions()
    hooks.go             -- Hook function type definitions
    hook_registry.go     -- PerkHooks struct, RegisterPerkHooks, GetPerkHooks
    behaviors.go         -- Individual perk behavior implementations
    system.go            -- ApplyStatPerks, RemoveStatPerks, equip/unequip
    queries.go           -- HasPerk, GetEquippedPerks, getActivePerkIDs, hook runners
```

### 2.2 PerkDefinition (Data-Driven, mirrors SpellDefinition)

Follows the exact pattern from `templates/spelldefinitions.go:34-47`:

```go
// tactical/perks/perkdefinition.go

type PerkLevel int
const (
    PerkLevelSquad     PerkLevel = iota // Equipped on squad entity
    PerkLevelUnit                       // Equipped on unit entity
    PerkLevelCommander                  // Equipped on commander entity
)

type PerkCategory int
const (
    CategorySpecialization PerkCategory = iota
    CategoryGeneralization
    CategoryAttackPattern
    CategoryAttribute
    CategoryAttackCounter
    CategoryDepth
    CategoryCommander
)

// PerkDefinition is a static blueprint loaded from JSON.
// Analogous to SpellDefinition in templates/spelldefinitions.go.
type PerkDefinition struct {
    ID            string       `json:"id"`
    Name          string       `json:"name"`
    Description   string       `json:"description"`
    Level         PerkLevel    `json:"level"`
    Category      PerkCategory `json:"category"`
    RoleGate      string       `json:"roleGate"`       // "" = any, "Tank", "DPS", "Support"
    ExclusiveWith []string     `json:"exclusiveWith"`
    UnlockCost    int          `json:"unlockCost"`

    // Stat modification (Tier 1 perks)
    StatModifiers []PerkStatModifier `json:"statModifiers,omitempty"`

    // Behavioral hook key (Tier 2-4 perks)
    BehaviorID string         `json:"behaviorId,omitempty"`
    Params     map[string]any `json:"params,omitempty"`
}

type PerkStatModifier struct {
    Stat     string  `json:"stat"`
    Modifier int     `json:"modifier,omitempty"`
    Percent  float64 `json:"percent,omitempty"`
}
```

### 2.3 ECS Components

```go
// tactical/perks/components.go

type SquadPerkData struct {
    EquippedPerks [3]string // Up to 3 perk IDs ("" = empty slot)
}

type UnitPerkData struct {
    EquippedPerks [2]string // Up to 2 perk IDs
}

type CommanderPerkData struct {
    EquippedPerks [3]string // Up to 3 perk IDs
}

type PerkUnlockData struct {
    UnlockedPerks map[string]bool // Perk IDs that have been unlocked
    PerkPoints    int             // Available points to spend
}

var (
    SquadPerkComponent     *ecs.Component
    UnitPerkComponent      *ecs.Component
    CommanderPerkComponent *ecs.Component
    PerkUnlockComponent    *ecs.Component
)
```

### 2.4 Subsystem Registration

Follows the `init()` pattern used by all existing subsystems:

```go
// tactical/perks/init.go
func init() {
    common.RegisterSubsystem(func(em *common.EntityManager) {
        SquadPerkComponent = em.World.NewComponent()
        UnitPerkComponent = em.World.NewComponent()
        CommanderPerkComponent = em.World.NewComponent()
        PerkUnlockComponent = em.World.NewComponent()
    })
}
```

### 2.5 Perk Registry

Mirrors `templates/spelldefinitions.go:50-55`:

```go
// tactical/perks/registry.go

var PerkRegistry = make(map[string]*PerkDefinition)

func GetPerkDefinition(id string) *PerkDefinition {
    return PerkRegistry[id]
}

func LoadPerkDefinitions() {
    // Read from assets/gamedata/perkdata.json
    // Parse JSON, populate PerkRegistry
    // Validate: no duplicate IDs, exclusive pairs symmetric, role gates valid
}
```

### 2.6 Required Change to Effects System (1 line)

```go
// In tactical/effects/components.go:26-29, add SourcePerk:
const (
    SourceSpell   EffectSource = iota
    SourceAbility
    SourcePerk    // NEW -- 1 line addition
)
```

### 2.7 Stat Perk Application

```go
// tactical/perks/system.go

func ApplyStatPerks(entityID ecs.EntityID, perkIDs []string, manager *common.EntityManager) {
    for _, perkID := range perkIDs {
        def := GetPerkDefinition(perkID)
        if def == nil || len(def.StatModifiers) == 0 {
            continue
        }
        entity := manager.FindEntityByID(entityID)
        if entity == nil {
            continue
        }
        attr := common.GetComponentType[*common.Attributes](entity, common.AttributeComponent)
        if attr == nil {
            continue
        }

        for _, mod := range def.StatModifiers {
            modifier := mod.Modifier
            if mod.Percent != 0 {
                baseStat := getBaseStat(attr, mod.Stat)
                modifier = int(float64(baseStat) * mod.Percent)
            }

            effect := effects.ActiveEffect{
                Name:           def.Name,
                Source:         effects.SourcePerk,
                Stat:           effects.ParseStatType(mod.Stat),
                Modifier:       modifier,
                RemainingTurns: -1, // Permanent
            }
            effects.ApplyEffect(entityID, effect, manager)
        }
    }
}
```

**Note on perk removal:** The existing `reverseModifierFromStat` in `tactical/effects/system.go:158` is **unexported** (lowercase). To remove perk effects by source, either:
1. Add an exported `RemoveEffectsBySource(entityID, source, manager)` function to the effects package (cleanest), or
2. Use `effects.RemoveAllEffects()` and re-apply non-perk effects (wasteful), or
3. Put perk removal logic inside the effects package itself

Option 1 is recommended -- add ~20 lines to `tactical/effects/system.go`:

```go
func RemoveEffectsBySource(entityID ecs.EntityID, source EffectSource, manager *common.EntityManager) {
    entity := manager.FindEntityByID(entityID)
    if entity == nil || !entity.HasComponent(ActiveEffectsComponent) {
        return
    }
    effectsData := common.GetComponentType[*ActiveEffectsData](entity, ActiveEffectsComponent)
    attr := common.GetComponentType[*common.Attributes](entity, common.AttributeComponent)
    if effectsData == nil || attr == nil {
        return
    }
    kept := effectsData.Effects[:0]
    for i := range effectsData.Effects {
        e := &effectsData.Effects[i]
        if e.Source == source {
            reverseModifierFromStat(attr, e.Stat, e.Modifier)
        } else {
            kept = append(kept, *e)
        }
    }
    effectsData.Effects = kept
}
```

---

## 3. Hook Types

Six typed function signatures covering all perk categories, plus a 7th for Guardian's damage redirect.

```go
// tactical/perks/hooks.go

// DamageModHook modifies damage modifiers before calculation.
// Called inside calculateDamage() for the attacking unit.
type DamageModHook func(
    attackerID, defenderID ecs.EntityID,
    modifiers *squads.DamageModifiers,
    manager *common.EntityManager,
)

// TargetOverrideHook overrides target selection.
// Returns nil to use default targeting.
type TargetOverrideHook func(
    attackerID, defenderSquadID ecs.EntityID,
    defaultTargets []ecs.EntityID,
    manager *common.EntityManager,
) []ecs.EntityID

// CounterModHook modifies counterattack behavior.
// Return skipCounter=true to suppress counterattack entirely.
type CounterModHook func(
    defenderID, attackerID ecs.EntityID,
    modifiers *squads.DamageModifiers,
    manager *common.EntityManager,
) (skipCounter bool)

// PostDamageHook runs after damage is recorded for a single attack.
type PostDamageHook func(
    attackerID, defenderID ecs.EntityID,
    damageDealt int, wasKill bool,
    manager *common.EntityManager,
)

// TurnStartHook runs at the start of a squad's turn.
type TurnStartHook func(
    squadID ecs.EntityID,
    manager *common.EntityManager,
)

// CoverModHook modifies cover calculation for a defender.
type CoverModHook func(
    attackerID, defenderID ecs.EntityID,
    coverBreakdown *squads.CoverBreakdown,
    manager *common.EntityManager,
)

// DamageRedirectHook intercepts damage before recordDamageToUnit.
// Returns reduced damage for original target, plus a redirect target and amount.
// Required for Guardian perk.
type DamageRedirectHook func(
    defenderID ecs.EntityID,
    damageAmount int,
    manager *common.EntityManager,
) (reducedDamage int, redirectTargetID ecs.EntityID, redirectAmount int)
```

---

## 4. Hook Registry

```go
// tactical/perks/hook_registry.go

// PerkHooks collects all hooks for a single perk.
// A perk only populates the hooks it needs -- nil slots are skipped.
type PerkHooks struct {
    DamageMod       DamageModHook
    TargetOverride  TargetOverrideHook
    CounterMod      CounterModHook
    PostDamage      PostDamageHook
    TurnStart       TurnStartHook
    CoverMod        CoverModHook
    DamageRedirect  DamageRedirectHook
}

var hookRegistry = map[string]*PerkHooks{}

func RegisterPerkHooks(perkID string, hooks *PerkHooks) {
    hookRegistry[perkID] = hooks
}

func GetPerkHooks(perkID string) *PerkHooks {
    return hookRegistry[perkID]
}

func init() {
    RegisterPerkHooks("riposte", &PerkHooks{CounterMod: riposteCounterMod})
    RegisterPerkHooks("armor_piercing", &PerkHooks{DamageMod: armorPiercingDamageMod})
    RegisterPerkHooks("focus_fire", &PerkHooks{
        TargetOverride: focusFireTargetOverride,
        DamageMod:      focusFireDamageMod,
    })
    RegisterPerkHooks("lifesteal", &PerkHooks{PostDamage: lifestealPostDamage})
    RegisterPerkHooks("stone_wall", &PerkHooks{
        CounterMod: stoneWallCounterMod,
        DamageMod:  stoneWallDamageMod,
    })
    RegisterPerkHooks("cleave", &PerkHooks{TargetOverride: cleaveTargetOverride})
    RegisterPerkHooks("impale", &PerkHooks{CoverMod: impaleCoverMod})
    RegisterPerkHooks("berserker", &PerkHooks{DamageMod: berserkerDamageMod})
    RegisterPerkHooks("inspiration", &PerkHooks{PostDamage: inspirationPostDamage})
    RegisterPerkHooks("war_medic", &PerkHooks{TurnStart: warMedicTurnStart})
    // ... etc.
}
```

---

## 5. Hook Runner Functions

```go
// tactical/perks/queries.go

// RunDamageModHooks runs all DamageMod hooks for an attacker's perks.
func RunDamageModHooks(attackerID, defenderID ecs.EntityID,
    modifiers *squads.DamageModifiers, manager *common.EntityManager) {
    for _, perkID := range getActivePerkIDs(attackerID, manager) {
        hooks := GetPerkHooks(perkID)
        if hooks != nil && hooks.DamageMod != nil {
            hooks.DamageMod(attackerID, defenderID, modifiers, manager)
        }
    }
}

// RunDefenderDamageModHooks runs hooks for the DEFENDER's perks
// (e.g., Stone Wall reducing damage taken).
func RunDefenderDamageModHooks(attackerID, defenderID ecs.EntityID,
    modifiers *squads.DamageModifiers, manager *common.EntityManager) {
    for _, perkID := range getActivePerkIDs(defenderID, manager) {
        hooks := GetPerkHooks(perkID)
        if hooks != nil && hooks.DamageMod != nil {
            hooks.DamageMod(attackerID, defenderID, modifiers, manager)
        }
    }
}

// RunTargetOverrideHooks applies target overrides from attacker perks.
func RunTargetOverrideHooks(attackerID, defenderSquadID ecs.EntityID,
    targets []ecs.EntityID, manager *common.EntityManager) []ecs.EntityID {
    for _, perkID := range getActivePerkIDs(attackerID, manager) {
        hooks := GetPerkHooks(perkID)
        if hooks != nil && hooks.TargetOverride != nil {
            targets = hooks.TargetOverride(attackerID, defenderSquadID, targets, manager)
        }
    }
    return targets
}

// RunCounterModHooks checks if counterattack should be suppressed or modified.
func RunCounterModHooks(defenderID, attackerID ecs.EntityID,
    modifiers *squads.DamageModifiers, manager *common.EntityManager) bool {
    for _, perkID := range getActivePerkIDs(defenderID, manager) {
        hooks := GetPerkHooks(perkID)
        if hooks != nil && hooks.CounterMod != nil {
            if hooks.CounterMod(defenderID, attackerID, modifiers, manager) {
                return true // Skip counter
            }
        }
    }
    return false
}

// RunPostDamageHooks runs post-damage hooks for the attacker.
func RunPostDamageHooks(attackerID, defenderID ecs.EntityID,
    damageDealt int, wasKill bool, manager *common.EntityManager) {
    for _, perkID := range getActivePerkIDs(attackerID, manager) {
        hooks := GetPerkHooks(perkID)
        if hooks != nil && hooks.PostDamage != nil {
            hooks.PostDamage(attackerID, defenderID, damageDealt, wasKill, manager)
        }
    }
}

// RunTurnStartHooks runs turn-start hooks for all units in a squad.
func RunTurnStartHooks(squadID ecs.EntityID, manager *common.EntityManager) {
    unitIDs := squads.GetUnitIDsInSquad(squadID, manager)
    for _, unitID := range unitIDs {
        for _, perkID := range getActivePerkIDs(unitID, manager) {
            hooks := GetPerkHooks(perkID)
            if hooks != nil && hooks.TurnStart != nil {
                hooks.TurnStart(squadID, manager)
            }
        }
    }
    // Also run squad-level turn start hooks
    for _, perkID := range getSquadPerkIDs(squadID, manager) {
        hooks := GetPerkHooks(perkID)
        if hooks != nil && hooks.TurnStart != nil {
            hooks.TurnStart(squadID, manager)
        }
    }
}

// RunCoverModHooks runs cover modification hooks.
func RunCoverModHooks(attackerID, defenderID ecs.EntityID,
    coverBreakdown *squads.CoverBreakdown, manager *common.EntityManager) {
    // Check attacker perks (e.g., Impale ignores cover)
    for _, perkID := range getActivePerkIDs(attackerID, manager) {
        hooks := GetPerkHooks(perkID)
        if hooks != nil && hooks.CoverMod != nil {
            hooks.CoverMod(attackerID, defenderID, coverBreakdown, manager)
        }
    }
}
```

---

## 6. Perk ID Collection

```go
// getActivePerkIDs returns all perk IDs from unit perks + parent squad perks.
func getActivePerkIDs(unitID ecs.EntityID, manager *common.EntityManager) []string {
    var ids []string
    // Unit perks
    if data := common.GetComponentTypeByID[*UnitPerkData](
        manager, unitID, UnitPerkComponent,
    ); data != nil {
        for _, id := range data.EquippedPerks {
            if id != "" { ids = append(ids, id) }
        }
    }
    // Squad perks (from parent squad)
    if memberData := common.GetComponentTypeByID[*squads.SquadMemberData](
        manager, unitID, squads.SquadMemberComponent,
    ); memberData != nil {
        ids = append(ids, getSquadPerkIDs(memberData.SquadID, manager)...)
    }
    return ids
}

func getSquadPerkIDs(squadID ecs.EntityID, manager *common.EntityManager) []string {
    var ids []string
    if data := common.GetComponentTypeByID[*SquadPerkData](
        manager, squadID, SquadPerkComponent,
    ); data != nil {
        for _, id := range data.EquippedPerks {
            if id != "" { ids = append(ids, id) }
        }
    }
    return ids
}
```

---

## 7. Example Perk Behaviors

```go
// tactical/perks/behaviors.go

// --- Riposte: Counterattacks deal 100% damage ---
func riposteCounterMod(defenderID, attackerID ecs.EntityID,
    modifiers *squads.DamageModifiers, manager *common.EntityManager) bool {
    modifiers.DamageMultiplier = 1.0 // Override 0.5 default
    modifiers.HitPenalty = 0         // Override -20 default
    return false                     // Don't skip counter
}

// --- Armor Piercing: Halve effective armor ---
func armorPiercingDamageMod(attackerID, defenderID ecs.EntityID,
    modifiers *squads.DamageModifiers, manager *common.EntityManager) {
    modifiers.ArmorReduction = 0.5 // New field on DamageModifiers
}

// --- Focus Fire: Single target at 2x damage ---
func focusFireTargetOverride(attackerID, defenderSquadID ecs.EntityID,
    defaultTargets []ecs.EntityID, manager *common.EntityManager) []ecs.EntityID {
    if len(defaultTargets) > 0 {
        return defaultTargets[:1]
    }
    return defaultTargets
}

func focusFireDamageMod(attackerID, defenderID ecs.EntityID,
    modifiers *squads.DamageModifiers, manager *common.EntityManager) {
    modifiers.DamageMultiplier *= 2.0
}

// --- Lifesteal: Heal 25% of damage dealt ---
func lifestealPostDamage(attackerID, defenderID ecs.EntityID,
    damageDealt int, wasKill bool, manager *common.EntityManager) {
    healAmount := damageDealt / 4
    if healAmount < 1 { healAmount = 1 }
    attr := common.GetComponentTypeByID[*common.Attributes](
        manager, attackerID, common.AttributeComponent,
    )
    if attr != nil {
        attr.CurrentHealth += healAmount
        if attr.CurrentHealth > attr.MaxHealth {
            attr.CurrentHealth = attr.MaxHealth
        }
    }
}

// --- Impale: MeleeColumn ignores cover ---
func impaleCoverMod(attackerID, defenderID ecs.EntityID,
    coverBreakdown *squads.CoverBreakdown, manager *common.EntityManager) {
    coverBreakdown.TotalReduction = 0
    coverBreakdown.Providers = nil
}

// --- Stone Wall: No counter, -30% damage taken ---
func stoneWallCounterMod(defenderID, attackerID ecs.EntityID,
    modifiers *squads.DamageModifiers, manager *common.EntityManager) bool {
    return true // Skip counterattack entirely
}

func stoneWallDamageMod(attackerID, defenderID ecs.EntityID,
    modifiers *squads.DamageModifiers, manager *common.EntityManager) {
    modifiers.DamageMultiplier *= 0.7 // -30% damage taken
}

// --- Berserker: +30% damage below 50% HP ---
func berserkerDamageMod(attackerID, defenderID ecs.EntityID,
    modifiers *squads.DamageModifiers, manager *common.EntityManager) {
    attr := common.GetComponentTypeByID[*common.Attributes](
        manager, attackerID, common.AttributeComponent,
    )
    if attr != nil && float64(attr.CurrentHealth)/float64(attr.MaxHealth) < 0.5 {
        modifiers.DamageMultiplier *= 1.3
    }
}

// --- Inspiration: On kill, buff squad ---
func inspirationPostDamage(attackerID, defenderID ecs.EntityID,
    damageDealt int, wasKill bool, manager *common.EntityManager) {
    if !wasKill { return }
    memberData := common.GetComponentTypeByID[*squads.SquadMemberData](
        manager, attackerID, squads.SquadMemberComponent,
    )
    if memberData == nil { return }
    unitIDs := squads.GetUnitIDsInSquad(memberData.SquadID, manager)
    effect := effects.ActiveEffect{
        Name: "Inspiration", Source: effects.SourcePerk,
        Stat: effects.StatStrength, Modifier: 2, RemainingTurns: 2,
    }
    effects.ApplyEffectToUnits(unitIDs, effect, manager)
}

// --- War Medic: Heal lowest HP ally each turn ---
func warMedicTurnStart(squadID ecs.EntityID, manager *common.EntityManager) {
    unitIDs := squads.GetUnitIDsInSquad(squadID, manager)
    var lowestID ecs.EntityID
    lowestHP := 999999
    for _, uid := range unitIDs {
        attr := common.GetComponentTypeByID[*common.Attributes](
            manager, uid, common.AttributeComponent,
        )
        if attr != nil && attr.CurrentHealth > 0 && attr.CurrentHealth < lowestHP {
            lowestHP = attr.CurrentHealth
            lowestID = uid
        }
    }
    if lowestID != 0 {
        attr := common.GetComponentTypeByID[*common.Attributes](
            manager, lowestID, common.AttributeComponent,
        )
        if attr != nil {
            attr.CurrentHealth += 3
            if attr.CurrentHealth > attr.MaxHealth {
                attr.CurrentHealth = attr.MaxHealth
            }
        }
    }
}

// --- Cleave: Hit target row + row behind ---
func cleaveTargetOverride(attackerID, defenderSquadID ecs.EntityID,
    defaultTargets []ecs.EntityID, manager *common.EntityManager) []ecs.EntityID {
    // Only applies to melee row attackers
    targetData := common.GetComponentTypeByID[*squads.TargetRowData](
        manager, attackerID, squads.TargetRowComponent,
    )
    if targetData == nil || targetData.AttackType != squads.AttackTypeMeleeRow {
        return defaultTargets
    }

    // Find what row the default targets are in
    if len(defaultTargets) == 0 { return defaultTargets }
    pos := common.GetComponentTypeByID[*squads.GridPositionData](
        manager, defaultTargets[0], squads.GridPositionComponent,
    )
    if pos == nil { return defaultTargets }

    // Add units from the next row
    nextRow := pos.AnchorRow + 1
    if nextRow <= 2 {
        extraTargets := squads.GetUnitsInRow(defenderSquadID, nextRow, manager)
        return append(defaultTargets, extraTargets...)
    }
    return defaultTargets
}

// --- Guardian: Redirect 50% damage to adjacent ally ---
func guardianDamageRedirect(defenderID ecs.EntityID,
    damageAmount int, manager *common.EntityManager) (int, ecs.EntityID, int) {
    memberData := common.GetComponentTypeByID[*squads.SquadMemberData](
        manager, defenderID, squads.SquadMemberComponent,
    )
    if memberData == nil { return damageAmount, 0, 0 }

    unitIDs := squads.GetUnitIDsInSquad(memberData.SquadID, manager)
    for _, uid := range unitIDs {
        if uid == defenderID { continue }
        if !HasPerk(uid, "guardian", manager) { continue }
        if !isAdjacent(uid, defenderID, manager) { continue }

        // Split damage: 50% to guardian, 50% to original target
        guardianDmg := damageAmount / 2
        remainingDmg := damageAmount - guardianDmg
        return remainingDmg, uid, guardianDmg
    }
    return damageAmount, 0, 0
}
```

---

## 8. Existing Code Changes

### `tactical/squads/squadcombat.go` -- DamageModifiers struct (line 16):
```go
type DamageModifiers struct {
    HitPenalty       int
    DamageMultiplier float64
    IsCounterattack  bool
    ArmorReduction   float64 // NEW: 0.0=full armor, 0.5=halve, 1.0=ignore
}
```

### `tactical/squads/squadcombat.go` -- calculateDamage() (line 147):

> **Landmine note:** These hook calls cannot directly reference `perks.Run*` from `squadcombat.go` due to circular imports (see [Section 13.1](#131-circular-import-prevention) and [Landmine 1](#landmine-1-hook-placement-precision)). The recommended solution is **callback injection**: define hook runner function types in the `squads` package, pass implementations from `combat/combatactionsystem.go`. The pseudo-code below shows logical placement, not literal import paths.

```go
// CRITICAL: Hooks must run BEFORE line 232 (baseDamage * DamageMultiplier),
// not after resistance calc. Berserker/Glass Cannon modify the multiplier,
// so they must execute before it is applied.

// Line ~230 (after hit/dodge/crit rolls succeed, BEFORE multiplier application):
damageModRunner(attackerID, defenderID, &modifiers, squadmanager)      // attacker perks
defenderDamageModRunner(attackerID, defenderID, &modifiers, squadmanager) // defender perks

// Line 232: baseDamage = int(float64(baseDamage) * modifiers.DamageMultiplier) -- EXISTING

// Line ~238 (after resistance calc):
if modifiers.ArmorReduction > 0 {
    resistance = int(float64(resistance) * (1.0 - modifiers.ArmorReduction))
}

// Line ~245 (after cover calculation):
coverBreakdown := CalculateCoverBreakdown(defenderID, squadmanager)
coverModRunner(attackerID, defenderID, &coverBreakdown, squadmanager)
```

### `tactical/squads/squadcombat.go` -- processAttackWithModifiers() (line 101):

> **Landmine note:** The current signature does NOT include `defenderSquadID`, but `TargetOverrideHook` needs it. This requires a **signature change** and updating all callers (`ProcessAttackOnTargets`, `ProcessCounterattackOnTargets`).

```go
// Signature change: add defenderSquadID parameter
func processAttackWithModifiers(attackerID ecs.EntityID,
    defenderSquadID ecs.EntityID, // NEW PARAMETER
    targetIDs []ecs.EntityID, ...) int {

    // NEW: Run target override hooks (via callback injection)
    targetIDs = targetOverrideRunner(attackerID, defenderSquadID, targetIDs, manager)

    for _, defenderID := range targetIDs {
        damage, event := calculateDamage(attackerID, defenderID, modifiers, manager)
        // ... existing code ...
        recordDamageToUnit(defenderID, damage, result, manager)

        // NEW: Post-damage hooks (lifesteal, inspiration, etc.)
        // NOTE: recordDamageToUnit only records to result.DamageByUnit --
        // actual HP changes happen later in ApplyRecordedDamage.
        // Hooks that check HP (Suppressing Fire debuff) see pre-damage HP.
        wasKill := event.WasKilled
        postDamageRunner(attackerID, defenderID, damage, wasKill, manager)

        log.AttackEvents = append(log.AttackEvents, *event)
    }
    return attackIndex
}
```

### `tactical/combat/combatactionsystem.go` -- ExecuteAttackAction() (line 86):
```go
// Before counterattack section (currently line 91):

// NEW: Check if counter should be suppressed by perks
counterModifiers := squads.DamageModifiers{
    HitPenalty: 20, DamageMultiplier: 0.5, IsCounterattack: true,
}
skipCounter := perks.RunCounterModHooks(defenderID, attackerID, &counterModifiers, cas.manager)

if defenderWouldSurvive && !skipCounter {
    // ... existing counterattack code, using counterModifiers ...
}
```

> **Landmine note:** Counter modifiers are currently hardcoded inside `ProcessCounterattackOnTargets` (squadcombat.go line 138), not in `ExecuteAttackAction`. They must be moved up so hooks can modify them before they're passed in. Additionally, `ExecuteSquadCounterattack` (line 96) also calls `ProcessCounterattackOnTargets` with hardcoded modifiers -- both call sites need the same hook treatment.

---

## 9. Combat Integration Strategy

How the hardest perks integrate with the existing combat pipeline.

### 9.1 Cleave (Hit target row + row behind)

**Integration point:** `TargetOverrideHook` after `SelectTargetUnits()` returns.

`selectMeleeRowTargets()` (squadcombat.go:420) currently tries rows 0, 1, 2 and returns the first non-empty row. Cleave needs to also include the next row.

**Existing code impact:** `getUnitsInRow` is currently unexported (lowercase helper). Either export it or add a query function.

### 9.2 Guardian (Takes 50% of damage dealt to adjacent allies)

**Integration point:** Damage redirect before `recordDamageToUnit()`.

This is the most complex perk. It needs to intercept damage BEFORE it's recorded and split it. The `DamageRedirectHook` runs inside `processAttackWithModifiers` after `calculateDamage` returns but before `recordDamageToUnit`. If `redirectTargetID != 0`, record `redirectAmount` as additional damage to the guardian.

**Note:** If Guardian is too complex for v1, **cut it explicitly** from the first implementation phase and document it as deferred.

### 9.3 Riposte (Counterattacks deal 100% damage)

**Integration point:** `CounterModHook` in `ExecuteAttackAction()`.

Currently counterattack modifiers are hardcoded at `combatactionsystem.go:93-113`:

```go
// Current: ProcessCounterattackOnTargets uses fixed modifiers (squadcombat.go:138)
modifiers := DamageModifiers{
    HitPenalty:       counterattackHitPenalty,    // 20
    DamageMultiplier: counterattackDamageMultiplier, // 0.5
    IsCounterattack:  true,
}
```

With perk hooks, this becomes:
```go
counterModifiers := squads.DamageModifiers{
    HitPenalty: 20, DamageMultiplier: 0.5, IsCounterattack: true,
}
skipCounter := perks.RunCounterModHooks(counterAttackerID, attackerID, &counterModifiers, cas.manager)
if !skipCounter {
    // Use counterModifiers (now potentially modified by Riposte)
    processAttackWithModifiers(counterAttackerID, targetIDs, result, log, attackIndex, counterModifiers, manager)
}
```

**Existing code impact:** The counter-modifiers construction moves from `ProcessCounterattackOnTargets` into `ExecuteAttackAction()` so hooks can modify them before they're used. `ProcessCounterattackOnTargets` either takes modifiers as a parameter or is replaced by a direct `processAttackWithModifiers` call.

### 9.4 Focus Fire (Single target at 2x damage)

**Integration point:** `TargetOverrideHook` + `DamageModHook`.

Two hooks work together:
1. `TargetOverrideHook` reduces target list to first target only
2. `DamageModHook` doubles damage multiplier

The hooks are independent -- target override runs in `processAttackWithModifiers` before the damage loop, and damage mod runs inside `calculateDamage` during each iteration.

### 9.5 Preemptive Strike (Damage BEFORE attacker resolves)

**Integration point:** Requires structural change to `ExecuteAttackAction()`.

This is the hardest perk. Currently the flow is:
1. Attacker attacks defender
2. If defender survives, defender counterattacks

With Preemptive Strike, a defending unit with this perk attacks FIRST, before the main attack resolves for that specific unit.

**Recommendation:** Defer Preemptive Strike to a later phase. Implement the ~40 simpler perks first.

---

## 10. Minimal Invasion Accounting

Exact existing file changes required.

| File | Change Description | Lines Added | Lines Modified |
|------|--------------------|-------------|----------------|
| `tactical/effects/components.go` | Add `SourcePerk` to EffectSource enum | +1 | 0 |
| `tactical/effects/system.go` | Add `RemoveEffectsBySource()` export | +20 | 0 |
| `tactical/squads/squadcombat.go` | Add `ArmorReduction` field to `DamageModifiers` | +1 | 0 |
| `tactical/squads/squadcombat.go` | Hook calls in `calculateDamage()` (via callback injection) | +10 | 0 |
| `tactical/squads/squadcombat.go` | Hook calls in `processAttackWithModifiers()` | +5 | 0 |
| `tactical/squads/squadcombat.go` | Add `defenderSquadID` param to `processAttackWithModifiers` | 0 | 3 |
| `tactical/squads/squadcombat.go` | Export `getUnitsInRow` (capitalize) | 0 | 1 |
| `tactical/combat/combatactionsystem.go` | Counter mod hook + modified counter construction | +12 | 2 |
| **Total** | | **~49** | **~6** |

Note: Most changes are **additive insertions**. One function signature changes (`processAttackWithModifiers` gains `defenderSquadID` parameter), which requires updating its callers (`ProcessAttackOnTargets`, `ProcessCounterattackOnTargets`). See [Section 14: Implementation Landmines](#14-implementation-landmines) for details on subtle integration issues.

---

## 11. Data-Driven JSON Schema

### perkdata.json (goes in `assets/gamedata/`)

```json
{
  "perks": [
    {
      "id": "iron_constitution",
      "name": "Iron Constitution",
      "description": "+20% max HP",
      "level": 1,
      "category": 3,
      "roleGate": "",
      "exclusiveWith": [],
      "unlockCost": 2,
      "statModifiers": [
        { "stat": "strength", "percent": 0.2 }
      ]
    },
    {
      "id": "hardened_armor",
      "name": "Hardened Armor",
      "description": "+4 armor",
      "level": 1,
      "category": 3,
      "roleGate": "",
      "exclusiveWith": [],
      "unlockCost": 2,
      "statModifiers": [
        { "stat": "armor", "modifier": 4 }
      ]
    },
    {
      "id": "riposte",
      "name": "Riposte",
      "description": "Counterattacks deal 100% damage instead of 50%",
      "level": 1,
      "category": 4,
      "roleGate": "",
      "exclusiveWith": ["stone_wall"],
      "unlockCost": 3,
      "behaviorId": "riposte"
    },
    {
      "id": "stone_wall",
      "name": "Stone Wall",
      "description": "Can't counterattack, but take 30% less damage",
      "level": 1,
      "category": 4,
      "roleGate": "",
      "exclusiveWith": ["riposte"],
      "unlockCost": 3,
      "behaviorId": "stone_wall"
    },
    {
      "id": "berserker",
      "name": "Berserker",
      "description": "+30% damage when below 50% HP",
      "level": 1,
      "category": 0,
      "roleGate": "DPS",
      "exclusiveWith": [],
      "unlockCost": 3,
      "behaviorId": "berserker"
    },
    {
      "id": "lifesteal",
      "name": "Lifesteal",
      "description": "Heal 25% of damage dealt",
      "level": 1,
      "category": 5,
      "roleGate": "",
      "exclusiveWith": [],
      "unlockCost": 3,
      "behaviorId": "lifesteal",
      "params": { "healPercent": 0.25 }
    },
    {
      "id": "focus_fire",
      "name": "Focus Fire",
      "description": "Hit single target at 2x damage instead of row/column",
      "level": 1,
      "category": 2,
      "roleGate": "",
      "exclusiveWith": ["cleave", "scatter_shot"],
      "unlockCost": 3,
      "behaviorId": "focus_fire"
    },
    {
      "id": "glass_cannon",
      "name": "Glass Cannon",
      "description": "+35% damage dealt, +20% damage taken",
      "level": 0,
      "category": 0,
      "roleGate": "",
      "exclusiveWith": ["shield_wall"],
      "unlockCost": 3,
      "behaviorId": "glass_cannon",
      "params": { "damageBonus": 0.35, "damageTakenIncrease": 0.2 }
    },
    {
      "id": "war_medic",
      "name": "War Medic",
      "description": "Heals 3 HP to lowest-HP ally each turn (passive)",
      "level": 1,
      "category": 5,
      "roleGate": "Support",
      "exclusiveWith": [],
      "unlockCost": 3,
      "behaviorId": "war_medic",
      "params": { "healAmount": 3 }
    },
    {
      "id": "efficient_casting",
      "name": "Efficient Casting",
      "description": "-20% mana cost on all spells",
      "level": 2,
      "category": 6,
      "roleGate": "",
      "exclusiveWith": ["overcharge"],
      "unlockCost": 3,
      "params": { "costReduction": 0.2 }
    }
  ]
}
```

### JSON Field Reference

| Field | Type | Description |
|-------|------|-------------|
| `id` | string | Unique identifier (snake_case) |
| `name` | string | Display name |
| `description` | string | Tooltip text |
| `level` | int | 0=Squad, 1=Unit, 2=Commander |
| `category` | int | 0-6 matching PerkCategory enum |
| `roleGate` | string | "" for any, "Tank"/"DPS"/"Support" |
| `exclusiveWith` | []string | Mutually exclusive perk IDs |
| `unlockCost` | int | Perk points to unlock |
| `statModifiers` | []obj | For Tier 1 stat perks |
| `behaviorId` | string | Key into hook registry |
| `params` | map | Per-behavior parameters |

---

## 12. Implementation Order

### Phase 1: Foundation (~2 hours)

Create `tactical/perks/` package with all shared infrastructure.

1. `init.go` -- subsystem registration
2. `components.go` -- ECS component definitions
3. `perkdefinition.go` -- PerkDefinition struct, PerkStatModifier, enums
4. `registry.go` -- PerkRegistry, LoadPerkDefinitions(), validation
5. `queries.go` -- getActivePerkIDs, getSquadPerkIDs, HasPerk helpers
6. Add `SourcePerk` to `tactical/effects/components.go`
7. Add `RemoveEffectsBySource()` to `tactical/effects/system.go`
8. Create `assets/gamedata/perkdata.json` with 5-10 starter perks

### Phase 2: Stat Perks (~1 hour)

Implement all 13 Tier 1 perks -- these need zero hook infrastructure.

1. `system.go` -- ApplyStatPerks, RemoveStatPerks
2. JSON entries for all stat perks
3. Unit tests for stat perk application/removal

### Phase 3: Hook Infrastructure (~2 hours)

Build the hook system.

1. `hooks.go` -- 7 hook function type definitions (include `DamageRedirectHook` for Guardian)
2. `hook_registry.go` -- PerkHooks struct, RegisterPerkHooks, GetPerkHooks
3. Hook runner functions in `queries.go`
4. Add `ArmorReduction` field to `DamageModifiers` in `squadcombat.go`
5. Define hook runner callback types in `squads` package (see [Landmine 1](#landmine-1-hook-placement-precision))
6. Document stacking rules (see [Section 15](#15-stacking-rules))

### Phase 4: Existing Code Integration (~2 hours)

Insert hook calls into existing combat pipeline. **This phase is more involved than originally estimated** due to callback injection, signature changes, and counterattack refactoring.

1. `calculateDamage()` -- inject damage mod + cover mod callbacks. Place BEFORE multiplier application at line 232 ([Landmine 1](#landmine-1-hook-placement-precision))
2. `processAttackWithModifiers()` -- add `defenderSquadID` parameter, update callers ([Landmine 2](#landmine-2-missing-defendersquadid-parameter))
3. `ExecuteAttackAction()` -- move counter modifier construction up from `ProcessCounterattackOnTargets`, add counter mod hooks ([Landmine 3](#landmine-3-counterattack-modifier-refactoring))
4. `ExecuteSquadCounterattack()` -- same counter hook treatment as above
5. Export `getUnitsInRow` if needed
6. Add perk ID caching at battle start ([Landmine 7](#landmine-7-getactiveperkids-performance))

### Phase 5: Core Behavior Implementations (~3 hours)

Implement the most impactful perks first.

1. `behaviors.go` -- riposte, armor piercing, stone wall (counter mods)
2. Focus fire, cleave (target overrides)
3. Berserker, lifesteal, inspiration (damage mods + post-damage)
4. War medic, adrenaline (turn start)
5. Impale (cover mod)
6. Unit tests for each behavior

### Phase 6: Commander Perks (~1 hour)

Simple if-chain in spell system.

1. Add commander perk checks in `tactical/spells/system.go` ExecuteSpellCast()
2. Efficient Casting, Spell Mastery, Overcharge (mana/damage mods)
3. Lingering Magic, Potent Enchantment (duration/modifier scaling)
4. Unit tests

### Phase 7: GUI Equip Screen (~4-6 hours)

Pre-battle perk management. **Must implement perk lifecycle management** (see [Landmine 5](#landmine-5-perk-lifecycle-management)).

1. New mode in `gui/guisquads/`
2. Unlocked perk pool display/
3. Squad/unit/commander perk slot management
4. Role gate and mutual exclusivity validation
5. Perk point spending UI
6. Equip/unequip lifecycle: `ApplyStatPerks` on equip, `RemoveEffectsBySource(SourcePerk)` + re-apply on unequip
7. Decide and enforce: perks cannot be changed mid-battle

### Phase 8: Remaining Perks + Testing (~11-16 hours)

Fill out to full ~50 perk set.

1. Remaining Tier 2 perks (~2-3 hours, simple hook registrations)
2. Remaining Tier 3 perks (~3-4 hours, each needs thought -- especially Guardian, Double Strike)
3. Tier 4 perks (~4-6 hours -- Overwatch, Preemptive Strike, etc. may need new hooks or structural changes)
4. **Interaction tests** -- verify stacking rules, exclusive perk enforcement, multi-perk combinations
5. **Edge case tests** -- dead unit perks, recursion guards (Double Strike), PostDamageHook timing
6. Integration tests with full combat scenarios
7. Build archetype validation (test the 5 archetypes from brainstorm doc)
8. Performance benchmarks: combat with 0 perks vs full perk loadout, verify < 5% slowdown

---

## 13. Go Design Pattern Analysis

### 13.1 Circular Import Prevention

**Risk:** `tactical/perks` needs types from `tactical/squads` (DamageModifiers, CoverBreakdown) and `tactical/effects` (ActiveEffect, ApplyEffect). Meanwhile `tactical/squads/squadcombat.go` will call perk hook runners.

**Solution -- Dependency direction:**

```
tactical/effects/   <-- no dependencies on perks or squads
tactical/squads/    <-- imports effects (already does via squadabilities.go)
tactical/perks/     <-- imports effects + squads (one-way)
tactical/combat/    <-- imports perks + squads (one-way)
```

The key insight: `tactical/squads/squadcombat.go` does NOT import `tactical/perks`. Instead, `tactical/combat/combatactionsystem.go` is the integration layer that imports both `squads` and `perks`. The hook calls in `calculateDamage()` and `processAttackWithModifiers()` are passed as **function parameters or called from the combat action system level**.

**Simplest approach:** Have `combatactionsystem.go` call perk runners directly. The `squadcombat.go` functions receive already-modified `DamageModifiers` and `targetIDs` -- they don't need to know about perks at all.

### 13.2 Function Types vs Interfaces

Go convention favors function types for single-method abstractions:

```go
// Preferred (Go-idiomatic):
type DamageModHook func(attackerID, defenderID ecs.EntityID,
    modifiers *squads.DamageModifiers, manager *common.EntityManager)

// Less Go-idiomatic:
type DamagePolicy interface {
    ModifyDamage(attackerID, defenderID ecs.EntityID,
        modifiers *squads.DamageModifiers, manager *common.EntityManager)
}
```

The function type approach has advantages:
- No struct needed for simple perks (just a named function)
- `nil` check is natural for optional hooks
- Closures capture parameters when needed

### 13.3 Composition Pattern

When a perk needs multiple hooks (Focus Fire: target override + damage mod), the `PerkHooks` struct naturally composes them:

```go
RegisterPerkHooks("focus_fire", &PerkHooks{
    TargetOverride: focusFireTargetOverride,
    DamageMod:      focusFireDamageMod,
})
```

Each hook runs independently at its own call site. No complex middleware chain or decorator wrapping needed.

---

## 14. Implementation Landmines

Issues identified during code review that are not immediately obvious from the design above. Each must be resolved during implementation or the perk system will break in subtle ways.

### Landmine 1: Hook Placement Precision

**Problem:** `DamageModifiers.DamageMultiplier` is consumed at a specific line:

```go
// squadcombat.go line 232
baseDamage = int(float64(baseDamage) * modifiers.DamageMultiplier)
```

DamageModHooks **must run BEFORE line 232** so that perks like Berserker (+30%) and Glass Cannon (+35%) modify the multiplier before it's applied. Placing hooks after resistance calc (line ~238) means the multiplier is already consumed and the hooks have no effect.

**Additionally:** `squadcombat.go` cannot import `tactical/perks` (circular import). But hook calls need to run inside `calculateDamage()`.

**Resolution:** Use **callback injection**. Define hook runner function types in the `squads` package:

```go
// In tactical/squads/squadcombat.go
type DamageHookRunner func(attackerID, defenderID ecs.EntityID,
    mods *DamageModifiers, mgr *common.EntityManager)

type CoverHookRunner func(attackerID, defenderID ecs.EntityID,
    cover *CoverBreakdown, mgr *common.EntityManager)
```

`combatactionsystem.go` wires the actual implementations from `perks` into `squadcombat.go` functions via these parameters. The `squads` package defines the function types; `perks` provides the implementations; `combat` connects them. No circular import.

---

### Landmine 2: Missing defenderSquadID Parameter

**Problem:** `processAttackWithModifiers()` (squadcombat.go line 101) takes `targetIDs []ecs.EntityID` but NOT the defender's squad ID. The `TargetOverrideHook` signature requires `defenderSquadID` to find adjacent rows (Cleave) or alternative targets.

**Impact:** Requires a **signature change**:

```go
// Before
func processAttackWithModifiers(attackerID ecs.EntityID, targetIDs []ecs.EntityID, ...) int

// After
func processAttackWithModifiers(attackerID ecs.EntityID, defenderSquadID ecs.EntityID,
    targetIDs []ecs.EntityID, ...) int
```

**Cascading changes:** All callers of `processAttackWithModifiers` must be updated:
- `ProcessAttackOnTargets` (squadcombat.go line 599)
- `ProcessCounterattackOnTargets` (squadcombat.go line 143)

---

### Landmine 3: Counterattack Modifier Refactoring

**Problem:** Counterattack modifiers are currently constructed inside `ProcessCounterattackOnTargets` (squadcombat.go line 138):

```go
modifiers := DamageModifiers{
    HitPenalty:       counterattackHitPenalty,    // 20
    DamageMultiplier: counterattackDamageMultiplier, // 0.5
    IsCounterattack:  true,
}
```

For perk hooks (Riposte, Stone Wall) to modify these values, the construction must move **up** to `ExecuteAttackAction()` in `combatactionsystem.go` where hooks can run on them before they're passed to the counterattack function.

**Additionally:** `ExecuteSquadCounterattack` (combatactionsystem.go line 96) also calls `ProcessCounterattackOnTargets` with hardcoded modifiers. Both call sites need the same treatment.

---

### Landmine 4: PostDamageHook Timing

**Problem:** `recordDamageToUnit` (squadcombat.go line 637) only records damage in `result.DamageByUnit` -- it does NOT apply HP changes. Actual HP reduction happens later in `ApplyRecordedDamage` (line 663).

This means a PostDamageHook runs when the unit's `CurrentHealth` is **still at pre-damage values**. Consequences:

- **Suppressing Fire** (apply -15% hit debuff): If the target was dealt lethal damage, the debuff is applied to a unit that will be "dead" when damage resolves. Wasteful but not harmful.
- **Lifesteal** (heal 25% of damage dealt): Works correctly because it reads `damageDealt` from the hook parameter, not from HP diff.
- **Inspiration** (on kill: buff allies): `wasKill` is determined from the damage event, which calculates lethality before HP is actually reduced. Verify that `event.WasKilled` is set correctly at this point.

**Resolution:** Document this timing clearly in the hook contract. PostDamageHooks run with **pre-damage HP** and should rely on the `damageDealt`/`wasKill` parameters, not entity HP values.

---

### Landmine 5: Perk Lifecycle Management

**Problem:** When are perk stat effects applied and removed?

| Event | Action | Notes |
|-------|--------|-------|
| Perk equipped (GUI) | Call `ApplyStatPerks()` | Stat effects become active immediately |
| Perk unequipped (GUI) | Call `RemoveEffectsBySource(SourcePerk)` then re-apply remaining perks | Must remove ALL perk effects and re-apply to avoid stale modifiers |
| Battle start | Verify stat perks are applied, cache active perk IDs per unit | See Landmine 7 for caching |
| Battle end | No action needed | Permanent stat effects persist, behavioral hooks are stateless |
| Unit dies | No action needed | Effects are irrelevant on dead units |

**Open questions:**
- Can perks be changed mid-battle? (Probably not, but should be stated explicitly)
- What about "Flexible Doctrine" (Tier 4) which switches formation once per battle -- this implies some perk state can change during combat
- Re-equipping: If a unit moves between squads, squad-level perk effects must be recalculated

---

### Landmine 6: Guardian Requires a 7th Hook Type

**Problem:** Guardian (redirect 50% damage to adjacent ally) cannot be cleanly implemented with the existing 6 hook types.

Faking it with PostDamageHook (heal original target, damage guardian after the fact) creates:
- Confusing combat log entries (unit takes full damage, then gets healed)
- Incorrect AI threat evaluation during the damage phase
- Race conditions if the guardian is also a damage target in the same attack

**Resolution:** Commit to the 7th hook type (`DamageRedirectHook`). This hook runs inside `processAttackWithModifiers` after `calculateDamage` returns but before `recordDamageToUnit`. If `redirectTargetID != 0`, record `redirectAmount` as additional damage to the guardian.

**Impact:** +1 hook type definition, +1 runner function, +3 lines in `processAttackWithModifiers`. Total: ~15 additional lines in new code, ~3 in existing code.

If Guardian is too complex for v1, **cut it explicitly** from the first implementation phase and document it as deferred.

---

### Landmine 7: getActivePerkIDs Performance

**Problem:** Every hook runner calls `getActivePerkIDs()` which does multiple component lookups. For a 9-unit squad with 2 unit perks + 3 squad perks each, a single `calculateDamage()` call runs 3 hook runners, each calling `getActivePerkIDs`. That's **~30 component lookups per damage calculation**, repeated for every unit in the target list.

**Resolution:** Cache active perk IDs and resolved hooks per unit at **battle start**:

```go
// Built once at battle start, stored on CombatActionSystem or a combat state entity
type CachedUnitPerks struct {
    PerkIDs  []string
    HasHooks map[string]bool // Quick lookup: does this unit have any behavioral perks?
}

var unitPerkCache map[ecs.EntityID]*CachedUnitPerks
```

Runners check the cache instead of doing component lookups each time. Cache is built in `InitializeCombat()` and never invalidated (perks don't change mid-battle).

---

### Landmine Summary

| # | Landmine | Severity | When It Bites |
|---|----------|----------|---------------|
| 1 | Hook placement before multiplier application | High | First perk test -- damage mods have no effect |
| 2 | Missing defenderSquadID in processAttackWithModifiers | Medium | First target override perk (Cleave, Focus Fire) |
| 3 | Counterattack modifiers hardcoded in wrong location | Medium | Riposte/Stone Wall implementation |
| 4 | PostDamageHook sees pre-damage HP | Low | Suppressing Fire debuff on dead units |
| 5 | Perk lifecycle (equip/unequip) unspecified | High | GUI equip screen -- stat perks don't apply or stack incorrectly |
| 6 | Guardian needs 7th hook type | Medium | Guardian implementation (can defer to v2) |
| 7 | getActivePerkIDs performance | Low | Noticeable with 9-unit squads, full perk loadouts |

---

## 15. Stacking Rules

When multiple perks modify the same values, stacking behavior must be defined explicitly.

- **DamageMultiplier: Multiplicative stacking** (each perk multiplies the current value)
  ```
  Glass Cannon: modifiers.DamageMultiplier *= 1.35  // Now 1.35
  Berserker:    modifiers.DamageMultiplier *= 1.30  // Now 1.755 (+75.5%, not +65%)
  ```
- **ArmorReduction: Max wins** (0.5 from Armor Piercing is not doubled by another perk)
- **HitPenalty/bonus: Additive stacking** (two +10% hit perks = +20% hit)
- **CoverReduction: Min wins** (Impale zeroes cover; stacking with another cover perk still zeroes it)

This is the correct behavior for a tactical game (multiplicative stacking creates meaningful power curves), but it must be documented as an explicit design decision.

---

## 16. Testing Strategy

### Unit Tests (per hook function)

```go
func TestBerserkerDamageMod(t *testing.T) {
    manager := setupTestManager()
    attacker := createTestUnit(manager, 10, 50) // 10 HP, 50 max

    modifiers := squads.DamageModifiers{DamageMultiplier: 1.0}
    berserkerDamageMod(attacker, 0, &modifiers, manager)

    // Below 50% HP: should get +30%
    if modifiers.DamageMultiplier != 1.3 {
        t.Errorf("expected 1.3, got %f", modifiers.DamageMultiplier)
    }
}
```

### Integration Tests (full pipeline)

```go
func TestRiposte_FullDamageCounter(t *testing.T) {
    manager := setupTestManager()
    attacker := createTestSquad(manager)
    defender := createTestSquadWithPerk(manager, "riposte")

    result := combat.ExecuteAttackAction(attacker, defender)
    // Verify counterattack damage is NOT halved
    for _, event := range result.CombatLog.AttackEvents {
        if event.IsCounterattack {
            // Counter damage multiplier should be 1.0, not 0.5
        }
    }
}
```

### Interaction Tests (critical)

```go
func TestGlassCannon_Plus_Berserker_Stacking(t *testing.T) {
    // Verify multiplicative stacking: 1.35 * 1.30 = 1.755
}

func TestRiposte_Plus_RecklessAssault_Conflict(t *testing.T) {
    // Reckless Assault on ATTACKER suppresses defender's counter
    // even if defender has Riposte
}
```

### Edge Case Tests

```go
func TestPerkOnDeadUnit(t *testing.T) {
    // Unit with Lifesteal is killed -- PostDamageHook should not heal a dead unit
}

func TestExclusivePerkEnforcement(t *testing.T) {
    // Verify Riposte and Stone Wall cannot both be equipped
}

func TestSquadPerkAppliesToAllUnits(t *testing.T) {
    // Glass Cannon (squad perk) should modify damage for every unit in the squad
}
```

### Recursion Guard Tests

```go
func TestDoubleStrike_NoInfiniteLoop(t *testing.T) {
    // Double Strike triggers a second attack
    // The second attack must NOT trigger another Double Strike
}
```

---

## Pros & Cons Summary

### Pros

1. **Smallest existing code footprint** (~49 lines across 3 files, ~6 lines modified)
2. **Explicit and traceable** -- `grep RunDamageModHooks` finds every call site
3. **Go-idiomatic** -- function types over single-method interfaces
4. **Near-zero runtime overhead when no perks equipped** -- runners iterate empty perk lists and return (see [Landmine 7](#landmine-7-getactiveperkids-performance) for nuance)
5. **Type-safe** -- each hook has its own typed signature
6. **Leverages existing systems maximally** -- ActiveEffect, DamageModifiers, CoverBreakdown
7. **Minimal lifecycle management** -- no event bus, no subscriptions, no cache invalidation (but perk equip/unequip still needs stat effect lifecycle -- see [Landmine 5](#landmine-5-perk-lifecycle-management))
8. **Easy to test** -- each hook function is standalone, testable in isolation

### Cons

1. Adding a new hook type requires: new function type + runner function + call site insertion
2. Hook execution order follows perk equip order (no explicit priority)
3. Perk interactions (two DamageMod hooks) run sequentially but could conflict
4. PerkHooks struct grows as hook types are added (currently 7 cover all ~50 perks)

### When to Reconsider

- **100+ perks with complex multi-phase interactions:** Consider middleware for better ordering control
- **Cross-squad event reactions:** Consider event-driven observer if perks need to react to other squads' combat events
- **Perk layering as core mechanic:** Consider strategy/policy pattern if explicit decorator composition becomes important

---

## Implementation Estimate

| Phase | Work | Time |
|-------|------|------|
| Foundation (package, components, registry, JSON) | Phases 1-2 | ~3 hours |
| Hook infrastructure + integration (revised up) | Phases 3-4 | ~4 hours |
| Core behaviors (15 perks) | Phase 5 | ~3 hours |
| Commander perks | Phase 6 | ~1 hour |
| GUI equip screen + lifecycle management | Phase 7 | ~4-6 hours |
| Remaining perks + testing (revised up) | Phase 8 | ~11-16 hours |
| **Total** | | **~26-33 hours** |
