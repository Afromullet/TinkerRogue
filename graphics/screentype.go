package graphics

// Contains the data we need to render the map. Also used for coordinate conversions
type ScreenData struct {
	ScreenWidth  int
	ScreenHeight int

	TileSize int

	DungeonWidth  int
	DungeonHeight int

	ScaleFactor  int
	LevelWidth   int
	LevelHeight  int
	PaddingRight int //Extra padding added to the right ouside of the map
}

func (s ScreenData) GetCanvasWidth() int {

	return int((float64(s.TileSize * s.DungeonWidth))) + s.PaddingRight

}

func (s ScreenData) GetCanvasHeight() int {

	return int((float64(s.TileSize * s.DungeonHeight)))

}

// Todo, need to do less work inside of here because this is called every time we do coordinate conversions
// All of this needs to only be calculated once. Currently, everything that needs Screen Information calls
// NewScreenData, which calculates things again. Those values don't change, so it only needs to be calcualted once
func NewScreenData() ScreenData {
	g := ScreenData{
		DungeonWidth:  100,
		DungeonHeight: 80,
	}
	tilePixels := 32

	// Use a single scale value for both X and Y
	g.TileSize = tilePixels
	g.ScaleFactor = 3

	// Calculate the level dimensions based on the tile size
	g.LevelHeight = g.DungeonHeight * g.TileSize
	g.LevelWidth = g.DungeonWidth * g.TileSize

	g.PaddingRight = 500

	return g
}
