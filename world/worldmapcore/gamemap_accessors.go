package worldmapcore

import (
	"game_main/core/coords"
)

// NewGameMapFromParts assembles a GameMap from generator output at the given
// dimensions. Used by NewGameMap, the save-system load path, and tests; it
// performs no image loading or stair placement.
func NewGameMapFromParts(width, height int, result GenerationResult) GameMap {
	return GameMap{
		tiles:                 result.Tiles,
		rooms:                 result.Rooms,
		width:                 width,
		height:                height,
		validPositions:        result.ValidPositions,
		biomeMap:              result.BiomeMap,
		pois:                  result.POIs,
		factionStartPositions: result.FactionStartPositions,
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
	if i < 0 || i >= len(gm.tiles) {
		return nil
	}
	return gm.tiles[i]
}

// Tiles returns the map's tile slice. Read-only by convention: callers may
// mutate individual tiles (Blocked, IsRevealed) but must not grow, shrink, or
// replace the slice. The returned slice is identity-stable so renderers can
// detect map replacement by comparing backing arrays.
func (gm *GameMap) Tiles() []*Tile {
	return gm.tiles
}

// TileCount returns the number of tiles in the map.
func (gm *GameMap) TileCount() int {
	return len(gm.tiles)
}

// Width returns the logical dungeon width in tiles.
func (gm *GameMap) Width() int {
	return gm.width
}

// Height returns the logical dungeon height in tiles.
func (gm *GameMap) Height() int {
	return gm.height
}

// Rooms returns the generated rooms. Read-only by convention.
func (gm *GameMap) Rooms() []Rect {
	return gm.rooms
}

// ValidPositions returns the walkable positions recorded during generation.
// Read-only by convention.
func (gm *GameMap) ValidPositions() []coords.LogicalPosition {
	return gm.validPositions
}

// POIs returns the points of interest placed during generation. Read-only by
// convention.
func (gm *GameMap) POIs() []POIData {
	return gm.pois
}

// FactionStartPositions returns faction starting positions chosen during
// generation. Read-only by convention.
func (gm *GameMap) FactionStartPositions() []FactionStartPosition {
	return gm.factionStartPositions
}

// MarkTileColorsDirty flags the tile colors as needing a re-render.
func (gm *GameMap) MarkTileColorsDirty() {
	gm.tileColorsDirty = true
}

// ConsumeTileColorsDirty returns whether tile colors changed since the last
// render and clears the flag. Intended for the renderer's read-then-clear
// protocol; there is deliberately no non-consuming getter.
func (gm *GameMap) ConsumeTileColorsDirty() bool {
	dirty := gm.tileColorsDirty
	gm.tileColorsDirty = false
	return dirty
}
