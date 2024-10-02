package graphics

// Contains the data we need to render the map. Also used for coordinate conversions
type ScreenData struct {
	ScreenWidth  int
	ScreenHeight int

	TileSize int

	DungeonWidth  int
	DungeonHeight int
	ScaleX        float64
	ScaleY        float64
	ScaleFactor   int
	LevelWidth    int
	LevelHeight   int
}

func (s ScreenData) GetCanvasWidth() int {

	return int((float64(s.TileSize * s.DungeonWidth)))

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
	tileWidthPixels := 32
	tileHeightPixels := 32

	// Calculate the scale based on the dungeon dimensions

	g.ScaleX = float64(g.DungeonWidth) * float64(tileWidthPixels) / float64(g.DungeonWidth)
	g.ScaleY = float64(g.DungeonHeight) * float64(tileHeightPixels) / float64(g.DungeonHeight)

	g.TileSize = int(g.ScaleX)

	g.LevelHeight = int(float64(g.DungeonHeight) * g.ScaleY)
	g.LevelWidth = int(float64(g.DungeonWidth) * g.ScaleX)

	g.ScaleFactor = 3

	return g
}
