package graphics

var GreenColorMatrix = ColorMatrix{0, 1, 0, 1, true}
var RedColorMatrix = ColorMatrix{1, 0, 0, 1, true}

var ScreenInfo = NewScreenData()
var CoordTransformer = NewCoordTransformer(ScreenInfo.DungeonWidth, ScreenInfo.TileSize)

var ViewableSquareSize = 30
var MAP_SCROLLING_ENABLED = true

// var StatsUIOffset int = 1000 //Offset to where the UI starts
var StatsUIOffset int = 1000 //Offset to where the UI starts
