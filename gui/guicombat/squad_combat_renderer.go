package guicombat

import (
	"game_main/common"
	"game_main/gui/guicomponents"
	"game_main/rendering"
	"game_main/squads"

	"github.com/bytearena/ecs"
	"github.com/hajimehoshi/ebiten/v2"
)

// SquadCombatRenderer handles rendering a squad's 3x3 grid of units
// for the combat animation screen.
type SquadCombatRenderer struct {
	queries *guicomponents.GUIQueries
}

// NewSquadCombatRenderer creates a new squad combat renderer
func NewSquadCombatRenderer(queries *guicomponents.GUIQueries) *SquadCombatRenderer {
	return &SquadCombatRenderer{
		queries: queries,
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
	// Get all units in the squad
	unitIDs := r.queries.SquadCache.GetUnitIDsInSquad(squadID)

	for _, unitID := range unitIDs {
		r.RenderUnit(screen, unitID, baseX, baseY, cellSize, facingLeft)
	}
}

// RenderUnit draws a single unit at its grid position
func (r *SquadCombatRenderer) RenderUnit(
	screen *ebiten.Image,
	unitID ecs.EntityID,
	baseX, baseY int,
	cellSize int,
	facingLeft bool,
) {
	// Get the unit entity
	entity := common.FindEntityByID(r.queries.ECSManager, unitID)
	if entity == nil {
		return
	}

	// Get grid position
	gridPos := common.GetComponentType[*squads.GridPositionData](entity, squads.GridPositionComponent)
	if gridPos == nil {
		return
	}

	// Get renderable (sprite) - ignore Visible flag since combat animation
	// should show all units regardless of world map visibility
	renderable := common.GetComponentType[*rendering.Renderable](entity, rendering.RenderableComponent)
	if renderable == nil || renderable.Image == nil {
		return
	}

	// Check if unit is alive
	attr := common.GetComponentType[*common.Attributes](entity, common.AttributeComponent)
	if attr == nil || attr.CurrentHealth <= 0 {
		return // Don't render dead units
	}

	// Calculate pixel position from grid position
	// Grid is 3 columns (0-2), 3 rows (0-2)
	// Apply 90° rotation so squads face each other
	var displayCol, displayRow int

	if facingLeft {
		// Defender: 90° clockwise rotation
		// Row 0 units appear on left side (facing attacker)
		displayCol = gridPos.AnchorRow
		displayRow = 2 - gridPos.AnchorCol
	} else {
		// Attacker: 90° counter-clockwise rotation
		// Row 0 units appear on right side (facing defender)
		displayCol = 2 - gridPos.AnchorRow
		displayRow = gridPos.AnchorCol
	}

	// Calculate pixel position
	pixelX := baseX + (displayCol * cellSize)
	pixelY := baseY + (displayRow * cellSize)

	// Get sprite dimensions
	spriteWidth := renderable.Image.Bounds().Dx()
	spriteHeight := renderable.Image.Bounds().Dy()

	// Calculate cell dimensions (multi-cell units span multiple cells)
	unitCellWidth := gridPos.Width * cellSize
	unitCellHeight := gridPos.Height * cellSize

	// Calculate scale to fit unit in its cell(s) while maintaining aspect ratio
	scaleX := float64(unitCellWidth) / float64(spriteWidth)
	scaleY := float64(unitCellHeight) / float64(spriteHeight)

	// Use the smaller scale to maintain aspect ratio
	scale := scaleX
	if scaleY < scaleX {
		scale = scaleY
	}

	// Center the sprite within the cell(s)
	scaledWidth := float64(spriteWidth) * scale
	scaledHeight := float64(spriteHeight) * scale
	offsetX := (float64(unitCellWidth) - scaledWidth) / 2
	offsetY := (float64(unitCellHeight) - scaledHeight) / 2

	// Create draw options
	op := &ebiten.DrawImageOptions{}

	// If facing left, flip the sprite horizontally
	if facingLeft {
		op.GeoM.Scale(-scale, scale)
		op.GeoM.Translate(scaledWidth, 0) // Adjust for flip
	} else {
		op.GeoM.Scale(scale, scale)
	}

	// Translate to final position
	op.GeoM.Translate(float64(pixelX)+offsetX, float64(pixelY)+offsetY)

	// Draw the sprite
	screen.DrawImage(renderable.Image, op)
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
	// Get all units in the squad
	unitIDs := r.queries.SquadCache.GetUnitIDsInSquad(squadID)

	for _, unitID := range unitIDs {
		// Check if this unit should be highlighted
		shouldHighlight := false
		for _, hID := range highlightUnits {
			if hID == unitID {
				shouldHighlight = true
				break
			}
		}

		if shouldHighlight && highlightColor != nil {
			r.RenderUnitWithColor(screen, unitID, baseX, baseY, cellSize, facingLeft, highlightColor)
		} else {
			r.RenderUnit(screen, unitID, baseX, baseY, cellSize, facingLeft)
		}
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
	// Get the unit entity
	entity := common.FindEntityByID(r.queries.ECSManager, unitID)
	if entity == nil {
		return
	}

	// Get grid position
	gridPos := common.GetComponentType[*squads.GridPositionData](entity, squads.GridPositionComponent)
	if gridPos == nil {
		return
	}

	// Get renderable (sprite) - ignore Visible flag since combat animation
	// should show all units regardless of world map visibility
	renderable := common.GetComponentType[*rendering.Renderable](entity, rendering.RenderableComponent)
	if renderable == nil || renderable.Image == nil {
		return
	}

	// Check if unit is alive
	attr := common.GetComponentType[*common.Attributes](entity, common.AttributeComponent)
	if attr == nil || attr.CurrentHealth <= 0 {
		return
	}

	// Calculate pixel position from grid position
	// Apply 90° rotation so squads face each other
	var displayCol, displayRow int

	if facingLeft {
		// Defender: 90° clockwise rotation
		// Row 0 units appear on left side (facing attacker)
		displayCol = gridPos.AnchorRow
		displayRow = 2 - gridPos.AnchorCol
	} else {
		// Attacker: 90° counter-clockwise rotation
		// Row 0 units appear on right side (facing defender)
		displayCol = 2 - gridPos.AnchorRow
		displayRow = gridPos.AnchorCol
	}

	pixelX := baseX + (displayCol * cellSize)
	pixelY := baseY + (displayRow * cellSize)

	// Get sprite dimensions
	spriteWidth := renderable.Image.Bounds().Dx()
	spriteHeight := renderable.Image.Bounds().Dy()

	// Calculate cell dimensions
	unitCellWidth := gridPos.Width * cellSize
	unitCellHeight := gridPos.Height * cellSize

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

	// Apply color scale for highlight effect
	op.ColorScale = *colorScale

	screen.DrawImage(renderable.Image, op)
}
