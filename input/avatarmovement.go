package input

import (
	"fmt"

	"game_main/avatar"
	"game_main/common"
	"game_main/graphics"
	"game_main/gui"
	"game_main/timesystem"
	"game_main/worldmap"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

var prevCursorX, prevCursorY int
var PrevThrowInds []int
var PrevRangedAttInds []int
var PrevTargetLineIndices []int

var prevPosX = -1
var prevPosY = -1

var TurnTaken bool

func MovementControls(ecsmanager *common.EntityManager, pl *avatar.PlayerData, gm *worldmap.GameMap) {

	x := 0
	y := 0
	actionQueue := common.GetComponentType[*timesystem.ActionQueue](pl.PlayerEntity, timesystem.ActionQueueComponent)
	// Should throw a panic here since the game can't proceed without the player having an action querue
	if actionQueue == nil {
		fmt.Println("No action queue for player")
	}

	if inpututil.IsKeyJustReleased(ebiten.KeyW) {

		y = -1
		act, cost := GetPlayerMoveAction(PlayerMoveAction, ecsmanager, pl, gm, x, y)
		AddPlayerAction(act, pl, cost, timesystem.MovementKind)
		pl.InputStates.HasKeyInput = true
	}

	if inpututil.IsKeyJustReleased(ebiten.KeyS) {
		y = 1
		act, cost := GetPlayerMoveAction(PlayerMoveAction, ecsmanager, pl, gm, x, y)
		AddPlayerAction(act, pl, cost, timesystem.MovementKind)
		pl.InputStates.HasKeyInput = true
	}

	if inpututil.IsKeyJustReleased(ebiten.KeyA) {
		x = -1
		act, cost := GetPlayerMoveAction(PlayerMoveAction, ecsmanager, pl, gm, x, y)
		AddPlayerAction(act, pl, cost, timesystem.MovementKind)
		pl.InputStates.HasKeyInput = true
	}

	if inpututil.IsKeyJustReleased(ebiten.KeyD) {
		x = 1
		act, cost := GetPlayerMoveAction(PlayerMoveAction, ecsmanager, pl, gm, x, y)
		AddPlayerAction(act, pl, cost, timesystem.MovementKind)
		pl.InputStates.HasKeyInput = true
	}

	//Diagonal movement
	if inpututil.IsKeyJustReleased(ebiten.KeyQ) {
		y = -1
		x = -1
		act, cost := GetPlayerMoveAction(PlayerMoveAction, ecsmanager, pl, gm, x, y)
		AddPlayerAction(act, pl, cost, timesystem.MovementKind)
		pl.InputStates.HasKeyInput = true
	}

	if inpututil.IsKeyJustReleased(ebiten.KeyE) {
		y = -1
		x = +1
		act, cost := GetPlayerMoveAction(PlayerMoveAction, ecsmanager, pl, gm, x, y)
		AddPlayerAction(act, pl, cost, timesystem.MovementKind)
		pl.InputStates.HasKeyInput = true
	}

	if inpututil.IsKeyJustReleased(ebiten.KeyZ) {
		y = +1
		x = -1
		act, cost := GetPlayerMoveAction(PlayerMoveAction, ecsmanager, pl, gm, x, y)
		AddPlayerAction(act, pl, cost, timesystem.MovementKind)
		pl.InputStates.HasKeyInput = true
	}

	if inpututil.IsKeyJustReleased(ebiten.KeyC) {
		y = +1
		x = +1
		act, cost := GetPlayerMoveAction(PlayerMoveAction, ecsmanager, pl, gm, x, y)
		AddPlayerAction(act, pl, cost, timesystem.MovementKind)
		pl.InputStates.HasKeyInput = true
	}

	if inpututil.IsKeyJustReleased(ebiten.KeySpace) {

		playerPos := common.GetPosition(pl.PlayerEntity)

		ind := graphics.CoordTransformer.IndexFromLogicalXY(playerPos.X, playerPos.Y)

		if gm.Tiles[ind].TileType == worldmap.STAIRS_DOWN {

			worldmap.GoDownStairs(gm)
			playerPos.X, playerPos.Y = gm.Rooms[0].Center()

		}

	}

}

// todo replace the keypressed with iskeyreleasedPlayerI
func PlayerActions(ecsmanager *common.EntityManager, pl *avatar.PlayerData, gm *worldmap.GameMap, playerUI *gui.PlayerUI, tm *timesystem.GameTurn) bool {

	actionQueue := common.GetComponentType[*timesystem.ActionQueue](pl.PlayerEntity, timesystem.ActionQueueComponent)

	// Should throw a panic here since the game can't proceed without the player having an action querue
	if actionQueue == nil {
		fmt.Println("No action queue for player")
	}

	MovementControls(ecsmanager, pl, gm)

	if inpututil.IsKeyJustReleased(ebiten.KeyF) {

		if !pl.InputStates.IsThrowing && pl.Equipment.EqRangedWeapon != nil {

			act, cost := GetSimplePlayerAction(PlayerSelRanged, pl, gm)
			AddPlayerAction(act, pl, cost, timesystem.RangedAttackKind)
			pl.InputStates.HasKeyInput = true
		}
	}

	if inpututil.IsKeyJustReleased(ebiten.KeyG) {
		act, cost := GetSimplePlayerAction(PlayerPickupFromFloor, pl, gm)
		AddPlayerAction(act, pl, cost, timesystem.PickupItemKind)
		pl.InputStates.HasKeyInput = true
	}

	if inpututil.IsKeyJustReleased(ebiten.KeyL) {

		//ind := common.GetTileIndexFromCursor()

		ind := graphics.CoordTransformer.IndexFromLogicalXY(pl.Pos.X, pl.Pos.Y)
		gm.ApplyColorMatrixToIndex(ind, graphics.GreenColorMatrix)

	}

	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButton2) {

		cursorX, cursorY := graphics.CursorPosition(*pl.Pos)

		playerUI.InformationUI.InfoSelectionWindow(cursorX, cursorY)
		pl.InputStates.InfoMeuOpen = true

	}

	if pl.InputStates.InfoMeuOpen == true {

		if inpututil.IsKeyJustReleased(ebiten.KeyEscape) {
			playerUI.InformationUI.CloseWindows()

		}

	}

	//Opening the info window. Todo, I need to use some sort of state variable to make sure the right click
	// Does not overlap with over right clicks such as in the throwing and shooting menu

	return pl.InputStates.HasKeyInput

}
