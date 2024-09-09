package main

/*
When setting up the project, run go mod tidy to install dependencies

*/
//Original import statmenets. Started adding ebiten UI stuff in the other import statements. This is to fall back on

/*
import (
	_ "image/png"
	"log"

	"github.com/bytearena/ecs"
	"github.com/hajimehoshi/ebiten/v2"
)*/

import (
	"game_main/avatar"
	"game_main/common"
	"game_main/graphics"
	"game_main/worldmap"
	_ "image/png"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
)

//Using https://www.fatoldyeti.com/categories/roguelike-tutorial/ as a starting point.
//Copying some of the code with modification. Whenever I change a name, it's to help me build a better mental model
//Of what the code is doing as I'm learning GoLang

type Game struct {
	common.EntityManager
	PlayerUI
	playerData avatar.PlayerData
	gameMap    worldmap.GameMap

	common.TimeSystem
}

// NewGame creates a new Game Object and initializes the data
// This is a pretty solid refactor candidate for later
func NewGame() *Game {
	g := &Game{}
	g.gameMap = worldmap.NewGameMap()
	g.playerData = avatar.PlayerData{}
	InitializeECS(&g.EntityManager)
	avatar.InitializePlayerData(&g.EntityManager, &g.playerData, &g.gameMap)

	g.Turn = common.PlayerTurn
	g.TurnCounter = 0

	//g.craftingUI.SetCraftingWindowLocation(g.screenData.screenWidth/2, g.screenData.screenWidth/2)

	CreateTestItems(g.World, g.WorldTags, &g.gameMap)
	CreateTestMonsters(g, g.World, &g.gameMap)
	SetupPlayerForTesting(g)
	UpdateContentsForTest(g)

	return g

}

// Update is called each tic.
// todo still need to remove game
func (g *Game) Update() error {

	g.mainPlayerInterface.Update()

	graphics.VXHandler.UpdateVisualEffects()
	// Update the Label text to indicate if the ui is currently being hovered over or not
	//g.headerLbl.Label = fmt.Sprintf("Game Demo!\nUI is hovered: %t", input.UIHovered)

	g.TurnCounter++

	if g.Turn == common.PlayerTurn && g.TurnCounter > 20 {

		PlayerActions(&g.EntityManager, &g.playerData, &g.gameMap, &g.PlayerUI, &g.TimeSystem)
	}
	if g.Turn == common.MonsterTurn {
		MonsterSystems(&g.EntityManager, &g.playerData, &g.gameMap, &g.TimeSystem)
	}

	return nil

}

// Draw is called each draw cycle and is where we will blit.
func (g *Game) Draw(screen *ebiten.Image) {
	g.gameMap.DrawLevel(screen)
	ProcessRenderables(&g.EntityManager, g.gameMap, screen)
	g.mainPlayerInterface.Draw(screen)
	ProcessUserLog(g, screen)

	graphics.VXHandler.DrawVisualEffects(screen)

}

// Layout will return the screen dimensions.
func (g *Game) Layout(w, h int) (int, int) {
	gd := graphics.NewScreenData()
	return gd.TileWidth * gd.ScreenWidth, gd.TileHeight * gd.ScreenHeight

}

func main() {

	g := NewGame()

	g.mainPlayerInterface = CreatePlayerUI(&g.PlayerUI, &g.playerData)

	ebiten.SetWindowResizable(true)

	ebiten.SetWindowTitle("Tower")

	if err := ebiten.RunGame(g); err != nil {
		log.Fatal(err)
	}
}
