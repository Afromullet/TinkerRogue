package main

import (
	"fmt"
	"game_main/graphics"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

var prevCursorX, prevCursorY int
var PrevThrowInds []int
var PrevRangedAttInds []int

// todo replace the keypressed with iskeyreleased
func PlayerActions(g *Game) {

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
		UpdateAttributes(g.playerData.PlayerEntity)
	}

	if inpututil.IsKeyJustReleased(ebiten.KeyF) {

		g.gameMap.ApplyColorMatrix(PrevRangedAttInds, graphics.NewEmptyMatrix())

		g.playerData.isTargeting = true
		g.playerData.PrepareRangedAttack()
		DrawRangedAttackAOE(g)

	}

	if inpututil.IsKeyJustReleased(ebiten.KeyG) {

		log.Print("Press G")

		itemFromTile, _ := g.gameMap.RemoveItemFromTile(0, g.playerData.position)

		if itemFromTile != nil {
			g.playerData.inventory.AddItem(itemFromTile)
		}

	}

	if inpututil.IsKeyJustReleased(ebiten.KeyT) {

		fmt.Println("Is window open ", g.mainPlayerInterface.IsWindowOpen(g.itemsUI.throwableItemDisplay.itemDisplay.rootWindow))

	}

	if inpututil.IsKeyJustReleased(ebiten.KeySpace) {

		turntaken = true
	}

	HandlePlayerThrowable(g)
	HandlePlayerRangedAttack(g)

	nextPosition := Position{
		X: g.playerData.position.X + x,
		Y: g.playerData.position.Y + y,
	}

	index := graphics.IndexFromXY(nextPosition.X, nextPosition.Y)
	nextTile := g.gameMap.Tiles[index]

	index = graphics.IndexFromXY(g.playerData.position.X, g.playerData.position.Y)
	oldTile := g.gameMap.Tiles[index]

	if !nextTile.Blocked {
		g.gameMap.PlayerVisible.Compute(g.gameMap, g.playerData.position.X, g.playerData.position.Y, 8)
		g.playerData.position.X = nextPosition.X
		g.playerData.position.Y = nextPosition.Y
		nextTile.Blocked = true
		oldTile.Blocked = false

	} else {
		//Determine if the tyle is blocked because there's a creature

		c := GetCreatureAtPosition(g, &nextPosition)

		if c != nil {

			MeleeAttackSystem(g, g.playerData.position, &nextPosition)
		}

	}

	//AttackSystem(g, g.playerData.position, defendingMonsterTestPosition)
	//AttackSystem(g, defendingMonsterTestPosition, g.playerData.position)
	if x != 0 || y != 0 || turntaken {
		g.Turn = GetNextState(g.Turn)
		g.TurnCounter = 0
	}
}
