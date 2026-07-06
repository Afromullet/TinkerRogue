package worldmapcore

import (
	"game_main/core/common"
	"game_main/core/coords"
	"game_main/visual/graphics"
	"testing"
)

func TestInBounds(t *testing.T) {
	gm := newTestMap(t, 20, 15)

	tests := []struct {
		name string
		x, y int
		want bool
	}{
		{"origin", 0, 0, true},
		{"far corner", 19, 14, true},
		{"negative x", -1, 0, false},
		{"negative y", 0, -1, false},
		{"x == width", 20, 0, false},
		{"y == height", 0, 15, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := gm.InBounds(tt.x, tt.y); got != tt.want {
				t.Errorf("InBounds(%d, %d) = %v, want %v", tt.x, tt.y, got, tt.want)
			}
		})
	}
}

func TestTileIndexing(t *testing.T) {
	gm := newTestMap(t, 20, 15)
	carveFloor(gm, 5, 7)

	pos := coords.LogicalPosition{X: 5, Y: 7}
	tile := gm.Tiles[coords.CoordManager.LogicalToIndex(pos)]

	if tile.TileCords != pos {
		t.Errorf("tile.TileCords = %+v, want %+v", tile.TileCords, pos)
	}
	if tile.PixelX != 5*16 || tile.PixelY != 7*16 {
		t.Errorf("tile pixels = (%d, %d), want (%d, %d)", tile.PixelX, tile.PixelY, 5*16, 7*16)
	}
	if tile.Blocked || tile.TileType != FLOOR {
		t.Errorf("carved tile Blocked=%v TileType=%v, want unblocked FLOOR", tile.Blocked, tile.TileType)
	}
}

func TestStartingPosition_Rooms(t *testing.T) {
	gm := newTestMap(t, 20, 15)
	gm.Rooms = []Rect{NewRect(2, 2, 6, 6)}

	got := gm.StartingPosition()
	want := coords.LogicalPosition{X: 5, Y: 5} // center of NewRect(2,2,6,6)
	if got != want {
		t.Errorf("StartingPosition() = %+v, want first room center %+v", got, want)
	}
}

func TestStartingPosition_CenterFallback(t *testing.T) {
	gm := newTestMap(t, 20, 15)
	carveFloor(gm, 10, 7) // map center (w/2, h/2)

	got := gm.StartingPosition()
	want := coords.LogicalPosition{X: 10, Y: 7}
	if got != want {
		t.Errorf("StartingPosition() = %+v, want walkable center %+v", got, want)
	}
}

func TestStartingPosition_ValidPositionsFallback(t *testing.T) {
	gm := newTestMap(t, 20, 15)
	// Center stays blocked; record a valid position without carving via the
	// helper so the center-walkable branch is skipped.
	gm.ValidPositions = []coords.LogicalPosition{{X: 3, Y: 3}}

	got := gm.StartingPosition()
	want := coords.LogicalPosition{X: 3, Y: 3}
	if got != want {
		t.Errorf("StartingPosition() = %+v, want first valid position %+v", got, want)
	}
}

func TestStartingPosition_FinalFallback(t *testing.T) {
	// All walls, no rooms, no valid positions: returns the (blocked) center.
	// Pins the documented last-resort behavior.
	gm := newTestMap(t, 20, 15)

	got := gm.StartingPosition()
	want := coords.LogicalPosition{X: 10, Y: 7}
	if got != want {
		t.Errorf("StartingPosition() = %+v, want blocked center %+v", got, want)
	}
}

func TestPlaceStairs_RoomBranch(t *testing.T) {
	common.SetRNGSeed(1, 2)
	gm := newTestMap(t, 20, 15)
	gm.Rooms = []Rect{
		NewRect(1, 1, 5, 5), // starting room — stairs must NOT be here
		NewRect(8, 8, 5, 5),
		NewRect(1, 8, 5, 5),
	}

	gm.PlaceStairs(TileImageSet{})

	if got := tileTypeCount(gm, STAIRS_DOWN); got != 1 {
		t.Fatalf("STAIRS_DOWN tile count = %d, want 1", got)
	}

	// The stairs must be at the center of a non-starting room (index >= 1).
	eligible := map[coords.LogicalPosition]bool{}
	for _, room := range gm.Rooms[1:] {
		x, y := room.Center()
		eligible[coords.LogicalPosition{X: x, Y: y}] = true
	}
	found := false
	for _, tile := range gm.Tiles {
		if tile.TileType == STAIRS_DOWN && eligible[tile.TileCords] {
			found = true
		}
	}
	if !found {
		t.Errorf("stairs not placed at any non-starting room center %v", eligible)
	}
}

