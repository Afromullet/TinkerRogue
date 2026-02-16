package gear

import (
	"game_main/common"
	"game_main/tactical/combat"
	"game_main/tactical/effects"
	"game_main/tactical/squads"
	"game_main/templates"
	testfx "game_main/testing"
	"game_main/world/coords"
	"testing"

	"github.com/bytearena/ecs"
)

// setupTestManager creates a test EntityManager with all subsystems initialized.
func setupTestManager() *common.EntityManager {
	manager := testfx.NewTestEntityManager()
	if err := squads.InitializeSquadData(manager); err != nil {
		panic(err)
	}
	common.InitializeSubsystems(manager)
	return manager
}

// setupTestArtifacts populates the artifact registry with test data.
func setupTestArtifacts() {
	templates.ArtifactRegistry = make(map[string]*templates.ArtifactDefinition)
	templates.ArtifactRegistry["iron_bulwark"] = &templates.ArtifactDefinition{
		ID:   "iron_bulwark",
		Name: "Iron Bulwark",
		Tier: "minor",
		StatModifiers: []templates.ArtifactStatModifier{
			{Stat: "armor", Modifier: 2},
		},
	}
	templates.ArtifactRegistry["berserkers_torc"] = &templates.ArtifactDefinition{
		ID:   "berserkers_torc",
		Name: "Berserker's Torc",
		Tier: "minor",
		StatModifiers: []templates.ArtifactStatModifier{
			{Stat: "strength", Modifier: 2},
			{Stat: "armor", Modifier: -1},
		},
	}
	templates.ArtifactRegistry["commanders_initiative_badge"] = &templates.ArtifactDefinition{
		ID:       "commanders_initiative_badge",
		Name:     "Commander's Initiative Badge",
		Tier:     "major",
		Behavior: BehaviorInitiativeFirst,
	}
	templates.ArtifactRegistry["vanguards_oath"] = &templates.ArtifactDefinition{
		ID:       "vanguards_oath",
		Name:     "Vanguard's Oath",
		Tier:     "major",
		Behavior: BehaviorVanguardMovement,
	}
}

// createTestPlayer creates a player entity with an artifact inventory.
func createTestPlayer(manager *common.EntityManager) ecs.EntityID {
	entity := manager.World.NewEntity().
		AddComponent(common.PlayerComponent, &common.Player{}).
		AddComponent(ArtifactInventoryComponent, NewArtifactInventory(20))
	return entity.GetID()
}

// createTestSquadWithUnits creates a squad entity with units for testing.
func createTestSquadWithUnits(manager *common.EntityManager, name string, unitCount int) ecs.EntityID {
	squadEntity := manager.World.NewEntity()
	squadID := squadEntity.GetID()

	squadEntity.AddComponent(squads.SquadComponent, &squads.SquadData{
		SquadID:   squadID,
		Name:      name,
		Formation: squads.FormationBalanced,
		MaxUnits:  9,
	})

	for i := 0; i < unitCount; i++ {
		unitEntity := manager.World.NewEntity()
		unitEntity.AddComponent(common.AttributeComponent, &common.Attributes{
			Strength:      10,
			Dexterity:     10,
			Armor:         2,
			Weapon:        2,
			MovementSpeed: 5,
			AttackRange:   1,
			CurrentHealth: 30,
			MaxHealth:     30,
			CanAct:        true,
		})
		unitEntity.AddComponent(squads.SquadMemberComponent, &squads.SquadMemberData{
			SquadID: squadID,
		})
	}

	return squadID
}

// addArtifactToInventory is a test helper that adds an artifact to the player inventory.
func addArtifactToInventory(playerID ecs.EntityID, artifactID string, manager *common.EntityManager) {
	inv := GetPlayerArtifactInventory(playerID, manager)
	if err := AddArtifactToInventory(inv, artifactID); err != nil {
		panic(err)
	}
}

// hasSpecificArtifactInFaction checks if any squad in the given list has a specific artifact equipped.
func hasSpecificArtifactInFaction(squadIDs []ecs.EntityID, artifactID string, manager *common.EntityManager) bool {
	for _, squadID := range squadIDs {
		data := GetEquipmentData(squadID, manager)
		if data == nil {
			continue
		}
		for _, equipped := range data.EquippedArtifacts {
			if equipped == artifactID {
				return true
			}
		}
	}
	return false
}

