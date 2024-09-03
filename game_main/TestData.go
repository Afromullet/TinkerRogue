package main

import (
	"log"
	"strconv"

	"github.com/bytearena/ecs"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

func SetupPlayerForTesting(g *Game) {
	w := CreateWeapon(g.World, "Weapon 1", *g.playerData.position, "assets/items/sword.png", 1)
	g.playerData.playerWeapon = w

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

	b := NewBurning(1, 1)
	b.MainProps.Duration = 5

	s := NewTileSquare(0, 0, 3)
	//CreateItem(manager, "Throwable Item"+strconv.Itoa(1), Position{X: startingPos.X, Y: startingPos.Y}, itemImageLoc,
	//NewThrowable(1, 5, 3, NewTileSquare(0, 0, 3)), NewBurning(1, 1))
	CreateItem(manager, "T0"+strconv.Itoa(1), Position{X: startingPos.X, Y: startingPos.Y}, itemImageLoc,
		NewThrowable(1, 5, 3, &s), b)

	s = NewTileSquare(0, 0, 2)
	//CreateItem(manager, "Throwable Item"+strconv.Itoa(1), Position{X: startingPos.X, Y: startingPos.Y}, itemImageLoc,
	//NewThrowable(1, 5, 3, NewTileSquare(0, 0, 3)), NewBurning(1, 1))
	CreateItem(manager, "T7"+strconv.Itoa(1), Position{X: startingPos.X, Y: startingPos.Y}, itemImageLoc,
		NewThrowable(1, 5, 3, &s), NewBurning(1, 1))

	l := NewTileLine(0, 0, 5, LineDown)
	CreateItem(manager, "T1"+strconv.Itoa(1), Position{X: startingPos.X, Y: startingPos.Y}, itemImageLoc,
		NewThrowable(1, 5, 3, &l))

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

func CreateTestMonsters(manager *ecs.Manager, gameMap *GameMap) {

	x, y := gameMap.Rooms[0].Center()

	CreateMonster(manager, gameMap, x, y+1)
	CreateMonster(manager, gameMap, x+1, y)
	CreateMonster(manager, gameMap, x+1, y+1)
	CreateMonster(manager, gameMap, x+1, y+2)
	CreateMonster(manager, gameMap, x+2, y+1)
	CreateMonster(manager, gameMap, x+2, y+2)

	CreateMoreTestMonsters(manager, gameMap)

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
			AddComponent(creature, &Creature{
				path: make([]Position, 0),
			}).
			AddComponent(renderable, &Renderable{
				Image:   elfImg,
				Visible: true,
			}).
			AddComponent(position, &pos).
			AddComponent(goToPlayer, &GoToPlayerMovement{}).
			AddComponent(healthComponent, &Health{
				MaxHealth:     5,
				CurrentHealth: 5}).AddComponent(userMessage, &UserMessage{
			AttackMessage:    "",
			GameStateMessage: "",
		})
	}

	/*
		//Don't create a creature in the starting room
		for _, r := range gameMap.Rooms[3:4] {

			x, y := r.Center()
			pos := Position{
				X: x,
				Y: y}

			manager.NewEntity().
				AddComponent(creature, &Creature{}).
				AddComponent(renderable, &Renderable{
					Image:   elfImg,
					Visible: true,
				}).
				AddComponent(position, &pos).
				AddComponent(noMove, &NoMovement{}).
				AddComponent(healthComponent, &Health{
					MaxHealth:     5,
					CurrentHealth: 5}).AddComponent(userMessage, &UserMessage{
				AttackMessage:    "",
				GameStateMessage: "",
			})
		}
	*/
	/*
		//Don't create a creature in the starting room
		for _, r := range gameMap.Rooms[4:] {

			x, y := r.Center()
			pos := Position{
				X: x,
				Y: y}

			manager.NewEntity().
				AddComponent(creature, &Creature{}).
				AddComponent(renderable, &Renderable{
					Image:   elfImg,
					Visible: true,
				}).
				AddComponent(position, &pos).
				AddComponent(goToPlayer, &GoToPlayerMovement{}).
				AddComponent(healthComponent, &Health{
					MaxHealth:     5,
					CurrentHealth: 5}).AddComponent(userMessage, &UserMessage{
				AttackMessage:    "",
				GameStateMessage: "",
			})
		}
	*/

}

func CreateMonster(manager *ecs.Manager, gameMap *GameMap, x, y int) {

	elfImg, _, err := ebitenutil.NewImageFromFile("assets/creatures/elf.png")
	if err != nil {
		log.Fatal(err)
	}

	manager.NewEntity().
		AddComponent(creature, &Creature{
			path: make([]Position, 0),
		}).
		AddComponent(renderable, &Renderable{
			Image:   elfImg,
			Visible: true,
		}).
		AddComponent(position, &Position{
			X: x,
			Y: y,
		}).
		AddComponent(noMove, &NoMovement{}).
		AddComponent(healthComponent, &Health{
			MaxHealth:     5,
			CurrentHealth: 5}).AddComponent(userMessage, &UserMessage{
		AttackMessage:    "",
		GameStateMessage: "",
	})

}

func UpdateContentsForTest(g *Game) {

	for _, item := range g.World.Query(g.WorldTags["items"]) {

		item_pos := item.Components[position].(*Position)

		g.gameMap.AddEntityToTile(item.Entity, item_pos)
		log.Print("Item Pos: \n")
		log.Print(item_pos)

	}

}

func GetTileInfo(g *Game, pos *Position, player *Player) {

	for _, item := range g.World.Query(g.WorldTags["items"]) {

		item_pos := item.Components[position].(*Position)
		log.Print("Item Pos: \n")
		log.Print(item_pos)

		if pos.IsEqual(item_pos) {
			log.Print("here\n")
		}

	}

}
