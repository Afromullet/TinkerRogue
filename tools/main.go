package main

import (
	"fmt"
	"os"

	balance "game_main/tools/combat_analysis/combat_balance"
	simulator "game_main/tools/combat_analysis/combat_simulator"
	visualizer "game_main/tools/combat_analysis/combat_visualizer"
	compressor "game_main/tools/combat_analysis/report_compressor"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}
	switch os.Args[1] {
	case "simulate", "sim":
		simulator.Run(os.Args[2:])
	case "balance":
		balance.Run(os.Args[2:])
	case "visualize", "viz":
		visualizer.Run(os.Args[2:])
	case "compress":
		compressor.Run(os.Args[2:])
	default:
		fmt.Fprintf(os.Stderr, "unknown subcommand: %s\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Fprintln(os.Stderr, "Usage: combat_tools <subcommand> [options]")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Subcommands:")
	fmt.Fprintln(os.Stderr, "  sim, simulate   Run combat simulations and export battle logs")
	fmt.Fprintln(os.Stderr, "  balance         Aggregate simulation logs into a CSV balance report")
	fmt.Fprintln(os.Stderr, "  viz, visualize  Visualize a battle log as ASCII output")
	fmt.Fprintln(os.Stderr, "  compress        Compress a balance CSV into a summary report")
}
