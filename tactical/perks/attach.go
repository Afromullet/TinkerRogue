package perks

import (
	"github.com/bytearena/ecs"
)

// AttachSquadPerkComponent adds an empty SquadPerkData component to an entity.
// Called by packages that create squad entities (e.g., squadservices, guisquads).
func AttachSquadPerkComponent(entity *ecs.Entity) {
	entity.AddComponent(SquadPerkComponent, &SquadPerkData{})
}

// AttachUnitPerkComponent adds an empty UnitPerkData component to an entity.
// Called by packages that create unit entities (e.g., squadservices).
func AttachUnitPerkComponent(entity *ecs.Entity) {
	entity.AddComponent(UnitPerkComponent, &UnitPerkData{})
}

// AttachCommanderPerkComponent adds an empty CommanderPerkData component to an entity.
// Called by the commander package during CreateCommander.
func AttachCommanderPerkComponent(entity *ecs.Entity) {
	entity.AddComponent(CommanderPerkComponent, &CommanderPerkData{})
}

// AttachPerkUnlockComponent adds a PerkUnlockData component with all perks unlocked and 999 points.
// This is the roster owner's unlock tracking. For now, everything is unlocked.
func AttachPerkUnlockComponent(entity *ecs.Entity) {
	unlocked := make(map[string]bool)
	for _, id := range GetAllPerkIDs() {
		unlocked[id] = true
	}
	entity.AddComponent(PerkUnlockComponent, &PerkUnlockData{
		UnlockedPerks: unlocked,
		PerkPoints:    999,
	})
}
