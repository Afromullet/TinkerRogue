package testing

import (
	"game_main/avatar"
	"game_main/common"
	"game_main/entitytemplates"
	"game_main/gear"
	"game_main/graphics"
	"game_main/rendering"
	"game_main/timesystem"
	"game_main/worldmap"
	"log"
	"strconv"

	"github.com/bytearena/ecs"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

var TestSquare = graphics.NewTileSquare(0, 0, 3)
var TestLine = graphics.NewTileLine(0, 0, 5, graphics.LinedDiagonalDownLeft)
var TestCone = graphics.NewTileCone(0, 0, 3, graphics.LineDiagonalUpRight)
var TestCircle = graphics.NewTileCircle(0, 0, 2)
var TestRect = graphics.NewTileRectangle(0, 0, 2, 3)
var TestBurning = gear.NewBurning(5, 2)
var TestSticky = gear.NewSticky(5, 2)
var TestFreezing = gear.NewFreezing(3, 5)

var TestFireEffect = graphics.NewFireEffect(0, 0, 1, 2, 1, 0.5)
var TestCloudEffect = graphics.NewCloudEffect(0, 0, 2)
var TestIceEffect = graphics.NewIceEffect(0, 0, 2)
var TestElectricEffect = graphics.NewElectricityEffect(0, 0, 2)
var TestStickyEffect = graphics.NewStickyGroundEffect(0, 0, 2)

func CreateTestConsumables(ecsmanager *common.EntityManager, gm *worldmap.GameMap) {
	ent := entitytemplates.CreateConsumableFromTemplate(*ecsmanager, entitytemplates.ConsumableTemplates[0])
	pos := common.GetPosition(ent)
	rend := common.GetComponentType[*rendering.Renderable](ent, rendering.RenderableComponent)
	rend.Visible = true
	pos.X = gm.StartingPosition().X + 1
	pos.Y = gm.StartingPosition().Y + 2
	gm.AddEntityToTile(ent, &common.Position{X: pos.X, Y: pos.Y})

	ent = entitytemplates.CreateConsumableFromTemplate(*ecsmanager, entitytemplates.ConsumableTemplates[1])
	pos = common.GetPosition(ent)
	rend = common.GetComponentType[*rendering.Renderable](ent, rendering.RenderableComponent)
	rend.Visible = true
	pos.X = gm.StartingPosition().X + 1
	pos.Y = gm.StartingPosition().Y + 2

	gm.AddEntityToTile(ent, &common.Position{X: pos.X, Y: pos.Y})

	ent = entitytemplates.CreateConsumableFromTemplate(*ecsmanager, entitytemplates.ConsumableTemplates[2])
	pos = common.GetPosition(ent)
	rend = common.GetComponentType[*rendering.Renderable](ent, rendering.RenderableComponent)
	rend.Visible = true
	pos.X = gm.StartingPosition().X + 1
	pos.Y = gm.StartingPosition().Y + 2

	gm.AddEntityToTile(ent, &common.Position{X: pos.X, Y: pos.Y})
}

func CreateTestThrowable(shape graphics.TileBasedShape, vx graphics.VisualEffect) *gear.Throwable {

	t := gear.NewThrowable(1, 2, 3, shape)
	t.VX = vx
	return t
}

func CreateTestItems(manager *ecs.Manager, tags map[string]ecs.Tag, gameMap *worldmap.GameMap) {

	/*
		swordImg, _, err := ebitenutil.NewImageFromFile("../assets/items/sword.png")
		if err != nil {
			log.Fatal(err)
		}
	*/

	itemImageLoc := "../assets/items/sword.png"

	//todo add testing location back

	startingPos := gameMap.StartingPosition()

	throwItem := CreateTestThrowable(TestSquare, TestFireEffect)

	CreateItem(manager, "SquareThrow"+strconv.Itoa(1), common.Position{X: startingPos.X, Y: startingPos.Y}, itemImageLoc,
		throwItem, TestFreezing)

	CreateItem(manager, "SquareThrow"+strconv.Itoa(1), common.Position{X: startingPos.X, Y: startingPos.Y}, itemImageLoc,
		throwItem, TestSticky)

	CreateItem(manager, "SquareThrow"+strconv.Itoa(1), common.Position{X: startingPos.X, Y: startingPos.Y}, itemImageLoc,
		throwItem, TestBurning, TestFreezing)

	throwItem = CreateTestThrowable(TestCircle, TestIceEffect)

	CreateItem(manager, "CircleThrow"+strconv.Itoa(1), common.Position{X: startingPos.X, Y: startingPos.Y}, itemImageLoc,
		throwItem, TestBurning, TestFreezing)

	throwItem = CreateTestThrowable(TestLine, TestFireEffect)

	CreateItem(manager, "LineThrow"+strconv.Itoa(1), common.Position{X: startingPos.X, Y: startingPos.Y}, itemImageLoc,
		throwItem, TestBurning, TestFreezing)

	throwItem = CreateTestThrowable(TestRect, TestElectricEffect)

	CreateItem(manager, "RectThrow"+strconv.Itoa(1), common.Position{X: startingPos.X, Y: startingPos.Y}, itemImageLoc,
		throwItem, TestBurning, TestFreezing)

	throwItem = CreateTestThrowable(TestCone, TestStickyEffect)

	CreateItem(manager, "ConeThrow"+strconv.Itoa(1), common.Position{X: startingPos.X, Y: startingPos.Y}, itemImageLoc,
		throwItem, TestBurning, TestFreezing)

	//CreateItem(manager, "Item"+strconv.Itoa(2), common.Position{X: startingPos.X, Y: startingPos.Y}, itemImageLoc, NewBurning(1, 1), NewFreezing(1, 2))

}

func UpdateContentsForTest(ecsmanager *common.EntityManager, gm *worldmap.GameMap) {

	for _, item := range ecsmanager.World.Query(ecsmanager.WorldTags["items"]) {

		item_pos := item.Components[common.PositionComponent].(*common.Position)

		gm.AddEntityToTile(item.Entity, item_pos)

	}

}

// Create an item with any number of Effects. ItemEffect is a wrapper around an ecs.Component to make
// Manipulating it easier
func CreateItem(manager *ecs.Manager, name string, pos common.Position, imagePath string, effects ...gear.StatusEffects) *ecs.Entity {

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
		AddComponent(common.PositionComponent, &common.Position{
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

// A weapon is an Item with a weapon component
func CreateWeapon(manager *ecs.Manager, name string, pos common.Position, imagePath string, MinDamage int, MaxDamage int, properties ...gear.StatusEffects) *ecs.Entity {

	weapon := CreateItem(manager, name, pos, imagePath, properties...)

	weapon.AddComponent(gear.MeleeWeaponComponent, &gear.MeleeWeapon{
		MinDamage:   MinDamage,
		MaxDamage:   MaxDamage,
		AttackSpeed: 3,
	})

	return weapon

}

func CreateArmor(manager *ecs.Manager, name string, pos common.Position, imagePath string, ac int, prot int, dodge float32) *ecs.Entity {

	armor := CreateItem(manager, name, pos, imagePath)

	armor.AddComponent(gear.ArmorComponent, &gear.Armor{
		ArmorClass:  ac,
		Protection:  prot,
		DodgeChance: dodge,
	})

	return armor

}

func CreatedRangedWeapon(manager *ecs.Manager, name string, imagePath string, pos common.Position, minDamage int, maxDamage int, shootingRange int, TargetArea graphics.TileBasedShape) *ecs.Entity {

	weapon := CreateItem(manager, name, pos, imagePath)
	weapon.AddComponent(gear.RangedWeaponComponent, &gear.RangedWeapon{
		MinDamage:     minDamage,
		MaxDamage:     maxDamage,
		ShootingRange: shootingRange,
		TargetArea:    TargetArea,
		ShootingVX:    graphics.NewProjectile(0, 0, 0, 0),
		AttackSpeed:   3,
	})

	return weapon

}

func InitTestActionManager(ecsmanager *common.EntityManager, pl *avatar.PlayerData, ts *timesystem.GameTurn) {

	actionQueue := common.GetComponentType[*timesystem.ActionQueue](pl.PlayerEntity, timesystem.ActionQueueComponent)
	actionQueue.Entity = pl.PlayerEntity

	ts.ActionDispatcher.AddActionQueue(actionQueue)

	for _, c := range ecsmanager.World.Query(ecsmanager.WorldTags["monsters"]) {

		actionQueue = common.GetComponentType[*timesystem.ActionQueue](c.Entity, timesystem.ActionQueueComponent)

		if actionQueue != nil {
			actionQueue.Entity = c.Entity
			ts.ActionDispatcher.AddActionQueue(actionQueue)

		}

	}

}
