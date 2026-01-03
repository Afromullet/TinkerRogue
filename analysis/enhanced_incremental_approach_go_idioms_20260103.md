# GUI Refactoring: Enhanced Incremental Approach with Go Idioms

**Generated:** 2026-01-03
**Author:** Go Standards Reviewer
**Target:** GUI-Game State Coupling Refactoring
**Source Documents:**
- `analysis/go_idiomatics_gui_refactoring_analysis_20260103.md`
- Current GUI implementation analysis

---

## EXECUTIVE SUMMARY

### The Enhanced Approach

This document presents **Enhanced Approach 3**, a comprehensive refactoring strategy that combines:
1. **Incremental Facade Pattern** (low-risk baseline from original Approach 3)
2. **Consumer-Defined Interfaces** (Go idiom from stdlib patterns)
3. **Small, Focused Interfaces** (Go proverb: "bigger interface, weaker abstraction")
4. **Testability Without Complexity** (no event buses, commands, or DTOs)

### Why Enhanced Approach 3 Is Superior

| Feature | Original Approach 3 | Approach 1 | Approach 2 | **Enhanced Approach 3** |
|---------|---------------------|------------|------------|-------------------------|
| **Incremental** | ✅ Yes | ⚠️ Partial | ❌ No (big rewrite) | ✅ Yes |
| **Go-Idiomatic** | ⚠️ Partial | ❌ No (provider interfaces) | ❌ No (commands/events) | ✅ Yes |
| **Testable** | ❌ No (concrete deps) | ✅ Yes | ✅ Yes | ✅ Yes |
| **Simple** | ✅ Very | ⚠️ Medium | ❌ Complex | ✅ Simple-Medium |
| **Risk Level** | Low | Medium | High | Low |
| **Effort (days)** | 4 | 3.5 | 5.5 | 4-5 |

### What Makes This "Go-Idiomatic"?

1. **Consumer-Defined Interfaces**: Interfaces live in `gui/guicombat/`, not in `tactical/` or `gui/interfaces/`
2. **Small Interfaces**: 2-3 methods each, matching `io.Reader` pattern (not 12+ method monsters)
3. **Accept Interfaces, Return Structs**: Constructor pattern from `net/http`, `database/sql`
4. **No DTOs**: Return concrete types like `combat.AttackResult`, not wrapper objects
5. **No Command Objects**: Direct function calls, not Java-style command pattern
6. **No Event Bus**: Synchronous calls or channels, not publish/subscribe indirection
7. **Explicit Dependencies**: Constructor injection, not service locator

### Key Differentiators from Original Approach 3

Original Approach 3 created a facade but **stopped short** of full decoupling:

```go
// Original Approach 3 - Still concrete dependency
type CombatMode struct {
    facade *gamefacade.GameFacade // Concrete!
}
```

Enhanced Approach 3 **completes the decoupling**:

```go
// Enhanced Approach 3 - Interface dependency
type CombatMode struct {
    turns    TurnManager      // Interface defined in guicombat
    combat   Attacker         // Interface defined in guicombat
    movement MovementProvider // Interface defined in guicombat
}
```

**Critical Difference**: GUI has ZERO compile-time dependency on facade or tactical packages.

---

## CORE GO PRINCIPLES APPLIED

### 1. Consumer-Defined Interfaces (Go FAQ)

**Official Go FAQ Quote:**
> "Go interfaces generally belong in the package that uses values of the interface type, not the package that implements those values."

**What This Means:**

```go
// ❌ WRONG - Provider-side interface (Approach 1 anti-pattern)
// File: tactical/interfaces/combat_interface.go
package interfaces

type ICombatController interface { ... }

// File: tactical/combatservices/combat_service.go
var _ interfaces.ICombatController = (*CombatService)(nil) // Creates dependency!

// ✅ CORRECT - Consumer-side interface (Go idiom)
// File: gui/guicombat/combat_interfaces.go
package guicombat

type TurnManager interface {
    GetCurrentFaction() ecs.EntityID
    EndTurn() error
}

// File: tactical/combatservices/combat_service.go
func (cs *CombatService) GetCurrentFaction() ecs.EntityID { ... }
func (cs *CombatService) EndTurn() error { ... }
// NO import of gui package, NO awareness of interface
```

**Why This Matters:**
- **Dependency Direction**: Tactical layer has ZERO knowledge of GUI
- **Flexibility**: Different GUI modes can define different interfaces for same service
- **Testing**: Each GUI package mocks only what it needs
- **Evolution**: Interfaces evolve with consumers, not providers

### 2. Small Interfaces (Go Proverb)

**Go Proverb:**
> "The bigger the interface, the weaker the abstraction"

**Standard Library Examples:**

```go
// io.Reader - Single method
type Reader interface {
    Read(p []byte) (n int, err error)
}

// http.Handler - Single method
type Handler interface {
    ServeHTTP(ResponseWriter, *Request)
}

// io.Writer - Single method
type Writer interface {
    Write(p []byte) (n int, err error)
}
```

**Application to TinkerRogue:**

```go
// ❌ TOO BIG (12 methods - violates Go proverb)
type ICombatController interface {
    InitializeCombat(factionIDs []ecs.EntityID) error
    CheckVictoryCondition() *VictoryInfo
    GetCurrentFaction() ecs.EntityID
    GetCurrentRound() int
    EndTurn() error
    ExecuteAttack(attackerID, defenderID ecs.EntityID) *AttackResult
    GetValidMovementTiles(squadID ecs.EntityID) []coords.LogicalPosition
    MoveSquad(squadID ecs.EntityID, newPos coords.LogicalPosition) *MoveResult
    GetAliveSquadsInFaction(factionID ecs.EntityID) []ecs.EntityID
    GetSquadInfo(squadID ecs.EntityID) *SquadInfo
    IsSquadPlayerControlled(squadID ecs.EntityID) bool
    ExecuteAITurn(factionID ecs.EntityID) *AITurnResult
}

// ✅ SMALL, FOCUSED (2-3 methods each - matches Go stdlib)
type TurnManager interface {
    GetCurrentFaction() ecs.EntityID
    GetCurrentRound() int
    EndTurn() error
}

type Attacker interface {
    ExecuteAttack(attackerID, defenderID ecs.EntityID) combat.AttackResult
}

type MovementProvider interface {
    GetValidMovementTiles(squadID ecs.EntityID) []coords.LogicalPosition
    MoveSquad(squadID ecs.EntityID, pos coords.LogicalPosition) error
}

type SquadQuerier interface {
    GetAliveSquadsInFaction(factionID ecs.EntityID) []ecs.EntityID
}

type VictoryChecker interface {
    CheckVictoryCondition() *combatservices.VictoryCheckResult
}
```

**Benefits of Small Interfaces:**
- **Easy Mocking**: Mock only 2-3 methods, not 12+
- **Clear Purpose**: Each interface has single responsibility
- **Flexible Composition**: Mix and match interfaces as needed
- **Interface Segregation**: Consumers depend only on what they use

### 3. Accept Interfaces, Return Structs

**Pattern from `io.Copy`:**

```go
// stdlib: Accept interfaces, return concrete type
func Copy(dst Writer, src Reader) (written int64, err error)
```

**Application to TinkerRogue:**

```go
// ✅ Constructor accepts interfaces
func NewCombatMode(
    turns TurnManager,         // Interface parameter
    combat Attacker,           // Interface parameter
    movement MovementProvider, // Interface parameter
) *CombatMode {              // Concrete return type
    return &CombatMode{
        turns:    turns,
        combat:   combat,
        movement: movement,
    }
}

// ❌ WRONG - Return interface (Approach 1 anti-pattern)
func NewCombatMode(config CombatConfig) ICombatMode { // NO!
    // ...
}
```

**Why:**
- **Flexibility at Call Site**: Caller can pass any implementation (real, mock, test double)
- **Clear Ownership**: Caller knows it owns `*CombatMode` concrete type
- **No Interface Casting**: No need for type assertions

### 4. No DTOs (Go Proverb)

**Go Proverb:**
> "A little copying is better than a little dependency"

**What stdlib does:**

```go
// net/http doesn't wrap http.Request in DTO
func (s *Server) ServeHTTP(w ResponseWriter, r *Request) {
    // Passes Request directly, no RequestDTO wrapper
}

// database/sql returns concrete Rows, not RowsDTO
func (db *DB) Query(query string, args ...interface{}) (*Rows, error)
```

**Application to TinkerRogue:**

```go
// ❌ DTO Wrapper (Approach 1 adds unnecessary layer)
type AttackResultDTO struct {
    Success      bool
    Damage       int
    TargetKilled bool
    // ... copy all fields from combat.AttackResult
}

func (cf *CombatFacade) ExecuteAttack(...) *AttackResultDTO {
    result := cf.service.ExecuteAttack(...)
    // Manual conversion - wasted effort!
    return &AttackResultDTO{
        Success:      result.Success,
        Damage:       result.Damage,
        TargetKilled: result.TargetKilled,
    }
}

// ✅ Direct Return (Go way - return concrete type)
func (cf *CombatFacade) ExecuteAttack(
    attackerID, defenderID ecs.EntityID,
) combat.AttackResult {
    return cf.service.CombatActSystem.ExecuteAttackAction(attackerID, defenderID)
}

// GUI extracts what it needs
result := facade.ExecuteAttack(attacker, defender)
if result.Success {
    cm.logManager.AddLog(fmt.Sprintf("%d damage!", result.Damage))
}
```

**Why No DTOs:**
- **Simpler Code**: No conversion boilerplate
- **Performance**: No allocation for wrapper objects
- **Type Safety**: Compiler catches field changes
- **DRY**: Single source of truth for data structure

### 5. No Command Objects (Go Simplicity)

**Go doesn't use command pattern** - check stdlib:

```go
// ❌ Command Pattern (Java/C# style - NOT in Go stdlib)
type Command interface {
    Execute() error
    Undo() error
    Redo() error
}

type AttackCommand struct {
    attackerID ecs.EntityID
    defenderID ecs.EntityID
    service    *CombatService
}

func (ac *AttackCommand) Execute() error {
    return ac.service.ExecuteAttack(ac.attackerID, ac.defenderID)
}

mediator.Execute(NewAttackCommand(a, d, service)) // Indirection!

// ✅ Go Way - Just call the function
err := combatService.ExecuteAttack(attacker, defender)
```

**When You MIGHT Use Commands in Go:**
- Building an undo/redo system (rare in games, common in editors)
- Implementing macro recording (automation tools)
- Command-line argument parsing (see `cobra`, `flag` packages)

**For TinkerRogue:** You don't need undo/redo in combat. Direct function calls are simpler.

### 6. No Event Bus (Go Channels Instead)

**Go doesn't use event buses** - check stdlib:

```go
// ❌ Event Bus (NOT in Go stdlib)
type EventBus struct {
    subscribers map[string][]EventHandler
}

func (eb *EventBus) Publish(eventType string, data interface{})
func (eb *EventBus) Subscribe(eventType string, handler EventHandler)

// ✅ Go Way Option 1 - Synchronous callback
type CombatMode struct {
    onAttackComplete func(result combat.AttackResult)
}

result := service.ExecuteAttack(a, d)
if cm.onAttackComplete != nil {
    cm.onAttackComplete(result)
}

// ✅ Go Way Option 2 - Channel (if async needed)
type CombatMode struct {
    events chan CombatEvent
}

go func() {
    result := service.ExecuteAttack(a, d)
    cm.events <- CombatEvent{Type: "attack", Result: result}
}()
```

**For TinkerRogue:** Synchronous calls are simpler. Turn-based combat doesn't need async.

### 7. Explicit Dependencies (No Service Locator)

**Anti-Pattern: Service Locator**

```go
// ❌ Service Locator (Approach 1 anti-pattern)
type GameSession struct {
    combatService  *CombatService
    squadService   *SquadService
    // ... 10 more services
}

func (gs *GameSession) CombatController() ICombatController {
    return gs.combatService
}

// Hides dependency - unclear what CombatMode needs
cm := NewCombatMode(gameSession)
```

