package rendering

import (
	"game_main/world/coords"
	"image/color"

	"github.com/bytearena/ecs"
	"github.com/hajimehoshi/ebiten/v2"
)

// SquadHighlightRenderer renders squad position highlights
type SquadHighlightRenderer struct {
	dataProvider    SquadInfoProvider
	selectedColor   color.Color
	factionColors   map[ecs.EntityID]color.Color // Maps faction ID to unique color
	defaultColor    color.Color                  // Fallback color for unknown factions
	borderThickness int
	viewport        CachedViewport
}

// NewSquadHighlightRenderer creates a renderer for squad highlights
func NewSquadHighlightRenderer(dataProvider SquadInfoProvider) *SquadHighlightRenderer {
	return &SquadHighlightRenderer{
		dataProvider:    dataProvider,
		selectedColor:   color.RGBA{R: 255, G: 255, B: 255, A: 255}, // White for selected
		factionColors:   make(map[ecs.EntityID]color.Color),
		defaultColor:    color.RGBA{R: 128, G: 128, B: 128, A: 150}, // Gray for unknown
		borderThickness: 3,
	}
}

// GetFactionColor returns a unique color for a faction, assigning one if needed
func (shr *SquadHighlightRenderer) GetFactionColor(factionID ecs.EntityID) color.Color {
	// Return cached color if already assigned
	if c, exists := shr.factionColors[factionID]; exists {
		return c
	}

	// Palette of distinct colors for factions
	colorPalette := []color.Color{
		color.RGBA{R: 0, G: 150, B: 255, A: 150},   // Blue
		color.RGBA{R: 255, G: 0, B: 0, A: 150},     // Red
		color.RGBA{R: 0, G: 200, B: 0, A: 150},     // Green
		color.RGBA{R: 255, G: 215, B: 0, A: 150},   // Gold/Yellow
		color.RGBA{R: 200, G: 0, B: 200, A: 150},   // Purple/Magenta
		color.RGBA{R: 255, G: 140, B: 0, A: 150},   // Orange
		color.RGBA{R: 0, G: 255, B: 255, A: 150},   // Cyan
		color.RGBA{R: 255, G: 105, B: 180, A: 150}, // Pink
	}

	// Assign color based on number of factions already assigned
	colorIndex := len(shr.factionColors) % len(colorPalette)
	assignedColor := colorPalette[colorIndex]
	shr.factionColors[factionID] = assignedColor

	return assignedColor
}

// Render draws highlights for all squads
func (shr *SquadHighlightRenderer) Render(
	screen *ebiten.Image,
	centerPos coords.LogicalPosition,
	currentFactionID ecs.EntityID,
	selectedSquadID ecs.EntityID,
) {
	vr := shr.viewport.GetRenderer(screen, centerPos)

	allSquads := shr.dataProvider.GetAllSquadIDs()

	for _, squadID := range allSquads {
		info := shr.dataProvider.GetSquadRenderInfo(squadID)
		if info == nil || info.IsDestroyed || info.Position == nil {
			continue
		}

		// Determine highlight color
		var highlightColor color.Color
		if squadID == selectedSquadID {
			// Selected squad gets white border
			highlightColor = shr.selectedColor
		} else {
			// Each faction gets its unique color
			highlightColor = shr.GetFactionColor(info.FactionID)
		}

		// Draw border
		vr.DrawTileBorder(screen, *info.Position, highlightColor, shr.borderThickness)
	}
}
