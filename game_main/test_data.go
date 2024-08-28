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

	startingPos := gameMap.GetStartingPosition()
	CreateItem(manager, "Throwable Item"+strconv.Itoa(1), Position{X: startingPos.X, Y: startingPos.Y}, itemImageLoc,
		NewThrowable(1, 5, 3, NewSquareAtPixel(0, 0, 3)), NewBurning(1, 1))

	CreateItem(manager, "Item"+strconv.Itoa(1), Position{X: startingPos.X, Y: startingPos.Y}, itemImageLoc, NewBurning(1, 1))
	CreateItem(manager, "Item"+strconv.Itoa(1), Position{X: startingPos.X, Y: startingPos.Y}, itemImageLoc, NewBurning(1, 1))
	CreateItem(manager, "Item"+strconv.Itoa(2), Position{X: startingPos.X, Y: startingPos.Y}, itemImageLoc, NewBurning(1, 1), NewFreezing(1, 2))
	CreateItem(manager, "T2"+strconv.Itoa(1), Position{X: startingPos.X, Y: startingPos.Y}, itemImageLoc,
		NewThrowable(1, 5, 3, NewSquareAtPixel(0, 0, 5)))

	//CreateItem(manager, "Item"+strconv.Itoa(2), Position{X: startingPos.X, Y: startingPos.Y}, itemImageLoc, NewBurning(1, 1), NewFreezing(1, 2))

}

var defendingMonsterTestPosition *Position = &Position{}

func CreateTestMonsters(manager *ecs.Manager, gameMap *GameMap) {

	elfImg, _, err := ebitenutil.NewImageFromFile("assets/creatures/elf.png")
	if err != nil {
		log.Fatal(err)
	}

	x, y := gameMap.Rooms[0].Center()
	pos := Position{
		X: x + 1,
		Y: y + 1}

	defendingMonsterTestPosition.X = pos.X
	defendingMonsterTestPosition.Y = pos.Y

	manager.NewEntity().
		AddComponent(creature, &Creature{
			path: make([]Position, 0),
		}).
		AddComponent(renderable, &Renderable{
			Image:   elfImg,
			visible: true,
		}).
		AddComponent(position, &pos).
		AddComponent(simpleWander, &NoMovement{}).
		AddComponent(healthComponent, &Health{
			MaxHealth:     5,
			CurrentHealth: 5}).AddComponent(userMessage, &UserMessage{
		AttackMessage:    "",
		GameStateMessage: "",
	})

	//Don't create a creature in the starting room
	for _, r := range gameMap.Rooms[1:3] {

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
				visible: true,
			}).
			AddComponent(position, &pos).
			AddComponent(simpleWander, &SimpleWander{}).
			AddComponent(healthComponent, &Health{
				MaxHealth:     5,
				CurrentHealth: 5}).AddComponent(userMessage, &UserMessage{
			AttackMessage:    "",
			GameStateMessage: "",
		})
	}

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
				visible: true,
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
				visible: true,
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
