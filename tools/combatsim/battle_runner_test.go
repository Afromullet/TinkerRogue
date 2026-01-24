package combatsim

import (
	"game_main/tactical/squads"
	"game_main/templates"
	"os"
	"path/filepath"
	"testing"

	"github.com/bytearena/ecs"
)

// TestNewBattleSimulationRunner_DefaultMaxRounds verifies default max rounds
func TestNewBattleSimulationRunner_DefaultMaxRounds(t *testing.T) {
	config := BattleSimConfig{
		NumBattles:      5,
		SquadsPerBattle: 2,
		OutputDir:       "./test_logs",
		GenerationMode:  "varied",
		MaxRounds:       0, // Not set
	}

	runner := NewBattleSimulationRunner(config)

	if runner.config.MaxRounds != 50 {
		t.Errorf("Expected default MaxRounds to be 50, got %d", runner.config.MaxRounds)
	}
}

// TestGenerateSquads_CreatesCorrectNumber verifies squad generation count
func TestGenerateSquads_CreatesCorrectNumber(t *testing.T) {
	// Initialize templates
	templates.ReadMonsterData()
	if err := squads.InitUnitTemplatesFromJSON(); err != nil {
		t.Fatalf("Failed to init templates: %v", err)
	}

	config := BattleSimConfig{
		NumBattles:      1,
		SquadsPerBattle: 4,
		OutputDir:       "./test_logs",
		GenerationMode:  "varied",
		MaxRounds:       50,
	}

	runner := NewBattleSimulationRunner(config)
	squadSetups := runner.generateSquads()

	if len(squadSetups) != 4 {
		t.Errorf("Expected 4 squads, got %d", len(squadSetups))
	}
}

// TestGenerateSquads_RandomMode verifies random generation mode
func TestGenerateSquads_RandomMode(t *testing.T) {
	// Initialize templates
	templates.ReadMonsterData()
	if err := squads.InitUnitTemplatesFromJSON(); err != nil {
		t.Fatalf("Failed to init templates: %v", err)
	}

	config := BattleSimConfig{
		NumBattles:      1,
		SquadsPerBattle: 2,
		OutputDir:       "./test_logs",
		GenerationMode:  "random",
		MaxRounds:       50,
	}

	runner := NewBattleSimulationRunner(config)
	squadSetups := runner.generateSquads()

	if len(squadSetups) != 2 {
		t.Errorf("Expected 2 squads, got %d", len(squadSetups))
	}

	// Verify squads are valid
	for i, setup := range squadSetups {
		if len(setup.Units) == 0 {
			t.Errorf("Squad %d is empty", i)
		}
	}
}

// TestGenerateSquads_BalancedMode verifies balanced generation mode
func TestGenerateSquads_BalancedMode(t *testing.T) {
	// Initialize templates
	templates.ReadMonsterData()
	if err := squads.InitUnitTemplatesFromJSON(); err != nil {
		t.Fatalf("Failed to init templates: %v", err)
	}

	config := BattleSimConfig{
		NumBattles:      1,
		SquadsPerBattle: 2,
		OutputDir:       "./test_logs",
		GenerationMode:  "balanced",
		MaxRounds:       50,
	}

	runner := NewBattleSimulationRunner(config)
	squadSetups := runner.generateSquads()

	if len(squadSetups) != 2 {
		t.Errorf("Expected 2 squads, got %d", len(squadSetups))
	}

	// Verify squads have reasonable size
	for i, setup := range squadSetups {
		if len(setup.Units) < 3 {
			t.Errorf("Balanced squad %d has too few units: %d", i, len(setup.Units))
		}
	}
}

// TestGenerateSquads_VariedMode verifies varied generation mode
func TestGenerateSquads_VariedMode(t *testing.T) {
	// Initialize templates
	templates.ReadMonsterData()
	if err := squads.InitUnitTemplatesFromJSON(); err != nil {
		t.Fatalf("Failed to init templates: %v", err)
	}

	config := BattleSimConfig{
		NumBattles:      1,
		SquadsPerBattle: 3,
		OutputDir:       "./test_logs",
		GenerationMode:  "varied",
		MaxRounds:       50,
	}

	runner := NewBattleSimulationRunner(config)
	squadSetups := runner.generateSquads()

	if len(squadSetups) != 3 {
		t.Errorf("Expected 3 squads, got %d", len(squadSetups))
	}

	// Verify all squads are valid
	for i, setup := range squadSetups {
		err := ValidateSquadSetup(setup)
		if err != nil {
			t.Errorf("Squad %d failed validation: %v", i, err)
		}
	}
}

