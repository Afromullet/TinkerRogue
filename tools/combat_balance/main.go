package main

import (
	"fmt"
	"log"
	"os"
)

const defaultLogsDir = "../../game_main/simulation_logs"
const defaultOutputFile = "combat_balance_report.csv"

func main() {
	logsDir := defaultLogsDir
	outputFile := defaultOutputFile

	// Parse arguments
	args := os.Args[1:]
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--help", "-h":
			printUsage()
			os.Exit(0)
		case "--dir":
			if i+1 >= len(args) {
				log.Fatal("--dir requires a path argument")
			}
			i++
			logsDir = args[i]
		case "--output":
			if i+1 >= len(args) {
				log.Fatal("--output requires a path argument")
			}
			i++
			outputFile = args[i]
		default:
			log.Fatalf("Unknown argument: %s (use --help for usage)", args[i])
		}
	}

	// Find all battle files
	files, err := FindAllBattles(logsDir)
	if err != nil {
		log.Fatalf("Error finding battles: %v", err)
	}
	if len(files) == 0 {
		log.Fatalf("No battle files found in %s", logsDir)
	}

	fmt.Printf("Found %d battle files in %s\n", len(files), logsDir)

	// Load all records
	var records []*BattleRecord
	for _, file := range files {
		record, err := LoadBattleRecord(file)
		if err != nil {
			fmt.Printf("Warning: skipping %s: %v\n", file, err)
			continue
		}
		records = append(records, record)
	}

	if len(records) == 0 {
		log.Fatal("No valid battle records loaded")
	}

	fmt.Printf("Loaded %d battle records\n", len(records))

	// Aggregate
	result := Aggregate(records)

	fmt.Printf("Found %d unique matchups\n", len(result.Matchups))

	// Write CSV
	if err := WriteCSV(outputFile, result); err != nil {
		log.Fatalf("Error writing CSV: %v", err)
	}

	fmt.Printf("Report written to %s\n", outputFile)
}

func printUsage() {
	fmt.Println("Combat Balance Analyzer")
	fmt.Println()
	fmt.Println("Aggregates combat simulation logs into a CSV report showing")
	fmt.Println("unit-vs-unit matchup statistics for balance tuning.")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  combat_balance                     Process all logs (default directory)")
	fmt.Println("  combat_balance --dir <path>         Override log directory")
	fmt.Println("  combat_balance --output <path>      Override output CSV path")
	fmt.Println("  combat_balance --help               Show this help message")
	fmt.Println()
	fmt.Println("Default log directory: " + defaultLogsDir)
	fmt.Println("Default output file:  " + defaultOutputFile)
}