func TestPlaceStairs_ValidPositionsBranch(t *testing.T) {
	common.SetRNGSeed(3, 4)
	gm := newTestMap(t, 20, 15)
	carveFloor(gm, 2, 2)
	carveFloor(gm, 3, 3)
	carveFloor(gm, 4, 4)

	gm.PlaceStairs(TileImageSet{})

	if got := tileTypeCount(gm, STAIRS_DOWN); got != 1 {
		t.Fatalf("STAIRS_DOWN tile count = %d, want 1", got)
	}
	carved := map[coords.LogicalPosition]bool{
		{X: 2, Y: 2}: true, {X: 3, Y: 3}: true, {X: 4, Y: 4}: true,
	}
	for _, tile := range gm.Tiles {
		if tile.TileType == STAIRS_DOWN && !carved[tile.TileCords] {
			t.Errorf("stairs at %+v, want one of the carved positions", tile.TileCords)
		}
	}
}

func TestPlaceStairs_EmptyNoop(t *testing.T) {
	gm := newTestMap(t, 20, 15)

	gm.PlaceStairs(TileImageSet{}) // must not panic

	if got := tileTypeCount(gm, STAIRS_DOWN); got != 0 {
		t.Errorf("STAIRS_DOWN tile count = %d, want 0 on empty map", got)
	}
}

func TestGetBiomeAt(t *testing.T) {
	gm := newTestMap(t, 20, 15)

	// Nil biome map falls back to grassland.
	if got := gm.GetBiomeAt(coords.LogicalPosition{X: 4, Y: 5}); got != BiomeGrassland {
		t.Errorf("GetBiomeAt with nil BiomeMap = %v, want BiomeGrassland", got)
	}

	gm.BiomeMap = make([]Biome, gm.NumTiles)
	idx := coords.CoordManager.LogicalToIndex(coords.LogicalPosition{X: 4, Y: 5})
	gm.BiomeMap[idx] = BiomeDesert

	if got := gm.GetBiomeAt(coords.LogicalPosition{X: 4, Y: 5}); got != BiomeDesert {
		t.Errorf("GetBiomeAt = %v, want BiomeDesert", got)
	}
	if got := gm.GetBiomeAt(coords.LogicalPosition{X: 25, Y: 20}); got != BiomeGrassland {
		t.Errorf("GetBiomeAt out of range = %v, want BiomeGrassland", got)
	}
	if got := gm.GetBiomeAt(coords.LogicalPosition{X: -1, Y: -1}); got != BiomeGrassland {
		t.Errorf("GetBiomeAt negative = %v, want BiomeGrassland", got)
	}
}

func TestIsOpaque(t *testing.T) {
	gm := newTestMap(t, 20, 15)
	carveFloor(gm, 5, 5)

	if !gm.IsOpaque(0, 0) {
		t.Error("IsOpaque(0,0) = false, want true for wall")
	}
	if gm.IsOpaque(5, 5) {
		t.Error("IsOpaque(5,5) = true, want false for carved floor")
	}
	// Out of bounds is treated as opaque (blocks vision).
	if !gm.IsOpaque(-1, 0) {
		t.Error("IsOpaque(-1,0) = false, want true for out-of-bounds")
	}
	if !gm.IsOpaque(0, 15) {
		t.Error("IsOpaque(0,15) = false, want true for out-of-bounds")
	}
}

func TestApplyColorMatrix_DirtyFlag(t *testing.T) {
	gm := newTestMap(t, 20, 15)
	m := graphics.ColorMatrix{R: 0.5, G: 0.25, B: 0.75, A: 1}

	idx := coords.CoordManager.LogicalToIndex(coords.LogicalPosition{X: 3, Y: 4})
	gm.ApplyColorMatrix([]int{idx}, m)

	if got := gm.Tiles[idx].GetColorMatrix(); got != m {
		t.Errorf("tile color matrix = %+v, want %+v", got, m)
	}
	if !gm.TileColorsDirty {
		t.Error("TileColorsDirty = false after ApplyColorMatrix, want true")
	}

	// Out-of-range index is silently skipped but the flag is still set (pinned).
	gm.TileColorsDirty = false
	gm.ApplyColorMatrix([]int{len(gm.Tiles)}, m)
	if !gm.TileColorsDirty {
		t.Error("TileColorsDirty = false after out-of-range ApplyColorMatrix, want true (pinned)")
	}
}

func TestApplyColorMatrixToIndex(t *testing.T) {
	gm := newTestMap(t, 20, 15)
	m := graphics.ColorMatrix{R: 1, G: 0.5, B: 0.5, A: 1}

	idx := coords.CoordManager.LogicalToIndex(coords.LogicalPosition{X: 7, Y: 2})
	gm.ApplyColorMatrixToIndex(idx, m)

	if got := gm.Tiles[idx].GetColorMatrix(); got != m {
		t.Errorf("tile color matrix = %+v, want %+v", got, m)
	}
	if !gm.TileColorsDirty {
		t.Error("TileColorsDirty = false after ApplyColorMatrixToIndex, want true")
	}

	// index == NumTiles is skipped but the flag is still set (pinned).
	gm.TileColorsDirty = false
	gm.ApplyColorMatrixToIndex(gm.NumTiles, m)
	if !gm.TileColorsDirty {
		t.Error("TileColorsDirty = false after out-of-range index, want true (pinned)")
	}
	// NOTE: negative indices currently panic; a guard is added in the
	// encapsulation phase, which is when the negative-index test lands.
}
