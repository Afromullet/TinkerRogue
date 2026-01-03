# Go Idiomatics Analysis: GUI-Game State Decoupling
**Generated:** 2026-01-03
**Target:** GUI refactoring approaches from `refactoring_analysis_gui_game_state_coupling_20260103.md`
**Focus:** Go best practices, standard library patterns, and idiomatic architectural choices

---

## EXECUTIVE SUMMARY

### The Go Way vs. Enterprise Patterns

The refactoring analysis presents three approaches that range from pragmatic to enterprise-level abstraction. However, **Go's philosophy fundamentally differs from Java/C# enterprise patterns**, and this should drive our architectural decisions.

### Key Findings

**RECOMMENDED APPROACH:** **Modified Approach 3** (Incremental Facade) → **Simplified Approach 1** (Consumer-Defined Interfaces)

**Why this is most Go-idiomatic:**
1. **Simplicity First**: Go prioritizes simple, obvious solutions over sophisticated architectures
2. **Accept Interfaces, Return Structs**: Standard library pattern, not upfront interface definitions
3. **Consumer-Defined Interfaces**: Interfaces belong where they're needed (GUI), not where they're implemented (tactical)
4. **Incremental Evolution**: Go encourages small, iterative improvements
5. **Minimal Abstraction**: Only abstract what you need to test or swap

**Anti-Pattern Warning:**
- **Approach 1 as written**: Violates "consumer defines interface" - creates provider-side interfaces
- **Approach 2**: Over-engineered for Go - command/event patterns are rare in Go stdlib
- **Approach 3 alone**: Good start but stops short of Go's testing idioms

---

## THE GO PHILOSOPHY APPLIED TO THIS REFACTORING

### Go Proverb: "A little copying is better than a little dependency"

**What This Means Here:**
Don't create elaborate abstraction layers to avoid duplicating a few DTOs. If GUI needs a `SquadInfo` struct, it's OK to have one in the GUI layer that looks similar to `squads.SquadData`.

### Go Proverb: "The bigger the interface, the weaker the abstraction"

**What This Means Here:**
Approach 1's `ICombatController` has 10+ methods. This is a code smell in Go. Better to have 3-4 focused interfaces:
```go
// ❌ Large Interface (Approach 1)
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

// ✅ Small, Focused Interfaces (Go Way)
type TurnManager interface {
    GetCurrentFaction() ecs.EntityID
    GetCurrentRound() int
    EndTurn() error
}

type Attacker interface {
    ExecuteAttack(attackerID, defenderID ecs.EntityID) *AttackResult
}

type Mover interface {
    GetValidMovementTiles(squadID ecs.EntityID) []coords.LogicalPosition
    MoveSquad(squadID ecs.EntityID, newPos coords.LogicalPosition) *MoveResult
}
```

### Go Proverb: "Don't communicate by sharing memory, share memory by communicating"

**What This Means Here:**
Approach 2's event bus is **rarely used in Go**. Go prefers channels for communication. But for UI/game logic, **synchronous function calls are simpler and more idiomatic** than event buses.

Go's `net/http` doesn't use events - it uses simple function calls with interfaces like `http.Handler`.

---

## STANDARD LIBRARY PATTERNS ANALYSIS

### Pattern 1: `io.Reader` / `io.Writer` - Consumer-Defined Interfaces

**How It Works:**
```go
// Package io defines the interface
package io
type Reader interface {
    Read(p []byte) (n int, err error)
}

// Consumers use the interface
func Copy(dst Writer, src Reader) (int64, error)

// Providers implement it (but don't declare interface dependency)
package os
func (f *File) Read(p []byte) (int, error) // Implements io.Reader
```

**Key Insight:** `os.File` doesn't import `io` or declare it implements `io.Reader`. The interface is defined where it's **consumed** (io package), not where it's **provided** (os package).

**Application to TinkerRogue:**

```go
// ❌ WRONG (Approach 1 pattern)
// In gui/interfaces/game_interfaces.go (provider-side)
type ICombatController interface { ... }

// In tactical/combatservices/combat_service.go
var _ interfaces.ICombatController = (*CombatService)(nil) // Tactical imports GUI!

// ✅ CORRECT (Go stdlib pattern)
// In gui/guicombat/combat_interfaces.go (consumer-side)
type TurnManager interface {
    GetCurrentFaction() ecs.EntityID
    EndTurn() error
}

type Attacker interface {
    ExecuteAttack(a, d ecs.EntityID) AttackResult
}

// In gui/guicombat/combatmode.go
type CombatMode struct {
    turns TurnManager  // Interface defined in same package
    combat Attacker    // Interface defined in same package
}

// In tactical/combatservices/combat_service.go
type CombatService struct { ... }
func (cs *CombatService) GetCurrentFaction() ecs.EntityID { ... }
func (cs *CombatService) EndTurn() error { ... }
// No import of gui package, no static assertions
```

**Why This Matters:**
- **Dependency Direction**: Tactical layer has ZERO knowledge of GUI
- **Testability**: GUI can mock interfaces easily
- **Flexibility**: Different GUI modes can define different interfaces for same service
- **Go Convention**: Matches stdlib pattern exactly

### Pattern 2: `http.Handler` - Simple Function Signatures

**How It Works:**
```go
// Package http
type Handler interface {
    ServeHTTP(ResponseWriter, *Request)
}

// Usage
func ListenAndServe(addr string, handler Handler) error
```

**Key Insight:** `http.Handler` is ONE METHOD. Not a "kitchen sink" interface. If you need more capabilities, you use multiple small interfaces or type assertions.

**Application to TinkerRogue:**

