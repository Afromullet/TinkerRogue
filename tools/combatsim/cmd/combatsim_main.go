package main

import (
	"flag"
	"fmt"
	"game_main/tactical/squads"
	"game_main/templates"
	"game_main/tools/combatsim"
	"log"
)

func main() {
	// Command-line flags
	iterations := flag.Int("iterations", 100, "Number of simulations to run")
	verbose := flag.Bool("verbose", false, "Print detailed combat logs")
	scenarioID := flag.String("scenario", "all", "Scenario to run: 'all' or scenario number (1-15)")

	// Analysis mode flags
	analysisMode := flag.String("analysis", "standard", "Analysis depth: quick, standard, comprehensive")

	// Sweep mode flags
	sweepMode := flag.Bool("sweep", false, "Run parameter sweep analysis")
	sweepAttr := flag.String("sweep-attr", "Strength", "Attribute to sweep (e.g., Strength, Dexterity, Armor)")
	sweepMin := flag.Int("sweep-min", 5, "Minimum value for sweep")
	sweepMax := flag.Int("sweep-max", 20, "Maximum value for sweep")
	sweepStep := flag.Int("sweep-step", 1, "Step size for sweep")
	sweepTarget := flag.String("sweep-target", "Attacker", "Target squad: Attacker or Defender")
	sweepUnit := flag.String("sweep-unit", "*", "Target unit template name or * for all")

	// Export flags
	exportJSON := flag.String("export-json", "", "Export report to JSON file")
	exportCSV := flag.String("export-csv", "", "Export metrics to CSV file")

	// Battle logging mode flags (NEW)
	battleLog := flag.Bool("battle-log", false, "Enable battle logging mode (generates JSON logs)")
	numBattles := flag.Int("num-battles", 10, "Number of battles to run in battle-log mode")
	squadsPerBattle := flag.Int("squads-per-battle", 2, "Squads per battle (default: 2 for 1v1)")
	logOutputDir := flag.String("log-output-dir", "./combat_logs", "Directory for battle logs")
	squadGen := flag.String("squad-gen", "varied", "Squad generation: random, balanced, varied")
	maxRounds := flag.Int("max-rounds", 50, "Maximum rounds per battle")

	flag.Parse()

	// Initialize unit templates
	templates.ReadMonsterData()
	err := squads.InitUnitTemplatesFromJSON()
	if err != nil {
		log.Fatalf("Failed to initialize unit templates: %v", err)
	}

	fmt.Printf("Loaded %d unit templates\n\n", len(squads.Units))

	// Handle battle logging mode (NEW)
	if *battleLog {
		runBattleLoggingMode(*numBattles, *squadsPerBattle, *logOutputDir, *squadGen, *maxRounds)
		return
	}

	// Get all test scenarios
	scenarios := GetAllTestScenarios()

	// Create simulator with analysis mode
	config := combatsim.SimulationConfig{
		Iterations:    *iterations,
		Verbose:       *verbose,
		TrackTimeline: *analysisMode != combatsim.AnalysisModeQuick,
		TrackUnits:    *analysisMode != combatsim.AnalysisModeQuick,
		AnalysisMode:  *analysisMode,
	}
	sim := combatsim.NewSimulator(config)

	// Handle sweep mode
	if *sweepMode {
		runSweepMode(sim, scenarios, *scenarioID, *sweepAttr, *sweepMin, *sweepMax, *sweepStep, *sweepTarget, *sweepUnit)
		return
	}

	// Run simulation(s) with enhanced analysis
	if *scenarioID == "all" {
		// Run all scenarios with quick reports
		fmt.Printf("Running ALL %d scenarios with %d iterations each...\n\n", len(scenarios), *iterations)
		for i, scenario := range scenarios {
			result, err := sim.Run(scenario)
			if err != nil {
				log.Printf("Scenario %d failed: %v\n", i+1, err)
				continue
			}
			fmt.Println(combatsim.FormatQuickReport(result))
		}
	} else {
		// Run single scenario with full analysis
		var scenarioNum int
		_, err := fmt.Sscanf(*scenarioID, "%d", &scenarioNum)
		if err != nil || scenarioNum < 1 || scenarioNum > len(scenarios) {
			log.Fatalf("Invalid scenario number: %s (must be 1-%d or 'all')", *scenarioID, len(scenarios))
		}

		scenario := scenarios[scenarioNum-1]
		fmt.Printf("Running Scenario %d: %s\n", scenarioNum, scenario.Name)
		fmt.Printf("Iterations: %d | Analysis: %s\n\n", *iterations, *analysisMode)

		// Run with full analysis
		result, timelines, unitPerf, err := sim.RunWithAnalysis(scenario)
		if err != nil {
			log.Fatalf("Simulation failed: %v", err)
		}

		// Generate and print balance report
		report := combatsim.GenerateBalanceReport(result, timelines, unitPerf, *analysisMode)
		fmt.Println(combatsim.FormatBalanceReport(report))

		// Export if requested
		if *exportJSON != "" {
			if err := combatsim.ExportJSON(report, *exportJSON); err != nil {
				log.Printf("Failed to export JSON: %v", err)
			} else {
				fmt.Printf("Exported report to %s\n", *exportJSON)
			}
		}
		if *exportCSV != "" {
			if err := combatsim.ExportCSV(report, *exportCSV); err != nil {
				log.Printf("Failed to export CSV: %v", err)
			} else {
				fmt.Printf("Exported metrics to %s\n", *exportCSV)
			}
		}
	}
}

