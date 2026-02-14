package bootstrap

import (
	"fmt"
	"game_main/common"
	"game_main/config"
	"game_main/tactical/commander"
	"game_main/templates"
	"game_main/world/coords"

	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

// CreateTestCommanders creates additional test commanders near the player's starting position.
// Each commander gets its own squad roster and a set of starting squads.
// Called during debug/test initialization after the initial commander is created.
func CreateTestCommanders(em *common.EntityManager, pd *common.PlayerData, startPos coords.LogicalPosition) error {
	roster := commander.GetPlayerCommanderRoster(pd.PlayerEntityID, em)
	if roster == nil {
		return fmt.Errorf("player has no commander roster component")
	}

	commanderImage, _, err := ebitenutil.NewImageFromFile(config.PlayerImagePath)
	if err != nil {
		return fmt.Errorf("failed to load commander image: %w", err)
	}

	testCommanders := []struct {
		Name   string
		Offset coords.LogicalPosition // Offset from player start
	}{
		{Name: "Vanguard", Offset: coords.LogicalPosition{X: 2, Y: 0}},
		{Name: "Sentinel", Offset: coords.LogicalPosition{X: -2, Y: 0}},
	}

	for _, tc := range testCommanders {
		pos := coords.LogicalPosition{
			X: startPos.X + tc.Offset.X,
			Y: startPos.Y + tc.Offset.Y,
		}

		cmdID := commander.CreateCommander(
			em,
			tc.Name,
			pos,
			config.DefaultCommanderMovementSpeed,
			config.DefaultCommanderMaxSquads,
			commanderImage,
			config.DefaultCommanderStartingMana,
			config.DefaultCommanderMaxMana,
			templates.GetAllSpellIDs(),
		)

		if err := roster.AddCommander(cmdID); err != nil {
			return fmt.Errorf("failed to add commander %s to roster: %w", tc.Name, err)
		}

		// Give each test commander some starting squads (prefixed with commander name)
		if err := CreateInitialPlayerSquads(cmdID, pd.PlayerEntityID, em, tc.Name); err != nil {
			return fmt.Errorf("failed to create squads for commander %s: %w", tc.Name, err)
		}

		fmt.Printf("Created test commander '%s' (ID: %d) at (%d,%d)\n", tc.Name, cmdID, pos.X, pos.Y)
	}

	return nil
}
