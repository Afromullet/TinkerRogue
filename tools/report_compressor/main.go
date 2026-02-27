package main

import (
	"fmt"
	"log"
	"os"
)

const defaultInputFile = "../../docs/combat_balance_report.csv"
const defaultOutputFile = "combat_balance_compressed.csv"

func main() {
	inputFile := defaultInputFile
	outputFile := defaultOutputFile

	args := os.Args[1:]
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--help", "-h":
			printUsage()
			os.Exit(0)
		case "--input":
			if i+1 >= len(args) {
				log.Fatal("--input requires a path argument")
			}
			i++
			inputFile = args[i]
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

	// Load CSV
	rows, healRows, err := LoadCSV(inputFile)
	if err != nil {
		log.Fatalf("Error loading CSV: %v", err)
	}
	fmt.Printf("Loaded %d damage rows, %d heal rows from %s\n", len(rows), len(healRows), inputFile)

	// Build sections
	units := BuildUnitOverview(rows, healRows)
	fmt.Printf("Unit overview: %d units\n", len(units))

	matchups := BuildCompressedMatchups(rows)
	fmt.Printf("Compressed matchups: %d pairs\n", len(matchups))

	alerts := DetectAlerts(units, matchups)
	fmt.Printf("Balance alerts: %d flagged\n", len(alerts))

	// Write output
	if err := WriteCompressedReport(outputFile, units, matchups, alerts); err != nil {
		log.Fatalf("Error writing report: %v", err)
	}

	fmt.Printf("Compressed report written to %s\n", outputFile)
}

func printUsage() {
	fmt.Println("Combat Balance Report Compressor")
	fmt.Println()
	fmt.Println("Reads a combat_balance_report CSV and produces a compressed")
	fmt.Println("summary with unit overviews, merged matchups, and balance alerts.")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  report_compressor                        Use defaults")
	fmt.Println("  report_compressor --input <path>         Custom input CSV")
	fmt.Println("  report_compressor --output <path>        Custom output CSV")
	fmt.Println("  report_compressor --help                 Show this help")
	fmt.Println()
	fmt.Println("Default input:  " + defaultInputFile)
	fmt.Println("Default output: " + defaultOutputFile)
}
