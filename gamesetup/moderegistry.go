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
	"game_main/mind/ai"
	"game_main/mind/combatlifecycle"
	"game_main/mind/encounter"
	"game_main/tactical/combat"
	"game_main/tactical/combatservices"
)

// RegisterTacticalModes registers all tactical UI modes with the coordinator.
func RegisterTacticalModes(coordinator *framework.GameModeCoordinator, manager *framework.UIModeManager, encounterService *encounter.EncounterService) {
	combatServiceFactory := newCombatServiceFactory()

	modes := []framework.UIMode{
		guiexploration.NewExplorationMode(manager),
		guicombat.NewCombatMode(manager, encounterService, combatServiceFactory),
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
	startCombat := func(starter combat.CombatStarter) error {
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
func RegisterRoguelikeTacticalModes(coordinator *framework.GameModeCoordinator, manager *framework.UIModeManager, encounterService *encounter.EncounterService) *guiraid.RaidMode {
	raidMode := guiraid.NewRaidMode(manager)

	modes := []framework.UIMode{
		guisquads.NewSquadEditorMode(manager),
		guisquads.NewUnitPurchaseMode(manager),
		guisquads.NewArtifactMode(manager),
		guiunitview.NewUnitViewMode(manager),
		guiexploration.NewExplorationMode(manager),
		guicombat.NewCombatMode(manager, encounterService, newCombatServiceFactory()),
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

// newCombatServiceFactory returns a factory that creates a CombatService with AI fully wired.
// This keeps the mind/ai import in gamesetup (bootstrapping layer) rather than in gui/guicombat.
func newCombatServiceFactory() func(*common.EntityManager) *combatservices.CombatService {
	return func(manager *common.EntityManager) *combatservices.CombatService {
		service := combatservices.NewCombatService(manager)
		aiSetup := ai.SetupCombatAI(
			manager, service.TurnManager, service.MovementSystem,
			service.CombatActSystem, service.CombatCache,
		)
		service.SetAIController(aiSetup.Controller)
		service.SetThreatProvider(aiSetup.ThreatProvider)
		service.SetThreatEvaluatorFactory(aiSetup.EvalFactory)
		return service
	}
}
