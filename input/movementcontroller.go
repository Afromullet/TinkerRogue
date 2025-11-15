package input

import (
	"game_main/common"
	"game_main/coords"
	"game_main/gear"
	"game_main/graphics"
	"game_main/rendering"
	"game_main/worldmap"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

type MovementController struct {
	ecsManager  *common.EntityManager
	playerData  *common.PlayerData
	gameMap     *worldmap.GameMap
	sharedState *SharedInputState
}

func NewMovementController(ecsManager *common.EntityManager, playerData *common.PlayerData,
	gameMap *worldmap.GameMap, sharedState *SharedInputState) *MovementController {
	return &MovementController{
		ecsManager:  ecsManager,
		playerData:  playerData,
		gameMap:     gameMap,
		sharedState: sharedState,
	}
}

func (mc *MovementController) CanHandle() bool {
	// Movement is always available unless in specific states
	return !mc.playerData.InputStates.IsThrowing
	// IsShooting check removed - squad system handles combat
}

func (mc *MovementController) OnActivate() {
	// No special activation needed for movement
}

func (mc *MovementController) OnDeactivate() {
	// No special deactivation needed for movement
}

func (mc *MovementController) HandleInput() bool {
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

	// Stairs interaction
	if inpututil.IsKeyJustReleased(ebiten.KeySpace) {
		if mc.handleStairsInteraction() {
			inputHandled = true
		}
	}

	// Pickup item
	if inpututil.IsKeyJustReleased(ebiten.KeyG) {
		mc.playerPickupItem()
		mc.playerData.InputStates.HasKeyInput = true
		inputHandled = true
	}

	// Debug tile highlighting
	if inpututil.IsKeyJustReleased(ebiten.KeyL) {
		mc.highlightCurrentTile()
		inputHandled = true
	}

	return inputHandled
}

func (mc *MovementController) movePlayer(xOffset, yOffset int) {
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
		mc.gameMap.PlayerVisible.Compute(mc.gameMap, mc.playerData.Pos.X, mc.playerData.Pos.Y, 8)

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

func (mc *MovementController) handleStairsInteraction() bool {
	playerPos := common.GetPositionByID(mc.ecsManager, mc.playerData.PlayerEntityID)
	if playerPos == nil {
		return false
	}
	logicalPos := coords.LogicalPosition{X: playerPos.X, Y: playerPos.Y}
	ind := coords.CoordManager.LogicalToIndex(logicalPos)

	if mc.gameMap.Tiles[ind].TileType == worldmap.STAIRS_DOWN {
		worldmap.GoDownStairs(mc.gameMap)
		playerPos.X, playerPos.Y = mc.gameMap.Rooms[0].Center()
		return true
	}
	return false
}

func (mc *MovementController) playerPickupItem() {
	itemEntityID, err := mc.gameMap.RemoveItemFromTile(0, mc.playerData.Pos)

	if err == nil && itemEntityID != 0 {
		itemEntity := gear.FindItemEntityByID(mc.ecsManager.World, itemEntityID)
		if itemEntity != nil {
			renderable := common.GetComponentType[*rendering.Renderable](itemEntity, rendering.RenderableComponent)
			renderable.Visible = false
			// Query inventory from player entity via ECS instead of using interface{}
			inv := common.GetComponentTypeByID[*gear.Inventory](mc.ecsManager, mc.playerData.PlayerEntityID, gear.InventoryComponent)
			if inv != nil {
				gear.AddItem(mc.ecsManager.World, inv, itemEntityID)
			}
		}
	}
}

func (mc *MovementController) highlightCurrentTile() {
	logicalPos := coords.LogicalPosition{X: mc.playerData.Pos.X, Y: mc.playerData.Pos.Y}
	ind := coords.CoordManager.LogicalToIndex(logicalPos)
	mc.gameMap.ApplyColorMatrixToIndex(ind, graphics.GreenColorMatrix)
}
