package chunks

import (
	"encoding/json"
	"fmt"
	"game_main/common"
	"game_main/savesystem"
	"game_main/visual/graphics"
	"game_main/world/coords"
	"game_main/world/worldmap"
)

func init() {
	savesystem.RegisterChunk(&MapChunk{})
}

// MapChunk saves/loads the GameMap: tile data, rooms, valid positions.
// Tile images are NOT saved — they're reconstructed from tile type + biome
// using LoadTileImages() on load.
type MapChunk struct {
	// GameMap is set externally before Save/Load (since GameMap lives on Game, not ECS).
	GameMap *worldmap.GameMap
}

func (c *MapChunk) ChunkID() string  { return "map" }
func (c *MapChunk) ChunkVersion() int { return 1 }

// --- Serialization structs ---

type savedMapChunkData struct {
	Width          int             `json:"width"`
	Height         int             `json:"height"`
	Tiles          []savedTile     `json:"tiles"`
	Rooms          []savedRect     `json:"rooms"`
	ValidPositions []savedPosition `json:"validPositions"`
	POIs           []savedPOIData  `json:"pois,omitempty"`
}

type savedTile struct {
	X          int    `json:"x"`
	Y          int    `json:"y"`
	TileType   int    `json:"type"`
	Blocked    bool   `json:"blocked"`
	Biome      int    `json:"biome"`
	POIType    string `json:"poi,omitempty"`
	IsRevealed bool   `json:"revealed"`
}

type savedRect struct {
	X1 int `json:"x1"`
	X2 int `json:"x2"`
	Y1 int `json:"y1"`
	Y2 int `json:"y2"`
}

type savedPOIData struct {
	X      int    `json:"x"`
	Y      int    `json:"y"`
	NodeID string `json:"nodeID"`
	Biome  int    `json:"biome"`
}

// --- Save ---

func (c *MapChunk) Save(em *common.EntityManager) (json.RawMessage, error) {
	if c.GameMap == nil {
		return nil, fmt.Errorf("MapChunk.GameMap not set")
	}

	gm := c.GameMap
	width := graphics.ScreenInfo.DungeonWidth
	height := graphics.ScreenInfo.DungeonHeight

	chunkData := savedMapChunkData{
		Width:  width,
		Height: height,
	}

	// Save tiles
	for _, tile := range gm.Tiles {
		if tile == nil {
			continue
		}
		st := savedTile{
			X:          tile.TileCords.X,
			Y:          tile.TileCords.Y,
			TileType:   int(tile.TileType),
			Blocked:    tile.Blocked,
			Biome:      int(tile.Biome),
			POIType:    tile.POIType,
			IsRevealed: tile.IsRevealed,
		}
		chunkData.Tiles = append(chunkData.Tiles, st)
	}

	// Save rooms
	for _, room := range gm.Rooms {
		chunkData.Rooms = append(chunkData.Rooms, savedRect{
			X1: room.X1, X2: room.X2, Y1: room.Y1, Y2: room.Y2,
		})
	}

	// Save valid positions
	for _, pos := range gm.ValidPositions {
		chunkData.ValidPositions = append(chunkData.ValidPositions, savedPosition{X: pos.X, Y: pos.Y})
	}

	// Save POIs
	for _, poi := range gm.POIs {
		chunkData.POIs = append(chunkData.POIs, savedPOIData{
			X: poi.Position.X, Y: poi.Position.Y,
			NodeID: poi.NodeID, Biome: int(poi.Biome),
		})
	}

	return json.Marshal(chunkData)
}

// --- Load ---

func (c *MapChunk) Load(em *common.EntityManager, data json.RawMessage, idMap *savesystem.EntityIDMap) error {
	if c.GameMap == nil {
		return fmt.Errorf("MapChunk.GameMap not set")
	}

	var chunkData savedMapChunkData
	if err := json.Unmarshal(data, &chunkData); err != nil {
		return fmt.Errorf("failed to unmarshal map data: %w", err)
	}

	// Validate saved dimensions match CoordManager (prevents index panics)
	if coords.CoordManager != nil {
		cmWidth := coords.CoordManager.GetDungeonWidth()
		cmHeight := coords.CoordManager.GetDungeonHeight()
		if chunkData.Width != cmWidth || chunkData.Height != cmHeight {
			return fmt.Errorf("saved map dimensions (%dx%d) do not match current CoordManager (%dx%d)",
				chunkData.Width, chunkData.Height, cmWidth, cmHeight)
		}
	}

	// Load tile images for reconstruction
	images := worldmap.LoadTileImages()

	width := chunkData.Width
	height := chunkData.Height

	gm := c.GameMap
	numTiles := width * height

	// Allocate tiles contiguously
	tileValues := make([]worldmap.Tile, numTiles)
	gm.Tiles = make([]*worldmap.Tile, numTiles)

	// Initialize all tiles as default walls
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			logicalPos := coords.LogicalPosition{X: x, Y: y}
			idx := coords.CoordManager.LogicalToIndex(logicalPos)
			tileValues[idx] = worldmap.NewTile(
				x*graphics.ScreenInfo.TileSize,
				y*graphics.ScreenInfo.TileSize,
				logicalPos, true, nil, worldmap.WALL, false,
			)
			gm.Tiles[idx] = &tileValues[idx]
		}
	}

	// Apply saved tile data and reconstruct images
	for _, st := range chunkData.Tiles {
		logicalPos := coords.LogicalPosition{X: st.X, Y: st.Y}
		idx := coords.CoordManager.LogicalToIndex(logicalPos)
		if idx < 0 || idx >= numTiles {
			continue
		}

		tile := gm.Tiles[idx]
		tile.TileType = worldmap.TileType(st.TileType)
		tile.Blocked = st.Blocked
		tile.Biome = worldmap.Biome(st.Biome)
		tile.POIType = st.POIType
		tile.IsRevealed = st.IsRevealed

		// Reconstruct image from type + biome (rendering logic lives in worldmap package)
		tile.Image = worldmap.SelectTileImage(images, tile.TileType, tile.Biome, tile.POIType)
	}

	// Restore rooms
	gm.Rooms = make([]worldmap.Rect, len(chunkData.Rooms))
	for i, sr := range chunkData.Rooms {
		gm.Rooms[i] = worldmap.Rect{X1: sr.X1, X2: sr.X2, Y1: sr.Y1, Y2: sr.Y2}
	}

	// Restore valid positions
	gm.ValidPositions = make([]coords.LogicalPosition, len(chunkData.ValidPositions))
	for i, sp := range chunkData.ValidPositions {
		gm.ValidPositions[i] = coords.LogicalPosition{X: sp.X, Y: sp.Y}
	}

	gm.NumTiles = numTiles

	// Restore POIs
	gm.POIs = make([]worldmap.POIData, len(chunkData.POIs))
	for i, sp := range chunkData.POIs {
		gm.POIs[i] = worldmap.POIData{
			Position: coords.LogicalPosition{X: sp.X, Y: sp.Y},
			NodeID:   sp.NodeID,
			Biome:    worldmap.Biome(sp.Biome),
		}
	}

	return nil
}

// --- RemapIDs ---

func (c *MapChunk) RemapIDs(em *common.EntityManager, idMap *savesystem.EntityIDMap) error {
	// Map has no entity ID references
	return nil
}

// Tile image selection logic has been moved to worldmap.SelectTileImage().
