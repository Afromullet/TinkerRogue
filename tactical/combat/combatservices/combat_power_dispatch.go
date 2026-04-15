package combatservices

import (
	"fmt"
	"game_main/common"
	"game_main/tactical/combat/combatstate"
	"game_main/tactical/powers/artifacts"
	"game_main/tactical/powers/perks"

	"github.com/bytearena/ecs"
)

// setupPowerDispatch configures loggers and creates the perk dispatcher.
// The actual dispatch wiring happens in NewCombatService via Fire* methods.
//
// Execution order per event (enforced by Fire* methods on CombatService):
//
//	PostReset:        artifacts.OnPostReset → perks.TurnStart
//	OnAttackComplete: artifacts.OnAttackComplete → perks state tracking
//	OnTurnEnd:        artifacts charge refresh + OnTurnEnd → perks round reset
//	OnMoveComplete:   perks movement tracking (no artifact hook)
func setupPowerDispatch(cs *CombatService, manager *common.EntityManager, cache *combatstate.CombatQueryCache) {
	artifacts.SetArtifactLogger(func(behaviorKey string, squadID ecs.EntityID, message string) {
		fmt.Printf("[GEAR] %s: %s (squad %d)\n", behaviorKey, message, squadID)
	})

	perks.SetPerkLogger(func(perkID perks.PerkID, squadID ecs.EntityID, message string) {
		fmt.Printf("[PERK] %s: %s (squad %d)\n", perkID, message, squadID)
	})

	cs.perkDispatcher = &perks.SquadPerkDispatcher{}
	cs.CombatActSystem.SetPerkDispatcher(cs.perkDispatcher)
}
