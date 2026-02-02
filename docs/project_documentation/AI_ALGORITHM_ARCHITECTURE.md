# TinkerRogue: AI Algorithms & System Architecture

**Last Updated:** 2026-02-01

This document provides a comprehensive architectural and implementation overview of the AI, threat assessment, encounter generation, and overworld simulation systems driving TinkerRogue's strategic and tactical gameplay.

---

## Table of Contents

1. [Overview](#overview)
2. [Directory Structure](#directory-structure)
3. [Overworld Package](#overworld-package)
4. [Encounter Package](#encounter-package)
5. [Behavior Package (Threat Assessment)](#behavior-package-threat-assessment)
6. [AI Package (Decision-Making)](#ai-package-decision-making)
7. [Power Evaluation System](#power-evaluation-system)
8. [System Integration](#system-integration)
9. [Data Flow Across Systems](#data-flow-across-systems)
10. [Configuration](#configuration)
11. [Debug Tools](#debug-tools)

---

## Overview

TinkerRogue's game loop operates across two primary layers:

1. **Strategic Layer (Overworld)**: Turn-based simulation of faction expansion, threat evolution, and resource management
2. **Tactical Layer (Combat)**: Real-time squad combat with AI-controlled enemies

These layers are bridged by a power-based encounter generation system that ensures combat challenges scale appropriately to player strength. The AI system uses multi-layered threat evaluation to make intelligent tactical decisions during combat.

### Key Design Principles

- **Composable Threat Layers**: Multiple independent layers calculate different aspects of tactical threat
- **Role-Based Weighting**: Different unit roles (Tank, DPS, Support) weight threats differently
- **Power-Based Balancing**: Unified power system ensures consistent encounter difficulty
- **Data-Driven Configuration**: JSON templates allow tuning without code changes
- **Lazy Evaluation**: Threat layers cache efficiently with dirty flag invalidation
- **Query-Based Architecture**: ECS queries drive all game logic

---

## Directory Structure

```
mind/
├── ai/                    # Decision-making and action execution
│   ├── ai_controller.go   # Main AI orchestration
│   └── action_evaluator.go # Action generation and scoring
│
├── behavior/              # Threat assessment and spatial awareness
│   ├── threat_layers.go      # ThreatLayer interface
│   ├── threat_composite.go   # Multi-layer combination
│   ├── threat_melee.go       # Melee threat calculation
│   ├── threat_ranged.go      # Ranged threat calculation
│   ├── threat_support.go     # Support value calculation
│   ├── threat_positional.go  # Positional risk calculation
│   ├── threat_painting.go    # Threat value painting algorithms
│   ├── threat_constants.go   # Configuration constants
│   ├── threat_queries.go     # Combat data queries
│   ├── dangerlevel.go        # Faction threat manager
│   └── dangervisualizer.go   # Debug visualization
│
├── evaluation/            # Power calculations and configuration
│   ├── power.go           # Power calculation algorithms
│   ├── power_config.go    # Power configuration and profiles
│   ├── roles.go           # Role multipliers and abilities
│   └── cache.go           # Dirty cache for lazy evaluation
│
└── encounter/             # Encounter generation and lifecycle
    ├── encounter_service.go   # Encounter coordination
    ├── encounter_generator.go # Enemy squad generation
    └── encounter_setup.go     # Combat initialization

world/overworld/           # Strategic layer simulation
├── components.go          # Threat node, faction, victory data
├── threat_system.go       # Threat evolution logic
├── faction_system.go      # Faction AI behavior
├── victory_system.go      # Win/loss condition checking
├── encounter_translation.go # Overworld → Combat bridge
└── overworld_config.go    # Strategic layer configuration
```

---

## Overworld Package

**Location:** `world/overworld/`

### Purpose

The overworld package simulates the **strategic turn-based world state**. It manages enemy threat nodes that grow and spread across the map, faction AI that expands territory and raids the player, and victory/defeat conditions based on survival or elimination goals.

### Core Responsibilities

- **Threat Evolution**: Enemy spawning points (threat nodes) increase in intensity over time
- **Faction AI Simulation**: Enemy factions expand territory, fortify positions, raid player squads, and retreat when weak
- **Global Turn System**: Orchestrates sequential updates to threats, factions, and victory checks
- **Influence Calculation**: Computes threat "pressure" across the map using spatial distance falloff
- **Victory/Defeat Evaluation**: Checks win/loss conditions based on threat levels, squad status, and survival time
- **Combat Translation**: Converts threat nodes into tactical combat encounters
- **Event Logging**: Records all significant strategic events for history and replay

### Key Algorithms

#### Threat Evolution Cycle

**Inputs:**
- Current tick number
- Threat node data (type, intensity, position, growth progress)
- Player squad proximity

**Process:**
1. Each threat node accumulates growth progress per tick
2. Growth rate is reduced if player squads are nearby (containment mechanic)
3. When growth progress reaches 1.0, the threat evolves:
   - Intensity increases (capped by threat type)
   - Influence radius expands
   - Type-specific effects trigger (Necromancer spawns children, Corruption spreads)

**Outputs:**
- Updated threat intensity and influence radius
- New child threat nodes (for certain types)
- Corruption spread to adjacent tiles

**Key Functions:** `UpdateThreatNodes()`, `EvolveThreatNode()`, `ExecuteThreatEvolutionEffect()`

**Utility Functions:** `GetCardinalNeighbors()`, `IsThreatAtPosition()` (shared by corruption spread and child spawning)

#### Faction AI Decision Making

**Inputs:**
- Faction strength and territory size
- Border positions and threat locations
- Intent timer and current disposition

**Process:**
1. Each faction evaluates four strategic intents when its timer expires:
   - **Expand**: Claim adjacent unoccupied territory
   - **Fortify**: Increase strength and consolidate position
   - **Raid**: Spawn high-intensity threats at player borders
   - **Retreat**: Abandon outermost territory when weak
2. Each intent is scored based on faction state
3. The highest-scoring intent is executed

**Outputs:**
- Territory tile ownership changes
- Faction strength modifications
- New threat nodes spawned
- Abandoned territory tiles

**Key Functions:** `EvaluateFactionIntent()`, `ExecuteFactionIntent()`

**Utility Functions:** `GetCardinalNeighbors()`, `GetRandomTileFromSlice()` (shared by expand, fortify, and raid actions)

#### Influence Calculation (Cached)

**Inputs:**
- All active threat nodes (position, intensity, type)
- Threat type parameters (base radius, intensity multiplier)

**Process:**
1. Uses dirty-flag caching to avoid unnecessary recalculation
2. When cache is invalidated (threat spawned/evolved/destroyed):
   - For each threat node:
     - Calculate effective radius: `baseRadius + intensity`
     - For each tile within square radius:
       - Calculate Manhattan distance
       - Apply linear falloff: `1.0 - (distance / (radius + 1))`
       - Accumulate influence (multiple threats overlap)
3. Cache results per tile

**Outputs:**
- Per-tile influence values (0.0 to unbounded, typically 0-50)
- Dirty flag cleared

**Key Functions:** `RebuildCache()`, `GetInfluenceAt()`, `MarkDirty()`

#### Victory Evaluation

**Inputs:**
- Current tick counter
- Total map influence
- Threat counts by tier
- Player squad status

**Process:**
Checks conditions in priority order:
1. **Defeat Conditions:**
   - Total influence exceeds 100.0 (overwhelming corruption)
   - High-intensity threats exceed threshold (configurable via `HighIntensityThreshold` and `MaxHighIntensityThreats` constants)
   - All player squads destroyed
2. **Victory Conditions:**
   - Survival goal reached (ticks >= target survival time)
   - All threats eliminated
   - Target faction completely destroyed

**Constants:**
```go
HighIntensityThreshold  = 8   // Intensity level considered "high"
MaxHighIntensityThreats = 10  // Maximum allowed before defeat
```

**Outputs:**
- `IsGameOver` flag set
- `VictoryAchieved` or `DefeatTriggered` status
- Final event log export

**Key Functions:** `CheckVictoryCondition()`, `CountHighIntensityThreats()`

### Shared Utility Functions

The overworld package provides reusable utilities to reduce code duplication:

| Function | Purpose | Used By |
|----------|---------|---------|
| `GetCardinalNeighbors(pos)` | Returns 4 adjacent positions (N, S, E, W) | Corruption spread, faction expansion |
| `GetRandomTileFromSlice(tiles)` | Returns random tile from slice, nil-safe | Faction territory operations |
| `IsThreatAtPosition(manager, pos)` | Checks if threat node exists at position | Child spawning, corruption spread |
| `CountHighIntensityThreats(manager, threshold)` | Counts threats at or above intensity | Victory/defeat evaluation |

### Data-Driven Item Drops

Item rewards are configured via `itemDropTables` map:

```go
var itemDropTables = map[ThreatType]ItemDropTable{
    ThreatNecromancer: {
        Basic:    []string{"Dark Essence", "Necromantic Scroll", ...},
        HighTier: []string{"Lich Phylactery", "Staff of Undeath"},
    },
    // ... other threat types
}
```

High-tier items are only available when `intensity >= HighTierIntensityThreshold` (default: 7).

### Integration Points

- **With Combat System**: Threat nodes remain on map during combat; combat resolution calls threat destruction
- **With Encounter System**: Translates threat parameters (type, intensity) into combat encounter configuration
- **With Squad System**: Checks deployed squad status for proximity calculations and defeat conditions
- **With Event System**: Records all strategic events to global recorder for JSON export

---

## Encounter Package

**Location:** `mind/encounter/`

### Purpose

The encounter package implements **power-based encounter balancing**. It calculates the combat strength of player squads and generates appropriately scaled enemy forces to create balanced tactical challenges.

### Core Responsibilities

- **Power Calculation**: Evaluate unit, squad, and roster strength using weighted attribute formulas
- **Encounter Generation**: Create balanced enemy squad compositions matching target power budgets
- **Difficulty Scaling**: Map encounter difficulty levels to power multipliers and squad counts
- **Enemy Composition**: Select unit types and squad counts based on encounter archetypes

### Key Algorithms

#### Unit Power Calculation

**Inputs:**
- Unit attributes (strength, dexterity, health, resistances)
- Equipment (weapon damage, armor)
- Role type and leader status
- Current health percentage

**Process:**
Calculates three power components (configurable weights):

1. **Offensive Power (60% default)**:
   - **Damage Component (60%)**: `(weaponDmg + strength/2) * roleMultiplier`
   - **Accuracy Component (40%)**: `effectiveDamage * hitRate * critMultiplier`

2. **Defensive Power (40% default)**:
   - **Health Component (50%)**: `(currentHP / maxHP) * healthMultiplier`
   - **Resistance Component (30%)**: `(physicalResist + magicResist) / 2`
   - **Avoidance Component (20%)**: `dodgeChance * dodgeScaling`

3. **Utility Power (20% default)**:
   - **Role Component (50%)**: `roleMultiplier * 10.0`
   - **Ability Component (30%)**: Leader abilities (Rally=15, Heal=20, BattleCry=12)
   - **Cover Component (20%)**: `coverValue * coverScaling * beneficiaryMultiplier`

**Outputs:**
- Single numeric power value (typically 10-200 per unit)

**Key Functions:** `CalculateUnitPower()`, `calculateHealthMultiplier()`, `calculateRoleValue()`

#### Squad Power Calculation

**Inputs:**
- All units in squad (individual unit powers)
- Squad morale (0-100)
- Attack type diversity (melee/ranged/magic composition)
- Squad health percentage

**Process:**
1. Sum all unit power values
2. Apply morale multiplier: `0.002 * morale` (0 morale = 0.8x, 100 morale = 1.2x)
3. Apply composition bonus based on attack type diversity:
   - 1 type: 0.8x (mono-type penalty)
   - 2 types: 1.1x (dual-type bonus)
   - 3+ types: 1.2x (optimal diversity)
4. Apply health multiplier: `(currentHP / maxHP) * 2.0` (minimum 10%)

**Outputs:**
- Single squad power value (typically 50-400 per squad)

**Key Functions:** `CalculateSquadPower()`

#### Roster Power Calculation

**Inputs:**
- All player squads (deployed and reserve)
- Deployment status per squad

**Process:**
1. Calculate power for each squad
2. Apply weighting based on deployment status:
   - Deployed squads: 1.0x weight
   - Reserve squads: 0.3x weight
3. Sum all weighted squad powers

**Outputs:**
- Total roster power value

**Key Functions:** `CalculateRosterPower()`, `CalculateDeployedSquadsPower()`

#### Encounter Generation

**Inputs:**
- Average deployed player squad power
- Difficulty level (1-5)
- Encounter type (Goblin, Bandit, Beast, Orc, etc.)
- Available unit templates

**Process:**
1. **Calculate Target Power**:
   - `targetPower = avgPlayerSquadPower * difficultyMultiplier`
   - Difficulty multipliers: 0.7x (easy) to 1.5x (boss)

2. **Determine Squad Count**:
   - Difficulty level maps to min/max squad range (2-7 squads)
   - Random count selected within range

3. **Generate Each Enemy Squad**:
   - Get squad type preferences from encounter type (melee/ranged/magic ratios)
   - Filter unit pool by squad type
   - Iteratively add units until reaching 95% of target power
   - Ensure minimum 3 units per squad
   - Create squad with 2x3 grid formation

4. **Position Squads**:
   - Distribute enemy squads circularly around player position
   - Player squads positioned in defensive arc

**Outputs:**
- Complete set of enemy squad entities with:
  - Units positioned in formations
  - Faction assignments
  - Action state initialization

**Key Functions:** `SetupBalancedEncounter()`, `generateEnemySquadsByPower()`, `createSquadForPowerBudget()`

### Power Calculation Constants

```
Offensive Weights: 60% total (36% damage, 24% accuracy)
Defensive Weights: 40% total (20% health, 12% resist, 8% dodge)
Utility Weights: 20% total (10% role, 6% ability, 4% cover)

Difficulty Table:
Level 1: 0.7x power, 2-3 squads (easy)
Level 2: 0.9x power, 3-4 squads (moderate)
Level 3: 1.0x power, 3-5 squads (balanced)
Level 4: 1.2x power, 4-6 squads (hard)
Level 5: 1.5x power, 5-7 squads (boss)
```

### Integration Points

- **With Overworld System**: Receives threat parameters (type, intensity) and translates to encounter configuration
- **With Combat System**: Called during combat setup to spawn enemy squads and create factions
- **With Squad System**: Uses squad creation functions and unit templates
- **With Evaluation System**: Shares role multipliers and leader bonus constants

---

## Behavior Package (Threat Assessment)

**Location:** `mind/behavior/`

### Purpose

The behavior package provides **AI threat assessment and tactical positioning evaluation**. It calculates danger levels from enemy squads and evaluates the tactical value of map positions for AI decision-making.

### Core Responsibilities

- **Threat Quantification**: Calculate expected damage from enemy squads at various ranges
- **Threat Zone Mapping**: Paint threat surfaces across the battlefield using distance falloff
- **Positional Risk Assessment**: Evaluate flanking exposure, isolation risk, and retreat options
- **Support Value Calculation**: Identify optimal positioning for healing and buffing
- **Role-Based Threat Weighting**: Provide position scores tailored to unit roles (Tank/DPS/Support)

### Layered Threat Model

The threat system uses composable layers where each calculates one aspect of tactical threat:

```
CompositeThreatEvaluator
├── MeleeThreatLayer       # Melee attack danger
├── RangedThreatLayer      # Ranged/magic danger
├── SupportValueLayer      # Healing/buff opportunities
└── PositionalRiskLayer    # Flanking, isolation, pressure
```

### ThreatLayer Interface

**Location:** `mind/behavior/threat_layers.go`

```go
type ThreatLayer interface {
    Compute()              // Recalculate threat values
    IsValid(round int) bool // Check if current for round
    MarkDirty()            // Force recomputation
}
```

### Melee Threat Layer

**Location:** `threat_melee.go`

Calculates danger from enemy melee squads.

**Formula:**
```
Threat = ExpectedDamageByRange[1] * RoleMultiplier * DistanceFalloff
```

**Process:**
- For each enemy squad:
  - Calculate threat radius: `movementSpeed + meleeRange`
  - Paint threat to map using linear falloff
  - Falloff formula: `1.0 - (distance / (maxRange + 1))`
  - Threat value = `expectedDamage * falloff * roleMultiplier`

**Outputs:**
- Per-tile melee threat values (accumulated from all melee threats)

### Ranged Threat Layer

**Location:** `threat_ranged.go`

Calculates danger from enemy ranged and magic squads.

**Formula:**
```
Threat = ExpectedDamageByRange[maxRange] * RoleModifier
```

**Process:**
- For each enemy squad with ranged units:
  - Calculate threat radius: `maxAttackRange`
  - Paint threat to map with **no distance falloff** (ranged advantage)
  - Full threat at all ranges within max range

**Outputs:**
- Per-tile ranged pressure values

### Support Value Layer

**Location:** `threat_support.go`

Identifies valuable positions for support units.

**Heal Priority Calculation:**
```
Priority = 1.0 - SquadHealthPercent
```
(More wounded = higher priority)

**Process:**
- For each wounded ally:
  - Calculate heal priority: `1.0 - currentHealth`
  - Paint support value to map using linear falloff
- Positions near wounded allies have high support value

**Configuration:**
- Heal radius: 3 tiles
- Ally proximity radius: 2 tiles

**Outputs:**
- Per-tile support value (high values indicate healing opportunities)

### Positional Risk Layer

**Location:** `threat_positional.go`

Calculates tactical positioning risks beyond raw damage.

**Four Sub-Metrics:**

| Metric | Description | Range |
|--------|-------------|-------|
| Flanking Risk | Attacked from multiple directions | 0-1 |
| Isolation Risk | Distance from nearest ally | 0-1 |
| Engagement Pressure | Combined melee+ranged threat | 0-1 |
| Retreat Quality | % of low-threat escape routes | 0-1 |

**Process:**
Computes four risk sub-components:

1. **Flanking Risk**:
   - Scan 8 directions from each enemy squad
   - Count unique threat directions per position
   - 1 direction = 0 risk, 2 directions = 0.5 risk, 3+ directions = 1.0 risk

2. **Isolation Risk**:
   - Calculate distance to nearest ally for each position
   - 0-2 tiles = 0 risk, 3-5 tiles = moderate risk, 6+ tiles = 1.0 risk

3. **Engagement Pressure**:
   - Combined threat: `(meleeThreat + rangedThreat) / 200.0`
   - Normalized to 0-1 range

4. **Retreat Quality**:
   - Count adjacent positions with low threat (< 10.0)
   - Quality = `lowThreatNeighbors / 8`
   - Inverted to penalty: `1.0 - quality`

**Total Risk Formula:**
```
totalRisk = flanking*0.4 + isolation*0.3 + pressure*0.2 + (1-retreat)*0.1
```

**Positional Risk Weights:**
- Flanking: 0.4
- Isolation: 0.3
- Pressure: 0.2
- Retreat: 0.1

**Outputs:**
- Per-tile positional risk values

### Composite Threat Evaluation

**Location:** `threat_composite.go`

Combines all layers with role-based weighting.

**Inputs:**
- All four threat layers (melee, ranged, support, positional)
- Squad role (Tank, DPS, Support)
- Target position to evaluate

**Process:**
1. Fetch threat values from each layer at target position
2. Apply role-specific weights:
   - **Tank**: MeleeWeight = -0.5 (SEEK danger), RangedWeight = 0.3, PositionalWeight = 0.5
   - **DPS**: MeleeWeight = 0.7 (AVOID danger), RangedWeight = 0.5, PositionalWeight = 0.6
   - **Support**: MeleeWeight = 1.0 (AVOID danger), SupportWeight = -1.0 (SEEK wounded allies)
3. Calculate weighted sum:
   ```
   totalThreat = melee*MeleeWeight + ranged*RangedWeight +
                 supportValue*SupportWeight + positional*PositionalWeight
   ```

**Combination Formula:**
```
TotalThreat =
    (meleeThreat * weights.MeleeWeight) +
    (rangedThreat * weights.RangedWeight) +
    (supportValue * weights.SupportWeight) +
    (positionalRisk * weights.PositionalWeight)
```

Lower score = better position (negative weights invert behavior).

**Outputs:**
- Single threat score (lower is better for that role)
- Negative weights indicate desirable positions (tanks seek melee, support seeks wounded)

**Key Functions:** `CompositeThreatEvaluator.GetRoleWeightedThreat()`, `Update()`

### Role Threat Weights

```go
type RoleThreatWeights struct {
    MeleeWeight      float64  // How much melee threat matters
    RangedWeight     float64  // How much ranged threat matters
    SupportWeight    float64  // How much support value matters
    PositionalWeight float64  // How much positional risk matters
}
```

**Default Weights by Role:**

| Role | Melee | Ranged | Support | Positional |
|------|-------|--------|---------|------------|
| Tank | -0.5 (seeks) | 0.3 | 0.0 | 0.5 |
| DPS | 0.7 (avoids) | 0.5 | 0.0 | 0.6 |
| Support | 1.0 (avoids) | 1.0 | -1.0 (seeks wounded) | 0.8 |

### Squad Danger Level Calculation

**Inputs:**
- Squad unit composition (weapon, dexterity, role, attack type, range)
- Target reference stats (strength, armor, dodge, resistance)

**Process:**
Calculates two metrics per range:

1. **DangerByRange** (heuristic):
   ```
   danger = Sum(weaponPower * roleMultiplier * leaderBonus) * compositionBonus
   ```
   - Composition bonus: 0.8x (mono), 1.1x (dual-type), 1.2x (triple-type)

2. **ExpectedDamageByRange** (accurate):
   ```
   For each unit:
     baseDamage = weapon + dexterity/2
     hitRate = calculateHitChance() * (1 - targetDodge)
     critMultiplier = 1.0 + (critChance * 0.5)
     damage = (baseDamage - resistance) * hitRate * critMultiplier * coverMultiplier
   Total = Sum(all unit damage)
   ```

**Outputs:**
- Danger values by range (for fast AI evaluation)
- Expected damage values by range (for realistic threat assessment)

**Key Functions:** `CalculateSquadDangerLevel()`, `FactionThreatLevelManager.UpdateFaction()`

### Threat Painting Mechanism

**Location:** `threat_painting.go`

All threat layers use a shared spatial painting algorithm:

```go
func PaintThreatToMap(
    threatMap map[coords.LogicalPosition]float64,
    center coords.LogicalPosition,
    radius int,
    threatValue float64,
    falloffFunc FalloffFunc,
)
```

**Process:**
1. For each threat source (squad):
   - Define center position and radius
   - For each tile within radius:
     - Calculate distance (Chebyshev for tactical ranges)
     - Apply falloff function (Linear, Quadratic, or NoFalloff)
     - Accumulate threat value: `threatMap[tile] += threatValue * falloff`

**Falloff Functions:**
- **Linear**: `1.0 - (distance / (maxRange + 1))` - Decreases uniformly (melee)
- **NoFalloff**: `1.0` - Constant threat (ranged)
- **Quadratic**: `1.0 - ((distance / (maxRange + 1))^2)` - Faster decrease (proximity effects)

**Key Property:** Threats **accumulate** (use +=), so multiple enemies create overlapping threat zones

**Key Functions:** `PaintThreatToMap()`, `PaintThreatToMapWithTracking()`

### Faction Threat Manager

**Location:** `dangerlevel.go`

Tracks threat across all factions:

```go
type FactionThreatLevelManager struct {
    factions map[ecs.EntityID]*FactionThreatLevel
}
```

**SquadThreatLevel:**
```go
type SquadThreatLevel struct {
    DangerByRange         map[int]float64  // Power rating by range
    ExpectedDamageByRange map[int]float64  // Actual damage by range
    SquadDistances        *SquadDistanceTracker
}
```

**SquadDistanceTracker:**
- Organizes squads by faction
- Groups by distance (AlliesByDistance, EnemiesByDistance)
- Lazy evaluation with caching

### Optimization: Dirty Flag Caching

All threat layers use lazy evaluation:
- `isDirty` flag tracks when recalculation is needed
- Marked dirty when:
  - Squad positions change
  - Squads are created/destroyed
  - Combat round number changes
- Only recalculates on next query if dirty flag is set

### Integration Points

- **With AI System**: Provides threat evaluation data for position scoring
- **With Combat System**: Uses combat queries for squad positions and faction data
- **With Squad System**: Queries squad composition, movement speed, health percentages
- **With Evaluation System**: Uses shared role multipliers and leader bonuses
- **With Graphics System**: Supports debug visualization of threat layers

---

## AI Package (Decision-Making)

**Location:** `mind/ai/`

### Purpose

The AI package implements **autonomous decision-making for computer-controlled factions** during tactical combat. It evaluates all possible actions for each enemy squad and executes the highest-scoring action based on threat assessment and role-specific tactics.

### Core Responsibilities

- **Action Generation**: Identify all legal moves, attacks, and wait actions for a squad
- **Action Scoring**: Evaluate each action using threat data, role preferences, and tactical heuristics
- **Action Execution**: Execute the best action via combat movement and action systems
- **Attack Animation Queueing**: Store attacks for sequential animation playback

### AIController

**Location:** `mind/ai/ai_controller.go`

The main orchestrator for AI decision-making. Controls computer-controlled factions during their turns.

**Key Responsibilities:**
- Updates all threat layers at the start of each AI turn
- Processes squads sequentially, executing actions until exhausted
- Marks threat layers dirty after each action (positions change)
- Coordinates attack animations via attack queue

**Entry Point:**
```go
func (ai *AIController) DecideFactionTurn(factionID ecs.EntityID)
```

### Action Evaluation

**Location:** `mind/ai/action_evaluator.go`

Generates and scores all possible actions for a squad.

**ScoredAction Structure:**
```go
type ScoredAction struct {
    Action      Action    // The action to execute
    Score       float64   // Utility score (higher = better)
    Description string    // Debug description
}
```

**Action Types:**

| Action | Base Score | Description |
|--------|------------|-------------|
| AttackAction | 100.0 | Attack target squad (highest priority) |
| MoveAction | 50.0 | Move to position |
| WaitAction | 0.0 | Skip turn (fallback) |

**ActionContext:**
```go
type ActionContext struct {
    SquadID       ecs.EntityID
    FactionID     ecs.EntityID
    ActionState   *combat.ActionStateData
    ThreatEval    *behavior.CompositeThreatEvaluator
    Manager       *common.EntityManager
    SquadRole     squads.UnitRole
    CurrentPos    coords.LogicalPosition
    SquadHealth   float64
}
```

### Key Algorithms

#### Faction Turn Decision Cycle

**Inputs:**
- Faction ID (which enemy faction is taking its turn)
- Combat state (all squad positions, action points, health)
- Threat layer data (updated before each faction turn)

**Process:**
```
For each alive squad in faction:
    While squad has action points remaining:
        1. Create ActionContext (role, position, health, threat evaluator)
        2. Generate all possible actions:
           - Movement actions to reachable tiles
           - Attack actions against in-range enemies
           - Wait action (fallback)
        3. Score each action:
           - Movement: baseScore - threat + allyProximity + approachBonus
           - Attack: baseScore + woundedPriority + roleThreat + counterBonus
           - Wait: fallback score
        4. Select highest-scoring action
        5. Execute action (move, attack, or wait)
        6. Mark threat layers dirty (positions changed)
        7. Queue attack for animation (if attack action)
```

**Outputs:**
- Updated squad positions
- Updated action point states
- Queued attack actions for animation
- Boolean indicating whether any actions were taken

**Key Functions:** `AIController.DecideFactionTurn()`, `ActionEvaluator.EvaluateAllActions()`

#### Movement Scoring

**Inputs:**
- Squad position and movement speed
- Squad role (Tank, DPS, Support)
- Threat evaluation data for all reachable tiles
- Allied squad positions
- Enemy squad positions and ranges

**Process:**
For each reachable tile (within movement speed, validated as movable):

1. **Calculate Base Score**: 50.0 (reference point)

2. **Subtract Threat Component**:
   - Query `CompositeThreatEvaluator.GetRoleWeightedThreat(squadID, position)`
   - Role-specific threat weights automatically applied
   - Lower threat = higher score

3. **Add Ally Proximity Bonus**:
   - Small bonus for staying near allies (avoid isolation)

4. **Add Approach Bonus** (role-specific):
   - Distance to nearest enemy
   - **Tank**: 15.0x multiplier (intercept role - seeks frontline)
   - **DPS**: 8.0x multiplier (engage role - seeks engagement range)
   - **Support**: -5.0x multiplier (maintain distance - seeks backline)

5. **Add In-Range Bonus**:
   - +20 if position allows attack on enemy
   - +10 if close to attack range

**Output:**
- Score per tile (higher = better position for that role)

**Key Functions:** `scoreMovementPosition()`, `scoreApproachEnemy()`

#### Attack Scoring

**Inputs:**
- Squad position and attack range
- All enemy squads in range
- Enemy health percentages
- Enemy roles and threat levels

**Process:**
For each attackable enemy:

1. **Calculate Base Score**: 100.0 (higher than movement to prefer attacking when possible)

2. **Add Wounded Priority**:
   - Formula: `(1.0 - enemyHealth) * 20.0`
   - Finish off wounded targets (0.5 health = +10, 0.2 health = +16)

3. **Add Role Threat Bonus**:
   - DPS targets: +15
   - Support targets: +10
   - Prioritize high-value threats

4. **Add Counter Bonus** (role matchup):
   - DPS attacking Support: +10
   - Tank attacking DPS: +10
   - Rock-paper-scissors preference

**Output:**
- Score per enemy target (higher = higher priority)

**Key Functions:** `scoreAttackTarget()`

#### Threat Layer Integration

The AI system depends heavily on the behavior package's threat evaluation:

**Update Flow:**
1. At start of faction turn: `FactionThreatLevelManager.UpdateAllFactions()`
2. For each enemy faction: `CompositeThreatEvaluator.Update(currentRound)`
3. All four threat layers recompute if dirty

**Query Flow:**
1. ActionEvaluator queries: `GetRoleWeightedThreat(squadID, position)`
2. Composite evaluator fetches from four layers:
   - Melee threat
   - Ranged threat
   - Support value
   - Positional risk
3. Applies role weights and returns combined threat score

**Dirty Flag Management:**
- After each squad action: `MarkDirty()` on all threat layers
- Ensures fresh threat data for next squad in turn sequence

### Action Types

The system uses three action implementations of the `SquadAction` interface:

1. **MoveAction**:
   - Validates movement via `CombatMovementSystem.CanMoveTo()`
   - Executes via `CombatMovementSystem.MoveSquad()`
   - Updates position component and position system atomically

2. **AttackAction**:
   - Executes via `CombatActionSystem.ExecuteAttackAction()`
   - Queues attack for animation playback
   - Reduces target health and applies combat formulas

3. **WaitAction**:
   - Fallback action to prevent infinite loops
   - Marks both action flags as used
   - Allows turn to progress when no better option exists

### Attack Animation Queueing

**Purpose:** Separate AI decision-making from animation playback

**Process:**
1. During AI turn, attacks are queued but not animated
2. After all squads in faction complete their turns, queued attacks are retrieved
3. Combat mode plays animations sequentially
4. Queue is cleared after animations complete

**Key Functions:** `QueueAttack()`, `GetQueuedAttacks()`, `ClearAttackQueue()`

### Integration Points

- **With Combat Service**: Receives all combat systems (movement, action, turn manager, query cache)
- **With Behavior System**: Uses `CompositeThreatEvaluator` for position scoring
- **With Squad System**: Queries squad status, movement speed, roles, health
- **With Combat GUI**: Called during AI faction turns, provides queued attacks for animation
- **With Turn Manager**: Orchestrated by turn system during AI faction phases

---

## Power Evaluation System

**Location:** `mind/evaluation/`

### Purpose

Unified power system for encounter generation and AI assessment.

### Power Calculation

**Location:** `mind/evaluation/power.go`

**CalculateUnitPower:**
```
Total = (OffensivePower * OffensiveWeight) +
        (DefensivePower * DefensiveWeight) +
        (UtilityPower * UtilityWeight)
```

**Offensive Components:**
- Physical + Magic damage (averaged)
- Hit rate probability
- Critical chance multiplier

**Defensive Components:**
- Current/max HP ratio
- Physical + Magic resistance (averaged)
- Dodge chance (scaled)

**Utility Components:**
- Role multiplier (Tank 1.2, DPS 1.5, Support 1.0)
- Leader abilities (Rally 15.0, Heal 20.0, BattleCry 12.0)
- Cover provision value

**CalculateSquadPower:**
```
SquadPower = SumOfUnitPower * MoraleBonus * CompositionBonus * HealthPenalty
```

**CalculateSquadPowerByRange:**
- Returns power contribution at each range 1..maxThreatRange
- Units contribute if movement + attack >= currentRange
- Used by AI for range-aware threat assessment

### Power Configuration

**Location:** `power_config.go`

```go
type PowerConfig struct {
    // Category weights (sum to 1.0)
    OffensiveWeight float64
    DefensiveWeight float64
    UtilityWeight   float64

    // Sub-weights
    DamageWeight, AccuracyWeight          // Offensive
    HealthWeight, ResistanceWeight        // Defensive
    RoleWeight, AbilityWeight, CoverWeight // Utility

    // Squad modifiers
    FormationBonus, MoraleMultiplier, HealthPenalty
}
```

**Power Profiles:**

| Profile | Offensive | Defensive | Utility |
|---------|-----------|-----------|---------|
| Balanced | 0.40 | 0.40 | 0.20 |
| Offensive | 0.60 | 0.25 | 0.15 |
| Defensive | 0.25 | 0.60 | 0.15 |

### Caching

**Location:** `cache.go`

**DirtyCache** for lazy evaluation:
```go
type DirtyCache struct {
    dirty       bool
    initialized bool
    lastRound   int
}
```

- `IsValid(round)` - checks if current for this round
- `MarkDirty()` - forces recomputation
- `MarkClean(round)` - marks as valid

---

## System Integration

### Package Dependencies

```
┌─────────────────────────────────────────────────────┐
│                  Combat GUI Layer                    │
│            (gui/guicombat/combatlifecycle)           │
└──────────────────────┬──────────────────────────────┘
                       │
        ┌──────────────┼──────────────┐
        │              │              │
        ▼              ▼              ▼
   ┌────────┐    ┌─────────┐    ┌──────────┐
   │mind/ai │◄───│ Combat  │◄───│ Overworld│
   │        │    │ System  │    │ Package  │
   └────┬───┘    └────┬────┘    └────┬─────┘
        │             │              │
        │        ┌────┴────┐         │
        │        │         │         │
        ▼        ▼         ▼         ▼
   ┌──────────┐      ┌─────────────┐
   │  mind/   │      │    mind/    │
   │ behavior │      │  encounter  │
   └──────────┘      └─────────────┘
        │                   │
        └───────┬───────────┘
                ▼
          ┌──────────────┐
          │mind/evaluation│
          │ + Squad System│
          └──────────────┘
```

### Key Integration Patterns

#### 1. Overworld → Encounter → Combat

**Flow:** Strategic threat translates to tactical challenge

```
Threat Node (overworld)
    ↓ [TranslateThreatToEncounter]
Encounter Parameters (threat type, intensity)
    ↓ [SetupBalancedEncounter]
Combat Configuration (enemy squads, power-balanced)
    ↓ [CombatLifecycle.SetupEncounter]
Active Tactical Combat
```

#### 2. Combat → Behavior → AI

**Flow:** Combat state informs threat assessment informs decisions

```
Combat State (squad positions, factions)
    ↓ [UpdateThreatLayers]
Threat Maps (melee, ranged, support, positional)
    ↓ [GetRoleWeightedThreat]
Position Scores (role-specific threat evaluation)
    ↓ [EvaluateAllActions]
AI Decision (move/attack/wait)
    ↓ [ExecuteAction]
Updated Combat State
```

#### 3. Encounter Power Calculation → Threat Evaluation

**Flow:** Shared power calculation ensures consistency

```
Unit Attributes + Role + Equipment
    ↓ [CalculateUnitPower]
Unit Power Value
    ↓ [CalculateSquadPower]
Squad Power Value
    ├─→ [Encounter: createSquadForPowerBudget] (enemy generation)
    └─→ [Behavior: CalculateSquadDangerLevel] (threat assessment)
```

### Shared Data Structures

1. **LogicalPosition** (coords package):
   - Used by all systems for spatial positioning
   - Ensures consistent coordinate handling

2. **EntityManager & ECS Components** (common package):
   - All game entities stored in ECS
   - Components: SquadComponent, PositionComponent, ActionStateComponent, AttributeComponent

3. **Role Multipliers** (evaluation package):
   - Shared by encounter power calculation and behavior threat evaluation
   - Ensures consistent role strength across systems

4. **CombatQueryCache** (combat package):
   - Performance optimization for frequent queries
   - Used by behavior and AI packages to access combat state

---

## Data Flow Across Systems

### Complete Game Loop Flow

#### Overworld Turn (Strategic Layer)

```
1. Player Action / Time Passes
    ↓
2. AdvanceTick()
    ├─→ UpdateThreatNodes() - evolve all threats
    ├─→ UpdateFactions() - execute faction AI
    └─→ CheckVictoryCondition() - evaluate win/loss
    ↓
3. Player Moves Squad Near Threat Node
    ↓
4. Trigger Combat
    ├─→ TranslateThreatToEncounter()
    │   ├─ Generate enemy composition
    │   ├─ Calculate rewards
    │   └─ Create OverworldEncounterComponent
    └─→ Switch to Combat Mode
```

#### Combat Setup (Tactical Layer)

```
5. CombatLifecycleManager.SetupEncounter()
    ↓
6. Calculate Player Power
    ├─→ CalculateDeployedSquadsPower()
    └─→ For each deployed squad:
        └─→ CalculateSquadPower()
            └─→ For each unit:
                └─→ CalculateUnitPower()
    ↓
7. Generate Balanced Enemy Force
    ├─→ Get difficulty multiplier
    ├─→ Target power = playerPower * multiplier
    └─→ generateEnemySquadsByPower()
        └─→ For each squad:
            ├─→ createSquadForPowerBudget()
            └─→ Iteratively add units to reach power budget
    ↓
8. Create Factions & Position Squads
    ├─→ Create player faction (deployed squads)
    ├─→ Create enemy faction (generated squads)
    └─→ Position squads on battlefield
    ↓
9. Initialize Turn Manager & Action States
    └─→ Ready for combat turns
```

#### Combat Turn (Tactical Layer - AI)

```
10. AI Faction Turn Begins
    ↓
11. Update Threat Evaluation
    ├─→ FactionThreatLevelManager.UpdateAllFactions()
    │   └─→ For each faction: CalculateSquadDangerLevel()
    │       ├─→ DangerByRange (heuristic)
    │       └─→ ExpectedDamageByRange (accurate)
    └─→ CompositeThreatEvaluator.Update()
        ├─→ MeleeThreatLayer.Compute()
        ├─→ RangedThreatLayer.Compute()
        ├─→ SupportValueLayer.Compute()
        └─→ PositionalRiskLayer.Compute()
    ↓
12. For Each Enemy Squad:
    ├─→ Create ActionContext (role, position, health)
    ├─→ ActionEvaluator.EvaluateAllActions()
    │   ├─→ Generate movement actions
    │   │   └─→ For each reachable tile:
    │   │       └─→ scoreMovementPosition()
    │   │           ├─→ GetRoleWeightedThreat(position)
    │   │           ├─→ Calculate ally proximity
    │   │           └─→ Calculate approach bonus
    │   ├─→ Generate attack actions
    │   │   └─→ For each in-range enemy:
    │   │       └─→ scoreAttackTarget()
    │   │           ├─→ Wounded priority
    │   │           ├─→ Role threat bonus
    │   │           └─→ Counter bonus
    │   └─→ Generate wait action
    ├─→ SelectBestAction()
    ├─→ ExecuteAction()
    │   ├─→ MoveAction: CombatMovementSystem.MoveSquad()
    │   ├─→ AttackAction: CombatActionSystem.ExecuteAttackAction()
    │   └─→ WaitAction: Mark action flags used
    └─→ MarkDirty() - invalidate threat layers
    ↓
13. Return Queued Attacks
    ↓
14. Combat Mode Plays Attack Animations
    ↓
15. Next Faction Turn (repeat from step 10)
```

#### Combat Resolution

```
16. Combat Ends (all enemies defeated or player retreats)
    ↓
17. If Victory:
    ├─→ DestroyThreatNode() (remove from overworld)
    ├─→ Award rewards (gold, XP, items)
    └─→ Update player roster
    ↓
18. Return to Overworld Mode
    └─→ Continue strategic layer (back to step 1)
```

### AI Turn Sequence

```
DecideFactionTurn(factionID)
│
├── updateThreatLayers(round)
│   ├── threatManager.UpdateAllFactions()
│   └── For each evaluator:
│       ├── meleeThreat.Compute()
│       ├── rangedThreat.Compute()
│       ├── supportValue.Compute()
│       └── positionalRisk.Compute()
│
└── For each alive squad in faction:
    └── executeSquadAction()
        │
        ├── Create ActionContext
        │   ├── Get squad position, health, role
        │   └── Get threat evaluator
        │
        ├── ActionEvaluator.EvaluateAllActions()
        │   │
        │   ├── evaluateMovement()
        │   │   ├── getValidMovementTiles()
        │   │   └── For each tile:
        │   │       └── scoreMovementPosition(pos)
        │   │           └── threatEval.GetRoleWeightedThreat()
        │   │
        │   ├── evaluateAttacks()
        │   │   ├── getAttackableTargets()
        │   │   └── For each target:
        │   │       └── scoreAttackTarget()
        │   │
        │   └── [WaitAction always available]
        │
        ├── SelectBestAction() → highest score
        │
        └── bestAction.Execute()
            ├── MoveAction → MovementSystem
            ├── AttackAction → CombatActionSystem + AnimationQueue
            └── WaitAction → mark ActionState used
```

### Threat Layer Recomputation

1. Threat layers compute **once per AI turn** (at start)
2. Marked dirty after each action (positions changed)
3. Lazy evaluation prevents redundant calculations

### Critical Data Transformations

| Stage | Input | Transformation | Output |
|-------|-------|----------------|--------|
| **Threat Evolution** | Threat type + intensity | Growth algorithm + type-specific effects | Updated intensity + new child nodes |
| **Faction Intent** | Faction strength + territory + disposition | Scoring function per intent | Executed intent (expand/fortify/raid/retreat) |
| **Influence Calculation** | Threat positions + intensities | Distance falloff accumulation | Per-tile influence values |
| **Encounter Translation** | Threat type + intensity | Enemy composition + reward calculation | Encounter parameters |
| **Power Calculation** | Unit attributes + equipment | Weighted offensive/defensive/utility formula | Unit power value |
| **Squad Power** | Unit powers + morale + composition | Multiplier application | Squad power value |
| **Enemy Generation** | Target power budget + unit pool | Iterative unit selection | Enemy squad composition |
| **Threat Mapping** | Squad positions + danger levels | Spatial painting with falloff | Per-tile threat values |
| **Positional Risk** | Threat maps + squad positions | Multi-component risk analysis | Per-tile risk scores |
| **Role Weighting** | Four threat layers + squad role | Weighted combination | Single position score |
| **Action Scoring** | Available actions + threat data + tactics | Heuristic evaluation | Scored action list |
| **AI Decision** | Scored actions | Max selection | Executed action |

---

## Configuration

### AI Configuration (`aiconfig.json`)

```json
{
  "flanking": {
    "rangeBonus": 3
  },
  "isolation": {
    "safe": 2,
    "moderate": 3,
    "high": 6
  },
  "engagement": {
    "maxThreshold": 200
  },
  "retreat": {
    "safeThreatThreshold": 10
  },
  "support": {
    "healRadius": 3,
    "allyProximityRadius": 2,
    "buffPriorityRange": 4
  },
  "positionalRiskWeights": {
    "flanking": 0.4,
    "isolation": 0.3,
    "pressure": 0.2,
    "retreat": 0.1
  },
  "roleThreatWeights": {
    "Tank": {
      "melee": -0.5,
      "ranged": 0.3,
      "support": 0.0,
      "positional": 0.5
    },
    "DPS": {
      "melee": 0.7,
      "ranged": 0.5,
      "support": 0.0,
      "positional": 0.6
    },
    "Support": {
      "melee": 1.0,
      "ranged": 1.0,
      "support": -1.0,
      "positional": 0.8
    }
  }
}
```

### Power Configuration (`powerconfig.json`)

```json
{
  "profiles": {
    "Balanced": {
      "offensiveWeight": 0.4,
      "defensiveWeight": 0.4,
      "utilityWeight": 0.2
    },
    "Offensive": {
      "offensiveWeight": 0.6,
      "defensiveWeight": 0.25,
      "utilityWeight": 0.15
    },
    "Defensive": {
      "offensiveWeight": 0.25,
      "defensiveWeight": 0.6,
      "utilityWeight": 0.15
    }
  },
  "roleMultipliers": {
    "Tank": 1.2,
    "DPS": 1.5,
    "Support": 1.0
  },
  "abilityPower": {
    "Rally": 15.0,
    "Heal": 20.0,
    "BattleCry": 12.0
  },
  "compositionBonus": {
    "1": 1.0,
    "2": 1.1,
    "3": 1.2,
    "4": 1.3
  }
}
```

---

## Debug Tools

### Danger Visualizer

**Location:** `mind/behavior/dangervisualizer.go`

Toggle threat visualization on the map:
- Switch between enemy threat and player threat views
- Cycle between danger metric (heuristic) and expected damage metric
- Color-codes tiles: red for danger, blue for expected damage
- Updates per round (cached)

---

## Implementation Notes

### Attack Queue
- AI decisions don't execute attacks immediately
- Attacks queued and animated after AI turn completes
- Allows proper animation sequencing

### Movement Validation
- All movement checked against `CombatMovementSystem.CanMoveTo()`
- Prevents invalid actions that would fail execution

### Role-Weighted Positioning
- Each role has different threat weights
- Tanks attracted to melee (negative weight)
- Support attracted to wounded allies (negative weight)

### Health Calculation
- Centralized in `squads.GetSquadHealthPercent()`
- Used by AI and behavior packages consistently

---

## Conclusion

TinkerRogue's game systems form a cohesive pipeline from strategic overworld simulation to tactical AI decision-making:

1. **Overworld** evolves threats and manages strategic state
2. **Encounter** translates threats into power-balanced combat challenges
3. **Behavior** evaluates the tactical landscape and quantifies threats
4. **AI** uses threat data to make intelligent squad decisions

Each system maintains clear responsibilities while integrating through well-defined interfaces. The power-based balancing ensures encounters scale appropriately, while the multi-layered threat evaluation enables emergent tactical behavior where unit roles naturally organize into sensible formations without explicit programming.

This architecture supports both strategic depth (faction expansion, threat evolution) and tactical complexity (positioning, threat assessment, role-based tactics) while maintaining computational efficiency through caching and dirty-flag optimization.
