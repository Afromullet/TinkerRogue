// Package worldmap handles game world generation, map management, and spatial operations.
// It provides dungeon generation algorithms, room management, tile systems,
// and map-based entity placement and retrieval.
package worldmapcore

import (
	"game_main/core/common"
	"game_main/core/coords"
	"game_main/visual/graphics"
)

// Rect represents a half-open rectangular region [X1, X2) × [Y1, Y2).
// NewRect(x, y, w, h) sets X2 = x+w, Y2 = y+h.
// CarveRoom carves the open interior (X1+1 ≤ x < X2, Y1+1 ≤ y < Y2).
// NOTE: Intersect currently uses inclusive bounds (<=, >=) — two rooms
// sharing an edge will report as intersecting even though their carved
// interiors do not overlap. See WORLD_TECH_DEBT 1.10. Changing Intersect to
// half-open is deferred until callers are audited.
type Rect struct {
	X1 int
	X2 int
	Y1 int
	Y2 int
}

func (r *Rect) Center() (int, int) {
	centerX := (r.X1 + r.X2) / 2
	centerY := (r.Y1 + r.Y2) / 2
	return centerX, centerY
}

func NewRect(x int, y int, width int, height int) Rect {
	return Rect{
		X1: x,
		Y1: y,
		X2: x + width,
		Y2: y + height,
	}
}

func (r *Rect) Intersect(other Rect) bool {
	return (r.X1 <= other.X2 && r.X2 >= other.X1 && r.Y1 <= other.Y2 && r.Y2 >= other.Y1)
}

// Holds the Map Information
type GameMap struct {
	Tiles                 []*Tile
	Rooms                 []Rect
	NumTiles              int
	Width                 int // Logical dungeon width (tiles)
	Height                int // Logical dungeon height (tiles)
	RightEdgeX            int
	TopEdgeY              int
	ValidPositions        []coords.LogicalPosition
	BiomeMap              []Biome
	POIs                  []POIData
	FactionStartPositions []FactionStartPosition
	TileColorsDirty       bool
}

// NewGameMap creates a new game map using the provided generator at the given
// dimensions. Callers (typically gamesetup or GUI bootstrap) build the
// GenContext from coords.ScreenInfo; this function does not touch globals so
// it can be driven at any size (headless tests, save-file migration, etc.).
func NewGameMap(gen MapGenerator, ctx GenContext) GameMap {
	images := LoadTileImages()

	result := gen.Generate(ctx, images)

	dungeonMap := GameMap{
		Tiles:                 result.Tiles,
		Rooms:                 result.Rooms,
		NumTiles:              len(result.Tiles),
		Width:                 ctx.Width,
		Height:                ctx.Height,
		ValidPositions:        result.ValidPositions,
		BiomeMap:              result.BiomeMap,
		POIs:                  result.POIs,
		FactionStartPositions: result.FactionStartPositions,
	}

	dungeonMap.PlaceStairs(images)

	return dungeonMap
}

func (gameMap *GameMap) Tile(pos *coords.LogicalPosition) *Tile {

	logicalPos := coords.LogicalPosition{X: pos.X, Y: pos.Y}
	index := coords.CoordManager.LogicalToIndex(logicalPos)
	return gameMap.Tiles[index]

}

func (gameMap *GameMap) StartingPosition() coords.LogicalPosition {
	// For room-based generators
	if len(gameMap.Rooms) > 0 {
		x, y := gameMap.Rooms[0].Center()
		return coords.LogicalPosition{X: x, Y: y}
	}

	// Fallback for non-room generators: use center of map
	centerX := gameMap.Width / 2
	centerY := gameMap.Height / 2

	// If center is not walkable, find first valid position
	logicalPos := coords.LogicalPosition{X: centerX, Y: centerY}
	idx := coords.CoordManager.LogicalToIndex(logicalPos)
	if idx < len(gameMap.Tiles) && !gameMap.Tiles[idx].Blocked {
		return logicalPos
	}

	// Last resort: return first valid position from ValidPositions
	if len(gameMap.ValidPositions) > 0 {
		return gameMap.ValidPositions[0]
	}

	// Shouldn't reach here, but return center as final fallback
	return coords.LogicalPosition{X: centerX, Y: centerY}
}

// Old generation methods removed - now handled by generator implementations
// See gen_rooms_corridors.go for the extracted algorithm

// Place the stairs in the center of a random room.
// The center of the room SHOULD not be blocked.
// Even if it is, that's not something to worry about now, since this is a short term approach
func (gm *GameMap) PlaceStairs(images TileImageSet) {

	var x, y int

	// For room-based generators with multiple rooms
	if len(gm.Rooms) >= 2 {
		//Starts at 1 so we don't create stairs in the starting room
		randRoom := common.GetRandomBetween(1, len(gm.Rooms)-1)
		x, y = gm.Rooms[randRoom].Center()
	} else if len(gm.ValidPositions) > 0 {
		// Fallback for non-room generators: place stairs at random valid position
		randIndex := common.GetRandomBetween(0, len(gm.ValidPositions)-1)
		pos := gm.ValidPositions[randIndex]
		x, y = pos.X, pos.Y
	} else {
		// No valid positions available - shouldn't happen, but return safely
		return
	}

	logicalPos := coords.LogicalPosition{X: x, Y: y}
	ind := coords.CoordManager.LogicalToIndex(logicalPos)

	gm.Tiles[ind].TileType = STAIRS_DOWN
	gm.Tiles[ind].Image = images.StairsDown

}

// Applies the scaling ColorMatrix to the tiles at the Indices
func (gameMap *GameMap) ApplyColorMatrix(indices []int, m graphics.ColorMatrix) {

	for _, ind := range indices {

		if ind < len(gameMap.Tiles) {
			gameMap.Tiles[ind].SetColorMatrix(m)
		}
	}
	gameMap.TileColorsDirty = true
}

// Applies the scaling ColorMatrix to the tiles at the Indices
func (gameMap *GameMap) ApplyColorMatrixToIndex(index int, m graphics.ColorMatrix) {

	if index < gameMap.NumTiles {
		gameMap.Tiles[index].SetColorMatrix(m)
	}
	gameMap.TileColorsDirty = true
}

func (gameMap *GameMap) InBounds(x, y int) bool {
	if x < 0 || x >= gameMap.Width || y < 0 || y >= gameMap.Height {
		return false
	}
	return true
}

// GetBiomeAt returns the biome at the given position, defaulting to BiomeGrassland
func (gm *GameMap) GetBiomeAt(pos coords.LogicalPosition) Biome {
	if gm.BiomeMap == nil {
		return BiomeGrassland
	}
	idx := coords.CoordManager.LogicalToIndex(pos)
	if idx < 0 || idx >= len(gm.BiomeMap) {
		return BiomeGrassland
	}
	return gm.BiomeMap[idx]
}

func (gameMap *GameMap) IsOpaque(x, y int) bool {
	logicalPos := coords.LogicalPosition{X: x, Y: y}
	idx := coords.CoordManager.LogicalToIndex(logicalPos)
	if idx < 0 || idx >= len(gameMap.Tiles) {
		return true // Out-of-bounds treated as opaque (blocks vision)
	}
	return gameMap.Tiles[idx].TileType == WALL
}