**Go Way: Constructor Injection**

```go
// ✅ Explicit Dependencies
func NewCombatMode(
    turns TurnManager,
    combat Attacker,
    movement MovementProvider,
) *CombatMode {
    return &CombatMode{
        turns:    turns,
        combat:   combat,
        movement: movement,
    }
}

// Clear what CombatMode depends on
facade := gamefacade.NewCombatFacade(em)
cm := guicombat.NewCombatMode(facade, facade, facade)
```

---

## THREE-PHASE IMPLEMENTATION

### Phase 1: Extract Facade (Baseline from Original Approach 3)

**Goal:** Create single point of contact for GUI, move all tactical operations to facade.

**Why Start Here:**
- **Low Risk**: Facade is simple wrapper, no behavior changes
- **Immediate Value**: Reduces import surface area
- **Foundation**: Prepares for interface extraction

#### Step 1.1: Create Combat Facade

```go
// game_main/gamefacade/combat_facade.go
package gamefacade

import (
    "game_main/common"
    "game_main/tactical/combatservices"
    "game_main/tactical/combat"
    "game_main/tactical/squads"
    "game_main/world/coords"
    "github.com/bytearena/ecs"
)

// CombatFacade provides simplified access to combat operations.
// This is a concrete struct that will later satisfy multiple small interfaces.
type CombatFacade struct {
    service *combatservices.CombatService
    em      *common.EntityManager
}

func NewCombatFacade(em *common.EntityManager) *CombatFacade {
    return &CombatFacade{
        service: combatservices.NewCombatService(em),
        em:      em,
    }
}

// Turn Management Operations
// Group related methods - these will become TurnManager interface

func (cf *CombatFacade) GetCurrentFaction() ecs.EntityID {
    return cf.service.TurnManager.GetCurrentFaction()
}

func (cf *CombatFacade) GetCurrentRound() int {
    return cf.service.TurnManager.GetCurrentRound()
}

func (cf *CombatFacade) EndTurn() error {
    return cf.service.TurnManager.EndTurn()
}

// Combat Action Operations
// Group related methods - these will become Attacker interface

func (cf *CombatFacade) ExecuteAttack(
    attackerID, defenderID ecs.EntityID,
) combat.AttackResult {
    // Return concrete type, no DTO wrapper
    return cf.service.CombatActSystem.ExecuteAttackAction(attackerID, defenderID)
}

// Movement Operations
// Group related methods - these will become MovementProvider interface

func (cf *CombatFacade) GetValidMovementTiles(
    squadID ecs.EntityID,
) []coords.LogicalPosition {
    tiles := cf.service.MovementSystem.GetValidMovementTiles(squadID)
    if tiles == nil {
        return []coords.LogicalPosition{} // Never return nil slice
    }
    return tiles
}

func (cf *CombatFacade) MoveSquad(
    squadID ecs.EntityID,
    pos coords.LogicalPosition,
) error {
    return cf.service.MovementSystem.MoveSquad(squadID, pos)
}

// Squad Query Operations
// Group related methods - these will become SquadQuerier interface

func (cf *CombatFacade) GetAliveSquadsInFaction(
    factionID ecs.EntityID,
) []ecs.EntityID {
    return cf.service.GetAliveSquadsInFaction(factionID)
}

func (cf *CombatFacade) GetSquadAtPosition(
    pos coords.LogicalPosition,
) (ecs.EntityID, bool) {
    // Returns (squadID, found)
    squads := common.GlobalPositionSystem.GetEntitiesAtPosition(pos)
    if len(squads) == 0 {
        return 0, false
    }

    // Find first squad entity
    for _, id := range squads {
        entity := cf.em.GetEntityByID(id)
        if entity != nil && entity.HasTag(squads.SquadTag) {
            return id, true
        }
    }
    return 0, false
}

// Victory Checking Operations
// Group related methods - these will become VictoryChecker interface

func (cf *CombatFacade) CheckVictoryCondition() *combatservices.VictoryCheckResult {
    return cf.service.CheckVictoryCondition()
}

// Combat Initialization Operations
// Group related methods - these will become CombatInitializer interface

func (cf *CombatFacade) InitializeCombat(factionIDs []ecs.EntityID) error {
    return cf.service.InitializeCombat(factionIDs)
}

func (cf *CombatFacade) EndCombat() {
    cf.service.EndCombat()
}
```

**Key Points:**
- **Methods Grouped by Responsibility**: Comments indicate future interface boundaries
- **No DTOs**: Returns `combat.AttackResult` directly, not wrapper
- **Concrete Type**: This is a struct, not interface (follows "return structs" idiom)
- **Simple Delegation**: Just wraps service calls, no business logic

#### Step 1.2: Create Squad Facade

```go
// game_main/gamefacade/squad_facade.go
package gamefacade

import (
    "game_main/common"
    "game_main/tactical/squads"
    "game_main/tactical/squadservices"
    "game_main/world/coords"
    "github.com/bytearena/ecs"
)

// SquadFacade provides simplified access to squad operations.
type SquadFacade struct {
    deploymentService *squadservices.SquadDeploymentService
    builderService    *squadservices.SquadBuilderService
    em                *common.EntityManager
}

func NewSquadFacade(em *common.EntityManager) *SquadFacade {
    return &SquadFacade{
        deploymentService: squadservices.NewSquadDeploymentService(em),
        builderService:    squadservices.NewSquadBuilderService(em),
        em:                em,
    }
}

// Squad Building Operations
// These will become SquadBuilder interface

func (sf *SquadFacade) CreateSquad(playerID ecs.EntityID, name string) (ecs.EntityID, error) {
    return sf.builderService.CreateSquad(playerID, name)
}

func (sf *SquadFacade) AddUnitToSquad(squadID, unitID ecs.EntityID) error {
    return sf.builderService.AddUnitToSquad(squadID, unitID)
}

func (sf *SquadFacade) RemoveUnitFromSquad(squadID, unitID ecs.EntityID) error {
    return sf.builderService.RemoveUnitFromSquad(squadID, unitID)
}

func (sf *SquadFacade) ValidateSquad(squadID ecs.EntityID) error {
    return sf.builderService.ValidateSquad(squadID)
}

// Squad Deployment Operations
// These will become SquadDeployer interface

func (sf *SquadFacade) CanDeployAtPosition(
    squadID ecs.EntityID,
    pos coords.LogicalPosition,
) bool {
    return sf.deploymentService.CanDeploySquad(squadID, pos)
}

func (sf *SquadFacade) DeploySquad(
    squadID ecs.EntityID,
    pos coords.LogicalPosition,
) error {
    return sf.deploymentService.DeploySquad(squadID, pos)
}

func (sf *SquadFacade) GetDeploymentZones() []coords.LogicalPosition {
    return sf.deploymentService.GetValidDeploymentZones()
}

// Squad Query Operations
// These will become SquadQuerier interface

func (sf *SquadFacade) GetPlayerSquads(playerID ecs.EntityID) []ecs.EntityID {
    return squads.GetPlayerSquads(sf.em, playerID)
}

func (sf *SquadFacade) GetSquadInfo(squadID ecs.EntityID) (*squads.SquadData, error) {
    entity := sf.em.GetEntityByID(squadID)
    if entity == nil {
        return nil, fmt.Errorf("squad %d not found", squadID)
    }

    data := common.GetComponentType[*squads.SquadData](entity, squads.SquadComponent)
    if data == nil {
        return nil, fmt.Errorf("squad %d has no SquadData", squadID)
    }

    return data, nil
}
```

#### Step 1.3: Update CombatMode to Use Facade

```go
// gui/guicombat/combatmode.go
package guicombat

import (
    "fmt"
    "game_main/common"
    "game_main/gui"
    "game_main/gui/core"
    "game_main/game_main/gamefacade" // NEW: Import facade, not tactical
    // REMOVED: "game_main/tactical/combatservices"
    // REMOVED: "game_main/tactical/combat"
    // REMOVED: "game_main/tactical/squads"

    "github.com/bytearena/ecs"
)

type CombatMode struct {
    gui.BaseMode

    // NEW: Single facade instead of multiple service imports
    facade *gamefacade.CombatFacade

    // ... rest of fields unchanged
}

func NewCombatMode(modeManager *core.UIModeManager) *CombatMode {
    cm := &CombatMode{
        logManager: NewCombatLogManager(),
    }
    cm.SetModeName("combat")
    cm.SetReturnMode("exploration")
    cm.ModeManager = modeManager
    return cm
}

func (cm *CombatMode) Initialize(ctx *core.UIContext) error {
    // Create facade (replaces combatService creation)
    cm.facade = gamefacade.NewCombatFacade(ctx.ECSManager)

    // ... rest of initialization unchanged
}

// Update methods to use facade instead of direct service calls
func (cm *CombatMode) handleEndTurn() {
    err := cm.facade.EndTurn() // Was: cm.combatService.TurnManager.EndTurn()
    if err != nil {
        cm.logManager.AddLog("Error ending turn: " + err.Error())
        return
    }

    currentFactionID := cm.facade.GetCurrentFaction()
    round := cm.facade.GetCurrentRound()
    // ... rest unchanged
}

func (cm *CombatMode) handleAttackAction(attackerID, defenderID ecs.EntityID) {
    result := cm.facade.ExecuteAttack(attackerID, defenderID)
    if result.Success {
        msg := fmt.Sprintf("%d damage dealt!", result.Damage)
        cm.logManager.AddLog(msg)
    } else {
        cm.logManager.AddLog("Attack failed: " + result.ErrorReason)
    }
}
```

**Phase 1 Complete When:**
- ✅ Facades created in `game_main/gamefacade/`
- ✅ CombatMode uses facade, no tactical imports
- ✅ SquadDeploymentMode uses facade, no tactical imports
- ✅ All tests pass
- ✅ Game runs without regressions

**Validation:**

```bash
# Check for tactical imports in GUI (should be none)
grep -r "game_main/tactical" gui/guicombat/*.go gui/guisquads/*.go

# Should only see facade imports
grep -r "game_main/game_main/gamefacade" gui/guicombat/*.go gui/guisquads/*.go

# Run tests
go test ./gui/guicombat/... ./gui/guisquads/...
```

---

### Phase 2: Identify Natural Interface Boundaries

**Goal:** Analyze facade methods and discover cohesive groupings that will become small interfaces.

**This is the KEY ENHANCEMENT over original Approach 3** - we're not stopping at facade, we're using it as a stepping stone to proper Go interfaces.

#### Step 2.1: Analyze CombatFacade Method Groups

Looking at `combat_facade.go`, we identified these natural boundaries:

| **Interface Name** | **Methods** | **Responsibility** | **Size** |
|--------------------|-------------|-------------------|----------|
| `TurnManager` | `GetCurrentFaction()`, `GetCurrentRound()`, `EndTurn()` | Turn progression | 3 methods ✅ |
| `Attacker` | `ExecuteAttack()` | Combat actions | 1 method ✅ |
| `MovementProvider` | `GetValidMovementTiles()`, `MoveSquad()` | Movement | 2 methods ✅ |
| `SquadQuerier` | `GetAliveSquadsInFaction()`, `GetSquadAtPosition()` | Squad queries | 2 methods ✅ |
| `VictoryChecker` | `CheckVictoryCondition()` | Victory/defeat | 1 method ✅ |
| `CombatInitializer` | `InitializeCombat()`, `EndCombat()` | Combat lifecycle | 2 methods ✅ |

**All interfaces are 1-3 methods - perfect Go size!**

#### Step 2.2: Analyze SquadFacade Method Groups

