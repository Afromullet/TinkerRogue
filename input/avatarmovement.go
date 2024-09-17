package input

import (
	"fmt"

	"game_main/avatar"
	"game_main/common"
	"game_main/gear"
	"game_main/graphics"
	"game_main/gui"
	"game_main/monsters"
	"game_main/timesystem"
	"game_main/worldmap"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

var prevCursorX, prevCursorY int
var PrevThrowInds []int
var PrevRangedAttInds []int

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
		pl.HasKeyInput = true
	}

	if inpututil.IsKeyJustReleased(ebiten.KeyS) {
		y = 1
		act, cost := GetPlayerMoveAction(PlayerMoveAction, ecsmanager, pl, gm, x, y)
		AddPlayerAction(act, pl, cost, timesystem.MovementKind)
		pl.HasKeyInput = true
	}

	if inpututil.IsKeyJustReleased(ebiten.KeyA) {
		x = -1
		act, cost := GetPlayerMoveAction(PlayerMoveAction, ecsmanager, pl, gm, x, y)
		AddPlayerAction(act, pl, cost, timesystem.MovementKind)
		pl.HasKeyInput = true
	}

	if inpututil.IsKeyJustReleased(ebiten.KeyD) {
		x = 1
		act, cost := GetPlayerMoveAction(PlayerMoveAction, ecsmanager, pl, gm, x, y)
		AddPlayerAction(act, pl, cost, timesystem.MovementKind)
		pl.HasKeyInput = true
	}

	//Diagonal movement
	if inpututil.IsKeyJustReleased(ebiten.KeyQ) {
		y = -1
		x = -1
		act, cost := GetPlayerMoveAction(PlayerMoveAction, ecsmanager, pl, gm, x, y)
		AddPlayerAction(act, pl, cost, timesystem.MovementKind)
		pl.HasKeyInput = true
	}

	if inpututil.IsKeyJustReleased(ebiten.KeyE) {
		y = -1
		x = +1
		act, cost := GetPlayerMoveAction(PlayerMoveAction, ecsmanager, pl, gm, x, y)
		AddPlayerAction(act, pl, cost, timesystem.MovementKind)
		pl.HasKeyInput = true
	}

	if inpututil.IsKeyJustReleased(ebiten.KeyZ) {
		y = +1
		x = -1
		act, cost := GetPlayerMoveAction(PlayerMoveAction, ecsmanager, pl, gm, x, y)
		AddPlayerAction(act, pl, cost, timesystem.MovementKind)
		pl.HasKeyInput = true
	}

	if inpututil.IsKeyJustReleased(ebiten.KeyC) {
		y = +1
		x = +1
		act, cost := GetPlayerMoveAction(PlayerMoveAction, ecsmanager, pl, gm, x, y)
		AddPlayerAction(act, pl, cost, timesystem.MovementKind)
		pl.HasKeyInput = true
	}

	if inpututil.IsKeyJustReleased(ebiten.KeySpace) {

		playerPos := common.GetPosition(pl.PlayerEntity)

		ind := graphics.IndexFromXY(playerPos.X, playerPos.Y)

		if gm.Tiles[ind].TileType == worldmap.STAIRS_DOWN {

			monsters.ClearAllCreatures(ecsmanager)
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

	if inpututil.IsKeyJustReleased(ebiten.KeyK) {

		armor := gear.GetArmor(pl.PlayerEntity)
		common.UpdateAttributes(pl.PlayerEntity, armor.ArmorClass, armor.Protection, armor.DodgeChance)
		pl.HasKeyInput = true
	}

	if inpututil.IsKeyJustReleased(ebiten.KeyF) {

		if !pl.IsThrowing {

			act, cost := GetSimplePlayerAction(PlayerSelRanged, pl, gm)
			AddPlayerAction(act, pl, cost, timesystem.RangedAttackKind)
			pl.HasKeyInput = true
		}
	}

	if inpututil.IsKeyJustReleased(ebiten.KeyG) {
		act, cost := GetSimplePlayerAction(PlayerPickupFromFloor, pl, gm)
		AddPlayerAction(act, pl, cost, timesystem.PickupItemKind)
		pl.HasKeyInput = true
	}

	return pl.HasKeyInput

}
