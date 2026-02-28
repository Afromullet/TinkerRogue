package framework

import (
	"fmt"

	"game_main/common"
	"game_main/world/worldmap"

	"github.com/bytearena/ecs"
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
	ECSManager       *common.EntityManager
	PlayerData       *common.PlayerData
	GameMap          *worldmap.GameMap
	ScreenWidth      int
	ScreenHeight     int
	TileSize         int
	ModeCoordinator  *GameModeCoordinator // For context switching
	Queries          *GUIQueries          // Shared queries for all UI modes
	SaveGameCallback func() error         // Called by UI to trigger save; nil if save not available
	LoadGameCallback func()               // Called by UI to request a load; nil if not available
}

// GetSquadRosterOwnerID returns the entity that owns the active squad roster.
// In overworld context, this is the selected commander. Falls back to player entity ID.
func (ctx *UIContext) GetSquadRosterOwnerID() ecs.EntityID {
	if ctx.ModeCoordinator != nil {
		owState := ctx.ModeCoordinator.GetOverworldState()
		if owState != nil && owState.SelectedCommanderID != 0 {
			return owState.SelectedCommanderID
		}
	}
	fmt.Println("WARNING: GetSquadRosterOwnerID - no commander selected, falling back to player entity")
	if ctx.PlayerData != nil {
		return ctx.PlayerData.PlayerEntityID
	}
	return 0
}

// InputState captures current frame's input
type InputState struct {
	MouseX            int
	MouseY            int
	MousePressed      bool
	MouseJustPressed  bool
	MouseButton       ebiten.MouseButton
	KeysPressed       map[ebiten.Key]bool
	KeysJustPressed   map[ebiten.Key]bool
	KeysJustReleased  map[ebiten.Key]bool
	PlayerInputStates *common.PlayerInputStates // Bridge to existing system

	// Per-button just-pressed tracking (more precise than MouseJustPressed which ORs all buttons)
	mouseJustPressedButtons map[ebiten.MouseButton]bool

	// Semantic actions resolved from the current mode's ActionMap
	ActionsActive map[InputAction]bool
}

// ActionActive returns true if the given semantic action is active this frame.
func (is *InputState) ActionActive(action InputAction) bool {
	return is.ActionsActive[action]
}

// AnyKeyJustPressed returns true if any key was just pressed this frame.
func (is *InputState) AnyKeyJustPressed() bool {
	for _, pressed := range is.KeysJustPressed {
		if pressed {
			return true
		}
	}
	return false
}

// MouseJustPressedButton returns true if the specific mouse button was just pressed this frame.
func (is *InputState) MouseJustPressedButton(button ebiten.MouseButton) bool {
	return is.mouseJustPressedButtons[button]
}

// ModeTransition represents a request to change modes
type ModeTransition struct {
	ToMode UIMode
	Reason string // For debugging
}
