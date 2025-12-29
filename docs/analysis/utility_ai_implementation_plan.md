# Utility AI Implementation Plan for TinkerRogue

**Last Updated:** 2025-12-28

## Overview

Implement a **Utility AI system** for computer-controlled factions that evaluates available actions using scoring functions. The AI will observe combat state through a **multi-layer threat evaluation system**, score possible moves/attacks using role-aware utility functions, and execute the highest-utility action via the existing command system.

---

## Current Architecture Strengths

Your codebase is **well-prepared** for AI integration:

- **Threat Analysis** - `SquadThreatLevel` already calculates danger/damage by range
- **Distance Tracking** - `SquadDistanceTracker` organizes allies/enemies by distance
- **Action Validation** - `CanSquadAttackWithReason()` provides detailed error messages
- **Command Pattern** - `AttackCommand` and `MoveSquadCommand` ready to use
- **Turn Management** - `TurnManager` controls faction turns with action states
- **Role System** - `UnitRole` (Tank/DPS/Support) and `AttackType` already defined
- **Ability System** - `AbilitySlotData` with triggers and cooldowns

**Missing:** Multi-layer threat maps and role-aware decision logic to convert state -> scored actions -> commands

---

## Multi-Layer Threat Evaluation System

### Design Philosophy

The current `FactionThreatLevelManager` provides a single aggregate threat value per squad. For deeper tactical decision-making, we expand this into **specialized threat layers** that each squad queries based on its role and current tactical situation.

### Threat Layer Architecture

```
tactical/behavior/
├── dangerlevel.go              # Existing - keep as-is
├── threat_layers.go            # NEW: Multi-layer threat system
├── threat_melee.go             # NEW: Melee danger computation
├── threat_ranged.go            # NEW: Ranged pressure computation
├── threat_support.go           # NEW: Support value/heal priority
├── threat_positional.go        # NEW: Positional risk/opportunity
└── threat_composite.go         # NEW: Layer combination strategies
```

---

## Threat Layer Definitions

### Layer 1: Melee Danger Map (`MeleeThreatLayer`)

**Purpose:** Identifies positions where melee-focused enemies can engage.

**Computation:**
- For each enemy squad with `AttackType == MeleeRow || MeleeColumn`:
  - Calculate threat radius = `MovementSpeed + AttackRange` (typically 3-5 tiles)
  - Weight by squad's `ExpectedDamageByRange[1]` (melee-range damage)
  - Apply Tank role multiplier (1.2x) for durability threat
  - Apply DPS role multiplier (1.5x) for damage threat

**Data Structure:**
```go
type MeleeThreatLayer struct {
    manager        *common.EntityManager
    factionID      ecs.EntityID
    threatByPos    map[coords.LogicalPos]float64  // Position -> threat value
    threatBySquad  map[ecs.EntityID]float64       // Squad -> total melee threat emitted
    effectiveRange map[ecs.EntityID]int           // Squad -> max melee engagement range
}

// Query API
func (m *MeleeThreatLayer) GetMeleeThreatAt(pos coords.LogicalPos) float64
func (m *MeleeThreatLayer) GetMeleeThreatFrom(squadID ecs.EntityID) float64
func (m *MeleeThreatLayer) GetSafestPositionFrom(pos coords.LogicalPos, radius int) coords.LogicalPos
func (m *MeleeThreatLayer) IsInMeleeZone(pos coords.LogicalPos) bool
```

**Use Cases:**
- Ranged squads avoid high-melee-threat positions
- Support squads maintain safe distance from melee threats
- Tank squads seek high-melee-threat positions to intercept

---

### Layer 2: Ranged Pressure Map (`RangedThreatLayer`)

**Purpose:** Identifies positions under threat from ranged attackers.

**Computation:**
- For each enemy squad with `AttackType == Ranged || Magic`:
  - Calculate threat radius = `AttackRange` (no movement needed for ranged)
  - Weight by `ExpectedDamageByRange[attackRange]`
  - Apply damage falloff for partial coverage (edge of range)
  - Magic attacks apply additional "AoE potential" modifier

