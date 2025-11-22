# Claude Skills and Commands Guide

**Last Updated**: 2025-11-21
**Purpose**: Comprehensive guide to custom Claude Code skills and slash commands for TinkerRogue

---

## Table of Contents

1. [System Overview](#system-overview)
2. [Skills Reference](#skills-reference)
3. [Commands Reference](#commands-reference)
4. [Usage Patterns](#usage-patterns)
5. [Quick Reference Matrix](#quick-reference-matrix)

---

## System Overview

### What Are Skills?

**Skills** are domain knowledge reference materials that provide Claude with:
- Specialized patterns and best practices
- Code examples and templates
- Architecture guidelines
- Quick decision-making frameworks

Skills are **passive knowledge bases** - Claude uses them automatically when relevant to the current task.

### What Are Commands?

**Commands** are **slash commands** (like `/build-verify`) that trigger specific workflows:
- Multi-step procedures
- Analysis tasks
- Code generation workflows
- Quality checks

Commands are **active workflows** - you invoke them explicitly when needed.

### How They Work Together

Skills provide the **knowledge** → Commands apply that **knowledge** in structured workflows

**Example**:
- Skill: `refactor-patterns.md` teaches refactoring patterns
- Command: `/refactor-analyze` applies those patterns to analyze specific code
- Result: Structured refactoring analysis using proven patterns

---

## Skills Reference

### 1. ECS Architecture Skill
**File**: `.claude/skills/ecs-architecture.md`
**LOC**: 108 lines

#### Purpose
Quick ECS compliance checks and pattern suggestions for Go game code

#### When Claude Uses This
Automatically invoked when:
- Working with component definitions
- Writing system functions
- Dealing with entity relationships
- Reviewing code for ECS violations

#### Key Capabilities
- Validate components are pure data (no methods)
- Check EntityID vs entity pointer usage
- Verify query-based relationships (not stored references)
- Suggest system function patterns
- Flag pointer map keys (50x performance issue!)

#### Example Pattern - Pure Data Components
```go
// ✅ GOOD: Pure data, no methods
type Inventory struct {
    ItemEntityIDs []ecs.EntityID
}

// ❌ BAD: Component with logic
type Inventory struct {
    Items []*Item
}
func (inv *Inventory) AddItem(item *Item) { ... }  // ❌ No!
```

#### Reference Implementations
- `squads/*.go` - Perfect ECS: 2675 LOC, 8 components, system-based combat
- `gear/Inventory.go` - Perfect ECS: 241 LOC, pure data + system functions
- `gear/items.go` - EntityID-based relationships (177 LOC)

#### Priority Levels
- **CRITICAL**: Pointer map keys (50x perf impact), entity pointers (crash risk)
- **HIGH**: Component methods (maintainability), missing EntityID usage
- **MEDIUM**: Inconsistent patterns, suboptimal queries
- **LOW**: Style preferences, documentation gaps

---

### 2. Go Testing Patterns Skill
**File**: `.claude/skills/go-testing-patterns.md`
**LOC**: 737 lines

#### Purpose
Generate idiomatic Go test patterns for game code

#### When Claude Uses This
Automatically invoked when:
- Creating test files
- Discussing testing strategies
- Writing benchmarks
- Adding test coverage

#### Key Capabilities
- Table-driven test templates
- Benchmark patterns for performance-critical code
- Mock/stub patterns for ECS systems
- Test helper function suggestions
- Coverage gap identification

#### Example Pattern - Table-Driven Tests
```go
func TestCalculateDamage(t *testing.T) {
    tests := []struct {
        name      string
        attack    int
        defense   int
        minDamage int
        maxDamage int
    }{
        {
            name:      "Normal damage",
            attack:    20,
            defense:   5,
            minDamage: 15,
            maxDamage: 30,  // With crit
        },
        {
            name:      "High defense",
            attack:    10,
            defense:   15,
            minDamage: 1,   // Minimum damage
            maxDamage: 2,   // With crit
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            attacker := &CombatStats{Attack: tt.attack}
            defender := &CombatStats{Defense: tt.defense}

            damage := CalculateDamage(attacker, defender)

            if damage < tt.minDamage || damage > tt.maxDamage {
                t.Errorf("Damage %d out of range [%d, %d]",
                    damage, tt.minDamage, tt.maxDamage)
            }
        })
    }
}
```

#### Testing Commands
```bash
# Run tests with coverage
go test ./... -cover

# Generate coverage report
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out

# Run benchmarks with memory stats
go test -bench=. -benchmem
```

#### Coverage Targets
- Core systems (squad, combat, inventory): >80%
- ECS components: >60%
- GUI code: >50%
- Utility functions: >90%

---

### 3. GUI Mode Patterns Skill
**File**: `.claude/skills/gui-mode-patterns.md`
**LOC**: 263 lines

#### Purpose
Quick UI mode creation and pattern suggestions for ebitenui

#### When Claude Uses This
Automatically invoked when:
- Working with ebitenui widgets
- Creating new GUI modes
- Implementing mode transitions
- Setting up input handlers

#### Key Capabilities
- Mode template generation (BaseMode embedding)
- Widget config pattern suggestions
- Component usage recommendations
- Layout constant selection
- GUIQueries integration patterns

#### Example Pattern - Mode Structure
```go
type NewMode struct {
    gui.BaseMode  // Embed BaseMode for common functionality

    // Mode-specific state
    selectedSquadID ecs.EntityID

    // UI components
    mainContainer *widget.Container
    buttons       []*widget.Button
}

func NewNewMode(manager *ecs.Manager, inputCoord *inputmanager.InputCoordinator) *NewMode {
    mode := &NewMode{}
    mode.InitBaseMode(manager, inputCoord, "NewMode")  // Initialize base

    mode.setupUI()
    mode.setupInputHandlers()

    return mode
}
```

#### Widget Best Practices
1. **Capture Loop Variables**: Always create local variable in loop closures
   ```go
   for _, squad := range squads {
       localSquad := squad  // IMPORTANT: Capture for closure
       button.OnClick = func() { useSquad(localSquad) }
   }
   ```

2. **Use ButtonConfig Pattern**: Consistent button creation
   ```go
   widgets.CreateButtonWithConfig(widgets.ButtonConfig{
       Text:    "Deploy Squad",
       Width:   200,
       Height:  40,
       OnClick: deployHandler,
   })
   ```

3. **Query via GUIQueries**: Don't access ECS directly in UI code
   ```go
   // ✅ GOOD
   data := guiqueries.GetSquadListData(manager)

   // ❌ BAD
   squads := manager.FilterByTag(SquadTag)  // Don't query directly in GUI
   ```

#### Reference Modes
- `gui/squadmanagement/` - List + selection pattern
- `gui/squaddeployment/` - Grid + placement pattern
- `gui/formationeditor/` - 3x3 grid editor pattern
- `gui/squadbuilder/` - Multi-panel creation pattern

---

### 4. Refactoring Patterns Skill
**File**: `.claude/skills/refactor-patterns.md`
**LOC**: 498 lines

#### Purpose
Quick refactoring pattern suggestions for common simplification tasks

#### When Claude Uses This
Automatically invoked when:
- Discussing code duplication
- Addressing complexity
- Tackling technical debt
- Planning simplifications

#### Key Capabilities
- Identify extract function opportunities
- Suggest consolidation patterns
- Recommend separation of concerns
- Flag over-engineering
- Propose interface extractions

#### Core Refactoring Patterns

##### 1. Consolidation Pattern (Type Explosion → Unified Type)
**When to Use**: Multiple similar types with duplicated logic

**Example**: Graphics shapes consolidation (8+ types → 3 variants)
```go
// ❌ BEFORE: 8+ shape types with duplication
type Rectangle struct { ... }
type FilledRectangle struct { ... }
type Circle struct { ... }
// ... 5+ more types

// ✅ AFTER: 1 base type, 3 variants
type BaseShape struct {
    Position coords.PixelPosition
    Color    color.Color
    Variant  ShapeVariant  // Rectangle, Circle, Line
}
```
**Result**: 390 LOC (reduced from 800+ LOC)

##### 2. Strategy Pattern (Hardcoded Algorithms → Pluggable)
**When to Use**: Multiple algorithms for same task, algorithm selection needed

**Example**: Worldmap generator (hardcoded → strategy pattern)
```go
// ✅ Strategy pattern with registry
type MapGenerator interface {
    Name() string
    Generate(width, height int, config GenerationConfig) *GenerationResult
}

var generators = map[string]MapGenerator{
    "rooms_corridors": &RoomsCorridorsGenerator{},
    "tactical_biome":  &TacticalBiomeGenerator{},
}

func NewGameMap(generatorName string) *GameMap {
    gen := generators[generatorName]
    result := gen.Generate(width, height, config)
    return buildMapFromResult(result)
}
```
**Benefits**: Add algorithms without modifying existing code (Open/Closed Principle)

##### 3. Configuration Object Pattern (Parameter Explosion → Config)
**When to Use**: Functions with 5+ parameters

**Example**: Button creation
```go
// ❌ BEFORE: Parameter explosion (8+ parameters)
func CreateButton(text string, x, y, width, height int,
                 onClick func(), textColor, bgColor, hoverColor color.Color,
                 disabled bool, tooltip string) *widget.Button { ... }

// ✅ AFTER: Configuration object
type ButtonConfig struct {
    Text     string
    Position coords.PixelPosition
    Size     coords.PixelSize
    OnClick  func()
    Colors   ButtonColors
    Disabled bool
    Tooltip  string
}

func CreateButtonWithConfig(config ButtonConfig) *widget.Button { ... }
```
**Benefits**: Named parameters (clarity), easier to extend, optional fields

#### TinkerRogue Refactoring Success Stories
1. **Input System** (2025-09-15) - Separation of Concerns
2. **Coordinate System** (2025-09-20) - Extract Function + Type Safety
3. **Entity Templates** (2025-10-06) - Factory → Data-Driven (283 LOC)
4. **Graphics Shapes** (2025-10-08) - Consolidation (390 LOC)
5. **Position System** (2025-10-12) - Value Keys (50x performance!)
6. **Inventory System** (2025-10-21) - ECS Refactor (533 LOC)
7. **GUI Buttons** (2025-11-07) - Configuration Object
8. **Worldmap Generator** (2025-11-08) - Strategy Pattern

**Total**: 8/8 completed = 100% simplification achieved

#### Pattern Selection Matrix
| Symptom | Pattern | Example |
|---------|---------|---------|
| Duplicated logic (3+ places) | Extract Function | CoordinateManager.LogicalToIndex() |
| 5+ similar types | Consolidation | BaseShape (8 → 3 variants) |
| Type doing too much | Separation of Concerns | GameMap refactor |
| Hardcoded algorithm selection | Strategy Pattern | MapGenerator interface |
| 5+ function parameters | Configuration Object | ButtonConfig |
| Need testing flexibility | Interface Extraction | DamageCalculator interface |
| Many factory functions | Entity Template | monsterdata.json |

---

### 5. Tactical Combat Design Skill
**File**: `.claude/skills/tactical-combat.md`
**LOC**: 341 lines

#### Purpose
Tactical RPG mechanics design and balance suggestions

#### When Claude Uses This
Automatically invoked when:
- Discussing squad formations
- Designing combat formulas
- Balancing ability systems
- Analyzing turn order

#### Key Capabilities
- Formation balance analysis (3x3 grid patterns)
- Combat formula suggestions (hit/dodge/crit/cover)
- Ability trigger condition recommendations
- Turn order and initiative systems
- Squad capacity balancing

#### Formation Archetypes

##### 3x3 Grid Tactical Positioning
```
Front Row:    [0,0] [1,0] [2,0]  ← Frontline (high HP/defense)
Middle Row:   [0,1] [1,1] [2,1]  ← Balanced (flexibility)
Back Row:     [0,2] [1,2] [2,2]  ← Support/ranged (protected)
```

##### Formation Designs

**Balanced Formation** (50% win rate vs all):
```
Front:  [Tank]  [DPS]   [DPS]
Middle: [DPS]   [Empty] [Support]
Back:   [Healer][Empty] [Support]
```

**Defensive Formation** (beats Ranged, loses to Offensive):
```
Front:  [Tank]  [Tank]  [Tank]
Middle: [Healer][Empty] [Healer]
Back:   [Support][Support][Support]
```

**Offensive Formation** (beats Defensive, loses to Ranged):
```
Front:  [Tank]  [DPS]   [DPS]
Middle: [DPS]   [DPS]   [DPS]
Back:   [DPS]   [Empty] [Support]
```

**Ranged Formation** (beats Offensive, loses to Defensive):
```
Front:  [Tank]  [Empty] [Tank]
Middle: [DPS]   [Empty] [DPS]
Back:   [Ranged][Ranged][Ranged]
```

#### Combat Formula Design

**Hit/Miss System**:
```go
baseHitChance := 0.75  // 75% base accuracy
dodgeReduction := defender.Dodge * 0.01
finalHitChance := baseHitChance - dodgeReduction
hit := rand.Float64() < finalHitChance
```

**Damage Calculation**:
```go
baseDamage := attacker.Attack - defender.Defense
damage := max(baseDamage, 1)  // Minimum 1 damage

// Critical hits
if rand.Float64() < attacker.CritRate {
    damage *= 2  // 2x damage on crit
}

// Cover mechanics (back row protected)
if targetInBackRow && frontRowAlive {
    damage *= 0.5  // 50% damage reduction
}
```

#### Stat Scaling Guidelines
```
Tank:     HP: 150-200, Attack: 10-15, Defense: 8-12,  Dodge: 3-5%,  Crit: 5-10%
DPS:      HP: 80-100,  Attack: 18-25, Defense: 2-5,   Dodge: 10-15%, Crit: 20-30%
Support:  HP: 70-90,   Attack: 8-12,  Defense: 2-4,   Dodge: 5-10%,  Crit: 10-15%
Healer:   HP: 80-100,  Attack: 6-10,  Defense: 2-4,   Dodge: 5-10%,  Crit: 3-8%
Ranged:   HP: 75-95,   Attack: 15-20, Defense: 2-4,   Dodge: 8-12%,  Crit: 15-25%
```

**Balance Guidelines**:
- Tank HP = 2x DPS HP (survivability role)
- DPS Attack = 1.5-2x Tank Attack (damage role)
- High Dodge ↔ Low HP (glass cannon trade-off)
- High Crit ↔ Lower base Attack (RNG vs consistency)

#### Ability Trigger Types
1. **HP Threshold** (e.g., Rally at HP < 50%) - Comeback mechanics
2. **Turn Count** (e.g., Shield Wall at Turn 1) - Setup abilities
3. **Enemy Count** (e.g., AOE when 3+ enemies) - Situational tactics
4. **Morale/Status** (e.g., Battle Cry when morale high) - Momentum
5. **Combat Start** (e.g., First Strike on turn 1) - Consistent openers

#### Fire Emblem / FFT Inspiration
- **Fire Emblem**: Weapon triangle, terrain bonuses, support bonuses
- **FFT**: Height advantage, charge time, job system
- **Adaptation**: Formation triangle, cover mechanics, abilities with cooldowns

---

## Commands Reference

### 1. /build-verify
**File**: `.claude/commands/build-verify.md`

#### Purpose
Verify build and basic functionality

#### Workflow
1. Run: `go build -o game_main/game_main.exe game_main/*.go`
2. Check: build succeeds without errors
3. Run: `go test ./...` (all tests)
4. Report: test results and failures categorized

#### When to Use
- Before committing code
- After major changes
- Before creating pull requests
- After merging branches
- Verifying clean state

#### Output
- Build success/failure status
- Test results summary
- Categorized failures (CRITICAL/HIGH/MEDIUM/LOW)
- Specific error locations with fixes

#### Example Usage
```bash
# Just type:
/build-verify

# Claude will:
# 1. Build the project
# 2. Run all tests
# 3. Report results with specific fixes if needed
```

---

### 2. /doc-update
**File**: `.claude/commands/doc-update.md`

#### Purpose
Update DOCUMENTATION.md for a specific system

#### Parameters
- `{system}` - System to document (e.g., "squad system", "inventory", "GUI modes")

#### Workflow
1. Read current documentation section
2. Analyze code changes in {system}
3. Identify outdated sections
4. Generate updated documentation:
   - Architecture overview
   - How it works (with code examples)
   - Integration points
   - Key files and LOC counts
   - Current status

#### When to Use
- After completing feature implementation
- After major refactoring
- Before marking task as complete
- When documentation is stale

#### Output
- Updated documentation section
- Concrete code examples from actual files
- Accurate LOC counts
- Current status (% complete)

#### Example Usage
```bash
/doc-update squad system

# Claude will:
# 1. Read current squad system docs
# 2. Analyze squads/*.go files
# 3. Update architecture, examples, status
# 4. Report accurate LOC and completion %
```

---

### 3. /ecs-check
**File**: `.claude/commands/ecs-check.md`

#### Purpose
Analyze code for ECS compliance

#### Parameters
- `{file or package name}` - Target to analyze

#### What It Checks
- Pure data components (no methods)
- EntityID usage (not entity pointers)
- Query-based relationships (not stored references)
- System-based logic (not component methods)
- Value map keys (not pointer keys)

#### Workflow
1. Read target code
2. Scan for ECS violations
3. Categorize by priority (CRITICAL/HIGH/MEDIUM/LOW)
4. Provide code examples showing violations
5. Recommend fixes referencing squad/inventory patterns
6. Separate quick wins from strategic improvements

#### When to Use
- Before committing new ECS code
- When reviewing component definitions
- After refactoring system functions
- When performance seems off (pointer map keys?)
- Code review for ECS compliance

#### Output
1. **Violations Found** (with priority)
2. **Code Examples** (showing problems)
3. **Recommended Fixes** (referencing reference implementations)
4. **Quick Wins** (easy fixes) vs **Strategic Improvements** (bigger refactors)

#### Example Usage
```bash
/ecs-check gear/equipment.go

# Claude will check:
# ✅ Are components pure data?
# ✅ Using EntityID or entity pointers?
# ✅ Query-based or stored references?
# ✅ System functions or component methods?
# ✅ Value map keys or pointer keys?
```

---

### 4. /gui-mode
**File**: `.claude/commands/gui-mode.md`

#### Purpose
Create GUI mode with structured two-phase workflow

#### Parameters
- `{mode-name}` - Name of mode to create (e.g., "squad-equipment", "ability-editor")

#### Phase 1: Design & Planning
1. Layout structure (panels, positions, sizes)
2. Widget hierarchy and data flow
3. ECS queries needed (via GUIQueries)
4. Component usage (SquadListComponent, DetailPanelComponent, etc.)
5. Input handling (hotkeys, button handlers)
6. Mode transitions

#### Phase 2: Implementation Plan
- Create `analysis/gui_{mode-name}_plan_[timestamp].md`
- Ask for approval before implementation
- Follow GUI_PATTERNS.md strictly
- Use BaseMode embedding pattern
- Reference similar modes for consistency

#### When to Use
- Creating new GUI screens
- Adding mode to mode system
- Implementing UI features
- Designing user interactions

#### Output
- Design document with layout diagrams
- Widget hierarchy plan
- ECS query requirements
- Implementation plan (requires approval)
- Final implementation (after approval)

#### Example Usage
```bash
/gui-mode squad-equipment

# Claude will:
# 1. Design layout (panels, widgets)
# 2. Plan ECS queries (via GUIQueries)
# 3. Identify components to use
# 4. Create analysis/gui_squad_equipment_plan_[timestamp].md
# 5. Ask for approval before coding
```

---

### 5. /reality-check
**File**: `.claude/commands/reality-check.md`

#### Purpose
Reality check for claimed feature completion (uses karen agent approach)

#### Parameters
- `{feature}` - Feature to reality check (e.g., "squad abilities", "inventory system")

#### Questions Answered
1. What was claimed to be complete?
2. What actually works end-to-end?
3. Missing functionality (edge cases, error handling, integration)
4. Incomplete implementations (TODOs, hardcoded values, empty catches)
5. Testing gaps (missing tests, untested paths)

#### When to Use
- Task marked complete but seems buggy
- Verifying actual vs claimed progress
- Before marking milestone as done
- When "complete" feature doesn't work as expected
- Planning remaining work for "almost done" features

#### Output
- **Honest completion percentage** (no BS)
- **Specific evidence** (file:line references)
- **Actionable completion plan** (what's actually left)
- **Minimum viable implementation criteria**

#### Philosophy
Uses **karen agent** approach: no BS, just facts. Cuts through:
- "90% complete" that's actually 40% done
- "Just needs polish" that's missing core functionality
- "Works on my machine" that has hardcoded paths
- "Feature complete" with TODO comments everywhere

#### Example Usage
```bash
/reality-check squad abilities

# Claude will:
# 1. Check claimed completion status
# 2. Test actual functionality end-to-end
# 3. Find missing edge cases, error handling
# 4. Identify TODOs, hardcoded values, empty catches
# 5. Report HONEST completion % with evidence
# 6. Provide actionable plan to finish
```

---

### 6. /refactor-analyze
**File**: `.claude/commands/refactor-analyze.md`

#### Purpose
Run refactoring workflow with multi-agent analysis

#### Parameters
- `{target}` - Code to analyze (file, package, or system)

#### Phase 1: Analysis (refactoring-synth agent)
- Current state assessment
- Pain points and duplication
- 2-4 refactoring approaches
- Trade-offs and risk assessment

#### Phase 2: Decision (requires user input)
- Review approaches
- Choose direction
- Document rationale

#### When to Use
- Code has duplication (3+ similar blocks)
- File growing too large (>500 LOC)
- Type explosion (many similar types)
- Hard to test (tight coupling)
- Planning major simplification

#### Output
- Analysis document in `analysis/` directory
- Multiple refactoring approaches with trade-offs
- Risk assessment for each approach
- Recommendation based on project goals
- Requires user decision before implementation

#### Example Usage
```bash
/refactor-analyze gear/equipment.go

# Claude will:
# 1. Analyze current state (pain points, duplication)
# 2. Generate 2-4 refactoring approaches
# 3. Assess trade-offs and risks
# 4. Create analysis/equipment_refactoring_[timestamp].md
# 5. Wait for your decision on approach
```

---

### 7. /squad-balance
**File**: `.claude/commands/squad-balance.md`

#### Purpose
Analyze tactical balance for formations

#### Parameters
- `{formation}` - Formation to analyze (e.g., "Offensive", "Defensive", "Ranged")

#### Analysis Performed
1. Unit distribution (Front/Middle/Back rows)
2. Role balance (Tank/DPS/Support ratios)
3. Coverage mechanics (front protecting back)
4. Capacity cost analysis
5. Formation strengths and weaknesses
6. Comparison to existing formations

#### When to Use
- Designing new formation presets
- Balancing existing formations
- Analyzing formation effectiveness
- Creating formation counters
- Planning squad compositions

#### Output
- Unit distribution breakdown
- Role balance analysis
- Formation archetype (Offensive/Defensive/Ranged/Balanced)
- Strengths and weaknesses
- Counter-relationships (what beats this formation)
- Capacity cost efficiency

#### Reference
- `squads/squadcreation.go` - Formation preset code
- Fire Emblem positioning tactics
- FFT job system balance principles

#### Example Usage
```bash
/squad-balance Offensive

# Claude will:
# 1. Analyze unit placement (Front/Middle/Back)
# 2. Check role distribution (Tank/DPS/Support)
# 3. Evaluate coverage mechanics
# 4. Compare to Defensive, Ranged, Balanced
# 5. Identify strengths/weaknesses
# 6. Suggest counter-formations
```

---

### 8. /test-gen
**File**: `.claude/commands/test-gen.md`

#### Purpose
Generate comprehensive Go tests for a file

#### Parameters
- `{file}` - File to generate tests for (e.g., `squads/squadcombat.go`)

#### What It Generates
1. Table-driven tests for all exported functions
2. Subtests for different scenarios (t.Run)
3. Edge cases: nil checks, empty inputs, boundary values
4. Error path testing
5. Benchmarks for performance-critical functions

#### When to Use
- New file created without tests
- Improving test coverage
- Adding edge case tests
- Creating benchmarks for performance code
- Establishing test baseline

#### Output
Complete test file (`{file}_test.go`) with:
- Table-driven test structure
- Subtests with t.Run()
- Edge case coverage
- Error path validation
- Benchmarks (if applicable)
- TestMain setup (if needed)

#### Reference Patterns
- `squads/squads_test.go` - Table-driven structure
- `squads/squadcombat_test.go` - Combat scenarios
- Go testing best practices

#### Example Usage
```bash
/test-gen squads/squadcombat.go

# Claude will generate:
# 1. TestCalculateDamage (table-driven)
# 2. TestExecuteSquadAttack (subtests for hit/miss/crit)
# 3. Edge cases (nil checks, zero values, boundaries)
# 4. BenchmarkExecuteSquadAttack (performance)
# 5. Output: squads/squadcombat_test.go
```

---

## Usage Patterns

### Pattern 1: Building and Verifying Code

**Scenario**: You've made changes and want to ensure everything still works

**Workflow**:
```bash
1. /build-verify              # Build + test everything
2. Fix any reported issues
3. /build-verify              # Verify fixes
4. Commit when green
```

**Skills Used**: None (command only)

---

### Pattern 2: Adding New GUI Mode

**Scenario**: Need to create a new UI screen

**Workflow**:
```bash
1. /gui-mode {mode-name}      # Generate design plan
2. Review analysis/gui_{mode-name}_plan.md
3. Approve or request changes
4. Claude implements following GUI patterns skill
```

**Skills Used**:
- `gui-mode-patterns.md` - Provides widget/layout patterns
- `ecs-architecture.md` - Ensures proper ECS queries via GUIQueries

---

### Pattern 3: Refactoring Complex Code

**Scenario**: File has grown too large or has code duplication

**Workflow**:
```bash
1. /refactor-analyze {target}              # Multi-agent analysis
2. Review analysis document (2-4 approaches)
3. Choose approach
4. /ecs-check {target}                     # Verify ECS compliance after refactor
5. /build-verify                           # Ensure nothing broke
```

**Skills Used**:
- `refactor-patterns.md` - Provides consolidation/extraction patterns
- `ecs-architecture.md` - Validates ECS compliance
- `go-testing-patterns.md` - Ensures tests cover refactored code

---

### Pattern 4: Implementing Squad Formation

**Scenario**: Creating new formation preset for tactical combat

**Workflow**:
```bash
1. /squad-balance {formation}              # Analyze formation design
2. Implement formation preset code
3. /test-gen squads/squadcombat.go         # Generate combat tests
4. Run combat-simulator agent              # Simulate battles
5. Adjust stats based on win rates
6. /build-verify                           # Final verification
```

**Skills Used**:
- `tactical-combat.md` - Formation archetypes and balance formulas
- `go-testing-patterns.md` - Combat scenario test patterns
- `ecs-architecture.md` - Ensures squad components are pure data

---

### Pattern 5: Reality Checking "Complete" Features

**Scenario**: Feature marked done but you suspect it's incomplete

**Workflow**:
```bash
1. /reality-check {feature}                # Karen agent assessment
2. Review honest completion % and evidence
3. /test-gen {missing-tests}               # Add missing tests
4. Implement identified gaps
5. /build-verify                           # Verify all tests pass
6. /doc-update {system}                    # Update docs with actual state
```

**Skills Used**:
- None directly (reality-check is pure analysis)
- `go-testing-patterns.md` - For adding missing test coverage

---

### Pattern 6: ECS Code Review

**Scenario**: Want to verify new code follows ECS principles

**Workflow**:
```bash
1. /ecs-check {file or package}            # Scan for violations
2. Review violations by priority (CRITICAL → LOW)
3. Fix CRITICAL violations first (pointer keys, entity pointers)
4. /ecs-check {file or package}            # Re-check after fixes
5. /build-verify                           # Ensure tests pass
```

**Skills Used**:
- `ecs-architecture.md` - ECS patterns and anti-patterns
- `refactor-patterns.md` - Refactoring guidance if violations require big changes

---

### Pattern 7: Creating Complete Feature with Tests

**Scenario**: Implementing new feature from scratch

**Workflow**:
```bash
1. Design and implement feature code
2. /test-gen {new-file.go}                 # Generate comprehensive tests
3. /ecs-check {new-file.go}                # Verify ECS compliance
4. Fix any violations
5. /build-verify                           # Build + test everything
6. /doc-update {system}                    # Document new feature
```

**Skills Used**:
- `ecs-architecture.md` - Pure data components, system functions
- `go-testing-patterns.md` - Table-driven tests, edge cases
- `refactor-patterns.md` - Keep code simple, avoid over-engineering

---

## Quick Reference Matrix

### By Task Type

| Task | Primary Command | Supporting Skills | Secondary Commands |
|------|----------------|-------------------|-------------------|
| **Build Verification** | `/build-verify` | None | None |
| **Creating GUI Mode** | `/gui-mode {name}` | gui-mode-patterns, ecs-architecture | `/build-verify` |
| **Refactoring Code** | `/refactor-analyze {target}` | refactor-patterns, ecs-architecture | `/ecs-check`, `/build-verify` |
| **Formation Design** | `/squad-balance {formation}` | tactical-combat | `/test-gen`, `/build-verify` |
| **Test Generation** | `/test-gen {file}` | go-testing-patterns | `/build-verify` |
| **ECS Compliance** | `/ecs-check {target}` | ecs-architecture | `/refactor-analyze`, `/build-verify` |
| **Feature Completion Check** | `/reality-check {feature}` | None | `/test-gen`, `/doc-update` |
| **Documentation Update** | `/doc-update {system}` | None | None |

### By Development Phase

#### Planning Phase
- `/refactor-analyze` - Analyze before big changes
- `/gui-mode` - Design UI before implementation
- `/squad-balance` - Design formations before coding

#### Implementation Phase
- Skills auto-invoked by Claude:
  - `ecs-architecture.md` - When writing components
  - `go-testing-patterns.md` - When creating tests
  - `gui-mode-patterns.md` - When building UI
  - `tactical-combat.md` - When implementing combat

#### Verification Phase
- `/build-verify` - Build + test everything
- `/ecs-check` - Verify ECS compliance
- `/reality-check` - Honest completion assessment
- `/test-gen` - Ensure test coverage

#### Documentation Phase
- `/doc-update` - Update system documentation

---

## Best Practices

### 1. Use Commands for Structured Workflows
Commands enforce a consistent process for common tasks. Don't skip them.

**Good**:
```bash
/refactor-analyze gear/equipment.go  # Structured analysis with trade-offs
```

**Bad**:
```
"Hey Claude, I think equipment.go is messy, can you refactor it?"
# Unstructured, no analysis document, no comparison of approaches
```

---

### 2. Let Skills Work Automatically
Skills are invoked automatically when relevant. You don't need to mention them.

**Good**:
```
"Create a new squad formation component"
# Claude automatically uses ecs-architecture.md and tactical-combat.md
```

**Bad**:
```
"Using the ECS architecture skill and tactical combat skill, create..."
# Unnecessary, Claude knows when to use skills
```

---

### 3. Chain Commands for Complex Workflows
Use multiple commands in sequence for comprehensive tasks.

**Example - Complete Feature Implementation**:
```bash
1. /gui-mode squad-equipment        # Design UI
2. # Implement feature
3. /ecs-check gear/equipment.go     # Verify ECS compliance
4. /test-gen gear/equipment.go      # Generate tests
5. /build-verify                    # Build + test
6. /doc-update equipment system     # Document
7. /reality-check equipment system  # Final verification
```

---

### 4. Use /reality-check Before Marking Complete
Don't trust your own completion assessment. Let karen agent verify.

**Good**:
```bash
/reality-check squad abilities
# Get honest assessment with evidence

# If 100% → Mark complete
# If <100% → See specific gaps and fix them
```

**Bad**:
```
"I think squad abilities are done, let's move on"
# Might miss TODOs, edge cases, integration issues
```

---

### 5. Run /build-verify Frequently
Catch issues early. Don't wait until the end.

**Good**:
```bash
# After implementing feature
/build-verify

# After refactoring
/build-verify

# Before committing
/build-verify

# Before pull request
/build-verify
```

**Bad**:
```
# Work for 3 days
# Try to build
# 47 errors
# Spend 2 hours fixing
```

---

### 6. Use /ecs-check for New ECS Code
ECS violations are easy to introduce. Check early and often.

**Good**:
```bash
# Just wrote new component
/ecs-check squads/squadmorale.go

# Fix violations immediately
# Re-check
/ecs-check squads/squadmorale.go
```

**Bad**:
```
# Write 500 LOC with component methods
# Store entity pointers everywhere
# Use pointer map keys
# Discover performance issues later
# Major refactor required
```

---

### 7. Document with /doc-update
Keep DOCUMENTATION.md current. Future you will thank present you.

**Good**:
```bash
# Completed squad abilities implementation
/doc-update squad system

# Documentation updated with:
# - Ability trigger types
# - Integration points
# - Accurate LOC counts
# - Current status (100%)
```

**Bad**:
```
# Complete feature
# Never update docs
# 6 months later: "How does this work again?"
```

---

## Skill + Command Synergies

### Synergy 1: ECS Architecture Skill + /ecs-check Command
- **Skill**: Provides ECS patterns and anti-patterns
- **Command**: Applies those patterns to scan code for violations
- **Result**: Automated ECS compliance checking with reference fixes

### Synergy 2: Refactoring Patterns Skill + /refactor-analyze Command
- **Skill**: Teaches consolidation, extraction, strategy patterns
- **Command**: Analyzes code and proposes specific pattern applications
- **Result**: Multi-approach refactoring analysis with proven patterns

### Synergy 3: GUI Mode Patterns Skill + /gui-mode Command
- **Skill**: Provides widget patterns, layout constants, component usage
- **Command**: Generates structured UI design plan using those patterns
- **Result**: Consistent GUI implementation following established patterns

### Synergy 4: Tactical Combat Skill + /squad-balance Command
- **Skill**: Teaches formation archetypes, combat formulas, balance principles
- **Command**: Analyzes specific formation using those principles
- **Result**: Balanced formation design with counter-relationships

### Synergy 5: Go Testing Patterns Skill + /test-gen Command
- **Skill**: Provides table-driven tests, benchmark patterns, edge cases
- **Command**: Generates complete test file using those patterns
- **Result**: Comprehensive test coverage following Go best practices

---

## Tips and Tricks

### Tip 1: Use /reality-check for Milestone Verification
Before claiming a milestone is done, reality check it:

```bash
/reality-check squad system
# Returns: "83% complete - Missing formation presets implementation"

# Now you know:
# - What's actually done (8/9 tasks)
# - What's left (formation presets)
# - Accurate completion % (83%, not "almost done")
```

### Tip 2: Chain /ecs-check After Refactoring
Refactoring can introduce ECS violations. Always check:

```bash
/refactor-analyze gear/equipment.go
# Review approaches, choose one, implement

/ecs-check gear/equipment.go
# Verify refactoring didn't break ECS principles

/build-verify
# Ensure tests still pass
```

### Tip 3: Use /test-gen for Legacy Code
Adding tests to untested code? Let command generate the baseline:

```bash
/test-gen legacy/oldcode.go

# Generates:
# - legacy/oldcode_test.go with full coverage
# - Table-driven tests for all functions
# - Edge cases you might have missed

# Review and adjust as needed
```

### Tip 4: Combine /squad-balance + combat-simulator Agent
Formation design should be data-driven:

```bash
/squad-balance NewFormation
# Analyze formation design

# Use combat-simulator agent to validate
# Simulate 1000 battles vs each formation
# Adjust stats based on win rates (target: 45-55%)

/squad-balance NewFormation
# Re-analyze after adjustments
```

### Tip 5: Use /doc-update Proactively
Don't wait until feature is "done". Document as you go:

```bash
# Implemented squad ability triggers
/doc-update squad system

# Implemented squad combat mechanics
/doc-update squad system

# Completed squad visualization
/doc-update squad system

# Result: Always-current documentation
```

---

## Common Mistakes to Avoid

### Mistake 1: Skipping /build-verify
**Problem**: "It works on my machine" → breaks in CI/CD

**Solution**: Run `/build-verify` before every commit

---

### Mistake 2: Trusting Completion Without /reality-check
**Problem**: "Feature is 90% done" → actually 40% done with hardcoded values

**Solution**: Always `/reality-check {feature}` before marking complete

---

### Mistake 3: Writing ECS Code Without /ecs-check
**Problem**: Component methods, entity pointers, pointer map keys (50x slower!)

**Solution**: `/ecs-check` every new component/system file immediately

---

### Mistake 4: Implementing GUI Without /gui-mode
**Problem**: Inconsistent patterns, missing BaseMode, direct ECS access in UI

**Solution**: Start with `/gui-mode {name}` for design plan, then implement

---

### Mistake 5: Refactoring Without /refactor-analyze
**Problem**: Jump straight to refactoring → miss better approaches

**Solution**: `/refactor-analyze` first → review multiple approaches → choose best

---

### Mistake 6: Ignoring Test Coverage
**Problem**: New code without tests → breaks silently later

**Solution**: `/test-gen {file}` immediately after creating new code

---

### Mistake 7: Stale Documentation
**Problem**: DOCUMENTATION.md says "80% complete" but code is 100% done

**Solution**: `/doc-update {system}` after every major milestone

---

## Conclusion

### Skills = Knowledge
- Passive reference materials
- Auto-invoked when relevant
- Teach patterns and principles
- 5 skills covering ECS, testing, GUI, refactoring, combat

### Commands = Workflows
- Active structured procedures
- Explicitly invoked with /command
- Apply skill knowledge systematically
- 8 commands covering verification, analysis, generation, documentation

### Best Results = Skills + Commands
Use commands to trigger workflows that apply skill knowledge systematically.

**Example**:
```bash
# GUI Mode Creation Workflow
/gui-mode squad-equipment
# → Uses gui-mode-patterns skill (widgets, layout)
# → Uses ecs-architecture skill (GUIQueries integration)
# → Generates structured design plan
# → Asks for approval
# → Implements following patterns
```

### Final Recommendations

1. **Run `/build-verify` frequently** - Catch issues early
2. **Use `/reality-check` before marking complete** - No BS completion assessment
3. **Run `/ecs-check` on new ECS code** - Prevent performance issues
4. **Start GUI work with `/gui-mode`** - Consistent patterns
5. **Use `/refactor-analyze` before big changes** - Compare approaches
6. **Generate tests with `/test-gen`** - Comprehensive coverage
7. **Document with `/doc-update`** - Keep docs current
8. **Balance formations with `/squad-balance`** - Tactical depth

---

**Remember**: These tools exist to make development faster and more consistent. Use them liberally. They're here to help!
