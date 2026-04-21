# TinkerRogue Entity & Component Reference

**Last Updated:** 2026-04-21

This document is the definitive reference for every ECS entity type in TinkerRogue. It documents the exact components attached to each entity at creation time, including field names, types, and purposes. Component definitions are sourced directly from the codebase; do not use this document as a substitute for reading source, but use it to quickly understand what data lives on any given entity.

---

## Table of Contents

1. [How to Read This Document](#how-to-read-this-document)
2. [Shared Component Definitions](#shared-component-definitions)
3. [Player & Commander Domain](#player--commander-domain)
   - [Player Entity](#player-entity)
   - [Commander Entity](#commander-entity)
4. [Squads & Units Domain](#squads--units-domain)
   - [Squad Entity](#squad-entity)
   - [Unit Entity](#unit-entity)
   - [Creature Entity](#creature-entity)
5. [Combat Domain](#combat-domain)
   - [Combat Faction Entity](#combat-faction-entity)
   - [Turn State Entity](#turn-state-entity)
   - [Action State Entity](#action-state-entity)
6. [Overworld Domain](#overworld-domain)
   - [Overworld Turn State Entity](#overworld-turn-state-entity)
   - [Overworld Node Entity](#overworld-node-entity)
   - [Overworld Faction Entity](#overworld-faction-entity)
   - [Tick State Entity](#tick-state-entity-singleton)
   - [Victory State Entity](#victory-state-entity-singleton)
   - [Interaction Entity (Dynamic Component)](#interaction-entity-dynamic-component)
7. [Encounters & Raids Domain](#encounters--raids-domain)
   - [Encounter Entity](#encounter-entity)
   - [Raid State Entity](#raid-state-entity)
   - [Floor State Entity](#floor-state-entity)
   - [Alert Data Entity](#alert-data-entity)
   - [Room Data Entity](#room-data-entity)
   - [Garrison Squad Entity](#garrison-squad-entity)
   - [Deployment Entity](#deployment-entity)
8. [Component Index](#component-index)

---

## How to Read This Document

### Table Format

Each entity section contains a component table with these columns:

| Column | Meaning |
|--------|---------|
| **Component Variable** | The global `*ecs.Component` variable used to access this component (e.g., `common.AttributeComponent`) |
| **Query Tag** | The `ecs.Tag` variable used to query entities by this component (if one exists) |
| **Data Type** | The Go struct stored in this component slot |
| **Key Fields** | The most important fields; full definitions are in [Shared Component Definitions](#shared-component-definitions) |
| **Notes** | Conditional additions, lifecycle notes, cross-domain relationships |

### Notation

- **Singleton** — Only one entity of this type should exist at a time in the ECS world.
- **Dynamic** — This component is added/removed during gameplay (not only at creation).
- **Optional** — Only added if a condition is met at creation time (noted in the entry).
- Component variables are prefixed with their package: `common.PositionComponent`, `squadcore.SquadComponent`, etc.
- Tag variables follow the same package-prefix convention: `squadcore.SquadTag`, `combatstate.FactionTag`.

### Package path cheatsheet

| Prefix | Import path |
|---|---|
| `common` | `game_main/core/common` |
| `coords` | `game_main/core/coords` |
| `squadcore` | `game_main/tactical/squads/squadcore` |
| `roster` | `game_main/tactical/squads/roster` |
| `unitprogression` | `game_main/tactical/squads/unitprogression` |
| `commander` | `game_main/tactical/commander` |
| `combatstate` | `game_main/tactical/combat/combatstate` |
| `spells` | `game_main/tactical/powers/spells` |
| `perks` | `game_main/tactical/powers/perks` |
| `progression` | `game_main/tactical/powers/progression` |
| `artifacts` | `game_main/tactical/powers/artifacts` |
| `core` (overworld) | `game_main/campaign/overworld/core` |
| `raid` | `game_main/campaign/raid` |

### Component Access Pattern

```go
// From a query result entity:
data := common.GetComponentType[*MyData](entity, pkg.MyComponent)

// From an entity ID only:
data := common.GetComponentTypeByID[*MyData](manager, entityID, pkg.MyComponent)

// Checking existence:
if manager.HasComponent(entityID, pkg.MyComponent) { ... }
```

---

## Shared Component Definitions

These components appear on multiple entity types. They are defined once here and referenced by name in subsequent sections.

### `common.PositionComponent` — `*coords.LogicalPosition`

Tile-based world position. Used by the spatial grid (`common.GlobalPositionSystem`) for O(1) position queries. Must be added/removed using `manager.RegisterEntityPosition` / `manager.CleanDisposeEntity` to keep the spatial grid in sync.

```go
type LogicalPosition struct {
    X, Y int // Tile coordinates in the game world
}
```

**Package:** `game_main/core/coords`
**Query Tag:** Included in `common.RenderablesTag` (combined with `RenderableComponent`)

### `common.AttributeComponent` — `*common.Attributes`

Core combat and capability statistics. The primary driver of all derived combat values (damage, health, hit rate, etc.).

```go
type Attributes struct {
    // Core Stats
    Strength   int // Physical damage, resistance, max HP
    Dexterity  int // Hit rate, crit chance, dodge
    Magic      int // Magic damage, healing, magic defense
    Leadership int // Unit capacity (squad size limit)
    Armor      int // Damage reduction modifier
    Weapon     int // Damage increase modifier

    // Turn-Based Combat
    MovementSpeed int // Tiles per turn
    AttackRange   int // Attack distance in tiles

    // Runtime State
    CurrentHealth int  // Current HP; changes during combat
    CanAct        bool // Whether unit can act this turn
}
```

**Derived methods:** `GetMaxHealth()`, `GetPhysicalDamage()`, `GetPhysicalResistance()`, `GetHitRate()`, `GetCritChance()`, `GetDodgeChance()`, `GetMagicDamage()`, `GetMagicDefense()`, `GetHealingAmount()`, `GetUnitCapacity()`, `GetCapacityCost()`, `GetMovementSpeed()`, `GetAttackRange()`

**Package:** `game_main/core/common`

### `common.NameComponent` — `*common.Name`

Display name string for a unit. Distinct from `squadcore.UnitTypeComponent`, which holds the template type identifier used for roster grouping.

```go
type Name struct {
    NameStr string // Displayed name (e.g., "Karathos the Warrior")
}
```

**Package:** `game_main/core/common`

### `common.RenderableComponent` — `*common.Renderable`

Sprite image and visibility flag. The rendering system skips entities where `Visible` is `false` or `Image` is `nil`. Units inside a squad have `Visible = false` because only the squad entity renders on the overworld.

```go
type Renderable struct {
    Image   *ebiten.Image // Sprite to draw
    Visible bool          // Whether to draw this entity
}
```

**Package:** `game_main/core/common`
**Query Tag:** `common.RenderablesTag` (combined with `common.PositionComponent`)

### `common.ResourceStockpileComponent` — `*common.ResourceStockpile`

Economy resources. Used by both the player entity and overworld faction entities.

```go
type ResourceStockpile struct {
    Gold  int
    Iron  int
    Wood  int
    Stone int
}
```

**Package:** `game_main/core/common`

---

## Player & Commander Domain

### Player Entity

The single player-controlled top-level entity. Tracks resources, roster limits, progression state, and links to all commanders and artifacts. There is exactly one player entity per game session.

**Created by:** `InitializePlayerData()` in `setup/gamesetup/playerinit.go`

| Component Variable | Query Tag | Data Type | Key Fields | Notes |
|---|---|---|---|---|
| `common.PlayerComponent` | part of `players` WorldTag | `*common.Player` | (empty marker struct) | Marks entity as the player; the tag `players = BuildTag(PlayerComponent, PositionComponent)` is stored on `WorldTags["players"]` |
| `common.RenderableComponent` | `common.RenderablesTag` | `*common.Renderable` | `Image`, `Visible: true` | Player sprite loaded from `config.PlayerImagePath` |
| `common.AttributeComponent` | — | `*common.Attributes` | `Strength`, `Dexterity`, `Magic`, `Leadership`, `Armor`, `Weapon` | Values sourced from `gameconfig.json` `player.attributes` section |
| `common.ResourceStockpileComponent` | — | `*common.ResourceStockpile` | `Gold`, `Iron`, `Wood`, `Stone` | Initial values from `gameconfig.json` `player.resources` section |
| `roster.UnitRosterComponent` | — | `*roster.UnitRoster` | `Units map[string]*UnitRosterEntry`, `MaxUnits int` | Tracks all unit entities owned by the player; `MaxUnits` from `gameconfig.json` `player.limits.maxUnits` |
| `commander.CommanderRosterComponent` | — | `*commander.CommanderRosterData` | `CommanderIDs []ecs.EntityID`, `MaxCommanders int` | Tracks all commander entity IDs; `MaxCommanders` from config |
| `artifacts.ArtifactInventoryComponent` | — | `*artifacts.ArtifactInventoryData` | `OwnedArtifacts map[string][]*ArtifactInstance`, `MaxArtifacts int` | Artifact collection; `MaxArtifacts` from config |
| `progression.ProgressionComponent` | `progression.ProgressionTag` | `*progression.ProgressionData` | `ArcanaPoints`, `SkillPoints`, `UnlockedSpellIDs []string`, `UnlockedPerkIDs []string` | Permanent player progression: currencies and unlocked-library state. Seeded via `progression.NewProgressionData()` with a starter perk/spell set. |
| `common.PositionComponent` | `players` WorldTag | `*coords.LogicalPosition` | `X`, `Y` | Added via `RegisterEntityPosition` using `gm.StartingPosition()`; also stored in `PlayerData.Pos` for direct access |

**`roster.UnitRoster` detail:**
```go
type UnitRoster struct {
    Units    map[string]*UnitRosterEntry // template name -> entry
    MaxUnits int
}
type UnitRosterEntry struct {
    UnitType      string
    TotalOwned    int
    UnitsInSquads map[ecs.EntityID]int // squadID -> count
    UnitEntities  []ecs.EntityID
}
```

**`commander.CommanderRosterData` detail:**
```go
type CommanderRosterData struct {
    CommanderIDs  []ecs.EntityID
    MaxCommanders int
}
```

**`artifacts.ArtifactInventoryData` detail:**
```go
type ArtifactInventoryData struct {
    OwnedArtifacts map[string][]*ArtifactInstance // artifactID -> instances
    MaxArtifacts   int
}
type ArtifactInstance struct {
    EquippedOn ecs.EntityID // 0 = available, >0 = equipped on this squad
}
```

**`progression.ProgressionData` detail:**
```go
type ProgressionData struct {
    ArcanaPoints     int
    SkillPoints      int
    UnlockedSpellIDs []string // spell IDs the player has unlocked
    UnlockedPerkIDs  []string // perk IDs the player has unlocked
}
```

---

### Commander Entity

A field commander that moves on the overworld map and leads squads. Multiple commanders can exist simultaneously (limited by `CommanderRosterData.MaxCommanders`).

**Created by:** `CreateCommander()` in `tactical/commander/system.go`

| Component Variable | Query Tag | Data Type | Key Fields | Notes |
|---|---|---|---|---|
| `commander.CommanderComponent` | `commander.CommanderTag` | `*commander.CommanderData` | `CommanderID`, `Name`, `IsActive bool` | Core identity; `CommanderID` is the entity's own ID |
| `commander.CommanderActionStateComponent` | `commander.CommanderActionTag` | `*commander.CommanderActionStateData` | `CommanderID`, `HasMoved`, `HasActed`, `MovementRemaining` | Per-turn action tracking; reset by `StartNewTurn()` at overworld turn start |
| `common.RenderableComponent` | `common.RenderablesTag` | `*common.Renderable` | `Image`, `Visible: true` | Sprite passed as parameter to `CreateCommander` |
| `common.AttributeComponent` | — | `*common.Attributes` | `MovementSpeed` | Only `MovementSpeed` is set at creation; other stats default to zero |
| `roster.SquadRosterComponent` | — | `*roster.SquadRoster` | `OwnedSquads []ecs.EntityID`, `MaxSquads int` | Squads commanded by this commander; `maxSquads` parameter |
| `common.PositionComponent` | — | `*coords.LogicalPosition` | `X`, `Y` | Added via `RegisterEntityPosition` using the `startPos` parameter |

**Note:** Spell casting state (`spells.ManaComponent`, `spells.SpellBookComponent`) is **no longer** attached to the commander entity. It is attached to the **squad** that the commander leads, by `spells.InitSquadSpellsFromLeader()` — the spell list is derived from the squad leader's unit type and filtered against the player's unlocked-spell library. See [Squad Entity](#squad-entity).

**`commander.CommanderData` detail:**
```go
type CommanderData struct {
    CommanderID ecs.EntityID
    Name        string
    IsActive    bool
}
```

**`commander.CommanderActionStateData` detail:**
```go
type CommanderActionStateData struct {
    CommanderID       ecs.EntityID
    HasMoved          bool
    HasActed          bool
    MovementRemaining int // Set from Attributes.MovementSpeed at turn start
}
```

**`roster.SquadRoster` detail:**
```go
type SquadRoster struct {
    OwnedSquads []ecs.EntityID
    MaxSquads   int
}
```

---

## Squads & Units Domain

### Squad Entity

A group of units that acts as a single tactical unit on the overworld and battle map. The squad entity is what moves, is positioned, participates in faction/combat systems, and holds the power layer state (mana, spellbook, perks, artifacts). Individual units are invisible sub-entities linked to the squad via `SquadMemberComponent`.

**Created by:** `CreateEmptySquad()` or `CreateSquadFromTemplate()` in `tactical/squads/squadcore/squadcreation.go`

| Component Variable | Query Tag | Data Type | Key Fields | Notes |
|---|---|---|---|---|
| `squadcore.SquadComponent` | `squadcore.SquadTag` | `*squadcore.SquadData` | `SquadID`, `Name`, `Formation`, `Morale`, `MaxUnits`, `IsDeployed`, `GarrisonedAtNodeID` | Core squad data; `Formation` only set by `CreateSquadFromTemplate` |
| `common.PositionComponent` | `squadcore.SquadTag` (combined) | `*coords.LogicalPosition` | `X`, `Y` | Added directly on creation; upgraded to tracked position when assigned to a faction via `AddSquadToFaction` |
| `common.RenderableComponent` | `common.RenderablesTag` | `*common.Renderable` | `Image` (leader's sprite), `Visible: true` | **Optional.** Only added by `SetSquadRenderableFromLeader()` after a leader exists. Not present on `CreateEmptySquad` squads until a leader is assigned |
| `spells.ManaComponent` | `spells.ManaTag` | `*spells.ManaData` | `CurrentMana`, `MaxMana` | **Optional.** Added by `spells.AddSpellCapabilityToSquad` / `InitSquadSpellsFromLeader` only when the leader's unit type has unlocked spells. Mana is a squad-scoped resource. |
| `spells.SpellBookComponent` | `spells.SpellBookTag` | `*spells.SpellBookData` | `SpellIDs []templates.SpellID` | **Optional.** Added alongside `ManaComponent`. Contains the intersection of the leader's unit-type spell list and the player's unlocked-spell library. |
| `perks.PerkSlotComponent` | `perks.PerkSlotTag` | `*perks.PerkSlotData` | `PerkIDs []perks.PerkID` | **Optional / Dynamic.** Added by `perks.EquipPerk` the first time a perk is equipped; tracks squad-equipped perks. Slot cap depends on squad progression. |
| `perks.PerkRoundStateComponent` | — | `*perks.PerkRoundState` | Shared tracking fields + `PerkState`, `PerkBattleState` maps | **Dynamic / Combat only.** Added at combat start by the perks system and removed by `mind/combatlifecycle/cleanup.go` when combat ends. |
| `artifacts.EquipmentComponent` | — | `*artifacts.EquipmentData` | `EquippedArtifacts []string` | **Optional / Dynamic.** Added the first time an artifact is equipped on this squad. |
| `combatstate.FactionMembershipComponent` | `combatstate.CombatFactionTag` | `*combatstate.CombatFactionData` | `FactionID ecs.EntityID` | **Dynamic / Combat only.** Added when squad enters combat via `AddSquadToFaction()`; removed when squad exits combat |

**`squadcore.SquadData` detail:**
```go
type SquadData struct {
    SquadID            ecs.EntityID  // Own entity ID
    Formation          FormationType // Balanced, Defensive, Offensive, Ranged
    Name               string
    Morale             int           // 0-100
    SquadLevel         int           // Average level for spawning
    TurnCount          int           // Turns taken (combat lifecycle)
    MaxUnits           int           // Typically 9 (3x3 grid)
    IsDeployed         bool          // true = on tactical map
    GarrisonedAtNodeID ecs.EntityID  // 0 = not garrisoned
}
```

**`spells.ManaData` detail:**
```go
type ManaData struct {
    CurrentMana int
    MaxMana     int
}
```

**`spells.SpellBookData` detail:**
```go
type SpellBookData struct {
    SpellIDs []templates.SpellID
}
```

**`perks.PerkSlotData` detail:**
```go
type PerkSlotData struct {
    PerkIDs []PerkID // Equipped perk IDs (max based on squad level)
}
```

**`combatstate.CombatFactionData` detail:**
```go
type CombatFactionData struct {
    FactionID ecs.EntityID // Faction that owns this squad in combat
}
```

---

### Unit Entity

An individual combatant that lives inside a squad. Units are not rendered on the world map (their `Renderable.Visible` is set to `false` when added to a squad). Their sprite image is read by the squad to set the squad's own renderable.

A unit entity is built from three cooperating functions:
1. `templates.CreateEntityFromTemplate()` adds the three base components (`NameComponent`, `PositionComponent`, `AttributeComponent`, `RenderableComponent`).
2. `squadcore.ApplyUnitComponents()` adds all squad-specific components (grid position, role, cover, ranges, progression).
3. `squadcore.AddLeaderComponents()` (optional) adds leader-only components to the first unit in a squad or when `IsLeader` is true.

**Created by:** `squadcore.CreateUnitEntity()` or directly inside `CreateSquadFromTemplate()`; called by `AddUnitToSquad()` and `CreateSquadFromTemplate()`.

| Component Variable | Query Tag | Data Type | Key Fields | Notes |
|---|---|---|---|---|
| `common.NameComponent` | — | `*common.Name` | `NameStr` | Procedurally generated display name (e.g., "Karathos the Warrior") |
| `common.PositionComponent` | — | `*coords.LogicalPosition` | `X`, `Y` | Set to squad's world position; not tracked by `GlobalPositionSystem` until squad deploys |
| `common.AttributeComponent` | — | `*common.Attributes` | All fields | Sourced from the unit template's attributes (derived from `JSONMonster` data) |
| `common.RenderableComponent` | — | `*common.Renderable` | `Image`, `Visible: false` | Image loaded from creature asset path; **always set `Visible = false` when unit is in a squad** |
| `squadcore.SquadMemberComponent` | `squadcore.SquadMemberTag` | `*squadcore.SquadMemberData` | `SquadID ecs.EntityID` | Links unit back to its parent squad; **not present on roster units** (added by `AddUnitToSquad` / `PlaceUnitInSquad`) |
| `squadcore.GridPositionComponent` | — | `*squadcore.GridPositionData` | `AnchorRow`, `AnchorCol`, `Width`, `Height` | Position in the 3x3 formation grid; supports multi-cell units |
| `squadcore.UnitTypeComponent` | — | `*squadcore.UnitTypeData` | `UnitType string` | Template type identifier (e.g., `"Goblin"`) for roster grouping; distinct from `NameComponent` |
| `squadcore.UnitRoleComponent` | — | `*squadcore.UnitRoleData` | `Role unitdefs.UnitRole` | `RoleTank`, `RoleDPS`, or `RoleSupport`; drives combat targeting |
| `squadcore.TargetRowComponent` | — | `*squadcore.TargetRowData` | `AttackType`, `TargetCells [][2]int` | Attack targeting mode; `TargetCells` only used for `AttackTypeMagic` and `AttackTypeHeal` |
| `squadcore.CoverComponent` | — | `*squadcore.CoverData` | `CoverValue float64`, `CoverRange int`, `RequiresActive bool` | **Optional.** Only added when `template.CoverValue > 0` |
| `squadcore.AttackRangeComponent` | — | `*squadcore.AttackRangeData` | `Range int` | World-tile attack range: Melee=1, Ranged=3, Magic=4 |
| `squadcore.MovementSpeedComponent` | — | `*squadcore.MovementSpeedData` | `Speed int` | Tiles per turn; squad speed = minimum of all member speeds |
| `unitprogression.ExperienceComponent` | — | `*unitprogression.ExperienceData` | `Level`, `CurrentXP`, `XPToNextLevel` | XP tracking; starts at Level 1 |
| `unitprogression.StatGrowthComponent` | — | `*unitprogression.StatGrowthData` | `Strength`, `Dexterity`, `Magic`, `Leadership`, `Armor`, `Weapon` (each `GrowthGrade`) | Per-stat level-up growth rates: S=90%, A=75%, B=60%, C=45%, D=30%, E=15%, F=5% |
| `squadcore.LeaderComponent` | `squadcore.LeaderTag` | `*squadcore.LeaderData` | `Leadership int`, `Experience int` | **Optional / Leader only.** Added by `AddLeaderComponents()` for the first unit or when `IsLeader = true` |
| `squadcore.AbilitySlotComponent` | — | `*squadcore.AbilitySlotData` | `Slots [4]AbilitySlot` | **Optional / Leader only.** Four ability slots (Rally, Heal, BattleCry, Fireball) |
| `squadcore.CooldownTrackerComponent` | — | `*squadcore.CooldownTrackerData` | `Cooldowns [4]int`, `MaxCooldowns [4]int` | **Optional / Leader only.** Per-slot cooldown tracking |

**`squadcore.GridPositionData` detail:**
```go
type GridPositionData struct {
    AnchorRow int // Top-left row (0-2)
    AnchorCol int // Top-left col (0-2)
    Width     int // Cells wide (1-3)
    Height    int // Cells tall (1-3)
}
```

**`squadcore.CoverData` detail:**
```go
type CoverData struct {
    CoverValue     float64 // Damage reduction (0.0-1.0)
    CoverRange     int     // Rows behind that benefit
    RequiresActive bool    // Dead units don't provide cover if true
}
```

**`squadcore.AbilitySlot` detail:**
```go
type AbilitySlot struct {
    AbilityType  AbilityType  // Rally, Heal, BattleCry, Fireball
    TriggerType  TriggerType  // SquadHPBelow, TurnCount, EnemyCount, MoraleBelow, CombatStart
    Threshold    float64      // Condition threshold value
    HasTriggered bool         // Once-per-combat flag
    IsEquipped   bool         // Whether slot is active
}
```

---

### Creature Entity

A standalone enemy or NPC entity that exists directly on the world map (not inside a squad). Created from a `JSONMonster` definition via the entity factory.

**Created by:** `createCreatureEntity()` in `templates/entity_factory.go` (called via `CreateEntityFromTemplate()`)

| Component Variable | Query Tag | Data Type | Key Fields | Notes |
|---|---|---|---|---|
| `common.NameComponent` | — | `*common.Name` | `NameStr` | Set from `EntityConfig.Name` |
| `common.RenderableComponent` | `common.RenderablesTag` | `*common.Renderable` | `Image`, `Visible` | Image loaded from `EntityConfig.AssetDir / EntityConfig.ImagePath` |
| `common.PositionComponent` | — | `*coords.LogicalPosition` | `X`, `Y` | If `EntityConfig.Position != nil`, uses `RegisterEntityPosition` (tracked); otherwise adds default `{0,0}` (untracked) |
| `common.AttributeComponent` | — | `*common.Attributes` | All fields | Derived from `JSONMonster.Attributes.NewAttributesFromJson()` |

**Note on map blocking:** When `EntityConfig.GameMap != nil` and `EntityConfig.Position != nil`, the tile at the entity's position is marked `Blocked = true` in the game map.

---

## Combat Domain

### Combat Faction Entity

Represents one side in a battle (player or enemy). Owns one or more squads. Created when combat is initialized; there are typically two factions per combat encounter.

**Created by:** `CreateFactionWithPlayer()` in `tactical/combat/combatstate`

| Component Variable | Query Tag | Data Type | Key Fields | Notes |
|---|---|---|---|---|
| `combatstate.CombatFactionComponent` | `combatstate.FactionTag` | `*combatstate.FactionData` | `FactionID`, `Name`, `Mana`, `MaxMana`, `IsPlayerControlled`, `PlayerID`, `PlayerName`, `EncounterID` | All fields set at creation |

**`combatstate.FactionData` detail:**
```go
type FactionData struct {
    FactionID          ecs.EntityID // Own entity ID
    Name               string       // Display name
    Mana               int          // Current mana (faction-level pool, distinct from squad ManaData)
    MaxMana            int          // Max mana
    IsPlayerControlled bool         // Derived from PlayerID > 0
    PlayerID           int          // 0 = AI, 1 = Player 1, 2 = Player 2
    PlayerName         string       // "Player 1" or custom
    EncounterID        ecs.EntityID // 0 if not from an encounter
}
```

**Note on squad membership:** Squads do not store a reference to their faction on the `CombatFactionEntity`. Instead, when a squad is added to a faction via `AddSquadToFaction()`, a `FactionMembershipComponent` (`combatstate.CombatFactionData`) is added directly to the **squad entity**. This is ECS-idiomatic: query squads for their faction, not the faction for its squads.

---

### Turn State Entity

Singleton that governs round and turn ordering for one combat session. Exactly one should exist per active combat.

**Created by:** `TurnManager.InitializeCombat()` in `tactical/combat/combatcore/turnmanager.go`

| Component Variable | Query Tag | Data Type | Key Fields | Notes |
|---|---|---|---|---|
| `combatstate.TurnStateComponent` | `combatstate.TurnStateTag` | `*combatstate.TurnStateData` | `CurrentRound`, `TurnOrder []ecs.EntityID`, `CurrentTurnIndex`, `CombatActive` | `TurnOrder` is faction IDs shuffled via Fisher-Yates at combat start; `TurnManager` caches the entity ID to avoid O(n) lookups |

**`combatstate.TurnStateData` detail:**
```go
type TurnStateData struct {
    CurrentRound     int            // Starts at 1
    TurnOrder        []ecs.EntityID // Faction IDs in randomized order
    CurrentTurnIndex int            // 0 to len(TurnOrder)-1
    CombatActive     bool           // false after EndCombat()
}
```

---

### Action State Entity

Tracks whether a specific squad has moved and acted during the current faction's turn. One entity exists per squad for the duration of combat.

**Created by:** `CreateActionStateForSquad()` in `tactical/combat/combatstate/combatqueries.go`; called during `InitializeCombat()`

| Component Variable | Query Tag | Data Type | Key Fields | Notes |
|---|---|---|---|---|
| `combatstate.ActionStateComponent` | `combatstate.ActionStateTag` | `*combatstate.ActionStateData` | `SquadID`, `HasMoved`, `HasActed`, `MovementRemaining`, `BonusAttackActive` | `MovementRemaining` initialized from squad's actual movement speed (falling back to `config.DefaultMovementSpeed` if 0); reset each time it is that faction's turn via `ResetSquadActions()` |

**`combatstate.ActionStateData` detail:**
```go
type ActionStateData struct {
    SquadID           ecs.EntityID // Squad this tracks
    HasMoved          bool         // Squad used movement this turn
    HasActed          bool         // Squad used its attack/skill this turn
    MovementRemaining int          // Tiles of movement remaining
    BonusAttackActive bool         // Consumes one attack without setting HasActed
}
```

---

## Overworld Domain

### Overworld Turn State Entity

Singleton tracking the player's overworld turn counter. One entity per game session.

**Created by:** `CreateOverworldTurnState()` in `tactical/commander/turnstate.go`

| Component Variable | Query Tag | Data Type | Key Fields | Notes |
|---|---|---|---|---|
| `commander.OverworldTurnStateComponent` | `commander.OverworldTurnTag` | `*commander.OverworldTurnStateData` | `CurrentTurn int`, `TurnActive bool` | `CurrentTurn` starts at 1; incremented by `EndTurn()`; queried via `GetOverworldTurnState()` |

**`commander.OverworldTurnStateData` detail:**
```go
type OverworldTurnStateData struct {
    CurrentTurn int  // Starts at 1
    TurnActive  bool // Starts true
}
```

---

### Overworld Node Entity

Represents a point of interest on the overworld map: a threat node (enemy territory), a player-placed settlement, fortress, or other landmark. All node types share this single unified entity type; the `NodeTypeID` and `Category` fields distinguish them.

**Created by:** `CreateNode()` in `campaign/overworld/node/system.go`

| Component Variable | Query Tag | Data Type | Key Fields | Notes |
|---|---|---|---|---|
| `common.PositionComponent` | `core.OverworldNodeTag` (combined) | `*coords.LogicalPosition` | `X`, `Y` | Added via `RegisterEntityPosition`; tracked by `GlobalPositionSystem` |
| `core.OverworldNodeComponent` | `core.OverworldNodeTag` | `*core.OverworldNodeData` | `NodeID`, `NodeTypeID`, `Category`, `OwnerID`, `EncounterID`, `Intensity`, `GrowthProgress`, `GrowthRate`, `IsContained`, `CreatedTick` | `GrowthRate` is 0 for non-threat nodes; `OwnerID` is `"player"`, `"Neutral"`, or a faction type string |
| `core.InfluenceComponent` | — | `*core.InfluenceData` | `Radius int`, `BaseMagnitude float64` | Threat nodes: radius = `NodeDef.BaseRadius + Intensity`, magnitude derived from intensity. Player nodes: radius = `NodeDef.BaseRadius`, magnitude = config default |
| `core.InteractionComponent` | `core.InteractionTag` | `*core.InteractionData` | `Interactions []NodeInteraction`, `NetModifier float64` | **Dynamic.** Added and cleared each tick by `UpdateInfluenceInteractions()`; only present when the node overlaps another node's influence radius |
| `core.GarrisonComponent` | — | `*core.GarrisonData` | `SquadIDs []ecs.EntityID` | **Optional.** Only present when one or more squads have been garrisoned at this node |

**`core.OverworldNodeData` detail:**
```go
type OverworldNodeData struct {
    NodeID         ecs.EntityID // Own entity ID
    NodeTypeID     string       // "necromancer", "town", "watchtower", etc.
    Category       NodeCategory // "threat", "settlement", "fortress"
    OwnerID        string       // "player", "Neutral", or faction name
    EncounterID    string       // Empty for non-combat nodes
    Intensity      int          // Threat strength (0 for settlements)
    GrowthProgress float64      // Progress toward next intensity level
    GrowthRate     float64      // 0.0 for non-growing nodes
    IsContained    bool         // Suppressed by player influence
    CreatedTick    int64        // Tick when node was created
}
```

**`core.InfluenceData` detail:**
```go
type InfluenceData struct {
    Radius        int     // Tiles affected
    BaseMagnitude float64 // Effect strength
}
```

**`core.InteractionData` detail:**
```go
type InteractionData struct {
    Interactions []NodeInteraction // Active interactions this tick
    NetModifier  float64           // Combined multiplier (1.0 = no effect)
}
type NodeInteraction struct {
    TargetID     ecs.EntityID    // The other node
    Relationship InteractionType // Synergy, Competition, or Suppression
    Modifier     float64         // Positive = boost, negative = suppress
    Distance     int             // Manhattan distance
}
```

---

### Overworld Faction Entity

An AI-controlled strategic faction that expands territory, spawns threats, and raids player nodes. Multiple factions can exist simultaneously.

**Created by:** `CreateFaction()` in `campaign/overworld/faction/system.go`

| Component Variable | Query Tag | Data Type | Key Fields | Notes |
|---|---|---|---|---|
| `core.OverworldFactionComponent` | `core.OverworldFactionTag` | `*core.OverworldFactionData` | `FactionID`, `FactionType`, `Strength`, `TerritorySize`, `Disposition`, `CurrentIntent`, `GrowthRate` | `Disposition` starts at -50 (hostile); `CurrentIntent` starts as `IntentExpand` |
| `core.TerritoryComponent` | — | `*core.TerritoryData` | `OwnedTiles []coords.LogicalPosition` | List of tiles this faction controls; initialized with `homePosition` |
| `core.StrategicIntentComponent` | — | `*core.StrategicIntentData` | `Intent`, `TargetPosition`, `TicksRemaining`, `Priority` | Current AI objective; re-evaluated when `TicksRemaining` reaches 0 |
| `common.ResourceStockpileComponent` | — | `*common.ResourceStockpile` | `Gold`, `Iron`, `Wood`, `Stone` | Initial values from `gameconfig.json` `factionAI` section; spent to spawn threats |

**`core.OverworldFactionData` detail:**
```go
type OverworldFactionData struct {
    FactionID     ecs.EntityID  // Own entity ID
    FactionType   FactionType   // Necromancers, Bandits, Beasts, Orcs, Cultists
    Strength      int           // Military power
    TerritorySize int           // Tiles controlled
    Disposition   int           // -100 (hostile) to +100 (allied); starts -50
    CurrentIntent FactionIntent // Expand, Fortify, Raid, Retreat, Idle
    GrowthRate    float64       // Expansion speed
}
```

**`core.StrategicIntentData` detail:**
```go
type StrategicIntentData struct {
    Intent         FactionIntent
    TargetPosition *coords.LogicalPosition // Where to expand/raid (may be nil)
    TicksRemaining int                     // Ticks until re-evaluation
    Priority       float64                 // 0.0-1.0
}
```

**Side effect at creation:** `CreateFaction` immediately calls `SpawnThreatForFaction`, which creates an initial threat node at `homePosition` (if the faction can afford the resource cost).

---

### Tick State Entity (Singleton)

Global tick counter for the entire overworld simulation. Exactly one per game session.

**Created by:** `CreateTickStateEntity()` in `campaign/overworld/tick/tickmanager.go`

| Component Variable | Query Tag | Data Type | Key Fields | Notes |
|---|---|---|---|---|
| `core.TickStateComponent` | `core.TickStateTag` | `*core.TickStateData` | `CurrentTick int64`, `IsGameOver bool` | Starts at tick 0; incremented by `AdvanceTick()`; `IsGameOver = true` halts all further tick processing |

**`core.TickStateData` detail:**
```go
type TickStateData struct {
    CurrentTick int64 // Global tick counter
    IsGameOver  bool  // true = no more ticks processed
}
```

**Note:** Creating this entity also calls `core.StartRecordingSession(0)` to begin the overworld event log.

---

### Victory State Entity (Singleton)

Tracks win/loss condition parameters and current victory state. Exactly one per game session.

**Created by:** `CreateVictoryStateEntity()` in `campaign/overworld/victory/system.go`

| Component Variable | Query Tag | Data Type | Key Fields | Notes |
|---|---|---|---|---|
| `core.VictoryStateComponent` | `core.VictoryStateTag` | `*core.VictoryStateData` | `Condition`, `TicksToSurvive`, `TargetFactionType`, `VictoryAchieved`, `DefeatReason` | `Condition` starts as `VictoryNone`; updated by `CheckVictoryCondition()` each tick |

**`core.VictoryStateData` detail:**
```go
type VictoryStateData struct {
    Condition         VictoryCondition // None, PlayerWins, PlayerLoses, TimeLimit, FactionDefeat
    TicksToSurvive    int64            // For survival victory (0 = not a survival game)
    TargetFactionType FactionType      // For faction-elimination victory (0 = any)
    VictoryAchieved   bool
    DefeatReason      string           // Set when player loses
}
```

**Victory conditions evaluated by `CheckVictoryCondition()`:**
1. **Defeat** (highest priority): All player commanders destroyed, or another defeat condition
2. **Survival**: `CurrentTick >= TicksToSurvive`
3. **Threat Elimination**: All threat nodes destroyed (only if no survival condition)
4. **Faction Defeat**: All factions of `TargetFactionType` eliminated

---

### Interaction Entity (Dynamic Component)

The `InteractionComponent` is not a separate entity type. It is a dynamic component added to and removed from `OverworldNode` entities during each tick's influence resolution pass. See [Overworld Node Entity](#overworld-node-entity) for the full component definition.

**Managed by:** `UpdateInfluenceInteractions()` in `campaign/overworld/influence/system.go`

The component is cleared at the start of each tick (`clearStaleInteractions`) and re-added for any pair of overlapping nodes. The query tag `core.InteractionTag` enumerates all nodes that currently have active interactions.

---

## Encounters & Raids Domain

### Encounter Entity

Metadata for a single combat encounter, created when a player engages a threat node or a faction raids a garrison. Carries display name, difficulty, type, and the link back to the originating threat node.

**Created by:** `createOverworldEncounter()` in `mind/encounter/encounter_trigger.go`

| Component Variable | Query Tag | Data Type | Key Fields | Notes |
|---|---|---|---|---|
| `core.OverworldEncounterComponent` | `core.OverworldEncounterTag` | `*core.OverworldEncounterData` | `Name`, `Level`, `EncounterType`, `IsDefeated`, `ThreatNodeID`, `IsGarrisonDefense`, `AttackingFactionType` | `ThreatNodeID = 0` for debug/random encounters; `IsGarrisonDefense` and `AttackingFactionType` only set for garrison defense scenarios |

**`core.OverworldEncounterData` detail:**
```go
type OverworldEncounterData struct {
    Name          string       // Display name (e.g., "Goblin Patrol (Level 3)")
    Level         int          // Difficulty, derived from threat intensity
    EncounterType string       // Encounter type ID from encounterdata.json
    IsDefeated    bool         // Set true after player wins
    ThreatNodeID  ecs.EntityID // Originating threat node (0 = none)

    // Garrison defense only:
    IsGarrisonDefense    bool
    AttackingFactionType FactionType
}
```

**Creation paths:**
- `TriggerCombatFromThreat()` — standard player-engages-threat flow
- `TriggerGarrisonDefense()` — faction raids a garrisoned player node
- `TriggerRandomEncounter()` — debug/testing with no overworld side effects

---

### Raid State Entity

Singleton within a raid session. Tracks global progress across all floors of a garrison defense.

**Created by:** `GenerateGarrison()` in `campaign/raid/garrison.go`

| Component Variable | Query Tag | Data Type | Key Fields | Notes |
|---|---|---|---|---|
| `raid.RaidStateComponent` | `raid.RaidStateTag` | `*raid.RaidStateData` | `CurrentFloor`, `TotalFloors`, `Status`, `CommanderID`, `PlayerEntityID`, `PlayerSquadIDs` | `Status` starts as `RaidActive`; transitions to `RaidVictory`, `RaidDefeat`, or `RaidRetreated` |

**`raid.RaidStateData` detail:**
```go
type RaidStateData struct {
    CurrentFloor   int            // Which floor player is on
    TotalFloors    int            // How many floors in this garrison
    Status         RaidStatus     // Active, Victory, Defeat, Retreated
    CommanderID    ecs.EntityID   // Commander leading the raid
    PlayerEntityID ecs.EntityID   // Player entity (for resource access)
    PlayerSquadIDs []ecs.EntityID // Squads available for the raid
}
```

---

### Floor State Entity

Tracks the clear state of a single floor within a garrison raid. One entity per floor.

**Created by:** `generateFloor()` in `campaign/raid/garrison.go`

| Component Variable | Query Tag | Data Type | Key Fields | Notes |
|---|---|---|---|---|
| `raid.FloorStateComponent` | `raid.FloorStateTag` | `*raid.FloorStateData` | `FloorNumber`, `RoomsCleared`, `RoomsTotal`, `GarrisonSquadIDs`, `ReserveSquadIDs`, `IsComplete` | `GarrisonSquadIDs` = squads assigned to rooms; `ReserveSquadIDs` = floating reinforcements |

**`raid.FloorStateData` detail:**
```go
type FloorStateData struct {
    FloorNumber      int
    RoomsCleared     int
    RoomsTotal       int
    GarrisonSquadIDs []ecs.EntityID // Room-assigned squads
    ReserveSquadIDs  []ecs.EntityID // Floating reinforcements
    IsComplete       bool
}
```

---

### Alert Data Entity

Tracks the alert level for a single garrison floor. Alert level rises as the player engages rooms. One entity per floor.

**Created by:** `generateFloor()` in `campaign/raid/garrison.go`

| Component Variable | Query Tag | Data Type | Key Fields | Notes |
|---|---|---|---|---|
| `raid.AlertDataComponent` | `raid.AlertDataTag` | `*raid.AlertData` | `FloorNumber`, `CurrentLevel`, `EncounterCount` | `CurrentLevel` is 0-3; `EncounterCount` tracks how many combats have been fought on this floor |

**`raid.AlertData` detail:**
```go
type AlertData struct {
    FloorNumber    int
    CurrentLevel   int // 0-3; higher = more reinforcements triggered
    EncounterCount int // Combats fought on this floor
}
```

---

### Room Data Entity

Represents one room within a floor's directed acyclic graph (DAG). Rooms unlock in dependency order: a room becomes accessible once all parent rooms are cleared.

**Created by:** `buildFloorGraph()` in `campaign/raid/floorgraph.go`; called by `generateFloor()`

| Component Variable | Query Tag | Data Type | Key Fields | Notes |
|---|---|---|---|---|
| `raid.RoomDataComponent` | `raid.RoomDataTag` | `*raid.RoomData` | `NodeID`, `RoomType`, `FloorNumber`, `IsCleared`, `IsAccessible`, `GarrisonSquadIDs`, `ChildNodeIDs`, `ParentNodeIDs`, `OnCriticalPath` | Entry room starts with `IsAccessible = true`; others start `false`; `GarrisonSquadIDs` populated after `buildFloorGraph` by `generateFloor` |

**`raid.RoomData` detail:**
```go
type RoomData struct {
    NodeID           int            // Node ID within this floor's DAG
    RoomType         string         // From worldmap.GarrisonRoomType constants
    FloorNumber      int
    IsCleared        bool           // Player has defeated this room's garrison
    IsAccessible     bool           // Player can enter (all parents cleared)
    GarrisonSquadIDs []ecs.EntityID // Squads defending this room
    ChildNodeIDs     []int          // Rooms unlocked when this is cleared
    ParentNodeIDs    []int          // Rooms that must be cleared first
    OnCriticalPath   bool           // On the mandatory path to the stairs
}
```

---

### Garrison Squad Entity

A Squad Entity (standard `squadcore.SquadComponent`) with an additional `GarrisonSquadComponent` marking it as part of the garrison defense system. See [Squad Entity](#squad-entity) for the base squad components.

**Created by:** `InstantiateGarrisonSquad()` in `campaign/raid/garrison.go`; builds via `squadcore.CreateSquadFromTemplate()`

All standard Squad Entity components apply. The additional component is:

| Component Variable | Query Tag | Data Type | Key Fields | Notes |
|---|---|---|---|---|
| `raid.GarrisonSquadComponent` | `raid.GarrisonSquadTag` | `*raid.GarrisonSquadData` | `ArchetypeName`, `FloorNumber`, `RoomNodeID`, `IsReserve`, `IsDestroyed` | `RoomNodeID = -1` for reserve squads; `IsDestroyed` set true when squad is eliminated in combat |

**`raid.GarrisonSquadData` detail:**
```go
type GarrisonSquadData struct {
    ArchetypeName string // Which archetype definition this squad came from
    FloorNumber   int    // Which floor this squad belongs to
    RoomNodeID    int    // Which room this squad defends (-1 = reserve)
    IsReserve     bool   // true = floating reinforcement
    IsDestroyed   bool   // true = eliminated in combat
}
```

**Note:** Garrison squads are created with a dummy position `{0, 0}`. They receive their actual combat position when placed on the battle map at the start of a room encounter.

---

### Deployment Entity

Singleton that records which player squads are deployed (active in the upcoming encounter) versus held in reserve. Exactly one entity should exist during encounter setup; it is created or reused by `SetDeployment()`.

**Created by:** `SetDeployment()` in `campaign/raid/deployment.go`

| Component Variable | Query Tag | Data Type | Key Fields | Notes |
|---|---|---|---|---|
| `raid.DeploymentComponent` | `raid.DeploymentTag` | `*raid.DeploymentData` | `DeployedSquadIDs []ecs.EntityID`, `ReserveSquadIDs []ecs.EntityID` | If an entity with `DeploymentTag` already exists, `SetDeployment()` reuses it (updating the component) rather than creating a new entity; maximum deployed squads enforced by `MaxDeployedPerEncounter()` |

**`raid.DeploymentData` detail:**
```go
type DeploymentData struct {
    DeployedSquadIDs []ecs.EntityID // Squads active in the current encounter
    ReserveSquadIDs  []ecs.EntityID // Squads held back
}
```

**Validation:** `SetDeployment()` rejects destroyed squads and enforces the deploy-count cap. `AutoDeploy()` uses `evaluation.CalculateSquadPower()` to rank and auto-select the strongest available squads.

---

## Component Index

Alphabetical listing of every component variable with the entity types that use it.

| Component Variable | Package | Entity Types |
|---|---|---|
| `artifacts.ArtifactInventoryComponent` | `tactical/powers/artifacts` | Player Entity |
| `artifacts.EquipmentComponent` | `tactical/powers/artifacts` | Squad Entity (when artifacts are equipped) |
| `combatstate.ActionStateComponent` | `tactical/combat/combatstate` | Action State Entity |
| `combatstate.CombatFactionComponent` | `tactical/combat/combatstate` | Combat Faction Entity |
| `combatstate.FactionMembershipComponent` | `tactical/combat/combatstate` | Squad Entity (dynamic, combat only) |
| `combatstate.TurnStateComponent` | `tactical/combat/combatstate` | Turn State Entity |
| `commander.CommanderActionStateComponent` | `tactical/commander` | Commander Entity |
| `commander.CommanderComponent` | `tactical/commander` | Commander Entity |
| `commander.CommanderRosterComponent` | `tactical/commander` | Player Entity |
| `commander.OverworldTurnStateComponent` | `tactical/commander` | Overworld Turn State Entity |
| `common.AttributeComponent` | `core/common` | Player Entity, Commander Entity, Unit Entity, Creature Entity |
| `common.NameComponent` | `core/common` | Unit Entity, Creature Entity |
| `common.PlayerComponent` | `core/common` | Player Entity |
| `common.PositionComponent` | `core/common` | Player Entity, Commander Entity, Squad Entity, Unit Entity, Creature Entity, Overworld Node Entity |
| `common.RenderableComponent` | `core/common` | Player Entity, Commander Entity, Squad Entity (optional), Unit Entity, Creature Entity |
| `common.ResourceStockpileComponent` | `core/common` | Player Entity, Overworld Faction Entity |
| `core.GarrisonComponent` | `campaign/overworld/core` | Overworld Node Entity (optional) |
| `core.InfluenceComponent` | `campaign/overworld/core` | Overworld Node Entity |
| `core.InteractionComponent` | `campaign/overworld/core` | Overworld Node Entity (dynamic) |
| `core.OverworldEncounterComponent` | `campaign/overworld/core` | Encounter Entity |
| `core.OverworldFactionComponent` | `campaign/overworld/core` | Overworld Faction Entity |
| `core.OverworldNodeComponent` | `campaign/overworld/core` | Overworld Node Entity |
| `core.StrategicIntentComponent` | `campaign/overworld/core` | Overworld Faction Entity |
| `core.TerritoryComponent` | `campaign/overworld/core` | Overworld Faction Entity |
| `core.TickStateComponent` | `campaign/overworld/core` | Tick State Entity |
| `core.TravelStateComponent` | `campaign/overworld/core` | (TravelState entity — see overworld travel system) |
| `core.VictoryStateComponent` | `campaign/overworld/core` | Victory State Entity |
| `perks.PerkRoundStateComponent` | `tactical/powers/perks` | Squad Entity (dynamic, combat only) |
| `perks.PerkSlotComponent` | `tactical/powers/perks` | Squad Entity (when perks are equipped) |
| `progression.ProgressionComponent` | `tactical/powers/progression` | Player Entity |
| `raid.AlertDataComponent` | `campaign/raid` | Alert Data Entity |
| `raid.DeploymentComponent` | `campaign/raid` | Deployment Entity |
| `raid.FloorStateComponent` | `campaign/raid` | Floor State Entity |
| `raid.GarrisonSquadComponent` | `campaign/raid` | Garrison Squad Entity |
| `raid.RaidStateComponent` | `campaign/raid` | Raid State Entity |
| `raid.RoomDataComponent` | `campaign/raid` | Room Data Entity |
| `roster.SquadRosterComponent` | `tactical/squads/roster` | Commander Entity |
| `roster.UnitRosterComponent` | `tactical/squads/roster` | Player Entity |
| `spells.ManaComponent` | `tactical/powers/spells` | Squad Entity (optional, when leader has spells) |
| `spells.SpellBookComponent` | `tactical/powers/spells` | Squad Entity (optional, when leader has spells) |
| `squadcore.AbilitySlotComponent` | `tactical/squads/squadcore` | Unit Entity (leader only) |
| `squadcore.AttackRangeComponent` | `tactical/squads/squadcore` | Unit Entity |
| `squadcore.CooldownTrackerComponent` | `tactical/squads/squadcore` | Unit Entity (leader only) |
| `squadcore.CoverComponent` | `tactical/squads/squadcore` | Unit Entity (optional, when `CoverValue > 0`) |
| `squadcore.GridPositionComponent` | `tactical/squads/squadcore` | Unit Entity |
| `squadcore.LeaderComponent` | `tactical/squads/squadcore` | Unit Entity (leader only) |
| `squadcore.MovementSpeedComponent` | `tactical/squads/squadcore` | Unit Entity |
| `squadcore.SquadComponent` | `tactical/squads/squadcore` | Squad Entity, Garrison Squad Entity |
| `squadcore.SquadMemberComponent` | `tactical/squads/squadcore` | Unit Entity (when in a squad) |
| `squadcore.TargetRowComponent` | `tactical/squads/squadcore` | Unit Entity |
| `squadcore.UnitRoleComponent` | `tactical/squads/squadcore` | Unit Entity |
| `squadcore.UnitTypeComponent` | `tactical/squads/squadcore` | Unit Entity |
| `unitprogression.ExperienceComponent` | `tactical/squads/unitprogression` | Unit Entity |
| `unitprogression.StatGrowthComponent` | `tactical/squads/unitprogression` | Unit Entity |
