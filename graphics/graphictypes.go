package graphics

var GreenColorMatrix = ColorMatrix{0, 1, 0, 1, true}
var RedColorMatrix = ColorMatrix{1, 0, 0, 1, true}

var ScreenInfo = NewScreenData()
var CoordTransformer = NewCoordTransformer(ScreenInfo.DungeonWidth, ScreenInfo.TileSize)
