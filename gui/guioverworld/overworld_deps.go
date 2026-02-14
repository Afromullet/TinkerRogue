package guioverworld

import (
	"game_main/common"
	"game_main/gui/framework"
	"game_main/mind/encounter"
	"game_main/tactical/commander"
)

// OverworldModeDeps consolidates shared dependencies for overworld handlers.
// Pattern from gui/guicombat/combatdeps.go.
type OverworldModeDeps struct {
	State             *framework.OverworldState
	Manager           *common.EntityManager
	PlayerData        *common.PlayerData
	EncounterService  *encounter.EncounterService
	Renderer          *OverworldRenderer
	ModeManager       *framework.UIModeManager
	ModeCoordinator   *framework.GameModeCoordinator
	CommanderMovement *commander.CommanderMovementSystem
	LogEvent          func(string) // callback to append to event log
	RefreshPanels     func()       // callback to trigger panel refresh
}
