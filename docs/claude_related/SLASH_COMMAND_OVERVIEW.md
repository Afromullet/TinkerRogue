
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