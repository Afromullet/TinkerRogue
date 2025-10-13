# Claude Code Agent Reference

> **Quick Reference:** Overview of specialized agents available for TinkerRogue development

---

## Table of Contents

1. [Multi-Agent Coordinators](#multi-agent-coordinators)
2. [Refactoring Specialists](#refactoring-specialists)
3. [Implementation Specialists](#implementation-specialists)
4. [Code Quality Agents](#code-quality-agents)
5. [Tactical Game Design](#tactical-game-design)
6. [Meta-Analysis Agents](#meta-analysis-agents)
7. [Quick Selection Guide](#quick-selection-guide)

---

## Multi-Agent Coordinators

These agents orchestrate multiple specialized agents to provide comprehensive analysis.

### refactoring-synth

**Type:** Multi-Agent Coordinator
**Purpose:** Comprehensive refactoring analysis with multiple perspectives

**What it does:**
- Coordinates `refactoring-pro`, `tactical-simplifier`, and `refactoring-critic`
- Analyzes code structure, duplication patterns, and architecture issues
- Generates 2-4 refactoring approaches with trade-offs
- Provides synthesis of all agent perspectives in final recommendation

**Collaborative Process:**
```
refactoring-synth
  ├─> refactoring-pro (analysis)
  ├─> tactical-simplifier (game architecture focus)
  └─> refactoring-critic (synthesis & evaluation)
      └─> Final consolidated report
```

**When to use:**
- Starting a refactoring workflow
- Complex systems with multiple improvement opportunities
- Need multiple perspectives on technical debt

**Output:**
- Comprehensive analysis document with current state, approaches, trade-offs, and recommendations

---

### implementation-synth

**Type:** Multi-Agent Coordinator
**Purpose:** Feature implementation planning with tactical gameplay and Go standards

**What it does:**
- Coordinates `trpg-creator` and `go-standards-reviewer`
- Balances tactical game design depth with Go programming best practices
- Generates multiple implementation approaches
- Provides synthesis with `implementation-critic` evaluation

**Collaborative Process:**
```
implementation-synth
  ├─> trpg-creator (gameplay design)
  ├─> go-standards-reviewer (code quality)
  └─> implementation-critic (synthesis & evaluation)
      └─> Final implementation plan
```

**When to use:**
- Implementing new tactical game features
- Need both gameplay depth and code quality
- Starting an implementation workflow

**Output:**
- Feature design document with architecture, approaches, integration strategy, and testing plan

---

## Refactoring Specialists

### refactoring-pro

**Type:** Specialized Analyst
**Purpose:** Deep refactoring analysis for complex codebases

**What it does:**
- Analyzes code structure and identifies consolidation opportunities
- Specializes in simplification and complexity reduction
- Focuses on transitioning between project goals
- Identifies unnecessary complexity and removal strategies

**When to use:**
- As part of `refactoring-synth` (automatic)
- Standalone for targeted refactoring analysis
- Simplifying specific complex systems

**Example use cases:**
- Consolidating 8+ shape types with code duplication
- Simplifying input system with scattered global state
- Removing unnecessary abstractions

---

### refactoring-critic

**Type:** Critical Evaluator
**Purpose:** Skeptical assessment of refactoring proposals

**What it does:**
- Analyzes refactoring approaches for practical value
- Distinguishes valuable improvements from theoretical over-engineering
- Evaluates software engineering soundness
- Provides reality-check on proposed changes

**When to use:**
- As synthesizer in `refactoring-synth` (automatic)
- Evaluating refactoring proposals from other agents
- Need critical assessment before implementation

**Key strength:**
- Prevents over-engineering and unnecessary complexity
- Ensures refactoring solves real problems

---

### tactical-simplifier

**Type:** Game Architecture Specialist
**Purpose:** Simplifying tactical turn-based game systems

**What it does:**
- Reduces mental complexity in tactical systems
- Clear separation of concerns for game mechanics
- Deep understanding of squad building, spells, abilities
- Preserves combat depth while simplifying architecture

**When to use:**
- As part of `refactoring-synth` for game systems (automatic)
- Refactoring tactical combat, squad, or ability systems
- Need to simplify complex game mechanics

**Specialization:**
- Tactical RPG systems (Fire Emblem, FFT-style)
- Squad-based combat
- Ability and spell systems

---

## Implementation Specialists

### trpg-creator

**Type:** Tactical Game Designer
**Purpose:** Implementing tactical RPG gameplay features

**What it does:**
- Designs tactical depth and meaningful player choices
- Creates detailed planning documents before implementation
- Inspired by Fire Emblem, FFT, Nephilim, Soul Nomad, Jagged Alliance
- Focuses on tactical combat systems

**When to use:**
- As part of `implementation-synth` for gameplay features (automatic)
- Implementing squad combat, abilities, targeting systems
- Need tactical game design expertise

**Specialization:**
- Turn-based tactical combat
- Squad-based gameplay
- Ability and targeting mechanics

---

### implementation-critic

**Type:** Critical Evaluator
**Purpose:** Skeptical assessment of implementation proposals

**What it does:**
- Analyzes implementation approaches for code quality
- Evaluates architectural soundness and maintainability
- Distinguishes well-designed solutions from over-engineering
- Reality-check on proposed implementations

**When to use:**
- As synthesizer in `implementation-synth` (automatic)
- Evaluating implementation plans from other agents
- Need critical assessment before building

**Key strength:**
- Prevents over-engineered solutions
- Ensures implementations are practical and maintainable

---

## Code Quality Agents

### go-standards-reviewer

**Type:** Go Standards Expert
**Purpose:** Comprehensive Go best practices analysis

**What it does:**
- Reviews code against strict Go programming practices
- Checks organization, naming, error handling, interfaces
- Flags allocations in hot paths
- Identifies unconventional patterns and performance issues

**When to use:**
- As part of `implementation-synth` (automatic)
- Standalone review of Go packages or files
- Before merging significant Go code changes

**Output:**
- Detailed analysis with priority levels, code examples, fixes, and references to Go guidelines

**Example command:**
```
"Review combat/attack.go for Go standards compliance"
```

---

### go-test-writer

**Type:** Test Generation Expert
**Purpose:** Creating comprehensive, idiomatic Go test suites

**What it does:**
- Analyzes source files and generates proper Go tests
- Creates table-driven tests, benchmarks, test helpers
- Follows testing best practices (TestMain, subtests, proper assertions)
- Comprehensive coverage for game development codebases

**When to use:**
- Need tests for existing code
- Implementing new features that need test coverage
- Want idiomatic Go test patterns

**Example command:**
```
"Write tests for combat/attack.go"
"Create tests for the inventory system"
```

---

## Tactical Game Design

### karen

**Type:** Reality-Check Specialist
**Purpose:** Assessing actual project completion state

**What it does:**
- Cuts through incomplete implementations
- Validates what's actually built vs. what was claimed
- Creates realistic plans to finish work
- Ensures implementations match requirements without over-engineering

**When to use:**
- Tasks marked complete but functionality questionable
- Need to validate actual vs. claimed progress
- Want no-bullshit assessment of project state
- Ensure implementations match requirements exactly

**Key strength:**
- Prevents "90% done" syndrome
- Reality-based progress assessment

**Example scenarios:**
- "Several tasks marked done but getting errors when testing"
- "Verify authentication system actually works as claimed"

---

## Meta-Analysis Agents

### insight-synthesizer

**Type:** Knowledge Management Expert
**Purpose:** Extracting insights from multi-agent interactions

**What it does:**
- Synthesizes insights from agent interactions
- Identifies patterns in collaborative problem-solving
- Builds collective intelligence from agent collaborations
- Masters cross-agent learning and best practice extraction

**When to use:**
- After multiple agent workflows
- Want to understand patterns in agent recommendations
- Building knowledge base from agent interactions
- Continuous system improvement

**Specialization:**
- Meta-analysis of agent effectiveness
- Pattern recognition across workflows
- Knowledge synthesis

---

### docs-architect

**Type:** Documentation Generator
**Purpose:** Creating comprehensive technical documentation

**What it does:**
- Analyzes architecture, design patterns, implementation details
- Produces long-form technical manuals and ebooks
- Creates system documentation and architecture guides
- Technical deep-dives from existing codebases

**When to use (proactively):**
- Need system documentation
- Architecture guides
- Technical deep-dives
- Comprehensive project documentation

**Output:**
- Long-form technical documentation
- Architecture guides
- API documentation

---

## Quick Selection Guide

### Decision Tree

```
What do you need?

├─ Improve existing code
│  ├─ Complex game system → refactoring-synth (includes tactical-simplifier)
│  ├─ General code → refactoring-synth (standard)
│  └─ Need reality-check → refactoring-critic (evaluation)
│
├─ Build new feature
│  ├─ Tactical gameplay → implementation-synth (includes trpg-creator)
│  ├─ General feature → implementation-synth (standard)
│  └─ Need reality-check → implementation-critic (evaluation)
│
├─ Code quality
│  ├─ Review Go code → go-standards-reviewer
│  └─ Need tests → go-test-writer
│
├─ Project assessment
│  ├─ Validate completion → karen
│  └─ Extract insights → insight-synthesizer
│
└─ Documentation
   └─ Technical docs → docs-architect
```

---

### Use Case Matrix

| **Task** | **Primary Agent** | **Collaborative Agents** |
|----------|------------------|-------------------------|
| Refactor complex game system | refactoring-synth | refactoring-pro, tactical-simplifier, refactoring-critic |
| Refactor general code | refactoring-synth | refactoring-pro, refactoring-critic |
| Implement tactical feature | implementation-synth | trpg-creator, go-standards-reviewer, implementation-critic |
| Implement general feature | implementation-synth | go-standards-reviewer, implementation-critic |
| Review Go code standards | go-standards-reviewer | (standalone) |
| Write Go tests | go-test-writer | (standalone) |
| Validate project state | karen | (standalone) |
| Extract workflow insights | insight-synthesizer | (standalone) |
| Generate documentation | docs-architect | (standalone) |

---

### Agent Selection Flowchart

```
START
  │
  ├─ Need ANALYSIS? ───────────────────┐
  │                                    │
  │                                    ▼
  │                         Is it game-specific?
  │                              │         │
  │                             Yes       No
  │                              │         │
  │                              ▼         ▼
  │                    refactoring-synth (full)  refactoring-synth (standard)
  │                    + tactical-simplifier
  │
  ├─ Need IMPLEMENTATION? ──────────────┐
  │                                     │
  │                                     ▼
  │                          Is it tactical gameplay?
  │                              │         │
  │                             Yes       No
  │                              │         │
  │                              ▼         ▼
  │                    implementation-synth (full)  implementation-synth (standard)
  │                    + trpg-creator
  │
  ├─ Need CODE QUALITY? ────────────────┐
  │                                     │
  │                                     ▼
  │                          What aspect?
  │                          │         │
  │                       Review    Tests
  │                          │         │
  │                          ▼         ▼
  │                  go-standards   go-test
  │                   -reviewer     -writer
  │
  └─ Need META-ANALYSIS? ───────────────┐
                                        │
                                        ▼
                             What type?
                        │         │        │
                    Progress  Insights  Docs
                        │         │        │
                        ▼         ▼        ▼
                     karen   insight-  docs-
                            synthesizer architect
```

---

## Agent Interaction Patterns

### Pattern 1: Synth → Specialist → Critic

**Used by:** `refactoring-synth`, `implementation-synth`

```
Step 1: SYNTH agent receives request
  │
  ├─> Launches specialist agents in parallel
  │   ├─> Agent A (domain expert)
  │   └─> Agent B (another perspective)
  │
Step 2: Specialist agents complete analysis
  │
  └─> CRITIC agent synthesizes all perspectives
      └─> Produces final consolidated report
```

**Why this pattern:**
- Multiple perspectives reduce blind spots
- Critic provides reality-check and synthesis
- Comprehensive analysis with skeptical evaluation

---

### Pattern 2: Standalone Specialist

**Used by:** `go-test-writer`, `go-standards-reviewer`, `karen`, `docs-architect`

```
Request → Agent → Output
```

**Why this pattern:**
- Focused, specialized task
- No need for multiple perspectives
- Direct execution more efficient

---

### Pattern 3: Meta-Analysis

**Used by:** `insight-synthesizer`

```
Multiple agent workflows
  │
  └─> insight-synthesizer analyzes patterns
      └─> Extracts lessons and best practices
```

**Why this pattern:**
- Learn from agent interactions
- Build institutional knowledge
- Continuous improvement

---

## Best Practices

### Working with Multi-Agent Coordinators

```
DO:
✅ Let synth agents coordinate sub-agents automatically
✅ Provide clear, specific inputs to the synth agent
✅ Review the final synthesized output thoroughly
✅ Trust the collaborative process

DON'T:
❌ Manually invoke sub-agents when using synth
❌ Skip the decision/review step after analysis
❌ Ignore the critic's reality-check sections
```

---

### Agent Output Locations

**Standard locations for agent outputs:**

```
analysis/
  ├─ refactoring_*.md         # From refactoring-synth
  ├─ implementation_*.md      # From implementation-synth
  ├─ standards_review_*.md    # From go-standards-reviewer
  ├─ test_plan_*.md           # From go-test-writer
  └─ insights_*.md            # From insight-synthesizer

docs/
  └─ *.md                     # From docs-architect
```

---

## Common Workflows

### Workflow 1: Refactoring Complex Game System

```
1. Invoke: refactoring-synth
   Input: "Analyze squads/ package for consolidation opportunities"

2. Agent Process (automatic):
   - refactoring-pro analyzes code structure
   - tactical-simplifier focuses on game architecture
   - refactoring-critic synthesizes and evaluates

3. You Receive:
   - Comprehensive refactoring analysis with approaches

4. You Decide:
   - Choose approach, document rationale

5. Implement:
   - Follow chosen approach
```

---

### Workflow 2: Implementing New Tactical Feature

```
1. Invoke: implementation-synth
   Input: "Design ability system with auto-triggering for squad combat"

2. Agent Process (automatic):
   - trpg-creator designs tactical mechanics
   - go-standards-reviewer ensures Go best practices
   - implementation-critic evaluates and synthesizes

3. You Receive:
   - Feature design with multiple approaches

4. You Decide:
   - Choose approach, adjust as needed

5. Implement:
   - Build feature incrementally
```

---

### Workflow 3: Code Quality Review

```
1. Invoke: go-standards-reviewer
   Input: "Review squads/squadcombat.go for Go standards"

2. Agent analyzes:
   - Code organization, naming, error handling
   - Performance issues, allocations
   - Go idioms and conventions

3. You Receive:
   - Detailed analysis with priorities and fixes

4. You Address:
   - Fix high-priority issues
   - Consider medium/low priority improvements
```

---

### Workflow 4: Reality Check

```
1. Invoke: karen
   Input: "Verify squad combat system is actually complete"

2. Agent assesses:
   - What's claimed vs. what's actually built
   - What works vs. what's broken
   - What's missing to truly complete the work

3. You Receive:
   - No-bullshit assessment of actual state
   - Realistic completion plan

4. You Act:
   - Address identified gaps
   - Complete remaining work
```

---

## Agent Capabilities Summary

| **Agent** | **Type** | **Collaborative** | **Best For** |
|-----------|----------|------------------|--------------|
| refactoring-synth | Coordinator | Yes (3 agents) | Complex refactoring analysis |
| refactoring-pro | Specialist | No | Deep refactoring analysis |
| refactoring-critic | Evaluator | No | Skeptical synthesis |
| tactical-simplifier | Specialist | No | Game architecture simplification |
| implementation-synth | Coordinator | Yes (3 agents) | Feature implementation planning |
| trpg-creator | Specialist | No | Tactical gameplay design |
| implementation-critic | Evaluator | No | Implementation skepticism |
| go-standards-reviewer | Specialist | No | Go code quality review |
| go-test-writer | Specialist | No | Go test generation |
| karen | Specialist | No | Reality-based completion assessment |
| insight-synthesizer | Meta | No | Cross-agent learning |
| docs-architect | Specialist | No | Technical documentation |

---

## Remember

**Multi-agent coordinators handle collaboration automatically:**
- `refactoring-synth` orchestrates refactoring analysis
- `implementation-synth` orchestrates implementation planning
- You only need to invoke the coordinator - it handles the rest

**Specialist agents work standalone:**
- Direct invocation for focused tasks
- No coordination needed
- Faster for specific, well-defined work

**Always review agent output:**
- Agents advise, humans decide
- Critical evaluation step is essential
- Your domain knowledge completes the picture

---

## Additional Resources

- **DEVELOPMENT_WORKFLOWS.md** - Detailed workflow guides using these agents
- **CLAUDE.md** - Project-specific agent usage and roadmap
- **analysis/** directory - Saved agent outputs and documentation

For agent-specific questions, refer to this document or the detailed workflow guide.
