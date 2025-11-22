package core

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
)

// GameContext represents which game context is currently active
type GameContext int

const (
	ContextOverworld GameContext = iota // Strategic layer (squad management, world map)
	ContextBattleMap                    // Tactical layer (dungeon exploration, combat)
)

// String returns a human-readable name for the context
func (gc GameContext) String() string {
	switch gc {
	case ContextOverworld:
		return "Overworld"
	case ContextBattleMap:
		return "BattleMap"
	default:
		return "Unknown"
	}
}

// GameModeCoordinator manages two independent UIModeManagers - one for Overworld context
// and one for BattleMap context. It handles context switching and state persistence.
type GameModeCoordinator struct {
	overworldManager      *UIModeManager  // Manages overworld modes (squad management, etc.)
	battleMapManager      *UIModeManager  // Manages battle map modes (exploration, combat)
	activeManager         *UIModeManager  // Points to currently active manager
	currentContext        GameContext     // Tracks which context is active
	overworldState        *OverworldState // Persistent overworld data
	battleMapState        *BattleMapState // Persistent battle data
	contextSwitchKeyDown  bool            // Tracks if context switch key is held (prevents rapid toggling)
}

// NewGameModeCoordinator creates a new coordinator with two separate mode managers
func NewGameModeCoordinator(ctx *UIContext) *GameModeCoordinator {
	overworldMgr := NewUIModeManager(ctx)
	battleMapMgr := NewUIModeManager(ctx)

	coordinator := &GameModeCoordinator{
		overworldManager: overworldMgr,
		battleMapManager: battleMapMgr,
		activeManager:    battleMapMgr, // Start in battle map context by default
		currentContext:   ContextBattleMap,
		overworldState:   &OverworldState{},
		battleMapState:   &BattleMapState{},
	}

	return coordinator
}

// RegisterOverworldMode registers a mode to the overworld manager
func (gmc *GameModeCoordinator) RegisterOverworldMode(mode UIMode) error {
	if err := gmc.overworldManager.RegisterMode(mode); err != nil {
		return fmt.Errorf("failed to register overworld mode %s: %w", mode.GetModeName(), err)
	}
	return nil
}

// RegisterBattleMapMode registers a mode to the battle map manager
func (gmc *GameModeCoordinator) RegisterBattleMapMode(mode UIMode) error {
	if err := gmc.battleMapManager.RegisterMode(mode); err != nil {
		return fmt.Errorf("failed to register battle map mode %s: %w", mode.GetModeName(), err)
	}
	return nil
}

// EnterBattleMap switches to the battle map context
func (gmc *GameModeCoordinator) EnterBattleMap(initialMode string) error {
	if gmc.currentContext == ContextBattleMap {
		// Already in battle map, just switch mode if needed
		if initialMode != "" {
			return gmc.battleMapManager.SetMode(initialMode)
		}
		return nil
	}

	// Save overworld state before switching
	gmc.saveOverworldState()

	// Switch to battle map manager
	gmc.activeManager = gmc.battleMapManager
	gmc.currentContext = ContextBattleMap

	// Enter the specified mode (or keep current battle map mode)
	if initialMode != "" {
		if err := gmc.battleMapManager.SetMode(initialMode); err != nil {
			return fmt.Errorf("failed to enter battle map mode %s: %w", initialMode, err)
		}
	}

	fmt.Printf("Context Switch: Overworld -> BattleMap (mode: %s)\n", initialMode)
	return nil
}

// ReturnToOverworld switches back to the overworld context
func (gmc *GameModeCoordinator) ReturnToOverworld(initialMode string) error {
	if gmc.currentContext == ContextOverworld {
		// Already in overworld, just switch mode if needed
		if initialMode != "" {
			return gmc.overworldManager.SetMode(initialMode)
		}
		return nil
	}

	// Save battle map state before switching
	gmc.saveBattleMapState()

	// Switch to overworld manager
	gmc.activeManager = gmc.overworldManager
	gmc.currentContext = ContextOverworld

	// Restore overworld state
	gmc.restoreOverworldState()

	// Enter the specified mode (or keep current overworld mode)
	if initialMode != "" {
		if err := gmc.overworldManager.SetMode(initialMode); err != nil {
			return fmt.Errorf("failed to enter overworld mode %s: %w", initialMode, err)
		}
	}

	fmt.Printf("Context Switch: BattleMap -> Overworld (mode: %s)\n", initialMode)
	return nil
}