**Data Structure:**
```go
type RangedThreatLayer struct {
    manager         *common.EntityManager
    factionID       ecs.EntityID
    pressureByPos   map[coords.LogicalPos]float64  // Position -> ranged pressure
    lineOfFireZones map[ecs.EntityID][]coords.LogicalPos // Squad -> threatened positions
    coverPositions  []coords.LogicalPos            // Positions with terrain cover (future)
}

// Query API
func (r *RangedThreatLayer) GetRangedPressureAt(pos coords.LogicalPos) float64
func (r *RangedThreatLayer) GetRangedThreatsToPosition(pos coords.LogicalPos) []ecs.EntityID
func (r *RangedThreatLayer) GetBestFiringPosition(squadID ecs.EntityID, targets []ecs.EntityID) coords.LogicalPos
func (r *RangedThreatLayer) GetCoverFromRanged(pos coords.LogicalPos, radius int) coords.LogicalPos
```

**Use Cases:**
- All squads factor ranged pressure into movement decisions
- Support squads especially avoid high-ranged-pressure positions
- Ranged squads seek positions with good line-of-fire coverage

---

### Layer 3: Support Value Map (`SupportValueLayer`)

**Purpose:** Identifies positions where support actions (healing, buffs) have high value.

**Computation:**
- For each allied squad:
  - Calculate heal priority = `(1 - CurrentHP/MaxHP) * SquadValue`
  - Calculate buff priority based on engagement state and ability availability
  - Weight positions by proximity to high-priority allies
  - Consider ability range (healing aura radius)

**Data Structure:**
```go
type SupportValueLayer struct {
    manager         *common.EntityManager
    factionID       ecs.EntityID
    healPriority    map[ecs.EntityID]float64       // Squad -> heal urgency
    buffPriority    map[ecs.EntityID]float64       // Squad -> buff value
    supportValuePos map[coords.LogicalPos]float64  // Position -> support value
    allyProximity   map[coords.LogicalPos]int      // Position -> count of nearby allies
}

// Query API
func (s *SupportValueLayer) GetHealPriority(squadID ecs.EntityID) float64
func (s *SupportValueLayer) GetSupportValueAt(pos coords.LogicalPos) float64
func (s *SupportValueLayer) GetBestSupportPosition(supporterID ecs.EntityID, abilityRange int) coords.LogicalPos
func (s *SupportValueLayer) GetAlliesInHealRange(pos coords.LogicalPos, healRange int) []ecs.EntityID
func (s *SupportValueLayer) GetMostDamagedAlly() ecs.EntityID
```

**Use Cases:**
- Support squads prioritize movement to high-support-value positions
- Healing ability targeting prioritizes high-heal-priority squads
- Formation logic keeps allies clustered for efficient support

---

### Layer 4: Positional Risk Map (`PositionalRiskLayer`)

**Purpose:** Identifies positions with tactical advantages or disadvantages.

**Computation:**
- **Flanking Risk:** Positions where enemies can attack from multiple directions
- **Isolation Risk:** Distance from nearest ally (isolated = higher risk)
- **Engagement Pressure:** Difference between total incoming damage and outgoing damage
- **Retreat Path Quality:** Access to safe fallback positions

**Data Structure:**
```go
type PositionalRiskLayer struct {
    manager           *common.EntityManager
    factionID         ecs.EntityID
    flankingRisk      map[coords.LogicalPos]float64  // Position -> flank exposure
    isolationRisk     map[coords.LogicalPos]float64  // Position -> isolation penalty
    engagementPressure map[coords.LogicalPos]float64 // Position -> net damage exposure
    retreatQuality    map[coords.LogicalPos]float64  // Position -> escape route quality
}

// Query API
func (p *PositionalRiskLayer) GetFlankingRiskAt(pos coords.LogicalPos) float64
func (p *PositionalRiskLayer) GetIsolationRiskAt(pos coords.LogicalPos) float64
func (p *PositionalRiskLayer) GetTotalRiskAt(pos coords.LogicalPos) float64
func (p *PositionalRiskLayer) GetBestRetreatPath(pos coords.LogicalPos) []coords.LogicalPos
func (p *PositionalRiskLayer) IsFlankingPosition(pos, targetPos coords.LogicalPos) bool
```

**Use Cases:**
- All squads avoid high-flanking-risk positions
- Retreat logic uses retreat quality scores
- Aggressive squads seek flanking positions against enemies

---

## Layer Composition System

### CompositeThreatEvaluator

Combines multiple layers into role-specific threat assessments.