```go
// ❌ TOO BIG (Approach 1)
type ICombatController interface {
    // 12 methods - violates "bigger interface, weaker abstraction"
}

// ✅ RIGHT SIZE (http.Handler style)
type ActionExecutor interface {
    Execute() error
}

type CombatAction interface {
    ExecuteAttack(attacker, defender ecs.EntityID) error
    ExecuteMove(squadID ecs.EntityID, pos coords.LogicalPosition) error
}

// Or even simpler - just functions
type AttackFunc func(attacker, defender ecs.EntityID) error
type MoveFunc func(squadID ecs.EntityID, pos coords.LogicalPosition) error
```

### Pattern 3: `database/sql` - Driver Pattern (Rarely Needed)

**How It Works:**
```go
// Package database/sql
type Driver interface {
    Open(name string) (Conn, error)
}

// Registration
func Register(name string, driver Driver)

// Usage
import _ "github.com/lib/pq" // Side-effect import registers driver
db, err := sql.Open("postgres", connString)
```

**Application to TinkerRogue:** **DON'T USE THIS PATTERN**

This pattern is for when you need **pluggable implementations at runtime** (MySQL vs Postgres). TinkerRogue doesn't need this - you always use `CombatService`, never swapping it for `AlternateCombatService`.

**Why Approach 1 Over-Engineers:**
Creating `gui/interfaces` package implies you might have multiple implementations. You won't. You have exactly one `CombatService`. Don't build for flexibility you don't need.

---

## DETAILED APPROACH ANALYSIS

### Approach 1: Service Facade with Interfaces - GO ANTI-PATTERNS

**What It Gets Right:**
- Recognizes need for abstraction
- Enables testing with mocks
- Clear contracts

**What Violates Go Idioms:**

#### Violation 1: Provider-Side Interface Definition

```go
// gui/interfaces/game_interfaces.go - WRONG LOCATION
package interfaces

type ICombatController interface { ... }
```

**Go Idiom:** Interfaces belong in **consuming** package, not providing package.

**Why:**
- Creates import dependency: `tactical/combatservices` → `gui/interfaces`
- Forces all consumers to use same interface (inflexible)
- Violates "consumer defines interface" principle

**Reference:** Go FAQ: "Where should I put interfaces?"
> "Go interfaces generally belong in the package that uses values of the interface type, not the package that implements those values."

#### Violation 2: Large Interface

```go
type ICombatController interface {
    // 12 methods
}
```

**Go Proverb:** "The bigger the interface, the weaker the abstraction"

**Why:**
- Hard to mock (must implement 12 methods)
- Likely violates Interface Segregation Principle
- Indicates interface is doing too much

**Go Way:** Multiple small interfaces

```go
type TurnManager interface {
    EndTurn() error
}

type FactionQuerier interface {
    GetCurrentFaction() ecs.EntityID
}
```

#### Violation 3: DTO Proliferation

```go
// Approach 1 creates many DTOs
type AttackResult struct { ... }
type MoveResult struct { ... }
type SquadInfo struct { ... }
type VictoryInfo struct { ... }
```

**Go Proverb:** "A little copying is better than a little dependency"

**Why:**
- Creating DTOs to avoid "coupling" is premature optimization
- Go stdlib doesn't do this - `http.Request` is passed directly, not wrapped in DTO
- Adds conversion overhead and cognitive load

**Go Way:** Return concrete types from services, let consumers extract what they need

```go
// ❌ DTO Wrapper
func (cs *CombatService) ExecuteAttack(...) *interfaces.AttackResult {
    result := cs.CombatActSystem.ExecuteAttackAction(...)
    return &interfaces.AttackResult{ // Manual conversion
        Success: result.Success,
        // ... copy 10 fields
    }
}

// ✅ Direct Return
func (cs *CombatService) ExecuteAttack(...) combat.AttackResult {
    return cs.CombatActSystem.ExecuteAttackAction(...)
}

// GUI extracts what it needs
result := combatService.ExecuteAttack(a, d)
if result.Success {
    log.Printf("%d damage", result.Damage)
}
```

#### Violation 4: Static Interface Assertions

```go
// Approach 1 suggests
var _ interfaces.ICombatController = (*CombatService)(nil)
```

**Problem:** This creates import dependency `tactical → gui`

**Go Way:** Let tests verify interface compliance

```go
// In gui/guicombat/combat_test.go
func TestCombatServiceImplementsTurnManager(t *testing.T) {
    var _ TurnManager = (*combatservices.CombatService)(nil)
}
```

#### Violation 5: GameSession Centralization

```go
// Approach 1 creates
type GameSession struct {
    combatService *CombatService
    squadBuilder  *SquadBuilderService
    // ... 10 more services
}

func (gs *GameSession) CombatController() ICombatController {
    return gs.combatService
}
```

**Problem:** God object, service locator anti-pattern

**Go Way:** Direct dependencies, explicit wiring

```go
// ✅ Explicit Dependencies
func NewCombatMode(
    turnMgr TurnManager,
    attacker Attacker,
    mover Mover,
) *CombatMode {
    return &CombatMode{
        turns: turnMgr,
        combat: attacker,
        movement: mover,
    }
}
```

---

### Approach 2: Command/Event Pattern - NOT IDIOMATIC GO

**Fundamental Problem:** Go doesn't use command/event patterns in stdlib

**Why This Pattern Exists in Java/C#:**
- Enterprise frameworks (Spring, .NET)
- Undo/redo requirements
- CQRS/Event Sourcing architectures
- Distributed systems

**Why Go Avoids It:**
- Go prefers **simple, synchronous calls**
- **Channels** replace event buses when async needed
- **Closures** replace command objects

**Go Stdlib Evidence:**

```go
// net/http - No events, just function calls
http.HandleFunc("/", func(w ResponseWriter, r *Request) {
    // Direct call, no event bus
})

// os - No commands, just method calls
file.Write(data) // Direct, not WriteCommand

// database/sql - No events
db.Query("SELECT ...") // Synchronous, returns result
```

