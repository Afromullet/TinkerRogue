# Combat Simulator

A tactical combat simulation tool for TinkerRogue that runs squad vs squad battles to analyze balance and mechanics.

## Quick Start

### Option 1: Using the Bash Script (Recommended)

The easiest way to run all scenarios:

```bash
# From project root
./scripts/run-combatsim.sh
```

This script automatically:
- Builds the simulator
- Runs all 15 scenarios with 100 iterations each
- Shows colored progress output

### Option 2: Manual Build and Run

#### Build the Simulator

```bash
cd game_main
go build -o combatsim_test.exe ../combatsim/cmd/*.go
```

#### Run Simulations

From the `game_main` directory:

```bash
# Run all 15 scenarios with 100 iterations each
./combatsim_test.exe -scenario=all -iterations=100

# Run a specific scenario (1-15)
./combatsim_test.exe -scenario=5 -iterations=100

# Run with verbose combat logs
./combatsim_test.exe -scenario=1 -iterations=10 -verbose
```

## Command-Line Flags

- `-scenario` - Which scenario(s) to run:
  - `all` - Run all 15 scenarios sequentially (default)
  - `1` to `15` - Run a specific scenario by number
- `-iterations` - Number of simulations per scenario (default: 100)
- `-verbose` - Print detailed combat logs (default: false)

## Test Scenarios

### Core Role Matchups (1-4)

1. **Tank vs Tank** - Knight/Fighter/Fighter vs Paladin/Fighter/Fighter
   - Tests: Cover mechanics, high armor combat, combat duration
   - 3v3, melee range

2. **DPS vs DPS** - Warrior/Swordsman/Rogue vs Assassin/Swordsman/Rogue
   - Tests: High evasion, crit rates, fast combat resolution
   - 3v3, melee range

3. **Tank vs DPS** - Knight/Paladin/Fighter vs Warrior/Swordsman/Rogue
   - Tests: Cover effectiveness vs high damage, tactical balance
   - 3v3, melee range

4. **Ranged vs Melee** - Archer/Crossbowman/Marksman vs Knight/Warrior/Fighter
   - Tests: Distance-based combat, range advantage
   - 3v3, distance=4 (ranged optimal)

### Magic & Support Testing (5-7)

5. **Magic vs Physical** - Wizard/Sorcerer/Mage vs Knight/Fighter/Warrior
   - Tests: Multi-target attacks, AOE patterns, magic damage
   - 3v3, distance=3

6. **Support Heavy** - Cleric/Priest/Paladin vs Warrior/Warrior/Warrior
   - Tests: Leadership/morale, healing potential, support effectiveness
   - 3v3, melee range

7. **Balanced Mixed** - Knight/Archer/Cleric vs Fighter/Wizard/Priest
   - Tests: Well-rounded squad composition balance
   - 3v3, distance=2

### Special Mechanics Tests (8-10)

8. **Multi-Cell Units** - Ogre/Orc Warrior vs Fighter x4
   - Tests: Large unit targeting, 2x2 and 2x1 unit mechanics
   - 2v4 (deliberately imbalanced to test multi-cell durability)

9. **Cover Stacking** - Knight/Knight/Archer vs Warrior/Warrior/Warrior
   - Tests: Multiple cover sources, backline protection
   - 3v3, melee range, specific grid positions for cover overlap

10. **Pierce Through** - Wizard/Sorcerer vs Fighter/Archer (sparse formation)
    - Tests: Pierce-through to back row when front empty
    - 2v2, distance=3

### Edge Cases & Stress Tests (11-15)

11. **Minimum Squad (1v1)** - Fighter vs Fighter
    - Tests: Simplest combat case
    - 1v1, melee range

12. **Size Asymmetry (2v4)** - Knight/Paladin vs Warrior/Swordsman/Rogue/Assassin
    - Tests: Quality vs quantity, outnumbered scenario
    - 2v4 (deliberately imbalanced)

