package rendering

import (
	"game_main/world/coords"
	"game_main/world/worldmap"
	"github.com/hajimehoshi/ebiten/v2"
)

// DrawMap renders the entire game map to the screen
func DrawMap(screen *ebiten.Image, gameMap *worldmap.GameMap, revealAll bool) {
	renderer := NewTileRenderer(gameMap.Tiles)
	renderer.Render(RenderOptions{
		RevealAll: revealAll,
		CenterOn:  nil, // Full map
		Screen:    screen,
	})
}

// DrawMapCentered renders map centered on a position with viewport
// Returns bounds information for UI layout (edges for positioning)
func DrawMapCentered(screen *ebiten.Image, gameMap *worldmap.GameMap,
	centerPos *coords.LogicalPosition, viewportSize int,
	revealAll bool) RenderedBounds {
	renderer := NewTileRenderer(gameMap.Tiles)
	return renderer.Render(RenderOptions{
		RevealAll:    revealAll,
		CenterOn:     centerPos,
		ViewportSize: viewportSize,
		Screen:       screen,
	})
}