```go
type CompositeThreatEvaluator struct {
    manager       *common.EntityManager
    factionID     ecs.EntityID

    // Individual layers (computed once per AI turn)
    meleeThreat   *MeleeThreatLayer
    rangedThreat  *RangedThreatLayer
    supportValue  *SupportValueLayer
    positionalRisk *PositionalRiskLayer

    // Cache invalidation
    lastUpdateRound int
    isDirty         bool
}

// Layer combination weights by role
type RoleThreatWeights struct {
    MeleeWeight      float64
    RangedWeight     float64
    SupportWeight    float64
    PositionalWeight float64
}

var DefaultRoleWeights = map[squads.UnitRole]RoleThreatWeights{
    squads.RoleTank: {
        MeleeWeight:      -0.5,  // Tanks seek melee danger (negative = attraction)
        RangedWeight:     0.3,   // Moderate concern for ranged
        SupportWeight:    0.2,   // Stay near support for heals
        PositionalWeight: 0.5,   // High concern for isolation
    },
    squads.RoleDPS: {
        MeleeWeight:      0.7,   // Avoid melee danger
        RangedWeight:     0.5,   // Moderate concern for ranged
        SupportWeight:    0.1,   // Low support priority
        PositionalWeight: 0.6,   // High concern for flanking
    },
    squads.RoleSupport: {
        MeleeWeight:      1.0,   // Strongly avoid melee danger
        RangedWeight:     0.8,   // Strongly avoid ranged pressure
        SupportWeight:    -1.0,  // Seek high support value positions
        PositionalWeight: 0.4,   // Moderate positional awareness
    },
}
```

### Role-Aware Threat Query API

```go
// GetRoleWeightedThreat returns combined threat score for a position
// based on the squad's role composition
func (c *CompositeThreatEvaluator) GetRoleWeightedThreat(
    squadID ecs.EntityID,
    pos coords.LogicalPos,
) float64 {
    role := c.getSquadPrimaryRole(squadID)
    weights := DefaultRoleWeights[role]

    meleeThreat := c.meleeThreat.GetMeleeThreatAt(pos)
    rangedThreat := c.rangedThreat.GetRangedPressureAt(pos)
    supportValue := c.supportValue.GetSupportValueAt(pos)
    positionalRisk := c.positionalRisk.GetTotalRiskAt(pos)

    return meleeThreat * weights.MeleeWeight +
           rangedThreat * weights.RangedWeight +
           supportValue * weights.SupportWeight +
           positionalRisk * weights.PositionalWeight
}

// GetOptimalPositionForRole finds the best position for a squad given its role
func (c *CompositeThreatEvaluator) GetOptimalPositionForRole(
    squadID ecs.EntityID,
    candidatePositions []coords.LogicalPos,
) coords.LogicalPos {
    bestPos := candidatePositions[0]
    bestScore := math.Inf(1)  // Lower is better (less threat)

    for _, pos := range candidatePositions {
        score := c.GetRoleWeightedThreat(squadID, pos)
        if score < bestScore {
            bestScore = score
            bestPos = pos
        }
    }

    return bestPos
}

// GetTargetPriority returns priority score for attacking a specific enemy
func (c *CompositeThreatEvaluator) GetTargetPriority(
    attackerID, targetID ecs.EntityID,
) float64 {
    attackerRole := c.getSquadPrimaryRole(attackerID)
    targetRole := c.getSquadPrimaryRole(targetID)

    // Base priority from existing threat data
    basePriority := c.getBaseThreatPriority(targetID)

    // Role counter bonuses
    counterBonus := c.getRoleCounterBonus(attackerRole, targetRole)

    // Focus fire bonus (wounded enemies)
    focusBonus := c.getFocusFireBonus(targetID)

    // Threat reduction value (eliminating dangerous enemies)
    threatValue := c.getThreatReductionValue(targetID)

    return basePriority + counterBonus + focusBonus + threatValue
}
```

---

## Squad Role Influence on Decision-Making

### Role Detection

