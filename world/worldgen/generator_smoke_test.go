package worldgen

import (
	"game_main/core/common"
	"game_main/core/coords"
	"game_main/world/worldmapcore"
	"testing"
)

// withTestCoords resizes the global coords.CoordManager and coords.ScreenInfo
// so generator index math matches the test's GenContext, restoring the
// originals via t.Cleanup. Tests in this package MUST NOT use t.Parallel
// (shared global CoordManager).
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

// checkResultInvariants asserts the structural contract that GameMap consumers
// (StartingPosition, InitWalkableGridFromMap) depend on: full tile coverage
// with no nils, and every ValidPosition pointing at an unblocked FLOOR tile.
func checkResultInvariants(t *testing.T, result worldmapcore.GenerationResult, w, h int) {
	t.Helper()

	if len(result.Tiles) != w*h {
		t.Fatalf("len(Tiles) = %d, want %d", len(result.Tiles), w*h)
	}
	for i, tile := range result.Tiles {
		if tile == nil {
			t.Fatalf("Tiles[%d] is nil", i)
		}
	}

	for _, pos := range result.ValidPositions {
		if pos.X < 0 || pos.X >= w || pos.Y < 0 || pos.Y >= h {
			t.Errorf("ValidPosition %+v out of %dx%d bounds", pos, w, h)
			continue
		}
		tile := result.Tiles[coords.CoordManager.LogicalToIndex(pos)]
		if tile.Blocked || tile.TileType != worldmapcore.FLOOR {
			t.Errorf("ValidPosition %+v: Blocked=%v TileType=%v, want unblocked FLOOR",
				pos, tile.Blocked, tile.TileType)
		}
	}
}

func TestRoomsCorridorsGeneratorInvariants(t *testing.T) {
	const w, h = 50, 40
	withTestCoords(t, w, h)
	ctx := worldmapcore.GenContext{Width: w, Height: h, TileSize: 16}

	cfg := DefaultRoomsCorridorsConfig()
	cfg.Seed = 12345
	gen := NewRoomsAndCorridorsGenerator(cfg)

	result := gen.Generate(ctx, worldmapcore.TileImageSet{})

	checkResultInvariants(t, result, w, h)

	if len(result.Rooms) < 1 {
		t.Fatal("expected at least one room")
	}
	if len(result.ValidPositions) == 0 {
		t.Fatal("expected carved valid positions")
	}
	// Placement rejects intersecting rooms, so the result must be pairwise
	// non-intersecting.
	for i := 0; i < len(result.Rooms); i++ {
		for j := i + 1; j < len(result.Rooms); j++ {
			if result.Rooms[i].Intersect(result.Rooms[j]) {
				t.Errorf("rooms %d and %d intersect: %+v vs %+v",
					i, j, result.Rooms[i], result.Rooms[j])
			}
		}
	}
}

func TestRoomsCorridorsGeneratorDeterminism(t *testing.T) {
	const w, h = 50, 40
	withTestCoords(t, w, h)
	ctx := worldmapcore.GenContext{Width: w, Height: h, TileSize: 16}

	cfg := DefaultRoomsCorridorsConfig()
	cfg.Seed = 12345

	first := NewRoomsAndCorridorsGenerator(cfg).Generate(ctx, worldmapcore.TileImageSet{})
	second := NewRoomsAndCorridorsGenerator(cfg).Generate(ctx, worldmapcore.TileImageSet{})

	if len(first.Rooms) != len(second.Rooms) {
		t.Fatalf("room counts differ across seeded runs: %d vs %d", len(first.Rooms), len(second.Rooms))
	}
	for i := range first.Rooms {
		if first.Rooms[i] != second.Rooms[i] {
			t.Errorf("room %d differs across seeded runs: %+v vs %+v", i, first.Rooms[i], second.Rooms[i])
		}
	}
}

func TestCavernGeneratorInvariants(t *testing.T) {
	const w, h = 60, 48
	withTestCoords(t, w, h)
	ctx := worldmapcore.GenContext{Width: w, Height: h, TileSize: 16}

	cfg := DefaultCavernConfig()
	cfg.Seed = 4242 // seeds the noise; the walk/CA use the global RNG below
	common.SetRNGSeed(4242, 4242)

	result := NewCavernGenerator(cfg).Generate(ctx, worldmapcore.TileImageSet{})

	checkResultInvariants(t, result, w, h)

	if len(result.ValidPositions) == 0 {
		t.Fatal("expected walkable cavern positions")
	}
	if len(result.Rooms) == 0 {
		t.Fatal("expected chambers recorded as rooms")
	}
	if len(result.BiomeMap) != w*h {
		t.Fatalf("len(BiomeMap) = %d, want %d", len(result.BiomeMap), w*h)
	}
}

func TestCavernGeneratorDeterminism(t *testing.T) {
	const w, h = 60, 48
	withTestCoords(t, w, h)
	ctx := worldmapcore.GenContext{Width: w, Height: h, TileSize: 16}

	cfg := DefaultCavernConfig()
	cfg.Seed = 4242

	// CavernGenerator seeds only its noise from config; chamber placement and
	// tunnels use the global RNG, so seed it explicitly before each run.
	common.SetRNGSeed(99, 99)
	first := NewCavernGenerator(cfg).Generate(ctx, worldmapcore.TileImageSet{})
	common.SetRNGSeed(99, 99)
	second := NewCavernGenerator(cfg).Generate(ctx, worldmapcore.TileImageSet{})

	if len(first.Rooms) != len(second.Rooms) {
		t.Fatalf("chamber counts differ across seeded runs: %d vs %d", len(first.Rooms), len(second.Rooms))
	}
	for i := range first.Rooms {
		if first.Rooms[i] != second.Rooms[i] {
			t.Errorf("chamber %d differs across seeded runs: %+v vs %+v", i, first.Rooms[i], second.Rooms[i])
		}
	}
}
