package perks

import (
	"game_main/common"
	testfx "game_main/testing"
	"game_main/tactical/combat/combatcore"
	"testing"

	"github.com/bytearena/ecs"
)

// ========================================
// Test Helpers
// ========================================

// setupTestManager creates a fully initialized EntityManager for perk tests.
func setupTestManager() *common.EntityManager {
	return testfx.NewTestEntityManager()
}

// createTestSquadWithPerks creates a squad entity and equips perks on it.
// Returns the squad's EntityID.
func createTestSquadWithPerks(manager *common.EntityManager, perkIDs ...string) ecs.EntityID {
	entity := manager.World.NewEntity()
	squadID := entity.GetID()
	slotData := &PerkSlotData{PerkIDs: perkIDs}
	entity.AddComponent(PerkSlotComponent, slotData)
	return squadID
}

// setupTestBalance sets minimal balance values for testing.
func setupTestBalance() {
	PerkBalance = PerkBalanceConfig{
		RecklessAssault:   RecklessAssaultBalance{AttackerMult: 1.3, DefenderMult: 1.2},
		Counterpunch:      CounterpunchBalance{DamageMult: 1.4},
		DeadshotsPatience: DeadshotsPatienceBalance{DamageMult: 1.5, AccuracyBonus: 20},
		AdaptiveArmor:     AdaptiveArmorBalance{MaxHits: 3, PerHitReduction: 0.1},
		Bloodlust:         BloodlustBalance{PerKillBonus: 0.15},
		Fortify:           FortifyBalance{MaxStationaryTurns: 3, PerTurnCoverBonus: 0.05},
		OpeningSalvo:      OpeningSalvoBalance{DamageMult: 1.35},
		Resolute:          ResoluteBalance{HPThreshold: 0.5},
		GrudgeBearer:      GrudgeBearerBalance{MaxStacks: 2, PerStackBonus: 0.2},
	}
}

// setupTestPerkRegistry populates PerkRegistry with test definitions.
func setupTestPerkRegistry() {
	PerkRegistry = map[string]*PerkDefinition{
		PerkCleave:           {ID: PerkCleave, ExclusiveWith: []string{PerkPrecisionStrike}},
		PerkPrecisionStrike:  {ID: PerkPrecisionStrike, ExclusiveWith: []string{PerkCleave}},
		PerkRecklessAssault:  {ID: PerkRecklessAssault, ExclusiveWith: []string{PerkStalwart}},
		PerkStalwart:         {ID: PerkStalwart, ExclusiveWith: []string{PerkRecklessAssault}},
		PerkBloodlust:        {ID: PerkBloodlust, ExclusiveWith: []string{PerkFieldMedic}},
		PerkFieldMedic:       {ID: PerkFieldMedic, ExclusiveWith: []string{PerkBloodlust}},
		PerkCounterpunch:     {ID: PerkCounterpunch, ExclusiveWith: []string{}},
		PerkFortify:          {ID: PerkFortify, ExclusiveWith: []string{}},
		PerkOpeningSalvo:     {ID: PerkOpeningSalvo, ExclusiveWith: []string{}},
		PerkResolute:         {ID: PerkResolute, ExclusiveWith: []string{}},
		PerkGrudgeBearer:     {ID: PerkGrudgeBearer, ExclusiveWith: []string{}},
		PerkAdaptiveArmor:    {ID: PerkAdaptiveArmor, ExclusiveWith: []string{}},
		PerkDeadshotsPatience: {ID: PerkDeadshotsPatience, ExclusiveWith: []string{}},
	}
}

// ========================================
// State Accessor Tests
// ========================================

func TestGetPerkState_NilMap(t *testing.T) {
	s := &PerkRoundState{}
	result := GetPerkState[*RecklessAssaultState](s, "any_perk")
	if result != nil {
		t.Errorf("Expected nil from nil map, got %v", result)
	}
}

func TestSetPerkState_LazyInit(t *testing.T) {
	s := &PerkRoundState{}
	if s.PerkState != nil {
		t.Fatal("PerkState should start nil")
	}
	SetPerkState(s, "test", &BloodlustState{KillsThisRound: 5})
	if s.PerkState == nil {
		t.Fatal("SetPerkState should lazily init map")
	}
	result := GetPerkState[*BloodlustState](s, "test")
	if result == nil || result.KillsThisRound != 5 {
		t.Errorf("Expected KillsThisRound=5, got %v", result)
	}
}

