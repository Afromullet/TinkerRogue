package combatlifecycle

import (
	"fmt"

	"game_main/core/common"
	"game_main/core/config"
	"game_main/tactical/combat/combatstate"
	"game_main/tactical/powers/perks"
	"game_main/tactical/squads/squadcore"

	"github.com/bytearena/ecs"
)

// CombatTeardown handles tactical-side entity disposal when exiting combat.
// Implemented by CombatService (satisfies via Go structural typing, no import needed).
// The implementation strips combat-only components from player squads internally
// (faction membership, perk round state, positions, IsDeployed) — the caller
// does not need to follow up.
// Invoked by EncounterService.ExitCombat as one step in the exit orchestration —
// it is NOT the full combat-exit flow.
type CombatTeardown interface {
	TeardownCombat(enemySquadIDs []ecs.EntityID)
}

// ApplyHPRecovery restores a percentage of max HP to all living units in a squad.
func ApplyHPRecovery(manager *common.EntityManager, squadID ecs.EntityID, hpPercent int) {
	for _, unitID := range squadcore.GetUnitIDsInSquad(squadID, manager) {
		attr := common.GetComponentTypeByID[*common.Attributes](manager, unitID, common.AttributeComponent)
		if attr != nil && attr.CurrentHealth > 0 {
			heal := attr.GetMaxHealth() * hpPercent / 100
			attr.CurrentHealth += heal
			if attr.CurrentHealth > attr.GetMaxHealth() {
				attr.CurrentHealth = attr.GetMaxHealth()
			}
		}
	}
}

// StripCombatComponents removes combat-only state from the given squads when
// leaving combat: faction membership, perk round state, positions, and the
// IsDeployed flag. Each component's cleanup lives in its owning package
// (combatstate, perks, squadcore) and is invoked directly here.
//
// Used by combat-exit orchestration outside of CombatService.TeardownCombat —
// currently only by the garrison-defense return-to-node path. CombatService
// performs the same sequence inline on player squads during teardown.
func StripCombatComponents(manager *common.EntityManager, squadIDs []ecs.EntityID) {
	for _, squadID := range squadIDs {
		entity := manager.FindEntityByID(squadID)
		if entity == nil {
			continue
		}

		combatstate.RemoveCombatMembership(entity)
		perks.RemovePerkRoundState(entity)
		squadcore.ResetSquadDeployment(manager, entity)

		if config.DEBUG_MODE {
			fmt.Printf("Stripped combat components from squad %d\n", squadID)
		}
	}
}
