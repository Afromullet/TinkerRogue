package core

import (
	"game_main/world/coords"

	"github.com/bytearena/ecs"
)

// Component and tag variables - shared across all overworld subsystems
var (
	OverworldFactionComponent *ecs.Component
	TickStateComponent        *ecs.Component
	InfluenceComponent        *ecs.Component
	TerritoryComponent        *ecs.Component
	StrategicIntentComponent  *ecs.Component
	VictoryStateComponent     *ecs.Component
	TravelStateComponent      *ecs.Component
	InteractionComponent      *ecs.Component
	OverworldNodeComponent    *ecs.Component

	OverworldFactionTag ecs.Tag
	TickStateTag        ecs.Tag
	VictoryStateTag     ecs.Tag
	TravelStateTag      ecs.Tag
	InteractionTag      ecs.Tag
	OverworldNodeTag    ecs.Tag
)

// OverworldEncounterComponent and tag for encounter entities
var (
	OverworldEncounterTag       ecs.Tag
	OverworldEncounterComponent *ecs.Component
)

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
	Radius        int     // Tiles affected by this threat
	BaseMagnitude float64 // Magnitude of effect (derived from intensity)
}

// TerritoryData - Tiles controlled by a faction
type TerritoryData struct {
	OwnedTiles []coords.LogicalPosition // List of controlled tiles
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
	Origin            coords.LogicalPosition // Starting position (for cancel)
	Destination       coords.LogicalPosition // Target position (threat node)
	TicksRemaining    int                    // Ticks until arrival
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

// InteractionType classifies how two overlapping influence nodes interact
type InteractionType int

const (
	InteractionSynergy     InteractionType = iota // Same-faction threats boost each other
	InteractionCompetition                        // Rival faction threats slow each other
	InteractionSuppression                        // Player nodes suppress threats
	InteractionPlayerBoost                        // Player nodes synergize with each other
)

// NodeInteraction records a single interaction between two nodes
type NodeInteraction struct {
	TargetID     ecs.EntityID    // The other node in this interaction
	Relationship InteractionType // Type of interaction
	Modifier     float64         // Additive: positive = boost, negative = suppress
	Distance     int             // Manhattan distance between nodes
}

// InteractionData - Pure data component tracking all influence interactions for a node
type InteractionData struct {
	Interactions []NodeInteraction // All active interactions
	NetModifier  float64           // Combined modifier (1.0 = no effect)
}

// OverworldNodeData - Unified data component for all overworld nodes (threats, settlements, POIs).
// Replaces both ThreatNodeData and PlayerNodeData with explicit faction ownership.
type OverworldNodeData struct {
	NodeID         ecs.EntityID // Entity ID of this node
	NodeTypeID     string       // "necromancer", "town", "watchtower", etc.
	Category       NodeCategory // Cached: "threat", "settlement", "fortress"
	OwnerID        string       // "player", "Neutral", "Necromancers", etc.
	EncounterID    string       // Empty for non-combat nodes
	Intensity      int          // 0 for settlements
	GrowthProgress float64      // 0.0 for non-growing nodes
	GrowthRate     float64      // 0.0 for settlements
	IsContained    bool
	CreatedTick    int64
}
