package framework

import (
	"fmt"

	"game_main/common"

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
	overworldManager *UIModeManager // Manages overworld modes (squad management, etc.)
	battleMapManager *UIModeManager // Manages battle map modes (exploration, combat)
	activeManager    *UIModeManager // Points to currently active manager
	currentContext   GameContext    // Tracks which context is active

	battleMapState  *BattleMapState  // Persistent battle data
	overworldState  *OverworldState  // Persistent overworld data

	context *UIContext // Reference to shared UIContext for cache management
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

		battleMapState: NewBattleMapState(),
		overworldState: NewOverworldState(),
		context:        ctx, // Store reference to UIContext for cache management
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
	return gmc.switchToContext(ContextBattleMap, gmc.battleMapManager, initialMode)
}

// ReturnToOverworld switches back to the overworld context
func (gmc *GameModeCoordinator) ReturnToOverworld(initialMode string) error {
	return gmc.switchToContext(ContextOverworld, gmc.overworldManager, initialMode)
}

// switchToContext handles the common logic for switching between game contexts.
func (gmc *GameModeCoordinator) switchToContext(targetContext GameContext, targetManager *UIModeManager, initialMode string) error {
	if gmc.currentContext == targetContext {
		// Already in target context, just switch mode if needed
		if initialMode != "" {
			return targetManager.SetMode(initialMode)
		}
		return nil
	}

	fromContext := gmc.currentContext

	// Switch to target manager
	gmc.activeManager = targetManager
	gmc.currentContext = targetContext

	// Enter the specified mode (or keep current mode)
	if initialMode != "" {
		if err := targetManager.SetMode(initialMode); err != nil {
			return fmt.Errorf("failed to enter %s mode %s: %w", targetContext, initialMode, err)
		}
	}

	fmt.Printf("Context Switch: %s -> %s (mode: %s)\n", fromContext, targetContext, initialMode)
	return nil
}

// Update updates the active manager and handles context switching
func (gmc *GameModeCoordinator) Update(deltaTime float64) error {

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

// GetBattleMapState returns the persistent battle map state for UI modes
func (gmc *GameModeCoordinator) GetBattleMapState() *BattleMapState {
	return gmc.battleMapState
}

// GetOverworldState returns the persistent overworld state for UI modes
func (gmc *GameModeCoordinator) GetOverworldState() *OverworldState {
	return gmc.overworldState
}

// GetPlayerData returns the player data from the UI context
func (gmc *GameModeCoordinator) GetPlayerData() *common.PlayerData {
	if gmc.context != nil {
		return gmc.context.PlayerData
	}
	return nil
}