func TestGetOrInitPerkState_InitializesOnce(t *testing.T) {
	s := &PerkRoundState{}
	callCount := 0
	initFn := func() *AdaptiveArmorState {
		callCount++
		return &AdaptiveArmorState{AttackedBy: make(map[ecs.EntityID]int)}
	}

	state1 := GetOrInitPerkState(s, PerkAdaptiveArmor, initFn)
	state1.AttackedBy[42] = 3

	state2 := GetOrInitPerkState(s, PerkAdaptiveArmor, initFn)
	if callCount != 1 {
		t.Errorf("Expected initFn called once, got %d", callCount)
	}
	if state2.AttackedBy[42] != 3 {
		t.Error("Expected same state instance returned on second call")
	}
}

func TestGetBattleState_NilMap(t *testing.T) {
	s := &PerkRoundState{}
	result := GetBattleState[*OpeningSalvoState](s, PerkOpeningSalvo)
	if result != nil {
		t.Errorf("Expected nil from nil map, got %v", result)
	}
}

func TestSetBattleState_LazyInit(t *testing.T) {
	s := &PerkRoundState{}
	SetBattleState(s, PerkOpeningSalvo, &OpeningSalvoState{HasAttackedThisCombat: true})
	if s.PerkBattleState == nil {
		t.Fatal("SetBattleState should lazily init map")
	}
	result := GetBattleState[*OpeningSalvoState](s, PerkOpeningSalvo)
	if result == nil || !result.HasAttackedThisCombat {
		t.Error("Expected HasAttackedThisCombat=true")
	}
}

func TestGetOrInitBattleState_InitializesOnce(t *testing.T) {
	s := &PerkRoundState{}
	callCount := 0
	initFn := func() *GrudgeBearerState {
		callCount++
		return &GrudgeBearerState{Stacks: make(map[ecs.EntityID]int)}
	}

	state1 := GetOrInitBattleState(s, PerkGrudgeBearer, initFn)
	state1.Stacks[99] = 2

	state2 := GetOrInitBattleState(s, PerkGrudgeBearer, initFn)
	if callCount != 1 {
		t.Errorf("Expected initFn called once, got %d", callCount)
	}
	if state2.Stacks[99] != 2 {
		t.Error("Expected same state instance on second call")
	}
}

func TestGetPerkState_WrongType(t *testing.T) {
	s := &PerkRoundState{}
	SetPerkState(s, "test", &BloodlustState{KillsThisRound: 1})
	// Ask for wrong type
	result := GetPerkState[*RecklessAssaultState](s, "test")
	if result != nil {
		t.Errorf("Expected nil for wrong type, got %v", result)
	}
}

// ========================================
// Reset Logic Tests
// ========================================

func TestResetPerkRoundStateTurn_Snapshots(t *testing.T) {
	s := &PerkRoundState{
		MovedThisTurn:       true,
		AttackedThisTurn:    false,
		WasAttackedThisTurn: true,
	}

	ResetPerkRoundStateTurn(s)

	// Verify snapshots
	if !s.WasAttackedLastTurn {
		t.Error("WasAttackedLastTurn should snapshot from WasAttackedThisTurn=true")
	}
	if !s.DidNotAttackLastTurn {
		t.Error("DidNotAttackLastTurn should be true (AttackedThisTurn was false)")
	}
	if s.WasIdleLastTurn {
		t.Error("WasIdleLastTurn should be false (MovedThisTurn was true)")
	}

	// Verify per-turn flags cleared
	if s.MovedThisTurn || s.AttackedThisTurn || s.WasAttackedThisTurn {
		t.Error("Per-turn flags should be cleared after reset")
	}
}

func TestResetPerkRoundStateTurn_IdleSnapshot(t *testing.T) {
	s := &PerkRoundState{
		MovedThisTurn:       false,
		AttackedThisTurn:    false,
		WasAttackedThisTurn: false,
	}

	ResetPerkRoundStateTurn(s)

	if !s.WasIdleLastTurn {
		t.Error("WasIdleLastTurn should be true when neither moved nor attacked")
	}
	if !s.DidNotAttackLastTurn {
		t.Error("DidNotAttackLastTurn should be true")
	}
	if s.WasAttackedLastTurn {
		t.Error("WasAttackedLastTurn should be false")
	}
}

