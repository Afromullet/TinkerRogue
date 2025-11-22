# Claude Code Extensions Recommendations for TinkerRogue

**Generated:** 2025-11-21
**Version:** 1.0
**Purpose:** Tailored recommendations for Claude Skills, Hooks, Slash Commands, and Custom Subagents based on TinkerRogue development patterns

---

## Table of Contents

1. [Claude Skills](#1-claude-skills)
2. [Claude Hooks](#2-claude-hooks)
3. [Custom Slash Commands](#3-custom-slash-commands)
4. [Custom Subagents](#4-custom-subagents)
5. [Priority Implementation Order](#5-priority-implementation-order)

---

## 1. Claude Skills

Claude Skills are specialized capabilities that can be invoked to perform specific task types. Based on your Go game development workflow and ECS architecture focus, here are recommended skills:

### 1.1 ECS Architecture Skill

**Name:** `ecs-architecture`
**Purpose:** Quick ECS compliance checks and pattern suggestions for Go game code
**Trigger:** When working with component definitions, system functions, or entity relationships

**Capabilities:**
- Validate components are pure data (no methods)
- Check EntityID vs entity pointer usage
- Verify query-based relationships
- Suggest system function patterns
- Flag pointer map keys (performance)

**Why This Fits:**
- Your project heavily emphasizes ECS best practices (CLAUDE.md)
- You have reference implementations (squad system, inventory system)
- Common pattern: reviewing code for ECS violations
- Supports your simplification goals

**Example Usage:**
```
User types code with component methods
Skill auto-triggers: "Component has logic methods. Move to system functions?"
Shows quick fix referencing gear/Inventory.go pattern
```

---

### 1.2 Go Testing Patterns Skill

**Name:** `go-test-patterns`
**Purpose:** Generate idiomatic Go test patterns for game code
**Trigger:** When creating test files or discussing testing strategies

**Capabilities:**
- Table-driven test templates
- Benchmark patterns for performance-critical code
- Mock/stub patterns for ECS systems
- Test helper function suggestions
- Coverage gap identification

**Why This Fits:**
- You use `go-test-writer` agent frequently
- Testing is critical for squad/combat systems (30+ tests)
- Pattern-based testing aligns with your workflow
- Supports quality assurance goals

**Example Usage:**
```
User creates new combat function
Skill suggests: "Table-driven test pattern for hit/dodge/crit scenarios?"
Generates test structure matching squads_test.go style
```

---

### 1.3 Tactical Combat Design Skill

**Name:** `tactical-combat`
**Purpose:** Tactical RPG mechanics design and balance suggestions
**Trigger:** When discussing squad formations, combat formulas, or ability systems

**Capabilities:**
- Formation balance analysis (3x3 grid patterns)
- Combat formula suggestions (hit/dodge/crit/cover)
- Ability trigger condition recommendations
- Turn order and initiative systems
- Squad capacity balancing

**Why This Fits:**
- Core game genre: tactical roguelike
- Existing trpg-creator agent handles implementation
- Complements gameplay design discussions
- Inspired by Fire Emblem, FFT (your references)

**Example Usage:**
```
User: "How should I balance the Offensive formation?"
Skill: "Front-heavy formations should trade defense for damage.
       Recommend: 1 Tank (front), 3 DPS (middle), 1 Support (back)
       Reference: FFT job placement patterns"
```

---

### 1.4 GUI Mode Patterns Skill

**Name:** `gui-mode-patterns`
**Purpose:** Quick UI mode creation and pattern suggestions
**Trigger:** When working with ebitenui widgets or mode system

**Capabilities:**
- Mode template generation (BaseMode embedding)
- Widget config pattern suggestions
- Component usage recommendations
- Layout constant selection
- GUIQueries integration patterns

**Why This Fits:**
- Mode-based GUI system (9 subpackages, 6398 LOC)
- GUI_PATTERNS.md provides foundation
- gui-architect agent for full implementation
- This skill for quick pattern lookups

**Example Usage:**
```
User: "Need a button for squad selection"
Skill: "widgets.CreateButtonWithConfig pattern:
       - Text: squad name
       - OnClick: selection handler
       - Capture loop variable: localID := squadID"
Shows GUI_PATTERNS.md reference
```

---

### 1.5 Refactoring Pattern Skill

**Name:** `refactor-patterns`
**Purpose:** Quick refactoring pattern suggestions for common simplification tasks
**Trigger:** When discussing code duplication, complexity, or technical debt

**Capabilities:**
- Identify extract function opportunities
- Suggest consolidation patterns
- Recommend separation of concerns
- Flag over-engineering
- Propose interface extractions

**Why This Fits:**
- Major theme: simplification (8/8 completed items in CLAUDE.md)
- refactoring-synth agent for deep analysis
- This skill for quick pattern recognition
- Supports continuous improvement

**Example Usage:**
```
User shares code with duplicated logic across 3 files
Skill: "Consolidation pattern detected:
       1. Extract common logic to shared function
       2. Keep domain-specific wrappers
       3. Reference: graphics/drawableshapes.go (8 types → 3)"
```

---

## 2. Claude Hooks

Claude Hooks are shell commands that execute in response to events. Based on your Go development workflow, here are recommended hooks:

### 2.1 Pre-Commit Build Verification

**Hook Type:** `user-prompt-submit-hook` (runs before tool execution)
**Purpose:** Ensure code builds before committing changes
**Trigger:** When user asks Claude to commit changes

**Command:**
```bash
go build -o game_main/game_main.exe game_main/*.go
if [ $? -ne 0 ]; then
    echo "❌ Build failed! Fix errors before committing."
    exit 1
fi
echo "✅ Build successful"
```

**Why This Fits:**
- Build command in CLAUDE.md
- Prevents broken commits
- Fast feedback on compilation errors
- Matches your quality standards

---

### 2.2 Test Suite Verification

**Hook Type:** `user-prompt-submit-hook` (conditional)
**Purpose:** Run relevant tests before marking tasks complete
**Trigger:** When user mentions "complete" or "finished"

**Command:**
```bash
# Detect changed packages
CHANGED_PKGS=$(git diff --name-only HEAD | grep "\.go$" | xargs -I {} dirname {} | sort -u)

if [ -n "$CHANGED_PKGS" ]; then
    echo "Running tests for changed packages..."
    for pkg in $CHANGED_PKGS; do
        go test ./$pkg -v
    done
else
    echo "No Go files changed"
fi
```

**Why This Fits:**
- Test-driven development culture
- Squad system has comprehensive test suite
- Ensures "complete" actually means tested
- Supports karen agent's reality checks

---

### 2.3 ECS Compliance Quick Check

**Hook Type:** `post-edit-hook` (runs after file edits)
**Purpose:** Quick scan for common ECS violations
**Trigger:** After editing component or system files

**Command:**
```bash
FILE=$1
if [[ $FILE == *"/components.go" ]] || [[ $FILE == *"system"* ]]; then
    echo "🔍 Quick ECS check..."

    # Check for component methods (violation)
    if grep -q "^func (.*\*.*) " "$FILE"; then
        echo "⚠️  Possible component method found (ECS violation)"
        echo "   Components should be pure data - move logic to system functions"
    fi

    # Check for entity pointers (violation)
    if grep -q "\*ecs.Entity" "$FILE"; then
        echo "⚠️  Entity pointer found (use ecs.EntityID instead)"
    fi

    # Check for pointer map keys (performance issue)
    if grep -q "map\[\*" "$FILE"; then
        echo "⚠️  Pointer map key found (use value-based keys)"
    fi
fi
```

**Why This Fits:**
- ECS architecture is critical (reference in all docs)
- ecs-reviewer agent for deep analysis
- This hook for immediate feedback
- Prevents common violations early

---

### 2.4 Documentation Update Reminder

**Hook Type:** `post-tool-hook` (runs after tool execution)
**Purpose:** Remind to update docs after significant changes
**Trigger:** After editing core systems (squad, inventory, GUI)

**Command:**
```bash
FILE=$1
if [[ $FILE == *"/squads/"* ]] || [[ $FILE == *"/gui/"* ]] || [[ $FILE == *"/gear/"* ]]; then
    echo "📚 Reminder: Update DOCUMENTATION.md if architecture changed"
    echo "   Modified: $FILE"
fi
```

**Why This Fits:**
- Comprehensive DOCUMENTATION.md (2831 lines)
- Architecture changes should be documented
- Supports maintainability goals
- Gentle reminder without blocking

---

### 2.5 TODO Comment Flagging

**Hook Type:** `post-edit-hook`
**Purpose:** Track and report TODO comments
**Trigger:** After file edits

**Command:**
```bash
FILE=$1
TODOS=$(grep -n "// TODO\|// FIXME\|// HACK" "$FILE" 2>/dev/null)

if [ -n "$TODOS" ]; then
    echo "📌 TODO comments found in $FILE:"
    echo "$TODOS"

    # Count TODOs
    COUNT=$(echo "$TODOS" | wc -l)
    if [ $COUNT -gt 5 ]; then
        echo "⚠️  $COUNT TODOs found - consider creating tasks for these"
    fi
fi
```

**Why This Fits:**
- Prevents TODO comment accumulation
- Supports task tracking discipline
- Aligns with karen agent's completion focus
- Encourages converting TODOs to tracked tasks

---

## 3. Custom Slash Commands

Slash commands are shortcuts that expand to prompts. Based on your development workflows, here are recommended commands:

### 3.1 `/ecs-check [file]`

**Purpose:** Quick ECS compliance review of a file or package
**Expands To:**
```
Please analyze the following for ECS compliance:
- Pure data components (no methods)
- EntityID usage (not entity pointers)
- Query-based relationships (not stored references)
- System-based logic (not component methods)
- Value map keys (not pointer keys)

Target: {file or package name}

Provide:
1. Violations found with priority (CRITICAL/HIGH/MEDIUM/LOW)
2. Code examples showing violations
3. Recommended fixes referencing squad/inventory systems
4. Quick wins vs strategic improvements

Use strict ECS principles from CLAUDE.md.
```

**Why This Fits:**
- Common workflow: checking ECS compliance
- ecs-reviewer agent provides deep analysis
- This command for quick checks
- Matches your quality standards

---

### 3.2 `/test-gen [file]`

**Purpose:** Generate comprehensive tests for a file
**Expands To:**
```
Generate comprehensive Go tests for {file}:

1. Table-driven tests for all exported functions
2. Subtests for different scenarios (t.Run)
3. Edge cases: nil checks, empty inputs, boundary values
4. Error path testing
5. Benchmarks for performance-critical functions

Follow patterns from:
- squads/squads_test.go (table-driven structure)
- squadcombat_test.go (combat scenarios)

Output: Complete test file with TestMain if needed
```

**Why This Fits:**
- go-test-writer agent for full implementation
- This command for quick test generation
- Matches your testing patterns
- Supports TDD workflow

---

### 3.3 `/refactor-analyze [target]`

**Purpose:** Start refactoring workflow analysis
**Expands To:**
```
Run refactoring workflow for {target}:

Phase 1: Analysis (refactoring-synth)
- Current state assessment
- Pain points and duplication
- 2-4 refactoring approaches
- Trade-offs and risk assessment

Phase 2: Decision (requires user input)
- Review approaches
- Choose direction
- Document rationale

Analyze following DEVELOPMENT_WORKFLOWS.md patterns.
Generate analysis document in analysis/ directory.
```

**Why This Fits:**
- Refactoring workflow is well-established
- Matches your two-phase process (analysis → decision)
- refactoring-synth agent handles execution
- Quick command to start workflow

---

### 3.4 `/gui-mode [mode-name]`

**Purpose:** Create new GUI mode following established patterns
**Expands To:**
```
Create GUI mode: {mode-name}

Phase 1: Design & Planning
1. Layout structure (panels, positions, sizes)
2. Widget hierarchy and data flow
3. ECS queries needed (via GUIQueries)
4. Component usage (SquadListComponent, DetailPanelComponent, etc.)
5. Input handling (hotkeys, button handlers)
6. Mode transitions

Phase 2: Implementation Plan
- Create analysis/gui_{mode-name}_plan_[timestamp].md
- Ask for approval before implementation

Follow GUI_PATTERNS.md strictly.
Use BaseMode embedding pattern.
Reference similar modes for consistency.
```

**Why This Fits:**
- gui-architect agent uses two-phase workflow
- This command starts the process
- Matches GUI development patterns
- Ensures planning before coding

---

### 3.5 `/squad-balance [formation]`

**Purpose:** Analyze tactical balance for squad formations
**Expands To:**
```
Analyze tactical balance for {formation} formation:

1. Unit distribution (Front/Middle/Back rows)
2. Role balance (Tank/DPS/Support ratios)
3. Coverage mechanics (front protecting back)
4. Capacity cost analysis
5. Formation strengths and weaknesses
6. Comparison to existing formations (Balanced, Defensive, Offensive, Ranged)

Reference: squads/squadcreation.go formation presets
Tactical principles: Fire Emblem, FFT positioning
```

**Why This Fits:**
- Tactical RPG focus
- Squad system is core (4951 LOC)
- Formation balance is critical
- Supports gameplay design

---

### 3.6 `/reality-check [feature]`

**Purpose:** Validate actual vs claimed completion
**Expands To:**
```
Reality check for {feature}:

1. What was claimed to be complete?
2. What actually works end-to-end?
3. Missing functionality (edge cases, error handling, integration)
4. Incomplete implementations (TODOs, hardcoded values, empty catches)
5. Testing gaps (missing tests, untested paths)

Provide:
- Honest completion percentage
- Specific evidence (file:line references)
- Actionable completion plan
- Minimum viable implementation criteria

Use karen agent approach: no BS, just facts.
```

**Why This Fits:**
- karen agent specializes in this
- Prevents "90% done" syndrome
- Supports quality standards
- Matches completion-focused culture

---

### 3.7 `/build-verify`

**Purpose:** Build and run basic verification
**Expands To:**
```
Verify build and basic functionality:

1. Run: go build -o game_main/game_main.exe game_main/*.go
2. Check: build succeeds without errors
3. Run: go test ./... (all tests)
4. Report: test results and any failures

If build fails: provide specific error locations and fixes
If tests fail: categorize failures (CRITICAL/HIGH/MEDIUM/LOW)
```

**Why This Fits:**
- Build command in CLAUDE.md
- Ensures code quality
- Quick verification workflow
- Prevents broken code

---

### 3.8 `/doc-update [system]`

**Purpose:** Update technical documentation for a system
**Expands To:**
```
Update DOCUMENTATION.md for {system}:

1. Read current documentation section
2. Analyze code changes in {system}
3. Identify outdated sections
4. Generate updated documentation:
   - Architecture overview
   - How it works (with code examples)
   - Integration points
   - Key files and LOC counts
   - Current status

Follow DOCUMENTATION.md structure and style.
Include concrete code examples from actual files.
```

**Why This Fits:**
- Comprehensive DOCUMENTATION.md maintained
- Architecture changes should be documented
- docs-architect agent for full rewrites
- This command for incremental updates

---

## 4. Custom Subagents

Custom subagents are specialized agents with specific expertise. Based on gaps in your current agent suite, here are recommendations:

### 4.1 Performance Profiler Agent

**Name:** `performance-profiler`
**Model:** `sonnet`
**Color:** `red`

**Purpose:** Analyze Go game code for performance bottlenecks and optimization opportunities

**Specialization:**
- Profile ECS query patterns (tag filtering, component access)
- Identify allocation hotspots in game loop
- Analyze spatial grid performance (O(1) vs O(n))
- Suggest caching strategies for frequent queries
- Benchmark comparison (before/after optimizations)

**When to Use:**
- Game loop performance issues
- Frame rate drops or stuttering
- Large entity counts causing slowdowns
- Combat system performance tuning
- Spatial query optimization

**Why This Fits:**
- You've already achieved 50x improvement (position system)
- Performance is critical for roguelike gameplay
- ECS architecture enables optimization opportunities
- Complements existing quality agents

**Example Usage:**
```
User: "Combat slows down with 10+ squads on screen"
Agent analyzes:
- Query frequency in combat system
- Component access patterns
- Spatial grid lookups
- Rendering loop efficiency
Provides: Specific optimization with benchmark comparisons
```

**Workflow:**
1. Read target code (combat, rendering, etc.)
2. Identify performance-critical paths
3. Analyze allocation patterns (`go build -gcflags=-m`)
4. Suggest caching, query optimization, or algorithm changes
5. Generate benchmark tests
6. Document expected improvements with evidence

---

### 4.2 Integration Validator Agent

**Name:** `integration-validator`
**Model:** `haiku` (fast)
**Color:** `blue`

**Purpose:** Verify that systems integrate correctly without breaking existing functionality

**Specialization:**
- Cross-system integration checks (squad ↔ combat ↔ GUI)
- Dependency analysis (what depends on what)
- Breaking change detection
- API compatibility verification
- Integration test generation

**When to Use:**
- After refactoring shared components
- Before merging feature branches
- When modifying core systems (ECS, position system)
- Integration between newly completed systems
- Validating "complete" features actually integrate

**Why This Fits:**
- Complex system interactions (squad, combat, GUI, inventory)
- Refactoring work can break integrations
- karen agent checks completion, this checks integration
- Supports quality assurance goals

**Example Usage:**
```
User: "Refactored squad components, does it break combat?"
Agent checks:
- Combat system's squad component usage
- GUI modes accessing squad data
- Position system integration
- Turn manager squad queries
Reports: Integration status with test coverage gaps
```

**Workflow:**
1. Map system dependencies (what uses what)
2. Identify integration points
3. Check for breaking changes (interface changes, missing fields)
4. Verify existing tests cover integration
5. Suggest integration tests if gaps found
6. Provide integration checklist

---

### 4.3 JSON Data Validator Agent

**Name:** `json-data-validator`
**Model:** `haiku` (fast)
**Color:** `green`

**Purpose:** Validate and optimize JSON game data (monsters, items, weapons)

**Specialization:**
- JSON schema validation
- Game balance analysis (stat distributions)
- Missing data detection
- Duplicate entry identification
- Template consistency checking

**When to Use:**
- Adding new monsters/items/weapons
- Balancing game content
- Validating entitytemplates data
- Before releasing new content
- Debugging spawning issues

**Why This Fits:**
- Entity Template System uses JSON (monsterdata.json, weapondata.json)
- Data-driven design is core architecture
- JSON errors cause runtime issues
- No current agent handles game data validation

**Example Usage:**
```
User: "Added 5 new monsters to monsterdata.json"
Agent validates:
- JSON syntax correctness
- Required fields present (name, health, damage)
- Stat balance (compared to existing monsters)
- Image file existence
- No duplicate names
Reports: Validation results + balance recommendations
```

**Workflow:**
1. Read JSON data file
2. Validate schema (required fields, types)
3. Check for duplicates or inconsistencies
4. Analyze balance (stat distributions, outliers)
5. Verify referenced assets exist (image files)
6. Suggest fixes or balance adjustments

---

### 4.4 Combat Simulator Agent

**Name:** `combat-simulator`
**Model:** `sonnet`
**Color:** `orange`

**Purpose:** Simulate tactical combat scenarios to validate mechanics and balance

**Specialization:**
- Squad vs squad combat simulation
- Formation effectiveness analysis
- Ability trigger probability calculation
- Combat outcome prediction
- Balance recommendations

**When to Use:**
- Testing new combat formulas
- Balancing formations or abilities
- Validating combat system changes
- Analyzing tactical depth
- Answering "what if" balance questions

**Why This Fits:**
- Squad combat is core (4951 LOC)
- Tactical balance is critical
- No current agent simulates gameplay
- Complements trpg-creator design work

**Example Usage:**
```
User: "Will Offensive formation beat Defensive?"
Agent simulates:
- 100 combats with each formation
- Analyzes win rates, damage output, survival
- Identifies formation strengths/weaknesses
- Suggests balance adjustments
Reports: Simulation results with statistical analysis
```

**Workflow:**
1. Parse combat parameters (formations, abilities, stats)
2. Run Monte Carlo simulation (100-1000 iterations)
3. Collect statistics (win rates, damage, kills)
4. Analyze formation matchups
5. Identify balance issues (dominant strategies)
6. Generate report with recommendations

---

### 4.5 Save/Load System Architect Agent

**Name:** `save-load-architect`
**Model:** `sonnet`
**Color:** `cyan`

**Purpose:** Design and implement save/load systems for ECS architecture

**Specialization:**
- ECS serialization strategies
- Component serialization (pure data advantage)
- EntityID mapping across sessions
- Game state snapshot design
- Incremental save optimization

**When to Use:**
- Designing save/load feature
- Adding new components (serialization impact)
- Debugging save/load issues
- Optimizing save file size
- Planning cloud save integration

**Why This Fits:**
- Pure data components enable easy serialization
- EntityID usage supports save/load (vs pointers)
- No current agent handles persistence
- Critical feature for roguelike gameplay

**Example Usage:**
```
User: "How should I implement save/load for squad system?"
Agent analyzes:
- SquadData components (pure data ✅)
- EntityID references (serialization-friendly ✅)
- Query-based relationships (reconstructible ✅)
- Position system state
Provides: Complete save/load architecture with code examples
```

**Workflow:**
1. Analyze current architecture (components, systems)
2. Identify serialization-friendly patterns (EntityID, pure data)
3. Design save format (JSON, binary, hybrid)
4. Plan entity reconstruction strategy
5. Handle edge cases (destroyed entities, references)
6. Generate implementation plan with examples

---

### 4.6 AI Behavior Designer Agent

**Name:** `ai-behavior-designer`
**Model:** `sonnet`
**Color:** `magenta`

**Purpose:** Design and implement enemy AI for tactical combat

**Specialization:**
- Enemy squad behavior patterns
- Formation selection AI
- Ability usage strategies
- Targeting priority systems
- Difficulty scaling algorithms

**When to Use:**
- Implementing enemy AI
- Balancing AI difficulty
- Creating AI personality types
- Testing player strategies
- Designing boss behaviors

**Why This Fits:**
- Enemy squads need AI (mentioned in roadmap)
- Tactical combat requires smart AI
- No current agent handles AI design
- Complements combat-simulator agent

**Example Usage:**
```
User: "Design AI for enemy squads"
Agent provides:
- Formation selection logic (counter player formations)
- Targeting priority (weak units, healers first)
- Ability usage triggers (heal when HP < 30%)
- Retreat conditions (squad morale low)
- Difficulty scaling (stat adjustments)
Implementation plan following ECS patterns
```

**Workflow:**
1. Analyze player capabilities (formations, abilities)
2. Design counter-strategies
3. Create decision trees for AI actions
4. Balance AI difficulty (fair but challenging)
5. Implement as ECS systems (AI components + logic systems)
6. Provide testing scenarios

---

### 4.7 Tutorial System Designer Agent

**Name:** `tutorial-designer`
**Model:** `sonnet`
**Color:** `teal`

**Purpose:** Design progressive tutorial system for complex tactical mechanics

**Specialization:**
- Tutorial flow design (gradual complexity)
- Interactive UI tutorials (GUI mode integration)
- Contextual help system
- Tutorial challenge design
- Player onboarding optimization

**When to Use:**
- Designing tutorial sequences
- Improving new player experience
- Explaining complex mechanics (formations, abilities)
- Creating in-game help
- Testing tutorial effectiveness

**Why This Fits:**
- Complex systems need good tutorials (squad formations, abilities)
- GUI mode system supports tutorial modes
- No current agent handles UX/tutorials
- Critical for player retention

**Example Usage:**
```
User: "Design tutorial for squad formation system"
Agent provides:
- Tutorial progression (basic → advanced)
- Interactive challenges (create balanced formation)
- Contextual tips (formation strengths/weaknesses)
- GUI mode integration (tutorial overlay)
- Success criteria (player understands formations)
```

**Workflow:**
1. Analyze game mechanics complexity
2. Design progressive tutorial steps
3. Create interactive challenges
4. Plan GUI integration (tutorial modes)
5. Define success criteria
6. Generate implementation plan

---

## 5. Priority Implementation Order

Based on immediate value and your current development focus:

### High Priority (Implement First)

1. **Slash Command: `/ecs-check`**
   - **Why:** ECS compliance is critical, this makes it instant
   - **Effort:** 10 minutes to create
   - **Impact:** Daily usage, prevents violations early

2. **Slash Command: `/build-verify`**
   - **Why:** Essential quality check before commits
   - **Effort:** 5 minutes to create
   - **Impact:** Prevents broken builds

3. **Hook: Pre-Commit Build Verification**
   - **Why:** Automated quality gate
   - **Effort:** 15 minutes to configure
   - **Impact:** Catches build errors immediately

4. **Skill: ECS Architecture**
   - **Why:** Frequent pattern recognition need
   - **Effort:** Medium (skill configuration)
   - **Impact:** Faster ECS compliance feedback

### Medium Priority (Next Wave)

5. **Slash Command: `/gui-mode`**
   - **Why:** GUI work is ongoing (60% refactor complete)
   - **Effort:** 10 minutes to create
   - **Impact:** Standardizes GUI development

6. **Subagent: Integration Validator**
   - **Why:** Refactoring work creates integration risk
   - **Effort:** 2-3 hours to create agent
   - **Impact:** Catches breaking changes early

7. **Hook: ECS Compliance Quick Check**
   - **Why:** Immediate feedback on ECS violations
   - **Effort:** 20 minutes to configure
   - **Impact:** Prevents common mistakes

8. **Subagent: Performance Profiler**
   - **Why:** You've seen 50x improvements, more opportunities exist
   - **Effort:** 3-4 hours to create agent
   - **Impact:** Systematic performance optimization

### Lower Priority (As Needed)

9. **Subagent: Combat Simulator**
   - **Why:** Useful for balance, but combat system stable
   - **Effort:** 4-5 hours to create agent
   - **Impact:** Better balance decisions

10. **Subagent: JSON Data Validator**
    - **Why:** Useful when adding content
    - **Effort:** 2-3 hours to create agent
    - **Impact:** Cleaner game data

11. **Slash Commands: Testing, Refactoring, etc.**
    - **Why:** Workflows already established
    - **Effort:** 5-10 minutes each
    - **Impact:** Convenience, slight time savings

12. **Skills: GUI, Tactical, Testing**
    - **Why:** Nice-to-have pattern suggestions
    - **Effort:** Medium (skill configuration)
    - **Impact:** Convenience enhancements

### Future Consideration

13. **Subagent: Save/Load Architect**
    - **Why:** Not yet implemented in game
    - **When:** When starting persistence feature
    - **Impact:** Critical for feature, but not yet needed

14. **Subagent: AI Behavior Designer**
    - **Why:** Enemy AI not yet implemented
    - **When:** After squad system 100% complete
    - **Impact:** Essential for gameplay

15. **Subagent: Tutorial Designer**
    - **Why:** Polish phase feature
    - **When:** After core gameplay complete
    - **Impact:** Better new player experience

---

## Quick Start Guide

### Creating a Slash Command

1. Create file: `.claude/commands/[name].md`
2. Add content (the expanded prompt)
3. Test: type `/[name]` in Claude Code

**Example:** `.claude/commands/ecs-check.md`
```markdown
Please analyze the following for ECS compliance:
- Pure data components (no methods)
- EntityID usage (not entity pointers)
...
```

### Creating a Hook

1. Edit `.claude/settings.json`
2. Add hook configuration under appropriate section
3. Test hook with relevant action

**Example:**
```json
{
  "hooks": {
    "user-prompt-submit-hook": "go build -o game_main/game_main.exe game_main/*.go"
  }
}
```

### Creating a Custom Subagent

1. Create file: `.claude/agents/[name].md`
2. Add frontmatter with metadata
3. Write agent instructions and workflow
4. Test by requesting agent in conversation

**Template:** See `.claude/agents/gui-architect.md` for reference structure

### Creating a Skill

*Note: Skills require Claude Code native support - check documentation for availability*

Recommended approach: Start with slash commands and hooks (immediate value), then add skills when supported.

---

## Summary

**Immediate Actions (30 minutes setup):**
1. Create `/ecs-check` slash command
2. Create `/build-verify` slash command
3. Add pre-commit build verification hook

**High-Value Extensions (next phase):**
- ECS Architecture skill
- Integration Validator subagent
- Performance Profiler subagent
- GUI mode slash command

**Specialized Needs (as required):**
- Combat Simulator (balance work)
- Save/Load Architect (persistence feature)
- AI Behavior Designer (enemy AI)
- Tutorial Designer (onboarding)

**Philosophy:**
Your recommendations prioritize:
- **Quality:** ECS compliance, testing, integration validation
- **Efficiency:** Quick commands for common workflows
- **Safety:** Hooks prevent broken code from being committed
- **Specialization:** Agents for complex domains (performance, combat, AI)

This aligns with your development culture: high standards, systematic workflows, and continuous improvement.

---

**END OF RECOMMENDATIONS**