```go
// getSquadPrimaryRole determines the dominant role of a squad
// based on unit composition
func (c *CompositeThreatEvaluator) getSquadPrimaryRole(squadID ecs.EntityID) squads.UnitRole {
    unitIDs := squads.GetUnitIDsInSquad(squadID, c.manager)

    roleCounts := map[squads.UnitRole]int{
        squads.RoleTank:    0,
        squads.RoleDPS:     0,
        squads.RoleSupport: 0,
    }

    for _, unitID := range unitIDs {
        entity := c.manager.FindEntityByID(unitID)
        if entity == nil {
            continue
        }

        roleData := common.GetComponentType[*squads.UnitRoleData](entity, squads.UnitRoleComponent)
        if roleData != nil {
            roleCounts[roleData.Role]++
        }
    }

    // Return role with highest count
    maxRole := squads.RoleDPS  // Default
    maxCount := 0
    for role, count := range roleCounts {
        if count > maxCount {
            maxCount = count
            maxRole = role
        }
    }

    return maxRole
}
```

### Role-Specific Behaviors

#### Tank Squads
- **Movement:** Seek positions that intercept enemy paths to allies
- **Targeting:** Prioritize enemies threatening support/DPS allies
- **Positioning:** Accept high threat positions to shield allies
- **Threat Weights:** Attracted to melee danger, seeks support proximity

```go
func (e *ActionEvaluator) evaluateTankMovement(ctx ActionContext) []ScoredAction {
    var actions []ScoredAction

    // Get valid movement tiles
    positions := ctx.MovementSystem.GetValidMovementTiles(ctx.SquadID)

    for _, pos := range positions {
        score := 0.0

        // Positive: Positions that block enemy access to allies
        interceptValue := ctx.ThreatEval.GetInterceptValue(ctx.SquadID, pos)
        score += interceptValue * 2.0

        // Positive: Proximity to support allies
        supportProximity := ctx.ThreatEval.supportValue.GetAllyProximityBonus(pos)
        score += supportProximity * 0.5

        // Negative: Isolation from allies
        isolationRisk := ctx.ThreatEval.positionalRisk.GetIsolationRiskAt(pos)
        score -= isolationRisk * 1.0

        actions = append(actions, ScoredAction{
            Action:      NewMoveCommand(ctx.SquadID, pos),
            Score:       score,
            Description: fmt.Sprintf("Tank move to %v (intercept=%.1f)", pos, interceptValue),
        })
    }

    return actions
}
```

#### DPS Squads
- **Movement:** Seek optimal engagement range based on attack type
- **Targeting:** Prioritize high-threat enemies and wounded targets
- **Positioning:** Avoid melee danger, seek flanking opportunities
- **Threat Weights:** Avoids melee/ranged danger, seeks positional advantage

```go
func (e *ActionEvaluator) evaluateDPSMovement(ctx ActionContext) []ScoredAction {
    var actions []ScoredAction

    // Determine optimal range based on squad's attack composition
    optimalRange := e.getOptimalEngagementRange(ctx.SquadID)

    positions := ctx.MovementSystem.GetValidMovementTiles(ctx.SquadID)
    nearestEnemy := ctx.ThreatEval.GetNearestEnemy(ctx.SquadID)

    for _, pos := range positions {
        score := 0.0

        // Score based on distance to optimal engagement range
        distanceToEnemy := coords.ChebyshevDistance(pos, nearestEnemy.Position)
        rangeDeviation := math.Abs(float64(distanceToEnemy - optimalRange))
        score -= rangeDeviation * 1.5  // Penalty for suboptimal range

        // Avoid melee danger
        meleeThreat := ctx.ThreatEval.meleeThreat.GetMeleeThreatAt(pos)
        score -= meleeThreat * 0.7

        // Bonus for flanking positions
        if ctx.ThreatEval.positionalRisk.IsFlankingPosition(pos, nearestEnemy.Position) {
            score += 3.0
        }

        actions = append(actions, ScoredAction{
            Action:      NewMoveCommand(ctx.SquadID, pos),
            Score:       score,
            Description: fmt.Sprintf("DPS move to %v (range=%d)", pos, distanceToEnemy),
        })
    }

    return actions
}
```

#### Support Squads
- **Movement:** Stay near damaged allies, maintain safe distance from enemies
- **Targeting:** Use abilities on highest-priority allies
- **Positioning:** Maximize ally coverage, minimize enemy exposure
- **Threat Weights:** Strongly avoids all danger, seeks support value positions

