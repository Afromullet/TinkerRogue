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
	"game_main/avatar"
	"game_main/behavior"
	"game_main/common"
	"game_main/entitytemplates"
	"game_main/gear"
	"game_main/graphics"
	"game_main/rendering"
	resmanager "game_main/resourcemanager"
	"math"
	"runtime"

	"game_main/gui"
	"game_main/input"
	"game_main/spawning"
	"game_main/testing"
	"game_main/timesystem"
	"game_main/worldmap"
	_ "image/png"
	"log"
	_ "net/http/pprof" // Blank import to register pprof handle

	"github.com/hajimehoshi/ebiten/v2"

	"net/http"
	_ "net/http/pprof" // Blank import to register pprof handlers
)

// Using https://www.fatoldyeti.com/categories/roguelike-tutorial/ as a starting point.
// Copying some of the code with modification. Whenever I change a name, it's to help me build a better mental model
// Of what the code is doing as I'm learning GoLang
var DEBUG_MODE = true
var ENABLE_BENCHMARKING = false

type Game struct {
	em         common.EntityManager
	gameUI     gui.PlayerUI
	playerData avatar.PlayerData
	gameMap    worldmap.GameMap //Logical map

	ts timesystem.GameTurn
}

// NewGame creates a new Game Object and initializes the data
// This is a pretty solid refactor candidate for later
func NewGame() *Game {
	g := &Game{}
	g.gameMap = worldmap.NewGameMap()

	g.playerData = avatar.PlayerData{}
	entitytemplates.ReadGameData()
	InitializeECS(&g.em)

	graphics.ScreenInfo.ScaleFactor = 1
	if graphics.MAP_SCROLLING_ENABLED {
		graphics.ScreenInfo.ScaleFactor = 3
	}
	InitializePlayerData(&g.em, &g.playerData, &g.gameMap)
	spawning.InitLootSpawnTables()

	g.ts.Turn = timesystem.PlayerTurn
	g.ts.TurnCounter = 0

	testing.CreateTestItems(g.em.World, g.em.WorldTags, &g.gameMap)

	testing.UpdateContentsForTest(&g.em, &g.gameMap)
	spawning.SpawnStartingCreatures(0, &g.em, &g.gameMap, &g.playerData)

	testing.InitTestActionManager(&g.em, &g.playerData, &g.ts)

	/*
		pX, pY := graphics.CoordTransformer.PixelsFromLogicalXY(g.playerData.Pos.X, g.playerData.Pos.Y)

		pos := g.gameMap.UnblockedLogicalCoords(pX, pY, 10)

		for _, p := range pos {

			it := spawning.SpawnThrowableItem(g.em.World, p.X, p.Y)

			g.gameMap.AddEntityToTile(it, &common.Position{X: p.X, Y: p.Y})

		}

			//TODO remove, the spawning functions are here for testing
			for _ = range 10 {
				sX, sY := g.gameMap.Rooms[0].Center()
				sX += 3

				it := spawning.SpawnThrowableItem(g.em.World, sX, sY)

				g.gameMap.AddEntityToTile(it, &common.Position{X: sX, Y: sY})

				sX, sY = g.gameMap.Rooms[0].Center()
				sX += 2
				it = spawning.SpawnConsumable(g.em.World, sX, sY)
				g.gameMap.AddEntityToTile(it, &common.Position{X: sX, Y: sY})

			}
	*/

	g.ts.ActionDispatcher.ResetActionManager()

	timesystem.TurnManager = &g.ts
	//spawning.SpawnStartingLoot(g.em, &g.gameMap)
	spawning.SpawnStartingEquipment(&g.em, &g.gameMap, &g.playerData)

	AddCreaturesToTracker(&g.em)

	return g

}

