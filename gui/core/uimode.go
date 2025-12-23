package core

import (
	"game_main/common"

	"github.com/ebitenui/ebitenui"
	"github.com/hajimehoshi/ebiten/v2"
)

// UIMode represents a distinct UI context (exploration, combat, squad management, etc.)
type UIMode interface {
	// Initialize is called once when mode is first created
	Initialize(ctx *UIContext) error

	// Enter is called when switching TO this mode
	// Receives the mode we're coming from (nil if starting game)
	Enter(fromMode UIMode) error

	// Exit is called when switching FROM this mode to another
	// Receives the mode we're going to
	Exit(toMode UIMode) error

	// Update is called every frame while mode is active
	// deltaTime in seconds
	Update(deltaTime float64) error

	// Render is called to draw this mode's UI
	// screen is the target ebiten image
	Render(screen *ebiten.Image)

	// HandleInput processes input events specific to this mode
	// Returns true if input was consumed (prevents propagation)
	HandleInput(inputState *InputState) bool

	// GetEbitenUI returns the root ebitenui.UI for this mode
	GetEbitenUI() *ebitenui.UI

	// GetModeName returns identifier for this mode (for debugging/logging)
	GetModeName() string
}

// UIContext provides shared game state to all UI modes
type UIContext struct {
	ECSManager      *common.EntityManager
	PlayerData      *common.PlayerData
	GameMap         interface{} // Todo, remove in future. Here for dangervisualzier, so that we can debug
	ScreenWidth     int
	ScreenHeight    int
	TileSize        int
	ModeCoordinator *GameModeCoordinator // For context switching
	Queries         interface{}          // *GUIQueries - Shared queries for all UI modes (interface{} to avoid circular import)
	// Add other commonly needed game state
}

// InputState captures current frame's input
type InputState struct {
	MouseX            int
	MouseY            int
	MousePressed      bool
	MouseReleased     bool
	MouseButton       ebiten.MouseButton
	KeysPressed       map[ebiten.Key]bool
	KeysJustPressed   map[ebiten.Key]bool
	PlayerInputStates *common.PlayerInputStates // Bridge to existing system
}

// ModeTransition represents a request to change modes
type ModeTransition struct {
	ToMode UIMode
	Reason string      // For debugging
	Data   interface{} // Optional data passed to new mode
}
