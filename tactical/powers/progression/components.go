// Package progression stores per-commander permanent progression state:
// Arcana/Skill point currencies and the unlocked perk/spell libraries that
// gate what a commander's squads can equip and cast.
//
// SCOPE-CHANGE WARNING: If you ever move ProgressionComponent off the
// Commander entity (it has previously lived on the Player entity), update
// all three of these docs in the same commit:
//   - resources/docs/project_documentation/Systems/PROGRESSION.md
//   - resources/docs/project_documentation/Architecture/ARCHITECTURE_LAYERS.md
//   - resources/docs/project_documentation/Architecture/ENTITY_REFERENCE.md
// Past contributors have shipped scope changes without doc updates and the
// drift took weeks to discover. See tech_debt_progression.md D1.
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
