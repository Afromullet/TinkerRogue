package main

import (
	"flag"
	"fmt"
	"game_main/tactical/combatsim"
	"game_main/tactical/squads"
	"game_main/templates"
	"log"
)

func main() {
	// Command-line flags
	iterations := flag.Int("iterations", 100, "Number of simulations to run")
	verbose := flag.Bool("verbose", false, "Print detailed combat logs")
	scenarioID := flag.String("scenario", "all", "Scenario to run: 'all' or scenario number (1-15)")

	flag.Parse()

	// Initialize unit templates
	templates.ReadMonsterData()
	err := squads.InitUnitTemplatesFromJSON()
	if err != nil {
		log.Fatalf("Failed to initialize unit templates: %v", err)
	}

	fmt.Printf("Loaded %d unit templates\n\n", len(squads.Units))

	// Get all test scenarios
	scenarios := GetAllTestScenarios()

	// Create simulator
	config := combatsim.SimulationConfig{
		Iterations: *iterations,
		Verbose:    *verbose,
	}
	sim := combatsim.NewSimulator(config)
	formatter := combatsim.NewReportFormatter(*verbose)

	// Run simulation(s)
	if *scenarioID == "all" {
		// Run all scenarios
		fmt.Printf("Running ALL %d scenarios with %d iterations each...\n\n", len(scenarios), *iterations)
		for i, scenario := range scenarios {
			fmt.Printf("\n========================================\n")
			fmt.Printf("Scenario %d/%d: %s\n", i+1, len(scenarios), scenario.Name)
			fmt.Printf("========================================\n")

			result, err := sim.Run(scenario)
			if err != nil {
				log.Printf("Scenario %d failed: %v\n", i+1, err)
				continue
			}

			report := formatter.FormatSimulationResult(result)
			fmt.Println(report)
		}
	} else {
		// Run single scenario by number
		var scenarioNum int
		_, err := fmt.Sscanf(*scenarioID, "%d", &scenarioNum)
		if err != nil || scenarioNum < 1 || scenarioNum > len(scenarios) {
			log.Fatalf("Invalid scenario number: %s (must be 1-%d or 'all')", *scenarioID, len(scenarios))
		}

		scenario := scenarios[scenarioNum-1]
		fmt.Printf("Running Scenario %d: %s\n", scenarioNum, scenario.Name)
		fmt.Printf("Iterations: %d\n\n", *iterations)

		result, err := sim.Run(scenario)
		if err != nil {
			log.Fatalf("Simulation failed: %v", err)
		}

		report := formatter.FormatSimulationResult(result)
		fmt.Println(report)
	}
}

// createTestScenario creates a simple test scenario
func createTestScenario() combatsim.CombatScenario {
	// Create attacker squad (front row fighters)
	attackerUnits := []combatsim.UnitConfig{
		{TemplateName: "Fighter", GridRow: 0, GridCol: 0, IsLeader: true},
		{TemplateName: "Fighter", GridRow: 0, GridCol: 1},
		{TemplateName: "Fighter", GridRow: 0, GridCol: 2},
	}

	// Create defender squad (front row warriors)
	defenderUnits := []combatsim.UnitConfig{
		{TemplateName: "Fighter", GridRow: 0, GridCol: 0, IsLeader: true},
		{TemplateName: "Fighter", GridRow: 0, GridCol: 1},
		{TemplateName: "Fighter", GridRow: 0, GridCol: 2},
	}

	// Build scenario
	scenario := combatsim.NewScenarioBuilder("Fighters vs Warriors").
		WithAttacker("Fighter Squad", attackerUnits).
		WithDefender("Warridddor Squad", defenderUnits).
		WithDistance(1).
		Build()

	return scenario
}
