package combatcore

import (
	"game_main/common"
	"game_main/tactical/combat/combatstate"
	"game_main/tactical/squads/squadcore"
	"game_main/world/coords"
	"testing"

	"github.com/bytearena/ecs"
)

// ========================================
// COMPONENT REGISTRATION TESTS
// ========================================

func TestCombatInitialization(t *testing.T) {
	manager := CreateTestCombatManager()

	// Verify components exist
	if combatstate.CombatFactionComponent == nil {
		t.Error("FactionComponent not initialized")
	}
	if combatstate.TurnStateComponent == nil {
		t.Error("TurnStateComponent not initialized")
	}
	if combatstate.ActionStateComponent == nil {
		t.Error("ActionStateComponent not initialized")
	}
	if combatstate.FactionMembershipComponent == nil {
		t.Error("FactionMembershipComponent not initialized")
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

	cache := combatstate.NewCombatQueryCache(manager)
	fm := combatstate.NewCombatFactionManager(manager, cache)
	factionID := fm.CreateCombatFaction("Test Faction", true)

	if factionID == 0 {
		t.Fatal("Failed to create faction")
	}

	// Verify faction data (using cache for O(1) lookup instead of O(n) query)
	faction := cache.FindFactionByID(factionID)
	if faction == nil {
		t.Fatal("Cannot find created faction")
	}

	factionData := common.GetComponentType[*combatstate.FactionData](faction, combatstate.CombatFactionComponent)
	if factionData.Name != "Test Faction" {
		t.Errorf("Expected name 'Test Faction', got '%s'", factionData.Name)
	}
	if !factionData.IsPlayerControlled {
		t.Error("Expected player-controlled faction")
	}
}

func TestAddSquadToFaction(t *testing.T) {
	manager := CreateTestCombatManager()

	cache := combatstate.NewCombatQueryCache(manager)
	fm := combatstate.NewCombatFactionManager(manager, cache)
	factionID := fm.CreateCombatFaction("Test Faction", true)
	squadID := CreateTestSquad(manager, "Test Squad", 5)

	pos := coords.LogicalPosition{X: 10, Y: 10}
	err := fm.AddSquadToFaction(factionID, squadID, pos)
	if err != nil {
		t.Fatalf("Failed to add squad to faction: %v", err)
	}

	// Verify squad has FactionMembershipComponent
	squad := manager.FindEntityByID(squadID)
	if squad == nil {
		t.Fatal("Squad not found")
	}

	combatFaction := common.GetComponentType[*combatstate.CombatFactionData](squad, combatstate.FactionMembershipComponent)
	if combatFaction == nil {
		t.Fatal("Squad does not have FactionMembershipComponent")
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

func TestEndTurn_AdvancesToNextFaction(t *testing.T) {
	manager := CreateTestCombatManager()

	cache := combatstate.NewCombatQueryCache(manager)
	fm := combatstate.NewCombatFactionManager(manager, cache)
	faction1 := fm.CreateCombatFaction("Faction 1", true)
	faction2 := fm.CreateCombatFaction("Faction 2", false)

	turnMgr := NewTurnManager(manager, cache)
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

	cache := combatstate.NewCombatQueryCache(manager)
	fm := combatstate.NewCombatFactionManager(manager, cache)
	faction1 := fm.CreateCombatFaction("Faction 1", true)
	faction2 := fm.CreateCombatFaction("Faction 2", false)

	turnMgr := NewTurnManager(manager, cache)
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
	unitIDs := squadcore.GetUnitIDsInSquad(squadID, manager)
	slowUnit := manager.FindEntityByID(unitIDs[0])
	speedData := common.GetComponentType[*squadcore.MovementSpeedData](slowUnit, squadcore.MovementSpeedComponent)
	speedData.Speed = 2

	cache := combatstate.NewCombatQueryCache(manager)
	moveSys := NewMovementSystem(manager, common.GlobalPositionSystem, cache)
	speed := moveSys.GetSquadMovementSpeed(squadID)

	if speed != 2 {
		t.Errorf("Expected speed 2, got %d", speed)
	}
}

func TestMoveSquad_UpdatesPosition(t *testing.T) {
	manager := CreateTestCombatManager()

	cache := combatstate.NewCombatQueryCache(manager)
	fm := combatstate.NewCombatFactionManager(manager, cache)
	factionID := fm.CreateCombatFaction("Test Faction", true)
	squadID := CreateTestSquad(manager, "Test Squad", 3)

	startPos := coords.LogicalPosition{X: 5, Y: 5}
	fm.AddSquadToFaction(factionID, squadID, startPos)

	// Create action state
	turnMgr := NewTurnManager(manager, cache)
	turnMgr.InitializeCombat([]ecs.EntityID{factionID})

	moveSys := NewMovementSystem(manager, common.GlobalPositionSystem, cache)
	targetPos := coords.LogicalPosition{X: 6, Y: 6}

	err := moveSys.MoveSquad(squadID, targetPos)
	if err != nil {
		t.Fatalf("Failed to move squad: %v", err)
	}

	// Verify position updated
	newPos, err := combatstate.GetSquadMapPosition(squadID, manager)
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

	cache := combatstate.NewCombatQueryCache(manager)
	combatSys := NewCombatActionSystem(manager, cache)
	maxRange := combatSys.getSquadAttackRange(squadID)

	if maxRange != 3 {
		t.Errorf("Expected max range 3, got %d", maxRange)
	}
}

func TestExecuteAttackAction_MeleeAttack(t *testing.T) {
	manager := CreateTestCombatManager()

	cache := combatstate.NewCombatQueryCache(manager)
	fm := combatstate.NewCombatFactionManager(manager, cache)
	playerFaction := fm.CreateCombatFaction("Player", true)
	enemyFaction := fm.CreateCombatFaction("Enemy", false)

	playerSquad := CreateTestSquad(manager, "Player Squad", 3)
	enemySquad := CreateTestSquad(manager, "Enemy Squad", 3)

	fm.AddSquadToFaction(playerFaction, playerSquad, coords.LogicalPosition{X: 5, Y: 5})
	fm.AddSquadToFaction(enemyFaction, enemySquad, coords.LogicalPosition{X: 6, Y: 5})

	// Initialize combat
	turnMgr := NewTurnManager(manager, cache)
	turnMgr.InitializeCombat([]ecs.EntityID{playerFaction, enemyFaction})

	combatSys := NewCombatActionSystem(manager, cache)
	result := combatSys.ExecuteAttackAction(playerSquad, enemySquad)
	if !result.Success {
		t.Fatalf("Failed to execute attack: %s", result.ErrorReason)
	}

	// Verify squad marked as acted (using cache for O(k) lookup instead of O(n) query)
	if combatstate.CanSquadAct(cache, playerSquad, manager) {
		t.Error("Squad should be marked as acted")
	}
}

func TestCounterattack_DamagePredictionPreventsDeadUnits(t *testing.T) {
	manager := CreateTestCombatManager()

	cache := combatstate.NewCombatQueryCache(manager)
	fm := combatstate.NewCombatFactionManager(manager, cache)
	playerFaction := fm.CreateCombatFaction("Player", true)
	enemyFaction := fm.CreateCombatFaction("Enemy", false)

	playerSquad := CreateTestSquad(manager, "Player Squad", 3)
	enemySquad := CreateTestSquad(manager, "Enemy Squad", 3)

	fm.AddSquadToFaction(playerFaction, playerSquad, coords.LogicalPosition{X: 5, Y: 5})
	fm.AddSquadToFaction(enemyFaction, enemySquad, coords.LogicalPosition{X: 6, Y: 5})

	turnMgr := NewTurnManager(manager, cache)
	turnMgr.InitializeCombat([]ecs.EntityID{playerFaction, enemyFaction})

	combatSys := NewCombatActionSystem(manager, cache)
	result := combatSys.ExecuteAttackAction(playerSquad, enemySquad)
	if !result.Success {
		t.Fatalf("Failed to execute attack: %s", result.ErrorReason)
	}

	// Verify that counterattack events only come from units that would survive
	if result.CombatLog != nil {
		for _, event := range result.CombatLog.AttackEvents {
			if !event.IsCounterattack {
				continue
			}
			// The counterattacker should have predicted HP > 0 after main attack damage
			counterAttackerID := event.AttackerID
			dmgFromMainAttack := result.DamageByUnit[counterAttackerID]
			entity := manager.FindEntityByID(counterAttackerID)
			if entity == nil {
				continue // Entity already disposed
			}
			attr := common.GetComponentType[*common.Attributes](entity, common.AttributeComponent)
			if attr == nil {
				continue
			}
			// After damage is applied, current health reflects post-combat state.
			// But the original HP was MaxHealth (30 from test fixtures).
			// A valid counterattacker must have had (originalHP - mainAttackDmg) > 0.
			originalHP := attr.GetMaxHealth()
			if originalHP-dmgFromMainAttack <= 0 {
				t.Errorf("Unit %d counterattacked but would have died from main attack (hp=%d, damage=%d)",
					counterAttackerID, originalHP, dmgFromMainAttack)
			}
		}
	}
}

// ========================================
// FULL COMBAT LOOP TEST
// ========================================

func TestFullCombatLoop_TwoFactions(t *testing.T) {
	manager := CreateTestCombatManager()

	cache := combatstate.NewCombatQueryCache(manager)
	fm := combatstate.NewCombatFactionManager(manager, cache)
	turnMgr := NewTurnManager(manager, cache)
	moveSys := NewMovementSystem(manager, common.GlobalPositionSystem, cache)

	// Create factions
	playerID := fm.CreateCombatFaction("Player", true)
	aiID := fm.CreateCombatFaction("Goblins", false)

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

	// Verify we're in round 4 (6 turns / 2 factions = 3 complete rounds + 1)
	currentRound := turnMgr.GetCurrentRound()
	if currentRound < 3 {
		t.Errorf("Expected at least round 3, got %d", currentRound)
	}
}

// ========================================
// RESET / DOUBLE TIME TESTS
// ========================================

func TestResetSquadActions_ClearsBonusAttackActive(t *testing.T) {
	manager := CreateTestCombatManager()
	cache := combatstate.NewCombatQueryCache(manager)
	fm := combatstate.NewCombatFactionManager(manager, cache)
	factionID := fm.CreateCombatFaction("Player", true)
	squadID := CreateTestSquad(manager, "Test Squad", 3)

	fm.AddSquadToFaction(factionID, squadID, coords.LogicalPosition{X: 5, Y: 5})

	turnMgr := NewTurnManager(manager, cache)
	turnMgr.InitializeCombat([]ecs.EntityID{factionID})

	// Set BonusAttackActive to true manually
	actionState := cache.FindActionStateBySquadID(squadID)
	if actionState == nil {
		t.Fatal("Expected action state for squad")
	}
	actionState.BonusAttackActive = true

	// Reset squad actions
	turnMgr.ResetSquadActions(factionID)

	// Verify flag is cleared
	actionState = cache.FindActionStateBySquadID(squadID)
	if actionState.BonusAttackActive {
		t.Error("Expected BonusAttackActive to be false after ResetSquadActions")
	}
}

// ========================================
// BONUS ATTACK FLAG TEST
// ========================================

func TestBonusAttackFlag_ConsumesWithoutMarkingActed(t *testing.T) {
	manager := CreateTestCombatManager()
	cache := combatstate.NewCombatQueryCache(manager)
	fm := combatstate.NewCombatFactionManager(manager, cache)
	factionID := fm.CreateCombatFaction("Player", true)
	squadID := CreateTestSquad(manager, "Test Squad", 3)

	fm.AddSquadToFaction(factionID, squadID, coords.LogicalPosition{X: 5, Y: 5})

	turnMgr := NewTurnManager(manager, cache)
	turnMgr.InitializeCombat([]ecs.EntityID{factionID})

	// Set BonusAttackActive flag
	actionState := cache.FindActionStateBySquadID(squadID)
	if actionState == nil {
		t.Fatal("Expected action state for squad")
	}
	actionState.BonusAttackActive = true

	// First call should consume the flag without marking acted
	combatstate.MarkSquadAsActed(cache, squadID, manager)

	if actionState.HasActed {
		t.Error("HasActed should be false after BonusAttack consumes the flag")
	}
	if actionState.BonusAttackActive {
		t.Error("BonusAttackActive should be consumed (false)")
	}

	// Second call should mark as acted normally
	combatstate.MarkSquadAsActed(cache, squadID, manager)

	if !actionState.HasActed {
		t.Error("HasActed should be true after normal markSquadAsActed")
	}
}

// ========================================
// ZONE OF CONTROL TESTS
// ========================================

// setupZoCTest creates two opposing factions with one squad each at the given positions.
// Returns manager, cache, faction manager, faction IDs, squad IDs.
func setupZoCTest(t *testing.T, pos1, pos2 coords.LogicalPosition) (
	*common.EntityManager,
	*combatstate.CombatQueryCache,
	ecs.EntityID, ecs.EntityID,
	ecs.EntityID, ecs.EntityID,
) {
	t.Helper()
	manager := CreateTestCombatManager()
	cache := combatstate.NewCombatQueryCache(manager)
	fm := combatstate.NewCombatFactionManager(manager, cache)

	playerFaction := fm.CreateCombatFaction("Player", true)
	enemyFaction := fm.CreateCombatFaction("Enemy", false)

	playerSquad := CreateTestSquad(manager, "Player Squad", 3)
	enemySquad := CreateTestSquad(manager, "Enemy Squad", 3)

	fm.AddSquadToFaction(playerFaction, playerSquad, pos1)
	fm.AddSquadToFaction(enemyFaction, enemySquad, pos2)

	turnMgr := NewTurnManager(manager, cache)
	turnMgr.InitializeCombat([]ecs.EntityID{playerFaction, enemyFaction})

	return manager, cache, playerFaction, enemyFaction, playerSquad, enemySquad
}

func TestIsInZoneOfControl_AdjacentEnemy(t *testing.T) {
	pos1 := coords.LogicalPosition{X: 5, Y: 5}
	pos2 := coords.LogicalPosition{X: 6, Y: 5} // Distance 1
	manager, _, _, _, playerSquad, enemySquad := setupZoCTest(t, pos1, pos2)

	if !combatstate.IsInZoneOfControl(playerSquad, manager) {
		t.Error("Player squad should be in ZoC (adjacent enemy)")
	}
	if !combatstate.IsInZoneOfControl(enemySquad, manager) {
		t.Error("Enemy squad should be in ZoC (adjacent player)")
	}
}

func TestIsInZoneOfControl_NoAdjacentEnemy(t *testing.T) {
	pos1 := coords.LogicalPosition{X: 5, Y: 5}
	pos2 := coords.LogicalPosition{X: 8, Y: 5} // Distance 3
	manager, _, _, _, playerSquad, enemySquad := setupZoCTest(t, pos1, pos2)

	if combatstate.IsInZoneOfControl(playerSquad, manager) {
		t.Error("Player squad should NOT be in ZoC (enemy at distance 3)")
	}
	if combatstate.IsInZoneOfControl(enemySquad, manager) {
		t.Error("Enemy squad should NOT be in ZoC (player at distance 3)")
	}
}

func TestIsInZoneOfControl_DiagonalEnemy(t *testing.T) {
	pos1 := coords.LogicalPosition{X: 5, Y: 5}
	pos2 := coords.LogicalPosition{X: 6, Y: 6} // Diagonal, Chebyshev distance 1
	manager, _, _, _, playerSquad, enemySquad := setupZoCTest(t, pos1, pos2)

	if !combatstate.IsInZoneOfControl(playerSquad, manager) {
		t.Error("Player squad should be in ZoC (diagonal enemy at Chebyshev distance 1)")
	}
	if !combatstate.IsInZoneOfControl(enemySquad, manager) {
		t.Error("Enemy squad should be in ZoC (diagonal player at Chebyshev distance 1)")
	}
}

func TestIsInZoneOfControl_FriendlyAdjacent(t *testing.T) {
	manager := CreateTestCombatManager()
	cache := combatstate.NewCombatQueryCache(manager)
	fm := combatstate.NewCombatFactionManager(manager, cache)

	playerFaction := fm.CreateCombatFaction("Player", true)
	squad1 := CreateTestSquad(manager, "Squad 1", 3)
	squad2 := CreateTestSquad(manager, "Squad 2", 3)

	fm.AddSquadToFaction(playerFaction, squad1, coords.LogicalPosition{X: 5, Y: 5})
	fm.AddSquadToFaction(playerFaction, squad2, coords.LogicalPosition{X: 6, Y: 5})

	turnMgr := NewTurnManager(manager, cache)
	turnMgr.InitializeCombat([]ecs.EntityID{playerFaction})

	if combatstate.IsInZoneOfControl(squad1, manager) {
		t.Error("Squad should NOT be in ZoC from friendly adjacent squad")
	}
}

func TestIsInZoneOfControl_DestroyedSquadIgnored(t *testing.T) {
	pos1 := coords.LogicalPosition{X: 5, Y: 5}
	pos2 := coords.LogicalPosition{X: 6, Y: 5}
	manager, _, _, _, playerSquad, enemySquad := setupZoCTest(t, pos1, pos2)

	// Destroy the enemy squad by killing all units
	unitIDs := squadcore.GetUnitIDsInSquad(enemySquad, manager)
	for _, unitID := range unitIDs {
		unit := manager.FindEntityByID(unitID)
		if unit == nil {
			continue
		}
		attr := common.GetComponentType[*common.Attributes](unit, common.AttributeComponent)
		if attr != nil {
			attr.CurrentHealth = 0
		}
	}

	if combatstate.IsInZoneOfControl(playerSquad, manager) {
		t.Error("Player squad should NOT be in ZoC from destroyed enemy squad")
	}
}

func TestGetValidMovementTiles_InZoC(t *testing.T) {
	pos1 := coords.LogicalPosition{X: 10, Y: 10}
	pos2 := coords.LogicalPosition{X: 11, Y: 10} // Adjacent enemy
	manager, cache, _, _, playerSquad, _ := setupZoCTest(t, pos1, pos2)

	moveSys := NewMovementSystem(manager, common.GlobalPositionSystem, cache)
	tiles := moveSys.GetValidMovementTiles(playerSquad)

	// All returned tiles should be within Chebyshev distance 1 of the squad position
	for _, tile := range tiles {
		distance := pos1.ChebyshevDistance(&tile)
		if distance > 1 {
			t.Errorf("Tile (%d,%d) is at distance %d, but ZoC should cap movement to 1",
				tile.X, tile.Y, distance)
		}
	}

	// Should have at least some valid tiles (not zero)
	if len(tiles) == 0 {
		t.Error("Expected at least some valid movement tiles even in ZoC")
	}
}

func TestGetValidMovementTiles_NotInZoC(t *testing.T) {
	pos1 := coords.LogicalPosition{X: 10, Y: 10}
	pos2 := coords.LogicalPosition{X: 20, Y: 20} // Far away enemy
	manager, cache, _, _, playerSquad, _ := setupZoCTest(t, pos1, pos2)

	moveSys := NewMovementSystem(manager, common.GlobalPositionSystem, cache)
	tiles := moveSys.GetValidMovementTiles(playerSquad)

	// Should have tiles beyond distance 1 (squad speed is 5)
	hasDistantTile := false
	for _, tile := range tiles {
		distance := pos1.ChebyshevDistance(&tile)
		if distance > 1 {
			hasDistantTile = true
			break
		}
	}

	if !hasDistantTile {
		t.Error("Expected tiles beyond distance 1 when not in ZoC (squad has speed 5)")
	}
}

func TestZoC_MutualEffect(t *testing.T) {
	pos1 := coords.LogicalPosition{X: 10, Y: 10}
	pos2 := coords.LogicalPosition{X: 11, Y: 10}
	manager, cache, _, _, playerSquad, enemySquad := setupZoCTest(t, pos1, pos2)

	moveSys := NewMovementSystem(manager, common.GlobalPositionSystem, cache)

	playerTiles := moveSys.GetValidMovementTiles(playerSquad)
	enemyTiles := moveSys.GetValidMovementTiles(enemySquad)

	// Both should be capped to distance 1
	for _, tile := range playerTiles {
		if pos1.ChebyshevDistance(&tile) > 1 {
			t.Error("Player movement should be capped to 1 in mutual ZoC")
		}
	}
	for _, tile := range enemyTiles {
		if pos2.ChebyshevDistance(&tile) > 1 {
			t.Error("Enemy movement should be capped to 1 in mutual ZoC")
		}
	}
}
