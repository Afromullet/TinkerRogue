package perks

import (
	"game_main/common"
	"game_main/tactical/effects"
	"game_main/tactical/squads"
	testfx "game_main/testing"
	"testing"

	"github.com/bytearena/ecs"
)

// ========================================
// TEST SETUP HELPERS
// ========================================

func setupTestManager(t *testing.T) *common.EntityManager {
	manager := testfx.NewTestEntityManager()
	populateTestPerkRegistry()
	return manager
}

// populateTestPerkRegistry loads perk definitions directly for testing
// (avoids file system dependency on perkdata.json).
func populateTestPerkRegistry() {
	if len(PerkRegistry) > 0 {
		return // Already populated
	}

	PerkRegistry["iron_constitution"] = &PerkDefinition{
		ID: "iron_constitution", Name: "Iron Constitution", Level: PerkLevelUnit,
		StatModifiers: []PerkStatModifier{{Stat: "strength", Percent: 0.2}},
	}
	PerkRegistry["hardened_armor"] = &PerkDefinition{
		ID: "hardened_armor", Name: "Hardened Armor", Level: PerkLevelUnit,
		StatModifiers: []PerkStatModifier{{Stat: "armor", Modifier: 4}},
	}
	PerkRegistry["arcane_attunement"] = &PerkDefinition{
		ID: "arcane_attunement", Name: "Arcane Attunement", Level: PerkLevelUnit,
		StatModifiers: []PerkStatModifier{{Stat: "magic", Modifier: 5}},
	}
	PerkRegistry["fleet_footed"] = &PerkDefinition{
		ID: "fleet_footed", Name: "Fleet Footed", Level: PerkLevelUnit,
		StatModifiers: []PerkStatModifier{{Stat: "movementspeed", Modifier: 1}},
	}
	PerkRegistry["riposte"] = &PerkDefinition{
		ID: "riposte", Name: "Riposte", Level: PerkLevelUnit,
		BehaviorID: "riposte", ExclusiveWith: []string{"stone_wall"},
	}
	PerkRegistry["stone_wall"] = &PerkDefinition{
		ID: "stone_wall", Name: "Stone Wall", Level: PerkLevelUnit,
		BehaviorID: "stone_wall", ExclusiveWith: []string{"riposte"},
	}
	PerkRegistry["berserker"] = &PerkDefinition{
		ID: "berserker", Name: "Berserker", Level: PerkLevelUnit,
		BehaviorID: "berserker", RoleGate: "DPS",
	}
	PerkRegistry["armor_piercing"] = &PerkDefinition{
		ID: "armor_piercing", Name: "Armor Piercing", Level: PerkLevelUnit,
		BehaviorID: "armor_piercing",
	}
	PerkRegistry["glass_cannon"] = &PerkDefinition{
		ID: "glass_cannon", Name: "Glass Cannon", Level: PerkLevelSquad,
		BehaviorID: "glass_cannon",
	}
	PerkRegistry["focus_fire"] = &PerkDefinition{
		ID: "focus_fire", Name: "Focus Fire", Level: PerkLevelUnit,
		BehaviorID: "focus_fire", ExclusiveWith: []string{"cleave"},
	}
	PerkRegistry["cleave"] = &PerkDefinition{
		ID: "cleave", Name: "Cleave", Level: PerkLevelUnit,
		BehaviorID: "cleave", ExclusiveWith: []string{"focus_fire"},
	}
	PerkRegistry["lifesteal"] = &PerkDefinition{
		ID: "lifesteal", Name: "Lifesteal", Level: PerkLevelUnit,
		BehaviorID: "lifesteal",
	}
	PerkRegistry["inspiration"] = &PerkDefinition{
		ID: "inspiration", Name: "Inspiration", Level: PerkLevelUnit,
		BehaviorID: "inspiration",
	}
	PerkRegistry["impale"] = &PerkDefinition{
		ID: "impale", Name: "Impale", Level: PerkLevelUnit,
		BehaviorID: "impale",
	}
	PerkRegistry["war_medic"] = &PerkDefinition{
		ID: "war_medic", Name: "War Medic", Level: PerkLevelUnit,
		BehaviorID: "war_medic", RoleGate: "Support",
	}
}

