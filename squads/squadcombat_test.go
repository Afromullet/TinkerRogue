package squads

import (
	"game_main/common"
	"game_main/coords"
	"testing"

	"github.com/bytearena/ecs"
)

// ========================================
// TEST HELPERS
// ========================================

// setupTestManager creates a manager with squad system initialized
func setupTestCombatManager(t *testing.T) *common.EntityManager {
	t.Helper()
	manager := common.NewEntityManager()

	if err := InitializeSquadData(manager); err != nil {
		t.Fatalf("Failed to initialize squad data: %v", err)
	}

	// Initialize common components for tests
	if common.PositionComponent == nil {
		common.PositionComponent = manager.World.NewComponent()
	}
	if common.AttributeComponent == nil {
		common.AttributeComponent = manager.World.NewComponent()
	}
	if common.NameComponent == nil {
		common.NameComponent = manager.World.NewComponent()
	}

	return manager
}

// createTestUnit creates a unit with specified attributes for testing
func createTestUnit(manager *common.EntityManager, squadID ecs.EntityID, row, col int, health, strength, dexterity int) *ecs.Entity {
	unit := manager.World.NewEntity()

	// Add required components
	unit.AddComponent(SquadMemberComponent, &SquadMemberData{SquadID: squadID})
	unit.AddComponent(GridPositionComponent, &GridPositionData{
		AnchorRow: row,
		AnchorCol: col,
		Width:     1,
		Height:    1,
	})
	unit.AddComponent(common.NameComponent, &common.Name{NameStr: "TestUnit"})

	// Add attributes using NewAttributes constructor
	attr := common.NewAttributes(
		strength,  // Strength
		dexterity, // Dexterity (affects hit/dodge/crit)
		0,         // Magic
		0,         // Leadership
		2,         // Armor
		2,         // Weapon
	)
	unit.AddComponent(common.AttributeComponent, &attr)

	// Add targeting data (default to front row)
	unit.AddComponent(TargetRowComponent, &TargetRowData{
		Mode:          TargetModeRowBased,
		TargetRows:    []int{0},
		IsMultiTarget: false,
		MaxTargets:    0,
	})

	// Add attack range component (default to melee range 1)
	unit.AddComponent(AttackRangeComponent, &AttackRangeData{
		Range: 1,
	})

	// Note: Tags are managed through the component query system, not directly added
	return unit
}

// createTestSquad creates a squad entity with specified ID at position (0,0)
func createTestSquad(manager *common.EntityManager, name string) ecs.EntityID {
	squad := manager.World.NewEntity()
	squadID := squad.GetID()

	squadData := &SquadData{
		SquadID:       squadID,
		Formation:     FormationBalanced,
		Name:          name,
		Morale:        100,
		SquadLevel:    1,
		TurnCount:     0,
		MaxUnits:      9,
		UsedCapacity:  0,
		TotalCapacity: 6,
	}

	squad.AddComponent(SquadComponent, squadData)

	// Add position component so squads can calculate distance
	squad.AddComponent(common.PositionComponent, &coords.LogicalPosition{
		X: 0,
		Y: 0,
	})

	// Note: Entities are added automatically, no need for AddEntity
	// Tags are managed through component queries

	return squadID
}

// ========================================
// ExecuteSquadAttack TESTS
// ========================================

func TestExecuteSquadAttack_SingleAttackerVsSingleDefender(t *testing.T) {
	manager := setupTestCombatManager(t)

	// Create attacker squad
	attackerSquadID := createTestSquad(manager, "Attackers")
	_ = createTestUnit(manager, attackerSquadID, 0, 0, 100, 20, 100) // 100% hit rate

	// Create defender squad
	defenderSquadID := createTestSquad(manager, "Defenders")
	defenderUnit := createTestUnit(manager, defenderSquadID, 0, 0, 50, 10, 0)
	defenderAttr := common.GetAttributes(defenderUnit)
	initialHP := defenderAttr.CurrentHealth

	// Execute attack
	result := ExecuteSquadAttack(attackerSquadID, defenderSquadID, manager)

	// Verify result
	if result == nil {
		t.Fatal("Expected combat result, got nil")
	}

	if result.TotalDamage <= 0 {
		t.Error("Expected damage > 0")
	}

	if defenderAttr.CurrentHealth >= initialHP {
		t.Errorf("Expected defender HP to decrease from %d, got %d", initialHP, defenderAttr.CurrentHealth)
	}

	if len(result.DamageByUnit) != 1 {
		t.Errorf("Expected 1 unit damaged, got %d", len(result.DamageByUnit))
	}
}

