# Squad and Combat System Documentation

**Last Updated:** 2026-02-18

---

## Related Documents

- [AI Algorithm Architecture](AI_ALGORITHM_ARCHITECTURE.md) - AI controller, action selection, power evaluation, configuration
- [Behavior & Threat Layers](BEHAVIOR_THREAT_LAYERS.md) - Threat layer subsystems, spatial analysis, visualization
- [Encounter System](ENCOUNTER_SYSTEM.md) - Encounter generation, lifecycle, rewards
- [Artifact System Architecture](ARTIFACT_SYSTEM.md) - Artifact data model, behaviors, charge tracking
- [GUI Documentation](GUI_DOCUMENTATION.md) - All GUI modes, panels, input handling

---

## Table of Contents

1. [Executive Summary](#1-executive-summary)
2. [Architecture Overview](#2-architecture-overview)
3. [Squad System](#3-squad-system)
   - 3.1 [ECS Components](#31-ecs-components)
   - 3.2 [Squad Creation](#32-squad-creation)
   - 3.3 [Unit Templates](#33-unit-templates)
   - 3.4 [Grid Formation System](#34-grid-formation-system)
   - 3.5 [Squad Capacity](#35-squad-capacity)
   - 3.6 [Cover System](#36-cover-system)
   - 3.7 [Leader Abilities](#37-leader-abilities)
   - 3.8 [Experience and Stat Growth](#38-experience-and-stat-growth)
   - 3.9 [Squad Queries and Caching](#39-squad-queries-and-caching)
4. [Command System](#4-command-system)
   - 4.1 [Command Interface and Executor](#41-command-interface-and-executor)
   - 4.2 [Available Commands](#42-available-commands)
5. [Service Layer](#5-service-layer)
   - 5.1 [Unit Purchase Service](#51-unit-purchase-service)
   - 5.2 [Squad Deployment Service](#52-squad-deployment-service)
6. [Combat System](#6-combat-system)
   - 6.1 [Combat Components](#61-combat-components)
   - 6.2 [Faction Manager](#62-faction-manager)
   - 6.3 [Turn Manager](#63-turn-manager)
   - 6.4 [Combat Movement System](#64-combat-movement-system)
   - 6.5 [Combat Action System](#65-combat-action-system)
   - 6.6 [Damage Calculation](#66-damage-calculation)
   - 6.7 [Targeting Logic](#67-targeting-logic)
   - 6.8 [Counterattack System](#68-counterattack-system)
   - 6.9 [Victory Conditions](#69-victory-conditions)
   - 6.10 [Combat Query Cache](#610-combat-query-cache)
7. [Combat Service](#7-combat-service)
   - 7.1 [CombatService Initialization](#71-combatservice-initialization)
   - 7.2 [Combat Lifecycle Callbacks](#72-combat-lifecycle-callbacks)
   - 7.3 [Combat Cleanup](#73-combat-cleanup)
8. [Effects System](#8-effects-system)
9. [AI System](#9-ai-system)
10. [Battle Log System](#10-battle-log-system)
11. [Gear and Artifact System](#11-gear-and-artifact-system)
12. [GUI Integration](#12-gui-integration)
13. [Data Flow: A Complete Combat Round](#13-data-flow-a-complete-combat-round)
14. [Package Dependency Map](#14-package-dependency-map)

---

## 1. Executive Summary

TinkerRogue implements a turn-based tactical combat system built on top of the `bytearena/ecs` Entity Component System library. Combat is organized around two main concepts: the **Squad** (a group of up to 9 units arranged in a 3x3 formation grid) and the **Faction** (a collection of one or more squads fighting under a single banner).

Combat proceeds in rounds. Each round, every faction takes one turn, with the order determined by a randomized Fisher-Yates shuffle at combat start. During a faction's turn, the player or AI moves and attacks with each squad. After all squads have acted, the turn ends and the next faction activates.

Damage calculation is per-unit: each unit in an attacking squad independently targets units in the defender's formation based on its `AttackType` (MeleeRow, MeleeColumn, Ranged, or Magic). A single attack resolves a hit roll, a dodge roll, a crit roll, damage calculation, armor reduction, and a cover reduction check. All damage is first accumulated without modifying HP, then applied atomically at the end of the attack phase.

The AI system uses a layered threat map architecture. At each turn, the AI updates `CombatThreatLayer`, `SupportValueLayer`, and `PositionalRiskLayer` for its faction. Each squad's `ActionEvaluator` generates candidate move and attack actions, scores them with role-specific weights, and selects the highest-scoring action.

---

## 2. Architecture Overview

The squad and combat system spans several packages:

```
tactical/
    squads/           -- Squad ECS components, unit creation, queries, combat math
    squadcommands/    -- Command pattern: undoable squad management operations
    squadservices/    -- Service wrappers for deployment and purchase
    combat/           -- Combat ECS components, TurnManager, ActionSystem, MovementSystem
        battlelog/    -- Optional JSON export of battle engagements
    combatservices/   -- CombatService facade; wires subsystems together
    effects/          -- Temporary stat modifier components
    spells/           -- Spell casting components and system
    commander/        -- Commander entity components and movement

mind/
    ai/               -- AIController, ActionEvaluator, action types
    behavior/         -- Threat layer computation (CombatThreat, Support, Positional)
    encounter/        -- Encounter setup, resolution, and rewards

gear/                 -- Artifact inventory, behavior hooks, charge tracking

gui/
    guicombat/        -- CombatMode, CombatTurnFlow, animation, input
    guisquads/        -- SquadEditorMode, SquadDeploymentMode, UnitPurchaseMode
```

Data flows in one direction: the GUI calls `CombatService` methods, which delegate to `TurnManager`, `CombatActionSystem`, and `CombatMovementSystem`. The combat subsystems read and write ECS components. The AI controller goes through the same `CombatActionSystem` and `CombatMovementSystem` as the player, ensuring identical rules apply to both.

---

## 3. Squad System

### 3.1 ECS Components

All squad-related components are declared in `tactical/squads/squadcomponents.go`. Component variables are global package-level variables initialized by the ECS subsystem registration pattern in `init()`.

**Global Component Variables** (`tactical/squads/squadcomponents.go`):

```go
var (
    SquadComponent           *ecs.Component
    SquadMemberComponent     *ecs.Component
    GridPositionComponent    *ecs.Component
    UnitRoleComponent        *ecs.Component
    CoverComponent           *ecs.Component
    LeaderComponent          *ecs.Component
    TargetRowComponent       *ecs.Component
    AbilitySlotComponent     *ecs.Component
    CooldownTrackerComponent *ecs.Component
    AttackRangeComponent     *ecs.Component
    MovementSpeedComponent   *ecs.Component
    ExperienceComponent      *ecs.Component
    StatGrowthComponent      *ecs.Component

    SquadTag       ecs.Tag
    SquadMemberTag ecs.Tag
    LeaderTag      ecs.Tag
)
```

**Squad Entity Component** (`SquadData`):

The squad entity is the top-level entity. It holds metadata about the entire squad and its world-map position via `common.PositionComponent`.

```go
type SquadData struct {
    SquadID            ecs.EntityID  // Unique squad identifier (native entity ID)
    Formation          FormationType // Current formation: Balanced, Defensive, Offensive, Ranged
    Name               string        // Squad display name
    Morale             int           // Squad-wide morale (0-100)
    SquadLevel         int           // Average level for spawning
    TurnCount          int           // Number of turns this squad has taken
    MaxUnits           int           // Maximum squad size (typically 9)
    IsDeployed         bool          // true if squad is on the tactical map
    GarrisonedAtNodeID ecs.EntityID  // 0 = not garrisoned
}
```

**Formation Types**:

```go
const (
    FormationBalanced  FormationType = iota // Mix of roles
    FormationDefensive                      // Tank-heavy
    FormationOffensive                      // DPS-focused
    FormationRanged                         // Back-line heavy
)
```

**Unit Entity Components**:

Each unit inside a squad is a separate ECS entity. Units carry the following components:

| Component | Type | Purpose |
|---|---|---|
| `SquadMemberComponent` | `*SquadMemberData` | Links unit to its parent squad via `SquadID` |
| `GridPositionComponent` | `*GridPositionData` | Position in the 3x3 formation grid |
| `UnitRoleComponent` | `*UnitRoleData` | Tank, DPS, or Support |
| `TargetRowComponent` | `*TargetRowData` | Attack type and optional target cell pattern |
| `CoverComponent` | `*CoverData` | How much cover this unit provides to units behind it |
| `AttackRangeComponent` | `*AttackRangeData` | World-tile attack range (Melee=1, Ranged=3, Magic=4) |
| `MovementSpeedComponent` | `*MovementSpeedData` | World-map movement speed in tiles per turn |
| `ExperienceComponent` | `*ExperienceData` | Level, current XP, XP to next level |
| `StatGrowthComponent` | `*StatGrowthData` | Per-stat growth grades for level-up rolls |
| `common.AttributeComponent` | `*common.Attributes` | HP, attack, defense, hit rate, etc. |
| `common.PositionComponent` | `*coords.LogicalPosition` | World position (same as the squad's position) |
| `rendering.RenderableComponent` | `*rendering.Renderable` | Sprite; set to `Visible=false` on most units |

The squad entity itself also has `rendering.RenderableComponent`, initialized to the leader's sprite via `setSquadRenderableFromLeader`. Unit renderables are invisible by default because the squad tile on the world map shows the leader's image.

**SquadMemberData**:

```go
type SquadMemberData struct {
    SquadID ecs.EntityID // Parent squad's entity ID
}
```

**GridPositionData**:

```go
type GridPositionData struct {
    AnchorRow int // Top-left row (0-2)
    AnchorCol int // Top-left col (0-2)
    Width     int // Number of columns occupied (1-3)
    Height    int // Number of rows occupied (1-3)
}
```

Multi-cell units (e.g., a 2x1 unit) occupy more than one cell. `GetOccupiedCells()` returns all cells. `OccupiesCell(row, col)` tests membership.

**UnitRole Values**:

```go
const (
    RoleTank    UnitRole = iota // Takes hits first, high defense
    RoleDPS                     // High damage output
    RoleSupport                 // Buffs, heals, utility
    RoleError                   // Invalid / error sentinel
)
```

**AttackType Values**:

```go
const (
    AttackTypeMeleeRow    AttackType = iota // Targets entire front row (up to 3 targets)
    AttackTypeMeleeColumn                   // Targets the column directly across (piercing)
    AttackTypeRanged                        // Targets same row as attacker
    AttackTypeMagic                         // Cell-based pattern, no pierce-through
)
```

### 3.2 Squad Creation

Two creation paths exist in `tactical/squads/squadcreation.go`:

**Empty squad** (for player squad management):

```go
func CreateEmptySquad(squadmanager *common.EntityManager, squadName string) ecs.EntityID
```

Creates a squad entity with `Morale=100`, `MaxUnits=9`, `IsDeployed=false`, and a zero-value `LogicalPosition`. Returns the squad's entity ID.

**Populated squad from templates** (for encounter spawning):

```go
func CreateSquadFromTemplate(
    ecsmanager *common.EntityManager,
    squadName string,
    formation FormationType,
    worldPos coords.LogicalPosition,
    unitTemplates []UnitTemplate,
) ecs.EntityID
```

Iterates the `unitTemplates` slice, validates grid positions and capacity for each unit, creates unit entities, sets all components (membership, grid position, role, targeting, cover, range, speed, experience, growth, leader), and marks occupied cells to prevent overlap. After all units are placed, the squad's renderable is set to the leader's sprite.

**Adding a single unit to an existing squad**:

```go
func AddUnitToSquad(
    squadID ecs.EntityID,
    squadmanager *common.EntityManager,
    unit UnitTemplate,
    gridRow, gridCol int,
) (ecs.EntityID, error)
```

Validates the grid position is in bounds (0-2), checks for occupancy conflicts, checks squad capacity, creates the unit entity via `CreateUnitEntity`, adds `SquadMemberComponent`, and updates the `GridPositionData` with the actual grid row/col.

**Removing a unit**:

```go
func RemoveUnitFromSquad(unitEntityID ecs.EntityID, squadmanager *common.EntityManager) error
```

Disposes the unit entity using `CleanDisposeEntity`, which removes it from the position system and the ECS world.

**Moving a unit within the grid**:

```go
func MoveUnitInSquad(unitEntityID ecs.EntityID, newRow, newCol int, ecsmanager *common.EntityManager) error
```

Validates the new anchor position fits in the grid (considering multi-cell width/height), checks that no other unit occupies any of the target cells, and updates `GridPositionData.AnchorRow` / `AnchorCol`.

**Squad disposal**:

```go
func DisposeDeadUnitsInSquad(squadID ecs.EntityID, manager *common.EntityManager) int
func DisposeSquadAndUnits(squadID ecs.EntityID, manager *common.EntityManager)
```

`DisposeDeadUnitsInSquad` disposes units with `CurrentHealth <= 0` and returns the count. `DisposeSquadAndUnits` disposes all units and the squad entity itself; used when a squad is eliminated from combat.

### 3.3 Unit Templates

Unit templates are the data-driven specification for creating unit entities. The `UnitTemplate` struct in `tactical/squads/units.go` holds all fields needed by `CreateUnitEntity`:

```go
type UnitTemplate struct {
    Name         string
    Attributes   common.Attributes
    EntityType   templates.EntityType
    EntityConfig templates.EntityConfig
    EntityData   any           // JSONMonster or other source data
    GridRow      int           // Anchor row (0-2)
    GridCol      int           // Anchor col (0-2)
    GridWidth    int           // Width in cells (1-3), defaults to 1
    GridHeight   int           // Height in cells (1-3), defaults to 1
    Role         UnitRole
    AttackType   AttackType
    TargetCells  [][2]int      // For magic: pattern cells
    IsLeader     bool
    CoverValue   float64       // Damage reduction provided (0.0-1.0)
    CoverRange   int           // Rows behind that receive cover
    RequiresActive bool
    AttackRange  int
    MovementSpeed int
    StatGrowths  StatGrowthData
}
```

Templates are loaded from JSON monster data at startup via `InitUnitTemplatesFromJSON()`, which iterates `templates.MonsterTemplates` and calls `CreateUnitTemplates(monster)`.

`GetRole(roleString string)` converts JSON strings "Tank", "DPS", "Support" to `UnitRole` values.

`GetAttackType(attackTypeString string, attackRange int)` converts "MeleeRow", "MeleeColumn", "Ranged", "Magic" strings to `AttackType` values, with a fallback from `attackRange` integer for backward compatibility.

`GetTemplateByName(name string) *UnitTemplate` finds a loaded template by name.

### 3.4 Grid Formation System

Each squad uses a 3x3 grid where units occupy one or more cells. Row 0 is the front row (faces the enemy) and row 2 is the back row. Columns are 0, 1, 2 from left to right.

The grid determines targeting: melee attacks target the front row, ranged attacks target the corresponding row, and column attacks pierce through an entire column. Units at lower row numbers are considered to be in front of units at higher row numbers for cover calculation.

Multi-cell units have an anchor (top-left cell) plus `Width` and `Height` fields. A 2x1 unit at `AnchorRow=0, AnchorCol=1, Width=2, Height=1` occupies cells (0,1) and (0,2).

`GetOccupiedCells()` returns `[][2]int` of all occupied cells:

```go
func (g *GridPositionData) GetOccupiedCells() [][2]int {
    // iterates r from AnchorRow to AnchorRow+Height-1
    // iterates c from AnchorCol to AnchorCol+Width-1
    // clamps at 3 for safety
}
```

`GetUnitIDsAtGridPosition(squadID, row, col, manager)` returns all unit IDs occupying a specific cell, supporting multi-cell units via `OccupiesCell`.

### 3.5 Squad Capacity

Squad capacity is governed by the leader's `Leadership` attribute via `attr.GetUnitCapacity()`. Each unit has a capacity cost computed by `attr.GetCapacityCost()`.

```go
const DefaultSquadCapacity = 6 // Base capacity for squads without a leader

func GetSquadTotalCapacity(squadID ecs.EntityID, squadmanager *common.EntityManager) int
func GetSquadUsedCapacity(squadID ecs.EntityID, squadmanager *common.EntityManager) float64
func GetSquadRemainingCapacity(squadID ecs.EntityID, squadmanager *common.EntityManager) float64
func CanAddUnitToSquad(squadID ecs.EntityID, unitCapacityCost float64, squadmanager *common.EntityManager) bool
func IsSquadOverCapacity(squadID ecs.EntityID, squadmanager *common.EntityManager) bool
```

If a squad has no leader, `GetSquadTotalCapacity` returns `DefaultSquadCapacity` (6).

### 3.6 Cover System

Units with `CoverComponent` provide damage reduction to units behind them in the same column(s).

```go
type CoverData struct {
    CoverValue     float64 // Damage reduction percentage (0.0-1.0)
    CoverRange     int     // Rows behind that receive cover
    RequiresActive bool    // If true, dead/stunned units don't provide cover
}
```

`GetCoverProvidersFor(defenderID, defenderSquadID, defenderPos, manager)` finds all units that provide cover to the defender. A unit provides cover when it:

1. Has a `CoverComponent`
2. Is in a row with a lower index (further forward) than the defender
3. Is within `CoverRange` rows
4. Occupies at least one column that the defender also occupies

Cover reduction is calculated in `CalculateCoverBreakdown(defenderID, manager)`, which sums all provider cover values and caps the total at 1.0 (100% reduction). The resulting `CoverBreakdown.TotalReduction` is applied as `finalDamage = int(float64(damage) * (1.0 - totalReduction))`, with a floor of 1.

`CoverBreakdown` struct (returned for logging):

```go
type CoverBreakdown struct {
    Providers      []CoverProvider
    TotalReduction float64
}

type CoverProvider struct {
    UnitID     ecs.EntityID
    UnitName   string
    CoverValue float64
    GridRow    int
    GridCol    int
}
```

### 3.7 Leader Abilities

The squad leader is identified by `LeaderTag` (requires `LeaderComponent`). Leaders have four ability slots stored in `AbilitySlotData` and cooldown tracking in `CooldownTrackerData`.

```go
type LeaderData struct {
    Leadership int // Bonus to squad stats
    Experience int // Leader progression (future)
}

type AbilitySlotData struct {
    Slots [4]AbilitySlot
}

type AbilitySlot struct {
    AbilityType  AbilityType // Rally, Heal, BattleCry, Fireball
    TriggerType  TriggerType // When to activate
    Threshold    float64     // Condition threshold
    HasTriggered bool        // Once-per-combat flag
    IsEquipped   bool        // Whether slot is active
}

type CooldownTrackerData struct {
    Cooldowns    [4]int // Turns remaining for slots 0-3
    MaxCooldowns [4]int // Base cooldown durations
}
```

**Ability Types**:

| Ability | Effect | BaseCooldown |
|---|---|---|
| `AbilityRally` | +5 strength to all units for 3 turns | 5 turns |
| `AbilityHeal` | Restore 10 HP to all alive units | 4 turns |
| `AbilityBattleCry` | +10 morale, +3 strength | 999 (once per combat) |
| `AbilityFireball` | 15 direct damage to all units in first enemy squad | 3 turns |

**Trigger Types**:

| Trigger | Condition |
|---|---|
| `TriggerSquadHPBelow` | Squad average HP% < threshold |
| `TriggerTurnCount` | Squad's `TurnCount == int(threshold)` |
| `TriggerCombatStart` | Squad's `TurnCount == 1` |
| `TriggerEnemyCount` | Enemy squad count >= threshold |
| `TriggerMoraleBelow` | Squad morale < threshold |

`CheckAndTriggerAbilities(squadID, manager)` is called at combat start and at the start of each faction's turn. It finds the leader, iterates the four slots, evaluates each trigger, executes the ability if triggered, sets the cooldown, and marks `HasTriggered = true`. Then it decrements all non-zero cooldowns.

`EquipAbilityToLeader(leaderEntityID, slotIndex, abilityType, triggerType, threshold, manager)` modifies an ability slot and updates `CooldownTrackerData`.

`AddLeaderComponents` and `RemoveLeaderComponents` manage the three leader-related components (`LeaderComponent`, `AbilitySlotComponent`, `CooldownTrackerComponent`) atomically.

### 3.8 Experience and Stat Growth

Every unit starts at level 1 with `ExperienceData{Level: 1, CurrentXP: 0, XPToNextLevel: 100}`.

`StatGrowthData` defines per-stat growth rates using letter grades:

```go
type StatGrowthData struct {
    Strength   GrowthGrade // S/A/B/C/D/E/F
    Dexterity  GrowthGrade
    Magic      GrowthGrade
    Leadership GrowthGrade
    Armor      GrowthGrade
    Weapon     GrowthGrade
}
```

Grade-to-probability mapping:

| Grade | Chance of +1 on level up |
|---|---|
| S | 90% |
| A | 75% |
| B | 60% |
| C | 45% |
| D | 30% |
| E | 15% |
| F | 5% |

XP award logic and the level-up roll are implemented in `tactical/squads/experience.go`. Combat events that award XP are processed in `tactical/squads/combatevents.go`.

### 3.9 Squad Queries and Caching

Two APIs exist for squad queries:

**Canonical API** (`squadqueries.go`): Uses `World.Query()` on every call. Correct for game logic, systems, and tests. O(n) over all entities.

```go
func GetUnitIDsInSquad(squadID ecs.EntityID, squadmanager *common.EntityManager) []ecs.EntityID
func GetUnitIDsAtGridPosition(squadID ecs.EntityID, row, col int, manager *common.EntityManager) []ecs.EntityID
func GetSquadEntity(squadID ecs.EntityID, squadmanager *common.EntityManager) *ecs.Entity
func GetLeaderID(squadID ecs.EntityID, squadmanager *common.EntityManager) ecs.EntityID
func IsSquadDestroyed(squadID ecs.EntityID, squadmanager *common.EntityManager) bool
func WouldSquadSurvive(squadID ecs.EntityID, predictedDamage map[ecs.EntityID]int, manager *common.EntityManager) bool
func GetSquadName(squadID ecs.EntityID, squadmanager *common.EntityManager) string
func GetSquadHealthPercent(squadID ecs.EntityID, manager *common.EntityManager) float64
func GetSquadPrimaryRole(squadID ecs.EntityID, manager *common.EntityManager) UnitRole
func GetSquadDistance(squad1ID, squad2ID ecs.EntityID, squadmanager *common.EntityManager) int
func GetSquadMovementSpeed(squadID ecs.EntityID, squadmanager *common.EntityManager) int
```

`GetSquadDistance` uses Chebyshev distance (max of |dx|, |dy|), consistent with the movement system.

`GetSquadMovementSpeed` returns the minimum `MovementSpeedData.Speed` across all alive units with a `MovementSpeedComponent`. Returns 0 if no valid units are found.

**Cached API** (`squadcache.go`): Uses ECS `View` objects which are auto-maintained by the library. O(k) where k is the number of entities with the relevant component.

```go
type SquadQueryCache struct {
    SquadView       *ecs.View
    SquadMemberView *ecs.View
    LeaderView      *ecs.View
}

func (c *SquadQueryCache) GetSquadEntity(squadID ecs.EntityID) *ecs.Entity
func (c *SquadQueryCache) GetUnitIDsInSquad(squadID ecs.EntityID) []ecs.EntityID
func (c *SquadQueryCache) GetLeaderID(squadID ecs.EntityID) ecs.EntityID
func (c *SquadQueryCache) GetSquadName(squadID ecs.EntityID) string
func (c *SquadQueryCache) FindAllSquads() []ecs.EntityID
```

The cached API is preferred for GUI hot paths (e.g., list refresh on every frame). Both APIs return identical results.

---

## 4. Command System

### 4.1 Command Interface and Executor

The `squadcommands` package implements the Command design pattern for all squad management operations. Commands are validated, executed, and optionally undone. File: `tactical/squadcommands/command.go`.

```go
type SquadCommand interface {
    Validate() error
    Execute() error
    Undo() error
    Description() string
}
```

`CommandExecutor` manages an undo/redo history stack with a configurable maximum size (defaults to 20):

```go
type CommandExecutor struct {
    history    []SquadCommand
    redoStack  []SquadCommand
    maxHistory int
}

func (ce *CommandExecutor) Execute(cmd SquadCommand) *CommandResult
func (ce *CommandExecutor) Undo() *CommandResult
func (ce *CommandExecutor) Redo() *CommandResult
func (ce *CommandExecutor) CanUndo() bool
func (ce *CommandExecutor) CanRedo() bool
func (ce *CommandExecutor) GetHistoryCount() int
func (ce *CommandExecutor) ClearHistory()
```

`Execute` validates before executing. On success, the command is pushed onto `history` and `redoStack` is cleared. `Undo` pops from `history` and pushes to `redoStack`. `Redo` pops from `redoStack` and pushes back to `history`.

### 4.2 Available Commands

| Command File | Purpose |
|---|---|
| `add_unit_command.go` | Adds a unit from the player's roster to a squad at a specific grid position |
| `remove_unit_command.go` | Removes a unit from a squad and returns it to the roster |
| `move_unit_command.go` | Moves a unit to a different grid position within the same squad |
| `move_squad_command.go` | Moves a squad to a new world map position during combat |
| `change_leader_command.go` | Changes the leader unit within a squad |
| `purchase_unit_command.go` | Purchases a new unit and adds it to the player's roster |
| `rename_squad_command.go` | Renames a squad |
| `reorder_squads_command.go` | Reorders squads within the roster |

**AddUnitCommand** (`tactical/squadcommands/add_unit_command.go`):

```go
type AddUnitCommand struct {
    manager      *common.EntityManager
    playerID     ecs.EntityID
    squadID      ecs.EntityID
    templateName string
    gridRow      int
    gridCol      int
    addedUnitID  ecs.EntityID // set during Execute, used by Undo
}
```

`Validate` checks: squad exists, roster exists, template is available in roster, grid position is valid and unoccupied. `Execute` retrieves the unit entity from the roster, creates a `UnitTemplate` from its current attributes, calls `AddUnitToSquad`, adds the new entity ID to the roster, and marks it as in-squad. `Undo` marks the unit available in the roster, removes it from the squad (disposing the entity), and removes it from the roster.

**MoveSquadCommand** (`tactical/squadcommands/move_squad_command.go`):

Used by the AI and can be triggered by the player. Delegates to `CombatMovementSystem.MoveSquad()` after validation.

---

## 5. Service Layer

### 5.1 Unit Purchase Service

`tactical/squadservices/unit_purchase_service.go` provides the transaction logic for purchasing units from a shop.

```go
type UnitPurchaseService struct {
    entityManager *common.EntityManager
}

func (ups *UnitPurchaseService) GetUnitCost(template squads.UnitTemplate) int
func (ups *UnitPurchaseService) CanPurchaseUnit(playerID ecs.EntityID, template squads.UnitTemplate) *PurchaseValidationResult
func (ups *UnitPurchaseService) PurchaseUnit(playerID ecs.EntityID, template squads.UnitTemplate) *PurchaseResult
func (ups *UnitPurchaseService) RefundUnitPurchase(playerID ecs.EntityID, unitID ecs.EntityID, costPaid int) *RefundResult
func (ups *UnitPurchaseService) GetPlayerPurchaseInfo(playerID ecs.EntityID) *PlayerPurchaseInfo
func (ups *UnitPurchaseService) GetUnitOwnedCount(playerID ecs.EntityID, templateName string) (totalOwned, available int)
```

`PurchaseUnit` is an atomic three-step transaction with rollback: (1) create unit entity, (2) add to roster, (3) deduct gold. If any step fails, earlier steps are reversed. Cost is currently calculated from the unit name via a hash-based formula (`baseCost = 100 + sum(charValue % 50)`); a proper cost field in `UnitTemplate` is a noted TODO.

`PurchaseValidationResult` contains:

- `CanPurchase bool`
- `Error string`
- `PlayerGold int`
- `UnitCost int`
- `RosterCount, RosterCapacity int`

### 5.2 Squad Deployment Service

`tactical/squadservices/squad_deployment_service.go` manages placing squads on the world map before combat.

```go
type SquadDeploymentService struct {
    entityManager *common.EntityManager
}

func (sds *SquadDeploymentService) ClearAllSquadPositions() *ClearAllSquadsResult
func (sds *SquadDeploymentService) GetAllSquadPositions() map[ecs.EntityID]coords.LogicalPosition
```

`ClearAllSquadPositions` resets every squad with a `PositionComponent` to position (0, 0) using `manager.MoveEntity` for atomic updates.

`GetAllSquadPositions` returns a snapshot of all current squad positions.

---

## 6. Combat System

### 6.1 Combat Components

All combat components are declared in `tactical/combat/combatcomponents.go` and self-registered via `init()`:

```go
var (
    CombatFactionComponent     *ecs.Component
    TurnStateComponent         *ecs.Component
    ActionStateComponent       *ecs.Component
    FactionMembershipComponent *ecs.Component

    FactionTag       ecs.Tag
    TurnStateTag     ecs.Tag
    ActionStateTag   ecs.Tag
    CombatFactionTag ecs.Tag
)
```

**FactionData** (attached to the faction entity):

```go
type FactionData struct {
    FactionID          ecs.EntityID
    Name               string
    Mana               int
    MaxMana            int
    IsPlayerControlled bool
    PlayerID           int          // 0 = AI
    PlayerName         string
    EncounterID        ecs.EntityID // 0 if not from encounter
}
```

**TurnStateData** (one entity per combat):

```go
type TurnStateData struct {
    CurrentRound     int
    TurnOrder        []ecs.EntityID // Faction IDs in randomized order
    CurrentTurnIndex int
    CombatActive     bool
}
```

**ActionStateData** (one entity per squad per combat):

```go
type ActionStateData struct {
    SquadID           ecs.EntityID
    HasMoved          bool
    HasActed          bool
    MovementRemaining int  // Tiles left to move (starts at squad speed)
    BonusAttackActive bool // When true, next markSquadAsActed is consumed without setting HasActed
}
```

`BonusAttackActive` is set by artifact behaviors (specifically the Twin Strike Banner artifact) to grant a free additional attack.

**CombatFactionData** (attached to squad entities during combat):

```go
type CombatFactionData struct {
    FactionID ecs.EntityID
}
```

A squad entity gains `FactionMembershipComponent` when it enters combat and loses it when removed from the map. This component is how the combat system distinguishes active combat squads from reserve squads.

### 6.2 Faction Manager

`tactical/combat/combatfactionmanager.go` manages faction creation and squad assignment.

```go
type CombatFactionManager struct {
    manager     *common.EntityManager
    combatCache *CombatQueryCache
}

func (fm *CombatFactionManager) CreateCombatFaction(name string, isPlayer bool) ecs.EntityID
func (fm *CombatFactionManager) CreateFactionWithPlayer(name string, playerID int, playerName string, encounterID ecs.EntityID) ecs.EntityID
func (fm *CombatFactionManager) AddSquadToFaction(factionID, squadID ecs.EntityID, position coords.LogicalPosition) error
func (fm *CombatFactionManager) GetFactionMana(factionID ecs.EntityID) (current, max int)
func (fm *CombatFactionManager) GetFactionName(factionID ecs.EntityID) string
```

`AddSquadToFaction` adds `FactionMembershipComponent` to the squad entity with the faction's ID. If the squad has no position, it adds `PositionComponent` and registers in `GlobalPositionSystem`. If the squad already has a position, it moves it atomically via `manager.MoveEntity`.

### 6.3 Turn Manager

`tactical/combat/turnmanager.go` manages the round/turn lifecycle.

```go
type TurnManager struct {
    manager           *common.EntityManager
    combatCache       *CombatQueryCache
    turnStateEntityID ecs.EntityID
    movementSystem    *CombatMovementSystem
    onTurnEnd         func(round int)
    postResetHook     func(factionID ecs.EntityID, squadIDs []ecs.EntityID)
}
```

**Initialization** (`InitializeCombat(factionIDs []ecs.EntityID) error`):

1. Shuffles `factionIDs` using Fisher-Yates into `TurnOrder`
2. Creates the `TurnStateData` entity (one per combat)
3. Caches `turnStateEntityID` to avoid O(n) queries
4. Creates `ActionStateData` entities for all squads in all factions
5. Calls `CheckAndTriggerAbilities` for each squad (combat-start abilities like BattleCry)
6. Calls `ResetSquadActions` for the first faction

**ResetSquadActions** (`ResetSquadActions(factionID ecs.EntityID) error`):

For each squad in the faction:
- Sets `HasMoved = false`, `HasActed = false`, `BonusAttackActive = false`
- Sets `MovementRemaining` from the squad's current movement speed
- Calls `effects.TickEffectsForUnits` to decrement and remove expired effects
- Calls `squads.CheckAndTriggerAbilities` to check ability triggers

After processing all squads, fires `postResetHook` (used by artifact behaviors).

**EndTurn** (`EndTurn() error`):

Increments `CurrentTurnIndex`. If it reaches the end of `TurnOrder`, wraps around to 0 and increments `CurrentRound`. Calls `ResetSquadActions` for the new faction and fires `onTurnEnd`.

**GetCurrentFaction** returns the faction ID at `TurnOrder[CurrentTurnIndex]`.

**GetCurrentRound** returns `TurnStateData.CurrentRound`.

**EndCombat** sets `CombatActive = false` and invalidates `turnStateEntityID`.

### 6.4 Combat Movement System

`tactical/combat/combatmovementsystem.go` manages world-map squad movement during combat.

```go
type CombatMovementSystem struct {
    manager     *common.EntityManager
    posSystem   *common.PositionSystem
    combatCache *CombatQueryCache
    onMoveComplete func(squadID ecs.EntityID)
}
```

**GetSquadMovementSpeed** delegates to `squads.GetSquadMovementSpeed`. Falls back to `DefaultMovementSpeed` (3) if the squad returns 0.

**CanMoveTo** checks whether a target position is unoccupied using `posSystem.GetEntityIDAt`. Any entity at the position (whether squad or terrain) blocks movement. Squads cannot share positions with any other entity.

**MoveSquad** (`MoveSquad(squadID ecs.EntityID, targetPos coords.LogicalPosition) error`):

1. Validates `HasMoved` state via `canSquadMove` (checks `MovementRemaining > 0`)
2. Gets current position
3. Computes `movementCost = currentPos.ChebyshevDistance(&targetPos)`
4. Checks `MovementRemaining >= movementCost`
5. Validates `CanMoveTo`
6. Calls `manager.MoveSquadAndMembers` to atomically update positions for the squad and all its unit entities
7. Calls `decrementMovementRemaining` and `markSquadAsMoved`
8. Fires `onMoveComplete`

**GetValidMovementTiles** (`GetValidMovementTiles(squadID ecs.EntityID) []coords.LogicalPosition`):

Performs a simple flood-fill within the Chebyshev distance equal to `MovementRemaining`. For each candidate position, calls `CanMoveTo` to validate. Returns all passable positions.

### 6.5 Combat Action System

`tactical/combat/combatactionsystem.go` orchestrates complete attack resolution.

```go
type CombatActionSystem struct {
    manager        *common.EntityManager
    combatCache    *CombatQueryCache
    battleRecorder *battlelog.BattleRecorder
    onAttackComplete func(attackerID, defenderID ecs.EntityID, result *squads.CombatResult)
}
```

**ExecuteAttackAction** (`ExecuteAttackAction(attackerID, defenderID ecs.EntityID) *squads.CombatResult`):

The complete attack resolution sequence:

1. **Validation**: `canSquadAttackWithReason` checks action state, positions, faction membership, same-faction restriction, and range.
2. **Combat log initialization**: Creates `CombatLog` with squad names and distance. Snapshots attacking units and all defender units for logging.
3. **Main attack loop**: Iterates attacker's units. For each unit, calls `CanUnitAttack` (alive, `CanAct=true`, range >= distance). For units that can attack, calls `SelectTargetUnits` then `ProcessAttackOnTargets`.
4. **Counterattack check**: Calls `squads.WouldSquadSurvive(defenderID, result.DamageByUnit, manager)`. If the defender survives, gets counterattacking units (alive defenders in range, `CanAct` not required). For each counterattacker, verifies they survive the incoming damage, then calls `ProcessCounterattackOnTargets`.
5. **Finalize**: `FinalizeCombatLog` sets totals. Sets `TargetDestroyed` and `AttackerDestroyed` flags.
6. **Apply damage**: Calls `ApplyRecordedDamage` — this is the single point of HP modification.
7. **Mark acted**: `markSquadAsActed` consumes the squad's attack action (or `BonusAttackActive` if set).
8. **Record**: If `battleRecorder` is enabled, records the engagement.
9. **Remove destroyed squads**: If either squad is destroyed, calls `RemoveSquadFromMap`.
10. **Trigger abilities**: For surviving squads, calls `CheckAndTriggerAbilities` and `DisposeDeadUnitsInSquad`.
11. **Fires `onAttackComplete` hook**.

`canSquadAttackWithReason` returns `(string, bool)` — the reason as a string and whether the attack is valid. It checks in order: action state, attacker position, defender position, faction membership (non-zero), same-faction restriction, and range (distance <= max range of any unit in the attacker squad).

**CombatResult**:

```go
type CombatResult struct {
    Success           bool
    ErrorReason       string
    TargetDestroyed   bool
    AttackerDestroyed bool
    TotalDamage       int
    UnitsKilled       []ecs.EntityID
    DamageByUnit      map[ecs.EntityID]int
    CombatLog         *CombatLog
}
```

### 6.6 Damage Calculation

All damage calculation is in `calculateDamage(attackerID, defenderID ecs.EntityID, modifiers DamageModifiers, manager) (int, *AttackEvent)`:

```go
type DamageModifiers struct {
    HitPenalty       int     // Subtracted from hit threshold (counterattack: -20)
    DamageMultiplier float64 // Multiplied against base damage (counterattack: 0.5)
    IsCounterattack  bool
}
```

**Calculation steps**:

1. Load `attackerAttr` and `defenderAttr` (if either is nil, return 0 damage with `HitTypeMiss`)
2. **Hit roll**: `roll = RandInt(100)`, hit if `roll <= (hitRate - HitPenalty)`. Hit threshold clamped to 0 minimum.
3. **Dodge roll**: `roll = RandInt(100)`, dodged if `roll <= dodgeChance`. Returns `HitTypeDodge`.
4. **Attack type check**: If attacker has `AttackTypeMagic`, uses `GetMagicDamage()` / `GetMagicDefense()`. Otherwise uses `GetPhysicalDamage()` / `GetPhysicalResistance()`.
5. **Crit roll**: `roll = RandInt(100)`, crit if `roll <= critChance`. Critical multiplies base damage by 1.5 and sets `HitTypeCritical`.
6. **Apply `DamageMultiplier`** from modifiers (counterattack path uses 0.5).
7. **Apply resistance**: `totalDamage = baseDamage - resistance`, minimum 1.
8. **Apply cover**: `totalDamage = int(float64(totalDamage) * (1.0 - coverReduction))`, minimum 1.
9. Returns `finalDamage` and an `AttackEvent` with full detail.

`AttackEvent` fields include `AttackerID`, `DefenderID`, `IsCounterattack`, `DefenderHPBefore`, `DefenderHPAfter`, `WasKilled`, `BaseDamage`, `FinalDamage`, `CritMultiplier`, `ResistanceAmount`, `CoverReduction`, and `HitResult`.

**Counterattack penalties** (constants in `tactical/squads/squadcombat.go`):

```go
const (
    counterattackDamageMultiplier = 0.5 // 50% damage
    counterattackHitPenalty       = 20  // -20% hit chance
)
```

**Damage application separation**: Damage is first accumulated in `result.DamageByUnit` (a `map[ecs.EntityID]int`) without modifying any entity HP. `ApplyRecordedDamage` then applies all accumulated damage at once. This ensures that during attack calculation, all units still have their original HP values — preventing a unit killed by an early attack from affecting later targeting or counterattack eligibility calculations.

### 6.7 Targeting Logic

`SelectTargetUnits(attackerID, defenderSquadID, manager)` dispatches to one of four targeting functions based on `TargetRowData.AttackType`:

**MeleeRow** (`selectMeleeRowTargets`):
- Tries rows 0, 1, 2 in order
- Returns all alive units in the first non-empty row
- Maximum 3 targets per attack

**MeleeColumn** (`selectMeleeColumnTargets`):
- Uses attacker's column index
- Tries attacker's column, then (col+1)%3, then (col+2)%3
- Returns all alive units in the first non-empty column (piercing)

**Ranged** (`selectRangedTargets`):
- Targets the same row index as the attacker in the defender's formation
- If that row is empty, falls back to `selectLowestArmorTarget` (lowest armor, furthest row, leftmost column as tiebreakers)

**Magic** (`selectMagicTargets`):
- Uses `TargetRowData.TargetCells` (specific cell coordinates)
- Gets units at each listed cell via `GetUnitIDsAtGridPosition`
- No pierce-through; each cell is independently checked
- Deduplicates via a `seen` map

Only alive units are returned (`getAliveUnitAttributes` check).

### 6.8 Counterattack System

The defender counterattacks automatically if `WouldSquadSurvive` returns true after main attack damage prediction. There is no player choice involved.

Counterattackers must:
- Be alive
- Be within attack range of the attacker squad (range >= distance)
- Not be killed by the incoming main attack (checked unit-by-unit using `result.DamageByUnit`)

`CanAct` is NOT checked for counterattacks — counterattacks are free actions that do not consume the squad's action state.

Counterattack damage uses `ProcessCounterattackOnTargets` with `DamageModifiers{HitPenalty: 20, DamageMultiplier: 0.5, IsCounterattack: true}`.

Counterattack targeting uses the same `SelectTargetUnits` logic as normal attacks, with the roles reversed (defender becomes attacker, attacker becomes target squad).

### 6.9 Victory Conditions

`CombatService.CheckVictoryCondition()` returns a `VictoryCheckResult`:

```go
type VictoryCheckResult struct {
    BattleOver       bool
    VictorFaction    ecs.EntityID
    VictorName       string
    IsPlayerVictory  bool
    DefeatedFactions []ecs.EntityID
    RoundsCompleted  int
}
```

A battle is over when at most one faction has active (non-destroyed) squads. "Active" means the squad is not destroyed per `IsSquadDestroyed` (all units dead or no units).

`GetActiveSquadsForFaction(factionID, manager)` filters the full squad list to exclude destroyed squads.

### 6.10 Combat Query Cache

`tactical/combat/combatqueriescache.go` provides two `ecs.View`-backed caches:

```go
type CombatQueryCache struct {
    ActionStateView *ecs.View // All ActionStateTag entities
    FactionView     *ecs.View // All FactionTag entities
}

func (c *CombatQueryCache) FindActionStateEntity(squadID ecs.EntityID) *ecs.Entity
func (c *CombatQueryCache) FindActionStateBySquadID(squadID ecs.EntityID) *ActionStateData
func (c *CombatQueryCache) FindFactionByID(factionID ecs.EntityID) *ecs.Entity
func (c *CombatQueryCache) FindFactionDataByID(factionID ecs.EntityID) *FactionData
```

Views are automatically maintained by the ECS library when components are added or removed.

---

## 7. Combat Service

`tactical/combatservices/combat_service.go` is the facade that the GUI uses as its single entry point to all combat logic.

### 7.1 CombatService Initialization

```go
type CombatService struct {
    EntityManager   *common.EntityManager
    TurnManager     *combat.TurnManager
    FactionManager  *combat.CombatFactionManager
    MovementSystem  *combat.CombatMovementSystem
    CombatCache     *combat.CombatQueryCache
    CombatActSystem *combat.CombatActionSystem
    BattleRecorder  *battlelog.BattleRecorder
    ThreatManager   *behavior.FactionThreatLevelManager
    LayerEvaluators map[ecs.EntityID]*behavior.CompositeThreatEvaluator
    aiController    *ai.AIController        // lazy-initialized
    chargeTracker   *gear.ArtifactChargeTracker
    // callback slices (private)
}

func NewCombatService(manager *common.EntityManager) *CombatService
```

`NewCombatService` creates all subsystems, wires the `BattleRecorder` into `CombatActSystem`, and sets up internal hook forwarding so that all registered callbacks fire on the relevant events. Artifact behavior dispatch is set up via `setupBehaviorDispatch`.

**InitializeCombat** (`InitializeCombat(factionIDs []ecs.EntityID) error`):

1. Resets `chargeTracker` for the new battle
2. Identifies the player faction
3. Assigns unassigned deployed squads (squads with positions but no `FactionMembershipComponent`) to the player faction
4. Applies minor artifact stat effects to all factions via `gear.ApplyArtifactStatEffects`
5. Delegates to `TurnManager.InitializeCombat`

**GetAIController** returns the `AIController`, creating it on first call (lazy initialization).

### 7.2 Combat Lifecycle Callbacks

Four callback types allow the GUI and artifact systems to react to combat events:

```go
type OnAttackCompleteFunc func(attackerID, defenderID ecs.EntityID, result *squads.CombatResult)
type OnMoveCompleteFunc func(squadID ecs.EntityID)
type OnTurnEndFunc func(round int)
type PostResetHookFunc func(factionID ecs.EntityID, squadIDs []ecs.EntityID)

func (cs *CombatService) RegisterOnAttackComplete(fn OnAttackCompleteFunc)
func (cs *CombatService) RegisterOnMoveComplete(fn OnMoveCompleteFunc)
func (cs *CombatService) RegisterOnTurnEnd(fn OnTurnEndFunc)
func (cs *CombatService) RegisterPostResetHook(fn PostResetHookFunc)
func (cs *CombatService) ClearCallbacks()
```

The GUI registers cache invalidation callbacks (`MarkSquadDirty`, `InvalidateSquad`, `MarkAllSquadsDirty`) on `OnAttackComplete` and `OnMoveComplete`. Artifact behaviors are dispatched through `PostResetHook`, `OnAttackComplete`, and `OnTurnEnd` via `setupBehaviorDispatch`.

### 7.3 Combat Cleanup

**CleanupCombat** (`CleanupCombat(enemySquadIDs []ecs.EntityID)`):

1. Clears all registered callbacks (GUI state is being torn down)
2. Removes all active effects from all unit entities via `cleanupEffects`
3. Removes player squads from the map, resets their `IsDeployed` flag, and removes their `FactionMembershipComponent` via `resetPlayerSquadsToOverworld`
4. Disposes all faction entities (`FactionTag`)
5. Disposes all action state entities (`ActionStateTag`)
6. Disposes all turn state entities (`TurnStateTag`)
7. Disposes enemy squad entities and all their unit entities

The cleanup distinguishes player squads (retained in the ECS world for reuse in future battles) from enemy squads (fully disposed).

---

## 8. Effects System

`tactical/effects/` implements temporary stat modifiers applied by abilities, spells, and items.

```go
// tactical/effects/components.go

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

type EffectSource int

const (
    SourceSpell   EffectSource = iota
    SourceAbility
    SourceItem
)

type ActiveEffect struct {
    Name           string
    Source         EffectSource
    Stat           StatType
    Modifier       int  // positive = buff, negative = debuff
    RemainingTurns int  // -1 = permanent, 0 = expired
}

type ActiveEffectsData struct {
    Effects []ActiveEffect
}

var ActiveEffectsComponent *ecs.Component
var ActiveEffectsTag       ecs.Tag
```

Key functions in `tactical/effects/system.go`:

```go
func ApplyEffectToUnits(unitIDs []ecs.EntityID, effect ActiveEffect, manager *common.EntityManager)
func TickEffectsForUnits(unitIDs []ecs.EntityID, manager *common.EntityManager)
func HasActiveEffects(unitID ecs.EntityID, manager *common.EntityManager) bool
func RemoveAllEffects(unitID ecs.EntityID, manager *common.EntityManager)
```

`TickEffectsForUnits` decrements `RemainingTurns` for each effect and removes any with `RemainingTurns <= 0`. Called at the start of each faction's turn by `TurnManager.ResetSquadActions`.

`RemoveAllEffects` is called during `CombatService.cleanupEffects` to strip all buffs/debuffs before returning to the overworld.

`ParseStatType(stat string) StatType` converts JSON stat name strings ("strength", "dexterity", etc.) to the `StatType` enum.

---

## 9. AI System

The AI system uses a layered threat map architecture to drive tactical decision-making for computer-controlled factions. The `AIController` (`mind/ai/`) orchestrates each AI faction's turn by evaluating and scoring possible move and attack actions for each squad, then executing the highest-scoring action.

For complete documentation of the AI system, see:

- **[AI Algorithm Architecture](AI_ALGORITHM_ARCHITECTURE.md)** -- AIController, ActionEvaluator, power evaluation, action scoring formulas, configuration system, and extension points
- **[Behavior & Threat Layers](BEHAVIOR_THREAT_LAYERS.md)** -- CombatThreatLayer, SupportValueLayer, PositionalRiskLayer, CompositeThreatEvaluator, FactionThreatLevelManager, and difficulty scaling

### How AI Integrates with Combat

The AI goes through the same `CombatActionSystem` and `CombatMovementSystem` as the player, ensuring identical rules apply to both. Key integration points:

- `CombatService.GetAIController()` lazily creates the `AIController` on first call
- `CombatTurnFlow.executeAITurnIfNeeded()` calls `AIController.DecideFactionTurn()` when the current faction is AI-controlled
- Attacks are queued in `AIController.attackQueue` for GUI animation playback after the AI turn completes
- Threat layers are updated once per faction turn, then marked dirty after each action

---

## 10. Battle Log System

`tactical/combat/battlelog/` provides optional JSON export of combat engagements for post-battle analysis.

**BattleRecorder** (`battle_recorder.go`):

```go
type BattleRecorder struct {
    enabled      bool
    battleID     string
    startTime    time.Time
    engagements  []EngagementRecord
    nextIndex    int
    currentRound int
}

func (br *BattleRecorder) SetEnabled(enabled bool)
func (br *BattleRecorder) Start()
func (br *BattleRecorder) SetCurrentRound(round int)
func (br *BattleRecorder) RecordEngagement(log *squads.CombatLog)
func (br *BattleRecorder) Finalize(victor *VictoryInfo) *BattleRecord
func (br *BattleRecorder) Clear()
```

Recording is controlled by `config.ENABLE_COMBAT_LOG_EXPORT`. When enabled, `CombatMode.Enter` calls `Start()` and enables the recorder. `CombatMode.Exit` calls `Finalize()` and exports to JSON via `battlelog.ExportBattleJSON(record, config.COMBAT_LOG_EXPORT_DIR)`.

**BattleRecord** contains:
- `BattleID` (timestamp-based, millisecond precision)
- `StartTime`, `EndTime`
- `FinalRound`
- `VictorFactionID`, `VictorName`
- `Engagements []EngagementRecord`

**EngagementRecord** wraps `*squads.CombatLog` with `Index`, `Round`, and a generated `*EngagementSummary`.

**EngagementSummary** contains `UnitActionSummary` per unit for both squads, aggregating attacks, hits, misses, dodges, criticals, total damage, and units killed.

---

## 11. Gear and Artifact System

Artifacts are equippable items that augment squad capabilities during combat. Minor artifacts apply passive stat modifiers at battle start; major artifacts carry active behaviors that fire in response to combat events or player input.

For complete documentation, see **[Artifact System Architecture](ARTIFACT_SYSTEM.md)**.

### Combat Integration Summary

Artifacts integrate with the combat system at these points:

- **Battle start** (`CombatService.InitializeCombat`): `gear.ApplyArtifactStatEffects` applies all equipped artifact stat modifiers to units as permanent `ActiveEffect` entries
- **Post-reset hook**: Major artifact behaviors fire `OnPostReset` after each faction's action states are reset (start of turn)
- **Attack complete hook**: Major artifact behaviors fire `OnAttackComplete` after every attack resolves (e.g., Twin Strike Banner grants bonus attacks)
- **Turn end hook**: `ArtifactChargeTracker.RefreshRoundCharges` is called, then behaviors fire `OnTurnEnd`
- **Battle end** (`CombatService.CleanupCombat`): All artifact-applied effects are removed; equipment assignments persist across battles

---

## 12. GUI Integration

For complete documentation of all GUI modes, panels, and input handling, see **[GUI Documentation](GUI_DOCUMENTATION.md)**.

### Combat-Relevant GUI Modes

The following modes interact with the squad and combat systems:

| Mode | Package | Purpose |
|---|---|---|
| `CombatMode` | `gui/guicombat/` | Primary combat UI; owns `CombatService`, `CombatTurnFlow`, action/input handlers, and visualization |
| `SquadEditorMode` | `gui/guisquads/` | 3x3 grid-based squad management; add/remove/swap units, manage roster |
| `SquadDeploymentMode` | `gui/guisquads/` | Click-to-map squad placement before combat |
| `UnitPurchaseMode` | `gui/guisquads/` | Unit shop using `UnitPurchaseService` via the command pattern |

### CombatTurnFlow

`gui/guicombat/combat_turn_flow.go` manages the turn lifecycle from the GUI perspective. It bridges user input to `CombatService`:

1. **HandleEndTurn**: Clears move history, calls `TurnManager.EndTurn()`, checks victory, then calls `executeAITurnIfNeeded()`
2. **executeAITurnIfNeeded**: Detects AI faction, calls `AIController.DecideFactionTurn()`, chains attack animations, then recursively advances through consecutive AI factions
3. **HandleFlee**: Creates a defeat result and transitions to exploration mode
4. **CheckAndHandleVictory**: Calls `combatService.CheckVictoryCondition()` and transitions out of combat if the battle is over

---

## 13. Data Flow: A Complete Combat Round

This section traces the execution path of a full player combat turn followed by an AI turn.

**1. Player ends turn** (`CombatMode.handleEndTurnClick`):
- `CombatTurnFlow.HandleEndTurn()` is called
- `actionHandler.ClearMoveHistory()` resets the movement undo stack
- `CombatService.TurnManager.EndTurn()` advances `CurrentTurnIndex`, resets action states for the new faction (calls `TickEffectsForUnits`, `CheckAndTriggerAbilities`, fires `postResetHook`), and fires `onTurnEnd` (which triggers artifact `OnTurnEnd` hooks and refreshes round charges)

**2. Victory check**:
- `CombatTurnFlow.CheckAndHandleVictory()` calls `CombatService.CheckVictoryCondition()`
- If battle continues, UI components are refreshed

**3. AI turn executes**:
- `CombatTurnFlow.executeAITurnIfNeeded()` detects non-player faction
- `AIController.DecideFactionTurn(factionID)` updates threat layers, then loops over each squad:
  - `ActionEvaluator.EvaluateAllActions()` scores moves and attacks
  - Best action is selected and executed (move via `CombatMovementSystem`, attack via `CombatActionSystem`)
  - Attack results fire `onAttackComplete` hooks (cache invalidation, artifact behaviors)
  - Attacks are queued in `attackQueue` for animation

**4. AI attack animations**:
- `playAIAttackAnimations` chains mode transitions to `combat_animation` for each queued attack
- After all animations, `advanceAfterAITurn` ends the AI's turn

**5. Player's turn begins**:
- `TurnManager.EndTurn()` is called for the AI faction (advancing to the player faction)
- `ResetSquadActions` initializes movement points and triggers abilities for player squads
- UI shows the player faction's turn indicator

**6. Player moves a squad** (`CombatMode.handleMoveClick`):
- `CombatActionHandler.ToggleMoveMode()` enters move mode
- Player clicks a tile; input handler calls `CombatMovementSystem.MoveSquad(squadID, targetPos)`
- `MoveSquad` validates movement remaining, calls `manager.MoveSquadAndMembers` (atomically updates position component and `GlobalPositionSystem`), decrements movement, fires `onMoveComplete`

**7. Player attacks** (`CombatMode.handleAttackClick`):
- `CombatActionHandler.ToggleAttackMode()` enters attack mode
- Player clicks an enemy squad; input handler calls `CombatActionSystem.ExecuteAttackAction(attackerID, defenderID)`
- Damage is calculated per-unit, accumulated, counterattack is resolved, then all damage is applied atomically
- Result fires `onAttackComplete` hooks (cache invalidation)
- Combat animation mode is entered for visual feedback

---

## 14. Package Dependency Map

```
gui/guicombat          --> combatservices, squads, combat, behavior, encounter, gear
gui/guisquads          --> squadcommands, squadservices, squads, commander, gear
tactical/combatservices --> combat, squads, effects, gear, mind/ai, mind/behavior, mind/encounter
tactical/combat        --> squads, effects, common, coords
tactical/squads        --> common, templates, coords, effects, rendering
tactical/effects       --> (ecs only, no game logic imports)
tactical/squadcommands --> squads, common, combat
tactical/squadservices --> squads, common
mind/ai                --> combat, squads, behavior, common
mind/behavior          --> combat, squads, evaluation, templates
mind/encounter         --> combat, squads, common
gear                   --> combat, squads, common
```

The layering is strict: `squads` does not import `combat`. Combat components (`ActionStateData`, `CombatFactionData`) are separate from squad components (`SquadData`, `GridPositionData`). The `combatservices` package is the only place all subsystems are wired together.

---

*Key source files referenced in this document:*

- `C:/Users/Afromullet/Desktop/TinkerRogue/tactical/squads/squadcomponents.go`
- `C:/Users/Afromullet/Desktop/TinkerRogue/tactical/squads/squadcreation.go`
- `C:/Users/Afromullet/Desktop/TinkerRogue/tactical/squads/squadqueries.go`
- `C:/Users/Afromullet/Desktop/TinkerRogue/tactical/squads/squadcombat.go`
- `C:/Users/Afromullet/Desktop/TinkerRogue/tactical/squads/squadabilities.go`
- `C:/Users/Afromullet/Desktop/TinkerRogue/tactical/squads/units.go`
- `C:/Users/Afromullet/Desktop/TinkerRogue/tactical/squads/squadcache.go`
- `C:/Users/Afromullet/Desktop/TinkerRogue/tactical/combat/combatcomponents.go`
- `C:/Users/Afromullet/Desktop/TinkerRogue/tactical/combat/turnmanager.go`
- `C:/Users/Afromullet/Desktop/TinkerRogue/tactical/combat/combatactionsystem.go`
- `C:/Users/Afromullet/Desktop/TinkerRogue/tactical/combat/combatmovementsystem.go`
- `C:/Users/Afromullet/Desktop/TinkerRogue/tactical/combat/combatqueries.go`
- `C:/Users/Afromullet/Desktop/TinkerRogue/tactical/combat/combatfactionmanager.go`
- `C:/Users/Afromullet/Desktop/TinkerRogue/tactical/combat/combatqueriescache.go`
- `C:/Users/Afromullet/Desktop/TinkerRogue/tactical/combatservices/combat_service.go`
- `C:/Users/Afromullet/Desktop/TinkerRogue/tactical/combatservices/combat_events.go`
- `C:/Users/Afromullet/Desktop/TinkerRogue/tactical/squadcommands/command.go`
- `C:/Users/Afromullet/Desktop/TinkerRogue/tactical/squadcommands/command_executor.go`
- `C:/Users/Afromullet/Desktop/TinkerRogue/tactical/squadservices/unit_purchase_service.go`
- `C:/Users/Afromullet/Desktop/TinkerRogue/tactical/squadservices/squad_deployment_service.go`
- `C:/Users/Afromullet/Desktop/TinkerRogue/tactical/effects/components.go`
- `C:/Users/Afromullet/Desktop/TinkerRogue/tactical/combat/battlelog/battle_recorder.go`
- `C:/Users/Afromullet/Desktop/TinkerRogue/mind/ai/ai_controller.go`
- `C:/Users/Afromullet/Desktop/TinkerRogue/mind/ai/action_evaluator.go`
- `C:/Users/Afromullet/Desktop/TinkerRogue/mind/behavior/threat_composite.go`
- `C:/Users/Afromullet/Desktop/TinkerRogue/mind/behavior/threat_combat.go`
- `C:/Users/Afromullet/Desktop/TinkerRogue/mind/behavior/threat_constants.go`
- `C:/Users/Afromullet/Desktop/TinkerRogue/gear/components.go`
- `C:/Users/Afromullet/Desktop/TinkerRogue/gear/artifactbehavior.go`
- `C:/Users/Afromullet/Desktop/TinkerRogue/gui/guicombat/combatmode.go`
- `C:/Users/Afromullet/Desktop/TinkerRogue/gui/guicombat/combat_turn_flow.go`
- `C:/Users/Afromullet/Desktop/TinkerRogue/gui/guisquads/squadeditormode.go`
- `C:/Users/Afromullet/Desktop/TinkerRogue/gui/guisquads/squaddeploymentmode.go`
