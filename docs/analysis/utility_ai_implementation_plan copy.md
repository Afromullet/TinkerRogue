# Utility AI Implementation Plan for TinkerRogue

**Last Updated:** 2025-12-27

## Overview

Implement a **Utility AI system** for computer-controlled factions that evaluates available actions using scoring functions. The AI will observe combat state, score possible moves/attacks, and execute the highest-utility action via the existing command system.

---

## Current Architecture Strengths

Your codebase is **well-prepared** for AI integration:

✅ **Threat Analysis** - `SquadThreatLevel` already calculates danger/damage by range
✅ **Distance Tracking** - `SquadDistanceTracker` organizes allies/enemies by distance
✅ **Action Validation** - `CanSquadAttackWithReason()` provides detailed error messages
✅ **Command Pattern** - `AttackCommand` and `MoveSquadCommand` ready to use
✅ **Turn Management** - `TurnManager` controls faction turns with action states

**Missing:** Decision-making logic to convert state → scored actions → commands

---

## Utility AI Architecture

### Core Concept

For each AI-controlled squad's turn:
1. **Generate** possible actions (move positions, attack targets)
2. **Score** each action using utility functions
3. **Execute** highest-scoring action via command system

### Key Components

```
tactical/ai/
├── ai_controller.go          # Main AI orchestrator
├── action_evaluator.go       # Generates and scores actions
├── utility_functions.go      # Scoring calculations
└── considerations/
    ├── tactical_position.go  # Position quality scoring
    ├── target_priority.go    # Target selection scoring
    └── squad_state.go        # Squad condition checks
```

---

## Implementation Phases

### Phase 1: AI Controller (Core Loop)

**File:** `tactical/ai/ai_controller.go`

**Purpose:** Orchestrates AI decision-making for a faction's turn

**Key Functions:**
```go
// Main entry point - called by TurnManager for AI factions
func DecideFactionTurn(factionID ecs.EntityID, combatService *combatservices.CombatService)

// Per-squad decision loop
func DecideSquadAction(squadID ecs.EntityID, combatService *combatservices.CombatService) SquadCommand
```

**Logic Flow:**
```
1. Get all alive squads in faction
2. For each squad with available actions:
   a. Check ActionStateData (HasMoved? HasActed?)
   b. Generate possible actions (via ActionEvaluator)
   c. Execute highest-scoring action
   d. Continue until no squads have actions remaining
```

**Integration Point:**
- Hook into `TurnManager.EndTurn()` or beginning of AI faction's turn
- Check `FactionData.IsPlayerControlled == false`

---

### Phase 2: Action Evaluator (Generate + Score)

**File:** `tactical/ai/action_evaluator.go`

**Purpose:** Generates all possible actions and scores them

**Key Structures:**
```go
type ScoredAction struct {
    Action      SquadCommand
    Score       float64
    Description string  // For debugging
}

type ActionContext struct {
    SquadID        ecs.EntityID
    CombatService  *combatservices.CombatService
    ThreatManager  *behavior.FactionThreatLevelManager
    ActionState    *combat.ActionStateData
}
```

**Key Functions:**
```go
// Returns all possible actions with scores
func EvaluateAllActions(ctx ActionContext) []ScoredAction

// Generates movement options
func EvaluateMoveActions(ctx ActionContext) []ScoredAction

// Generates attack options
func EvaluateAttackActions(ctx ActionContext) []ScoredAction

// Picks best action from scored list
func SelectBestAction(actions []ScoredAction) SquadCommand
```

**Action Generation:**
- **Moves:** `GetValidMovementTiles()` → score each position
- **Attacks:** `CanSquadAttackWithReason()` → score each valid target
- **Special:** Check ability cooldowns, morale-based retreats

---

### Phase 3: Utility Functions (Scoring Logic)

**File:** `tactical/ai/utility_functions.go`

**Purpose:** Core scoring calculations using existing threat data

**Movement Scoring:**
```go
func ScorePosition(position coords.LogicalPos, ctx ActionContext) float64 {
    score := 0.0

    // Use existing threat data
    threatLevel := ctx.ThreatManager.GetSquadThreatLevel(ctx.SquadID)

    // Positive factors
    score += proximityToEnemies(position, threatLevel)      // Engage/retreat logic
    score += coverageOfAllies(position, ctx)                // Support positioning
    score += terrainValue(position, ctx)                    // Future: cover/terrain

    // Negative factors
    score -= exposureToDanger(position, threatLevel)        // Avoid danger zones
    score -= isolationFromAllies(position, ctx)             // Stay grouped

    return score
}
```

