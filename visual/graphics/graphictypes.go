package graphics

import (
	"game_main/world/coords"
)

var GreenColorMatrix = ColorMatrix{0, 1, 0, 1, true}
var RedColorMatrix = ColorMatrix{1, 0, 0, 1, true}

// ScreenInfo and CoordManager now come from coords package
var ScreenInfo = coords.NewScreenData()
var CoordManager = coords.CoordManager

// MouseToLogicalPosition converts mouse screen coordinates to logical tile coordinates.
// Automatically handles both scrolling modes via CoordinateManager.
func MouseToLogicalPosition(mouseX, mouseY int, centerPos coords.LogicalPosition) coords.LogicalPosition {
	return coords.CoordManager.ScreenToLogical(mouseX, mouseY, &centerPos)
}
