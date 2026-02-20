# AI Algorithm Architecture

**Last Updated:** 2026-02-20

Overview and index for TinkerRogue's AI decision-making, power evaluation, action selection, and configuration systems.

---

## Document Index

| Document | Purpose |
|----------|---------|
| **[AI Controller](AI_CONTROLLER.md)** | Turn orchestration, action evaluation, scoring algorithms, decision tree, edge cases |
| **[Power Evaluation](POWER_EVALUATION.md)** | Unit/squad/range power calculations, role multipliers, ability values, DirtyCache |
| **[AI Configuration](AI_CONFIGURATION.md)** | aiconfig.json, powerconfig.json, accessor patterns, tuning guide |
| **[Behavior & Threat Layers](BEHAVIOR_THREAT_LAYERS.md)** | Threat layer subsystems, spatial analysis, visualization, difficulty scaling |
| **[Encounter System](ENCOUNTER_SYSTEM.md)** | Encounter generation, combat lifecycle, rewards |

---

## Executive Summary

TinkerRogue's AI uses a **layered threat assessment** system combined with **role-based behavior weights** to create tactically-aware computer opponents. The architecture separates concerns into distinct subsystems:

- **AI Controller** (`mind/ai/`) - Orchestrates turn execution and action selection ([details](AI_CONTROLLER.md))
- **Threat Evaluation** (`mind/behavior/`) - Multi-layered spatial threat analysis ([details](BEHAVIOR_THREAT_LAYERS.md))
- **Power Calculation** (`mind/evaluation/`) - Unified combat power assessment ([details](POWER_EVALUATION.md))
- **Encounter Generation** (`mind/encounter/`) - Dynamic enemy creation based on power budgets ([details](ENCOUNTER_SYSTEM.md))
- **Configuration** (`assets/gamedata/`) - Data-driven weights and thresholds ([details](AI_CONFIGURATION.md))

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
|  * DecideFactionTurn()           [AI_CONTROLLER.md]          |
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
|  [BEHAVIOR_THREAT_LAYERS.md]                                |
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
|  [POWER_EVALUATION.md]                                      |
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
- **Config Templates**: JSON-loaded weights and parameters ([details](AI_CONFIGURATION.md))
- **GlobalDifficulty**: Runtime difficulty overlay applied to AI and encounter parameters

---

## System Initialization

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

## Performance Considerations

### Threat Layer Computation

**Complexity:** O(factions x squads x mapRadius^2)

**Optimization Techniques:**

1. **Dirty Flag Caching** - Layers only recompute when marked dirty. Start-of-turn update pays the full cost once.
2. **Map Reuse** - Threat maps use `clear()` instead of reallocating. Reduces GC pressure.
3. **Lazy Evaluator Creation** - CompositeThreatEvaluator created per-faction on-demand.
4. **Combat Query Cache (ECS Views)** - `ActionStateView` and `FactionView` avoid full-world queries.
5. **IterateMapGrid vs Sparse Painting** - `PositionalRiskLayer` iterates full grid (cache-friendly); `CombatThreatLayer` uses sparse painting (only around enemies).

**Bottlenecks:**

- **PaintThreatToMap()**: O(radius^2) per squad. Mitigated by small map sizes (30x30).
- **IterateMapGrid()**: O(mapWidth x mapHeight) per layer. 3 passes for PositionalRiskLayer = ~2700 callbacks at 30x30.
- **GetRoleWeightedThreat()**: Called per movement candidate. ~120 queries per AI turn (precomputed layers = O(1) lookup).

### Power Calculation

**Complexity:** O(units) per squad. See [Power Evaluation - Performance](POWER_EVALUATION.md#performance-considerations).

### ECS Query Patterns

1. **Use ECS Views for Repeated Queries**
   ```go
   actionState := cache.FindActionStateBySquadID(squadID)  // View-based, O(k)
   ```

2. **Component Access by EntityID**
   ```go
   data := common.GetComponentTypeByID[*SquadData](manager, squadID, SquadComponent)
   ```

3. **Avoid Entity Pointer Storage** - Always use EntityID and query on-demand.

4. **Batch Component Reads**
   ```go
   combatData := GetUnitCombatData(unitID, manager)
   ```

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

**Scaling:** 3 factions = ~162KB. Negligible compared to ECS entity data.

---

**End of Document**

For detailed information, follow the links in the [Document Index](#document-index) above.
