package bootstrap

import (
	"game_main/common"
	"game_main/overworld/core"
	"game_main/overworld/faction"
	"game_main/world/coords"
	"game_main/world/worldmap"
	"log"
)

// InitializeOverworldFactions creates starting NPC factions on the overworld.
// Uses generator-provided positions when available, falls back to hardcoded positions.
func InitializeOverworldFactions(em *common.EntityManager, pd *common.PlayerData, gm *worldmap.GameMap) {
	factionTypes := []core.FactionType{
		core.FactionNecromancers,
		core.FactionBandits,
		core.FactionOrcs,
		core.FactionCultists,
	}

	for i, fType := range factionTypes {
		var pos coords.LogicalPosition
		strength := 6 + common.RandomInt(5)

		if i < len(gm.FactionStartPositions) {
			pos = gm.FactionStartPositions[i].Position
		} else {
			// Fallback hardcoded positions
			pos = coords.LogicalPosition{X: 15 + i*35, Y: 15 + (i%2)*50}
		}

		factionID := faction.CreateFaction(em, fType, pos, strength)
		log.Printf("Created %s faction at (%d, %d) with strength %d (ID: %d)\n",
			fType.String(), pos.X, pos.Y, strength, factionID)
	}
}