| **Interface Name** | **Methods** | **Responsibility** | **Size** |
|--------------------|-------------|-------------------|----------|
| `SquadBuilder` | `CreateSquad()`, `AddUnitToSquad()`, `RemoveUnitFromSquad()`, `ValidateSquad()` | Squad composition | 4 methods ⚠️ |
| `SquadDeployer` | `CanDeployAtPosition()`, `DeploySquad()`, `GetDeploymentZones()` | Deployment | 3 methods ✅ |
| `SquadQuerier` | `GetPlayerSquads()`, `GetSquadInfo()` | Squad queries | 2 methods ✅ |

**Note:** `SquadBuilder` is 4 methods - consider splitting if needed:

```go
// Option 1: Keep as 4 methods (acceptable for cohesive operations)
type SquadBuilder interface {
    CreateSquad(playerID ecs.EntityID, name string) (ecs.EntityID, error)
    AddUnitToSquad(squadID, unitID ecs.EntityID) error
    RemoveUnitFromSquad(squadID, unitID ecs.EntityID) error
    ValidateSquad(squadID ecs.EntityID) error
}

// Option 2: Split into smaller interfaces (if needed)
type SquadCreator interface {
    CreateSquad(playerID ecs.EntityID, name string) (ecs.EntityID, error)
}

type SquadCompositionEditor interface {
    AddUnitToSquad(squadID, unitID ecs.EntityID) error
    RemoveUnitFromSquad(squadID, unitID ecs.EntityID) error
}

type SquadValidator interface {
    ValidateSquad(squadID ecs.EntityID) error
}
```

**Recommendation:** Start with Option 1 (4 methods). Only split if you find you're mocking 4 methods when you only need 2.

#### Step 2.3: Map Interfaces to GUI Consumers

**CombatMode needs:**
- `TurnManager` - For turn order display, end turn button
- `Attacker` - For attack actions
- `MovementProvider` - For movement highlighting and execution
- `SquadQuerier` - For squad list display
- `VictoryChecker` - For victory/defeat detection
- `CombatInitializer` - For entering/exiting combat

**SquadDeploymentMode needs:**
- `SquadDeployer` - For placing squads
- `SquadQuerier` - For squad list display

**SquadEditorMode needs:**
- `SquadBuilder` - For editing squad composition
- `SquadQuerier` - For displaying squad info

**Key Insight:** Different modes need different interfaces - this validates our consumer-defined approach!

---

### Phase 3: Extract Consumer-Defined Interfaces

**Goal:** Create small interfaces in GUI packages, update constructors to accept interfaces instead of concrete facades.

**This is where the magic happens** - GUI becomes testable, tactical layer stays decoupled.

#### Step 3.1: Define Combat Interfaces

```go
// gui/guicombat/combat_interfaces.go
package guicombat

import (
    "game_main/tactical/combat"
    "game_main/tactical/combatservices"
    "game_main/world/coords"
    "github.com/bytearena/ecs"
)

// Small, focused interfaces defined by CombatMode (the consumer).
// These represent what CombatMode NEEDS, not what CombatService PROVIDES.
//
// Pattern: "Accept interfaces, return structs" (from io.Copy, http.Handler)
// Pattern: "Bigger interface, weaker abstraction" (Go proverb - keep small)

// TurnManager handles turn progression.
// Satisfied by: gamefacade.CombatFacade
type TurnManager interface {
    GetCurrentFaction() ecs.EntityID
    GetCurrentRound() int
    EndTurn() error
}

// Attacker executes combat actions.
// Satisfied by: gamefacade.CombatFacade
type Attacker interface {
    ExecuteAttack(attackerID, defenderID ecs.EntityID) combat.AttackResult
}

// MovementProvider handles squad movement.
// Satisfied by: gamefacade.CombatFacade
type MovementProvider interface {
    GetValidMovementTiles(squadID ecs.EntityID) []coords.LogicalPosition
    MoveSquad(squadID ecs.EntityID, pos coords.LogicalPosition) error
}

// SquadQuerier provides squad information.
// Satisfied by: gamefacade.CombatFacade
type SquadQuerier interface {
    GetAliveSquadsInFaction(factionID ecs.EntityID) []ecs.EntityID
    GetSquadAtPosition(pos coords.LogicalPosition) (ecs.EntityID, bool)
}

// VictoryChecker determines combat outcome.
// Satisfied by: gamefacade.CombatFacade
type VictoryChecker interface {
    CheckVictoryCondition() *combatservices.VictoryCheckResult
}

// CombatInitializer manages combat lifecycle.
// Satisfied by: gamefacade.CombatFacade
type CombatInitializer interface {
    InitializeCombat(factionIDs []ecs.EntityID) error
    EndCombat()
}
```

**Critical Notes:**

1. **Package Location**: `gui/guicombat/` (consumer), not `tactical/` (provider)
2. **Small Size**: Largest is 3 methods, smallest is 1 method
3. **Concrete Returns**: Returns `combat.AttackResult`, not DTO wrapper
4. **Documentation**: Comments explain what satisfies each interface
5. **No Static Assertions**: No `var _ TurnManager = (*CombatFacade)(nil)` in tactical layer

#### Step 3.2: Define Squad Interfaces

```go
// gui/guisquads/squad_interfaces.go
package guisquads

import (
    "game_main/tactical/squads"
    "game_main/world/coords"
    "github.com/bytearena/ecs"
)

// Small, focused interfaces defined by squad GUI modes (the consumers).

// SquadBuilder manages squad composition.
// Satisfied by: gamefacade.SquadFacade
type SquadBuilder interface {
    CreateSquad(playerID ecs.EntityID, name string) (ecs.EntityID, error)
    AddUnitToSquad(squadID, unitID ecs.EntityID) error
    RemoveUnitFromSquad(squadID, unitID ecs.EntityID) error
    ValidateSquad(squadID ecs.EntityID) error
}

// SquadDeployer handles squad deployment to map.
// Satisfied by: gamefacade.SquadFacade
type SquadDeployer interface {
    CanDeployAtPosition(squadID ecs.EntityID, pos coords.LogicalPosition) bool
    DeploySquad(squadID ecs.EntityID, pos coords.LogicalPosition) error
    GetDeploymentZones() []coords.LogicalPosition
}

// SquadQuerier provides squad information.
// Satisfied by: gamefacade.SquadFacade
type SquadQuerier interface {
    GetPlayerSquads(playerID ecs.EntityID) []ecs.EntityID
    GetSquadInfo(squadID ecs.EntityID) (*squads.SquadData, error)
}
```

#### Step 3.3: Update CombatMode Constructor

```go
// gui/guicombat/combatmode.go
package guicombat

import (
    "fmt"
    "game_main/gui"
    "game_main/gui/core"
    // NO import of gamefacade!
    // NO import of tactical packages!

    "github.com/bytearena/ecs"
)

type CombatMode struct {
    gui.BaseMode

    // NEW: Store interfaces, not concrete facade
    turns      TurnManager
    combat     Attacker
    movement   MovementProvider
    squads     SquadQuerier
    victory    VictoryChecker
    initializer CombatInitializer

    // ... rest of fields unchanged
}

// NewCombatMode accepts interfaces, returns struct (Go idiom)
func NewCombatMode(
    modeManager *core.UIModeManager,
    turns TurnManager,
    combat Attacker,
    movement MovementProvider,
    squads SquadQuerier,
    victory VictoryChecker,
    initializer CombatInitializer,
) *CombatMode {
    cm := &CombatMode{
        logManager:  NewCombatLogManager(),
        turns:       turns,
        combat:      combat,
        movement:    movement,
        squads:      squads,
        victory:     victory,
        initializer: initializer,
    }
    cm.SetModeName("combat")
    cm.SetReturnMode("exploration")
    cm.ModeManager = modeManager
    return cm
}

// Initialize no longer creates services (injected via constructor)
func (cm *CombatMode) Initialize(ctx *core.UIContext) error {
    // Build UI using ModeBuilder (no service creation)
    err := gui.NewModeBuilder(&cm.BaseMode, gui.ModeConfig{
        ModeName:   "combat",
        ReturnMode: "exploration",
        Panels: []gui.PanelSpec{
            {CustomBuild: cm.buildTurnOrderPanel},
            {CustomBuild: cm.buildFactionInfoPanel},
            // ... rest of panels
        },
    }).Build(ctx)

    if err != nil {
        return err
    }

    // ... rest of initialization
    return nil
}

// Methods use injected interfaces
func (cm *CombatMode) handleEndTurn() {
    err := cm.turns.EndTurn() // Uses TurnManager interface
    if err != nil {
        cm.logManager.AddLog("Error ending turn: " + err.Error())
        return
    }

    currentFactionID := cm.turns.GetCurrentFaction()
    round := cm.turns.GetCurrentRound()
    cm.updateTurnDisplay(currentFactionID, round)
}

func (cm *CombatMode) handleAttackAction(attackerID, defenderID ecs.EntityID) {
    result := cm.combat.ExecuteAttack(attackerID, defenderID) // Uses Attacker interface
    if result.Success {
        msg := fmt.Sprintf("%d damage dealt!", result.Damage)
        cm.logManager.AddLog(msg)
    }
}

func (cm *CombatMode) handleMoveAction(squadID ecs.EntityID) {
    tiles := cm.movement.GetValidMovementTiles(squadID) // Uses MovementProvider interface
    cm.highlightTiles(tiles)
}
```

**Key Changes:**
1. **No Facade Import**: CombatMode doesn't import `gamefacade` package
2. **Interface Fields**: Stores 6 small interfaces, not 1 concrete facade
3. **Constructor Injection**: Dependencies passed in constructor, not created in `Initialize()`
4. **Testable**: Can pass mocks to constructor

#### Step 3.4: Update SquadDeploymentMode Constructor

```go
// gui/guisquads/squaddeploymentmode.go
package guisquads

import (
    "game_main/gui"
    "game_main/gui/core"
    // NO import of gamefacade!
    // NO import of tactical/squadservices!

    "github.com/bytearena/ecs"
)

type SquadDeploymentMode struct {
    gui.BaseMode

    // NEW: Store interfaces, not concrete service
    deployer SquadDeployer
    squads   SquadQuerier

    // ... rest of fields unchanged
}

func NewSquadDeploymentMode(
    modeManager *core.UIModeManager,
    deployer SquadDeployer,
    squads SquadQuerier,
) *SquadDeploymentMode {
    mode := &SquadDeploymentMode{
        deployer: deployer,
        squads:   squads,
    }
    mode.SetModeName("squad_deployment")
    mode.SetReturnMode("exploration")
    mode.ModeManager = modeManager
    return mode
}

func (sdm *SquadDeploymentMode) Initialize(ctx *core.UIContext) error {
    // No service creation - already injected

    err := gui.NewModeBuilder(&sdm.BaseMode, gui.ModeConfig{
        ModeName:   "squad_deployment",
        ReturnMode: "exploration",
        Panels: []gui.PanelSpec{
            {CustomBuild: sdm.buildSquadList},
            {CustomBuild: sdm.buildDetailPanel},
        },
    }).Build(ctx)

    return err
}

func (sdm *SquadDeploymentMode) handleDeploymentClick(x, y int) {
    pos := coords.ScreenToLogical(x, y)

    if sdm.deployer.CanDeployAtPosition(sdm.selectedSquadID, pos) {
        err := sdm.deployer.DeploySquad(sdm.selectedSquadID, pos)
        if err != nil {
            sdm.showError("Deployment failed: " + err.Error())
        }
    }
}
```

#### Step 3.5: Wire Up in Main