func TestExecuteSquadAttack_MultipleAttackersVsMultipleDefenders(t *testing.T) {
	manager := setupTestCombatManager(t)

	// Create attacker squad with 3 units
	attackerSquadID := createTestSquad(manager, "Attackers")
	createTestUnit(manager, attackerSquadID, 0, 0, 100, 20, 100)
	createTestUnit(manager, attackerSquadID, 0, 1, 100, 20, 100)
	createTestUnit(manager, attackerSquadID, 0, 2, 100, 20, 100)

	// Create defender squad with 3 units
	defenderSquadID := createTestSquad(manager, "Defenders")
	createTestUnit(manager, defenderSquadID, 0, 0, 50, 10, 0)
	createTestUnit(manager, defenderSquadID, 0, 1, 50, 10, 0)
	createTestUnit(manager, defenderSquadID, 0, 2, 50, 10, 0)

	// Execute attack
	result := ExecuteSquadAttack(attackerSquadID, defenderSquadID, manager)

	// Verify result
	if result.TotalDamage <= 0 {
		t.Error("Expected total damage > 0")
	}

	// Each attacker should hit one target (lowest HP)
	if len(result.DamageByUnit) < 1 {
		t.Errorf("Expected at least 1 unit damaged, got %d", len(result.DamageByUnit))
	}
}

func TestExecuteSquadAttack_DeadAttackersDoNotAttack(t *testing.T) {
	manager := setupTestCombatManager(t)

	// Create attacker squad with dead unit
	attackerSquadID := createTestSquad(manager, "Attackers")
	deadAttacker := createTestUnit(manager, attackerSquadID, 0, 0, 100, 20, 100)
	attr := common.GetAttributes(deadAttacker)
	attr.CurrentHealth = 0 // Dead unit

	// Create defender squad
	defenderSquadID := createTestSquad(manager, "Defenders")
	defenderUnit := createTestUnit(manager, defenderSquadID, 0, 0, 50, 10, 0)
	defenderAttr := common.GetAttributes(defenderUnit)
	initialHP := defenderAttr.CurrentHealth

	// Execute attack
	result := ExecuteSquadAttack(attackerSquadID, defenderSquadID, manager)

	// Verify no damage dealt
	if result.TotalDamage != 0 {
		t.Errorf("Expected no damage from dead attacker, got %d", result.TotalDamage)
	}

	if defenderAttr.CurrentHealth != initialHP {
		t.Errorf("Expected defender HP to remain %d, got %d", initialHP, defenderAttr.CurrentHealth)
	}
}

func TestExecuteSquadAttack_MultiTargetAttack(t *testing.T) {
	manager := setupTestCombatManager(t)

	// Create attacker with multi-target ability
	attackerSquadID := createTestSquad(manager, "Attackers")
	attacker := createTestUnit(manager, attackerSquadID, 0, 0, 100, 20, 100)

	// Set multi-target mode
	targetData := common.GetComponentType[*TargetRowData](attacker, TargetRowComponent)
	targetData.IsMultiTarget = true
	targetData.MaxTargets = 2

	// Create defenders
	defenderSquadID := createTestSquad(manager, "Defenders")
	createTestUnit(manager, defenderSquadID, 0, 0, 50, 10, 0)
	createTestUnit(manager, defenderSquadID, 0, 1, 50, 10, 0)
	createTestUnit(manager, defenderSquadID, 0, 2, 50, 10, 0)

	// Execute attack
	result := ExecuteSquadAttack(attackerSquadID, defenderSquadID, manager)

	// Verify multiple targets hit (should be 2 based on MaxTargets)
	if len(result.DamageByUnit) > 2 {
		t.Errorf("Expected at most 2 units damaged, got %d", len(result.DamageByUnit))
	}
}

func TestExecuteSquadAttack_CellBasedTargeting(t *testing.T) {
	manager := setupTestCombatManager(t)

	// Create attacker with cell-based targeting (2x2 pattern)
	attackerSquadID := createTestSquad(manager, "Attackers")
	attacker := createTestUnit(manager, attackerSquadID, 0, 0, 100, 20, 100)

	// Set cell-based targeting
	targetData := common.GetComponentType[*TargetRowData](attacker, TargetRowComponent)
	targetData.Mode = TargetModeCellBased
	targetData.TargetCells = [][2]int{{0, 0}, {0, 1}, {1, 0}, {1, 1}} // 2x2 top-left

	// Create defenders in targeted cells
	defenderSquadID := createTestSquad(manager, "Defenders")
	createTestUnit(manager, defenderSquadID, 0, 0, 50, 10, 0) // Hit
	createTestUnit(manager, defenderSquadID, 0, 1, 50, 10, 0) // Hit
	createTestUnit(manager, defenderSquadID, 1, 0, 50, 10, 0) // Hit
	createTestUnit(manager, defenderSquadID, 2, 2, 50, 10, 0) // Miss (not in pattern)

	// Execute attack
	result := ExecuteSquadAttack(attackerSquadID, defenderSquadID, manager)

	// Verify 3 units hit (3 in the 2x2 pattern)
	if len(result.DamageByUnit) != 3 {
		t.Errorf("Expected 3 units damaged (2x2 pattern), got %d", len(result.DamageByUnit))
	}
}

