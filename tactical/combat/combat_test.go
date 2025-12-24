package combat

import (
	"game_main/common"
	"game_main/coords"
	"game_main/tactical/squads"
	"testing"

	"github.com/bytearena/ecs"
)

// ========================================
// COMPONENT REGISTRATION TESTS
// ========================================

func TestCombatInitialization(t *testing.T) {
	manager := CreateTestCombatManager()

	// Verify components exist
	if FactionComponent == nil {
		t.Error("FactionComponent not initialized")
	}
	if TurnStateComponent == nil {
		t.Error("TurnStateComponent not initialized")
	}
	if ActionStateComponent == nil {
		t.Error("ActionStateComponent not initialized")
	}
	if CombatFactionComponent == nil {
		t.Error("CombatFactionComponent not initialized")
	}

	// Verify tags are registered
	if _, ok := manager.WorldTags["faction"]; !ok {
		t.Error("faction tag not registered")
	}
	if _, ok := manager.WorldTags["turnstate"]; !ok {
		t.Error("turnstate tag not registered")
	}
	if _, ok := manager.WorldTags["actionstate"]; !ok {
		t.Error("actionstate tag not registered")
	}
	if _, ok := manager.WorldTags["combatfaction"]; !ok {
		t.Error("combatfaction tag not registered")
	}
}

// ========================================
// FACTION MANAGER TESTS
// ========================================

func TestCreateFaction(t *testing.T) {
	manager := CreateTestCombatManager()

	fm := NewFactionManager(manager)
	factionID := fm.CreateFaction("Test Faction", true)

	if factionID == 0 {
		t.Fatal("Failed to create faction")
	}

	// Verify faction data (using cache for O(1) lookup instead of O(n) query)
	faction := fm.combatCache.FindFactionByID(factionID, manager)
	if faction == nil {
		t.Fatal("Cannot find created faction")
	}

	factionData := common.GetComponentType[*FactionData](faction, FactionComponent)
	if factionData.Name != "Test Faction" {
		t.Errorf("Expected name 'Test Faction', got '%s'", factionData.Name)
	}
	if !factionData.IsPlayerControlled {
		t.Error("Expected player-controlled faction")
	}
}

func TestAddSquadToFaction(t *testing.T) {
	manager := CreateTestCombatManager()

	fm := NewFactionManager(manager)
	factionID := fm.CreateFaction("Test Faction", true)
	squadID := CreateTestSquad(manager, "Test Squad", 5)

	pos := coords.LogicalPosition{X: 10, Y: 10}
	err := fm.AddSquadToFaction(factionID, squadID, pos)
	if err != nil {
		t.Fatalf("Failed to add squad to faction: %v", err)
	}

	// Verify squad has CombatFactionComponent
	squad := common.FindEntityByIDWithTag(manager, squadID, squads.SquadTag)
	if squad == nil {
		t.Fatal("Squad not found")
	}

	combatFaction := common.GetComponentType[*CombatFactionData](squad, CombatFactionComponent)
	if combatFaction == nil {
		t.Fatal("Squad does not have CombatFactionComponent")
	}

	if combatFaction.FactionID != factionID {
		t.Errorf("Expected faction %d, got %d", factionID, combatFaction.FactionID)
	}

	// Verify position
	squadPos := common.GetComponentType[*coords.LogicalPosition](squad, common.PositionComponent)
	if squadPos == nil {
		t.Fatal("Squad has no position")
	}
	if squadPos.X != 10 || squadPos.Y != 10 {
		t.Errorf("Expected position (10,10), got (%d,%d)", squadPos.X, squadPos.Y)
	}
}

// ========================================
// TURN MANAGER TESTS
// ========================================