**Attack Scoring:**
```go
func ScoreTarget(targetID ecs.EntityID, ctx ActionContext) float64 {
    score := 0.0

    // Use existing threat data
    threatLevel := ctx.ThreatManager.GetSquadThreatLevel(targetID)

    // Priority factors
    score += expectedDamage(ctx.SquadID, targetID, ctx)     // Kill potential
    score += threatReduction(targetID, threatLevel)         // Eliminate dangerous enemies
    score += focusFire(targetID, ctx)                       // Finish wounded squads
    score += roleCounters(ctx.SquadID, targetID, ctx)       // Tank > DPS > Support

    return score
}
```

---

### Phase 4: Considerations (Modular Scoring)

**Files:** `tactical/ai/considerations/*.go`

**Purpose:** Isolate specific scoring concerns for readability/tuning

**Tactical Position** (`tactical_position.go`)
- Distance to nearest enemy (engagement range)
- Distance to nearest ally (support range)
- Flank opportunities (future)
- Choke point control (future)

**Target Priority** (`target_priority.go`)
- Threat level (use `DangerByRange` from `SquadThreatLevel`)
- Current health percentage
- Role matchups (Tank counters Support, DPS counters Tank, etc.)
- Focus fire (number of allies already targeting)

**Squad State** (`squad_state.go`)
- Morale level (retreat if low)
- Health percentage (defensive if wounded)
- Ability cooldowns (offensive if abilities ready)
- Faction mana (conservative if low)

---

## Integration with Existing Systems

### Threat Manager Integration

**Already Available:**
```go
// Get threat data for decision-making
threatManager := combatService.GetFactionThreatManager()
squadThreat := threatManager.GetSquadThreatLevel(squadID)

// Use existing calculations
dangerAtRange3 := squadThreat.DangerByRange[3]
expectedDamage := squadThreat.ExpectedDamageByRange[2]

// Get distance tracking
tracker := squadThreat.SquadDistances
nearestEnemy := tracker.GetNearestEnemy()
alliesInRange := tracker.AlliesByDistance[2]  // All allies 2 tiles away
```

**Extend with:**
- `GetSquadsByDangerLevel(factionID)` - Sort targets by threat
- `GetSafePositions(squadID, radius)` - Find low-danger zones
- `GetOptimalEngagementRange(squadID)` - Best range for squad composition

### Command Execution

**Immediate Execution:**
```go
executor := combatService.GetCommandExecutor()
cmd := squadcommands.NewMoveSquadCommand(squadID, targetPos, combatService, manager)
executor.Execute(cmd)  // Validates, executes, adds to history
```

**Queued Execution** (for multi-turn moves):
```go
cmd := squadcommands.NewMoveSquadCommand(squadID, distantPos, combatService, manager)
squadcommands.QueueCommand(manager, squadID, cmd)
// Automatically creates MoveStepCommands for multi-turn path
```

### Turn Manager Hook

**Add AI Decision Hook:**
```go
// In TurnManager.EndTurn() or start of next faction's turn
func (tm *TurnManager) AdvanceTurn() {
    tm.CurrentTurnIndex++
    factionID := tm.GetCurrentFaction()

    // Reset action states
    tm.ResetSquadActions(factionID)

    // AI decision point
    faction := common.GetComponentTypeByID[*FactionData](tm.manager, factionID, FactionComponent)
    if !faction.IsPlayerControlled {
        ai.DecideFactionTurn(factionID, tm.combatService)
    }
}
```

---

## Scoring Function Examples

### Movement: Engage vs. Retreat

```go
func proximityToEnemies(pos coords.LogicalPos, threat *behavior.SquadThreatLevel) float64 {
    nearestEnemy := threat.SquadDistances.GetNearestEnemy()
    distance := coords.ChebyshevDistance(pos, nearestEnemy.Position)

    // Tank: wants close (score increases as distance decreases)
    if isRole(threat.SquadID, Tank) {
        return 10.0 / (1.0 + float64(distance))
    }

    // Ranged: wants optimal range 3-5
    if isRole(threat.SquadID, Ranged) {
        optimalRange := 4.0
        return -math.Abs(float64(distance) - optimalRange)
    }

    // Support: wants medium distance
    return 5.0 / (1.0 + math.Abs(float64(distance)-3.0))
}
```

### Attack: Focus Fire

```go
func focusFire(targetID ecs.EntityID, ctx ActionContext) float64 {
    squad := squads.GetSquadEntity(targetID, ctx.Manager)
    currentHP := squad.TotalHP  // Hypothetical query
    maxHP := squad.MaxHP

    healthPercent := float64(currentHP) / float64(maxHP)

    // Heavily prioritize wounded targets (finish kills)
    if healthPercent < 0.3 {
        return 20.0  // High bonus
    } else if healthPercent < 0.6 {
        return 10.0  // Medium bonus
    }
    return 0.0
}
```

### Threat Reduction