```go
// game_main/main.go
package main

import (
    "game_main/common"
    "game_main/game_main/gamefacade"
    "game_main/gui/core"
    "game_main/gui/guicombat"
    "game_main/gui/guisquads"
)

func setupGUI(em *common.EntityManager, modeManager *core.UIModeManager) error {
    // Create facades (concrete implementations)
    combatFacade := gamefacade.NewCombatFacade(em)
    squadFacade := gamefacade.NewSquadFacade(em)

    // Create CombatMode with facade satisfying all interfaces
    combatMode := guicombat.NewCombatMode(
        modeManager,
        combatFacade, // TurnManager
        combatFacade, // Attacker
        combatFacade, // MovementProvider
        combatFacade, // SquadQuerier
        combatFacade, // VictoryChecker
        combatFacade, // CombatInitializer
    )

    // Create SquadDeploymentMode with facade satisfying interfaces
    deploymentMode := guisquads.NewSquadDeploymentMode(
        modeManager,
        squadFacade, // SquadDeployer
        squadFacade, // SquadQuerier
    )

    // Create SquadEditorMode
    editorMode := guisquads.NewSquadEditorMode(
        modeManager,
        squadFacade, // SquadBuilder
        squadFacade, // SquadQuerier
    )

    // Register modes
    modeManager.RegisterMode("combat", combatMode)
    modeManager.RegisterMode("squad_deployment", deploymentMode)
    modeManager.RegisterMode("squad_editor", editorMode)

    return nil
}
```

**Pattern Notice:** Facade is passed multiple times to same constructor - this is OK! Each parameter represents a different interface (different contract).

**Alternative (if you prefer single struct):**

```go
// If passing facade 6 times feels repetitive, create a combiner:
type CombatDependencies struct {
    *gamefacade.CombatFacade
}

// Facade automatically satisfies all interfaces
combatMode := guicombat.NewCombatMode(
    modeManager,
    CombatDependencies{combatFacade},
    // ... but this is less explicit
)
```

**Recommendation:** Explicit is better than implicit (Go proverb). Pass facade 6 times.

---

## DETAILED CODE EXAMPLES

### Combat System Transformation

#### combatmode.go Transformation

**BEFORE (Current - Direct Tactical Imports):**

```go
// gui/guicombat/combatmode.go
package guicombat

import (
    "game_main/tactical/combatservices" // ❌ Direct tactical import
    "game_main/tactical/combat"         // ❌ Direct tactical import
)

type CombatMode struct {
    combatService *combatservices.CombatService // ❌ Concrete dependency
}

func (cm *CombatMode) Initialize(ctx *core.UIContext) error {
    cm.combatService = combatservices.NewCombatService(ctx.ECSManager) // ❌ Creates own dependencies
    // ...
}

func (cm *CombatMode) handleEndTurn() {
    err := cm.combatService.TurnManager.EndTurn() // ❌ Reaches through service
    // ...
}
```

**AFTER PHASE 1 (Facade - Still Concrete):**

```go
// gui/guicombat/combatmode.go
package guicombat

import (
    "game_main/game_main/gamefacade" // ✅ Only facade import
)

type CombatMode struct {
    facade *gamefacade.CombatFacade // ⚠️ Still concrete, but better
}

func (cm *CombatMode) Initialize(ctx *core.UIContext) error {
    cm.facade = gamefacade.NewCombatFacade(ctx.ECSManager) // ⚠️ Still creates dependencies
    // ...
}

func (cm *CombatMode) handleEndTurn() {
    err := cm.facade.EndTurn() // ✅ Cleaner API
    // ...
}
```

**AFTER PHASE 3 (Enhanced - Interfaces):**

```go
// gui/guicombat/combatmode.go
package guicombat

// ✅ NO imports of gamefacade or tactical!

type CombatMode struct {
    turns TurnManager // ✅ Interface from same package
}

func NewCombatMode(modeManager *core.UIModeManager, turns TurnManager) *CombatMode {
    return &CombatMode{turns: turns} // ✅ Dependency injection
}

func (cm *CombatMode) Initialize(ctx *core.UIContext) error {
    // ✅ No service creation, already injected
    // ...
}

func (cm *CombatMode) handleEndTurn() {
    err := cm.turns.EndTurn() // ✅ Uses interface
    // ...
}
```

#### combat_action_handler.go Transformation

**BEFORE:**

```go
// gui/guicombat/combat_action_handler.go
package guicombat

import (
    "game_main/tactical/combat"         // ❌ Direct import
    "game_main/tactical/combatservices" // ❌ Direct import
)

type CombatActionHandler struct {
    combatService *combatservices.CombatService // ❌ Concrete
}

func (cah *CombatActionHandler) ExecuteAttack(a, d ecs.EntityID) {
    result := cah.combatService.CombatActSystem.ExecuteAttackAction(a, d) // ❌ Reaches through
    // ...
}
```

**AFTER ENHANCED:**

```go
// gui/guicombat/combat_action_handler.go
package guicombat

// ✅ NO tactical imports!

type CombatActionHandler struct {
    combat Attacker // ✅ Interface from same package
}

func NewCombatActionHandler(combat Attacker) *CombatActionHandler {
    return &CombatActionHandler{combat: combat}
}

func (cah *CombatActionHandler) ExecuteAttack(a, d ecs.EntityID) {
    result := cah.combat.ExecuteAttack(a, d) // ✅ Clean interface call
    // ...
}
```

#### combat_input_handler.go Transformation

**BEFORE:**

```go
// gui/guicombat/combat_input_handler.go
package guicombat

import (
    "game_main/tactical/combatservices" // ❌ Direct import
)

type CombatInputHandler struct {
    combatService *combatservices.CombatService // ❌ Concrete
}

func (cih *CombatInputHandler) HandleMoveClick(squadID ecs.EntityID) {
    tiles := cih.combatService.MovementSystem.GetValidMovementTiles(squadID) // ❌ Reaches through
    // ...
}
```

**AFTER ENHANCED:**

```go
// gui/guicombat/combat_input_handler.go
package guicombat

// ✅ NO tactical imports!

type CombatInputHandler struct {
    movement MovementProvider // ✅ Interface from same package
}

func NewCombatInputHandler(movement MovementProvider) *CombatInputHandler {
    return &CombatInputHandler{movement: movement}
}

func (cih *CombatInputHandler) HandleMoveClick(squadID ecs.EntityID) {
    tiles := cih.movement.GetValidMovementTiles(squadID) // ✅ Clean interface call
    // ...
}
```

#### combat_interfaces.go (NEW FILE)

```go
// gui/guicombat/combat_interfaces.go
package guicombat

import (
    "game_main/tactical/combat"
    "game_main/tactical/combatservices"
    "game_main/world/coords"
    "github.com/bytearena/ecs"
)

// Consumer-defined interfaces (Go idiom).
// These live in GUI package, not tactical package.

type TurnManager interface {
    GetCurrentFaction() ecs.EntityID
    GetCurrentRound() int
    EndTurn() error
}

type Attacker interface {
    ExecuteAttack(attackerID, defenderID ecs.EntityID) combat.AttackResult
}

type MovementProvider interface {
    GetValidMovementTiles(squadID ecs.EntityID) []coords.LogicalPosition
    MoveSquad(squadID ecs.EntityID, pos coords.LogicalPosition) error
}

type SquadQuerier interface {
    GetAliveSquadsInFaction(factionID ecs.EntityID) []ecs.EntityID
    GetSquadAtPosition(pos coords.LogicalPosition) (ecs.EntityID, bool)
}

type VictoryChecker interface {
    CheckVictoryCondition() *combatservices.VictoryCheckResult
}

type CombatInitializer interface {
    InitializeCombat(factionIDs []ecs.EntityID) error
    EndCombat()
}
```

---

### Squad System Transformation

#### squaddeploymentmode.go Transformation

**BEFORE:**

```go
// gui/guisquads/squaddeploymentmode.go
package guisquads

import (
    "game_main/tactical/squadservices" // ❌ Direct import
)

type SquadDeploymentMode struct {
    deploymentService *squadservices.SquadDeploymentService // ❌ Concrete
}

func (sdm *SquadDeploymentMode) Initialize(ctx *core.UIContext) error {
    sdm.deploymentService = squadservices.NewSquadDeploymentService(ctx.ECSManager) // ❌ Creates dependency
    // ...
}
```

**AFTER ENHANCED:**

```go
// gui/guisquads/squaddeploymentmode.go
package guisquads

// ✅ NO tactical imports!

type SquadDeploymentMode struct {
    deployer SquadDeployer // ✅ Interface from same package
    squads   SquadQuerier  // ✅ Interface from same package
}

func NewSquadDeploymentMode(
    modeManager *core.UIModeManager,
    deployer SquadDeployer,
    squads SquadQuerier,
) *SquadDeploymentMode {
    return &SquadDeploymentMode{
        deployer: deployer,
        squads:   squads,
    }
}

func (sdm *SquadDeploymentMode) Initialize(ctx *core.UIContext) error {
    // ✅ No service creation
    // ...
}
```

#### squadeditormode.go Transformation

**BEFORE:**

```go
// gui/guisquads/squadeditormode.go
package guisquads

import (
    "game_main/tactical/squadservices" // ❌ Direct import
)

type SquadEditorMode struct {
    builderService *squadservices.SquadBuilderService // ❌ Concrete
}

func (sem *SquadEditorMode) Initialize(ctx *core.UIContext) error {
    sem.builderService = squadservices.NewSquadBuilderService(ctx.ECSManager) // ❌ Creates dependency
    // ...
}

func (sem *SquadEditorMode) handleAddUnit(unitID ecs.EntityID) {
    err := sem.builderService.AddUnitToSquad(sem.selectedSquadID, unitID) // ❌ Direct service call
    // ...
}
```

**AFTER ENHANCED:**

```go
// gui/guisquads/squadeditormode.go
package guisquads

// ✅ NO tactical imports!

type SquadEditorMode struct {
    builder SquadBuilder // ✅ Interface from same package
    squads  SquadQuerier // ✅ Interface from same package
}

func NewSquadEditorMode(
    modeManager *core.UIModeManager,
    builder SquadBuilder,
    squads SquadQuerier,
) *SquadEditorMode {
    return &SquadEditorMode{
        builder: builder,
        squads:  squads,
    }
}

func (sem *SquadEditorMode) Initialize(ctx *core.UIContext) error {
    // ✅ No service creation
    // ...
}

func (sem *SquadEditorMode) handleAddUnit(unitID ecs.EntityID) {
    err := sem.builder.AddUnitToSquad(sem.selectedSquadID, unitID) // ✅ Interface call
    // ...
}
```

#### unitpurchasemode.go Transformation

**BEFORE:**

```go
// gui/guisquads/unitpurchasemode.go
package guisquads

import (
    "game_main/tactical/squadservices" // ❌ Direct import
)

type UnitPurchaseMode struct {
    squadService *squadservices.SquadBuilderService // ❌ Concrete
}
```

**AFTER ENHANCED:**

```go
// gui/guisquads/unitpurchasemode.go
package guisquads

// ✅ NO tactical imports!

type UnitPurchaseMode struct {
    builder SquadBuilder // ✅ Interface from same package
}

func NewUnitPurchaseMode(
    modeManager *core.UIModeManager,
    builder SquadBuilder,
) *UnitPurchaseMode {
    return &UnitPurchaseMode{builder: builder}
}
```

#### squad_interfaces.go (NEW FILE)

```go
// gui/guisquads/squad_interfaces.go
package guisquads

import (
    "game_main/tactical/squads"
    "game_main/world/coords"
    "github.com/bytearena/ecs"
)

// Consumer-defined interfaces for squad GUI modes.

type SquadBuilder interface {
    CreateSquad(playerID ecs.EntityID, name string) (ecs.EntityID, error)
    AddUnitToSquad(squadID, unitID ecs.EntityID) error
    RemoveUnitFromSquad(squadID, unitID ecs.EntityID) error
    ValidateSquad(squadID ecs.EntityID) error
}

type SquadDeployer interface {
    CanDeployAtPosition(squadID ecs.EntityID, pos coords.LogicalPosition) bool
    DeploySquad(squadID ecs.EntityID, pos coords.LogicalPosition) error
    GetDeploymentZones() []coords.LogicalPosition
}

type SquadQuerier interface {
    GetPlayerSquads(playerID ecs.EntityID) []ecs.EntityID
    GetSquadInfo(squadID ecs.EntityID) (*squads.SquadData, error)
}
```

---

### Core GUI Management

