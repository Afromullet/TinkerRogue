package encounter

import (
	"game_main/common"

	"github.com/bytearena/ecs"
)

var (
	OverworldEncounterTag       ecs.Tag
	OverworldEncounterComponent *ecs.Component
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
}

// InitEncounterTags creates tags for querying encounter-related entities
func InitEncounterTags(manager *common.EntityManager) {
	OverworldEncounterTag = ecs.BuildTag(OverworldEncounterComponent)
}

// OverworldEncounterData - Pure data for encounter entities
type OverworldEncounterData struct {
	Name          string // Display name (e.g., "Goblin Patrol")
	Level         int    // Difficulty level
	EncounterType string // Type identifier for spawn logic
	IsDefeated    bool   // Marked true after victory
}