// createTestSquadWithUnit creates a squad at (0,0) with a single unit.
// Returns (squadID, unitID).
func createTestSquadWithUnit(manager *common.EntityManager, strength, dexterity, health int) (ecs.EntityID, ecs.EntityID) {
	squad := manager.World.NewEntity()
	squadID := squad.GetID()

	squad.AddComponent(squads.SquadComponent, &squads.SquadData{
		SquadID:    squadID,
		Name:       "TestSquad",
		Morale:     100,
		SquadLevel: 1,
		MaxUnits:   9,
	})

	unit := manager.World.NewEntity()
	unitID := unit.GetID()

	unit.AddComponent(squads.SquadMemberComponent, &squads.SquadMemberData{SquadID: squadID})
	unit.AddComponent(squads.GridPositionComponent, &squads.GridPositionData{
		AnchorRow: 0, AnchorCol: 0, Width: 1, Height: 1,
	})
	unit.AddComponent(common.NameComponent, &common.Name{NameStr: "TestUnit"})

	attr := common.NewAttributes(strength, dexterity, 0, 0, 2, 2)
	attr.CurrentHealth = health
	unit.AddComponent(common.AttributeComponent, &attr)

	unit.AddComponent(squads.TargetRowComponent, &squads.TargetRowData{
		AttackType: squads.AttackTypeMeleeRow,
	})
	unit.AddComponent(squads.AttackRangeComponent, &squads.AttackRangeData{Range: 1})
	unit.AddComponent(squads.UnitRoleComponent, &squads.UnitRoleData{Role: squads.RoleDPS})

	return squadID, unitID
}

// addPerkComponent adds a UnitPerkData component to a unit.
func addPerkComponent(manager *common.EntityManager, unitID ecs.EntityID, perks [2]string) {
	entity := manager.FindEntityByID(unitID)
	if entity != nil {
		entity.AddComponent(UnitPerkComponent, &UnitPerkData{EquippedPerks: perks})
	}
}

// addSquadPerkComponent adds a SquadPerkData component to a squad.
func addSquadPerkComponent(manager *common.EntityManager, squadID ecs.EntityID, perks [3]string) {
	entity := manager.FindEntityByID(squadID)
	if entity != nil {
		entity.AddComponent(SquadPerkComponent, &SquadPerkData{EquippedPerks: perks})
	}
}

// ========================================
// STAT PERK TESTS
// ========================================

func TestApplyStatPerks_FlatModifier(t *testing.T) {
	manager := setupTestManager(t)
	_, unitID := createTestSquadWithUnit(manager, 10, 5, 40)

	// Apply hardened_armor (+4 armor)
	ApplyStatPerks(unitID, []string{"hardened_armor"}, manager)

	entity := manager.FindEntityByID(unitID)
	attr := common.GetComponentType[*common.Attributes](entity, common.AttributeComponent)
	if attr.Armor != 6 { // 2 base + 4
		t.Errorf("expected armor 6, got %d", attr.Armor)
	}
}

func TestApplyStatPerks_PercentModifier(t *testing.T) {
	manager := setupTestManager(t)
	_, unitID := createTestSquadWithUnit(manager, 10, 5, 40)

	// Apply iron_constitution (+20% strength)
	ApplyStatPerks(unitID, []string{"iron_constitution"}, manager)

	entity := manager.FindEntityByID(unitID)
	attr := common.GetComponentType[*common.Attributes](entity, common.AttributeComponent)
	// 10 str * 0.2 = +2 str
	if attr.Strength != 12 {
		t.Errorf("expected strength 12, got %d", attr.Strength)
	}
}

func TestRemoveStatPerks(t *testing.T) {
	manager := setupTestManager(t)
	_, unitID := createTestSquadWithUnit(manager, 10, 5, 40)

	// Apply then remove
	ApplyStatPerks(unitID, []string{"hardened_armor"}, manager)
	RemoveStatPerks(unitID, manager)

	entity := manager.FindEntityByID(unitID)
	attr := common.GetComponentType[*common.Attributes](entity, common.AttributeComponent)
	if attr.Armor != 2 { // Back to base
		t.Errorf("expected armor 2 after removal, got %d", attr.Armor)
	}
}

