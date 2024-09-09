package main

import (
	"fmt"
	"game_main/avatar"
	"game_main/common"
	"game_main/equipment"
	"game_main/graphics"
	"game_main/worldmap"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

var prevCursorX, prevCursorY int
var PrevThrowInds []int
var PrevRangedAttInds []int

// todo replace the keypressed with iskeyreleased
func PlayerActions(ecsmanager *common.EntityManager, pl *avatar.PlayerData, gm *worldmap.GameMap, playerUI *PlayerUI, tm *common.TimeSystem) {

	turntaken := false
	//players := g.WorldTags["players"]

	x := 0
	y := 0

	if ebiten.IsKeyPressed(ebiten.KeyW) {
		y = -1
	}

	if ebiten.IsKeyPressed(ebiten.KeyS) {
		y = 1
	}

	if ebiten.IsKeyPressed(ebiten.KeyA) {
		x = -1
	}

	if ebiten.IsKeyPressed(ebiten.KeyD) {
		x = 1
	}

	if inpututil.IsKeyJustReleased(ebiten.KeyK) {

		armor := equipment.GetArmor(pl.PlayerEntity)
		common.UpdateAttributes(pl.PlayerEntity, armor.ArmorClass, armor.Protection, armor.DodgeChance)
	}

	if inpututil.IsKeyJustReleased(ebiten.KeyF) {

		gm.ApplyColorMatrix(PrevRangedAttInds, graphics.NewEmptyMatrix())

		pl.Targeting = true
		pl.PrepareRangedAttack()
		DrawRangedAttackAOE(pl, gm)

	}

	if inpututil.IsKeyJustReleased(ebiten.KeyG) {

		log.Print("Press G")

		itemFromTile, _ := gm.RemoveItemFromTile(0, pl.Pos)

		if itemFromTile != nil {
			pl.Inv.AddItem(itemFromTile)
		}

	}

	if inpututil.IsKeyJustReleased(ebiten.KeyT) {

		fmt.Println("Is window open ", playerUI.mainPlayerInterface.IsWindowOpen(playerUI.itemsUI.throwableItemDisplay.itemDisplay.rootWindow))

	}

	if inpututil.IsKeyJustReleased(ebiten.KeySpace) {

		turntaken = true
	}

	HandlePlayerThrowable(ecsmanager, pl, gm, playerUI)
	HandlePlayerRangedAttack(ecsmanager, pl, gm)

	nextPosition := common.Position{
		X: pl.Pos.X + x,
		Y: pl.Pos.Y + y,
	}

	index := graphics.IndexFromXY(nextPosition.X, nextPosition.Y)
	nextTile := gm.Tiles[index]

	index = graphics.IndexFromXY(pl.Pos.X, pl.Pos.Y)
	oldTile := gm.Tiles[index]

	if !nextTile.Blocked {
		gm.PlayerVisible.Compute(gm, pl.Pos.X, pl.Pos.Y, 8)
		pl.Pos.X = nextPosition.X
		pl.Pos.Y = nextPosition.Y
		nextTile.Blocked = true
		oldTile.Blocked = false

	} else {
		//Determine if the tyle is blocked because there's a creature

		c := GetCreatureAtPosition(ecsmanager, &nextPosition)

		if c != nil {

			MeleeAttackSystem(ecsmanager, pl, gm, pl.Pos, &nextPosition)
		}

	}

	//AttackSystem(g, pl.position, defendingMonsterTestPosition)
	//AttackSystem(g, defendingMonsterTestPosition, pl.position)
	if x != 0 || y != 0 || turntaken {
		tm.Turn = common.GetNextState(tm.Turn)
		tm.TurnCounter = 0
	}
}