func TestResetPerkRoundStateRound_ClearsPerkStatePreservesBattleState(t *testing.T) {
	s := &PerkRoundState{}
	SetPerkState(s, PerkBloodlust, &BloodlustState{KillsThisRound: 3})
	SetBattleState(s, PerkOpeningSalvo, &OpeningSalvoState{HasAttackedThisCombat: true})

	ResetPerkRoundStateRound(s)

	if s.PerkState != nil {
		t.Error("PerkState should be nil after round reset")
	}
	// Battle state should survive
	result := GetBattleState[*OpeningSalvoState](s, PerkOpeningSalvo)
	if result == nil || !result.HasAttackedThisCombat {
		t.Error("PerkBattleState should survive round reset")
	}
}

// ========================================
// Equip / Unequip Tests
// ========================================

func TestEquipPerk_Success(t *testing.T) {
	manager := setupTestManager()
	setupTestPerkRegistry()

	squadID := createTestSquadWithPerks(manager)
	err := EquipPerk(squadID, PerkCounterpunch, MaxPerkSlots, manager)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if !HasPerk(squadID, PerkCounterpunch, manager) {
		t.Error("Perk should be equipped after EquipPerk")
	}
}

func TestEquipPerk_AlreadyEquipped(t *testing.T) {
	manager := setupTestManager()
	setupTestPerkRegistry()

	squadID := createTestSquadWithPerks(manager, PerkCounterpunch)
	err := EquipPerk(squadID, PerkCounterpunch, MaxPerkSlots, manager)
	if err == nil {
		t.Error("Expected error when equipping already-equipped perk")
	}
}

func TestEquipPerk_SlotsFull(t *testing.T) {
	manager := setupTestManager()
	setupTestPerkRegistry()

	squadID := createTestSquadWithPerks(manager, PerkCounterpunch, PerkFortify, PerkOpeningSalvo)
	err := EquipPerk(squadID, PerkResolute, MaxPerkSlots, manager)
	if err == nil {
		t.Error("Expected error when all slots are full")
	}
}

func TestEquipPerk_ExclusionBlocks(t *testing.T) {
	manager := setupTestManager()
	setupTestPerkRegistry()

	// cleave <-> precision_strike
	squadID := createTestSquadWithPerks(manager, PerkCleave)
	err := EquipPerk(squadID, PerkPrecisionStrike, MaxPerkSlots, manager)
	if err == nil {
		t.Error("Expected error for mutually exclusive perks cleave/precision_strike")
	}

	// reckless_assault <-> stalwart
	squadID2 := createTestSquadWithPerks(manager, PerkRecklessAssault)
	err = EquipPerk(squadID2, PerkStalwart, MaxPerkSlots, manager)
	if err == nil {
		t.Error("Expected error for mutually exclusive perks reckless_assault/stalwart")
	}

	// bloodlust <-> field_medic
	squadID3 := createTestSquadWithPerks(manager, PerkBloodlust)
	err = EquipPerk(squadID3, PerkFieldMedic, MaxPerkSlots, manager)
	if err == nil {
		t.Error("Expected error for mutually exclusive perks bloodlust/field_medic")
	}
}

func TestEquipPerk_NonExclusiveAllowed(t *testing.T) {
	manager := setupTestManager()
	setupTestPerkRegistry()

	squadID := createTestSquadWithPerks(manager, PerkCounterpunch)
	err := EquipPerk(squadID, PerkFortify, MaxPerkSlots, manager)
	if err != nil {
		t.Fatalf("Expected no error for non-exclusive perks, got %v", err)
	}
}

func TestUnequipPerk_Success(t *testing.T) {
	manager := setupTestManager()

	squadID := createTestSquadWithPerks(manager, PerkCounterpunch, PerkFortify)
	err := UnequipPerk(squadID, PerkCounterpunch, manager)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if HasPerk(squadID, PerkCounterpunch, manager) {
		t.Error("Perk should be removed after UnequipPerk")
	}
	if !HasPerk(squadID, PerkFortify, manager) {
		t.Error("Other perks should remain after unequip")
	}
}

func TestUnequipPerk_NotEquipped(t *testing.T) {
	manager := setupTestManager()

	squadID := createTestSquadWithPerks(manager, PerkCounterpunch)
	err := UnequipPerk(squadID, PerkFortify, manager)
	if err == nil {
		t.Error("Expected error when unequipping perk that is not equipped")
	}
}

// ========================================
// Stateful Perk Lifecycle Tests
// ========================================

