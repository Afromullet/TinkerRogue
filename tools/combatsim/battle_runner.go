package combatsim

import (
	"fmt"
	"game_main/common"
	"game_main/tactical/combat/battlelog"

	"github.com/bytearena/ecs"
)

// BattleSimConfig contains configuration for battle simulation runs.
type BattleSimConfig struct {
	NumBattles      int    // N - total battles to run
	SquadsPerBattle int    // M - squads per battle (2 for 1v1, 3+ for multi-squad)
	OutputDir       string // JSON output directory
	GenerationMode  string // "random", "balanced", "varied"
	MaxRounds       int    // Maximum rounds per battle (default: 50)
	Verbose         bool   // Print detailed progress
}

// BattleSimulationRunner orchestrates N battles with automated squad generation and logging.
type BattleSimulationRunner struct {
	config    BattleSimConfig
	generator *SquadCompositionGenerator
}

// NewBattleSimulationRunner creates a new battle simulation runner.
func NewBattleSimulationRunner(config BattleSimConfig) *BattleSimulationRunner {
	// Set default max rounds if not specified
	if config.MaxRounds == 0 {
		config.MaxRounds = 50
	}

	return &BattleSimulationRunner{
		config:    config,
		generator: NewSquadCompositionGenerator(),
	}
}

// RunBattleSet executes all N battles and exports JSON logs.
func (r *BattleSimulationRunner) RunBattleSet() error {
	fmt.Printf("Starting battle simulation: %d battles with %d squads each\n", r.config.NumBattles, r.config.SquadsPerBattle)
	fmt.Printf("Output directory: %s\n", r.config.OutputDir)
	fmt.Printf("Generation mode: %s\n", r.config.GenerationMode)
	fmt.Println()

	successCount := 0
	errorCount := 0

	for i := 1; i <= r.config.NumBattles; i++ {
		if r.config.Verbose {
			fmt.Printf("--- Battle %d/%d ---\n", i, r.config.NumBattles)
		}

		err := r.runSingleBattle(i)
		if err != nil {
			fmt.Printf("Battle %d failed: %v\n", i, err)
			errorCount++
		} else {
			successCount++
			if !r.config.Verbose {
				fmt.Printf("Battle %d/%d complete\n", i, r.config.NumBattles)
			}
		}
	}

	fmt.Println()
	fmt.Printf("=== Battle Simulation Complete ===\n")
	fmt.Printf("Successful: %d/%d\n", successCount, r.config.NumBattles)
	if errorCount > 0 {
		fmt.Printf("Failed: %d/%d\n", errorCount, r.config.NumBattles)
	}

	return nil
}

// runSingleBattle executes one battle with M squads.
func (r *BattleSimulationRunner) runSingleBattle(battleNum int) error {
	// Step 1: Create isolated EntityManager
	manager := common.NewEntityManager()

	// Step 2: Initialize ECS components
	initializeComponents(manager)

	// Step 3: Create BattleRecorder and start recording
	recorder := battlelog.NewBattleRecorder()
	recorder.SetEnabled(true)
	recorder.Start()

	// Step 4: Generate M squads
	squadSetups := r.generateSquads()

	if r.config.Verbose {
		fmt.Printf("Generated %d squads:\n", len(squadSetups))
		for i, setup := range squadSetups {
			fmt.Printf("  %d. %s (%d units)\n", i+1, setup.Name, len(setup.Units))
		}
	}

	// Step 5: Build squads in ECS
	squadIDs := make([]ecs.EntityID, 0, len(squadSetups))
	for _, setup := range squadSetups {
		squadID, err := buildSquad(manager, setup)
		if err != nil {
			return fmt.Errorf("failed to build squad %s: %w", setup.Name, err)
		}
		squadIDs = append(squadIDs, squadID)
	}

	// Step 6: Execute battle
	executor := NewRecordedCombatExecutor(manager, recorder)
	victorID, err := executor.RunBattle(squadIDs, r.config.MaxRounds)
	if err != nil {
		// Still export the log even if battle timed out
		if r.config.Verbose {
			fmt.Printf("Battle ended: %v\n", err)
		}
	}

	// Step 7: Finalize and export
	victorName := r.getSquadName(victorID, squadSetups, squadIDs)
	battleRecord := executor.Finalize(victorID, victorName)

	if battleRecord != nil {
		err := battlelog.ExportBattleJSON(battleRecord, r.config.OutputDir)
		if err != nil {
			return fmt.Errorf("failed to export battle log: %w", err)
		}

		if r.config.Verbose {
			fmt.Printf("Battle completed: %d rounds, Victor: %s\n", executor.GetCurrentRound(), victorName)
			fmt.Printf("Exported to: %s.json\n", battleRecord.BattleID)
		}
	}

	// Step 8: Cleanup entities
	cleanupEntities(manager)

	return nil
}

// generateSquads generates M squads using the configured generation strategy.
func (r *BattleSimulationRunner) generateSquads() []SquadSetup {
	squads := make([]SquadSetup, 0, r.config.SquadsPerBattle)

	for i := 0; i < r.config.SquadsPerBattle; i++ {
		squadName := fmt.Sprintf("Squad_%d", i+1)

		// Position squads in a line with spacing
		posX := i * 3 // 3 tiles apart
		posY := 0

		var setup SquadSetup

		switch r.config.GenerationMode {
		case "random":
			setup = r.generator.GenerateRandomSquad(squadName, posX, posY)
		case "balanced":
			setup = r.generator.GenerateBalancedSquad(squadName, posX, posY)
		case "varied":
			// Mix of different strategies
			setup = r.generator.SelectRandomComposition(squadName, posX, posY)
		default:
			// Default to varied
			setup = r.generator.SelectRandomComposition(squadName, posX, posY)
		}

		squads = append(squads, setup)
	}

	return squads
}

// getSquadName retrieves the squad name from the setup.
func (r *BattleSimulationRunner) getSquadName(squadID ecs.EntityID, setups []SquadSetup, squadIDs []ecs.EntityID) string {
	if squadID == 0 {
		return "Draw"
	}

	// Find the squad in our list
	for i, id := range squadIDs {
		if id == squadID && i < len(setups) {
			return setups[i].Name
		}
	}

	return fmt.Sprintf("Squad_%d", squadID)
}

// BattleSimStats contains statistics about a completed battle simulation run.
type BattleSimStats struct {
	TotalBattles    int
	SuccessfulBattles int
	FailedBattles   int
	TotalRounds     int
	AverageRounds   float64
	VictorCounts    map[string]int // Squad name -> victory count
}

// GenerateStats analyzes completed battle logs and returns statistics.
// This can be called after RunBattleSet to provide summary data.
func (r *BattleSimulationRunner) GenerateStats() *BattleSimStats {
	// This is a placeholder for future analysis functionality
	// Could read JSON files from output directory and aggregate stats
	return &BattleSimStats{
		TotalBattles: r.config.NumBattles,
		VictorCounts: make(map[string]int),
	}
}
