# TinkerRogue Development Workflows

**Version:** 3.0
**Last Updated:** 2025-11-29

Comprehensive guide to development workflows in TinkerRogue, with emphasis on agent-driven development for systematic code improvement and feature implementation.

---

## Table of Contents

### Quick Navigation
- [Quick Reference: When to Use Which Agent](#quick-reference-when-to-use-which-agent)
- [Standard Development Cycle](#standard-development-cycle)
- [Common Development Scenarios](#common-development-scenarios)

### Agent-Driven Workflows
- [Agent-Driven Development Overview](#agent-driven-development-overview)
- [Codebase Analysis Workflow](#codebase-analysis-workflow)
- [Code Standards Review Workflow](#code-standards-review-workflow)
- [ECS Architecture Review Workflow](#ecs-architecture-review-workflow)
- [Integration Validation Workflow](#integration-validation-workflow)
- [Feature Implementation Workflow](#feature-implementation-workflow)
- [Refactoring Workflow](#refactoring-workflow)

### Traditional Workflows
- [Bug Fix Workflow](#bug-fix-workflow)
- [Testing Workflow](#testing-workflow)
- [Code Review Workflow](#code-review-workflow)

### Resources
- [Best Practices](#best-practices)
- [Common Pitfalls](#common-pitfalls)
- [Related Documentation](#related-documentation)

---

## Quick Reference: When to Use Which Agent

### Agent Decision Matrix

| When You Need To... | Use This Agent | Required? | When to Run | Output |
|---------------------|---------------|-----------|-------------|--------|
| **Analyze package architecture at scale** | `codebase-analyzer` | Optional | Before large refactorings | `analysis/architecture_analysis_*.md` |
| **Verify Go code standards** | `go-standards-reviewer` | **REQUIRED** | After ANY code changes | `analysis/go_standards_review_*.md` |
| **Validate ECS architecture** | `ecs-reviewer` | **REQUIRED** | After ANY ECS changes | `analysis/ecs_review_*.md` |
| **Check integration safety** | `integration-validator` | **REQUIRED** | After ANY code changes | Integration report |
| **Implement new features** | `feature-implementer` | Optional | For systematic feature dev | Code + commits |
| **Execute refactoring plans** | `refactoring-implementer` | Optional | For systematic refactoring | Code + commits |
| **Write tests** | `go-test-writer` | Recommended | After implementation | `*_test.go` files |

### Priority Flowchart

```
┌──────────────────────────────────────────────────────┐
│         AFTER MAKING ANY CODE CHANGES                │
└────────────────┬─────────────────────────────────────┘
                 │
                 ▼
         ┌──────────────────┐
         │  Did you change  │
         │  code structure? │
         └────┬───────┬─────┘
              │ YES   │ NO
              ▼       ▼
    ┌─────────────────────────┐
    │ Run go-standards-reviewer│
    │      (REQUIRED)          │
    └────────┬────────────────┘
             │
             ▼
    ┌─────────────────────────┐
    │ Did you change ECS code?│
    └────┬───────────┬────────┘
         │ YES       │ NO
         ▼           ▼
┌──────────────────────────────┐
│   Run ecs-reviewer           │
│      (REQUIRED)               │
└────────┬─────────────────────┘
         │
         ▼
┌──────────────────────────────┐
│ Run integration-validator    │
│      (REQUIRED)               │
└────────┬─────────────────────┘
         │
         ▼
    ┌────────────┐
    │ Tests Pass?│
    └┬──────────┬┘
     │ YES      │ NO
     ▼          ▼
   DONE      FIX ISSUES
```

### Agent Combinations by Scenario

| Scenario | Agent Chain | Why |
|----------|-------------|-----|
| **Adding new ECS feature** | `feature-implementer` → `go-standards-reviewer` → `ecs-reviewer` → `integration-validator` | Implement systematically, verify standards, check ECS compliance, ensure integration safety |
| **Refactoring ECS systems** | `refactoring-implementer` → `go-standards-reviewer` → `ecs-reviewer` → `integration-validator` | Execute refactoring plan, verify standards, check ECS compliance, ensure no breakage |
| **Large-scale code reorganization** | `codebase-analyzer` → `refactoring-implementer` → `go-standards-reviewer` → `integration-validator` | Analyze architecture, execute plan, verify standards, check integration |
| **Quick bug fix** | `go-standards-reviewer` → `integration-validator` | Verify standards, ensure no breakage |
| **Performance optimization** | `go-standards-reviewer` (check for allocations) → `integration-validator` | Check performance patterns, ensure no breakage |

---

## Agent-Driven Development Overview

### Core Principles

```
┌────────────────────────────────────────────────────────┐
│           AGENT-DRIVEN DEVELOPMENT PRINCIPLES          │
├────────────────────────────────────────────────────────┤
│                                                        │
│  1. AGENTS ANALYZE, HUMANS DECIDE                     │
│     Agents provide comprehensive analysis              │
│     Developers make informed decisions                 │
│                                                        │
│  2. SYSTEMATIC VALIDATION                              │
│     Every code change is validated                     │
│     Standards are enforced automatically               │
│                                                        │
│  3. INCREMENTAL SAFETY                                 │
│     Work in small, validated steps                     │
│     Create checkpoints for rollback                    │
│                                                        │
│  4. COMPREHENSIVE DOCUMENTATION                        │
│     Every analysis is documented                       │
│     Decisions are traceable                            │
│                                                        │
└────────────────────────────────────────────────────────┘
```

### Mandatory vs. Optional Agents

**Mandatory (Quality Gates):**
- `go-standards-reviewer` - **After ANY code changes**
- `ecs-reviewer` - **After ANY ECS changes**
- `integration-validator` - **After ANY code changes**

**Optional (Productivity Tools):**
- `codebase-analyzer` - For large-scale architectural analysis
- `feature-implementer` - For systematic feature development
- `refactoring-implementer` - For systematic refactoring execution
- `go-test-writer` - For test generation

---

## Codebase Analysis Workflow

> **Purpose:** Analyze package architecture at scale to identify high-value refactoring opportunities

### When to Use

- Before starting large refactoring initiatives
- When planning architectural improvements
- To understand package dependencies
- To identify coupling issues
- NOT for feature-specific analysis (use other agents)

### Workflow

```
┌──────────────────────────────────────────────────────┐
│                CODEBASE ANALYSIS WORKFLOW            │
└────────────────┬─────────────────────────────────────┘
                 │
    Step 1       ▼
    ┌─────────────────────────┐
    │  Invoke Agent           │
    │  "Analyze entire        │
    │   codebase architecture"│
    └────────┬────────────────┘
             │
    Step 2   ▼
    ┌─────────────────────────┐
    │  Review Analysis        │
    │  - Package boundaries   │
    │  - Coupling issues      │
    │  - Recommendations      │
    └────────┬────────────────┘
             │
    Step 3   ▼
    ┌─────────────────────────┐
    │  Make Decisions         │
    │  - Prioritize issues    │
    │  - Choose approaches    │
    │  - Plan refactorings    │
    └────────┬────────────────┘
             │
    Step 4   ▼
    ┌─────────────────────────┐
    │  Document Decisions     │
    │  Save in analysis/      │
    └─────────────────────────┘
```

### Example Usage

```
"Analyze the entire codebase architecture"
"What packages need refactoring?"
"Analyze the combat package architecture"
```

### Output

```
analysis/architecture_analysis_[scope]_[YYYYMMDD].md
- Executive Summary
- Package Analysis (purpose, cohesion, dependencies)
- Dependency Issues (circular deps, inappropriate deps)
- Game Architecture Assessment (ECS, state, input)
- Prioritized Recommendations
```

### Example Scenario

```
Developer: "I want to refactor the combat system, but I'm not sure where to start"

1. Invoke codebase-analyzer: "Analyze the combat package architecture"

2. Review analysis output:
   - Combat package mixes turn management, damage calculation, and UI concerns
   - Tight coupling to squad package creates circular dependency risk
   - CombatState should be separated from combat logic

3. Make decision:
   - Prioritize: Separate CombatState first
   - Then: Extract damage calculation to dedicated module
   - Later: Refactor turn management

4. Document decision in project notes

5. Continue with refactoring workflow using chosen approach
```

---

## Code Standards Review Workflow

> **Purpose:** Verify Go programming standards compliance after ANY code changes

### When to Use

**REQUIRED after:**
- Adding new features
- Refactoring existing code
- Fixing bugs that touch multiple files
- ANY code modifications

### Workflow

```
┌──────────────────────────────────────────────────────┐
│           CODE STANDARDS REVIEW WORKFLOW             │
└────────────────┬─────────────────────────────────────┘
                 │
    Step 1       ▼
    ┌─────────────────────────┐
    │  Make Code Changes      │
    │  (feature, refactor,    │
    │   or bug fix)           │
    └────────┬────────────────┘
             │
    Step 2   ▼
    ┌─────────────────────────┐
    │  Invoke Agent           │
    │  "Review [files/package]│
    │   for Go standards"     │
    └────────┬────────────────┘
             │
    Step 3   ▼
    ┌─────────────────────────┐
    │  Review Violations      │
    │  - Critical (fix now)   │
    │  - High (fix soon)      │
    │  - Medium/Low (later)   │
    └────────┬────────────────┘
             │
    Step 4   ▼
    ┌─────────────────────────┐
    │  Fix Critical Issues    │
    │  Apply recommended fixes│
    └────────┬────────────────┘
             │
    Step 5   ▼
    ┌─────────────────────────┐
    │  Re-run Agent           │
    │  Verify compliance      │
    └─────────────────────────┘
```

### Example Usage

```
"Review combat/attack.go for Go standards compliance"
"Check if the entity package follows Go conventions"
"Analyze the combat package for Go best practices"
```

### Output

```
analysis/go_standards_review_[target]_[YYYYMMDD_HHMMSS].md
- Executive Summary (compliance level, total issues)
- Code Organization violations
- Naming Conventions violations
- Error Handling violations
- Interface Design violations
- Performance Patterns violations (HOT PATH ALLOCATIONS!)
- Concurrency violations
- Priority Matrix (Critical → Low)
- Implementation Roadmap
```

### Critical Focus Areas

**1. Performance (Critical for Game Dev):**
- ❌ **Allocations in hot paths** (render/update loops)
- ❌ **Pointer map keys** (50x slower lookups)
- ❌ **String concatenation in loops**
- ❌ **Defers in performance-critical code**

**2. Go Idioms:**
- ❌ **Name stuttering** (`entity.EntityManager` → `entity.Manager`)
- ❌ **Ignored errors** without justification
- ❌ **Large interfaces** (violates small interface principle)
- ❌ **Premature concurrency**

### Example Scenario

```
Developer: "Just added new combat ability system"

1. Run go-standards-reviewer:
   "Review squads/abilities.go for Go standards"

2. Review finds:
   - CRITICAL: Allocation in UpdateAbilities() (called 60fps)
   - HIGH: Function signature doesn't match Go naming conventions
   - MEDIUM: Error handling could be improved

3. Fix critical issue immediately:
   // ❌ BEFORE
   func UpdateAbilities() {
       for _, ability := range abilities {
           data := make([]byte, 100)  // Allocates every frame!
       }
   }

   // ✅ AFTER
   type AbilityUpdater struct {
       buffer []byte  // Reuse buffer
   }

4. Verify fix:
   "Re-review squads/abilities.go for Go standards"

5. Address high-priority issues in next commit
```

---

## ECS Architecture Review Workflow

> **Purpose:** Verify ECS architecture compliance after ANY ECS code changes

### When to Use

**REQUIRED after:**
- Modifying ECS components
- Adding new ECS systems
- Changing entity relationships
- Modifying query patterns
- ANY ECS-related changes

### Workflow

```
┌──────────────────────────────────────────────────────┐
│             ECS ARCHITECTURE REVIEW WORKFLOW         │
└────────────────┬─────────────────────────────────────┘
                 │
    Step 1       ▼
    ┌─────────────────────────┐
    │  Modify ECS Code        │
    │  (components, systems,  │
    │   queries)              │
    └────────┬────────────────┘
             │
    Step 2   ▼
    ┌─────────────────────────┐
    │  Invoke Agent           │
    │  "Review [files/package]│
    │   for ECS compliance"   │
    └────────┬────────────────┘
             │
    Step 3   ▼
    ┌─────────────────────────┐
    │  Review Violations      │
    │  - Component purity     │
    │  - EntityID usage       │
    │  - Query patterns       │
    │  - System architecture  │
    │  - Map key performance  │
    └────────┬────────────────┘
             │
    Step 4   ▼
    ┌─────────────────────────┐
    │  Fix Violations         │
    │  Reference squad/       │
    │  inventory systems      │
    └────────┬────────────────┘
             │
    Step 5   ▼
    ┌─────────────────────────┐
    │  Verify Compliance      │
    │  Re-run agent           │
    └─────────────────────────┘
```

### Example Usage

```
"Review combat/combatdata.go for ECS compliance"
"Check if the ability system follows ECS patterns"
"Analyze the inventory package for ECS best practices"
```

### Output

```
analysis/ecs_review_[target]_[YYYYMMDD_HHMMSS].md
- Executive Summary (ECS compliance level)
- Pure Data Components analysis
- Native EntityID Usage analysis
- Query-Based Relationships analysis
- System-Based Logic analysis
- Value Map Keys analysis (PERFORMANCE CRITICAL)
- Reference violations (compared to squad/inventory)
- ECS Compliance Scorecard
- Alignment with reference implementations
```

### Five ECS Principles

**1. Pure Data Components:**
```go
// ✅ CORRECT - Pure data
type SquadData struct {
    Name     string
    Members  []ecs.EntityID
}

// ❌ WRONG - Has logic method
func (s *SquadData) AddMember(id ecs.EntityID) { ... }
```

**2. Native EntityID Usage:**
```go
// ✅ CORRECT - Use EntityID
type Item struct {
    Properties ecs.EntityID
}

// ❌ WRONG - Entity pointer
type Item struct {
    Properties *ecs.Entity
}
```

**3. Query-Based Relationships:**
```go
// ✅ CORRECT - Query on demand
func GetSquadMembers(manager *ecs.Manager, squadID ecs.EntityID) []*ecs.Entity {
    // Query for members
}

// ❌ WRONG - Cached references
type Squad struct {
    Members []*ecs.Entity
}
```

**4. System-Based Logic:**
```go
// ✅ CORRECT - System function
func ApplyDamage(manager *ecs.Manager, targetID ecs.EntityID, damage int) {
    // Logic here
}

// ❌ WRONG - Component method
func (c *Combat) TakeDamage(amount int) { ... }
```

**5. Value Map Keys:**
```go
// ✅ CORRECT - Value key (50x faster)
grid := make(map[coords.LogicalPosition]ecs.EntityID)

// ❌ WRONG - Pointer key
grid := make(map[*coords.LogicalPosition]ecs.EntityID)
```

### Example Scenario

```
Developer: "Added new ability component to squad system"

1. Run ecs-reviewer:
   "Review squads/abilities.go for ECS compliance"

2. Review finds:
   - CRITICAL: Component has method (violates pure data principle)
   - HIGH: Uses *ecs.Entity instead of ecs.EntityID
   - Reference: squad system uses pure components

3. Fix violations:
   // ❌ BEFORE
   type AbilityData struct {
       Name string
       Power int
   }
   func (a *AbilityData) Activate() { ... }  // Method on component!

   // ✅ AFTER
   type AbilityData struct {
       Name string
       Power int
   }
   // System function instead
   func ActivateAbility(manager *ecs.Manager, abilityID ecs.EntityID) { ... }

4. Verify compliance:
   "Re-review squads/abilities.go for ECS compliance"

5. Commit with clean ECS architecture
```

---

## Integration Validation Workflow

> **Purpose:** Verify cross-system integration safety after ANY code changes

### When to Use

**REQUIRED after:**
- Refactoring shared components
- Modifying core systems (ECS, coordinates, input)
- Changing component structures
- Modifying function signatures
- ANY code changes that could affect other systems

### Workflow

```
┌──────────────────────────────────────────────────────┐
│          INTEGRATION VALIDATION WORKFLOW             │
└────────────────┬─────────────────────────────────────┘
                 │
    Step 1       ▼
    ┌─────────────────────────┐
    │  Make Code Changes      │
    │  (refactor, feature,    │
    │   modification)         │
    └────────┬────────────────┘
             │
    Step 2   ▼
    ┌─────────────────────────┐
    │  Invoke Agent           │
    │  "Validate integration  │
    │   for [changes]"        │
    └────────┬────────────────┘
             │
    Step 3   ▼
    ┌─────────────────────────┐
    │  Review Analysis        │
    │  - Breaking changes     │
    │  - Affected systems     │
    │  - Integration points   │
    │  - Test coverage        │
    └────────┬────────────────┘
             │
    Step 4   ▼
    ┌─────────────────────────┐
    │  Fix Breaking Changes   │
    │  Update all call sites  │
    │  Add integration tests  │
    └────────┬────────────────┘
             │
    Step 5   ▼
    ┌─────────────────────────┐
    │  Verify Integration     │
    │  - Tests pass           │
    │  - No compile errors    │
    │  - Systems work together│
    └─────────────────────────┘
```

### Example Usage

```
"Validate integration after refactoring squad components"
"Check integration safety for combat system changes"
"Verify no breaking changes in position system refactor"
```

### Output

```
Integration Validation Report
- Dependency Map (who depends on what)
- Integration Points Analysis
- Breaking Changes Detected (CRITICAL/HIGH/MEDIUM/LOW)
- Integration Test Coverage assessment
- Integration Checklist
- Risk Level (LOW/MEDIUM/HIGH/CRITICAL)
```

### Common Breaking Changes

**1. Component Structure Changes:**
```go
// BEFORE
type SquadData struct {
    Name     string
    Members  []ecs.EntityID
}

// AFTER (BREAKING)
type SquadData struct {
    Name string
    // Members removed - breaks GetSquadMembers!
}
```

**2. Function Signature Changes:**
```go
// BEFORE
func AddItem(manager *ecs.Manager, inv *Inventory, itemID ecs.EntityID)

// AFTER (BREAKING - added parameter)
func AddItem(manager *ecs.Manager, inv *Inventory, itemID ecs.EntityID, quantity int)
```

**3. Query Return Type Changes:**
```go
// BEFORE
func GetSquadMembers(...) []*ecs.Entity

// AFTER (BREAKING - changed return type)
func GetSquadMembers(...) []ecs.EntityID
```

### Example Scenario

```
Developer: "Refactored Position component to remove redundant field"

1. Run integration-validator:
   "Validate integration after position component refactor"

2. Review finds:
   - CRITICAL: 15 systems access removed Position.TileIndex field
   - HIGH: Combat system depends on old field for spatial queries
   - Breaking changes in: combat/, rendering/, movement/

3. Integration report shows:
   - Combat system: 7 call sites broken
   - Rendering system: 3 call sites broken
   - Movement system: 5 call sites broken

4. Fix all breaking changes:
   - Update combat system to use new field
   - Update rendering to use coordinate manager
   - Update movement queries

5. Add integration tests:
   - Test combat ↔ position integration
   - Test rendering ↔ position integration
   - Test movement ↔ position integration

6. Verify integration:
   - All tests pass
   - No compile errors
   - Systems work together correctly

7. Safe to commit
```

---

## Feature Implementation Workflow

> **Purpose:** Systematically implement new features with safety checkpoints

### When to Use

- Implementing complex new features
- Building new game mechanics
- Adding new systems to codebase
- When you want systematic, safe execution with rollback points

### Workflow

```
┌──────────────────────────────────────────────────────┐
│          FEATURE IMPLEMENTATION WORKFLOW             │
└────────────────┬─────────────────────────────────────┘
                 │
    Step 1       ▼
    ┌─────────────────────────┐
    │  Define Requirements    │
    │  - Feature description  │
    │  - Success criteria     │
    │  - Constraints          │
    └────────┬────────────────┘
             │
    Step 2   ▼
    ┌─────────────────────────┐
    │  Invoke Agent           │
    │  "Implement [feature]"  │
    │  OR provide plan file   │
    └────────┬────────────────┘
             │
    Step 3   ▼
    ┌─────────────────────────┐
    │  Agent Proposes Plan    │
    │  - Milestones           │
    │  - Dependencies         │
    │  - Integration points   │
    └────────┬────────────────┘
             │
    Step 4   ▼
    ┌─────────────────────────┐
    │  Review & Approve Plan  │
    │  Make adjustments       │
    └────────┬────────────────┘
             │
    Step 5   ▼
    ┌─────────────────────────┐
    │  Agent Creates Initial  │
    │  Git Checkpoint (LOCAL) │
    └────────┬────────────────┘
             │
             ▼
    ┌─────────────────────────┐
    │  FOR EACH MILESTONE:    │
    │  1. Implement milestone │
    │  2. Run relevant tests  │
    │  3. Create checkpoint   │
    │  4. Ask for confirmation│
    └────────┬────────────────┘
             │
    Step 6   ▼
    ┌─────────────────────────┐
    │  Run Quality Gates      │
    │  - go-standards-reviewer│
    │  - ecs-reviewer         │
    │  - integration-validator│
    └────────┬────────────────┘
             │
    Step 7   ▼
    ┌─────────────────────────┐
    │  Write Tests            │
    │  Use go-test-writer     │
    └─────────────────────────┘
```

### Agent Capabilities

**With Implementation Plan File:**
- Reads and parses implementation plan
- Verifies prerequisites exist
- Breaks down into milestones
- Creates LOCAL git checkpoints

**Without Implementation Plan File:**
- Researches codebase patterns
- Designs implementation approach
- Proposes milestones for approval
- Follows existing ECS/Go patterns

### Safety Protocols

**Agent WILL:**
- Stop between milestones for confirmation
- Stop immediately on errors
- Create LOCAL checkpoints (never push)
- Run relevant tests after changes
- Follow ECS best practices

**Agent WON'T:**
- Proceed without confirmation
- Attempt automatic error fixes
- Push to remote repository
- Make improvements beyond plan without approval
- Create new branches

### Example Usage

**With Plan File:**
```
"Implement feature from tactical_cover_system_plan.md"
```

**Without Plan File:**
```
"Add a morale system to squad combat - units get bonuses/penalties based on events"
```

### Example Scenario

```
Developer: "I want to add a cover system to tactical combat"

1. Invoke feature-implementer:
   "Add cover system to tactical combat - units can take cover behind obstacles for defense bonus"

2. Agent researches:
   - Reviews combat system architecture
   - Studies squad ability patterns
   - Checks position system integration

3. Agent proposes plan:
   "Based on existing combat system, I propose:
    - Milestone 1: Add CoverComponent (pure data)
    - Milestone 2: Create cover detection system functions
    - Milestone 3: Integrate cover into combat calculations
    - Milestone 4: Add cover visualization to combat UI

    Does this approach work?"

4. Developer approves plan

5. Agent executes:
   - Creates initial checkpoint
   - Implements Milestone 1 (CoverComponent)
   - Runs tests
   - Creates checkpoint "Feature: Cover System - Add CoverComponent"
   - STOPS: "Milestone 1 completed. Tests pass. Continue?"

6. Developer confirms, agent continues through milestones

7. After completion:
   - Run go-standards-reviewer on new files
   - Run ecs-reviewer on CoverComponent
   - Run integration-validator for combat integration
   - Write tests with go-test-writer

8. Feature complete with quality gates passed
```

---

## Refactoring Workflow

> **Purpose:** Systematically execute refactoring plans with safety checkpoints

### When to Use

- Executing planned refactorings
- Improving code structure
- Consolidating duplicate code
- Simplifying complex systems
- When you want systematic, safe execution with rollback points

### Workflow

```
┌──────────────────────────────────────────────────────┐
│              REFACTORING WORKFLOW                    │
└────────────────┬─────────────────────────────────────┘
                 │
    Step 1       ▼
    ┌─────────────────────────┐
    │  Identify Refactoring   │
    │  Target                 │
    │  (optional: use         │
    │   codebase-analyzer)    │
    └────────┬────────────────┘
             │
    Step 2   ▼
    ┌─────────────────────────┐
    │  Invoke Agent           │
    │  "Implement refactoring │
    │   from [plan file]"     │
    └────────┬────────────────┘
             │
    Step 3   ▼
    ┌─────────────────────────┐
    │  Agent Parses Plan      │
    │  - Verifies files exist │
    │  - Creates todo list    │
    │  - Identifies deps      │
    └────────┬────────────────┘
             │
    Step 4   ▼
    ┌─────────────────────────┐
    │  Review & Approve Plan  │
    └────────┬────────────────┘
             │
    Step 5   ▼
    ┌─────────────────────────┐
    │  Agent Creates Initial  │
    │  Git Checkpoint (LOCAL) │
    └────────┬────────────────┘
             │
             ▼
    ┌─────────────────────────┐
    │  FOR EACH SECTION:      │
    │  1. Implement changes   │
    │  2. Run relevant tests  │
    │  3. Create checkpoint   │
    │  4. Ask for confirmation│
    └────────┬────────────────┘
             │
    Step 6   ▼
    ┌─────────────────────────┐
    │  Run Quality Gates      │
    │  - go-standards-reviewer│
    │  - ecs-reviewer (if ECS)│
    │  - integration-validator│
    └────────┬────────────────┘
             │
    Step 7   ▼
    ┌─────────────────────────┐
    │  Update/Write Tests     │
    │  Use go-test-writer     │
    └─────────────────────────┘
```

### Agent Capabilities

**Reads Refactoring Plans:**
- Parses plan file structure
- Verifies all mentioned files exist
- Breaks down into major sections
- Creates step-by-step execution plan

**Executes Systematically:**
- Implements one section at a time
- Runs tests after each section
- Creates LOCAL checkpoints
- Stops for confirmation between sections

### Safety Protocols

**Agent WILL:**
- Stop between sections for confirmation
- Stop immediately on errors
- Create LOCAL checkpoints (never push)
- Run relevant tests after changes
- Verify files exist before starting

**Agent WON'T:**
- Proceed without confirmation
- Attempt automatic error fixes
- Push to remote repository
- Make improvements beyond plan without approval
- Create new branches

### Example Usage

```
"Implement refactoring from combat_consolidation_plan.md"
```

### Example Scenario

```
Developer: "I have a refactoring plan to consolidate combat calculations"

1. Create refactoring plan:
   File: analysis/combat_consolidation_plan.md
   - Section 1: Extract damage calculation to combat/damage.go
   - Section 2: Consolidate hit chance logic
   - Section 3: Update call sites

2. Invoke refactoring-implementer:
   "Implement refactoring from combat_consolidation_plan.md"

3. Agent verifies:
   - Reads plan
   - Checks all files exist
   - Creates todo list:
     - [ ] Initial checkpoint
     - [ ] Section 1: Extract damage calculation
     - [ ] Section 2: Consolidate hit chance logic
     - [ ] Section 3: Update call sites

4. Agent asks: "Plan verified. Proceed with initial checkpoint?"

5. Developer approves

6. Agent executes:
   - Creates checkpoint "Pre-refactoring checkpoint"
   - Implements Section 1
   - Runs tests
   - Creates checkpoint "Refactoring: Extract damage calculation"
   - STOPS: "Section 1 complete. Tests pass. Continue?"

7. Developer confirms, agent continues through sections

8. After completion:
   - Run go-standards-reviewer on modified files
   - Run ecs-reviewer if combat components changed
   - Run integration-validator to ensure no breakage
   - Update tests with go-test-writer

9. Refactoring complete with quality gates passed
```

---

## Bug Fix Workflow

> **Purpose:** Fix bugs quickly while maintaining code quality

### When to Use

- Fixing reported bugs
- Addressing test failures
- Correcting incorrect behavior
- Resolving crashes or errors

### Workflow

```
┌──────────────────────────────────────────────────────┐
│                  BUG FIX WORKFLOW                    │
└────────────────┬─────────────────────────────────────┘
                 │
    Step 1       ▼
    ┌─────────────────────────┐
    │  Identify Bug           │
    │  - Reproduce issue      │
    │  - Locate code          │
    │  - Understand cause     │
    └────────┬────────────────┘
             │
    Step 2   ▼
    ┌─────────────────────────┐
    │  Write Failing Test     │
    │  Demonstrates bug       │
    └────────┬────────────────┘
             │
    Step 3   ▼
    ┌─────────────────────────┐
    │  Implement Fix          │
    │  Minimal changes needed │
    └────────┬────────────────┘
             │
    Step 4   ▼
    ┌─────────────────────────┐
    │  Verify Fix             │
    │  - Test passes          │
    │  - Bug doesn't recur    │
    │  - No regressions       │
    └────────┬────────────────┘
             │
    Step 5   ▼
    ┌─────────────────────────┐
    │  Run Quality Gates      │
    │  - go-standards-reviewer│
    │  - integration-validator│
    └────────┬────────────────┘
             │
    Step 6   ▼
    ┌─────────────────────────┐
    │  Commit Fix             │
    │  Clear commit message   │
    └─────────────────────────┘
```

### Quality Gates for Bug Fixes

**Always Run:**
- `go-standards-reviewer` - Check if fix introduces issues
- `integration-validator` - Ensure fix doesn't break other systems

**Run if Applicable:**
- `ecs-reviewer` - If fix touches ECS code

### Best Practices

**Write Test First:**
```go
// 1. Write failing test
func TestCombatDamageCalculation(t *testing.T) {
    damage := CalculateDamage(10, 5)
    assert.Equal(t, 5, damage)  // FAILS - exposes bug
}

// 2. Fix code
func CalculateDamage(attack, defense int) int {
    return max(0, attack - defense)  // Now handles negative correctly
}

// 3. Test passes
```

**Minimal Changes:**
- Fix only what's broken
- Don't refactor during bug fix
- Save improvements for separate commits

**Clear Commit Messages:**
```
Fix: Combat damage calculation handles negative defense

Previously, negative defense values caused incorrect damage.
Now properly clamps damage to minimum of 0.

Fixes #123
```

### Example Scenario

```
Developer: "Squad formation doesn't update when squad moves"

1. Reproduce issue:
   - Create squad with formation
   - Move squad
   - Formation stays at old position

2. Locate code:
   - squads/squadcombat.go - movement function
   - squads/formations.go - formation update

3. Write failing test:
   func TestFormationUpdatesOnMove(t *testing.T) {
       squad := CreateSquadAt(Position{5, 5})
       MoveSquad(squad, Position{10, 10})

       formation := GetFormation(squad)
       assert.Equal(t, Position{10, 10}, formation.Center)  // FAILS
   }

4. Implement fix:
   func MoveSquad(squadID ecs.EntityID, newPos coords.LogicalPosition) {
       // Existing move code...

       // FIX: Update formation position
       UpdateFormationPosition(squadID, newPos)
   }

5. Verify:
   - Test now passes
   - Formation updates correctly
   - No regressions in other tests

6. Run quality gates:
   - go-standards-reviewer: "Review squads/squadcombat.go"
   - integration-validator: "Validate after formation fix"

7. Commit:
   "Fix: Update formation position when squad moves

    Previously formation stayed at old position after squad moved.
    Now UpdateFormationPosition is called after movement.

    Test: TestFormationUpdatesOnMove
    Fixes #234"

8. Done
```

---

## Testing Workflow

> **Purpose:** Ensure code quality through comprehensive testing

### When to Use

- After implementing features
- After refactoring
- When test coverage is insufficient
- Before merging changes

### Workflow

```
┌──────────────────────────────────────────────────────┐
│                   TESTING WORKFLOW                   │
└────────────────┬─────────────────────────────────────┘
                 │
    Step 1       ▼
    ┌─────────────────────────┐
    │  Identify Test Needs    │
    │  - New features         │
    │  - Refactored code      │
    │  - Coverage gaps        │
    └────────┬────────────────┘
             │
    Step 2   ▼
    ┌─────────────────────────┐
    │  Generate Tests         │
    │  Use go-test-writer     │
    │  "Write tests for [file]"│
    └────────┬────────────────┘
             │
    Step 3   ▼
    ┌─────────────────────────┐
    │  Review Generated Tests │
    │  - Coverage adequate?   │
    │  - Edge cases included? │
    │  - Domain logic correct?│
    └────────┬────────────────┘
             │
    Step 4   ▼
    ┌─────────────────────────┐
    │  Customize Tests        │
    │  Add domain-specific    │
    │  scenarios              │
    └────────┬────────────────┘
             │
    Step 5   ▼
    ┌─────────────────────────┐
    │  Run Tests              │
    │  go test ./...          │
    └────────┬────────────────┘
             │
    Step 6   ▼
    ┌─────────────────────────┐
    │  Fix Failing Tests      │
    │  Update implementation  │
    │  or test expectations   │
    └─────────────────────────┘
```

### Test Types

**Unit Tests:**
```bash
go test ./squads/...              # Test squad package
go test ./combat/...              # Test combat package
go test -v ./...                  # Verbose output
go test -cover ./...              # With coverage
```

**Integration Tests:**
```go
// Test cross-system integration
func TestSquadCombatIntegration(t *testing.T) {
    manager := ecs.NewManager()
    squadID := CreateSquadWithMembers(manager, "Alpha", 3)
    targetID := CreateTestEnemy(manager)

    result := ExecuteSquadAttack(manager, squadID, targetID)

    assert.NotNil(t, result)
    members := GetSquadMembers(manager, squadID)
    assert.Equal(t, 3, len(members))
}
```

**Performance Tests:**
```go
// Benchmark hot paths
func BenchmarkRenderLoop(b *testing.B) {
    setup()
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        RenderFrame()
    }
}
```

### Using go-test-writer

**Example Usage:**
```
"Write tests for squads/squadcombat.go"
"Create tests for the inventory system"
"Generate tests for combat/damage.go implementing damage calculation"
```

**Output:**
- Table-driven test functions
- Subtests with `t.Run()`
- Test helper functions
- Benchmarks for performance
- Edge case coverage

**Customize Generated Tests:**
```go
// Generated by go-test-writer
func TestCalculateDamage(t *testing.T) {
    tests := []struct {
        name     string
        attack   int
        defense  int
        expected int
    }{
        {"normal damage", 10, 3, 7},
        {"zero defense", 10, 0, 10},
        {"negative defense", 10, -5, 15},  // ADD: Domain-specific case
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := CalculateDamage(tt.attack, tt.defense)
            assert.Equal(t, tt.expected, result)
        })
    }
}
```

### Example Scenario

```
Developer: "Just implemented new ability system in squads/abilities.go"

1. Generate tests:
   "Write tests for squads/abilities.go implementing ability system"

2. go-test-writer creates:
   - squads/abilities_test.go with:
     - TestActivateAbility (table-driven)
     - TestAbilityEffects (subtests)
     - TestAbilityTargeting
     - BenchmarkActivateAbility

3. Review generated tests:
   - Coverage looks good
   - Missing domain-specific edge case: "ability with no valid targets"

4. Add custom test case:
   func TestActivateAbilityNoTargets(t *testing.T) {
       manager := ecs.NewManager()
       abilityID := CreateAbility(manager, "Fireball")

       result := ActivateAbility(manager, abilityID, []ecs.EntityID{})

       assert.False(t, result.Success)
       assert.Contains(t, result.Error, "no valid targets")
   }

5. Run tests:
   go test ./squads/... -v

6. All tests pass - commit with confidence
```

---

## Code Review Workflow

> **Purpose:** Pre-commit checklist to ensure code quality

### When to Use

- Before committing changes
- Before creating pull requests
- During peer review
- As final validation step

### Checklist

```
┌──────────────────────────────────────────────────────┐
│               CODE REVIEW CHECKLIST                  │
├──────────────────────────────────────────────────────┤
│                                                      │
│  ECS COMPLIANCE (see docs/ecs_best_practices.md)    │
│  ─────────────────────────────────────────────────   │
│  ☐ No logic in components (pure data only)          │
│  ☐ Uses ecs.EntityID (not entity pointers)          │
│  ☐ Query-based relationships (no caching)           │
│  ☐ Logic in system functions (not components)       │
│  ☐ Value-based map keys (not pointers)              │
│                                                      │
│  GO STANDARDS                                        │
│  ─────────────────────────────────────────────────   │
│  ☐ Proper naming conventions (no stuttering)        │
│  ☐ Error handling (no ignored errors)               │
│  ☐ No allocations in hot paths                      │
│  ☐ go fmt applied                                    │
│  ☐ go vet passes                                     │
│                                                      │
│  ARCHITECTURE                                        │
│  ─────────────────────────────────────────────────   │
│  ☐ Uses CoordManager.LogicalToIndex() for tiles     │
│  ☐ Proper entity lifecycle (cleanup)                │
│  ☐ GUI state separation (UI state vs game state)    │
│  ☐ Generator registration (if applicable)           │
│                                                      │
│  QUALITY GATES                                       │
│  ─────────────────────────────────────────────────   │
│  ☐ go-standards-reviewer run (REQUIRED)             │
│  ☐ ecs-reviewer run if ECS changes (REQUIRED)       │
│  ☐ integration-validator run (REQUIRED)             │
│  ☐ All tests pass (go test ./...)                   │
│  ☐ Test coverage adequate                           │
│                                                      │
│  DOCUMENTATION                                       │
│  ─────────────────────────────────────────────────   │
│  ☐ Public functions documented                      │
│  ☐ Complex logic explained (WHY, not WHAT)          │
│  ☐ TODOs include context                            │
│  ☐ README/docs updated if needed                    │
│                                                      │
└──────────────────────────────────────────────────────┘
```

### Quick Commands

```bash
# Format code
go fmt ./...

# Check for issues
go vet ./...

# Run tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run specific package tests
go test ./squads/... -v
```

### Agent-Driven Review

**Mandatory Quality Gates:**
```
1. "Review [changed files] for Go standards"
2. "Review [changed files] for ECS compliance" (if ECS changes)
3. "Validate integration after [changes]"
```

### Example Scenario

```
Developer: "Ready to commit new combat ability feature"

1. Run code review checklist:

   ECS Compliance:
   ☐ No logic in components
   ☐ Uses ecs.EntityID
   ☐ Query-based relationships
   ☐ Logic in system functions
   ☐ Value-based map keys

2. Run quality gates:

   a) Go standards review:
      "Review squads/abilities.go for Go standards"
      Result: ✅ PASS (compliance level: Excellent)

   b) ECS review:
      "Review squads/abilities.go for ECS compliance"
      Result: ✅ PASS (all 5 ECS principles followed)

   c) Integration validation:
      "Validate integration after ability system changes"
      Result: ⚠️  MEDIUM RISK (need integration tests)

3. Address integration concern:
   - Add integration tests for ability ↔ combat system
   - Re-run integration-validator
   - Result: ✅ PASS (risk now LOW)

4. Run final checks:
   go fmt ./...
   go vet ./...
   go test ./... -v

5. All checks pass - safe to commit

6. Create clear commit message:
   "Add: Squad ability system with combat integration

    Implements ability activation, targeting, and effects.
    Follows ECS patterns from squad system.

    - Pure data components (AbilityData)
    - System-based activation logic
    - Query-based targeting
    - Integration tests for combat system

    ✅ go-standards-reviewer: Excellent
    ✅ ecs-reviewer: Full compliance
    ✅ integration-validator: Low risk
    ✅ All tests passing"
```

---

## Standard Development Cycle

> **Purpose:** Typical day-to-day development workflow

### Overview

```
┌──────────────────────────────────────────────────────┐
│           STANDARD DEVELOPMENT CYCLE                 │
└────────────────┬─────────────────────────────────────┘
                 │
                 ▼
    ┌─────────────────────────┐
    │  1. PLAN                │
    │  - Understand task      │
    │  - Check existing code  │
    │  - Design approach      │
    └────────┬────────────────┘
             │
             ▼
    ┌─────────────────────────┐
    │  2. IMPLEMENT           │
    │  - Write code           │
    │  - Follow ECS patterns  │
    │  - Incremental commits  │
    └────────┬────────────────┘
             │
             ▼
    ┌─────────────────────────┐
    │  3. QUALITY GATES       │
    │  - go-standards-reviewer│
    │  - ecs-reviewer         │
    │  - integration-validator│
    └────────┬────────────────┘
             │
             ▼
    ┌─────────────────────────┐
    │  4. TEST                │
    │  - Write tests          │
    │  - Run test suite       │
    │  - Fix failures         │
    └────────┬────────────────┘
             │
             ▼
    ┌─────────────────────────┐
    │  5. REVIEW              │
    │  - Code review checklist│
    │  - Final validation     │
    └────────┬────────────────┘
             │
             ▼
    ┌─────────────────────────┐
    │  6. COMMIT              │
    │  - Clear message        │
    │  - Reference issues     │
    └─────────────────────────┘
```

### Daily Workflow Example

```
Morning: Feature Development
─────────────────────────────
1. Review task: "Add morale system to combat"
2. Research existing code:
   - Study combat/combatdata.go
   - Check squad system patterns
   - Review status effects implementation

3. Implement incrementally:
   - Create MoraleComponent (pure data)
   - Add morale system functions
   - Integrate with combat calculations
   - Add UI display

4. After each step:
   - Run relevant tests
   - Commit working increment

5. Run quality gates:
   "Review combat/morale.go for Go standards"
   "Review combat/morale.go for ECS compliance"
   "Validate integration after morale system"

6. Fix any issues found

7. Write comprehensive tests:
   "Write tests for combat/morale.go"

8. Final review and commit

Afternoon: Bug Fix
──────────────────
1. Report: "Formation editor crashes on empty squad"
2. Reproduce and locate bug
3. Write failing test
4. Fix bug (minimal changes)
5. Verify test passes
6. Run quality gates
7. Commit fix

End of Day: Code Review
───────────────────────
1. Review all commits from today
2. Run full test suite
3. Check code review checklist
4. Verify all quality gates passed
5. Push to remote (if ready)
```

---

## Common Development Scenarios

### Scenario 1: Adding a Simple Feature

**Task:** Add new ability type to squad system

**Workflow:**
```
1. Research existing abilities pattern

2. Implement:
   - Add ability data to AbilityData component
   - Add activation logic to ability system
   - Update ability queries

3. Run quality gates:
   "Review squads/abilities.go for Go standards"
   "Review squads/abilities.go for ECS compliance"
   "Validate integration after ability changes"

4. Write tests:
   "Write tests for squads/abilities.go"

5. Commit
```

**Time:** 1-2 hours

---

### Scenario 2: Large Refactoring

**Task:** Consolidate duplicate combat calculations

**Workflow:**
```
1. Optional: Analyze architecture
   "Analyze combat package architecture"

2. Create refactoring plan:
   - Document in analysis/combat_refactor_plan.md
   - Define sections and dependencies

3. Use refactoring-implementer:
   "Implement refactoring from combat_refactor_plan.md"

4. Agent executes systematically:
   - Section by section
   - Checkpoints after each
   - Asks for confirmation

5. After completion, run quality gates:
   "Review combat/ for Go standards"
   "Review combat/ for ECS compliance"
   "Validate integration after combat refactor"

6. Update tests:
   "Write tests for refactored combat files"

7. Verify all tests pass

8. Done
```

**Time:** 4-8 hours (depending on scale)

---

### Scenario 3: Complex New System

**Task:** Implement tactical cover system

**Workflow:**
```
1. Use feature-implementer:
   "Add tactical cover system - units can take cover behind obstacles for defense bonus"

2. Agent proposes plan:
   - Milestone 1: CoverComponent
   - Milestone 2: Cover detection
   - Milestone 3: Combat integration
   - Milestone 4: UI visualization

3. Review and approve plan

4. Agent implements systematically:
   - Milestone by milestone
   - Checkpoints and confirmations
   - Follows ECS patterns

5. After completion, run quality gates:
   "Review combat/cover.go for Go standards"
   "Review combat/cover.go for ECS compliance"
   "Validate integration after cover system"

6. Write comprehensive tests:
   "Write tests for combat/cover.go"
   "Write integration tests for cover system"

7. Done
```

**Time:** 1-3 days (depending on complexity)

---

### Scenario 4: Critical Bug Fix

**Task:** Game crashes when squad is disbanded during combat

**Workflow:**
```
1. Reproduce crash:
   - Create squad
   - Enter combat
   - Disband squad
   - CRASH: nil pointer dereference

2. Locate bug:
   - combat/turnmanager.go line 234
   - Tries to access disbanded squad

3. Write failing test:
   func TestDisbandSquadDuringCombat(t *testing.T) {
       // Test that exposes crash
   }

4. Implement fix:
   - Add nil check before accessing squad
   - Properly handle disbanded squad case

5. Test passes

6. Run quality gates:
   "Review combat/turnmanager.go for Go standards"
   "Validate integration after turnmanager fix"

7. Commit:
   "Fix: Handle disbanded squad in combat turn manager

    Previously crashed with nil pointer when squad disbanded during combat.
    Now properly checks if squad exists before processing turn.

    Test: TestDisbandSquadDuringCombat
    Fixes #456"

8. Done
```

**Time:** 1-2 hours

---

### Scenario 5: Performance Optimization

**Task:** Reduce allocations in render loop

**Workflow:**
```
1. Profile code:
   go test -bench=. -benchmem ./rendering/...

2. Identify issue:
   - RenderFrame allocates []Position every frame
   - 3600 allocations/minute at 60fps

3. Fix:
   type Renderer struct {
       posBuffer []Position  // Reuse buffer
   }

   func (r *Renderer) RenderFrame() {
       r.posBuffer = r.posBuffer[:0]  // Reset, no allocation
       // ... use buffer
   }

4. Run quality gates:
   "Review rendering/renderer.go for Go standards"
   - Agent should flag if optimization introduces issues
   - Verify no allocations in hot path

5. Benchmark improvement:
   go test -bench=. -benchmem ./rendering/...
   - Verify allocations eliminated
   - Verify performance improved

6. Commit:
   "Perf: Eliminate allocations in render loop

    Previously allocated position buffer every frame (3600/min).
    Now reuses buffer, eliminating 3600 allocations/minute.

    Benchmark: BenchmarkRenderFrame
    Before: 3600 allocs/min
    After: 0 allocs/min"

7. Done
```

**Time:** 2-4 hours

---

## Best Practices

### General Development

**1. Small, Incremental Changes:**
```
✅ GOOD - Focused commits
- "Add: CoverComponent for tactical combat"
- "Add: Cover detection system functions"
- "Add: Cover integration with combat calculations"

❌ BAD - Huge commits
- "Add entire cover system with UI and tests"
```

**2. Test-Driven Development:**
```
✅ GOOD - Write test first
1. Write failing test
2. Implement feature
3. Test passes
4. Refactor if needed

❌ BAD - Test as afterthought
1. Implement feature
2. Maybe write test later
3. Test might not catch issues
```

**3. Use Reference Implementations:**
```
✅ GOOD - Follow existing patterns
- Study squads/ for ECS patterns
- Study gear/Inventory.go for pure components
- Study systems/positionsystem.go for value keys

❌ BAD - Reinvent patterns
- Create new component style
- Ignore existing patterns
```

### Agent Usage

**1. Use Agents Systematically:**
```
✅ GOOD - Follow workflow
- Run quality gates after changes
- Use feature-implementer for complex features
- Let agents create checkpoints

❌ BAD - Skip quality gates
- "I'll just commit without review"
- "Don't need integration validation"
```

**2. Review Agent Output:**
```
✅ GOOD - Critical review
- Read analysis thoroughly
- Question recommendations
- Adjust for context

❌ BAD - Blind acceptance
- "Agent said so, must be right"
- Skip reading analysis
```

**3. Provide Clear Context:**
```
✅ GOOD - Specific requests
- "Review combat/attack.go for Go standards"
- "Validate integration after squad refactor"

❌ BAD - Vague requests
- "Check my code"
- "Is this okay?"
```

### Code Quality

**1. Follow ECS Principles:**
```
✅ GOOD - See docs/ecs_best_practices.md
- Pure data components
- Native EntityID
- Query-based relationships
- System functions
- Value map keys

❌ BAD - Violate ECS
- Logic in components
- Entity pointers
- Cached references
```

**2. Performance Awareness:**
```
✅ GOOD - Game development focus
- No allocations in hot paths
- Value-based map keys
- Pre-allocate when size known
- Benchmark critical code

❌ BAD - Ignore performance
- Allocate in render loop
- Pointer map keys
- String concatenation in loops
```

**3. Clear Documentation:**
```
✅ GOOD - Explain WHY
// Use CoordinateManager to prevent index out of bounds
tileIdx := coords.CoordManager.LogicalToIndex(pos)

// TODO: Add formation validation (30min)

❌ BAD - State WHAT
// Get tile index
tileIdx := y*width + x

// TODO: Fix this
```

---

## Common Pitfalls

### 1. Skipping Quality Gates

**Problem:**
```
Developer commits code without running quality gates
→ Issues discovered later
→ More expensive to fix
→ Breaks other systems
```

**Solution:**
```
ALWAYS run after ANY code changes:
1. go-standards-reviewer
2. ecs-reviewer (if ECS changes)
3. integration-validator
```

---

### 2. Large, Untested Commits

**Problem:**
```
Developer implements entire feature at once
→ Hundreds of lines changed
→ Hard to review
→ Tests fail, unclear why
→ Difficult to rollback
```

**Solution:**
```
Work incrementally:
1. Implement one piece
2. Test it
3. Commit it
4. Move to next piece

Use feature-implementer for systematic execution
```

---

### 3. Ignoring Agent Recommendations

**Problem:**
```
go-standards-reviewer flags allocation in hot path
Developer: "I'll fix it later"
→ Performance degrades
→ Harder to fix later
→ Technical debt accumulates
```

**Solution:**
```
Address CRITICAL and HIGH priority issues immediately
Document why if postponing MEDIUM/LOW issues
```

---

### 4. Manual Index Calculation

**Problem:**
```go
// ❌ WRONG - Manual calculation
idx := y*width + x
result.Tiles[idx] = &tile  // CRASHES if width != CoordManager.dungeonWidth
```

**Solution:**
```go
// ✅ CORRECT - Use CoordinateManager
tileIdx := coords.CoordManager.LogicalToIndex(logicalPos)
result.Tiles[tileIdx] = &tile
```

---

### 5. Violating ECS Principles

**Problem:**
```go
// ❌ Component with method
type Squad struct {
    Members []ecs.EntityID
}
func (s *Squad) AddMember(id ecs.EntityID) { ... }
```

**Solution:**
```go
// ✅ System function
func AddSquadMember(manager *ecs.Manager, squadID, memberID ecs.EntityID) {
    // Logic here
}
```

**Prevention:**
Run ecs-reviewer after ANY ECS changes

---

### 6. Breaking Integration

**Problem:**
```
Refactor Position component, remove field
→ 15 systems break
→ Didn't realize dependency scope
→ Hours of fixing
```

**Solution:**
```
ALWAYS run integration-validator before committing changes to:
- Shared components
- Core systems
- Function signatures
```

---

### 7. Premature Optimization

**Problem:**
```
Developer optimizes cold path
→ Complicates code
→ No measurable benefit
→ Creates maintenance burden
```

**Solution:**
```
1. Profile first
2. Optimize hot paths only
3. Benchmark to verify improvement
4. Run go-standards-reviewer to check for issues
```

---

## Related Documentation

### Core Documentation
- **CLAUDE.md** - Quick reference, ECS patterns, critical warnings
- **docs/ecs_best_practices.md** - Comprehensive ECS architecture guide
- **docs/gui_documentation/GUI_PATTERNS.md** - GUI development patterns
- **docs/DOCUMENTATION.md** - Overview of all documentation

### Agent Specifications
- **.claude/agents/codebase-analyzer.md** - Architecture analysis agent
- **.claude/agents/go-standards-reviewer.md** - Go standards agent
- **.claude/agents/ecs-reviewer.md** - ECS compliance agent
- **.claude/agents/integration-validator.md** - Integration safety agent
- **.claude/agents/feature-implementer.md** - Feature implementation agent
- **.claude/agents/refactoring-implementer.md** - Refactoring execution agent
- **.claude/agents/go-test-writer.md** - Test generation agent

### Reference Implementations
- **squads/** - Perfect ECS example (2358 LOC, 8 pure components)
- **gear/Inventory.go** - Pure ECS component (241 LOC)
- **systems/positionsystem.go** - Value-based map keys

### Testing
- Run tests: `go test ./...`
- With coverage: `go test -cover ./...`
- Specific package: `go test ./squads/... -v`
- Benchmarks: `go test -bench=. -benchmem ./rendering/...`

---

## Summary

### Key Takeaways

**1. Agent-Driven Development:**
- Use agents for systematic analysis and execution
- Always run mandatory quality gates
- Review agent output critically
- Document decisions

**2. Quality First:**
- Run quality gates after ANY changes
- Fix critical issues immediately
- Test continuously
- Commit incrementally

**3. Follow Patterns:**
- ECS best practices (5 core principles)
- Go standards (idiomatic Go)
- Reference implementations (squad, inventory)
- Existing codebase patterns

**4. Systematic Approach:**
- Plan before implementing
- Work incrementally
- Validate continuously
- Document thoroughly

### Quick Command Reference

```bash
# Build and run
go build -o game_main/game_main.exe game_main/*.go && ./game_main/game_main.exe

# Test
go test ./...                    # All tests
go test ./squads/... -v         # Specific package
go test -cover ./...            # With coverage

# Quality
go fmt ./...                    # Format
go vet ./...                    # Check issues

# Agents (invocation examples)
"Review [files] for Go standards"
"Review [files] for ECS compliance"
"Validate integration after [changes]"
"Implement feature from [plan file]"
```

### Development Mantra

```
┌────────────────────────────────────────────────────┐
│                                                    │
│  1. Plan systematically                            │
│  2. Implement incrementally                        │
│  3. Validate continuously (QUALITY GATES!)         │
│  4. Test comprehensively                           │
│  5. Review thoroughly                              │
│  6. Commit clearly                                 │
│                                                    │
│  Agents assist, humans decide, quality gates      │
│  ensure excellence.                                │
│                                                    │
└────────────────────────────────────────────────────┘
```

---

**End of Development Workflows Guide**

*For questions or issues with workflows, refer to this document or consult related documentation.*
