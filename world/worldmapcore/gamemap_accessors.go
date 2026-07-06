package worldmapcore

import (
	"game_main/core/coords"
)

// NewGameMapFromParts assembles a GameMap from generator output at the given
// dimensions. Used by NewGameMap, the save-system load path, and tests; it
// performs no image loading or stair placement.
func NewGameMapFromParts(width, height int, result GenerationResult) GameMap {
	return GameMap{
		Tiles:                 result.Tiles,
		Rooms:                 result.Rooms,
		NumTiles:              len(result.Tiles),
		Width:                 width,
		Height:                height,
		ValidPositions:        result.ValidPositions,
		BiomeMap:              result.BiomeMap,
		POIs:                  result.POIs,
		FactionStartPositions: result.FactionStartPositions,
	}
}

// TileAt returns the tile at the given logical position, or nil if the
// position is out of bounds.
func (gm *GameMap) TileAt(pos coords.LogicalPosition) *Tile {
	return gm.TileAtIndex(coords.CoordManager.LogicalToIndex(pos))
}

// TileAtIndex returns the tile at the given flat index, or nil if the index
// is out of range.
func (gm *GameMap) TileAtIndex(i int) *Tile {
	if i < 0 || i >= len(gm.Tiles) {
		return nil
	}
	return gm.Tiles[i]
}

// TileCount returns the number of tiles in the map.
func (gm *GameMap) TileCount() int {
	return len(gm.Tiles)
}

// MarkTileColorsDirty flags the tile colors as needing a re-render.
func (gm *GameMap) MarkTileColorsDirty() {
	gm.TileColorsDirty = true
}

// ConsumeTileColorsDirty returns whether tile colors changed since the last
// render and clears the flag. Intended for the renderer's read-then-clear
// protocol; there is deliberately no non-consuming getter.
func (gm *GameMap) ConsumeTileColorsDirty() bool {
	dirty := gm.TileColorsDirty
	gm.TileColorsDirty = false
	return dirty
}
