// Package main implements the core game loop and initialization for the roguelike game.
// It uses the Ebiten 2D game engine and manages the ECS (Entity Component System),
// input handling, rendering, and overall game state coordination.
//
// Setup: Run `go mod tidy` to install dependencies
// Build: Run `go build -o game_main/game_main.exe game_main/*.go`
// Run: Execute `go run game_main/*.go`
package main

import (
	"fmt"
	"game_main/common"
	"game_main/config"
	"game_main/gui/framework"
	"game_main/input"
	"game_main/overworld/core"
	"game_main/testing"
	"game_main/visual/graphics"
	"game_main/visual/rendering"
	"game_main/world/coords"
	"game_main/world/worldmap"
	"log"
	"math"

	_ "image/png" // Required for PNG image loading

	"github.com/hajimehoshi/ebiten/v2"
)

// Game holds all game state and systems.
// It is the main struct passed to the Ebiten game engine.
type Game struct {
	em                  common.EntityManager
	gameModeCoordinator *framework.GameModeCoordinator // Coordinates Overworld and Tactical UI contexts
	playerData          common.PlayerData
	gameMap             worldmap.GameMap
	inputCoordinator    *input.InputCoordinator
	renderingCache      *rendering.RenderingCache // Cached view for rendering hot path (3-5x faster)
}

// NewGame creates and initializes a new Game instance.
// All initialization logic is delegated to the setup package.
func NewGame() *Game {
	g := &Game{}
	SetupNewGame(g)
	return g
}

// HandleInput processes all player input and updates game state.
// It handles movement, combat, UI interactions through the InputCoordinator,
// updates player stats, processes status effects, and cleans up dead entities.
func HandleInput(g *Game) {
	// Handle all input through the InputCoordinator
	g.inputCoordinator.HandleInput()

	if g.playerData.InputStates.HasKeyInput {
		g.playerData.InputStates.HasKeyInput = false
	}
}

// Update is called each frame by the Ebiten engine.
// It processes UI updates, visual effects, debug input, and main game logic.
func (g *Game) Update() error {
	// Update game mode coordinator (handles input and UI state for active context)
	deltaTime := 1.0 / 60.0 // 60 FPS
	if err := g.gameModeCoordinator.Update(deltaTime); err != nil {
		return err
	}

	graphics.VXHandler.UpdateVisualEffects()

	HandleInput(g)

	return nil
}

// Draw renders the game to the screen buffer.
// It handles map rendering, entity rendering, UI drawing, and visual effects.
func (g *Game) Draw(screen *ebiten.Image) {
	// Update screen dimensions
	graphics.ScreenInfo.ScreenWidth = screen.Bounds().Dx()
	graphics.ScreenInfo.ScreenHeight = screen.Bounds().Dy()
	coords.CoordManager.UpdateScreenDimensions(screen.Bounds().Dx(), screen.Bounds().Dy())

	// Phase 1: Render tactical map only when in tactical context
	if g.gameModeCoordinator.GetCurrentContext() == framework.ContextTactical {
		if coords.MAP_SCROLLING_ENABLED {
			bounds := rendering.DrawMapCentered(screen, &g.gameMap, g.playerData.Pos, config.DefaultZoomNumberOfSquare, config.DEBUG_MODE)
			g.gameMap.RightEdgeX = bounds.RightEdgeX
			g.gameMap.TopEdgeY = bounds.TopEdgeY
			rendering.ProcessRenderablesInSquare(g.gameMap, screen, g.playerData.Pos, config.DefaultZoomNumberOfSquare, g.renderingCache)
		} else {
			rendering.DrawMap(screen, &g.gameMap, config.DEBUG_MODE)
			rendering.ProcessRenderables(g.gameMap, screen, g.renderingCache)
		}

		graphics.VXHandler.DrawVisualEffects(screen)
	}

	// Phase 2: EbitenUI rendering (modal UI via coordinator)
	g.gameModeCoordinator.Render(screen)
}

// Layout returns the game's logical screen dimensions.
// It calculates canvas size based on tile size, dungeon dimensions, and device scale.
func (g *Game) Layout(w, h int) (int, int) {
	scale := ebiten.DeviceScaleFactor()
	canvasWidth := int(math.Ceil(float64(graphics.ScreenInfo.TileSize*graphics.ScreenInfo.DungeonWidth) * scale))
	canvasHeight := int(math.Ceil(float64(graphics.ScreenInfo.TileSize*graphics.ScreenInfo.DungeonHeight) * scale))
	return canvasWidth + config.DefaultStaticUIOffset, canvasHeight
}

// main is the entry point for the game.
// It orchestrates initialization and starts the Ebiten game loop.
func main() {
	// Setup profiling if enabled
	SetupBenchmarking()

	// Create and initialize game
	g := NewGame()

	// Setup UI and input systems
	SetupUI(g)
	SetupInputCoordinator(g)

	testing.CreateTestItems(&g.gameMap)

	testing.UpdateContentsForTest(&g.em, &g.gameMap)

	// Configure window
	ebiten.SetWindowResizable(true)
	ebiten.SetWindowTitle("Tower")

	// Start game loop
	if err := ebiten.RunGame(g); err != nil {
		log.Fatal(err)
	}

	// Export overworld log on game exit (if not already exported by victory/defeat)
	ctx := core.GetContext()
	if ctx.Recorder != nil && ctx.Recorder.IsEnabled() && ctx.Recorder.EventCount() > 0 {
		if err := core.FinalizeRecording("Exit", "Game closed"); err != nil {
			fmt.Printf("WARNING: Failed to export overworld log on exit: %v\n", err)
		}
	}
}
