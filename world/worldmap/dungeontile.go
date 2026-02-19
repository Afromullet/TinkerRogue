package worldmap

import (
	"game_main/visual/graphics"
	"game_main/world/coords"

	"github.com/bytearena/ecs"
	"github.com/hajimehoshi/ebiten/v2"
)

type TileType int

const (
	WALL TileType = iota
	FLOOR
	STAIRS_DOWN
	//STAIRS_UP
)

// Holds any entities that are on a tile, whether it's items, creatures, etc.
// Currently only used for holding items.
// Fixed ECS violation: Uses EntityIDs instead of entity pointers
type TileContents struct {
	EntityIDs []ecs.EntityID
}

// TileCords keeps track of the x,y coordinates in the TileMap
type Tile struct {
	PixelX       int
	PixelY       int
	TileCords    coords.LogicalPosition
	Blocked      bool
	Image        *ebiten.Image // Exported for rendering package
	tileContents TileContents
	TileType     TileType
	IsRevealed   bool
	cm           graphics.ColorMatrix
	Biome        Biome  // Terrain biome (default BiomeGrassland)
	POIType      string // POI type at this tile (POITown, POITemple, etc.), empty if not a POI
}

func NewTile(pixelX, pixelY int, tileCords coords.LogicalPosition, blocked bool, img *ebiten.Image, tileType TileType, isRevealed bool) Tile {

	tile := Tile{
		PixelX:     pixelX,
		PixelY:     pixelY,
		TileCords:  tileCords,
		Blocked:    blocked,
		Image:      img,
		TileType:   tileType,
		IsRevealed: isRevealed,
		cm: graphics.ColorMatrix{
			R:           0,
			G:           0,
			B:           0,
			A:           0,
			ApplyMatrix: false,
		}}

	return tile
}

func (t *Tile) SetColorMatrix(c graphics.ColorMatrix) {

	t.cm = c
}

func (t *Tile) GetColorMatrix() graphics.ColorMatrix {
	return t.cm
}
