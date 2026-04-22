package progression

import "github.com/bytearena/ecs"

// ProgressionData holds a Commander entity's permanent progression state.
// Each commander has their own Arcana/Skill points and their own unlocked
// perk/spell library; these unlocks apply to the squads that commander leads.
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