// ToggleContext switches between Overworld and BattleMap contexts
func (gmc *GameModeCoordinator) ToggleContext() error {
	switch gmc.currentContext {
	case ContextOverworld:
		// Switch to battle map with default mode
		return gmc.EnterBattleMap("exploration")
	case ContextBattleMap:
		// Switch to overworld with default mode
		return gmc.ReturnToOverworld("squad_management")
	default:
		return fmt.Errorf("unknown context: %v", gmc.currentContext)
	}
}

// Update updates the active manager and handles context switching
func (gmc *GameModeCoordinator) Update(deltaTime float64) error {
	// Check for context switch hotkey (Tab key)
	if ebiten.IsKeyPressed(ebiten.KeyTab) && ebiten.IsKeyPressed(ebiten.KeyControl) {
		// Wait for key release to avoid rapid toggling
		if !gmc.contextSwitchKeyDown {
			gmc.contextSwitchKeyDown = true
			if err := gmc.ToggleContext(); err != nil {
				fmt.Printf("Failed to toggle context: %v\n", err)
			}
		}
	} else {
		gmc.contextSwitchKeyDown = false
	}

	// Update active manager
	if gmc.activeManager != nil {
		return gmc.activeManager.Update(deltaTime)
	}
	return nil
}

// Render renders the active manager and context indicator
func (gmc *GameModeCoordinator) Render(screen *ebiten.Image) {
	if gmc.activeManager != nil {
		gmc.activeManager.Render(screen)
	}

	// Display current context in top-right corner (for debugging/info)
	// This could be removed or made conditional in production
	gmc.renderContextIndicator(screen)
}

// renderContextIndicator draws the current context name on screen
func (gmc *GameModeCoordinator) renderContextIndicator(screen *ebiten.Image) {
	// Simple text rendering would go here
	// For now, just print to console when context changes
	// (actual text rendering would require the ebitenutil package)
}

// GetCurrentContext returns the active game context
func (gmc *GameModeCoordinator) GetCurrentContext() GameContext {
	return gmc.currentContext
}

// GetCurrentMode returns the active mode from the active manager
func (gmc *GameModeCoordinator) GetCurrentMode() UIMode {
	if gmc.activeManager != nil {
		return gmc.activeManager.GetCurrentMode()
	}
	return nil
}

// GetOverworldManager returns the overworld mode manager (for registration)
func (gmc *GameModeCoordinator) GetOverworldManager() *UIModeManager {
	return gmc.overworldManager
}

// GetBattleMapManager returns the battle map mode manager (for registration)
func (gmc *GameModeCoordinator) GetBattleMapManager() *UIModeManager {
	return gmc.battleMapManager
}

// saveOverworldState captures current overworld state before leaving
func (gmc *GameModeCoordinator) saveOverworldState() {
	// TODO: Implement state capture
	// - Current squad selection
	// - UI state (scroll positions, selections)
	// - Any pending actions
	fmt.Println("Saving overworld state...")
}

// restoreOverworldState restores overworld state when returning
func (gmc *GameModeCoordinator) restoreOverworldState() {
	// TODO: Implement state restoration
	// - Restore squad selections
	// - Restore UI state
	fmt.Println("Restoring overworld state...")
}

// saveBattleMapState captures current battle map state before leaving
func (gmc *GameModeCoordinator) saveBattleMapState() {
	// TODO: Implement state capture
	// - Current map state
	// - Combat state (turn order, positions)
	// - Battle results (loot, XP, casualties)
	fmt.Println("Saving battle map state...")
}