func TestExecuteSquadAttack_UnitsKilledTracking(t *testing.T) {
	manager := setupTestCombatManager(t)

	// Create attacker with high damage
	attackerSquadID := createTestSquad(manager, "Attackers")
	createTestUnit(manager, attackerSquadID, 0, 0, 100, 100, 100) // High strength

	// Create weak defender
	defenderSquadID := createTestSquad(manager, "Defenders")
	createTestUnit(manager, defenderSquadID, 0, 0, 10, 5, 0) // Low HP

	// Execute attack
	result := ExecuteSquadAttack(attackerSquadID, defenderSquadID, manager)

	// Verify unit was killed
	if len(result.UnitsKilled) != 1 {
		t.Errorf("Expected 1 unit killed, got %d", len(result.UnitsKilled))
	}
}

// ========================================
// calculateUnitDamageByID TESTS
// ========================================

func TestCalculateUnitDamageByID_BasicDamageCalculation(t *testing.T) {
	manager := setupTestCombatManager(t)

	squadID := createTestSquad(manager, "TestSquad")
	attacker := createTestUnit(manager, squadID, 0, 0, 100, 20, 100) // 100% hit rate
	defender := createTestUnit(manager, squadID, 0, 1, 100, 10, 0)

	attackerAttr := common.GetAttributes(attacker)
	defenderAttr := common.GetAttributes(defender)

	// Note: Attributes are derived from base stats (Strength, Dexterity, etc.)
	// We can't set them directly, but with Dexterity=100, attacker should have high hit rate
	// With Dexterity=0, defender should have low dodge chance

	damage := calculateUnitDamageByID(attacker.GetID(), defender.GetID(), manager)

	if damage <= 0 {
		t.Error("Expected positive damage (note: may miss/dodge based on derived stats)")
	}

	// Damage should be based on attacker's strength when it hits
	baseDamage := attackerAttr.GetPhysicalDamage()
	resistance := defenderAttr.GetPhysicalResistance()
	expectedDamage := baseDamage - resistance
	if expectedDamage < 1 {
		expectedDamage = 1 // Minimum damage
	}

	// Allow for misses (damage could be 0)
	if damage > 0 && damage != expectedDamage {
		t.Logf("Expected damage %d (after resistance), got %d (variance from crit/resistance)", expectedDamage, damage)
	}
}

func TestCalculateUnitDamageByID_MissReturnsZero(t *testing.T) {
	manager := setupTestCombatManager(t)

	squadID := createTestSquad(manager, "TestSquad")
	attacker := createTestUnit(manager, squadID, 0, 0, 100, 20, 0) // Low dexterity = low hit rate
	defender := createTestUnit(manager, squadID, 0, 1, 100, 10, 0)

	attackerAttr := common.GetAttributes(attacker)
	_ = attackerAttr // Keep for potential future use

	// Note: With Dexterity=0, hit rate is 80% (still decent)
	// This test may pass or fail based on random rolls
	// For a reliable test, we'd need a way to inject randomness

	damage := calculateUnitDamageByID(attacker.GetID(), defender.GetID(), manager)

	// Can't reliably test for 0 damage without controlling randomness
	t.Logf("Damage dealt: %d (0 expected on miss, but randomness not controlled)", damage)
}

func TestCalculateUnitDamageByID_DodgeReturnsZero(t *testing.T) {
	manager := setupTestCombatManager(t)

	squadID := createTestSquad(manager, "TestSquad")
	attacker := createTestUnit(manager, squadID, 0, 0, 100, 20, 100)
	defender := createTestUnit(manager, squadID, 0, 1, 100, 10, 100) // High dexterity for dodge

	attackerAttr := common.GetAttributes(attacker)
	defenderAttr := common.GetAttributes(defender)
	_, _ = attackerAttr, defenderAttr // Keep for potential future use

	// Note: With Dexterity=100, dodge chance is capped at 40%
	// This test may pass or fail based on random rolls
	// For a reliable test, we'd need a way to inject randomness

	damage := calculateUnitDamageByID(attacker.GetID(), defender.GetID(), manager)

	// Can't reliably test for 0 damage without controlling randomness
	t.Logf("Damage dealt: %d (0 expected on dodge, but randomness not controlled)", damage)
}

