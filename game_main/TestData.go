package main

import (
	"log"
	"strconv"

	"github.com/bytearena/ecs"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

func SetupPlayerForTesting(g *Game) {
	w := CreateWeapon(g.World, "Weapon 1", *g.playerData.position, "assets/items/sword.png", 5, 10)

	wepArea := NewTileRectangle(0, 0, 3, 3)
	r := CreatedRangedWeapon(g.World, "Ranged Weapon 1", "assets/items/sword.png", *g.playerData.position, 5, 10, 3, &wepArea)

	g.playerData.PlayerWeapon = w
	g.playerData.PlayerRangedWeapon = r

}

func CreateTestItems(manager *ecs.Manager, tags map[string]ecs.Tag, gameMap *GameMap) {

	swordImg, _, err := ebitenutil.NewImageFromFile("assets/items/sword.png")
	if err != nil {
		log.Fatal(err)
	}
	log.Print(swordImg)

	itemImageLoc := "assets/items/sword.png"

	//todo add testing location back

	startingPos := gameMap.StartingPosition()

	b := NewBurning(5, 2)

	f := NewFreezing(3, 5)
	f.MainProps.Duration = 10

	st := NewSticky(9, 2)

	s := NewTileSquare(0, 0, 3)
	//CreateItem(manager, "Throwable Item"+strconv.Itoa(1), Position{X: startingPos.X, Y: startingPos.Y}, itemImageLoc,
	//NewThrowable(1, 5, 3, NewTileSquare(0, 0, 3)), NewBurning(1, 1))
	CreateItem(manager, "T0"+strconv.Itoa(1), Position{X: startingPos.X, Y: startingPos.Y}, itemImageLoc,
		NewThrowable(1, 2, 3, &s), b, f)

	sout := NewTileCircleOutline(0, 0, 2)
	CreateItem(manager, "T8"+strconv.Itoa(1), Position{X: startingPos.X, Y: startingPos.Y}, itemImageLoc,
		NewThrowable(1, 2, 3, &sout), b, f)

	s = NewTileSquare(0, 0, 2)
	//CreateItem(manager, "Throwable Item"+strconv.Itoa(1), Position{X: startingPos.X, Y: startingPos.Y}, itemImageLoc,
	//NewThrowable(1, 5, 3, NewTileSquare(0, 0, 3)), NewBurning(1, 1))
	CreateItem(manager, "T7"+strconv.Itoa(1), Position{X: startingPos.X, Y: startingPos.Y}, itemImageLoc,
		NewThrowable(1, 3, 3, &s), NewBurning(1, 1), st, f)

	l := NewTileLine(0, 0, 5, LineDown)
	CreateItem(manager, "T1"+strconv.Itoa(1), Position{X: startingPos.X, Y: startingPos.Y}, itemImageLoc,
		NewThrowable(1, 2, 3, &l))

	l = NewTileLine(0, 0, 2, LineDown)
	CreateItem(manager, "T9"+strconv.Itoa(1), Position{X: startingPos.X, Y: startingPos.Y}, itemImageLoc,
		NewThrowable(1, 5, 3, &l))

	c := NewTileCone(0, 0, 5, LineDown)
	CreateItem(manager, "T2"+strconv.Itoa(1), Position{X: startingPos.X, Y: startingPos.Y}, itemImageLoc,
		NewThrowable(1, 5, 3, &c))

	ci := NewTileCircle(0, 0, 2)
	CreateItem(manager, "T3"+strconv.Itoa(1), Position{X: startingPos.X, Y: startingPos.Y}, itemImageLoc,
		NewThrowable(1, 5, 3, &ci))

	re := NewTileRectangle(0, 0, 2, 3)
	CreateItem(manager, "T4"+strconv.Itoa(1), Position{X: startingPos.X, Y: startingPos.Y}, itemImageLoc,
		NewThrowable(1, 5, 3, &re))

	CreateItem(manager, "Item"+strconv.Itoa(1), Position{X: startingPos.X, Y: startingPos.Y}, itemImageLoc, NewBurning(1, 1))
	CreateItem(manager, "Item"+strconv.Itoa(1), Position{X: startingPos.X, Y: startingPos.Y}, itemImageLoc, NewBurning(1, 1))
	CreateItem(manager, "Item"+strconv.Itoa(2), Position{X: startingPos.X, Y: startingPos.Y}, itemImageLoc, NewBurning(1, 1), NewFreezing(1, 2))

	//CreateItem(manager, "Item"+strconv.Itoa(2), Position{X: startingPos.X, Y: startingPos.Y}, itemImageLoc, NewBurning(1, 1), NewFreezing(1, 2))

}

func CreateTestMonsters(g *Game, manager *ecs.Manager, gameMap *GameMap) {

	x, y := gameMap.Rooms[0].Center()

	wepArea := NewTileRectangle(0, 0, 1, 1)

	wep := RangedWeapon{
		MinDamage:     3,
		MaxDamage:     5,
		ShootingRange: 5,
		TargetArea:    &wepArea,
	}

	c := CreateMonster(g, manager, gameMap, x, y+1, "assets/creatures/elf.png")

	c.AddComponent(distanceRangeAttack, &DistanceRangedAttack{})
	c.AddComponent(RangedWeaponComponent, &wep)

	c = CreateMonster(g, manager, gameMap, x+1, y, "assets/creatures/unseen_horror.png")
	c.AddComponent(simpleWanderComp, &SimpleWander{})

	c = CreateMonster(g, manager, gameMap, x+1, y+1, "assets/creatures/angel.png")
	c.AddComponent(entityFollowComp, &EntityFollow{target: g.playerData.PlayerEntity})

	c = CreateMonster(g, manager, gameMap, x+1, y+2, "assets/creatures/ancient_lich.png")
	c.AddComponent(withinRadiusComp, &DistanceToEntityMovement{target: g.playerData.PlayerEntity, distance: 3})

	c = CreateMonster(g, manager, gameMap, x+2, y+1, "assets/creatures/starcursed_mass.png")
	c.AddComponent(withinRangeComponent, &DistanceToEntityMovement{distance: 2, target: g.playerData.PlayerEntity})
	//CreateMonster(g, manager, gameMap, x+2, y+2, "assets/creatures/balrug.png")

	CreateMoreTestMonsters(manager, gameMap)

	//CreateMoreTestMonsters(manager, gameMap)

}

func CreateMoreTestMonsters(manager *ecs.Manager, gameMap *GameMap) {

	elfImg, _, err := ebitenutil.NewImageFromFile("assets/creatures/elf.png")
	if err != nil {
		log.Fatal(err)
	}

	//Don't create a creature in the starting room
	for _, r := range gameMap.Rooms[1:9] {

		x, y := r.Center()
		pos := Position{
			X: x,
			Y: y}

		manager.NewEntity().
			AddComponent(CreatureComponent, &Creature{
				Path: make([]Position, 0),
			}).
			AddComponent(RenderableComponent, &Renderable{
				Image:   elfImg,
				Visible: true,
			}).
			AddComponent(PositionComponent, &pos).
			AddComponent(entityFollowComp, &EntityFollow{}).
			AddComponent(AttributeComponent, &Attributes{MaxHealth: 5, CurrentHealth: 5}).AddComponent(userMessage, &UserMessage{
			AttackMessage:    "",
			GameStateMessage: "",
		})
	}

}

func CreateMonster(g *Game, manager *ecs.Manager, gameMap *GameMap, x, y int, img string) *ecs.Entity {

	elfImg, _, err := ebitenutil.NewImageFromFile(img)
	if err != nil {
		log.Fatal(err)
	}

	ind := IndexFromXY(x, y)
	gameMap.Tiles[ind].Blocked = true

	ent := manager.NewEntity().
		AddComponent(CreatureComponent, &Creature{
			Path: make([]Position, 0),
		}).
		AddComponent(RenderableComponent, &Renderable{
			Image:   elfImg,
			Visible: true,
		}).
		AddComponent(PositionComponent, &Position{
			X: x,
			Y: y,
		}).
		AddComponent(AttributeComponent, &Attributes{MaxHealth: 5, CurrentHealth: 5}).
		AddComponent(userMessage, &UserMessage{
			AttackMessage:    "",
			GameStateMessage: "",
		}).
		AddComponent(ArmorComponent, &Armor{15, 3, 30}).
		AddComponent(WeaponComponent, &Weapon{
			MinDamage: 3,
			MaxDamage: 5,
		})

	UpdateAttributes(ent)

	return ent

}

func UpdateContentsForTest(g *Game) {

	for _, item := range g.World.Query(g.WorldTags["items"]) {

		item_pos := item.Components[PositionComponent].(*Position)

		g.gameMap.AddEntityToTile(item.Entity, item_pos)

	}

}

func GetTileInfo(g *Game, pos *Position, player *Player) {

	for _, item := range g.World.Query(g.WorldTags["items"]) {

		item_pos := item.Components[PositionComponent].(*Position)
		log.Print("Item Pos: \n")
		log.Print(item_pos)

		if pos.IsEqual(item_pos) {
			log.Print("here\n")
		}

	}

}

// Create an item with any number of Effects. ItemEffect is a wrapper around an ecs.Component to make
// Manipulating it easier
func CreateItem(manager *ecs.Manager, name string, pos Position, imagePath string, effects ...StatusEffects) *ecs.Entity {

	img, _, err := ebitenutil.NewImageFromFile(imagePath)
	if err != nil {
		log.Fatal(err)
	}

	item := &Item{Count: 1, Properties: manager.NewEntity()}

	for _, prop := range effects {
		item.Properties.AddComponent(prop.StatusEffectComponent(), &prop)

	}

	itemEntity := manager.NewEntity().
		AddComponent(RenderableComponent, &Renderable{
			Image:   img,
			Visible: true,
		}).
		AddComponent(PositionComponent, &Position{
			X: pos.X,
			Y: pos.Y,
		}).
		AddComponent(nameComponent, &Name{
			NameStr: name,
		}).
		AddComponent(ItemComponent, item)

	//TODO where shoudl I add the tags?

	return itemEntity

}

// A weapon is an Item with a weapon component
func CreateWeapon(manager *ecs.Manager, name string, pos Position, imagePath string, MinDamage int, MaxDamage int, properties ...StatusEffects) *ecs.Entity {

	weapon := CreateItem(manager, name, pos, imagePath, properties...)

	weapon.AddComponent(WeaponComponent, &Weapon{
		MinDamage: MinDamage,
		MaxDamage: MaxDamage,
	})

	return weapon

}

func CreatedRangedWeapon(manager *ecs.Manager, name string, imagePath string, pos Position, minDamage int, maxDamage int, shootingRange int, TargetArea TileBasedShape) *ecs.Entity {

	weapon := CreateItem(manager, name, pos, imagePath)
	weapon.AddComponent(RangedWeaponComponent, &RangedWeapon{
		MinDamage:     minDamage,
		MaxDamage:     maxDamage,
		ShootingRange: shootingRange,
		TargetArea:    TargetArea,
	})

	return weapon

}
