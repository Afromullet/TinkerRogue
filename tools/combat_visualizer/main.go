package main

import (
	"fmt"
	"log"
	"os"
)

const defaultCombatLogsDir = "../../game_main/combat_logs"

func main() {
	// Check for arguments
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	arg := os.Args[1]

	// Determine which files to process
	var files []string
	var err error

	switch arg {
	case "--latest":
		// Find most recent battle
		file, err := FindLatestBattle(defaultCombatLogsDir)
		if err != nil {
			log.Fatalf("Error finding latest battle: %v", err)
		}
		files = []string{file}

	case "--all":
		// Find all battles
		files, err = FindAllBattles(defaultCombatLogsDir)
		if err != nil {
			log.Fatalf("Error finding battles: %v", err)
		}
		if len(files) == 0 {
			log.Fatalf("No battle files found in %s", defaultCombatLogsDir)
		}

	case "--help", "-h":
		printUsage()
		os.Exit(0)

	default:
		// Assume it's a file path
		files = []string{arg}
	}

	// Process each file
	for i, file := range files {
		// Load battle record
		record, err := LoadBattleRecord(file)
		if err != nil {
			log.Printf("Error loading %s: %v\n", file, err)
			continue
		}

		// Visualize
		output := Visualize(record)

		// Print output
		fmt.Println(output)

		// Add separator between multiple battles (except after last)
		if len(files) > 1 && i < len(files)-1 {
			fmt.Println()
			fmt.Println("════════════════════════════════════════════════════")
			fmt.Println()
		}
	}
}

// printUsage prints command-line usage information.
func printUsage() {
	fmt.Println("Combat Log Visualizer")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  combat_visualizer <file.json>    Visualize a specific battle")
	fmt.Println("  combat_visualizer --latest        Visualize the most recent battle")
	fmt.Println("  combat_visualizer --all           Visualize all battles")
	fmt.Println("  combat_visualizer --help          Show this help message")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  go run tools/combat_visualizer/main.go combat_logs/battle_20251231_035035.json")
	fmt.Println("  go run tools/combat_visualizer/main.go --latest")
	fmt.Println("  go run tools/combat_visualizer/main.go --all > output.txt")
	fmt.Println()
	fmt.Println("Default combat logs directory: " + defaultCombatLogsDir)
}
