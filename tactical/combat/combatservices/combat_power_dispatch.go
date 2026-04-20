package combatservices

import (
	"fmt"
	"game_main/core/common"
	"game_main/tactical/combat/combatstate"
	"game_main/tactical/powers/artifacts"
	"game_main/tactical/powers/perks"
	"game_main/tactical/powers/powercore"

	"github.com/bytearena/ecs"
)

// setupPowerDispatch configures the shared PowerLogger and creates the perk dispatcher.
// The actual dispatch wiring happens in NewCombatService, where subscribers are
// registered on cs.powerPipeline in declared execution order.
//
// Execution order per event (declared by On* calls in NewCombatService):
//
//	PostReset:        artifacts.OnPostReset → perks.TurnStart
//	OnAttackComplete: artifacts.OnAttackComplete → perks state tracking → GUI
//	OnTurnEnd:        artifacts charge refresh + OnTurnEnd → perks round reset → GUI
//	OnMoveComplete:   perks movement tracking → GUI (no artifact hook)
func setupPowerDispatch(cs *CombatService, manager *common.EntityManager, cache *combatstate.CombatQueryCache) {
	// Single PowerLogger shared by artifacts and perks. Source tags ("engagement_chains",
	// "counterpunch") flow through unchanged; the [GEAR] / [PERK] prefix is decided
	// here by asking the artifacts registry whether the source is a known behavior.
	logger := powercore.LoggerFunc(func(source string, squadID ecs.EntityID, message string) {
		prefix := "[PERK]"
		if artifacts.IsRegisteredBehavior(source) {
			prefix = "[GEAR]"
		}
		fmt.Printf("%s %s: %s (squad %d)\n", prefix, source, message, squadID)
	})

	cs.artifactDispatcher.SetLogger(logger)

	perkDispatcher := &perks.SquadPerkDispatcher{}
	perkDispatcher.SetLogger(logger)
	cs.perkDispatcher = perkDispatcher
	cs.CombatActSystem.SetPerkDispatcher(cs.perkDispatcher)
}
