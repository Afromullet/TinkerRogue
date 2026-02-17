# TinkerRogue Overworld Architecture

**Version:** 2.0
**Last Updated:** 2026-02-17
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
10. [Commander System](#commander-system)
11. [Garrison System](#garrison-system)
12. [Victory Conditions](#victory-conditions)
13. [GUI Integration](#gui-integration)
14. [Configuration](#configuration)
15. [Code Paths & Flows](#code-paths--flows)
16. [Best Practices](#best-practices)

---

## Executive Summary

The **Overworld System** is TinkerRogue's strategic layer, providing a turn-based grand strategy experience where the player manages commanders, combats NPC factions, and engages with dynamic threats. Built on a pure ECS architecture, the overworld simulates a living world where factions compete for territory, threats evolve over time, and player decisions have cascading consequences.

The overworld is organized as a strategic map: the player controls one or more **commanders** who move across the map, engage threats, place settlements, and manage garrisons. Each commander has limited movement points per turn. Pressing End Turn advances the global tick, which in turn triggers faction AI, threat evolution, and influence interactions.

### Key Features

- **Turn-Based Simulation**: Discrete tick system advancing all game state
- **Commander Movement**: Player moves named commanders across the overworld map each turn
- **Dynamic Threats**: Self-evolving threat nodes with growth mechanics and influence zones
- **Faction AI**: Multiple NPC factions with strategic intent (Expand, Fortify, Raid, Retreat)
- **Influence Interactions**: Spatial relationships between nodes (synergy, competition, suppression)
- **Garrison Defense**: Strategic node defense with squad assignments
- **Data-Driven Design**: JSON-configured node types, factions, and balancing parameters
- **Recording System**: Full session event export for analysis and debugging

### Design Philosophy

1. **Pure ECS**: Zero logic in components, all behavior in system functions
2. **Data-Driven**: Node types, factions, and mechanics configured via JSON
3. **Separation of Concerns**: Clear boundaries between simulation (`overworld/`) and presentation (`gui/guioverworld/`)
4. **Unified Node Model**: Single `OverworldNodeComponent` for threats, settlements, and fortresses
5. **Event-Driven UI**: Dirty-checking minimizes redundant panel refreshes

---

## Architecture Overview

### System Boundaries

```
┌─────────────────────────────────────────────────────────────────┐
│                      Overworld Layer                            │
│  ┌────────────────────────────────────────────────────────────┐ │
│  │  Tick Manager  (overworld/tick)                            │ │
│  │    ├─ Influence System  (overworld/influence)              │ │
│  │    ├─ Threat System     (overworld/threat)                 │ │
│  │    └─ Faction System    (overworld/faction)                │ │
│  └────────────────────────────────────────────────────────────┘ │
│                             ↕                                   │
│  ┌────────────────────────────────────────────────────────────┐ │
│  │  Core Package  (overworld/core)                            │ │
│  │    ├─ Node Registry  (data-driven node definitions)        │ │
│  │    ├─ Walkability Grid                                     │ │
│  │    ├─ Event Logging & Recording                            │ │
│  │    └─ ECS Component Registration                           │ │
│  └────────────────────────────────────────────────────────────┘ │
│                             ↕                                   │
│  ┌────────────────────────────────────────────────────────────┐ │
│  │  Support Packages                                          │ │
│  │    ├─ overworld/node      - Node lifecycle                 │ │
│  │    ├─ overworld/garrison  - Garrison management            │ │
│  │    └─ overworld/victory   - Victory condition evaluation   │ │
│  └────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────┘
                             ↕
┌─────────────────────────────────────────────────────────────────┐
│               ECS Layer  (common package)                       │
│    EntityManager | GlobalPositionSystem | Components            │
└─────────────────────────────────────────────────────────────────┘
                             ↕
┌─────────────────────────────────────────────────────────────────┐
│     Commander Layer  (tactical/commander)                       │
│    CommanderMovementSystem | Turn State | Roster                │
└─────────────────────────────────────────────────────────────────┘
                             ↕
┌─────────────────────────────────────────────────────────────────┐
│           GUI Layer  (gui/guioverworld)                         │
│    OverworldMode | OverworldRenderer | Input/Action Handlers    │
└─────────────────────────────────────────────────────────────────┘
```

### Major Components

| Component | Responsibility | Package |
|-----------|----------------|---------|
| **Tick Manager** | Orchestrates turn advancement, calls subsystems in sequence | `overworld/tick` |
| **Node System** | Creates and destroys overworld nodes (threats, settlements) | `overworld/node` |
| **Threat System** | Evolves threats, spawns child nodes, handles growth | `overworld/threat` |
| **Faction System** | NPC faction AI, territory management, raiding | `overworld/faction` |
| **Influence System** | Calculates spatial interactions between nodes | `overworld/influence` |
| **Garrison System** | Squad assignment, NPC garrison creation, ownership transfer | `overworld/garrison` |
| **Victory System** | Win/loss condition evaluation | `overworld/victory` |
| **Node Registry** | Data-driven node definitions from JSON | `overworld/core` |
| **Commander System** | Commander movement, turn state, roster management | `tactical/commander` |
| **Overworld GUI** | Rendering, input handling, panel management | `gui/guioverworld` |

---

## Core Concepts

### Unified Node Model

The overworld uses a **single unified component** (`OverworldNodeComponent`) for all node types:

- **Threats**: Hostile nodes owned by NPC factions (Necromancers, Bandits, Orcs, etc.)
- **Settlements**: Neutral or player-owned service nodes (Towns, Guild Halls, Temples)
- **Fortresses**: Player-owned defensive nodes (Watchtowers)

This unification allows:
- A single ECS View for all overworld entities (`core.OverworldNodeView`)
- Consistent position/influence handling
- Owner-based filtering (hostile, neutral, player)

### Turn-Based Tick System

The overworld is **turn-based**. A turn consists of:
1. The player moves their commanders and performs actions (engage threats, garrison management, recruiting)
2. The player presses **End Turn** (Space or Enter)
3. The global tick increments and all subsystems update in sequence

Commander movement points are consumed by movement during the player's turn and are restored on End Turn. The world state (threats, factions, influence) only changes when a tick advances.

### Commander Movement

Players control one or more **commanders** on the overworld map. Each commander has:
- A movement point budget (restored each turn from `Attributes.MovementSpeed`)
- A squad roster (squads can engage threats and be garrisoned)
- A position tracked in the ECS position system and `GlobalPositionSystem`

Movement cost is **Chebyshev distance** (diagonal movement costs the same as cardinal). Valid movement tiles are highlighted in blue when move mode is active.

### Influence Zones

Every overworld node projects an **influence zone** defined by a radius:
- **Threats**: Radius scales with intensity (`baseRadius + intensity`)
- **Settlements/Fortresses**: Fixed radius per node type

Overlapping influence zones create **interactions**:
- **Synergy**: Same-faction threats boost each other's growth
- **Competition**: Rival-faction threats slow each other
- **Suppression**: Player/neutral nodes slow nearby threats

### Faction AI

NPC factions use **strategic intent scoring**:
1. Evaluate all possible intents (Expand, Fortify, Raid, Retreat, Idle)
2. Score each intent based on faction state (strength, territory size)
3. Apply values from `overworldconfig.json` scoring tables
4. Execute highest-scoring intent each tick

---

## Package Structure

```
overworld/
├── core/                       # Core types, components, events
│   ├── components.go          # ECS component data definitions
│   ├── types.go               # Enums (FactionType, ThreatType, NodeCategory, etc.)
│   ├── config.go              # Config accessors (difficulty-scaled getters)
│   ├── events.go              # Event logging and recording context
│   ├── init.go                # ECS subsystem self-registration
│   ├── utils.go               # Utility functions (GetCurrentTick, GetCardinalNeighbors, etc.)
│   ├── node_registry.go       # Data-driven node/encounter definitions
│   ├── resources.go           # Resource cost helpers (CanAfford, SpendResources)
│   └── walkability.go         # Terrain walkability grid (WalkableGrid)
│
├── node/                       # Node lifecycle
│   ├── system.go              # CreateNode, DestroyNode
│   ├── queries.go             # Node lookup queries
│   └── validation.go          # Placement validation
│
├── threat/                     # Threat-specific logic
│   ├── system.go              # Threat evolution, growth, spawning, destruction
│   └── queries.go             # Threat counting and stats
│
├── faction/                    # Faction AI
│   ├── system.go              # Intent execution, territory management
│   ├── scoring.go             # Intent scoring functions
│   ├── scoring_test.go        # Scoring unit tests
│   └── archetype.go           # Faction archetypes and archetype bonuses
│
├── influence/                  # Influence system
│   ├── system.go              # Interaction calculation and NetModifier update
│   ├── queries.go             # Overlap detection (FindOverlappingNodes)
│   └── effects.go             # Interaction classification and modifier calculation
│
├── garrison/                   # Garrison system
│   ├── system.go              # Squad assignment, NPC garrison creation, ownership transfer
│   └── queries.go             # Garrison lookup queries
│
├── victory/                    # Victory condition evaluation
│   ├── system.go              # CheckVictoryCondition, CreateVictoryStateEntity
│   └── queries.go             # Defeat condition queries
│
├── tick/                       # Tick orchestration
│   └── tickmanager.go         # AdvanceTick - master tick advancement function
│
└── overworldlog/               # Recording and export
    ├── overworld_recorder.go  # Event accumulation
    ├── overworld_summary.go   # Statistics aggregation (threat/faction/combat summaries)
    └── overworld_export.go    # JSON export to file

tactical/commander/             # Commander system (not under overworld/)
├── components.go              # Commander ECS data (CommanderData, ActionState, Roster)
├── init.go                    # ECS subsystem self-registration
├── movement.go                # CommanderMovementSystem (GetValidMovementTiles, MoveCommander)
├── queries.go                 # Commander lookups (GetCommanderAt, GetCommanderData, etc.)
├── roster.go                  # GetPlayerCommanderRoster, roster helpers
├── turnstate.go               # EndTurn, StartNewTurn, OverworldTurnState
└── system.go                  # CreateCommander

world/worldmap/                 # Map generation (used by both overworld and tactical layers)
├── generator.go               # MapGenerator interface, generator registry
├── gen_overworld.go           # StrategicOverworldGenerator (fBm noise, biomes, POIs)
├── gen_rooms_corridors.go     # RoomsAndCorridorsGenerator (classic dungeon)
├── gen_cavern.go              # CavernGenerator (cellular automata caves)
├── gen_tactical_biome.go      # TacticalBiomeGenerator (biome-themed tactical maps)
├── gen_helpers.go             # Shared helpers (connectivity, carving, image selection)
├── biome.go                   # Biome enum and String()
├── dungeontile.go             # Tile struct and TileType
├── dungeongen.go              # GameMap struct, NewGameMap, tile manipulation
├── tileconfig.go              # POI type constants, asset path config
├── GameMapUtil.go             # Additional GameMap utilities
└── astar.go                   # A* pathfinding
```

---

## Data Models & Components

### Core Components

#### OverworldNodeData

**The unified node component** representing any overworld node (threat, settlement, fortress). All rendering, influence, and faction logic filters by `Category` and `OwnerID`.

```go
// overworld/core/components.go
type OverworldNodeData struct {
    NodeID         ecs.EntityID  // Entity ID of this node
    NodeTypeID     string        // "necromancer", "town", "watchtower", etc. (from JSON)
    Category       NodeCategory  // "threat", "settlement", "fortress"
    OwnerID        string        // "player", "Neutral", "Necromancers", etc.
    EncounterID    string        // Encounter variant ID (empty for non-combat nodes)
    Intensity      int           // 0 for settlements; threat evolution level for threats
    GrowthProgress float64       // Accumulated growth toward next evolution (0.0-1.0)
    GrowthRate     float64       // Base growth per tick (from nodeDefinitions.json)
    IsContained    bool          // Player presence slows growth
    CreatedTick    int64         // Tick when this node was created
}
```

**Owner Semantics**:
- `"player"`: Player-owned nodes (settlements, fortresses placed by the player)
- `"Neutral"`: Neutral settlements (towns, temples generated at map creation)
- Faction name strings (e.g., `"Necromancers"`, `"Bandits"`): Threat nodes owned by NPC factions

**Category Filtering**:
```go
// Query all overworld nodes and filter by category
for _, result := range core.OverworldNodeView.Get() {
    data := common.GetComponentType[*core.OverworldNodeData](result.Entity, core.OverworldNodeComponent)
    if data.Category == core.NodeCategoryThreat {
        // process threat
    }
}
```

**Owner Helpers**:
```go
core.IsHostileOwner(ownerID string) bool  // true if not "player" and not "Neutral"
core.IsFriendlyOwner(ownerID string) bool // true if == "player"
```

#### InfluenceData

Cached influence radius and magnitude per node. Updated when threats evolve.

```go
type InfluenceData struct {
    Radius        int     // Tiles affected (influences overlap detection)
    BaseMagnitude float64 // Base effect strength (intensity * BaseMagnitudeMultiplier)
}
```

- Threats: `Radius = baseRadius + intensity`, `BaseMagnitude = intensity * 0.1`
- Settlements: Fixed radius and magnitude from node definition

#### InteractionData

Interaction records accumulated each tick by the influence system.

```go
type InteractionData struct {
    Interactions []NodeInteraction // All active interactions with other nodes this tick
    NetModifier  float64           // Combined growth modifier (1.0 = no effect)
}

type NodeInteraction struct {
    TargetID     ecs.EntityID
    Relationship InteractionType  // InteractionSynergy, InteractionCompetition, InteractionSuppression
    Modifier     float64          // Signed additive modifier
    Distance     int              // Manhattan distance between nodes
}
```

`NetModifier` is recalculated each tick. A threat's effective growth rate is `baseGrowthRate * NetModifier`. See the [Influence System](#influence-system) section for the stacking formula.

#### OverworldFactionData

Represents an NPC faction entity. Factions are persistent across ticks and maintain territory and strategic intent.

```go
type OverworldFactionData struct {
    FactionID     ecs.EntityID
    FactionType   FactionType   // FactionNecromancers, FactionBandits, FactionOrcs, etc.
    Strength      int           // Military power (increases via Fortify)
    TerritorySize int           // Number of owned tiles
    Disposition   int           // -100 (hostile) to +100 (allied); default -50
    CurrentIntent FactionIntent // IntentExpand, IntentFortify, IntentRaid, IntentRetreat, IntentIdle
    GrowthRate    float64       // Expansion speed modifier
}
```

#### TerritoryData

Tiles claimed and owned by a faction.

```go
type TerritoryData struct {
    OwnedTiles []coords.LogicalPosition
}
```

#### StrategicIntentData

The current faction objective. Re-evaluated periodically.

```go
type StrategicIntentData struct {
    Intent         FactionIntent           // Current action being executed
    TargetPosition *coords.LogicalPosition // Optional target tile (raids, expansion)
    TicksRemaining int                     // Ticks until intent is re-evaluated
    Priority       float64                 // Importance score (0.0-1.0)
}
```

#### GarrisonData

Squads defending a node. Optional component—added to node entities on first garrison assignment, removed when all squads are withdrawn.

```go
type GarrisonData struct {
    SquadIDs []ecs.EntityID // Squads assigned to defend this node
}
```

#### TickStateData

Singleton entity tracking the global tick counter. Queried via `TickStateTag`.

```go
type TickStateData struct {
    CurrentTick int64 // Global tick counter, monotonically increasing
    IsGameOver  bool  // Prevents further tick advancement after victory/defeat
}
```

#### TravelStateData

Singleton for tracking in-progress player travel. Retained for potential future use; the current implementation uses direct commander movement rather than multi-tick travel.

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

#### OverworldEncounterData

Encounter metadata created when the player engages a threat or when a garrison is raided.

```go
type OverworldEncounterData struct {
    Name                 string       // e.g., "Necromancer (Level 3)"
    Level                int          // Difficulty level
    EncounterType        string       // Encounter type ID for spawn logic
    IsDefeated           bool         // Set to true after combat victory
    ThreatNodeID         ecs.EntityID // Link to overworld threat node (0 if garrison defense)
    IsGarrisonDefense    bool         // True if this is a garrison raid encounter
    AttackingFactionType FactionType  // Faction that initiated the raid
}
```

#### VictoryStateData

Optional singleton for configuring win/loss conditions. If absent, the default victory is eliminating all threats.

```go
type VictoryStateData struct {
    Condition         VictoryCondition  // None, PlayerWins, PlayerLoses, TimeLimit, FactionDefeat
    TicksToSurvive    int64            // Ticks required for survival victory (0 = disabled)
    TargetFactionType FactionType      // Faction to eliminate for FactionDefeat victory
    VictoryAchieved   bool
    DefeatReason      string
}
```

#### PendingRaid

Transient data propagated from faction AI through the tick manager to the GUI layer. Not stored in ECS—returned as part of `TickResult`.

```go
type PendingRaid struct {
    AttackingFactionType FactionType
    AttackingStrength    int
    TargetNodeID         ecs.EntityID
    TargetNodePosition   coords.LogicalPosition
}
```

### Commander Components

These live in `tactical/commander/` because commanders are shared between the overworld and tactical layers.

```go
// CommanderData - core identity
type CommanderData struct {
    CommanderID ecs.EntityID
    Name        string
    IsActive    bool
}

// CommanderActionStateData - per-turn tracking
type CommanderActionStateData struct {
    CommanderID       ecs.EntityID
    HasMoved          bool
    HasActed          bool
    MovementRemaining int  // Decremented by movement; restored by StartNewTurn
}

// OverworldTurnStateData - singleton turn counter
type OverworldTurnStateData struct {
    CurrentTurn int
    TurnActive  bool
}

// CommanderRosterData - on the player entity
type CommanderRosterData struct {
    CommanderIDs  []ecs.EntityID
    MaxCommanders int
}
```

---

## Key Systems

### Node System (`overworld/node`)

**Responsibilities**: Create and destroy overworld nodes with proper ECS and position system integration.

**Key Functions**:

```go
// Create any node type (threat, settlement, fortress)
entityID, err := node.CreateNode(manager, node.CreateNodeParams{
    Position:         coords.LogicalPosition{X: 10, Y: 20},
    NodeTypeID:       "necromancer",   // Must match a key in nodeDefinitions.json
    OwnerID:          "Necromancers",  // Faction owner string
    InitialIntensity: 1,
    EncounterID:      "necromancer_encounter",
    CurrentTick:      tickState.CurrentTick,
})

// Destroy a node (removes from position system and disposes ECS entity)
node.DestroyNode(manager, nodeEntity)
```

**Placement Validation** (`node/validation.go`):
1. Position must be walkable (`core.IsTileWalkable`)
2. No existing overworld node at the position
3. Distance from existing player nodes or player entity must be within `MaxPlacementRange` (15 tiles)
4. Player must not exceed `MaxNodes` limit (10 nodes)

### Threat System (`overworld/threat`)

**Responsibilities**: Create faction-specific threats, evolve them each tick, spawn child nodes, and handle destruction.

**Growth Mechanics** (per tick, in `UpdateThreatNodes`):

```go
// Base growth scaled by difficulty
growthAmount := nodeData.GrowthRate * templates.GlobalDifficulty.Overworld().ThreatGrowthScale

// Containment: player presence slows growth
if nodeData.IsContained {
    growthAmount *= core.GetContainmentSlowdown() // e.g., 0.5
}

// Influence interactions from this tick
if manager.HasComponent(entity.GetID(), core.InteractionComponent) {
    growthAmount *= interactionData.NetModifier
}

nodeData.GrowthProgress += growthAmount

// Evolution: when growth progress reaches 1.0, intensity increases
if nodeData.GrowthProgress >= 1.0 {
    EvolveThreatNode(manager, entity, nodeData)
    nodeData.GrowthProgress = 0.0
}
```

**Evolution Effects** (`ExecuteThreatEvolutionEffect`):
- **Necromancer**: Spawns a child necromancer node nearby when `intensity % childNodeSpawnThreshold == 0`
- **Corruption**: Spreads to a random adjacent walkable tile each evolution

**Child Spawning Algorithm** (`SpawnChildThreatNode`):
```
For up to 10 attempts:
    Pick random offset (-3 to +3 in X and Y)
    If new position is walkable AND no existing threat there:
        CreateThreatNode at that position with intensity 1
        Return true (success)
Return false (no valid position found)
```

**Resource Cost Checks** (enforced in `SpawnThreatForFaction`): Factions must have sufficient resources (from their `ResourceStockpile` component) to spawn new threats. The cost is defined per-node in `nodeDefinitions.json`.

### Faction System (`overworld/faction`)

**Responsibilities**: Run NPC faction AI each tick—evaluate intent, execute the chosen action, manage territory.

**Intent Evaluation** (`EvaluateFactionIntent`):

Each intent is scored based on the faction's current state. Scoring weights are read from `overworldconfig.json`:

| Intent | When Scored High | Key Signals |
|--------|-----------------|-------------|
| **Expand** | Faction is strong, territory is small | Strength >= strong threshold, TerritorySize < limit |
| **Fortify** | Faction is weak, needs to consolidate | Strength < weak threshold |
| **Raid** | Faction is very strong | Strength > raid minimum threshold |
| **Retreat** | Faction is critically weak | Strength < critical threshold |
| **Idle** | All scores below `IdleScoreThreshold` | Weak faction with no good options |

**Intent Execution** (`ExecuteFactionIntent`):

| Intent | Concrete Action |
|--------|----------------|
| **Expand** | Claim one adjacent walkable tile; spawn a threat there with some probability |
| **Fortify** | Increase `Strength` by 1; optionally spawn threats or create NPC garrisons on owned nodes |
| **Raid** | If a garrisoned player node is adjacent to territory, return a `PendingRaid`; otherwise spawn a high-intensity threat near territory border |
| **Retreat** | Remove one tile from `OwnedTiles` |
| **Idle** | Do nothing |

**Faction Resource Costs**: Threat spawning during faction actions requires the faction to afford the node cost from its own `ResourceStockpile`. If the faction cannot afford the spawn, it is skipped.

### Influence System (`overworld/influence`)

**Responsibilities**: Detect overlapping influence zones each tick and compute `NetModifier` for each node's growth.

**Update Flow** (called once per tick, before threat and faction updates):

```
1. Clear stale interactions from previous tick on all entities with InteractionComponent
2. Find all overlapping node pairs (O(N^2) pairwise Manhattan distance check)
3. For each overlapping pair:
   a. Classify the interaction (Synergy, Competition, or Suppression)
   b. Calculate the signed modifier
   c. Record the interaction on both entities (reciprocal)
4. Finalize NetModifier for each entity with interactions:
   - Group interactions by type
   - Keep top-2 strongest interactions per type group
   - Sum: NetModifier = 1.0 + sum(top-2 modifiers per type)
```

**Interaction Classification** (`ClassifyInteraction`):

```
Both nodes hostile (different faction owners) → InteractionCompetition
Both nodes hostile (same faction owner)       → InteractionSynergy
One hostile, one friendly/neutral             → InteractionSuppression
```

**NetModifier Stacking**:

The current implementation keeps the **top 2 strongest modifiers per interaction type** and sums them additively (no diminishing returns):

```go
// For each interaction type group (Synergy, Competition, Suppression):
//   Sort by absolute value descending
//   Take first two
//   Sum them into netEffect
NetModifier = 1.0 + netEffect
```

This means a threat surrounded by many synergistic allies gains at most 2 synergy stacks, and player suppression from multiple nodes is capped at the 2 strongest effects.

**Example**: A threat overlapping with 3 same-faction threats (synergy +0.25 each) and 1 player watchtower (suppression −0.40):
- Synergy: top-2 = 0.25 + 0.25 = 0.50
- Suppression: top-2 = −0.40 (only one)
- NetModifier = 1.0 + 0.50 − 0.40 = **1.10** (10% growth boost)

### Garrison System (`overworld/garrison`)

**Responsibilities**: Manage squad assignment to player-owned nodes, create NPC garrisons for factions, and transfer node ownership after garrison defeats.

**Player Garrison Assignment** (`AssignSquadToNode`):

Constraints:
1. Node must exist and be player-owned (`OwnerID == "player"`)
2. Squad must not be deployed in combat (`!squad.IsDeployed`)
3. Squad must not already be garrisoned elsewhere (`squad.GarrisonedAtNodeID == 0`)
4. No duplicate assignment to the same node

On success: `GarrisonComponent` is added to the node entity (or updated if it already exists), and `squad.GarrisonedAtNodeID` is set.

**NPC Garrison Creation** (`CreateNPCGarrison`):

Called during faction **Fortify** intent (30% chance per owned node per tick):
1. Determine squad composition (3–4 random units from the unit pool)
2. Scale squad level with faction strength (`1 + strength / 20`)
3. Create the squad via `squads.CreateSquadFromTemplate`
4. Attach `GarrisonComponent` to the node entity

**Ownership Transfer** (`TransferNodeOwnership`):

Called after a garrison defense combat defeat:
1. Updates `nodeData.OwnerID` to the attacking faction's string name
2. Removes `GarrisonComponent` from the node (garrison squads are already cleaned up by combat)
3. Logs `EventNodeCaptured`

---

## Map Generation

### Generator Registry

All map generators self-register using Go's `init()` mechanism and the `RegisterGenerator` function. The registry maps generator names to `MapGenerator` interface implementations:

```go
// world/worldmap/generator.go
type MapGenerator interface {
    Generate(width, height int, images TileImageSet) GenerationResult
    Name() string
    Description() string
}

// Retrieve a generator by name (falls back to "rooms_corridors" if not found)
gen := worldmap.GetGeneratorOrDefault("overworld")
```

**Registered Generators**:

| Name | Type | Use Case |
|------|------|----------|
| `"overworld"` | `StrategicOverworldGenerator` | Strategic world map with biomes and POIs |
| `"rooms_corridors"` | `RoomsAndCorridorsGenerator` | Classic dungeon layout |
| `"cavern"` | `CavernGenerator` | Organic caves for squad combat |
| `"tactical_biome"` | `TacticalBiomeGenerator` | Biome-themed tactical battle maps |

### Strategic Overworld Generator (`world/worldmap/gen_overworld.go`)

**Algorithm**: Fractal Brownian Motion (fBm) noise for elevation and moisture, followed by biome classification, connectivity verification, and terrain-aware POI placement.

**Generation Steps**:

1. **Multi-Octave fBm Noise**:
   - `elevationMap`: 4 octaves, scale 0.035, OpenSimplex noise
   - `moistureMap`: 3 octaves, scale 0.045, different seed
   - Both normalized to [0, 1]

2. **Continent Shaping**:
   - Apply radial distance falloff: `elevation *= 1.0 - (distFromCenter * 0.6)`
   - Map edges trend toward water (impassable terrain)

3. **Biome Classification** (elevation × moisture thresholds):

   | Condition | Biome | Passable |
   |-----------|-------|---------|
   | elevation < 0.28 | Swamp | No |
   | elevation > 0.72 | Mountain | No |
   | moisture > 0.70 AND elevation < 0.40 | Swamp | No |
   | elevation > 0.60 AND moisture < 0.35 | Desert | Yes |
   | moisture > 0.55 | Forest | Yes |
   | Default | Grassland | Yes |

4. **Connectivity Verification**:
   - Flood-fill to identify the largest connected walkable region
   - Carve L-shaped corridors from isolated walkable regions to the largest region
   - Carved tiles become Grassland biome

5. **Faction Starting Positions**:
   - Map divided into 4 corner sectors (offset 10 tiles from edges)
   - For each sector: find walkable tile with most walkable neighbors within 5-tile radius
   - Prefer Grassland or Forest biome

6. **Typed POI Placement** (terrain-aware, in order):
   - **Towns** (3 default): Grassland or Forest, minimum spacing 12 tiles
   - **Temples** (2 default): Elevation > 0.55 or Desert, spacing 15 tiles
   - **Watchtowers** (3 default): Elevation > 0.50, not Swamp/Mountain, spacing 10 tiles
   - **Guild Halls** (2 default): Within 20 tiles of a placed town, any walkable terrain

**Output** (`GenerationResult`):
- `Tiles`: Flat array of `*Tile`, indexed by `CoordManager.LogicalToIndex`
- `BiomeMap`: Flat array of `Biome`, same indexing
- `ValidPositions`: All walkable tile positions
- `POIs`: Typed point-of-interest positions with `NodeID` and `Biome`
- `FactionStartPositions`: One per map sector with position and biome

**Walkability Grid**: After generation, `core.WalkableGrid` must be populated from the `BiomeMap`. This is done during game setup, not inside the generator itself. The grid is used by the faction AI, commander movement, and node placement validation.

### Rooms and Corridors Generator (`gen_rooms_corridors.go`)

Classic roguelike algorithm:
- Generate up to `MaxRooms` rectangular rooms with collision detection
- Connect adjacent rooms with L-shaped corridors (randomly choosing horizontal-first or vertical-first)
- Used as the default fallback for dungeon maps

### Cavern Generator (`gen_cavern.go`)

Designed for squad-based tactical combat in cave environments:
- Seed guaranteed circular chambers in a 3×2 sector grid
- Random-fill inter-chamber space
- 6 iterations of cellular automata smoothing (5+ wall neighbors → wall)
- Re-stamp chambers at radius-1 to preserve interiors
- Widen narrow passages (expand any passage with < 8 walkable tiles in 5×5 neighborhood)
- Enforce 2-tile solid border
- Ensure full connectivity
- Place 2×2 pillar obstacles inside chambers for tactical cover
- Place two faction start positions on opposite sides (left zone vs. right zone)

### Tactical Biome Generator (`gen_tactical_biome.go`)

Creates biome-themed battle maps for encounter combat:
- Randomly selects a biome (Grassland, Forest, Desert, Mountain, Swamp)
- Biome-specific obstacle density and tactical features
- Cellular automata smoothing
- Biome-specific features: cover clusters, choke point columns, open area circles
- Ensures connectivity
- Clears a 5-tile radius spawn area in the map center

---

## Faction AI

### Intent Scoring

Intent scores are computed by `faction/scoring.go` using parameters from `overworldconfig.json`. The highest-scoring intent is adopted; if all scores fall below `IdleScoreThreshold`, the faction becomes idle.

**Scoring Parameters (configurable)**:

```json
{
  "factionScoring": {
    "expansion": {
      "strongBonus": 5.0,
      "smallTerritoryBonus": 3.0,
      "maxTerritoryPenalty": -10.0
    },
    "fortification": {
      "weakBonus": 6.0,
      "baseValue": 2.0
    },
    "raiding": {
      "strongBonus": 3.0,
      "veryStrongOffset": 10
    },
    "retreat": {
      "criticalWeakBonus": 8.0,
      "smallTerritoryPenalty": -5.0,
      "minTerritorySize": 1
    }
  },
  "factionScoringControl": {
    "idleScoreThreshold": 0.5,
    "raidBaseIntensity": 3,
    "raidIntensityScale": 0.33
  }
}
```

### Intent Execution Table

| Intent | Concrete Action | Probability | Effects |
|--------|----------------|-------------|---------|
| **Expand** | Claim adjacent walkable tile | Always (if space available) | +1 territory; chance to spawn threat |
| **Fortify** | Consolidate position | Always | +1 strength; chance spawn threat; 30% garrison faction nodes |
| **Raid** | Attack player/rival | If garrisoned node found | Return `PendingRaid`; otherwise spawn high-intensity threat |
| **Retreat** | Abandon outermost tile | Always (if territory > 1) | -1 territory |
| **Idle** | Nothing | — | No changes |

### Faction-to-Threat Mapping

The `NodeRegistry` resolves faction-to-threat type via the `FactionID` field in `nodeDefinitions.json`:

```go
// overworld/core/utils.go
func MapFactionToThreatType(factionType FactionType) ThreatType {
    return GetNodeRegistry().GetThreatTypeForFaction(factionType)
}
```

The actual growth rates, radii, and special behaviors are all defined in `nodeDefinitions.json`. The code constants (`ThreatNecromancer`, `ThreatBanditCamp`, etc.) exist only for IDE autocomplete and type safety.

### Threat Spawn Cost

Factions have a `ResourceStockpile` component. Each node type in `nodeDefinitions.json` has an optional `cost` field (Iron, Wood, Stone). `SpawnThreatForFaction` deducts the cost before spawning; if the faction cannot afford it, the spawn is skipped silently.

---

## Commander System

The commander system (`tactical/commander/`) implements the player-facing units on the overworld map.

### ECS Components

Each commander entity has:

| Component | Data |
|-----------|------|
| `CommanderComponent` | `CommanderData{Name, IsActive}` |
| `CommanderActionStateComponent` | `CommanderActionStateData{HasMoved, HasActed, MovementRemaining}` |
| `common.PositionComponent` | `coords.LogicalPosition` (tracked in `GlobalPositionSystem`) |
| `rendering.RenderableComponent` | Commander sprite image |
| `common.AttributeComponent` | `Attributes{MovementSpeed}` |
| `squads.SquadRosterComponent` | List of squad IDs under this commander |
| `spells.ManaComponent` | Mana pool for commander abilities |
| `spells.SpellBookComponent` | Available spells |

The **player entity** additionally holds `CommanderRosterComponent` listing all commander entity IDs.

### Movement System (`CommanderMovementSystem`)

```go
// tactical/commander/movement.go
type CommanderMovementSystem struct {
    manager   *common.EntityManager
    posSystem *common.PositionSystem
}
```

**Movement Cost**: Chebyshev distance from current position to target. Moving diagonally costs 1 movement point, the same as cardinal movement.

**MoveCommander**: Validates movement cost <= `MovementRemaining`, checks `CanMoveTo` (walkable + no other commander), then calls `manager.MoveEntity` and decrements `MovementRemaining`.

**GetValidMovementTiles**: Scans a square of radius `MovementRemaining` around the commander, filters by `CanMoveTo`, returns all reachable tiles. This result is cached in `OverworldState.ValidMoveTiles` for rendering.

**CanMoveTo Checks**:
1. `core.IsTileWalkable(targetPos)` — checks `core.WalkableGrid`
2. No other commander at the target tile (`GetCommanderAt`)

### Turn Management (`turnstate.go`)

**EndTurn** orchestrates the turn cycle:
1. Calls `tick.AdvanceTick` (advances global tick, runs all subsystems)
2. Increments `OverworldTurnStateData.CurrentTurn`
3. Calls `StartNewTurn` to reset all commanders

**StartNewTurn**: For each commander in the player's roster, resets `HasMoved`, `HasActed`, and sets `MovementRemaining = Attributes.MovementSpeed`.

### Commander Recruitment

Commanders are recruited during the overworld via `OverworldActionHandler.RecruitCommander`:
1. Must be on a player-owned settlement or fortress node
2. Costs gold from `PlayerData.ResourceStockpile`
3. Commander count must be below `CommanderRosterData.MaxCommanders`
4. New commander is created at the current commander's position and added to the player's roster

### Commander Rendering

In `OverworldRenderer.renderCommanders`, each commander is drawn with:
- Its `Renderable.Image` sprite at its logical position (converted to screen coordinates with camera offset)
- A colored border rectangle based on roster index (cyan for first, orange for second, etc.)
- A brighter white highlight border if it is the currently selected commander (`OverworldState.SelectedCommanderID`)

---

## Garrison System

### Player Garrison Workflow

```
Player selects node (must be player-owned) and a commander
Player presses G (or clicks Garrison in node menu)
    ↓
OverworldInputHandler.handleGarrison validates selection
    ↓
Shows garrison dialog with:
    - Currently garrisoned squads ("Click to REMOVE")
    - Available squads under selected commander ("Click to ASSIGN")
    ↓
Player selects a squad
    ↓
AssignSquadToNode OR RemoveSquadFromNode
    ↓
GarrisonComponent updated on node entity
SquadData.GarrisonedAtNodeID updated
LogEvent(EventGarrisonAssigned / EventGarrisonRemoved)
```

### Raid Flow

Garrison raids are triggered by the faction AI during the **Raid** intent:

```
faction.ExecuteRaid detects garrisoned player node near territory
    ↓
Returns PendingRaid to tick.AdvanceTick
    ↓
tick.AdvanceTick returns TickResult{PendingRaid: raid}
    ↓
OverworldActionHandler.EndTurn receives TickResult
    ↓
HandleRaid called:
    ├─ encounter.TriggerGarrisonDefense (creates encounter entity)
    └─ encounterService.StartGarrisonDefense (switches to combat mode)
        ↓
Combat resolves:
    ├─ Victory: garrison survives, node ownership unchanged
    └─ Defeat: garrison.TransferNodeOwnership(newOwner = factionType.String())
                node captured, GarrisonComponent removed
```

---

## Victory Conditions

### Condition Evaluation Order (`victory/system.go`)

Conditions are checked in priority order. The first match terminates the game:

1. **Defeat** (highest priority): Checked first via `CheckPlayerDefeat`
2. **Survival Victory**: If `VictoryState.TicksToSurvive > 0` and `currentTick >= TicksToSurvive`
3. **Threat Elimination Victory**: All threat nodes destroyed (`CountThreatNodes() == 0`)
4. **Faction Defeat Victory**: All factions of `TargetFactionType` eliminated

If no `VictoryStateData` entity exists, the default condition is threat elimination.

### Session Recording

The `overworldlog` package records all events during a session for post-game analysis:

- **`OverworldRecorder`**: Accumulates `EventRecord` entries; disabled by default, enabled via config
- **Recording starts**: When `tick.CreateTickStateEntity` creates the tick singleton
- **Recording ends**: On victory, defeat, or when the player exits the overworld (`FinalizeRecording`)
- **Export**: JSON file named `journey_YYYYMMDD_HHMMSS.json` containing all events plus aggregated summaries

---

## GUI Integration

### Context Switching

The game uses a `GameModeCoordinator` (`gui/framework/coordinator.go`) to manage two independent mode stacks:

- **Overworld context** (`ContextOverworld`): `OverworldMode`, `NodePlacementMode`
- **Tactical context** (`ContextTactical`): `ExplorationMode`, `CombatMode`, squad editor modes

Pressing ESC in the overworld returns to the tactical context (`EnterTactical("exploration")`). Engaging a threat or defending a garrison transitions from overworld to tactical context.

### OverworldMode Architecture

`OverworldMode` (`gui/guioverworld/overworldmode.go`) extends `framework.BaseMode` and owns:

| Field | Purpose |
|-------|---------|
| `state *framework.OverworldState` | Transient UI state (camera, selection, move mode) |
| `renderer *OverworldRenderer` | Renders map, nodes, commanders, selection |
| `actionHandler *OverworldActionHandler` | Executes game-state changes |
| `inputHandler *OverworldInputHandler` | Dispatches keyboard and mouse input |
| `subMenus *subMenuController` | Manages sub-menu panel visibility |
| `encounterService *encounter.EncounterService` | Starts encounters (combat context switch) |

**Dirty-Checking** in `Update()`:
- Tick display panels refresh only when `tickState.CurrentTick != lastTick`
- Threat info panel refreshes only when `state.SelectedNodeID != lastSelectedNode`
- This avoids redundant string formatting every frame

### OverworldState (UI State Only)

```go
// gui/framework/contextstate.go
type OverworldState struct {
    CameraX        int          // Camera offset (logical tiles)
    CameraY        int          // Camera offset (logical tiles)
    SelectedNodeID ecs.EntityID // Currently selected overworld node

    ShowInfluence bool          // Toggle influence zone overlay

    // Commander UI state
    SelectedCommanderID ecs.EntityID             // Currently selected commander
    InMoveMode          bool                     // Movement overlay active
    ValidMoveTiles      []coords.LogicalPosition // Cached reachable tiles
}
```

> This is **UI state only**. Game state (tick count, threats, positions) lives in ECS components.

### Panel Registry

Panels registered in `overworld_panels_registry.go`:

| Panel Constant | Content | Refresh Trigger |
|----------------|---------|-----------------|
| `OverworldPanelResources` | Gold / Iron / Wood / Stone | Tick change |
| `OverworldPanelThreatInfo` | Selected node details | Selection change |
| `OverworldPanelTickStatus` | Turn and tick counter, game status | Tick change |
| `OverworldPanelEventLog` | Recent events (prepended) | Every event |
| `OverworldPanelThreatStats` | Total threats, average intensity | Tick change |
| `OverworldPanelTickControls` | Button bar: Debug / Node / Move / Engage / End Turn / Return | Always visible |
| `OverworldPanelDebugMenu` | Sub-menu: End Turn, Toggle Influence, Start Random Encounter | Toggle |
| `OverworldPanelNodeMenu` | Sub-menu: Place Nodes, Garrison, Recruit | Toggle |

### Rendering Layer Order

`OverworldRenderer.Render` draws in this order (back to front):

1. **Map tiles** (`renderOverworldMap`): All tiles revealed in strategic view
2. **Influence zones** (`renderInfluenceZones`): Only if `state.ShowInfluence`; colored circles per node
3. **Valid movement tiles** (`renderValidMovementTiles`): Blue overlay when `state.InMoveMode`
4. **Nodes** (`renderNodes`): Threats as circles scaled by intensity; settlements/fortresses as squares with owner-colored borders
5. **Commanders** (`renderCommanders`): Sprite + colored roster border + white highlight for selected
6. **Selection highlight** (`renderSelectionHighlight`): Outline around selected node

**Coordinate Conversion** (in `OverworldRenderer`):
```go
// Camera offset applied before drawing
screenX := (pos.X - r.state.CameraX) * r.tileSize
screenY := (pos.Y - r.state.CameraY) * r.tileSize

// Reverse: screen coordinates to logical tile (for mouse clicks)
logicalX := (screenX / r.tileSize) + r.state.CameraX
logicalY := (screenY / r.tileSize) + r.state.CameraY
```

Note that the overworld renderer does **not** use `coords.CoordManager.LogicalToScreen`. It applies camera offset directly because the overworld uses a simple 1:1 tile-to-pixel scale without the viewport scaling used in the tactical layer.

### Input Handling

**Keyboard Bindings** (`OverworldInputHandler.HandleInput`):

| Key | Action |
|-----|--------|
| **ESC** | Cancel move mode if active; otherwise return to tactical context |
| **Space / Enter** | End turn (advance tick) |
| **M** | Toggle move mode for selected commander |
| **Tab** | Cycle to next commander in roster |
| **I** | Toggle influence zone visibility |
| **E** | Engage threat at commander's position (commander must be on same tile) |
| **G** | Open garrison management dialog for selected node |
| **R** | Recruit a new commander (requires player-owned settlement/fortress) |
| **S** | Open squad editor for selected commander |
| **N** | Enter node placement mode |
| **Left Click** | Select commander / threat / node at cursor; in move mode, move to clicked tile |

**Mouse Click Resolution Priority** (in move mode):
1. If clicked tile is in `ValidMoveTiles`: move selected commander there
2. Otherwise: exit move mode

**Mouse Click Resolution Priority** (not in move mode):
1. Commander at clicked position → select commander; also select any node at same tile
2. Threat at clicked position → select threat node
3. Any node at clicked position → select node
4. Empty tile → clear selection

---

## Configuration

### Difficulty Scaling

All overworld numeric thresholds are accessed via getter functions in `overworld/core/config.go` that apply difficulty multipliers from `templates.GlobalDifficulty.Overworld()`:

```go
GetContainmentSlowdown()        // Base * ContainmentSlowdownScale
GetMaxThreatIntensity()         // Base + MaxThreatIntensityOffset
GetExpansionThreatSpawnChance() // Base * SpawnChanceScale (clamped 0-100)
GetFortifyThreatSpawnChance()   // Base * SpawnChanceScale (clamped 0-100)
GetFortificationStrengthGain()  // Scaled, minimum 1
GetRaidBaseIntensity()          // Base * RaidIntensityScale
```

This ensures all difficulty adjustments flow through a single point, and the JSON config files store the base values.

### Key Config Files

| File | Purpose |
|------|---------|
| `assets/gamedata/nodeDefinitions.json` | Node type definitions (category, color, growth rate, radius, cost, faction) |
| `assets/gamedata/encounterdata.json` | Encounter definitions (encounter type, squad preferences, difficulty, faction) |
| `assets/gamedata/overworldconfig.json` | Core parameters (faction AI, threat growth, spawn probabilities, victory conditions) |
| `assets/gamedata/influenceconfig.json` | Influence parameters (synergy/competition/suppression modifiers, diminishing returns) |

### Node Definitions Schema

```json
{
  "id": "necromancer",
  "category": "threat",
  "displayName": "Necromancer Stronghold",
  "color": { "r": 150, "g": 50, "b": 150, "a": 255 },
  "overworld": {
    "baseGrowthRate": 0.05,
    "baseRadius": 3,
    "canSpawnChildren": true
  },
  "factionId": "Necromancers",
  "cost": { "iron": 5, "wood": 0, "stone": 0 }
}
```

```json
{
  "id": "town",
  "category": "settlement",
  "displayName": "Town",
  "color": { "r": 50, "g": 150, "b": 200, "a": 255 },
  "overworld": { "baseRadius": 1 }
}
```

---

## Code Paths & Flows

### End Turn Flow

```
Player presses Space or Enter
    ↓
OverworldInputHandler → actionHandler.EndTurn()
    ↓
commander.EndTurn(manager, playerData)
    ├─ tick.AdvanceTick(manager, playerData)
    │   ├─ tickState.CurrentTick++
    │   ├─ influence.UpdateInfluenceInteractions(manager, tick)
    │   │   ├─ Clear stale interactions
    │   │   ├─ FindOverlappingNodes (O(N²))
    │   │   ├─ ClassifyInteraction per pair
    │   │   └─ finalizeNetModifiers (top-2 stacking)
    │   ├─ threat.UpdateThreatNodes(manager, tick)
    │   │   ├─ growthAmount = GrowthRate * DifficultyScale * ContainmentFactor * NetModifier
    │   │   ├─ GrowthProgress += growthAmount
    │   │   └─ If GrowthProgress >= 1.0: EvolveThreatNode (intensity++, optional child spawn)
    │   └─ faction.UpdateFactions(manager, tick)
    │       ├─ Decrement TicksRemaining per faction
    │       ├─ If expired: EvaluateFactionIntent (score all intents)
    │       ├─ ExecuteFactionIntent (Expand/Fortify/Raid/Retreat/Idle)
    │       └─ Return PendingRaid if raid targets a garrisoned player node
    ├─ OverworldTurnState.CurrentTurn++
    └─ StartNewTurn (reset all commander MovementRemaining from Attributes.MovementSpeed)
    ↓
OverworldActionHandler.EndTurn receives TickResult
    ├─ state.ExitMoveMode()
    ├─ refreshAllPanels()
    └─ If tickResult.PendingRaid != nil: HandleRaid(raid)
        ├─ encounter.TriggerGarrisonDefense(manager, raid)
        └─ encounterService.StartGarrisonDefense(encounterID, nodeID)
```

### Commander Movement Flow

```
Player presses M (with a commander selected)
    ↓
inputHandler.toggleMoveMode()
    ├─ CommanderMovementSystem.GetValidMovementTiles(commanderID)
    │   └─ Returns all tiles within Chebyshev distance = MovementRemaining
    ├─ state.InMoveMode = true
    └─ state.ValidMoveTiles = tiles (rendered as blue overlay)

Player clicks a blue tile
    ↓
inputHandler.handleMouseClick (in move mode)
    ├─ ScreenToLogical converts click to logical position
    ├─ Check if position is in ValidMoveTiles
    └─ actionHandler.MoveSelectedCommander(targetPos)
        ↓
        CommanderMovementSystem.MoveCommander(cmdID, targetPos)
            ├─ movementCost = currentPos.ChebyshevDistance(targetPos)
            ├─ Check actionState.MovementRemaining >= movementCost
            ├─ manager.MoveEntity(cmdID, entity, oldPos, targetPos)
            └─ actionState.MovementRemaining -= movementCost
        ↓
        Update ValidMoveTiles (recalculate after move)
        If MovementRemaining == 0: ExitMoveMode
```

### Threat Engagement Flow

```
Player moves commander onto same tile as a threat node
Player presses E (or clicks "Engage" button)
    ↓
actionHandler.EngageThreat(state.SelectedNodeID)
    ├─ Validate: commander entity exists and has position
    ├─ Auto-discover: if SelectedNodeID == 0, GetThreatNodeAt(commander position)
    ├─ Validate: threat entity and data exist
    ├─ Validate: commander position == threat position (must be on same tile)
    ├─ encounter.TriggerCombatFromThreat(manager, threatEntity)
    │   └─ Creates OverworldEncounterData entity with ThreatNodeID, Level, EncounterType
    └─ encounterService.StartEncounter(encounterID, nodeID, threatName, pos, cmdID)
        └─ Switches context to tactical combat mode
            ↓
            Combat resolves
            ├─ Victory → threat.DestroyThreatNode (removes ECS entity and position)
            └─ Defeat → (game-over or retreat handling)
            ↓
            Return to overworld context
```

### Garrison Defense Flow

```
faction.ExecuteRaid detects garrisoned player node adjacent to territory
    ↓
Returns &PendingRaid{AttackingFactionType, AttackingStrength, TargetNodeID, TargetNodePosition}
    ↓
tick.AdvanceTick returns TickResult{PendingRaid: raid}
    ↓
OverworldActionHandler.EndTurn → HandleRaid(raid)
    ├─ encounter.TriggerGarrisonDefense(manager, nodeID, factionType, strength)
    │   └─ Creates OverworldEncounterData{IsGarrisonDefense=true, AttackingFactionType=...}
    └─ encounterService.StartGarrisonDefense(encounterID, nodeID)
        └─ Switches to combat mode with garrison squads vs. faction units
            ↓
            Combat resolves
            ├─ Victory: garrison squads survive; node ownership unchanged
            └─ Defeat: garrison.TransferNodeOwnership(nodeID, factionType.String())
                       nodeData.OwnerID updated; GarrisonComponent removed
            ↓
            Return to overworld context
```

---

## Best Practices

### ECS Patterns

**Always use `ecs.EntityID`, never `*ecs.Entity` pointers in data**:
```go
// Correct
garrisonData.SquadIDs = []ecs.EntityID{123, 456}

// Wrong
garrisonEntities = []*ecs.Entity{entity1, entity2}
```

**Use `OverworldNodeView` for all node queries**:
```go
// Correct: single unified view
for _, result := range core.OverworldNodeView.Get() {
    data := common.GetComponentType[*core.OverworldNodeData](result.Entity, core.OverworldNodeComponent)
    if data.Category == core.NodeCategoryThreat {
        // handle threat
    }
}

// Wrong: separate queries per category
for _, result := range manager.World.Query(ThreatNodeTag) { ... }
```

**Access components via `GetComponentType` (from entity) or `GetComponentTypeByID` (from ID)**:
```go
// From entity (inside a query loop)
data := common.GetComponentType[*core.OverworldNodeData](entity, core.OverworldNodeComponent)

// From EntityID only
data := common.GetComponentTypeByID[*core.OverworldNodeData](manager, entityID, core.OverworldNodeComponent)
```

**Use `CoordManager.LogicalToIndex` for all tile array indexing**:
```go
// Correct
idx := coords.CoordManager.LogicalToIndex(pos)
core.WalkableGrid[idx] = true

// Wrong: may differ from CoordManager.dungeonWidth
idx := y*width + x
```

### Overworld-Specific Patterns

**Influence NetModifier is reset each tick**: Do not cache it. It is recalculated fresh in `UpdateInfluenceInteractions` at the start of every tick.

**PendingRaid is transient**: It is returned through the call stack, not stored in ECS. The GUI layer must handle it immediately after `EndTurn` returns.

**Node ownership is a string, not an enum**: Use the helper functions rather than comparing raw strings:
```go
// Correct
core.IsFriendlyOwner(nodeData.OwnerID)  // true if OwnerID == "player"
core.IsHostileOwner(nodeData.OwnerID)   // true if not "player" and not "Neutral"

// Wrong
nodeData.OwnerID == "player"
```

**Don't store game state in OverworldState**: That struct holds UI-only state (camera, selection, move mode flags). Game state lives in ECS components.

```go
// Correct: UI state in OverworldState
state.SelectedCommanderID = commanderID

// Wrong: game data in OverworldState
state.CachedThreatCount = threat.CountThreatNodes(manager)
```

### Testing

- **Unit tests**: `overworld/faction/scoring_test.go` verifies intent scoring; `overworld/overworldlog/overworld_recorder_test.go` verifies recording
- `go test ./overworld/...` runs all overworld package tests
- `go test ./tactical/...` runs commander and tactical tests

**Manual testing checklist**:
- [ ] Threats grow and evolve correctly each turn
- [ ] Necromancers spawn child nodes at correct intensity thresholds
- [ ] Corruption spreads to adjacent tiles
- [ ] Factions expand into unclaimed territory
- [ ] Factions fortify and create NPC garrisons
- [ ] Faction raids trigger garrison defense encounters
- [ ] Node ownership transfers on garrison defeat
- [ ] Commander movement respects movement points and walkability
- [ ] Movement points restore on End Turn
- [ ] Commander recruitment requires gold and a player-owned node
- [ ] Influence zones visually appear and disappear with I key
- [ ] Synergistic threats grow faster; suppressed threats grow slower
- [ ] Victory/defeat conditions trigger and export the session log

---

## Appendix: Key Files Reference

### Core Overworld Files

| File | Key Exports |
|------|------------|
| `overworld/core/components.go` | `OverworldNodeData`, `InfluenceData`, `InteractionData`, `GarrisonData`, `TickStateData`, etc. |
| `overworld/core/types.go` | `NodeCategory`, `ThreatType`, `FactionType`, `FactionIntent`, `VictoryCondition`, `EventType`, owner constants |
| `overworld/core/node_registry.go` | `NodeRegistry`, `GetNodeRegistry`, `NodeDefinition`, `EncounterDefinition` |
| `overworld/core/init.go` | `OverworldNodeView`, `OverworldFactionView`, ECS subsystem registration |
| `overworld/core/walkability.go` | `WalkableGrid`, `IsTileWalkable`, `SetTileWalkable` |
| `overworld/core/utils.go` | `GetCurrentTick`, `GetTickState`, `GetCardinalNeighbors`, `GetThreatNodeAt`, `GetNodeAtPosition` |

### System Files

| File | Key Functions |
|------|--------------|
| `overworld/tick/tickmanager.go` | `AdvanceTick`, `CreateTickStateEntity` |
| `overworld/node/system.go` | `CreateNode`, `DestroyNode` |
| `overworld/threat/system.go` | `CreateThreatNode`, `UpdateThreatNodes`, `EvolveThreatNode`, `DestroyThreatNode`, `SpawnChildThreatNode`, `SpawnThreatForFaction` |
| `overworld/faction/system.go` | `CreateFaction`, `UpdateFactions`, `EvaluateFactionIntent`, `ExecuteFactionIntent`, `ExpandTerritory`, `FortifyTerritory`, `ExecuteRaid` |
| `overworld/influence/system.go` | `UpdateInfluenceInteractions` |
| `overworld/influence/queries.go` | `FindOverlappingNodes` |
| `overworld/influence/effects.go` | `ClassifyInteraction`, `CalculateInteractionModifier` |
| `overworld/garrison/system.go` | `AssignSquadToNode`, `RemoveSquadFromNode`, `CreateNPCGarrison`, `TransferNodeOwnership` |
| `overworld/victory/system.go` | `CheckVictoryCondition`, `CreateVictoryStateEntity` |

### Commander Files

| File | Key Functions |
|------|--------------|
| `tactical/commander/system.go` | `CreateCommander` |
| `tactical/commander/movement.go` | `MoveCommander`, `GetValidMovementTiles`, `CanMoveTo` |
| `tactical/commander/turnstate.go` | `EndTurn`, `StartNewTurn`, `GetOverworldTurnState`, `GetCommanderActionState` |
| `tactical/commander/roster.go` | `GetPlayerCommanderRoster` |
| `tactical/commander/queries.go` | `GetCommanderAt`, `GetCommanderData`, `GetAllCommanders` |

### GUI Files

| File | Key Responsibilities |
|------|---------------------|
| `gui/guioverworld/overworldmode.go` | Mode lifecycle (Initialize, Enter, Exit, Update, Render, HandleInput); dirty-check refresh |
| `gui/guioverworld/overworld_renderer.go` | `Render`, `renderOverworldMap`, `renderNodes`, `renderInfluenceZones`, `renderCommanders`, `renderValidMovementTiles`, `ScreenToLogical` |
| `gui/guioverworld/overworld_action_handler.go` | `EndTurn`, `MoveSelectedCommander`, `EngageThreat`, `HandleRaid`, `ToggleInfluence`, `AssignSquadToGarrison`, `RecruitCommander` |
| `gui/guioverworld/overworld_input_handler.go` | `HandleInput`, `handleMouseClick`, `toggleMoveMode`, `cycleCommander`, `handleGarrison`, `showGarrisonDialog` |
| `gui/guioverworld/overworld_panels_registry.go` | Panel constants, `init()` panel registration, widget accessor functions |
| `gui/guioverworld/overworld_formatters.go` | `FormatThreatInfo` and other display formatting functions |
| `gui/guioverworld/overworld_deps.go` | `OverworldModeDeps` dependency bundle |
| `gui/framework/contextstate.go` | `OverworldState`, `TacticalState` |
| `gui/framework/coordinator.go` | `GameModeCoordinator`, `EnterTactical`, `ReturnToOverworld` |

### Map Generation Files

| File | Key Exports |
|------|------------|
| `world/worldmap/generator.go` | `MapGenerator`, `GenerationResult`, `POIData`, `FactionStartPosition`, `RegisterGenerator`, `GetGeneratorOrDefault` |
| `world/worldmap/gen_overworld.go` | `StrategicOverworldGenerator`, `StrategicOverworldConfig`, `DefaultStrategicOverworldConfig` |
| `world/worldmap/dungeongen.go` | `GameMap`, `NewGameMap`, `GetBiomeAt`, `PlaceStairs` |
| `world/worldmap/dungeontile.go` | `Tile`, `TileType` (WALL, FLOOR, STAIRS_DOWN) |
| `world/worldmap/biome.go` | `Biome` enum, `String()` |
| `world/worldmap/tileconfig.go` | `POITown`, `POITemple`, `POIGuildHall`, `POIWatchtower` constants |

---

## Conclusion

The TinkerRogue Overworld System is a data-driven, ECS-based strategic layer that combines turn-based commander movement with emergent gameplay from faction AI, dynamic threats, and spatial influence mechanics.

**Key Design Decisions**:
- **Unified `OverworldNodeComponent`** eliminates the need for multiple ECS tags and separate query loops for different node types
- **Commander movement points** (replacing a travel-tick system) give players direct, tactile control over their agents each turn
- **Top-2 influence stacking** (rather than unbounded diminishing returns) caps the impact of massed synergy while still making node clustering meaningful
- **`PendingRaid` propagated through the call stack** (not stored in ECS) keeps the garrison raid a transient event handled by the GUI layer immediately after the tick

**Future Enhancement Areas**:
- Diplomacy and player-faction negotiations
- Dynamic quest generation from overworld state
- Fog of war on the strategic map
- Multiple simultaneous pending raid resolution

---

**Document Version:** 2.0
**Updated By:** Claude Sonnet 4.6
**Date:** 2026-02-17