func TestApplyStatPerks_DeadUnitSkipped(t *testing.T) {
	manager := setupTestManager(t)
	_, unitID := createTestSquadWithUnit(manager, 10, 5, 0) // 0 HP = dead

	ApplyStatPerks(unitID, []string{"hardened_armor"}, manager)

	entity := manager.FindEntityByID(unitID)
	attr := common.GetComponentType[*common.Attributes](entity, common.AttributeComponent)
	if attr.Armor != 2 { // Should not have changed
		t.Errorf("expected armor 2 on dead unit, got %d", attr.Armor)
	}
}

// ========================================
// PERK QUERY TESTS
// ========================================

func TestHasPerk(t *testing.T) {
	manager := setupTestManager(t)
	_, unitID := createTestSquadWithUnit(manager, 10, 5, 40)
	addPerkComponent(manager, unitID, [2]string{"berserker", ""})

	if !HasPerk(unitID, "berserker", manager) {
		t.Error("expected unit to have berserker perk")
	}

	if HasPerk(unitID, "lifesteal", manager) {
		t.Error("unit should not have lifesteal perk")
	}
}

func TestGetActivePerkIDs_IncludesSquadPerks(t *testing.T) {
	manager := setupTestManager(t)
	squadID, unitID := createTestSquadWithUnit(manager, 10, 5, 40)

	addPerkComponent(manager, unitID, [2]string{"berserker", ""})
	addSquadPerkComponent(manager, squadID, [3]string{"glass_cannon", "", ""})

	ids := getActivePerkIDs(unitID, manager)
	if len(ids) != 2 {
		t.Errorf("expected 2 active perk IDs, got %d: %v", len(ids), ids)
	}

	foundBerserker := false
	foundGlassCannon := false
	for _, id := range ids {
		if id == "berserker" {
			foundBerserker = true
		}
		if id == "glass_cannon" {
			foundGlassCannon = true
		}
	}
	if !foundBerserker || !foundGlassCannon {
		t.Errorf("missing expected perk IDs in %v", ids)
	}
}

func TestGetEquippedPerks(t *testing.T) {
	manager := setupTestManager(t)
	_, unitID := createTestSquadWithUnit(manager, 10, 5, 40)
	addPerkComponent(manager, unitID, [2]string{"berserker", "lifesteal"})

	equipped := GetEquippedPerks(unitID, manager)
	if len(equipped) != 2 {
		t.Errorf("expected 2 equipped perks, got %d", len(equipped))
	}
}

// ========================================
// HOOK RUNNER TESTS
// ========================================

func TestRunDamageModHooks_Berserker(t *testing.T) {
	manager := setupTestManager(t)
	_, unitID := createTestSquadWithUnit(manager, 10, 5, 10) // 10 HP, MaxHP=40
	addPerkComponent(manager, unitID, [2]string{"berserker", ""})

	modifiers := squads.DamageModifiers{DamageMultiplier: 1.0}
	RunDamageModHooks(unitID, 0, &modifiers, manager)

	// 10/40 = 25% HP, below 50% threshold => +30%
	if modifiers.DamageMultiplier < 1.29 || modifiers.DamageMultiplier > 1.31 {
		t.Errorf("expected ~1.3 damage multiplier, got %f", modifiers.DamageMultiplier)
	}
}

func TestRunDamageModHooks_Berserker_AboveThreshold(t *testing.T) {
	manager := setupTestManager(t)
	_, unitID := createTestSquadWithUnit(manager, 10, 5, 30) // 30 HP, MaxHP=40
	addPerkComponent(manager, unitID, [2]string{"berserker", ""})

	modifiers := squads.DamageModifiers{DamageMultiplier: 1.0}
	RunDamageModHooks(unitID, 0, &modifiers, manager)

	// 30/40 = 75% HP, above 50% threshold => no bonus
	if modifiers.DamageMultiplier != 1.0 {
		t.Errorf("expected 1.0 damage multiplier, got %f", modifiers.DamageMultiplier)
	}
}

