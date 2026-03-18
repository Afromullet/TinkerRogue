# Dependency Boundary Design Review

**Reviewed:** 2026-03-18
**Source Document:** `docs/To_Review/DEPENDENCY_BOUNDARIES.md`
**Scope:** All interfaces, patterns, and import boundaries documented in the source document, validated against actual source code.

---

## Overall Verdict

**Solid foundation.** The architecture is well-designed and ready for continued growth. Import boundaries are clean, structural typing is used idiomatically, and the wiring layer is properly centralized. The three core patterns — structural typing, registry/plugin, and dependency injection — are applied consistently and correctly throughout the codebase.

Every boundary claim in the document was verified against actual imports. No violations found.

---

## Interface-by-Interface Assessment

### Combat Pipeline (`tactical/combat/combat_contracts.go`)

| Interface | Methods | Cohesion | Verdict |
|-----------|---------|----------|---------|
| `CombatStarter` | 1 | Perfect | Clean — textbook single-method interface |
| `CombatTransitioner` | 1 | Perfect | Clean — single responsibility, structural typing works well |
| `CombatStartRollback` | 1 | Perfect | Clean — optional interface via type assertion, excellent ISP |
| `CombatCleaner` | 1 | Perfect | Clean — GUI never imports CombatService directly for cleanup |
| `EncounterCallbacks` | 3 | Acceptable | Mixes action (`ExitCombat`) with queries (`GetRosterOwnerID`, `GetCurrentEncounterID`), but all consumers use all methods. No split needed. |

**`CombatSetup` struct:** 9 fields including domain-specific flags (`IsGarrisonDefense`, `IsRaidCombat`, `PostCombatReturnMode`). Acceptable as a data transfer object — the flags are minimal and documented.

### Combat Lifecycle (`mind/combatlifecycle/pipeline.go`)

| Interface | Methods | Cohesion | Verdict |
|-----------|---------|----------|---------|
| `CombatResolver` | 1 | Perfect | Clean — each domain implements its own resolver without combatlifecycle importing those packages |

`ExecuteCombatStart` and `ExecuteResolution` are well-designed entry points. `combatlifecycle` imports only `common` and `tactical/combat` — minimal dependency footprint.

### Encounter-GUI Boundary (`mind/encounter/types.go`)

| Interface | Methods | Cohesion | Verdict |
|-----------|---------|----------|---------|
| `CombatTransitionHandler` | 6 | Moderate | See Issue #2 below |

### AI and Threat System (`tactical/combatservices/ai_interfaces.go`)

| Interface | Methods | Cohesion | Verdict |
|-----------|---------|----------|---------|
| `AITurnController` | 4 | Good | Clean — tight cohesion around decision-making and attack queueing |
| `ThreatProvider` | 3 | Good | Clean — all methods relate to faction-level threat management |
| `ThreatLayerEvaluator` | 9 | Good but wide | See Issue #3 below |

`ai_interfaces.go` imports only `world/coords` and `ecs` — zero imports of `mind/*` packages despite defining interfaces for AI behavior. Dependency inversion done correctly.

### Rendering (`visual/rendering/renderdata.go`)

| Interface | Methods | Cohesion | Verdict |
|-----------|---------|----------|---------|
| `SquadInfoProvider` | 2 | Perfect | Clean — minimal data provider |
| `UnitInfoProvider` | 2 | Perfect | Clean — minimal data provider |

### UI Modes (`gui/framework/uimode.go`)

| Interface | Methods | Cohesion | Verdict |
|-----------|---------|----------|---------|
| `UIMode` | 8 | Moderate | Pragmatically acceptable — every mode needs all 8 methods (lifecycle + game loop + UI framework). Splitting would add ceremony without benefit. |
| `OverlayRenderer` | 1 | Perfect | Clean optional interface via type assertion |
| `ActionMapProvider` | 1 | Perfect | Clean optional interface via type assertion |

The optional interface pattern (`OverlayRenderer`, `ActionMapProvider`) is a strong design choice that scales well. Modes opt-in to capabilities without bloating the required `UIMode` interface.

### Registry/Plugin Patterns

| Registry | Interface Methods | Pattern | Verdict |
|----------|------------------|---------|---------|
| `MapGenerator` | 3 | `init()` self-registration | Clean. See Issue #4 for duplicate detection. |
| `ArtifactBehavior` | 7 | `init()` self-registration + `BaseBehavior` embedding | `BaseBehavior` pattern is excellent — concrete behaviors override only needed hooks. 7 methods is at the edge but justified by the number of lifecycle hooks. |
| `SaveChunk` | 5 + optional `Validatable` | `init()` self-registration | Three-phase load (Load → RemapIDs → Validate) is well-designed. Optional `Validatable` via type assertion follows the same pattern as `CombatStartRollback`. |

