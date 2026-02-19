// Package worldmap handles game world generation, map management, and spatial operations.
// It provides dungeon generation algorithms, room management, tile systems,
// and map-based entity placement and retrieval.
package worldmap

import (
	"errors"

	"game_main/common"
	"game_main/visual/graphics"
	"game_main/world/coords"

	"github.com/bytearena/ecs"
)

// Rect represents a rectangular room or area on the game map.
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
	RightEdgeX            int
	TopEdgeY              int
	ValidPositions        []coords.LogicalPosition
	BiomeMap              []Biome
	POIs                  []POIData
	FactionStartPositions []FactionStartPosition
	TileColorsDirty       bool
}

// NewGameMap creates a new game map using the specified generator algorithm
func NewGameMap(generatorName string) GameMap {
	images := LoadTileImages()

	dungeonMap := GameMap{}

	// Get generator or fall back to default
	gen := GetGeneratorOrDefault(generatorName)

	// Generate the map
	result := gen.Generate(
		graphics.ScreenInfo.DungeonWidth,
		graphics.ScreenInfo.DungeonHeight,
		images,
	)

	dungeonMap.Tiles = result.Tiles
	dungeonMap.Rooms = result.Rooms
	dungeonMap.NumTiles = len(dungeonMap.Tiles)
	dungeonMap.ValidPositions = result.ValidPositions
	dungeonMap.BiomeMap = result.BiomeMap
	dungeonMap.POIs = result.POIs
	dungeonMap.FactionStartPositions = result.FactionStartPositions

	dungeonMap.PlaceStairs(images)

	return dungeonMap
}

// NewGameMapDefault creates a game map with the default generator
// Provides backward compatibility for existing code
func NewGameMapDefault() GameMap {
	return NewGameMap("rooms_corridors")
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
	centerX := graphics.ScreenInfo.DungeonWidth / 2
	centerY := graphics.ScreenInfo.DungeonHeight / 2

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

// The Entity Manager continues to track an entity when it is added to a tile.
// Since a tile has a position, we use the pos parameter to determine which tile to add it to
func (gameMap *GameMap) AddEntityToTile(entity *ecs.Entity, pos *coords.LogicalPosition) {

	tile := gameMap.Tile(pos)

	if tile.tileContents.EntityIDs == nil {
		tile.tileContents.EntityIDs = make([]ecs.EntityID, 0)
	}

	tile.tileContents.EntityIDs = append(tile.tileContents.EntityIDs, entity.ID)
}

// This removes the item at the specified index from the tile.
// Right now it's just used for the inventory
// The item is removed from the tile but still exists in the entity manager.
// Since this removes the item from tile.tileContents, the caller will have to store it somewhere
// Otherwise, it'll only exist in the entity manager
// Returns EntityID instead of entity pointer (ECS compliance)
func (gameMap *GameMap) RemoveItemFromTile(index int, pos *coords.LogicalPosition) (ecs.EntityID, error) {

	tile := gameMap.Tile(pos)

	if tile.tileContents.EntityIDs == nil {
		return 0, errors.New("entityIDs slice is nil")
	}

	entityIDs := tile.tileContents.EntityIDs

	if index < 0 || index >= len(entityIDs) {
		return 0, errors.New("index out of range")
	}

	entityID := entityIDs[index]

	tile.tileContents.EntityIDs = append(entityIDs[:index], entityIDs[index+1:]...)

	return entityID, nil
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

func (gameMap GameMap) InBounds(x, y int) bool {

	if x < 0 || x >= graphics.ScreenInfo.DungeonWidth || y < 0 || y >= graphics.ScreenInfo.DungeonHeight {
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

// TODO: Change this to check for WALL, not blocked
// Shouldn't this be a pointer?
func (gameMap GameMap) IsOpaque(x, y int) bool {
	logicalPos := coords.LogicalPosition{X: x, Y: y}
	idx := coords.CoordManager.LogicalToIndex(logicalPos)
	return gameMap.Tiles[idx].TileType == WALL
}

