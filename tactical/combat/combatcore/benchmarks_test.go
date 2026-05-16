package combatcore

import (
	"testing"
)
// BENCHMARK TESTS
// ========================================

func BenchmarkExecuteSquadAttack_SingleVsSingle(b *testing.B) {
	manager := setupCombatTestManager(&testing.T{})

	attackerSquadID := createTestSquad(manager, "Attackers")
	createTestUnit(manager, attackerSquadID, 0, 0, 100, 20, 100)

	defenderSquadID := createTestSquad(manager, "Defenders")
	createTestUnit(manager, defenderSquadID, 0, 0, 100, 10, 0)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		executeTestAttack(attackerSquadID, defenderSquadID, manager)
	}
}

func BenchmarkExecuteSquadAttack_FullSquadVsFullSquad(b *testing.B) {
	manager := setupCombatTestManager(&testing.T{})

	// Create full squads (9 units each)
	attackerSquadID := createTestSquad(manager, "Attackers")
	for row := 0; row < 3; row++ {
		for col := 0; col < 3; col++ {
			createTestUnit(manager, attackerSquadID, row, col, 100, 20, 100)
		}
	}

	defenderSquadID := createTestSquad(manager, "Defenders")
	for row := 0; row < 3; row++ {
		for col := 0; col < 3; col++ {
			createTestUnit(manager, defenderSquadID, row, col, 100, 10, 0)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		executeTestAttack(attackerSquadID, defenderSquadID, manager)
	}
}
