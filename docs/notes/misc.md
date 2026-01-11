### Running Unit Tests ###


  go test ./...

  Here are other useful test commands from your project:

  go test ./...                    # Run all tests
  go test ./squads/...            # Run tests in specific package
  go test -v ./...                # Run all tests with verbose output
  go test -cover ./...            # Run all tests with coverage report
 
### Running Combat Sim ###

scripts\run-combatsim.sh


  # Comprehensive analysis
  ./combatsim_test.exe -scenario=1 -iterations=100 -analysis=comprehensive

  # Parameter sweep
  ./combatsim_test.exe -scenario=3 -sweep -sweep-attr=Strength -sweep-min=5 -sweep-max=20 -iterations=50

  # Export to JSON/CSV
  ./combatsim_test.exe -scenario=1 -export-json=report.json -export-csv=metrics.csv


# Checking for Dead Code:

deadcode ./...

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




  tools\combat_visualizer\combat_visualizer.exe game_main\combat_logs\battle_20260111_070533.json >> all_battles.txt