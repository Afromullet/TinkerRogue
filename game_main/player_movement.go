package main

import (
	"github.com/hajimehoshi/ebiten/v2"
)

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

	if ebiten.IsKeyPressed(ebiten.KeyG) {

		itemFromTile, _ := g.gameMap.GrabItemFromTile(0, g.playerData.position)

		if itemFromTile != nil {
			g.playerData.inventory.AddItemToInventory(itemFromTile)
		}

	}
	if ebiten.IsKeyPressed(ebiten.KeySpace) {
		turntaken = true
	}

	index := GetIndexFromXY(g.playerData.position.X+x, g.playerData.position.Y+y)
	tile := g.gameMap.Tiles[index]

	if !tile.Blocked {
		g.playerData.position.X += x
		g.playerData.position.Y += y

	}

	if x != 0 || y != 0 || turntaken {
		g.Turn = GetNextState(g.Turn)
		g.TurnCounter = 0
	}
}
