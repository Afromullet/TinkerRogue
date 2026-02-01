# AI Algorithm Architecture

**Last Updated:** 2026-02-01

This document describes the AI decision-making and threat assessment systems in TinkerRogue.

---

## Table of Contents

1. [Overview](#overview)
2. [Directory Structure](#directory-structure)
3. [AI Decision-Making System](#ai-decision-making-system)
4. [Threat Assessment System](#threat-assessment-system)
5. [Power Evaluation System](#power-evaluation-system)
6. [Data Flow](#data-flow)
7. [Configuration](#configuration)
8. [Integration Points](#integration-points)

---

## Overview

The AI system uses a **layered threat model** combined with **role-weighted utility scoring** to make tactical decisions. Each AI-controlled unit evaluates possible actions (move, attack, wait) and selects the highest-scoring option based on its role and the current tactical situation.

### Key Design Principles

- **Composable Threat Layers**: Multiple independent layers calculate different aspects of tactical threat
- **Role-Based Weighting**: Different unit roles (Tank, DPS, Support) weight threats differently
- **Data-Driven Configuration**: JSON templates allow tuning without code changes
- **Lazy Evaluation**: Threat layers cache efficiently with dirty flag invalidation
- **Unified Power System**: Same calculations used for encounter generation and AI assessment

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
```

---

## AI Decision-Making System

### AIController (`mind/ai/ai_controller.go`)

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

### Action Evaluation (`mind/ai/action_evaluator.go`)

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

### Scoring Logic

**Movement Scoring:**
- Base score: 50.0
- Modified by threat avoidance (lower threat = higher score)
- Modified by ally proximity
- Offensive roles get bonus for positions closer to enemies
- Support units get penalty for approaching enemies

**Attack Scoring:**
- Base score: 100.0 (always prefer attack over movement)
- Modified by target health (wounded targets preferred)
- Modified by target role (high-value targets preferred)

---

## Threat Assessment System

### Layered Threat Model

The threat system uses composable layers where each calculates one aspect of tactical threat:

```
CompositeThreatEvaluator
├── MeleeThreatLayer       # Melee attack danger
├── RangedThreatLayer      # Ranged/magic danger
├── SupportValueLayer      # Healing/buff opportunities
└── PositionalRiskLayer    # Flanking, isolation, pressure
```

### ThreatLayer Interface (`mind/behavior/threat_layers.go`)

```go
type ThreatLayer interface {
    Compute()              // Recalculate threat values
    IsValid(round int) bool // Check if current for round
    MarkDirty()            // Force recomputation
}
```

### Melee Threat Layer (`threat_melee.go`)

Calculates danger from enemy melee squads.

**Formula:**
```
Threat = ExpectedDamageByRange[1] * RoleMultiplier * DistanceFalloff
```

- **Effective Range**: Movement Speed + Attack Range
- **Distance Falloff**: Linear (closer = higher threat)
- Stores threat by position and by squad

### Ranged Threat Layer (`threat_ranged.go`)

Calculates danger from enemy ranged and magic squads.

**Formula:**
```
Threat = ExpectedDamageByRange[maxRange] * RoleModifier
```

- **No Distance Falloff**: Full threat at all ranges (ranged advantage)
- Tracks line-of-fire zones per squad

### Support Value Layer (`threat_support.go`)

Identifies valuable positions for support units.

**Heal Priority Calculation:**
```
Priority = 1.0 - SquadHealthPercent
```
(More wounded = higher priority)

- Paints support value around wounded squads with linear falloff
- Tracks ally proximity for grouping bonus
- Heal radius: 3 tiles
- Ally proximity radius: 2 tiles

### Positional Risk Layer (`threat_positional.go`)

Calculates tactical positioning risks beyond raw damage.

**Four Sub-Metrics:**

| Metric | Description | Range |
|--------|-------------|-------|
| Flanking Risk | Attacked from multiple directions | 0-1 |
| Isolation Risk | Distance from nearest ally | 0-1 |
| Engagement Pressure | Combined melee+ranged threat | 0-1 |
| Retreat Quality | % of low-threat escape routes | 0-1 |

**Positional Risk Weights:**
- Flanking: 0.4
- Isolation: 0.3
- Pressure: 0.2
- Retreat: 0.1

### Composite Threat Evaluation (`threat_composite.go`)

Combines all layers with role-based weighting:

```go
func (c *CompositeThreatEvaluator) GetRoleWeightedThreat(
    squadID ecs.EntityID,
    pos coords.LogicalPosition,
) float64
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

### Threat Painting (`threat_painting.go`)

Utility functions for painting threat values onto position maps:

```go
func PaintThreatToMap(
    threatMap map[coords.LogicalPosition]float64,
    center coords.LogicalPosition,
    radius int,
    threatValue float64,
    falloffFunc FalloffFunc,
)
```

**Falloff Functions:**
- **LinearFalloff**: `1.0 - (distance / (maxRange+1))`
- **NoFalloff**: `1.0` (full threat at all ranges)
- **QuadraticFalloff**: `1.0 - (distance/maxRange)²`

### Faction Threat Manager (`dangerlevel.go`)

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

---

## Power Evaluation System

### Power Calculation (`mind/evaluation/power.go`)

Unified power system for encounter generation and AI assessment.

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

### Power Configuration (`power_config.go`)

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

### Caching (`cache.go`)

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

## Data Flow

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

## Integration Points

### AIController Initialization

```go
NewAIController(
    entityManager,
    turnManager,
    movementSystem,
    combatActSystem,
    combatCache,
    threatManager,        // FactionThreatLevelManager
    layerEvaluators,      // map[factionID]*CompositeThreatEvaluator
)
```

### Combat System Integration

- **Movement**: Uses `CombatMovementSystem.CanMoveTo()` for validation
- **Attack**: Uses `CombatActionSystem.ExecuteAttackAction()`
- **Animation**: Attacks queued via `QueueAttack()` for sequencing

### Combat Cache

- `combat.CombatQueryCache` caches action states by squad ID
- Used to find squad's current action state each decision

### Encounter Integration

- `EncounterService` coordinates encounter lifecycle
- `EncounterGenerator` creates AI-controlled enemy squads
- Power calculations determine encounter difficulty

---

## Debug Tools

### Danger Visualizer (`dangervisualizer.go`)

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