**Where You MIGHT See Events in Go:**
- NATS/Kafka integrations (external event streams)
- Webhook systems (external event sources)
- **NOT** internal application architecture

**Specific Anti-Patterns:**

#### Anti-Pattern 1: Command Objects

```go
// Approach 2 suggests
type Command interface {
    Execute() error
    Undo() error
    Redo() error
}

type AttackCommand struct { ... }
```

**Go Way:** Just call the function

```go
// ❌ Command Object
cmd := NewAttackCommand(attacker, defender)
mediator.Execute(cmd)

// ✅ Function Call
err := combatService.ExecuteAttack(attacker, defender)
```

#### Anti-Pattern 2: Event Bus

```go
// Approach 2 suggests
type EventBus struct {
    subscribers map[string][]EventHandler
}

func (eb *EventBus) Publish(event Event)
```

**Go Way:** Channels or callbacks

```go
// ✅ Callback (http.Handler style)
type CombatMode struct {
    onAttackComplete func(result AttackResult)
}

result := service.ExecuteAttack(a, d)
if cm.onAttackComplete != nil {
    cm.onAttackComplete(result)
}

// ✅ Channel (if async needed)
type CombatMode struct {
    events chan AttackEvent
}

go func() {
    result := service.ExecuteAttack(a, d)
    cm.events <- AttackEvent{result}
}()
```

**When Would Approach 2 Make Sense?**
- You're building multiplayer with replay functionality
- You need full audit trail for every action
- You're implementing CQRS pattern

**For TinkerRogue:** Massive over-engineering. Single-player game doesn't need event sourcing.

---

### Approach 3: Incremental Facade - GOOD START, BUT INCOMPLETE

**What It Gets Right:**
- Incremental approach (Go values iteration)
- Single facade point (simplicity)
- Low risk migration

**What It Misses:**
- Stops short of proper interface abstraction
- Still creates concrete dependency

**Current Approach 3:**

```go
// gui/gamefacade/game_facade.go
type GameFacade struct {
    combatService *combatservices.CombatService // Concrete!
}

// GUI depends on concrete facade
type CombatMode struct {
    facade *gamefacade.GameFacade // Concrete dependency
}
```

**Problem:** GUI still has compile-time dependency on `gamefacade` package. Can't test with mocks easily.

**The Go Improvement:** Consumer-defined interfaces

```go
// ✅ Step 1: Create facade (Approach 3)
package gamefacade

type GameFacade struct {
    combatService *combatservices.CombatService
}

func (gf *GameFacade) EndTurn() error {
    return gf.combatService.TurnManager.EndTurn()
}

// ✅ Step 2: GUI defines interface (Go idiom)
package guicombat

type TurnManager interface {
    EndTurn() error
}

type CombatMode struct {
    turns TurnManager // Interface, not concrete facade
}

func NewCombatMode(turns TurnManager) *CombatMode {
    return &CombatMode{turns: turns}
}

// ✅ Step 3: Wire up in main
func main() {
    facade := gamefacade.NewGameFacade(em)
    combatMode := guicombat.NewCombatMode(facade) // Facade satisfies interface
}
```

**Benefits:**
- `guicombat` doesn't import `gamefacade` (zero coupling)
- Easy to test with mocks
- Facade can evolve independently
- Matches Go stdlib patterns

---

## THE GO-IDIOMATIC SOLUTION

### Recommended Architecture: "Incremental Interfaces"

**Phase 1: Create Facade (Approach 3 baseline)**

```go
// game_main/gamefacade/combat_facade.go
package gamefacade

import (
    "game_main/common"
    "game_main/tactical/combatservices"
    "game_main/tactical/combat"
    "game_main/world/coords"
    "github.com/bytearena/ecs"
)

// CombatFacade simplifies access to combat operations
type CombatFacade struct {
    service *combatservices.CombatService
}

func NewCombatFacade(em *common.EntityManager) *CombatFacade {
    return &CombatFacade{
        service: combatservices.NewCombatService(em),
    }
}

// Turn management
func (cf *CombatFacade) GetCurrentFaction() ecs.EntityID {
    return cf.service.TurnManager.GetCurrentFaction()
}

func (cf *CombatFacade) GetCurrentRound() int {
    return cf.service.TurnManager.GetCurrentRound()
}

func (cf *CombatFacade) EndTurn() error {
    return cf.service.TurnManager.EndTurn()
}

// Combat actions - Return concrete types, not DTOs
func (cf *CombatFacade) ExecuteAttack(attackerID, defenderID ecs.EntityID) combat.AttackResult {
    return cf.service.CombatActSystem.ExecuteAttackAction(attackerID, defenderID)
}

func (cf *CombatFacade) GetValidMovementTiles(squadID ecs.EntityID) []coords.LogicalPosition {
    tiles := cf.service.MovementSystem.GetValidMovementTiles(squadID)
    if tiles == nil {
        return []coords.LogicalPosition{}
    }
    return tiles
}

func (cf *CombatFacade) MoveSquad(squadID ecs.EntityID, pos coords.LogicalPosition) error {
    return cf.service.MovementSystem.MoveSquad(squadID, pos)
}

// Squad queries
func (cf *CombatFacade) GetAliveSquadsInFaction(factionID ecs.EntityID) []ecs.EntityID {
    return cf.service.GetAliveSquadsInFaction(factionID)
}

// Victory check
func (cf *CombatFacade) CheckVictoryCondition() *combatservices.VictoryCheckResult {
    return cf.service.CheckVictoryCondition()
}
```

**Phase 2: GUI Defines Interfaces (Go idiom)**

