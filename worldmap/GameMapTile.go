package worldmap

import (
	"game_main/ecshelper"
	"game_main/graphics"

	"github.com/bytearena/ecs"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/colorm"
)

type TileType int

const (
	WALL TileType = iota
	FLOOR
)

// Holds any entities that are on a tile, whether it's items, creatures, etc.
// Currently only used for holding items.
type TileContents struct {
	entities []*ecs.Entity
}

// TileCords keeps track of the x,y coordinates in the TileMap
type Tile struct {
	PixelX       int
	PixelY       int
	TileCords    ecshelper.Position
	Blocked      bool
	Image        *ebiten.Image
	tileContents TileContents
	TileType     TileType
	IsRevealed   bool
	cm           graphics.ColorMatrix
}

func NewTile(pixelX, pixelY int, tileCords ecshelper.Position, blocked bool, img *ebiten.Image, tileType TileType, isRevealed bool) Tile {

	var cm colorm.ColorM
	cm.Reset()

	tile := Tile{
		PixelX:     pixelX,
		PixelY:     pixelY,
		TileCords:  tileCords,
		Blocked:    true,
		Image:      img,
		TileType:   WALL,
		IsRevealed: false,
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
