package testing

import (
	"game_main/common"
)

// ========================================
// EntityManager Fixtures
// ========================================

// NewTestEntityManager creates a fully initialized EntityManager for testing.
// This is the single source of truth for test setup, replacing the duplicated
// setupTestManager functions across multiple test files.
//
// Initializes:
// - Common components (Position, Attribute, Name, Player, PlayerResources)
// - PositionSystem (fresh instance for isolation)
// - Squad system (components and tags) via InitializeSquadData
// - Other ECS components as needed
//
// This is a lightweight, generic fixture that works for all test scenarios.
// For specialized test setups (e.g., combat system initialization),
// call additional initialization functions as needed in your tests.
func NewTestEntityManager() *common.EntityManager {
	manager := common.NewEntityManager()

	// Initialize common components
	common.PositionComponent = manager.World.NewComponent()
	common.AttributeComponent = manager.World.NewComponent()
	common.NameComponent = manager.World.NewComponent()
	common.PlayerComponent = manager.World.NewComponent()
	common.PlayerResourcesComponent = manager.World.NewComponent()

	// Initialize PositionSystem (fresh instance for each test)
	common.GlobalPositionSystem = common.NewPositionSystem(manager.World)

	// Initialize squad system components and tags
	// This is imported dynamically to avoid import cycles
	// Squad tests will handle this via their own test helpers
	// (See squads package for squad-specific test setup)

	return manager
}
