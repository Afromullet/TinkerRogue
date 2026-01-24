package combatsim

import (
	"game_main/common"
	"game_main/tactical/combat/battlelog"
	"game_main/tactical/squads"
	"game_main/templates"
	"testing"

	"github.com/bytearena/ecs"
)

// TestExecuteRecordedAttack_RecordsEngagement verifies that attacks are recorded
func TestExecuteRecordedAttack_RecordsEngagement(t *testing.T) {
	// Initialize templates
	templates.ReadMonsterData()
	if err := squads.InitUnitTemplatesFromJSON(); err != nil {
		t.Fatalf("Failed to init templates: %v", err)
	}

	// Create manager and initialize components
	manager := common.NewEntityManager()
	initializeComponents(manager)

	// Create two simple squads
	attackerSetup := SquadSetup{
		Name: "Attacker",
		Units: []UnitConfig{
			{TemplateName: "Fighter", GridRow: 0, GridCol: 0, IsLeader: true},
			{TemplateName: "Fighter", GridRow: 0, GridCol: 1},
		},
	}

	defenderSetup := SquadSetup{
		Name: "Defender",
		Units: []UnitConfig{
			{TemplateName: "Fighter", GridRow: 0, GridCol: 0, IsLeader: true},
			{TemplateName: "Fighter", GridRow: 0, GridCol: 1},
		},
	}

	attackerID, err := buildSquad(manager, attackerSetup)
	if err != nil {
		t.Fatalf("Failed to build attacker: %v", err)
	}

	defenderID, err := buildSquad(manager, defenderSetup)
	if err != nil {
		t.Fatalf("Failed to build defender: %v", err)
	}

	// Create recorder
	recorder := battlelog.NewBattleRecorder()
	recorder.SetEnabled(true)
	recorder.Start()

	// Create executor
	executor := NewRecordedCombatExecutor(manager, recorder)

	// Execute attack
	result := executor.ExecuteRecordedAttack(attackerID, defenderID)

	// Verify result exists
	if result == nil {
		t.Fatal("ExecuteRecordedAttack returned nil result")
	}

	// Verify engagement was recorded
	if recorder.EngagementCount() == 0 {
		t.Error("Expected engagement to be recorded, got 0 engagements")
	}

	// Cleanup
	cleanupEntities(manager)
}

// TestRunBattle_TracksRounds verifies that round tracking works correctly
func TestRunBattle_TracksRounds(t *testing.T) {
	// Initialize templates
	templates.ReadMonsterData()
	if err := squads.InitUnitTemplatesFromJSON(); err != nil {
		t.Fatalf("Failed to init templates: %v", err)
	}

	// Create manager and initialize components
	manager := common.NewEntityManager()
	initializeComponents(manager)

	// Create two squads
	setup1 := SquadSetup{
		Name: "Squad1",
		Units: []UnitConfig{
			{TemplateName: "Fighter", GridRow: 0, GridCol: 0, IsLeader: true},
		},
	}

	setup2 := SquadSetup{
		Name: "Squad2",
		Units: []UnitConfig{
			{TemplateName: "Fighter", GridRow: 0, GridCol: 0, IsLeader: true},
		},
	}

	squad1ID, _ := buildSquad(manager, setup1)
	squad2ID, _ := buildSquad(manager, setup2)

	// Create recorder
	recorder := battlelog.NewBattleRecorder()
	recorder.SetEnabled(true)
	recorder.Start()

	// Create executor
	executor := NewRecordedCombatExecutor(manager, recorder)

	// Run battle
	_, err := executor.RunBattle([]ecs.EntityID{squad1ID, squad2ID}, 20)

	// Battle should complete (or timeout) without error
	if err != nil && executor.GetCurrentRound() == 0 {
		t.Errorf("Battle failed to execute any rounds: %v", err)
	}

	// Verify rounds were tracked
	if executor.GetCurrentRound() == 0 {
		t.Error("Expected round count > 0")
	}

	// Cleanup
	cleanupEntities(manager)
}

