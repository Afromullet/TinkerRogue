// Package main implements the core game loop and initialization for the roguelike game.
// It uses the Ebiten 2D game engine and manages the ECS (Entity Component System),
// input handling, rendering, and overall game state coordination.
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
var DEBUG_MODE = false
var ENABLE_BENCHMARKING = false

type Game struct {
	em               common.EntityManager
	gameUI           gui.PlayerUI
	playerData       avatar.PlayerData
	gameMap          worldmap.GameMap //Logical map
	inputCoordinator *input.InputCoordinator
}

// NewGame creates and initializes a new Game instance with all necessary components.
// It sets up the ECS world, player data, game map, spawning systems, and UI.
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

	testing.CreateTestItems(g.em.World, g.em.WorldTags, &g.gameMap)

	testing.UpdateContentsForTest(&g.em, &g.gameMap)
	spawning.SpawnStartingCreatures(0, &g.em, &g.gameMap, &g.playerData)

	testing.InitTestActionManager(&g.em, &g.playerData)

	/*
		logicalPos := graphics.LogicalPosition{X: g.playerData.Pos.X, Y: g.playerData.Pos.Y}
		pixelPos := graphics.CoordManager.LogicalToPixel(logicalPos)
		pX, pY := pixelPos.X, pixelPos.Y

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

	spawning.SpawnStartingEquipment(&g.em, &g.gameMap, &g.playerData)

	AddCreaturesToTracker(&g.em)

	return g

}

// HandleInput processes all player input and updates game state.
// It handles movement, combat, UI interactions through the InputCoordinator,
// updates player stats, processes status effects, and cleans up dead entities.
func HandleInput(g *Game) {

	gear.UpdateEntityAttributes(g.playerData.PlayerEntity)
	g.gameUI.StatsUI.StatsTextArea.SetText(g.playerData.PlayerAttributes().DisplayString())

	// Handle all input through the InputCoordinator
	g.inputCoordinator.HandleInput()

	if g.playerData.InputStates.HasKeyInput {
		gear.RunEffectTracker(g.playerData.PlayerEntity)
		g.gameUI.StatsUI.StatsTextArea.SetText(g.playerData.PlayerAttributes().DisplayString())
		g.playerData.InputStates.HasKeyInput = false
	}

	// Clean up dead entities
	resmanager.RemoveDeadEntities(&g.em, &g.gameMap)
}

// Update is called each frame by the Ebiten engine.
// It processes UI updates, visual effects, debug input, and main game logic.
func (g *Game) Update() error {

	g.gameUI.MainPlayerInterface.Update()

	gui.SetContainerLocation(g.gameUI.StatsUI.StatUIContainer, g.gameMap.RightEdgeX, 0)

	graphics.VXHandler.UpdateVisualEffects()

	input.PlayerDebugActions(&g.playerData)

	HandleInput(g)

	return nil

}

// BenchmarkSetup initializes performance profiling tools when benchmarking is enabled.
// It starts an HTTP server for pprof and configures CPU/memory profiling rates.
func BenchmarkSetup() {

	if ENABLE_BENCHMARKING {

		go func() {
			http.ListenAndServe("localhost:6060", nil)
		}()

		runtime.SetCPUProfileRate(1000)
		runtime.MemProfileRate = 1

	}

}

// Draw renders the game to the screen buffer.
// It handles map rendering, entity rendering, UI drawing, and visual effects.
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

// Layout returns the game's logical screen dimensions.
// It calculates canvas size based on tile size, dungeon dimensions, and device scale.
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

	// Initialize the InputCoordinator after the UI is created
	g.inputCoordinator = input.NewInputCoordinator(&g.em, &g.playerData, &g.gameMap, &g.gameUI)

	ebiten.SetWindowResizable(true)

	ebiten.SetWindowTitle("Tower")

	if err := ebiten.RunGame(g); err != nil {
		log.Fatal(err)
	}
}
