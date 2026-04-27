package gamesetup

import (
	"fmt"
	"game_main/campaign/overworld/core"
	"game_main/campaign/overworld/faction"
	"game_main/core/common"
	"game_main/core/coords"
	"game_main/templates"
	"game_main/world/worldmapcore"
	"log"
)

// CreateInitialFactions creates starting NPC factions on the overworld using the
// list defined in initialsetup.json. Uses generator-provided positions when
// available, falling back to JSON-configured positions otherwise.
func CreateInitialFactions(em *common.EntityManager, pd *common.PlayerData, gm *worldmapcore.GameMap) error {
	cfg := templates.InitialSetupTemplate.Factions

	for i, entry := range cfg.Entries {
		fType, ok := factionTypeFromString(entry.Type)
		if !ok {
			return fmt.Errorf("unknown faction type %q (validation should have caught this)", entry.Type)
		}

		var pos coords.LogicalPosition
		if i < len(gm.FactionStartPositions) {
			pos = gm.FactionStartPositions[i].Position
		} else {
			fp := cfg.FallbackPositions[i]
			pos = coords.LogicalPosition{X: fp.X, Y: fp.Y}
		}

		strength := common.GetRandomBetween(cfg.StrengthMin, cfg.StrengthMax)
		factionID := faction.CreateFaction(em, fType, pos, strength)
		log.Printf("Created %s faction at (%d, %d) with strength %d (ID: %d)\n",
			fType.String(), pos.X, pos.Y, strength, factionID)
	}
	return nil
}

func factionTypeFromString(s string) (core.FactionType, bool) {
	switch s {
	case "necromancers":
		return core.FactionNecromancers, true
	case "bandits":
		return core.FactionBandits, true
	case "orcs":
		return core.FactionOrcs, true
	case "cultists":
		return core.FactionCultists, true
	}
	return 0, false
}
