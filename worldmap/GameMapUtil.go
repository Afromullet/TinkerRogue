package worldmap

import (
	"os"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

// TileImageSet holds all images needed for tile rendering
type TileImageSet struct {
	WallImages  []*ebiten.Image
	FloorImages []*ebiten.Image
	StairsDown  *ebiten.Image
}

// LoadTileImages loads all tile images from disk
// Returns a TileImageSet instead of setting global variables
func LoadTileImages() TileImageSet {
	images := TileImageSet{
		WallImages:  make([]*ebiten.Image, 0),
		FloorImages: make([]*ebiten.Image, 0),
	}

	// Load floor tiles
	dir := "../assets//tiles/floors/limestone"
	files, _ := os.ReadDir(dir)

	for _, file := range files {
		if !file.IsDir() {
			floor, _, _ := ebitenutil.NewImageFromFile(dir + "/" + file.Name())
			images.FloorImages = append(images.FloorImages, floor)
		}
	}

	// Load wall tiles (from marble directory)
	dir = "../assets//tiles/walls/marble"
	files, _ = os.ReadDir(dir)

	for _, file := range files {
		if !file.IsDir() {
			wall, _, _ := ebitenutil.NewImageFromFile(dir + "/" + file.Name())
			images.WallImages = append(images.WallImages, wall)
		}
	}

	// Note: The original code loaded from marble directory twice, keeping that behavior
	files, _ = os.ReadDir(dir)
	for _, file := range files {
		if !file.IsDir() {
			wall, _, _ := ebitenutil.NewImageFromFile(dir + "/" + file.Name())
			images.WallImages = append(images.WallImages, wall)
		}
	}

	// Load stairs
	fn := "../assets//tiles/stairs1.png"
	s, _, _ := ebitenutil.NewImageFromFile(fn)
	images.StairsDown = s

	return images
}
