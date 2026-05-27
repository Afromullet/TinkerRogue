package worldmapcore

import (
	"game_main/core/coords"
)

// POIData describes a placed point of interest with type and terrain info
type POIData struct {
	Position coords.LogicalPosition
	NodeID   string // Maps to nodeDefinitions.json ID: POITown, POITemple, POIGuildHall, POIWatchtower
	Biome    Biome
}

// FactionStartPosition describes a generator-chosen starting position for a faction
type FactionStartPosition struct {
	Position coords.LogicalPosition
	Biome    Biome
	Sector   int // Which map sector (0-4)
}

// GenerationResult contains the output of a map generation algorithm
type GenerationResult struct {
	Tiles                 []*Tile
	Rooms                 []Rect
	ValidPositions        []coords.LogicalPosition
	POIs                  []POIData              // Typed points of interest
	FactionStartPositions []FactionStartPosition // Generator-chosen faction positions
	BiomeMap              []Biome                // Flat array indexed by PositionToIndex
}

// GenContext carries the dimensions a generator needs without forcing it to
// read package globals. Callers (typically gamesetup, which has legitimate
// access to coords.ScreenInfo) build a context and pass it to NewGameMap;
// headless tests and save-file migration construct their own.
type GenContext struct {
	Width    int
	Height   int
	TileSize int
}

// MapGenerator defines the interface for all map generation algorithms
type MapGenerator interface {
	// Generate creates a new map layout sized and pixel-scaled per ctx.
	Generate(ctx GenContext, images TileImageSet) GenerationResult

	// Name returns the algorithm name for selection
	Name() string

	// Description returns a human-readable description
	Description() string
}