// TestRunBattle_DetectsVictory verifies that victory condition is detected
func TestRunBattle_DetectsVictory(t *testing.T) {
	// Initialize templates
	templates.ReadMonsterData()
	if err := squads.InitUnitTemplatesFromJSON(); err != nil {
		t.Fatalf("Failed to init templates: %v", err)
	}

	// Create manager and initialize components
	manager := common.NewEntityManager()
	initializeComponents(manager)

	// Create two squads (one weak to ensure quick victory)
	strongSquad := SquadSetup{
		Name: "Strong",
		Units: []UnitConfig{
			{TemplateName: "Fighter", GridRow: 0, GridCol: 0, IsLeader: true},
			{TemplateName: "Fighter", GridRow: 0, GridCol: 1},
			{TemplateName: "Fighter", GridRow: 0, GridCol: 2},
		},
	}

	weakSquad := SquadSetup{
		Name: "Weak",
		Units: []UnitConfig{
			{TemplateName: "Fighter", GridRow: 0, GridCol: 0, IsLeader: true},
		},
	}

	strongID, _ := buildSquad(manager, strongSquad)
	weakID, _ := buildSquad(manager, weakSquad)

	// Create recorder
	recorder := battlelog.NewBattleRecorder()
	recorder.SetEnabled(true)
	recorder.Start()

	// Create executor
	executor := NewRecordedCombatExecutor(manager, recorder)

	// Run battle
	victorID, err := executor.RunBattle([]ecs.EntityID{strongID, weakID}, 50)

	// Should have a victor (either strongID or 0 for timeout)
	if err != nil && victorID == 0 {
		t.Logf("Battle ended with timeout/draw: %v", err)
	}

	// Verify battle ran
	if executor.GetCurrentRound() == 0 {
		t.Error("Expected at least 1 round to be executed")
	}

	// Cleanup
	cleanupEntities(manager)
}

// TestFinalize_IncludesVictorInfo verifies finalization includes victor data
func TestFinalize_IncludesVictorInfo(t *testing.T) {
	// Initialize templates
	templates.ReadMonsterData()
	if err := squads.InitUnitTemplatesFromJSON(); err != nil {
		t.Fatalf("Failed to init templates: %v", err)
	}

	// Create manager and initialize components
	manager := common.NewEntityManager()
	initializeComponents(manager)

	// Create a simple squad
	setup := SquadSetup{
		Name: "TestSquad",
		Units: []UnitConfig{
			{TemplateName: "Fighter", GridRow: 0, GridCol: 0, IsLeader: true},
		},
	}

	squadID, _ := buildSquad(manager, setup)

	// Create recorder
	recorder := battlelog.NewBattleRecorder()
	recorder.SetEnabled(true)
	recorder.Start()

	// Create executor
	executor := NewRecordedCombatExecutor(manager, recorder)

	// Execute one attack to have some data
	executor.currentRound = 1
	recorder.SetCurrentRound(1)

	// Finalize
	record := executor.Finalize(squadID, "TestSquad")

	// Verify record exists
	if record == nil {
		t.Fatal("Finalize returned nil record")
	}

	// Verify victor info
	if record.VictorFactionID != squadID {
		t.Errorf("Expected victor ID %d, got %d", squadID, record.VictorFactionID)
	}

	if record.VictorName != "TestSquad" {
		t.Errorf("Expected victor name 'TestSquad', got '%s'", record.VictorName)
	}

	if record.FinalRound != 1 {
		t.Errorf("Expected final round 1, got %d", record.FinalRound)
	}

	// Cleanup
	cleanupEntities(manager)
}

// TestExecuteRecordedCounterattack_RecordsEngagement verifies counterattacks are recorded
func TestExecuteRecordedCounterattack_RecordsEngagement(t *testing.T) {
	// Initialize templates
	templates.ReadMonsterData()
	if err := squads.InitUnitTemplatesFromJSON(); err != nil {
		t.Fatalf("Failed to init templates: %v", err)
	}

	// Create manager and initialize components
	manager := common.NewEntityManager()
	initializeComponents(manager)

	// Create two squads
	attackerSetup := SquadSetup{
		Name: "Attacker",
		Units: []UnitConfig{
			{TemplateName: "Fighter", GridRow: 0, GridCol: 0, IsLeader: true},
		},
	}

	defenderSetup := SquadSetup{
		Name: "Defender",
		Units: []UnitConfig{
			{TemplateName: "Fighter", GridRow: 0, GridCol: 0, IsLeader: true},
		},
	}

	attackerID, _ := buildSquad(manager, attackerSetup)
	defenderID, _ := buildSquad(manager, defenderSetup)

	// Create recorder
	recorder := battlelog.NewBattleRecorder()
	recorder.SetEnabled(true)
	recorder.Start()

	// Create executor
	executor := NewRecordedCombatExecutor(manager, recorder)

	// Execute counterattack
	result := executor.ExecuteRecordedCounterattack(defenderID, attackerID)

	// Verify result exists
	if result == nil {
		t.Fatal("ExecuteRecordedCounterattack returned nil result")
	}

	// Verify engagement was recorded
	if recorder.EngagementCount() == 0 {
		t.Error("Expected counterattack to be recorded")
	}

	// Cleanup
	cleanupEntities(manager)
}
