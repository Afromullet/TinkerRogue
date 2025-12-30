package combat

import (
	"fmt"
	"game_main/common"
	"game_main/tactical/squads"
	testfx "game_main/testing"
	"game_main/world/coords"

	"github.com/bytearena/ecs"
)

// ========================================
// TEST SETUP FIXTURES (Using shared testing fixtures)
// ========================================

// CreateTestCombatManager creates a fully initialized EntityManager with combat system.
// This is the combat-specific version of testing.NewTestEntityManager() that also
// initializes the combat system components and tags.
//
// Replaces the old setupTestManager() local function.
func CreateTestCombatManager() *common.EntityManager {
	manager := testfx.NewTestEntityManager()
	// Initialize squad system (required by combat tests)
	if err := squads.InitializeSquadData(manager); err != nil {
		panic(fmt.Sprintf("Failed to initialize squad data: %v", err))
	}
	// Initialize combat system
	InitializeCombatSystem(manager)
	return manager
}

// ========================================
// TEST HELPER FUNCTIONS
// ========================================

// CreateTestFaction creates a faction for testing
func CreateTestFaction(manager *common.EntityManager, name string, isPlayer bool) ecs.EntityID {
	fm := NewFactionManager(manager)
	return fm.CreateFaction(name, isPlayer)
}

// CreateTestSquad creates a squad with test units
func CreateTestSquad(manager *common.EntityManager, name string, unitCount int) ecs.EntityID {
	// Create squad entity
	squadEntity := manager.World.NewEntity()
	squadID := squadEntity.GetID()

	squadEntity.AddComponent(squads.SquadComponent, &squads.SquadData{
		SquadID:   squadID,
		Name:      name,
		Formation: squads.FormationBalanced,
		MaxUnits:  9,
	})

	// Create test units
	for i := 0; i < unitCount; i++ {
		CreateTestUnit(manager, squadID, i)
	}

	return squadID
}

// CreateTestUnit creates a unit entity with default attributes
func CreateTestUnit(manager *common.EntityManager, squadID ecs.EntityID, index int) ecs.EntityID {
	unitEntity := manager.World.NewEntity()
	unitID := unitEntity.GetID()

	// Add attributes
	unitEntity.AddComponent(common.AttributeComponent, &common.Attributes{
		Strength:      10,
		Dexterity:     10,
		Magic:         0,
		Leadership:    0,
		Armor:         2,
		Weapon:        2,
		MovementSpeed: 5,
		AttackRange:   1,
		CurrentHealth: 30,
		MaxHealth:     30,
		CanAct:        true,
	})

	// Add squad membership
	unitEntity.AddComponent(squads.SquadMemberComponent, &squads.SquadMemberData{
		SquadID: squadID,
	})

	// Position in 3x3 grid
	row := index / 3
	col := index % 3
	unitEntity.AddComponent(squads.GridPositionComponent, &squads.GridPositionData{
		AnchorRow: row,
		AnchorCol: col,
		Width:     1,
		Height:    1,
	})

	// Add attack range component (matches production code in units.go)
	unitEntity.AddComponent(squads.AttackRangeComponent, &squads.AttackRangeData{
		Range: 1, // Melee range
	})

	return unitID
}

// CreateTestRangedUnit creates a ranged unit entity
func CreateTestRangedUnit(manager *common.EntityManager, squadID ecs.EntityID, index int, attackRange int) ecs.EntityID {
	unitEntity := manager.World.NewEntity()
	unitID := unitEntity.GetID()

	// Add attributes
	unitEntity.AddComponent(common.AttributeComponent, &common.Attributes{
		Strength:      8,
		Dexterity:     12,
		Magic:         0,
		Leadership:    0,
		Armor:         1,
		Weapon:        2,
		MovementSpeed: 4,
		AttackRange:   attackRange,
		CurrentHealth: 25,
		MaxHealth:     25,
		CanAct:        true,
	})

	// Add squad membership
	unitEntity.AddComponent(squads.SquadMemberComponent, &squads.SquadMemberData{
		SquadID: squadID,
	})

	// Position in 3x3 grid
	row := index / 3
	col := index % 3
	unitEntity.AddComponent(squads.GridPositionComponent, &squads.GridPositionData{
		AnchorRow: row,
		AnchorCol: col,
		Width:     1,
		Height:    1,
	})

	// Add attack range component (matches production code in units.go)
	unitEntity.AddComponent(squads.AttackRangeComponent, &squads.AttackRangeData{
		Range: attackRange,
	})

	return unitID
}

// CreateTestMixedSquad creates a squad with melee and ranged units
func CreateTestMixedSquad(manager *common.EntityManager, name string, meleeCount, rangedCount int) ecs.EntityID {
	// Create squad entity
	squadEntity := manager.World.NewEntity()
	squadID := squadEntity.GetID()

	squadEntity.AddComponent(squads.SquadComponent, &squads.SquadData{
		SquadID:   squadID,
		Name:      name,
		Formation: squads.FormationBalanced,
		MaxUnits:  9,
	})

	// Create melee units
	for i := 0; i < meleeCount; i++ {
		CreateTestUnit(manager, squadID, i)
	}

	// Create ranged units
	for i := 0; i < rangedCount; i++ {
		CreateTestRangedUnit(manager, squadID, meleeCount+i, 3) // Range 3
	}

	return squadID
}

// PlaceSquadOnMap places a squad at a position for testing (using new CombatFactionComponent approach)
func PlaceSquadOnMap(manager *common.EntityManager, factionID, squadID ecs.EntityID, pos coords.LogicalPosition) {
	squadEntity := manager.FindEntityByID(squadID)
	if squadEntity == nil {
		return
	}

	// Add CombatFactionComponent to squad (squad enters combat)
	squadEntity.AddComponent(CombatFactionComponent, &CombatFactionData{
		FactionID: factionID,
	})

	// Add or update PositionComponent on squad entity
	if !manager.HasComponent(squadID, common.PositionComponent) {
		// Squad has no position yet - add it
		posPtr := new(coords.LogicalPosition)
		*posPtr = pos
		squadEntity.AddComponent(common.PositionComponent, posPtr)
		// Register in PositionSystem (canonical position source)
		common.GlobalPositionSystem.AddEntity(squadID, pos)
	} else {
		// Squad already has position - move it atomically
		oldPos := common.GetComponentTypeByID[*coords.LogicalPosition](manager, squadID, common.PositionComponent)
		if oldPos != nil {
			// Use MoveEntity to synchronize position component and position system
			manager.MoveEntity(squadID, squadEntity, *oldPos, pos)
		}
	}
}

// InitializeTestCombat sets up a basic test combat scenario
func InitializeTestCombat(manager *common.EntityManager, factionCount int) ([]ecs.EntityID, *TurnManager) {
	// Create factions
	factionIDs := make([]ecs.EntityID, factionCount)
	fm := NewFactionManager(manager)

	for i := 0; i < factionCount; i++ {
		if i == 0 {
			factionIDs[i] = fm.CreateFaction("Player", true)
		} else {
			factionIDs[i] = fm.CreateFaction(fmt.Sprintf("Enemy %d", i), false)
		}
	}

	// Initialize turn manager
	turnMgr := NewTurnManager(manager)
	turnMgr.InitializeCombat(factionIDs)

	return factionIDs, turnMgr
}
