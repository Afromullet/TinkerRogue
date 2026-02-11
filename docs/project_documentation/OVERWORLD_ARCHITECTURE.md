# TinkerRogue Overworld Architecture

**Version:** 1.0
**Last Updated:** 2026-02-11
**Status:** Production

---

## Table of Contents

1. [Executive Summary](#executive-summary)
2. [Architecture Overview](#architecture-overview)
3. [Core Concepts](#core-concepts)
4. [Package Structure](#package-structure)
5. [Data Models & Components](#data-models--components)
6. [Key Systems](#key-systems)
7. [Map Generation](#map-generation)
8. [Faction AI](#faction-ai)
9. [Influence System](#influence-system)
10. [Travel & Encounters](#travel--encounters)
11. [Garrison System](#garrison-system)
12. [Victory Conditions](#victory-conditions)
13. [GUI Integration](#gui-integration)
14. [Configuration](#configuration)
15. [Code Paths & Flows](#code-paths--flows)
16. [Best Practices](#best-practices)

---

## Executive Summary

The **Overworld System** is TinkerRogue's strategic layer, providing a turn-based grand strategy experience where the player manages territory, combats NPC factions, and engages with dynamic threats. Built on a pure ECS architecture, the overworld simulates a living world where factions compete for territory, threats evolve over time, and player decisions have cascading consequences.

### Key Features

- **Turn-Based Simulation**: Discrete tick system advancing all game state
- **Dynamic Threats**: Self-evolving threat nodes with growth mechanics and influence zones
- **Faction AI**: Multiple NPC factions with strategic intent (Expand, Fortify, Raid, Retreat)
- **Influence Interactions**: Spatial relationships between nodes (synergy, competition, suppression)
- **Garrison Defense**: Strategic node defense with squad assignments
- **Data-Driven Design**: JSON-configured node types, factions, and balancing parameters
- **Recording System**: Full session replay export for analysis and debugging

### Design Philosophy

1. **Pure ECS**: Zero logic in components, all behavior in system functions
2. **Data-Driven**: Node types, factions, and mechanics configured via JSON
3. **Separation of Concerns**: Clear boundaries between simulation (overworld) and presentation (GUI)
4. **Unified Node Model**: Single `OverworldNodeComponent` for threats, settlements, and fortresses
5. **Event-Driven UI**: Dirty-checking minimizes redundant refreshes

---

## Architecture Overview

### System Boundaries

```
┌─────────────────────────────────────────────────────────────┐
│                    Overworld Layer                          │
│  ┌──────────────────────────────────────────────────────┐   │
│  │  Tick Manager (Orchestration)                        │   │
│  │    ├─ Travel System                                  │   │
│  │    ├─ Influence System                               │   │
│  │    ├─ Threat System (Growth & Evolution)             │   │
│  │    ├─ Faction System (AI & Territory)                │   │
│  │    ├─ Garrison System (Defense)                      │   │
│  │    └─ Victory System (Win/Loss Conditions)           │   │
│  └──────────────────────────────────────────────────────┘   │
│                           ↕                                  │
│  ┌──────────────────────────────────────────────────────┐   │
│  │  Core Package (Components, Types, Events)            │   │
│  │    ├─ Node Registry (Data-Driven Definitions)        │   │
│  │    ├─ Event Logging                                  │   │
│  │    └─ Recording System                               │   │
│  └──────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
                           ↕
┌─────────────────────────────────────────────────────────────┐
│              ECS Layer (common package)                      │
│    EntityManager | GlobalPositionSystem | Components        │
└─────────────────────────────────────────────────────────────┘
                           ↕
┌─────────────────────────────────────────────────────────────┐
│           GUI Layer (guioverworld package)                   │
│    OverworldMode | Renderer | Input/Action Handlers         │
└─────────────────────────────────────────────────────────────┘
```

### Major Components

| Component | Responsibility | Package |
|-----------|----------------|---------|
| **Tick Manager** | Orchestrates turn advancement, calls subsystems in sequence | `overworld/tick` |
| **Node System** | Creates and destroys overworld nodes (threats, settlements) | `overworld/node` |
| **Threat System** | Evolves threats, spawns child nodes, handles growth | `overworld/threat` |
| **Faction System** | NPC faction AI, territory management, raiding | `overworld/faction` |
| **Influence System** | Calculates spatial interactions between nodes | `overworld/influence` |
| **Garrison System** | Squad assignment, garrison defense encounters | `overworld/garrison` |
| **Travel System** | Player movement to threat nodes, combat initiation | `overworld/travel` |
| **Victory System** | Win/loss condition evaluation | `overworld/victory` |
| **Node Registry** | Data-driven node definitions from JSON | `overworld/core` |
| **Overworld GUI** | Rendering, input handling, panel management | `gui/guioverworld` |

---

## Core Concepts

### Unified Node Model

The overworld uses a **single unified component** (`OverworldNodeComponent`) for all node types:

- **Threats**: Hostile nodes owned by NPC factions (Necromancers, Bandits, Orcs, etc.)
- **Settlements**: Neutral or player-owned service nodes (Towns, Guild Halls, Temples)
- **Fortresses**: Player-owned defensive nodes (Watchtowers)

This unification allows:
- Single query for all overworld entities (`core.OverworldNodeView`)
- Consistent position/influence handling
- Owner-based filtering (hostile, neutral, player)

### Turn-Based Tick System

The overworld operates on **discrete ticks**:
- Each tick advances all subsystems in sequence
- Player actions (engage threat, advance tick, place node) consume ticks
- World state changes only on tick advancement

### Influence Zones

Every node projects an **influence zone** (Manhattan distance radius):
- **Threats**: Radius scales with intensity (`baseRadius + intensity`)
- **Settlements/Fortresses**: Fixed radius per node type

Overlapping influence zones create **interactions**:
- **Synergy**: Same-faction threats boost each other's growth
- **Competition**: Rival-faction threats slow each other
- **Suppression**: Player/neutral nodes slow nearby threats
- **Player Boost**: Player nodes synergize with each other

### Faction AI

NPC factions use **strategic intent scoring**:
1. Evaluate all possible intents (Expand, Fortify, Raid, Retreat)
2. Score each intent based on faction state (strength, territory size)
3. Apply faction archetype bonuses (Expansionist, Aggressor, Defensive, etc.)
4. Execute highest-scoring intent each tick

---

## Package Structure

```
overworld/
├── core/                       # Core types, components, events
│   ├── components.go          # ECS component definitions
│   ├── types.go               # Enums (FactionType, ThreatType, etc.)
│   ├── config.go              # Config accessors
│   ├── events.go              # Event logging and context
│   ├── init.go                # ECS subsystem registration
│   ├── utils.go               # Utility functions
│   ├── node_registry.go       # Data-driven node definitions
│   └── walkability.go         # Terrain walkability grid
│
├── node/                       # Node lifecycle (create/destroy/queries)
│   ├── system.go              # Node creation/destruction
│   ├── queries.go             # Node lookup queries
│   └── validation.go          # Placement validation
│
├── threat/                     # Threat-specific logic
│   ├── system.go              # Threat evolution, growth, spawning
│   └── queries.go             # Threat counting and stats
│
├── faction/                    # Faction AI
│   ├── system.go              # Intent execution, territory management
│   ├── scoring.go             # Intent scoring functions
│   └── archetype.go           # Faction archetypes and bonuses
│
├── influence/                  # Influence system
│   ├── system.go              # Interaction calculation and update
│   ├── queries.go             # Overlap detection
│   └── effects.go             # Interaction classification and modifiers
│
├── garrison/                   # Garrison system
│   ├── system.go              # Squad assignment, NPC garrison creation
│   └── queries.go             # Garrison lookup queries
│
├── travel/                     # Player travel system
│   └── system.go              # Travel initiation, progress, cancellation
│
├── victory/                    # Victory condition evaluation
│   ├── system.go              # Condition checking
│   └── queries.go             # Defeat condition queries
│
├── tick/                       # Tick orchestration
│   └── tickmanager.go         # Master tick advancement
│
└── overworldlog/               # Recording and export
    ├── overworld_recorder.go  # Event recording
    ├── overworld_summary.go   # Statistics aggregation
    └── overworld_export.go    # JSON export
```

---

## Data Models & Components

### Core Components

#### OverworldNodeData

**The unified node component** - represents any overworld node (threat, settlement, fortress).

```go
type OverworldNodeData struct {
    NodeID         ecs.EntityID  // Entity ID of this node
    NodeTypeID     string        // "necromancer", "town", "watchtower", etc.
    Category       NodeCategory  // "threat", "settlement", "fortress"
    OwnerID        string        // "player", "Neutral", "Necromancers", etc.
    EncounterID    string        // Empty for non-combat nodes
    Intensity      int           // 0 for settlements (growth level for threats)
    GrowthProgress float64       // 0.0 for non-growing nodes
    GrowthRate     float64       // 0.0 for settlements
    IsContained    bool          // Player presence slows growth
    CreatedTick    int64         // Tick when node was created
}
```

**Owner Semantics**:
- `"player"`: Player-owned nodes (settlements, fortresses)
- `"Neutral"`: Neutral settlements (towns, temples)
- `"Necromancers"`, `"Bandits"`, etc.: Faction-owned threats

**Category Filtering**:
```go
// Get all threats
for _, result := range core.OverworldNodeView.Get() {
    data := common.GetComponentType[*core.OverworldNodeData](...)
    if data.Category == core.NodeCategoryThreat {
        // Process threat
    }
}
```

#### InfluenceData

Cached influence radius and magnitude.

```go
type InfluenceData struct {
    Radius        int     // Tiles affected (Manhattan distance)
    BaseMagnitude float64 // Base effect strength
}
```

**Calculation**:
- Threats: `radius = baseRadius + intensity`, `magnitude = intensity * 0.1`
- Settlements: Fixed radius, fixed magnitude (0.1)

#### InteractionData

Records influence interactions with other nodes.

```go
type InteractionData struct {
    Interactions []NodeInteraction // All active interactions
    NetModifier  float64           // Combined growth modifier (1.0 = no effect)
}

type NodeInteraction struct {
    TargetID     ecs.EntityID
    Relationship InteractionType  // Synergy, Competition, Suppression, PlayerBoost
    Modifier     float64          // Additive modifier
    Distance     int
}
```

**NetModifier Calculation**:
- Base: `1.0`
- Synergy: `+0.25` per overlapping same-faction threat
- Competition: `-0.20` per overlapping rival-faction threat
- Suppression: `-0.40` per overlapping player node (scaled by node type)
- Stacking: Additive with diminishing returns (`factor *= 0.75` per additional interaction)

#### OverworldFactionData

Represents an NPC faction entity.

```go
type OverworldFactionData struct {
    FactionID     ecs.EntityID
    FactionType   FactionType   // Undead, Bandits, Orcs, etc.
    Strength      int           // Military power (1-20)
    TerritorySize int           // Number of tiles controlled
    Disposition   int           // -100 (hostile) to +100 (allied)
    CurrentIntent FactionIntent // Expand, Fortify, Raid, Retreat
    GrowthRate    float64       // Expansion speed
}
```

#### TerritoryData

Tiles controlled by a faction.

```go
type TerritoryData struct {
    OwnedTiles []coords.LogicalPosition
}
```

#### StrategicIntentData

Current faction objective.

```go
type StrategicIntentData struct {
    Intent         FactionIntent           // Current action
    TargetPosition *coords.LogicalPosition // Target tile (for raids/expansion)
    TicksRemaining int                     // Ticks until re-evaluation
    Priority       float64                 // Importance (0.0-1.0)
}
```

#### GarrisonData

Squads assigned to defend a node.

```go
type GarrisonData struct {
    SquadIDs []ecs.EntityID // Garrisoned squads
}
```

**Attached to node entities** - not all nodes have garrisons.

#### TravelStateData

Singleton tracking player travel.

```go
type TravelStateData struct {
    IsTraveling       bool
    Origin            coords.LogicalPosition
    Destination       coords.LogicalPosition
    TicksRemaining    int
    TargetThreatID    ecs.EntityID
    TargetEncounterID ecs.EntityID
}
```

#### TickStateData

Singleton global tick counter.

```go
type TickStateData struct {
    CurrentTick int64
    IsGameOver  bool
}
```

#### VictoryStateData

Victory condition tracking.

```go
type VictoryStateData struct {
    Condition         VictoryCondition  // None, PlayerWins, PlayerLoses, etc.
    TicksToSurvive    int64            // For survival victory
    TargetFactionType FactionType      // For faction elimination victory
    VictoryAchieved   bool
    DefeatReason      string
}
```

---

## Key Systems

### Node System (`overworld/node`)

**Responsibilities**:
- Create unified overworld nodes
- Destroy nodes (with proper cleanup)
- Validate node placement
- Query node positions and counts

**Key Functions**:

```go
// Create any node type (threat, settlement, fortress)
CreateNode(manager, CreateNodeParams{
    Position:         coords.LogicalPosition{X: 10, Y: 20},
    NodeTypeID:       "necromancer",  // From nodeDefinitions.json
    OwnerID:          "Necromancers", // Faction owner
    InitialIntensity: 1,
    EncounterID:      "necromancer_encounter",
    CurrentTick:      tickState.CurrentTick,
})

// Validate player node placement
ValidatePlayerPlacement(manager, pos, playerData)

// Query nodes by owner
CountNodesByOwner(manager, core.OwnerPlayer)
GetNodePositionsByOwner(manager, "Necromancers")
```

**Placement Rules** (from `node/validation.go`):
1. Must be on walkable terrain
2. No existing node at position
3. Within `MaxPlacementRange` (15 tiles) of player or existing player node
4. Under `MaxNodes` limit (10)

### Threat System (`overworld/threat`)

**Responsibilities**:
- Create threat nodes from faction actions
- Evolve threats (increase intensity)
- Spawn child threats (Necromancers, Corruption)
- Handle containment effects

**Growth Mechanics**:

```go
// Each tick
growthAmount = nodeData.GrowthRate  // e.g., 0.05 for Necromancers

// Apply containment (player presence)
if nodeData.IsContained {
    growthAmount *= 0.5  // 50% slowdown
}

// Apply influence interactions
if hasInteractions {
    growthAmount *= interactionData.NetModifier  // Synergy/Competition/Suppression
}

nodeData.GrowthProgress += growthAmount

// Evolution threshold
if nodeData.GrowthProgress >= 1.0 {
    EvolveThreatNode(...)  // Increase intensity, expand radius
    nodeData.GrowthProgress = 0.0
}
```

**Evolution Effects** (from `threat/system.go`):
- **Necromancer (intensity 3+)**: Spawns child necromancer node nearby
- **Corruption (always)**: Spreads to adjacent tile

**Child Spawning Algorithm**:
```go
// Try 10 random offsets within 3-tile radius
for attempts := 0; attempts < 10; attempts++ {
    offsetX := RandomInt(7) - 3  // -3 to +3
    offsetY := RandomInt(7) - 3
    newPos := parentPos + offset

    if IsTileWalkable(newPos) && !IsThreatAtPosition(newPos) {
        CreateThreatNode(manager, newPos, threatType, 1, currentTick)
        return true
    }
}
```

### Faction System (`overworld/faction`)

**Responsibilities**:
- Execute faction AI (intent scoring and execution)
- Manage faction territory (expand, abandon tiles)
- Spawn threats on owned tiles
- Launch raids on player/rival factions
- Create NPC garrisons

**Intent Scoring** (from `faction/scoring.go`):

Each intent is scored based on faction state, then the highest-scoring intent is executed.

**Expand Intent**:
```go
score = 0.0

// Favor expansion when strong (strength >= 7)
if strength >= 7 {
    score += 5.0
}

// Favor expansion when territory is small (< 20 tiles)
if territorySize < 20 {
    score += 3.0
}

// Penalize if at territory limit (>= 30 tiles)
if territorySize >= 30 {
    score -= 10.0
}

// Apply faction archetype bonus (e.g., Expansionist +3.0)
score += archetypeBonuses.ExpansionBonus
```

**Fortify Intent**:
```go
score = 0.0

// Favor fortify when weak (strength < 3)
if strength < 3 {
    score += 6.0
}

// Always some value to fortifying
score += 2.0

// Apply archetype bonus (e.g., Defensive +2.0)
score += archetypeBonuses.FortificationBonus
```

**Raid Intent**:
```go
score = 0.0

// Minimum strength required (>= 7)
if strength < 7 {
    return 0.0
}

// Raid if very strong (strength > 10)
if strength > 10 {
    score += 3.0
}

// Apply archetype bonus (e.g., Aggressor +4.0)
score += archetypeBonuses.RaidingBonus
```

**Retreat Intent**:
```go
score = 0.0

// Only retreat if critically weak (strength < 2)
if strength < 2 {
    score += 8.0
}

// Don't retreat if territory is small (<= 1 tile)
if territorySize <= 1 {
    score -= 5.0
}

// Apply archetype penalty (less likely to retreat)
score -= archetypeBonuses.RetreatPenalty
```

**Intent Execution**:

| Intent | Action | Effects |
|--------|--------|---------|
| **Expand** | Claim adjacent walkable tile | +1 territory, 20% chance spawn threat |
| **Fortify** | Increase strength, spawn threats | +1 strength, 30% chance spawn threat, 30% chance create NPC garrison |
| **Raid** | Attack player/rival faction | Spawn high-intensity threat OR trigger garrison defense |
| **Retreat** | Abandon outermost tile | -1 territory |
| **Idle** | Do nothing | No changes |

**Faction Archetypes** (from `faction/archetype.go`):

Defined in `overworldconfig.json` under `strategyBonuses`:

| Archetype | Description | Bonuses |
|-----------|-------------|---------|
| **Expansionist** | Aggressive territory expansion | Expansion +3.0, Raiding +1.0 |
| **Aggressor** | Combat-focused | Expansion +2.0, Raiding +4.0 |
| **Raider** | Pure raiding | Raiding +5.0, Retreat Penalty +2.0 |
| **Defensive** | Fortification-focused | Fortification +2.0, Retreat Penalty -2.0 |
| **Territorial** | Holds ground | Fortification +1.0, Expansion -1.0, Retreat Penalty +3.0 |

### Influence System (`overworld/influence`)

**Responsibilities**:
- Find overlapping influence zones
- Classify interactions (synergy, competition, suppression)
- Calculate modifiers
- Apply to threat growth rates

**Update Flow** (called once per tick before threat/faction updates):

```go
func UpdateInfluenceInteractions(manager, currentTick) {
    // 1. Clear stale interactions from previous tick
    clearStaleInteractions()

    // 2. Find all overlapping node pairs (O(N^2) scan)
    pairs := FindOverlappingNodes()  // Manhattan distance check

    // 3. For each pair: classify, calculate, record
    for pair in pairs {
        interactionType := ClassifyInteraction(pair.A, pair.B)
        modifier := CalculateInteractionModifier(interactionType, pair)

        // Add reciprocal interactions
        addInteraction(pair.A, {TargetID: pair.B, Type: interactionType, Modifier: modifier})
        addInteraction(pair.B, {TargetID: pair.A, Type: interactionType, Modifier: modifier})
    }

    // 4. Calculate NetModifier for each entity
    finalizeNetModifiers()  // Additive with diminishing returns
}
```

**Interaction Classification**:

```go
func ClassifyInteraction(entityA, entityB) InteractionType {
    aHostile := IsHostileOwner(dataA.OwnerID)
    bHostile := IsHostileOwner(dataB.OwnerID)

    // Both hostile (threat factions)
    if aHostile && bHostile {
        if dataA.OwnerID == dataB.OwnerID {
            return InteractionSynergy  // Same faction
        }
        return InteractionCompetition  // Rival factions
    }

    // Both friendly/neutral (player + neutral)
    if !aHostile && !bHostile {
        return InteractionPlayerBoost
    }

    // Mixed (friendly + hostile)
    return InteractionSuppression
}
```

**Modifier Calculation**:

| Interaction | Modifier | Formula |
|-------------|----------|---------|
| **Synergy** | +0.25 | Flat bonus |
| **Competition** | -0.20 | Flat penalty |
| **Suppression** | -0.40 | Base penalty × node type multiplier (e.g., Watchtower 1.5×) |
| **Player Boost** | +0.15 or +0.25 | Base or complementary bonus (e.g., Town + Guild Hall) |

**Stacking with Diminishing Returns**:

```go
// Group interactions by type
groups := {
    Synergy: [0.25, 0.25, 0.25],
    Suppression: [-0.40, -0.40]
}

// Apply diminishing returns per type (strongest first)
netEffect := 0.0
for _, modifiers := range groups {
    sort(modifiers)  // Strongest first
    factor := 1.0
    for _, mod := range modifiers {
        netEffect += mod * factor
        factor *= 0.75  // Diminishing factor
    }
}

NetModifier = 1.0 + netEffect
```

**Example**:
- Threat A overlaps with 2 same-faction threats and 1 player node
- Synergy: `0.25 + (0.25 * 0.75) = 0.4375`
- Suppression: `-0.40`
- NetModifier: `1.0 + 0.4375 - 0.40 = 1.0375` (net 3.75% growth boost)

### Garrison System (`overworld/garrison`)

**Responsibilities**:
- Assign player squads to nodes
- Remove squads from garrisons
- Create NPC garrisons for factions
- Find garrisoned nodes near faction territory

**Player Garrison Assignment**:

```go
func AssignSquadToNode(manager, squadID, nodeID) error {
    // 1. Validate node is player-owned
    // 2. Validate squad is not already garrisoned or deployed
    // 3. Add or update GarrisonComponent on node
    // 4. Mark squad as garrisoned (squad.GarrisonedAtNodeID = nodeID)
}
```

**Garrison Constraints**:
- Squad cannot be deployed in combat
- Squad cannot be garrisoned at multiple nodes
- Node must be player-owned

**NPC Garrison Creation** (called during Fortify intent):

```go
func CreateNPCGarrison(manager, nodeEntity, factionType, strength) {
    // 1. Determine squad composition (3-4 units)
    // 2. Scale power with faction strength
    // 3. Create squad using CreateSquadFromTemplate
    // 4. Add GarrisonComponent to node
}
```

**Raid Handling** (from `faction/system.go` → `gui/guioverworld/overworld_action_handler.go`):

```go
// Faction AI detects garrisoned player node during Raid intent
func ExecuteRaid(manager, factionEntity, factionData) *PendingRaid {
    playerNodes := garrison.FindPlayerNodesNearFaction(manager, faction.TerritoryTiles)

    for _, nodeID := range playerNodes {
        if garrison.IsNodeGarrisoned(manager, nodeID) {
            // Return pending raid for GUI to handle
            return &PendingRaid{
                AttackingFactionType: factionData.FactionType,
                AttackingStrength:    factionData.Strength,
                TargetNodeID:         nodeID,
                TargetNodePosition:   nodePos,
            }
        }
    }

    // No garrisoned target - spawn threat near border instead
    SpawnHighIntensityThreat(...)
}

// GUI overworld mode handles raid after tick advancement
func HandleRaid(raid *PendingRaid) {
    // 1. Create garrison defense encounter
    encounterID := encounter.TriggerGarrisonDefense(raid)

    // 2. Switch to combat mode
    encounterService.StartGarrisonDefense(encounterID, raid.TargetNodeID, playerEntityID)
}
```

**Garrison Defense Outcome** (resolved in combat, applied in overworld):
- **Victory**: Garrison squads survive, node ownership unchanged
- **Defeat**: Node ownership transfers to attacking faction, garrison squads destroyed

### Travel System (`overworld/travel`)

**Responsibilities**:
- Initiate player travel to threat nodes
- Advance travel progress each tick
- Cancel travel (return to origin)
- Trigger combat on arrival

**Travel Flow**:

```go
// 1. Player selects threat and presses Enter
func EngageThreat(nodeID) {
    // Create encounter entity
    encounterID := encounter.TriggerCombatFromThreat(manager, threatEntity)

    // Start travel
    travel.StartTravel(manager, playerData, threatPos, nodeID, encounterID)
}

// 2. Each tick during travel
func AdvanceTravelTick(manager, playerData) (bool, error) {
    travelState.TicksRemaining--

    if travelState.TicksRemaining <= 0 {
        // Move player to destination
        manager.MoveEntity(playerEntityID, currentPos, travelState.Destination)
        travelState.IsTraveling = false
        return true, nil  // Travel completed
    }

    return false, nil  // Still traveling
}

// 3. On arrival, start combat
func StartCombatAfterTravel() {
    encounterService.StartEncounter(
        travelState.TargetEncounterID,
        travelState.TargetThreatID,
        threatName,
        threatPos,
        playerEntityID,
    )
}
```

**Travel Time Calculation**:
```go
distance := playerPos.ManhattanDistance(threatPos)
movementSpeed := playerAttributes.MovementSpeed  // Default: 1
ticksNeeded := ceil(distance / movementSpeed)
```

**Auto-Travel Mode**:
- Enabled with `A` key during travel
- Automatically advances ticks until arrival
- Can be disabled with `A` key or canceled with `C`

### Victory System (`overworld/victory`)

**Responsibilities**:
- Check victory/defeat conditions each tick
- Export session recording on game end
- Handle multiple victory types

**Victory Conditions**:

| Condition | Check | Description |
|-----------|-------|-------------|
| **Threat Elimination** | `CountThreatNodes() == 0` | Default victory - all threats destroyed |
| **Survival** | `currentTick >= ticksToSurvive` | Survive N ticks (configured) |
| **Faction Defeat** | No factions of target type exist | Defeat specific faction |

**Defeat Conditions**:

| Condition | Check | Threshold |
|-----------|-------|-----------|
| **Threat Overwhelm** | `TotalThreatInfluence > maxInfluence` | Sum of all threat intensities > 100.0 |
| **High-Intensity Threats** | `CountHighIntensityThreats(4) >= 10` | 10+ threats at intensity 4+ |

**Evaluation Order** (from `victory/system.go`):
1. **Defeat** (highest priority)
2. **Survival Victory** (if configured)
3. **Threat Elimination Victory** (default)
4. **Faction Defeat Victory** (if configured)

**Session Recording**:
- Enabled via `config.ENABLE_OVERWORLD_LOG_EXPORT`
- Records all events (`LogEvent` calls)
- Exports JSON on victory/defeat/exit
- File format: `journey_YYYYMMDD_HHMMSS.json`

---

## Map Generation

### Strategic Overworld Generator (`world/worldmap/gen_overworld.go`)

**Algorithm**: Fractal Brownian Motion (fBm) noise with biome classification

**Generation Steps**:

1. **Multi-Octave Noise Generation**:
   ```go
   elevationMap := generateFBmMap(width, height, 0.035, 4 octaves)
   moistureMap := generateFBmMap(width, height, 0.045, 3 octaves)
   ```

2. **Continent Shaping**:
   - Apply radial distance falloff from center
   - Map edges trend toward water

3. **Biome Classification** (elevation × moisture):
   ```
   Elevation < 0.28  → Swamp (impassable)
   Elevation > 0.72  → Mountain (impassable)
   Moisture > 0.70   → Swamp (if elevation < 0.40)
   Elevation > 0.60 + Moisture < 0.35 → Desert
   Moisture > 0.55   → Forest
   Default           → Grassland
   ```

4. **Connectivity Verification**:
   - Flood-fill walkable regions
   - Carve corridors between disconnected regions
   - Convert carved tiles to grassland

5. **Faction Starting Positions**:
   - Divide map into 4 quadrant sectors
   - Find position with most walkable neighbors in each sector
   - Prefer grassland/forest biomes

6. **POI Placement** (terrain-aware):
   - **Towns**: Grassland or forest edge
   - **Temples**: Elevated terrain or desert (isolated)
   - **Watchtowers**: Elevated terrain near mountains
   - **Guild Halls**: Within 20 tiles of towns

**Default Configuration**:
```go
ElevationOctaves: 4
ElevationScale:   0.035
MoistureOctaves:  3
MoistureScale:    0.045
Persistence:      0.5
Lacunarity:       2.0

TownCount:        3
TempleCount:      2
GuildHallCount:   2
WatchtowerCount:  3
POIMinDistance:   12

FactionCount:      4
FactionMinSpacing: 25
```

**Walkability Grid**:
- Populated from biome map during generation
- Stored in `core.WalkableGrid` (global)
- Used for placement validation, faction expansion, child spawning

---

## Faction AI

### Archetype System

Faction behavior is driven by **archetype bonuses** applied during intent scoring.

**Archetype Definitions** (from `overworldconfig.json`):

```json
{
  "strategyBonuses": {
    "Expansionist": {
      "expansionBonus": 3.0,
      "fortificationBonus": 0.0,
      "raidingBonus": 1.0,
      "retreatPenalty": 0.0
    },
    "Aggressor": {
      "expansionBonus": 2.0,
      "fortificationBonus": 0.0,
      "raidingBonus": 4.0,
      "retreatPenalty": 0.0
    },
    "Raider": {
      "expansionBonus": 0.0,
      "fortificationBonus": 0.0,
      "raidingBonus": 5.0,
      "retreatPenalty": -2.0
    },
    "Defensive": {
      "expansionBonus": 0.0,
      "fortificationBonus": 2.0,
      "raidingBonus": 0.0,
      "retreatPenalty": 2.0
    },
    "Territorial": {
      "expansionBonus": -1.0,
      "fortificationBonus": 1.0,
      "raidingBonus": 0.0,
      "retreatPenalty": -3.0
    }
  }
}
```

**Intent Re-Evaluation**:
- Every `DefaultIntentTickDuration` ticks (10)
- Scores all intents, switches to highest-scoring

**Strength Thresholds** (from `overworldconfig.json`):
```json
{
  "strengthThresholds": {
    "weak": 3,      // Fortify if below this
    "strong": 7,    // Expand/Raid if above this
    "critical": 2   // Retreat if below this
  }
}
```

### Territory Management

**Expansion**:
```go
// Pick random owned tile
randomTile := faction.OwnedTiles[rand]

// Try to claim adjacent tile
for _, adj := range GetCardinalNeighbors(randomTile) {
    if IsTileWalkable(adj) && !IsTileOwnedByAnyFaction(adj) {
        faction.OwnedTiles = append(faction.OwnedTiles, adj)
        faction.TerritorySize++

        // 20% chance spawn threat on new tile
        if RandomInt(100) < 20 {
            SpawnThreatForFaction(manager, faction, adj)
        }
        return
    }
}
```

**Fortification**:
```go
// Increase strength
faction.Strength += 1

// 30% chance spawn threat on random owned tile
if RandomInt(100) < 30 {
    randomTile := faction.OwnedTiles[rand]
    SpawnThreatForFaction(manager, faction, randomTile)
}

// 30% chance create NPC garrison on ungarrisoned faction nodes
for _, node := range faction.Nodes {
    if !IsNodeGarrisoned(node) && RandomInt(100) < 30 {
        CreateNPCGarrison(manager, node, faction.Type, faction.Strength)
    }
}
```

**Raiding**:
```go
// Check for nearby garrisoned player nodes
playerNodes := garrison.FindPlayerNodesNearFaction(manager, faction.OwnedTiles)

for _, nodeID := range playerNodes {
    if garrison.IsNodeGarrisoned(manager, nodeID) {
        // Trigger garrison defense encounter
        return &PendingRaid{
            AttackingFactionType: faction.Type,
            AttackingStrength:    faction.Strength,
            TargetNodeID:         nodeID,
        }
    }
}

// No garrisoned target - spawn high-intensity threat
intensity := 3 + int(faction.Strength * 0.33)  // Scale with strength
SpawnThreatAtBorder(faction, intensity)
```

**Retreat**:
```go
// Abandon outermost tile (simplified)
if len(faction.OwnedTiles) > 1 {
    faction.OwnedTiles = faction.OwnedTiles[:len-1]
    faction.TerritorySize--
}
```

### Faction-Threat Mapping

Each faction type spawns specific threat types:

| Faction | Threat Type | Growth Rate | Radius | Special |
|---------|-------------|-------------|--------|---------|
| **Necromancers** | necromancer | 0.05 | 3 + intensity | Spawns children at tier 3+ |
| **Bandits** | banditcamp | 0.08 | 2 + intensity | Fast growth |
| **Orcs** | orcwarband | 0.07 | 3 + intensity | Combat debuff |
| **Beasts** | beastnest | 0.06 | 2 + intensity | - |
| **Cultists** | corruption | 0.03 | 5 + intensity | Spreads to adjacent tiles |

---

## Influence System

### Overlap Detection

**Algorithm**: O(N^2) pairwise distance check (acceptable for small node counts ~10-50)

```go
func FindOverlappingNodes(manager) []NodePair {
    nodes := collectAllNodes()  // Query OverworldNodeView once

    var pairs []NodePair
    for i := 0; i < len(nodes); i++ {
        for j := i+1; j < len(nodes); j++ {
            dist := nodes[i].Pos.ManhattanDistance(nodes[j].Pos)
            combinedRadii := nodes[i].Radius + nodes[j].Radius

            if dist <= combinedRadii {
                pairs = append(pairs, NodePair{A: nodes[i], B: nodes[j], Distance: dist})
            }
        }
    }
    return pairs
}
```

### Interaction Modifiers

**Synergy** (same-faction threats):
```go
modifier := +0.25  // Flat bonus
```

**Competition** (rival-faction threats):
```go
modifier := -0.20  // Flat penalty
```

**Suppression** (player/neutral nodes on threats):
```go
baseModifier := -0.40
nodeTypeMult := GetNodeTypeMultiplier(suppressorNode.NodeTypeID)  // From influenceconfig.json

modifier := baseModifier * nodeTypeMult
```

**Node Type Multipliers**:
```json
{
  "suppression": {
    "nodeTypeMultipliers": {
      "watchtower": 1.5,   // Watchtowers most effective
      "temple": 1.2,
      "guild_hall": 1.0,
      "town": 0.8          // Towns least effective
    }
  }
}
```

**Player Boost** (player nodes synergizing):
```go
baseBonus := 0.15

// Check if complementary pair (e.g., Town + Guild Hall)
if isComplementaryPair(nodeA.NodeTypeID, nodeB.NodeTypeID) {
    return 0.25  // Stronger bonus
}

return baseBonus
```

**Complementary Pairs** (from `influenceconfig.json`):
```json
{
  "playerSynergy": {
    "complementaryPairs": [
      ["town", "guild_hall"],
      ["guild_hall", "temple"],
      ["watchtower", "town"]
    ]
  }
}
```

### Stacking Formula

**Additive with Diminishing Returns**:

```go
netEffect := 0.0
diminishingFactor := 0.75  // From influenceconfig.json

// Group interactions by type
groups := make(map[InteractionType][]float64)
for _, interaction := range interactions {
    groups[interaction.Type] = append(groups[interaction.Type], interaction.Modifier)
}

// Apply diminishing returns per type
for _, modifiers := range groups {
    // Sort by absolute value descending (strongest first)
    sort.Slice(modifiers, func(i, j int) bool {
        return abs(modifiers[i]) > abs(modifiers[j])
    })

    factor := 1.0
    for _, mod := range modifiers {
        netEffect += mod * factor
        factor *= diminishingFactor
    }
}

NetModifier = 1.0 + netEffect
```

**Example Calculation**:

Threat A overlaps with:
- 3 same-faction threats (synergy)
- 1 rival-faction threat (competition)
- 2 player watchtowers (suppression)

```
Synergy group: [0.25, 0.25, 0.25]
  = 0.25 + (0.25 * 0.75) + (0.25 * 0.75 * 0.75)
  = 0.25 + 0.1875 + 0.140625
  = 0.578125

Competition group: [-0.20]
  = -0.20

Suppression group: [-0.60, -0.60]  (watchtower multiplier: -0.40 * 1.5)
  = -0.60 + (-0.60 * 0.75)
  = -0.60 - 0.45
  = -1.05

Total netEffect = 0.578125 - 0.20 - 1.05 = -0.671875
NetModifier = 1.0 + (-0.671875) = 0.328125

Growth rate = baseRate * 0.328125  (67% reduction)
```

---

## Travel & Encounters

### Travel Mechanics

**Initiation**:
1. Player selects threat node (mouse click)
2. Presses Enter to engage
3. System creates encounter entity via `encounter.TriggerCombatFromThreat`
4. Travel system calculates ticks needed and stores state

**Travel State Machine**:
```
[Idle] --Engage--> [Traveling] --Cancel--> [Idle]
                       |
                       +--Arrival--> [Combat]
```

**Tick Advancement**:
- Manual: Space key
- Auto-Travel: Automatic (toggle with A key)

**Cancellation**:
- Press C during travel
- Returns player to origin position
- Disposes encounter entity
- Resets travel state

### Encounter Creation

**Threat Encounters** (from `mind/encounter`):

```go
func TriggerCombatFromThreat(manager, threatEntity) (ecs.EntityID, error) {
    threatData := GetOverworldNodeData(threatEntity)

    // Create encounter entity
    encounterEntity := manager.World.NewEntity()

    // Add OverworldEncounterComponent
    encounterData := &OverworldEncounterData{
        Name:          threatData.NodeTypeID + " (Level " + threatData.Intensity + ")",
        Level:         threatData.Intensity,
        EncounterType: threatData.EncounterID,  // Random variant selected at threat creation
        IsDefeated:    false,
        ThreatNodeID:  threatData.NodeID,
    }
    encounterEntity.AddComponent(OverworldEncounterComponent, encounterData)

    return encounterEntity.GetID(), nil
}
```

**Garrison Defense Encounters**:

```go
func TriggerGarrisonDefense(manager, nodeID, attackingFaction, strength) (ecs.EntityID, error) {
    encounterEntity := manager.World.NewEntity()

    encounterData := &OverworldEncounterData{
        Name:                 attackingFaction.String() + " Raid",
        Level:                strength / 5,  // Scale with faction strength
        EncounterType:        MapFactionToEncounterType(attackingFaction),
        IsGarrisonDefense:    true,
        AttackingFactionType: attackingFaction,
        ThreatNodeID:         0,  // Not from specific threat
    }
    encounterEntity.AddComponent(OverworldEncounterComponent, encounterData)

    return encounterEntity.GetID(), nil
}
```

### Combat Resolution

**Threat Encounter Victory**:
1. Combat mode calls `overworld.HandleCombatVictory(threatNodeID)`
2. Threat node is destroyed via `threat.DestroyThreatNode`
3. Encounter entity marked as defeated
4. Player returns to overworld mode

**Threat Encounter Defeat**:
1. Player respawns (game-over handling in progress)

**Garrison Defense Victory**:
1. Garrison squads survive
2. Node ownership unchanged
3. Player returns to overworld mode

**Garrison Defense Defeat**:
1. Garrison squads destroyed (combat cleanup)
2. Node ownership transferred to attacking faction
3. Garrison component removed from node
4. Player returns to overworld mode

---

## Garrison System

### Constraints & Rules

**Squad Assignment Constraints**:
- Squad must not be deployed in combat (`!squad.IsDeployed`)
- Squad must not be garrisoned at another node (`squad.GarrisonedAtNodeID == 0`)
- Node must be player-owned (`nodeData.OwnerID == "player"`)

**NPC Garrison Creation**:
- Called during faction Fortify intent
- 30% chance per tick if node is ungarrisoned
- Squad power scales with faction strength

**Garrison Detection**:
```go
func FindPlayerNodesNearFaction(manager, factionTerritoryTiles) []ecs.EntityID {
    // Build set of faction tiles
    factionTileSet := make(map[LogicalPosition]bool)

    // Find player nodes adjacent to faction territory
    for _, playerNode := range AllPlayerNodes {
        for _, adj := range GetCardinalNeighbors(playerNode.Pos) {
            if factionTileSet[adj] {
                // Player node is adjacent to faction territory
                return playerNode.ID
            }
        }
    }
}
```

### Raid Encounter Flow

**Complete Flow** (faction AI → tick manager → GUI → combat):

```
1. Faction AI (faction.ExecuteRaid)
   ├─ Detect garrisoned player node near territory
   └─ Return PendingRaid struct to tick manager

2. Tick Manager (tick.AdvanceTick)
   ├─ Receive PendingRaid from faction update
   └─ Return TickResult with PendingRaid to GUI

3. GUI Overworld Mode (overworld_action_handler.go)
   ├─ Receive TickResult after tick advancement
   ├─ Call encounter.TriggerGarrisonDefense(raid)
   │  └─ Create garrison defense encounter entity
   └─ Call encounterService.StartGarrisonDefense(encounterID, nodeID)
       └─ Switch to combat mode

4. Combat Mode
   ├─ Load garrisoned squads as player units
   ├─ Spawn enemy units based on faction type and strength
   └─ Resolve combat

5. Combat Resolution
   ├─ Victory: Garrison squads survive, node ownership unchanged
   └─ Defeat: Garrison squads destroyed, node ownership transfers to faction
```

---

## Victory Conditions

### Condition Types

**Threat Elimination** (default):
```go
func HasPlayerEliminatedAllThreats(manager) bool {
    return threat.CountThreatNodes(manager) == 0
}
```

**Survival Victory**:
```go
if victoryState.TicksToSurvive > 0 {
    if currentTick >= victoryState.TicksToSurvive {
        return VictoryTimeLimit
    }
}
```

**Faction Defeat Victory**:
```go
func HasPlayerDefeatedFactionType(manager, factionType) bool {
    for _, faction := range AllFactions {
        if faction.Type == factionType {
            return false  // Faction still exists
        }
    }
    return true
}
```

### Defeat Conditions

**Threat Overwhelm**:
```go
totalInfluence := sum(threat.Intensity for all threats)
if totalInfluence > 100.0 {
    return VictoryPlayerLoses
}
```

**High-Intensity Threshold**:
```go
highCount := threat.CountHighIntensityThreats(manager, 4)
if highCount >= 10 {
    return VictoryPlayerLoses
}
```

### Recording Export

**On Victory/Defeat**:
```go
func FinalizeRecording(outcome, reason) error {
    record := recorder.Finalize(outcome, reason)

    // Aggregate statistics
    record.ThreatSummary = GenerateThreatSummary(events)
    record.FactionSummary = GenerateFactionSummary(events)
    record.CombatSummary = GenerateCombatSummary(events)

    // Export JSON
    filename := fmt.Sprintf("journey_%s.json", timestamp)
    return ExportOverworldJSON(record, exportDir)
}
```

**Record Structure**:
```json
{
  "session_id": "journey_20260211_143025.123",
  "start_time": "2026-02-11T14:30:25Z",
  "end_time": "2026-02-11T14:45:30Z",
  "start_tick": 0,
  "final_tick": 150,
  "total_ticks": 150,
  "outcome": "Victory",
  "outcome_reason": "All threats eliminated",
  "events": [
    {
      "index": 1,
      "tick": 0,
      "type": "Threat Spawned",
      "entity_id": 123,
      "description": "Necromancer spawned at (10, 20) with intensity 1",
      "data": {}
    }
  ],
  "threat_summary": {
    "total_spawned": 25,
    "total_destroyed": 25,
    "max_concurrent": 8,
    "evolutions": 15
  },
  "faction_summary": {
    "factions": [
      {
        "type": "Necromancers",
        "expansions": 12,
        "raids": 3,
        "peak_strength": 15
      }
    ]
  },
  "combat_summary": {
    "total_combats": 25,
    "victories": 25,
    "defeats": 0
  }
}
```

---

## GUI Integration

### Mode Architecture

**OverworldMode** (`gui/guioverworld/overworldmode.go`):
- Extends `framework.BaseMode`
- Manages UI state via `framework.OverworldState`
- Delegates actions to `OverworldActionHandler`
- Delegates input to `OverworldInputHandler`

**State Management**:
```go
type OverworldState struct {
    CameraX, CameraY    int          // Map camera position
    SelectedNodeID      ecs.EntityID // Currently selected node
    ShowInfluence       bool         // Toggle influence zone rendering
    IsAutoTraveling     bool         // Auto-advance ticks during travel
}
```

**Panel Registry** (from `overworld_panels_registry.go`):
- `OverworldPanelTickControls`: Advance Tick, Toggle Influence buttons
- `OverworldPanelThreatInfo`: Selected node details
- `OverworldPanelTickStatus`: Current tick and game status
- `OverworldPanelEventLog`: Recent events
- `OverworldPanelThreatStats`: Threat count and average intensity

### Rendering

**Layer Order** (front to back):
1. **Map Tiles** (background)
2. **Influence Zones** (if enabled)
3. **Nodes** (threats, settlements, fortresses)
4. **Player Avatar**
5. **Selection Highlight**

**Node Rendering**:
- **Threats**: Circles, radius scales with intensity (`8 + intensity * 2`), color from JSON
- **Settlements/Fortresses**: Squares with owner-colored border

**Influence Rendering**:
- Color by owner type:
  - Hostile: `RGBA{255, 200, 100, 50}` (warm orange)
  - Neutral: `RGBA{220, 200, 100, 40}` (muted yellow)
  - Player: `RGBA{100, 200, 255, 50}` (cool blue)

### Input Handling

**Mouse Input**:
- Click: Select node at cursor position
- Camera: Drag or arrow keys

**Keyboard Input**:
| Key | Action |
|-----|--------|
| **Space** | Advance tick |
| **Enter** | Engage selected threat (start travel) |
| **C** | Cancel travel |
| **A** | Toggle auto-travel |
| **I** | Toggle influence zone visibility |
| **Esc** | Exit overworld mode |

### Action Handler

**Responsibilities**:
- Advance ticks (call `tick.AdvanceTick`)
- Engage threats (call `travel.StartTravel`)
- Cancel travel (call `travel.CancelTravel`)
- Start combat after travel completion
- Handle pending raids (create garrison defense encounters)

**Event-Driven Refresh**:
- Dirty-check tick counter: Only refresh tick/stats panels when tick changes
- Dirty-check selection: Only refresh threat info when selection changes

---

## Configuration

### Overworld Config (`overworldconfig.json`)

**Threat Growth**:
```json
{
  "threatGrowth": {
    "containmentSlowdown": 0.5,      // Player presence penalty
    "maxThreatIntensity": 5,         // Cap on threat evolution
    "childNodeSpawnThreshold": 3     // Intensity for child spawning
  }
}
```

**Faction AI**:
```json
{
  "factionAI": {
    "defaultIntentTickDuration": 10, // Ticks between intent re-evaluation
    "expansionTerritoryLimit": 20,   // Favor expansion below this
    "fortificationStrengthGain": 1,  // Strength gained per fortify
    "maxTerritorySize": 30           // Hard limit on territory
  }
}
```

**Strength Thresholds**:
```json
{
  "strengthThresholds": {
    "weak": 3,                        // Fortify if below
    "strong": 7,                      // Expand/Raid if above
    "critical": 2                     // Retreat if below
  }
}
```

**Victory Conditions**:
```json
{
  "victoryConditions": {
    "highIntensityThreshold": 4,     // Intensity level considered "high"
    "maxHighIntensityThreats": 10,   // Defeat threshold
    "maxThreatInfluence": 100.0      // Defeat threshold (sum of intensities)
  }
}
```

**Spawn Probabilities**:
```json
{
  "spawnProbabilities": {
    "expansionThreatSpawnChance": 20, // % chance on territory expansion
    "fortifyThreatSpawnChance": 30,   // % chance on fortify
    "bonusItemDropChance": 30         // % chance for bonus loot
  }
}
```

**Player Nodes**:
```json
{
  "playerNodes": {
    "maxPlacementRange": 15,         // Max distance from player/nodes
    "maxNodes": 10                   // Max player-owned nodes
  }
}
```

### Node Definitions (`nodeDefinitions.json`)

**Threat Node Example**:
```json
{
  "id": "necromancer",
  "category": "threat",
  "displayName": "Necromancer",
  "color": { "r": 150, "g": 50, "b": 150, "a": 255 },
  "overworld": {
    "baseGrowthRate": 0.05,
    "baseRadius": 3,
    "primaryEffect": "SpawnBoost",
    "canSpawnChildren": true
  },
  "factionId": "Necromancers"
}
```

**Settlement Node Example**:
```json
{
  "id": "town",
  "category": "settlement",
  "displayName": "Marketplace",
  "color": { "r": 50, "g": 150, "b": 200, "a": 255 },
  "overworld": {
    "baseRadius": 1
  },
  "services": ["trade", "repair"]
}
```

### Influence Config (`influenceconfig.json`)

**Synergy/Competition/Suppression**:
```json
{
  "baseMagnitudeMultiplier": 0.1,
  "synergy": {
    "growthBonus": 0.25
  },
  "competition": {
    "growthPenalty": 0.20
  },
  "suppression": {
    "growthPenalty": 0.40,
    "nodeTypeMultipliers": {
      "watchtower": 1.5,
      "temple": 1.2,
      "guild_hall": 1.0,
      "town": 0.8
    }
  },
  "playerSynergy": {
    "baseBonus": 0.15,
    "complementaryBonus": 0.25,
    "complementaryPairs": [
      ["town", "guild_hall"],
      ["guild_hall", "temple"],
      ["watchtower", "town"]
    ]
  },
  "diminishingFactor": 0.75
}
```

---

## Code Paths & Flows

### Tick Advancement Flow

```
User presses Space
    ↓
gui/guioverworld/overworld_action_handler.go::AdvanceTick()
    ↓
overworld/tick/tickmanager.go::AdvanceTick(manager, playerData)
    ├─ Increment tick counter
    ├─ travel.AdvanceTravelTick(manager, playerData)
    │   └─ Returns travelCompleted=true if arrived
    ├─ influence.UpdateInfluenceInteractions(manager, tick)
    │   ├─ FindOverlappingNodes (O(N^2))
    │   ├─ ClassifyInteraction (synergy/competition/suppression)
    │   └─ finalizeNetModifiers (additive stacking)
    ├─ threat.UpdateThreatNodes(manager, tick)
    │   ├─ For each threat: GrowthProgress += GrowthRate * NetModifier
    │   ├─ If GrowthProgress >= 1.0: EvolveThreatNode
    │   └─ ExecuteThreatEvolutionEffect (spawn children, spread)
    ├─ faction.UpdateFactions(manager, tick)
    │   ├─ For each faction: Decrement IntentTicksRemaining
    │   ├─ If timer expired: EvaluateFactionIntent (score all intents)
    │   ├─ ExecuteFactionIntent (Expand/Fortify/Raid/Retreat)
    │   └─ Return PendingRaid if raid targets garrisoned node
    └─ Return TickResult{TravelCompleted, PendingRaid}
    ↓
Back to GUI handler
    ├─ If PendingRaid: HandleRaid (create garrison defense encounter)
    └─ If TravelCompleted: StartCombatAfterTravel
```

### Threat Engagement Flow

```
User clicks threat node, presses Enter
    ↓
gui/guioverworld/overworld_action_handler.go::EngageThreat(nodeID)
    ↓
mind/encounter/encounterservice.go::TriggerCombatFromThreat(manager, threatEntity)
    ├─ Create encounter entity
    ├─ Add OverworldEncounterComponent
    └─ Return encounterID
    ↓
overworld/travel/system.go::StartTravel(manager, playerData, threatPos, threatID, encounterID)
    ├─ Calculate ticksNeeded (distance / movementSpeed)
    ├─ Set TravelStateData{IsTraveling=true, TicksRemaining=N, ...}
    └─ Log travel start
    ↓
Each tick: travel.AdvanceTravelTick(manager, playerData)
    ├─ Decrement TicksRemaining
    ├─ If TicksRemaining <= 0:
    │   ├─ Move player to destination
    │   ├─ Set IsTraveling=false
    │   └─ Return travelCompleted=true
    ↓
GUI receives travelCompleted=true
    ↓
overworld_action_handler.go::StartCombatAfterTravel()
    ↓
encounterService.StartEncounter(encounterID, threatID, threatName, pos, playerEntityID)
    └─ Switch to combat mode
```

### Combat Resolution Flow

```
Combat ends (victory or defeat)
    ↓
If victory:
    ├─ mind/encounter/encounterservice.go::HandleCombatVictory(threatNodeID)
    │   └─ overworld/threat/system.go::DestroyThreatNode(manager, threatEntity)
    │       ├─ LogEvent(EventThreatDestroyed)
    │       └─ node.DestroyNode(manager, threatEntity)
    │           ├─ GlobalPositionSystem.RemoveEntity(threatID, pos)
    │           └─ manager.World.DisposeEntities(threatEntity)
    └─ Return to overworld mode

If defeat:
    ├─ Game over handling (in progress)
    └─ Return to overworld mode or main menu
```

### Garrison Defense Flow

```
Faction AI executes Raid intent
    ↓
faction.ExecuteRaid(manager, factionEntity, factionData)
    ├─ FindPlayerNodesNearFaction(manager, faction.TerritoryTiles)
    ├─ For each nearby player node:
    │   └─ If IsNodeGarrisoned(manager, nodeID):
    │       └─ Return &PendingRaid{FactionType, Strength, NodeID, NodePos}
    └─ If no garrisoned targets: Spawn high-intensity threat instead
    ↓
tick.AdvanceTick returns TickResult{PendingRaid: raid}
    ↓
GUI overworld_action_handler.go::HandleRaid(raid)
    ├─ encounter.TriggerGarrisonDefense(manager, raid.TargetNodeID, raid.FactionType, raid.Strength)
    │   ├─ Create encounter entity
    │   ├─ Add OverworldEncounterComponent{IsGarrisonDefense=true, AttackingFactionType=...}
    │   └─ Return encounterID
    └─ encounterService.StartGarrisonDefense(encounterID, nodeID, playerEntityID)
        └─ Switch to combat mode, load garrison squads vs faction units
    ↓
Combat ends
    ├─ If victory: Garrison squads survive, node ownership unchanged
    └─ If defeat:
        ├─ garrison.TransferNodeOwnership(manager, nodeID, factionType.String())
        │   ├─ nodeData.OwnerID = factionID
        │   ├─ Remove GarrisonComponent
        │   └─ LogEvent(EventNodeCaptured)
        └─ Garrison squads already disposed by combat cleanup
    ↓
Return to overworld mode
```

### Node Placement Flow

```
User selects node type, clicks map position
    ↓
node.ValidatePlayerPlacement(manager, pos, playerData)
    ├─ Check IsTileWalkable(pos)
    ├─ Check !IsAnyNodeAtPosition(manager, pos)
    ├─ Check CountPlayerNodes(manager) < MaxNodes
    └─ Check distance from player or existing player nodes <= MaxPlacementRange
    ↓
If valid:
    ↓
node.CreatePlayerNode(manager, pos, nodeTypeID, currentTick)
    ├─ node.CreateNode(manager, CreateNodeParams{...})
    │   ├─ Create entity
    │   ├─ Add PositionComponent
    │   ├─ Add OverworldNodeComponent
    │   ├─ Add InfluenceComponent
    │   └─ GlobalPositionSystem.AddEntity(entityID, pos)
    └─ LogEvent(EventPlayerNodePlaced)
```

---

## Best Practices

### ECS Patterns

**Always use EntityID, never *ecs.Entity pointers**:
```go
// ✅ CORRECT
squadIDs := []ecs.EntityID{123, 456, 789}

// ❌ WRONG
squadEntities := []*ecs.Entity{entity1, entity2, entity3}
```

**Query-based relationships, no caching**:
```go
// ✅ CORRECT - Query every time
func GetPlayerNodes(manager) []ecs.EntityID {
    var result []ecs.EntityID
    for _, qr := range core.OverworldNodeView.Get() {
        data := common.GetComponentType[*core.OverworldNodeData](qr.Entity, core.OverworldNodeComponent)
        if data != nil && data.OwnerID == core.OwnerPlayer {
            result = append(result, qr.Entity.GetID())
        }
    }
    return result
}

// ❌ WRONG - Cached in global variable
var cachedPlayerNodes []ecs.EntityID
```

**Use unified OverworldNodeComponent**:
```go
// ✅ CORRECT - Single unified query
for _, result := range core.OverworldNodeView.Get() {
    data := common.GetComponentType[*core.OverworldNodeData](result.Entity, core.OverworldNodeComponent)

    if data.Category == core.NodeCategoryThreat {
        // Process threat
    }
}

// ❌ WRONG - Separate queries for threats and settlements
for _, result := range manager.World.Query(ThreatNodeTag) { ... }
for _, result := range manager.World.Query(SettlementNodeTag) { ... }
```

**Component access patterns**:
```go
// From entity (when you have entity from query)
data := common.GetComponentType[*OverworldNodeData](entity, OverworldNodeComponent)

// By EntityID (when you only have ID)
data := common.GetComponentTypeByID[*OverworldNodeData](manager, entityID, OverworldNodeComponent)

// With existence check
if componentData, ok := manager.GetComponent(entityID, OverworldNodeComponent); ok {
    data := componentData.(*OverworldNodeData)
}
```

**Use CoordManager for indexing**:
```go
// ✅ CORRECT
idx := coords.CoordManager.LogicalToIndex(pos)
WalkableGrid[idx] = true

// ❌ WRONG - Width may differ from CoordManager.dungeonWidth
idx := y*width + x
WalkableGrid[idx] = true
```

### System Design

**Separate concerns**:
- **Core**: Data structures, types, events
- **Node**: Node lifecycle (create/destroy/validate)
- **Threat**: Threat-specific logic (growth, evolution)
- **Faction**: Faction AI (intent, territory)
- **Influence**: Spatial interactions
- **Tick**: Orchestration (calls subsystems in order)

**Delegate to subsystems**:
```go
// ✅ CORRECT - Tick manager delegates
func AdvanceTick(manager, playerData) {
    travel.AdvanceTravelTick(manager, playerData)
    influence.UpdateInfluenceInteractions(manager, tick)
    threat.UpdateThreatNodes(manager, tick)
    faction.UpdateFactions(manager, tick)
}

// ❌ WRONG - Tick manager contains threat logic
func AdvanceTick(manager, playerData) {
    // Inline threat evolution code here
    for _, threat := range threats {
        threat.GrowthProgress += threat.GrowthRate
        // ...
    }
}
```

**Event logging**:
```go
// ✅ CORRECT - Log significant events
core.LogEvent(core.EventThreatSpawned, currentTick, entityID,
    fmt.Sprintf("Necromancer spawned at (%d, %d) with intensity %d", pos.X, pos.Y, intensity),
    nil)

// Use data map for structured information
core.LogEvent(core.EventFactionExpanded, currentTick, factionID,
    fmt.Sprintf("%s expanded to (%d, %d)", factionType.String(), pos.X, pos.Y),
    map[string]interface{}{
        "faction_type":   factionType.String(),
        "territory_size": factionData.TerritorySize,
        "strength":       factionData.Strength,
    })
```

### Configuration

**Data-driven node definitions**:
- Add new threat types by editing `nodeDefinitions.json`
- No code changes required for new node types

**Tuning faction behavior**:
- Edit `overworldconfig.json` to adjust:
  - Intent scoring weights
  - Strength thresholds
  - Territory limits
  - Spawn probabilities

**Balancing influence**:
- Edit `influenceconfig.json` to adjust:
  - Synergy/competition/suppression strength
  - Node type multipliers
  - Complementary pairs
  - Diminishing returns factor

### Testing

**Unit tests**:
- `overworld/faction/scoring_test.go` - Intent scoring verification
- `overworld/travel/system_test.go` - Travel mechanics

**Manual testing checklist**:
- [ ] Threats evolve correctly (growth + intensity)
- [ ] Factions expand/fortify/raid/retreat as expected
- [ ] Influence interactions modify growth rates
- [ ] Garrison defense encounters trigger on raids
- [ ] Player nodes suppress nearby threats
- [ ] Travel completes after calculated ticks
- [ ] Combat victory destroys threats
- [ ] Garrison defeat transfers node ownership
- [ ] Victory/defeat conditions trigger correctly

---

## Appendix: Key Files Reference

### Core Files

| File | Purpose | Key Functions |
|------|---------|---------------|
| `overworld/core/components.go` | Component data structures | OverworldNodeData, InfluenceData, etc. |
| `overworld/core/types.go` | Enums and constants | ThreatType, FactionType, EventType |
| `overworld/core/node_registry.go` | Data-driven node definitions | GetNodeByID, GetEncounterForNode |
| `overworld/core/events.go` | Event logging and recording | LogEvent, StartRecordingSession |

### System Files

| File | Purpose | Key Functions |
|------|---------|---------------|
| `overworld/tick/tickmanager.go` | Tick orchestration | AdvanceTick, CreateTickStateEntity |
| `overworld/node/system.go` | Node lifecycle | CreateNode, CreatePlayerNode, DestroyNode |
| `overworld/threat/system.go` | Threat evolution | UpdateThreatNodes, EvolveThreatNode |
| `overworld/faction/system.go` | Faction AI | UpdateFactions, ExecuteFactionIntent |
| `overworld/influence/system.go` | Influence interactions | UpdateInfluenceInteractions |
| `overworld/garrison/system.go` | Garrison management | AssignSquadToNode, CreateNPCGarrison |
| `overworld/travel/system.go` | Travel mechanics | StartTravel, AdvanceTravelTick |
| `overworld/victory/system.go` | Victory conditions | CheckVictoryCondition |

### GUI Files

| File | Purpose | Key Functions |
|------|---------|---------------|
| `gui/guioverworld/overworldmode.go` | Mode coordination | Initialize, Update, Render |
| `gui/guioverworld/overworld_renderer.go` | Visual rendering | Render, renderNodes, renderInfluenceZones |
| `gui/guioverworld/overworld_action_handler.go` | Game-state actions | AdvanceTick, EngageThreat, HandleRaid |
| `gui/guioverworld/overworld_input_handler.go` | Input processing | HandleInput, handleMouseClick |

### Config Files

| File | Purpose |
|------|---------|
| `assets/gamedata/overworldconfig.json` | Core overworld parameters |
| `assets/gamedata/nodeDefinitions.json` | Node type definitions |
| `assets/gamedata/influenceconfig.json` | Influence interaction parameters |

---

## Conclusion

The TinkerRogue Overworld System is a data-driven, ECS-based strategic layer that provides emergent gameplay through faction AI, dynamic threats, and spatial influence mechanics. Its modular design separates simulation logic from presentation, allowing for robust testing and iterative balancing.

**Key Strengths**:
- Pure ECS architecture with zero logic in components
- Data-driven design enables easy content expansion
- Clear separation between overworld simulation and GUI
- Event-driven UI minimizes redundant refreshes
- Comprehensive recording system for analysis

**Future Enhancements**:
- Diplomacy system (faction alliances, player negotiations)
- Dynamic quest generation from overworld state
- Multiple victory paths (economic, diplomatic, military)
- Faction-specific mechanics (unique abilities per faction)
- Advanced terrain effects (fog of war, weather, seasons)

---

**Document Version:** 1.0
**Author:** Claude Sonnet 4.5
**Date:** 2026-02-11
**Reviewed By:** [Pending]
