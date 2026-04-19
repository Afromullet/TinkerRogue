package progression

import "github.com/bytearena/ecs"

// ProgressionData holds a Player entity's permanent progression state.
// Perk and spell IDs are stored as plain strings to avoid a common <-> perks
// dependency cycle; conversion to typed IDs happens at consumption sites.
type ProgressionData struct {
	ArcanaPoints     int
	SkillPoints      int
	UnlockedSpellIDs []string
	UnlockedPerkIDs  []string
}

var (
	ProgressionComponent *ecs.Component
	ProgressionTag       ecs.Tag
)
