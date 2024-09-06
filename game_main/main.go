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
	_ "image/png"
	"log"

	"github.com/bytearena/ecs"
	"github.com/ebitenui/ebitenui"
	"github.com/hajimehoshi/ebiten/v2"
)

//Using https://www.fatoldyeti.com/categories/roguelike-tutorial/ as a starting point.
//Copying some of the code with modification. Whenever I change a name, it's to help me build a better mental model
//Of what the code is doing as I'm learning GoLang

type Game struct {
	gameMap             GameMap
	screenData          ScreenData
	World               *ecs.Manager
	WorldTags           map[string]ecs.Tag
	Turn                TurnState
	TurnCounter         int
	mainPlayerInterface *ebitenui.UI

	playerData PlayerData
	itemsUI    PlayerItemsUI
}

// Throwing an item will show a square to represent the AOE of the throwable.
// Right now it's a function of Game until I separate the UI more.
// Not going to try to generalize/abstract this until I figure out how I want to handle this
// The impression I get now is that this will take a "state machine" since the throwable window closes
// Once I click out of it
func (g *Game) IsThrowableItemSelected() bool {

	return g.itemsUI.throwableItemDisplay.ThrowableItemSelected

}

func (g *Game) SetThrowableItemSelected(selected bool) {

	g.itemsUI.throwableItemDisplay.ThrowableItemSelected = selected

}

// NewGame creates a new Game Object and initializes the data
// This is a pretty solid refactor candidate for later
func NewGame() *Game {
	g := &Game{}
	g.gameMap = NewGameMap()
	g.playerData = PlayerData{}
	InitializeECS(g)
	InitializePlayerData(g)

	g.Turn = PlayerTurn
	g.TurnCounter = 0
	g.screenData = NewScreenData() //todo change all calls to screendata to reference this one

	//g.craftingUI.SetCraftingWindowLocation(g.screenData.screenWidth/2, g.screenData.screenWidth/2)

	CreateTestItems(g.World, g.WorldTags, &g.gameMap)
	CreateTestMonsters(g, g.World, &g.gameMap)
	SetupPlayerForTesting(g)
	UpdateContentsForTest(g)

	return g

}

// Update is called each tic.
func (g *Game) Update() error {

	g.mainPlayerInterface.Update()
	UpdateVisualEffects()
	// Update the Label text to indicate if the ui is currently being hovered over or not
	//g.headerLbl.Label = fmt.Sprintf("Game Demo!\nUI is hovered: %t", input.UIHovered)

	g.TurnCounter++

	if g.Turn == PlayerTurn && g.TurnCounter > 20 {

		PlayerActions(g)
	}
	if g.Turn == MonsterTurn {
		MonsterSystems(g)
	}

	return nil

}

// Draw is called each draw cycle and is where we will blit.
func (g *Game) Draw(screen *ebiten.Image) {
	g.gameMap.DrawLevel(screen)
	ProcessRenderables(g, g.gameMap, screen)
	g.mainPlayerInterface.Draw(screen)
	ProcessUserLog(g, screen)
	DrawVisualEffects(screen)

}

// Layout will return the screen dimensions.
func (g *Game) Layout(w, h int) (int, int) {
	gd := NewScreenData()
	return gd.TileWidth * gd.ScreenWidth, gd.TileHeight * gd.ScreenHeight

}

func main() {

	g := NewGame()

	g.mainPlayerInterface = g.CreatePlayerUI()

	ebiten.SetWindowResizable(true)

	ebiten.SetWindowTitle("Tower")

	if err := ebiten.RunGame(g); err != nil {
		log.Fatal(err)
	}
}
