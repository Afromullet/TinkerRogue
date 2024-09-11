package input

import (
	"fmt"
	"game_main/actionmanager"
	"game_main/avatar"
	"game_main/common"
	"game_main/equipment"
	"game_main/gui"
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

// todo replace the keypressed with iskeyreleased
func PlayerActions(ecsmanager *common.EntityManager, pl *avatar.PlayerData, gm *worldmap.GameMap, playerUI *gui.PlayerUI, tm *timesystem.GameTurn) bool {

	actionQueue := common.GetComponentType[*actionmanager.ActionQueue](pl.PlayerEntity, actionmanager.ActionQueueComponent)

	// Should throw a panic here since the game can't proceed without the player having an action querue
	if actionQueue == nil {
		fmt.Println("No action queue for player")
	}

	keyPressed := false

	x := 0
	y := 0

	if inpututil.IsKeyJustReleased(ebiten.KeyW) {
		y = -1

		act, cost := GetPlayerMoveAction(PlayerMoveAction, ecsmanager, pl, gm, x, y)
		AddPlayerAction(act, pl, cost, actionmanager.MovementKind)
		keyPressed = true

	}

	if inpututil.IsKeyJustReleased(ebiten.KeyS) {
		y = 1
		act, cost := GetPlayerMoveAction(PlayerMoveAction, ecsmanager, pl, gm, x, y)
		AddPlayerAction(act, pl, cost, actionmanager.MovementKind)
		keyPressed = true

	}

	if inpututil.IsKeyJustReleased(ebiten.KeyA) {
		x = -1
		act, cost := GetPlayerMoveAction(PlayerMoveAction, ecsmanager, pl, gm, x, y)
		AddPlayerAction(act, pl, cost, actionmanager.MovementKind)
		keyPressed = true

	}

	if inpututil.IsKeyJustReleased(ebiten.KeyD) {
		x = 1
		act, cost := GetPlayerMoveAction(PlayerMoveAction, ecsmanager, pl, gm, x, y)
		AddPlayerAction(act, pl, cost, actionmanager.MovementKind)
		keyPressed = true

	}

	if inpututil.IsKeyJustReleased(ebiten.KeyK) {

		armor := equipment.GetArmor(pl.PlayerEntity)
		common.UpdateAttributes(pl.PlayerEntity, armor.ArmorClass, armor.Protection, armor.DodgeChance)
		keyPressed = true

	}

	if inpututil.IsKeyJustReleased(ebiten.KeyF) {

		act, cost := GetSimplePlayerAction(PlayerSelRanged, pl, gm)
		AddPlayerAction(act, pl, cost, actionmanager.RangedAttackKind)
		keyPressed = true

	}

	if inpututil.IsKeyJustReleased(ebiten.KeyG) {

		act, cost := GetSimplePlayerAction(PlayerPickupFromFloor, pl, gm)

		AddPlayerAction(act, pl, cost, actionmanager.PickupItemKind)
		keyPressed = true

	}

	if inpututil.IsKeyJustReleased(ebiten.KeySpace) {

	}

	return keyPressed

}
