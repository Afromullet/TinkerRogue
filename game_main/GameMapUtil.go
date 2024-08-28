package main

import (
	"log"

	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

// GetIndexFromXY gets the index of the map array from a given X,Y TILE coordinate.
// This coordinate is logical tiles, not pixels.
func GetIndexFromXY(x int, y int) int {
	gd := NewScreenData()
	return (y * gd.ScreenWidth) + x
}

func GetIndexFromPixels(pixelX, pixelY int) int {

	gd := NewScreenData()

	gridX := pixelX / gd.TileWidth
	gridY := pixelY / gd.TileHeight

	idx := GetIndexFromXY(gridX, gridY)

	return idx
}

func GetPositionFromPixels(pixelX, pixelY int) Position {

	gd := NewScreenData()

	return Position{
		X: pixelX / gd.TileWidth,
		Y: pixelY / gd.TileHeight,
	}

}

func loadTileImages() {
	if floor != nil && wall != nil {
		return
	}
	var err error

	floor, _, err = ebitenutil.NewImageFromFile("assets//tiles/floor.png")
	if err != nil {
		log.Fatal(err)
	}

	wall, _, err = ebitenutil.NewImageFromFile("assets//tiles/wall.png")
	if err != nil {
		log.Fatal(err)
	}
}