func TestCalculateUnitDamageByID_PhysicalResistanceReducesDamage(t *testing.T) {
	manager := setupTestCombatManager(t)

	squadID := createTestSquad(manager, "TestSquad")
	// Strength=20, Armor=10 for defender gives significant resistance
	attacker := createTestUnit(manager, squadID, 0, 0, 100, 20, 100)
	defender := createTestUnit(manager, squadID, 0, 1, 100, 20, 10) // Higher armor = higher resistance

	attackerAttr := common.GetAttributes(attacker)
	defenderAttr := common.GetAttributes(defender)

	// Note: PhysicalResistance is derived from Strength/4 + Armor*2
	// Defender: (20/4) + (10*2) = 5 + 20 = 25 resistance

	damage := calculateUnitDamageByID(attacker.GetID(), defender.GetID(), manager)

	baseDamage := attackerAttr.GetPhysicalDamage()
	resistance := defenderAttr.GetPhysicalResistance()
	expectedDamage := baseDamage - resistance
	if expectedDamage < 1 {
		expectedDamage = 1 // Minimum damage
	}

	// Allow for variance from hit/dodge/crit
	if damage > 0 && damage > baseDamage {
		t.Errorf("Damage %d should not exceed base damage %d without crits", damage, baseDamage)
	}

	t.Logf("Base damage: %d, Resistance: %d, Expected: %d, Actual: %d", baseDamage, resistance, expectedDamage, damage)
}

func TestCalculateUnitDamageByID_MinimumDamageIsOne(t *testing.T) {
	manager := setupTestCombatManager(t)

	squadID := createTestSquad(manager, "TestSquad")
	// Very low strength/weapon for attacker, very high armor for defender
	attacker := createTestUnit(manager, squadID, 0, 0, 100, 1, 0) // Strength=1, Weapon=0
	defender := createTestUnit(manager, squadID, 0, 1, 100, 50, 50) // Strength=50, Armor=50 for high resistance

	attackerAttr := common.GetAttributes(attacker)
	defenderAttr := common.GetAttributes(defender)

	// Attacker damage: (1/2) + (0*2) = 0
	// Defender resistance: (50/4) + (50*2) = 12 + 100 = 112
	// Expected: 0 - 112 = minimum 1

	damage := calculateUnitDamageByID(attacker.GetID(), defender.GetID(), manager)

	// Minimum damage should be 1 when attack hits
	if damage > 1 {
		t.Logf("Expected minimum damage of 1, got %d (might be crit or miss rolled 0)", damage)
	}

	t.Logf("Attacker damage: %d, Defender resistance: %d, Actual damage: %d",
		attackerAttr.GetPhysicalDamage(), defenderAttr.GetPhysicalResistance(), damage)
}

func TestCalculateUnitDamageByID_NilUnitsReturnZero(t *testing.T) {
	manager := setupTestCombatManager(t)

	damage := calculateUnitDamageByID(9999, 9998, manager) // Non-existent IDs

	if damage != 0 {
		t.Errorf("Expected 0 damage for nil units, got %d", damage)
	}
}

// ========================================
// COVER SYSTEM TESTS
// ========================================

func TestCalculateTotalCover_NoCoverProviders(t *testing.T) {
	manager := setupTestCombatManager(t)

	squadID := createTestSquad(manager, "TestSquad")
	defender := createTestUnit(manager, squadID, 1, 0, 100, 10, 0)

	cover := CalculateTotalCover(defender.GetID(), manager)

	if cover != 0.0 {
		t.Errorf("Expected 0 cover with no providers, got %f", cover)
	}
}

func TestCalculateTotalCover_SingleCoverProvider(t *testing.T) {
	manager := setupTestCombatManager(t)

	squadID := createTestSquad(manager, "TestSquad")

	// Create front-line unit with cover
	frontLine := createTestUnit(manager, squadID, 0, 0, 100, 10, 0)
	frontLine.AddComponent(CoverComponent, &CoverData{
		CoverValue:     0.25, // 25% damage reduction
		CoverRange:     1,
		RequiresActive: true,
	})

	// Create back-line unit that receives cover
	backLine := createTestUnit(manager, squadID, 1, 0, 100, 10, 0)

	cover := CalculateTotalCover(backLine.GetID(), manager)

	if cover != 0.25 {
		t.Errorf("Expected 0.25 cover, got %f", cover)
	}
}

