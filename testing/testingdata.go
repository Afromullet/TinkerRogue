package testing

import (
	"game_main/common"
	"game_main/gear"
	"game_main/visual/graphics"
	"game_main/world/coords"
	"game_main/world/worldmap"
	"strconv"

	"github.com/bytearena/ecs"
)

var TestSquare = graphics.NewSquare(0, 0, graphics.MediumShape)
var TestLine = graphics.NewLine(0, 0, graphics.LinedDiagonalDownLeft, graphics.MediumShape)
var TestCone = graphics.NewCone(0, 0, graphics.LineDiagonalUpRight, graphics.MediumShape)
var TestCircle = graphics.NewCircle(0, 0, graphics.MediumShape)
var TestRect = graphics.NewRectangle(0, 0, graphics.MediumShape)
var TestBurning = gear.NewBurning(5, 2)
var TestSticky = gear.NewSticky(5, 2)
var TestFreezing = gear.NewFreezing(3, 5)

var TestFireEffect = graphics.NewFireEffect(0, 0, 1, 2, 1, 0.5)
var TestCloudEffect = graphics.NewCloudEffect(0, 0, 2)
var TestIceEffect = graphics.NewIceEffect(0, 0, 2)
var TestElectricEffect = graphics.NewElectricityEffect(0, 0, 2)
var TestStickyEffect = graphics.NewStickyGroundEffect(0, 0, 2)

func CreateTestThrowable(shape graphics.TileBasedShape, vx graphics.VisualEffect) *gear.ThrowableAction {

	t := gear.NewThrowableAction(1, 2, 3, shape)
	t.VX = vx
	return t
}

func CreateTestItems(manager *ecs.Manager, tags map[string]ecs.Tag, gameMap *worldmap.GameMap) {

	itemImageLoc := "../assets/items/sword.png"

	//todo add testing location back

	startingPos := gameMap.StartingPosition()

	throwItem := CreateTestThrowable(TestSquare, TestFireEffect)

	gear.CreateItemWithActions(manager, "SquareThrow"+strconv.Itoa(1), coords.LogicalPosition{X: startingPos.X, Y: startingPos.Y}, itemImageLoc,
		[]gear.ItemAction{throwItem}, TestFreezing)

	gear.CreateItemWithActions(manager, "SquareThrow"+strconv.Itoa(1), coords.LogicalPosition{X: startingPos.X, Y: startingPos.Y}, itemImageLoc,
		[]gear.ItemAction{throwItem}, TestSticky)

	gear.CreateItemWithActions(manager, "SquareThrow"+strconv.Itoa(1), coords.LogicalPosition{X: startingPos.X, Y: startingPos.Y}, itemImageLoc,
		[]gear.ItemAction{throwItem}, TestFreezing, TestFreezing)

	throwItem = CreateTestThrowable(TestCircle, TestIceEffect)

	gear.CreateItemWithActions(manager, "CircleThrow"+strconv.Itoa(1), coords.LogicalPosition{X: startingPos.X, Y: startingPos.Y}, itemImageLoc,
		[]gear.ItemAction{throwItem}, TestBurning, TestFreezing)

	throwItem = CreateTestThrowable(TestLine, TestFireEffect)

	gear.CreateItemWithActions(manager, "LineThrow"+strconv.Itoa(1), coords.LogicalPosition{X: startingPos.X, Y: startingPos.Y}, itemImageLoc,
		[]gear.ItemAction{throwItem}, TestFreezing, TestFreezing)

	throwItem = CreateTestThrowable(TestRect, TestElectricEffect)

	gear.CreateItemWithActions(manager, "RectThrow"+strconv.Itoa(1), coords.LogicalPosition{X: startingPos.X, Y: startingPos.Y}, itemImageLoc,
		[]gear.ItemAction{throwItem}, TestSticky, TestFreezing)

	throwItem = CreateTestThrowable(TestCone, TestStickyEffect)

	gear.CreateItemWithActions(manager, "ConeThrow"+strconv.Itoa(1), coords.LogicalPosition{X: startingPos.X, Y: startingPos.Y}, itemImageLoc,
		[]gear.ItemAction{throwItem}, TestBurning, TestFreezing)

	//CreateItem(manager, "Item"+strconv.Itoa(2), coords.LogicalPosition{X: startingPos.X, Y: startingPos.Y}, itemImageLoc, NewBurning(1, 1), NewFreezing(1, 2))

}

func UpdateContentsForTest(ecsmanager *common.EntityManager, gm *worldmap.GameMap) {

	for _, item := range ecsmanager.World.Query(gear.ItemsTag) {

		item_pos := common.GetComponentType[*coords.LogicalPosition](item.Entity, common.PositionComponent)

		gm.AddEntityToTile(item.Entity, item_pos)

		// Register item with PositionSystem for proper tracking
		if common.GlobalPositionSystem != nil {
			common.GlobalPositionSystem.AddEntity(item.Entity.GetID(), *item_pos)
		}

	}

}

// REMOVED: CreateWeapon, CreateArmor, CreatedRangedWeapon - weapon/armor system replaced by squad system
// See CLAUDE.md Section 7 (Squad System Infrastructure) for replacement system

// This function is no longer needed since we removed the action queue system
func InitTestActionManager(ecsmanager *common.EntityManager, pl *common.PlayerData) {
	// No action queue initialization needed anymore
}
