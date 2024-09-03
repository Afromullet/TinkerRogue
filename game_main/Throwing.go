package main

import (
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// Applies the throwable
func ApplyThrowable(g *Game, item *Item) {

	t := item.GetItemEffect(THROWABLE_NAME).(Throwable)

	pos := GetTilePositions(t.shape)

	//TODO, this will be slow in case there are a lot of creatures
	for _, c := range g.World.Query(g.WorldTags["monsters"]) {

		curPos := c.Components[position].(*Position)
		crea := c.Components[creature].(*Creature)

		for _, p := range pos {
			if curPos.IsEqual(&p) {
				crea.AddEffects(item.properties)
			}
		}

	}
}

func DrawThrowableAOE(g *Game) {

	cursorX, cursorY := ebiten.CursorPosition()

	s := g.playerData.shape
	var indices []int
	if cursorX != prevCursorX || cursorY != prevCursorY {

		if prevCursorX != 0 && prevCursorY != 0 {
			g.gameMap.ApplyColorMatrix(previousIndices, NewEmptyMatrix())

		}

	}

	s.UpdatePosition(cursorX, cursorY)
	indices = s.GetIndices()
	cm := ColorMatrix{1, 0, 0, 0.5, true}
	g.gameMap.ApplyColorMatrix(indices, cm)
	prevCursorX, prevCursorY = cursorX, cursorY
	previousIndices = indices

}
func HandleThrowable(g *Game) {

	if g.IsThrowableItemSelected() {

		DrawThrowableAOE(g)

		//Press middle mouse button to throw

		//if inpututil.IsMouseButtonJustPressed(ebiten.MouseButton1)
		if inpututil.IsKeyJustReleased(ebiten.KeyT) {

			log.Println("Throwing item")
			//log.Println("Removing throwable")
			//g.gameMap.ApplyColorMatrix(previousIndices, NewEmptyMatrix())
			//g.SetThrowableItemSelected(false)
			////HandleThrowable(g)

			g.playerData.ThrowPreparedItem()
			ApplyThrowable(g, g.playerData.throwableItem)

		}

		//Cancel throwing
		if inpututil.IsMouseButtonJustPressed(ebiten.MouseButton2) {

			log.Println("Removing throwable")
			g.gameMap.ApplyColorMatrix(previousIndices, NewEmptyMatrix())
			g.SetThrowableItemSelected(false)
			////HandleThrowable(g)

		}
	}

}
