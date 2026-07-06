package worldmapcore

import (
	"game_main/core/coords"
	"testing"
)

// withTestCoords resizes the global coords.CoordManager and coords.ScreenInfo
// to w×h tiles (tile size 16) and restores the originals via t.Cleanup.
//
// Tests in this package MUST NOT use t.Parallel: all index math routes through
// the shared global CoordManager, so parallel tests with different map sizes
// would corrupt each other's indexing.
func withTestCoords(t *testing.T, w, h int) {
	t.Helper()
	origCM := coords.CoordManager
	origSI := coords.ScreenInfo
	sd := coords.ScreenData{
		DungeonWidth:  w,
		DungeonHeight: h,
		TileSize:      16,
		ScaleFactor:   1,
	}
	coords.ScreenInfo = sd
	coords.CoordManager = coords.NewCoordinateManager(sd)
	t.Cleanup(func() {
		coords.CoordManager = origCM
		coords.ScreenInfo = origSI
	})
}

// newTestMap builds a w×h all-wall GameMap headlessly (nil tile images),
// mirroring NewGameMap's assembly without touching LoadTileImages, which
// requires an Ebiten graphics context and assets on disk.
func newTestMap(t *testing.T, w, h int) *GameMap {
	t.Helper()
	withTestCoords(t, w, h)

	numTiles := w * h
	tileValues := make([]Tile, numTiles)
	tiles := make([]*Tile, numTiles)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			logicalPos := coords.LogicalPosition{X: x, Y: y}
			idx := coords.CoordManager.LogicalToIndex(logicalPos)
			tileValues[idx] = NewTile(x*16, y*16, logicalPos, true, nil, WALL, false)
			tiles[idx] = &tileValues[idx]
		}
	}

	return &GameMap{
		Tiles:    tiles,
		NumTiles: numTiles,
		Width:    w,
		Height:   h,
	}
}

// carveFloor turns the tile at (x, y) into an unblocked floor and records it
// in ValidPositions, mirroring the per-tile effect of worldgen carving.
func carveFloor(gm *GameMap, x, y int) {
	pos := coords.LogicalPosition{X: x, Y: y}
	idx := coords.CoordManager.LogicalToIndex(pos)
	gm.Tiles[idx].Blocked = false
	gm.Tiles[idx].TileType = FLOOR
	gm.ValidPositions = append(gm.ValidPositions, pos)
}

// tileTypeCount counts tiles of the given type across the whole map.
func tileTypeCount(gm *GameMap, tt TileType) int {
	count := 0
	for _, tile := range gm.Tiles {
		if tile.TileType == tt {
			count++
		}
	}
	return count
}
