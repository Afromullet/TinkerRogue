# Development Workflows Guide

> **Quick Reference:** Agent-assisted workflows for systematic code improvement and feature implementation

---

## Table of Contents

1. [Quick Start Guide](#quick-start-guide)
2. [Workflow Decision Tree](#workflow-decision-tree)
3. [Workflow Overview](#workflow-overview)
4. [Refactoring Workflow](#workflow-1-refactoring-workflow)
5. [Implementation Workflow](#workflow-2-implementation-workflow)
6. [Agent Roles Reference](#agent-roles-summary)
7. [Best Practices](#workflow-best-practices)
8. [Success Metrics](#success-metrics)

---

## Quick Start Guide

### I Need To... 🎯

| **Goal** | **Use This Workflow** | **Start With** |
|----------|----------------------|----------------|
| Fix duplicate code | Refactoring | `refactoring-synth` |
| Simplify complex system | Refactoring | `refactoring-synth` |
| Add new feature | Implementation | `implementation-synth` |
| Build new mechanic | Implementation | `implementation-synth` |
| Pay down technical debt | Refactoring | `refactoring-synth` |
| Extend existing system | Implementation | `implementation-synth` |

### The Universal Pattern

```
┌─────────────────┐      ┌─────────────────┐      ┌─────────────────┐
│  1. ANALYSIS    │ ───> │  2. DECISION     │ ───> │ 3. EXECUTE      │
│  (Agent-driven) │      │  (Human-driven)  │      │ (Collaborative) │
└─────────────────┘      └─────────────────┘      └─────────────────┘
```

---

## Workflow Decision Tree

```
                    ┌─────────────────────────┐
                    │  What do you need to do? │
                    └────────────┬─────────────┘
                                 │
                 ┌───────────────┴───────────────┐
                 │                               │
                 ▼                               ▼
      ┌──────────────────────┐        ┌──────────────────────┐
      │ Improve existing code │        │  Add new functionality│
      │  No new features      │        │   New capabilities   │
      └──────────┬───────────┘        └──────────┬───────────┘
                 │                               │
                 ▼                               ▼
      ┌──────────────────────┐        ┌──────────────────────┐
      │ REFACTORING WORKFLOW │        │IMPLEMENTATION WORKFLOW│
      │                      │        │                      │
      │ • Eliminate duplication       │ • Build new features │
      │ • Simplify architecture       │ • Add mechanics      │
      │ • Consolidate systems         │ • Extend capabilities│
      │ • Restructure code            │ • Create components  │
      └──────────────────────┘        └──────────────────────┘
                 │                               │
                 └───────────────┬───────────────┘
                                 │
                                 ▼
                    ┌─────────────────────────┐
                    │   Sometimes need both:  │
                    │ Refactor first, then    │
                    │ implement new feature   │
                    └─────────────────────────┘
```qui

---

## Workflow Overview

The TinkerRogue project uses **two primary workflows** for systematic development:

### 🔧 Refactoring Workflow
**Purpose:** Improve existing code structure without changing functionality
**Goal:** Better maintainability, reduced complexity, eliminated duplication

### ✨ Implementation Workflow
**Purpose:** Add new features, mechanics, and capabilities
**Goal:** New functionality that integrates cleanly with existing systems

---

### Common Pattern

Both workflows follow the same three-phase structure:

| **Phase** | **Who** | **Purpose** | **Output** |
|-----------|---------|-------------|------------|
| **1. Analysis** | Agent | Generate comprehensive documentation | Multi-perspective analysis with options |
| **2. Decision** | Human | Choose best approach | Selected strategy with adjustments |
| **3. Execution** | Collaborative | Implement the plan | Working, tested code |

> **Key Principle:** Agents provide analysis and options. Humans make decisions. Implementation is collaborative.

---

## Workflow 1: Refactoring Workflow

> **TL;DR:** Improve existing code structure, eliminate duplication, simplify architecture. No new features.

```
REFACTORING WORKFLOW
════════════════════════════════════════════════════════════════════

Step 1              Step 2              Step 3              Step 4
┌─────────────┐    ┌─────────────┐    ┌─────────────┐    ┌─────────────┐
│  ANALYSIS   │ -> │   REVIEW    │ -> │CREATE PLAN  │ -> │  IMPLEMENT  │
│             │    │             │    │             │    │             │
│ Agent-driven│    │Human-driven │    │Agent-driven │    │Collaborative│
│             │    │             │    │             │    │             │
│refactoring- │    │   You       │    │refactoring- │    │   You or    │
│   synth     │    │  decide     │    │implementer  │    │   Agent     │
└─────────────┘    └─────────────┘    └─────────────┘    └─────────────┘
```

---

### 📋 At a Glance

| **Aspect** | **Details** |
|------------|-------------|
| **Purpose** | Improve code structure without changing functionality |
| **Primary Agent** | `refactoring-synth` (analysis) + `refactoring-implementer` (planning) |
| **Duration** | Varies (analysis: 5-15 min, implementation: hours to days) |
| **Risk Level** | Medium (tests must pass, behavior unchanged) |
| **Output** | Cleaner, more maintainable code with same functionality |

---

### ✅ When to Use This Workflow

**Use refactoring when you see these patterns:**

- ❌ **Duplication:** Multiple functions or files share duplicate code patterns
- ❌ **Complexity:** A system has grown complex and needs architectural simplification
- ❌ **Scattered Logic:** Related functionality is spread across multiple locations
- ❌ **Maintenance Pain:** Code structure makes it difficult to add new features
- ❌ **Technical Debt:** You need to consolidate or unify similar components

**Don't use refactoring for:**
- ✋ Adding new features (use Implementation Workflow)
- ✋ Fixing bugs (just fix them directly)
- ✋ Changing behavior or adding capabilities

---

### Step 1: Generate Refactoring Analysis 🔍

**Agent:** `refactoring-synth` | **Role:** Analysis Coordinator

#### What Happens

```
INPUT                    PROCESS                     OUTPUT
┌──────────────┐        ┌──────────────┐            ┌──────────────┐
│ Files/Systems│ ────>  │Multi-Agent   │ ────>      │Comprehensive │
│ to Refactor  │        │Analysis      │            │Documentation │
└──────────────┘        └──────────────┘            └──────────────┘
                             │
                             ├─> Code structure
                             ├─> Duplication patterns
                             ├─> Architecture issues
                             ├─> Dependencies
                             └─> Impact assessment
```

#### Your Actions

1. **Identify** the files or systems that need refactoring
2. **Invoke** the `refactoring-synth` agent with target files
3. **Wait** for comprehensive analysis (typically 5-15 minutes)

#### Agent Process

The refactoring-synth coordinates specialized sub-agents analyzing:

- **Code Structure:** Patterns, organization, modularity
- **Duplication Analysis:** Repeated code, consolidation opportunities
- **Architecture Review:** Design issues, coupling, cohesion
- **Dependencies:** Relationships, impact radius
- **Risk Assessment:** What could break, migration strategies

#### Output Documentation

You'll receive a comprehensive analysis containing:

| **Section** | **Content** |
|-------------|-------------|
| **Current State** | What the code looks like now, identified problems |
| **Pain Points** | Specific issues making maintenance difficult |
| **Approaches** | 2-4 different refactoring strategies with pros/cons |
| **Trade-offs** | Complexity vs. benefit, risk vs. reward analysis |
| **Risk Assessment** | What could go wrong, mitigation strategies |
| **Migration Path** | How to get from current state to desired state |
| **Recommendations** | Agent's suggested approach with rationale |

---

### Step 2: Review and Decision 🤔

**Actor:** Developer (you) | **Role:** Decision Maker

> ⚠️ **Critical Human Decision Point:** Agents provide options, you choose the path forward.

#### Your Actions

```
REVIEW PROCESS
┌─────────────────────────────────────────────────────────────┐
│                                                             │
│  1. READ     Complete analysis thoroughly                  │
│              Don't skip sections, even if they seem obvious │
│                                                             │
│  2. EVALUATE Multiple proposed approaches                  │
│              Compare trade-offs and complexity              │
│                                                             │
│  3. CONSIDER Project context and goals                     │
│              How does this fit with other work?            │
│                                                             │
│  4. DECIDE   Choose refactoring strategy                   │
│              Document your rationale                        │
│                                                             │
│  5. ADJUST   Modify plan as needed                         │
│              Adapt to your specific constraints            │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

#### Decision Framework

**Ask yourself these key questions:**

| **Question** | **Why It Matters** |
|--------------|-------------------|
| 💰 **Does the refactoring justify the effort?** | ROI analysis: time investment vs. maintenance gains |
| ⚠️ **What risks are acceptable?** | Some approaches are safer but less impactful |
| 🔗 **How does this fit with other work?** | Avoid conflicts with parallel development |
| 📋 **Are there dependencies first?** | Some refactoring requires other changes first |
| 🎯 **Which approach aligns with project goals?** | Choose based on long-term architecture vision |
| ⏰ **What's the implementation timeline?** | Consider deadlines and resource availability |

#### Decision Output

**Document your decision with:**

- ✅ **Chosen Approach:** Which refactoring strategy you selected
- 📝 **Rationale:** Why this approach over alternatives
- 🔧 **Adjustments:** Any modifications to the proposed plan
- ⚡ **Priority:** When this fits in your development schedule

---

### Step 3: Create Implementation Plan 📋

**Agent:** `refactoring-implementer` | **Role:** Planning Specialist

#### What Happens

```
INPUT                    PROCESS                     OUTPUT
┌──────────────┐        ┌──────────────┐            ┌──────────────┐
│Your Chosen   │ ────>  │Break Down    │ ────>      │Step-by-Step  │
│Approach      │        │Into Steps    │            │Plan          │
└──────────────┘        └──────────────┘            └──────────────┘
                             │
                             ├─> Small, testable steps
                             ├─> Verification points
                             ├─> Rollback strategies
                             └─> File-specific changes
```

#### Your Actions

1. **Provide** your chosen approach from Step 2
2. **Include** any adjustments or constraints
3. **Wait** for detailed implementation plan

#### Agent Process

The refactoring-implementer creates a plan with these characteristics:

- **Small Steps:** Each change is minimal and focused
- **Testable:** Every step can be verified before proceeding
- **Reversible:** Clear rollback strategy if something goes wrong
- **Ordered:** Dependencies managed, safe progression guaranteed
- **Specific:** Exact files and changes for each step

#### Output Plan

You'll receive a step-by-step guide containing:

| **Component** | **Details** |
|--------------|-------------|
| **Step Sequence** | Numbered, ordered list of changes (Step 1, 2, 3...) |
| **File Targets** | Specific files to modify at each step |
| **Code Changes** | What to change, where, and how |
| **Test Checkpoints** | How to verify each step worked correctly |
| **Rollback Strategy** | What to do if a step causes problems |
| **Expected Outcomes** | What success looks like at each stage |

> 💡 **Best Practice:** Treat each step as a commit point. Don't combine steps.

---

### Step 4: Implementation ⚙️

**Actor:** Developer or Agent-Assisted | **Role:** Executor

#### Implementation Approaches

```
Choose your implementation style:

┌─────────────────────┐  ┌─────────────────────┐  ┌─────────────────────┐
│   MANUAL            │  │  AGENT-ASSISTED     │  │   HYBRID            │
│                     │  │                     │  │                     │
│ You do all steps   │  │ Agent executes      │  │ You: critical code  │
│                     │  │ under supervision   │  │ Agent: boilerplate  │
│                     │  │                     │  │                     │
│ ✓ Maximum control   │  │ ✓ Faster execution  │  │ ✓ Best of both     │
│ ✓ Learn deeply      │  │ ✓ Consistent style  │  │ ✓ Efficient        │
│ ✗ Time-consuming    │  │ ✗ Review overhead   │  │ ✓ Control + speed   │
└─────────────────────┘  └─────────────────────┘  └─────────────────────┘
```

#### Execution Workflow

```
FOR EACH STEP IN PLAN:
  ┌─────────────────────────────────────────┐
  │ 1. READ step instructions               │
  │ 2. IMPLEMENT changes in specified files │
  │ 3. TEST verify functionality unchanged  │
  │ 4. COMMIT incremental progress          │
  └─────────────────────────────────────────┘
       │
       ├─> ✅ Tests Pass? → Continue to next step
       │
       └─> ❌ Tests Fail? → Rollback, debug, retry
```

#### Best Practices

| **Practice** | **Why** |
|-------------|---------|
| ✅ **Follow plan order** | Steps are sequenced to minimize risk |
| ✅ **Test after each step** | Catch problems early, narrow debugging scope |
| ✅ **Commit incrementally** | Create safe rollback points |
| ✅ **Don't skip steps** | Each step prepares for the next |
| ✅ **Don't combine steps** | Smaller changes = easier debugging |
| ✅ **Refer to docs** | If unclear, check original analysis |

#### When Things Go Wrong

```
PROBLEM DETECTED
      │
      ├─> Minor issue?
      │   └─> Debug and fix, continue
      │
      ├─> Step doesn't work?
      │   └─> Rollback to previous commit
      │       Review refactoring docs
      │       Adjust approach
      │
      └─> Fundamental problem?
          └─> Return to Step 2 (Review & Decision)
              Choose different approach
```

> ⚠️ **Critical Rule:** All tests must pass before marking refactoring complete.

---

## Workflow 2: Implementation Workflow

> **TL;DR:** Add new features, mechanics, and capabilities. Build new functionality.

```
IMPLEMENTATION WORKFLOW
════════════════════════════════════════════════════════════════════

Step 1              Step 2              Step 3
┌─────────────┐    ┌─────────────┐    ┌─────────────┐
│  ANALYSIS   │ -> │   REVIEW    │ -> │  IMPLEMENT  │
│             │    │             │    │             │
│ Agent-driven│    │Human-driven │    │Collaborative│
│             │    │             │    │             │
│implementation-   │   You       │    │   You or    │
│   synth     │    │  decide     │    │   Agent     │
└─────────────┘    └─────────────┘    └─────────────┘
```

---

### 📋 At a Glance

| **Aspect** | **Details** |
|------------|-------------|
| **Purpose** | Add new features, mechanics, and capabilities |
| **Primary Agent** | `implementation-synth` (analysis) |
| **Duration** | Varies (analysis: 10-30 min, implementation: hours to weeks) |
| **Risk Level** | Medium-High (new code, integration challenges) |
| **Output** | New functionality integrated with existing systems |

---

### ✅ When to Use This Workflow

**Use implementation when you need to:**

- ✨ **New Mechanics:** Add a new game mechanic or system
- 📋 **Todo Items:** Implement a feature from the todo list
- 🔧 **New Components:** Build a new component or capability
- 🔌 **Extensions:** Extend existing systems with new behavior
- 🎮 **Gameplay Elements:** Create new gameplay features

**Don't use implementation for:**
- ✋ Improving existing code structure (use Refactoring Workflow)
- ✋ Fixing bugs (just fix them directly)
- ✋ Simplifying or consolidating code

---

### Step 1: Generate Implementation Analysis 🔍

**Agent:** `implementation-synth` | **Role:** Requirements & Design Analyzer

#### What Happens

```
INPUT                    PROCESS                     OUTPUT
┌──────────────┐        ┌──────────────┐            ┌──────────────┐
│Feature       │ ────>  │Multi-Agent   │ ────>      │Comprehensive │
│Requirements  │        │Analysis      │            │Design Docs   │
└──────────────┘        └──────────────┘            └──────────────┘
                             │
                             ├─> Technical requirements
                             ├─> Integration strategy
                             ├─> Data modeling
                             ├─> UI/UX considerations
                             └─> Testing approach
```

#### Your Actions

1. **Describe** the feature or capability you want to implement
2. **Provide** requirements, constraints, and context
3. **Invoke** the `implementation-synth` agent
4. **Wait** for comprehensive analysis (typically 10-30 minutes)

#### Agent Process

The implementation-synth coordinates specialized sub-agents analyzing:

- **Technical Requirements:** What needs to be built, constraints, dependencies
- **Integration Points:** How this connects to existing systems
- **Data Model:** Structures, state management, persistence needs
- **UI/UX Considerations:** Player-facing aspects, user experience
- **Testing Strategy:** How to validate the feature works correctly
- **Architecture Design:** Overall structure and component organization

#### Output Documentation

You'll receive comprehensive implementation docs containing:

| **Section** | **Content** |
|-------------|-------------|
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

### Step 2: Review and Decision 🤔

**Actor:** Developer (you) | **Role:** Design Decision Maker

> ⚠️ **Critical Human Decision Point:** Choose the implementation approach that fits your vision.

#### Your Actions

```
REVIEW PROCESS
┌─────────────────────────────────────────────────────────────┐
│                                                             │
│  1. READ     Complete implementation analysis              │
│              Understand all proposed approaches            │
│                                                             │
│  2. EVALUATE Architecture and design options               │
│              Compare complexity vs. extensibility          │
│                                                             │
│  3. CONSIDER Integration and dependencies                  │
│              How does this fit existing architecture?     │
│                                                             │
│  4. ASSESS   Testing strategy and validation              │
│              Can you verify this works correctly?          │
│                                                             │
│  5. DECIDE   Choose implementation approach                │
│              Document rationale and adjustments            │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

#### Decision Framework

**Ask yourself these key questions:**

| **Question** | **Why It Matters** |
|--------------|-------------------|
| 🏗️ **Does it align with existing patterns?** | Consistency matters for maintainability |
| 🧪 **Is the testing strategy adequate?** | Need confidence the feature works correctly |
| 🔗 **How does it interact with other features?** | Avoid conflicts and integration issues |
| 🔧 **Refactoring prerequisites?** | Sometimes existing code needs improvement first |
| ⏰ **What's the timeline and effort?** | Balance scope with available time |
| 📈 **How extensible is this approach?** | Consider future enhancements and variations |
| ⚡ **What's the performance impact?** | Ensure acceptable game performance |

#### Decision Output

**Document your decision with:**

- ✅ **Chosen Approach:** Which implementation strategy you selected
- 📝 **Rationale:** Why this approach fits best
- 🔧 **Customizations:** Adjustments to proposed design
- 📋 **Prerequisites:** Any refactoring or prep work needed first
- 🎯 **Success Criteria:** How you'll know the feature works correctly

---

### Step 3: Implementation ⚙️

**Actor:** Developer or Agent-Assisted | **Role:** Feature Builder

#### Implementation Approaches

```
Choose your implementation style:

┌─────────────────────┐  ┌─────────────────────┐  ┌─────────────────────┐
│  INDEPENDENT        │  │  AGENT-ASSISTED     │  │   COLLABORATIVE     │
│                     │  │                     │  │                     │
│ You build it all   │  │ Agent builds under  │  │ You: core logic     │
│ using docs as guide│  │ your direction      │  │ Agent: boilerplate  │
│                     │  │                     │  │                     │
│ ✓ Full ownership    │  │ ✓ Faster execution  │  │ ✓ Optimal balance  │
│ ✓ Deep learning     │  │ ✓ Less manual work  │  │ ✓ Focus on critical │
│ ✗ Time-consuming    │  │ ✗ Need clear specs  │  │ ✓ Leverage both     │
└─────────────────────┘  └─────────────────────┘  └─────────────────────┘
```

#### Recommended Implementation Order

```
IMPLEMENTATION PROGRESSION
┌─────────────────────────────────────────────────────────────┐
│                                                             │
│  1. DATA STRUCTURES    Define models, state, interfaces    │
│                        Foundation for everything else       │
│                                                             │
│  2. CORE LOGIC         Implement business rules            │
│                        No UI, pure functionality            │
│                                                             │
│  3. INTEGRATION        Connect to existing systems         │
│                        Gradual, tested integration          │
│                                                             │
│  4. UI/UX              Add player-facing elements          │
│                        Once core logic is stable            │
│                                                             │
│  5. POLISH             Edge cases, error handling          │
│                        Refinement and optimization          │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

#### Best Practices

| **Practice** | **Why** | **Example** |
|-------------|---------|-------------|
| ✅ **Start with data** | Foundation affects everything else | Define structs before methods |
| ✅ **Build incrementally** | Catch problems early | Implement one feature aspect at a time |
| ✅ **Test frequently** | Fast feedback loop | Test after each logical component |
| ✅ **Integrate gradually** | Minimize breaking changes | Connect one system at a time |
| ✅ **Write tests early** | Design driver, catches regressions | Create tests as you build, not after |
| ✅ **Commit regularly** | Safe rollback points | Commit each working increment |
| ✅ **Refer to docs** | Stay aligned with plan | Check implementation analysis when stuck |

#### Development Cycle

```
FOR EACH FEATURE COMPONENT:
  ┌─────────────────────────────────────────┐
  │ 1. DESIGN   Data structures/interfaces │
  │ 2. IMPLEMENT Core logic for component  │
  │ 3. TEST      Verify functionality      │
  │ 4. INTEGRATE Connect to existing code  │
  │ 5. VALIDATE  Test integration points   │
  │ 6. COMMIT    Save working progress     │
  └─────────────────────────────────────────┘
       │
       ├─> ✅ Works correctly? → Next component
       │
       └─> ❌ Problems? → Debug, refine, retry
```

#### When Implementation Reveals Issues

```
DISCOVERED ISSUE
      │
      ├─> Design problem?
      │   └─> Return to Step 2 (Review & Decision)
      │       Adjust approach
      │
      ├─> Integration challenge?
      │   └─> May need refactoring first
      │       Switch to Refactoring Workflow
      │
      └─> Implementation detail?
          └─> Debug and solve
              Continue implementation
```

> 💡 **Pro Tip:** If implementation feels harder than expected, existing code may need refactoring first.

---

## Choosing the Right Workflow

### Decision Matrix

| **Situation** | **Workflow** | **Why** |
|--------------|-------------|---------|
| Code works but hard to maintain | 🔧 Refactoring | Structure improvement needed |
| Duplicate code everywhere | 🔧 Refactoring | Consolidation opportunity |
| Can't add features easily | 🔧 Refactoring | Architecture blocking progress |
| System too complex | 🔧 Refactoring | Simplification required |
| Need new game mechanic | ✨ Implementation | New functionality required |
| Todo list item to build | ✨ Implementation | Feature development |
| Extend existing system | ✨ Implementation | New capabilities needed |
| Feature needs code cleanup first | 🔧🔧 Both (Refactor → Implement) | Prerequisites exist |
| Implementation reveals tech debt | ✨🔧 Both (Implement → Refactor) | Discovered during work |

---

### 🔄 Combining Workflows

Sometimes you need both workflows. Here's how they interact:

```
SCENARIO 1: Refactor-First Pattern
═══════════════════════════════════════════════════════════

You want to add a feature but existing code is blocking you.

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

```
SCENARIO 2: Implementation-First Pattern
═══════════════════════════════════════════════════════════

You start implementing and discover technical debt.

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

**Start Here:**
```
What's your primary goal?

┌─────────────────────────────────────┐
│ Make existing code better?          │ → Refactoring Workflow
├─────────────────────────────────────┤
│ Add something new to the game?      │ → Implementation Workflow
├─────────────────────────────────────┤
│ Add feature BUT code is messy?      │ → Refactor First, Then Implement
└─────────────────────────────────────┘
```

---

## Agent Roles Summary

### Quick Reference Card

```
┌─────────────────────────────────────────────────────────────────┐
│                       AGENT DIRECTORY                           │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  🔧 REFACTORING AGENTS                                          │
│  ────────────────────────────────────────────────────────────   │
│                                                                 │
│  refactoring-synth                                              │
│  └─> Analyzes code, generates refactoring documentation         │
│  └─> Input: Files/systems needing improvement                  │
│  └─> Output: Multi-approach analysis with trade-offs           │
│  └─> When: Start of refactoring workflow                       │
│                                                                 │
│  refactoring-implementer                                        │
│  └─> Creates step-by-step refactoring plans                    │
│  └─> Input: Chosen refactoring approach                        │
│  └─> Output: Ordered steps with testing checkpoints            │
│  └─> When: After you've decided on approach                    │
│                                                                 │
│  ─────────────────────────────────────────────────────────────  │
│                                                                 │
│  ✨ IMPLEMENTATION AGENTS                                       │
│  ────────────────────────────────────────────────────────────   │
│                                                                 │
│  implementation-synth                                           │
│  └─> Analyzes features, generates implementation docs          │
│  └─> Input: Feature requirements and descriptions              │
│  └─> Output: Multi-approach design with integration plan       │
│  └─> When: Start of implementation workflow                    │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
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

---

## Workflow Best Practices

### Core Principles

```
┌─────────────────────────────────────────────────────────────────┐
│                    WORKFLOW PRINCIPLES                          │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  👁️  AGENTS ADVISE, HUMANS DECIDE                              │
│      Use agent analysis to inform decisions, not make them     │
│                                                                 │
│  🔍  REVIEW THOROUGHLY                                          │
│      Don't skip the review step - it's where value is added    │
│                                                                 │
│  🧩  INCREMENTAL PROGRESS                                       │
│      Small, tested steps beat big bang changes                 │
│                                                                 │
│  📚  PRESERVE CONTEXT                                           │
│      Document decisions and link them to implementation        │
│                                                                 │
│  🎯  VALIDATE CONTINUOUSLY                                      │
│      Test after each change, commit working increments         │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

---

### Detailed Best Practices

#### 1. Documentation Review

| **Practice** | **Why It Matters** |
|-------------|-------------------|
| ✅ **Don't skip review** | Agent analysis provides options, you make the choice |
| ✅ **Compare approaches** | Multiple perspectives reveal trade-offs |
| ✅ **Document decisions** | Future you (and others) need to understand why |
| ✅ **Adjust for context** | Your specific situation may require modifications |
| ✅ **Question assumptions** | Agents don't know everything about your project |

#### 2. Incremental Progress

| **Practice** | **Implementation** |
|-------------|-------------------|
| ✅ **Small steps** | Break large changes into minimal, focused increments |
| ✅ **Frequent commits** | Commit after each working step, not all at once |
| ✅ **Test constantly** | Verify functionality after every significant change |
| ✅ **Pause when needed** | Stop if something's wrong, don't push forward |
| ✅ **Rollback ready** | Keep escape hatches - know how to undo changes |

> 💡 **Rule of Thumb:** If a change can't be tested independently, it's too big.

#### 3. Context Preservation

| **What to Save** | **Where/How** |
|-----------------|---------------|
| **Agent documentation** | Save in `analysis/` directory with descriptive names |
| **Decision rationale** | Add to commit messages or project notes |
| **Lessons learned** | Update CLAUDE.md or create retrospective docs |
| **Workflow outcomes** | Document what worked and what didn't |
| **Integration notes** | How systems connect, gotchas discovered |

#### 4. Agent Coordination

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

### How to Measure Success

#### 🔧 Refactoring Success Indicators

```
BEFORE ────────> AFTER
  │                 │
  ├─> Complex   ───> Simpler, clearer structure
  ├─> Duplicated ──> Unified, DRY code
  ├─> Scattered  ──> Consolidated, cohesive
  ├─> Hard to modify > Easy to extend
  └─> Tests pass ──> Tests still pass ✅
```

| **Metric** | **Good Outcome** |
|-----------|------------------|
| **Understandability** | Code is easier to understand and explain |
| **Duplication** | Reduced or eliminated duplicate patterns |
| **Consolidation** | Related functionality brought together |
| **Extensibility** | Future features easier to add |
| **Functionality** | All tests pass, behavior unchanged |
| **LOC Trend** | Usually (but not always) reduced lines of code |

> ⚠️ **Critical:** If functionality changes or tests break, the refactoring failed.

#### ✨ Implementation Success Indicators

```
BEFORE ────────> AFTER
  │                 │
  ├─> No feature ──> Feature exists and works
  ├─> Missing X  ──> X is implemented
  ├─> Basic only ──> Extended capabilities
  └─> Tests ?    ──> Tests validate feature ✅
```

| **Metric** | **Good Outcome** |
|-----------|------------------|
| **Functionality** | Feature works as specified |
| **Integration** | Fits cleanly with existing systems |
| **Testability** | Can be tested and validated |
| **Maintainability** | Future developers can understand and modify |
| **User Experience** | Player-facing features feel good |
| **Technical Debt** | Not significantly increased (ideally reduced) |
| **Performance** | Acceptable game performance maintained |

> 💡 **Ideal Outcome:** New feature works, code quality stays high or improves.

---

## Getting Started Checklist

### Your First Workflow

Follow these steps to get started:

```
┌─────────────────────────────────────────────────────────────────┐
│                     GETTING STARTED                             │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ☐ 1. IDENTIFY YOUR NEED                                       │
│      What are you trying to accomplish?                        │
│      • Improve existing code → Refactoring                     │
│      • Add new functionality → Implementation                  │
│                                                                 │
│  ☐ 2. CHOOSE YOUR WORKFLOW                                     │
│      Review the decision tree and decision matrix              │
│      Select the appropriate workflow                           │
│                                                                 │
│  ☐ 3. INVOKE THE SYNTHESIS AGENT                               │
│      • Refactoring: use refactoring-synth                      │
│      • Implementation: use implementation-synth                │
│      Provide clear, specific inputs                            │
│                                                                 │
│  ☐ 4. REVIEW ANALYSIS THOROUGHLY                               │
│      Read the complete generated documentation                 │
│      Understand all proposed approaches                        │
│      Don't skip this step!                                     │
│                                                                 │
│  ☐ 5. MAKE YOUR DECISION                                       │
│      Choose the approach that fits best                        │
│      Document your rationale                                   │
│      Note any adjustments needed                               │
│                                                                 │
│  ☐ 6. CREATE IMPLEMENTATION PLAN (Refactoring Only)            │
│      Invoke refactoring-implementer                            │
│      Get step-by-step execution plan                           │
│                                                                 │
│  ☐ 7. EXECUTE INCREMENTALLY                                    │
│      Small, testable steps                                     │
│      Test after each change                                    │
│      Commit working increments                                 │
│                                                                 │
│  ☐ 8. VALIDATE RESULTS                                         │
│      All tests pass                                            │
│      Functionality works as expected                           │
│      Success metrics achieved                                  │
│                                                                 │
│  ☐ 9. DOCUMENT OUTCOMES                                        │
│      Update project documentation                              │
│      Record lessons learned                                    │
│      Share insights with team                                  │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

---

### Quick Command Reference

```bash
# Refactoring Workflow
1. Invoke refactoring-synth with target files
2. Review generated analysis
3. Make decision, invoke refactoring-implementer
4. Execute plan step-by-step

# Implementation Workflow
1. Invoke implementation-synth with feature requirements
2. Review generated design documentation
3. Make decision and start implementation
4. Build incrementally with testing
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