// containsArtifact checks if EquippedArtifacts contains the given artifact ID.
func containsArtifact(data *EquipmentData, artifactID string) bool {
	for _, id := range data.EquippedArtifacts {
		if id == artifactID {
			return true
		}
	}
	return false
}

// ========================================
// EQUIP / UNEQUIP TESTS
// ========================================

func TestEquipBlocksWhenSlotsFull(t *testing.T) {
	manager := setupTestManager()
	setupTestArtifacts()
	playerID := createTestPlayer(manager)
	squadID := createTestSquadWithUnits(manager, "Test Squad", 3)
	addArtifactToInventory(playerID, "iron_bulwark", manager)
	addArtifactToInventory(playerID, "berserkers_torc", manager)
	addArtifactToInventory(playerID, "commanders_initiative_badge", manager)
	addArtifactToInventory(playerID, "vanguards_oath", manager)

	// Fill all 3 slots
	if err := EquipArtifact(playerID, squadID, "iron_bulwark", manager); err != nil {
		t.Fatalf("Failed to equip first artifact: %v", err)
	}
	if err := EquipArtifact(playerID, squadID, "berserkers_torc", manager); err != nil {
		t.Fatalf("Failed to equip second artifact: %v", err)
	}
	if err := EquipArtifact(playerID, squadID, "commanders_initiative_badge", manager); err != nil {
		t.Fatalf("Failed to equip third artifact: %v", err)
	}

	// 4th should fail
	err := EquipArtifact(playerID, squadID, "vanguards_oath", manager)
	if err == nil {
		t.Error("Expected error when equipping to full slots")
	}

	data := GetEquipmentData(squadID, manager)
	if len(data.EquippedArtifacts) != MaxArtifactSlots {
		t.Errorf("Expected %d equipped artifacts, got %d", MaxArtifactSlots, len(data.EquippedArtifacts))
	}
}

func TestEquipUnknownArtifact(t *testing.T) {
	manager := setupTestManager()
	setupTestArtifacts()
	playerID := createTestPlayer(manager)
	squadID := createTestSquadWithUnits(manager, "Test Squad", 3)

	err := EquipArtifact(playerID, squadID, "nonexistent_artifact", manager)
	if err == nil {
		t.Error("Expected error for unknown artifact")
	}
}

// ========================================
// OWNERSHIP TESTS
// ========================================

func TestEquipRequiresOwnership(t *testing.T) {
	manager := setupTestManager()
	setupTestArtifacts()
	playerID := createTestPlayer(manager)
	squadID := createTestSquadWithUnits(manager, "Test Squad", 3)

	// Try to equip without adding to inventory first
	err := EquipArtifact(playerID, squadID, "iron_bulwark", manager)
	if err == nil {
		t.Error("Expected error when equipping artifact not in inventory")
	}
}

func TestEquipAlreadyEquippedElsewhere(t *testing.T) {
	manager := setupTestManager()
	setupTestArtifacts()
	playerID := createTestPlayer(manager)
	squad1 := createTestSquadWithUnits(manager, "Squad A", 3)
	squad2 := createTestSquadWithUnits(manager, "Squad B", 3)
	addArtifactToInventory(playerID, "iron_bulwark", manager)

	// Equip on first squad
	err := EquipArtifact(playerID, squad1, "iron_bulwark", manager)
	if err != nil {
		t.Fatalf("Failed to equip on squad 1: %v", err)
	}

	// Try to equip same artifact on second squad
	err = EquipArtifact(playerID, squad2, "iron_bulwark", manager)
	if err == nil {
		t.Error("Expected error when equipping artifact already on another squad")
	}
}