13. **Full AOE Assault** - Wizard/Sorcerer/Warlock vs Knight/Fighter/Paladin/Warrior
    - Tests: Maximum AOE damage, cover against magic
    - 3v4, distance=3

14. **Mixed Range Engagement** - Archer/Scout/Marksman vs Crossbowman/Ranger/Spearman
    - Tests: Various range values (2-4), ranged vs ranged
    - 3v3, distance=3

15. **Goblin Swarm (4v2)** - Goblin Raider x4 vs Knight/Fighter
    - Tests: Many weak units vs few strong units
    - 4v2 (deliberately imbalanced)

## Understanding Results

### Win Rate Analysis
- **Balanced**: 45-55% win rate
- **⚠ Imbalanced**: Outside 45-55% range
- Shows attacker/defender win percentages and draw rate

### Combat Duration
- Average, min, and max turns until combat ends
- Short combats (<3 turns) may indicate low HP or high damage

### Damage Analysis
- Damage dealt and taken per squad
- Average units killed per simulation

### Combat Mechanics
- **Hit Rate**: Percentage of attacks that connect
- **Dodge Rate**: Percentage of attacks dodged
- **Crit Rate**: Percentage of hits that crit
- **Miss Rate**: Percentage of attacks that miss
- **Cover Applications**: How often cover mechanics triggered
- **Cover Reduction**: Average damage reduction from cover

### Insights & Recommendations
- Automatically generated observations about balance
- Suggestions for tuning combat parameters

## File Structure

```
combatsim/
├── cmd/
│   ├── combatsim_main.go    # Main entry point with flag parsing
│   └── test_scenarios.go     # All 15 scenario definitions
├── scenario.go              # Scenario builder and structures
├── simulator.go             # Combat simulation engine
├── statistics.go            # Result aggregation
├── report.go                # Result formatting
└── README.md                # This file
```

## Adding New Scenarios

1. Edit `combatsim/cmd/test_scenarios.go`
2. Create a new function following the pattern:

```go
func createScenario_YourName() combatsim.CombatScenario {
    attackerUnits := []combatsim.UnitConfig{
        {TemplateName: "Knight", GridRow: 0, GridCol: 0, IsLeader: true},
        // ... more units
    }

    defenderUnits := []combatsim.UnitConfig{
        {TemplateName: "Fighter", GridRow: 0, GridCol: 0, IsLeader: true},
        // ... more units
    }

    return combatsim.NewScenarioBuilder("Scenario Name").
        WithAttacker("Squad A", attackerUnits).
        WithDefender("Squad B", defenderUnits).
        WithDistance(1). // 1=melee, 2-3=mid, 4+=ranged
        Build()
}
```

3. Add to `GetAllTestScenarios()` function
4. Rebuild the simulator

## Unit Templates

Available unit types (from `monsterdata.json`):

**Tanks**: Knight, Paladin, Fighter, Spearman, Orc Warrior (2x1), Ogre (2x2)

**DPS**: Warrior, Swordsman, Rogue, Assassin

**Ranged**: Archer, Crossbowman, Marksman, Skeleton Archer

**Magic**: Wizard, Sorcerer, Warlock, Mage, Battle Mage

**Support**: Cleric, Priest, Scout, Ranger

**Other**: Goblin Raider

## Grid Positions

Squads use a 3x3 grid:

```
[0,0] [0,1] [0,2]  ← Front row
[1,0] [1,1] [1,2]  ← Middle row
[2,0] [2,1] [2,2]  ← Back row
```

- Multi-cell units: Specify top-left corner position only
- Leave cells empty for sparse formations (pierce-through testing)
- One unit per squad must have `IsLeader: true`

## Combat Distance

- **1**: Melee range (close combat)
- **2-3**: Mid-range (some ranged units effective)
- **4+**: Full ranged advantage

## Notes

- All unit templates must match names exactly from `monsterdata.json` (case-sensitive)
- Simulations are deterministic for the same seed
- High iteration counts (100+) provide more reliable statistics
- Some scenarios are deliberately imbalanced to stress-test mechanics
