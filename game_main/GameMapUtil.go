package main

import (
	"fmt"
	"os"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

var floorImgs = make([]*ebiten.Image, 0)
var wallImgs = make([]*ebiten.Image, 0)
var decorationImgs = make([]*ebiten.Image, 0)

// Contains the data we need to render the map
type ScreenData struct {
	ScreenWidth  int
	ScreenHeight int
	TileWidth    int
	TileHeight   int
	UIHeight     int
}

func NewScreenData() ScreenData {
	g := ScreenData{
		ScreenWidth:  80,
		ScreenHeight: 50,
		TileWidth:    32,
		TileHeight:   32,
		UIHeight:     10,
	}

	levelHeight = g.ScreenHeight - g.UIHeight

	//Todo refactor this. Only doing this here at the moment due to hwo we use NewScreenData in the code whenever
	//We want to access the parameters. Done that way because it was originally intended to eb stateless.
	//Probably do not need it to be stateless.

	return g
}

// IndexFromXY gets the index of the map array from a given X,Y TILE coordinate.
// This coordinate is logical tiles, not pixels.
func IndexFromXY(x int, y int) int {
	gd := NewScreenData()
	return (y * gd.ScreenWidth) + x
}

func PositionFromIndex(i int) Position {
	gd := NewScreenData()
	return Position{
		X: i % gd.ScreenWidth,
		Y: i / gd.ScreenWidth,
	}

}

func XYFromIndex(i int) (int, int) {
	gd := NewScreenData()
	return i % gd.ScreenWidth, i / gd.ScreenWidth
}

func PixelsFromPosition(pos *Position) (int, int) {
	gd := NewScreenData()
	return pos.X * gd.TileWidth, pos.Y * gd.TileHeight
}

func PixelsFromIndex(i int) (int, int) {
	gd := NewScreenData()
	x, y := XYFromIndex(i)
	return x * gd.TileWidth, y * gd.TileHeight
}

// Return the Grid X,Y coordinates from pixel positions
func GridXYFromPixels(x, y int) (int, int) {
	gd := NewScreenData()
	return x / gd.TileWidth, y / gd.TileHeight

}

func GridPositionFromPixels(x, y int) Position {
	gd := NewScreenData()
	return Position{
		X: x / gd.TileWidth,
		Y: y / gd.TileHeight,
	}
}

func loadTileImages() {

	// Not doing any error checking here. Just want something quick to test so that the map looks more varied

	dir := "assets//tiles/floors/limestone"
	files, _ := os.ReadDir(dir)

	for _, file := range files {
		if !file.IsDir() { // Ensure it's a file, not a subdirectory
			fmt.Println(file.Name())

			floor, _, _ := ebitenutil.NewImageFromFile(dir + "/" + file.Name())

			floorImgs = append(floorImgs, floor)
		}

	}

	dir = "assets//tiles/walls/marble"
	files, _ = os.ReadDir(dir)

	for _, file := range files {
		if !file.IsDir() { // Ensure it's a file, not a subdirectory
			fmt.Println(file.Name())

			wall, _, _ := ebitenutil.NewImageFromFile(dir + "/" + file.Name())

			wallImgs = append(wallImgs, wall)
		}

	}

	dir = "assets//tiles/walls/marble"
	files, _ = os.ReadDir(dir)

	for _, file := range files {
		if !file.IsDir() { // Ensure it's a file, not a subdirectory
			fmt.Println(file.Name())

			wall, _, _ := ebitenutil.NewImageFromFile(dir + "/" + file.Name())

			wallImgs = append(wallImgs, wall)
		}

	}

}