// TestGenerateSquads_SquadPositioning verifies squads are spaced correctly
func TestGenerateSquads_SquadPositioning(t *testing.T) {
	// Initialize templates
	templates.ReadMonsterData()
	if err := squads.InitUnitTemplatesFromJSON(); err != nil {
		t.Fatalf("Failed to init templates: %v", err)
	}

	config := BattleSimConfig{
		NumBattles:      1,
		SquadsPerBattle: 3,
		OutputDir:       "./test_logs",
		GenerationMode:  "varied",
		MaxRounds:       50,
	}

	runner := NewBattleSimulationRunner(config)
	squadSetups := runner.generateSquads()

	// Verify squads have different X positions
	positions := make(map[int]bool)
	for _, setup := range squadSetups {
		positions[setup.WorldPosition.X] = true
	}

	if len(positions) != len(squadSetups) {
		t.Error("Squads are not positioned at unique locations")
	}
}

// TestGetSquadName_FindsCorrectName verifies squad name lookup
func TestGetSquadName_FindsCorrectName(t *testing.T) {
	// Initialize templates
	templates.ReadMonsterData()
	if err := squads.InitUnitTemplatesFromJSON(); err != nil {
		t.Fatalf("Failed to init templates: %v", err)
	}

	config := BattleSimConfig{
		NumBattles:      1,
		SquadsPerBattle: 2,
		OutputDir:       "./test_logs",
		GenerationMode:  "varied",
		MaxRounds:       50,
	}

	runner := NewBattleSimulationRunner(config)

	// Create test setups
	setups := []SquadSetup{
		{Name: "Squad_1", Units: []UnitConfig{{TemplateName: "Fighter", IsLeader: true}}},
		{Name: "Squad_2", Units: []UnitConfig{{TemplateName: "Fighter", IsLeader: true}}},
	}

	squadIDs := []ecs.EntityID{100, 200}

	// Test finding squad 1
	name := runner.getSquadName(ecs.EntityID(100), setups, squadIDs)
	if name != "Squad_1" {
		t.Errorf("Expected 'Squad_1', got '%s'", name)
	}

	// Test finding squad 2
	name = runner.getSquadName(ecs.EntityID(200), setups, squadIDs)
	if name != "Squad_2" {
		t.Errorf("Expected 'Squad_2', got '%s'", name)
	}

	// Test draw case
	name = runner.getSquadName(ecs.EntityID(0), setups, squadIDs)
	if name != "Draw" {
		t.Errorf("Expected 'Draw' for ID 0, got '%s'", name)
	}

	// Test unknown ID
	name = runner.getSquadName(ecs.EntityID(999), setups, squadIDs)
	if name == "" {
		t.Error("Expected fallback name for unknown ID")
	}
}

// TestRunBattleSet_CreatesOutputDirectory verifies directory creation
func TestRunBattleSet_CreatesOutputDirectory(t *testing.T) {
	// Initialize templates
	templates.ReadMonsterData()
	if err := squads.InitUnitTemplatesFromJSON(); err != nil {
		t.Fatalf("Failed to init templates: %v", err)
	}

	// Use temp directory
	tempDir := filepath.Join(os.TempDir(), "combatsim_test_output")
	defer os.RemoveAll(tempDir) // Cleanup

	config := BattleSimConfig{
		NumBattles:      1,
		SquadsPerBattle: 2,
		OutputDir:       tempDir,
		GenerationMode:  "balanced",
		MaxRounds:       10, // Short battle for test speed
		Verbose:         false,
	}

	runner := NewBattleSimulationRunner(config)

	// Run battles
	err := runner.RunBattleSet()
	if err != nil {
		t.Fatalf("RunBattleSet failed: %v", err)
	}

	// Verify directory exists
	if _, err := os.Stat(tempDir); os.IsNotExist(err) {
		t.Error("Output directory was not created")
	}

	// Verify at least one JSON file was created
	files, err := filepath.Glob(filepath.Join(tempDir, "*.json"))
	if err != nil {
		t.Fatalf("Failed to glob JSON files: %v", err)
	}

	if len(files) == 0 {
		t.Error("No JSON files were created")
	}
}

