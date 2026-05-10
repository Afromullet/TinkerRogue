package combatinput

import (
	"game_main/core/common"
	"game_main/tactical/combat/combatstate"
	"game_main/tactical/squads/squadcore"
	"game_main/visual/graphics"

	"github.com/bytearena/ecs"
)

// EnterDebugKillMode activates click-to-kill mode for debug purposes.
func (cih *CombatInputHandler) EnterDebugKillMode() {
	cih.inDebugKillMode = true
}

func (cih *CombatInputHandler) handleDebugKillClick(mouseX, mouseY int) {
	if cih.playerPos == nil {
		return
	}

	clickedPos := graphics.MouseToLogicalPosition(mouseX, mouseY, *cih.playerPos)
	clickedSquadID := combatstate.GetSquadAtPosition(clickedPos, cih.deps.Queries.ECSManager)

	if clickedSquadID == 0 {
		return
	}

	cih.actionHandler.DebugKillSquad(clickedSquadID)
}

func (cih *CombatInputHandler) killAllEnemySquads() {
	encounterID := cih.deps.Encounter.GetCurrentEncounterID()
	if encounterID == 0 {
		println("[DEBUG] No current encounter, cannot kill enemies")
		return
	}

	playerFactionID := cih.deps.Queries.GetPlayerFactionForEncounter(encounterID)
	if playerFactionID == 0 {
		println("[DEBUG] No player faction found, cannot kill enemies")
		return
	}

	enemySquads := cih.deps.Queries.GetEnemySquadsForEncounter(playerFactionID, encounterID)
	println("[DEBUG] Ctrl+K pressed - Found", len(enemySquads), "enemy squads in encounter", encounterID, "to kill")

	if len(enemySquads) == 0 {
		return
	}

	totalKilled := 0
	for _, squadID := range enemySquads {
		killed := cih.killAllUnitsInSquad(squadID)
		totalKilled += killed
		println("[DEBUG] Killed", killed, "units in squad", squadID)
	}

	println("[DEBUG] Total units killed:", totalKilled)
}

func (cih *CombatInputHandler) killAllUnitsInSquad(squadID ecs.EntityID) int {
	unitIDs := squadcore.GetUnitIDsInSquad(squadID, cih.deps.Queries.ECSManager)
	killed := 0

	for _, unitID := range unitIDs {
		unitEntity := cih.deps.Queries.ECSManager.FindEntityByID(unitID)
		if unitEntity == nil {
			continue
		}

		attr := common.GetComponentType[*common.Attributes](unitEntity, common.AttributeComponent)
		if attr != nil {
			oldHealth := attr.CurrentHealth
			attr.CurrentHealth = 0
			killed++
			println("[DEBUG]   Unit", unitID, "health:", oldHealth, "-> 0")
		}
	}

	return killed
}
