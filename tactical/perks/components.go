package perks

import (
	"github.com/bytearena/ecs"
)

// SquadPerkData holds up to 3 perk IDs for a squad entity.
type SquadPerkData struct {
	EquippedPerks [3]string // Up to 3 perk IDs ("" = empty slot)
}

// UnitPerkData holds up to 2 perk IDs for a unit entity.
type UnitPerkData struct {
	EquippedPerks [2]string // Up to 2 perk IDs ("" = empty slot)
}

// CommanderPerkData holds up to 3 perk IDs for a commander entity.
type CommanderPerkData struct {
	EquippedPerks [3]string // Up to 3 perk IDs ("" = empty slot)
}

// PerkUnlockData tracks which perks have been unlocked and available points.
type PerkUnlockData struct {
	UnlockedPerks map[string]bool // Perk IDs that have been unlocked
	PerkPoints    int             // Available points to spend
}

// ECS component and tag variables.
var (
	SquadPerkComponent     *ecs.Component
	UnitPerkComponent      *ecs.Component
	CommanderPerkComponent *ecs.Component
	PerkUnlockComponent    *ecs.Component

	SquadPerkTag ecs.Tag
	UnitPerkTag  ecs.Tag
)
