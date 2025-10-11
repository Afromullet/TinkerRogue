package input

import (
	"game_main/avatar"
	"game_main/graphics"
	"game_main/gui"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

type UIController struct {
	playerData  *avatar.PlayerData
	playerUI    *gui.PlayerUI
	sharedState *SharedInputState
}

func NewUIController(playerData *avatar.PlayerData, playerUI *gui.PlayerUI, sharedState *SharedInputState) *UIController {
	return &UIController{
		playerData:  playerData,
		playerUI:    playerUI,
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
	inputHandled := false

	// Handle info menu opening
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButton2) {
		// Only open info menu if not in combat modes
		// IsShooting check removed - squad system handles combat
		if !uc.playerData.InputStates.IsThrowing {
			cursorX, cursorY := graphics.CursorPosition(*uc.playerData.Pos)
			uc.playerUI.InformationUI.InfoSelectionWindow(cursorX, cursorY)
			uc.playerData.InputStates.InfoMeuOpen = true
			inputHandled = true
		}
	}

	// Handle info menu closing
	if uc.playerData.InputStates.InfoMeuOpen {
		if inpututil.IsKeyJustReleased(ebiten.KeyEscape) {
			uc.playerUI.InformationUI.CloseWindows()
			uc.playerData.InputStates.InfoMeuOpen = false
			inputHandled = true
		}
	}

	// Handle throwable item selection state
	if uc.playerUI.IsThrowableItemSelected() {
		uc.playerData.InputStates.IsThrowing = true
	} else {
		uc.playerData.InputStates.IsThrowing = false
	}

	return inputHandled
}