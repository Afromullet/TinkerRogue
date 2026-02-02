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


  # From TinkerRogue root directory
  go build -o combatsim.exe ./tools/combatsim/cmd



  # Running the combat sim

Change directory to game_main

#Basic Example

 C:\Users\Afromullet\Desktop\TinkerRogue\combatsim.exe -battle-log -num-battles=5


 C:\Users\Afromullet\Desktop\TinkerRogue\combatsim.exe -battle-log -num-battles=3  -squad-gen=balanced 

# Parameters are like this 



------------------



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


---------------------------

# Checking for Dead Code:

deadcode ./...

deadcode -test ./...

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