// runSweepMode handles parameter sweep analysis
func runSweepMode(sim *combatsim.Simulator, scenarios []combatsim.CombatScenario, scenarioID string, attr string, minVal, maxVal, step int, target, unit string) {
	// Select scenario
	var scenario combatsim.CombatScenario
	var scenarioNum int
	_, err := fmt.Sscanf(scenarioID, "%d", &scenarioNum)
	if err != nil || scenarioNum < 1 || scenarioNum > len(scenarios) {
		fmt.Println("Sweep mode requires a specific scenario (use -scenario=N)")
		fmt.Printf("Using scenario 1: %s\n", scenarios[0].Name)
		scenario = scenarios[0]
	} else {
		scenario = scenarios[scenarioNum-1]
	}

	fmt.Printf("Running Parameter Sweep on: %s\n", scenario.Name)
	fmt.Printf("Attribute: %s | Range: %d-%d (step %d)\n", attr, minVal, maxVal, step)
	fmt.Printf("Target: %s / %s\n\n", target, unit)

	// Create sweep config
	sweepConfig := combatsim.SweepConfig{
		Name:              fmt.Sprintf("%s Sweep", attr),
		TargetSquad:       target,
		TargetUnit:        unit,
		Attribute:         attr,
		MinValue:          minVal,
		MaxValue:          maxVal,
		StepSize:          step,
		IterationsPerStep: sim.GetIterations(),
	}

	// Run sweep
	result, err := combatsim.RunSweep(sim, scenario, sweepConfig)
	if err != nil {
		log.Fatalf("Sweep failed: %v", err)
	}

	// Print sweep report
	fmt.Println(combatsim.FormatSweepReport(result))
}

// runBattleLoggingMode handles battle logging mode with JSON export
func runBattleLoggingMode(numBattles, squadsPerBattle int, outputDir, squadGen string, maxRounds int) {
	fmt.Println("=== Battle Logging Mode ===")
	fmt.Println()

	// Validate parameters
	if squadsPerBattle < 2 {
		log.Fatalf("squads-per-battle must be at least 2, got %d", squadsPerBattle)
	}
	if squadsPerBattle > 10 {
		fmt.Printf("Warning: %d squads per battle may be slow\n", squadsPerBattle)
	}
	if numBattles < 1 {
		log.Fatalf("num-battles must be at least 1, got %d", numBattles)
	}

	// Create config
	config := combatsim.BattleSimConfig{
		NumBattles:      numBattles,
		SquadsPerBattle: squadsPerBattle,
		OutputDir:       outputDir,
		GenerationMode:  squadGen,
		MaxRounds:       maxRounds,
		Verbose:         false, // Can be made configurable if desired
	}

	// Create runner
	runner := combatsim.NewBattleSimulationRunner(config)

	// Run battle set
	if err := runner.RunBattleSet(); err != nil {
		log.Fatalf("Battle simulation failed: %v", err)
	}

	fmt.Println()
	fmt.Printf("Battle logs exported to: %s\n", outputDir)
}