func TestRunCounterModHooks_Riposte(t *testing.T) {
	manager := setupTestManager(t)
	_, unitID := createTestSquadWithUnit(manager, 10, 5, 40)
	addPerkComponent(manager, unitID, [2]string{"riposte", ""})

	modifiers := squads.DamageModifiers{
		HitPenalty:       20,
		DamageMultiplier: 0.5,
		IsCounterattack:  true,
	}
	skipCounter := RunCounterModHooks(unitID, 0, &modifiers, manager)

	if skipCounter {
		t.Error("riposte should not skip counter")
	}
	if modifiers.DamageMultiplier != 1.0 {
		t.Errorf("expected 1.0 damage multiplier for riposte, got %f", modifiers.DamageMultiplier)
	}
	if modifiers.HitPenalty != 0 {
		t.Errorf("expected 0 hit penalty for riposte, got %d", modifiers.HitPenalty)
	}
}

func TestRunCounterModHooks_StoneWall(t *testing.T) {
	manager := setupTestManager(t)
	_, unitID := createTestSquadWithUnit(manager, 10, 5, 40)
	addPerkComponent(manager, unitID, [2]string{"stone_wall", ""})

	modifiers := squads.DamageModifiers{
		HitPenalty:       20,
		DamageMultiplier: 0.5,
		IsCounterattack:  true,
	}
	skipCounter := RunCounterModHooks(unitID, 0, &modifiers, manager)

	if !skipCounter {
		t.Error("stone_wall should skip counter")
	}
}

func TestRunDamageModHooks_ArmorPiercing(t *testing.T) {
	manager := setupTestManager(t)
	_, unitID := createTestSquadWithUnit(manager, 10, 5, 40)
	addPerkComponent(manager, unitID, [2]string{"armor_piercing", ""})

	modifiers := squads.DamageModifiers{DamageMultiplier: 1.0}
	RunDamageModHooks(unitID, 0, &modifiers, manager)

	if modifiers.ArmorReduction != 0.5 {
		t.Errorf("expected 0.5 armor reduction, got %f", modifiers.ArmorReduction)
	}
}

func TestRunDamageModHooks_FocusFire(t *testing.T) {
	manager := setupTestManager(t)
	_, unitID := createTestSquadWithUnit(manager, 10, 5, 40)
	addPerkComponent(manager, unitID, [2]string{"focus_fire", ""})

	modifiers := squads.DamageModifiers{DamageMultiplier: 1.0}
	RunDamageModHooks(unitID, 0, &modifiers, manager)

	if modifiers.DamageMultiplier != 2.0 {
		t.Errorf("expected 2.0 damage multiplier, got %f", modifiers.DamageMultiplier)
	}
}

func TestRunTargetOverrideHooks_FocusFire(t *testing.T) {
	manager := setupTestManager(t)
	_, unitID := createTestSquadWithUnit(manager, 10, 5, 40)
	addPerkComponent(manager, unitID, [2]string{"focus_fire", ""})

	targets := []ecs.EntityID{1, 2, 3}
	result := RunTargetOverrideHooks(unitID, 0, targets, manager)

	if len(result) != 1 {
		t.Errorf("expected 1 target from focus fire, got %d", len(result))
	}
}

// ========================================
// EXCLUSIVE PERK TESTS
// ========================================

func TestCanEquipPerk_ExclusiveWithConflict(t *testing.T) {
	manager := setupTestManager(t)
	_, unitID := createTestSquadWithUnit(manager, 10, 5, 40)
	addPerkComponent(manager, unitID, [2]string{"riposte", ""})

	reason := CanEquipPerk(unitID, "stone_wall", 1, manager)
	if reason == "" {
		t.Error("expected exclusive conflict reason, got empty string")
	}
}

func TestCanEquipPerk_NoConflict(t *testing.T) {
	manager := setupTestManager(t)
	_, unitID := createTestSquadWithUnit(manager, 10, 5, 40)
	addPerkComponent(manager, unitID, [2]string{"berserker", ""})

	reason := CanEquipPerk(unitID, "lifesteal", 1, manager)
	if reason != "" {
		t.Errorf("expected no conflict, got: %s", reason)
	}
}

func TestCanEquipPerk_RoleGate(t *testing.T) {
	manager := setupTestManager(t)
	_, unitID := createTestSquadWithUnit(manager, 10, 5, 40) // Role is DPS
	addPerkComponent(manager, unitID, [2]string{"", ""})

	// war_medic requires Support role
	reason := CanEquipPerk(unitID, "war_medic", 0, manager)
	if reason == "" {
		t.Error("expected role gate rejection for war_medic on DPS unit")
	}
}

