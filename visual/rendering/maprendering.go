package rendering

import (
	"game_main/world/coords"
	"game_main/world/worldmap"

	"github.com/hajimehoshi/ebiten/v2"
)

// tilesChanged detects when the map's tile slice has been replaced (e.g., new level).
func tilesChanged(tiles []*worldmap.Tile) bool {
	if cachedTiles == nil {
		cachedTiles = tiles
		return false
	}
	if len(tiles) != len(cachedTiles) || (len(tiles) > 0 && &tiles[0] != &cachedTiles[0]) {
		cachedTiles = tiles
		return true
	}
	return false
}

var (
	cachedFullRenderer     *TileRenderer
	cachedViewportRenderer *TileRenderer
	cachedTiles            []*worldmap.Tile // Track tile slice to detect new maps
)

// ResetMapRenderers clears cached renderers so they are rebuilt on next draw.
// Call this when the map changes (e.g., new level).
func ResetMapRenderers() {
	cachedFullRenderer = nil
	cachedViewportRenderer = nil
	cachedTiles = nil
}

// DrawMap renders the entire game map to the screen
func DrawMap(screen *ebiten.Image, gameMap *worldmap.GameMap, revealAll bool) {
	if tilesChanged(gameMap.Tiles) {
		ResetMapRenderers()
	}
	if cachedFullRenderer == nil {
		cachedFullRenderer = NewTileRenderer(gameMap.Tiles)
	}
	cachedFullRenderer.Render(RenderOptions{
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
	if tilesChanged(gameMap.Tiles) {
		ResetMapRenderers()
	}
	if cachedViewportRenderer == nil {
		cachedViewportRenderer = NewTileRenderer(gameMap.Tiles)
	}
	return cachedViewportRenderer.Render(RenderOptions{
		RevealAll:    revealAll,
		CenterOn:     centerPos,
		ViewportSize: viewportSize,
		Screen:       screen,
	})
}
