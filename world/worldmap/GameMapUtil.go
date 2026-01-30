package worldmap

import (
	"os"
	"path/filepath"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

// TileImageSet holds all images needed for tile rendering
type TileImageSet struct {
	WallImages  []*ebiten.Image
	FloorImages []*ebiten.Image
	StairsDown  *ebiten.Image
	BiomeImages map[Biome]*BiomeTileSet
}

// BiomeTileSet holds images for a specific biome
type BiomeTileSet struct {
	WallImages  []*ebiten.Image
	FloorImages []*ebiten.Image
}

// LoadTileImages loads all tile images from disk
// Returns a TileImageSet instead of setting global variables
func LoadTileImages() TileImageSet {
	images := TileImageSet{
		WallImages:  make([]*ebiten.Image, 0),
		FloorImages: make([]*ebiten.Image, 0),
		BiomeImages: make(map[Biome]*BiomeTileSet),
	}

	// Load floor tiles
	dir := filepath.Join("..", "assets", "tiles", "floors", "limestone")
	files, _ := os.ReadDir(dir)

	for _, file := range files {
		if !file.IsDir() {
			imagePath := filepath.Join(dir, file.Name())
			floor, _, _ := ebitenutil.NewImageFromFile(imagePath)
			images.FloorImages = append(images.FloorImages, floor)
		}
	}

	// Load wall tiles (from marble directory)
	dir = filepath.Join("..", "assets", "tiles", "walls", "marble")
	files, _ = os.ReadDir(dir)

	for _, file := range files {
		if !file.IsDir() {
			imagePath := filepath.Join(dir, file.Name())
			wall, _, _ := ebitenutil.NewImageFromFile(imagePath)
			images.WallImages = append(images.WallImages, wall)
		}
	}

	// Load stairs
	fn := filepath.Join("..", "assets", "tiles", "stairs1.png")
	s, _, _ := ebitenutil.NewImageFromFile(fn)
	images.StairsDown = s

	// Load biome-specific images
	biomes := []Biome{BiomeGrassland, BiomeForest, BiomeDesert, BiomeMountain, BiomeSwamp}
	for _, biome := range biomes {
		images.BiomeImages[biome] = loadBiomeTiles(biome)
	}

	return images
}

// loadBiomeTiles loads floor and wall images for a specific biome
func loadBiomeTiles(biome Biome) *BiomeTileSet {
	biomeTiles := &BiomeTileSet{
		WallImages:  make([]*ebiten.Image, 0),
		FloorImages: make([]*ebiten.Image, 0),
	}

	biomeName := biome.String()

	// Load floor tiles for this biome
	floorDir := filepath.Join("..", "assets", "tiles", "floors", biomeName)
	if files, err := os.ReadDir(floorDir); err == nil {
		for _, file := range files {
			if !file.IsDir() {
				imagePath := filepath.Join(floorDir, file.Name())
				if img, _, err := ebitenutil.NewImageFromFile(imagePath); err == nil {
					biomeTiles.FloorImages = append(biomeTiles.FloorImages, img)
				}
			}
		}
	}

	// Load wall tiles for this biome
	wallDir := filepath.Join("..", "assets", "tiles", "walls", biomeName)
	if files, err := os.ReadDir(wallDir); err == nil {
		for _, file := range files {
			if !file.IsDir() {
				imagePath := filepath.Join(wallDir, file.Name())
				if img, _, err := ebitenutil.NewImageFromFile(imagePath); err == nil {
					biomeTiles.WallImages = append(biomeTiles.WallImages, img)
				}
			}
		}
	}

	// Fall back to default images if biome-specific images aren't available
	if len(biomeTiles.FloorImages) == 0 {
		biomeTiles.FloorImages = make([]*ebiten.Image, 0)
	}
	if len(biomeTiles.WallImages) == 0 {
		biomeTiles.WallImages = make([]*ebiten.Image, 0)
	}

	return biomeTiles
}
