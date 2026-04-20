# AI Controller & Action Selection

**Last Updated:** 2026-02-20

Technical reference for TinkerRogue's AI turn orchestration, action evaluation, and scoring algorithms.

---

## Related Documents

- [AI Algorithm Architecture](AI_ALGORITHM_ARCHITECTURE.md) - Overview, system diagram, performance considerations
- [Power Evaluation](POWER_EVALUATION.md) - Power calculation shared by AI threat and encounter generation
- [AI Configuration](AI_CONFIGURATION.md) - Config files, accessor patterns, tuning guide
- [Behavior & Threat Layers](BEHAVIOR_THREAT_LAYERS.md) - Threat layer subsystems and spatial analysis
- [Encounter System](ENCOUNTER_SYSTEM.md) - Encounter generation, lifecycle, and rewards

---

## Table of Contents

1. [AIController](#aicontroller)
2. [ActionContext](#actioncontext)
3. [ActionEvaluator](#actionevaluator)
4. [Action Selection Algorithm](#action-selection-algorithm)
5. [AI Turn Execution Flow](#ai-turn-execution-flow)
6. [Extension Points](#extension-points)
7. [Troubleshooting](#troubleshooting)
8. [File Reference](#file-reference)

---

## AIController

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

## ActionContext

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

## ActionEvaluator

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

## AI Turn Execution Flow

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

## Troubleshooting

### AI Not Moving

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

### AI Not Attacking

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

### AI Suicidal Behavior

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

## File Reference

### Core AI Files

| File | Purpose | Key Functions |
|------|---------|---------------|
| `mind/ai/ai_controller.go` | Turn orchestration | `DecideFactionTurn()`, `NewActionContext()`, `QueueAttack()`, `GetQueuedAttacks()` |
| `mind/ai/action_evaluator.go` | Action generation and scoring | `EvaluateAllActions()`, `scoreMovementPosition()`, `scoreAttackTarget()`, `scoreApproachEnemy()` |

### Combat System Files

| File | Purpose | Key Functions |
|------|---------|---------------|
| `tactical/combat/turnmanager.go` | Turn management | `InitializeCombat()`, `EndTurn()`, `ResetSquadActions()`, `GetCurrentRound()`, `SetOnTurnEnd()`, `SetPostResetHook()` |
| `tactical/combat/combatactionsystem.go` | Action execution | `ExecuteAttackAction()`, `SetOnAttackComplete()`, `SetBattleRecorder()` |
| `tactical/combat/combatcomponents.go` | ECS components | `ActionStateData` (HasMoved, HasActed, MovementRemaining, BonusAttackActive), `TurnStateData`, `FactionData` |
| `tactical/combat/combatqueries.go` | Combat queries | `GetSquadFaction()`, `GetSquadMapPosition()`, `GetAllFactions()`, `GetSquadsForFaction()`, `GetActiveSquadsForFaction()`, `CreateActionStateForSquad()`, `RemoveSquadFromMap()` |
| `tactical/combat/combatqueriescache.go` | ECS View cache | `CombatQueryCache`, `FindActionStateBySquadID()`, `FindFactionByID()` |
| `tactical/combat/combatfactionmanager.go` | Faction management | `CreateFactionWithPlayer()`, `AddSquadToFaction()` |

---

**End of Document**

For questions or clarifications, consult the source code or the [AI Algorithm Architecture](AI_ALGORITHM_ARCHITECTURE.md) overview.