```go
// gui/guicombat/combat_interfaces.go
package guicombat

import (
    "game_main/tactical/combat"
    "game_main/tactical/combatservices"
    "game_main/world/coords"
    "github.com/bytearena/ecs"
)

// Small, focused interfaces defined by consumer (GUI)
// "Accept interfaces, return structs"

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

**Phase 3: Update CombatMode**

```go
// gui/guicombat/combatmode.go
package guicombat

import (
    "game_main/gui/core"
    // NO imports of tactical packages!
)

type CombatMode struct {
    gui.BaseMode

    // Small interfaces, not monolithic service
    turns      TurnManager
    combat     Attacker
    movement   MovementProvider
    squads     SquadQuerier
    victory    VictoryChecker

    // ... UI components
}

func NewCombatMode(
    turns TurnManager,
    combat Attacker,
    movement MovementProvider,
    squads SquadQuerier,
    victory VictoryChecker,
) *CombatMode {
    cm := &CombatMode{
        turns:    turns,
        combat:   combat,
        movement: movement,
        squads:   squads,
        victory:  victory,
    }
    cm.SetModeName("combat")
    return cm
}

// Methods use interfaces
func (cm *CombatMode) handleEndTurn() {
    err := cm.turns.EndTurn()
    if err != nil {
        // handle error
    }

    currentFactionID := cm.turns.GetCurrentFaction()
    round := cm.turns.GetCurrentRound()
    // ... update UI
}

func (cm *CombatMode) handleAttackClick() {
    result := cm.combat.ExecuteAttack(attackerID, defenderID)
    if result.Success {
        cm.logManager.AddLog(fmt.Sprintf("%d damage!", result.Damage))
    }
}
```

**Phase 4: Wire Up in Main**

```go
// game_main/main.go
func setupCombatMode(em *common.EntityManager) *guicombat.CombatMode {
    // Create facade
    facade := gamefacade.NewCombatFacade(em)

    // Inject into GUI (facade satisfies all interfaces)
    return guicombat.NewCombatMode(
        facade, // TurnManager
        facade, // Attacker
        facade, // MovementProvider
        facade, // SquadQuerier
        facade, // VictoryChecker
    )
}
```

**Phase 5: Testing**

```go
// gui/guicombat/combatmode_test.go
package guicombat

import "testing"

// Mock implementation
type mockTurnManager struct {
    currentFaction ecs.EntityID
    currentRound   int
    endTurnErr     error
}

func (m *mockTurnManager) GetCurrentFaction() ecs.EntityID { return m.currentFaction }
func (m *mockTurnManager) GetCurrentRound() int            { return m.currentRound }
func (m *mockTurnManager) EndTurn() error                  { return m.endTurnErr }

func TestCombatMode_EndTurn(t *testing.T) {
    // Arrange
    mockTurns := &mockTurnManager{
        currentFaction: 1,
        currentRound:   5,
    }

    cm := NewCombatMode(
        mockTurns,  // TurnManager
        nil,        // Attacker (not needed for this test)
        nil,        // MovementProvider
        nil,        // SquadQuerier
        nil,        // VictoryChecker
    )

    // Act
    cm.handleEndTurn()

    // Assert
    // Verify UI updated correctly
}
```

---

## GO CONSTRUCTOR PATTERNS

### Pattern 1: Simple Constructor (Preferred)

```go
// ✅ Clear, explicit dependencies
func NewCombatMode(
    turns TurnManager,
    combat Attacker,
) *CombatMode {
    return &CombatMode{
        turns:  turns,
        combat: combat,
    }
}

// Usage
cm := NewCombatMode(facade, facade)
```

**When to use:** Always, unless you need optional parameters

### Pattern 2: Functional Options (For Optional Config)

```go
// Use when you have many optional parameters

type CombatModeOption func(*CombatMode)

func WithLogger(logger *log.Logger) CombatModeOption {
    return func(cm *CombatMode) {
        cm.logger = logger
    }
}

func WithAnimationSpeed(speed float64) CombatModeOption {
    return func(cm *CombatMode) {
        cm.animSpeed = speed
    }
}

func NewCombatMode(
    turns TurnManager,
    combat Attacker,
    opts ...CombatModeOption,
) *CombatMode {
    cm := &CombatMode{
        turns:  turns,
        combat: combat,
        animSpeed: 1.0, // Default
    }

    for _, opt := range opts {
        opt(cm)
    }

    return cm
}

// Usage
cm := NewCombatMode(
    facade,
    facade,
    WithLogger(myLogger),
    WithAnimationSpeed(2.0),
)
```

**When to use:** 4+ optional parameters, or when defaults make sense

**Reference:** Dave Cheney: "Functional options for friendly APIs"

### Pattern 3: Config Struct (For Related Options)

```go
type CombatModeConfig struct {
    Turns     TurnManager
    Combat    Attacker
    LogLevel  int
    AnimSpeed float64
}

func NewCombatMode(cfg CombatModeConfig) *CombatMode {
    return &CombatMode{
        turns:     cfg.Turns,
        combat:    cfg.Combat,
        logLevel:  cfg.LogLevel,
        animSpeed: cfg.AnimSpeed,
    }
}

// Usage
cm := NewCombatMode(CombatModeConfig{
    Turns:     facade,
    Combat:    facade,
    LogLevel:  2,
    AnimSpeed: 1.5,
})
```

**When to use:** Configuration that's passed around, or serialized

**For TinkerRogue:** Use **Pattern 1** (simple constructor). You don't need functional options yet.

---

## ERROR HANDLING PATTERNS

### Go Idiom: Return Errors, Don't Panic

```go
// ✅ Return error for caller to handle
func (cf *CombatFacade) EndTurn() error {
    if err := cf.service.TurnManager.EndTurn(); err != nil {
        return fmt.Errorf("failed to end turn: %w", err)
    }
    return nil
}