```go
func (e *ActionEvaluator) evaluateSupportMovement(ctx ActionContext) []ScoredAction {
    var actions []ScoredAction

    positions := ctx.MovementSystem.GetValidMovementTiles(ctx.SquadID)

    for _, pos := range positions {
        score := 0.0

        // Strong priority: Support value (allies needing help)
        supportValue := ctx.ThreatEval.supportValue.GetSupportValueAt(pos)
        score += supportValue * 2.0

        // Strong avoidance: Melee danger
        meleeThreat := ctx.ThreatEval.meleeThreat.GetMeleeThreatAt(pos)
        score -= meleeThreat * 1.0

        // Strong avoidance: Ranged pressure
        rangedPressure := ctx.ThreatEval.rangedThreat.GetRangedPressureAt(pos)
        score -= rangedPressure * 0.8

        // Bonus: Positions with good retreat paths
        retreatQuality := ctx.ThreatEval.positionalRisk.GetRetreatQuality(pos)
        score += retreatQuality * 0.3

        actions = append(actions, ScoredAction{
            Action:      NewMoveCommand(ctx.SquadID, pos),
            Score:       score,
            Description: fmt.Sprintf("Support move to %v (support=%.1f)", pos, supportValue),
        })
    }

    return actions
}
```

---

## Integration with Existing Systems

### Extending FactionThreatLevelManager

The existing `FactionThreatLevelManager` remains as the core threat data source. We add a new `MultiLayerThreatManager` that wraps it and provides layer-specific queries.

```go
// File: tactical/behavior/multilayer_threat.go

type MultiLayerThreatManager struct {
    baseThreatManager *FactionThreatLevelManager
    manager           *common.EntityManager
    cache             *combat.CombatQueryCache

    // Layer instances (lazy-initialized per faction)
    layers map[ecs.EntityID]*CompositeThreatEvaluator
}

func NewMultiLayerThreatManager(
    baseThreatManager *FactionThreatLevelManager,
    manager *common.EntityManager,
    cache *combat.CombatQueryCache,
) *MultiLayerThreatManager {
    return &MultiLayerThreatManager{
        baseThreatManager: baseThreatManager,
        manager:           manager,
        cache:             cache,
        layers:            make(map[ecs.EntityID]*CompositeThreatEvaluator),
    }
}

func (m *MultiLayerThreatManager) GetEvaluatorForFaction(
    factionID ecs.EntityID,
) *CompositeThreatEvaluator {
    if evaluator, exists := m.layers[factionID]; exists {
        return evaluator
    }

    // Create new evaluator for this faction
    evaluator := NewCompositeThreatEvaluator(factionID, m.manager, m.cache)
    m.layers[factionID] = evaluator
    return evaluator
}

func (m *MultiLayerThreatManager) UpdateAllLayers(currentRound int) {
    // Update base threat data first
    m.baseThreatManager.UpdateAllFactions()

    // Then update composite layers
    for _, evaluator := range m.layers {
        evaluator.Update(currentRound)
    }
}
```

### CombatService Integration

```go
// Modified CombatService to include multi-layer threat
type CombatService struct {
    EntityManager       *common.EntityManager
    TurnManager         *combat.TurnManager
    FactionManager      *combat.FactionManager
    MovementSystem      *combat.CombatMovementSystem
    CombatCache         *combat.CombatQueryCache
    CombatActSystem     *combat.CombatActionSystem

    // NEW: Multi-layer threat evaluation
    ThreatManager       *behavior.FactionThreatLevelManager
    MultiLayerThreat    *behavior.MultiLayerThreatManager
}

func (cs *CombatService) GetThreatEvaluator(factionID ecs.EntityID) *behavior.CompositeThreatEvaluator {
    return cs.MultiLayerThreat.GetEvaluatorForFaction(factionID)
}
```

---

## Utility AI Architecture

### Core Components

```
tactical/ai/
├── ai_controller.go          # Main AI orchestrator
├── action_evaluator.go       # Generates and scores actions
├── utility_functions.go      # Scoring calculations
├── role_evaluators.go        # Role-specific evaluation logic
└── considerations/
    ├── tactical_position.go  # Position quality scoring (uses threat layers)
    ├── target_priority.go    # Target selection scoring (uses threat layers)
    └── squad_state.go        # Squad condition checks
```

### ActionContext with Threat Layers