func TestRecklessAssault_Lifecycle(t *testing.T) {
	setupTestBalance()
	s := &PerkRoundState{}

	ctx := &HookContext{
		RoundState: s,
	}

	// TurnStart resets vulnerability
	recklessAssaultTurnStart(ctx)
	state := GetPerkState[*RecklessAssaultState](s, PerkRecklessAssault)
	if state == nil || state.Vulnerable {
		t.Error("TurnStart should reset reckless assault vulnerability to false")
	}

	// Before attacking: defender mod should not apply (not vulnerable)
	defMods0 := &combatcore.DamageModifiers{DamageMultiplier: 1.0}
	recklessAssaultDefenderMod(ctx, defMods0)
	if defMods0.DamageMultiplier != 1.0 {
		t.Errorf("Expected 1.0 when not vulnerable, got %v", defMods0.DamageMultiplier)
	}

	// AttackerDamageMod should boost damage AND set vulnerability
	mods := &combatcore.DamageModifiers{DamageMultiplier: 1.0}
	recklessAssaultAttackerMod(ctx, mods)
	if mods.DamageMultiplier != 1.3 {
		t.Errorf("Expected 1.3 attack mult, got %v", mods.DamageMultiplier)
	}
	state = GetPerkState[*RecklessAssaultState](s, PerkRecklessAssault)
	if state == nil || !state.Vulnerable {
		t.Error("AttackerDamageMod should set vulnerability")
	}

	// DefenderDamageMod should increase incoming damage when vulnerable
	defMods := &combatcore.DamageModifiers{DamageMultiplier: 1.0}
	recklessAssaultDefenderMod(ctx, defMods)
	if defMods.DamageMultiplier != 1.2 {
		t.Errorf("Expected 1.2 defender mult when vulnerable, got %v", defMods.DamageMultiplier)
	}

	// Next TurnStart clears vulnerability
	recklessAssaultTurnStart(ctx)
	state = GetPerkState[*RecklessAssaultState](s, PerkRecklessAssault)
	if state.Vulnerable {
		t.Error("TurnStart should clear vulnerability")
	}
}

func TestCounterpunch_ArmFireReset(t *testing.T) {
	setupTestBalance()
	s := &PerkRoundState{
		WasAttackedLastTurn:  true,
		DidNotAttackLastTurn: true,
	}

	ctx := &HookContext{RoundState: s}

	// TurnStart should arm
	counterpunchTurnStart(ctx)
	state := GetPerkState[*CounterpunchState](s, PerkCounterpunch)
	if state == nil || !state.Ready {
		t.Error("Counterpunch should be armed when attacked last turn and did not attack")
	}

	// DamageMod should fire and disarm
	mods := &combatcore.DamageModifiers{DamageMultiplier: 1.0}
	counterpunchDamageMod(ctx, mods)
	if mods.DamageMultiplier != 1.4 {
		t.Errorf("Expected 1.4 damage mult, got %v", mods.DamageMultiplier)
	}
	state = GetPerkState[*CounterpunchState](s, PerkCounterpunch)
	if state.Ready {
		t.Error("Counterpunch should be disarmed after firing")
	}

	// Second attack should not get bonus
	mods2 := &combatcore.DamageModifiers{DamageMultiplier: 1.0}
	counterpunchDamageMod(ctx, mods2)
	if mods2.DamageMultiplier != 1.0 {
		t.Errorf("Expected 1.0 damage mult after disarm, got %v", mods2.DamageMultiplier)
	}
}

func TestCounterpunch_NotArmedWhenNotAttacked(t *testing.T) {
	setupTestBalance()
	s := &PerkRoundState{
		WasAttackedLastTurn:  false,
		DidNotAttackLastTurn: true,
	}

	ctx := &HookContext{RoundState: s}
	counterpunchTurnStart(ctx)

	state := GetPerkState[*CounterpunchState](s, PerkCounterpunch)
	if state != nil && state.Ready {
		t.Error("Counterpunch should NOT be armed when not attacked last turn")
	}
}