// ========================================
// MULTIPLICATIVE STACKING TESTS
// ========================================

func TestDamageModStacking_GlassCannonPlusBerserker(t *testing.T) {
	manager := setupTestManager(t)
	squadID, unitID := createTestSquadWithUnit(manager, 10, 5, 10) // Low HP for berserker

	addPerkComponent(manager, unitID, [2]string{"berserker", ""})
	addSquadPerkComponent(manager, squadID, [3]string{"glass_cannon", "", ""})

	modifiers := squads.DamageModifiers{DamageMultiplier: 1.0}
	RunDamageModHooks(unitID, 0, &modifiers, manager)

	// Glass cannon: *1.35, Berserker: *1.3 = 1.755
	expected := 1.35 * 1.3
	if modifiers.DamageMultiplier < expected-0.01 || modifiers.DamageMultiplier > expected+0.01 {
		t.Errorf("expected ~%.3f damage multiplier, got %f", expected, modifiers.DamageMultiplier)
	}
}

// ========================================
// POST-DAMAGE HOOK TESTS
// ========================================

func TestLifestealPostDamage(t *testing.T) {
	manager := setupTestManager(t)
	_, unitID := createTestSquadWithUnit(manager, 10, 5, 30) // 30 HP, MaxHP=40
	addPerkComponent(manager, unitID, [2]string{"lifesteal", ""})

	RunPostDamageHooks(unitID, 0, 20, false, manager)

	entity := manager.FindEntityByID(unitID)
	attr := common.GetComponentType[*common.Attributes](entity, common.AttributeComponent)

	// 20 damage * 25% = 5 HP healed. 30 + 5 = 35
	if attr.CurrentHealth != 35 {
		t.Errorf("expected 35 HP after lifesteal, got %d", attr.CurrentHealth)
	}
}

func TestLifestealPostDamage_CapsAtMaxHP(t *testing.T) {
	manager := setupTestManager(t)
	_, unitID := createTestSquadWithUnit(manager, 10, 5, 39) // 39 HP, MaxHP=40
	addPerkComponent(manager, unitID, [2]string{"lifesteal", ""})

	RunPostDamageHooks(unitID, 0, 100, false, manager)

	entity := manager.FindEntityByID(unitID)
	attr := common.GetComponentType[*common.Attributes](entity, common.AttributeComponent)

	if attr.CurrentHealth != 40 {
		t.Errorf("expected 40 HP (capped), got %d", attr.CurrentHealth)
	}
}

