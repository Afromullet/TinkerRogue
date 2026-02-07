package main

import (
	"fmt"
	"game_main/tactical/combat/battlelog"
	"os"
)

const outputDir = "./simulation_logs"

func main() {
	fmt.Println("=== Combat Simulator ===")
	fmt.Println()

	// 1. Bootstrap: load game data and unit templates (once)
	bootstrapECS()

	// 2. Get all scenarios
	scenarios := AllScenarios()
	fmt.Printf("Running %d scenarios...\n\n", len(scenarios))

	successCount := 0
	failCount := 0

	for i, scenario := range scenarios {
		fmt.Printf("[%d/%d] %s\n", i+1, len(scenarios), scenario.Name)

		// Fresh ECS manager per scenario (prevents state leaking)
		manager := newSimManager()

		// Create squads for this scenario
		sideA, sideB := createScenarioSquads(manager, scenario)
		fmt.Printf("  Side A: %d squads, Side B: %d squads\n", len(sideA), len(sideB))

		// Run the battle
		record := RunBattle(manager, sideA, sideB)
		if record == nil {
			fmt.Printf("  FAILED: battle returned nil record\n\n")
			failCount++
			continue
		}

		// Export battle log
		if err := battlelog.ExportBattleJSON(record, outputDir); err != nil {
			fmt.Printf("  FAILED to export: %v\n\n", err)
			failCount++
			continue
		}

		fmt.Printf("  Result: %s won in %d rounds (%d engagements)\n\n",
			record.VictorName, record.FinalRound, len(record.Engagements))
		successCount++
	}

	// Summary
	fmt.Println("=== Summary ===")
	fmt.Printf("Completed: %d/%d scenarios\n", successCount, len(scenarios))
	if failCount > 0 {
		fmt.Printf("Failed: %d scenarios\n", failCount)
	}
	fmt.Printf("Logs exported to: %s\n", outputDir)

	if failCount > 0 {
		os.Exit(1)
	}
}
