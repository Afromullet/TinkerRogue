package input

import (
	"game_main/common"
	"game_main/gear"
	"game_main/gui/framework"
	"game_main/visual/graphics"
	"game_main/visual/rendering"
	"game_main/world/coords"
	"game_main/world/worldmap"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
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

func (mc *CameraController) HandleInput() bool {
	inputHandled := false

	// Movement controls
	if inpututil.IsKeyJustReleased(ebiten.KeyW) {
		mc.movePlayer(0, -1)
		mc.playerData.InputStates.HasKeyInput = true
		inputHandled = true
	}

	if inpututil.IsKeyJustReleased(ebiten.KeyS) {
		mc.movePlayer(0, 1)
		mc.playerData.InputStates.HasKeyInput = true
		inputHandled = true
	}

	if inpututil.IsKeyJustReleased(ebiten.KeyA) {
		mc.movePlayer(-1, 0)
		mc.playerData.InputStates.HasKeyInput = true
		inputHandled = true
	}

	if inpututil.IsKeyJustReleased(ebiten.KeyD) {
		mc.movePlayer(1, 0)
		mc.playerData.InputStates.HasKeyInput = true
		inputHandled = true
	}

	// Diagonal movement
	if inpututil.IsKeyJustReleased(ebiten.KeyQ) {
		mc.movePlayer(-1, -1)
		mc.playerData.InputStates.HasKeyInput = true
		inputHandled = true
	}

	if inpututil.IsKeyJustReleased(ebiten.KeyE) {
		mc.movePlayer(1, -1)
		mc.playerData.InputStates.HasKeyInput = true
		inputHandled = true
	}

	if inpututil.IsKeyJustReleased(ebiten.KeyZ) {
		mc.movePlayer(-1, 1)
		mc.playerData.InputStates.HasKeyInput = true
		inputHandled = true
	}

	if inpututil.IsKeyJustReleased(ebiten.KeyC) {
		mc.movePlayer(1, 1)
		mc.playerData.InputStates.HasKeyInput = true
		inputHandled = true
	}

	// Pickup item
	if inpututil.IsKeyJustReleased(ebiten.KeyG) {
		mc.playerPickupItem()
		mc.playerData.InputStates.HasKeyInput = true
		inputHandled = true
	}

	// Debug tile highlighting
	if inpututil.IsKeyJustReleased(ebiten.KeyB) {
		mc.highlightCurrentTile()
		inputHandled = true
	}

	// Toggle map scrolling (M key)
	if inpututil.IsKeyJustReleased(ebiten.KeyM) {
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

func (mc *CameraController) playerPickupItem() {
	itemEntityID, err := mc.gameMap.RemoveItemFromTile(0, mc.playerData.Pos)

	if err == nil && itemEntityID != 0 {
		itemEntity := mc.ecsManager.FindEntityByID(itemEntityID)
		if itemEntity != nil {
			renderable := common.GetComponentType[*rendering.Renderable](itemEntity, rendering.RenderableComponent)
			renderable.Visible = false
			// Query inventory from player entity via ECS instead of using interface{}
			inv := common.GetComponentTypeByID[*gear.Inventory](mc.ecsManager, mc.playerData.PlayerEntityID, gear.InventoryComponent)
			if inv != nil {
				gear.AddItem(mc.ecsManager, inv, itemEntityID)
			}
		}
	}
}

func (mc *CameraController) highlightCurrentTile() {
	logicalPos := coords.LogicalPosition{X: mc.playerData.Pos.X, Y: mc.playerData.Pos.Y}
	ind := coords.CoordManager.LogicalToIndex(logicalPos)
	mc.gameMap.ApplyColorMatrixToIndex(ind, graphics.GreenColorMatrix)
}