func TestFortify_StationaryAccumulation(t *testing.T) {
	setupTestBalance()
	s := &PerkRoundState{MovedThisTurn: false}

	ctx := &HookContext{RoundState: s}

	// Three consecutive stationary turns
	fortifyTurnStart(ctx)
	if s.TurnsStationary != 1 {
		t.Errorf("Expected TurnsStationary=1, got %d", s.TurnsStationary)
	}
	fortifyTurnStart(ctx)
	if s.TurnsStationary != 2 {
		t.Errorf("Expected TurnsStationary=2, got %d", s.TurnsStationary)
	}
	fortifyTurnStart(ctx)
	if s.TurnsStationary != 3 {
		t.Errorf("Expected TurnsStationary=3 (max), got %d", s.TurnsStationary)
	}
	// Should cap at max
	fortifyTurnStart(ctx)
	if s.TurnsStationary != 3 {
		t.Errorf("Expected TurnsStationary to cap at 3, got %d", s.TurnsStationary)
	}

	// Movement resets
	s.MovedThisTurn = true
	fortifyTurnStart(ctx)
	if s.TurnsStationary != 0 {
		t.Errorf("Expected TurnsStationary=0 after movement, got %d", s.TurnsStationary)
	}
}

func TestFortify_CoverMod(t *testing.T) {
	setupTestBalance()
	s := &PerkRoundState{TurnsStationary: 2}
	ctx := &HookContext{RoundState: s}

	cover := &combatcore.CoverBreakdown{TotalReduction: 0.1}
	fortifyCoverMod(ctx, cover)

	expected := 0.1 + 2*0.05 // 0.2
	if cover.TotalReduction != expected {
		t.Errorf("Expected cover %.2f, got %.2f", expected, cover.TotalReduction)
	}
}

func TestBloodlust_KillTracking(t *testing.T) {
	setupTestBalance()
	s := &PerkRoundState{}
	ctx := &HookContext{RoundState: s}

	// Track kills
	bloodlustPostDamage(ctx, 10, true)
	bloodlustPostDamage(ctx, 10, true)
	bloodlustPostDamage(ctx, 10, false) // not a kill

	state := GetPerkState[*BloodlustState](s, PerkBloodlust)
	if state == nil || state.KillsThisRound != 2 {
		t.Errorf("Expected 2 kills tracked, got %v", state)
	}

	// Damage bonus should stack
	mods := &combatcore.DamageModifiers{DamageMultiplier: 1.0}
	bloodlustDamageMod(ctx, mods)
	expected := 1.0 + 2*0.15 // 1.3
	if mods.DamageMultiplier != expected {
		t.Errorf("Expected %.2f damage mult, got %.2f", expected, mods.DamageMultiplier)
	}
}

func TestAdaptiveArmor_StackingReduction(t *testing.T) {
	setupTestBalance()
	s := &PerkRoundState{}
	ctx := &HookContext{
		RoundState:      s,
		AttackerSquadID: 42,
	}

	// First hit: no reduction (0 prior hits)
	mods := &combatcore.DamageModifiers{DamageMultiplier: 1.0}
	adaptiveArmorDamageMod(ctx, mods)
	if mods.DamageMultiplier != 1.0 {
		t.Errorf("Expected 1.0 on first hit, got %v", mods.DamageMultiplier)
	}

	// Second hit: 10% reduction (1 prior hit)
	mods2 := &combatcore.DamageModifiers{DamageMultiplier: 1.0}
	adaptiveArmorDamageMod(ctx, mods2)
	if mods2.DamageMultiplier != 0.9 {
		t.Errorf("Expected 0.9 on second hit, got %v", mods2.DamageMultiplier)
	}

	// Third hit: 20% reduction (2 prior hits)
	mods3 := &combatcore.DamageModifiers{DamageMultiplier: 1.0}
	adaptiveArmorDamageMod(ctx, mods3)
	expected := 1.0 - 2*0.1
	if mods3.DamageMultiplier != expected {
		t.Errorf("Expected %.2f on third hit, got %v", expected, mods3.DamageMultiplier)
	}
}

func TestOpeningSalvo_FirstAttackOnly(t *testing.T) {
	setupTestBalance()
	s := &PerkRoundState{}
	ctx := &HookContext{RoundState: s}

	// First attack gets bonus
	mods := &combatcore.DamageModifiers{DamageMultiplier: 1.0}
	openingSalvoDamageMod(ctx, mods)
	if mods.DamageMultiplier != 1.35 {
		t.Errorf("Expected 1.35 on first attack, got %v", mods.DamageMultiplier)
	}

	// Second attack: no bonus
	mods2 := &combatcore.DamageModifiers{DamageMultiplier: 1.0}
	openingSalvoDamageMod(ctx, mods2)
	if mods2.DamageMultiplier != 1.0 {
		t.Errorf("Expected 1.0 on second attack, got %v", mods2.DamageMultiplier)
	}
}