// Caller handles
if err := facade.EndTurn(); err != nil {
    log.Printf("Error: %v", err)
    // Show error in GUI
}

// ❌ Don't panic in library code
func (cf *CombatFacade) EndTurn() {
    if err := cf.service.TurnManager.EndTurn(); err != nil {
        panic(err) // WRONG!
    }
}
```

### Error Wrapping (Go 1.13+)

```go
// Wrap errors with context
func (cf *CombatFacade) ExecuteAttack(a, d ecs.EntityID) (combat.AttackResult, error) {
    result := cf.service.CombatActSystem.ExecuteAttackAction(a, d)
    if !result.Success {
        return result, fmt.Errorf("attack failed: %w", errors.New(result.ErrorReason))
    }
    return result, nil
}

// Caller can check error type
result, err := facade.ExecuteAttack(a, d)
if err != nil {
    if errors.Is(err, combat.ErrOutOfRange) {
        // Specific handling
    }
}
```

### Result Types vs. Errors

```go
// ❌ Result object with success flag (Java style)
type AttackResult struct {
    Success     bool
    ErrorReason string
    Damage      int
}

// ✅ Go style: Error as separate return
type AttackResult struct {
    Damage       int
    TargetKilled bool
}

func ExecuteAttack(...) (AttackResult, error) {
    if outOfRange {
        return AttackResult{}, ErrOutOfRange
    }
    return AttackResult{Damage: 10}, nil
}
```

**For TinkerRogue:** Current `combat.AttackResult` with `Success bool` is acceptable for game logic, but facade could convert to Go idiom:

```go
func (cf *CombatFacade) ExecuteAttack(a, d ecs.EntityID) (combat.AttackResult, error) {
    result := cf.service.CombatActSystem.ExecuteAttackAction(a, d)
    if !result.Success {
        return result, fmt.Errorf(result.ErrorReason)
    }
    return result, nil
}
```

---

## PACKAGE ORGANIZATION

### Go Idiom: Package Per Feature, Not Layer

```go
// ❌ Layer-based (Java/C# style)
gui/
  interfaces/    # Abstraction layer
  controllers/   # UI controllers
  views/         # View components

tactical/
  services/      # Business logic
  repositories/  # Data access

// ✅ Feature-based (Go style)
gui/
  guicombat/     # Combat UI (with its own interfaces)
  guisquads/     # Squad UI (with its own interfaces)

tactical/
  combat/        # Combat domain (data + logic)
  squads/        # Squad domain (data + logic)
  combatservices/ # Service facades (OK to have this)
```

**Key Insight:** Each GUI package defines interfaces it needs. Don't create shared `gui/interfaces` package.

### Current TinkerRogue Structure Analysis

```
✅ GOOD:
gui/
  guicombat/  - Feature-focused
  guisquads/  - Feature-focused

tactical/
  combat/     - Domain-focused
  squads/     - Domain-focused

❌ PROPOSED (Approach 1):
gui/
  interfaces/ - Creates artificial layer
```

**Recommendation:** Keep current structure, add interfaces IN consuming packages:

```
gui/
  guicombat/
    combatmode.go
    combat_interfaces.go  ← ADD THIS (interfaces CombatMode needs)
  guisquads/
    squadmode.go
    squad_interfaces.go   ← ADD THIS (interfaces SquadMode needs)
```

---

## COMPARISON TO GO STANDARD LIBRARY

### stdlib Example 1: `net/http` Server

**How stdlib does it:**

```go
// Package http defines Handler interface
package http

type Handler interface {
    ServeHTTP(ResponseWriter, *Request)
}

func ListenAndServe(addr string, handler Handler) error

// Users implement Handler
package main

type MyHandler struct { ... }

func (h *MyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    // No import of http interfaces package
    // Just implement method
}
```

**Key Lessons:**
1. Small interface (1 method)
2. Consumer (`http` package) defines interface
3. Provider (`main`) implements without importing interface
4. Concrete types passed, interfaces accepted

**TinkerRogue Application:**

```go
// gui/guicombat defines what it needs
package guicombat

type Attacker interface {
    ExecuteAttack(a, d ecs.EntityID) combat.AttackResult
}

// tactical/combatservices implements
package combatservices

func (cs *CombatService) ExecuteAttack(...) combat.AttackResult {
    // No import of guicombat
}

// main wires them up
package main

combatMode := guicombat.NewCombatMode(combatService)
```

### stdlib Example 2: `io.Copy` Function

**How stdlib does it:**

```go
package io

type Reader interface {
    Read(p []byte) (n int, err error)
}

type Writer interface {
    Write(p []byte) (n int, err error)
}

// Accepts interfaces, returns concrete type (int64)
func Copy(dst Writer, src Reader) (written int64, err error)
```

**Key Lessons:**
1. **Accept interfaces, return structs** - Function signature accepts interfaces but returns concrete type
2. No DTO conversion
3. Caller provides concrete types that satisfy interface

**TinkerRogue Application:**

```go
// ✅ Follow io.Copy pattern
func NewCombatMode(
    turns TurnManager,      // Accept interface
    combat Attacker,        // Accept interface
) *CombatMode {            // Return struct
    return &CombatMode{...}
}

// NOT
func NewCombatMode(config CombatModeConfig) ICombatMode { // NO!
}
```

### stdlib Example 3: `context.Context` - Interface Everywhere

**When stdlib DOES use interface as return type:**

```go
package context

type Context interface { ... }