#### How GameSession/UIContext Changes

**BEFORE (If using GameSession - Approach 1 pattern):**

```go
// ❌ Service Locator Anti-Pattern
type GameSession struct {
    combatService *combatservices.CombatService
    squadService  *squadservices.SquadBuilderService
    // ... 10 more services
}

func (gs *GameSession) GetCombatService() *combatservices.CombatService {
    return gs.combatService
}

// CombatMode pulls service from session
func (cm *CombatMode) Initialize(ctx *core.UIContext) error {
    cm.combatService = ctx.GameSession.GetCombatService()
}
```

**AFTER ENHANCED (No GameSession needed):**

```go
// ✅ Constructor Injection
// main.go creates facades and wires them directly
func setupGUI(em *common.EntityManager, modeManager *core.UIModeManager) error {
    combatFacade := gamefacade.NewCombatFacade(em)
    squadFacade := gamefacade.NewSquadFacade(em)

    combatMode := guicombat.NewCombatMode(
        modeManager,
        combatFacade, // All dependencies explicit
        combatFacade,
        combatFacade,
        combatFacade,
        combatFacade,
        combatFacade,
    )

    modeManager.RegisterMode("combat", combatMode)
    return nil
}

// CombatMode doesn't create or locate services
func (cm *CombatMode) Initialize(ctx *core.UIContext) error {
    // Dependencies already injected via constructor
    // Just build UI
    return cm.buildUI(ctx)
}
```

**Why This Is Better:**
- **No Hidden Dependencies**: Constructor signature shows exactly what CombatMode needs
- **Easier Testing**: Just pass mocks to constructor
- **No God Object**: No GameSession holding everything
- **Clear Ownership**: Each mode owns its dependencies

---

## TESTING STRATEGY

### Unit Testing GUI Components (The Payoff!)

This is where Enhanced Approach 3 shines - GUI becomes trivially testable.

#### Testing CombatMode Without ECS

```go
// gui/guicombat/combatmode_test.go
package guicombat

import (
    "testing"
    "game_main/tactical/combat"
    "game_main/tactical/combatservices"
    "github.com/bytearena/ecs"
)

// ===================================================================
// MOCK IMPLEMENTATIONS
// ===================================================================

// Mock TurnManager
type mockTurnManager struct {
    currentFaction ecs.EntityID
    currentRound   int
    endTurnCalled  bool
    endTurnError   error
}

func (m *mockTurnManager) GetCurrentFaction() ecs.EntityID {
    return m.currentFaction
}

func (m *mockTurnManager) GetCurrentRound() int {
    return m.currentRound
}

func (m *mockTurnManager) EndTurn() error {
    m.endTurnCalled = true
    return m.endTurnError
}

// Mock Attacker
type mockAttacker struct {
    attackResult combat.AttackResult
    attackCalled bool
    lastAttacker ecs.EntityID
    lastDefender ecs.EntityID
}

func (m *mockAttacker) ExecuteAttack(a, d ecs.EntityID) combat.AttackResult {
    m.attackCalled = true
    m.lastAttacker = a
    m.lastDefender = d
    return m.attackResult
}

// Mock MovementProvider
type mockMovementProvider struct {
    tiles     []coords.LogicalPosition
    moveError error
}

func (m *mockMovementProvider) GetValidMovementTiles(squadID ecs.EntityID) []coords.LogicalPosition {
    return m.tiles
}

func (m *mockMovementProvider) MoveSquad(squadID ecs.EntityID, pos coords.LogicalPosition) error {
    return m.moveError
}

// ===================================================================
// TESTS
// ===================================================================

func TestCombatMode_HandleEndTurn_Success(t *testing.T) {
    // Arrange
    mockTurns := &mockTurnManager{
        currentFaction: 1,
        currentRound:   5,
    }

    cm := NewCombatMode(
        nil,       // modeManager (not needed for this test)
        mockTurns, // TurnManager
        nil,       // Attacker (not needed)
        nil,       // MovementProvider (not needed)
        nil,       // SquadQuerier (not needed)
        nil,       // VictoryChecker (not needed)
        nil,       // CombatInitializer (not needed)
    )

    // Act
    cm.handleEndTurn()

    // Assert
    if !mockTurns.endTurnCalled {
        t.Error("Expected EndTurn to be called")
    }
}

func TestCombatMode_HandleEndTurn_Error(t *testing.T) {
    // Arrange
    mockTurns := &mockTurnManager{
        currentFaction: 1,
        currentRound:   5,
        endTurnError:   fmt.Errorf("turn end failed"),
    }

    cm := NewCombatMode(nil, mockTurns, nil, nil, nil, nil, nil)

    // Act
    cm.handleEndTurn()

    // Assert
    if !mockTurns.endTurnCalled {
        t.Error("Expected EndTurn to be called even if it errors")
    }

    // Verify error was logged (check cm.logManager)
    logs := cm.logManager.GetLogs()
    if len(logs) == 0 {
        t.Error("Expected error to be logged")
    }
}

func TestCombatMode_HandleAttack_Success(t *testing.T) {
    // Arrange
    mockAttack := &mockAttacker{
        attackResult: combat.AttackResult{
            Success: true,
            Damage:  25,
            TargetKilled: false,
        },
    }

    cm := NewCombatMode(nil, nil, mockAttack, nil, nil, nil, nil)

    // Act
    cm.handleAttackAction(100, 200)

    // Assert
    if !mockAttack.attackCalled {
        t.Error("Expected ExecuteAttack to be called")
    }

    if mockAttack.lastAttacker != 100 {
        t.Errorf("Expected attacker 100, got %d", mockAttack.lastAttacker)
    }

    if mockAttack.lastDefender != 200 {
        t.Errorf("Expected defender 200, got %d", mockAttack.lastDefender)
    }

    // Verify log contains damage
    logs := cm.logManager.GetLogs()
    if len(logs) == 0 {
        t.Error("Expected attack to be logged")
    }

    if !strings.Contains(logs[0], "25") {
        t.Error("Expected log to contain damage amount")
    }
}

func TestCombatMode_HandleAttack_Failed(t *testing.T) {
    // Arrange
    mockAttack := &mockAttacker{
        attackResult: combat.AttackResult{
            Success:     false,
            ErrorReason: "out of range",
        },
    }

    cm := NewCombatMode(nil, nil, mockAttack, nil, nil, nil, nil)

    // Act
    cm.handleAttackAction(100, 200)

    // Assert
    logs := cm.logManager.GetLogs()
    if len(logs) == 0 || !strings.Contains(logs[0], "failed") {
        t.Error("Expected failure to be logged")
    }
}

func TestCombatMode_HandleMovement_HighlightsTiles(t *testing.T) {
    // Arrange
    expectedTiles := []coords.LogicalPosition{
        {X: 1, Y: 2},
        {X: 1, Y: 3},
        {X: 2, Y: 2},
    }

    mockMovement := &mockMovementProvider{
        tiles: expectedTiles,
    }

    cm := NewCombatMode(nil, nil, nil, mockMovement, nil, nil, nil)

    // Act
    cm.handleMovementSelection(100)

    // Assert
    // Verify movement renderer received correct tiles
    if cm.movementRenderer == nil {
        t.Fatal("Movement renderer not initialized")
    }

    highlighted := cm.movementRenderer.GetHighlightedTiles()
    if len(highlighted) != len(expectedTiles) {
        t.Errorf("Expected %d tiles highlighted, got %d", len(expectedTiles), len(highlighted))
    }
}
```

**Key Points:**
- **No ECS Setup**: Tests run in milliseconds, no entity manager needed
- **Focused Tests**: Each test verifies single behavior
- **Easy Mocking**: Mocks are ~10 lines each
- **Table-Driven**: Can easily add more test cases

#### Table-Driven Tests (Go Idiom)

```go
func TestCombatMode_HandleAttack_Scenarios(t *testing.T) {
    tests := []struct {
        name         string
        attackResult combat.AttackResult
        wantLoggedMessage string
    }{
        {
            name: "successful attack",
            attackResult: combat.AttackResult{
                Success: true,
                Damage:  25,
            },
            wantLoggedMessage: "25 damage",
        },
        {
            name: "attack missed",
            attackResult: combat.AttackResult{
                Success:     false,
                ErrorReason: "missed",
            },
            wantLoggedMessage: "failed",
        },
        {
            name: "target killed",
            attackResult: combat.AttackResult{
                Success:      true,
                Damage:       50,
                TargetKilled: true,
            },
            wantLoggedMessage: "killed",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Arrange
            mockAttack := &mockAttacker{
                attackResult: tt.attackResult,
            }

            cm := NewCombatMode(nil, nil, mockAttack, nil, nil, nil, nil)

            // Act
            cm.handleAttackAction(100, 200)

            // Assert
            logs := cm.logManager.GetLogs()
            if len(logs) == 0 {
                t.Fatal("Expected log message")
            }

            if !strings.Contains(logs[0], tt.wantLoggedMessage) {
                t.Errorf("Expected log to contain %q, got %q", tt.wantLoggedMessage, logs[0])
            }
        })
    }
}
```

### Integration Testing Facades