func TestUnequipReturnsToInventory(t *testing.T) {
	manager := setupTestManager()
	setupTestArtifacts()
	playerID := createTestPlayer(manager)
	squadID := createTestSquadWithUnits(manager, "Test Squad", 3)
	addArtifactToInventory(playerID, "iron_bulwark", manager)

	EquipArtifact(playerID, squadID, "iron_bulwark", manager)

	inv := GetPlayerArtifactInventory(playerID, manager)
	if IsArtifactAvailable(inv, "iron_bulwark") {
		t.Error("Artifact should be marked equipped, not available")
	}

	UnequipArtifact(playerID, squadID, "iron_bulwark", manager)

	if !IsArtifactAvailable(inv, "iron_bulwark") {
		t.Error("Artifact should be available after unequip")
	}
}

// ========================================
// APPLY EFFECTS TESTS
// ========================================

func TestApplyArtifactStatEffects_SingleStat(t *testing.T) {
	manager := setupTestManager()
	setupTestArtifacts()
	playerID := createTestPlayer(manager)
	squadID := createTestSquadWithUnits(manager, "Test Squad", 3)
	addArtifactToInventory(playerID, "iron_bulwark", manager)

	EquipArtifact(playerID, squadID, "iron_bulwark", manager)

	unitIDs := squads.GetUnitIDsInSquad(squadID, manager)
	// Record original armor
	origArmor := make(map[ecs.EntityID]int)
	for _, uid := range unitIDs {
		attr := common.GetComponentTypeByID[*common.Attributes](manager, uid, common.AttributeComponent)
		origArmor[uid] = attr.Armor
	}

	ApplyArtifactStatEffects([]ecs.EntityID{squadID}, manager)

	for _, uid := range unitIDs {
		attr := common.GetComponentTypeByID[*common.Attributes](manager, uid, common.AttributeComponent)
		expected := origArmor[uid] + 2
		if attr.Armor != expected {
			t.Errorf("Unit %d: expected armor %d, got %d", uid, expected, attr.Armor)
		}
	}
}

func TestApplyArtifactStatEffects_MultiStat(t *testing.T) {
	manager := setupTestManager()
	setupTestArtifacts()
	playerID := createTestPlayer(manager)
	squadID := createTestSquadWithUnits(manager, "Test Squad", 2)
	addArtifactToInventory(playerID, "berserkers_torc", manager)

	EquipArtifact(playerID, squadID, "berserkers_torc", manager)

	unitIDs := squads.GetUnitIDsInSquad(squadID, manager)
	origStr := make(map[ecs.EntityID]int)
	origArmor := make(map[ecs.EntityID]int)
	for _, uid := range unitIDs {
		attr := common.GetComponentTypeByID[*common.Attributes](manager, uid, common.AttributeComponent)
		origStr[uid] = attr.Strength
		origArmor[uid] = attr.Armor
	}

	ApplyArtifactStatEffects([]ecs.EntityID{squadID}, manager)

	for _, uid := range unitIDs {
		attr := common.GetComponentTypeByID[*common.Attributes](manager, uid, common.AttributeComponent)
		if attr.Strength != origStr[uid]+2 {
			t.Errorf("Unit %d: expected strength %d, got %d", uid, origStr[uid]+2, attr.Strength)
		}
		if attr.Armor != origArmor[uid]-1 {
			t.Errorf("Unit %d: expected armor %d, got %d", uid, origArmor[uid]-1, attr.Armor)
		}
	}
}

func TestApplyArtifactStatEffects_NoArtifact(t *testing.T) {
	manager := setupTestManager()
	setupTestArtifacts()
	squadID := createTestSquadWithUnits(manager, "Test Squad", 3)

	unitIDs := squads.GetUnitIDsInSquad(squadID, manager)
	origArmor := make(map[ecs.EntityID]int)
	for _, uid := range unitIDs {
		attr := common.GetComponentTypeByID[*common.Attributes](manager, uid, common.AttributeComponent)
		origArmor[uid] = attr.Armor
	}

	// Apply with no artifact equipped -- should be no-op
	ApplyArtifactStatEffects([]ecs.EntityID{squadID}, manager)

	for _, uid := range unitIDs {
		attr := common.GetComponentTypeByID[*common.Attributes](manager, uid, common.AttributeComponent)
		if attr.Armor != origArmor[uid] {
			t.Errorf("Unit %d: armor changed unexpectedly from %d to %d", uid, origArmor[uid], attr.Armor)
		}
	}
}