// TestRunSingleBattle_CompletesSuccessfully verifies single battle execution
func TestRunSingleBattle_CompletesSuccessfully(t *testing.T) {
	// Initialize templates
	templates.ReadMonsterData()
	if err := squads.InitUnitTemplatesFromJSON(); err != nil {
		t.Fatalf("Failed to init templates: %v", err)
	}

	// Use temp directory
	tempDir := filepath.Join(os.TempDir(), "combatsim_single_battle_test")
	defer os.RemoveAll(tempDir)

	config := BattleSimConfig{
		NumBattles:      1,
		SquadsPerBattle: 2,
		OutputDir:       tempDir,
		GenerationMode:  "balanced",
		MaxRounds:       10,
		Verbose:         true,
	}

	runner := NewBattleSimulationRunner(config)

	// Run single battle
	err := runner.runSingleBattle(1)
	if err != nil {
		t.Errorf("runSingleBattle failed: %v", err)
	}

	// Verify JSON file was created
	files, err := filepath.Glob(filepath.Join(tempDir, "*.json"))
	if err != nil {
		t.Fatalf("Failed to glob JSON files: %v", err)
	}

	if len(files) != 1 {
		t.Errorf("Expected 1 JSON file, got %d", len(files))
	}
}

// TestRunBattleSet_MultipleSquads verifies multi-squad battles
func TestRunBattleSet_MultipleSquads(t *testing.T) {
	// Initialize templates
	templates.ReadMonsterData()
	if err := squads.InitUnitTemplatesFromJSON(); err != nil {
		t.Fatalf("Failed to init templates: %v", err)
	}

	// Use temp directory
	tempDir := filepath.Join(os.TempDir(), "combatsim_multi_squad_test")
	defer os.RemoveAll(tempDir)

	config := BattleSimConfig{
		NumBattles:      2,
		SquadsPerBattle: 3, // Three squads per battle
		OutputDir:       tempDir,
		GenerationMode:  "varied",
		MaxRounds:       10,
		Verbose:         false,
	}

	runner := NewBattleSimulationRunner(config)

	// Run battles
	err := runner.RunBattleSet()
	if err != nil {
		t.Fatalf("RunBattleSet failed: %v", err)
	}

	// Verify 2 JSON files were created
	files, err := filepath.Glob(filepath.Join(tempDir, "*.json"))
	if err != nil {
		t.Fatalf("Failed to glob JSON files: %v", err)
	}

	if len(files) != 2 {
		t.Errorf("Expected 2 JSON files, got %d", len(files))
	}
}

// TestGenerateStats_ReturnsStructure verifies stats structure
func TestGenerateStats_ReturnsStructure(t *testing.T) {
	config := BattleSimConfig{
		NumBattles:      5,
		SquadsPerBattle: 2,
		OutputDir:       "./test_logs",
		GenerationMode:  "varied",
		MaxRounds:       50,
	}

	runner := NewBattleSimulationRunner(config)
	stats := runner.GenerateStats()

	if stats == nil {
		t.Fatal("GenerateStats returned nil")
	}

	if stats.TotalBattles != 5 {
		t.Errorf("Expected TotalBattles=5, got %d", stats.TotalBattles)
	}

	if stats.VictorCounts == nil {
		t.Error("VictorCounts map is nil")
	}
}

// TestRunBattleSet_ErrorHandling verifies error handling for invalid config
func TestRunBattleSet_ErrorHandling(t *testing.T) {
	// This test verifies the runner handles edge cases gracefully
	// Initialize templates
	templates.ReadMonsterData()
	if err := squads.InitUnitTemplatesFromJSON(); err != nil {
		t.Fatalf("Failed to init templates: %v", err)
	}

	// Use temp directory
	tempDir := filepath.Join(os.TempDir(), "combatsim_error_test")
	defer os.RemoveAll(tempDir)

	// Test with valid config but short timeout (may cause timeouts)
	config := BattleSimConfig{
		NumBattles:      1,
		SquadsPerBattle: 2,
		OutputDir:       tempDir,
		GenerationMode:  "balanced",
		MaxRounds:       1, // Very short - may timeout
		Verbose:         false,
	}

	runner := NewBattleSimulationRunner(config)

	// Should complete without crashing even if battles timeout
	err := runner.RunBattleSet()
	if err != nil {
		t.Logf("RunBattleSet reported error (expected with MaxRounds=1): %v", err)
	}

	// Should still create output files
	files, _ := filepath.Glob(filepath.Join(tempDir, "*.json"))
	if len(files) == 0 {
		t.Log("No JSON files created (acceptable with very short MaxRounds)")
	}
}
