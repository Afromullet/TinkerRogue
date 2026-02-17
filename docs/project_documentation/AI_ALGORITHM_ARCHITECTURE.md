# AI Algorithm Architecture

**Last Updated:** 2026-02-17

A technical reference for TinkerRogue's AI decision-making, power evaluation, action selection, and configuration systems.

---

## Related Documents

- [Behavior & Threat Layers](BEHAVIOR_THREAT_LAYERS.md) - Threat layer subsystems, spatial analysis, visualization, and threat layer extension points
- [Encounter System](ENCOUNTER_SYSTEM.md) - Encounter generation, lifecycle, and rewards

---

## Table of Contents

1. [Executive Summary](#executive-summary)
2. [Architecture Overview](#architecture-overview)
3. [Core AI Systems](#core-ai-systems)
4. [Power Evaluation System](#power-evaluation-system)
5. [Action Selection Algorithm](#action-selection-algorithm)
6. [Configuration System](#configuration-system)
7. [Data Flow and Dependencies](#data-flow-and-dependencies)
8. [Performance Considerations](#performance-considerations)
9. [Extension Points](#extension-points)
10. [Troubleshooting](#troubleshooting)
11. [Appendix: File Reference](#appendix-file-reference)

---

## Executive Summary

TinkerRogue's AI uses a **layered threat assessment** system combined with **role-based behavior weights** to create tactically-aware computer opponents. The architecture separates concerns into distinct subsystems:

- **AI Controller** (`mind/ai/`) - Orchestrates turn execution and action selection
- **Threat Evaluation** (`mind/behavior/`) - Multi-layered spatial threat analysis (see [Behavior & Threat Layers](BEHAVIOR_THREAT_LAYERS.md))
- **Power Calculation** (`mind/evaluation/`) - Unified combat power assessment
- **Encounter Generation** (`mind/encounter/`) - Dynamic enemy creation based on power budgets (see [Encounter System](ENCOUNTER_SYSTEM.md))

**Key Design Principles:**

1. **Data-Driven Configuration** - All weights, thresholds, and multipliers loaded from JSON
2. **Separation of Concerns** - Power calculation shared between AI and encounter generation
3. **Layer Composition** - Multiple threat layers combined via role-specific weights
4. **Cache-Friendly** - Dirty flag invalidation prevents redundant recomputation
5. **ECS-First** - Pure component queries, no entity pointer caching
6. **Difficulty Scaling** - Global difficulty settings overlay all AI and encounter parameters at runtime

---

## Architecture Overview

### System Boundaries

```
+-------------------------------------------------------------+
|                    GUI / Game Loop                           |
+------------------------+------------------------------------+
                         |
                         v
+-------------------------------------------------------------+
|                   AIController                               |
|  * DecideFactionTurn()                                       |
|  * Attack queue management                                   |
|  * Turn orchestration                                        |
+----------+---------------------+------------------+---------+
           |                     |                  |
           v                     v                  v
+------------------+  +------------------+  +--------------+
| ActionEvaluator  |  | CompositeThreat  |  | TurnManager  |
|                  |  |   Evaluator      |  |              |
| * Movement score |  |                  |  | * Initiative |
| * Attack score   |  | * Role weights   |  | * Round mgmt |
| * Fallback wait  |  | * Layer queries  |  |              |
+----------+-------+  +--------+---------+  +--------------+
           |                   |
           v                   v
+-------------------------------------------------------------+
|              Threat Layer Subsystems                         |
|  (see BEHAVIOR_THREAT_LAYERS.md)                            |
+------------------------+------------------------------------+
                         |
                         v
+-------------------------------------------------------------+
|           FactionThreatLevelManager                          |
|  * SquadThreatLevel (ThreatByRange map)                     |
|  * Uses shared power calculation                            |
+------------------------+------------------------------------+
                         |
                         v
+-------------------------------------------------------------+
|              Power Evaluation (shared)                       |
|  * CalculateSquadPower()                                     |
|  * CalculateSquadPowerByRange()                              |
|  * Used by AI threat + encounter generation                  |
+-------------------------------------------------------------+
```

### Key Dependencies

- **EntityManager**: ECS world access for all systems
- **CombatQueryCache**: Optimized faction/squad queries using ECS Views
- **CoordinateManager**: Spatial indexing and distance calculations
- **GlobalPositionSystem**: O(1) entity lookup by position
- **Config Templates**: JSON-loaded weights and parameters
- **GlobalDifficulty**: Runtime difficulty overlay applied to AI and encounter parameters

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
  2. Update threat layers at start of AI turn (once per turn, not per action)
  3. Get all alive squads in faction
  4. For each squad:
     a. While squad has actions remaining:
        - Create ActionContext (current state + threat evaluator)
        - Generate and score all possible actions
        - Select best action (highest score)
        - Execute action
        - Mark ALL faction evaluators dirty (positions may have changed)
  5. Return true if any actions executed
```

**Key Features:**
- **Exhaustive Action Processing**: Each squad uses ALL available actions before moving to next squad
- **Start-of-Turn Threat Update**: Layers updated once per faction turn (not per action) for performance
- **Post-Action Dirty Marking**: Layers marked dirty after each action, but NOT recomputed until next turn start
- **Attack Queueing**: Stores attacks for GUI animation after AI turn completes
- **Faction-Scoped Evaluators**: Each faction maintains separate threat evaluator, created lazily on first use

**Important Performance Note:**
Subsequent squads in the same turn use threat data from the start of the turn, not re-evaluated after each move. This is intentional - recomputing after every action would be expensive and the gradual nature of position changes makes slight staleness acceptable.

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

    // Threat evaluation
    ThreatEval  *CompositeThreatEvaluator  // Role-weighted threat queries

    // Systems access
    Manager        *EntityManager
    MovementSystem *CombatMovementSystem   // For validating tiles
    AIController   *AIController           // Reference for attack queueing

    // Cached squad info
    SquadRole  UnitRole
    CurrentPos LogicalPosition
    // Note: SquadHealth is NOT cached here; accessed on-demand via squads.GetSquadHealthPercent()
}
```

**Why Context Object?**
- Eliminates repetitive parameter passing
- Guarantees consistent data snapshot for action evaluation
- Pre-caches expensive queries (role, position)
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
    default: multiplier = 5.0    // Slight approach preference

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

## Power Evaluation System

**Location:** `mind/evaluation/`

**Purpose:** Unified combat power assessment shared by AI threat evaluation and encounter generation.

### Architecture

```
Power Calculation (Shared)
|
+- Unit Power
|  +- Offensive Power (damage output)
|  +- Defensive Power (survivability)
|  +- Utility Power (role, abilities, cover)
|
+- Squad Power
|  +- Sum unit powers
|  +- Composition bonus (attack type diversity)
|  +- Health multiplier (wounded penalty)
|
+- Squad Power By Range
   +- Map of range -> power (for threat assessment)
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

  // Fallback to defaults (DefaultOffensiveWeight=0.4, DefaultDefensiveWeight=0.4, DefaultUtilityWeight=0.2)
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
  // Sum all utility components (no sub-weights, just add together)
  return calculateRoleValue(roleData) + calculateAbilityValue(entity) + calculateCoverValue(entity)

calculateRoleValue(roleData):
  roleMultiplier = GetRoleMultiplierFromConfig(role)  // From powerconfig.json
  return roleMultiplier * RoleScalingFactor  // 10.0

calculateAbilityValue(entity):
  if !IsLeader(entity):
    return 0.0

  // Iterate AbilitySlotData.Slots (IsEquipped = true)
  totalPower = 0.0
  for each equipped ability slot:
    totalPower += GetAbilityPowerValue(slot.AbilityType)  // From powerconfig.json

  return totalPower

calculateCoverValue(entity):
  coverData = GetCoverData(entity)
  if coverData == nil:
    return 0.0

  return coverData.CoverValue * CoverScalingFactor * CoverBeneficiaryMultiplier
  // CoverScalingFactor=100.0, CoverBeneficiaryMultiplier=2.5
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
    attackTypes.add(unit.AttackType)  // Via TargetRowData.AttackType

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
- Penalizes mono-composition squads (0.8x)
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
    // Get attack range (defaults to 1 if AttackRangeComponent missing)
    attackRange = rangeData.Range  // From AttackRangeComponent

    // Track attack types for composition bonus
    attackTypeCount[attackType]++

    // Use FULL unit power calculation (same as CalculateSquadPower)
    unitPower = calculateUnitPower(unitID, manager, config)

    units.append({power: unitPower, attackRange: attackRange})

  // Find maximum threat range
  maxThreatRange = max(movementRange + unit.attackRange) for all units

  // Calculate power at each range
  powerByRange = map[int]float64

  for currentRange in [1, maxThreatRange]:
    rangePower = 0.0

    for each unit:
      effectiveThreatRange = movementRange + unit.attackRange

      if effectiveThreatRange >= currentRange:
        rangePower += unit.power  // Full unit power already includes role multiplier

    powerByRange[currentRange] = rangePower

  // Apply composition bonus
  compositionBonus = GetCompositionBonusFromConfig(len(attackTypeCount))
  for each range:
    powerByRange[range] *= compositionBonus

  return powerByRange
```

**Important Implementation Note:**
This function uses the **full** `calculateUnitPower()` calculation (including offensive, defensive, and utility components). Earlier documentation described a simplified formula (weapon + dex/2), which was a prior implementation. The current code uses the same unified power calculation as `CalculateSquadPower()`.

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

**Scaling Constants:**

```go
const (
    RoleScalingFactor          = 10.0   // Base multiplier for role value
    CoverScalingFactor         = 100.0  // Scale cover value (0.0-0.5) to comparable range (0-50)
    CoverBeneficiaryMultiplier = 2.5    // Average units protected per cover provider
)
```

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
- Only leaders have abilities (checked via `LeaderComponent`)
- Multiple abilities stack via `AbilitySlotData.Slots` (sum of all equipped slots)

**Leader Bonus (powerconfig.json):**

```json
{
  "leaderBonus": 1.3
}
```

**Note:** The `leaderBonus` field exists in `powerconfig.json` and is accessible via `GetLeaderBonusFromConfig()` in `roles.go`. However, the current `CalculateSquadPowerByRange()` implementation uses `calculateUnitPower()` which bundles leader ability values into utility power rather than applying a separate multiplier. The `leaderBonus` value may be used in future systems or in template-based estimation (`EstimateUnitPowerFromTemplate`).

---

### DirtyCache

**Location:** `mind/evaluation/cache.go`

**Purpose:** Lazy evaluation with round-based invalidation. Embedded in all threat layers and `ThreatVisualizer`.

**Data Structure:**

```go
type DirtyCache struct {
  lastUpdateRound int
  isDirty         bool
  isInitialized   bool
}
```

**API:**

```go
NewDirtyCache() *DirtyCache  // Created in dirty state; first access triggers computation

IsValid(currentRound int) bool  // True if initialized, not dirty, and round matches
MarkDirty()                     // Invalidate; forces recomputation on next access
MarkClean(currentRound int)     // Mark as valid for given round
IsDirty() bool                  // Whether recomputation is needed
IsInitialized() bool            // Whether computed at least once
GetLastUpdateRound() int        // Round number of last update
```

**Usage Pattern:**

```go
// In a layer's Compute() method:
func (layer *SomeLayer) Compute(currentRound int) {
    // ... compute data ...
    layer.markClean(currentRound)  // Calls DirtyCache.MarkClean internally
}

// In CompositeThreatEvaluator.Update():
if !cte.isDirty && cte.lastUpdateRound == currentRound {
    return  // Use IsValid() pattern
}
```

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
|
+- Generate Movement Actions (if !HasMoved)
|  |
|  +- Get valid tiles (CanMoveTo validation)
|  +- Score each tile:
|  |  +- Base score (50.0)
|  |  +- Role-weighted threat
|  |  +- Ally proximity bonus
|  |  +- Approach enemy bonus
|  +- Add to action list
|
+- Generate Attack Actions (if !HasActed)
|  |
|  +- Get attackable enemies (range check)
|  +- Score each target:
|  |  +- Base score (100.0)
|  |  +- Wounded target bonus
|  |  +- Threat priority bonus
|  |  +- Role counter bonus
|  +- Add to action list
|
+- Add Wait Action (score 0.0)
|
+- Select Best Action (highest score)
|
+- Execute Action
   +- Success: Mark ALL faction evaluators dirty, continue
   +- Failure: Break (squad done)
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

For how difficulty scaling modifies these values at runtime, see [Behavior & Threat Layers - Difficulty Scaling](BEHAVIOR_THREAT_LAYERS.md#difficulty-scaling).

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

**Leader Bonus:** 1.3 (present in config for future use; current ECS-based calculation includes leader abilities via utility power rather than a direct multiplier)

---

### Accessor Pattern

**Why Data-Driven?**
- Eliminates hardcoded constants
- Enables designer tuning without code changes
- Centralizes balance parameters
- Supports A/B testing and playtesting iterations

**Implementation Pattern:**

```go
// Configuration accessor with fallback + difficulty scaling
func GetParameterFromConfig() ValueType {
  // Load base from JSON template
  base := defaultValue
  if templates.ConfigTemplate.Parameter != 0:
    base = templates.ConfigTemplate.Parameter

  // Apply difficulty scaling
  result := base + templates.GlobalDifficulty.AI().ParameterOffset
  if result < 1:
    return 1  // Clamp to minimum
  return result
}
```

**Examples:**

```go
// AI behavior parameter (with difficulty offset)
func GetIsolationThreshold() int {
  base := 3
  if templates.AIConfigTemplate.ThreatCalculation.IsolationThreshold > 0:
    base = templates.AIConfigTemplate.ThreatCalculation.IsolationThreshold
  result := base + templates.GlobalDifficulty.AI().IsolationThresholdOffset
  if result < 1: return 1
  return result
}

// Power calculation parameter (no difficulty scaling)
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
|
+- Load Config Templates (aiconfig.json, powerconfig.json)
|
+- Create EntityManager
|
+- Create CombatQueryCache (ECS Views: ActionStateView, FactionView)
|
+- Create FactionThreatLevelManager
|  +- For each faction:
|     +- Create FactionThreatLevel
|        +- For each squad:
|           +- Create SquadThreatLevel (with ThreatByRange)
|
+- Create AIController
   +- Dependencies: EntityManager, TurnManager, MovementSystem, CombatActionSystem
   |
   +- layerEvaluators map (per faction, created lazily on first DecideFactionTurn)
      +- CompositeThreatEvaluator (created via getThreatEvaluator())
         +- CombatThreatLayer
         +- SupportValueLayer
         +- PositionalRiskLayer
```

---

### AI Turn Execution Flow

```
GUI Calls AIController.DecideFactionTurn(factionID)
|
+- 1. Clear attack queue
|
+- 2. Update threat layers (ONCE per faction turn)
|  +- FactionThreatLevelManager.UpdateAllFactions()
|  |  +- For each squad: CalculateThreatLevels()
|  |     +- ThreatByRange = CalculateSquadPowerByRange()
|  |
|  +- For each faction evaluator: evaluator.Update(currentRound)
|     (see BEHAVIOR_THREAT_LAYERS.md for layer computation details)
|
+- 3. For each squad in faction:
|  |
|  +- While squad has actions remaining (HasMoved && HasActed == false):
|     |
|     +- Create ActionContext
|     |  +- Get action state (HasMoved, HasActed)
|     |  +- Get or create threat evaluator for faction
|     |  +- Get squad role (cached in context)
|     |  +- Get current position (cached in context)
|     |
|     +- Create ActionEvaluator
|     |
|     +- EvaluateAllActions()
|     |  +- Generate movement actions (if !HasMoved)
|     |  |  +- Get valid tiles (CanMoveTo validation)
|     |  |  +- Score each tile
|     |  |     +- Base score (50.0)
|     |  |     +- Role-weighted threat
|     |  |     +- Ally proximity bonus
|     |  |     +- Approach enemy bonus
|     |  |
|     |  +- Generate attack actions (if !HasActed)
|     |  |  +- Get attackable enemies
|     |  |  +- Score each target
|     |  |     +- Base score (100.0)
|     |  |     +- Wounded bonus
|     |  |     +- Threat priority bonus
|     |  |     +- Role counter bonus
|     |  |
|     |  +- Add Wait action (score 0.0)
|     |
|     +- SelectBestAction() (highest score)
|     |
|     +- Execute Action
|     |  +- MoveAction: MovementSystem.MoveSquad() via MoveSquadCommand
|     |  +- AttackAction: CombatActionSystem.ExecuteAttackAction()
|     |  |  +- Queue attack for animation via AIController.QueueAttack()
|     |  +- WaitAction: Mark HasMoved and HasActed via ActionState
|     |
|     +- Mark ALL faction evaluators dirty (for next iteration)
|        +- evaluator.MarkDirty() for each evaluator in layerEvaluators map
|
+- 4. Return queued attacks to GUI for animation
```

---

## Performance Considerations

### Threat Layer Computation

**Complexity:** O(factions x squads x mapRadius^2)

**Optimization Techniques:**

1. **Dirty Flag Caching**
   - Layers only recompute when marked dirty
   - Prevents redundant calculations within same round
   - Invalidated after each AI action (positions change)
   - Start-of-turn update pays the full recompute cost once

2. **Map Reuse**
   - Threat maps use `clear()` instead of reallocating
   - Reduces GC pressure during AI turns
   - Go's `clear()` is optimized for map zeroing

3. **Lazy Evaluator Creation**
   - CompositeThreatEvaluator created per-faction on-demand via `getThreatEvaluator()`
   - Not all factions may execute AI (e.g., player faction)
   - Stored in `layerEvaluators` map for reuse

4. **Combat Query Cache (ECS Views)**
   - `CombatQueryCache` uses `ecs.View` objects (automatically maintained by ECS library)
   - `ActionStateView` and `FactionView` avoid full-world queries
   - Updated incrementally by the ECS library when components change

5. **IterateMapGrid vs Sparse Painting**
   - `PositionalRiskLayer` iterates full map grid for isolation/engagement/retreat
   - `CombatThreatLayer` uses sparse painting (only around enemy squads)
   - Trade-off: grid iteration is O(mapSize) but cache-friendly

**Bottlenecks:**

- **PaintThreatToMap()**: O(radius^2) per squad
  - Mitigated by typical small map sizes (30x30)
  - Could use spatial partitioning for massive maps

- **IterateMapGrid()**: O(mapWidth x mapHeight) per layer
  - Called 3 times per PositionalRiskLayer update (isolation, pressure, retreat)
  - Typical cost at 30x30: 900 iterations x 3 = 2700 callback invocations

- **GetRoleWeightedThreat()**: Called per movement candidate
  - Typical AI turn: 3 squads x 10 tiles x 4 layer queries = 120 queries
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
   - `GetUnitCombatData()` bundles all components in one query
   - Reduces component access overhead
   - Pre-calculates derived values (role, attack type, range)

3. **Config Caching**
   - PowerConfig loaded once per profile
   - Role/ability lookups iterate small arrays (<10 entries)
   - Could use maps for O(1) lookup if needed

**Bottlenecks:**

- **CalculateSquadPowerByRange()**: O(units x ranges)
  - Now uses full `calculateUnitPower()` per unit (vs. old simplified formula)
  - Called once per squad during threat update
  - Typical cost: 4 units x 5 ranges = 20 iterations

---

### ECS Query Patterns

**Best Practices:**

1. **Use ECS Views for Repeated Queries**
   ```go
   // CombatQueryCache wraps ECS Views
   actionState := cache.FindActionStateBySquadID(squadID)  // View-based, O(k)
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
    meleeThreatByPos:    ~30x30 float64 = 7.2KB
    rangedPressureByPos: ~30x30 float64 = 7.2KB

  SupportValueLayer:
    supportValuePos:     ~30x30 float64 = 7.2KB
    allyProximity:       ~30x30 int = 3.6KB

  PositionalRiskLayer:
    4 maps x ~30x30 float64 = 28.8KB

Total per faction: ~54KB
```

**Scaling:**
- 3 factions: ~162KB
- Negligible compared to ECS entity data
- Could use sparse maps if memory becomes issue

---

## Extension Points

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

### Adding New Threat Layers

See [Behavior & Threat Layers - Extension Points](BEHAVIOR_THREAT_LAYERS.md#extension-points) for the step-by-step guide to adding new threat layers.

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
| Harder encounters globally | Increase GlobalDifficulty encounter PowerMultiplierScale |
| AI more aggressive globally | Decrease GlobalDifficulty AI FlankingRangeBonusOffset |

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

**Symptoms:** AI charges into overwhelming threat, ignores danger.

**Possible Causes:**
- Approach bonus too high (overpowers threat avoidance)
- Threat layers not computing correctly
- Role weights incorrect (should be positive for avoidance)
- Attack score too high (AI always prefers attacking over survival)

**Debug Steps:**
1. Log role weights - positive should avoid, negative should attract
2. Enable `ThreatVisualizer` in `VisualizerModeThreat` to check threat values visually
3. Reduce approach multipliers
4. Increase threat layer weights in role configuration

---

For additional troubleshooting:
- **AI Ignores Wounded Allies** - See [Behavior & Threat Layers - Troubleshooting](BEHAVIOR_THREAT_LAYERS.md#troubleshooting)
- **Threat Layers Not Updating** - See [Behavior & Threat Layers - Troubleshooting](BEHAVIOR_THREAT_LAYERS.md#troubleshooting)
- **Encounter Spawning Errors** - See [Encounter System - Troubleshooting](ENCOUNTER_SYSTEM.md#troubleshooting)

---

## Appendix: File Reference

### Core AI Files

| File | Purpose | Key Functions |
|------|---------|---------------|
| `mind/ai/ai_controller.go` | Turn orchestration | `DecideFactionTurn()`, `NewActionContext()`, `QueueAttack()`, `GetQueuedAttacks()` |
| `mind/ai/action_evaluator.go` | Action generation and scoring | `EvaluateAllActions()`, `scoreMovementPosition()`, `scoreAttackTarget()`, `scoreApproachEnemy()` |

### Evaluation Files

| File | Purpose | Key Functions |
|------|---------|---------------|
| `mind/evaluation/power.go` | Power calculation | `CalculateSquadPower()`, `CalculateSquadPowerByRange()`, `CalculateOffensivePower()`, `CalculateDefensivePower()`, `CalculateUtilityPower()`, `EstimateUnitPowerFromTemplate()` |
| `mind/evaluation/power_config.go` | Config loading | `GetPowerConfigByProfile()` |
| `mind/evaluation/roles.go` | Role/ability config | `GetRoleMultiplierFromConfig()`, `GetAbilityPowerValue()`, `GetCompositionBonusFromConfig()` |
| `mind/evaluation/cache.go` | Dirty flag system | `DirtyCache`, `MarkDirty()`, `MarkClean()`, `IsValid()`, `IsDirty()`, `IsInitialized()` |

### Combat System Files

| File | Purpose | Key Functions |
|------|---------|---------------|
| `tactical/combat/turnmanager.go` | Turn management | `InitializeCombat()`, `EndTurn()`, `ResetSquadActions()`, `GetCurrentRound()`, `SetOnTurnEnd()`, `SetPostResetHook()` |
| `tactical/combat/combatactionsystem.go` | Action execution | `ExecuteAttackAction()`, `SetOnAttackComplete()`, `SetBattleRecorder()` |
| `tactical/combat/combatcomponents.go` | ECS components | `ActionStateData` (HasMoved, HasActed, MovementRemaining, BonusAttackActive), `TurnStateData`, `FactionData` |
| `tactical/combat/combatqueries.go` | Combat queries | `GetSquadFaction()`, `GetSquadMapPosition()`, `GetAllFactions()`, `GetSquadsForFaction()`, `GetActiveSquadsForFaction()`, `CreateActionStateForSquad()`, `RemoveSquadFromMap()` |
| `tactical/combat/combatqueriescache.go` | ECS View cache | `CombatQueryCache`, `FindActionStateBySquadID()`, `FindFactionByID()` |
| `tactical/combat/combatfactionmanager.go` | Faction management | `CreateFactionWithPlayer()`, `AddSquadToFaction()` |

### Additional File References

- **Behavior/Threat Files** - See [Behavior & Threat Layers - File Reference](BEHAVIOR_THREAT_LAYERS.md#file-reference)
- **Encounter Files** - See [Encounter System - File Reference](ENCOUNTER_SYSTEM.md#file-reference)

---

**End of Document**

For questions or clarifications, consult the source code or contact the development team.
