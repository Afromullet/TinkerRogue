package testing

import (
	"game_main/visual/graphics"
	"game_main/visual/vfx"
	"game_main/world/worldmapcore"
)

var TestSquare = graphics.NewSquare(0, 0, graphics.MediumShape)
var TestLine = graphics.NewLine(0, 0, graphics.LinedDiagonalDownLeft, graphics.MediumShape)
var TestCone = graphics.NewCone(0, 0, graphics.LineDiagonalUpRight, graphics.MediumShape)
var TestCircle = graphics.NewCircle(0, 0, graphics.MediumShape)
var TestRect = graphics.NewRectangle(0, 0, graphics.MediumShape)

var TestFireEffect = vfx.NewFireEffect(0, 0, 2)
var TestCloudEffect = vfx.NewCloudEffect(0, 0, 2)
var TestIceEffect = vfx.NewIceEffect(0, 0, 2)
var TestElectricEffect = vfx.NewElectricityEffect(0, 0, 2)
var TestStickyEffect = vfx.NewStickyGroundEffect(0, 0, 2)

func CreateTestItems(manager *worldmapcore.GameMap) {
	// No throwable items to create - squad system handles combat
}
