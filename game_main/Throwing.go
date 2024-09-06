package main

import (
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// Applies the throwable
func ApplyThrowable(g *Game, item *Item, throwerPos *Position) {

	t := item.ItemEffect(THROWABLE_NAME).(*Throwable)

	pos := GetTilePositions(t.Shape)

	//TODO, this will be slow in case there are a lot of creatures
	for _, c := range g.World.Query(g.WorldTags["monsters"]) {

		curPos := c.Components[PositionComponent].(*Position)
		crea := c.Components[CreatureComponent].(*Creature)

		for _, p := range pos {
			if curPos.IsEqual(&p) && curPos.InRange(throwerPos, t.ThrowingRange) {
				crea.AddEffects(item.Properties)
			}
		}

	}
}

func DrawThrowableAOE(g *Game) {

	cursorX, cursorY := ebiten.CursorPosition()

	s := g.playerData.ThrowingAOEShape
	var indices []int
	if cursorX != prevCursorX || cursorY != prevCursorY {

		if prevCursorX != 0 && prevCursorY != 0 {
			g.gameMap.ApplyColorMatrix(PrevThrowInds, NewEmptyMatrix())

		}

	}

	throwable := g.playerData.ThrowableItem.ItemEffect(THROWABLE_NAME).(*Throwable)

	s.UpdatePosition(cursorX, cursorY)
	indices = s.GetIndices()

	for _, i := range indices {

		pos := PositionFromIndex(i)

		if pos.InRange(g.playerData.position, throwable.ThrowingRange) {
			g.gameMap.ApplyColorMatrixToIndex(i, GreenColorMatrix)

		} else {

			g.gameMap.ApplyColorMatrixToIndex(i, RedColorMatrix)

		}

	}

	prevCursorX, prevCursorY = cursorX, cursorY
	PrevThrowInds = indices

}

func HandlePlayerThrowable(g *Game) {

	if g.IsThrowableItemSelected() {

		DrawThrowableAOE(g)

		//Press middle mouse button to throw

		//if inpututil.IsMouseButtonJustPressed(ebiten.MouseButton1)
		if inpututil.IsKeyJustReleased(ebiten.KeyT) {

			log.Println("Throwing item")

			g.playerData.ThrowPreparedItem()

			ApplyThrowable(g, g.playerData.ThrowableItem, g.playerData.position)

		}

		//Cancel throwing
		if inpututil.IsMouseButtonJustPressed(ebiten.MouseButton2) {

			log.Println("Removing throwable")
			g.gameMap.ApplyColorMatrix(PrevThrowInds, NewEmptyMatrix())
			g.SetThrowableItemSelected(false)

		}
	}

}
