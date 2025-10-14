package input

import (
	"game_main/avatar"
)

// UIController handles UI-related input
// NOTE: Most UI input is now handled by the UIModeManager in the main game loop
// This controller is kept for compatibility but may be deprecated in the future
type UIController struct {
	playerData  *avatar.PlayerData
	sharedState *SharedInputState
}

func NewUIController(playerData *avatar.PlayerData, sharedState *SharedInputState) *UIController {
	return &UIController{
		playerData:  playerData,
		sharedState: sharedState,
	}
}

func (uc *UIController) CanHandle() bool {
	return true // UI input can always be handled
}

func (uc *UIController) OnActivate() {
	// No special activation needed for UI
}

func (uc *UIController) OnDeactivate() {
	// No special deactivation needed for UI
}

func (uc *UIController) HandleInput() bool {
	// UI input is now handled by UIModeManager in the main game loop
	// This method is kept for compatibility but does nothing
	// All right-click, ESC, and mode switching is handled by ExplorationMode.HandleInput()
	return false
}