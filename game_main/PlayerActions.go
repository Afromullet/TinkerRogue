package main

import (
	"game_main/ecshelper"
	"game_main/graphics"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

/*
Anything related to player actions. Functions here were split between files.
It will be easier to track what changes I'm making by putting them in one file
Not too many player actions yet. They will be broken out as they evolve.

The way player ranged attacks and throwing is handled is different from monsters.
There's information that has to be displayed to the user, such as the AOE of their attacks
That's what this file is for

*/

// Applies the throwable
func ApplyThrowable(g *Game, item *Item, shape graphics.TileBasedShape, throwerPos *ecshelper.Position) {

	t := item.ItemEffect(THROWABLE_NAME).(*Throwable)

	if t.vx != nil {

		t.vx.ResetVX()
		graphics.AddVXArea(graphics.NewVisualEffectArea(t.Shape, t.vx))

	}

	//t.ReadyThrowAreaVX()
	pos := GetTilePositions(t.Shape)

	//TODO, this will be slow in case there are a lot of creatures
	for _, c := range g.World.Query(g.WorldTags["monsters"]) {

		curPos := c.Components[ecshelper.PositionComponent].(*ecshelper.Position)
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
			g.gameMap.ApplyColorMatrix(PrevThrowInds, graphics.NewEmptyMatrix())

		}

	}

	throwable := g.playerData.ThrowableItem.ItemEffect(THROWABLE_NAME).(*Throwable)

	s.UpdatePosition(cursorX, cursorY)
	indices = s.GetIndices()

	for _, i := range indices {

		pos := PositionFromIndex(i)

		if pos.InRange(g.playerData.position, throwable.ThrowingRange) {
			g.gameMap.ApplyColorMatrixToIndex(i, graphics.GreenColorMatrix)

		} else {

			g.gameMap.ApplyColorMatrixToIndex(i, graphics.RedColorMatrix)

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

			g.playerData.ThrowPreparedItem(g.playerData.inventory)

			ApplyThrowable(g, g.playerData.ThrowableItem, g.playerData.ThrowingAOEShape, g.playerData.position)

		}

		//Cancel throwing
		if inpututil.IsMouseButtonJustPressed(ebiten.MouseButton2) {

			log.Println("Removing throwable")
			g.gameMap.ApplyColorMatrix(PrevThrowInds, graphics.NewEmptyMatrix())
			g.SetThrowableItemSelected(false)

		}
	}

}

func DrawRangedAttackAOE(g *Game) {

	cursorX, cursorY := ebiten.CursorPosition()

	s := g.playerData.RangedWeaponAOEShape
	var indices []int
	if cursorX != prevCursorX || cursorY != prevCursorY {

		if prevCursorX != 0 && prevCursorY != 0 {
			g.gameMap.ApplyColorMatrix(PrevRangedAttInds, graphics.NewEmptyMatrix())

		}

	}

	s.UpdatePosition(cursorX, cursorY)
	indices = s.GetIndices()

	for _, i := range indices {

		pos := PositionFromIndex(i)

		if pos.InRange(g.playerData.position, g.playerData.RangedWeaponMaxDistance) {
			g.gameMap.ApplyColorMatrixToIndex(i, graphics.GreenColorMatrix)

		} else {

			g.gameMap.ApplyColorMatrixToIndex(i, graphics.RedColorMatrix)

		}

	}

	prevCursorX, prevCursorY = cursorX, cursorY
	PrevRangedAttInds = indices

}

func HandlePlayerRangedAttack(g *Game) {

	if g.playerData.isTargeting {

		msg := ecshelper.GetComponentType[*UserMessage](g.playerData.PlayerEntity, userMessage)

		msg.GameStateMessage = "Shooting"
		DrawRangedAttackAOE(g)

		//Cancel throwing
		if inpututil.IsMouseButtonJustPressed(ebiten.MouseButton2) {

			g.playerData.isTargeting = false
			g.gameMap.ApplyColorMatrix(PrevRangedAttInds, graphics.NewEmptyMatrix())
			//log.Println("Removing throwable")

		}

		if inpututil.IsMouseButtonJustPressed(ebiten.MouseButton1) {

			RangedAttackSystem(g, g.playerData.position)

		}

	}

}
