package main

import (
	"fmt"
	"game_main/graphics"
	"os"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

var floorImgs = make([]*ebiten.Image, 0)
var wallImgs = make([]*ebiten.Image, 0)
var decorationImgs = make([]*ebiten.Image, 0)

func PixelsFromPosition(pos *Position) (int, int) {
	gd := graphics.NewScreenData()
	return pos.X * gd.TileWidth, pos.Y * gd.TileHeight
}

func PositionFromIndex(i int) Position {
	gd := graphics.NewScreenData()
	return Position{
		X: i % gd.ScreenWidth,
		Y: i / gd.ScreenWidth,
	}

}

func GridPositionFromPixels(x, y int) Position {
	gd := graphics.NewScreenData()
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