func TestOpeningSalvo_IgnoresCounterattack(t *testing.T) {
	setupTestBalance()
	s := &PerkRoundState{}
	ctx := &HookContext{RoundState: s}

	mods := &combatcore.DamageModifiers{DamageMultiplier: 1.0, IsCounterattack: true}
	openingSalvoDamageMod(ctx, mods)
	if mods.DamageMultiplier != 1.0 {
		t.Error("Opening Salvo should not trigger on counterattacks")
	}
}

// ========================================
// Stalwart Counter Tests
// ========================================

func TestStalwart_FullDamageCounterWhenStationary(t *testing.T) {
	s := &PerkRoundState{MovedThisTurn: false}
	ctx := &HookContext{RoundState: s}

	mods := &combatcore.DamageModifiers{DamageMultiplier: 0.5} // Default counter penalty
	skipCounter := stalwartCounterMod(ctx, mods)

	if skipCounter {
		t.Error("Stalwart should not skip counter")
	}
	if mods.DamageMultiplier != 1.0 {
		t.Errorf("Expected 1.0 counter damage when stationary, got %v", mods.DamageMultiplier)
	}
}

func TestStalwart_NoEffectWhenMoved(t *testing.T) {
	s := &PerkRoundState{MovedThisTurn: true}
	ctx := &HookContext{RoundState: s}

	mods := &combatcore.DamageModifiers{DamageMultiplier: 0.5}
	stalwartCounterMod(ctx, mods)

	if mods.DamageMultiplier != 0.5 {
		t.Errorf("Stalwart should not modify counter when moved, got %v", mods.DamageMultiplier)
	}
}

// ========================================
// Logger Tests
// ========================================

func TestPerkLogger_CalledOnActivation(t *testing.T) {
	var logged []string
	SetPerkLogger(func(perkID string, squadID ecs.EntityID, message string) {
		logged = append(logged, perkID+":"+message)
	})
	defer SetPerkLogger(nil)

	logPerkActivation("test_perk", 1, "test message")
	if len(logged) != 1 || logged[0] != "test_perk:test message" {
		t.Errorf("Expected logger to be called, got %v", logged)
	}
}

func TestPerkLogger_NilSafe(t *testing.T) {
	SetPerkLogger(nil)
	// Should not panic
	logPerkActivation("test_perk", 1, "test message")
}

// ========================================
// forEachPerkHook Tests
// ========================================

func TestForEachPerkHook_PassesPerkID(t *testing.T) {
	manager := setupTestManager()
	squadID := createTestSquadWithPerks(manager, PerkCounterpunch, PerkBloodlust)

	var seenIDs []string
	forEachPerkHook(squadID, manager, func(perkID string, hooks *PerkHooks) bool {
		seenIDs = append(seenIDs, perkID)
		return true
	})

	if len(seenIDs) != 2 {
		t.Errorf("Expected 2 perk IDs, got %d: %v", len(seenIDs), seenIDs)
	}
}

func TestForEachPerkHook_EarlyExit(t *testing.T) {
	manager := setupTestManager()
	squadID := createTestSquadWithPerks(manager, PerkCounterpunch, PerkBloodlust, PerkFortify)

	count := 0
	forEachPerkHook(squadID, manager, func(perkID string, hooks *PerkHooks) bool {
		count++
		return false // stop after first
	})

	if count != 1 {
		t.Errorf("Expected early exit after 1, got %d iterations", count)
	}
}

// ========================================
// Integration: InitializeRoundState / CleanupRoundState
// ========================================

func TestInitializeAndCleanupRoundState(t *testing.T) {
	manager := setupTestManager()
	squadID := createTestSquadWithPerks(manager, PerkCounterpunch)

	// Before init: no round state
	if GetRoundState(squadID, manager) != nil {
		t.Error("Expected no round state before initialization")
	}

	// Initialize
	InitializeRoundState(squadID, manager)
	rs := GetRoundState(squadID, manager)
	if rs == nil {
		t.Fatal("Expected round state after initialization")
	}

	// Cleanup
	CleanupRoundState(squadID, manager)
	if GetRoundState(squadID, manager) != nil {
		t.Error("Expected no round state after cleanup")
	}
}