```go
type ActionContext struct {
    SquadID        ecs.EntityID
    FactionID      ecs.EntityID
    CombatService  *combatservices.CombatService
    ActionState    *combat.ActionStateData

    // Multi-layer threat access
    ThreatEval     *behavior.CompositeThreatEvaluator

    // Cached squad info
    SquadRole      squads.UnitRole
    CurrentPos     coords.LogicalPos
    SquadHealth    float64  // Average HP percentage
}

func NewActionContext(
    squadID ecs.EntityID,
    combatService *combatservices.CombatService,
) ActionContext {
    factionID := combat.GetSquadFaction(squadID, combatService.EntityManager)

    return ActionContext{
        SquadID:       squadID,
        FactionID:     factionID,
        CombatService: combatService,
        ThreatEval:    combatService.GetThreatEvaluator(factionID),
        ActionState:   combatService.CombatCache.FindActionStateBySquadID(squadID),
        SquadRole:     getSquadPrimaryRole(squadID, combatService.EntityManager),
        CurrentPos:    combat.GetSquadPosition(squadID, combatService.EntityManager),
        SquadHealth:   calculateSquadHealthPercent(squadID, combatService.EntityManager),
    }
}
```

### Main Decision Loop

```go
// File: tactical/ai/ai_controller.go

func DecideFactionTurn(
    factionID ecs.EntityID,
    combatService *combatservices.CombatService,
) {
    // Update threat layers at start of AI turn
    combatService.MultiLayerThreat.UpdateAllLayers(
        combatService.TurnManager.GetCurrentRound(),
    )

    // Get all alive squads
    squadIDs := combatService.GetAliveSquadsInFaction(factionID)

    // Process each squad
    for _, squadID := range squadIDs {
        ctx := NewActionContext(squadID, combatService)

        // Skip if no actions remaining
        if ctx.ActionState.HasMoved && ctx.ActionState.HasActed {
            continue
        }

        // Evaluate and execute best action
        action := DecideSquadAction(ctx)
        if action != nil {
            ExecuteAction(action, combatService)
        }
    }
}

func DecideSquadAction(ctx ActionContext) SquadCommand {
    evaluator := NewActionEvaluator(ctx)

    // Generate all possible actions with scores
    actions := evaluator.EvaluateAllActions()

    if len(actions) == 0 {
        return nil
    }

    // Select best action
    return SelectBestAction(actions)
}
```

---

## Layer Computation Details

### Melee Threat Computation

```go
func (m *MeleeThreatLayer) Compute() {
    m.threatByPos = make(map[coords.LogicalPos]float64)
    m.threatBySquad = make(map[ecs.EntityID]float64)

    // Get all enemy squads
    enemyFactions := m.getEnemyFactions()

    for _, enemyFactionID := range enemyFactions {
        squadIDs := combat.GetSquadsForFaction(enemyFactionID, m.manager)

        for _, squadID := range squadIDs {
            squadPos := combat.GetSquadPosition(squadID, m.manager)

            // Check if squad has melee capability
            if !m.hasMeleeUnits(squadID) {
                continue
            }

            // Calculate threat radius and value
            moveSpeed := squads.GetSquadMovementSpeed(squadID, m.manager)
            maxMeleeRange := m.getMaxMeleeRange(squadID)
            threatRadius := moveSpeed + maxMeleeRange

            // Get melee damage output
            baseThreat := m.baseThreatManager.GetSquadThreatLevel(squadID)
            meleeDamage := baseThreat.ExpectedDamageByRange[1]

            // Apply role modifiers
            roleModifier := m.getRoleModifier(squadID)
            totalThreat := meleeDamage * roleModifier

            m.threatBySquad[squadID] = totalThreat

            // Paint threat on map
            m.paintThreatRadius(squadPos, threatRadius, totalThreat)
        }
    }
}

func (m *MeleeThreatLayer) paintThreatRadius(
    center coords.LogicalPos,
    radius int,
    threat float64,
) {
    for dx := -radius; dx <= radius; dx++ {
        for dy := -radius; dy <= radius; dy++ {
            pos := coords.LogicalPos{X: center.X + dx, Y: center.Y + dy}
            distance := coords.ChebyshevDistance(center, pos)

            if distance <= radius {
                // Threat decreases with distance (can still reach but less likely)
                falloff := 1.0 - (float64(distance) / float64(radius+1))
                m.threatByPos[pos] += threat * falloff
            }
        }
    }
}
```

### Ranged Pressure Computation