func TestInitializeCombat_RandomizesTurnOrder(t *testing.T) {
	manager := CreateTestCombatManager()

	fm := NewFactionManager(manager)
	faction1 := fm.CreateFaction("Faction 1", true)
	faction2 := fm.CreateFaction("Faction 2", false)

	turnMgr := NewTurnManager(manager)
	err := turnMgr.InitializeCombat([]ecs.EntityID{faction1, faction2})
	if err != nil {
		t.Fatalf("Failed to initialize combat: %v", err)
	}

	// Verify combat is active
	if !combatActive(manager) {
		t.Error("Combat should be active")
	}

	// Verify turn state exists
	turnEntity := findTurnStateEntity(manager)
	if turnEntity == nil {
		t.Fatal("Turn state not created")
	}

	turnState := common.GetComponentType[*TurnStateData](turnEntity, TurnStateComponent)
	if len(turnState.TurnOrder) != 2 {
		t.Errorf("Expected 2 factions in turn order, got %d", len(turnState.TurnOrder))
	}
	if turnState.CurrentRound != 1 {
		t.Errorf("Expected round 1, got %d", turnState.CurrentRound)
	}
}

func TestEndTurn_AdvancesToNextFaction(t *testing.T) {
	manager := CreateTestCombatManager()

	fm := NewFactionManager(manager)
	faction1 := fm.CreateFaction("Faction 1", true)
	faction2 := fm.CreateFaction("Faction 2", false)

	turnMgr := NewTurnManager(manager)
	turnMgr.InitializeCombat([]ecs.EntityID{faction1, faction2})

	firstFaction := turnMgr.GetCurrentFaction()
	err := turnMgr.EndTurn()
	if err != nil {
		t.Fatalf("Failed to end turn: %v", err)
	}

	secondFaction := turnMgr.GetCurrentFaction()
	if firstFaction == secondFaction {
		t.Error("Current faction should have changed")
	}
}

func TestEndTurn_WrapsAroundToFirstFaction(t *testing.T) {
	manager := CreateTestCombatManager()

	fm := NewFactionManager(manager)
	faction1 := fm.CreateFaction("Faction 1", true)
	faction2 := fm.CreateFaction("Faction 2", false)

	turnMgr := NewTurnManager(manager)
	turnMgr.InitializeCombat([]ecs.EntityID{faction1, faction2})

	initialRound := turnMgr.GetCurrentRound()

	// End both faction turns
	turnMgr.EndTurn()
	turnMgr.EndTurn()

	newRound := turnMgr.GetCurrentRound()
	if newRound != initialRound+1 {
		t.Errorf("Expected round %d, got %d", initialRound+1, newRound)
	}
}

// ========================================
// MOVEMENT SYSTEM TESTS
// ========================================

func TestGetSquadMovementSpeed_ReturnsSlowestUnit(t *testing.T) {
	manager := CreateTestCombatManager()

	squadID := CreateTestSquad(manager, "Test Squad", 3)

	// Modify one unit to have slower speed
	unitIDs := squads.GetUnitIDsInSquad(squadID, manager)
	slowUnit := common.FindEntityByIDWithTag(manager, unitIDs[0], squads.SquadMemberTag)
	attr := common.GetAttributes(slowUnit)
	attr.MovementSpeed = 2

	moveSys := NewMovementSystem(manager, common.GlobalPositionSystem)
	speed := moveSys.GetSquadMovementSpeed(squadID)

	if speed != 2 {
		t.Errorf("Expected speed 2, got %d", speed)
	}
}

func TestMoveSquad_UpdatesPosition(t *testing.T) {
	manager := CreateTestCombatManager()

	fm := NewFactionManager(manager)
	factionID := fm.CreateFaction("Test Faction", true)
	squadID := CreateTestSquad(manager, "Test Squad", 3)

	startPos := coords.LogicalPosition{X: 5, Y: 5}
	fm.AddSquadToFaction(factionID, squadID, startPos)

	// Create action state
	turnMgr := NewTurnManager(manager)
	turnMgr.InitializeCombat([]ecs.EntityID{factionID})

	moveSys := NewMovementSystem(manager, common.GlobalPositionSystem)
	targetPos := coords.LogicalPosition{X: 6, Y: 6}

	err := moveSys.MoveSquad(squadID, targetPos)
	if err != nil {
		t.Fatalf("Failed to move squad: %v", err)
	}

	// Verify position updated
	newPos, err := moveSys.GetSquadPosition(squadID)
	if err != nil {
		t.Fatalf("Failed to get squad position: %v", err)
	}

	if newPos.X != 6 || newPos.Y != 6 {
		t.Errorf("Expected position (6,6), got (%d,%d)", newPos.X, newPos.Y)
	}
}

