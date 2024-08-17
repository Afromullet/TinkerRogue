package main

import (
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

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

	if ebiten.IsKeyPressed(ebiten.KeyI) {

		g.craftingUI.UpdateCraftingInventory(g)

	}

	if inpututil.IsKeyJustReleased(ebiten.KeyG) {
		//if ebiten.IsKeyPressed(ebiten.KeyG) {

		log.Print("Press G")

		itemFromTile, _ := g.gameMap.GrabItemFromTile(0, g.playerData.position)

		if itemFromTile != nil {
			g.playerData.inventory.AddItemToInventory(itemFromTile)
		}

	}
	if ebiten.IsKeyPressed(ebiten.KeySpace) {
		turntaken = true
	}

	if inpututil.IsMouseButtonJustReleased(ebiten.MouseButton0) {
		log.Print("Throwing stuff")
	}

	nextPosition := Position{
		X: g.playerData.position.X + x,
		Y: g.playerData.position.Y + y,
	}

	index := GetIndexFromXY(nextPosition.X, nextPosition.Y)
	nextTile := g.gameMap.Tiles[index]

	index = GetIndexFromXY(g.playerData.position.X, g.playerData.position.Y)
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
			log.Print("Creature here")
			AttackSystem(g, g.playerData.position, &nextPosition)
		}

	}

	//AttackSystem(g, g.playerData.position, defendingMonsterTestPosition)
	//AttackSystem(g, defendingMonsterTestPosition, g.playerData.position)
	if x != 0 || y != 0 || turntaken {
		g.Turn = GetNextState(g.Turn)
		g.TurnCounter = 0
	}
}
