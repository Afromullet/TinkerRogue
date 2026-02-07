package main

import (
	"fmt"
	"game_main/common"
	"game_main/tactical/combat"
	"game_main/tactical/combat/battlelog"
	"game_main/tactical/combatservices"
	"game_main/tactical/squads"
	"game_main/world/coords"

	"github.com/bytearena/ecs"
)

const maxRounds = 100

// RunBattle executes a full combat simulation between two sides.
// Both sides are AI-controlled. No movement - squads are adjacent and trade blows.
// Returns the completed BattleRecord for export.
func RunBattle(manager *common.EntityManager, sideASquadIDs, sideBSquadIDs []ecs.EntityID) *battlelog.BattleRecord {
	// 1. Create CombatService
	combatService := combatservices.NewCombatService(manager)

	// 2. Create factions (both AI-controlled)
	factionA := combatService.FactionManager.CreateCombatFaction("Side A", false)
	factionB := combatService.FactionManager.CreateCombatFaction("Side B", false)

	// 3. Assign squads to factions
	posA := coords.LogicalPosition{X: 10, Y: 10}
	posB := coords.LogicalPosition{X: 11, Y: 10}

	for _, id := range sideASquadIDs {
		if err := combatService.FactionManager.AddSquadToFaction(factionA, id, posA); err != nil {
			fmt.Printf("  WARNING: Failed to add squad %d to Side A: %v\n", id, err)
		}
	}
	for _, id := range sideBSquadIDs {
		if err := combatService.FactionManager.AddSquadToFaction(factionB, id, posB); err != nil {
			fmt.Printf("  WARNING: Failed to add squad %d to Side B: %v\n", id, err)
		}
	}

	// 4. Initialize combat (creates turn order, action states)
	factionIDs := []ecs.EntityID{factionA, factionB}
	if err := combatService.InitializeCombat(factionIDs); err != nil {
		fmt.Printf("  ERROR: Failed to initialize combat: %v\n", err)
		return nil
	}

	// 5. Enable battle recorder
	combatService.BattleRecorder.SetEnabled(true)
	combatService.BattleRecorder.Start()

	// 6. Combat loop
	var victory *combatservices.VictoryCheckResult

	for round := 0; round < maxRounds; round++ {
		combatService.BattleRecorder.SetCurrentRound(combatService.TurnManager.GetCurrentRound())
		currentFaction := combatService.TurnManager.GetCurrentFaction()
		if currentFaction == 0 {
			break
		}

		// Determine enemy faction
		enemyFaction := factionB
		if currentFaction == factionB {
			enemyFaction = factionA
		}

		// Get alive squads for current faction
		aliveSquads := combatService.GetAliveSquadsInFaction(currentFaction)

		for _, squadID := range aliveSquads {
			// Find best enemy target
			targetID := selectBestTarget(squadID, enemyFaction, combatService, manager)
			if targetID == 0 {
				continue
			}

			// Execute attack
			combatService.CombatActSystem.ExecuteAttackAction(squadID, targetID)
		}

		// Check victory
		victory = combatService.CheckVictoryCondition()
		if victory.BattleOver {
			break
		}

		// End turn (advances to next faction, resets action states)
		if err := combatService.TurnManager.EndTurn(); err != nil {
			fmt.Printf("  WARNING: EndTurn error: %v\n", err)
			break
		}
	}

	// 7. Finalize and return battle record
	if victory == nil {
		victory = combatService.CheckVictoryCondition()
	}

	victorInfo := &battlelog.VictoryInfo{
		RoundsCompleted: victory.RoundsCompleted,
		VictorFaction:   victory.VictorFaction,
		VictorName:      victory.VictorName,
	}

	record := combatService.BattleRecorder.Finalize(victorInfo)
	return record
}

// selectBestTarget picks the best enemy squad to attack.
// Priority: most damaged first (focus fire), then smallest squad.
func selectBestTarget(attackerID ecs.EntityID, enemyFactionID ecs.EntityID, cs *combatservices.CombatService, manager *common.EntityManager) ecs.EntityID {
	enemySquads := combat.GetActiveSquadsForFaction(enemyFactionID, manager)
	if len(enemySquads) == 0 {
		return 0
	}

	bestTarget := ecs.EntityID(0)
	lowestHP := 2.0 // HP percent is 0.0-1.0, so 2.0 means "not set"

	for _, enemyID := range enemySquads {
		if squads.IsSquadDestroyed(enemyID, manager) {
			continue
		}

		hp := squads.GetSquadHealthPercent(enemyID, manager)

		// Prefer most damaged (lowest HP %)
		if hp < lowestHP {
			lowestHP = hp
			bestTarget = enemyID
		}
	}

	return bestTarget
}