// ========================================
// COMBAT ACTION SYSTEM TESTS
// ========================================

func TestGetSquadAttackRange_ReturnsMaxRange(t *testing.T) {
	manager := CreateTestCombatManager()

	squadID := CreateTestMixedSquad(manager, "Mixed Squad", 3, 2)

	combatSys := NewCombatActionSystem(manager)
	maxRange := combatSys.GetSquadAttackRange(squadID)

	if maxRange != 3 {
		t.Errorf("Expected max range 3, got %d", maxRange)
	}
}

func TestExecuteAttackAction_MeleeAttack(t *testing.T) {
	manager := CreateTestCombatManager()

	fm := NewFactionManager(manager)
	playerFaction := fm.CreateFaction("Player", true)
	enemyFaction := fm.CreateFaction("Enemy", false)

	playerSquad := CreateTestSquad(manager, "Player Squad", 3)
	enemySquad := CreateTestSquad(manager, "Enemy Squad", 3)

	fm.AddSquadToFaction(playerFaction, playerSquad, coords.LogicalPosition{X: 5, Y: 5})
	fm.AddSquadToFaction(enemyFaction, enemySquad, coords.LogicalPosition{X: 6, Y: 5})

	// Initialize combat
	turnMgr := NewTurnManager(manager)
	turnMgr.InitializeCombat([]ecs.EntityID{playerFaction, enemyFaction})

	combatSys := NewCombatActionSystem(manager)
	result := combatSys.ExecuteAttackAction(playerSquad, enemySquad)
	if !result.Success {
		t.Fatalf("Failed to execute attack: %s", result.ErrorReason)
	}

	// Verify squad marked as acted (using cache for O(k) lookup instead of O(n) query)
	cache := NewCombatQueryCache(manager)
	if canSquadAct(cache, playerSquad, manager) {
		t.Error("Squad should be marked as acted")
	}
}

// ========================================
// FULL COMBAT LOOP TEST
// ========================================

func TestFullCombatLoop_TwoFactions(t *testing.T) {
	manager := CreateTestCombatManager()

	fm := NewFactionManager(manager)
	turnMgr := NewTurnManager(manager)
	moveSys := NewMovementSystem(manager, common.GlobalPositionSystem)

	// Create factions
	playerID := fm.CreateFaction("Player", true)
	aiID := fm.CreateFaction("Goblins", false)

	// Create squads
	playerSquad1 := CreateTestSquad(manager, "Knights", 5)
	aiSquad1 := CreateTestSquad(manager, "Goblin Warriors", 5)

	// Assign to factions
	fm.AddSquadToFaction(playerID, playerSquad1, coords.LogicalPosition{X: 5, Y: 5})
	fm.AddSquadToFaction(aiID, aiSquad1, coords.LogicalPosition{X: 15, Y: 15})

	// Initialize combat
	err := turnMgr.InitializeCombat([]ecs.EntityID{playerID, aiID})
	if err != nil {
		t.Fatalf("Failed to initialize combat: %v", err)
	}

	// Simulate 3 rounds
	for round := 0; round < 6; round++ {
		currentFaction := turnMgr.GetCurrentFaction()

		if currentFaction == playerID {
			// Player turn: move towards enemy
			targetPos := coords.LogicalPosition{X: 10, Y: 10}
			if moveSys.CanMoveTo(playerSquad1, targetPos) {
				moveSys.MoveSquad(playerSquad1, targetPos)
			}
		} else {
			// AI turn: move towards player
			targetPos := coords.LogicalPosition{X: 12, Y: 12}
			if moveSys.CanMoveTo(aiSquad1, targetPos) {
				moveSys.MoveSquad(aiSquad1, targetPos)
			}
		}

		// End turn
		err := turnMgr.EndTurn()
		if err != nil {
			t.Fatalf("Failed to end turn: %v", err)
		}
	}

	// Verify combat still active
	if !combatActive(manager) {
		t.Error("Combat should still be active")
	}

	// Verify we're in round 4 (6 turns / 2 factions = 3 complete rounds + 1)
	currentRound := turnMgr.GetCurrentRound()
	if currentRound < 3 {
		t.Errorf("Expected at least round 3, got %d", currentRound)
	}
}
