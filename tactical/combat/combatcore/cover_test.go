package combatcore

import (
	"game_main/core/common"
	"game_main/tactical/combat/combatmath"
	"game_main/tactical/squads/squadcore"
	"testing"
)
// ========================================
// COVER SYSTEM TESTS
// ========================================

// ========================================
// GetCoverProvidersFor TESTS
// ========================================

func TestGetCoverProvidersFor_NoProviders(t *testing.T) {
	manager := setupCombatTestManager(t)

	squadID := createTestSquad(manager, "TestSquad")
	defender := createTestUnit(manager, squadID, 1, 0, 100, 10, 0)
	defenderPos := common.GetComponentType[*squadcore.GridPositionData](defender, squadcore.GridPositionComponent)

	providers := combatmath.GetCoverProvidersFor(defender.GetID(), squadID, defenderPos, manager)

	if len(providers) != 0 {
		t.Errorf("Expected 0 providers, got %d", len(providers))
	}
}

func TestGetCoverProvidersFor_SingleProvider(t *testing.T) {
	manager := setupCombatTestManager(t)

	squadID := createTestSquad(manager, "TestSquad")

	frontLine := createTestUnit(manager, squadID, 0, 0, 100, 10, 0)
	frontLine.AddComponent(squadcore.CoverComponent, &squadcore.CoverData{
		CoverValue:     0.25,
		CoverRange:     1,
		RequiresActive: true,
	})

	backLine := createTestUnit(manager, squadID, 1, 0, 100, 10, 0)
	backLinePos := common.GetComponentType[*squadcore.GridPositionData](backLine, squadcore.GridPositionComponent)

	providers := combatmath.GetCoverProvidersFor(backLine.GetID(), squadID, backLinePos, manager)

	if len(providers) != 1 {
		t.Errorf("Expected 1 provider, got %d", len(providers))
	}

	if len(providers) > 0 && providers[0] != frontLine.GetID() {
		t.Error("Expected front-line unit to be the provider")
	}
}

func TestGetCoverProvidersFor_MultipleProviders(t *testing.T) {
	manager := setupCombatTestManager(t)

	squadID := createTestSquad(manager, "TestSquad")

	// Two front-line units in same column
	frontLine1 := createTestUnit(manager, squadID, 0, 0, 100, 10, 0)
	frontLine1.AddComponent(squadcore.CoverComponent, &squadcore.CoverData{
		CoverValue:     0.15,
		CoverRange:     2,
		RequiresActive: true,
	})

	midLine := createTestUnit(manager, squadID, 1, 0, 100, 10, 0)
	midLine.AddComponent(squadcore.CoverComponent, &squadcore.CoverData{
		CoverValue:     0.10,
		CoverRange:     1,
		RequiresActive: true,
	})

	backLine := createTestUnit(manager, squadID, 2, 0, 100, 10, 0)
	backLinePos := common.GetComponentType[*squadcore.GridPositionData](backLine, squadcore.GridPositionComponent)

	providers := combatmath.GetCoverProvidersFor(backLine.GetID(), squadID, backLinePos, manager)

	if len(providers) != 2 {
		t.Errorf("Expected 2 providers, got %d", len(providers))
	}
}

func TestGetCoverProvidersFor_DoesNotIncludeSelf(t *testing.T) {
	manager := setupCombatTestManager(t)

	squadID := createTestSquad(manager, "TestSquad")

	unit := createTestUnit(manager, squadID, 0, 0, 100, 10, 0)
	unit.AddComponent(squadcore.CoverComponent, &squadcore.CoverData{
		CoverValue:     0.25,
		CoverRange:     1,
		RequiresActive: true,
	})
	unitPos := common.GetComponentType[*squadcore.GridPositionData](unit, squadcore.GridPositionComponent)

	providers := combatmath.GetCoverProvidersFor(unit.GetID(), squadID, unitPos, manager)

	if len(providers) != 0 {
		t.Errorf("Expected 0 providers (unit should not provide cover to itself), got %d", len(providers))
	}
}

func TestGetCoverProvidersFor_OnlyFromSameSquad(t *testing.T) {
	manager := setupCombatTestManager(t)

	// Squad 1
	squad1ID := createTestSquad(manager, "Squad1")
	squad1Unit := createTestUnit(manager, squad1ID, 0, 0, 100, 10, 0)
	squad1Unit.AddComponent(squadcore.CoverComponent, &squadcore.CoverData{
		CoverValue:     0.25,
		CoverRange:     1,
		RequiresActive: true,
	})

	// Squad 2
	squad2ID := createTestSquad(manager, "Squad2")
	squad2Unit := createTestUnit(manager, squad2ID, 1, 0, 100, 10, 0)
	squad2UnitPos := common.GetComponentType[*squadcore.GridPositionData](squad2Unit, squadcore.GridPositionComponent)

	// Squad 2 unit should not get cover from Squad 1
	providers := combatmath.GetCoverProvidersFor(squad2Unit.GetID(), squad2ID, squad2UnitPos, manager)

	if len(providers) != 0 {
		t.Errorf("Expected 0 providers from different squad, got %d", len(providers))
	}
}

