package input

import (
	"game_main/avatar"
	"game_main/common"
	"game_main/graphics"
	"game_main/gui"
	"game_main/worldmap"
)

type SharedInputState struct {
	PrevCursor           graphics.PixelPosition
	PrevThrowInds        []int
	PrevRangedAttInds    []int
	PrevTargetLineInds   []int
	TurnTaken            bool
}

func NewSharedInputState() *SharedInputState {
	return &SharedInputState{
		PrevCursor:         graphics.PixelPosition{X: -1, Y: -1},
		PrevThrowInds:      make([]int, 0),
		PrevRangedAttInds:  make([]int, 0),
		PrevTargetLineInds: make([]int, 0),
		TurnTaken:          false,
	}
}

type InputController interface {
	HandleInput() bool
	CanHandle() bool
	OnActivate()
	OnDeactivate()
}

type InputCoordinator struct {
	movementController *MovementController
	combatController   *CombatController
	uiController       *UIController
	sharedState        *SharedInputState

	ecsManager   *common.EntityManager
	playerData   *avatar.PlayerData
	gameMap      *worldmap.GameMap
	playerUI     *gui.PlayerUI
}

func NewInputCoordinator(ecsManager *common.EntityManager, playerData *avatar.PlayerData,
	gameMap *worldmap.GameMap, playerUI *gui.PlayerUI) *InputCoordinator {

	sharedState := NewSharedInputState()

	return &InputCoordinator{
		movementController: NewMovementController(ecsManager, playerData, gameMap, sharedState),
		combatController:   NewCombatController(ecsManager, playerData, gameMap, playerUI, sharedState),
		uiController:       NewUIController(playerData, playerUI, sharedState),
		sharedState:        sharedState,
		ecsManager:         ecsManager,
		playerData:         playerData,
		gameMap:            gameMap,
		playerUI:           playerUI,
	}
}

func (ic *InputCoordinator) HandleInput() bool {
	inputHandled := false

	// Check UI input first (highest priority)
	if ic.uiController.CanHandle() {
		inputHandled = ic.uiController.HandleInput() || inputHandled
	}

	// Then combat input (throwing/shooting)
	if ic.combatController.CanHandle() {
		inputHandled = ic.combatController.HandleInput() || inputHandled
	}

	// Finally movement input (lowest priority)
	if ic.movementController.CanHandle() {
		inputHandled = ic.movementController.HandleInput() || inputHandled
	}

	return inputHandled
}

func (ic *InputCoordinator) GetSharedState() *SharedInputState {
	return ic.sharedState
}