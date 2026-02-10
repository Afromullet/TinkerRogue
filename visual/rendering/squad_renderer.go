package rendering

import (
	"github.com/bytearena/ecs"
	"github.com/hajimehoshi/ebiten/v2"
)

// SquadCombatRenderer handles rendering a squad's 3x3 grid of units
// for the combat animation screen.
type SquadCombatRenderer struct {
	unitProvider UnitInfoProvider
}

// NewSquadCombatRenderer creates a new squad combat renderer
func NewSquadCombatRenderer(unitProvider UnitInfoProvider) *SquadCombatRenderer {
	return &SquadCombatRenderer{
		unitProvider: unitProvider,
	}
}

// RenderSquad draws a squad's units at their grid positions.
// Parameters:
//   - screen: The ebiten image to draw on
//   - squadID: The squad to render
//   - baseX, baseY: Top-left corner of the 3x3 grid area
//   - cellSize: Size of each grid cell in pixels
//   - facingLeft: If true, mirror the grid horizontally (for defender)
func (r *SquadCombatRenderer) RenderSquad(
	screen *ebiten.Image,
	squadID ecs.EntityID,
	baseX, baseY int,
	cellSize int,
	facingLeft bool,
) {
	unitIDs := r.unitProvider.GetUnitIDsInSquad(squadID)

	for _, unitID := range unitIDs {
		r.RenderUnitWithColor(screen, unitID, baseX, baseY, cellSize, facingLeft, nil)
	}
}

// RenderSquadWithHighlight draws a squad with an optional highlight color overlay
// for animation effects (e.g., attacking units flash)
func (r *SquadCombatRenderer) RenderSquadWithHighlight(
	screen *ebiten.Image,
	squadID ecs.EntityID,
	baseX, baseY int,
	cellSize int,
	facingLeft bool,
	highlightUnits []ecs.EntityID,
	highlightColor *ebiten.ColorScale,
) {
	unitIDs := r.unitProvider.GetUnitIDsInSquad(squadID)

	for _, unitID := range unitIDs {
		shouldHighlight := false
		for _, hID := range highlightUnits {
			if hID == unitID {
				shouldHighlight = true
				break
			}
		}

		var unitColor *ebiten.ColorScale
		if shouldHighlight {
			unitColor = highlightColor
		}
		r.RenderUnitWithColor(screen, unitID, baseX, baseY, cellSize, facingLeft, unitColor)
	}
}

// RenderUnitWithColor draws a unit with a color overlay
func (r *SquadCombatRenderer) RenderUnitWithColor(
	screen *ebiten.Image,
	unitID ecs.EntityID,
	baseX, baseY int,
	cellSize int,
	facingLeft bool,
	colorScale *ebiten.ColorScale,
) {
	info := r.unitProvider.GetUnitRenderInfo(unitID)
	if info == nil || !info.IsAlive || info.Image == nil {
		return
	}

	// Calculate pixel position from grid position
	// Apply 90 degree rotation so squads face each other
	var displayCol, displayRow int

	if facingLeft {
		// Defender: 90 degree clockwise rotation
		displayCol = info.AnchorRow
		displayRow = 2 - info.AnchorCol
	} else {
		// Attacker: 90 degree counter-clockwise rotation
		displayCol = 2 - info.AnchorRow
		displayRow = info.AnchorCol
	}

	pixelX := baseX + (displayCol * cellSize)
	pixelY := baseY + (displayRow * cellSize)

	// Get sprite dimensions
	spriteWidth := info.Image.Bounds().Dx()
	spriteHeight := info.Image.Bounds().Dy()

	// Calculate cell dimensions
	unitCellWidth := info.Width * cellSize
	unitCellHeight := info.Height * cellSize

	// Calculate scale
	scaleX := float64(unitCellWidth) / float64(spriteWidth)
	scaleY := float64(unitCellHeight) / float64(spriteHeight)
	scale := scaleX
	if scaleY < scaleX {
		scale = scaleY
	}

	// Center the sprite
	scaledWidth := float64(spriteWidth) * scale
	scaledHeight := float64(spriteHeight) * scale
	offsetX := (float64(unitCellWidth) - scaledWidth) / 2
	offsetY := (float64(unitCellHeight) - scaledHeight) / 2

	// Create draw options with color scale
	op := &ebiten.DrawImageOptions{}

	if facingLeft {
		op.GeoM.Scale(-scale, scale)
		op.GeoM.Translate(scaledWidth, 0)
	} else {
		op.GeoM.Scale(scale, scale)
	}

	op.GeoM.Translate(float64(pixelX)+offsetX, float64(pixelY)+offsetY)

	// Apply color scale for highlight effect (if provided)
	if colorScale != nil {
		op.ColorScale = *colorScale
	}

	screen.DrawImage(info.Image, op)
}
