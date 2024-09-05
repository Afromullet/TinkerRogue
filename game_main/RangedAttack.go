package main

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

func DrawRangedAttackAOE(g *Game) {

	cursorX, cursorY := ebiten.CursorPosition()

	s := g.playerData.RangedWeaponAOEShape
	var indices []int
	if cursorX != prevCursorX || cursorY != prevCursorY {

		if prevCursorX != 0 && prevCursorY != 0 {
			g.gameMap.ApplyColorMatrix(PrevRangedAttInds, NewEmptyMatrix())

		}

	}

	s.UpdatePosition(cursorX, cursorY)
	indices = s.GetIndices()

	inRangeCM := ColorMatrix{0, 1, 0, 0.5, true}
	outOfRangeCM := ColorMatrix{1, 0, 0, 0.5, true}

	for _, i := range indices {

		pos := PositionFromIndex(i)

		if pos.InRange(g.playerData.position, g.playerData.RangedWeaponMaxDistance) {
			g.gameMap.ApplyColorMatrixToIndex(i, inRangeCM)

		} else {

			g.gameMap.ApplyColorMatrixToIndex(i, outOfRangeCM)

		}

	}

	prevCursorX, prevCursorY = cursorX, cursorY
	PrevRangedAttInds = indices

}

func HandlePlayerRangedAttack(g *Game) {

	if g.playerData.isTargeting {

		DrawRangedAttackAOE(g)

		//Cancel throwing
		if inpututil.IsMouseButtonJustPressed(ebiten.MouseButton2) {

			g.playerData.isTargeting = false
			g.gameMap.ApplyColorMatrix(PrevRangedAttInds, NewEmptyMatrix())
			//log.Println("Removing throwable")

		}

		if inpututil.IsMouseButtonJustPressed(ebiten.MouseButton1) {

			RangedAttackSystem(g, g.playerData.position)

		}

	}

}
