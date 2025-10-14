// Package main implements the core game loop and initialization for the roguelike game.
// It uses the Ebiten 2D game engine and manages the ECS (Entity Component System),
// input handling, rendering, and overall game state coordination.
//
// Setup: Run `go mod tidy` to install dependencies
// Build: Run `go build -o game_main/game_main.exe game_main/*.go`
// Run: Execute `go run game_main/*.go`
package main

import (
	"game_main/avatar"
	"game_main/common"
	"game_main/graphics"
	"game_main/gui"
	"game_main/input"
	"game_main/rendering"
	"game_main/testing"
	"game_main/worldmap"
	"log"
	"math"

	_ "image/png" // Required for PNG image loading

	"github.com/hajimehoshi/ebiten/v2"
)

// Game holds all game state and systems.
// It is the main struct passed to the Ebiten game engine.
type Game struct {
	em               common.EntityManager
	uiModeManager    *gui.UIModeManager // NEW: Modal UI system
	playerData       avatar.PlayerData
	gameMap          worldmap.GameMap
	inputCoordinator *input.InputCoordinator
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
	// Update UI mode manager (handles input and UI state)
	deltaTime := 1.0 / 60.0 // 60 FPS
	if err := g.uiModeManager.Update(deltaTime); err != nil {
		return err
	}

	graphics.VXHandler.UpdateVisualEffects()

	input.PlayerDebugActions(&g.playerData)

	HandleInput(g)

	return nil
}

// Draw renders the game to the screen buffer.
// It handles map rendering, entity rendering, UI drawing, and visual effects.
func (g *Game) Draw(screen *ebiten.Image) {
	// Update screen dimensions
	graphics.ScreenInfo.ScreenWidth = screen.Bounds().Dx()
	graphics.ScreenInfo.ScreenHeight = screen.Bounds().Dy()

	// Phase 1: Ebiten rendering (game world)
	if graphics.MAP_SCROLLING_ENABLED {
		g.gameMap.DrawLevelCenteredSquare(screen, g.playerData.Pos, graphics.ViewableSquareSize, DEBUG_MODE)
		rendering.ProcessRenderablesInSquare(&g.em, g.gameMap, screen, g.playerData.Pos, graphics.ViewableSquareSize, DEBUG_MODE)
	} else {
		g.gameMap.DrawLevel(screen, DEBUG_MODE)
		rendering.ProcessRenderables(&g.em, g.gameMap, screen, DEBUG_MODE)
	}

	graphics.VXHandler.DrawVisualEffects(screen)

	// Phase 2: EbitenUI rendering (modal UI)
	g.uiModeManager.Render(screen)
}

// Layout returns the game's logical screen dimensions.
// It calculates canvas size based on tile size, dungeon dimensions, and device scale.
func (g *Game) Layout(w, h int) (int, int) {
	scale := ebiten.DeviceScaleFactor()
	canvasWidth := int(math.Ceil(float64(graphics.ScreenInfo.TileSize*graphics.ScreenInfo.DungeonWidth) * scale))
	canvasHeight := int(math.Ceil(float64(graphics.ScreenInfo.TileSize*graphics.ScreenInfo.DungeonHeight) * scale))
	return canvasWidth + graphics.StatsUIOffset, canvasHeight
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

	testing.CreateTestItems(g.em.World, g.em.WorldTags, &g.gameMap)

	testing.UpdateContentsForTest(&g.em, &g.gameMap)

	// Configure window
	ebiten.SetWindowResizable(true)
	ebiten.SetWindowTitle("Tower")

	// Start game loop
	if err := ebiten.RunGame(g); err != nil {
		log.Fatal(err)
	}
}
