### Running Unit Tests ###


  go test ./...

  Here are other useful test commands from your project:

  go test ./...                    # Run all tests
  go test ./squads/...            # Run tests in specific package
  go test -v ./...                # Run all tests with verbose output
  go test -cover ./...            # Run all tests with coverage report
 

---------------------------

# Checking for Dead Code:

deadcode ./...

deadcode -test ./...


# Checking for Unused Variables

golangci-lint run


---------------------------

### Combat Visualizer ###


# Combat Log Visualizer

The combat visualizer is a standalone tool that generates ASCII visualizations of combat battles
from exported JSON logs. It displays squad formations, attack flows, and detailed statistics.

## Location
  tools/combat_visualizer/

### Running the Combat Visualizer ###

  # 1. Enable export
  # Edit config/config.go: ENABLE_COMBAT_LOG_EXPORT = true


  ____

# Running from Tinker Rogue root dir

  tools\combat_visualizer\combat_visualizer.exe game_main\combat_logs\battle_20260124_065313.641.json >> all_battles.txt