func TestHasAnyPerks(t *testing.T) {
	manager := setupTestManager()

	emptySquadID := createTestSquadWithPerks(manager)
	if HasAnyPerks(emptySquadID, manager) {
		t.Error("Expected false for squad with no perks")
	}

	perkSquadID := createTestSquadWithPerks(manager, PerkCounterpunch)
	if !HasAnyPerks(perkSquadID, manager) {
		t.Error("Expected true for squad with perks")
	}
}

// ========================================
// Multi-Perk Interaction Tests
// ========================================

func TestMultiPerk_BloodlustPlusOpeningSalvo(t *testing.T) {
	// A squad with both Bloodlust and Opening Salvo.
	// First attack should get both bonuses (Opening Salvo + Bloodlust if kills exist).
	setupTestBalance()
	manager := setupTestManager()

	squadID := createTestSquadWithPerks(manager, PerkBloodlust, PerkOpeningSalvo)
	InitializeRoundState(squadID, manager)

	rs := GetRoundState(squadID, manager)
	if rs == nil {
		t.Fatal("Expected round state")
	}

	// Simulate a kill first to arm bloodlust
	bloodlustCtx := &HookContext{RoundState: rs, AttackerSquadID: squadID}
	bloodlustPostDamage(bloodlustCtx, 20, true) // 1 kill

	// Now run AttackerDamageMod hooks through the pipeline for both perks
	mods := &combatcore.DamageModifiers{DamageMultiplier: 1.0}
	ctx := buildCombatContext(squadID, 100, 200, squadID, ecs.EntityID(999), manager)
	if ctx == nil {
		t.Fatal("Expected combat context")
	}

	// Run hooks manually like the runner does
	forEachPerkHook(squadID, manager, func(perkID string, hooks *PerkHooks) bool {
		if hooks.AttackerDamageMod != nil {
			hooks.AttackerDamageMod(ctx, mods)
		}
		return true
	})

	// Bloodlust: 1 kill -> 1.0 + 0.15 = 1.15
	// Opening Salvo: * 1.35
	// Combined: 1.15 * 1.35 = 1.5525
	expected := (1.0 + 1*0.15) * 1.35
	if mods.DamageMultiplier != expected {
		t.Errorf("Expected combined mult %.4f (Bloodlust+OpeningSalvo), got %.4f", expected, mods.DamageMultiplier)
	}
}

func TestMultiPerk_CounterpunchPlusGrudgeBearer(t *testing.T) {
	// A squad with Counterpunch and Grudge Bearer.
	// Attacked last turn -> Counterpunch armed -> attack should stack both bonuses.
	setupTestBalance()
	manager := setupTestManager()

	squadID := createTestSquadWithPerks(manager, PerkCounterpunch, PerkGrudgeBearer)
	InitializeRoundState(squadID, manager)

	rs := GetRoundState(squadID, manager)

	// Set up Counterpunch: attacked last turn, did not attack
	rs.WasAttackedLastTurn = true
	rs.DidNotAttackLastTurn = true
	counterpunchTurnStart(&HookContext{RoundState: rs})

	// Set up Grudge Bearer: enemy squad 50 damaged us twice
	enemySquadID := ecs.EntityID(50)
	SetBattleState(rs, PerkGrudgeBearer, &GrudgeBearerState{
		Stacks: map[ecs.EntityID]int{enemySquadID: 2},
	})

	// Run combined damage mod hooks
	mods := &combatcore.DamageModifiers{DamageMultiplier: 1.0}
	ctx := buildCombatContext(squadID, 100, 200, squadID, enemySquadID, manager)

	forEachPerkHook(squadID, manager, func(perkID string, hooks *PerkHooks) bool {
		if hooks.AttackerDamageMod != nil {
			hooks.AttackerDamageMod(ctx, mods)
		}
		return true
	})

	// Counterpunch: * 1.4
	// Grudge Bearer: 2 stacks -> * (1 + 2*0.2) = * 1.4
	// Combined: 1.4 * 1.4 = 1.96
	expected := 1.4 * 1.4
	diff := mods.DamageMultiplier - expected
	if diff > 0.0001 || diff < -0.0001 {
		t.Errorf("Expected combined mult ~%.4f (Counterpunch+GrudgeBearer), got %.4f", expected, mods.DamageMultiplier)
	}

	// Verify Counterpunch disarmed after firing
	cpState := GetPerkState[*CounterpunchState](rs, PerkCounterpunch)
	if cpState.Ready {
		t.Error("Counterpunch should be disarmed after firing")
	}
}