```go
func threatReduction(targetID ecs.EntityID, threat *behavior.SquadThreatLevel) float64 {
    // Use existing expected damage calculations
    maxDanger := 0.0
    for _, danger := range threat.ExpectedDamageByRange {
        if danger > maxDanger {
            maxDanger = danger
        }
    }

    // Prioritize high-threat targets
    return maxDanger * 0.5  // Scale to reasonable score range
}
```

---

## Data Structure Additions

### Optional: AI Configuration

**File:** `tactical/ai/ai_config.go`

```go
type AIConfig struct {
    Aggression      float64  // 0.0 = defensive, 1.0 = aggressive
    FocusFireBonus  float64  // Weight for finishing wounded enemies
    ThresholdRetreat float64  // Health % to trigger retreat
    RangePreference int      // Preferred engagement range
}

// Per-faction or per-squad configurations
var FactionAIConfigs = map[ecs.EntityID]*AIConfig{
    // Allows different AI personalities
}
```

### Optional: Action History

Track AI decisions for debugging:

```go
type AIDecisionLog struct {
    Round       int
    SquadID     ecs.EntityID
    ActionTaken string
    Score       float64
    Alternatives []ScoredAction  // Top 3 rejected actions
}
```

---

## Implementation Order

### Iteration 1: Basic AI
1. Create `ai_controller.go` with `DecideFactionTurn()`
2. Implement simple `ActionEvaluator` (move to nearest enemy, attack weakest)
3. Hook into `TurnManager`
4. **Test:** AI makes valid moves/attacks every turn

### Iteration 2: Scoring System
1. Implement `utility_functions.go` with position/target scoring
2. Add considerations for role-based behavior
3. Integrate `SquadThreatLevel` data for scoring
4. **Test:** AI engages at appropriate ranges, focuses fire

### Iteration 3: Advanced Tactics
1. Add ability usage (check cooldowns, score abilities)
2. Implement retreat logic (morale/health thresholds)
3. Add formation/role-based positioning
4. Extend threat maps with additional state (if needed)
5. **Test:** AI uses abilities, retreats intelligently, coordinates squads

### Iteration 4: Polish
1. Add AI difficulty settings (config-based aggression/skill)
2. Implement decision logging for debugging
3. Balance scoring weights via playtesting
4. **Test:** Multiple AI personalities, consistent challenge level

---

## Key Design Decisions

### Why Utility AI?

**Pros:**
- **Transparent:** Easy to debug (scores are visible)
- **Tunable:** Adjust weights without rewriting logic
- **Extensible:** Add new considerations independently
- **No State:** Stateless evaluation (no behavior tree memory)

**Cons:**
- Requires weight balancing (playtesting needed)
- Can be computationally expensive (mitigated by existing threat calculations)

### Alternatives Considered

**Behavior Trees:**
- More complex state management
- Harder to balance/debug
- Better for complex sequences (not needed here)

**Goal-Oriented Action Planning (GOAP):**
- Overkill for turn-based tactics
- More expensive planning phase
- Better for long-term strategy games

**Rule-Based:**
- Brittle (if-else chains)
- Hard to extend
- Less interesting AI behavior

---

## Critical Files to Modify/Create

### Create:
- `tactical/ai/ai_controller.go` - Main decision loop
- `tactical/ai/action_evaluator.go` - Action generation/scoring
- `tactical/ai/utility_functions.go` - Core scoring logic
- `tactical/ai/considerations/tactical_position.go` - Position scoring
- `tactical/ai/considerations/target_priority.go` - Target scoring
- `tactical/ai/considerations/squad_state.go` - State checks

### Modify:
- `tactical/combat/TurnManager.go` - Add AI hook in turn advancement
- `tactical/behavior/dangerlevel.go` - (Optional) Add helper queries for AI

### Reference:
- `tactical/squadcommands/command.go` - Command interface
- `tactical/behavior/dangerlevel.go` - Existing threat data
- `tactical/combatservices/combat_service.go` - Service access

---

## Success Criteria

✅ AI-controlled factions make decisions autonomously
✅ Actions respect game rules (validation passes)
✅ Scoring system produces reasonable tactical decisions
✅ AI engages at appropriate ranges for squad composition
✅ AI prioritizes wounded enemies (focus fire)
✅ AI retreats when overwhelmed
✅ System is extensible (easy to add new considerations)

---

## Notes

- **Leverage existing systems:** Threat maps, distance tracking, command validation all exist
- **Start simple:** Basic move-to-engage + attack-weakest is a solid foundation
- **Iterate:** Add complexity incrementally (abilities → retreat → formations)
- **Debug with logs:** Track scored actions to understand AI decisions
- **Balance via play:** Adjust scoring weights based on observed behavior

This design integrates cleanly with your existing architecture and provides a clear path from basic AI to sophisticated tactical decision-making.
