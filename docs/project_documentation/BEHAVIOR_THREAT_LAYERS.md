# Behavior & Threat Layers

**Last Updated:** 2026-02-17

Technical reference for TinkerRogue's threat layer subsystems, spatial analysis, and visualization tools used by the AI decision-making system.

---

## Related Documents

- [AI Algorithm Architecture](AI_ALGORITHM_ARCHITECTURE.md) - Core AI controller, action selection, power evaluation, and configuration
- [Encounter System](ENCOUNTER_SYSTEM.md) - Encounter generation, lifecycle, and rewards

---

## Table of Contents

1. [Overview](#overview)
2. [Composite Threat Evaluator](#composite-threat-evaluator)
3. [Combat Threat Layer](#combat-threat-layer)
4. [Support Value Layer](#support-value-layer)
5. [Positional Risk Layer](#positional-risk-layer)
6. [Faction Threat Level Manager](#faction-threat-level-manager)
7. [Grid Utilities](#grid-utilities)
8. [Threat Visualizer](#threat-visualizer)
9. [Difficulty Scaling](#difficulty-scaling)
10. [Extension Points](#extension-points)
11. [Troubleshooting](#troubleshooting)
12. [File Reference](#file-reference)

---

## Overview

The threat layer system provides the spatial intelligence that drives AI positioning decisions. Each layer analyzes a different aspect of the battlefield, and the `CompositeThreatEvaluator` combines them using role-specific weights to produce tactical position scores.

**Integration with AI Controller:** The AI controller (documented in [AI Algorithm Architecture](AI_ALGORITHM_ARCHITECTURE.md)) calls `GetRoleWeightedThreat()` during movement scoring. Threat layers are updated once per faction turn and marked dirty after each action.

**Integration with Power System:** Threat layers consume power-by-range data from `FactionThreatLevelManager`, which uses the shared power calculation system (documented in [AI Algorithm Architecture](AI_ALGORITHM_ARCHITECTURE.md#power-evaluation-system)).

---

## Composite Threat Evaluator

**Location:** `mind/behavior/threat_composite.go`

**Purpose:** Combines multiple threat layers with role-specific weights to produce tactical position scores.

**Architecture:**

```
CompositeThreatEvaluator
|
+- CombatThreatLayer (unified melee + ranged)
|  +- Melee threat (linear falloff over move+attack range)
|  +- Ranged threat (no falloff, full threat at all ranges)
|
+- SupportValueLayer
|  +- Heal priority (inverse of squad health)
|  +- Ally proximity (count of nearby allies)
|
+- PositionalRiskLayer
   +- Flanking risk (attacked from multiple directions)
   +- Isolation risk (distance from nearest ally)
   +- Engagement pressure (normalized total threat)
   +- Retreat quality (low-threat adjacent tiles)
```

**Role-Weighted Threat Query:**

```go
GetRoleWeightedThreat(squadID, pos):
  role = GetSquadPrimaryRole(squadID)
  weights = GetRoleBehaviorWeights(role)  // From aiconfig.json (with difficulty scaling)

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
  CombatLayer.Compute(currentRound)

  // 2. Compute derived layers (depend on combat data)
  SupportLayer.Compute(currentRound)
  PositionalLayer.Compute(currentRound)

  // 3. Mark clean
  isDirty = false
  lastUpdateRound = currentRound
```

**Dirty Flag Management:**
- Marked dirty after each AI action (positions change)
- `MarkDirty()` cascades to all child layers
- Prevents redundant recomputation within same round
- Each layer tracks own dirty state via embedded `DirtyCache`

**Public Accessor Methods:**
- `GetCombatLayer() *CombatThreatLayer` - Direct access to combat threat data
- `GetSupportLayer() *SupportValueLayer` - Direct access to support value data
- `GetPositionalLayer() *PositionalRiskLayer` - Direct access to positional risk data
- `GetOptimalPositionForRole(squadID, candidatePositions)` - Returns lowest-threat position from candidates

---

## Combat Threat Layer

**Location:** `mind/behavior/threat_combat.go`

**Purpose:** Unified layer computing both melee and ranged threat from enemy squads.

**Data Structures:**

```go
type CombatThreatLayer struct {
  *ThreatLayerBase

  // Melee threat data
  meleeThreatByPos   map[LogicalPosition]float64  // Position -> melee threat

  // Ranged threat data
  rangedPressureByPos map[LogicalPosition]float64  // Position -> ranged pressure

  baseThreatMgr *FactionThreatLevelManager  // Source of ThreatByRange
}
```

**Note on lineOfFireZones:** Earlier versions tracked `lineOfFireZones` for visualization, but the current implementation passes `trackPositions=false` to `PaintThreatToMap` for both melee and ranged calculations. Visualization is now handled separately by `ThreatVisualizer`.

**Melee Threat Calculation:**

```go
computeMeleeThreat(squadID, squadPos, squadThreat):
  moveSpeed = GetSquadMovementSpeed(squadID)
  maxMeleeRange = getMaxRangeForAttackTypes(squadID, MeleeAttackTypes, 1)
  threatRadius = moveSpeed + maxMeleeRange

  // Use danger at range 1 (includes role multipliers from power system)
  totalThreat = squadThreat.ThreatByRange[1]

  // Paint threat with linear falloff (trackPositions=false)
  PaintThreatToMap(
    meleeThreatByPos,
    squadPos,
    threatRadius,
    totalThreat,
    LinearFalloff,
    false
  )
```

**Ranged Threat Calculation:**

```go
computeRangedThreat(squadID, squadPos, squadThreat):
  maxRange = getMaxRangeForAttackTypes(squadID, RangedAttackTypes, 3)

  // Use danger at max range (includes role multipliers)
  rangedDanger = squadThreat.ThreatByRange[maxRange]

  // Paint threat with NO falloff (archers equally dangerous at all ranges)
  // trackPositions=false; visualization handled by ThreatVisualizer
  PaintThreatToMap(
    rangedPressureByPos,
    squadPos,
    maxRange,
    rangedDanger,
    NoFalloff,
    false
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

## Support Value Layer

**Location:** `mind/behavior/threat_support.go`

**Purpose:** Identifies valuable positions for support squads (healers, buffers).

**Core Concept:**
- Wounded allies create "support value" radiating from their position
- Support squads attracted to high-value positions (negative weight)
- Ally proximity tracking helps all units avoid isolation

**Data Structures:**

```go
type SupportValueLayer struct {
  *ThreatLayerBase
  healPriority    map[EntityID]float64            // Squad -> heal urgency (0-1)
  supportValuePos map[LogicalPosition]float64     // Position -> support value
  allyProximity   map[LogicalPosition]int         // Position -> nearby ally count
}
```

**Computation:**

```go
Compute(currentRound):
  clear(healPriority, supportValuePos, allyProximity)

  squadIDs = GetActiveSquadsForFaction(factionID)

  for each squadID:
    // Calculate heal priority (inverse of health)
    avgHP = GetSquadHealthPercent(squadID)  // Centralized calculation from squads package
    healPriority[squadID] = 1.0 - avgHP

    squadPos = GetSquadMapPosition(squadID)

    // Paint support value around wounded allies
    healRadius, proximityRadius = GetSupportLayerParams()  // From config (with difficulty scaling)
    PaintThreatToMap(supportValuePos, squadPos, healRadius, healPriority, LinearFalloff, false)

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
```

**Note:** `GetMostDamagedAlly()` is NOT present in the current implementation. The support layer exposes only the two query functions above.

**Role Behavior:**
- **Support squads**: Negative weight (-1.0) attracts them to high support value
- **Other roles**: Low positive weight (0.1-0.2) for minor heal consideration
- Creates emergent behavior: supports move toward wounded allies

---

## Positional Risk Layer

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
  *ThreatLayerBase
  flankingRisk       map[LogicalPosition]float64  // 0-1 (0=safe, 1=flanked)
  isolationRisk      map[LogicalPosition]float64  // 0-1 (0=supported, 1=isolated)
  engagementPressure map[LogicalPosition]float64  // 0-1 (normalized)
  retreatQuality     map[LogicalPosition]float64  // 0-1 (0=trapped, 1=safe exits)

  baseThreatMgr *FactionThreatLevelManager
  combatLayer   *CombatThreatLayer  // Dependency: reads melee/ranged data
}
```

**Flanking Risk Computation:**

```go
computeFlankingRisk(enemyFactions):
  threatDirections = map[LogicalPosition]map[int]bool  // pos -> set of attack angles

  for each enemyFaction:
    for each enemySquad:
      moveSpeed = GetSquadMovementSpeed(enemySquad)
      threatRange = moveSpeed + GetFlankingThreatRangeBonus()  // Config + difficulty offset

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
  threshold = GetIsolationThreshold()  // Config + difficulty offset
  maxDist = 8  // Internal constant (isolationMaxDistance)

  allyPositions = collect all allied squad positions

  // Iterates full map grid using IterateMapGrid() from threat_gridutils.go
  IterateMapGrid(func(pos):
    minDistance = distance to nearest ally

    if minDistance >= maxDist:
      isolationRisk[pos] = 1.0  // Fully isolated
    else if minDistance > threshold:
      // Linear gradient from threshold to maxDist
      isolationRisk[pos] = (minDistance - threshold) / (maxDist - threshold)
    else:
      isolationRisk[pos] = 0.0  // Well-supported
  )
```

**Engagement Pressure Computation:**

```go
computeEngagementPressure():
  maxPressure = 200  // Normalizer constant (engagementPressureMax)

  IterateMapGrid(func(pos):
    meleeThreat = CombatLayer.GetMeleeThreatAt(pos)
    rangedThreat = CombatLayer.GetRangedPressureAt(pos)

    totalPressure = meleeThreat + rangedThreat
    engagementPressure[pos] = min(totalPressure / maxPressure, 1.0)
  )
```

**Retreat Quality Computation:**

```go
computeRetreatQuality():
  retreatThreshold = GetRetreatSafeThreatThreshold()  // Config + difficulty offset

  IterateMapGrid(func(pos):
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
  )
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

## Faction Threat Level Manager

**Location:** `mind/behavior/dangerlevel.go`

**Purpose:** Base threat data source for all threat layers. Computes raw power-by-range for each squad.

**Architecture:**

```
FactionThreatLevelManager
|
+- FactionThreatLevel (per faction)
   +- SquadThreatLevel (per squad)
      +- ThreatByRange map[int]float64
```

**Data Structures:**

```go
type SquadThreatLevel struct {
  manager       *EntityManager
  cache         *CombatQueryCache
  squadID       EntityID
  ThreatByRange map[int]float64  // Range -> threat power
}
```

**Note:** The `SquadDistanceTracker` described in earlier versions of this document has been removed from the implementation. `SquadThreatLevel` only tracks `ThreatByRange`.

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

## Grid Utilities

**Location:** `mind/behavior/threat_gridutils.go`

**Purpose:** Centralized map iteration utilities used by positional and support layers.

**Functions:**

```go
// IterateMapGrid iterates over all tiles in map bounds, calling callback for each position.
// Used by PositionalRiskLayer to compute isolation risk, engagement pressure, and retreat quality.
func IterateMapGrid(callback GridIterator)

// IterateViewport iterates over tiles within a viewport around a center position.
// Used by ThreatVisualizer to only process visible tiles.
func IterateViewport(center LogicalPosition, viewportSize int, callback GridIterator)
```

**Why Centralized?**
- Multiple layers iterate the full map grid with the same pattern
- Single function ensures consistent bounds checking across all layers
- Viewport iteration enables efficient visualization without full-map scans

---

## Threat Visualizer

**Location:** `mind/behavior/threatvisualizer.go`

**Purpose:** Unified visualization system for debugging threat layers and danger projections in the game's combat view. Replaces the former separate `DangerVisualizer` and `LayerVisualizer`.

**Visualization Modes:**

```go
type VisualizerMode int
const (
    VisualizerModeThreat VisualizerMode = iota  // Danger projection from squads (red gradient)
    VisualizerModeLayer                          // Individual threat layer inspection
)
```

**Layer Modes (within VisualizerModeLayer):**

```go
type LayerMode int
const (
    LayerMelee              LayerMode = iota  // Orange gradient
    LayerRanged                               // Cyan gradient
    LayerSupport                              // Green gradient
    LayerPositionalFlanking                   // Yellow gradient
    LayerPositionalIsolation                  // Purple gradient
    LayerPositionalEngagement                 // Red-orange gradient
    LayerPositionalRetreat                    // Green gradient (high=good)
    LayerModeCount
)
```

**Data Structure:**

```go
type ThreatVisualizer struct {
    manager       *EntityManager
    gameMap       *GameMap
    threatManager *FactionThreatLevelManager

    evaluators map[EntityID]*CompositeThreatEvaluator  // Per-faction (for layer mode)

    // Faction cycling
    factionIDs       []EntityID  // All factions in combat
    viewFactionIndex int         // Index into factionIDs for currently viewed faction

    *DirtyCache
    isActive         bool
    mode             VisualizerMode
    layerMode        LayerMode
    currentFactionID EntityID
}
```

**Key API:**

```go
// Core control
Toggle()                                   // Enable/disable visualization
SetFactions(factionIDs []EntityID)         // Set factions available for cycling
SetEvaluators(map[EntityID]*CompositeThreatEvaluator)
CycleFaction()                             // Advance to next faction view
SetMode(mode VisualizerMode)               // Switch between threat/layer modes
CycleLayerMode()                           // Cycle through LayerMelee ... LayerPositionalRetreat

// Rendering
Update(currentFactionID, currentRound, playerPos, viewportSize)
ClearVisualization()

// Query
IsActive() bool
GetViewFactionID() EntityID
GetMode() VisualizerMode
GetLayerMode() LayerMode
GetLayerModeInfo() LayerModeInfo          // Name, Description, ColorKey for HUD display
```

**Threat Mode (VisualizerModeThreat):**
- Reads `ThreatByRange` from `FactionThreatLevelManager` (direct power lookup by distance)
- Uses Manhattan distance from squad to tile
- Colors: red gradient from 0.2 (low) to 0.9 (very high) based on thresholds 50/100/150

**Layer Mode (VisualizerModeLayer):**
- Reads from `CompositeThreatEvaluator` layers (melee, ranged, support, positional sub-layers)
- Normalizes raw values to 0-1 range (maxThreatValue=200 for melee/ranged, 1.0 for support)
- Each layer uses a distinct color gradient for visual differentiation
- Uses `IterateViewport()` to only render visible tiles

---

## Difficulty Scaling

All config accessor functions apply a `templates.GlobalDifficulty.AI()` overlay at runtime. This means the effective values differ from the JSON values based on the active difficulty setting:

```go
// Example: GetFlankingThreatRangeBonus()
base = aiconfig.json value (or default 3)
result = base + templates.GlobalDifficulty.AI().FlankingRangeBonusOffset

// Example: getSharedRangedWeight()
base = aiconfig.json value (or default 0.5)
result = base * templates.GlobalDifficulty.AI().SharedRangedWeightScale
```

Difficulty offsets/scales available:
- `FlankingRangeBonusOffset` - Added to `flankingThreatRangeBonus`
- `IsolationThresholdOffset` - Added to `isolationThreshold`
- `RetreatSafeThresholdOffset` - Added to `retreatSafeThreatThreshold`
- `SharedRangedWeightScale` - Multiplied with `sharedRangedWeight`
- `SharedPositionalWeightScale` - Multiplied with `sharedPositionalWeight`

All results are clamped to a minimum of 1 to prevent degenerate behavior.

For the full AI and power configuration reference, see [AI Algorithm Architecture - Configuration System](AI_ALGORITHM_ARCHITECTURE.md#configuration-system).

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
   func (ntl *NewThreatLayer) Compute(currentRound int) {
     clear(ntl.customData)

     // Compute threat values
     // Use IterateMapGrid() from threat_gridutils.go for full-map passes
     // Use PaintThreatToMap() from threat_painting.go for painted threats

     ntl.markClean(currentRound)  // ThreatLayerBase method
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

7. **Update ThreatVisualizer** (optional, for debugging)
   Add a new `LayerMode` constant and handle it in `getLayerValueAt()` and `getLayerGradientFunction()`.

---

## Troubleshooting

### AI Ignores Wounded Allies

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
5. Enable `ThreatVisualizer` in `VisualizerModeLayer` + `LayerSupport` mode

---

### Threat Layers Not Updating

**Symptoms:** AI makes decisions based on stale positions.

**Possible Causes:**
- Layers not marked dirty after actions
- Dirty flag not checked in Update()
- BaseThreatMgr not updating ThreatByRange

**Debug Steps:**
1. Verify `MarkDirty()` called after each action execution (in `DecideFactionTurn`)
2. Check dirty flag propagation - `CompositeThreatEvaluator.MarkDirty()` cascades to child layers
3. Confirm FactionThreatLevelManager.UpdateAllFactions() called at start of turn
4. Log ThreatByRange values (should change as squads move)

---

## File Reference

| File | Purpose | Key Functions |
|------|---------|---------------|
| `mind/behavior/threat_composite.go` | Layer composition | `GetRoleWeightedThreat()`, `Update()`, `MarkDirty()`, `GetCombatLayer()`, `GetSupportLayer()`, `GetPositionalLayer()`, `GetOptimalPositionForRole()` |
| `mind/behavior/threat_combat.go` | Melee/ranged threat | `Compute()`, `GetMeleeThreatAt()`, `GetRangedPressureAt()` |
| `mind/behavior/threat_support.go` | Support positioning | `Compute()`, `GetSupportValueAt()`, `GetAllyProximityAt()` |
| `mind/behavior/threat_positional.go` | Tactical risks | `Compute()`, `GetTotalRiskAt()`, `GetFlankingRiskAt()`, `GetIsolationRiskAt()`, `GetEngagementPressureAt()`, `GetRetreatQuality()` |
| `mind/behavior/threat_layers.go` | Base layer utilities | `ThreatLayerBase`, `getEnemyFactions()` |
| `mind/behavior/threat_constants.go` | Config accessors (difficulty-aware) | `GetRoleBehaviorWeights()`, `GetIsolationThreshold()`, `GetFlankingThreatRangeBonus()`, `GetRetreatSafeThreatThreshold()`, `GetSupportLayerParams()` |
| `mind/behavior/threat_painting.go` | Spatial threat painting | `PaintThreatToMap()`, `LinearFalloff`, `NoFalloff` |
| `mind/behavior/threat_queries.go` | Unit data queries | `GetUnitCombatData()`, `hasUnitsWithAttackType()`, `getMaxRangeForAttackTypes()`, `MeleeAttackTypes`, `RangedAttackTypes` |
| `mind/behavior/threat_gridutils.go` | Map iteration utilities | `IterateMapGrid()`, `IterateViewport()` |
| `mind/behavior/threatvisualizer.go` | Debug visualization | `ThreatVisualizer`, `Toggle()`, `SetFactions()`, `CycleFaction()`, `SetMode()`, `CycleLayerMode()`, `Update()` |
| `mind/behavior/dangerlevel.go` | Base threat tracking | `FactionThreatLevelManager`, `SquadThreatLevel`, `CalculateThreatLevels()` |

---

**End of Document**

For questions or clarifications, consult the source code or the [AI Algorithm Architecture](AI_ALGORITHM_ARCHITECTURE.md) document.