func TestApplyArtifactStatEffects_EmptySquadList(t *testing.T) {
	manager := setupTestManager()
	setupTestArtifacts()

	// Should not panic with empty list
	ApplyArtifactStatEffects([]ecs.EntityID{}, manager)
}

// ========================================
// FACTION QUERY TESTS
// ========================================

func TesthasSpecificArtifactInFaction(t *testing.T) {
	manager := setupTestManager()
	setupTestArtifacts()
	playerID := createTestPlayer(manager)

	squad1 := createTestSquadWithUnits(manager, "Squad A", 2)
	squad2 := createTestSquadWithUnits(manager, "Squad B", 2)
	addArtifactToInventory(playerID, "commanders_initiative_badge", manager)

	EquipArtifact(playerID, squad2, "commanders_initiative_badge", manager)

	squadIDs := []ecs.EntityID{squad1, squad2}

	if !hasSpecificArtifactInFaction(squadIDs, "commanders_initiative_badge", manager) {
		t.Error("Expected to find commanders_initiative_badge in faction squads")
	}

	if hasSpecificArtifactInFaction(squadIDs, "vanguards_oath", manager) {
		t.Error("Should not find vanguards_oath in faction squads")
	}
}

// ========================================
// MULTI-INSTANCE INTEGRATION TESTS
// ========================================

func TestEquipSameArtifactOnTwoSquads(t *testing.T) {
	manager := setupTestManager()
	setupTestArtifacts()
	playerID := createTestPlayer(manager)
	squad1 := createTestSquadWithUnits(manager, "Squad A", 3)
	squad2 := createTestSquadWithUnits(manager, "Squad B", 3)

	// Add two copies of iron_bulwark
	addArtifactToInventory(playerID, "iron_bulwark", manager)
	addArtifactToInventory(playerID, "iron_bulwark", manager)

	// Equip on both squads
	if err := EquipArtifact(playerID, squad1, "iron_bulwark", manager); err != nil {
		t.Fatalf("Failed to equip on squad 1: %v", err)
	}
	if err := EquipArtifact(playerID, squad2, "iron_bulwark", manager); err != nil {
		t.Fatalf("Failed to equip on squad 2: %v", err)
	}

	// Verify both squads have the artifact
	data1 := GetEquipmentData(squad1, manager)
	data2 := GetEquipmentData(squad2, manager)
	if !containsArtifact(data1, "iron_bulwark") {
		t.Error("Squad 1 should have iron_bulwark equipped")
	}
	if !containsArtifact(data2, "iron_bulwark") {
		t.Error("Squad 2 should have iron_bulwark equipped")
	}

	// No more available copies
	inv := GetPlayerArtifactInventory(playerID, manager)
	if IsArtifactAvailable(inv, "iron_bulwark") {
		t.Error("All copies should be equipped")
	}
}

func TestSeedAllArtifacts(t *testing.T) {
	manager := setupTestManager()
	setupTestArtifacts()
	playerID := createTestPlayer(manager)

	inv := GetPlayerArtifactInventory(playerID, manager)
	for id := range templates.ArtifactRegistry {
		for i := 0; i < 3; i++ {
			if err := AddArtifactToInventory(inv, id); err != nil {
				t.Fatalf("Failed to seed artifact %q copy %d: %v", id, i+1, err)
			}
		}
	}

	current, _ := GetArtifactCount(inv)
	expected := len(templates.ArtifactRegistry) * 3
	if current != expected {
		t.Errorf("Expected %d total instances, got %d", expected, current)
	}
}

// ========================================
// ACTIVATION TESTS (via ActivateArtifact)
// ========================================

