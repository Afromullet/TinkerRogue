# AI Algorithm Architecture

**Last Updated:** 2026-02-11

A comprehensive technical reference for TinkerRogue's AI decision-making, threat assessment, and combat behavior systems.

---

## Table of Contents

1. [Executive Summary](#executive-summary)
2. [Architecture Overview](#architecture-overview)
3. [Core AI Systems](#core-ai-systems)
4. [Threat Assessment Framework](#threat-assessment-framework)
5. [Power Evaluation System](#power-evaluation-system)
6. [Action Selection Algorithm](#action-selection-algorithm)
7. [Configuration System](#configuration-system)
8. [Data Flow and Dependencies](#data-flow-and-dependencies)
9. [Performance Considerations](#performance-considerations)
10. [Extension Points](#extension-points)

---

## Executive Summary

TinkerRogue's AI uses a **layered threat assessment** system combined with **role-based behavior weights** to create tactically-aware computer opponents. The architecture separates concerns into distinct subsystems:

- **AI Controller** (`mind/ai/`) - Orchestrates turn execution and action selection
- **Threat Evaluation** (`mind/behavior/`) - Multi-layered spatial threat analysis
- **Power Calculation** (`mind/evaluation/`) - Unified combat power assessment
- **Encounter Generation** (`mind/encounter/`) - Dynamic enemy creation based on power budgets

**Key Design Principles:**

1. **Data-Driven Configuration** - All weights, thresholds, and multipliers loaded from JSON
2. **Separation of Concerns** - Power calculation shared between AI and encounter generation
3. **Layer Composition** - Multiple threat layers combined via role-specific weights
4. **Cache-Friendly** - Dirty flag invalidation prevents redundant recomputation
5. **ECS-First** - Pure component queries, no entity pointer caching

---

## Architecture Overview

### System Boundaries

```
┌─────────────────────────────────────────────────────────────┐
│                    GUI / Game Loop                           │
└───────────────────────┬─────────────────────────────────────┘
                        │
                        ▼
┌─────────────────────────────────────────────────────────────┐
│                   AIController                               │
│  • DecideFactionTurn()                                       │
│  • Attack queue management                                   │
│  • Turn orchestration                                        │
└──────────┬─────────────────────┬──────────────────┬─────────┘
           │                     │                  │
           ▼                     ▼                  ▼
┌──────────────────┐  ┌──────────────────┐  ┌──────────────┐
│ ActionEvaluator  │  │ CompositeThreat  │  │ TurnManager  │
│                  │  │   Evaluator      │  │              │
│ • Movement score │  │                  │  │ • Initiative │
│ • Attack score   │  │ • Role weights   │  │ • Round mgmt │
│ • Fallback wait  │  │ • Layer queries  │  │              │
└──────────┬───────┘  └────────┬─────────┘  └──────────────┘
           │                   │
           ▼                   ▼
┌─────────────────────────────────────────────────────────────┐
│              Threat Layer Subsystems                         │
│                                                              │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────────┐  │
│  │ CombatThreat │  │ SupportValue │  │ PositionalRisk   │  │
│  │   Layer      │  │    Layer     │  │     Layer        │  │
│  │              │  │              │  │                  │  │
│  │ • Melee      │  │ • Heal prior │  │ • Flanking risk  │  │
│  │ • Ranged     │  │ • Ally prox  │  │ • Isolation risk │  │
│  └──────────────┘  └──────────────┘  └──────────────────┘  │
└────────────────────────┬────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────┐
│           FactionThreatLevelManager                          │
│  • SquadThreatLevel (ThreatByRange map)                     │
│  • Distance tracking                                         │
│  • Uses shared power calculation                            │
└────────────────────────┬────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────┐
│              Power Evaluation (shared)                       │
│  • CalculateSquadPower()                                     │
│  • CalculateSquadPowerByRange()                              │
│  • Used by AI threat + encounter generation                  │
└─────────────────────────────────────────────────────────────┘
```

### Key Dependencies

- **EntityManager**: ECS world access for all systems
- **CombatQueryCache**: Optimized faction/squad queries
- **CoordinateManager**: Spatial indexing and distance calculations
- **GlobalPositionSystem**: O(1) entity lookup by position
- **Config Templates**: JSON-loaded weights and parameters

---

## Core AI Systems

### AIController

**Location:** `mind/ai/ai_controller.go`

**Responsibilities:**
- Orchestrates AI turn execution for enemy factions
- Manages threat layer updates
- Queues attacks for animation playback
- Delegates action selection to ActionEvaluator

**Algorithm:**

```go
DecideFactionTurn(factionID):
  1. Clear attack queue from previous turn
  2. Update threat layers (mark dirty after each action)
  3. Get all alive squads in faction
  4. For each squad:
     a. While squad has actions remaining:
        - Create ActionContext (current state + threat evaluator)
        - Generate and score all possible actions
        - Select best action (highest score)
        - Execute action
        - Mark threat layers dirty
  5. Return true if any actions executed
```

**Key Features:**
- **Exhaustive Action Processing**: Each squad uses ALL available actions before moving to next squad
- **Incremental Threat Updates**: Layers marked dirty after each action (positions change)
- **Attack Queueing**: Stores attacks for GUI animation after AI turn completes
- **Faction-Scoped Evaluators**: Each faction maintains separate threat evaluator

**Performance Notes:**
- Threat layer updates are lazy (dirty flag prevents redundant recalculation)
- Combat cache reduces ECS query overhead
- Attack queue prevents immediate GUI blocking during AI turn

---

### ActionContext

**Location:** `mind/ai/ai_controller.go`

**Purpose:** Bundles all data needed for action evaluation in one structure.

**Contents:**
```go
type ActionContext struct {
    SquadID     ecs.EntityID
    FactionID   ecs.EntityID
    ActionState *ActionStateData  // HasMoved, HasActed flags

    ThreatEval  *CompositeThreatEvaluator  // Role-weighted threat queries
    Manager     *EntityManager
    MovementSystem *CombatMovementSystem   // For validating tiles

    // Cached squad data
    SquadRole   UnitRole
    CurrentPos  LogicalPosition
    SquadHealth float64  // Average HP percentage (0-1)
}
```

**Why Context Object?**
- Eliminates repetitive parameter passing
- Guarantees consistent data snapshot for action evaluation
- Pre-caches expensive queries (role, position, health)
- Provides reference to AI controller for attack queueing

---

### ActionEvaluator

**Location:** `mind/ai/action_evaluator.go`

**Responsibilities:**
- Generates all valid actions for a squad
- Scores actions based on role and threat
- Provides fallback wait action

**Action Types:**

1. **MoveAction** - Movement to valid tile
2. **AttackAction** - Attack enemy squad in range
3. **WaitAction** - Skip turn (marks HasMoved and HasActed)

**Evaluation Algorithm:**

```go
EvaluateAllActions():
  actions = []

  // Generate movement actions if not moved
  if !HasMoved:
    for each valid movement tile:
      score = scoreMovementPosition(tile)
      actions.append(MoveAction{tile, score})

  // Generate attack actions if not acted
  if !HasActed:
    for each attackable enemy:
      score = scoreAttackTarget(enemy)
      actions.append(AttackAction{enemy, score})

  // Always include fallback
  actions.append(WaitAction{score=0.0})

  return actions
```

**Movement Validation:**
- Uses `MovementSystem.CanMoveTo()` to validate tiles
- CRITICAL: Without validation, AI generates invalid moves that fail execution
- Considers: occupied tiles, blocked terrain, move speed

**Movement Scoring:**

```go
scoreMovementPosition(pos):
  // Base score
  baseScore = 50.0

  // Threat evaluation (role-weighted)
  threat = ThreatEval.GetRoleWeightedThreat(squadID, pos)
  score = baseScore - threat

  // Ally proximity bonus (avoid isolation)
  allyProximity = SupportLayer.GetAllyProximityAt(pos)
  score += allyProximity * 3.0

  // Approach enemy bonus (offensive roles)
  approachBonus = scoreApproachEnemy(pos)
  score += approachBonus

  return score
```

**Attack Scoring:**

```go
scoreAttackTarget(targetID):
  baseScore = 100.0  // CRITICAL: Higher than movement base (50)
                     // Ensures AI prefers attacking when in range

  // Prioritize wounded targets (focus fire)
  targetHealth = GetSquadHealthPercent(targetID)
  score += (1.0 - targetHealth) * 20.0

  // Prioritize high-threat targets
  targetRole = GetSquadPrimaryRole(targetID)
  if targetRole == DPS:
    score += 15.0
  else if targetRole == Support:
    score += 10.0

  // Role counter bonuses
  if myRole == DPS && targetRole == Support:
    score += 10.0  // DPS hunts support
  else if myRole == Tank && targetRole == DPS:
    score += 10.0  // Tank locks down DPS

  return score
```

**Approach Enemy Scoring:**

```go
scoreApproachEnemy(pos):
  nearestEnemy, currentDistance = findNearestEnemy()
  newDistance = distance(pos, enemyPos)
  distanceImprovement = currentDistance - newDistance

  // Role-based multipliers
  switch squadRole:
    Tank:    multiplier = 15.0   // Strongly seek melee
    DPS:     multiplier = 8.0    // Moderately engage
    Support: multiplier = -5.0   // Maintain distance

  approachScore = distanceImprovement * multiplier

  // Bonus for attack range proximity
  maxRange = getMaxAttackRange()
  if newDistance <= maxRange:
    approachScore += 20.0  // In range next turn
  else if newDistance <= maxRange+2:
    approachScore += 10.0  // Close to range

  return approachScore
```

**Why Approach Bonus Exists:**
- Without it, AI only avoids threat and flees
- Creates offensive pressure from tanks/DPS
- Balances threat avoidance with engagement

---

## Threat Assessment Framework

### Composite Threat Evaluator

**Location:** `mind/behavior/threat_composite.go`

**Purpose:** Combines multiple threat layers with role-specific weights to produce tactical position scores.

**Architecture:**

```
CompositeThreatEvaluator
│
├─ CombatThreatLayer (unified melee + ranged)
│  ├─ Melee threat (linear falloff over move+attack range)
│  └─ Ranged threat (no falloff, line-of-fire zones)
│
├─ SupportValueLayer
│  ├─ Heal priority (inverse of squad health)
│  └─ Ally proximity (count of nearby allies)
│
└─ PositionalRiskLayer
   ├─ Flanking risk (attacked from multiple directions)
   ├─ Isolation risk (distance from nearest ally)
   ├─ Engagement pressure (normalized total threat)
   └─ Retreat quality (low-threat adjacent tiles)
```

**Role-Weighted Threat Query:**

```go
GetRoleWeightedThreat(squadID, pos):
  role = GetSquadPrimaryRole(squadID)
  weights = GetRoleBehaviorWeights(role)  // From aiconfig.json

  meleeThreat = CombatLayer.GetMeleeThreatAt(pos)
  rangedThreat = CombatLayer.GetRangedPressureAt(pos)
  supportValue = SupportLayer.GetSupportValueAt(pos)
  positionalRisk = PositionalLayer.GetTotalRiskAt(pos)

  // Combine with role-specific weights
  // Negative weights = attraction, Positive weights = avoidance
  totalThreat = meleeThreat * weights.MeleeWeight +
                rangedThreat * weights.RangedWeight +
                supportValue * weights.SupportWeight +
                positionalRisk * weights.PositionalWeight

  return totalThreat
```

**Layer Update Cycle:**

```go
Update(currentRound):
  if !isDirty && lastUpdateRound == currentRound:
    return  // Skip if already up-to-date

  // 1. Compute combat layer (provides melee + ranged data)
  CombatLayer.Compute()

  // 2. Compute derived layers (depend on combat data)
  SupportLayer.Compute()
  PositionalLayer.Compute()

  // 3. Mark clean
  isDirty = false
  lastUpdateRound = currentRound
```

**Dirty Flag Management:**
- Marked dirty after each AI action (positions change)
- Prevents redundant recomputation within same round
- Each layer tracks own dirty state independently

---

### Combat Threat Layer

**Location:** `mind/behavior/threat_combat.go`

**Purpose:** Unified layer computing both melee and ranged threat from enemy squads.

**Data Structures:**

```go
type CombatThreatLayer struct {
  // Melee threat data
  meleeThreatByPos   map[LogicalPosition]float64  // Position -> melee threat
  meleeThreatBySquad map[EntityID]float64         // Squad -> total emitted threat

  // Ranged threat data
  rangedPressureByPos map[LogicalPosition]float64  // Position -> ranged pressure
  lineOfFireZones     map[EntityID][]LogicalPosition  // Squad -> threatened tiles

  baseThreatMgr *FactionThreatLevelManager  // Source of ThreatByRange
}
```

**Melee Threat Calculation:**

```go
computeMeleeThreat(squadID, squadPos, squadThreat):
  moveSpeed = GetSquadMovementSpeed(squadID)
  maxMeleeRange = getMaxRangeForAttackTypes(squadID, MeleeAttackTypes, 1)
  threatRadius = moveSpeed + maxMeleeRange

  // Use danger at range 1 (includes role multipliers from power system)
  totalThreat = squadThreat.ThreatByRange[1]

  // Paint threat with linear falloff
  PaintThreatToMap(
    meleeThreatByPos,
    squadPos,
    threatRadius,
    totalThreat,
    LinearFalloff,
    trackPositions=false
  )

  meleeThreatBySquad[squadID] = totalThreat
```

**Ranged Threat Calculation:**

```go
computeRangedThreat(squadID, squadPos, squadThreat):
  maxRange = getMaxRangeForAttackTypes(squadID, RangedAttackTypes, 3)

  // Use danger at max range (includes role multipliers)
  rangedDanger = squadThreat.ThreatByRange[maxRange]

  // Paint threat with NO falloff (archers equally dangerous at all ranges)
  lineOfFireZones[squadID] = PaintThreatToMap(
    rangedPressureByPos,
    squadPos,
    maxRange,
    rangedDanger,
    NoFalloff,
    trackPositions=true  // Track for visualization
  )
```

**Why Unified Layer?**
- Reduces code duplication (originally separate layers)
- Shares common dependencies (baseThreatMgr, cache)
- Simplifies layer update orchestration
- Maintains separate query APIs for backward compatibility

**Threat Painting Algorithm:**

```go
PaintThreatToMap(threatMap, center, radius, threatValue, falloffFunc, trackPositions):
  paintedPositions = []

  for dx in [-radius, radius]:
    for dy in [-radius, radius]:
      pos = center + (dx, dy)
      distance = ChebyshevDistance(center, pos)

      if distance > 0 && distance <= radius:
        falloff = falloffFunc(distance, radius)
        threatMap[pos] += threatValue * falloff

        if trackPositions:
          paintedPositions.append(pos)

  return paintedPositions
```

**Falloff Functions:**

```go
// Linear: threat decreases linearly with distance
LinearFalloff(distance, maxRange):
  return 1.0 - (distance / (maxRange + 1))

// No Falloff: full threat at all ranges
NoFalloff(distance, maxRange):
  return 1.0
```

---

### Support Value Layer

**Location:** `mind/behavior/threat_support.go`

**Purpose:** Identifies valuable positions for support squads (healers, buffers).

**Core Concept:**
- Wounded allies create "support value" radiating from their position
- Support squads attracted to high-value positions (negative weight)
- Ally proximity tracking helps all units avoid isolation

**Data Structures:**

```go
type SupportValueLayer struct {
  healPriority    map[EntityID]float64            // Squad -> heal urgency (0-1)
  supportValuePos map[LogicalPosition]float64     // Position -> support value
  allyProximity   map[LogicalPosition]int         // Position -> nearby ally count
}
```

**Computation:**

```go
Compute():
  clear(healPriority, supportValuePos, allyProximity)

  squadIDs = GetActiveSquadsForFaction(factionID)

  for each squadID:
    // Calculate heal priority (inverse of health)
    avgHP = GetSquadHealthPercent(squadID)
    healPriority[squadID] = 1.0 - avgHP

    squadPos = GetSquadMapPosition(squadID)

    // Paint support value around wounded allies
    healRadius, proximityRadius = GetSupportLayerParams()  // From config
    PaintThreatToMap(supportValuePos, squadPos, healRadius, healPriority, LinearFalloff)

    // Track ally proximity separately
    for each position within proximityRadius:
      allyProximity[pos]++
```

**Configuration (aiconfig.json):**

```json
{
  "supportLayer": {
    "healRadius": 3  // Proximity radius derived as healRadius - 1
  }
}
```

**Query APIs:**

```go
GetSupportValueAt(pos):
  return supportValuePos[pos]  // Higher = better for healers

GetAllyProximityAt(pos):
  return allyProximity[pos]  // Count of nearby allies

GetMostDamagedAlly():
  return squadID with highest healPriority
```

**Role Behavior:**
- **Support squads**: Negative weight (-1.0) attracts them to high support value
- **Other roles**: Low positive weight (0.1-0.2) for minor heal consideration
- Creates emergent behavior: supports move toward wounded allies

---

### Positional Risk Layer

**Location:** `mind/behavior/threat_positional.go`

**Purpose:** Evaluates tactical positioning risks beyond raw damage threat.

**Risk Components:**

1. **Flanking Risk** - Being attacked from multiple directions
2. **Isolation Risk** - Distance from allied support
3. **Engagement Pressure** - Total damage exposure (normalized)
4. **Retreat Quality** - Availability of low-threat escape routes

**Data Structures:**

```go
type PositionalRiskLayer struct {
  flankingRisk       map[LogicalPosition]float64  // 0-1 (0=safe, 1=flanked)
  isolationRisk      map[LogicalPosition]float64  // 0-1 (0=supported, 1=isolated)
  engagementPressure map[LogicalPosition]float64  // 0-1 (normalized)
  retreatQuality     map[LogicalPosition]float64  // 0-1 (0=trapped, 1=safe exits)
}
```

**Flanking Risk Computation:**

```go
computeFlankingRisk(enemyFactions):
  threatDirections = map[LogicalPosition]map[int]bool  // pos -> set of attack angles

  for each enemyFaction:
    for each enemySquad:
      moveSpeed = GetSquadMovementSpeed(enemySquad)
      threatRange = moveSpeed + GetFlankingThreatRangeBonus()  // From config

      // Paint threat directions (8-directional)
      for each position in threatRange:
        angle = getDirection(dx, dy)  // 0-7 (N, NE, E, SE, S, SW, W, NW)
        threatDirections[pos][angle] = true

  // Calculate risk based on direction count
  for each pos, directions:
    numDirections = len(directions)
    if numDirections >= 3:
      flankingRisk[pos] = 1.0  // High risk
    else if numDirections == 2:
      flankingRisk[pos] = 0.5  // Moderate risk
    else:
      flankingRisk[pos] = 0.0  // Safe (single direction)
```

**Isolation Risk Computation:**

```go
computeIsolationRisk(alliedSquads):
  threshold = GetIsolationThreshold()  // From config (e.g., 3)
  maxDist = 8  // Internal constant

  allyPositions = collect all allied squad positions

  for each map position:
    minDistance = distance to nearest ally

    if minDistance >= maxDist:
      isolationRisk[pos] = 1.0  // Fully isolated
    else if minDistance > threshold:
      // Linear gradient from threshold to maxDist
      isolationRisk[pos] = (minDistance - threshold) / (maxDist - threshold)
    else:
      isolationRisk[pos] = 0.0  // Well-supported
```

**Engagement Pressure Computation:**

```go
computeEngagementPressure():
  maxPressure = 200  // Normalizer constant

  for each map position:
    meleeThreat = CombatLayer.GetMeleeThreatAt(pos)
    rangedThreat = CombatLayer.GetRangedPressureAt(pos)

    totalPressure = meleeThreat + rangedThreat
    engagementPressure[pos] = min(totalPressure / maxPressure, 1.0)
```

**Retreat Quality Computation:**

```go
computeRetreatQuality():
  retreatThreshold = GetRetreatSafeThreatThreshold()  // From config (e.g., 10)

  for each map position:
    retreatScore = 0.0
    checkedDirs = 0

    // Check all 8 adjacent positions
    for each adjacent position:
      meleeThreat = CombatLayer.GetMeleeThreatAt(adjacentPos)
      rangedThreat = CombatLayer.GetRangedPressureAt(adjacentPos)

      if meleeThreat < retreatThreshold && rangedThreat < retreatThreshold:
        retreatScore += 1.0  // Safe exit
      checkedDirs++

    // Retreat quality = percentage of safe adjacent tiles
    retreatQuality[pos] = retreatScore / checkedDirs
```

**Total Risk Calculation:**

```go
GetTotalRiskAt(pos):
  flanking = flankingRisk[pos]
  isolation = isolationRisk[pos]
  pressure = engagementPressure[pos]
  retreatPenalty = 1.0 - retreatQuality[pos]

  // Simple average of all risk factors
  return (flanking + isolation + pressure + retreatPenalty) * 0.25
```

---

### Faction Threat Level Manager

**Location:** `mind/behavior/dangerlevel.go`

**Purpose:** Base threat data source for all threat layers. Computes raw power-by-range for each squad.

**Architecture:**

```
FactionThreatLevelManager
│
├─ FactionThreatLevel (per faction)
│  └─ SquadThreatLevel (per squad)
│     ├─ ThreatByRange map[int]float64
│     └─ SquadDistanceTracker
```

**Data Structures:**

```go
type SquadThreatLevel struct {
  squadID       EntityID
  ThreatByRange map[int]float64  // Range -> threat power

  SquadDistances *SquadDistanceTracker
}

type SquadDistanceTracker struct {
  SourceSquadID     EntityID
  EnemiesByDistance map[int][]EntityID  // Distance -> enemy squad IDs

  lastUpdateRound int
  isDirty         bool
  isInitialized   bool
}
```

**Threat Calculation:**

```go
CalculateThreatLevels():
  // Use shared power calculation (mind/evaluation/power.go)
  config = GetPowerConfigByProfile("Balanced")
  ThreatByRange = CalculateSquadPowerByRange(squadID, manager, config)
```

**Why Shared Power System?**
- Eliminates duplication between AI and encounter generation
- Ensures consistent threat assessment
- Single source of truth for combat power
- ThreatByRange already includes role multipliers from power config

**Update Cycle:**

```go
UpdateAllFactions():
  for each faction:
    for each squad in faction:
      CalculateThreatLevels()  // Recomputes ThreatByRange
```

**Integration with Combat Layer:**
- CombatThreatLayer reads `ThreatByRange[1]` for melee (close-range power)
- CombatThreatLayer reads `ThreatByRange[maxRange]` for ranged (max-range power)
- Powers already scaled by role multipliers (Tank=1.2, DPS=1.5, Support=1.0)

---

## Power Evaluation System

**Location:** `mind/evaluation/`

**Purpose:** Unified combat power assessment shared by AI threat evaluation and encounter generation.

### Architecture

```
Power Calculation (Shared)
│
├─ Unit Power
│  ├─ Offensive Power (damage output)
│  ├─ Defensive Power (survivability)
│  └─ Utility Power (role, abilities, cover)
│
├─ Squad Power
│  ├─ Sum unit powers
│  ├─ Composition bonus (attack type diversity)
│  └─ Health multiplier (wounded penalty)
│
└─ Squad Power By Range
   └─ Map of range -> power (for threat assessment)
```

### Power Configuration

**Location:** `mind/evaluation/power_config.go`

**Data Structure:**

```go
type PowerConfig struct {
  OffensiveWeight float64  // Weight for damage output (0.0-1.0)
  DefensiveWeight float64  // Weight for survivability (0.0-1.0)
  UtilityWeight   float64  // Weight for utility (0.0-1.0)
  HealthPenalty   float64  // Exponent for health scaling (e.g., 2.0)
}
```

**Configuration (powerconfig.json):**

```json
{
  "profiles": [
    {
      "name": "Balanced",
      "offensiveWeight": 0.4,
      "defensiveWeight": 0.4,
      "utilityWeight": 0.2,
      "healthPenalty": 2.0
    }
  ]
}
```

**Accessor Pattern:**

```go
GetPowerConfigByProfile("Balanced"):
  // Try to find in loaded config
  for each profile in templates.PowerConfigTemplate.Profiles:
    if profile.Name == profileName:
      return PowerConfig from profile

  // Fallback to defaults
  return default Balanced config
```

---

### Unit Power Calculation

**Function:** `calculateUnitPower()` in `mind/evaluation/power.go`

**Algorithm:**

```go
calculateUnitPower(unitID, manager, config):
  // Get components
  attr = GetAttributes(unitID)
  roleData = GetUnitRole(unitID)

  // Calculate category powers
  offensivePower = CalculateOffensivePower(attr, config)
  defensivePower = CalculateDefensivePower(attr, config)
  utilityPower = CalculateUtilityPower(entity, attr, roleData, config)

  // Weighted sum
  totalPower = (offensivePower * config.OffensiveWeight) +
               (defensivePower * config.DefensiveWeight) +
               (utilityPower * config.UtilityWeight)

  return totalPower
```

**Offensive Power:**

```go
CalculateOffensivePower(attr, config):
  avgDamage = (PhysicalDamage + MagicDamage) / 2.0
  hitRate = HitRate / 100.0
  critMultiplier = 1.0 + (CritChance / 100.0 * CritDamageBonus)

  // Expected damage per attack
  return avgDamage * hitRate * critMultiplier
```

**Defensive Power:**

```go
CalculateDefensivePower(attr, config):
  // Effective health based on current HP
  healthRatio = CurrentHealth / MaxHealth
  effectiveHealth = MaxHealth * healthRatio

  // Resistance provides damage reduction
  avgResistance = (PhysicalResistance + MagicDefense) / 2.0

  // Dodge multiplier: HP / (1 - dodgeChance)
  dodgeChance = DodgeChance / 100.0
  dodgeMultiplier = 1.0 / max(1.0 - dodgeChance, 0.5)  // Cap at 2x

  return (effectiveHealth * dodgeMultiplier) + avgResistance
```

**Utility Power:**

```go
CalculateUtilityPower(entity, attr, roleData, config):
  roleValue = calculateRoleValue(roleData)
  abilityValue = calculateAbilityValue(entity)
  coverValue = calculateCoverValue(entity)

  return roleValue + abilityValue + coverValue

calculateRoleValue(roleData):
  roleMultiplier = GetRoleMultiplierFromConfig(role)  // From powerconfig.json
  return roleMultiplier * RoleScalingFactor  // 10.0

calculateAbilityValue(entity):
  if !IsLeader(entity):
    return 0.0

  totalPower = 0.0
  for each equipped ability:
    totalPower += GetAbilityPowerValue(ability)  // From powerconfig.json

  return totalPower

calculateCoverValue(entity):
  coverData = GetCoverData(entity)
  if coverData == nil:
    return 0.0

  return coverData.CoverValue * CoverScaling * CoverBeneficiaryMultiplier
```

---

### Squad Power Calculation

**Function:** `CalculateSquadPower()` in `mind/evaluation/power.go`

**Algorithm:**

```go
CalculateSquadPower(squadID, manager, config):
  unitIDs = GetUnitIDsInSquad(squadID)
  if len(unitIDs) == 0:
    return 0.0

  // Sum unit powers
  totalUnitPower = 0.0
  for each unitID:
    unitPower = calculateUnitPower(unitID, manager, config)
    totalUnitPower += unitPower

  basePower = totalUnitPower

  // Apply squad-level modifiers
  compositionMod = CalculateSquadCompositionBonus(squadID, manager)
  basePower *= compositionMod

  healthPercent = GetSquadHealthPercent(squadID, manager)
  basePower *= CalculateHealthMultiplier(healthPercent, config.HealthPenalty)

  return basePower
```

**Composition Bonus:**

```go
CalculateSquadCompositionBonus(squadID, manager):
  attackTypes = set()

  for each unit in squad:
    attackTypes.add(unit.AttackType)

  uniqueTypes = len(attackTypes)
  return GetCompositionBonusFromConfig(uniqueTypes)
```

**Configuration (powerconfig.json):**

```json
{
  "compositionBonuses": [
    {"uniqueTypes": 1, "bonus": 0.8},  // Mono-type penalty
    {"uniqueTypes": 2, "bonus": 1.1},  // Dual-type bonus
    {"uniqueTypes": 3, "bonus": 1.2},  // Triple-type bonus
    {"uniqueTypes": 4, "bonus": 1.3}   // Quad-type bonus
  ]
}
```

**Health Multiplier:**

```go
CalculateHealthMultiplier(healthPercent, healthPenalty):
  // Health penalty as exponent
  // e.g., 50% health with penalty 2.0 = 0.5^2 = 0.25 power
  return pow(healthPercent, healthPenalty)
```

**Why Composition Matters:**
- Encourages diverse unit types (melee row, melee column, ranged, magic)
- Penalties mono-composition squads (0.8x)
- Rewards mixed squads (up to 1.3x for 4 types)
- Applies to both player and AI squads

---

### Squad Power By Range

**Function:** `CalculateSquadPowerByRange()` in `mind/evaluation/power.go`

**Purpose:** Computes how threatening a squad is at each distance. Used by AI threat assessment.

**Algorithm:**

```go
CalculateSquadPowerByRange(squadID, manager, config):
  unitIDs = GetUnitIDsInSquad(squadID)
  movementRange = GetSquadMovementSpeed(squadID)

  // Collect unit data
  units = []
  attackTypeCount = map[AttackType]int

  for each unitID:
    attr = GetAttributes(unitID)
    role = GetUnitRole(unitID)
    attackRange = GetAttackRange(unitID)

    // Simplified power for threat (weapon + dex/2)
    basePower = Weapon + Dexterity/2
    roleMultiplier = GetRoleMultiplierFromConfig(role)

    units.append({
      power: basePower * roleMultiplier,
      attackRange: attackRange,
      isLeader: HasLeaderComponent(entity)
    })

    attackTypeCount[attackType]++

  // Find maximum threat range
  maxThreatRange = 0
  for each unit:
    threatRange = movementRange + unit.attackRange
    if threatRange > maxThreatRange:
      maxThreatRange = threatRange

  // Calculate power at each range
  powerByRange = map[int]float64

  for currentRange in [1, maxThreatRange]:
    rangePower = 0.0

    for each unit:
      effectiveThreatRange = movementRange + unit.attackRange

      if effectiveThreatRange >= currentRange:
        leaderBonus = unit.isLeader ? GetLeaderBonusFromConfig() : 1.0
        unitPower = unit.power * leaderBonus
        rangePower += unitPower

    powerByRange[currentRange] = rangePower

  // Apply composition bonus
  compositionBonus = GetCompositionBonusFromConfig(len(attackTypeCount))
  for each range:
    powerByRange[range] *= compositionBonus

  return powerByRange
```

**Example Output:**

```go
// Squad with move=2, melee units (range 1), ranged units (range 3)
powerByRange = {
  1: 150.0,  // All units threaten at range 1 (move 2 + attack 1 >= 1)
  2: 150.0,  // All units threaten at range 2
  3: 150.0,  // All units threaten at range 3
  4: 80.0,   // Only ranged units threaten (move 2 + range 3 >= 4)
  5: 80.0    // Only ranged units threaten
}
```

**Usage in Threat System:**

```go
// CombatThreatLayer uses this data:
meleeThreat = squadThreat.ThreatByRange[1]  // Close-range power
rangedThreat = squadThreat.ThreatByRange[maxRange]  // Long-range power
```

---

### Role Multipliers and Ability Values

**Location:** `mind/evaluation/roles.go`

**Role Multipliers (powerconfig.json):**

```json
{
  "roleMultipliers": [
    {"role": "Tank", "multiplier": 1.2},
    {"role": "DPS", "multiplier": 1.5},
    {"role": "Support", "multiplier": 1.0}
  ]
}
```

**Purpose:**
- DPS squads are inherently more threatening (1.5x)
- Tanks are moderately threatening (1.2x)
- Support provides utility but lower threat (1.0x)

**CRITICAL NOTE (from MEMORY.md):**
- `roleMultipliers` (powerconfig.json) controls POWER SCALING (single positive scalar)
- `roleBehaviors` (aiconfig.json) controls AI POSITIONING WEIGHTS (4 floats, can be negative)
- These are NOT redundant - they serve orthogonal purposes

**Ability Power Values (powerconfig.json):**

```json
{
  "abilityValues": [
    {"ability": "Rally", "power": 15.0},
    {"ability": "Heal", "power": 20.0},
    {"ability": "BattleCry", "power": 12.0},
    {"ability": "Fireball", "power": 18.0},
    {"ability": "None", "power": 0.0}
  ]
}
```

**Usage:**
- Adds to unit's utility power
- Only leaders have abilities
- Multiple abilities stack (sum of all equipped)

**Leader Bonus (powerconfig.json):**

```json
{
  "leaderBonus": 1.3
}
```

**Applied to:**
- Unit power in squad calculations
- Power-by-range calculations
- Represents command/morale boost

---

## Action Selection Algorithm

### Overview

AI action selection uses a **greedy best-first** approach:

1. Generate all valid actions (movement + attacks)
2. Score each action based on role and threat
3. Select highest-scoring action
4. Execute immediately
5. Repeat until squad exhausted

**No lookahead or planning** - each action is evaluated independently.

---

### Action Scoring Details

#### Movement Score Components

```
Total Movement Score = Base Score - Threat + Ally Bonus + Approach Bonus

Base Score:        50.0 (constant)
Threat:            Role-weighted threat (can be negative for attraction)
Ally Bonus:        allyProximity * 3.0
Approach Bonus:    distanceImprovement * roleMultiplier + rangeBonus
```

**Example Calculation (Tank):**

```go
// Tank squad considering position (10, 5)
baseScore = 50.0

// Role-weighted threat
weights = {MeleeWeight: -0.5, RangedWeight: 0.5, SupportWeight: 0.2, PositionalWeight: 0.5}
threat = (20.0 * -0.5) + (10.0 * 0.5) + (5.0 * 0.2) + (8.0 * 0.5)
       = -10.0 + 5.0 + 1.0 + 4.0
       = 0.0  // Net neutral (tank attracted to melee, avoiding ranged)

allyProximity = 2
allyBonus = 2 * 3.0 = 6.0

// Approach enemy
nearestEnemyDistance = 8
newDistance = 6
distanceImprovement = 8 - 6 = 2
approachMultiplier = 15.0  // Tank role
approachBonus = 2 * 15.0 + 0 = 30.0  // Not in attack range yet

totalScore = 50.0 - 0.0 + 6.0 + 30.0 = 86.0
```

**Example Calculation (Support):**

```go
// Support squad considering same position
baseScore = 50.0

// Role-weighted threat
weights = {MeleeWeight: 1.0, RangedWeight: 0.5, SupportWeight: -1.0, PositionalWeight: 0.5}
threat = (20.0 * 1.0) + (10.0 * 0.5) + (5.0 * -1.0) + (8.0 * 0.5)
       = 20.0 + 5.0 - 5.0 + 4.0
       = 24.0  // High threat (support avoids all danger, not attracted to wounded)

allyBonus = 6.0

// Approach enemy
approachMultiplier = -5.0  // Support role (negative = flee)
approachBonus = 2 * -5.0 + 0 = -10.0  // Penalty for closing distance

totalScore = 50.0 - 24.0 + 6.0 - 10.0 = 22.0  // Much lower than tank
```

**Key Insights:**
- Negative weights create attraction (tanks seek melee, supports seek wounded)
- Positive weights create avoidance (supports flee all danger)
- Approach bonus differentiates offensive vs defensive roles
- Ally proximity universally valued (avoid isolation)

---

#### Attack Score Components

```
Total Attack Score = Base Score + Wounded Bonus + Threat Bonus + Counter Bonus

Base Score:      100.0 (CRITICAL: higher than movement base 50.0)
Wounded Bonus:   (1.0 - targetHealth) * 20.0
Threat Bonus:    Role-based target priority
Counter Bonus:   Role matchup bonuses
```

**Example Calculation:**

```go
// DPS attacking wounded Support squad
baseScore = 100.0

targetHealth = 0.4  // 40% HP
woundedBonus = (1.0 - 0.4) * 20.0 = 12.0

targetRole = Support
threatBonus = 10.0  // Support is medium priority

myRole = DPS, targetRole = Support
counterBonus = 10.0  // DPS counters Support

totalScore = 100.0 + 12.0 + 10.0 + 10.0 = 132.0
```

**Why Base Score is 100:**
- Movement base is 50
- Attack base must be higher to ensure AI attacks when in range
- Otherwise, AI might keep repositioning instead of engaging
- Creates clear preference: attack in range > move toward enemy

---

### Decision Tree

```
For each squad with actions remaining:
│
├─ Generate Movement Actions (if !HasMoved)
│  │
│  ├─ Get valid tiles (CanMoveTo validation)
│  ├─ Score each tile:
│  │  ├─ Base score (50.0)
│  │  ├─ Role-weighted threat
│  │  ├─ Ally proximity bonus
│  │  └─ Approach enemy bonus
│  └─ Add to action list
│
├─ Generate Attack Actions (if !HasActed)
│  │
│  ├─ Get attackable enemies (range check)
│  ├─ Score each target:
│  │  ├─ Base score (100.0)
│  │  ├─ Wounded target bonus
│  │  ├─ Threat priority bonus
│  │  └─ Role counter bonus
│  └─ Add to action list
│
├─ Add Wait Action (score 0.0)
│
├─ Select Best Action (highest score)
│
└─ Execute Action
   ├─ Success: Mark threat layers dirty, continue
   └─ Failure: Break (squad done)
```

---

### Edge Cases

**No Valid Moves:**
- If all tiles blocked/occupied, movement actions list is empty
- Attack or Wait will be selected instead
- Prevents infinite loops from invalid action attempts

**No Enemies in Range:**
- If no attacks available and no moves valid, Wait is selected
- Wait marks HasMoved and HasActed (ends turn)
- Prevents squad from blocking turn progression

**Negative Scores:**
- Movement can have negative scores (high threat, no allies nearby)
- Wait has score 0.0
- Wait selected only if all movement/attack scores are negative

**Tie Scores:**
- SelectBestAction uses first action in list if tied
- Order: Movement actions, Attack actions, Wait
- Effectively biases toward first evaluated option

---

## Configuration System

### Configuration Files

**Location:** `assets/gamedata/`

1. **aiconfig.json** - AI behavior weights and thresholds
2. **powerconfig.json** - Power calculation weights and multipliers
3. **influenceconfig.json** - (Not covered in this doc)
4. **overworldconfig.json** - (Not covered in this doc)

---

### AI Configuration (aiconfig.json)

**Structure:**

```json
{
  "threatCalculation": {
    "flankingThreatRangeBonus": 3,
    "isolationThreshold": 3,
    "retreatSafeThreatThreshold": 10
  },
  "roleBehaviors": [
    {
      "role": "Tank",
      "meleeWeight": -0.5,
      "supportWeight": 0.2
    },
    {
      "role": "DPS",
      "meleeWeight": 0.7,
      "supportWeight": 0.1
    },
    {
      "role": "Support",
      "meleeWeight": 1.0,
      "supportWeight": -1.0
    }
  ],
  "supportLayer": {
    "healRadius": 3
  },
  "sharedRangedWeight": 0.5,
  "sharedPositionalWeight": 0.5
}
```

**Parameter Details:**

| Parameter | Purpose | Range | Default |
|-----------|---------|-------|---------|
| `flankingThreatRangeBonus` | Extra range for flanking detection | 1-5 | 3 |
| `isolationThreshold` | Distance before isolation risk | 1-5 | 3 |
| `retreatSafeThreatThreshold` | Threat level for "safe" retreat | 5-20 | 10 |
| `healRadius` | Support value paint radius | 2-5 | 3 |
| `sharedRangedWeight` | Ranged threat weight (all roles) | 0.0-2.0 | 0.5 |
| `sharedPositionalWeight` | Positional risk weight (all roles) | 0.0-2.0 | 0.5 |

**Role Behaviors:**

| Role | Melee Weight | Support Weight | Interpretation |
|------|--------------|----------------|----------------|
| Tank | -0.5 | 0.2 | Attracted to melee, stay near support |
| DPS | 0.7 | 0.1 | Avoid melee, low support priority |
| Support | 1.0 | -1.0 | Flee all danger, seek wounded allies |

**Weight Semantics:**
- **Negative weight** = Attraction (lower threat score = better position)
- **Positive weight** = Avoidance (higher threat score = worse position)
- **Zero weight** = Ignore this layer

**Shared vs Role-Specific:**
- `sharedRangedWeight` and `sharedPositionalWeight` apply to ALL roles
- `meleeWeight` and `supportWeight` are role-specific
- Allows designers to tune role differentiation without affecting shared behaviors

---

### Power Configuration (powerconfig.json)

**Structure:**

```json
{
  "profiles": [
    {
      "name": "Balanced",
      "offensiveWeight": 0.4,
      "defensiveWeight": 0.4,
      "utilityWeight": 0.2,
      "healthPenalty": 2.0
    }
  ],
  "roleMultipliers": [
    {"role": "Tank", "multiplier": 1.2},
    {"role": "DPS", "multiplier": 1.5},
    {"role": "Support", "multiplier": 1.0}
  ],
  "abilityValues": [
    {"ability": "Rally", "power": 15.0},
    {"ability": "Heal", "power": 20.0},
    {"ability": "BattleCry", "power": 12.0},
    {"ability": "Fireball", "power": 18.0},
    {"ability": "None", "power": 0.0}
  ],
  "compositionBonuses": [
    {"uniqueTypes": 1, "bonus": 0.8},
    {"uniqueTypes": 2, "bonus": 1.1},
    {"uniqueTypes": 3, "bonus": 1.2},
    {"uniqueTypes": 4, "bonus": 1.3}
  ],
  "leaderBonus": 1.3
}
```

**Profile Weights:**

| Parameter | Purpose | Recommended Range | Notes |
|-----------|---------|-------------------|-------|
| `offensiveWeight` | Damage output importance | 0.3-0.5 | Should sum to 1.0 |
| `defensiveWeight` | Survivability importance | 0.3-0.5 | with other weights |
| `utilityWeight` | Support/role importance | 0.1-0.3 | |
| `healthPenalty` | Wounded squad penalty | 1.5-3.0 | Higher = steeper penalty |

**Role Multipliers:**

| Role | Multiplier | Rationale |
|------|------------|-----------|
| Tank | 1.2 | Moderate threat (damage soak) |
| DPS | 1.5 | High threat (damage dealer) |
| Support | 1.0 | Baseline (utility focus) |

**CRITICAL: Not Redundant with roleBehaviors**
- `roleMultipliers` (powerconfig.json): Combat power scaling (AI sees DPS as 1.5x more dangerous)
- `roleBehaviors` (aiconfig.json): Positioning preferences (DPS avoids melee, Support seeks wounded)
- Both needed for complete AI behavior

**Ability Power Values:**

| Ability | Power | Rationale |
|---------|-------|-----------|
| Heal | 20.0 | Highest (sustains squads) |
| Fireball | 18.0 | High (damage burst) |
| Rally | 15.0 | Medium (buff) |
| BattleCry | 12.0 | Medium (debuff) |
| None | 0.0 | No ability |

**Composition Bonuses:**

| Unique Types | Bonus | Interpretation |
|--------------|-------|----------------|
| 1 | 0.8 | Mono-type penalty (-20%) |
| 2 | 1.1 | Dual-type bonus (+10%) |
| 3 | 1.2 | Triple-type bonus (+20%) |
| 4 | 1.3 | Quad-type bonus (+30%) |

**Leader Bonus:** 1.3 (leaders provide +30% power multiplier)

---

### Accessor Pattern

**Why Data-Driven?**
- Eliminates hardcoded constants
- Enables designer tuning without code changes
- Centralizes balance parameters
- Supports A/B testing and playtesting iterations

**Implementation Pattern:**

```go
// Configuration accessor with fallback
func GetParameterFromConfig() ValueType {
  // Try to load from JSON template
  if templates.ConfigTemplate.Parameter != nil:
    return templates.ConfigTemplate.Parameter

  // Fallback to hardcoded default
  return defaultValue
}
```

**Examples:**

```go
// AI behavior parameter
func GetIsolationThreshold() int {
  if templates.AIConfigTemplate.ThreatCalculation.IsolationThreshold > 0:
    return templates.AIConfigTemplate.ThreatCalculation.IsolationThreshold
  return 3  // Default
}

// Power calculation parameter
func GetRoleMultiplierFromConfig(role UnitRole) float64 {
  roleStr := role.String()
  for _, rm := range templates.PowerConfigTemplate.RoleMultipliers:
    if rm.Role == roleStr:
      return rm.Multiplier

  // Fallback defaults
  switch role:
    case RoleTank:    return 1.2
    case RoleDPS:     return 1.5
    case RoleSupport: return 1.0
    default:          return 1.0
}
```

**Benefits:**
- JSON missing/malformed? Gracefully falls back to defaults
- Designer can experiment without code knowledge
- Version control tracks balance changes separately from logic
- Multiple profiles supported (future extensibility)

---

## Data Flow and Dependencies

### System Initialization

```
Game Initialization
│
├─ Load Config Templates (aiconfig.json, powerconfig.json)
│
├─ Create EntityManager
│
├─ Create CombatQueryCache
│
├─ Create FactionThreatLevelManager
│  └─ For each faction:
│     └─ Create FactionThreatLevel
│        └─ For each squad:
│           └─ Create SquadThreatLevel (with ThreatByRange)
│
└─ Create AIController
   ├─ Dependencies: EntityManager, TurnManager, MovementSystem, CombatActionSystem
   │
   └─ Create CompositeThreatEvaluator (per faction, lazy)
      ├─ CombatThreatLayer
      ├─ SupportValueLayer
      └─ PositionalRiskLayer
```

---

### AI Turn Execution Flow

```
GUI Calls AIController.DecideFactionTurn(factionID)
│
├─ 1. Clear attack queue
│
├─ 2. Update threat layers
│  ├─ FactionThreatLevelManager.UpdateAllFactions()
│  │  └─ For each squad: CalculateThreatLevels()
│  │     └─ ThreatByRange = CalculateSquadPowerByRange()
│  │
│  └─ CompositeThreatEvaluator.Update(currentRound)
│     ├─ CombatThreatLayer.Compute()
│     │  ├─ Read ThreatByRange[1] for melee
│     │  ├─ Read ThreatByRange[maxRange] for ranged
│     │  └─ Paint threat maps
│     │
│     ├─ SupportValueLayer.Compute()
│     │  ├─ Calculate heal priorities
│     │  └─ Paint support value maps
│     │
│     └─ PositionalRiskLayer.Compute()
│        ├─ Calculate flanking risk
│        ├─ Calculate isolation risk
│        ├─ Calculate engagement pressure
│        └─ Calculate retreat quality
│
├─ 3. For each squad in faction:
│  │
│  └─ While squad has actions remaining:
│     │
│     ├─ Create ActionContext
│     │  ├─ Get action state (HasMoved, HasActed)
│     │  ├─ Get threat evaluator for faction
│     │  ├─ Get squad role (cached)
│     │  ├─ Get current position (cached)
│     │  └─ Get squad health (cached)
│     │
│     ├─ Create ActionEvaluator
│     │
│     ├─ EvaluateAllActions()
│     │  ├─ Generate movement actions (if !HasMoved)
│     │  │  ├─ Get valid tiles (CanMoveTo validation)
│     │  │  └─ Score each tile
│     │  │     ├─ Base score
│     │  │     ├─ Role-weighted threat
│     │  │     ├─ Ally proximity bonus
│     │  │     └─ Approach enemy bonus
│     │  │
│     │  ├─ Generate attack actions (if !HasActed)
│     │  │  ├─ Get attackable enemies
│     │  │  └─ Score each target
│     │  │     ├─ Base score
│     │  │     ├─ Wounded bonus
│     │  │     ├─ Threat priority bonus
│     │  │     └─ Role counter bonus
│     │  │
│     │  └─ Add Wait action (score 0.0)
│     │
│     ├─ SelectBestAction() (highest score)
│     │
│     ├─ Execute Action
│     │  ├─ MoveAction: MovementSystem.MoveSquad()
│     │  ├─ AttackAction: CombatActionSystem.ExecuteAttackAction()
│     │  │  └─ Queue attack for animation
│     │  └─ WaitAction: Mark HasMoved and HasActed
│     │
│     └─ Mark threat layers dirty (positions changed)
│        ├─ CombatThreatLayer.MarkDirty()
│        ├─ SupportValueLayer.MarkDirty()
│        └─ PositionalRiskLayer.MarkDirty()
│
└─ 4. Return queued attacks to GUI for animation
```

---

### Encounter Generation Flow

```
Player Triggers Encounter
│
├─ GenerateEncounterSpec()
│  │
│  ├─ Get player deployed squads
│  │
│  ├─ Calculate total player power
│  │  └─ For each squad: CalculateSquadPower()
│  │
│  ├─ Calculate average squad power
│  │
│  ├─ Determine difficulty modifier (based on encounter level)
│  │  ├─ Power multiplier (e.g., 1.2x for medium difficulty)
│  │  ├─ Squad count (e.g., 2-3 squads)
│  │  ├─ Min/Max target power bounds
│  │  └─ Role distribution weights
│  │
│  ├─ Calculate target enemy squad power
│  │  └─ avgPlayerPower * difficultyMultiplier
│  │
│  ├─ Generate enemy squads
│  │  └─ For each enemy squad (count from difficulty):
│  │     ├─ Select role (weighted random)
│  │     ├─ Build squad from templates
│  │     │  └─ Add units until power target reached
│  │     │     └─ EstimateUnitPowerFromTemplate()
│  │     │
│  │     └─ Assign spawn position
│  │
│  └─ Return EncounterSpec
│     ├─ PlayerSquadIDs
│     ├─ EnemySquads (templates)
│     ├─ Difficulty
│     └─ EncounterType
│
└─ SetupCombatFromEncounter()
   ├─ Create enemy faction
   ├─ Instantiate enemy squads from templates
   ├─ Deploy squads to map
   └─ Initialize combat turn manager
```

**Key Insight:**
- Uses same `CalculateSquadPower()` as AI threat system
- Ensures balanced encounters (enemy power ≈ player power * modifier)
- Enemy squad composition uses power estimation before spawning
- Templates converted to ECS entities only after validation

---

## Performance Considerations

### Threat Layer Computation

**Complexity:** O(factions × squads × mapRadius²)

**Optimization Techniques:**

1. **Dirty Flag Caching**
   - Layers only recompute when marked dirty
   - Prevents redundant calculations within same round
   - Invalidated after each AI action (positions change)

2. **Map Reuse**
   - Threat maps use `clear()` instead of reallocating
   - Reduces GC pressure during AI turns
   - Go's `clear()` is optimized for map zeroing

3. **Lazy Evaluator Creation**
   - CompositeThreatEvaluator created per-faction on-demand
   - Not all factions may execute AI (e.g., player faction)
   - Stored in map for reuse

4. **Combat Query Cache**
   - Pre-indexed faction-to-squad mappings
   - Avoids repeated ECS queries
   - Updated incrementally during combat

**Bottlenecks:**

- **PaintThreatToMap()**: O(radius²) per squad
  - Mitigated by typical small map sizes (30×30)
  - Could use spatial partitioning for massive maps

- **GetRoleWeightedThreat()**: Called per movement candidate
  - Typical AI turn: 3 squads × 10 tiles × 4 layer queries = 120 queries
  - Layers are precomputed, so query is O(1) map lookup

---

### Power Calculation

**Complexity:** O(units) per squad

**Optimization Techniques:**

1. **Shared Calculation**
   - Single implementation for AI and encounter generation
   - Eliminates code duplication and maintenance
   - Ensures consistent threat/power assessment

2. **Component Batching**
   - UnitCombatData bundles all components in one query
   - Reduces component access overhead
   - Pre-calculates derived values (role multiplier)

3. **Config Caching**
   - PowerConfig loaded once per profile
   - Role/ability lookups iterate small arrays (<10 entries)
   - Could use maps for O(1) lookup if needed

**Bottlenecks:**

- **CalculateSquadPowerByRange()**: O(units × ranges)
  - Called once per squad during threat update
  - Typical cost: 4 units × 5 ranges = 20 iterations
  - Results cached in ThreatByRange map

---

### ECS Query Patterns

**Best Practices:**

1. **Use Combat Cache for Repeated Queries**
   ```go
   // Avoid repeated ECS queries
   squadIDs := cache.GetSquadsForFaction(factionID)  // Cached
   ```

2. **Component Access by EntityID**
   ```go
   // Preferred: Direct component access
   data := common.GetComponentTypeByID[*SquadData](manager, squadID, SquadComponent)
   ```

3. **Avoid Entity Pointer Storage**
   ```go
   // Never store entity pointers (violates ECS patterns)
   // Always use EntityID and query on-demand
   ```

4. **Batch Component Reads**
   ```go
   // Get all components in one helper
   combatData := GetUnitCombatData(unitID, manager)
   ```

---

### Memory Footprint

**Per-Faction Threat Data:**

```
CompositeThreatEvaluator (per faction):
  CombatThreatLayer:
    meleeThreatByPos:    ~30×30 float64 = 7.2KB
    rangedPressureByPos: ~30×30 float64 = 7.2KB
    lineOfFireZones:     ~5 squads × 50 positions × 16 bytes = 4KB

  SupportValueLayer:
    supportValuePos:     ~30×30 float64 = 7.2KB
    allyProximity:       ~30×30 int = 3.6KB

  PositionalRiskLayer:
    4 maps × ~30×30 float64 = 28.8KB

Total per faction: ~58KB
```

**Scaling:**
- 3 factions: ~174KB
- Negligible compared to ECS entity data
- Could use sparse maps if memory becomes issue

---

## Extension Points

### Adding New Threat Layers

**Steps:**

1. **Define Layer Struct** in `mind/behavior/`
   ```go
   type NewThreatLayer struct {
     *ThreatLayerBase
     customData map[LogicalPosition]float64
   }
   ```

2. **Implement Compute()**
   ```go
   func (ntl *NewThreatLayer) Compute() {
     clear(ntl.customData)

     // Compute threat values
     // ...

     ntl.markClean(0)
   }
   ```

3. **Add to CompositeThreatEvaluator**
   ```go
   type CompositeThreatEvaluator struct {
     // Existing layers
     combatThreat *CombatThreatLayer

     // New layer
     newThreat *NewThreatLayer
   }
   ```

4. **Update GetRoleWeightedThreat()**
   ```go
   func (cte *CompositeThreatEvaluator) GetRoleWeightedThreat(...) {
     // Existing layers
     meleeThreat := cte.combatThreat.GetMeleeThreatAt(pos)

     // New layer
     newThreat := cte.newThreat.GetNewThreatAt(pos)

     totalThreat = ... + newThreat * weights.NewWeight
   }
   ```

5. **Add Weight to RoleThreatWeights**
   ```go
   type RoleThreatWeights struct {
     MeleeWeight      float64
     RangedWeight     float64
     SupportWeight    float64
     PositionalWeight float64
     NewWeight        float64  // New weight
   }
   ```

6. **Update aiconfig.json**
   ```json
   {
     "roleBehaviors": [
       {
         "role": "Tank",
         "meleeWeight": -0.5,
         "supportWeight": 0.2,
         "newWeight": 0.3
       }
     ]
   }
   ```

---

### Adding New Action Types

**Steps:**

1. **Define Action Struct** in `mind/ai/action_evaluator.go`
   ```go
   type NewAction struct {
     squadID ecs.EntityID
     // Action-specific data
   }
   ```

2. **Implement SquadAction Interface**
   ```go
   func (na *NewAction) Execute(manager, movementSystem, combatActSystem, cache) bool {
     // Execute action logic
     return success
   }

   func (na *NewAction) GetDescription() string {
     return "New action description"
   }
   ```

3. **Add Generation in ActionEvaluator**
   ```go
   func (ae *ActionEvaluator) EvaluateAllActions() []ScoredAction {
     // Existing actions
     actions = append(actions, ae.evaluateMovement()...)
     actions = append(actions, ae.evaluateAttacks()...)

     // New actions
     actions = append(actions, ae.evaluateNewActions()...)

     return actions
   }
   ```

4. **Implement Scoring Function**
   ```go
   func (ae *ActionEvaluator) evaluateNewActions() []ScoredAction {
     var actions []ScoredAction

     // Generate candidates
     // Score each candidate
     // Append to actions

     return actions
   }
   ```

5. **Update ActionContext if Needed**
   ```go
   type ActionContext struct {
     // Existing fields

     // New data needed for action evaluation
     NewContextData DataType
   }
   ```

---

### Adding New Power Factors

**Steps:**

1. **Define Component** (if not existing)
   ```go
   type NewPowerData struct {
     PowerValue float64
   }
   var NewPowerComponent *ecs.Component
   ```

2. **Update CalculateUtilityPower()**
   ```go
   func CalculateUtilityPower(entity, attr, roleData, config) float64 {
     roleValue := calculateRoleValue(roleData)
     abilityValue := calculateAbilityValue(entity)
     coverValue := calculateCoverValue(entity)

     // New power factor
     newValue := calculateNewPowerFactor(entity)

     return roleValue + abilityValue + coverValue + newValue
   }
   ```

3. **Implement Calculation**
   ```go
   func calculateNewPowerFactor(entity *ecs.Entity) float64 {
     data := GetComponentType[*NewPowerData](entity, NewPowerComponent)
     if data == nil:
       return 0.0

     return data.PowerValue * scalingConstant
   }
   ```

4. **Update powerconfig.json (if configurable)**
   ```json
   {
     "newPowerFactorScaling": 25.0
   }
   ```

5. **Update EstimateUnitPowerFromTemplate()**
   ```go
   // If new factor applies to templates
   func EstimateUnitPowerFromTemplate(unit UnitTemplate, config) {
     // Existing calculations

     // New factor (if applicable to templates)
     newValue := unit.NewPowerValue * scalingConstant

     utilityPower += newValue
   }
   ```

---

### Tuning AI Behavior

**Common Tuning Tasks:**

1. **Make Role More Aggressive**
   - Increase approach multiplier in `scoreApproachEnemy()`
   - Decrease `meleeWeight` (more negative = more attraction)
   - Increase `attackBaseScore` relative to `movementBaseScore`

2. **Improve Survivability**
   - Increase `rangedWeight` and `positionalWeight`
   - Increase isolation risk weight
   - Decrease approach multiplier

3. **Focus Fire Better**
   - Increase wounded target bonus
   - Add persistence (track previous target)
   - Implement squad coordination (multiple AIs targeting same enemy)

4. **Improve Positioning**
   - Increase ally proximity bonus
   - Increase flanking risk weight
   - Add terrain awareness layer

**Configuration-Only Tuning:**

| Goal | Configuration Change |
|------|---------------------|
| Tanks more aggressive | Decrease Tank.meleeWeight (more negative) |
| Support stays farther back | Increase Support.meleeWeight |
| All units stick together | Increase isolationThreshold |
| Fewer flanking maneuvers | Decrease flankingThreatRangeBonus |
| DPS prioritized by AI | Increase DPS roleMultiplier |
| Healers more valuable | Increase Heal abilityValue |

---

## Troubleshooting

### Common Issues

**AI Not Moving:**

**Symptoms:** Squads select Wait action every turn.

**Possible Causes:**
- All movement tiles fail `CanMoveTo()` validation
- Movement scores all negative (very high threat)
- ActionState not resetting (HasMoved stuck true)

**Debug Steps:**
1. Log output of `getValidMovementTiles()` - should be non-empty
2. Log movement scores - should have positive values
3. Check ActionState after turn reset
4. Verify threat layers computing correctly

---

**AI Not Attacking:**

**Symptoms:** AI moves but never attacks even when in range.

**Possible Causes:**
- Attack base score too low (< movement base score)
- No enemies in range (distance check failing)
- ActionState.HasActed stuck true
- Attack action execution failing

**Debug Steps:**
1. Verify attack base score > movement base score (100 > 50)
2. Log `getAttackableTargets()` output
3. Check range calculation (squad distance vs attack range)
4. Add logging in `AttackAction.Execute()`

---

**AI Suicidal Behavior:**

**Symptoms:** AI charges into overwhelm ing threat, ignores danger.

**Possible Causes:**
- Approach bonus too high (overpowers threat avoidance)
- Threat layers not computing correctly
- Role weights incorrect (should be positive for avoidance)
- Attack score too high (AI always prefers attacking over survival)

**Debug Steps:**
1. Log role weights - positive should avoid, negative should attract
2. Visualize threat layers (check if threat values reasonable)
3. Reduce approach multipliers
4. Increase threat layer weights in role configuration

---

**AI Ignores Wounded Allies:**

**Symptoms:** Support squads don't move toward damaged units.

**Possible Causes:**
- SupportValueLayer not computing correctly
- Support.supportWeight not negative (should attract)
- Heal priority calculation wrong
- Support value paint radius too small

**Debug Steps:**
1. Check SupportValueLayer.Compute() executes
2. Verify Support.supportWeight is negative (-1.0)
3. Log healPriority values (should be 1.0 - healthPercent)
4. Increase healRadius in aiconfig.json

---

**Threat Layers Not Updating:**

**Symptoms:** AI makes decisions based on stale positions.

**Possible Causes:**
- Layers not marked dirty after actions
- Dirty flag not checked in Update()
- BaseThreatMgr not updating ThreatByRange

**Debug Steps:**
1. Verify `MarkDirty()` called after each action execution
2. Check dirty flag in Update() - should skip if clean
3. Confirm FactionThreatLevelManager.UpdateAllFactions() called
4. Log ThreatByRange values (should change as squads move)

---

## Appendix: File Reference

### Core AI Files

| File | Purpose | Key Functions |
|------|---------|---------------|
| `mind/ai/ai_controller.go` | Turn orchestration | `DecideFactionTurn()`, `NewActionContext()` |
| `mind/ai/action_evaluator.go` | Action generation and scoring | `EvaluateAllActions()`, `scoreMovementPosition()`, `scoreAttackTarget()` |

### Behavior/Threat Files

| File | Purpose | Key Functions |
|------|---------|---------------|
| `mind/behavior/threat_composite.go` | Layer composition | `GetRoleWeightedThreat()`, `Update()` |
| `mind/behavior/threat_combat.go` | Melee/ranged threat | `Compute()`, `GetMeleeThreatAt()`, `GetRangedPressureAt()` |
| `mind/behavior/threat_support.go` | Support positioning | `Compute()`, `GetSupportValueAt()` |
| `mind/behavior/threat_positional.go` | Tactical risks | `Compute()`, `GetTotalRiskAt()` |
| `mind/behavior/threat_layers.go` | Base layer utilities | `ThreatLayerBase`, `getEnemyFactions()` |
| `mind/behavior/threat_constants.go` | Config accessors | `GetRoleBehaviorWeights()`, `GetIsolationThreshold()` |
| `mind/behavior/threat_painting.go` | Spatial threat painting | `PaintThreatToMap()`, `LinearFalloff`, `NoFalloff` |
| `mind/behavior/threat_queries.go` | Unit data queries | `GetUnitCombatData()`, `hasUnitsWithAttackType()` |
| `mind/behavior/dangerlevel.go` | Base threat tracking | `FactionThreatLevelManager`, `SquadThreatLevel` |

### Evaluation Files

| File | Purpose | Key Functions |
|------|---------|---------------|
| `mind/evaluation/power.go` | Power calculation | `CalculateSquadPower()`, `CalculateSquadPowerByRange()` |
| `mind/evaluation/power_config.go` | Config loading | `GetPowerConfigByProfile()` |
| `mind/evaluation/roles.go` | Role/ability config | `GetRoleMultiplierFromConfig()`, `GetAbilityPowerValue()` |
| `mind/evaluation/cache.go` | Dirty flag system | `DirtyCache`, `MarkDirty()` |

### Encounter Files

| File | Purpose | Key Functions |
|------|---------|---------------|
| `mind/encounter/encounter_generator.go` | Enemy creation | `GenerateEncounterSpec()` |
| `mind/encounter/encounter_setup.go` | Combat initialization | `SetupBalancedEncounter()` |
| `mind/encounter/encounter_config.go` | Difficulty config | `getDifficultyModifier()` |

### Combat System Files

| File | Purpose | Key Functions |
|------|---------|---------------|
| `tactical/combat/turnmanager.go` | Turn management | `InitializeCombat()`, `EndTurn()` |
| `tactical/combat/combatactionsystem.go` | Action execution | `ExecuteAttackAction()` |
| `tactical/combat/combatcomponents.go` | ECS components | `ActionStateData`, `TurnStateData` |
| `tactical/combat/combatqueries.go` | Combat queries | `GetSquadFaction()`, `GetSquadMapPosition()` |

---

**End of Document**

For questions or clarifications, consult the source code or contact the development team.
