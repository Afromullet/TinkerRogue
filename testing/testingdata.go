package testing

import (
	"game_main/avatar"
	"game_main/common"
	entitytemplates "game_main/datareader"
	"game_main/equipment"
	"game_main/graphics"
	monster "game_main/monsters"
	"game_main/timesystem"
	"game_main/worldmap"
	"log"
	"strconv"

	"github.com/bytearena/ecs"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

var TestSquare = graphics.NewTileSquare(0, 0, 3)
var TestLine = graphics.NewTileLine(0, 0, 2, graphics.LineDown)
var TestCone = graphics.NewTileCone(0, 0, 3, graphics.LineRight)
var TestCircle = graphics.NewTileCircle(0, 0, 2)
var TestRect = graphics.NewTileRectangle(0, 0, 2, 3)
var TestBurning = equipment.NewBurning(5, 2)
var TestSticky = equipment.NewSticky(9, 2)
var TestFreezing = equipment.NewFreezing(3, 5)

var TestFireEffect = graphics.NewFireEffect(0, 0, 1, 2, 1, 0.5)
var TestCloudEffect = graphics.NewCloudEffect(0, 0, 2)
var TestIceEffect = graphics.NewIceEffect(0, 0, 2)
var TestElectricEffect = graphics.NewElectricityEffect(0, 0, 2)
var TestStickyEffect = graphics.NewStickyGroundEffect(0, 0, 2)

func SetupPlayerForTesting(ecsmanager *common.EntityManager, pl *avatar.PlayerData) {
	w := CreateWeapon(ecsmanager.World, "Weapon 1", *pl.Pos, "../assets/items/sword.png", 5, 10)

	wepArea := graphics.NewTileRectangle(0, 0, 3, 3)
	r := CreatedRangedWeapon(ecsmanager.World, "Ranged Weapon 1", "../assets/items/sword.png", *pl.Pos, 5, 10, 3, &wepArea)

	pl.PlayerWeapon = w
	pl.PlayerRangedWeapon = r

}

func CreateTestThrowable(shape graphics.TileBasedShape, vx graphics.VisualEffect) *equipment.Throwable {

	t := equipment.NewThrowable(1, 2, 3, shape)
	t.VX = vx
	return t
}

func CreateTestItems(manager *ecs.Manager, tags map[string]ecs.Tag, gameMap *worldmap.GameMap) {

	swordImg, _, err := ebitenutil.NewImageFromFile("../assets/items/sword.png")
	if err != nil {
		log.Fatal(err)
	}
	log.Print(swordImg)

	itemImageLoc := "../assets/items/sword.png"

	//todo add testing location back

	startingPos := gameMap.StartingPosition()

	TestBurning.MainProps.Duration = 10
	TestFreezing.MainProps.Duration = 10
	TestSticky.MainProps.Duration = 10

	throwItem := CreateTestThrowable(&TestSquare, TestFireEffect)

	CreateItem(manager, "SquareThrow"+strconv.Itoa(1), common.Position{X: startingPos.X, Y: startingPos.Y}, itemImageLoc,
		throwItem, TestBurning, TestFreezing)

	throwItem = CreateTestThrowable(&TestCircle, TestIceEffect)

	CreateItem(manager, "CircleThrow"+strconv.Itoa(1), common.Position{X: startingPos.X, Y: startingPos.Y}, itemImageLoc,
		throwItem, TestBurning, TestFreezing)

	throwItem = CreateTestThrowable(&TestLine, TestCloudEffect)

	CreateItem(manager, "LineThrow"+strconv.Itoa(1), common.Position{X: startingPos.X, Y: startingPos.Y}, itemImageLoc,
		throwItem, TestBurning, TestFreezing)

	throwItem = CreateTestThrowable(&TestRect, TestElectricEffect)

	CreateItem(manager, "RectThrow"+strconv.Itoa(1), common.Position{X: startingPos.X, Y: startingPos.Y}, itemImageLoc,
		throwItem, TestBurning, TestFreezing)

	throwItem = CreateTestThrowable(&TestCone, TestStickyEffect)

	CreateItem(manager, "ConeThrow"+strconv.Itoa(1), common.Position{X: startingPos.X, Y: startingPos.Y}, itemImageLoc,
		throwItem, TestBurning, TestFreezing)

	//CreateItem(manager, "Item"+strconv.Itoa(2), common.Position{X: startingPos.X, Y: startingPos.Y}, itemImageLoc, NewBurning(1, 1), NewFreezing(1, 2))

}

func CreateTestMonsters(manager *ecs.Manager, pl *avatar.PlayerData, gameMap *worldmap.GameMap) {
	x, y := gameMap.Rooms[0].Center()

	wepArea := graphics.NewTileRectangle(0, 0, 1, 1)
	wep := equipment.RangedWeapon{
		MinDamage:     3,
		MaxDamage:     5,
		ShootingRange: 5,
		TargetArea:    &wepArea,
		AttackSpeed:   5,
	}

	ent := entitytemplates.CreateCreatureFromTemplate(manager, entitytemplates.MonsterTemplates[0], gameMap, x+1, y)

	//ent.AddComponent(monsters.DistanceRangeAttackComp, &monsters.DistanceRangedAttack{})
	ent.AddComponent(equipment.RangedWeaponComponent, &wep)
	//ent.AddComponent(monster.WithinRangeComponent, &monsters.DistanceToEntityMovement{Target: pl.PlayerEntity, Distance: 3})
	ent.AddComponent(monster.RangeAttackBehaviorComp, &monster.AttackBehavior{})

	ent = entitytemplates.CreateCreatureFromTemplate(manager, entitytemplates.MonsterTemplates[0], gameMap, x+2, y)

	ent.AddComponent(monster.ChargeAttackComp, &monster.AttackBehavior{})
	/*
		ent = entitytemplates.CreateCreatureFromTemplate(manager, entitytemplates.MonsterTemplates[0], gameMap, x+2, y)
		ent.AddComponent(monsters.WithinRangeComponent, &monsters.DistanceToEntityMovement{Distance: 5, Target: pl.PlayerEntity})

		ent = entitytemplates.CreateCreatureFromTemplate(manager, entitytemplates.MonsterTemplates[0], gameMap, x+3, y)
		ent.AddComponent(monsters.EntityFollowComp, &monsters.EntityFollow{Target: pl.PlayerEntity})
	*/

}

func CreateMonster(manager *ecs.Manager, gameMap *worldmap.GameMap, x, y int, img string) *ecs.Entity {

	elfImg, _, err := ebitenutil.NewImageFromFile(img)
	if err != nil {
		log.Fatal(err)
	}

	ind := graphics.IndexFromXY(x, y)
	gameMap.Tiles[ind].Blocked = true
	testArmor := equipment.Armor{15, 3, 30}

	ent := manager.NewEntity().
		AddComponent(monster.CreatureComponent, &monster.Creature{
			Path: make([]common.Position, 0),
		}).
		AddComponent(common.RenderableComponent, &common.Renderable{
			Image:   elfImg,
			Visible: true,
		}).
		AddComponent(common.PositionComponent, &common.Position{
			X: x,
			Y: y,
		}).
		AddComponent(common.AttributeComponent, &common.Attributes{MaxHealth: 5, CurrentHealth: 5, TotalAttackSpeed: 30, TotalMovementSpeed: 1}).
		AddComponent(equipment.ArmorComponent, &testArmor).
		AddComponent(equipment.WeaponComponent, &equipment.MeleeWeapon{
			MinDamage:   3,
			MaxDamage:   5,
			AttackSpeed: 30,
		}).
		AddComponent(timesystem.ActionQueueComponent, &timesystem.ActionQueue{TotalActionPoints: 100})

	armor := equipment.GetArmor(ent)
	common.UpdateAttributes(ent, armor.ArmorClass, armor.Protection, armor.DodgeChance)

	return ent

}

func UpdateContentsForTest(ecsmanager *common.EntityManager, gm *worldmap.GameMap) {

	for _, item := range ecsmanager.World.Query(ecsmanager.WorldTags["items"]) {

		item_pos := item.Components[common.PositionComponent].(*common.Position)

		gm.AddEntityToTile(item.Entity, item_pos)

	}

}

// Create an item with any number of Effects. ItemEffect is a wrapper around an ecs.Component to make
// Manipulating it easier
func CreateItem(manager *ecs.Manager, name string, pos common.Position, imagePath string, effects ...equipment.StatusEffects) *ecs.Entity {

	img, _, err := ebitenutil.NewImageFromFile(imagePath)
	if err != nil {
		log.Fatal(err)
	}

	item := &equipment.Item{Count: 1, Properties: manager.NewEntity()}

	for _, prop := range effects {
		item.Properties.AddComponent(prop.StatusEffectComponent(), &prop)

	}

	itemEntity := manager.NewEntity().
		AddComponent(common.RenderableComponent, &common.Renderable{
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
		AddComponent(equipment.ItemComponent, item)

	//TODO where shoudl I add the tags?

	return itemEntity

}

// A weapon is an Item with a weapon component
func CreateWeapon(manager *ecs.Manager, name string, pos common.Position, imagePath string, MinDamage int, MaxDamage int, properties ...equipment.StatusEffects) *ecs.Entity {

	weapon := CreateItem(manager, name, pos, imagePath, properties...)

	weapon.AddComponent(equipment.WeaponComponent, &equipment.MeleeWeapon{
		MinDamage:   MinDamage,
		MaxDamage:   MaxDamage,
		AttackSpeed: 3,
	})

	return weapon

}

func CreatedRangedWeapon(manager *ecs.Manager, name string, imagePath string, pos common.Position, minDamage int, maxDamage int, shootingRange int, TargetArea graphics.TileBasedShape) *ecs.Entity {

	weapon := CreateItem(manager, name, pos, imagePath)
	weapon.AddComponent(equipment.RangedWeaponComponent, &equipment.RangedWeapon{
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
