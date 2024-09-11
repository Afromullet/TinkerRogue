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
	"fmt"
	"game_main/actionmanager"
	"game_main/avatar"
	"game_main/common"
	"game_main/graphics"
	"game_main/gui"
	"game_main/input"
	"game_main/monsters"
	"game_main/rendering"
	"game_main/testing"
	"game_main/timesystem"
	"game_main/worldmap"
	_ "image/png"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
)

//Using https://www.fatoldyeti.com/categories/roguelike-tutorial/ as a starting point.
//Copying some of the code with modification. Whenever I change a name, it's to help me build a better mental model
//Of what the code is doing as I'm learning GoLang

type Game struct {
	em         common.EntityManager
	gameUI     gui.PlayerUI
	playerData avatar.PlayerData
	gameMap    worldmap.GameMap

	ts timesystem.GameTurn
}

// NewGame creates a new Game Object and initializes the data
// This is a pretty solid refactor candidate for later
func NewGame() *Game {
	g := &Game{}
	g.gameMap = worldmap.NewGameMap()
	g.playerData = avatar.PlayerData{}
	InitializeECS(&g.em)
	InitializePlayerData(&g.em, &g.playerData, &g.gameMap)

	g.ts.Turn = timesystem.PlayerTurn
	g.ts.TurnCounter = 0

	//g.craftingUI.SetCraftingWindowLocation(g.screenData.screenWidth/2, g.screenData.screenWidth/2)

	testing.CreateTestItems(g.em.World, g.em.WorldTags, &g.gameMap)
	testing.CreateTestMonsters(g.em.World, &g.playerData, &g.gameMap)
	testing.SetupPlayerForTesting(&g.em, &g.playerData)
	testing.UpdateContentsForTest(&g.em, &g.gameMap)

	testing.InitTestActionManager(&g.em, &g.playerData, &actionmanager.ActionDispatcher)

	return g

}

// Update is called each tic.
// todo still need to remove game
func (g *Game) Update() error {

	g.gameUI.MainPlayerInterface.Update()

	graphics.VXHandler.UpdateVisualEffects()
	// Update the Label text to indicate if the ui is currently being hovered over or not
	//g.headerLbl.Label = fmt.Sprintf("Game Demo!\nUI is hovered: %t", input.UIHovered)

	keyPressed := false
	if g.ts.Turn == timesystem.PlayerTurn && !keyPressed {

		keyPressed = input.PlayerActions(&g.em, &g.playerData, &g.gameMap, &g.gameUI, &g.ts)
		if keyPressed {

			g.ts.Turn = timesystem.MonsterTurn
		}

		//input.HandlePlayerThrowable(&g.em, &g.playerData, &g.gameMap, &g.gameUI)

		//input.HandlePlayerRangedAttack(&g.em, &g.playerData, &g.gameMap)

	}
	if g.ts.Turn == timesystem.MonsterTurn && keyPressed {
		monsters.MonsterSystems(&g.em, &g.playerData, &g.gameMap, &g.ts)
		g.ts.Turn = timesystem.ExecuteActions
	}

	if g.ts.Turn == timesystem.ExecuteActions && keyPressed {

		actionmanager.ActionDispatcher.ExecuteFirst()
		//actionmanager.ActionDispatcher.CleanController()
		g.ts.Turn = timesystem.PlayerTurn
		actionmanager.ActionDispatcher.DebugOutput()
		fmt.Println("Exectugin actions")
		keyPressed = false

	}

	return nil

}

// Draw is called each draw cycle and is where we will blit.
func (g *Game) Draw(screen *ebiten.Image) {
	g.gameMap.DrawLevel(screen)
	rendering.ProcessRenderables(&g.em, g.gameMap, screen)
	g.gameUI.MainPlayerInterface.Draw(screen)
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

	g.gameUI.MainPlayerInterface = gui.CreatePlayerUI(&g.gameUI, g.playerData.Inv, &g.playerData)

	ebiten.SetWindowResizable(true)

	ebiten.SetWindowTitle("Tower")

	if err := ebiten.RunGame(g); err != nil {
		log.Fatal(err)
	}
}