// Once the player performs an action, the Action Manager adds Monster actions to the queue.
// Performs all of the actions. Then it reorders them.
// When the Turn Counter hits 0, we reset all action points. That's our "unit of time"
func ManageTurn(g *Game) {

	gear.UpdateEntityAttributes(g.playerData.PlayerEntity)
	//g.playerData.UpdatePlayerAttributes()
	g.gameUI.StatsUI.StatsTextArea.SetText(g.playerData.PlayerAttributes().DisplayString())
	if g.ts.Turn == timesystem.PlayerTurn && !g.playerData.InputStates.HasKeyInput {

		//Apply Consumabl Effects at beginning of player turn
		//gear.ConsumableEffectApplier(g.playerData.PlayerEntity)

		input.PlayerActions(&g.em, &g.playerData, &g.gameMap, &g.gameUI, &g.ts)
		if g.playerData.InputStates.HasKeyInput {

			gear.RunEffectTracker(g.playerData.PlayerEntity)

			g.gameUI.StatsUI.StatsTextArea.SetText(g.playerData.PlayerAttributes().DisplayString())
			g.ts.Turn = timesystem.MonsterTurn

		}

		// The drawing and throwing still work after changing the way the input and actions work
		// Uncommented now because we need to figure out how to implement this in the Action Energy based ystem
		if g.gameUI.IsThrowableItemSelected() {
			g.playerData.InputStates.IsThrowing = true

		} else {
			g.playerData.InputStates.IsThrowing = false
		}
		input.HandlePlayerThrowable(&g.em, &g.playerData, &g.gameMap, &g.gameUI)
		input.HandlePlayerRangedAttack(&g.em, &g.playerData, &g.gameMap)

	}
	if g.ts.Turn == timesystem.MonsterTurn && g.playerData.InputStates.HasKeyInput {
		behavior.MonsterSystems(&g.em, &g.playerData, &g.gameMap, &g.ts)

		// Returns true if the next action is the player.

		//ExecuteActionsUntilPlayer2 places the queue back in priority order. The old function executes each action only once
		// untilk the player

		//todo why is here twice

		//resmanager.RemoveDeadEntities(&g.em, &g.gameMap)
		g.ts.ActionDispatcher.CleanController()
		if g.ts.ActionDispatcher.ExecuteActionsUntilPlayer2(&g.playerData) {

			//Perform the players action
			g.ts.ActionDispatcher.ExecuteFirst()

		}

		//g.ts.ActionDispatcher.ReorderActions() // If executefirst inserts in priority order I won't need this
		g.ts.UpdateTurnCounter()

		g.playerData.InputStates.HasKeyInput = false
		g.ts.Turn = timesystem.PlayerTurn

		if g.ts.TotalNumTurns%10 == 0 {

			//addspawning.SpawnMonster(g.em, &g.gameMap)
		}

		spawning.SpawnLootAroundPlayer(g.ts.TotalNumTurns, g.playerData, g.em.World, &g.gameMap)

		resmanager.RemoveDeadEntities(&g.em, &g.gameMap)

	}

}

// Update is called each tic.
// todo still need to remove game
func (g *Game) Update() error {

	g.gameUI.MainPlayerInterface.Update()

	gui.SetContainerLocation(g.gameUI.StatsUI.StatUIContainer, g.gameMap.RightEdgeX, 0)

	graphics.VXHandler.UpdateVisualEffects()

	input.PlayerDebugActions(&g.playerData)

	ManageTurn(g)

	return nil

}

func BenchmarkSetup() {

	if ENABLE_BENCHMARKING {

		go func() {
			fmt.Println("Running server")
			http.ListenAndServe("localhost:6060", nil)
		}()

		runtime.SetCPUProfileRate(1000)
		runtime.MemProfileRate = 1

	}

}

// Draw is called each draw cycle and is where we will blit.
func (g *Game) Draw(screen *ebiten.Image) {

	//Not sure how to get the screen outside of the draw function, so I guess I will do it here for now
	graphics.ScreenInfo.ScreenWidth = screen.Bounds().Dx()
	graphics.ScreenInfo.ScreenHeight = screen.Bounds().Dy()

	if graphics.MAP_SCROLLING_ENABLED {
		g.gameMap.DrawLevelCenteredSquare(screen, g.playerData.Pos, graphics.ViewableSquareSize, DEBUG_MODE)
		rendering.ProcessRenderablesInSquare(&g.em, g.gameMap, screen, g.playerData.Pos, graphics.ViewableSquareSize, DEBUG_MODE)
	} else {
		g.gameMap.DrawLevel(screen, DEBUG_MODE)
		rendering.ProcessRenderables(&g.em, g.gameMap, screen, DEBUG_MODE)
	}

	gui.ProcessUserLog(g.em, screen, &g.gameUI.MsgUI)

	graphics.VXHandler.DrawVisualEffects(screen)
	g.gameUI.MainPlayerInterface.Draw(screen)

}

/*
// Layout will return the screen dimensions.
func (g *Game) Layout(w, h int) (int, int) {
	gd := graphics.NewScreenData()
	//return gd.TileWidth * gd.DungeonWidth, gd.TileHeight * gd.DungeonHeight
	return gd.TileWidth * gd.DungeonWidth, gd.TileHeight * gd.DungeonHeight

}
*/

func (g *Game) Layout(w, h int) (int, int) {
	scale := ebiten.DeviceScaleFactor()

	//return gd.TileWidth * gd.DungeonWidth, gd.TileHeight * gd.DungeonHeight
	canvasWidth := int(math.Ceil(float64(graphics.ScreenInfo.TileSize*graphics.ScreenInfo.DungeonWidth) * scale))
	canvasHeight := int(math.Ceil(float64(graphics.ScreenInfo.TileSize*graphics.ScreenInfo.DungeonHeight) * scale))
	return canvasWidth + graphics.StatsUIOffset, canvasHeight

}

func main() {

	//log.Println(http.ListenAndServe("localhost:6060", nil))

	BenchmarkSetup()
	g := NewGame()

	g.gameUI.CreateMainInterface(&g.playerData, &g.em)

	ebiten.SetWindowResizable(true)

	ebiten.SetWindowTitle("Tower")

	if err := ebiten.RunGame(g); err != nil {
		log.Fatal(err)
	}
}