func TestActivateDoubleTime(t *testing.T) {
	manager := setupTestManager()
	setupTestArtifacts()

	cache := combat.NewCombatQueryCache(manager)
	fm := combat.NewCombatFactionManager(manager, cache)
	factionID := fm.CreateCombatFaction("Player", true)

	squadID := createTestSquadWithUnits(manager, "Test Squad", 3)
	fm.AddSquadToFaction(factionID, squadID, coords.LogicalPosition{X: 5, Y: 5})

	turnMgr := combat.NewTurnManager(manager, cache)
	turnMgr.InitializeCombat([]ecs.EntityID{factionID}, 0)

	charges := NewArtifactChargeTracker()
	ctx := &BehaviorContext{Manager: manager, Cache: cache, ChargeTracker: charges}

	err := ActivateArtifact(BehaviorDoubleTime, squadID, ctx)
	if err != nil {
		t.Fatalf("Failed to activate Double Time: %v", err)
	}

	actionState := cache.FindActionStateBySquadID(squadID)
	if actionState == nil {
		t.Fatal("Expected action state")
	}
	if !actionState.DoubleTimeActive {
		t.Error("Expected DoubleTimeActive to be true")
	}
	if charges.IsAvailable(BehaviorDoubleTime) {
		t.Error("Expected double_time charge to be consumed")
	}

	// Second activation should fail
	err = ActivateArtifact(BehaviorDoubleTime, squadID, ctx)
	if err == nil {
		t.Error("Expected error on second activation")
	}
}

func TestActivateDoubleTime_AlreadyActed(t *testing.T) {
	manager := setupTestManager()
	setupTestArtifacts()

	cache := combat.NewCombatQueryCache(manager)
	fm := combat.NewCombatFactionManager(manager, cache)
	factionID := fm.CreateCombatFaction("Player", true)

	squadID := createTestSquadWithUnits(manager, "Test Squad", 3)
	fm.AddSquadToFaction(factionID, squadID, coords.LogicalPosition{X: 5, Y: 5})

	turnMgr := combat.NewTurnManager(manager, cache)
	turnMgr.InitializeCombat([]ecs.EntityID{factionID}, 0)

	actionState := cache.FindActionStateBySquadID(squadID)
	if actionState == nil {
		t.Fatal("Expected action state")
	}
	actionState.HasActed = true

	charges := NewArtifactChargeTracker()
	ctx := &BehaviorContext{Manager: manager, Cache: cache, ChargeTracker: charges}

	err := ActivateArtifact(BehaviorDoubleTime, squadID, ctx)
	if err == nil {
		t.Error("Expected error when activating Double Time on already-acted squad")
	}
	if !charges.IsAvailable(BehaviorDoubleTime) {
		t.Error("Expected double_time charge to still be available after failed activation")
	}
}

func TestActivateSaboteursHourglass(t *testing.T) {
	charges := NewArtifactChargeTracker()
	ctx := &BehaviorContext{ChargeTracker: charges}

	err := ActivateArtifact(BehaviorSaboteurWsHourglass, 0, ctx)
	if err != nil {
		t.Fatalf("Failed to activate Saboteur's Hourglass: %v", err)
	}

	consumed := charges.ConsumePendingEffects(BehaviorSaboteurWsHourglass)
	if len(consumed) != 1 {
		t.Errorf("Expected 1 pending effect, got %d", len(consumed))
	}
	if charges.IsAvailable(BehaviorSaboteurWsHourglass) {
		t.Error("Expected saboteurs_hourglass charge to be consumed")
	}

	err = ActivateArtifact(BehaviorSaboteurWsHourglass, 0, ctx)
	if err == nil {
		t.Error("Expected error on second activation")
	}
}

// ========================================
// NEW BEHAVIOR TESTS
// ========================================

// setupCombatContext creates a standard combat test context with one faction and one squad.
func setupCombatContext(manager *common.EntityManager, squadName string, unitCount int, pos coords.LogicalPosition) (
	cache *combat.CombatQueryCache,
	factionID ecs.EntityID,
	squadID ecs.EntityID,
	charges *ArtifactChargeTracker,
	ctx *BehaviorContext,
) {
	cache = combat.NewCombatQueryCache(manager)
	fm := combat.NewCombatFactionManager(manager, cache)
	factionID = fm.CreateCombatFaction("Player", true)
	squadID = createTestSquadWithUnits(manager, squadName, unitCount)
	fm.AddSquadToFaction(factionID, squadID, pos)
	charges = NewArtifactChargeTracker()
	ctx = &BehaviorContext{Manager: manager, Cache: cache, ChargeTracker: charges}
	return
}

