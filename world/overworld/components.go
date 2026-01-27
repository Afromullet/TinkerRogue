package overworld

import (
	"game_main/world/coords"

	"github.com/bytearena/ecs"
)

// Component and tag variables
var (
	ThreatNodeComponent       *ecs.Component
	OverworldFactionComponent *ecs.Component
	TickStateComponent        *ecs.Component
	InfluenceComponent        *ecs.Component
	TerritoryComponent        *ecs.Component
	StrategicIntentComponent  *ecs.Component
	VictoryStateComponent     *ecs.Component
	PlayerResourcesComponent  *ecs.Component
	InfluenceCacheComponent   *ecs.Component

	ThreatNodeTag       ecs.Tag
	OverworldFactionTag ecs.Tag
	TickStateTag        ecs.Tag
	VictoryStateTag     ecs.Tag
	PlayerResourcesTag  ecs.Tag
	InfluenceCacheTag   ecs.Tag
)

// ThreatNodeData - Pure data component for threat nodes
type ThreatNodeData struct {
	ThreatID       ecs.EntityID // Entity ID of this threat node
	ThreatType     ThreatType   // Enum: Necromancer, Bandit, Corruption, etc.
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