### GUI Dependency Injection Structs

| Struct | File | Verdict |
|--------|------|---------|
| `CombatModeDeps` | `gui/guicombat/combatdeps.go` | See Issue #1 — `CombatService` is concrete type |
| `SpellCastingDeps` | `gui/guispells/spell_deps.go` | Clean — all fields are either interfaces or framework types |
| `ArtifactActivationDeps` | `gui/guiartifacts/artifact_deps.go` | See Issue #1 — `CombatService` is concrete type |

All three correctly use `combat.EncounterCallbacks` (interface) instead of importing `mind/encounter` directly.

---

## Issues Found

### Issue 1: `CombatService` Passed as Concrete Type Through GUI Deps (MEDIUM)

**Problem:** `CombatModeDeps` and `ArtifactActivationDeps` hold `*combatservices.CombatService` directly. This exposes 6+ public subsystem fields (`TurnManager`, `MovementSystem`, `CombatActSystem`, `BattleRecorder`, etc.) to GUI code. The GUI can reach into any internal system, which undermines the purpose of the Deps struct pattern — narrowing what consumers see.

**Files:**
- `gui/guicombat/combatdeps.go:21`
- `gui/guiartifacts/artifact_deps.go:12`

**Why it matters:** If `CombatService` internals change (rename a field, restructure subsystems), every GUI file touching those fields breaks. The coupling is invisible until refactor time.

**Why it's not urgent:** The concrete type works correctly today. The risk is maintainability during future refactors, not correctness. Extracting a narrow interface requires auditing every GUI method that touches `CombatService` to determine the minimal surface area — a significant task.

**Recommendation:** When `CombatService` next gets a significant refactor, define a `CombatActions` interface in `tactical/combat` that exposes only what GUI needs (attack, move, end turn, get turn state, get threat data). For now, document this as a known coupling point in `DEPENDENCY_BOUNDARIES.md`.

**Potential future interface shape:**
```go
// In tactical/combat/combat_contracts.go
type CombatActions interface {
    ExecuteAttack(attackerID, defenderID ecs.EntityID) (*CombatResult, error)
    ExecuteMove(squadID ecs.EntityID, path []coords.LogicalPosition) error
    EndTurn()
    GetCurrentFaction() ecs.EntityID
    IsPlayerTurn() bool
    // ... only methods GUI actually calls
}
```

---

### Issue 2: `CombatTransitionHandler` Is Wider Than Necessary (LOW-MEDIUM)

**Problem:** 6 methods mixing three concerns in one interface:

| Concern | Methods |
|---------|---------|
| State setup | `SetPostCombatReturnMode`, `SetTriggeredEncounterID`, `ResetTacticalState` |
| Mode activation | `EnterCombatMode` |
| Player queries | `GetPlayerEntityID`, `GetPlayerPosition` |

**File:** `mind/encounter/types.go`

**Why it matters:** If a second type ever needs to satisfy this interface (e.g., a test double, or a different coordinator for a different game mode), it would be forced to implement unrelated methods.

**Why it's not urgent:** Only one implementor exists (`GameModeCoordinator`), and there's unlikely to be a second. The interface is stable and well-understood.

**Recommendation:** No action now. If a second implementor is ever needed, split into:
```go
type CombatModeActivator interface {
    SetPostCombatReturnMode(mode string)
    SetTriggeredEncounterID(id ecs.EntityID)
    ResetTacticalState()
    EnterCombatMode() error
}

type PlayerLocationProvider interface {
    GetPlayerEntityID() ecs.EntityID
    GetPlayerPosition() *coords.LogicalPosition
}
```

---

### Issue 3: `ThreatLayerEvaluator` Has 9 Methods (LOW-MEDIUM)

**Problem:** Widest interface in the codebase — 2 cache management methods + 7 position-based threat layer queries.

**File:** `tactical/combatservices/ai_interfaces.go:36-46`

**Why it's acceptable:** All 9 methods are cohesive (threat evaluation at map positions). The width comes from having 7 distinct threat layers, which is a domain modeling choice, not an interface design flaw.