func TestStandDown_SkipsAttack(t *testing.T) {
	manager := setupTestManager()
	setupTestArtifacts()

	cache := combat.NewCombatQueryCache(manager)
	fm := combat.NewCombatFactionManager(manager, cache)

	playerFaction := fm.CreateCombatFaction("Player", true)
	enemyFaction := fm.CreateCombatFaction("Enemy", false)

	playerSquad := createTestSquadWithUnits(manager, "Player Squad", 3)
	enemySquad := createTestSquadWithUnits(manager, "Enemy Squad", 3)

	fm.AddSquadToFaction(playerFaction, playerSquad, coords.LogicalPosition{X: 1, Y: 1})
	fm.AddSquadToFaction(enemyFaction, enemySquad, coords.LogicalPosition{X: 5, Y: 5})

	turnMgr := combat.NewTurnManager(manager, cache)
	turnMgr.InitializeCombat([]ecs.EntityID{playerFaction, enemyFaction}, 0)

	charges := NewArtifactChargeTracker()
	ctx := &BehaviorContext{Manager: manager, Cache: cache, ChargeTracker: charges}

	// Activate Stand Down targeting enemy squad
	err := ActivateArtifact(BehaviorStandDown, enemySquad, ctx)
	if err != nil {
		t.Fatalf("Failed to activate Stand Down: %v", err)
	}

	// Simulate enemy faction's PostReset
	b := GetBehavior(BehaviorStandDown)
	b.OnPostReset(ctx, enemyFaction, []ecs.EntityID{enemySquad})

	// Verify enemy squad's HasActed is true
	actionState := cache.FindActionStateBySquadID(enemySquad)
	if actionState == nil {
		t.Fatal("Expected action state for enemy squad")
	}
	if !actionState.HasActed {
		t.Error("Expected enemy squad HasActed = true after Stand Down")
	}

	// Verify charge consumed
	if charges.IsAvailable(BehaviorStandDown) {
		t.Error("Expected stand_down charge to be consumed")
	}
}

func TestDeadlockShackles_SkipsActivation(t *testing.T) {
	manager := setupTestManager()
	setupTestArtifacts()

	cache := combat.NewCombatQueryCache(manager)
	fm := combat.NewCombatFactionManager(manager, cache)

	playerFaction := fm.CreateCombatFaction("Player", true)
	enemyFaction := fm.CreateCombatFaction("Enemy", false)

	playerSquad := createTestSquadWithUnits(manager, "Player Squad", 3)
	enemySquad := createTestSquadWithUnits(manager, "Enemy Squad", 3)

	fm.AddSquadToFaction(playerFaction, playerSquad, coords.LogicalPosition{X: 1, Y: 1})
	fm.AddSquadToFaction(enemyFaction, enemySquad, coords.LogicalPosition{X: 5, Y: 5})

	turnMgr := combat.NewTurnManager(manager, cache)
	turnMgr.InitializeCombat([]ecs.EntityID{playerFaction, enemyFaction}, 0)

	charges := NewArtifactChargeTracker()
	ctx := &BehaviorContext{Manager: manager, Cache: cache, ChargeTracker: charges}

	// Activate Deadlock Shackles targeting enemy squad
	err := ActivateArtifact(BehaviorDeadlockShackles, enemySquad, ctx)
	if err != nil {
		t.Fatalf("Failed to activate Deadlock Shackles: %v", err)
	}

	// Simulate enemy faction's PostReset
	b := GetBehavior(BehaviorDeadlockShackles)
	b.OnPostReset(ctx, enemyFaction, []ecs.EntityID{enemySquad})

	actionState := cache.FindActionStateBySquadID(enemySquad)
	if actionState == nil {
		t.Fatal("Expected action state for enemy squad")
	}
	if !actionState.HasActed {
		t.Error("Expected HasActed = true")
	}
	if !actionState.HasMoved {
		t.Error("Expected HasMoved = true")
	}
	if actionState.MovementRemaining != 0 {
		t.Errorf("Expected MovementRemaining = 0, got %d", actionState.MovementRemaining)
	}
}

