package main

import (
	"log"
	"strconv"

	"github.com/bytearena/ecs"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

func CreateTestItems(manager *ecs.Manager, tags map[string]ecs.Tag, gameMap *GameMap) {

	swordImg, _, err := ebitenutil.NewImageFromFile("assets/items/sword.png")
	if err != nil {
		log.Fatal(err)
	}
	log.Print(swordImg)

	itemImageLoc := "assets/items/sword.png"

	//todo add testing location back

	startingPos := gameMap.GetStartingPosition()
	CreateItem(manager, "Item"+strconv.Itoa(1), Position{X: startingPos.X, Y: startingPos.Y}, itemImageLoc, NewBurning(1, 1))
	//CreateItem(manager, "Item"+strconv.Itoa(2), Position{X: startingPos.X, Y: startingPos.Y}, itemImageLoc, NewBurning(1, 1), NewFreezing(1, 2))

}

func CreateTestMonsters(manager *ecs.Manager, gameMap *GameMap) {

	elfImg, _, err := ebitenutil.NewImageFromFile("assets/creatures/elf.png")
	if err != nil {
		log.Fatal(err)
	}

	//Don't create a creature in the starting room
	for _, r := range gameMap.Rooms[0:3] {

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
			AddComponent(simpleWander, &SimpleWander{})
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
			AddComponent(noMove, &NoMovement{})
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
			AddComponent(goToPlayer, &GoToPlayerMovement{})
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