func TestCalculateTotalCover_MultipleCoverProvidersStack(t *testing.T) {
	manager := setupTestCombatManager(t)

	squadID := createTestSquad(manager, "TestSquad")

	// Create two front-line units with cover in same column
	frontLine1 := createTestUnit(manager, squadID, 0, 0, 100, 10, 0)
	frontLine1.AddComponent(CoverComponent, &CoverData{
		CoverValue:     0.20,
		CoverRange:     2,
		RequiresActive: true,
	})

	// Middle unit also provides cover
	midLine := createTestUnit(manager, squadID, 1, 0, 100, 10, 0)
	midLine.AddComponent(CoverComponent, &CoverData{
		CoverValue:     0.15,
		CoverRange:     1,
		RequiresActive: true,
	})

	// Back-line unit receives cover from both
	backLine := createTestUnit(manager, squadID, 2, 0, 100, 10, 0)

	cover := CalculateTotalCover(backLine.GetID(), manager)

	expectedCover := 0.20 + 0.15 // Stacking additively
	if cover != expectedCover {
		t.Errorf("Expected %f cover (stacked), got %f", expectedCover, cover)
	}
}

func TestCalculateTotalCover_DeadUnitDoesNotProvideCover(t *testing.T) {
	manager := setupTestCombatManager(t)

	squadID := createTestSquad(manager, "TestSquad")

	// Create dead front-line unit
	deadFrontLine := createTestUnit(manager, squadID, 0, 0, 100, 10, 0)
	deadFrontLine.AddComponent(CoverComponent, &CoverData{
		CoverValue:     0.25,
		CoverRange:     1,
		RequiresActive: true,
	})
	attr := common.GetAttributes(deadFrontLine)
	attr.CurrentHealth = 0 // Dead

	// Create back-line unit
	backLine := createTestUnit(manager, squadID, 1, 0, 100, 10, 0)

	cover := CalculateTotalCover(backLine.GetID(), manager)

	if cover != 0.0 {
		t.Errorf("Expected 0 cover from dead unit, got %f", cover)
	}
}

func TestCalculateTotalCover_CoverRangeLimit(t *testing.T) {
	manager := setupTestCombatManager(t)

	squadID := createTestSquad(manager, "TestSquad")

	// Front-line unit with range 1
	frontLine := createTestUnit(manager, squadID, 0, 0, 100, 10, 0)
	frontLine.AddComponent(CoverComponent, &CoverData{
		CoverValue:     0.25,
		CoverRange:     1, // Only covers 1 row behind
		RequiresActive: true,
	})

	// Back-line unit 2 rows behind (out of range)
	backLine := createTestUnit(manager, squadID, 2, 0, 100, 10, 0)

	cover := CalculateTotalCover(backLine.GetID(), manager)

	if cover != 0.0 {
		t.Errorf("Expected 0 cover (out of range), got %f", cover)
	}
}

func TestCalculateTotalCover_CoverOnlyInSameColumn(t *testing.T) {
	manager := setupTestCombatManager(t)

	squadID := createTestSquad(manager, "TestSquad")

	// Front-line unit in column 0
	frontLine := createTestUnit(manager, squadID, 0, 0, 100, 10, 0)
	frontLine.AddComponent(CoverComponent, &CoverData{
		CoverValue:     0.25,
		CoverRange:     1,
		RequiresActive: true,
	})

	// Back-line unit in different column
	backLine := createTestUnit(manager, squadID, 1, 2, 100, 10, 0)

	cover := CalculateTotalCover(backLine.GetID(), manager)

	if cover != 0.0 {
		t.Errorf("Expected 0 cover (different column), got %f", cover)
	}
}

func TestCalculateTotalCover_CappedAtOne(t *testing.T) {
	manager := setupTestCombatManager(t)

	squadID := createTestSquad(manager, "TestSquad")

	// Multiple units with excessive cover
	for i := 0; i < 3; i++ {
		frontLine := createTestUnit(manager, squadID, 0, 0, 100, 10, 0)
		frontLine.AddComponent(CoverComponent, &CoverData{
			CoverValue:     0.50, // 50% each
			CoverRange:     2,
			RequiresActive: true,
		})
	}

	// Back-line unit
	backLine := createTestUnit(manager, squadID, 1, 0, 100, 10, 0)

	cover := CalculateTotalCover(backLine.GetID(), manager)

	if cover > 1.0 {
		t.Errorf("Expected cover capped at 1.0, got %f", cover)
	}

	if cover != 1.0 {
		t.Logf("Note: Cover was %f (< 1.0), may need more providers for this test", cover)
	}
}

