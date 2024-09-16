package worldmap

import (
	"fmt"
	"os"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

var floorImgs = make([]*ebiten.Image, 0)
var wallImgs = make([]*ebiten.Image, 0)
var stairs_down *ebiten.Image

func loadTileImages() {

	// Not doing any error checking here. Just want something quick to test so that the map looks more varied

	dir := "../assets//tiles/floors/limestone"
	files, _ := os.ReadDir(dir)

	for _, file := range files {
		if !file.IsDir() { // Ensure it's a file, not a subdirectory
			fmt.Println(file.Name())

			floor, _, _ := ebitenutil.NewImageFromFile(dir + "/" + file.Name())

			floorImgs = append(floorImgs, floor)
		}

	}

	dir = "../assets//tiles/walls/marble"
	files, _ = os.ReadDir(dir)

	for _, file := range files {
		if !file.IsDir() { // Ensure it's a file, not a subdirectory
			fmt.Println(file.Name())

			wall, _, _ := ebitenutil.NewImageFromFile(dir + "/" + file.Name())

			wallImgs = append(wallImgs, wall)
		}

	}

	dir = "../assets//tiles/walls/marble"
	files, _ = os.ReadDir(dir)

	for _, file := range files {
		if !file.IsDir() { // Ensure it's a file, not a subdirectory
			fmt.Println(file.Name())

			wall, _, _ := ebitenutil.NewImageFromFile(dir + "/" + file.Name())

			wallImgs = append(wallImgs, wall)
		}

	}

	fn := "../assets//tiles/stairs1.png"

	s, _, _ := ebitenutil.NewImageFromFile(fn)
	stairs_down = s

}