Facades need integration tests with real ECS (they're the boundary).

```go
// game_main/gamefacade/combat_facade_test.go
package gamefacade

import (
    "testing"
    "game_main/common"
    "game_main/tactical/combat"
    "game_main/tactical/squads"
    "github.com/bytearena/ecs"
)

func TestCombatFacade_EndTurn_Integration(t *testing.T) {
    // Setup real ECS
    em := common.NewEntityManager()

    // Create test factions
    faction1 := combat.CreateFaction(em, "Player", true)
    faction2 := combat.CreateFaction(em, "Enemy", false)

    // Create facade
    facade := NewCombatFacade(em)

    // Initialize combat
    err := facade.InitializeCombat([]ecs.EntityID{faction1, faction2})
    if err != nil {
        t.Fatalf("InitializeCombat failed: %v", err)
    }

    // Test turn management
    startFaction := facade.GetCurrentFaction()
    startRound := facade.GetCurrentRound()

    err = facade.EndTurn()
    if err != nil {
        t.Errorf("EndTurn failed: %v", err)
    }

    newFaction := facade.GetCurrentFaction()
    if newFaction == startFaction {
        t.Error("Faction should change after EndTurn")
    }

    // If both factions had turns, round should increment
    err = facade.EndTurn()
    if err != nil {
        t.Errorf("Second EndTurn failed: %v", err)
    }

    newRound := facade.GetCurrentRound()
    if newRound != startRound+1 {
        t.Errorf("Expected round %d, got %d", startRound+1, newRound)
    }
}

func TestCombatFacade_ExecuteAttack_Integration(t *testing.T) {
    // Setup
    em := common.NewEntityManager()
    facade := NewCombatFacade(em)

    // Create test squads
    attackerID := squads.CreateTestSquad(em, "Attacker", 100)
    defenderID := squads.CreateTestSquad(em, "Defender", 50)

    // Execute attack
    result := facade.ExecuteAttack(attackerID, defenderID)

    // Verify result
    if !result.Success {
        t.Errorf("Expected attack to succeed, got error: %s", result.ErrorReason)
    }

    if result.Damage <= 0 {
        t.Error("Expected positive damage")
    }
}
```

### Interface Compliance Testing

Verify facades implement interfaces (compile-time safety in tests).

```go
// gui/guicombat/combatmode_test.go
package guicombat

import (
    "testing"
    "game_main/game_main/gamefacade"
)

// Compile-time verification that CombatFacade implements all interfaces
func TestCombatFacade_ImplementsInterfaces(t *testing.T) {
    // These will fail to compile if CombatFacade doesn't implement interfaces
    var _ TurnManager = (*gamefacade.CombatFacade)(nil)
    var _ Attacker = (*gamefacade.CombatFacade)(nil)
    var _ MovementProvider = (*gamefacade.CombatFacade)(nil)
    var _ SquadQuerier = (*gamefacade.CombatFacade)(nil)
    var _ VictoryChecker = (*gamefacade.CombatFacade)(nil)
    var _ CombatInitializer = (*gamefacade.CombatFacade)(nil)

    // If this test compiles, all interfaces are satisfied
    t.Log("CombatFacade implements all combat interfaces")
}

func TestSquadFacade_ImplementsInterfaces(t *testing.T) {
    var _ SquadBuilder = (*gamefacade.SquadFacade)(nil)
    var _ SquadDeployer = (*gamefacade.SquadFacade)(nil)
    var _ SquadQuerier = (*gamefacade.SquadFacade)(nil)

    t.Log("SquadFacade implements all squad interfaces")
}
```

---

## IMPLEMENTATION ROADMAP

### Overview

| Phase | Duration | Risk | Dependencies |
|-------|----------|------|--------------|
| Phase 1: Create Facades | 2 days | Low | None |
| Phase 2: Define Interfaces | 1 day | Low | Phase 1 |
| Phase 3: Refactor GUI | 1.5 days | Medium | Phase 2 |
| Phase 4: Testing | 1 day | Low | Phase 3 |
| **TOTAL** | **5.5 days** | **Low-Medium** | Sequential |

### Week 1: Foundation (Phase 1 & 2)

#### Day 1: Combat Facade

**Morning: Create facade structure**

```bash
# Create package
mkdir -p game_main/gamefacade
touch game_main/gamefacade/combat_facade.go
```

Tasks:
- [ ] Create `CombatFacade` struct
- [ ] Implement `NewCombatFacade` constructor
- [ ] Add turn management methods (3 methods)
- [ ] Add combat action methods (1 method)
- [ ] Add movement methods (2 methods)
- [ ] Add squad query methods (2 methods)
- [ ] Add victory check methods (1 method)
- [ ] Add combat lifecycle methods (2 methods)

**Afternoon: Integration test**

- [ ] Create `combat_facade_test.go`
- [ ] Test turn management flow
- [ ] Test attack execution
- [ ] Test movement operations
- [ ] Run tests: `go test ./game_main/gamefacade/...`

**End of Day Checkpoint:**
```bash
go test ./game_main/gamefacade/...
# All tests pass
```

#### Day 2: Squad Facade & CombatMode Migration

**Morning: Squad Facade**

- [ ] Create `squad_facade.go`
- [ ] Implement squad building methods (4 methods)
- [ ] Implement deployment methods (3 methods)
- [ ] Implement squad query methods (2 methods)
- [ ] Write integration tests

**Afternoon: Migrate CombatMode to use facade**

- [ ] Update `combatmode.go` imports (remove tactical, add gamefacade)
- [ ] Replace `combatService` field with `facade` field
- [ ] Update `Initialize()` to create facade
- [ ] Update all method calls to use facade
- [ ] Run game and test combat mode

**End of Day Checkpoint:**
```bash
go build -o game_main/game_main.exe game_main/*.go
./game_main/game_main.exe
# Combat mode works, no crashes
```

#### Day 3: Interface Definition & Compliance Tests

**Morning: Define combat interfaces**

- [ ] Create `gui/guicombat/combat_interfaces.go`
- [ ] Define `TurnManager` interface (3 methods)
- [ ] Define `Attacker` interface (1 method)
- [ ] Define `MovementProvider` interface (2 methods)
- [ ] Define `SquadQuerier` interface (2 methods)
- [ ] Define `VictoryChecker` interface (1 method)
- [ ] Define `CombatInitializer` interface (2 methods)
- [ ] Add documentation comments

**Afternoon: Define squad interfaces & compliance tests**

- [ ] Create `gui/guisquads/squad_interfaces.go`
- [ ] Define `SquadBuilder` interface (4 methods)
- [ ] Define `SquadDeployer` interface (3 methods)
- [ ] Define `SquadQuerier` interface (2 methods)
- [ ] Create compliance test in `combatmode_test.go`
- [ ] Create compliance test in `squaddeploymentmode_test.go`

**End of Day Checkpoint:**
```bash
go test ./gui/guicombat/... ./gui/guisquads/...
# Compliance tests pass (facades satisfy interfaces)
```

### Week 2: Refactoring (Phase 3)

#### Day 4: CombatMode Constructor Refactor

**Morning: Update CombatMode constructor**

- [ ] Change `NewCombatMode` signature to accept 6 interfaces
- [ ] Replace facade field with interface fields
- [ ] Update `Initialize()` to NOT create facade
- [ ] Remove tactical imports from `combatmode.go`

**Afternoon: Update CombatMode methods**

- [ ] Update `handleEndTurn()` to use `turns` interface
- [ ] Update `handleAttackAction()` to use `combat` interface
- [ ] Update `handleMoveAction()` to use `movement` interface
- [ ] Update squad list display to use `squads` interface
- [ ] Update victory check to use `victory` interface
- [ ] Verify no tactical imports remain

**End of Day Checkpoint:**
```bash
grep -r "game_main/tactical" gui/guicombat/*.go
# Should return NO results

go build ./gui/guicombat/...
# Compiles successfully
```

#### Day 5: Combat Sub-Components & Squad Modes

**Morning: Refactor CombatActionHandler & CombatInputHandler**

- [ ] Update `CombatActionHandler` to accept `Attacker` interface
- [ ] Update `CombatInputHandler` to accept `MovementProvider` interface
- [ ] Update constructors for both
- [ ] Remove tactical imports from both files

**Afternoon: Refactor SquadDeploymentMode**

- [ ] Update `NewSquadDeploymentMode` to accept interfaces
- [ ] Replace service fields with interface fields
- [ ] Update `Initialize()` to NOT create services
- [ ] Remove tactical imports
- [ ] Test deployment flow

**End of Day Checkpoint:**
```bash
grep -r "game_main/tactical" gui/guicombat/*.go gui/guisquads/*.go
# Should return NO results (except imports in test files)

go build ./gui/...
# All GUI packages compile
```

#### Day 6: SquadEditorMode, UnitPurchaseMode & Wiring

**Morning: Refactor remaining squad modes**

- [ ] Update `SquadEditorMode` to accept interfaces
- [ ] Update `UnitPurchaseMode` to accept interfaces
- [ ] Remove tactical imports from both

**Afternoon: Wire everything in main.go**

- [ ] Update `setupGUI()` to create facades
- [ ] Pass facades to CombatMode constructor (6 parameters)
- [ ] Pass facades to SquadDeploymentMode constructor (2 parameters)
- [ ] Pass facades to SquadEditorMode constructor (2 parameters)
- [ ] Pass facades to UnitPurchaseMode constructor (1 parameter)

**End of Day Checkpoint:**
```bash
go build -o game_main/game_main.exe game_main/*.go
./game_main/game_main.exe
# Full game runs, all modes work
```

### Week 3: Testing & Validation (Phase 4)

#### Day 7: Unit Tests for CombatMode

**Morning: Create mocks**

- [ ] Create `mockTurnManager` in `combatmode_test.go`
- [ ] Create `mockAttacker`
- [ ] Create `mockMovementProvider`
- [ ] Create `mockSquadQuerier`
- [ ] Create `mockVictoryChecker`
- [ ] Create `mockCombatInitializer`

**Afternoon: Write unit tests**

- [ ] Test `handleEndTurn()` with mock
- [ ] Test `handleAttackAction()` with mock
- [ ] Test `handleMoveAction()` with mock
- [ ] Test victory detection with mock
- [ ] Table-driven tests for attack scenarios

**End of Day Checkpoint:**
```bash
go test -v ./gui/guicombat/...
# All unit tests pass
# Tests run in < 100ms (no ECS needed)
```

#### Day 8: Unit Tests for Squad Modes

**Morning: Squad deployment tests**

- [ ] Create mocks for `SquadDeployer`, `SquadQuerier`
- [ ] Test deployment click handling
- [ ] Test invalid deployment positions
- [ ] Test deployment zone highlighting

**Afternoon: Squad editor tests**

- [ ] Create mocks for `SquadBuilder`
- [ ] Test adding units to squad
- [ ] Test removing units from squad
- [ ] Test squad validation
- [ ] Test unit purchase flow

**End of Day Checkpoint:**
```bash
go test -v ./gui/guisquads/...
# All unit tests pass
# Fast tests (< 200ms total)
```

#### Day 9: Integration Tests & Performance

**Morning: Facade integration tests**

- [ ] Expand `combat_facade_test.go` with more scenarios
- [ ] Expand `squad_facade_test.go` with more scenarios
- [ ] Test error paths
- [ ] Test edge cases (empty squads, defeated factions)

**Afternoon: Performance validation**

- [ ] Run benchmarks for facade methods
- [ ] Verify no allocation regressions
- [ ] Play through full combat scenario
- [ ] Check for frame rate issues
- [ ] Profile if needed

**End of Day Checkpoint:**
```bash
go test -bench=. ./game_main/gamefacade/...
# Benchmarks show no significant overhead

go test -v ./...
# All tests pass (unit + integration)
```

#### Day 10: Documentation & Final Validation

**Morning: Update documentation**

- [ ] Update `CLAUDE.md` with facade pattern
- [ ] Add interface examples to `ecs_best_practices.md`
- [ ] Document consumer-defined interface pattern
- [ ] Add testing examples

**Afternoon: Final validation**

- [ ] Full playthrough (exploration → deployment → combat)
- [ ] Verify all modes work
- [ ] Check for any tactical imports in GUI
- [ ] Run full test suite
- [ ] Create pull request

**End of Day Checkpoint:**
```bash
# No tactical imports in GUI
find gui -name "*.go" -not -name "*_test.go" -exec grep -l "game_main/tactical" {} \;
# Should return empty

# All tests pass
go test ./...

# Game runs
go build -o game_main/game_main.exe game_main/*.go && ./game_main/game_main.exe
```

### Rollback Plan (If Needed)

If refactoring fails at any phase:

**Phase 1 Rollback:**
```bash
git checkout -- game_main/gamefacade/
git checkout -- gui/guicombat/combatmode.go
# Facades not working, revert to direct imports
```

**Phase 2 Rollback:**
```bash
git checkout -- gui/guicombat/combat_interfaces.go
git checkout -- gui/guisquads/squad_interfaces.go
# Interfaces not compiling, revert to Phase 1 (facade-only)
```

**Phase 3 Rollback:**
```bash
git checkout -- gui/guicombat/*.go
git checkout -- gui/guisquads/*.go
git checkout -- game_main/main.go
# Constructor changes broke game, revert to Phase 2
```

**Critical**: Each phase is a valid stopping point. You can ship after Phase 1 (facades) and continue later.

---

## COMPARISON WITH OTHER APPROACHES

### Side-by-Side: Original Approach 3 vs Enhanced

| Aspect | **Original Approach 3** | **Enhanced Approach 3** |
|--------|------------------------|------------------------|
| **Facade** | ✅ Yes - `GameFacade` | ✅ Yes - `CombatFacade`, `SquadFacade` |
| **Interfaces** | ❌ No | ✅ Yes - consumer-defined |
| **GUI Dependency** | ⚠️ Concrete `*GameFacade` | ✅ Interfaces only |
| **Testability** | ❌ Requires real facade | ✅ Easy mocks |
| **Import Direction** | `gui → gamefacade → tactical` | `gui → (interfaces in gui)` |
| **Tactical Awareness** | Neutral (doesn't know GUI exists) | Neutral (doesn't know GUI exists) |
| **Go-Idiomatic** | ⚠️ Partial | ✅ Yes |

**Example Code Comparison:**

```go
// Original Approach 3
type CombatMode struct {
    facade *gamefacade.GameFacade // Concrete dependency
}

func (cm *CombatMode) Initialize(ctx *core.UIContext) error {
    cm.facade = gamefacade.NewGameFacade(ctx.ECSManager) // Creates dependency
}

// Testing requires real facade (hard to isolate)
```

```go
// Enhanced Approach 3
type CombatMode struct {
    turns TurnManager // Interface from guicombat package
}

func NewCombatMode(turns TurnManager) *CombatMode {
    return &CombatMode{turns: turns} // Injected
}

// Testing uses simple mock (easy to isolate)
```

### Enhanced vs Approach 1 (Provider-Side Interfaces)

| Aspect | **Approach 1** | **Enhanced Approach 3** |
|--------|---------------|------------------------|
| **Interface Location** | `gui/interfaces/` (provider) | `gui/guicombat/` (consumer) |
| **Interface Size** | Large (12+ methods) | Small (1-3 methods) |
| **DTOs** | Yes (manual conversion) | No (direct types) |
| **Service Locator** | GameSession (anti-pattern) | Constructor injection |
| **Tactical Imports GUI?** | ⚠️ Yes (`var _ ICombat = ...`) | ✅ No |
| **Go-Idiomatic** | ❌ Violates FAQ | ✅ Matches stdlib |

**Why Enhanced Is Better:**

```go
// Approach 1 - Violates "consumer defines interface"
// File: gui/interfaces/combat_controller.go
package interfaces

type ICombatController interface {
    // 12 methods - too big!
}

// File: tactical/combatservices/combat_service.go
var _ interfaces.ICombatController = (*CombatService)(nil) // Tactical imports GUI!

// Enhanced Approach 3 - Follows Go FAQ
// File: gui/guicombat/combat_interfaces.go
package guicombat

type TurnManager interface {
    // 3 methods - right size!
}

// File: tactical/combatservices/combat_service.go
// NO import of gui, NO awareness of interface
```

### Enhanced vs Approach 2 (Command/Event Pattern)

| Aspect | **Approach 2** | **Enhanced Approach 3** |
|--------|---------------|------------------------|
| **Commands** | Yes (attack command, move command) | No (direct calls) |
| **Events** | Yes (event bus) | No (synchronous) |
| **Mediator** | Yes | No |
| **Complexity** | High (many indirections) | Low (direct calls) |
| **Stdlib Pattern** | ❌ Not in Go stdlib | ✅ Matches stdlib |
| **Effort** | 5.5 days | 4-5 days |

**Why Enhanced Is Simpler:**

```go
// Approach 2 - Command pattern (NOT Go idiomatic)
type AttackCommand struct {
    attackerID ecs.EntityID
    defenderID ecs.EntityID
}

func (ac *AttackCommand) Execute() error {
    // ...
}

mediator.Execute(NewAttackCommand(a, d)) // Indirection!

// Enhanced Approach 3 - Direct call (Go way)
result := cm.combat.ExecuteAttack(a, d) // Simple!
```

---

## GO STANDARD LIBRARY PATTERN MATCHING

### Pattern 1: `io.Reader` / `io.Writer`

**How stdlib does it:**

```go
// Package io (CONSUMER) defines interface
package io

type Reader interface {
    Read(p []byte) (n int, err error)
}

// Package os (PROVIDER) implements (no import of io needed)
package os

func (f *File) Read(p []byte) (int, error) {
    // Implementation
}

// Package io (CONSUMER) uses interface
func Copy(dst Writer, src Reader) (int64, error) {
    // Uses interface
}
```

**TinkerRogue Enhanced Approach 3 matches this EXACTLY:**

```go
// gui/guicombat (CONSUMER) defines interface
package guicombat

type Attacker interface {
    ExecuteAttack(a, d ecs.EntityID) combat.AttackResult
}

// tactical/combatservices (PROVIDER) implements (no import of gui needed)
package combatservices

func (cs *CombatService) ExecuteAttack(a, d ecs.EntityID) combat.AttackResult {
    // Implementation
}

// gui/guicombat (CONSUMER) uses interface
func NewCombatMode(combat Attacker) *CombatMode {
    // Uses interface
}
```

**Perfect Match! ✅**

### Pattern 2: `http.Handler`

**How stdlib does it:**

```go
// Package http (CONSUMER) defines small interface
package http

type Handler interface {
    ServeHTTP(ResponseWriter, *Request)
}

// User code (PROVIDER) implements
type MyHandler struct{}

func (h *MyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    // No import of http interfaces package
}

// Package http (CONSUMER) uses
func ListenAndServe(addr string, handler Handler) error
```

**TinkerRogue Enhanced Approach 3 matches this:**

```go
// gui/guicombat (CONSUMER) defines small interface
package guicombat

type TurnManager interface {
    EndTurn() error
}

// gamefacade (PROVIDER) implements
type CombatFacade struct{}

func (cf *CombatFacade) EndTurn() error {
    // No import of guicombat
}

// gui/guicombat (CONSUMER) uses
func NewCombatMode(turns TurnManager) *CombatMode
```

**Perfect Match! ✅**

### Pattern 3: `database/sql` Driver Registration

**How stdlib does it:**

```go
// Package database/sql
package sql

type Driver interface {
    Open(name string) (Conn, error)
}

func Register(name string, driver Driver)

// Provider (postgres driver)
package pq

func init() {
    sql.Register("postgres", &Driver{})
}
```

**TinkerRogue DOES NOT need this pattern:**

- No pluggable implementations (always use CombatService)
- No runtime registration
- Static wiring in main.go is sufficient

**This is CORRECT - don't over-engineer! ✅**

### Pattern 4: `context.Context` - Exception Case

**Stdlib DOES return interface here:**

```go
package context

type Context interface { ... }

func Background() Context // Returns interface!
```

**Why stdlib breaks its own rule:**
- Context must be passed through entire call stack
- Many implementations (background, TODO, WithCancel, etc.)
- Truly needs runtime polymorphism

**TinkerRogue CombatService does NOT need this:**
- Not passed through entire stack
- Single implementation (CombatFacade)
- Constructor injection sufficient

**Enhanced Approach 3 correctly returns structs, not interfaces. ✅**

---

## ANTI-PATTERN AVOIDANCE CHECKLIST

Use this checklist during implementation to avoid common pitfalls.

### ❌ Anti-Pattern 1: Provider-Side Interfaces

**What to avoid:**

```go
// ❌ WRONG - Interface in tactical package
// File: tactical/interfaces/combat.go
package interfaces

type ICombatService interface { ... }

// ❌ WRONG - Tactical imports GUI interfaces
// File: tactical/combatservices/service.go
var _ interfaces.ICombatService = (*CombatService)(nil)
```

**What to do instead:**

```go
// ✅ CORRECT - Interface in GUI package
// File: gui/guicombat/combat_interfaces.go
package guicombat

type TurnManager interface { ... }

// ✅ CORRECT - Tactical has no awareness of GUI
// File: tactical/combatservices/service.go
func (cs *CombatService) EndTurn() error {
    // No interface import, no static assertion
}
```

**Validation:**

```bash
# Should find NO interfaces in tactical packages
find tactical -name "*interface*.go"

# Interfaces should only be in GUI
find gui -name "*interface*.go"
# Should find: gui/guicombat/combat_interfaces.go, gui/guisquads/squad_interfaces.go
```

### ❌ Anti-Pattern 2: Large Interfaces

**What to avoid:**

```go
// ❌ WRONG - 12+ methods
type ICombatController interface {
    InitializeCombat(factionIDs []ecs.EntityID) error
    CheckVictoryCondition() *VictoryInfo
    GetCurrentFaction() ecs.EntityID
    GetCurrentRound() int
    EndTurn() error
    ExecuteAttack(attackerID, defenderID ecs.EntityID) *AttackResult
    GetValidMovementTiles(squadID ecs.EntityID) []coords.LogicalPosition
    MoveSquad(squadID ecs.EntityID, newPos coords.LogicalPosition) *MoveResult
    GetAliveSquadsInFaction(factionID ecs.EntityID) []ecs.EntityID
    GetSquadInfo(squadID ecs.EntityID) *SquadInfo
    IsSquadPlayerControlled(squadID ecs.EntityID) bool
    ExecuteAITurn(factionID ecs.EntityID) *AITurnResult
}
```

**What to do instead:**

```go
// ✅ CORRECT - Multiple small interfaces
type TurnManager interface {
    GetCurrentFaction() ecs.EntityID
    GetCurrentRound() int
    EndTurn() error
}

type Attacker interface {
    ExecuteAttack(attackerID, defenderID ecs.EntityID) combat.AttackResult
}

type MovementProvider interface {
    GetValidMovementTiles(squadID ecs.EntityID) []coords.LogicalPosition
    MoveSquad(squadID ecs.EntityID, pos coords.LogicalPosition) error
}
```

**Validation:**

```go
// Add this test to check interface size
func TestInterfaceSizes(t *testing.T) {
    // Use reflection to count methods
    turnMgrType := reflect.TypeOf((*TurnManager)(nil)).Elem()
    if turnMgrType.NumMethod() > 3 {
        t.Errorf("TurnManager has %d methods, should be <= 3", turnMgrType.NumMethod())
    }

    attackerType := reflect.TypeOf((*Attacker)(nil)).Elem()
    if attackerType.NumMethod() > 3 {
        t.Errorf("Attacker has %d methods, should be <= 3", attackerType.NumMethod())
    }
}
```

### ❌ Anti-Pattern 3: DTO Proliferation

**What to avoid:**

```go
// ❌ WRONG - Unnecessary DTO wrapper
type AttackResultDTO struct {
    Success      bool
    Damage       int
    TargetKilled bool
}

func (cf *CombatFacade) ExecuteAttack(...) *AttackResultDTO {
    result := cf.service.ExecuteAttack(...)

    // Manual conversion - waste of effort!
    return &AttackResultDTO{
        Success:      result.Success,
        Damage:       result.Damage,
        TargetKilled: result.TargetKilled,
    }
}
```

**What to do instead:**

```go
// ✅ CORRECT - Return concrete type directly
func (cf *CombatFacade) ExecuteAttack(
    attackerID, defenderID ecs.EntityID,
) combat.AttackResult {
    return cf.service.CombatActSystem.ExecuteAttackAction(attackerID, defenderID)
}

// GUI extracts what it needs
result := facade.ExecuteAttack(a, d)
if result.Success {
    log.Printf("%d damage", result.Damage)
}
```

**Validation:**

```bash
# Should find NO files named *dto*.go in facades
find game_main/gamefacade -name "*dto*.go"

# Should return empty
```

### ❌ Anti-Pattern 4: Command Objects

**What to avoid:**

```go
// ❌ WRONG - Command pattern (not Go idiom)
type Command interface {
    Execute() error
    Undo() error
    Redo() error
}

type AttackCommand struct {
    attackerID ecs.EntityID
    defenderID ecs.EntityID
    service    *CombatService
}

func (ac *AttackCommand) Execute() error {
    return ac.service.ExecuteAttack(ac.attackerID, ac.defenderID)
}

// Usage
mediator.Execute(NewAttackCommand(attacker, defender))
```

**What to do instead:**

```go
// ✅ CORRECT - Direct function call
err := combatService.ExecuteAttack(attacker, defender)

// ✅ CORRECT - If you need deferred execution, use closure
action := func() error {
    return combatService.ExecuteAttack(attacker, defender)
}

// Execute later
err := action()
```

**Validation:**

```bash
# Should find NO command pattern files
grep -r "type.*Command struct" gui/ game_main/

# Should return empty
```

### ❌ Anti-Pattern 5: Event Bus

**What to avoid:**

```go
// ❌ WRONG - Event bus (not Go stdlib pattern)
type EventBus struct {
    subscribers map[string][]EventHandler
}

func (eb *EventBus) Publish(eventType string, data interface{})
func (eb *EventBus) Subscribe(eventType string, handler EventHandler)

// Usage
eventBus.Subscribe("attack_complete", handleAttack)
eventBus.Publish("attack_complete", attackData)
```

**What to do instead:**

```go
// ✅ CORRECT Option 1 - Synchronous callback
type CombatMode struct {
    onAttackComplete func(result combat.AttackResult)
}

result := service.ExecuteAttack(a, d)
if cm.onAttackComplete != nil {
    cm.onAttackComplete(result)
}

// ✅ CORRECT Option 2 - Channel (if async needed)
type CombatMode struct {
    events chan CombatEvent
}

go func() {
    result := service.ExecuteAttack(a, d)
    cm.events <- CombatEvent{Type: "attack", Result: result}
}()
```

**Validation:**

```bash
# Should find NO event bus
grep -r "type EventBus" game_main/ gui/

# Should return empty
```

### ❌ Anti-Pattern 6: Service Locator

**What to avoid:**

```go
// ❌ WRONG - Service locator (hides dependencies)
type GameSession struct {
    combatService *CombatService
    squadService  *SquadService
    // ... 10 more services
}

func (gs *GameSession) GetCombatService() *CombatService {
    return gs.combatService
}

// Hidden dependency - unclear what CombatMode needs
cm := NewCombatMode(gameSession)
cm.Initialize(ctx)
```

**What to do instead:**

```go
// ✅ CORRECT - Explicit dependency injection
func NewCombatMode(
    turns TurnManager,
    combat Attacker,
    movement MovementProvider,
) *CombatMode {
    return &CombatMode{
        turns:    turns,
        combat:   combat,
        movement: movement,
    }
}

// Clear what CombatMode depends on
cm := NewCombatMode(facade, facade, facade)
```

**Validation:**

```bash
# Should find NO GameSession or ServiceLocator
grep -r "type GameSession struct" game_main/ gui/
grep -r "type ServiceLocator" game_main/ gui/

# Should return empty
```

---

## SUCCESS METRICS & VALIDATION

### Code Quality Metrics

#### Metric 1: Import Dependency Analysis

**Before Refactoring:**

```bash
# Count tactical imports in GUI
grep -r "game_main/tactical" gui/ | wc -l
# Expected: ~30-40 imports
```

**After Enhanced Approach 3:**

```bash
# Count tactical imports in GUI (excluding test files)
find gui -name "*.go" -not -name "*_test.go" -exec grep -l "game_main/tactical" {} \; | wc -l
# Expected: 0 imports ✅
```

**Target:** ZERO tactical imports in GUI production code.

#### Metric 2: Interface Size Distribution

**Target Distribution:**

| Interface Size | Count | Percentage |
|---------------|-------|------------|
| 1 method      | 2-3   | 20-30%     |
| 2 methods     | 4-5   | 40-50%     |
| 3 methods     | 2-3   | 20-30%     |
| 4+ methods    | 0-1   | 0-10%      |

**Validation Script:**

```go
// tools/check_interface_sizes.go
package main

import (
    "fmt"
    "go/ast"
    "go/parser"
    "go/token"
    "os"
)

func main() {
    fset := token.NewFileSet()

    // Parse combat_interfaces.go
    f, err := parser.ParseFile(fset, "gui/guicombat/combat_interfaces.go", nil, 0)
    if err != nil {
        panic(err)
    }

    for _, decl := range f.Decls {
        if gen, ok := decl.(*ast.GenDecl); ok && gen.Tok == token.TYPE {
            for _, spec := range gen.Specs {
                if ts, ok := spec.(*ast.TypeSpec); ok {
                    if iface, ok := ts.Type.(*ast.InterfaceType); ok {
                        methodCount := len(iface.Methods.List)
                        fmt.Printf("%s: %d methods\n", ts.Name.Name, methodCount)

                        if methodCount > 3 {
                            fmt.Printf("  WARNING: Interface too large!\n")
                        }
                    }
                }
            }
        }
    }
}
```

#### Metric 3: Test Coverage

**Before Refactoring:**

```bash
go test -cover ./gui/guicombat/...
# Expected: ~20-30% (hard to test with concrete dependencies)
```

**After Enhanced Approach 3:**

```bash
go test -cover ./gui/guicombat/...
# Target: >80% (easy to test with mocks)
```

**Validation:**

```bash
go test -coverprofile=coverage.out ./gui/guicombat/...
go tool cover -func=coverage.out
# Check that critical functions (handleEndTurn, handleAttackAction) are covered
```

#### Metric 4: Test Execution Speed

**Before Refactoring:**

```bash
go test -v ./gui/guicombat/...
# Expected: ~2-5 seconds (requires ECS setup)
```

**After Enhanced Approach 3:**

```bash
go test -v ./gui/guicombat/...
# Target: <500ms (mocks, no ECS)
```

### Compile-Time Safety Validation

#### Check 1: Interface Compliance

```bash
go test ./gui/guicombat/... ./gui/guisquads/...
# Run compliance tests (TestCombatFacade_ImplementsInterfaces)
# Should PASS
```

#### Check 2: No Tactical Imports

```bash
# Script: tools/check_imports.sh
#!/bin/bash

echo "Checking for tactical imports in GUI..."

# Find all Go files in GUI (excluding tests)
GUI_FILES=$(find gui -name "*.go" -not -name "*_test.go")

VIOLATIONS=0
for file in $GUI_FILES; do
    if grep -q "game_main/tactical" "$file"; then
        echo "❌ VIOLATION: $file imports tactical package"
        VIOLATIONS=$((VIOLATIONS + 1))
    fi
done

if [ $VIOLATIONS -eq 0 ]; then
    echo "✅ PASS: No tactical imports in GUI"
    exit 0
else
    echo "❌ FAIL: Found $VIOLATIONS files with tactical imports"
    exit 1
fi
```

#### Check 3: Facade Methods Match Interfaces

```go
// tools/check_facade_completeness_test.go
package main

import (
    "testing"
    "game_main/game_main/gamefacade"
    "game_main/gui/guicombat"
    "game_main/gui/guisquads"
)

// Verify CombatFacade implements all combat interfaces
func TestCombatFacadeCompleteness(t *testing.T) {
    var facade *gamefacade.CombatFacade

    var _ guicombat.TurnManager = facade
    var _ guicombat.Attacker = facade
    var _ guicombat.MovementProvider = facade
    var _ guicombat.SquadQuerier = facade
    var _ guicombat.VictoryChecker = facade
    var _ guicombat.CombatInitializer = facade

    t.Log("✅ CombatFacade implements all combat interfaces")
}

// Verify SquadFacade implements all squad interfaces
func TestSquadFacadeCompleteness(t *testing.T) {
    var facade *gamefacade.SquadFacade

    var _ guisquads.SquadBuilder = facade
    var _ guisquads.SquadDeployer = facade
    var _ guisquads.SquadQuerier = facade

    t.Log("✅ SquadFacade implements all squad interfaces")
}
```

### Runtime Validation

#### Playthrough Checklist

- [ ] Launch game successfully
- [ ] Enter combat mode from exploration
- [ ] Display turn order correctly
- [ ] Execute attack action
- [ ] Execute move action
- [ ] End turn successfully
- [ ] Detect victory condition
- [ ] Return to exploration mode
- [ ] Enter squad deployment mode
- [ ] Deploy squad to map
- [ ] Return to exploration mode
- [ ] Enter squad editor mode
- [ ] Add unit to squad
- [ ] Remove unit from squad
- [ ] Validate squad composition
- [ ] Save and load game (if applicable)

#### Performance Checklist

- [ ] Combat mode renders at 60fps
- [ ] No frame drops during attack animations
- [ ] Turn end is responsive (<100ms)
- [ ] Squad list updates smoothly
- [ ] No allocation spikes (profile with pprof)

```bash
# Profile memory allocations
go test -memprofile=mem.prof ./game_main/gamefacade/...
go tool pprof mem.prof
# Check that facade methods don't allocate excessively
```

---

## CONCLUSION

### Overall Verdict

Enhanced Approach 3 represents the **optimal Go-idiomatic solution** for decoupling TinkerRogue's GUI from game state:

1. **Incremental** - Low-risk phased approach (can stop after Phase 1)
2. **Go-Idiomatic** - Matches stdlib patterns (`io.Reader`, `http.Handler`)
3. **Testable** - Easy mocking, fast tests (<500ms)
4. **Simple** - No event buses, commands, or DTOs
5. **Maintainable** - Clear dependencies, small interfaces

### Key Takeaways

1. **Consumer-Defined Interfaces Are Non-Negotiable**
   - Interfaces belong in `gui/guicombat/`, not `tactical/` or `gui/interfaces/`
   - This is not a preference, it's **Go FAQ guidance**
   - Violating this creates wrong dependency direction

2. **Small Interfaces Are Superior**
   - 2-3 methods is the sweet spot
   - Large interfaces (12+ methods) are Java/C# thinking, not Go
   - "Bigger interface, weaker abstraction" is a Go proverb for a reason

3. **Don't Fight the Stdlib**
   - Go doesn't use DTOs - return concrete types
   - Go doesn't use commands - call functions directly
   - Go doesn't use event buses - use channels or callbacks
   - When in doubt, check stdlib for patterns

4. **Incremental Is Better Than Perfect**
   - Phase 1 (facades) provides immediate value
   - Phase 2 (interfaces) adds testability
   - Phase 3 (refactoring) completes decoupling
   - Each phase is shippable

5. **Testing Drives Design**
   - If you can't test it, refactor it
   - Mocks should be ~10 lines, not 100
   - Tests should run in milliseconds, not seconds
   - Consumer-defined interfaces make this trivial

### Next Steps

1. **Review This Document**
   - Share with team
   - Discuss any concerns
   - Clarify Go idioms if needed

2. **Create Feature Branch**
   ```bash
   git checkout -b feature/gui-refactor-enhanced-approach-3
   ```

3. **Start Phase 1**
   - Day 1: Create `combat_facade.go`
   - Day 2: Create `squad_facade.go` and migrate CombatMode
   - Day 3: Define all interfaces
   - Checkpoint: All tests pass, game runs

4. **Iterate Through Phases**
   - Phase 2: Define interfaces (1 day)
   - Phase 3: Refactor GUI (1.5 days)
   - Phase 4: Add tests (1 day)
   - Total: ~5 days of focused work

5. **Validate Success**
   - Run validation scripts
   - Check metrics (import count, test coverage, speed)
   - Full playthrough
   - Code review

### Final Thoughts

Enhanced Approach 3 is not just a refactoring - it's an **education in Go idioms**. By following stdlib patterns, we get:

- **Testability** (small interfaces, easy mocks)
- **Maintainability** (clear dependencies, no hidden coupling)
- **Simplicity** (no over-engineering, no enterprise patterns)
- **Flexibility** (interfaces evolve with consumers)

This approach respects Go's philosophy: **simple, obvious, and effective**.

---

## REFERENCES

### Official Go Documentation

- [Effective Go](https://golang.org/doc/effective_go)
- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- [Go Proverbs](https://go-proverbs.github.io/)
- [Go FAQ: Interfaces](https://golang.org/doc/faq#interfaces)

### Go Interface Patterns

- [Accept Interfaces, Return Structs](https://bryanftan.medium.com/accept-interfaces-return-structs-in-go-d4cab29a301b)
- [Go interfaces: the tricky parts](https://stianeikeland.wordpress.com/2017/06/09/go-interfaces-the-tricky-parts/)
- [Practical Go: Real world advice](https://dave.cheney.net/practical-go/presentations/qcon-china.html)

### Performance Resources

- [Go Performance Tips](https://github.com/dgryski/go-perfbook)
- [High Performance Go Workshop](https://dave.cheney.net/high-performance-go-workshop/dotgo-paris.html)

### Testing in Go

- [Table Driven Tests](https://github.com/golang/go/wiki/TableDrivenTests)
- [Testing Techniques](https://talks.golang.org/2014/testing.slide)

### Game Development in Go

- [Ebiten Best Practices](https://ebiten.org/documents/)
- [Go Game Development Patterns](https://threedots.tech/)

---

**END OF DOCUMENT**

**Generated:** 2026-01-03
**Reviewer:** Go Standards Expert
**Document Version:** 1.0
**Total Length:** ~2000 lines
**File:** `analysis/enhanced_incremental_approach_go_idioms_20260103.md`
