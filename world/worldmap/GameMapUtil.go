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
	POIImages   map[string]*ebiten.Image
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
		POIImages:   make(map[string]*ebiten.Image),
	}

	// Load floor tiles
	dir := defaultFloorPath()
	files, _ := os.ReadDir(dir)

	for _, file := range files {
		if !file.IsDir() {
			imagePath := filepath.Join(dir, file.Name())
			floor, _, _ := ebitenutil.NewImageFromFile(imagePath)
			images.FloorImages = append(images.FloorImages, floor)
		}
	}

	// Load wall tiles
	dir = defaultWallPath()
	files, _ = os.ReadDir(dir)

	for _, file := range files {
		if !file.IsDir() {
			imagePath := filepath.Join(dir, file.Name())
			wall, _, _ := ebitenutil.NewImageFromFile(imagePath)
			images.WallImages = append(images.WallImages, wall)
		}
	}

	// Load stairs
	s, _, _ := ebitenutil.NewImageFromFile(stairsPath())
	images.StairsDown = s

	// Load biome-specific images
	for _, biome := range allBiomes {
		images.BiomeImages[biome] = loadBiomeTiles(biome)
	}

	// Load POI-specific images
	for poiType := range poiAssetConfig {
		assetPath := poiAssetPath(poiType)
		if img, _, err := ebitenutil.NewImageFromFile(assetPath); err == nil {
			images.POIImages[poiType] = img
		}
	}

	return images
}

// loadBiomeTiles loads floor and wall images for a specific biome
func loadBiomeTiles(biome Biome) *BiomeTileSet {
	biomeTiles := &BiomeTileSet{
		WallImages:  make([]*ebiten.Image, 0),
		FloorImages: make([]*ebiten.Image, 0),
	}

	// Load floor tiles for this biome
	floorDir := biomeFloorPath(biome)
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
	wallDir := biomeWallPath(biome)
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

	return biomeTiles
}