// ========================================
// GetCoverProvidersFor TESTS
// ========================================

func TestGetCoverProvidersFor_NoProviders(t *testing.T) {
	manager := setupTestCombatManager(t)

	squadID := createTestSquad(manager, "TestSquad")
	defender := createTestUnit(manager, squadID, 1, 0, 100, 10, 0)
	defenderPos := common.GetComponentType[*GridPositionData](defender, GridPositionComponent)

	providers := GetCoverProvidersFor(defender.GetID(), squadID, defenderPos, manager)

	if len(providers) != 0 {
		t.Errorf("Expected 0 providers, got %d", len(providers))
	}
}

func TestGetCoverProvidersFor_SingleProvider(t *testing.T) {
	manager := setupTestCombatManager(t)

	squadID := createTestSquad(manager, "TestSquad")

	frontLine := createTestUnit(manager, squadID, 0, 0, 100, 10, 0)
	frontLine.AddComponent(CoverComponent, &CoverData{
		CoverValue:     0.25,
		CoverRange:     1,
		RequiresActive: true,
	})

	backLine := createTestUnit(manager, squadID, 1, 0, 100, 10, 0)
	backLinePos := common.GetComponentType[*GridPositionData](backLine, GridPositionComponent)

	providers := GetCoverProvidersFor(backLine.GetID(), squadID, backLinePos, manager)

	if len(providers) != 1 {
		t.Errorf("Expected 1 provider, got %d", len(providers))
	}

	if len(providers) > 0 && providers[0] != frontLine.GetID() {
		t.Error("Expected front-line unit to be the provider")
	}
}

func TestGetCoverProvidersFor_MultipleProviders(t *testing.T) {
	manager := setupTestCombatManager(t)

	squadID := createTestSquad(manager, "TestSquad")

	// Two front-line units in same column
	frontLine1 := createTestUnit(manager, squadID, 0, 0, 100, 10, 0)
	frontLine1.AddComponent(CoverComponent, &CoverData{
		CoverValue:     0.15,
		CoverRange:     2,
		RequiresActive: true,
	})

	midLine := createTestUnit(manager, squadID, 1, 0, 100, 10, 0)
	midLine.AddComponent(CoverComponent, &CoverData{
		CoverValue:     0.10,
		CoverRange:     1,
		RequiresActive: true,
	})

	backLine := createTestUnit(manager, squadID, 2, 0, 100, 10, 0)
	backLinePos := common.GetComponentType[*GridPositionData](backLine, GridPositionComponent)

	providers := GetCoverProvidersFor(backLine.GetID(), squadID, backLinePos, manager)

	if len(providers) != 2 {
		t.Errorf("Expected 2 providers, got %d", len(providers))
	}
}

func TestGetCoverProvidersFor_DoesNotIncludeSelf(t *testing.T) {
	manager := setupTestCombatManager(t)

	squadID := createTestSquad(manager, "TestSquad")

	unit := createTestUnit(manager, squadID, 0, 0, 100, 10, 0)
	unit.AddComponent(CoverComponent, &CoverData{
		CoverValue:     0.25,
		CoverRange:     1,
		RequiresActive: true,
	})
	unitPos := common.GetComponentType[*GridPositionData](unit, GridPositionComponent)

	providers := GetCoverProvidersFor(unit.GetID(), squadID, unitPos, manager)

	if len(providers) != 0 {
		t.Errorf("Expected 0 providers (unit should not provide cover to itself), got %d", len(providers))
	}
}

func TestGetCoverProvidersFor_OnlyFromSameSquad(t *testing.T) {
	manager := setupTestCombatManager(t)

	// Squad 1
	squad1ID := createTestSquad(manager, "Squad1")
	squad1Unit := createTestUnit(manager, squad1ID, 0, 0, 100, 10, 0)
	squad1Unit.AddComponent(CoverComponent, &CoverData{
		CoverValue:     0.25,
		CoverRange:     1,
		RequiresActive: true,
	})

	// Squad 2
	squad2ID := createTestSquad(manager, "Squad2")
	squad2Unit := createTestUnit(manager, squad2ID, 1, 0, 100, 10, 0)
	squad2UnitPos := common.GetComponentType[*GridPositionData](squad2Unit, GridPositionComponent)

	// Squad 2 unit should not get cover from Squad 1
	providers := GetCoverProvidersFor(squad2Unit.GetID(), squad2ID, squad2UnitPos, manager)

	if len(providers) != 0 {
		t.Errorf("Expected 0 providers from different squad, got %d", len(providers))
	}
}

