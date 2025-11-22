# Overworld vs Battle Map GUI Architecture Approaches

**Status:** Design Document
**Date:** 2025-11-22
**Context:** Separating GUI modes between Overworld (strategic layer) and Battle Map (tactical combat)

---

## Current State Analysis

### Existing GUI Mode System
- **Architecture:** UIMode interface + UIModeManager
- **Current Modes (8 total):**
  - `ExplorationMode` - Dungeon exploration (Battle Map context)
  - `CombatMode` - Turn-based squad combat (Battle Map context)
  - `SquadManagementMode` - Managing squad composition (Overworld context)
  - `SquadBuilderMode` - Creating new squads (Overworld context)
  - `SquadDeploymentMode` - Deploying squads to map (Bridge between contexts)
  - `FormationEditorMode` - Editing squad formations (Overworld context)
  - `InventoryMode` - Managing items (Both contexts)
  - `InfoMode` - Inspecting entities (Battle Map context)

### Key Observation
**Some modes already belong to different contexts** - we need architecture to **enforce context boundaries** and prevent invalid transitions (e.g., can't access ExplorationMode from Overworld).

---

## Approach 1: Context-Aware Mode Manager (Recommended)

### Overview
Extend existing `UIModeManager` with **context awareness**. Modes declare which context(s) they belong to, and manager enforces valid transitions.

### Architecture
```go
// Context enumeration
type UIContext int
const (
    ContextOverworld UIContext = iota
    ContextBattleMap
    ContextBoth  // For modes valid in either context (Inventory, etc.)
)

// Extended UIMode interface
type UIMode interface {
    // ... existing methods ...
    GetContext() UIContext  // NEW: Declare mode's context
}

// Extended UIModeManager
type UIModeManager struct {
    currentMode       UIMode
    currentContext    UIContext  // NEW: Track active context
    modes             map[string]UIMode
    pendingTransition *ModeTransition
    // ... existing fields ...
}

// Context transition method
func (umm *UIModeManager) SwitchContext(newContext UIContext, initialMode string) error {
    // Validate initialMode belongs to newContext
    // Exit current mode
    // Update currentContext
    // Enter initialMode
}
```

### Mode Classification
**Overworld Modes:**
- `OverworldMapMode` (NEW) - Strategic world map view
- `SquadManagementMode` - Manage squads between missions
- `SquadBuilderMode` - Create/recruit squads
- `FormationEditorMode` - Edit formations
- `TownMode` (NEW) - Visit towns, shops, quests
- `CampaignMapMode` (NEW) - Campaign progression

**Battle Map Modes:**
- `ExplorationMode` - Dungeon exploration
- `CombatMode` - Turn-based combat
- `InfoMode` - Inspect entities on battlefield

**Both Contexts:**
- `InventoryMode` - Manage items anywhere
- `OptionsMode` (NEW) - Settings menu

### Pros
✅ **Minimal architectural change** - extends existing system
✅ **Type-safe context enforcement** - compiler catches invalid transitions
✅ **Clear mode ownership** - each mode declares its context
✅ **Smooth transitions** - `SwitchContext()` handles cleanup
✅ **Backward compatible** - existing modes add one method

### Cons
⚠️ Requires updating all 8 existing modes to implement `GetContext()`
⚠️ Context switching logic in one manager (could grow complex)

### Implementation Estimate
**4-6 hours**
1. Add `UIContext` enum and `GetContext()` to interface (30 min)
2. Update all 8 existing modes (1 hour)
3. Add `currentContext` tracking to manager (1 hour)
4. Implement `SwitchContext()` with validation (2 hours)
5. Add context transition logging/debugging (30 min)
6. Test all mode transitions (1 hour)

---

## Approach 2: Separate Mode Managers

### Overview
Create **two independent UIModeManagers** - one for Overworld, one for Battle Map. A top-level coordinator switches between them.

### Architecture
```go
// Top-level coordinator
type GameModeCoordinator struct {
    overworldManager *UIModeManager
    battleMapManager *UIModeManager
    activeManager    *UIModeManager  // Points to current context
    overworldState   *OverworldState  // Persistent overworld data
    battleMapState   *BattleMapState  // Persistent battle data
}

func (gmc *GameModeCoordinator) EnterBattleMap(mapID string) {
    // Save overworld state
    // Initialize battle map
    // Switch to battleMapManager
}

func (gmc *GameModeCoordinator) ReturnToOverworld() {
    // Save battle results
    // Restore overworld state
    // Switch to overworldManager
}
```

### Pros
✅ **Complete separation** - no risk of cross-context transitions
✅ **Independent evolution** - change one manager without affecting other
✅ **Clear ownership** - each manager owns its modes exclusively
✅ **Easier testing** - test contexts independently

### Cons
⚠️ **Duplicate infrastructure** - two managers with similar code
⚠️ **State synchronization** - need to pass data between contexts
⚠️ **More complex coordinator** - top-level logic to switch managers
⚠️ **Harder to share modes** - InventoryMode would need special handling

### Implementation Estimate
**6-8 hours**
1. Create `GameModeCoordinator` (2 hours)
2. Separate modes into two managers (2 hours)
3. Implement state persistence/restoration (2 hours)
4. Handle shared modes (InventoryMode) (1 hour)
5. Test context switching (1-2 hours)

---

## Approach 3: Mode Hierarchies with Composition

### Overview
Create **base mode types** for each context. Modes inherit behavior from their context base.

### Architecture
```go
// Base mode for overworld context
type OverworldBaseMode struct {
    gui.BaseMode
    OverworldData *OverworldState
}

// Base mode for battle map context
type BattleMapBaseMode struct {
    gui.BaseMode
    BattleData *BattleMapState
}

// Example modes
type OverworldMapMode struct {
    OverworldBaseMode  // Inherits overworld context
}

type ExplorationMode struct {
    BattleMapBaseMode  // Inherits battle map context
}

// Manager validates transitions by checking mode type
func (umm *UIModeManager) SetMode(modeName string) error {
    newMode := umm.modes[modeName]

    // Type assertion to check context compatibility
    switch cm := umm.currentMode.(type) {
    case *OverworldBaseMode:
        if _, ok := newMode.(*BattleMapBaseMode); ok {
            return errors.New("cannot transition from Overworld to BattleMap mode directly")
        }
    case *BattleMapBaseMode:
        if _, ok := newMode.(*OverworldBaseMode); ok {
            return errors.New("cannot transition from BattleMap to Overworld mode directly")
        }
    }
}
```

### Pros
✅ **Type safety through inheritance** - modes naturally grouped by context
✅ **Shared context data** - each base mode provides context-specific state
✅ **Runtime validation** - manager checks type compatibility
✅ **Context-specific utilities** - add helper methods to base modes

### Cons
⚠️ **Go's limited inheritance** - composition is more verbose than inheritance
⚠️ **Type assertion overhead** - runtime checks instead of compile-time
⚠️ **Refactor existing modes** - all modes need to change base type
⚠️ **Shared modes awkward** - InventoryMode can't embed both bases

### Implementation Estimate
**5-7 hours**
1. Create `OverworldBaseMode` and `BattleMapBaseMode` (2 hours)
2. Refactor existing modes to use new bases (2 hours)
3. Add type validation to manager transitions (1 hour)
4. Handle shared modes with interfaces (1-2 hours)
5. Test and validate (1 hour)

---

## Approach 4: Strategy Pattern with Context Switching

### Overview
Treat Overworld and Battle Map as **different game states** with their own mode registries. Use strategy pattern to swap active state.

### Architecture
```go
// Game state interface
type GameState interface {
    Initialize(ctx *UIContext) error
    EnterState(fromState GameState) error
    ExitState(toState GameState) error
    Update(deltaTime float64) error
    Render(screen *ebiten.Image)
    GetModeManager() *UIModeManager  // Each state has its own manager
}

// Concrete implementations
type OverworldState struct {
    modeManager *UIModeManager
    worldMap    *WorldMap
    // ... overworld-specific data
}

type BattleMapState struct {
    modeManager *UIModeManager
    currentMap  *GameMap
    // ... battle-specific data
}

// Top-level coordinator
type GameStateCoordinator struct {
    currentState    GameState
    overworldState  *OverworldState
    battleMapState  *BattleMapState
}

func (gsc *GameStateCoordinator) TransitionToBattleMap(mapID string) {
    gsc.currentState.ExitState(gsc.battleMapState)
    gsc.currentState = gsc.battleMapState
    gsc.battleMapState.EnterState(gsc.overworldState)
}
```

### Pros
✅ **Clean separation** - each state is self-contained
✅ **Flexible expansion** - easy to add new states (MenuState, CutsceneState)
✅ **State persistence** - easy to save/load entire state
✅ **Clear lifecycle** - Enter/Exit hooks for state transitions

### Cons
⚠️ **Most complex architecture** - introduces new abstraction layer
⚠️ **Significant refactoring** - wraps existing manager in state object
⚠️ **Over-engineering risk** - may be overkill for two contexts
⚠️ **State data duplication** - both states need access to ECS, etc.

### Implementation Estimate
**8-10 hours**
1. Design `GameState` interface (1 hour)
2. Implement `OverworldState` and `BattleMapState` (3 hours)
3. Create `GameStateCoordinator` (2 hours)
4. Refactor main game loop to use coordinator (2 hours)
5. Implement state transitions (1 hour)
6. Test and debug (1-2 hours)

---

## Comparison Matrix

| Criteria | Approach 1: Context-Aware | Approach 2: Separate Managers | Approach 3: Hierarchies | Approach 4: Strategy |
|----------|---------------------------|------------------------------|------------------------|---------------------|
| **Complexity** | Low | Medium | Medium | High |
| **Refactoring Scope** | Small | Medium | Large | Large |
| **Type Safety** | High | High | Medium | High |
| **Flexibility** | Medium | Low | Medium | High |
| **Implementation Time** | 4-6h | 6-8h | 5-7h | 8-10h |
| **Maintenance** | Easy | Medium | Medium | Complex |
| **Future Expansion** | Good | Poor | Good | Excellent |
| **Shared Modes** | Easy | Hard | Hard | Medium |

---

## Recommendation: Approach 1 (Context-Aware Mode Manager)

### Justification
1. **Least invasive** - extends existing architecture with minimal changes
2. **Clear semantics** - `mode.GetContext()` is self-documenting
3. **Fastest implementation** - 4-6 hours vs 8-10 hours for strategy pattern
4. **Easy maintenance** - single manager, clear rules
5. **Handles shared modes** - `ContextBoth` allows InventoryMode to work everywhere

### Migration Path
1. **Phase 1:** Add context support (2 hours)
   - Define `UIContext` enum
   - Add `GetContext()` to interface
   - Update existing modes

2. **Phase 2:** Enforce context boundaries (2 hours)
   - Add `currentContext` tracking
   - Implement `SwitchContext()` method
   - Validate transitions in `SetMode()`

3. **Phase 3:** Create Overworld modes (Future work)
   - `OverworldMapMode` - Strategic world view
   - `TownMode` - Visit towns
   - `CampaignMapMode` - Campaign progression

### Example Usage
```go
// Initialize with battle map context
manager.SwitchContext(ContextBattleMap, "exploration")

// User clicks "Return to Base" button
manager.SwitchContext(ContextOverworld, "overworld_map")

// Try invalid transition (caught at runtime)
manager.SetMode("exploration")  // ERROR: exploration is ContextBattleMap, currently in ContextOverworld
```

---

## Alternative Recommendation: Approach 4 (If Building Long-Term)

If planning significant expansion (cutscenes, menus, multiple battle types), **Approach 4 (Strategy Pattern)** provides best long-term flexibility:

- Easy to add `MenuState`, `CutsceneState`, `MultiplayerState`
- Clean save/load system (serialize entire state)
- Clear separation of concerns
- Industry-standard pattern for game states

**Trade-off:** Higher upfront cost (8-10 hours) but cleaner architecture for complex games.

---

## Next Steps

1. **Decide on approach** (User choice based on project timeline)
2. **Create implementation plan** with tasks
3. **Design Overworld modes** (what functionality do they provide?)
4. **Plan state persistence** (how does data flow between contexts?)
5. **Implement context switching** with validation
6. **Create transition UI** (loading screens, fade transitions)

---

## Questions to Answer

1. **What triggers context switches?**
   - Mission start → Overworld → BattleMap
   - Mission complete → BattleMap → Overworld
   - Town visit → Overworld → Town mode

2. **What data persists between contexts?**
   - Squad composition
   - Inventory
   - Campaign progress
   - Resources (gold, materials)

3. **What data is context-specific?**
   - Current dungeon state (BattleMap)
   - Overworld map exploration (Overworld)
   - Turn order (BattleMap only)

4. **How do shared systems work?**
   - Inventory accessible from both contexts
   - Squad management available in Overworld, read-only in BattleMap?
   - Formation editing Overworld-only or both?

---

## References

- `gui/core/uimode.go` - UIMode interface definition
- `gui/core/modemanager.go` - Current UIModeManager implementation
- `gui/basemode.go` - BaseMode with common infrastructure
- `gui/guimodes/explorationmode.go` - Example Battle Map mode
- `gui/guisquads/squadmanagementmode.go` - Example Overworld mode