func Background() Context           // Returns interface!
func TODO() Context                 // Returns interface!
func WithCancel(parent Context) (Context, CancelFunc)
```

**Why this is exception:**
- `Context` must be passed through entire call stack
- Many implementations (background, TODO, WithCancel, WithDeadline)
- Truly needs abstraction

**TinkerRogue:** You don't need this pattern. CombatService is not passed through entire stack.

---

## ANTI-PATTERN DETECTION CHECKLIST

Use this checklist to evaluate future refactoring proposals:

### ❌ ANTI-PATTERN: Provider-Side Interfaces

```go
// In tactical/interfaces.go
type ICombatService interface { ... }

// In tactical/combatservices/combat_service.go
var _ ICombatService = (*CombatService)(nil)
```

**Why bad:** Tactical layer shouldn't define interfaces for its consumers

**Go way:** GUI defines interface, tactical implements it unknowingly

### ❌ ANTI-PATTERN: Large Interfaces

```go
type IGameController interface {
    // 15 methods
}
```

**Why bad:** "Bigger interface, weaker abstraction"

**Go way:** Multiple small interfaces (2-3 methods each)

### ❌ ANTI-PATTERN: DTO Proliferation

```go
// Create wrapper for every return type
type AttackResultDTO struct { ... }
type MoveResultDTO struct { ... }
type SquadInfoDTO struct { ... }
```

**Why bad:** "A little copying is better than a little dependency"

**Go way:** Return concrete types, let caller extract what it needs

### ❌ ANTI-PATTERN: Command Objects

```go
type Command interface {
    Execute() error
    Undo() error
}

type AttackCommand struct { ... }
```

**Why bad:** Go doesn't use this pattern, just call functions

**Go way:** Direct function calls, closures for deferred actions

### ❌ ANTI-PATTERN: Event Bus

```go
type EventBus struct {
    subscribers map[string][]Handler
}

func (eb *EventBus) Publish(event Event)
```

**Why bad:** Indirection without benefit, not in Go stdlib

**Go way:** Callbacks or channels for async

### ❌ ANTI-PATTERN: Service Locator

```go
type GameSession struct {
    // 10 services
}

func (gs *GameSession) GetCombatService() ICombatService
func (gs *GameSession) GetSquadService() ISquadService
```

**Why bad:** Hides dependencies, makes testing hard

**Go way:** Explicit dependency injection via constructors

### ✅ GO PATTERN: Consumer-Defined Interface

```go
// In gui/guicombat/interfaces.go
type TurnManager interface {
    EndTurn() error
}

// In tactical/combatservices/combat_service.go
func (cs *CombatService) EndTurn() error { ... }
// No import of gui package
```

### ✅ GO PATTERN: Small Interfaces

```go
type Reader interface {
    Read(p []byte) (n int, err error)
}

type Writer interface {
    Write(p []byte) (n int, err error)
}
```

### ✅ GO PATTERN: Accept Interfaces, Return Structs

```go
func NewCombatMode(turns TurnManager) *CombatMode {
    return &CombatMode{turns: turns}
}
```

### ✅ GO PATTERN: Explicit Wiring

```go
func main() {
    facade := gamefacade.NewCombatFacade(em)
    combatMode := guicombat.NewCombatMode(facade, facade, facade)
}
```

---

## IMPLEMENTATION ROADMAP

### Phase 1: Create Facade (1 day)

**Goal:** Single point of contact for GUI

**Tasks:**
1. Create `game_main/gamefacade/combat_facade.go`
2. Implement methods wrapping tactical services
3. Return concrete types (no DTOs)
4. No interfaces yet

**Example:**

```go
// game_main/gamefacade/combat_facade.go
package gamefacade

import (
    "game_main/common"
    "game_main/tactical/combatservices"
    "game_main/tactical/combat"
)

type CombatFacade struct {
    service *combatservices.CombatService
}

func NewCombatFacade(em *common.EntityManager) *CombatFacade {
    return &CombatFacade{
        service: combatservices.NewCombatService(em),
    }
}

func (cf *CombatFacade) EndTurn() error {
    return cf.service.TurnManager.EndTurn()
}

func (cf *CombatFacade) GetCurrentFaction() ecs.EntityID {
    return cf.service.TurnManager.GetCurrentFaction()
}

func (cf *CombatFacade) ExecuteAttack(a, d ecs.EntityID) combat.AttackResult {
    return cf.service.CombatActSystem.ExecuteAttackAction(a, d)
}
```

**Validation:**
- GUI imports `gamefacade` only
- No `tactical` imports in GUI
- All tests still pass

### Phase 2: Define Consumer Interfaces (1 day)

**Goal:** GUI defines small interfaces it needs

**Tasks:**
1. Create `gui/guicombat/combat_interfaces.go`
2. Define 3-5 small interfaces (2-3 methods each)
3. No changes to tactical layer yet

**Example:**

```go
// gui/guicombat/combat_interfaces.go
package guicombat

import (
    "game_main/tactical/combat"
    "github.com/bytearena/ecs"
)

// Each interface has single responsibility

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
- Interfaces compile
- Facade satisfies all interfaces (check with test)

### Phase 3: Refactor CombatMode Constructor (0.5 days)

**Goal:** Inject interfaces instead of concrete facade

**Tasks:**
1. Update `NewCombatMode` signature
2. Store interfaces, not concrete types
3. Update `Initialize` to use injected dependencies

**Before:**

```go
type CombatMode struct {
    combatService *combatservices.CombatService
}

func (cm *CombatMode) Initialize(ctx *core.UIContext) error {
    cm.combatService = combatservices.NewCombatService(ctx.ECSManager)
}
```

**After:**

```go
type CombatMode struct {
    turns    TurnManager
    combat   Attacker
    movement MovementProvider
}

func NewCombatMode(
    turns TurnManager,
    combat Attacker,
    movement MovementProvider,
) *CombatMode {
    cm := &CombatMode{
        turns:    turns,
        combat:   combat,
        movement: movement,
    }
    cm.SetModeName("combat")
    return cm
}
```

