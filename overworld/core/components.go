package core

import (
	"game_main/world/coords"

	"github.com/bytearena/ecs"
)

// Component and tag variables - shared across all overworld subsystems
var (
	ThreatNodeComponent       *ecs.Component
	OverworldFactionComponent *ecs.Component
	TickStateComponent        *ecs.Component
	InfluenceComponent        *ecs.Component
	TerritoryComponent        *ecs.Component
	StrategicIntentComponent  *ecs.Component
	VictoryStateComponent     *ecs.Component
	TravelStateComponent      *ecs.Component
	PlayerNodeComponent       *ecs.Component

	ThreatNodeTag       ecs.Tag
	OverworldFactionTag ecs.Tag
	TickStateTag        ecs.Tag
	VictoryStateTag     ecs.Tag
	TravelStateTag      ecs.Tag
	PlayerNodeTag       ecs.Tag
)

// OverworldEncounterComponent and tag for encounter entities
var (
	OverworldEncounterTag       ecs.Tag
	OverworldEncounterComponent *ecs.Component
)

// ThreatNodeData - Pure data component for threat nodes
type ThreatNodeData struct {
	ThreatID       ecs.EntityID // Entity ID of this threat node
	ThreatType     ThreatType   // Enum: Necromancer, Bandit, Corruption, etc.
	EncounterID    string       // Specific encounter variant (e.g., "necromancer_elite")
	Intensity      int          // Current power level (0-10)
	GrowthProgress float64      // 0.0-1.0 progress to next intensity level
	GrowthRate     float64      // Growth per tick (e.g., 0.05)
	IsContained    bool         // Player nearby, slows growth
	SpawnedTick    int64        // Tick when threat was created
}

// OverworldFactionData - Renamed to avoid conflict with combat.FactionComponent
// Represents persistent strategic factions on the overworld
type OverworldFactionData struct {
	FactionID     ecs.EntityID
	FactionType   FactionType   // Enum: Undead, Bandits, Corruption
	Strength      int           // Military power
	TerritorySize int           // Number of tiles controlled
	Disposition   int           // -100 (hostile) to +100 (allied)
	CurrentIntent FactionIntent // Expand, Fortify, Raid, Retreat
	GrowthRate    float64       // How fast faction expands
}

// TickStateData - Global singleton component for turn-based tick system
type TickStateData struct {
	CurrentTick int64 // Global tick counter
	IsGameOver  bool  // Game ended (victory/defeat) - prevents further ticks
}

// InfluenceData - Cached influence radius
type InfluenceData struct {
	Radius         int // Tiles affected by this threat
	EffectType     InfluenceEffect
	EffectStrength float64 // Magnitude of effect
}

// TerritoryData - Tiles controlled by a faction
type TerritoryData struct {
	OwnedTiles    []coords.LogicalPosition // List of controlled tiles
	BorderTiles   []coords.LogicalPosition // Tiles adjacent to territory (cached)
	ContestedTile *coords.LogicalPosition  // Tile currently being fought over
}

// StrategicIntentData - Current faction objective
type StrategicIntentData struct {
	Intent         FactionIntent
	TargetPosition *coords.LogicalPosition // Expand toward this, raid this
	TicksRemaining int                     // Ticks until intent re-evaluation
	Priority       float64                 // How important this action is (0.0-1.0)
}

// TravelStateData - Singleton component tracking player travel state
type TravelStateData struct {
	IsTraveling       bool                   // Currently traveling
	Origin            coords.LogicalPosition // Starting position
	Destination       coords.LogicalPosition // Target position (threat node)
	TotalDistance     float64                // Euclidean distance (calculated once at start)
	RemainingDistance float64                // Distance left to travel
	TargetThreatID    ecs.EntityID           // Threat being traveled to
	TargetEncounterID ecs.EntityID           // Encounter entity created for this travel
}

// OverworldEncounterData - Encounter metadata created from overworld threats
type OverworldEncounterData struct {
	Name          string       // Display name (e.g., "Goblin Patrol")
	Level         int          // Difficulty level
	EncounterType string       // Type identifier for spawn logic
	IsDefeated    bool         // Marked true after victory
	ThreatNodeID  ecs.EntityID // Link to overworld threat node (0 if not from threat)
}

// PlayerNodeData - Pure data component for player-placed nodes (settlements, fortresses)
type PlayerNodeData struct {
	NodeID     ecs.EntityID // Entity ID of this player node
	NodeTypeID NodeTypeID   // References nodeDefinitions.json ID (e.g., "town", "watchtower")
	PlacedTick int64        // Tick when placed
}