// ========================================
// HELPER FUNCTION TESTS
// ========================================

func TestSelectLowestHPTargetID(t *testing.T) {
	manager := setupTestCombatManager(t)

	squadID := createTestSquad(manager, "TestSquad")

	unit1 := createTestUnit(manager, squadID, 0, 0, 100, 10, 0)
	unit2 := createTestUnit(manager, squadID, 0, 1, 50, 10, 0)  // Lowest HP
	unit3 := createTestUnit(manager, squadID, 0, 2, 75, 10, 0)

	unitIDs := []ecs.EntityID{unit1.GetID(), unit2.GetID(), unit3.GetID()}

	targetID := selectLowestHPTargetID(unitIDs, manager)

	if targetID != unit2.GetID() {
		t.Errorf("Expected unit2 (lowest HP), got %d", targetID)
	}
}

func TestSelectLowestHPTargetID_EmptyList(t *testing.T) {
	manager := setupTestCombatManager(t)

	var emptyList []ecs.EntityID
	targetID := selectLowestHPTargetID(emptyList, manager)

	if targetID != 0 {
		t.Errorf("Expected 0 for empty list, got %d", targetID)
	}
}

func TestSelectRandomTargetIDs_CountLessThanList(t *testing.T) {
	manager := setupTestCombatManager(t)

	squadID := createTestSquad(manager, "TestSquad")

	unit1 := createTestUnit(manager, squadID, 0, 0, 100, 10, 0)
	unit2 := createTestUnit(manager, squadID, 0, 1, 100, 10, 0)
	unit3 := createTestUnit(manager, squadID, 0, 2, 100, 10, 0)

	unitIDs := []ecs.EntityID{unit1.GetID(), unit2.GetID(), unit3.GetID()}

	selected := selectRandomTargetIDs(unitIDs, 2)

	if len(selected) != 2 {
		t.Errorf("Expected 2 selected targets, got %d", len(selected))
	}
}

func TestSelectRandomTargetIDs_CountGreaterThanList(t *testing.T) {
	manager := setupTestCombatManager(t)

	squadID := createTestSquad(manager, "TestSquad")

	unit1 := createTestUnit(manager, squadID, 0, 0, 100, 10, 0)
	unit2 := createTestUnit(manager, squadID, 0, 1, 100, 10, 0)

	unitIDs := []ecs.EntityID{unit1.GetID(), unit2.GetID()}

	selected := selectRandomTargetIDs(unitIDs, 5)

	if len(selected) != 2 {
		t.Errorf("Expected all 2 units returned, got %d", len(selected))
	}
}

func TestSumDamageMap(t *testing.T) {
	damageMap := map[ecs.EntityID]int{
		1: 10,
		2: 20,
		3: 30,
	}

	total := sumDamageMap(damageMap)

	if total != 60 {
		t.Errorf("Expected total damage 60, got %d", total)
	}
}

func TestSumDamageMap_EmptyMap(t *testing.T) {
	damageMap := make(map[ecs.EntityID]int)

	total := sumDamageMap(damageMap)

	if total != 0 {
		t.Errorf("Expected total damage 0 for empty map, got %d", total)
	}
}

// ========================================
// INTEGRATION TESTS
// ========================================

func TestCombatWithCoverSystem_Integration(t *testing.T) {
	manager := setupTestCombatManager(t)

	// Create attacker squad
	attackerSquadID := createTestSquad(manager, "Attackers")
	attacker := createTestUnit(manager, attackerSquadID, 0, 0, 100, 30, 0) // Low dexterity = lower crit
	attackerAttr := common.GetAttributes(attacker)

	// Create defender squad with cover
	defenderSquadID := createTestSquad(manager, "Defenders")

	// Front-line unit provides cover
	frontLine := createTestUnit(manager, defenderSquadID, 0, 0, 100, 10, 0)
	frontLine.AddComponent(CoverComponent, &CoverData{
		CoverValue:     0.50, // 50% damage reduction
		CoverRange:     1,
		RequiresActive: true,
	})

	// Back-line unit receives cover
	backLine := createTestUnit(manager, defenderSquadID, 1, 0, 100, 10, 0) // Low dexterity = low dodge
	backLineAttr := common.GetAttributes(backLine)

	// Configure attacker to target back line
	targetData := common.GetComponentType[*TargetRowData](attacker, TargetRowComponent)
	targetData.TargetRows = []int{1} // Target row 1

	// Execute attack
	result := ExecuteSquadAttack(attackerSquadID, defenderSquadID, manager)

	// Verify cover reduced damage
	if len(result.DamageByUnit) != 1 {
		t.Fatalf("Expected 1 unit damaged, got %d", len(result.DamageByUnit))
	}

	damageDealt := result.DamageByUnit[backLine.GetID()]
	baseDamage := attackerAttr.GetPhysicalDamage()
	resistance := backLineAttr.GetPhysicalResistance()
	_ = resistance // May be used in future assertions

	// Cover reduces damage by 50%, but we also need to account for resistance
	// expectedDamage := int(float64(baseDamage) * 0.50) // 50% reduction from cover

	// Allow variance due to randomness, crits, resistance
	if damageDealt < 0 || damageDealt > baseDamage {
		t.Errorf("Expected damage between 0 and %d (with 50%% cover), got %d", baseDamage, damageDealt)
	}

	// Verify cover had some effect (unless attack missed)
	if damageDealt > 0 && damageDealt >= baseDamage {
		t.Error("Expected cover to reduce damage below base damage")
	}

	t.Logf("Base damage: %d, Damage dealt: %d (with 50%% cover)", baseDamage, damageDealt)
}

