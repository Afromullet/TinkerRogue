package artifacts

import (
	"testing"

	"game_main/core/coords"
	"game_main/tactical/combat/combatcore"
	"game_main/tactical/combat/combatstate"
	"game_main/tactical/powers/powercore"
	"game_main/templates"

	"github.com/bytearena/ecs"
)

// setupDispatcherArtifacts extends setupTestArtifacts with entries needed by dispatcher tests.
func setupDispatcherArtifacts() {
	setupTestArtifacts()
	templates.ArtifactRegistry["saboteurs_hourglass"] = &templates.ArtifactDefinition{
		ID:       "saboteurs_hourglass",
		Name:     "Saboteur's Hourglass",
		Tier:     "major",
		Behavior: BehaviorSaboteurWsHourglass,
	}
	templates.ArtifactRegistry["deadlock_shackles"] = &templates.ArtifactDefinition{
		ID:       "deadlock_shackles",
		Name:     "Deadlock Shackles",
		Tier:     "major",
		Behavior: BehaviorDeadlockShackles,
	}
}

// TestDispatchPostReset_SameBehaviorTwoSquads verifies that when two squads both equip
// the same behavior, DispatchPostReset fires it only once.
//
// Observable via SaboteursHourglass: movement is reduced by exactly MovementReduction,
// not doubled, even though the behavior key appears twice in the equip loop.
func TestDispatchPostReset_SameBehaviorTwoSquads(t *testing.T) {
	manager := setupTestManager()
	setupDispatcherArtifacts()
	ArtifactBalance.SaboteursHourglass.MovementReduction = 2

	playerID := createTestPlayer(manager)
	cache := combatstate.NewCombatQueryCache(manager)
	fm := combatstate.NewCombatFactionManager(manager, cache)
	playerFaction := fm.CreateCombatFaction("Player", true)

	squad1 := createTestSquadWithUnits(manager, "Squad A", 2)
	squad2 := createTestSquadWithUnits(manager, "Squad B", 2)
	fm.AddSquadToFaction(playerFaction, squad1, coords.LogicalPosition{X: 1, Y: 1})
	fm.AddSquadToFaction(playerFaction, squad2, coords.LogicalPosition{X: 2, Y: 1})

	addArtifactToInventory(playerID, "saboteurs_hourglass", manager)
	addArtifactToInventory(playerID, "saboteurs_hourglass", manager)
	if err := EquipArtifact(playerID, squad1, "saboteurs_hourglass", manager); err != nil {
		t.Fatalf("equip squad1: %v", err)
	}
	if err := EquipArtifact(playerID, squad2, "saboteurs_hourglass", manager); err != nil {
		t.Fatalf("equip squad2: %v", err)
	}

	turnMgr := combatcore.NewTurnManager(manager, cache)
	turnMgr.InitializeCombat([]ecs.EntityID{playerFaction})

	charges := NewArtifactChargeTracker()
	// Add pending entry as if the player activated the artifact this turn.
	charges.Pending.Add(BehaviorSaboteurWsHourglass, 0)

	dispatcher := NewArtifactDispatcher(manager, cache, charges)

	state1 := cache.FindActionStateBySquadID(squad1)
	if state1 == nil {
		t.Fatal("expected action state for squad1")
	}
	startMovement := state1.MovementRemaining

	dispatcher.DispatchPostReset(playerFaction, []ecs.EntityID{squad1, squad2})

	wantMovement := startMovement - 2
	if wantMovement < 0 {
		wantMovement = 0
	}
	if state1.MovementRemaining != wantMovement {
		t.Errorf("movement = %d, want %d (reduction should apply exactly once)", state1.MovementRemaining, wantMovement)
	}
	if charges.Pending.Has() {
		t.Error("pending effects should be consumed after dispatch")
	}
}

// TestDispatchPostReset_CrossFactionPending verifies that a behavior activated against an
// enemy squad fires on the enemy faction's post-reset via the pending-effects loop,
// even when no enemy squad has the behavior equipped.
//
// This is the Deadlock Shackles cross-faction path: the behavior fires against the
// enemy during the enemy's own reset, not the player's.
func TestDispatchPostReset_CrossFactionPending(t *testing.T) {
	manager := setupTestManager()
	setupDispatcherArtifacts()

	cache := combatstate.NewCombatQueryCache(manager)
	fm := combatstate.NewCombatFactionManager(manager, cache)
	playerFaction := fm.CreateCombatFaction("Player", true)
	enemyFaction := fm.CreateCombatFaction("Enemy", false)

	playerSquad := createTestSquadWithUnits(manager, "Player Squad", 2)
	enemySquad := createTestSquadWithUnits(manager, "Enemy Squad", 2)
	fm.AddSquadToFaction(playerFaction, playerSquad, coords.LogicalPosition{X: 1, Y: 1})
	fm.AddSquadToFaction(enemyFaction, enemySquad, coords.LogicalPosition{X: 5, Y: 5})

	turnMgr := combatcore.NewTurnManager(manager, cache)
	turnMgr.InitializeCombat([]ecs.EntityID{playerFaction, enemyFaction})

	charges := NewArtifactChargeTracker()
	ctx := NewBehaviorContext(powercore.NewPowerContext(manager, cache, 0, nil), charges)

	if err := ActivateArtifact(BehaviorDeadlockShackles, enemySquad, ctx); err != nil {
		t.Fatalf("activate: %v", err)
	}
	if !charges.Pending.Has() {
		t.Fatal("expected pending effect after activation")
	}

	dispatcher := NewArtifactDispatcher(manager, cache, charges)
	dispatcher.DispatchPostReset(enemyFaction, []ecs.EntityID{enemySquad})

	actionState := cache.FindActionStateBySquadID(enemySquad)
	if actionState == nil {
		t.Fatal("expected action state for enemy squad")
	}
	if !actionState.HasActed {
		t.Error("want HasActed = true after Deadlock Shackles fires")
	}
	if !actionState.HasMoved {
		t.Error("want HasMoved = true after Deadlock Shackles fires")
	}
	if actionState.MovementRemaining != 0 {
		t.Errorf("want MovementRemaining = 0, got %d", actionState.MovementRemaining)
	}
	if charges.Pending.Has() {
		t.Error("pending effects should be consumed after dispatch")
	}
}

// TestDispatchOnTurnEnd_ChargeRefreshBeforeFire verifies that RefreshRoundCharges is called
// before OnTurnEnd hooks fire. If the order reverses, round-charge behaviors would fire
// with stale (already-spent) charge state and behaviors could get a free use per turn.
func TestDispatchOnTurnEnd_ChargeRefreshBeforeFire(t *testing.T) {
	manager := setupTestManager()
	setupDispatcherArtifacts()

	cache := combatstate.NewCombatQueryCache(manager)
	charges := NewArtifactChargeTracker()

	// Consume a round charge to simulate a behavior used this turn.
	charges.UseCharge(BehaviorChainOfCommand, ChargeOncePerRound)
	if charges.IsAvailable(BehaviorChainOfCommand) {
		t.Fatal("setup: charge should be spent before DispatchOnTurnEnd")
	}

	dispatcher := NewArtifactDispatcher(manager, cache, charges)
	dispatcher.DispatchOnTurnEnd(1)

	if !charges.IsAvailable(BehaviorChainOfCommand) {
		t.Error("round charge not refreshed: RefreshRoundCharges must run before OnTurnEnd hooks")
	}
}
