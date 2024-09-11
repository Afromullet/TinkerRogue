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
func PlayerActions(ecsmanager *common.EntityManager, pl *avatar.PlayerData, gm *worldmap.GameMap, playerUI *gui.PlayerUI, tm *timesystem.GameTurn) {

	actionQueue := common.GetComponentType[*actionmanager.ActionQueue](pl.PlayerEntity, actionmanager.ActionQueueComponent)

	// Should throw a panic here since the game can't proceed without the player having an action querue
	if actionQueue == nil {
		fmt.Println("No action queue for player")
	}

	TurnTaken = false

	x := 0
	y := 0

	if ebiten.IsKeyPressed(ebiten.KeyW) {
		y = -1
		TurnTaken = true
	}

	if ebiten.IsKeyPressed(ebiten.KeyS) {
		y = 1
		TurnTaken = true
	}

	if ebiten.IsKeyPressed(ebiten.KeyA) {
		x = -1
		TurnTaken = true
	}

	if ebiten.IsKeyPressed(ebiten.KeyD) {
		x = 1
		TurnTaken = true
	}

	if inpututil.IsKeyJustReleased(ebiten.KeyK) {

		armor := equipment.GetArmor(pl.PlayerEntity)
		common.UpdateAttributes(pl.PlayerEntity, armor.ArmorClass, armor.Protection, armor.DodgeChance)
		TurnTaken = true
	}

	if inpututil.IsKeyJustReleased(ebiten.KeyF) {

		act, cost := GetSimplePlayerAction(PlayerSelRanged, pl, gm)
		AddPlayerAction(act, pl, cost, actionmanager.RangedAttackKind)
		TurnTaken = true

	}

	if inpututil.IsKeyJustReleased(ebiten.KeyG) {

		act, cost := GetSimplePlayerAction(PlayerPickupFromFloor, pl, gm)

		AddPlayerAction(act, pl, cost, actionmanager.PickupItemKind)
		TurnTaken = true

	}

	if inpututil.IsKeyJustReleased(ebiten.KeySpace) {

		TurnTaken = true
	}

	if x != prevPosX || y != prevPosY {

		act, cost := GetPlayerMoveAction(PlayerMoveAction, ecsmanager, pl, gm, x, y)
		prevPosX = x
		prevPosY = y
		TurnTaken = true
		AddPlayerAction(act, pl, cost, actionmanager.MovementKind)

	}

	HandlePlayerThrowable(ecsmanager, pl, gm, playerUI)

	HandlePlayerRangedAttack(ecsmanager, pl, gm)

	PerformAllActions(ecsmanager, pl, tm, x, y)

}

// A placeholder for testing the action queue
func PerformAllActions2(ecsmanager *common.EntityManager, pl *avatar.PlayerData, tm *timesystem.GameTurn, x, y int) {

	//turntaken = false

	if x != 0 || y != 0 || TurnTaken {
		//tm.Turn = timesystem.GetNextState(tm.Turn)
		//tm.TurnCounter = 0
		actionmanager.ActionDispatcher.DebugOutput()
		actionmanager.ActionDispatcher.CleanController()
		actionmanager.ActionDispatcher.ExecuteFirst()
	}

}

// A placeholder for testing the action queue
func PerformAllActions(ecsmanager *common.EntityManager, pl *avatar.PlayerData, tm *timesystem.GameTurn, x, y int) {

	for _, c := range ecsmanager.World.Query(ecsmanager.WorldTags["monsters"]) {
		actionQueue := common.GetComponentType[*actionmanager.ActionQueue](c.Entity, actionmanager.ActionQueueComponent)

		for _, acts := range actionQueue.AllActions {

			acts.ActWrapper.Execute(actionQueue)

		}
	}

	actionQueue := common.GetComponentType[*actionmanager.ActionQueue](pl.PlayerEntity, actionmanager.ActionQueueComponent)

	for _, acts := range actionQueue.AllActions {
		acts.ActWrapper.Execute(actionQueue)
	}

	if x != 0 || y != 0 || TurnTaken {
		tm.Turn = timesystem.GetNextState(tm.Turn)
		tm.TurnCounter = 0
		actionmanager.ActionDispatcher.DebugOutput()
	}

}
