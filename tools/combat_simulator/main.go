package main

import (
	"flag"
	"fmt"
	"game_main/tactical/combat/battlelog"
	"os"
)

const outputDir = "./simulation_logs"

func main() {
	suiteName := flag.String("suite", "", "Run only a specific suite (duels, compositions, encounters, stress, legacy)")
	listSuites := flag.Bool("list", false, "List available suites and scenario counts")
	flag.Parse()

	fmt.Println("=== Combat Simulator ===")
	fmt.Println()

	// 1. Bootstrap: load game data and unit templates (once)
	bootstrapECS()

	// 2. Get all suites
	suites := AllSuites()

	// 3. Handle --list
	if *listSuites {
		fmt.Println("Available suites:")
		totalScenarios := 0
		for _, s := range suites {
			fmt.Printf("  %-15s %d scenarios\n", s.Name, len(s.Scenarios))
			totalScenarios += len(s.Scenarios)
		}
		fmt.Printf("\n  Total: %d scenarios\n", totalScenarios)
		return
	}

	// 4. Filter by --suite if given
	if *suiteName != "" {
		var filtered []Suite
		for _, s := range suites {
			if s.Name == *suiteName {
				filtered = append(filtered, s)
			}
		}
		if len(filtered) == 0 {
			fmt.Printf("Unknown suite: %s\n", *suiteName)
			fmt.Println("Available suites:")
			for _, s := range suites {
				fmt.Printf("  %s\n", s.Name)
			}
			os.Exit(1)
		}
		suites = filtered
	}

	// 5. Count total scenarios
	totalScenarios := 0
	for _, s := range suites {
		totalScenarios += len(s.Scenarios)
	}

	fmt.Printf("Running %d scenarios across %d suite(s)...\n\n", totalScenarios, len(suites))

	totalSuccess := 0
	totalFail := 0
	scenarioNum := 0
	victories := map[string]int{}

	// 6. Run each suite
	for _, suite := range suites {
		fmt.Printf("--- Suite: %s (%d scenarios) ---\n\n", suite.Name, len(suite.Scenarios))
		suiteSuccess := 0
		suiteFail := 0

		for _, scenario := range suite.Scenarios {
			scenarioNum++
			fmt.Printf("[%d/%d] %s\n", scenarioNum, totalScenarios, scenario.Name)

			// Fresh ECS manager per scenario (prevents state leaking)
			manager := newSimManager()

			// Create squads for this scenario
			sideA, sideB := createScenarioSquads(manager, scenario)
			fmt.Printf("  Side A: %d squads, Side B: %d squads\n", len(sideA), len(sideB))

			// Run the battle
			record := RunBattle(manager, sideA, sideB)
			if record == nil {
				fmt.Printf("  FAILED: battle returned nil record\n\n")
				suiteFail++
				totalFail++
				continue
			}

			// Export battle log
			if err := battlelog.ExportBattleJSON(record, outputDir); err != nil {
				fmt.Printf("  FAILED to export: %v\n\n", err)
				suiteFail++
				totalFail++
				continue
			}

			fmt.Printf("  Result: %s won in %d rounds (%d engagements)\n\n",
				record.VictorName, record.FinalRound, len(record.Engagements))
			victories[record.VictorName]++
			suiteSuccess++
			totalSuccess++
		}

		fmt.Printf("--- Suite %s: %d/%d completed ---\n\n", suite.Name, suiteSuccess, len(suite.Scenarios))
	}

	// 7. Summary
	fmt.Println("=== Summary ===")
	fmt.Printf("Completed: %d/%d scenarios\n", totalSuccess, totalScenarios)
	if totalFail > 0 {
		fmt.Printf("Failed: %d scenarios\n", totalFail)
	}

	fmt.Println("\nVictory distribution:")
	for name, count := range victories {
		fmt.Printf("  %-10s %d wins\n", name, count)
	}

	fmt.Printf("\nLogs exported to: %s\n", outputDir)

	if totalFail > 0 {
		os.Exit(1)
	}
}