func TestMultiPerk_ExclusionPreventsCleaveAndPrecisionStrike(t *testing.T) {
	// Verify that the exclusion system prevents both from being equipped
	// on the same squad through the EquipPerk path.
	manager := setupTestManager()
	setupTestPerkRegistry()

	squadID := createTestSquadWithPerks(manager)
	err := EquipPerk(squadID, PerkCleave, MaxPerkSlots, manager)
	if err != nil {
		t.Fatal(err)
	}
	err = EquipPerk(squadID, PerkPrecisionStrike, MaxPerkSlots, manager)
	if err == nil {
		t.Error("Should not be able to equip both Cleave and Precision Strike")
	}

	// Reverse order should also fail
	squadID2 := createTestSquadWithPerks(manager)
	err = EquipPerk(squadID2, PerkPrecisionStrike, MaxPerkSlots, manager)
	if err != nil {
		t.Fatal(err)
	}
	err = EquipPerk(squadID2, PerkCleave, MaxPerkSlots, manager)
	if err == nil {
		t.Error("Should not be able to equip both Precision Strike and Cleave")
	}
}

// ========================================
// Balance Config Validation Tests
// ========================================

func TestValidatePerkBalance_ZeroMultipliersWarn(t *testing.T) {
	// validatePerkBalance should detect zero/negative multipliers.
	// We can't easily capture stdout warnings, but we can verify the function
	// doesn't panic and that a zero-valued config is clearly invalid.
	cfg := &PerkBalanceConfig{} // All zero values
	// Should not panic
	validatePerkBalance(cfg)
}

func TestValidatePerkBalance_ValidConfig(t *testing.T) {
	cfg := &PerkBalanceConfig{
		BraceForImpact:       BraceForImpactBalance{CoverBonus: 0.15},
		ExecutionersInstinct: ExecutionersInstinctBalance{HPThreshold: 0.3, CritBonus: 25},
		ShieldwallDiscipline: ShieldwallDisciplineBalance{MaxTanks: 3, PerTankReduction: 0.05},
		IsolatedPredator:     IsolatedPredatorBalance{Range: 3, DamageMult: 1.25},
		FieldMedic:           FieldMedicBalance{HealDivisor: 10},
		LastLine:             LastLineBalance{DamageMult: 1.2, HitBonus: 20},
		Cleave:               CleaveBalance{DamageMult: 0.7},
		GuardianProtocol:     GuardianProtocolBalance{RedirectFraction: 4},
		RecklessAssault:      RecklessAssaultBalance{AttackerMult: 1.3, DefenderMult: 1.2},
		Fortify:              FortifyBalance{MaxStationaryTurns: 3, PerTurnCoverBonus: 0.05},
		Counterpunch:         CounterpunchBalance{DamageMult: 1.4},
		DeadshotsPatience:    DeadshotsPatienceBalance{DamageMult: 1.5, AccuracyBonus: 20},
		AdaptiveArmor:        AdaptiveArmorBalance{MaxHits: 3, PerHitReduction: 0.1},
		Bloodlust:            BloodlustBalance{PerKillBonus: 0.15},
		OpeningSalvo:         OpeningSalvoBalance{DamageMult: 1.35},
		Resolute:             ResoluteBalance{HPThreshold: 0.5},
		GrudgeBearer:         GrudgeBearerBalance{MaxStacks: 2, PerStackBonus: 0.2},
	}
	// Should not panic and should pass all checks
	validatePerkBalance(cfg)
}

func TestPerkBalanceConfig_ZeroFieldsAreDetectable(t *testing.T) {
	// A config loaded from empty JSON would have all zero values.
	// Verify that behaviors using zero balance values produce no-op results
	// (no damage multiplied by zero, no panics).
	setupTestBalance()

	// Zero out one specific field to test
	savedMult := PerkBalance.RecklessAssault.AttackerMult
	PerkBalance.RecklessAssault.AttackerMult = 0
	defer func() { PerkBalance.RecklessAssault.AttackerMult = savedMult }()

	s := &PerkRoundState{}
	ctx := &HookContext{RoundState: s}

	// Should multiply by 0, producing 0 damage — detectable as a bug
	mods := &combatcore.DamageModifiers{DamageMultiplier: 1.0}
	recklessAssaultAttackerMod(ctx, mods)
	if mods.DamageMultiplier != 0.0 {
		t.Errorf("Expected 0.0 with zero balance mult, got %v", mods.DamageMultiplier)
	}
}
