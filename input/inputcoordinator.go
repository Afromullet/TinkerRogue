// Package input manages all player input handling including movement, combat, and UI interactions.
// It coordinates between different input controllers (movement, combat, UI) and processes
// keyboard input to trigger appropriate game actions and state changes.
package input

import (
	"game_main/common"
	"game_main/world/coords"
	"game_main/world/worldmap"
)

type SharedInputState struct {
	PrevCursor         coords.PixelPosition
	PrevThrowInds      []int
	PrevRangedAttInds  []int
	PrevTargetLineInds []int
	TurnTaken          bool
}

func NewSharedInputState() *SharedInputState {
	return &SharedInputState{
		PrevCursor:         coords.PixelPosition{X: -1, Y: -1},
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

	ecsManager *common.EntityManager
	playerData *common.PlayerData
	gameMap    *worldmap.GameMap
}

func NewInputCoordinator(ecsManager *common.EntityManager, playerData *common.PlayerData,
	gameMap *worldmap.GameMap, dummyUI interface{}) *InputCoordinator {

	sharedState := NewSharedInputState()

	return &InputCoordinator{
		movementController: NewMovementController(ecsManager, playerData, gameMap, sharedState),
		combatController:   NewCombatController(ecsManager, playerData, gameMap, sharedState),
		uiController:       NewUIController(playerData, sharedState),
		sharedState:        sharedState,
		ecsManager:         ecsManager,
		playerData:         playerData,
		gameMap:            gameMap,
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
