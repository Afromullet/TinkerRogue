# Development Workflows Guide

> **Purpose:** Agent-assisted workflows for systematic code improvement and feature implementation

---

## Table of Contents

### Quick Navigation
- [Quick Start](#quick-start) - Start here if you need immediate guidance
- [Decision Tree](#decision-tree) - Visual guide for choosing workflows
- [Agent Directory](#agent-directory) - Which agent to use and when

### Workflows
- [Refactoring Workflow](#refactoring-workflow) - Improve existing code structure
- [Implementation Workflow](#implementation-workflow) - Add new features and capabilities
- [Combining Workflows](#combining-workflows) - Using both workflows together

### Reference
- [Best Practices](#best-practices) - Core principles and guidelines
- [Success Metrics](#success-metrics) - How to measure outcomes
- [Getting Started Checklist](#getting-started-checklist) - Step-by-step for your first workflow

---

## Quick Start

### What Do You Need To Do?

| Goal | Workflow | Starting Agent |
|------|----------|----------------|
| Fix duplicate code | [Refactoring](#refactoring-workflow) | `refactoring-synth` |
| Simplify complex system | [Refactoring](#refactoring-workflow) | `refactoring-synth` |
| Add new feature | [Implementation](#implementation-workflow) | `implementation-synth` |
| Build new mechanic | [Implementation](#implementation-workflow) | `implementation-synth` |
| Pay down technical debt | [Refactoring](#refactoring-workflow) | `refactoring-synth` |
| Extend existing system | [Implementation](#implementation-workflow) | `implementation-synth` |

### The Universal Pattern

All workflows follow the same structure:

```
┌─────────────────┐      ┌─────────────────┐      ┌─────────────────┐
│  1. ANALYSIS    │ ---> │  2. DECISION    │ ---> │  3. EXECUTE     │
│  (Agent-driven) │      │  (Human-driven) │      │ (Collaborative) │
└─────────────────┘      └─────────────────┘      └─────────────────┘
```

**Key Principle:** Agents provide analysis and options. Humans make decisions. Implementation is collaborative.

---

## Decision Tree

```
                    ┌──────────────────────┐
                    │ What do you need?    │
                    └──────────┬───────────┘
                               │
               ┌───────────────┴────────────────┐
               │                                │
               ▼                                ▼
    ┌──────────────────────┐       ┌──────────────────────┐
    │ Improve existing code│       │ Add new functionality│
    │   (No new features)  │       │  (New capabilities)  │
    └──────────┬───────────┘       └──────────┬───────────┘
               │                               │
               ▼                               ▼
    ┌──────────────────────┐       ┌──────────────────────┐
    │  REFACTORING         │       │  IMPLEMENTATION      │
    │                      │       │                      │
    │ • Eliminate duplicate│       │ • Build features     │
    │ • Simplify code      │       │ • Add mechanics      │
    │ • Consolidate systems│       │ • Extend systems     │
    │ • Restructure        │       │ • Create components  │
    └──────────────────────┘       └──────────────────────┘
               │                               │
               └───────────────┬───────────────┘
                               │
                               ▼
                  ┌────────────────────────┐
                  │   Sometimes need both: │
                  │ Refactor first, then   │
                  │ implement new feature  │
                  └────────────────────────┘
```

### Quick Decision Matrix

| Situation | Workflow | Why |
|-----------|----------|-----|
| Code works but hard to maintain | Refactoring | Structure improvement needed |
| Duplicate code everywhere | Refactoring | Consolidation opportunity |
| Can't add features easily | Refactoring | Architecture blocking progress |
| System too complex | Refactoring | Simplification required |
| Need new game mechanic | Implementation | New functionality required |
| Todo list item to build | Implementation | Feature development |
| Extend existing system | Implementation | New capabilities needed |
| Feature needs code cleanup first | Both (Refactor → Implement) | Prerequisites exist |
| Implementation reveals tech debt | Both (Implement → Refactor) | Discovered during work |

---

## Refactoring Workflow

> **Purpose:** Improve existing code structure without changing functionality

```
REFACTORING WORKFLOW
═══════════════════════════════════════════════════════════════════════

Step 1          Step 2          Step 3          Step 4          Step 5
┌──────────┐   ┌──────────┐   ┌──────────┐   ┌──────────┐   ┌──────────┐
│ ANALYSIS │-> │  REVIEW  │-> │  PLAN    │-> │IMPLEMENT │-> │  TESTS   │
│  (Agent) │   │  (Human) │   │ (Agent)  │   │ (Collab) │   │ (Agent)  │
└──────────┘   └──────────┘   └──────────┘   └──────────┘   └──────────┘
```

### At a Glance

| Aspect | Details |
|--------|---------|
| **Purpose** | Improve code structure without changing functionality |
| **Primary Agents** | `refactoring-synth` (analysis) + `refactoring-implementer` (planning) |
| **Duration** | Analysis: 5-15 min, Implementation: hours to days |
| **Risk Level** | Medium (tests must pass, behavior unchanged) |
| **Output** | Cleaner, more maintainable code with same functionality |

### When to Use This Workflow

**Use refactoring when you see:**

- **Duplication:** Multiple functions or files share duplicate code patterns
- **Complexity:** A system has grown complex and needs architectural simplification
- **Scattered Logic:** Related functionality is spread across multiple locations
- **Maintenance Pain:** Code structure makes it difficult to add new features
- **Technical Debt:** You need to consolidate or unify similar components

**Don't use refactoring for:**
- Adding new features (use Implementation Workflow)
- Fixing bugs (just fix them directly)
- Changing behavior or adding capabilities

---

### Step 1: Generate Refactoring Analysis

**Agent:** `refactoring-synth` | **Duration:** 5-15 minutes

#### What Happens

The agent analyzes your code from multiple perspectives:

- **Code Structure** - Patterns, organization, modularity
- **Duplication Analysis** - Repeated code, consolidation opportunities
- **Architecture Review** - Design issues, coupling, cohesion
- **Dependencies** - Relationships, impact radius
- **Risk Assessment** - What could break, migration strategies

#### Your Actions

1. **Identify** the files or systems that need refactoring
2. **Invoke** the `refactoring-synth` agent with target files
3. **Wait** for comprehensive analysis

#### Output You Receive

| Section | Content |
|---------|---------|
| **Current State** | What the code looks like now, identified problems |
| **Pain Points** | Specific issues making maintenance difficult |
| **Approaches** | 2-4 different refactoring strategies with pros/cons |
| **Trade-offs** | Complexity vs. benefit, risk vs. reward analysis |
| **Risk Assessment** | What could go wrong, mitigation strategies |
| **Migration Path** | How to get from current state to desired state |
| **Recommendations** | Agent's suggested approach with rationale |

---

### Step 2: Review and Decision

**Actor:** Developer (you) | **Duration:** Your choice

> **Critical Human Decision Point:** Agents provide options, you choose the path forward.

#### Decision Framework

Ask yourself these key questions:

| Question | Why It Matters |
|----------|----------------|
| **Does the refactoring justify the effort?** | ROI analysis: time investment vs. maintenance gains |
| **What risks are acceptable?** | Some approaches are safer but less impactful |
| **How does this fit with other work?** | Avoid conflicts with parallel development |
| **Are there dependencies first?** | Some refactoring requires other changes first |
| **Which approach aligns with project goals?** | Choose based on long-term architecture vision |
| **What's the implementation timeline?** | Consider deadlines and resource availability |

#### Document Your Decision

- **Chosen Approach** - Which refactoring strategy you selected
- **Rationale** - Why this approach over alternatives
- **Adjustments** - Any modifications to the proposed plan
- **Priority** - When this fits in your development schedule

---

### Step 3: Create Implementation Plan

**Agent:** `refactoring-implementer` | **Duration:** 5-10 minutes

#### What Happens

The agent breaks down your chosen approach into a step-by-step plan with:

- **Small Steps** - Each change is minimal and focused
- **Testable** - Every step can be verified before proceeding
- **Reversible** - Clear rollback strategy if something goes wrong
- **Ordered** - Dependencies managed, safe progression guaranteed
- **Specific** - Exact files and changes for each step

#### Your Actions

1. **Provide** your chosen approach from Step 2
2. **Include** any adjustments or constraints
3. **Wait** for detailed implementation plan

#### Output You Receive

| Component | Details |
|-----------|---------|
| **Step Sequence** | Numbered, ordered list of changes |
| **File Targets** | Specific files to modify at each step |
| **Code Changes** | What to change, where, and how |
| **Test Checkpoints** | How to verify each step worked correctly |
| **Rollback Strategy** | What to do if a step causes problems |
| **Expected Outcomes** | What success looks like at each stage |

> **Best Practice:** Treat each step as a commit point. Don't combine steps.

---

### Step 4: Implementation

**Actor:** Developer or Agent-Assisted | **Duration:** Varies

#### Execution Workflow

```
FOR EACH STEP IN PLAN:
  ┌──────────────────────────────────────┐
  │ 1. READ step instructions            │
  │ 2. IMPLEMENT changes in files        │
  │ 3. TEST verify functionality         │
  │ 4. COMMIT incremental progress       │
  └──────────────────────────────────────┘
       │
       ├─> ✅ Tests Pass? → Continue to next step
       │
       └─> ❌ Tests Fail? → Rollback, debug, retry
```

#### Best Practices

| Practice | Why |
|----------|-----|
| **Follow plan order** | Steps are sequenced to minimize risk |
| **Test after each step** | Catch problems early, narrow debugging scope |
| **Commit incrementally** | Create safe rollback points |
| **Don't skip steps** | Each step prepares for the next |
| **Don't combine steps** | Smaller changes = easier debugging |
| **Refer to docs** | If unclear, check original analysis |

#### When Things Go Wrong

```
PROBLEM DETECTED
      │
      ├─> Minor issue? → Debug and fix, continue
      │
      ├─> Step doesn't work? → Rollback to previous commit
      │                         Review docs, adjust approach
      │
      └─> Fundamental problem? → Return to Step 2
                                  Choose different approach
```

> **Critical Rule:** All tests must pass before marking refactoring complete.

---

### Step 5: Add/Update Tests

**Agent:** `go-test-writer` | **Duration:** 5-15 minutes

#### Why This Step Matters

After refactoring, your test suite may need updates:
- New test cases for consolidated functionality
- Modified tests that reflect new structure
- Additional coverage for edge cases revealed during refactoring
- Benchmarks to verify performance improvements

#### Your Actions

1. **Identify** files that were refactored
2. **Invoke** `go-test-writer` with refactored file paths
3. **Review** generated/updated tests
4. **Run** tests to verify coverage

#### Output You Receive

| Component | Details |
|-----------|---------|
| **Test Functions** | `TestFunctionName` with table-driven patterns |
| **Subtests** | `t.Run()` blocks for each scenario |
| **Test Helpers** | Setup/teardown functions for common operations |
| **Benchmarks** | `BenchmarkFunctionName` for performance testing |
| **Edge Cases** | Nil checks, boundary values, error conditions |

#### Example Invocation

```
"Write tests for graphics/drawableshapes.go after refactoring"
"Update tests for input/inputcoordinator.go to reflect new structure"
```

> **Pro Tip:** Use go-test-writer proactively during refactoring, not just at the end.

---

## Implementation Workflow

> **Purpose:** Add new features, mechanics, and capabilities

```
IMPLEMENTATION WORKFLOW
═══════════════════════════════════════════════════════════════════════

Step 1          Step 2          Step 3          Step 4
┌──────────┐   ┌──────────┐   ┌──────────┐   ┌──────────┐
│ ANALYSIS │-> │  REVIEW  │-> │IMPLEMENT │-> │  TESTS   │
│  (Agent) │   │  (Human) │   │ (Collab) │   │ (Agent)  │
└──────────┘   └──────────┘   └──────────┘   └──────────┘
```

### At a Glance

| Aspect | Details |
|--------|---------|
| **Purpose** | Add new features, mechanics, and capabilities |
| **Primary Agent** | `implementation-synth` (analysis) |
| **Duration** | Analysis: 10-30 min, Implementation: hours to weeks |
| **Risk Level** | Medium-High (new code, integration challenges) |
| **Output** | New functionality integrated with existing systems |

### When to Use This Workflow

**Use implementation when you need to:**

- **New Mechanics:** Add a new game mechanic or system
- **Todo Items:** Implement a feature from the todo list
- **New Components:** Build a new component or capability
- **Extensions:** Extend existing systems with new behavior
- **Gameplay Elements:** Create new gameplay features

**Don't use implementation for:**
- Improving existing code structure (use Refactoring Workflow)
- Fixing bugs (just fix them directly)
- Simplifying or consolidating code

---

### Step 1: Generate Implementation Analysis

**Agent:** `implementation-synth` | **Duration:** 10-30 minutes

#### What Happens

The agent analyzes your feature from multiple perspectives:

- **Technical Requirements** - What needs to be built, constraints, dependencies
- **Integration Points** - How this connects to existing systems
- **Data Model** - Structures, state management, persistence needs
- **UI/UX Considerations** - Player-facing aspects, user experience
- **Testing Strategy** - How to validate the feature works correctly
- **Architecture Design** - Overall structure and component organization

#### Your Actions

1. **Describe** the feature or capability you want to implement
2. **Provide** requirements, constraints, and context
3. **Invoke** the `implementation-synth` agent
4. **Wait** for comprehensive analysis

#### Output You Receive

| Section | Content |
|---------|---------|
| **Requirements & Scope** | What the feature does, boundaries, success criteria |
| **System Architecture** | High-level design, components, data flow |
| **Approaches** | 2-4 different implementation strategies with pros/cons |
| **Trade-offs** | Complexity vs. extensibility, performance vs. simplicity |
| **Integration Strategy** | How this fits with existing code, what needs modification |
| **Data Structures** | Models, state management, storage considerations |
| **Testing Plan** | How to validate functionality, edge cases to consider |
| **Challenges & Risks** | Potential problems and mitigation strategies |
| **Timeline Estimate** | Rough effort estimation for each approach |

---

### Step 2: Review and Decision

**Actor:** Developer (you) | **Duration:** Your choice

> **Critical Human Decision Point:** Choose the implementation approach that fits your vision.

#### Decision Framework

Ask yourself these key questions:

| Question | Why It Matters |
|----------|----------------|
| **Does it align with existing patterns?** | Consistency matters for maintainability |
| **Is the testing strategy adequate?** | Need confidence the feature works correctly |
| **How does it interact with other features?** | Avoid conflicts and integration issues |
| **Refactoring prerequisites?** | Sometimes existing code needs improvement first |
| **What's the timeline and effort?** | Balance scope with available time |
| **How extensible is this approach?** | Consider future enhancements and variations |
| **What's the performance impact?** | Ensure acceptable game performance |

#### Document Your Decision

- **Chosen Approach** - Which implementation strategy you selected
- **Rationale** - Why this approach fits best
- **Customizations** - Adjustments to proposed design
- **Prerequisites** - Any refactoring or prep work needed first
- **Success Criteria** - How you'll know the feature works correctly

---

### Step 3: Implementation

**Actor:** Developer or Agent-Assisted | **Duration:** Varies

#### Recommended Implementation Order

```
IMPLEMENTATION PROGRESSION
┌──────────────────────────────────────────────────────────┐
│                                                          │
│  1. DATA STRUCTURES    Define models, state, interfaces │
│                        Foundation for everything else    │
│                                                          │
│  2. CORE LOGIC         Implement business rules         │
│                        No UI, pure functionality         │
│                                                          │
│  3. INTEGRATION        Connect to existing systems      │
│                        Gradual, tested integration       │
│                                                          │
│  4. UI/UX              Add player-facing elements       │
│                        Once core logic is stable         │
│                                                          │
│  5. POLISH             Edge cases, error handling       │
│                        Refinement and optimization       │
│                                                          │
└──────────────────────────────────────────────────────────┘
```

#### Best Practices

| Practice | Why | Example |
|----------|-----|---------|
| **Start with data** | Foundation affects everything else | Define structs before methods |
| **Build incrementally** | Catch problems early | Implement one feature aspect at a time |
| **Test frequently** | Fast feedback loop | Test after each logical component |
| **Integrate gradually** | Minimize breaking changes | Connect one system at a time |
| **Write tests early** | Design driver, catches regressions | Create tests as you build, not after |
| **Commit regularly** | Safe rollback points | Commit each working increment |
| **Refer to docs** | Stay aligned with plan | Check implementation analysis when stuck |

#### Development Cycle

```
FOR EACH FEATURE COMPONENT:
  ┌──────────────────────────────────────┐
  │ 1. DESIGN   Data structures/interfaces│
  │ 2. IMPLEMENT Core logic for component│
  │ 3. TEST      Verify functionality    │
  │ 4. INTEGRATE Connect to existing code│
  │ 5. VALIDATE  Test integration points │
  │ 6. COMMIT    Save working progress   │
  └──────────────────────────────────────┘
       │
       ├─> ✅ Works correctly? → Next component
       │
       └─> ❌ Problems? → Debug, refine, retry
```

#### When Implementation Reveals Issues

```
DISCOVERED ISSUE
      │
      ├─> Design problem? → Return to Step 2, adjust approach
      │
      ├─> Integration challenge? → May need refactoring first
      │                            Switch to Refactoring Workflow
      │
      └─> Implementation detail? → Debug and solve, continue
```

> **Pro Tip:** If implementation feels harder than expected, existing code may need refactoring first.

---

### Step 4: Add Tests

**Agent:** `go-test-writer` | **Duration:** 5-20 minutes

#### Why This Step Matters

New features require comprehensive test coverage:
- **Functional tests** verify feature works as designed
- **Integration tests** ensure compatibility with existing systems
- **Edge case tests** catch boundary conditions and errors
- **Regression prevention** for future modifications

#### Your Actions

1. **Identify** files implementing the new feature
2. **Invoke** `go-test-writer` with feature file paths
3. **Review** generated tests against requirements
4. **Add** domain-specific test scenarios
5. **Run** tests to verify feature correctness

#### Output You Receive

| Component | Details |
|-----------|---------|
| **Feature Tests** | Validate core feature functionality |
| **Integration Tests** | Verify feature works with existing code |
| **Edge Case Tests** | Boundary conditions, nil checks, limits |
| **Benchmarks** | Performance baseline and regression detection |
| **Test Data** | Fixtures and helpers for test scenarios |

#### Example Invocation

```
"Write tests for squads/squadcombat.go implementing combat system"
"Create tests for the new ability system in squads/abilities.go"
```

> **Pro Tip:** Use go-test-writer multiple times during implementation - once per component.

> **Critical:** Don't consider a feature complete until it has comprehensive test coverage.

---

## Combining Workflows

Sometimes you need both workflows. Here's how they interact:

### Scenario 1: Refactor-First Pattern

You want to add a feature but existing code is blocking you.

```
┌──────────────────┐
│  Feature Idea    │
└────────┬─────────┘
         │
         ▼
┌──────────────────┐
│ implementation-  │  Understand requirements
│     synth        │
└────────┬─────────┘
         │
         ▼
┌──────────────────┐
│  DISCOVER:       │  Existing code needs work!
│  Code needs      │
│  refactoring     │
└────────┬─────────┘
         │
         ▼
┌──────────────────┐
│ refactoring-     │  Analyze existing code
│     synth        │
└────────┬─────────┘
         │
         ▼
┌──────────────────┐
│ Refactor code    │  Clean up existing systems
│ (Steps 2-4)      │
└────────┬─────────┘
         │
         ▼
┌──────────────────┐
│ Implement feature│  Now add new functionality
│ (Step 3)         │
└──────────────────┘
```

### Scenario 2: Implementation-First Pattern

You start implementing and discover technical debt.

```
┌──────────────────┐
│ Build feature    │
│ using            │
│implementation-   │
│     synth        │
└────────┬─────────┘
         │
         ▼
┌──────────────────┐
│ Start coding...  │
         │
         ▼
┌──────────────────┐
│  DISCOVER:       │  This code is messy!
│  Tech debt       │
│  blocking        │
└────────┬─────────┘
         │
         ▼
┌──────────────────┐
│ Pause feature    │  Save WIP, commit if possible
│ development      │
└────────┬─────────┘
         │
         ▼
┌──────────────────┐
│ Run refactoring  │  Clean up the mess
│ workflow         │
└────────┬─────────┘
         │
         ▼
┌──────────────────┐
│ Resume feature   │  Continue with cleaner code
│ implementation   │
└──────────────────┘
```

### Quick Decision Guide

```
What's your primary goal?

┌──────────────────────────────────┐
│ Make existing code better?       │ → Refactoring Workflow
├──────────────────────────────────┤
│ Add something new to the game?   │ → Implementation Workflow
├──────────────────────────────────┤
│ Add feature BUT code is messy?   │ → Refactor First, Then Implement
└──────────────────────────────────┘
```

---

## Agent Directory

### Quick Reference

```
┌──────────────────────────────────────────────────────────────┐
│                       AGENT DIRECTORY                        │
├──────────────────────────────────────────────────────────────┤
│                                                              │
│  🔧 REFACTORING AGENTS                                       │
│  ──────────────────────────────────────────────────────────  │
│                                                              │
│  refactoring-synth                                           │
│  └─> Analyzes code, generates refactoring documentation      │
│  └─> Input: Files/systems needing improvement               │
│  └─> Output: Multi-approach analysis with trade-offs        │
│  └─> When: Start of refactoring workflow                    │
│                                                              │
│  refactoring-implementer                                     │
│  └─> Creates step-by-step refactoring plans                 │
│  └─> Input: Chosen refactoring approach                     │
│  └─> Output: Ordered steps with testing checkpoints         │
│  └─> When: After you've decided on approach                 │
│                                                              │
│  ──────────────────────────────────────────────────────────  │
│                                                              │
│  ✨ IMPLEMENTATION AGENTS                                    │
│  ──────────────────────────────────────────────────────────  │
│                                                              │
│  implementation-synth                                        │
│  └─> Analyzes features, generates implementation docs       │
│  └─> Input: Feature requirements and descriptions           │
│  └─> Output: Multi-approach design with integration plan    │
│  └─> When: Start of implementation workflow                 │
│                                                              │
│  ──────────────────────────────────────────────────────────  │
│                                                              │
│  🧪 TESTING AGENT                                            │
│  ──────────────────────────────────────────────────────────  │
│                                                              │
│  go-test-writer                                              │
│  └─> Generates comprehensive, idiomatic Go test suites      │
│  └─> Input: Files to test (source or feature files)         │
│  └─> Output: Table-driven tests, benchmarks, helpers        │
│  └─> When: After refactoring or implementing features       │
│                                                              │
└──────────────────────────────────────────────────────────────┘
```

### Detailed Agent Profiles

<details>
<summary><strong>refactoring-synth</strong> - Refactoring Analysis Coordinator</summary>

**Role:** Comprehensive code analysis and refactoring strategy generator

**What it does:**
- Coordinates multiple specialized analysis agents
- Examines code structure, duplication, architecture
- Identifies problems and improvement opportunities
- Generates multiple refactoring approaches with trade-offs

**Input you provide:**
- Files or systems that need improvement
- Context about pain points and goals

**Output you receive:**
- Current state analysis
- Identified problems
- 2-4 refactoring approaches with pros/cons
- Risk assessment and migration strategies
- Implementation recommendations

**When to use:**
- Step 1 of Refactoring Workflow
- Before making structural changes to code
- When you need comprehensive analysis of technical debt

</details>

<details>
<summary><strong>refactoring-implementer</strong> - Refactoring Planning Specialist</summary>

**Role:** Step-by-step refactoring plan generator

**What it does:**
- Breaks down refactoring into small, testable steps
- Creates ordered sequence of changes
- Includes verification and rollback strategies
- Ensures safe, incremental progress

**Input you provide:**
- Your chosen refactoring approach from refactoring-synth
- Any adjustments or constraints

**Output you receive:**
- Numbered sequence of implementation steps
- Specific files to modify at each step
- Testing checkpoints
- Rollback strategies
- Expected outcomes

**When to use:**
- Step 3 of Refactoring Workflow
- After you've decided on a refactoring approach
- Before starting actual code changes

</details>

<details>
<summary><strong>implementation-synth</strong> - Feature Implementation Analyzer</summary>

**Role:** Feature design and implementation strategy generator

**What it does:**
- Coordinates multiple specialized planning agents
- Analyzes technical requirements and constraints
- Designs architecture and integration strategy
- Generates multiple implementation approaches

**Input you provide:**
- Feature description and requirements
- Constraints and context

**Output you receive:**
- Requirements and scope definition
- System architecture and design
- 2-4 implementation approaches with trade-offs
- Integration strategy with existing code
- Data structures and testing plan
- Timeline estimates

**When to use:**
- Step 1 of Implementation Workflow
- Before building new features
- When you need comprehensive feature design

</details>

<details>
<summary><strong>go-test-writer</strong> - Test Generation Specialist</summary>

**Role:** Comprehensive Go test suite generator

**What it does:**
- Analyzes source files and generates idiomatic Go tests
- Creates table-driven tests with comprehensive scenarios
- Generates test helpers, benchmarks, and edge case coverage
- Follows Go testing best practices (TestMain, subtests, assertions)

**Input you provide:**
- Files that need test coverage (source or feature files)
- Context about what functionality to test

**Output you receive:**
- Complete `*_test.go` files with:
  - Table-driven test functions
  - Subtests for each scenario
  - Test helper functions
  - Benchmarks for performance testing
  - Edge case and error path coverage

**When to use:**
- Step 5 of Refactoring Workflow (after implementation)
- Step 4 of Implementation Workflow (after building feature)
- Anytime you need test coverage for existing or new code
- During implementation (alongside coding, not just at end)

**Example commands:**
```
"Write tests for combat/attack.go"
"Create tests for the inventory system"
"Generate tests for squads/squadcombat.go"
```

</details>

---

## Best Practices

### Core Principles

```
┌──────────────────────────────────────────────────────────────┐
│                    WORKFLOW PRINCIPLES                       │
├──────────────────────────────────────────────────────────────┤
│                                                              │
│  👁️  AGENTS ADVISE, HUMANS DECIDE                           │
│      Use agent analysis to inform decisions, not make them  │
│                                                              │
│  🔍  REVIEW THOROUGHLY                                       │
│      Don't skip the review step - it's where value is added │
│                                                              │
│  🧩  INCREMENTAL PROGRESS                                    │
│      Small, tested steps beat big bang changes              │
│                                                              │
│  📚  PRESERVE CONTEXT                                        │
│      Document decisions and link them to implementation     │
│                                                              │
│  🎯  VALIDATE CONTINUOUSLY                                   │
│      Test after each change, commit working increments      │
│                                                              │
└──────────────────────────────────────────────────────────────┘
```

### Documentation Review

| Practice | Why It Matters |
|----------|----------------|
| **Don't skip review** | Agent analysis provides options, you make the choice |
| **Compare approaches** | Multiple perspectives reveal trade-offs |
| **Document decisions** | Future you (and others) need to understand why |
| **Adjust for context** | Your specific situation may require modifications |
| **Question assumptions** | Agents don't know everything about your project |

### Incremental Progress

| Practice | Implementation |
|----------|----------------|
| **Small steps** | Break large changes into minimal, focused increments |
| **Frequent commits** | Commit after each working step, not all at once |
| **Test constantly** | Verify functionality after every significant change |
| **Pause when needed** | Stop if something's wrong, don't push forward |
| **Rollback ready** | Keep escape hatches - know how to undo changes |

> **Rule of Thumb:** If a change can't be tested independently, it's too big.

### Context Preservation

| What to Save | Where/How |
|-------------|-----------|
| **Agent documentation** | Save in `analysis/` directory with descriptive names |
| **Decision rationale** | Add to commit messages or project notes |
| **Lessons learned** | Update CLAUDE.md or create retrospective docs |
| **Workflow outcomes** | Document what worked and what didn't |
| **Integration notes** | How systems connect, gotchas discovered |

### Agent Coordination

**How to work effectively with agents:**

```
DO:
✅ Provide clear, specific inputs
   "Analyze graphics/drawableshapes.go for consolidation opportunities"

✅ Give context and constraints
   "Must maintain backward compatibility"

✅ Review outputs critically
   "This approach makes sense for X but not Y because..."

✅ Adjust based on domain knowledge
   "The agent suggested A, but B fits our architecture better"

✅ Use agents as advisors
   "What are the trade-offs of this approach?"

DON'T:
❌ Give vague inputs
   "Make the code better"

❌ Accept outputs blindly
   "The agent said to do it this way, so..."

❌ Skip the decision step
   Jumping from analysis to implementation

❌ Use agents as autopilots
   "Just implement whatever the agent suggested"
```

---

## Success Metrics

### Refactoring Success Indicators

```
BEFORE ────────> AFTER
  │                 │
  ├─> Complex   ───> Simpler, clearer structure
  ├─> Duplicated ──> Unified, DRY code
  ├─> Scattered  ──> Consolidated, cohesive
  ├─> Hard to modify > Easy to extend
  └─> Tests pass ──> Tests still pass ✅
```

| Metric | Good Outcome |
|--------|--------------|
| **Understandability** | Code is easier to understand and explain |
| **Duplication** | Reduced or eliminated duplicate patterns |
| **Consolidation** | Related functionality brought together |
| **Extensibility** | Future features easier to add |
| **Functionality** | All tests pass, behavior unchanged |
| **LOC Trend** | Usually (but not always) reduced lines of code |

> **Critical:** If functionality changes or tests break, the refactoring failed.

### Implementation Success Indicators

```
BEFORE ────────> AFTER
  │                 │
  ├─> No feature ──> Feature exists and works
  ├─> Missing X  ──> X is implemented
  ├─> Basic only ──> Extended capabilities
  └─> Tests ?    ──> Tests validate feature ✅
```

| Metric | Good Outcome |
|--------|--------------|
| **Functionality** | Feature works as specified |
| **Integration** | Fits cleanly with existing systems |
| **Testability** | Can be tested and validated |
| **Maintainability** | Future developers can understand and modify |
| **User Experience** | Player-facing features feel good |
| **Technical Debt** | Not significantly increased (ideally reduced) |
| **Performance** | Acceptable game performance maintained |

> **Ideal Outcome:** New feature works, code quality stays high or improves.

---

## Getting Started Checklist

Follow these steps for your first workflow:

```
┌──────────────────────────────────────────────────────────────┐
│                     GETTING STARTED                          │
├──────────────────────────────────────────────────────────────┤
│                                                              │
│  ☐ 1. IDENTIFY YOUR NEED                                    │
│      What are you trying to accomplish?                     │
│      • Improve existing code → Refactoring                  │
│      • Add new functionality → Implementation               │
│                                                              │
│  ☐ 2. CHOOSE YOUR WORKFLOW                                  │
│      Review the decision tree and decision matrix           │
│      Select the appropriate workflow                        │
│                                                              │
│  ☐ 3. INVOKE THE SYNTHESIS AGENT                            │
│      • Refactoring: use refactoring-synth                   │
│      • Implementation: use implementation-synth             │
│      Provide clear, specific inputs                         │
│                                                              │
│  ☐ 4. REVIEW ANALYSIS THOROUGHLY                            │
│      Read the complete generated documentation              │
│      Understand all proposed approaches                     │
│      Don't skip this step!                                  │
│                                                              │
│  ☐ 5. MAKE YOUR DECISION                                    │
│      Choose the approach that fits best                     │
│      Document your rationale                                │
│      Note any adjustments needed                            │
│                                                              │
│  ☐ 6. CREATE IMPLEMENTATION PLAN (Refactoring Only)         │
│      Invoke refactoring-implementer                         │
│      Get step-by-step execution plan                        │
│                                                              │
│  ☐ 7. EXECUTE INCREMENTALLY                                 │
│      Small, testable steps                                  │
│      Test after each change                                 │
│      Commit working increments                              │
│                                                              │
│  ☐ 8. ADD/UPDATE TESTS                                      │
│      Invoke go-test-writer for changed/new files            │
│      Review and customize generated tests                   │
│      Run tests to verify coverage                           │
│                                                              │
│  ☐ 9. VALIDATE RESULTS                                      │
│      All tests pass                                         │
│      Functionality works as expected                        │
│      Success metrics achieved                               │
│                                                              │
│  ☐ 10. DOCUMENT OUTCOMES                                    │
│       Update project documentation                          │
│       Record lessons learned                                │
│       Share insights with team                              │
│                                                              │
└──────────────────────────────────────────────────────────────┘
```

### Quick Command Reference

```bash
# Refactoring Workflow
1. Invoke refactoring-synth with target files
2. Review generated analysis
3. Make decision, invoke refactoring-implementer
4. Execute plan step-by-step
5. Invoke go-test-writer for refactored files

# Implementation Workflow
1. Invoke implementation-synth with feature requirements
2. Review generated design documentation
3. Make decision and start implementation
4. Build incrementally with testing
5. Invoke go-test-writer for new feature files
```

---

## Remember

> **These workflows are guidelines to help you work effectively.**
> **Adapt them to your specific needs and context.**

**The key principles:**
- Agents advise, humans decide
- Review thoroughly before acting
- Work incrementally with frequent validation
- Document decisions and outcomes
- Test continuously, commit regularly

**When in doubt:**
- Start with analysis (synthesis agent)
- Take time for careful review
- Choose the simpler approach
- Break work into smaller steps
- Ask for help or clarification

---

## Additional Resources

- **CLAUDE.md** - Project-specific configuration and roadmap
- **analysis/** directory - Saved agent documentation and analysis
- **Agent profiles** - See detailed agent descriptions above
- **Decision trees** - Visual guides for choosing workflows

For questions or issues with workflows, refer to this document or consult with your team.
