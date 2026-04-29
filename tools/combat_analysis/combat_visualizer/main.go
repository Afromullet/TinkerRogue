package visualizer

import (
	"fmt"
	"game_main/tools/combat_analysis/shared"
	"log"
	"os"
)

const defaultCombatLogsDir = "../../game_main/combat_logs"

func Run(args []string) {
	if len(args) < 1 {
		printUsage()
		os.Exit(1)
	}

	arg := args[0]

	// Determine which files to process
	var files []string
	var err error

	switch arg {
	case "--latest":
		// Find most recent battle
		file, err := shared.FindLatestBattle(defaultCombatLogsDir)
		if err != nil {
			log.Fatalf("Error finding latest battle: %v", err)
		}
		files = []string{file}

	case "--all":
		// Find all battles
		files, err = shared.FindAllBattles(defaultCombatLogsDir)
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
	fmt.Println("  combat_tools viz <file.json>    Visualize a specific battle")
	fmt.Println("  combat_tools viz --latest        Visualize the most recent battle")
	fmt.Println("  combat_tools viz --all           Visualize all battles")
	fmt.Println("  combat_tools viz --help          Show this help message")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  combat_tools viz combat_logs/battle_20251231_035035.json")
	fmt.Println("  combat_tools viz --latest")
	fmt.Println("  combat_tools viz --all > output.txt")
	fmt.Println()
	fmt.Println("Default combat logs directory: " + defaultCombatLogsDir)
}
