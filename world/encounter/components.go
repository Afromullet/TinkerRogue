package encounter

import (
	"game_main/common"

	"github.com/bytearena/ecs"
)

var (
	OverworldEncounterTag       ecs.Tag
	OverworldEncounterComponent *ecs.Component
	EvaluationConfigComponent   *ecs.Component // Configuration for power calculations
)

// init registers encounter component initialization with the common subsystem registry
func init() {
	common.RegisterSubsystem(func(em *common.EntityManager) {
		InitEncounterComponents(em)
		InitEncounterTags(em)
	})
}

// InitEncounterComponents initializes encounter components
func InitEncounterComponents(manager *common.EntityManager) {
	OverworldEncounterComponent = manager.World.NewComponent()
	EvaluationConfigComponent = manager.World.NewComponent()
}

// InitEncounterTags creates tags for querying encounter-related entities
func InitEncounterTags(manager *common.EntityManager) {
	OverworldEncounterTag = ecs.BuildTag(OverworldEncounterComponent)
}

// OverworldEncounterData - Pure data for encounter entities
type OverworldEncounterData struct {
	Name          string       // Display name (e.g., "Goblin Patrol")
	Level         int          // Difficulty level
	EncounterType string       // Type identifier for spawn logic
	IsDefeated    bool         // Marked true after victory
	ThreatNodeID  ecs.EntityID // Link to overworld threat node (0 if not from threat)
}

// EvaluationConfigData holds configurable weights for power calculations
// Pure data component - no logic
type EvaluationConfigData struct {
	ProfileName string // "Offensive", "Defensive", "Balanced"

	// Unit-level weights (0.0-1.0 range, sum should equal 1.0 for each category)
	OffensiveWeight float64 // Weight for offensive stats (damage, hit, crit)
	DefensiveWeight float64 // Weight for defensive stats (HP, resistance, dodge)
	UtilityWeight   float64 // Weight for utility (role, abilities, cover)

	// Offensive sub-weights (should sum to 1.0)
	DamageWeight   float64 // Physical/magic damage output
	AccuracyWeight float64 // Hit rate and crit chance

	// Defensive sub-weights (should sum to 1.0)
	HealthWeight     float64 // Max HP and current HP
	ResistanceWeight float64 // Physical/magic resistance
	AvoidanceWeight  float64 // Dodge chance

	// Utility sub-weights (should sum to 1.0)
	RoleWeight    float64 // Role multiplier importance
	AbilityWeight float64 // Leader ability value
	CoverWeight   float64 // Cover provision value

	// Squad-level modifiers
	FormationBonus   float64         // Bonus per formation type
	MoraleMultiplier float64         // Morale impact (0.01 per morale point)
	LeaderBonus      float64         // Leader presence multiplier (1.2-1.5)
	CompositionBonus map[int]float64 // Bonus by unique attack type count (1→0.8, 2→1.1, 3→1.2, 4→1.3)
	HealthPenalty    float64         // Penalty multiplier for low HP squads

	// Roster-level modifiers
	DeployedWeight float64 // Weight for deployed squads (default 1.0)
	ReserveWeight  float64 // Weight for reserve squads (default 0.3)
}