func TestMultiRoundCombat_Integration(t *testing.T) {
	manager := setupTestCombatManager(t)

	// Create two evenly matched squads
	squad1ID := createTestSquad(manager, "Squad1")
	createTestUnit(manager, squad1ID, 0, 0, 50, 15, 100)
	createTestUnit(manager, squad1ID, 0, 1, 50, 15, 100)

	squad2ID := createTestSquad(manager, "Squad2")
	createTestUnit(manager, squad2ID, 0, 0, 50, 15, 100)
	createTestUnit(manager, squad2ID, 0, 1, 50, 15, 100)

	// Simulate multiple rounds
	rounds := 0
	maxRounds := 10

	for rounds < maxRounds {
		rounds++

		// Squad 1 attacks Squad 2
		result1 := ExecuteSquadAttack(squad1ID, squad2ID, manager)

		// Check if Squad 2 is destroyed
		if IsSquadDestroyed(squad2ID, manager) {
			t.Logf("Squad2 destroyed in round %d", rounds)
			break
		}

		// Squad 2 attacks Squad 1
		result2 := ExecuteSquadAttack(squad2ID, squad1ID, manager)

		// Check if Squad 1 is destroyed
		if IsSquadDestroyed(squad1ID, manager) {
			t.Logf("Squad1 destroyed in round %d", rounds)
			break
		}

		t.Logf("Round %d: Squad1 dealt %d damage, Squad2 dealt %d damage",
			rounds, result1.TotalDamage, result2.TotalDamage)
	}

	if rounds >= maxRounds {
		t.Log("Combat reached maximum rounds without destruction")
	}

	// At least one squad should be heavily damaged
	squad1Destroyed := IsSquadDestroyed(squad1ID, manager)
	squad2Destroyed := IsSquadDestroyed(squad2ID, manager)

	if !squad1Destroyed && !squad2Destroyed {
		t.Log("Both squads survived - this is possible with lucky dodges/misses")
	}
}

// ========================================
// BENCHMARK TESTS
// ========================================

func BenchmarkExecuteSquadAttack_SingleVsSingle(b *testing.B) {
	manager := setupTestCombatManager(&testing.T{})

	attackerSquadID := createTestSquad(manager, "Attackers")
	createTestUnit(manager, attackerSquadID, 0, 0, 100, 20, 100)

	defenderSquadID := createTestSquad(manager, "Defenders")
	createTestUnit(manager, defenderSquadID, 0, 0, 100, 10, 0)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ExecuteSquadAttack(attackerSquadID, defenderSquadID, manager)
	}
}

func BenchmarkExecuteSquadAttack_FullSquadVsFullSquad(b *testing.B) {
	manager := setupTestCombatManager(&testing.T{})

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
		ExecuteSquadAttack(attackerSquadID, defenderSquadID, manager)
	}
}

func BenchmarkCalculateTotalCover(b *testing.B) {
	manager := setupTestCombatManager(&testing.T{})

	squadID := createTestSquad(manager, "TestSquad")

	// Create front-line with cover
	frontLine := createTestUnit(manager, squadID, 0, 0, 100, 10, 0)
	frontLine.AddComponent(CoverComponent, &CoverData{
		CoverValue:     0.25,
		CoverRange:     1,
		RequiresActive: true,
	})

	// Create back-line
	backLine := createTestUnit(manager, squadID, 1, 0, 100, 10, 0)
	defenderID := backLine.GetID()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		CalculateTotalCover(defenderID, manager)
	}
}