func TestInspirationPostDamage_OnKill(t *testing.T) {
	manager := setupTestManager(t)
	squadID, unitID := createTestSquadWithUnit(manager, 10, 5, 40)
	addPerkComponent(manager, unitID, [2]string{"inspiration", ""})

	// Create a second unit in same squad to receive the buff
	unit2 := manager.World.NewEntity()
	unit2.AddComponent(squads.SquadMemberComponent, &squads.SquadMemberData{SquadID: squadID})
	unit2.AddComponent(squads.GridPositionComponent, &squads.GridPositionData{
		AnchorRow: 1, AnchorCol: 0, Width: 1, Height: 1,
	})
	attr2 := common.NewAttributes(10, 5, 0, 0, 2, 2)
	unit2.AddComponent(common.AttributeComponent, &attr2)

	RunPostDamageHooks(unitID, 999, 10, true, manager) // wasKill=true

	// Check unit2 got the buff
	if !unit2.HasComponent(effects.ActiveEffectsComponent) {
		t.Fatal("expected unit2 to have active effects from inspiration")
	}

	effectsData := common.GetComponentType[*effects.ActiveEffectsData](unit2, effects.ActiveEffectsComponent)
	found := false
	for _, e := range effectsData.Effects {
		if e.Name == "Inspiration" && e.Stat == effects.StatStrength && e.Modifier == 2 {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected Inspiration effect on unit2")
	}
}

// ========================================
// EQUIP/UNEQUIP TESTS
// ========================================

func TestEquipPerk_AppliesStatEffects(t *testing.T) {
	manager := setupTestManager(t)
	_, unitID := createTestSquadWithUnit(manager, 10, 5, 40)
	entity := manager.FindEntityByID(unitID)
	entity.AddComponent(UnitPerkComponent, &UnitPerkData{})

	err := EquipPerk(unitID, "hardened_armor", 0, manager)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	attr := common.GetComponentType[*common.Attributes](entity, common.AttributeComponent)
	if attr.Armor != 6 {
		t.Errorf("expected armor 6 after equipping hardened_armor, got %d", attr.Armor)
	}
}

func TestUnequipPerk_ReversesStatEffects(t *testing.T) {
	manager := setupTestManager(t)
	_, unitID := createTestSquadWithUnit(manager, 10, 5, 40)
	entity := manager.FindEntityByID(unitID)
	entity.AddComponent(UnitPerkComponent, &UnitPerkData{})

	EquipPerk(unitID, "hardened_armor", 0, manager)
	err := UnequipPerk(unitID, PerkLevelUnit, 0, manager)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	attr := common.GetComponentType[*common.Attributes](entity, common.AttributeComponent)
	if attr.Armor != 2 {
		t.Errorf("expected armor 2 after unequipping, got %d", attr.Armor)
	}
}

func TestEquipPerk_UnknownPerk(t *testing.T) {
	manager := setupTestManager(t)
	_, unitID := createTestSquadWithUnit(manager, 10, 5, 40)
	entity := manager.FindEntityByID(unitID)
	entity.AddComponent(UnitPerkComponent, &UnitPerkData{})

	err := EquipPerk(unitID, "nonexistent_perk", 0, manager)
	if err == nil {
		t.Error("expected error for unknown perk")
	}
}

// ========================================
// HOOK REGISTRY TESTS
// ========================================

func TestHookRegistry_RegisteredPerks(t *testing.T) {
	expectedPerks := []string{
		"riposte", "stone_wall", "berserker", "armor_piercing",
		"glass_cannon", "focus_fire", "cleave", "lifesteal",
		"inspiration", "impale", "war_medic",
	}

	for _, perkID := range expectedPerks {
		hooks := GetPerkHooks(perkID)
		if hooks == nil {
			t.Errorf("expected hooks registered for perk '%s'", perkID)
		}
	}
}

func TestHookRegistry_UnregisteredPerk(t *testing.T) {
	hooks := GetPerkHooks("nonexistent")
	if hooks != nil {
		t.Error("expected nil for unregistered perk")
	}
}

// ========================================
// COVER MOD TESTS
// ========================================

func TestImpaleCoverMod_MeleeColumn(t *testing.T) {
	manager := setupTestManager(t)
	_, unitID := createTestSquadWithUnit(manager, 10, 5, 40)

	// Change to melee column attack type
	entity := manager.FindEntityByID(unitID)
	targetData := common.GetComponentType[*squads.TargetRowData](entity, squads.TargetRowComponent)
	targetData.AttackType = squads.AttackTypeMeleeColumn

	addPerkComponent(manager, unitID, [2]string{"impale", ""})

	cover := squads.CoverBreakdown{
		TotalReduction: 0.25,
		Providers:      []squads.CoverProvider{{UnitID: 1, CoverValue: 0.25}},
	}
	RunCoverModHooks(unitID, 0, &cover, manager)

	if cover.TotalReduction != 0 {
		t.Errorf("expected 0 cover after impale, got %f", cover.TotalReduction)
	}
	if len(cover.Providers) != 0 {
		t.Errorf("expected 0 providers after impale, got %d", len(cover.Providers))
	}
}

func TestImpaleCoverMod_NotMeleeColumn(t *testing.T) {
	manager := setupTestManager(t)
	_, unitID := createTestSquadWithUnit(manager, 10, 5, 40)
	// Default is MeleeRow, impale should not trigger
	addPerkComponent(manager, unitID, [2]string{"impale", ""})

	cover := squads.CoverBreakdown{TotalReduction: 0.25}
	RunCoverModHooks(unitID, 0, &cover, manager)

	if cover.TotalReduction != 0.25 {
		t.Errorf("expected cover unchanged for non-melee-column, got %f", cover.TotalReduction)
	}
}
