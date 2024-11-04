package input

import (
	"game_main/avatar"
	"game_main/common"
	"game_main/gear"
	"game_main/graphics"
	"game_main/worldmap"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

func DrawLineFromPlayer(pl *avatar.PlayerData, endPos common.Position, gm *worldmap.GameMap) {

	indices := graphics.GetLineTo(*pl.Pos, endPos)

	gm.ApplyColorMatrix(indices, graphics.GreenColorMatrix)

	PrevTargetLineIndices = indices

}

func DrawRangedAttackAOE(pl *avatar.PlayerData, gm *worldmap.GameMap) {

	cursorX, cursorY := graphics.CursorPosition(*pl.Pos)

	s := pl.Equipment.RangedWeaponAOEShape

	rangedWep := common.GetComponentType[*gear.RangedWeapon](pl.Equipment.EqRangedWeapon, gear.RangedWeaponComponent)

	updater := UpdateDirection(&rangedWep.TargetArea)
	rangedWep.TargetArea.UpdateShape(updater)
	gm.ApplyColorMatrix(PrevRangedAttInds, graphics.NewEmptyMatrix()) //Clears previously applied rotation if there is any

	var indices []int
	if cursorX != prevCursorX || cursorY != prevCursorY {

		if prevCursorX != 0 && prevCursorY != 0 {
			gm.ApplyColorMatrix(PrevRangedAttInds, graphics.NewEmptyMatrix())

		}

	}

	s.UpdatePosition(cursorX, cursorY)
	indices = s.GetIndices()

	for _, i := range indices {

		//pos := common.PositionFromIndex(i, graphics.ScreenInfo.DungeonWidth)
		x, y := graphics.CoordTransformer.LogicalXYFromIndex(i)
		pos := common.Position{X: x, Y: y}

		if pos.InRange(pl.Pos, pl.Equipment.RangedWeaponMaxDistance) {
			gm.ApplyColorMatrixToIndex(i, graphics.GreenColorMatrix)

		} else {

			gm.ApplyColorMatrixToIndex(i, graphics.RedColorMatrix)

		}

	}

	prevCursorX, prevCursorY = cursorX, cursorY
	PrevRangedAttInds = indices

}

func DrawThrowableAOE(pl *avatar.PlayerData, gm *worldmap.GameMap) {

	cursorX, cursorY := graphics.CursorPosition(*pl.Pos)

	s := pl.Throwables.ThrowingAOEShape

	var indices []int
	if cursorX != prevCursorX || cursorY != prevCursorY {

		if prevCursorX != 0 && prevCursorY != 0 {
			gm.ApplyColorMatrix(PrevThrowInds, graphics.NewEmptyMatrix())

		}

	}

	throwable := pl.Throwables.ThrowableItem.ItemEffect(gear.THROWABLE_NAME).(*gear.Throwable)

	s.UpdatePosition(cursorX, cursorY)
	indices = s.GetIndices()

	for _, i := range indices {

		x, y := graphics.CoordTransformer.LogicalXYFromIndex(i)
		pos := common.Position{X: x, Y: y}

		if pos.InRange(pl.Pos, throwable.ThrowingRange) {
			gm.ApplyColorMatrixToIndex(i, graphics.GreenColorMatrix)

		} else {

			gm.ApplyColorMatrixToIndex(i, graphics.RedColorMatrix)

		}

	}

	prevCursorX, prevCursorY = cursorX, cursorY
	PrevThrowInds = indices

}

// Handle rotation if the shape has a direction
func UpdateDirection(shape *graphics.TileBasedShape) graphics.ShapeUpdater {

	updater := graphics.ExtractShapeParams(*shape)
	newDir := graphics.GetDirection(*shape)
	if newDir != graphics.NoDirection {

		if inpututil.IsKeyJustReleased(ebiten.KeyDigit1) {

			newDir := graphics.GetDirection(*shape)
			newDir = graphics.RotateLeft(newDir)
			updater.Direction = newDir

		} else if inpututil.IsKeyJustReleased(ebiten.KeyDigit2) {
			newDir := graphics.GetDirection(*shape)
			newDir = graphics.RotateRight(newDir)
			updater.Direction = newDir
		}

	}

	return updater
}