func TestAnthemPerseverance_BonusAttack(t *testing.T) {
	manager := setupTestManager()
	setupTestArtifacts()

	cache, factionID, squadID, charges, ctx := setupCombatContext(
		manager, "Test Squad", 3, coords.LogicalPosition{X: 5, Y: 5})

	turnMgr := combat.NewTurnManager(manager, cache)
	turnMgr.InitializeCombat([]ecs.EntityID{factionID}, 0)

	// Mark squad as acted
	actionState := cache.FindActionStateBySquadID(squadID)
	if actionState == nil {
		t.Fatal("Expected action state")
	}
	actionState.HasActed = true

	err := ActivateArtifact(BehaviorAnthemPerseverance, squadID, ctx)
	if err != nil {
		t.Fatalf("Failed to activate Anthem of Perseverance: %v", err)
	}

	if actionState.HasActed {
		t.Error("Expected HasActed = false after Anthem of Perseverance")
	}
	if charges.IsAvailable(BehaviorAnthemPerseverance) {
		t.Error("Expected anthem_perseverance charge to be consumed")
	}
}

func TestRallyingHorn_BonusOnDefense(t *testing.T) {
	manager := setupTestManager()
	setupTestArtifacts()
	playerID := createTestPlayer(manager)

	// Register rallying horn artifact for equipping
	templates.ArtifactRegistry["rallying_war_horn"] = &templates.ArtifactDefinition{
		ID:       "rallying_war_horn",
		Name:     "Rallying War Horn",
		Tier:     "major",
		Behavior: BehaviorRallyingHorn,
	}

	cache := combat.NewCombatQueryCache(manager)
	fm := combat.NewCombatFactionManager(manager, cache)

	defenderFaction := fm.CreateCombatFaction("Defender", true)
	attackerFaction := fm.CreateCombatFaction("Attacker", false)

	defenderSquad1 := createTestSquadWithUnits(manager, "Def Squad 1", 3)
	defenderSquad2 := createTestSquadWithUnits(manager, "Def Squad 2", 3)
	attackerSquad := createTestSquadWithUnits(manager, "Atk Squad", 3)

	fm.AddSquadToFaction(defenderFaction, defenderSquad1, coords.LogicalPosition{X: 1, Y: 1})
	fm.AddSquadToFaction(defenderFaction, defenderSquad2, coords.LogicalPosition{X: 2, Y: 2})
	fm.AddSquadToFaction(attackerFaction, attackerSquad, coords.LogicalPosition{X: 5, Y: 5})

	// Equip rallying horn on defender squad 1
	addArtifactToInventory(playerID, "rallying_war_horn", manager)
	EquipArtifact(playerID, defenderSquad1, "rallying_war_horn", manager)

	turnMgr := combat.NewTurnManager(manager, cache)
	turnMgr.InitializeCombat([]ecs.EntityID{defenderFaction, attackerFaction}, 0)

	// Mark defender squad 2 as already acted (so it can be rallied)
	as2 := cache.FindActionStateBySquadID(defenderSquad2)
	if as2 == nil {
		t.Fatal("Expected action state for defender squad 2")
	}
	as2.HasActed = true
	as2.HasMoved = true

	charges := NewArtifactChargeTracker()
	ctx := &BehaviorContext{Manager: manager, Cache: cache, ChargeTracker: charges}

	// Simulate an attack on the defender
	combatResult := &squads.CombatResult{
		Success:         true,
		TargetDestroyed: false,
	}

	b := GetBehavior(BehaviorRallyingHorn)
	b.OnAttackComplete(ctx, attackerSquad, defenderSquad1, combatResult)

	// Verify defender squad 2 got bonus activation
	if as2.HasActed {
		t.Error("Expected defender squad 2 HasActed = false after rallying horn")
	}
	if as2.HasMoved {
		t.Error("Expected defender squad 2 HasMoved = false after rallying horn")
	}
	if charges.IsAvailable(BehaviorRallyingHorn) {
		t.Error("Expected rallying_horn charge to be consumed")
	}
}