```go
func (r *RangedThreatLayer) Compute() {
    r.pressureByPos = make(map[coords.LogicalPos]float64)
    r.lineOfFireZones = make(map[ecs.EntityID][]coords.LogicalPos)

    enemyFactions := r.getEnemyFactions()

    for _, enemyFactionID := range enemyFactions {
        squadIDs := combat.GetSquadsForFaction(enemyFactionID, r.manager)

        for _, squadID := range squadIDs {
            if !r.hasRangedUnits(squadID) {
                continue
            }

            squadPos := combat.GetSquadPosition(squadID, r.manager)
            maxRange := r.getMaxRangedRange(squadID)

            baseThreat := r.baseThreatManager.GetSquadThreatLevel(squadID)
            rangedDamage := baseThreat.ExpectedDamageByRange[maxRange]

            // Paint ranged pressure
            var zonePositions []coords.LogicalPos

            for dx := -maxRange; dx <= maxRange; dx++ {
                for dy := -maxRange; dy <= maxRange; dy++ {
                    pos := coords.LogicalPos{X: squadPos.X + dx, Y: squadPos.Y + dy}
                    distance := coords.ChebyshevDistance(squadPos, pos)

                    if distance <= maxRange && distance > 0 {
                        // Full threat within range
                        r.pressureByPos[pos] += rangedDamage
                        zonePositions = append(zonePositions, pos)
                    }
                }
            }

            r.lineOfFireZones[squadID] = zonePositions
        }
    }
}
```

### Support Value Computation

```go
func (s *SupportValueLayer) Compute() {
    s.healPriority = make(map[ecs.EntityID]float64)
    s.buffPriority = make(map[ecs.EntityID]float64)
    s.supportValuePos = make(map[coords.LogicalPos]float64)

    // Calculate heal priority for each allied squad
    squadIDs := combat.GetSquadsForFaction(s.factionID, s.manager)

    for _, squadID := range squadIDs {
        // Heal priority = inverse of health percentage
        avgHP := calculateAverageHP(squadID, s.manager)
        s.healPriority[squadID] = 1.0 - avgHP

        // Buff priority based on engagement state and abilities
        s.buffPriority[squadID] = s.calculateBuffPriority(squadID)

        // Paint support value around squad positions
        squadPos := combat.GetSquadPosition(squadID, s.manager)
        healPriority := s.healPriority[squadID]

        // Support value radiates from wounded squads
        supportRadius := 3  // Typical healing aura range
        for dx := -supportRadius; dx <= supportRadius; dx++ {
            for dy := -supportRadius; dy <= supportRadius; dy++ {
                pos := coords.LogicalPos{X: squadPos.X + dx, Y: squadPos.Y + dy}
                distance := coords.ChebyshevDistance(squadPos, pos)

                if distance <= supportRadius {
                    falloff := 1.0 - (float64(distance) / float64(supportRadius+1))
                    s.supportValuePos[pos] += healPriority * falloff
                }
            }
        }
    }
}

func (s *SupportValueLayer) calculateBuffPriority(squadID ecs.EntityID) float64 {
    // Higher priority if squad is about to engage
    distTracker := s.baseThreatManager.GetSquadDistanceTracker(squadID)
    if distTracker == nil {
        return 0.0
    }

    // Check if enemy is within 2 turns of engagement
    nearestEnemy := distTracker.GetNearestEnemy()
    if nearestEnemy.Distance <= 4 {  // Within 2 turns
        return 0.5
    }

    return 0.1
}
```

---

## Performance Considerations

### Layer Caching Strategy

```go
type LayerCache struct {
    round         int
    meleeThreat   *MeleeThreatLayer
    rangedThreat  *RangedThreatLayer
    supportValue  *SupportValueLayer
    positionalRisk *PositionalRiskLayer
}

func (c *CompositeThreatEvaluator) Update(currentRound int) {
    // Only recompute if round changed or marked dirty
    if c.lastUpdateRound == currentRound && !c.isDirty {
        return
    }

    // Parallel layer computation (layers are independent)
    var wg sync.WaitGroup

    wg.Add(4)
    go func() { c.meleeThreat.Compute(); wg.Done() }()
    go func() { c.rangedThreat.Compute(); wg.Done() }()
    go func() { c.supportValue.Compute(); wg.Done() }()
    go func() { c.positionalRisk.Compute(); wg.Done() }()

    wg.Wait()

    c.lastUpdateRound = currentRound
    c.isDirty = false
}
```

