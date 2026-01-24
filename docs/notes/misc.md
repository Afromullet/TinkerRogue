### Running Unit Tests ###


  go test ./...

  Here are other useful test commands from your project:

  go test ./...                    # Run all tests
  go test ./squads/...            # Run tests in specific package
  go test -v ./...                # Run all tests with verbose output
  go test -cover ./...            # Run all tests with coverage report
 
### Running Combat Sim ###

scripts\run-combatsim.sh


  # 1. Build the combat simulator


 Correct Build Command (Windows)

  # From TinkerRogue root directory
  go build -o combatsim.exe ./tools/combatsim/cmd

  Or specify the directory:

  go build -o combatsim.exe tools/combatsim/cmd

  Full Process

  # 1. Navigate to project root
  cd C:\Users\Afromullet\Desktop\TinkerRogue

  # 2. Build (use package path, not wildcard)
  go build -o combatsim.exe ./tools/combatsim/cmd

  # 3. Verify the build worked - check for new flags
  combatsim.exe -help

  # 4. Run from game_main directory
  cd game_main
  ..\combatsim.exe -battle-log -num-battles=5

  Alternative: Build in Place

  # Navigate to the cmd directory
  cd C:\Users\Afromullet\Desktop\TinkerRogue\tools\combatsim\cmd

  # Build in current directory
  go build -o combatsim.exe

  # Move back to game_main and run
  cd ..\..\..\game_main
  ..\tools\combatsim\cmd\combatsim.exe -battle-log -num-battles=5

  # 2. Change to game_main directory (required for asset loading)
  cd game_main

  # 3. Run with battle logging enabled
  ../combatsim.exe -battle-log

  That's it! This will run 10 battles (default) and create JSON logs in ./combat_logs/.

  Common Usage Examples

  # Basic: 10 battles, 1v1 format
  ..\combatsim.exe -battle-log



  # More battles for better data
  ..\combatsim.exe -battle-log -num-battles=50

  # Longer battles (more rounds before timeout)
  ..\combatsim.exe -battle-log -num-battles=20 -max-rounds=100

  # Custom output directory
  ..\combatsim.exe -battle-log -log-output-dir=../my_test_logs

  # Multi-squad battles (3-way free-for-all)
  ..\combatsim.exe -battle-log -squads-per-battle=3 -num-battles=10

  # Specific squad generation strategy
  ..\combatsim.exe -battle-log -squad-gen=balanced -num-battles=30
  ..\combatsim.exe -battle-log -squad-gen=random -num-battles=30

  All Available Flags
  ┌────────────────────┬───────────────┬───────────────────────────────────────────┐
  │        Flag        │    Default    │                Description                │
  ├────────────────────┼───────────────┼───────────────────────────────────────────┤
  │ -battle-log        │ false         │ Enable battle logging mode (REQUIRED)     │
  ├────────────────────┼───────────────┼───────────────────────────────────────────┤
  │ -num-battles       │ 10            │ Number of battles to run                  │
  ├────────────────────┼───────────────┼───────────────────────────────────────────┤
  │ -squads-per-battle │ 2             │ Squads per battle (2=1v1, 3+=multi-squad) │
  ├────────────────────┼───────────────┼───────────────────────────────────────────┤
  │ -log-output-dir    │ ./combat_logs │ Directory for JSON output                 │
  ├────────────────────┼───────────────┼───────────────────────────────────────────┤
  │ -squad-gen         │ varied        │ Generation mode: random, balanced, varied │
  ├────────────────────┼───────────────┼───────────────────────────────────────────┤
  │ -max-rounds        │ 50            │ Maximum rounds per battle                 │
  └────────────────────┴───────────────┴───────────────────────────────────────────┘
  Squad Generation Modes

  - varied (recommended): Randomly picks from all strategies - gives diverse data
  - random: Pure random unit selection with role balance
  - balanced: Mixed frontline/backline compositions

  What You Get

  Each battle creates a JSON file like:
  combat_logs/battle_20260124_055244.772.json

  The JSON contains:
  - Battle metadata (ID, timestamps, winner)
  - All combat engagements (round-by-round)
  - Detailed attack events (hit/miss/crit/dodge)
  - Unit performance data
  - Damage breakdowns

  Viewing the Help

  ../combatsim.exe -help


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