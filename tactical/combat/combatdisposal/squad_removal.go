// Package combatdisposal owns the squad-removal lifecycle operation. It lives
// at a leaf position in the import graph so callers across combat (combatcore,
// GUI handlers, spells) can reach it without inducing cycles — putting this
// function in combatstate (a state/query package) or combatlifecycle (which
// imports spells) would violate layering.
package combatdisposal

import (
	"fmt"

	"game_main/core/common"
	"game_main/core/coords"
	"game_main/tactical/combat/combatstate"
	"game_main/tactical/squads/squadcore"

	"github.com/bytearena/ecs"
)

// RemoveSquadFromMap detaches a squad from the tactical map and disposes the
// squad entity along with all its units. Called when a squad is destroyed in
// combat, surrenders via the GUI, or is wiped by a spell.
//
// Three steps:
//  1. Remove the squad's position from GlobalPositionSystem.
//  2. Strip FactionMembershipComponent (squad exits combat).
//  3. Dispose the squad entity and every member unit via squadcore.DisposeSquadAndUnits.
func RemoveSquadFromMap(squadID ecs.EntityID, manager *common.EntityManager) error {
	squad := manager.FindEntityByID(squadID)
	if squad == nil {
		return fmt.Errorf("squad %d not found", squadID)
	}

	if position := common.GetComponentType[*coords.LogicalPosition](squad, common.PositionComponent); position != nil {
		common.GlobalPositionSystem.RemoveEntity(squadID, *position)
	}

	combatstate.RemoveCombatMembership(squad)
	squadcore.DisposeSquadAndUnits(squadID, manager)

	return nil
}
