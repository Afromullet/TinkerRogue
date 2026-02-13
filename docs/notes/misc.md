# Developer Tools and Commands

Quick reference for common development tasks, testing, and analysis tools.

---

## Running Unit Tests

```bash
# Run all tests
go test ./...

# Run tests in specific package
go test ./squads/...

# Run all tests with verbose output
go test -v ./...

# Run all tests with coverage report
go test -cover ./...
```

---

## Combat Simulator

### Building the Combat Simulator

```bash
# From TinkerRogue root directory
go build -o combatsim.exe ./tools/combatsim/cmd
```

## Code Quality Checks

### Checking for Dead Code

```bash
# Check for dead code
deadcode ./...

# Check for dead code including test files
deadcode -test ./...
```

### Checking for Unused Variables

```bash
golangci-lint run
```

---

## Combat Visualizer

The combat visualizer is a standalone tool that generates ASCII visualizations of combat battles from exported JSON logs. It displays squad formations, attack flows, and detailed statistics.

**Location:** `tools/combat_visualizer/`

### Setup

Enable combat log export by editing `config/config.go`:
```go
ENABLE_COMBAT_LOG_EXPORT = true
```

### Running the Visualizer

```bash
# From TinkerRogue root directory
tools\combat_visualizer\combat_visualizer.exe game_main\simulation_logs\battle_20260208_081424.628.json >> all_battles.txt
```

### Running Combat Simulator Directly

```bash
# From TinkerRogue root directory
cd game_main && go run ../tools/combat_simulator/
```

---

## Combat Balance Report Creator

Generates balance analysis reports from combat simulation logs.

**Requirements:** Simulation logs from the combat simulator

**Location:** `tools/combat_balance/`

### Running the Report Creator

```bash
cd tools/combat_balance && go run .
```



# Running the Combat Simlulation Pipelien Script

Run it from the game_main directory

  scripts\run_combat_pipeline.bat

    Extra arguments get forwarded to the simulator, e.g. scripts\run_combat_pipeline.bat --suite duels.