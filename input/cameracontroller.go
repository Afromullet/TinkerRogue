package input

import (
	"game_main/common"
	"game_main/gui/framework"
	"game_main/visual/graphics"
	"game_main/world/coords"
	"game_main/world/worldmap"
)

type CameraController struct {
	ecsManager *common.EntityManager
	playerData *common.PlayerData
	gameMap    *worldmap.GameMap
}

func NewCameraController(ecsManager *common.EntityManager, playerData *common.PlayerData,
	gameMap *worldmap.GameMap, coordinator *framework.GameModeCoordinator) *CameraController {
	return &CameraController{
		ecsManager: ecsManager,
		playerData: playerData,
		gameMap:    gameMap,
	}
}

// HandleInput processes camera/movement input using the framework's InputState.
// Uses KeysJustReleased for movement (matching the original inpututil.IsKeyJustReleased behavior).
func (mc *CameraController) HandleInput(inputState *framework.InputState) bool {
	inputHandled := false

	// Movement controls (via action map or raw just-released)
	if inputState.ActionActive(framework.ActionCameraMoveUp) {
		mc.movePlayer(0, -1)
		mc.playerData.InputStates.HasKeyInput = true
		inputHandled = true
	}

	if inputState.ActionActive(framework.ActionCameraMoveDown) {
		mc.movePlayer(0, 1)
		mc.playerData.InputStates.HasKeyInput = true
		inputHandled = true
	}

	if inputState.ActionActive(framework.ActionCameraMoveLeft) {
		mc.movePlayer(-1, 0)
		mc.playerData.InputStates.HasKeyInput = true
		inputHandled = true
	}

	if inputState.ActionActive(framework.ActionCameraMoveRight) {
		mc.movePlayer(1, 0)
		mc.playerData.InputStates.HasKeyInput = true
		inputHandled = true
	}

	// Diagonal movement
	if inputState.ActionActive(framework.ActionCameraMoveUpLeft) {
		mc.movePlayer(-1, -1)
		mc.playerData.InputStates.HasKeyInput = true
		inputHandled = true
	}

	if inputState.ActionActive(framework.ActionCameraMoveUpRight) {
		mc.movePlayer(1, -1)
		mc.playerData.InputStates.HasKeyInput = true
		inputHandled = true
	}

	if inputState.ActionActive(framework.ActionCameraMoveDownLeft) {
		mc.movePlayer(-1, 1)
		mc.playerData.InputStates.HasKeyInput = true
		inputHandled = true
	}

	if inputState.ActionActive(framework.ActionCameraMoveDownRight) {
		mc.movePlayer(1, 1)
		mc.playerData.InputStates.HasKeyInput = true
		inputHandled = true
	}

	// Debug tile highlighting
	if inputState.ActionActive(framework.ActionCameraHighlight) {
		mc.highlightCurrentTile()
		inputHandled = true
	}

	// Toggle map scrolling
	if inputState.ActionActive(framework.ActionCameraToggleScroll) {
		coords.MAP_SCROLLING_ENABLED = !coords.MAP_SCROLLING_ENABLED
		inputHandled = true
	}

	return inputHandled
}

func (mc *CameraController) movePlayer(xOffset, yOffset int) {
	nextPosition := coords.LogicalPosition{
		X: mc.playerData.Pos.X + xOffset,
		Y: mc.playerData.Pos.Y + yOffset,
	}

	nextLogicalPos := coords.LogicalPosition{X: nextPosition.X, Y: nextPosition.Y}
	index := coords.CoordManager.LogicalToIndex(nextLogicalPos)
	nextTile := mc.gameMap.Tiles[index]

	currentLogicalPos := coords.LogicalPosition{X: mc.playerData.Pos.X, Y: mc.playerData.Pos.Y}
	index = coords.CoordManager.LogicalToIndex(currentLogicalPos)
	oldTile := mc.gameMap.Tiles[index]

	if !nextTile.Blocked {
		// Update PositionSystem before moving player
		if common.GlobalPositionSystem != nil {
			common.GlobalPositionSystem.MoveEntity(
				mc.playerData.PlayerEntityID,
				currentLogicalPos,
				nextLogicalPos,
			)
		}

		mc.playerData.Pos.X = nextPosition.X
		mc.playerData.Pos.Y = nextPosition.Y

		nextTile.Blocked = true
		oldTile.Blocked = false
	} else {
		// Melee combat removed - squad system will handle combat
		// Creature detection still available via common.GetCreatureAtPosition()
	}
}

func (mc *CameraController) highlightCurrentTile() {
	logicalPos := coords.LogicalPosition{X: mc.playerData.Pos.X, Y: mc.playerData.Pos.Y}
	ind := coords.CoordManager.LogicalToIndex(logicalPos)
	mc.gameMap.ApplyColorMatrixToIndex(ind, graphics.GreenColorMatrix)
}
