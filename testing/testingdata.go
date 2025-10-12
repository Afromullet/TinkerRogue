package testing

import (
	"game_main/avatar"
	"game_main/common"
	"game_main/coords"
	"game_main/entitytemplates"
	"game_main/gear"
	"game_main/graphics"
	"game_main/rendering"
	"game_main/worldmap"
	"log"
	"strconv"

	"github.com/bytearena/ecs"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

var TestSquare = graphics.NewSquare(0, 0, common.NormalQuality)
var TestLine = graphics.NewLine(0, 0, graphics.LinedDiagonalDownLeft, common.NormalQuality)
var TestCone = graphics.NewCone(0, 0, graphics.LineDiagonalUpRight, common.NormalQuality)
var TestCircle = graphics.NewCircle(0, 0, common.NormalQuality)
var TestRect = graphics.NewRectangle(0, 0, common.NormalQuality)
var TestBurning = gear.NewBurning(5, 2)
var TestSticky = gear.NewSticky(5, 2)
var TestFreezing = gear.NewFreezing(3, 5)

var TestFireEffect = graphics.NewFireEffect(0, 0, 1, 2, 1, 0.5)
var TestCloudEffect = graphics.NewCloudEffect(0, 0, 2)
var TestIceEffect = graphics.NewIceEffect(0, 0, 2)
var TestElectricEffect = graphics.NewElectricityEffect(0, 0, 2)
var TestStickyEffect = graphics.NewStickyGroundEffect(0, 0, 2)

func CreateTestConsumables(ecsmanager *common.EntityManager, gm *worldmap.GameMap) {
	ent := entitytemplates.CreateEntityFromTemplate(*ecsmanager, entitytemplates.EntityConfig{
		Type:      entitytemplates.EntityConsumable,
		Name:      entitytemplates.ConsumableTemplates[0].Name,
		ImagePath: entitytemplates.ConsumableTemplates[0].ImgName,
		AssetDir:  "../assets/items/",
		Visible:   false,
		Position:  nil,
	}, entitytemplates.ConsumableTemplates[0])
	pos := common.GetPosition(ent)
	rend := common.GetComponentType[*rendering.Renderable](ent, rendering.RenderableComponent)
	rend.Visible = true
	pos.X = gm.StartingPosition().X + 1
	pos.Y = gm.StartingPosition().Y + 2
	gm.AddEntityToTile(ent, &coords.LogicalPosition{X: pos.X, Y: pos.Y})

	ent = entitytemplates.CreateEntityFromTemplate(*ecsmanager, entitytemplates.EntityConfig{
		Type:      entitytemplates.EntityConsumable,
		Name:      entitytemplates.ConsumableTemplates[1].Name,
		ImagePath: entitytemplates.ConsumableTemplates[1].ImgName,
		AssetDir:  "../assets/items/",
		Visible:   false,
		Position:  nil,
	}, entitytemplates.ConsumableTemplates[1])
	pos = common.GetPosition(ent)
	rend = common.GetComponentType[*rendering.Renderable](ent, rendering.RenderableComponent)
	rend.Visible = true
	pos.X = gm.StartingPosition().X + 1
	pos.Y = gm.StartingPosition().Y + 2

	gm.AddEntityToTile(ent, &coords.LogicalPosition{X: pos.X, Y: pos.Y})

	ent = entitytemplates.CreateEntityFromTemplate(*ecsmanager, entitytemplates.EntityConfig{
		Type:      entitytemplates.EntityConsumable,
		Name:      entitytemplates.ConsumableTemplates[2].Name,
		ImagePath: entitytemplates.ConsumableTemplates[2].ImgName,
		AssetDir:  "../assets/items/",
		Visible:   false,
		Position:  nil,
	}, entitytemplates.ConsumableTemplates[2])
	pos = common.GetPosition(ent)
	rend = common.GetComponentType[*rendering.Renderable](ent, rendering.RenderableComponent)
	rend.Visible = true
	pos.X = gm.StartingPosition().X + 1
	pos.Y = gm.StartingPosition().Y + 2

	gm.AddEntityToTile(ent, &coords.LogicalPosition{X: pos.X, Y: pos.Y})
}

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

	for _, item := range ecsmanager.World.Query(ecsmanager.WorldTags["items"]) {

		item_pos := item.Components[common.PositionComponent].(*coords.LogicalPosition)

		gm.AddEntityToTile(item.Entity, item_pos)

		// Register item with PositionSystem for proper tracking
		if common.GlobalPositionSystem != nil {
			common.GlobalPositionSystem.AddEntity(item.Entity.GetID(), *item_pos)
		}

	}

}

// Create an item with any number of Effects. ItemEffect is a wrapper around an ecs.Component to make
// Manipulating it easier
func CreateItem(manager *ecs.Manager, name string, pos coords.LogicalPosition, imagePath string, effects ...gear.StatusEffects) *ecs.Entity {

	img, _, err := ebitenutil.NewImageFromFile(imagePath)
	if err != nil {
		log.Fatal(err)
	}

	item := &gear.Item{Count: 1, Properties: manager.NewEntity()}

	for _, prop := range effects {
		item.Properties.AddComponent(prop.StatusEffectComponent(), &prop)

	}

	itemEntity := manager.NewEntity().
		AddComponent(rendering.RenderableComponent, &rendering.Renderable{
			Image:   img,
			Visible: true,
		}).
		AddComponent(common.PositionComponent, &coords.LogicalPosition{
			X: pos.X,
			Y: pos.Y,
		}).
		AddComponent(common.NameComponent, &common.Name{
			NameStr: name,
		}).
		AddComponent(gear.ItemComponent, item)

	//TODO where shoudl I add the tags?

	return itemEntity

}

// REMOVED: CreateWeapon, CreateArmor, CreatedRangedWeapon - weapon/armor system replaced by squad system
// See CLAUDE.md Section 7 (Squad System Infrastructure) for replacement system

// This function is no longer needed since we removed the action queue system
func InitTestActionManager(ecsmanager *common.EntityManager, pl *avatar.PlayerData) {
	// No action queue initialization needed anymore
}