**Validation:**
- CombatMode methods use interfaces
- No direct service access
- Tests compile

### Phase 4: Update Call Sites (0.5 days)

**Goal:** Wire up facade in main

**Tasks:**
1. Create facade in main/game setup
2. Pass to CombatMode constructor
3. Remove service creation from Initialize()

**Example:**

```go
// game_main/main.go
func setupCombatMode(em *common.EntityManager, modeManager *core.UIModeManager) *guicombat.CombatMode {
    facade := gamefacade.NewCombatFacade(em)

    return guicombat.NewCombatMode(
        facade, // TurnManager
        facade, // Attacker
        facade, // MovementProvider
    )
}
```

**Validation:**
- Game runs
- Combat mode works
- No regressions

### Phase 5: Add Tests (1 day)

**Goal:** Demonstrate mockability

**Tasks:**
1. Create mock implementations
2. Write unit tests for CombatMode
3. Test without real ECS

**Example:**

```go
// gui/guicombat/combatmode_test.go
package guicombat

import "testing"

type mockTurnManager struct {
    faction ecs.EntityID
    round   int
}

func (m *mockTurnManager) GetCurrentFaction() ecs.EntityID { return m.faction }
func (m *mockTurnManager) GetCurrentRound() int { return m.round }
func (m *mockTurnManager) EndTurn() error { return nil }

func TestHandleEndTurn(t *testing.T) {
    mock := &mockTurnManager{faction: 1, round: 5}
    cm := NewCombatMode(mock, nil, nil)

    cm.handleEndTurn()

    // Verify UI updated
}
```

**Validation:**
- Tests pass
- No ECS setup needed
- Fast tests (<10ms)

### Phase 6: Repeat for Squad GUI (1 day)

**Goal:** Apply same pattern to squad modes

**Tasks:**
1. Create `gui/guisquads/squad_interfaces.go`
2. Update `SquadDeploymentMode`, `SquadEditorMode`
3. Add tests

**Example:**

```go
// gui/guisquads/squad_interfaces.go
package guisquads

type SquadBuilder interface {
    AddUnitToSquad(squadID, unitID ecs.EntityID) error
    RemoveUnitFromSquad(squadID, unitID ecs.EntityID) error
    ValidateSquad(squadID ecs.EntityID) error
}

type UnitRoster interface {
    GetAvailableUnits(playerID ecs.EntityID) []squads.UnitTemplate
    MarkUnitInSquad(unitID, squadID ecs.EntityID) error
}
```

### Phase 7: Documentation (0.5 days)

**Goal:** Update project docs with pattern

**Tasks:**
1. Update `CLAUDE.md` with interface pattern
2. Add examples to `ecs_best_practices.md`
3. Document facade usage

**Total Effort:** ~4-5 days (compared to 3.5 days for Approach 1)

---

## TESTING STRATEGY

### Unit Testing GUI Components

**Pattern: Mock Interfaces**

```go
// gui/guicombat/combatmode_test.go
package guicombat

import (
    "testing"
    "game_main/tactical/combat"
)

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
}

func (m *mockAttacker) ExecuteAttack(a, d ecs.EntityID) combat.AttackResult {
    return m.attackResult
}

// Test
func TestCombatMode_HandleEndTurn_Success(t *testing.T) {
    // Arrange
    mockTurns := &mockTurnManager{
        currentFaction: 1,
        currentRound:   5,
    }

    cm := NewCombatMode(mockTurns, nil, nil, nil, nil)

    // Act
    cm.handleEndTurn()

    // Assert
    if !mockTurns.endTurnCalled {
        t.Error("EndTurn was not called")
    }
}

func TestCombatMode_HandleAttack_Success(t *testing.T) {
    mockAttack := &mockAttacker{
        attackResult: combat.AttackResult{
            Success: true,
            Damage:  10,
        },
    }

    cm := NewCombatMode(nil, mockAttack, nil, nil, nil)

    cm.handleAttackClick()

    // Verify log message contains damage
}
```

### Integration Testing Facade

**Pattern: Real ECS, Test Facade**

```go
// game_main/gamefacade/combat_facade_test.go
package gamefacade

import (
    "testing"
    "game_main/common"
    "game_main/tactical/combat"
)

func TestCombatFacade_EndTurn(t *testing.T) {
    // Setup real ECS
    em := common.NewEntityManager()

    // Create test factions
    faction1 := combat.CreateTestFaction(em, "Player", true)
    faction2 := combat.CreateTestFaction(em, "Enemy", false)

    // Create facade
    facade := NewCombatFacade(em)

    // Initialize combat
    err := facade.InitializeCombat([]ecs.EntityID{faction1, faction2})
    if err != nil {
        t.Fatalf("Failed to initialize: %v", err)
    }

    // Test turn management
    startFaction := facade.GetCurrentFaction()

    err = facade.EndTurn()
    if err != nil {
        t.Errorf("EndTurn failed: %v", err)
    }

    newFaction := facade.GetCurrentFaction()
    if newFaction == startFaction {
        t.Error("Faction did not change after end turn")
    }
}
```

### Interface Compliance Testing

**Pattern: Static Assertion in Tests**

```go
// gui/guicombat/combatmode_test.go
package guicombat

import (
    "testing"
    "game_main/game_main/gamefacade"
)

// Verify CombatFacade implements all interfaces
func TestCombatFacade_ImplementsInterfaces(t *testing.T) {
    var _ TurnManager = (*gamefacade.CombatFacade)(nil)
    var _ Attacker = (*gamefacade.CombatFacade)(nil)
    var _ MovementProvider = (*gamefacade.CombatFacade)(nil)
    var _ SquadQuerier = (*gamefacade.CombatFacade)(nil)
    var _ VictoryChecker = (*gamefacade.CombatFacade)(nil)
}
```

