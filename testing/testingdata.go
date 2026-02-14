package testing

import (
	"game_main/common"

	"game_main/visual/graphics"
	"game_main/world/worldmap"
)

var TestSquare = graphics.NewSquare(0, 0, graphics.MediumShape)
var TestLine = graphics.NewLine(0, 0, graphics.LinedDiagonalDownLeft, graphics.MediumShape)
var TestCone = graphics.NewCone(0, 0, graphics.LineDiagonalUpRight, graphics.MediumShape)
var TestCircle = graphics.NewCircle(0, 0, graphics.MediumShape)
var TestRect = graphics.NewRectangle(0, 0, graphics.MediumShape)

var TestFireEffect = graphics.NewFireEffect(0, 0, 2)
var TestCloudEffect = graphics.NewCloudEffect(0, 0, 2)
var TestIceEffect = graphics.NewIceEffect(0, 0, 2)
var TestElectricEffect = graphics.NewElectricityEffect(0, 0, 2)
var TestStickyEffect = graphics.NewStickyGroundEffect(0, 0, 2)

func CreateTestItems(manager *worldmap.GameMap) {
	// No throwable items to create - squad system handles combat
}

// REMOVED: CreateWeapon, CreateArmor, CreatedRangedWeapon - weapon/armor system replaced by squad system
// See CLAUDE.md Section 7 (Squad System Infrastructure) for replacement system

// This function is no longer needed since we removed the action queue system
func InitTestActionManager(ecsmanager *common.EntityManager, pl *common.PlayerData) {
	// No action queue initialization needed anymore
}
