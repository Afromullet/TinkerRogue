#!/bin/bash
# Combat Simulation Runner
# Builds and runs all combat simulation scenarios with default settings
#
# Usage:
#   ./scripts/run-combatsim.sh           # Run all scenarios (100 iterations)
#
# Requirements:
#   - Go compiler installed
#   - Run from project root or scripts will auto-navigate
#
# Output:
#   - Builds combatsim_test.exe in game_main/
#   - Runs all 15 scenarios sequentially
#   - Displays detailed simulation reports

set -e  # Exit on error

# Script configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
BUILD_DIR="game_main"
OUTPUT_BINARY="combatsim_test.exe"
DEFAULT_ITERATIONS=100

# Color output for better readability
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Print header
print_header() {
    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE}  Combat Simulation Runner${NC}"
    echo -e "${BLUE}========================================${NC}"
    echo ""
}

# Check if Go is installed
check_go() {
    if ! command -v go &> /dev/null; then
        echo -e "${RED}Error: Go compiler not found${NC}"
        echo "Please install Go from https://golang.org/dl/"
        exit 1
    fi
    echo -e "${GREEN}✓${NC} Go compiler found: $(go version)"
}

# Check project directories exist
check_directories() {
    if [ ! -d "$PROJECT_ROOT/$BUILD_DIR" ]; then
        echo -e "${RED}Error: $BUILD_DIR directory not found${NC}"
        echo "Expected path: $PROJECT_ROOT/$BUILD_DIR"
        exit 1
    fi

    if [ ! -d "$PROJECT_ROOT/combatsim/cmd" ]; then
        echo -e "${RED}Error: combatsim/cmd directory not found${NC}"
        echo "Expected path: $PROJECT_ROOT/combatsim/cmd"
        exit 1
    fi

    echo -e "${GREEN}✓${NC} Project directories verified"
}

# Build the combat simulator
build_simulator() {
    echo ""
    echo -e "${YELLOW}Building combat simulator...${NC}"

    cd "$PROJECT_ROOT/$BUILD_DIR"

    if go build -o "$OUTPUT_BINARY" ../combatsim/cmd/*.go; then
        echo -e "${GREEN}✓${NC} Build successful: $OUTPUT_BINARY"
    else
        echo -e "${RED}✗${NC} Build failed"
        exit 1
    fi
}

# Run all combat simulations
run_simulations() {
    echo ""
    echo -e "${YELLOW}Running all combat scenarios...${NC}"
    echo -e "${BLUE}Scenarios: 15 | Iterations per scenario: $DEFAULT_ITERATIONS${NC}"
    echo ""

    if ./"$OUTPUT_BINARY" -scenario=all -iterations="$DEFAULT_ITERATIONS"; then
        echo ""
        echo -e "${GREEN}✓${NC} All simulations completed successfully"
    else
        echo -e "${RED}✗${NC} Simulation execution failed"
        exit 1
    fi
}

# Display completion summary
print_completion() {
    echo ""
    echo -e "${BLUE}========================================${NC}"
    echo -e "${GREEN}  Combat Simulation Complete${NC}"
    echo -e "${BLUE}========================================${NC}"
    echo ""
    echo "Output binary: $BUILD_DIR/$OUTPUT_BINARY"
    echo "Total scenarios: 15"
    echo "Iterations per scenario: $DEFAULT_ITERATIONS"
    echo ""
}

# Keep window open
keep_window_open() {
    echo -e "${YELLOW}Press any key to close...${NC}"
    read -n 1 -s -r
}

# Main execution
main() {
    print_header
    check_go
    check_directories
    build_simulator
    run_simulations
    print_completion
    keep_window_open
}

# Run the script
main
