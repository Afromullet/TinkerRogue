package main

import (
	"github.com/bytearena/ecs"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/colorm"
)

// Used to apply rgba color transformations to a tile
// ApplyMatrix tells us whether we want to apply the transformation
type ColorMatrix struct {
	r           float32
	g           float32
	b           float32
	a           float32
	ApplyMatrix bool
}

func NewEmptyMatrix() ColorMatrix {
	return ColorMatrix{
		r:           0,
		g:           0,
		b:           0,
		a:           0,
		ApplyMatrix: true,
	}
}

func (c ColorMatrix) IsEmpty() bool {
	if c.r == 0 && c.g == 0 && c.b == 0 && c.a == 0 {
		return true
	}

	return false
}

type TileType int

const (
	WALL TileType = iota
	FLOOR
)

// Holds any entities that are on a tile, whether it's items, creatures, etc.
// Currently only used for holding items.
type TileContents struct {
	entities *[]ecs.Entity
}

type Tile struct {
	PixelX        int
	PixelY        int
	Blocked       bool
	Image         *ebiten.Image
	tileContents  TileContents
	TileType      TileType
	IsRevealed    bool
	colorMatrix   ColorMatrix
	OriginalImage *ebiten.Image
}

func NewTile(pixelX, pixelY int, blocked bool, img *ebiten.Image, tileType TileType, isRevealed bool) Tile {

	var cm colorm.ColorM
	cm.Reset()

	tile := Tile{
		PixelX:     pixelX,
		PixelY:     pixelY,
		Blocked:    true,
		Image:      img,
		TileType:   WALL,
		IsRevealed: false,
		colorMatrix: ColorMatrix{
			r:           0,
			g:           0,
			b:           0,
			a:           0,
			ApplyMatrix: false,
		},
		OriginalImage: img}

	return tile
}

func (t *Tile) SetColorMatrix(c ColorMatrix) {

	t.colorMatrix = c
}
