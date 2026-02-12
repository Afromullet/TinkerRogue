package main

import (
	"fmt"
	"game_main/common"
	"game_main/tactical/squads"
	"game_main/templates"

	"github.com/bytearena/ecs"
)

// bootstrapECS initializes the minimum ECS state needed for headless combat simulation.
// Replicates game_main/componentinit.go logic without any rendering/ebiten dependencies.
func bootstrapECS() {
	// 1. Load game data from JSON files (monster templates, configs, etc.)
	templates.ReadGameData()

	// 2. Convert JSON monster data into UnitTemplates
	if err := squads.InitUnitTemplatesFromJSON(); err != nil {
		panic(fmt.Sprintf("failed to initialize unit templates: %v", err))
	}

	fmt.Printf("Loaded %d unit templates\n", len(squads.Units))
}

// newSimManager creates a fresh EntityManager with all subsystem components registered.
// Each scenario gets its own manager to avoid state leaking between battles.
func newSimManager() *common.EntityManager {
	manager := common.NewEntityManager()

	// Register core components (replicates game_main/componentinit.go:registerCoreComponents)
	common.PositionComponent = manager.World.NewComponent()
	common.NameComponent = manager.World.NewComponent()
	common.AttributeComponent = manager.World.NewComponent()
	common.PlayerComponent = manager.World.NewComponent()
	common.ResourceStockpileComponent = manager.World.NewComponent()

	// Build core tags (replicates game_main/componentinit.go:buildCoreTags)
	common.AllEntitiesTag = ecs.BuildTag()
	manager.WorldTags["all"] = common.AllEntitiesTag

	// Initialize all registered subsystems (squads, combat, gear, etc.)
	common.InitializeSubsystems(manager)

	// Initialize position system
	common.GlobalPositionSystem = common.NewPositionSystem(manager.World)

	return manager
}
