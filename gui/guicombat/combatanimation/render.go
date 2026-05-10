package combatanimation

import (
	"image/color"

	"github.com/bytearena/ecs"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

func (cam *CombatAnimationMode) ensureGridImageCached() {
	if cam.cachedGridImage != nil && cam.cachedGridCellSize == cam.cellSize {
		return
	}

	cam.cachedGridImage = ebiten.NewImage(cam.gridWidth, cam.gridHeight)
	cam.cachedGridCellSize = cam.cellSize

	gridColor := color.RGBA{R: 60, G: 60, B: 70, A: 255}

	for row := 0; row <= 3; row++ {
		y := float32(row * cam.cellSize)
		vector.StrokeLine(cam.cachedGridImage, 0, y, float32(cam.gridWidth), y, 1, gridColor, false)
	}

	for col := 0; col <= 3; col++ {
		x := float32(col * cam.cellSize)
		vector.StrokeLine(cam.cachedGridImage, x, 0, x, float32(cam.gridHeight), 1, gridColor, false)
	}
}

func (cam *CombatAnimationMode) drawGridBackground(screen *ebiten.Image, baseX, baseY int) {
	cam.ensureGridImageCached()
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(baseX), float64(baseY))
	screen.DrawImage(cam.cachedGridImage, op)
}

func (cam *CombatAnimationMode) drawSquadNames(screen *ebiten.Image) {
	_ = cam.Queries.SquadCache.GetSquadName(cam.attackerSquadID)
	_ = cam.Queries.SquadCache.GetSquadName(cam.defenderSquadID)
}

func (cam *CombatAnimationMode) renderSquadWithUnitColors(
	screen *ebiten.Image, squadID ecs.EntityID,
	baseX, baseY, cellSize int, facingLeft bool, isAttacker bool,
) {
	unitIDs := cam.Queries.SquadCache.GetUnitIDsInSquad(squadID)

	for _, unitID := range unitIDs {
		var colorScale *ebiten.ColorScale

		if isAttacker {
			colorScale = cam.getAttackHighlightColor(unitID)
		} else {
			colorScale = cam.getDefenderHighlightColor(unitID)
		}

		cam.squadRenderer.RenderUnitWithColor(screen, unitID, baseX, baseY, cellSize, facingLeft, colorScale)
	}
}