### Bounded Map Queries

```go
// Limit threat map to relevant combat area
func (m *MeleeThreatLayer) paintThreatRadius(
    center coords.LogicalPos,
    radius int,
    threat float64,
) {
    // Clamp to map bounds
    minX := max(0, center.X - radius)
    maxX := min(mapWidth-1, center.X + radius)
    minY := max(0, center.Y - radius)
    maxY := min(mapHeight-1, center.Y + radius)

    for x := minX; x <= maxX; x++ {
        for y := minY; y <= maxY; y++ {
            pos := coords.LogicalPos{X: x, Y: y}
            distance := coords.ChebyshevDistance(center, pos)

            if distance <= radius {
                falloff := 1.0 - (float64(distance) / float64(radius+1))
                m.threatByPos[pos] += threat * falloff
            }
        }
    }
}
```

---

## Implementation Phases

### Phase 1: Core Layer Infrastructure
1. Create `threat_layers.go` with layer interfaces and base types
2. Implement `MeleeThreatLayer` with basic computation
3. Implement `RangedThreatLayer` with basic computation
4. Create `CompositeThreatEvaluator` with layer combination
5. **Test:** Verify layer values are computed correctly

### Phase 2: Support and Positional Layers
1. Implement `SupportValueLayer` with heal priority
2. Implement `PositionalRiskLayer` with flanking/isolation detection
3. Integrate all layers into `CompositeThreatEvaluator`
4. **Test:** Verify composite scoring produces sensible values

### Phase 3: Role-Aware AI Controller
1. Create `ai_controller.go` with `DecideFactionTurn()`
2. Implement `ActionEvaluator` with role-specific evaluators
3. Integrate multi-layer threat queries into scoring
4. Hook into `TurnManager`
5. **Test:** AI makes role-appropriate decisions

### Phase 4: Advanced Tactics
1. Add ability usage evaluation (cooldowns, target selection)
2. Implement retreat logic using positional risk
3. Add formation-aware positioning
4. **Test:** AI uses abilities, retreats intelligently

### Phase 5: Polish and Tuning
1. Add AI difficulty settings (weight multipliers)
2. Implement decision logging for debugging
3. Balance weights via playtesting
4. **Test:** Multiple AI personalities, consistent challenge

---

## Success Criteria

- AI-controlled factions make decisions autonomously
- Tanks intercept threats and protect allies
- DPS squads engage at optimal ranges and focus fire
- Support squads maintain safe positions and heal wounded allies
- Ranged squads avoid melee danger zones
- All squads retreat when overwhelmed
- Layer queries are O(1) after initial computation
- System is extensible (easy to add new layers or role types)

---

## Files to Create/Modify

### Create:
- `tactical/behavior/threat_layers.go` - Layer interfaces and types
- `tactical/behavior/threat_melee.go` - Melee threat computation
- `tactical/behavior/threat_ranged.go` - Ranged pressure computation
- `tactical/behavior/threat_support.go` - Support value computation
- `tactical/behavior/threat_positional.go` - Positional risk computation
- `tactical/behavior/threat_composite.go` - Layer combination
- `tactical/behavior/multilayer_threat.go` - Manager wrapper
- `tactical/ai/ai_controller.go` - Main decision loop
- `tactical/ai/action_evaluator.go` - Action generation/scoring
- `tactical/ai/role_evaluators.go` - Role-specific logic
- `tactical/ai/utility_functions.go` - Scoring helpers

### Modify:
- `tactical/combatservices/combat_service.go` - Add multi-layer threat manager
- `tactical/combat/TurnManager.go` - Add AI hook in turn advancement
- `tactical/behavior/dangerlevel.go` - (Optional) Add helper queries

### Reference:
- `tactical/squadcommands/command.go` - Command interface
- `tactical/behavior/dangerlevel.go` - Existing threat data
- `tactical/squads/squadcomponents.go` - Role definitions

---

## Notes

- **Layers are complementary:** Each layer captures one aspect of tactical state
- **Role weights are tunable:** Adjust `DefaultRoleWeights` based on playtesting
- **Caching is critical:** Compute layers once per AI turn, not per query
- **Start with melee/ranged:** Support and positional layers add depth later
- **Debug with visualization:** Consider adding threat map rendering for tuning
