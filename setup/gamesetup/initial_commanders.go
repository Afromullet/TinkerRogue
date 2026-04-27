package gamesetup

import (
	"fmt"
	"game_main/core/common"
	"game_main/core/config"
	"game_main/core/coords"
	"game_main/tactical/commander"
	"game_main/templates"

	"github.com/bytearena/ecs"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

// CreateInitialCommanders creates every commander defined in initialsetup.json,
// wires them into the player's commander roster, seeds starter perks/spells, and
// creates each commander's starting squads. The first entry flagged isPrimary
// must have offset (0,0); its ID is returned for callers that need to identify
// the player's primary commander.
func CreateInitialCommanders(em *common.EntityManager, pd *common.PlayerData, startPos coords.LogicalPosition) (ecs.EntityID, error) {
	roster := commander.GetPlayerCommanderRoster(pd.PlayerEntityID, em)
	if roster == nil {
		return 0, fmt.Errorf("player has no commander roster component")
	}

	commanderImage, _, err := ebitenutil.NewImageFromFile(config.PlayerImagePath)
	if err != nil {
		return 0, fmt.Errorf("failed to load commander image: %w", err)
	}

	cfg := templates.GameConfig.Commander
	var primaryID ecs.EntityID
	primaryFound := false

	for _, c := range templates.InitialSetupTemplate.Commanders {
		pos := coords.LogicalPosition{
			X: startPos.X + c.OffsetX,
			Y: startPos.Y + c.OffsetY,
		}

		cmdID := commander.CreateCommander(
			em,
			c.Name,
			pos,
			cfg.MovementSpeed,
			cfg.MaxSquads,
			commanderImage,
		)

		// Seed starter perks/spells onto the commander's progression before any
		// squads are created; InitSquadSpellsFromLeader filters against this list.
		commander.SeedStarters(cmdID, em)

		if err := roster.AddCommander(cmdID); err != nil {
			return 0, fmt.Errorf("failed to add commander %s to roster: %w", c.Name, err)
		}

		if err := CreateSquadsForCommander(cmdID, pd.PlayerEntityID, em, c.Squads); err != nil {
			return 0, fmt.Errorf("failed to create squads for commander %s: %w", c.Name, err)
		}

		fmt.Printf("Created commander '%s' (ID: %d) at (%d,%d)\n", c.Name, cmdID, pos.X, pos.Y)

		if c.IsPrimary {
			primaryID = cmdID
			primaryFound = true
		}
	}

	if !primaryFound {
		// Validation should have caught this; defensive only.
		return 0, fmt.Errorf("no primary commander defined in initialsetup.json")
	}
	return primaryID, nil
}
