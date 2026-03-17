package gamesetup

import (
	"log"

	"game_main/common"
	"game_main/gui/framework"
	"game_main/gui/guicombat"
	"game_main/gui/guiexploration"
	"game_main/gui/guinodeplacement"
	"game_main/gui/guioverworld"
	"game_main/gui/guiraid"
	"game_main/gui/guisquads"
	"game_main/gui/guiunitview"
	"game_main/mind/combatlifecycle"
	"game_main/mind/encounter"
	"game_main/tactical/combat"
)

// RegisterTacticalModes registers all tactical UI modes with the coordinator.
func RegisterTacticalModes(coordinator *framework.GameModeCoordinator, manager *framework.UIModeManager, encounterService *encounter.EncounterService) {

	modes := []framework.UIMode{
		guiexploration.NewExplorationMode(manager, nil),
		guicombat.NewCombatMode(manager, encounterService),
		guicombat.NewCombatAnimationMode(manager),
		guisquads.NewSquadDeploymentMode(manager),
	}

	for _, mode := range modes {
		if err := coordinator.RegisterTacticalMode(mode); err != nil {
			log.Fatalf("Failed to register tactical mode '%s': %v", mode.GetModeName(), err)
		}
	}
}

// RegisterOverworldModes registers all overworld UI modes with the coordinator.
// This reduces boilerplate by iterating over a slice of mode constructors.
func RegisterOverworldModes(coordinator *framework.GameModeCoordinator, manager *framework.UIModeManager, encounterService *encounter.EncounterService, ecsManager *common.EntityManager) {
	startCombat := func(starter combat.CombatStarter) (*combat.CombatStartResult, error) {
		return combatlifecycle.ExecuteCombatStart(encounterService, ecsManager, starter)
	}

	modes := []framework.UIMode{
		guioverworld.NewOverworldMode(manager, encounterService, startCombat),
		guinodeplacement.NewNodePlacementMode(manager),
		guisquads.NewUnitPurchaseMode(manager),
		guisquads.NewSquadEditorMode(manager),
		guisquads.NewArtifactMode(manager),
		guiunitview.NewUnitViewMode(manager),
	}

	for _, mode := range modes {
		if err := coordinator.RegisterOverworldMode(mode); err != nil {
			log.Fatalf("Failed to register overworld mode '%s': %v", mode.GetModeName(), err)
		}
	}
}

// RegisterRoguelikeTacticalModes registers squad + core tactical modes for roguelike.
// Squad modes are registered first so ExplorationMode.Initialize() detects squad_editor
// in the tactical manager and shows only the "Squad" button (no overworld button).
// Returns the RaidMode reference for RaidRunner injection.
func RegisterRoguelikeTacticalModes(coordinator *framework.GameModeCoordinator, manager *framework.UIModeManager, encounterService *encounter.EncounterService, ecsManager *common.EntityManager) *guiraid.RaidMode {
	raidMode := guiraid.NewRaidMode(manager)

	startCombat := func(starter combat.CombatStarter) (*combat.CombatStartResult, error) {
		return combatlifecycle.ExecuteCombatStart(encounterService, ecsManager, starter)
	}

	modes := []framework.UIMode{
		guisquads.NewSquadEditorMode(manager),
		guisquads.NewUnitPurchaseMode(manager),
		guisquads.NewArtifactMode(manager),
		guiunitview.NewUnitViewMode(manager),
		guiexploration.NewExplorationMode(manager, startCombat),
		guicombat.NewCombatMode(manager, encounterService),
		guicombat.NewCombatAnimationMode(manager),
		guisquads.NewSquadDeploymentMode(manager),
		raidMode,
	}

	for _, mode := range modes {
		if err := coordinator.RegisterTacticalMode(mode); err != nil {
			log.Fatalf("Failed to register tactical mode '%s': %v", mode.GetModeName(), err)
		}
	}

	return raidMode
}
