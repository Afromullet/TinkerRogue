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
		OverworldEncounterComponent = em.World.NewComponent()
		OverworldEncounterTag = ecs.BuildTag(OverworldEncounterComponent)
	})
}

// OverworldEncounterData - Encounter metadata created from overworld threats
type OverworldEncounterData struct {
	Name          string       // Display name (e.g., "Goblin Patrol")
	Level         int          // Difficulty level
	EncounterType string       // Type identifier for spawn logic
	IsDefeated    bool         // Marked true after victory
	ThreatNodeID  ecs.EntityID // Link to overworld threat node (0 if not from threat)
}