func TestChainOfCommand_PassAttack(t *testing.T) {
	manager := setupTestManager()
	setupTestArtifacts()
	playerID := createTestPlayer(manager)

	// Register chain of command artifact
	templates.ArtifactRegistry["chain_of_command_scepter"] = &templates.ArtifactDefinition{
		ID:       "chain_of_command_scepter",
		Name:     "Chain of Command Scepter",
		Tier:     "major",
		Behavior: BehaviorChainOfCommand,
	}

	cache := combat.NewCombatQueryCache(manager)
	fm := combat.NewCombatFactionManager(manager, cache)
	factionID := fm.CreateCombatFaction("Player", true)

	sourceSquad := createTestSquadWithUnits(manager, "Source Squad", 3)
	targetSquad := createTestSquadWithUnits(manager, "Target Squad", 3)

	// Place squads adjacent (within Chebyshev distance 2)
	fm.AddSquadToFaction(factionID, sourceSquad, coords.LogicalPosition{X: 5, Y: 5})
	fm.AddSquadToFaction(factionID, targetSquad, coords.LogicalPosition{X: 6, Y: 6})

	// Equip chain of command on source squad
	addArtifactToInventory(playerID, "chain_of_command_scepter", manager)
	EquipArtifact(playerID, sourceSquad, "chain_of_command_scepter", manager)

	turnMgr := combat.NewTurnManager(manager, cache)
	turnMgr.InitializeCombat([]ecs.EntityID{factionID}, 0)

	// Mark target as acted (so it can receive the passed attack)
	targetState := cache.FindActionStateBySquadID(targetSquad)
	if targetState == nil {
		t.Fatal("Expected action state for target squad")
	}
	targetState.HasActed = true

	charges := NewArtifactChargeTracker()
	ctx := &BehaviorContext{Manager: manager, Cache: cache, ChargeTracker: charges}

	err := ActivateArtifact(BehaviorChainOfCommand, targetSquad, ctx)
	if err != nil {
		t.Fatalf("Failed to activate Chain of Command: %v", err)
	}

	// Verify source squad has acted (gave up its attack)
	sourceState := cache.FindActionStateBySquadID(sourceSquad)
	if sourceState == nil {
		t.Fatal("Expected action state for source squad")
	}
	if !sourceState.HasActed {
		t.Error("Expected source squad HasActed = true after passing attack")
	}

	// Verify target squad can act again
	if targetState.HasActed {
		t.Error("Expected target squad HasActed = false after receiving attack")
	}

	if charges.IsAvailable(BehaviorChainOfCommand) {
		t.Error("Expected chain_of_command charge to be consumed")
	}
}

// ========================================
// CLEANUP INTEGRATION TEST
// ========================================

func TestArtifactStatEffectsCleanedByRemoveAll(t *testing.T) {
	manager := setupTestManager()
	setupTestArtifacts()
	playerID := createTestPlayer(manager)
	squadID := createTestSquadWithUnits(manager, "Test Squad", 2)
	addArtifactToInventory(playerID, "iron_bulwark", manager)

	EquipArtifact(playerID, squadID, "iron_bulwark", manager)

	unitIDs := squads.GetUnitIDsInSquad(squadID, manager)
	origArmor := make(map[ecs.EntityID]int)
	for _, uid := range unitIDs {
		attr := common.GetComponentTypeByID[*common.Attributes](manager, uid, common.AttributeComponent)
		origArmor[uid] = attr.Armor
	}

	ApplyArtifactStatEffects([]ecs.EntityID{squadID}, manager)

	// Verify effects applied
	for _, uid := range unitIDs {
		attr := common.GetComponentTypeByID[*common.Attributes](manager, uid, common.AttributeComponent)
		if attr.Armor != origArmor[uid]+2 {
			t.Fatalf("Setup failed: armor not applied correctly")
		}
	}

	// Remove all effects (same as combat cleanup)
	for _, uid := range unitIDs {
		effects.RemoveAllEffects(uid, manager)
	}

	// Verify effects reversed
	for _, uid := range unitIDs {
		attr := common.GetComponentTypeByID[*common.Attributes](manager, uid, common.AttributeComponent)
		if attr.Armor != origArmor[uid] {
			t.Errorf("Unit %d: expected armor %d after cleanup, got %d", uid, origArmor[uid], attr.Armor)
		}
	}
}