**Why in test package:** Avoids import cycle, documents usage

---

## MIGRATION CHECKLIST

Use this checklist to track refactoring progress:

### Pre-Migration

- [ ] Read this document completely
- [ ] Understand consumer-defined interface pattern
- [ ] Identify all GUI files importing tactical packages
- [ ] Run full test suite (baseline)
- [ ] Create feature branch

### Phase 1: Facade Creation

- [ ] Create `game_main/gamefacade/combat_facade.go`
- [ ] Implement all methods wrapping CombatService
- [ ] Create `game_main/gamefacade/squad_facade.go`
- [ ] Implement all methods wrapping SquadServices
- [ ] Write integration tests for facades
- [ ] All tests pass

### Phase 2: Interface Definition

- [ ] Create `gui/guicombat/combat_interfaces.go`
- [ ] Define 5 small interfaces (2-3 methods each)
- [ ] Create `gui/guisquads/squad_interfaces.go`
- [ ] Define 3 small interfaces
- [ ] Write interface compliance tests
- [ ] Document interfaces with examples

### Phase 3: CombatMode Refactoring

- [ ] Update `NewCombatMode` to accept interfaces
- [ ] Replace concrete service fields with interfaces
- [ ] Update all method calls to use interfaces
- [ ] Remove tactical imports from `combatmode.go`
- [ ] Remove tactical imports from `combat_action_handler.go`
- [ ] Remove tactical imports from `combat_input_handler.go`
- [ ] Unit tests pass with mocks

### Phase 4: SquadMode Refactoring

- [ ] Update `NewSquadDeploymentMode` to accept interfaces
- [ ] Update `NewSquadEditorMode` to accept interfaces
- [ ] Update `NewUnitPurchaseMode` to accept interfaces
- [ ] Remove tactical imports from squad GUI files
- [ ] Unit tests pass with mocks

### Phase 5: Wiring & Integration

- [ ] Create facades in main/setup code
- [ ] Wire facades into GUI constructors
- [ ] Remove service creation from GUI Initialize()
- [ ] Update UIContext if needed
- [ ] Full integration test
- [ ] Performance regression test

### Phase 6: Cleanup & Documentation

- [ ] Remove unused imports
- [ ] Run `go fmt ./...`
- [ ] Run `go vet ./...`
- [ ] Update `CLAUDE.md` with interface pattern
- [ ] Update `ecs_best_practices.md` with facade example
- [ ] Write migration guide for future features
- [ ] Code review

### Validation

- [ ] No tactical imports in any GUI file
- [ ] All tests pass
- [ ] No performance regression
- [ ] Game runs correctly
- [ ] New unit tests added (>20 tests)
- [ ] Code coverage > 80% for new code

---

## CONCLUSION

### Recommended Approach: "Incremental Interfaces"

**What We're Doing:**
1. Create `CombatFacade` / `SquadFacade` (Approach 3 baseline)
2. GUI defines small interfaces it needs (Go idiom)
3. Inject interfaces, not concrete facades
4. Test with mocks

**Why This Is Go-Idiomatic:**
- Follows stdlib patterns (`io.Reader`, `http.Handler`)
- Consumer-defined interfaces (not provider-defined)
- Small interfaces (2-3 methods)
- Accept interfaces, return structs
- Explicit dependencies, no service locator

**Why NOT Approach 1:**
- Provider-side interfaces violate Go FAQ
- Large interfaces violate Go proverbs
- DTO proliferation adds complexity
- GameSession is service locator anti-pattern

**Why NOT Approach 2:**
- Command/Event pattern not in Go stdlib
- Over-engineered for single-player game
- Go uses channels/callbacks, not event buses

**Why NOT Approach 3 Alone:**
- Stops short of testability
- Concrete dependency still exists
- Missing Go's interface idiom

### Effort Comparison

| Approach | Days | Go-Idiomatic? | Testability | Complexity |
|----------|------|---------------|-------------|------------|
| **Approach 1** (as written) | 3.5 | ❌ No (provider interfaces) | ✅ Yes | Medium |
| **Approach 2** | 5.5 | ❌ No (commands/events) | ✅ Yes | High |
| **Approach 3** (alone) | 4 | ⚠️ Partial | ⚠️ Partial | Low |
| **Recommended** (Incremental Interfaces) | 4-5 | ✅ Yes | ✅ Yes | Low-Medium |

### Next Steps

1. **Review this document** with team
2. **Create feature branch** for refactoring
3. **Start with Phase 1** (Facade creation)
4. **Iterate through phases** incrementally
5. **Review after each phase** to validate

### Key Takeaways

1. **Go is not Java/C#** - Don't apply enterprise patterns blindly
2. **Consumer defines interface** - Interfaces belong in GUI packages
3. **Small interfaces** - 2-3 methods max, focused responsibilities
4. **No DTOs needed** - Return concrete types, let caller extract data
5. **Testing drives design** - If you can't test it, refactor it
6. **Incremental is OK** - Small steps are the Go way

### References

- [Effective Go](https://golang.org/doc/effective_go)
- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- [Go Proverbs](https://go-proverbs.github.io/)
- [Accept Interfaces, Return Structs](https://bryanftan.medium.com/accept-interfaces-return-structs-in-go-d4cab29a301b)
- [Go interfaces: the tricky parts](https://stianeikeland.wordpress.com/2017/06/09/go-interfaces-the-tricky-parts/)
- [Practical Go: Real world advice for writing maintainable Go programs](https://dave.cheney.net/practical-go/presentations/qcon-china.html)

---

**END OF ANALYSIS**