**Alternative considered:** Consolidating to `GetThreatAt(pos, layerType) float64` would reduce interface width from 9 to 3, but at the cost of losing compile-time type safety and making call sites less readable (string/enum layer names instead of named methods).

**Recommendation:** Leave as-is unless new threat layers are added frequently. If the layer count grows beyond 9-10, consolidate to a generic getter with a `ThreatLayer` enum.

---

### Issue 4: Registries Lack Duplicate Detection (LOW)

**Problem:** All three registries silently accept duplicate registrations:

| Registry | File | Behavior |
|----------|------|----------|
| MapGenerator | `world/worldmap/generator.go` | Silent map overwrite |
| ArtifactBehavior | `gear/artifactbehavior.go` | Silent map overwrite |
| SaveChunk | `savesystem/savesystem.go` | Allows duplicate appends (slice, no dedup) |

**Why it's low priority:** Registration only happens in `init()` functions during startup. Duplicates would be a developer error caught quickly during testing.

**Recommendation:** Add a panic on duplicate registration in each registry. One-line defensive check:

```go
// In each Register* function:
func RegisterGenerator(gen MapGenerator) {
    if _, exists := generators[gen.Name()]; exists {
        panic(fmt.Sprintf("duplicate map generator registration: %s", gen.Name()))
    }
    generators[gen.Name()] = gen
}
```

Using `panic` (not `fmt.Printf`) because duplicate registration is always a bug, and `init()` panics are caught immediately at startup.

---

## Import Boundary Verification

All claims in `DEPENDENCY_BOUNDARIES.md` verified against actual source code:

| Boundary Claim | Status | Evidence |
|----------------|--------|----------|
| No GUI package imports `mind/encounter` | **Verified** | GUI uses `combat.EncounterCallbacks` interface instead |
| No GUI package imports `mind/behavior` | **Verified** | Zero imports found |
| `combatservices` does not import `mind/ai` or `mind/behavior` | **Verified** | Defines interfaces they implement; `ai_interfaces.go` imports only `coords` and `ecs` |
| `mind/combatlifecycle` imports only `common` and `tactical/combat` | **Verified** | Minimal dependency footprint confirmed |
| `gui/guicombat` imports `mind/ai` for wiring only | **Verified** | Used exclusively in `combatmode.go:88-95` (`SetupCombatAI` call in `Initialize`) — single call site, justified wiring |
| Wiring layer is centralized in `setup.go` + `moderegistry.go` | **Verified** | No reach-through anti-patterns found |

---

## Patterns That Scale Well

These patterns are worth preserving and extending as the codebase grows:

### 1. Single-Method Interfaces for Pipeline Steps
`CombatStarter`, `CombatTransitioner`, `CombatResolver`, `CombatCleaner` — each with exactly one method. New combat contexts (e.g., arena mode, siege mode) can be added by implementing a single method. This is the strongest pattern in the codebase.

### 2. Optional Interfaces via Type Assertion
`OverlayRenderer`, `ActionMapProvider`, `CombatStartRollback`, `Validatable` — modes and starters opt-in to capabilities without bloating required interfaces. To add a new optional capability to the UI mode system, define a 1-method interface and check with `if x, ok := mode.(NewCapability); ok { ... }`.

### 3. `BaseBehavior` Embedding for Large Interfaces
When an interface must have many methods (like `ArtifactBehavior` with 7), providing a `Base*` struct with no-op defaults allows implementations to override only what they need. This pattern should be used for any future interface with 4+ methods where most implementations only need a subset.

### 4. Structural Typing for Cross-Layer Boundaries
`EncounterService` satisfying `CombatTransitioner` and `EncounterCallbacks` without importing the defining package. This avoids circular dependencies without introducing a shared "interfaces" package. New cross-layer boundaries should follow this pattern.

### 5. Callback Fields for Single-Listener Hooks
`PostCombatCallback`, `SaveGameCallback`, `LoadGameCallback` — bare function fields for optional, single-listener hooks. Use this pattern when: (a) there's exactly one listener, (b) the hook is optional (nil-checked), (c) a full interface would be overkill.

---

## Recommended Updates to DEPENDENCY_BOUNDARIES.md

1. **Add a "Known Coupling Points" section** documenting that `CombatService` is passed as a concrete type through `CombatModeDeps` and `ArtifactActivationDeps`, with rationale for why this is accepted and when to revisit (Issue #1).

2. **Update the `CombatStarter` implementors table** to include `MultiFactionCombatStarter` (mind/encounter/starters.go), added in recent work.
