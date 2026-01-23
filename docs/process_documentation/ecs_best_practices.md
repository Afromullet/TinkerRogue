# TinkerRogue ECS Architecture Guide

**Version:** 2.1
**Last Updated:** 2025-12-08
**Status:** Comprehensive Technical Documentation

This document serves as the definitive guide to Entity Component System (ECS) development in TinkerRogue. It covers architectural principles, implementation patterns, performance optimizations, and practical examples drawn from the codebase.

---

## Table of Contents

1. [Introduction](#introduction)
2. [Core ECS Principles](#core-ecs-principles)
3. [Step-by-Step: Creating a New ECS System](#step-by-step-creating-a-new-ecs-system)
4. [Component Design](#component-design)
5. [System Architecture](#system-architecture)
6. [Query Patterns](#query-patterns)
7. [Entity Lifecycle Management](#entity-lifecycle-management)
8. [Component Access Patterns](#component-access-patterns)
9. [Performance Optimizations](#performance-optimizations)
10. [File Organization](#file-organization)
11. [Reference Implementations](#reference-implementations)
12. [Integration Patterns](#integration-patterns)
13. [Common Mistakes and Anti-Patterns](#common-mistakes-and-anti-patterns)
14. [Testing ECS Code](#testing-ecs-code)
15. [Migration Guide](#migration-guide)

---

## Introduction

### What is ECS?

Entity Component System (ECS) is an architectural pattern that separates data (Components) from logic (Systems). Entities are simply unique identifiers that bind components together.

**Core Concepts:**
- **Entity**: A unique ID (type: `ecs.EntityID`) representing a game object
- **Component**: Pure data container with zero logic
- **System**: Functions that operate on components to implement behavior
- **Tag**: Query filter to find entities with specific component combinations

### Why ECS in TinkerRogue?

TinkerRogue uses ECS for:
1. **Modularity**: Add/remove capabilities by adding/removing components
2. **Performance**: Cache-friendly data layout, efficient queries
3. **Maintainability**: Clear separation of data and logic
4. **Testability**: Systems are pure functions that can be tested in isolation
5. **Flexibility**: Easy to add new features without modifying existing code

### Architecture Overview

```
┌─────────────────────────────────────────────────────────────┐
│                     EntityManager                            │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐      │
│  │   ecs.World  │  │  WorldTags   │  │ GlobalSystems│      │
│  │   Manager    │  │  (Registry)  │  │ (Position)   │      │
│  └──────────────┘  └──────────────┘  └──────────────┘      │
└─────────────────────────────────────────────────────────────┘
                            │
            ┌───────────────┼───────────────┐
            │               │               │
      ┌─────▼─────┐   ┌────▼────┐   ┌─────▼─────┐
      │  Squads   │   │  Combat │   │   Gear    │
      │  Package  │   │ Package │   │  Package  │
      │           │   │         │   │           │
      │ Components│   │Components│  │Components │
      │  Queries  │   │ Queries  │  │  Systems  │
      │  Systems  │   │ Systems  │  │           │
      └───────────┘   └──────────┘   └───────────┘
```

---

## Core ECS Principles

TinkerRogue follows five fundamental ECS principles. **All new code must adhere to these patterns.**

### Principle 1: Pure Data Components

Components contain **ZERO logic** - only data fields. No methods, no behavior.

**✅ CORRECT:**
```go
// squads/squadcomponents.go:38-50
type SquadData struct {
    SquadID       ecs.EntityID  // Unique squad identifier
    Formation     FormationType // Current formation layout
    Name          string        // Squad display name
    Morale        int           // Squad-wide morale (0-100)
    SquadLevel    int           // Average level for spawning
    TurnCount     int           // Number of turns this squad has taken
    MaxUnits      int           // Maximum squad size
    UsedCapacity  float64       // Current capacity consumed
    TotalCapacity int           // Total capacity from leader
}
```

**❌ WRONG:**
```go
type SquadData struct {
    SquadID  ecs.EntityID
    Morale   int
    // ... fields ...
}

// NO METHODS ON COMPONENTS!
func (s *SquadData) GetMorale() int {
    return s.Morale
}

func (s *SquadData) IncreaseMorale(amount int) {
    s.Morale += amount
}
```

**Why This Matters:**

1. **Data-Oriented Design**: ECS is fundamentally about separating data from logic. Components are the data layer.
2. **Cache Efficiency**: Pure data structures optimize CPU cache usage when systems iterate over components.
3. **Testability**: Systems test logic, components are just data - no need to mock.
4. **Flexibility**: Multiple systems can read/write the same component without coupling.

**Exception:**
Utility methods that perform pure calculations (no state modification) are acceptable:

```go
// squads/squadcomponents.go:98-106
type GridPositionData struct {
    AnchorRow int
    AnchorCol int
    Width     int
    Height    int
}

// ✅ ACCEPTABLE - Pure calculation, no state mutation
func (g *GridPositionData) GetOccupiedCells() [][2]int {
    var cells [][2]int
    for r := g.AnchorRow; r < g.AnchorRow+g.Height && r < 3; r++ {
        for c := g.AnchorCol; c < g.AnchorCol+g.Width && c < 3; c++ {
            cells = append(cells, [2]int{r, c})
        }
    }
    return cells
}
```

**Guideline**: If the method modifies state or implements game logic, it belongs in a system function.

---

### Principle 2: EntityID Only - No Entity Pointers

Always use `ecs.EntityID` (native int type) for relationships. **Never store entity pointers.**

**✅ CORRECT:**
```go
// squads/squadcomponents.go:82-85
type SquadMemberData struct {
    SquadID ecs.EntityID // Parent squad's entity ID
}

// gear/items.go:42-46
type Item struct {
    Properties ecs.EntityID // Status effects entity ID
    Actions    []ItemAction
    Count      int
}
```

**❌ WRONG:**
```go
type SquadMemberData struct {
    Squad *ecs.Entity  // NEVER store entity pointers!
}

type Item struct {
    PropertiesEntity *ecs.Entity  // WRONG!
}
```

**Why This Matters:**

1. **Entity Lifecycle Safety**: Entity pointers become invalid when entities are disposed. EntityIDs remain stable.
2. **Memory Safety**: Prevents dangling pointers and use-after-free bugs.
3. **Serialization**: EntityIDs are simple integers, easy to save/load.
4. **Query-Based Access**: Forces explicit querying, which is the ECS way.

**Historical Context:**

TinkerRogue has undergone several refactorings to eliminate entity pointers:

- **Inventory System (2025-10-21)**: Converted from `*ecs.Entity` to `ecs.EntityID` - eliminated dangling pointer bugs
- **TileContents (2025-11-08)**: Changed `[]*ecs.Entity` to `[]ecs.EntityID` - fixed memory leaks
- **Position System (2025-10-15)**: Used value-based map keys instead of pointer keys - 50x performance improvement

**When You Need Entity Pointers:**

Only use `common.FindEntityByID()` when you need the entity pointer for ECS operations:

```go
// Valid use case: Adding components requires entity pointer
entity := common.FindEntityByIDWithTag(manager, unitID, SquadMemberTag)
if entity != nil {
    entity.AddComponent(LeaderComponent, &LeaderData{})
    entity.AddTag(LeaderTag)
}
```

For component access, use the helper functions:

```go
// Preferred: Direct component access by ID
squadData := common.GetComponentTypeByIDWithTag[*SquadData](
    manager, squadID, SquadTag, SquadComponent)
```

---

### Principle 3: Query-Based Relationships

Discover relationships through ECS queries, **don't cache them in components**.

**✅ CORRECT:**
```go
// squads/squadqueries.go:39-55
func GetUnitIDsInSquad(squadID ecs.EntityID, squadmanager *common.EntityManager) []ecs.EntityID {
    var unitIDs []ecs.EntityID

    for _, result := range squadmanager.World.Query(SquadMemberTag) {
        unitEntity := result.Entity
        memberData := common.GetComponentType[*SquadMemberData](unitEntity, SquadMemberComponent)

        if memberData.SquadID == squadID {
            unitID := unitEntity.GetID()
            unitIDs = append(unitIDs, unitID)
        }
    }

    return unitIDs
}
```

**❌ WRONG:**
```go
type SquadData struct {
    SquadID  ecs.EntityID
    UnitIDs  []ecs.EntityID  // ❌ Cached relationship - must manually sync!
}

// Now you need synchronization logic everywhere:
func AddUnitToSquad(squadID, unitID ecs.EntityID, manager *EntityManager) {
    // ... create unit ...

    // Manual sync - error-prone and verbose
    squadData := GetSquadData(squadID, manager)
    squadData.UnitIDs = append(squadData.UnitIDs, unitID)
}

func RemoveUnitFromSquad(unitID ecs.EntityID, manager *EntityManager) {
    memberData := GetMemberData(unitID, manager)
    squadData := GetSquadData(memberData.SquadID, manager)

    // Manual sync - can get out of sync
    for i, id := range squadData.UnitIDs {
        if id == unitID {
            squadData.UnitIDs = append(squadData.UnitIDs[:i], squadData.UnitIDs[i+1:]...)
            break
        }
    }
}
```

**Why This Matters:**

1. **Always Up-to-Date**: Queries reflect current state, cached data can become stale.
2. **No Synchronization**: No need to update cached relationships when entities are added/removed.
3. **Reduced Coupling**: Components don't need to know about each other's internal structure.
4. **Bug Prevention**: Eliminates entire class of "desynced state" bugs.

**Performance Considerations:**

Query performance is excellent in practice:
- Most queries operate over 10-100 entities (tagged subsets)
- O(n) linear scan is fast when n is small
- Modern CPUs handle linear memory access efficiently

**Example Performance:**
```go
// Query 50 squad members: ~1-2 microseconds
for _, result := range manager.World.Query(SquadMemberTag) {
    // Process member
}
```

**When to Cache:**

Only cache if profiling shows a specific query is a bottleneck AND:
1. Query is called in tight inner loop (e.g., every frame for rendering)
2. Cached data is read-only or rarely modified
3. You can prove with benchmarks that caching improves performance

**If you must cache, follow this pattern:**

```go
type SystemWithCache struct {
    manager *common.EntityManager
    cache   map[ecs.EntityID][]ecs.EntityID
    dirty   bool
}

func (s *SystemWithCache) GetUnitsInSquad(squadID ecs.EntityID) []ecs.EntityID {
    if s.dirty {
        s.rebuildCache()
        s.dirty = false
    }
    return s.cache[squadID]
}

func (s *SystemWithCache) OnEntityAdded() {
    s.dirty = true  // Invalidate cache
}
```

**Document why caching is needed:**
```go
// PERFORMANCE: This query is called 1000+ times per frame in rendering loop.
// Profiling showed 40% of frame time spent in squad queries.
// Caching reduced query time from 12ms to 0.1ms per frame.
```

---

### Principle 4: System-Based Logic

All behavior belongs in **system functions**, not component methods.

**✅ CORRECT:**
```go
// squads/squadcombat.go:20-106
func ExecuteSquadAttack(attackerSquadID, defenderSquadID ecs.EntityID,
    squadmanager *common.EntityManager) *CombatResult {

    result := &CombatResult{
        DamageByUnit: make(map[ecs.EntityID]int),
        UnitsKilled:  []ecs.EntityID{},
    }

    // Calculate distance between squads
    squadDistance := GetSquadDistance(attackerSquadID, defenderSquadID, squadmanager)
    if squadDistance < 0 {
        return result
    }

    // Get attacker units
    attackerUnitIDs := GetUnitIDsInSquad(attackerSquadID, squadmanager)

    // Process each attacker
    for _, attackerID := range attackerUnitIDs {
        // Check if unit is alive
        attackerAttr := common.GetAttributesByIDWithTag(squadmanager, attackerID, SquadMemberTag)
        if attackerAttr == nil || attackerAttr.CurrentHealth <= 0 {
            continue
        }

        // Check attack range
        if !squadmanager.HasComponentByIDWithTag(attackerID, SquadMemberTag, AttackRangeComponent) {
            continue
        }
        rangeData := common.GetComponentTypeByIDWithTag[*AttackRangeData](
            squadmanager, attackerID, SquadMemberTag, AttackRangeComponent)
        if rangeData.Range < squadDistance {
            continue
        }

        // Get targeting data
        targetRowData := common.GetComponentTypeByIDWithTag[*TargetRowData](
            squadmanager, attackerID, SquadMemberTag, TargetRowComponent)

        // Select targets and apply damage
        // ... (combat logic continues)
    }

    return result
}
```

**❌ WRONG:**
```go
type SquadData struct {
    SquadID ecs.EntityID
    Morale  int
}

// ❌ NO! Logic belongs in system functions
func (s *SquadData) Attack(targetSquadID ecs.EntityID, manager *EntityManager) *CombatResult {
    // This is wrong - component has no access to other entities,
    // can't query for units, can't modify other components
}

// ❌ NO! State modification in component methods
func (s *SquadData) TakeDamage(amount int) {
    s.Morale -= amount
}
```

**Why This Matters:**

1. **Access to ECS**: System functions receive EntityManager, can query any entity.
2. **Multi-Entity Operations**: Systems coordinate between multiple entities.
3. **Testability**: Systems are functions - easy to unit test with mock EntityManager.
4. **Composability**: Different systems can operate on same components independently.

**System Function Patterns:**

```go
// Creation systems
func CreateSquad(name string, formation FormationType, manager *EntityManager) ecs.EntityID

// Update systems
func UpdateSquadCapacity(squadID ecs.EntityID, manager *EntityManager)

// Action systems
func ExecuteSquadAttack(attackerID, defenderID ecs.EntityID, manager *EntityManager) *CombatResult

// Query systems
func GetUnitIDsInSquad(squadID ecs.EntityID, manager *EntityManager) []ecs.EntityID
```

**Service Layer Pattern:**

For complex business logic, wrap system functions in service classes:

```go
// squads/squadservices/squad_builder_service.go
type SquadBuilderService struct {
    entityManager *common.EntityManager
}

func (sbs *SquadBuilderService) PlaceUnit(
    squadID ecs.EntityID,
    unit UnitTemplate,
    gridRow, gridCol int,
) *PlaceUnitResult {
    // Validation
    result := &PlaceUnitResult{UnitName: unit.Name}

    // Call base system function
    err := squads.AddUnitToSquad(squadID, sbs.entityManager, unit, gridRow, gridCol)
    if err != nil {
        result.Error = err.Error()
        return result
    }

    // Post-processing
    result.Success = true
    result.RemainingCapacity = squads.GetSquadRemainingCapacity(squadID, sbs.entityManager)
    return result
}
```

---

### Principle 5: Value-Based Map Keys

Use **value types** as map keys, not pointers. This is critical for performance.

**✅ CORRECT:**
```go
// systems/positionsystem.go:17-20
type PositionSystem struct {
    manager     *ecs.Manager
    spatialGrid map[coords.LogicalPosition][]ecs.EntityID  // ✅ Value key
}
```

**❌ WRONG:**
```go
type PositionSystem struct {
    manager     *ecs.Manager
    spatialGrid map[*coords.LogicalPosition][]ecs.EntityID  // ❌ Pointer key
}
```

**Performance Impact:**

**Measured Results** (Position System Refactor, 2025-10-15):
- **Before** (pointer keys): 500µs per 1000 lookups
- **After** (value keys): 10µs per 1000 lookups
- **Improvement**: **50x faster**

**Why Value Keys Are Faster:**

1. **Struct Equality**: Go compiler optimizes struct field comparison
2. **No Pointer Chasing**: Direct memory comparison, no indirection
3. **Hash Function**: Value keys use field-based hash (fast), pointer keys use address hash (slower)
4. **Lookup Pattern**:
   ```go
   // Value key: Direct comparison
   pos := coords.LogicalPosition{X: 10, Y: 20}
   entities := grid[pos]  // Fast: hash(10, 20)

   // Pointer key: Must create pointer for lookup
   pos := &coords.LogicalPosition{X: 10, Y: 20}
   entities := grid[pos]  // Slow: allocate pointer, hash(address)
   ```

**When to Use Value Keys:**

- Positions: `coords.LogicalPosition{X, Y}`
- Grid coordinates: `struct{Row, Col int}`
- Small structs (< 32 bytes)
- Immutable lookup data

**When Pointer Keys Might Be Needed:**

- Very large structs (> 256 bytes) where copying is expensive
- Structs with mutex locks or other non-copyable fields

**Guideline**: Default to value keys. Only use pointer keys if profiling proves copying is slower than pointer indirection.

---

## Step-by-Step: Creating a New ECS System

This section provides a complete walkthrough for creating a new ECS system from scratch, following TinkerRogue's architectural patterns and best practices.

### Overview

Creating a new ECS system involves:
1. Designing data components
2. Creating component registration
3. Defining tags for queries
4. Creating system functions
5. Implementing queries
6. Integrating with EntityManager
7. Writing tests

We'll build a **Status Effect System** as a practical example throughout this guide.

---

### Step 1: Design Your Components

**Goal:** Define what data your system needs to track.

**Questions to Ask:**
- What entities will have this component?
- What data needs to be stored?
- What relationships exist between entities?
- What queries will we need to perform?

**Example: Status Effect System**

We want to track temporary status effects (poison, stun, haste) on entities.

**Design Decisions:**
- Entities: Creatures (players, monsters, squad members)
- Data: Effect type, duration, intensity, source
- Relationships: Which entity applied the effect
- Queries: Find all effects on an entity, find entities with specific effects

**File: `statuseffects/components.go`**

```go
package statuseffects

import (
    "github.com/bytearena/ecs"
)

// ========================================
// COMPONENT REGISTRATION
// ========================================

var (
    // Components
    StatusEffectComponent *ecs.Component

    // Tags
    StatusEffectTag ecs.Tag
)

// ========================================
// DATA STRUCTURES
// ========================================

// StatusEffectType represents different kinds of effects
type StatusEffectType int

const (
    EffectPoison StatusEffectType = iota
    EffectStun
    EffectHaste
    EffectShield
    EffectRegen
)

// StatusEffectData - Pure data component (ZERO logic)
type StatusEffectData struct {
    EffectType   StatusEffectType // Type of effect
    TargetID     ecs.EntityID     // Entity affected by this effect
    SourceID     ecs.EntityID     // Entity that applied this effect
    Duration     int              // Turns remaining (0 = expired)
    Intensity    int              // Effect strength (1-100)
    TickDamage   int              // Damage per turn (for poison)
    AppliedTurn  int              // Game turn when applied
}

// String returns human-readable effect name
func (t StatusEffectType) String() string {
    switch t {
    case EffectPoison:
        return "Poison"
    case EffectStun:
        return "Stun"
    case EffectHaste:
        return "Haste"
    case EffectShield:
        return "Shield"
    case EffectRegen:
        return "Regeneration"
    default:
        return "Unknown"
    }
}
```

**✅ Best Practices Applied:**
- Pure data struct (no logic beyond String() for enums)
- EntityID for relationships (not entity pointers)
- Clear naming: `StatusEffectData`, `StatusEffectComponent`
- Component and Tag variables declared together
- Descriptive field comments

---

### Step 2: Register Components and Tags

**Goal:** Initialize components in the ECS world and create tags for querying.

**File: `statuseffects/components.go`** (continued)

```go
// ========================================
// INITIALIZATION
// ========================================

func InitializeComponents(manager *ecs.Manager) {
    // Create component types
    StatusEffectComponent = manager.NewComponent()

    // Create tags for querying
    StatusEffectTag = manager.NewTag(StatusEffectComponent)

    fmt.Println("Status Effect components initialized")
}
```

**File: `common/EntityManager.go`** (update)

Add to `NewEntityManager()` function:

```go
func NewEntityManager() *EntityManager {
    manager := &EntityManager{
        World:         ecs.NewManager(),
        WorldTags:     &worldtags.Tags{},
        CreatureCount: 0,
    }

    // ... existing initializations ...

    // Initialize status effect components
    statuseffects.InitializeComponents(manager.World)

    return manager
}
```

**✅ Best Practices Applied:**
- Components registered during manager initialization
- Tags created with `NewTag(components...)`
- Single source of truth for component registration

---

### Step 3: Create Entity with Components

**Goal:** Write system functions to create entities with your components.

**File: `statuseffects/system.go`**

```go
package statuseffects

import (
    "fmt"
    "game_main/common"

    "github.com/bytearena/ecs"
)

// ========================================
// CREATION SYSTEMS
// ========================================

// ApplyStatusEffect creates a new status effect entity
func ApplyStatusEffect(
    manager *common.EntityManager,
    targetID ecs.EntityID,
    sourceID ecs.EntityID,
    effectType StatusEffectType,
    duration int,
    intensity int,
    currentTurn int,
) ecs.EntityID {
    // Create new entity for the effect
    entity := manager.World.NewEntity()
    effectID := entity.ID()

    // Create effect data
    effectData := &StatusEffectData{
        EffectType:  effectType,
        TargetID:    targetID,
        SourceID:    sourceID,
        Duration:    duration,
        Intensity:   intensity,
        AppliedTurn: currentTurn,
    }

    // Set tick damage for poison
    if effectType == EffectPoison {
        effectData.TickDamage = intensity / 10 // 10% of intensity per turn
    }

    // Add components
    entity.AddComponent(StatusEffectComponent, effectData)

    // Add tags
    manager.World.AddTagToEntity(StatusEffectTag, entity)

    // Add name component for debugging
    nameData := &common.Name{NameStr: fmt.Sprintf("%s Effect", effectType.String())}
    entity.AddComponent(common.NameComponent, nameData)

    fmt.Printf("Applied %s to entity %d (duration: %d turns)\n",
        effectType.String(), targetID, duration)

    return effectID
}

// RemoveStatusEffect removes a status effect entity
func RemoveStatusEffect(manager *common.EntityManager, effectID ecs.EntityID) {
    entity := manager.World.Entity(effectID)
    if entity == nil {
        return
    }

    // Get effect data for logging
    effectData := common.GetComponentType[*StatusEffectData](entity, StatusEffectComponent)
    if effectData != nil {
        fmt.Printf("Removed %s from entity %d\n",
            effectData.EffectType.String(), effectData.TargetID)
    }

    // Dispose entity (removes all components and tags)
    manager.World.DisposeEntity(entity)
}
```

**✅ Best Practices Applied:**
- System function (not component method)
- Returns EntityID (not entity pointer)
- Uses `GetComponentType` helper for type safety
- Clear function names: `ApplyStatusEffect`, `RemoveStatusEffect`
- Proper entity cleanup with `DisposeEntity`

---

### Step 4: Implement Query Functions

**Goal:** Create read-only functions to find entities with specific components.

**File: `statuseffects/queries.go`**

```go
package statuseffects

import (
    "game_main/common"

    "github.com/bytearena/ecs"
)

// ========================================
// QUERY FUNCTIONS (Read-Only)
// ========================================

// GetActiveEffectsOnTarget returns all active effects on a target entity
func GetActiveEffectsOnTarget(manager *common.EntityManager, targetID ecs.EntityID) []*StatusEffectData {
    effects := make([]*StatusEffectData, 0)

    // Query all status effect entities (tag-scoped for performance)
    for _, result := range manager.World.Query(StatusEffectTag) {
        entity := result.Entity
        effectData := common.GetComponentType[*StatusEffectData](entity, StatusEffectComponent)

        if effectData != nil && effectData.TargetID == targetID && effectData.Duration > 0 {
            effects = append(effects, effectData)
        }
    }

    return effects
}

// HasEffect checks if an entity has a specific type of effect
func HasEffect(manager *common.EntityManager, targetID ecs.EntityID, effectType StatusEffectType) bool {
    for _, result := range manager.World.Query(StatusEffectTag) {
        entity := result.Entity
        effectData := common.GetComponentType[*StatusEffectData](entity, StatusEffectComponent)

        if effectData != nil &&
           effectData.TargetID == targetID &&
           effectData.EffectType == effectType &&
           effectData.Duration > 0 {
            return true
        }
    }

    return false
}

// GetAllExpiredEffects returns effect IDs that have expired
func GetAllExpiredEffects(manager *common.EntityManager) []ecs.EntityID {
    expiredIDs := make([]ecs.EntityID, 0)

    for _, result := range manager.World.Query(StatusEffectTag) {
        entity := result.Entity
        effectData := common.GetComponentType[*StatusEffectData](entity, StatusEffectComponent)

        if effectData != nil && effectData.Duration <= 0 {
            expiredIDs = append(expiredIDs, entity.ID())
        }
    }

    return expiredIDs
}

// GetEffectData retrieves effect data by effect entity ID
func GetEffectData(manager *common.EntityManager, effectID ecs.EntityID) *StatusEffectData {
    entity := common.FindEntityByIDWithTag(manager, effectID, StatusEffectTag)
    if entity == nil {
        return nil
    }

    return common.GetComponentType[*StatusEffectData](entity, StatusEffectComponent)
}

// CountEffectsOnTarget returns the number of active effects on an entity
func CountEffectsOnTarget(manager *common.EntityManager, targetID ecs.EntityID) int {
    count := 0

    for _, result := range manager.World.Query(StatusEffectTag) {
        entity := result.Entity
        effectData := common.GetComponentType[*StatusEffectData](entity, StatusEffectComponent)

        if effectData != nil && effectData.TargetID == targetID && effectData.Duration > 0 {
            count++
        }
    }

    return count
}
```

**✅ Best Practices Applied:**
- Queries are read-only (don't modify data)
- Tag-scoped queries for performance
- Type-safe component access
- Clear naming: `Get*`, `Has*`, `Count*`
- Documented return values

---

### Step 5: Implement Update Systems

**Goal:** Create functions that modify component data based on game logic.

**File: `statuseffects/system.go`** (continued)

```go
// ========================================
// UPDATE SYSTEMS
// ========================================

// TickAllEffects decrements duration and applies per-turn effects
// Returns number of effects that expired
func TickAllEffects(manager *common.EntityManager, currentTurn int) int {
    expiredCount := 0
    effectsToRemove := make([]ecs.EntityID, 0)

    // Process all active effects
    for _, result := range manager.World.Query(StatusEffectTag) {
        entity := result.Entity
        effectData := common.GetComponentType[*StatusEffectData](entity, StatusEffectComponent)

        if effectData == nil || effectData.Duration <= 0 {
            continue
        }

        // Decrement duration
        effectData.Duration--

        // Apply per-turn effects
        applyEffectTick(manager, effectData)

        // Mark expired effects for removal
        if effectData.Duration <= 0 {
            effectsToRemove = append(effectsToRemove, entity.ID())
            expiredCount++
        }
    }

    // Remove expired effects
    for _, effectID := range effectsToRemove {
        RemoveStatusEffect(manager, effectID)
    }

    return expiredCount
}

// applyEffectTick applies the per-turn effect to the target
func applyEffectTick(manager *common.EntityManager, effectData *StatusEffectData) {
    targetEntity := manager.World.Entity(effectData.TargetID)
    if targetEntity == nil {
        return
    }

    switch effectData.EffectType {
    case EffectPoison:
        // Apply poison damage
        if effectData.TickDamage > 0 {
            applyDamage(manager, effectData.TargetID, effectData.TickDamage)
            fmt.Printf("Poison deals %d damage to entity %d\n",
                effectData.TickDamage, effectData.TargetID)
        }

    case EffectRegen:
        // Apply healing
        healAmount := effectData.Intensity / 10
        applyHealing(manager, effectData.TargetID, healAmount)
        fmt.Printf("Regen heals %d to entity %d\n",
            healAmount, effectData.TargetID)

    case EffectStun:
        // Stun effect is checked by combat system, no tick needed

    case EffectHaste:
        // Haste effect is checked by movement system, no tick needed

    case EffectShield:
        // Shield effect is checked by damage system, no tick needed
    }
}

// applyDamage reduces target's HP (assumes Attributes component exists)
func applyDamage(manager *common.EntityManager, targetID ecs.EntityID, damage int) {
    entity := manager.World.Entity(targetID)
    if entity == nil {
        return
    }

    // Get attributes component
    attr := common.GetComponentType[*common.Attributes](entity, common.AttributesComponent)
    if attr != nil {
        attr.CurrentHealth -= damage
        if attr.CurrentHealth < 0 {
            attr.CurrentHealth = 0
        }
    }
}

// applyHealing increases target's HP (assumes Attributes component exists)
func applyHealing(manager *common.EntityManager, targetID ecs.EntityID, healing int) {
    entity := manager.World.Entity(targetID)
    if entity == nil {
        return
    }

    // Get attributes component
    attr := common.GetComponentType[*common.Attributes](entity, common.AttributesComponent)
    if attr != nil {
        maxHP := attr.GetMaxHealth()
        attr.CurrentHealth += healing
        if attr.CurrentHealth > maxHP {
            attr.CurrentHealth = maxHP
        }
    }
}
```

**✅ Best Practices Applied:**
- System functions modify data (queries don't)
- Clear separation: tick effects, apply effects, update attributes
- Uses queries to find entities
- Modifies components directly (not through getters/setters)
- Returns useful metrics (expired count)

---

### Step 6: Integration with Game Loop

**Goal:** Call your system functions from the appropriate places in the game loop.

**File: `game_main/main.go`** or relevant game loop file

```go
// In turn processing or update loop
func processTurn(manager *common.EntityManager, currentTurn int) {
    // ... existing turn logic ...

    // Tick all status effects
    expiredCount := statuseffects.TickAllEffects(manager, currentTurn)
    if expiredCount > 0 {
        fmt.Printf("%d status effects expired\n", expiredCount)
    }

    // ... continue turn logic ...
}

// Example: Applying effect when attacked
func handlePoisonAttack(manager *common.EntityManager, attackerID, targetID ecs.EntityID, currentTurn int) {
    // Check if poison already active
    if !statuseffects.HasEffect(manager, targetID, statuseffects.EffectPoison) {
        statuseffects.ApplyStatusEffect(
            manager,
            targetID,
            attackerID,
            statuseffects.EffectPoison,
            3,    // duration: 3 turns
            20,   // intensity: 20
            currentTurn,
        )
    }
}
```

**✅ Best Practices Applied:**
- System functions called from game loop (not scattered)
- Clear integration points
- Check before applying effects (no duplicates)

---

### Step 7: Write Tests

**Goal:** Ensure your system works correctly with unit tests.

**File: `statuseffects/system_test.go`**

```go
package statuseffects_test

import (
    "game_main/common"
    "game_main/statuseffects"
    "testing"

    "github.com/bytearena/ecs"
)

func TestApplyStatusEffect(t *testing.T) {
    // Setup
    manager := common.NewEntityManager()

    // Create target entity
    target := manager.World.NewEntity()
    targetID := target.ID()

    // Create source entity
    source := manager.World.NewEntity()
    sourceID := source.ID()

    // Apply poison effect
    effectID := statuseffects.ApplyStatusEffect(
        manager,
        targetID,
        sourceID,
        statuseffects.EffectPoison,
        3, // duration
        20, // intensity
        1, // current turn
    )

    // Verify effect was created
    if effectID == 0 {
        t.Error("Expected effect ID, got 0")
    }

    // Verify effect data
    effectData := statuseffects.GetEffectData(manager, effectID)
    if effectData == nil {
        t.Fatal("Expected effect data, got nil")
    }

    if effectData.EffectType != statuseffects.EffectPoison {
        t.Errorf("Expected EffectPoison, got %v", effectData.EffectType)
    }

    if effectData.Duration != 3 {
        t.Errorf("Expected duration 3, got %d", effectData.Duration)
    }

    if effectData.TargetID != targetID {
        t.Errorf("Expected target %d, got %d", targetID, effectData.TargetID)
    }
}

func TestHasEffect(t *testing.T) {
    manager := common.NewEntityManager()

    target := manager.World.NewEntity()
    targetID := target.ID()

    source := manager.World.NewEntity()
    sourceID := source.ID()

    // Initially no effect
    if statuseffects.HasEffect(manager, targetID, statuseffects.EffectPoison) {
        t.Error("Expected no poison effect initially")
    }

    // Apply poison
    statuseffects.ApplyStatusEffect(manager, targetID, sourceID, statuseffects.EffectPoison, 3, 20, 1)

    // Now has effect
    if !statuseffects.HasEffect(manager, targetID, statuseffects.EffectPoison) {
        t.Error("Expected poison effect after applying")
    }

    // Does not have other effects
    if statuseffects.HasEffect(manager, targetID, statuseffects.EffectStun) {
        t.Error("Expected no stun effect")
    }
}

func TestTickAllEffects(t *testing.T) {
    manager := common.NewEntityManager()

    target := manager.World.NewEntity()
    targetID := target.ID()

    // Add attributes component (needed for damage)
    attr := &common.Attributes{CurrentHealth: 100}
    target.AddComponent(common.AttributesComponent, attr)

    source := manager.World.NewEntity()
    sourceID := source.ID()

    // Apply effect with duration 2
    statuseffects.ApplyStatusEffect(manager, targetID, sourceID, statuseffects.EffectPoison, 2, 20, 1)

    // Tick once - duration should be 1
    expiredCount := statuseffects.TickAllEffects(manager, 2)
    if expiredCount != 0 {
        t.Errorf("Expected 0 expired, got %d", expiredCount)
    }

    effects := statuseffects.GetActiveEffectsOnTarget(manager, targetID)
    if len(effects) != 1 {
        t.Fatalf("Expected 1 active effect, got %d", len(effects))
    }

    if effects[0].Duration != 1 {
        t.Errorf("Expected duration 1 after tick, got %d", effects[0].Duration)
    }

    // Tick again - should expire
    expiredCount = statuseffects.TickAllEffects(manager, 3)
    if expiredCount != 1 {
        t.Errorf("Expected 1 expired, got %d", expiredCount)
    }

    // No active effects remain
    effects = statuseffects.GetActiveEffectsOnTarget(manager, targetID)
    if len(effects) != 0 {
        t.Errorf("Expected 0 active effects after expiry, got %d", len(effects))
    }
}

func TestGetActiveEffectsOnTarget(t *testing.T) {
    manager := common.NewEntityManager()

    target := manager.World.NewEntity()
    targetID := target.ID()

    source := manager.World.NewEntity()
    sourceID := source.ID()

    // Apply multiple effects
    statuseffects.ApplyStatusEffect(manager, targetID, sourceID, statuseffects.EffectPoison, 3, 20, 1)
    statuseffects.ApplyStatusEffect(manager, targetID, sourceID, statuseffects.EffectHaste, 5, 30, 1)

    // Get all effects
    effects := statuseffects.GetActiveEffectsOnTarget(manager, targetID)

    if len(effects) != 2 {
        t.Errorf("Expected 2 effects, got %d", len(effects))
    }
}
```

**Run Tests:**
```bash
go test ./statuseffects/... -v
```

**✅ Best Practices Applied:**
- Test each public function
- Setup and teardown properly
- Test edge cases (no effects, multiple effects, expiry)
- Clear test names
- Use subtests for related scenarios

---

### Step 8: File Organization Checklist

Verify your package follows TinkerRogue conventions:

```
statuseffects/
├── components.go       # Component definitions and registration
├── queries.go          # Read-only query functions
├── system.go           # Creation, update, and action functions
├── system_test.go      # Unit tests
└── README.md           # Package documentation (optional)
```

**components.go:**
- [ ] Component variables declared
- [ ] Tag variables declared
- [ ] Data structures defined (pure data, zero logic)
- [ ] `InitializeComponents()` function
- [ ] Enum String() methods only

**queries.go:**
- [ ] All functions are read-only
- [ ] Functions named `Get*`, `Find*`, `Has*`, `Count*`
- [ ] Use tag-scoped queries
- [ ] Use type-safe helpers (`GetComponentType`)
- [ ] Return EntityID (not entity pointers)

**system.go:**
- [ ] Creation functions (`Create*`, `Apply*`)
- [ ] Update functions (`Update*`, `Tick*`, `Process*`)
- [ ] Deletion functions (`Remove*`, `Delete*`)
- [ ] Functions modify data (not queries)
- [ ] Clear function names and documentation

**system_test.go:**
- [ ] Test creation functions
- [ ] Test query functions
- [ ] Test update/tick functions
- [ ] Test edge cases
- [ ] Use `NewEntityManager()` for setup

---

### Complete Example: Usage in Combat

**File: `combat/combatactions.go`**

```go
package combat

import (
    "game_main/common"
    "game_main/statuseffects"

    "github.com/bytearena/ecs"
)

// ExecutePoisonAttack performs an attack that applies poison
func ExecutePoisonAttack(
    manager *common.EntityManager,
    attackerID ecs.EntityID,
    targetID ecs.EntityID,
    currentTurn int,
) {
    // Calculate damage
    damage := calculateAttackDamage(manager, attackerID, targetID)

    // Apply damage
    applyDamage(manager, targetID, damage)

    // 30% chance to apply poison
    if rollChance(30) {
        // Check if already poisoned (don't stack)
        if !statuseffects.HasEffect(manager, targetID, statuseffects.EffectPoison) {
            statuseffects.ApplyStatusEffect(
                manager,
                targetID,
                attackerID,
                statuseffects.EffectPoison,
                3,  // 3 turns
                20, // 20 intensity
                currentTurn,
            )

            fmt.Printf("Target poisoned!\n")
        }
    }
}

// ExecuteTurn processes combat turn, including status effects
func ExecuteTurn(manager *common.EntityManager, currentTurn int) {
    // ... existing combat logic ...

    // Tick all status effects at end of turn
    expiredCount := statuseffects.TickAllEffects(manager, currentTurn)
    if expiredCount > 0 {
        fmt.Printf("%d status effects expired this turn\n", expiredCount)
    }

    // ... continue turn processing ...
}
```

---

### Summary Checklist

When creating a new ECS system, follow this checklist:

**Design Phase:**
- [ ] Identify entities that will use this system
- [ ] Design component data structures (pure data)
- [ ] Plan relationships (use EntityID, not pointers)
- [ ] Define queries needed

**Implementation Phase:**
- [ ] Create `components.go` with data structures
- [ ] Register components and tags in `InitializeComponents()`
- [ ] Add initialization to `EntityManager`
- [ ] Create `queries.go` with read-only functions
- [ ] Create `system.go` with creation/update functions
- [ ] Write tests in `system_test.go`

**Integration Phase:**
- [ ] Call system functions from game loop
- [ ] Integrate with existing systems
- [ ] Add GUI queries if needed
- [ ] Document usage examples

**Quality Gates:**
- [ ] Run `go-standards-reviewer` agent
- [ ] Run `ecs-reviewer` agent
- [ ] Run `integration-validator` agent
- [ ] All tests pass (`go test ./...`)
- [ ] Code follows 5 core ECS principles

**Reference Examples:**
- Squad System: `squads/` (perfect ECS example)
- Inventory: `gear/Inventory.go` (pure component)
- Position System: `systems/positionsystem.go` (value maps, O(1) queries)

---

## Component Design

### Component Structure

Components are pure data containers registered with the ECS.

**File: `squads/squadcomponents.go`**
```go
package squads

import "github.com/bytearena/ecs"

// Component registration variables (global scope)
var (
    SquadComponent        *ecs.Component
    SquadMemberComponent  *ecs.Component
    GridPositionComponent *ecs.Component
    // ... more components

    SquadTag       ecs.Tag
    SquadMemberTag ecs.Tag
    LeaderTag      ecs.Tag  // Multi-component tag
)

// ========================================
// COMPONENT DATA STRUCTURES
// ========================================

// SquadData - Pure data, zero logic
type SquadData struct {
    SquadID       ecs.EntityID  // Native entity ID
    Formation     FormationType
    Name          string
    Morale        int
    MaxUnits      int
    UsedCapacity  float64
    TotalCapacity int
}

// SquadMemberData - Relationship via EntityID
type SquadMemberData struct {
    SquadID ecs.EntityID  // Parent reference
}

// FormationType - Enum for formation types
type FormationType int

const (
    FormationBalanced  FormationType = iota
    FormationDefensive
    FormationOffensive
    FormationRanged
)

// String() method is acceptable for display
func (f FormationType) String() string {
    switch f {
    case FormationBalanced:
        return "Balanced"
    case FormationDefensive:
        return "Defensive"
    case FormationOffensive:
        return "Offensive"
    case FormationRanged:
        return "Ranged"
    default:
        return "Unknown"
    }
}
```

### Component Initialization

Components must be registered with the ECS before use. **This is critical - unregistered components will be nil and cause panics.**

**File: `squads/squadmanager.go`**
```go
package squads

import (
    "game_main/common"
    "github.com/bytearena/ecs"
)

// Step 1: Initialize components (create ecs.Component instances)
func InitSquadComponents(manager *common.EntityManager) {
    SquadComponent = manager.World.NewComponent()
    SquadMemberComponent = manager.World.NewComponent()
    GridPositionComponent = manager.World.NewComponent()
    UnitRoleComponent = manager.World.NewComponent()
    CoverComponent = manager.World.NewComponent()
    LeaderComponent = manager.World.NewComponent()
    TargetRowComponent = manager.World.NewComponent()
    AbilitySlotComponent = manager.World.NewComponent()
    CooldownTrackerComponent = manager.World.NewComponent()
    AttackRangeComponent = manager.World.NewComponent()
    MovementSpeedComponent = manager.World.NewComponent()
}

// Step 2: Build tags for querying
func InitSquadTags(manager *common.EntityManager) {
    // Single-component tag
    SquadTag = ecs.BuildTag(SquadComponent)
    SquadMemberTag = ecs.BuildTag(SquadMemberComponent)

    // Multi-component tag (entity must have ALL components)
    LeaderTag = ecs.BuildTag(LeaderComponent, SquadMemberComponent)

    // Register tags for name-based lookup
    manager.WorldTags["squad"] = SquadTag
    manager.WorldTags["squadmember"] = SquadMemberTag
    manager.WorldTags["leader"] = LeaderTag
}

// Step 3: Main initialization entry point
func InitializeSquadData(manager *common.EntityManager) error {
    InitSquadComponents(manager)
    InitSquadTags(manager)

    // Load templates or other data
    if err := InitUnitTemplatesFromJSON(); err != nil {
        return fmt.Errorf("failed to initialize units: %w", err)
    }

    return nil
}
```

**Game Startup:**
```go
// game_main/gameinit.go
func InitializeGame() {
    manager := common.NewEntityManager()

    // Initialize all subsystems
    if err := squads.InitializeSquadData(manager); err != nil {
        log.Fatal(err)
    }
    if err := combat.InitializeCombatData(manager); err != nil {
        log.Fatal(err)
    }
    if err := gear.InitializeGearData(manager); err != nil {
        log.Fatal(err)
    }

    // Now safe to create entities with components
    return manager
}
```

### Multi-Component Tags

Tags can require multiple components, creating specialized queries:

```go
// Entity with LeaderTag must have BOTH LeaderComponent AND SquadMemberComponent
LeaderTag = ecs.BuildTag(LeaderComponent, SquadMemberComponent)

// Query only returns leaders (units that are squad members AND have leader status)
for _, result := range manager.World.Query(LeaderTag) {
    entity := result.Entity
    leaderData := common.GetComponentType[*LeaderData](entity, LeaderComponent)
    memberData := common.GetComponentType[*SquadMemberData](entity, SquadMemberComponent)

    // Both components guaranteed to exist
}
```

### Component Naming Conventions

**Data Structs:**
- Suffix with `Data`: `SquadData`, `ActionStateData`, `InventoryData`
- Exception: Common components like `Attributes`, `Name` don't need suffix

**Component Variables:**
- Suffix with `Component`: `SquadComponent`, `ActionStateComponent`

**Tags:**
- Suffix with `Tag`: `SquadTag`, `SquadMemberTag`, `LeaderTag`

**Consistency Example:**
```go
// Data struct
type SquadData struct { /* ... */ }

// Component registration variable
var SquadComponent *ecs.Component

// Tag for querying
var SquadTag ecs.Tag

// Usage
entity.AddComponent(SquadComponent, &SquadData{...})
for _, result := range manager.World.Query(SquadTag) { /* ... */ }
```

### Component Documentation

Document component purpose and relationships:

```go
// SquadData represents the squad entity's component data.
// Each squad is a separate entity with formation, morale, and capacity tracking.
// Units reference squads via SquadMemberData.SquadID.
//
// Relationships:
//   - 1 Squad : N SquadMembers (query via GetUnitIDsInSquad)
//   - Capacity calculated from leader's Leadership attribute
//   - Position tracked via separate PositionComponent
type SquadData struct {
    SquadID       ecs.EntityID  // Unique squad identifier
    Formation     FormationType // Current formation layout
    Name          string        // Squad display name
    Morale        int           // Squad-wide morale (0-100)
    SquadLevel    int           // Average level for spawning
    TurnCount     int           // Turns taken in combat
    MaxUnits      int           // Maximum squad size (typically 9)
    UsedCapacity  float64       // Capacity consumed by units
    TotalCapacity int           // Total capacity from leader
}
```

---

## System Architecture

### System Function Patterns

Systems are **stateless functions** that operate on components. They receive `EntityManager` as a parameter.

**Query Systems** (Read-Only):
```go
// squads/squadqueries.go

// GetUnitIDsInSquad returns all unit IDs belonging to a squad
func GetUnitIDsInSquad(squadID ecs.EntityID, squadmanager *common.EntityManager) []ecs.EntityID {
    var unitIDs []ecs.EntityID

    for _, result := range squadmanager.World.Query(SquadMemberTag) {
        memberData := common.GetComponentType[*SquadMemberData](result.Entity, SquadMemberComponent)
        if memberData.SquadID == squadID {
            unitIDs = append(unitIDs, result.Entity.GetID())
        }
    }

    return unitIDs
}

// GetSquadName returns the squad's name, or "Unknown Squad" if not found
func GetSquadName(squadID ecs.EntityID, squadmanager *common.EntityManager) string {
    squadEntity := GetSquadEntity(squadID, squadmanager)
    if squadEntity == nil {
        return "Unknown Squad"
    }

    squadData := common.GetComponentType[*SquadData](squadEntity, SquadComponent)
    return squadData.Name
}

// IsSquadDestroyed checks if all units are dead
func IsSquadDestroyed(squadID ecs.EntityID, squadmanager *common.EntityManager) bool {
    unitIDs := GetUnitIDsInSquad(squadID, squadmanager)

    for _, unitID := range unitIDs {
        attr := common.GetAttributesByIDWithTag(squadmanager, unitID, SquadMemberTag)
        if attr != nil && attr.CurrentHealth > 0 {
            return false  // At least one unit alive
        }
    }

    return len(unitIDs) > 0  // Destroyed if had units but all dead
}
```

**Action Systems** (Modify State):
```go
// squads/squadcombat.go

// ExecuteSquadAttack performs combat between two squads
// Returns combat result with damage dealt and units killed
func ExecuteSquadAttack(
    attackerSquadID, defenderSquadID ecs.EntityID,
    squadmanager *common.EntityManager,
) *CombatResult {
    result := &CombatResult{
        DamageByUnit: make(map[ecs.EntityID]int),
        UnitsKilled:  []ecs.EntityID{},
    }

    // Calculate distance for range checking
    squadDistance := GetSquadDistance(attackerSquadID, defenderSquadID, squadmanager)
    if squadDistance < 0 {
        return result  // Invalid positions
    }

    // Get attacker units
    attackerUnitIDs := GetUnitIDsInSquad(attackerSquadID, squadmanager)

    // Process each attacker
    for _, attackerID := range attackerUnitIDs {
        // Check if alive
        attackerAttr := common.GetAttributesByIDWithTag(squadmanager, attackerID, SquadMemberTag)
        if attackerAttr == nil || attackerAttr.CurrentHealth <= 0 {
            continue
        }

        // Check range
        rangeData := common.GetComponentTypeByIDWithTag[*AttackRangeData](
            squadmanager, attackerID, SquadMemberTag, AttackRangeComponent)
        if rangeData.Range < squadDistance {
            continue  // Out of range
        }

        // Get targets
        targetRowData := common.GetComponentTypeByIDWithTag[*TargetRowData](
            squadmanager, attackerID, SquadMemberTag, TargetRowComponent)

        var targetIDs []ecs.EntityID
        if targetRowData.Mode == TargetModeCellBased {
            // Cell-based targeting
            for _, cell := range targetRowData.TargetCells {
                cellTargets := GetUnitIDsAtGridPosition(defenderSquadID, cell[0], cell[1], squadmanager)
                targetIDs = append(targetIDs, cellTargets...)
            }
        } else {
            // Row-based targeting
            for _, row := range targetRowData.TargetRows {
                rowTargets := GetUnitIDsInRow(defenderSquadID, row, squadmanager)
                targetIDs = append(targetIDs, rowTargets...)
            }
        }

        // Apply damage to targets
        for _, defenderID := range targetIDs {
            damage := calculateUnitDamageByID(attackerID, defenderID, squadmanager)
            applyDamageToUnitByID(defenderID, damage, result, squadmanager)
        }
    }

    result.TotalDamage = sumDamageMap(result.DamageByUnit)
    return result
}

// Private helper: Calculate damage
func calculateUnitDamageByID(attackerID, defenderID ecs.EntityID, squadmanager *common.EntityManager) int {
    attackerAttr := common.GetAttributesByIDWithTag(squadmanager, attackerID, SquadMemberTag)
    defenderAttr := common.GetAttributesByIDWithTag(squadmanager, defenderID, SquadMemberTag)

    if attackerAttr == nil || defenderAttr == nil {
        return 0
    }

    // Check hit rate
    if !rollHit(attackerAttr.GetHitRate()) {
        return 0  // Miss
    }

    // Check dodge
    if rollDodge(defenderAttr.GetDodgeChance()) {
        return 0  // Dodged
    }

    // Calculate base damage
    baseDamage := attackerAttr.GetPhysicalDamage()

    // Check critical hit
    if rollCrit(attackerAttr.GetCritChance()) {
        baseDamage = int(float64(baseDamage) * 1.5)
    }

    // Apply resistance
    totalDamage := baseDamage - defenderAttr.GetPhysicalResistance()
    if totalDamage < 1 {
        totalDamage = 1  // Minimum damage
    }

    // Apply cover
    coverReduction := CalculateTotalCover(defenderID, squadmanager)
    if coverReduction > 0.0 {
        totalDamage = int(float64(totalDamage) * (1.0 - coverReduction))
        if totalDamage < 1 {
            totalDamage = 1
        }
    }

    return totalDamage
}

// Private helper: Apply damage and track kills
func applyDamageToUnitByID(unitID ecs.EntityID, damage int, result *CombatResult, squadmanager *common.EntityManager) {
    attr := common.GetAttributesByIDWithTag(squadmanager, unitID, SquadMemberTag)
    if attr == nil {
        return
    }

    attr.CurrentHealth -= damage
    result.DamageByUnit[unitID] = damage

    if attr.CurrentHealth <= 0 {
        result.UnitsKilled = append(result.UnitsKilled, unitID)
    }
}
```

**Creation Systems:**
```go
// squads/squadcreation.go (example)

// CreateSquad creates a new squad entity
func CreateSquad(name string, formation FormationType, manager *common.EntityManager) ecs.EntityID {
    // Create entity
    entity := manager.World.CreateEntity()

    // Generate unique ID
    squadID := ecs.EntityID(manager.World.NewIDPool().GetID())

    // Add components
    entity.AddComponent(SquadComponent, &SquadData{
        SquadID:       squadID,
        Formation:     formation,
        Name:          name,
        Morale:        100,
        MaxUnits:      9,
        UsedCapacity:  0,
        TotalCapacity: 6,  // Default before leader assigned
    })

    // Add tags
    entity.AddTag(SquadTag)

    return squadID
}

// CreateUnit creates a unit entity and adds to squad
func CreateUnit(squadID ecs.EntityID, template UnitTemplate, gridRow, gridCol int, manager *common.EntityManager) ecs.EntityID {
    // Create entity
    entity := manager.World.CreateEntity()
    unitID := ecs.EntityID(manager.World.NewIDPool().GetID())

    // Add squad membership
    entity.AddComponent(SquadMemberComponent, &SquadMemberData{
        SquadID: squadID,
    })

    // Add grid position
    entity.AddComponent(GridPositionComponent, &GridPositionData{
        AnchorRow: gridRow,
        AnchorCol: gridCol,
        Width:     1,
        Height:    1,
    })

    // Add attributes from template
    entity.AddComponent(common.AttributeComponent, &common.Attributes{
        Strength:      template.Strength,
        Dexterity:     template.Dexterity,
        Magic:         template.Magic,
        CurrentHealth: template.MaxHealth,
        MaxHealth:     template.MaxHealth,
    })

    // Add name
    entity.AddComponent(common.NameComponent, &common.Name{
        NameStr: template.Name,
    })

    // Add tag
    entity.AddTag(SquadMemberTag)

    // Update squad capacity
    UpdateSquadCapacity(squadID, manager)

    return unitID
}
```

**Update Systems:**
```go
// squads/squadqueries.go:192-201

// UpdateSquadCapacity recalculates and updates cached capacity values
// Call when: adding/removing units, leader changes, or leader attributes change
func UpdateSquadCapacity(squadID ecs.EntityID, squadmanager *common.EntityManager) {
    squadEntity := GetSquadEntity(squadID, squadmanager)
    if squadEntity == nil {
        return
    }

    squadData := common.GetComponentType[*SquadData](squadEntity, SquadComponent)
    squadData.TotalCapacity = GetSquadTotalCapacity(squadID, squadmanager)
    squadData.UsedCapacity = GetSquadUsedCapacity(squadID, squadmanager)
}
```

### System Naming Conventions

**Query Functions** (read-only):
- `Get*`: Retrieve single value - `GetSquadName()`, `GetLeaderID()`
- `Find*`: Search for entity - `FindActionStateEntity()`, `FindSquadByName()`
- `Is*` / `Can*`: Boolean checks - `IsSquadDestroyed()`, `CanAddUnitToSquad()`
- Plural for collections: `GetUnitIDsInSquad()`, `GetAllSquads()`

**Action Functions** (modify state):
- Verb-based: `ExecuteSquadAttack()`, `ApplyDamage()`, `CreateSquad()`
- `Update*`: State updates - `UpdateSquadCapacity()`, `UpdateTurnOrder()`
- `Add*` / `Remove*`: Entity management - `AddUnitToSquad()`, `RemoveUnitFromSquad()`

**Trigger Functions** (events):
- `CheckAndTrigger*`: Conditional execution - `CheckAndTriggerAbilities()`

### Stateless vs Stateful Systems

**Preferred: Stateless Functions**
```go
// Pure function - no state
func GetUnitIDsInSquad(squadID ecs.EntityID, manager *EntityManager) []ecs.EntityID {
    // Query and return
}

// Can be tested easily
func TestGetUnitIDsInSquad(t *testing.T) {
    manager := CreateTestEntityManager()
    squadID := CreateSquad("Test", FormationBalanced, manager)
    unitID := AddUnitToSquad(squadID, template, 0, 0, manager)

    units := GetUnitIDsInSquad(squadID, manager)
    assert.Equal(t, 1, len(units))
    assert.Equal(t, unitID, units[0])
}
```

**When Stateful Systems Are Acceptable:**

For performance-critical systems (e.g., spatial indexing), stateful systems with caching are acceptable:

```go
// systems/positionsystem.go
type PositionSystem struct {
    manager     *ecs.Manager
    spatialGrid map[coords.LogicalPosition][]ecs.EntityID  // Cache
}

func NewPositionSystem(manager *ecs.Manager) *PositionSystem {
    return &PositionSystem{
        manager:     manager,
        spatialGrid: make(map[coords.LogicalPosition][]ecs.EntityID),
    }
}

// O(1) lookup using cached spatial grid
func (ps *PositionSystem) GetEntityIDAt(pos coords.LogicalPosition) ecs.EntityID {
    if ids, ok := ps.spatialGrid[pos]; ok && len(ids) > 0 {
        return ids[0]
    }
    return 0
}

// Mutation updates cache
func (ps *PositionSystem) AddEntity(entityID ecs.EntityID, pos coords.LogicalPosition) error {
    ps.spatialGrid[pos] = append(ps.spatialGrid[pos], entityID)
    return nil
}
```

**Guidelines for Stateful Systems:**
1. **Document why state is needed** (performance, spatial indexing, etc.)
2. **Maintain cache consistency** - update on entity add/remove
3. **Provide clear API** - hide cache implementation details
4. **Global singleton only if necessary** - `common.GlobalPositionSystem`

---

## Query Patterns

### Basic Query Pattern

Standard pattern for iterating over entities with a specific tag:

```go
// Iterate over all squads
for _, result := range manager.World.Query(SquadTag) {
    entity := result.Entity
    squadData := common.GetComponentType[*SquadData](entity, SquadComponent)

    // Process squad
    fmt.Printf("Squad: %s, Morale: %d\n", squadData.Name, squadData.Morale)
}
```

### Filtered Query Pattern

Query with conditional filtering:

```go
// squads/squadqueries.go:39-55
func GetUnitIDsInSquad(squadID ecs.EntityID, squadmanager *common.EntityManager) []ecs.EntityID {
    var unitIDs []ecs.EntityID

    // Query all squad members
    for _, result := range squadmanager.World.Query(SquadMemberTag) {
        entity := result.Entity
        memberData := common.GetComponentType[*SquadMemberData](entity, SquadMemberComponent)

        // Filter by squadID
        if memberData.SquadID == squadID {
            unitIDs = append(unitIDs, entity.GetID())
        }
    }

    return unitIDs
}
```

### Multi-Component Query

Query entities that have multiple components (using multi-component tag):

```go
// Query only leaders (must have both LeaderComponent and SquadMemberComponent)
for _, result := range manager.World.Query(LeaderTag) {
    entity := result.Entity

    // Both components guaranteed to exist
    leaderData := common.GetComponentType[*LeaderData](entity, LeaderComponent)
    memberData := common.GetComponentType[*SquadMemberData](entity, SquadMemberComponent)

    fmt.Printf("Leader of squad %d with leadership %d\n",
        memberData.SquadID, leaderData.Leadership)
}
```

### Component Existence Check

Check if entity has component before accessing:

```go
for _, result := range manager.World.Query(SquadMemberTag) {
    entity := result.Entity

    // Check if has grid position component
    if entity.HasComponent(GridPositionComponent) {
        gridPos := common.GetComponentType[*GridPositionData](entity, GridPositionComponent)
        fmt.Printf("Unit at (%d, %d)\n", gridPos.AnchorRow, gridPos.AnchorCol)
    }
}
```

### Complex Filtering

Multiple filter conditions:

```go
// squads/squadqueries.go:74-97
func GetUnitIDsInRow(squadID ecs.EntityID, row int, squadmanager *common.EntityManager) []ecs.EntityID {
    var unitIDs []ecs.EntityID
    seen := make(map[ecs.EntityID]bool)

    for col := 0; col < 3; col++ {
        idsAtPos := GetUnitIDsAtGridPosition(squadID, row, col, squadmanager)
        for _, unitID := range idsAtPos {
            if !seen[unitID] {
                // Filter: Must be alive
                attr := common.GetAttributesByIDWithTag(squadmanager, unitID, SquadMemberTag)
                if attr == nil {
                    continue
                }

                if attr.CurrentHealth > 0 {
                    unitIDs = append(unitIDs, unitID)
                    seen[unitID] = true
                }
            }
        }
    }

    return unitIDs
}
```

### Find-First Pattern

Return first entity matching condition:

```go
// squads/squadqueries.go:59-72
func GetSquadEntity(squadID ecs.EntityID, squadmanager *common.EntityManager) *ecs.Entity {
    for _, result := range squadmanager.World.Query(SquadTag) {
        squadEntity := result.Entity
        squadData := common.GetComponentType[*SquadData](squadEntity, SquadComponent)

        if squadData.SquadID == squadID {
            return squadEntity  // Return first match
        }
    }

    return nil  // Not found
}
```

### Count Pattern

Count entities matching condition:

```go
func CountAliveUnits(squadID ecs.EntityID, manager *EntityManager) int {
    count := 0

    for _, result := range manager.World.Query(SquadMemberTag) {
        memberData := common.GetComponentType[*SquadMemberData](result.Entity, SquadMemberComponent)
        if memberData.SquadID != squadID {
            continue
        }

        attr := common.GetComponentType[*Attributes](result.Entity, AttributeComponent)
        if attr != nil && attr.CurrentHealth > 0 {
            count++
        }
    }

    return count
}
```

### Aggregate Pattern

Calculate aggregate values:

```go
// squads/squadqueries.go:137-151
func GetSquadUsedCapacity(squadID ecs.EntityID, squadmanager *common.EntityManager) float64 {
    unitIDs := GetUnitIDsInSquad(squadID, squadmanager)
    totalUsed := 0.0

    for _, unitID := range unitIDs {
        attr := common.GetAttributesByIDWithTag(squadmanager, unitID, SquadMemberTag)
        if attr == nil {
            continue
        }

        totalUsed += attr.GetCapacityCost()
    }

    return totalUsed
}
```

### Performance Optimization: Tag-Scoped Queries

Use specific tags instead of `AllEntitiesTag` for better performance:

```go
// ❌ SLOW - Searches ALL entities (potentially thousands)
for _, result := range manager.World.Query(AllEntitiesTag) {
    if result.Entity.GetID() == targetID {
        // Found it
    }
}

// ✅ FAST - Searches only squad members (typically 10-100)
for _, result := range manager.World.Query(SquadMemberTag) {
    if result.Entity.GetID() == targetID {
        // Found it 10-100x faster
    }
}
```

**Performance Impact:**
- `AllEntitiesTag`: O(n) where n = all entities (1000+)
- Specific tag: O(n) where n = tagged entities (10-100)
- **Speedup: 10-100x**

---

## Entity Lifecycle Management

### Entity Creation

**Basic Creation:**
```go
// Create entity
entity := manager.World.CreateEntity()

// Generate unique ID
entityID := ecs.EntityID(manager.World.NewIDPool().GetID())

// Add components
entity.AddComponent(SquadComponent, &SquadData{
    SquadID: entityID,
    Name:    "New Squad",
    Morale:  100,
})

// Add tags
entity.AddTag(SquadTag)

return entityID
```

**Creation with Position:**
```go
func CreateUnit(template UnitTemplate, pos coords.LogicalPosition, manager *EntityManager) ecs.EntityID {
    // Create entity
    entity := manager.World.CreateEntity()
    unitID := ecs.EntityID(manager.World.NewIDPool().GetID())

    // Add components
    entity.AddComponent(common.NameComponent, &common.Name{NameStr: template.Name})
    entity.AddComponent(common.AttributeComponent, &common.Attributes{
        Strength:      template.Strength,
        Dexterity:     template.Dexterity,
        CurrentHealth: template.MaxHealth,
        MaxHealth:     template.MaxHealth,
    })
    entity.AddComponent(common.PositionComponent, &pos)

    // Register in spatial grid
    common.GlobalPositionSystem.AddEntity(unitID, pos)

    return unitID
}
```

### Adding Components Dynamically

```go
// Add component to existing entity
func PromoteToLeader(unitID ecs.EntityID, manager *EntityManager) error {
    // Find entity
    entity := common.FindEntityByIDWithTag(manager, unitID, SquadMemberTag)
    if entity == nil {
        return fmt.Errorf("unit not found")
    }

    // Add leader component
    entity.AddComponent(LeaderComponent, &LeaderData{
        Leadership: 10,
        Experience: 0,
    })

    // Add leader tag (must have both LeaderComponent and SquadMemberComponent)
    entity.AddTag(LeaderTag)

    // Update squad capacity (leader's Leadership stat affects capacity)
    memberData := common.GetComponentType[*SquadMemberData](entity, SquadMemberComponent)
    UpdateSquadCapacity(memberData.SquadID, manager)

    return nil
}
```

### Removing Components

```go
func DemoteLeader(unitID ecs.EntityID, manager *EntityManager) error {
    entity := common.FindEntityByIDWithTag(manager, unitID, LeaderTag)
    if entity == nil {
        return fmt.Errorf("not a leader")
    }

    // Remove leader component
    entity.RemoveComponent(LeaderComponent)

    // Remove leader tag
    entity.RemoveTag(LeaderTag)

    // Update squad capacity
    memberData := common.GetComponentType[*SquadMemberData](entity, SquadMemberComponent)
    UpdateSquadCapacity(memberData.SquadID, manager)

    return nil
}
```

### Entity Destruction

**Critical**: Entity destruction requires cleanup of all systems tracking the entity.

```go
func DestroySquad(squadID ecs.EntityID, manager *EntityManager) error {
    // Step 1: Find squad entity
    squadEntity := GetSquadEntity(squadID, manager)
    if squadEntity == nil {
        return fmt.Errorf("squad not found")
    }

    // Step 2: Remove from position system (if positioned)
    if squadEntity.HasComponent(common.PositionComponent) {
        pos := common.GetComponentType[*coords.LogicalPosition](squadEntity, common.PositionComponent)
        common.GlobalPositionSystem.RemoveEntity(squadEntity.GetID(), *pos)
    }

    // Step 3: Dispose entity from ECS
    manager.World.DisposeEntities(squadEntity)

    return nil
}
```

**Complete Cleanup Example:**
```go
func DestroyUnit(unitID ecs.EntityID, manager *EntityManager) error {
    // Find entity
    unitEntity := common.FindEntityByIDWithTag(manager, unitID, SquadMemberTag)
    if unitEntity == nil {
        return fmt.Errorf("unit not found")
    }

    // Get squad ID for capacity update
    memberData := common.GetComponentType[*SquadMemberData](unitEntity, SquadMemberComponent)
    squadID := memberData.SquadID

    // Remove from position system
    if unitEntity.HasComponent(common.PositionComponent) {
        pos := common.GetComponentType[*coords.LogicalPosition](unitEntity, common.PositionComponent)
        common.GlobalPositionSystem.RemoveEntity(unitID, *pos)
    }

    // Dispose entity
    manager.World.DisposeEntities(unitEntity)

    // Update squad capacity
    UpdateSquadCapacity(squadID, manager)

    return nil
}
```

### Entity Movement

Moving entities requires updating position system:

```go
func MoveUnit(unitID ecs.EntityID, newPos coords.LogicalPosition, manager *EntityManager) error {
    // Find entity
    unitEntity := common.FindEntityByIDWithTag(manager, unitID, SquadMemberTag)
    if unitEntity == nil {
        return fmt.Errorf("unit not found")
    }

    // Get old position
    oldPosComp := common.GetComponentType[*coords.LogicalPosition](unitEntity, common.PositionComponent)
    if oldPosComp == nil {
        return fmt.Errorf("unit has no position")
    }
    oldPos := *oldPosComp

    // Update position component
    *oldPosComp = newPos

    // Update position system
    if err := common.GlobalPositionSystem.MoveEntity(unitID, oldPos, newPos); err != nil {
        // Rollback position component
        *oldPosComp = oldPos
        return fmt.Errorf("failed to move entity: %w", err)
    }

    return nil
}
```

### Lifecycle Hooks Pattern

For complex lifecycle management, use hook pattern:

```go
type EntityLifecycleHook interface {
    OnEntityCreated(entityID ecs.EntityID, manager *EntityManager)
    OnEntityDestroyed(entityID ecs.EntityID, manager *EntityManager)
}

type SquadLifecycleManager struct {
    hooks []EntityLifecycleHook
}

func (slm *SquadLifecycleManager) RegisterHook(hook EntityLifecycleHook) {
    slm.hooks = append(slm.hooks, hook)
}

func (slm *SquadLifecycleManager) CreateSquad(name string, manager *EntityManager) ecs.EntityID {
    squadID := CreateSquad(name, FormationBalanced, manager)

    // Notify hooks
    for _, hook := range slm.hooks {
        hook.OnEntityCreated(squadID, manager)
    }

    return squadID
}

func (slm *SquadLifecycleManager) DestroySquad(squadID ecs.EntityID, manager *EntityManager) error {
    // Notify hooks before destruction
    for _, hook := range slm.hooks {
        hook.OnEntityDestroyed(squadID, manager)
    }

    return DestroySquad(squadID, manager)
}
```

---

## Component Access Patterns

TinkerRogue provides type-safe helper functions in `common/ecsutil.go` for component access. **Never access the underlying ECS library directly.**

### Access by Entity Pointer

Use when you already have entity from query result:

```go
// From query result
for _, result := range manager.World.Query(SquadTag) {
    entity := result.Entity

    // ✅ Type-safe access
    squadData := common.GetComponentType[*SquadData](entity, SquadComponent)

    fmt.Printf("Squad: %s\n", squadData.Name)
}
```

### Access by EntityID

Use when you only have entity ID:

```go
func ProcessUnit(unitID ecs.EntityID, manager *EntityManager) {
    // ✅ Direct component access by ID
    attr := common.GetComponentTypeByID[*Attributes](manager, unitID, AttributeComponent)
    if attr == nil {
        return  // Component not found
    }

    fmt.Printf("Unit HP: %d/%d\n", attr.CurrentHealth, attr.MaxHealth)
}
```

**Performance Note**: `GetComponentTypeByID` searches all entities. For better performance, use tag-scoped version:

### Access by EntityID with Tag (Recommended)

**10-100x faster** than `GetComponentTypeByID` because it searches only tagged entities:

```go
func ProcessUnit(unitID ecs.EntityID, manager *EntityManager) {
    // ✅ FASTER - Searches only SquadMemberTag entities (10-100 instead of 1000+)
    attr := common.GetAttributesByIDWithTag(manager, unitID, SquadMemberTag)
    if attr == nil {
        return
    }

    fmt.Printf("Unit HP: %d/%d\n", attr.CurrentHealth, attr.MaxHealth)
}
```

**Usage Example from Codebase:**
```go
// squads/squadcombat.go:39
attackerAttr := common.GetAttributesByIDWithTag(squadmanager, attackerID, SquadMemberTag)
```

### Specialized Helper Functions

**Position Access:**
```go
// From entity pointer
pos := common.GetPosition(entity)

// From EntityID (searches all)
pos := common.GetPositionByID(manager, entityID)

// From EntityID with tag (faster)
pos := common.GetPositionByIDWithTag(manager, entityID, SquadMemberTag)
```

**Attributes Access:**
```go
// From entity pointer
attr := common.GetAttributes(entity)

// From EntityID (searches all)
attr := common.GetAttributesByID(manager, entityID)

// From EntityID with tag (faster)
attr := common.GetAttributesByIDWithTag(manager, entityID, SquadMemberTag)
```

### Component Existence Check

Before accessing components, check if they exist:

```go
// Check by EntityID
if manager.HasComponent(unitID, GridPositionComponent) {
    gridPos := common.GetComponentTypeByID[*GridPositionData](manager, unitID, GridPositionComponent)
    // Use gridPos
}

// Check by EntityID with tag (faster)
if manager.HasComponentByIDWithTag(unitID, SquadMemberTag, GridPositionComponent) {
    gridPos := common.GetComponentTypeByIDWithTag[*GridPositionData](
        manager, unitID, SquadMemberTag, GridPositionComponent)
    // Use gridPos
}
```

### Spatial Queries (O(1) Performance)

For position-based lookups, use `GlobalPositionSystem`:

```go
// ✅ CORRECT - O(1) position lookup
pos := coords.LogicalPosition{X: 10, Y: 20}
entityIDs := common.GlobalPositionSystem.GetEntitiesAtPosition(pos)

// Get single entity at position
entityID := common.GlobalPositionSystem.GetEntityIDAt(pos)
if entityID != 0 {
    // Entity found at position
}

// ❌ WRONG - O(n) brute force search
for _, result := range manager.World.Query(AllEntitiesTag) {
    pos := common.GetPosition(result.Entity)
    if pos.X == targetX && pos.Y == targetY {
        // Found it (but searched entire entity list!)
    }
}
```

**Area Queries:**
```go
// Get all entities in radius
center := coords.LogicalPosition{X: 10, Y: 10}
radius := 3
entities := common.GlobalPositionSystem.GetEntitiesInRadius(center, radius)

// Process entities in area
for _, entityID := range entities {
    attr := common.GetAttributesByID(manager, entityID)
    // Apply AOE effect
}
```

### Function Selection Guide

| Scenario | Function | Performance |
|----------|----------|-------------|
| Have entity from query | `common.GetComponentType[T](entity, component)` | Fast |
| Have EntityID only | `common.GetComponentTypeByID[T](manager, id, component)` | Slow (searches all) |
| Have EntityID + know tag | `common.GetComponentTypeByIDWithTag[T](manager, id, tag, component)` | **Fast (10-100x)** |
| Need entity pointer | `common.FindEntityByID(manager, id)` | Slow (searches all) |
| Need entity pointer + tag | `common.FindEntityByIDWithTag(manager, id, tag)` | Fast |
| Check component exists | `manager.HasComponent(id, component)` | Slow |
| Check component + tag | `manager.HasComponentByIDWithTag(id, tag, component)` | Fast |
| Position lookup | `common.GlobalPositionSystem.GetEntityIDAt(pos)` | **O(1)** |

**Rule of Thumb**: Always use tag-scoped functions (`*WithTag`) when you know the entity type.

---

## Performance Optimizations

### Value-Based Map Keys (50x Speedup)

**Historical Context**: Position System Refactor (2025-10-15)

**Before** (pointer keys):
```go
type PositionSystem struct {
    spatialGrid map[*coords.LogicalPosition][]ecs.EntityID  // ❌ Pointer key
}

// Lookup requires creating temporary pointer
func (ps *PositionSystem) GetEntityAt(x, y int) ecs.EntityID {
    pos := &coords.LogicalPosition{X: x, Y: y}
    if ids, ok := ps.spatialGrid[pos]; ok {
        return ids[0]
    }
    return 0
}

// Benchmark: 500µs per 1000 lookups
```

**After** (value keys):
```go
// systems/positionsystem.go:17-20
type PositionSystem struct {
    spatialGrid map[coords.LogicalPosition][]ecs.EntityID  // ✅ Value key
}

// Lookup uses value directly
func (ps *PositionSystem) GetEntityAt(x, y int) ecs.EntityID {
    pos := coords.LogicalPosition{X: x, Y: y}
    if ids, ok := ps.spatialGrid[pos]; ok {
        return ids[0]
    }
    return 0
}

// Benchmark: 10µs per 1000 lookups
// Result: 50x faster!
```

**Why Value Keys Are Faster:**

1. **Hash Function**: Go optimizes struct hashing for value types
2. **No Allocation**: Value keys don't allocate temporary pointers
3. **Cache Locality**: Contiguous memory access patterns

**When to Use Value Keys:**
- Small structs (< 32 bytes): `Position{X, Y}`, `GridCoord{Row, Col}`
- Immutable lookup data
- High-frequency lookups

### Tag-Scoped Queries (10-100x Speedup)

**Slow**:
```go
// Searches ALL entities (1000+)
data := common.GetComponentTypeByID[*SquadData](manager, squadID, SquadComponent)
```

**Fast**:
```go
// Searches only SquadTag entities (10-100)
data := common.GetComponentTypeByIDWithTag[*SquadData](manager, squadID, SquadTag, SquadComponent)
```

**Benchmark** (1000 entities, 50 squads):
- `GetComponentTypeByID`: 100µs
- `GetComponentTypeByIDWithTag`: 1µs
- **Speedup: 100x**

### Spatial Indexing (O(1) vs O(n))

**Before** (linear search):
```go
// O(n) - Searches all entities with positions
func GetCreatureAtPosition(manager *EntityManager, targetPos *LogicalPosition) ecs.EntityID {
    for _, result := range manager.World.Query(MonsterTag) {
        pos := common.GetPosition(result.Entity)
        if pos.X == targetPos.X && pos.Y == targetPos.Y {
            return result.Entity.GetID()
        }
    }
    return 0
}

// Benchmark: 50µs for 100 entities
```

**After** (spatial grid):
```go
// common/ecsutil.go:198-224
func GetCreatureAtPosition(ecsmanager *EntityManager, pos *LogicalPosition) ecs.EntityID {
    // O(1) - Direct hash lookup
    if GlobalPositionSystem != nil {
        entityID := GlobalPositionSystem.GetEntityIDAt(*pos)
        if entityID == 0 {
            return 0
        }

        // Verify it's a monster (O(m) where m = monsters, not all entities)
        for _, result := range ecsmanager.World.Query(ecsmanager.WorldTags["monsters"]) {
            if result.Entity.GetID() == entityID {
                return entityID
            }
        }
    }
    return 0
}

// Benchmark: 0.5µs (100x faster)
```

**Position System Implementation:**
```go
// systems/positionsystem.go:30-38
func (ps *PositionSystem) GetEntityIDAt(pos coords.LogicalPosition) ecs.EntityID {
    if ids, ok := ps.spatialGrid[pos]; ok && len(ids) > 0 {
        return ids[0]
    }
    return 0
}
```

**Maintaining Spatial Index:**
```go
// On entity creation
common.GlobalPositionSystem.AddEntity(entityID, position)

// On entity movement
common.GlobalPositionSystem.MoveEntity(entityID, oldPos, newPos)

// On entity destruction
common.GlobalPositionSystem.RemoveEntity(entityID, position)
```

### ECS Views (50-200x Speedup for Cached Queries)

**Historical Context**: Frequent Query Optimization (2025-11-30)

ECS Views are automatically-maintained caches that filter entities by tag. Instead of scanning all world entities with `World.Query()`, views maintain a subset of matching entities, making repeated queries dramatically faster.

#### What Are Views?

A View is like a pre-filtered query result that stays up-to-date automatically:

```go
// Creating a view (one-time O(n) cost)
squadView := manager.World.CreateView(SquadTag)

// Using the view (O(k) where k = number of squad entities, typically 10-50)
for _, result := range squadView.Get() {
    entity := result.Entity
    data := common.GetComponentType[*SquadData](entity, SquadComponent)
    // Process squad
}
```

#### How Views Improve Performance

**Without Views** (full world scan):
```go
// O(n) - Searches ALL entities (1000+)
for _, result := range manager.World.Query(SquadTag) {  // Scans 1000 entities
    squadData := common.GetComponentType[*SquadData](result.Entity, SquadComponent)
    if squadData.SquadID == targetID {
        return result.Entity
    }
}

// Benchmark: 340µs per query
```

**With Views** (filtered cache):
```go
// O(k) - Searches only SquadTag entities (10-50)
squadView := manager.World.CreateView(SquadTag)  // Created once during init
for _, result := range squadView.Get() {         // Only ~20 squads to check
    squadData := common.GetComponentType[*SquadData](result.Entity, SquadComponent)
    if squadData.SquadID == targetID {
        return result.Entity
    }
}

// Benchmark: 2.5µs per query
// Speedup: 136x faster!
```

#### View Lifecycle

Views are **automatically maintained** by the ECS library:

1. **Creation** (one-time cost):
   ```go
   // squads/squadcache.go:32-39
   func NewSquadQueryCache(manager *common.EntityManager) *SquadQueryCache {
       return &SquadQueryCache{
           SquadView:       manager.World.CreateView(SquadTag),
           SquadMemberView: manager.World.CreateView(SquadMemberTag),
           LeaderView:      manager.World.CreateView(LeaderTag),
       }
   }
   ```

2. **Automatic Updates**:
   - When `SquadComponent` is added to entity → auto-added to `SquadView`
   - When `SquadComponent` is removed from entity → auto-removed from `SquadView`
   - No manual invalidation needed

3. **Thread-Safe**:
   - Views have built-in `RWMutex` locks
   - Safe to use from multiple goroutines

#### Common View Patterns

**Pattern 1: Basic View Query**
```go
// combat/combatqueriescache.go:24-36
type CombatQueryCache struct {
    ActionStateView *ecs.View  // All ActionStateTag entities
    FactionView     *ecs.View  // All FactionTag entities
}

// Usage
func (c *CombatQueryCache) FindActionStateEntity(squadID ecs.EntityID) *ecs.Entity {
    for _, result := range c.ActionStateView.Get() {
        actionState := common.GetComponentType[*ActionStateData](result.Entity, ActionStateComponent)
        if actionState != nil && actionState.SquadID == squadID {
            return result.Entity
        }
    }
    return nil
}
```

**Pattern 2: Multiple Views for Related Queries**
```go
// squads/squadcache.go:22-28
type SquadQueryCache struct {
    SquadView       *ecs.View  // GetSquadEntity queries
    SquadMemberView *ecs.View  // GetUnitIDsInSquad queries
    LeaderView      *ecs.View  // GetLeaderID queries
}
```

**Pattern 3: Exposing Views Across Packages**
```go
// squads/squadcache.go:23-26
type SquadQueryCache struct {
    // Exported so they can be accessed by other systems (e.g., GUI rendering)
    SquadView       *ecs.View
    SquadMemberView *ecs.View
    LeaderView      *ecs.View
}
```

#### Performance Benchmarks

Real-world benchmarks from TinkerRogue:

| Query | Before (World.Query) | After (View) | Speedup |
|-------|---------------------|--------------|---------|
| GetSquadEntity | 340µs | 2.5µs | **136x** |
| GetUnitIDsInSquad | 280µs | 4µs | **70x** |
| GetLeaderID | 200µs | 1µs | **200x** |
| FindActionStateEntity | O(n) | O(k) | **50-200x** |
| FindFactionByID | O(n) | O(k) | **100-500x** |

*Conditions: World with 1000+ entities, 20-180 entities per view*

#### When to Use Views

**Use Views when:**
- Query is called **frequently** (every frame or multiple times per frame)
- World has **1000+ entities** (more entities = greater speedup)
- Query filters by **specific tag** (SquadTag, FactionTag, ActionStateTag)
- You need **consistent performance** (not O(n) variance)

**Example use cases:**
- GUI rendering loops
- Combat system queries
- Squad management queries
- Frame-by-frame entity processing

**Don't use Views when:**
- Query called **once or twice** in entire application
- World is **small** (< 100 entities)
- Query has **complex multi-component filtering**

#### Implementation Pattern: Query Cache

Wrap views in a cache struct for clean API:

```go
// squads/squadcache.go (complete example)
type SquadQueryCache struct {
    SquadView       *ecs.View
    SquadMemberView *ecs.View
    LeaderView      *ecs.View
}

func NewSquadQueryCache(manager *common.EntityManager) *SquadQueryCache {
    return &SquadQueryCache{
        SquadView:       manager.World.CreateView(SquadTag),
        SquadMemberView: manager.World.CreateView(SquadMemberTag),
        LeaderView:      manager.World.CreateView(LeaderTag),
    }
}

// Query functions use views instead of World.Query()
func (c *SquadQueryCache) GetSquadEntity(squadID ecs.EntityID) *ecs.Entity {
    for _, result := range c.SquadView.Get() {  // ✅ Uses view, not World.Query
        squadData := common.GetComponentType[*SquadData](result.Entity, SquadComponent)
        if squadData != nil && squadData.SquadID == squadID {
            return result.Entity
        }
    }
    return nil
}
```

#### Reference Implementations

- **squads/squadcache.go** - Squad query cache with 3 views
- **combat/combatqueriescache.go** - Combat query cache with action state and faction views
- **rendering/renderingcache.go** - Rendering cache with spatial queries

#### Best Practices for Views

1. **Create views once during initialization**
   ```go
   // ✅ Good: Create once
   cache := NewSquadQueryCache(manager)

   // ❌ Bad: Creating new view every frame
   for {
       view := manager.World.CreateView(SquadTag)  // Don't do this!
   }
   ```

2. **Use specific tags, not AllEntitiesTag**
   ```go
   // ✅ Good: Specific tag
   view := manager.World.CreateView(SquadTag)

   // ❌ Bad: All entities
   view := manager.World.CreateView(AllEntitiesTag)
   ```

3. **Store views in persistent structs**
   ```go
   // ✅ Good: Cache struct with views
   type SquadQueryCache struct {
       SquadView *ecs.View
   }

   // ❌ Bad: Local variable that's recreated
   func ProcessSquads() {
       view := manager.World.CreateView(SquadTag)
       // ...
   }
   ```

4. **Export views only when needed across packages**
   ```go
   // ✅ Good: Exported for cross-package use
   type SquadQueryCache struct {
       SquadView *ecs.View  // Exported for GUI to access
   }

   // ❌ Bad: Unnecessary visibility
   type privateCache struct {
       view *ecs.View
   }
   ```

### Query Optimization Patterns

**Pattern 1: Early Exit**
```go
func IsSquadDestroyed(squadID ecs.EntityID, manager *EntityManager) bool {
    unitIDs := GetUnitIDsInSquad(squadID, manager)

    for _, unitID := range unitIDs {
        attr := common.GetAttributesByIDWithTag(manager, unitID, SquadMemberTag)
        if attr != nil && attr.CurrentHealth > 0 {
            return false  // ✅ Early exit - squad not destroyed
        }
    }

    return len(unitIDs) > 0
}
```

**Pattern 2: Deduplication**
```go
// squads/squadqueries.go:77
func GetUnitIDsInRow(squadID ecs.EntityID, row int, manager *EntityManager) []ecs.EntityID {
    var unitIDs []ecs.EntityID
    seen := make(map[ecs.EntityID]bool)  // ✅ Prevent duplicates

    for col := 0; col < 3; col++ {
        idsAtPos := GetUnitIDsAtGridPosition(squadID, row, col, manager)
        for _, unitID := range idsAtPos {
            if !seen[unitID] {
                // Process unit
                seen[unitID] = true
            }
        }
    }

    return unitIDs
}
```

**Pattern 3: Pre-allocation**
```go
func GetAllSquadIDs(manager *EntityManager) []ecs.EntityID {
    // ✅ Pre-allocate with estimated capacity
    squadIDs := make([]ecs.EntityID, 0, 50)

    for _, result := range manager.World.Query(SquadTag) {
        squadData := common.GetComponentType[*SquadData](result.Entity, SquadComponent)
        squadIDs = append(squadIDs, squadData.SquadID)
    }

    return squadIDs
}
```

### Caching Guidelines

**When to Cache:**
1. Query called in tight loop (e.g., every frame)
2. Profiling shows query is bottleneck (>10% of frame time)
3. Data is read-mostly (rarely changes)

**When NOT to Cache:**
1. Query called infrequently (once per turn, on user action)
2. Data changes frequently
3. Premature optimization (no profiling data)

**Caching Template:**
```go
type CachedSystem struct {
    manager     *EntityManager
    cache       map[ecs.EntityID][]ecs.EntityID
    cacheDirty  bool
}

func (cs *CachedSystem) Get(id ecs.EntityID) []ecs.EntityID {
    if cs.cacheDirty {
        cs.rebuildCache()
    }
    return cs.cache[id]
}

func (cs *CachedSystem) Invalidate() {
    cs.cacheDirty = true
}

func (cs *CachedSystem) rebuildCache() {
    cs.cache = make(map[ecs.EntityID][]ecs.EntityID)
    // Rebuild from queries
    cs.cacheDirty = false
}
```

### ECS Views for Query Caching (100-1000x Speedup)

**Historical Context**: GetSquadInfo Optimization (2025-12-08)

ECS Views provide **automatic query caching** maintained by the ECS library. Use Views when queries are called repeatedly (e.g., every frame) to eliminate O(n) scans.

#### The Problem: Repeated Query Scans

**Before** (O(N×M) catastrophe):
```go
// Called EVERY FRAME for EVERY visible squad
func (shr *SquadHighlightRenderer) Render(...) {
    allSquads := squads.FindAllSquads(shr.queries.ECSManager)

    for _, squadID := range allSquads {
        // Each call scans ALL entities multiple times!
        squadInfo := shr.queries.GetSquadInfo(squadID)
        // ... rendering logic
    }
}

// GetSquadInfo internally does:
// - Query ALL entities for name (O(n))
// - Query ALL units for members (O(m))
// - Query ALL entities per unit for attributes (O(u×n))
// Total: O(squads × (n + m + u×n))
```

**Benchmark** (10 squads, 5 units/squad, 500 entities):
- Entity scans per frame: **25,000+**
- Frame time: 30-50ms
- Result: **FPS drops to 20-30 with many squads**

#### The Solution: ECS Views

**Step 1: Create Views at Initialization**

Views cache query results and automatically update when entities are added/removed:

```go
// gui/guicomponents/guiqueries.go:15-35
type GUIQueries struct {
    ECSManager      *common.EntityManager
    factionManager  *combat.FactionManager

    // Cached ECS Views (automatically maintained by library)
    squadView       *ecs.View  // All SquadTag entities
    squadMemberView *ecs.View  // All SquadMemberTag entities
    actionStateView *ecs.View  // All ActionStateTag entities
}

func NewGUIQueries(ecsManager *common.EntityManager) *GUIQueries {
    return &GUIQueries{
        ECSManager:     ecsManager,
        factionManager: combat.NewFactionManager(ecsManager),

        // Initialize Views (one-time O(n) cost, then O(1) access)
        squadView:       ecsManager.World.CreateView(squads.SquadTag),
        squadMemberView: ecsManager.World.CreateView(squads.SquadMemberTag),
        actionStateView: ecsManager.World.CreateView(combat.ActionStateTag),
    }
}
```

**How Views Work:**
1. `CreateView(tag)` builds initial cache from Query results
2. Library automatically calls `view.add(entity)` when entities added
3. Library automatically calls `view.remove(entity)` when entities removed
4. Thread-safe with RWMutex
5. **No manual invalidation needed!**

**Step 2: Build Lookup Maps from Views**

Instead of repeated queries, iterate Views once per frame to build O(1) lookup maps:

```go
// gui/guicomponents/guiqueries.go:359-401
type SquadInfoCache struct {
    squadNames      map[ecs.EntityID]string
    squadMembers    map[ecs.EntityID][]ecs.EntityID
    actionStates    map[ecs.EntityID]*combat.ActionStateData
    squadFactions   map[ecs.EntityID]ecs.EntityID
    destroyedStatus map[ecs.EntityID]bool
}

func (gq *GUIQueries) BuildSquadInfoCache() *SquadInfoCache {
    cache := &SquadInfoCache{
        squadNames:      make(map[ecs.EntityID]string),
        squadMembers:    make(map[ecs.EntityID][]ecs.EntityID),
        actionStates:    make(map[ecs.EntityID]*combat.ActionStateData),
        squadFactions:   make(map[ecs.EntityID]ecs.EntityID),
        destroyedStatus: make(map[ecs.EntityID]bool),
    }

    // Single pass over squads View (not fresh query!)
    for _, result := range gq.squadView.Get() {
        entity := result.Entity
        squadData := common.GetComponentType[*squads.SquadData](entity, squads.SquadComponent)
        squadID := squadData.SquadID

        cache.squadNames[squadID] = squadData.Name
        cache.destroyedStatus[squadID] = squadData.IsDestroyed

        // Get faction if squad is in combat
        combatFaction := common.GetComponentType[*combat.CombatFactionData](entity, combat.FactionMembershipComponent)
        if combatFaction != nil {
            cache.squadFactions[squadID] = combatFaction.FactionID
        }
    }

    // Single pass over squad members View
    for _, result := range gq.squadMemberView.Get() {
        memberData := common.GetComponentType[*squads.SquadMemberData](result.Entity, squads.SquadMemberComponent)
        cache.squadMembers[memberData.SquadID] = append(cache.squadMembers[memberData.SquadID], result.Entity.GetID())
    }

    // Single pass over action states View
    for _, result := range gq.actionStateView.Get() {
        actionState := common.GetComponentType[*combat.ActionStateData](result.Entity, combat.ActionStateComponent)
        cache.actionStates[actionState.SquadID] = actionState
    }

    return cache
}
```

**Complexity Analysis:**
- **Before:** O(N × M) where N = squads, M = all entities
- **After:** O(S + U + A) where S = squads, U = units, A = action states
- **Speedup:** 100-1000x for typical scenarios

**Step 3: Use Cache for Queries**

```go
// gui/guicomponents/guiqueries.go:407-463
func (gq *GUIQueries) GetSquadInfoCached(squadID ecs.EntityID, cache *SquadInfoCache) *SquadInfo {
    // All lookups are O(1) map access (no queries!)
    name := cache.squadNames[squadID]
    unitIDs := cache.squadMembers[squadID]
    factionID := cache.squadFactions[squadID]
    isDestroyed := cache.destroyedStatus[squadID]
    actionState := cache.actionStates[squadID]

    // Calculate HP (now uses O(1) GetComponentTypeByID - see below)
    aliveUnits := 0
    totalHP := 0
    maxHP := 0
    for _, unitID := range unitIDs {
        attrs := common.GetAttributesByIDWithTag(gq.ECSManager, unitID, squads.SquadMemberTag)
        if attrs != nil {
            if attrs.CanAct {
                aliveUnits++
            }
            totalHP += attrs.CurrentHealth
            maxHP += attrs.MaxHealth
        }
    }

    // Position lookup (O(1) - see GetComponentTypeByID optimization)
    var position *coords.LogicalPosition
    squadPos := common.GetComponentTypeByID[*coords.LogicalPosition](gq.ECSManager, squadID, common.PositionComponent)
    if squadPos != nil {
        pos := *squadPos
        position = &pos
    }

    // Extract action state fields
    hasActed := false
    hasMoved := false
    movementRemaining := 0
    if actionState != nil {
        hasActed = actionState.HasActed
        hasMoved = actionState.HasMoved
        movementRemaining = actionState.MovementRemaining
    }

    return &SquadInfo{
        ID:                squadID,
        Name:              name,
        UnitIDs:           unitIDs,
        AliveUnits:        aliveUnits,
        TotalUnits:        len(unitIDs),
        CurrentHP:         totalHP,
        MaxHP:             maxHP,
        Position:          position,
        FactionID:         factionID,
        IsDestroyed:       isDestroyed,
        HasActed:          hasActed,
        HasMoved:          hasMoved,
        MovementRemaining: movementRemaining,
    }
}
```

**Step 4: Use in Performance-Critical Code**

```go
// gui/guimodes/guirenderers.go:169-195
func (shr *SquadHighlightRenderer) Render(screen *ebiten.Image, ...) {
    vr := shr.cachedRenderer

    // BUILD CACHE ONCE per render call (O(squads + units + states))
    cache := shr.queries.BuildSquadInfoCache()

    allSquads := squads.FindAllSquads(shr.queries.ECSManager)

    for _, squadID := range allSquads {
        // Use cached version (O(units_in_squad) vs O(all_entities))
        squadInfo := shr.queries.GetSquadInfoCached(squadID, cache)
        if squadInfo == nil || squadInfo.IsDestroyed || squadInfo.Position == nil {
            continue
        }

        // ... render with squadInfo ...
    }
}
```

**Performance Results:**
- Cache built once: ~100 operations
- Per-squad lookup: ~5-10 operations
- **Before:** 10 squads × 2500 ops = 25,000 ops/frame
- **After:** 100 + (10 × 10) = 200 ops/frame
- **Speedup: 125x for rendering alone**

#### Critical Fix: GetComponentTypeByID O(1) Optimization

Views alone aren't enough - you also need O(1) component lookups by EntityID:

**Before** (O(n) scan):
```go
// common/ecsutil.go:113-132 (OLD)
func GetComponentTypeByID[T any](manager *EntityManager, entityID ecs.EntityID, component *ecs.Component) T {
    for _, result := range manager.World.Query(AllEntitiesTag) {  // ❌ Scans ALL entities!
        if result.Entity.GetID() == entityID {
            if c, ok := result.Entity.GetComponentData(component); ok {
                return c.(T)
            }
        }
    }
    var nilValue T
    return nilValue
}
```

**After** (O(1) lookup):
```go
// common/ecsutil.go:113-135 (OPTIMIZED)
func GetComponentTypeByID[T any](manager *EntityManager, entityID ecs.EntityID, component *ecs.Component) T {
    defer func() {
        if r := recover(); r != nil {
            // ERROR HANDLING IN FUTURE
        }
    }()

    // Use ECS library's O(1) entitiesByID map lookup
    queryResult := manager.World.GetEntityByID(entityID)
    if queryResult == nil {
        var nilValue T
        return nilValue
    }

    entity := queryResult.Entity
    if c, ok := entity.GetComponentData(component); ok {
        return c.(T)
    }

    var nilValue T
    return nilValue
}
```

**Why This Matters:**
- ECS library maintains `entitiesByID map[EntityID]*Entity` internally
- `GetEntityByID()` does direct map lookup, not query scan
- **This change benefits the ENTIRE codebase** - any code using `GetComponentTypeByID`
- Speedup: 10-50x for individual component lookups

#### When to Use Views

**Use Views when:**
1. ✅ Query called **every frame** or in tight loops
2. ✅ Querying same tag repeatedly (SquadTag, UnitTag, etc.)
3. ✅ Building UI lists/filters that iterate all entities of a type
4. ✅ Profiling shows query as bottleneck (>10% frame time)

**Don't Use Views when:**
1. ❌ Query called infrequently (once per user click)
2. ❌ Single entity lookup (use `GetComponentTypeByID` instead)
3. ❌ Data changes every iteration (defeats caching)
4. ❌ Premature optimization (no profiling data)

#### View Lifecycle Management

**No Manual Invalidation Needed:**
```go
// Creating entities - View automatically updated
squad := manager.World.NewEntity()
squad.AddComponent(SquadComponent, squadData)
// ✅ squadView.add(squad) called automatically by library

// Destroying entities - View automatically updated
manager.World.DisposeEntities(squad)
// ✅ squadView.remove(squad) called automatically by library
```

**Thread Safety:**
- Views use `sync.RWMutex` internally
- Safe to call `view.Get()` from multiple goroutines
- Cache building should be single-threaded per frame

**Memory Overhead:**
- Each View stores slice of QueryResult pointers
- ~100 entities × 8 bytes = 800 bytes per View
- Negligible compared to performance gain

#### Real-World Example: UI List Building

**Before** (repeated queries):
```go
// gui/guicomponents/guicomponents.go:62-76 (OLD)
func (slc *SquadListComponent) Refresh() {
    allSquads := squads.FindAllSquads(slc.queries.ECSManager)

    for _, squadID := range allSquads {
        // Queries all entities for EACH squad!
        squadInfo := slc.queries.GetSquadInfo(squadID)
        squadName := squads.GetSquadName(squadID, slc.queries.ECSManager)  // Another full query!
        // ... create UI button
    }
}
```

**After** (View-based caching):
```go
// gui/guicomponents/guicomponents.go:62-79 (OPTIMIZED)
func (slc *SquadListComponent) Refresh() {
    // Build cache once for all squad queries
    cache := slc.queries.BuildSquadInfoCache()

    allSquads := squads.FindAllSquads(slc.queries.ECSManager)

    for _, squadID := range allSquads {
        // O(1) map lookup instead of O(n) query
        squadInfo := slc.queries.GetSquadInfoCached(squadID, cache)
        squadName := cache.squadNames[squadID]  // Direct map access!
        // ... create UI button
    }
}
```

**Performance Impact:**
- **Before:** N squads × 2 queries × M entities = 2NM operations
- **After:** 1 cache build (S+U+A) + N lookups = S+U+A+N operations
- **Example:** 20 squads, 100 units, 500 entities
  - Before: 20 × 2 × 500 = 20,000 ops
  - After: 20 + 100 + 10 + 20 = 150 ops
  - **Speedup: 133x**

#### Summary: Views Best Practices

1. **Create Views at initialization** - One-time setup cost
2. **Build lookup maps once per frame** - Single O(n) pass
3. **Reuse cache for all queries** - O(1) map lookups
4. **Combine with GetComponentTypeByID optimization** - O(1) component access
5. **Keep original methods for compatibility** - Gradual migration
6. **Profile before and after** - Verify actual performance gains

**Expected Results:**
- Rendering: 100-1000x faster
- UI building: 10-100x faster
- Filtering: 10-100x faster
- Frame time: 30-50ms → <16ms (60 FPS)

---

## File Organization

### Package Structure

One major system per package with consistent file naming:

```
squads/
├── squadcomponents.go     # Component data definitions
├── squadmanager.go        # Initialization functions
├── squadqueries.go        # Query functions (read-only)
├── squadcombat.go         # Combat system logic
├── squadabilities.go      # Ability system logic
├── squadcreation.go       # Creation system logic (example)
├── squads_test.go         # Tests
├── squadservices/         # Service layer (optional)
│   └── squad_builder_service.go
└── squadcommands/         # Command pattern (optional)
    ├── command.go
    └── rename_squad_command.go
```

### File Responsibilities

**`components.go` - Component Definitions Only**

```go
package squads

import "github.com/bytearena/ecs"

// Component registration variables
var (
    SquadComponent *ecs.Component
    SquadTag       ecs.Tag
)

// Component data structs (pure data)
type SquadData struct {
    SquadID   ecs.EntityID
    Formation FormationType
    Name      string
    Morale    int
}

type FormationType int

const (
    FormationBalanced FormationType = iota
    FormationDefensive
)
```

**Rules:**
- Only data struct definitions
- Component/tag registration variables
- Enums and constants
- NO logic, NO methods (except pure utility like `String()`), NO query functions

**`*manager.go` - Initialization Functions**

```go
package squads

import "game_main/common"

func InitSquadComponents(manager *common.EntityManager) {
    SquadComponent = manager.World.NewComponent()
    SquadMemberComponent = manager.World.NewComponent()
    // ... more components
}

func InitSquadTags(manager *common.EntityManager) {
    SquadTag = ecs.BuildTag(SquadComponent)
    SquadMemberTag = ecs.BuildTag(SquadMemberComponent)

    manager.WorldTags["squad"] = SquadTag
    manager.WorldTags["squadmember"] = SquadMemberTag
}

func InitializeSquadData(manager *common.EntityManager) error {
    InitSquadComponents(manager)
    InitSquadTags(manager)
    return nil
}
```

**Rules:**
- Component registration (`NewComponent()`)
- Tag building (`BuildTag()`)
- Main initialization entry point
- NO game logic

**`*queries.go` - Query Functions (Read-Only)**

```go
package squads

import "game_main/common"

// Get single entity
func GetSquadEntity(squadID ecs.EntityID, manager *common.EntityManager) *ecs.Entity {
    for _, result := range manager.World.Query(SquadTag) {
        squadData := common.GetComponentType[*SquadData](result.Entity, SquadComponent)
        if squadData.SquadID == squadID {
            return result.Entity
        }
    }
    return nil
}

// Get multiple entities
func GetUnitIDsInSquad(squadID ecs.EntityID, manager *common.EntityManager) []ecs.EntityID {
    var unitIDs []ecs.EntityID
    for _, result := range manager.World.Query(SquadMemberTag) {
        memberData := common.GetComponentType[*SquadMemberData](result.Entity, SquadMemberComponent)
        if memberData.SquadID == squadID {
            unitIDs = append(unitIDs, result.Entity.GetID())
        }
    }
    return unitIDs
}

// Boolean check
func IsSquadDestroyed(squadID ecs.EntityID, manager *common.EntityManager) bool {
    // Query implementation
}
```

**Rules:**
- Functions that search/filter entities
- Functions that check entity state
- Functions that extract data from components
- **Read-only** - NO modification of components
- Naming: `Get*`, `Find*`, `Is*`, `Can*`

**`*system.go` - System Logic (State Modification)**

```go
package squads

import "game_main/common"

// Action system - modifies state
func ExecuteSquadAttack(attackerSquad, defenderSquad ecs.EntityID, manager *common.EntityManager) *CombatResult {
    // Get components
    attackerData := common.GetComponentTypeByIDWithTag[*SquadData](manager, attackerSquad, SquadTag, SquadComponent)
    defenderData := common.GetComponentTypeByIDWithTag[*SquadData](manager, defenderSquad, SquadTag, SquadComponent)

    // Perform combat logic (modifies components)
    // ...

    return result
}

// Update system - recalculates state
func UpdateSquadCapacity(squadID ecs.EntityID, manager *common.EntityManager) {
    squadEntity := GetSquadEntity(squadID, manager)
    squadData := common.GetComponentType[*SquadData](squadEntity, SquadComponent)

    // Recalculate and update
    squadData.TotalCapacity = GetSquadTotalCapacity(squadID, manager)
    squadData.UsedCapacity = GetSquadUsedCapacity(squadID, manager)
}

// Creation system
func CreateSquad(name string, formation FormationType, manager *common.EntityManager) ecs.EntityID {
    entity := manager.World.CreateEntity()
    squadID := ecs.EntityID(manager.World.NewIDPool().GetID())

    entity.AddComponent(SquadComponent, &SquadData{
        SquadID:   squadID,
        Name:      name,
        Formation: formation,
    })
    entity.AddTag(SquadTag)

    return squadID
}
```

**Rules:**
- Functions that modify component data
- Functions that create/destroy entities
- Functions that implement game mechanics
- Can call query functions from `*queries.go`
- Naming: `Execute*`, `Update*`, `Create*`, `Apply*`

**`*_test.go` - Tests**

```go
package squads

import (
    "testing"
    "game_main/common"
)

func TestCreateSquad(t *testing.T) {
    manager := common.CreateTestEntityManager()
    InitializeSquadData(manager)

    squadID := CreateSquad("Test Squad", FormationBalanced, manager)

    if squadID == 0 {
        t.Fatal("Expected squad ID, got 0")
    }

    squadEntity := GetSquadEntity(squadID, manager)
    if squadEntity == nil {
        t.Fatal("Squad entity not found")
    }

    squadData := common.GetComponentType[*SquadData](squadEntity, SquadComponent)
    if squadData.Name != "Test Squad" {
        t.Errorf("Expected 'Test Squad', got '%s'", squadData.Name)
    }
}
```

### Optional Service Layer

For complex business logic, add service layer:

```
squads/
└── squadservices/
    ├── squad_builder_service.go
    └── squad_builder_service_test.go
```

**Service Pattern:**
```go
// squads/squadservices/squad_builder_service.go
type SquadBuilderService struct {
    entityManager *common.EntityManager
}

func NewSquadBuilderService(manager *common.EntityManager) *SquadBuilderService {
    return &SquadBuilderService{entityManager: manager}
}

// Service methods wrap system functions with validation and error handling
func (sbs *SquadBuilderService) PlaceUnit(
    squadID ecs.EntityID,
    unit UnitTemplate,
    gridRow, gridCol int,
) *PlaceUnitResult {
    result := &PlaceUnitResult{UnitName: unit.Name}

    // Validation
    if gridRow < 0 || gridRow > 2 || gridCol < 0 || gridCol > 2 {
        result.Error = "invalid grid position"
        return result
    }

    // Call base system function
    err := squads.AddUnitToSquad(squadID, sbs.entityManager, unit, gridRow, gridCol)
    if err != nil {
        result.Error = err.Error()
        return result
    }

    // Post-processing
    result.Success = true
    result.RemainingCapacity = squads.GetSquadRemainingCapacity(squadID, sbs.entityManager)
    return result
}
```

### Optional Command Layer

For undo/redo and transaction support:

```
squads/
└── squadcommands/
    ├── command.go
    ├── rename_squad_command.go
    └── change_formation_command.go
```

**Command Pattern:**
```go
// squadcommands/command.go
type Command interface {
    Execute(manager *common.EntityManager) error
    Undo(manager *common.EntityManager) error
}

// squadcommands/rename_squad_command.go
type RenameSquadCommand struct {
    SquadID  ecs.EntityID
    NewName  string
    OldName  string
}

func (cmd *RenameSquadCommand) Execute(manager *common.EntityManager) error {
    squadEntity := squads.GetSquadEntity(cmd.SquadID, manager)
    squadData := common.GetComponentType[*squads.SquadData](squadEntity, squads.SquadComponent)

    cmd.OldName = squadData.Name
    squadData.Name = cmd.NewName
    return nil
}

func (cmd *RenameSquadCommand) Undo(manager *common.EntityManager) error {
    squadEntity := squads.GetSquadEntity(cmd.SquadID, manager)
    squadData := common.GetComponentType[*squads.SquadData](squadEntity, squads.SquadComponent)

    squadData.Name = cmd.OldName
    return nil
}
```

---

## Reference Implementations

### Squad System (Perfect ECS Example)

**Package:** `squads/`
**Lines of Code:** 4,933
**Components:** 8 (SquadData, SquadMemberData, LeaderData, GridPositionData, etc.)
**Systems:** Combat, abilities, queries, creation, visualization

**Why Perfect:**
- Pure data components (zero logic)
- EntityID-based relationships
- Query-based squad member discovery
- System-based combat logic
- Multi-component tags (LeaderTag)

**Key Files:**
- `squadcomponents.go`: Component definitions
- `squadmanager.go`: Initialization
- `squadqueries.go`: Read-only queries
- `squadcombat.go`: Combat system
- `squadabilities.go`: Ability system

**Study Points:**
```go
// Pure data component
type SquadData struct {
    SquadID       ecs.EntityID  // Native ID
    Formation     FormationType
    Name          string
    Morale        int
}

// EntityID relationship
type SquadMemberData struct {
    SquadID ecs.EntityID  // Not entity pointer
}

// Query-based relationship discovery
func GetUnitIDsInSquad(squadID ecs.EntityID, manager *EntityManager) []ecs.EntityID {
    var unitIDs []ecs.EntityID
    for _, result := range manager.World.Query(SquadMemberTag) {
        memberData := common.GetComponentType[*SquadMemberData](result.Entity, SquadMemberComponent)
        if memberData.SquadID == squadID {
            unitIDs = append(unitIDs, result.Entity.GetID())
        }
    }
    return unitIDs
}

// System function (not component method)
func ExecuteSquadAttack(attackerSquad, defenderSquad ecs.EntityID, manager *EntityManager) *CombatResult {
    // Combat logic in system function
}
```

### Inventory System (Pure ECS Component)

**File:** `gear/Inventory.go`
**Lines of Code:** 241
**Components:** 1 (Inventory)
**Systems:** 9 functions (AddItem, RemoveItem, GetItems, etc.)

**Why Perfect:**
- Pure data component (ItemEntityIDs array)
- EntityID-based item relationships
- No cached state
- All logic in system functions
- No component methods

**Study Points:**
```go
// gear/Inventory.go:23-26
type Inventory struct {
    ItemEntityIDs []ecs.EntityID  // ✅ Uses EntityID, not pointers
}

// System function (not component method)
func AddItem(manager *ecs.Manager, inv *Inventory, itemEntityID ecs.EntityID) {
    itemEntity := common.FindEntityByIDInManager(manager, itemEntityID)
    if itemEntity == nil {
        return
    }

    // Check if item exists
    newItemName := common.GetComponentType[*common.Name](itemEntity, common.NameComponent).NameStr
    exists := false

    for _, existingID := range inv.ItemEntityIDs {
        existingEntity := common.FindEntityByIDInManager(manager, existingID)
        existingName := common.GetComponentType[*common.Name](existingEntity, common.NameComponent).NameStr

        if existingName == newItemName {
            exists = true
            itemComp := GetItemByID(manager, existingID)
            if itemComp != nil {
                itemComp.Count++
            }
            break
        }
    }

    if !exists {
        itemComp := GetItemByID(manager, itemEntityID)
        if itemComp != nil {
            itemComp.Count = 1
        }
        inv.ItemEntityIDs = append(inv.ItemEntityIDs, itemEntityID)
    }
}
```

### Item System (EntityID Relationships)

**File:** `gear/items.go`
**Lines of Code:** 177

**Why Perfect:**
- Uses EntityID for item-owner relationships
- No entity pointers
- Query-based item lookup

**Study Points:**
```go
// gear/items.go:42-46
type Item struct {
    Properties ecs.EntityID  // ✅ EntityID, not *ecs.Entity
    Actions    []ItemAction
    Count      int
}

// CreateItem creates item with EntityID reference
func CreateItem(manager *ecs.Manager, name string, pos coords.LogicalPosition,
    imagePath string, effects ...StatusEffects) *ecs.Entity {

    // Create properties entity
    propsEntity := manager.NewEntity()
    for _, prop := range effects {
        propsEntity.AddComponent(prop.StatusEffectComponent(), &prop)
    }

    // Reference via EntityID
    item := &Item{
        Count:      1,
        Properties: propsEntity.GetID(),  // ✅ Store ID, not pointer
        Actions:    make([]ItemAction, 0),
    }

    itemEntity := manager.NewEntity().
        AddComponent(ItemComponent, item)

    return itemEntity
}
```

### Position System (Value-Based Map Keys)

**File:** `systems/positionsystem.go`
**Lines of Code:** 183
**Data Structure:** O(1) spatial grid

**Why Perfect:**
- Value-based map keys (50x faster than pointers)
- O(1) position lookups
- Proper cache invalidation
- Global singleton pattern

**Study Points:**
```go
// systems/positionsystem.go:17-20
type PositionSystem struct {
    manager     *ecs.Manager
    spatialGrid map[coords.LogicalPosition][]ecs.EntityID  // ✅ Value key
}

// O(1) lookup
func (ps *PositionSystem) GetEntityIDAt(pos coords.LogicalPosition) ecs.EntityID {
    if ids, ok := ps.spatialGrid[pos]; ok && len(ids) > 0 {
        return ids[0]
    }
    return 0
}

// Proper cache maintenance
func (ps *PositionSystem) AddEntity(entityID ecs.EntityID, pos coords.LogicalPosition) error {
    ps.spatialGrid[pos] = append(ps.spatialGrid[pos], entityID)
    return nil
}

func (ps *PositionSystem) RemoveEntity(entityID ecs.EntityID, pos coords.LogicalPosition) error {
    ids, ok := ps.spatialGrid[pos]
    if !ok {
        return fmt.Errorf("no entities at position %v", pos)
    }

    // Remove by swapping with last element
    for i, id := range ids {
        if id == entityID {
            ids[i] = ids[len(ids)-1]
            ps.spatialGrid[pos] = ids[:len(ids)-1]

            // Clean up empty entries
            if len(ps.spatialGrid[pos]) == 0 {
                delete(ps.spatialGrid, pos)
            }
            return nil
        }
    }

    return fmt.Errorf("entity %d not found at position %v", entityID, pos)
}
```

### Attributes Component (Derived Stats Pattern)

**File:** `common/commoncomponents.go`
**Lines of Code:** 213

**Why Study:**
- Shows proper use of methods (pure calculations)
- Core attributes derive all combat stats
- Cached MaxHealth for performance
- No state mutation in getters

**Study Points:**
```go
// common/commoncomponents.go:19-45
type Attributes struct {
    // Core attributes
    Strength   int
    Dexterity  int
    Magic      int
    Leadership int
    Armor      int
    Weapon     int

    // Runtime state
    CurrentHealth int
    MaxHealth     int  // Cached derived stat
    CanAct        bool
}

// ✅ ACCEPTABLE - Pure calculation, no state mutation
func (a *Attributes) GetPhysicalDamage() int {
    return (a.Strength / 2) + (a.Weapon * 2)
}

func (a *Attributes) GetPhysicalResistance() int {
    return (a.Strength / 4) + (a.Armor * 2)
}

func (a *Attributes) GetMaxHealth() int {
    return 20 + (a.Strength * 2)
}

// Constructor caches derived stat
func NewAttributes(strength, dexterity, magic, leadership, armor, weapon int) Attributes {
    attr := Attributes{
        Strength:   strength,
        Dexterity:  dexterity,
        Magic:      magic,
        Leadership: leadership,
        Armor:      armor,
        Weapon:     weapon,
    }

    attr.MaxHealth = attr.GetMaxHealth()
    attr.CurrentHealth = attr.MaxHealth

    return attr
}
```

---

## Integration Patterns

### ECS and GUI Integration

GUI code queries ECS data but **never stores entity pointers**. GUI state is separate from game state.

**✅ CORRECT - Query on Demand:**
```go
// gui/guisquads/squad_builder_grid_manager.go (conceptual example)
type SquadBuilderGridManager struct {
    squadID       ecs.EntityID  // ✅ Store ID
    entityManager *common.EntityManager
}

func (sbgm *SquadBuilderGridManager) GetUnitAtCell(row, col int) *UnitDisplayInfo {
    // Query when needed
    unitIDs := squads.GetUnitIDsAtGridPosition(sbgm.squadID, row, col, sbgm.entityManager)
    if len(unitIDs) == 0 {
        return nil
    }

    unitID := unitIDs[0]

    // Get data for display
    name := common.GetComponentTypeByIDWithTag[*common.Name](
        sbgm.entityManager, unitID, squads.SquadMemberTag, common.NameComponent)
    attr := common.GetAttributesByIDWithTag(sbgm.entityManager, unitID, squads.SquadMemberTag)

    return &UnitDisplayInfo{
        UnitID: unitID,
        Name:   name.NameStr,
        HP:     attr.CurrentHealth,
        MaxHP:  attr.MaxHealth,
    }
}
```

**❌ WRONG - Storing Entity Pointers:**
```go
type SquadBuilderGridManager struct {
    squadEntity *ecs.Entity  // ❌ Can become invalid!
    units       [3][3]*ecs.Entity  // ❌ Can become invalid!
}
```

**GUI State vs Game State:**
```go
// ✅ GUI state - separate from ECS
type BattleMapState struct {
    SelectedSquadID  ecs.EntityID  // Reference by ID
    HighlightedCells [][2]int
    CameraOffset     Point
    UIMode           string
}

// ✅ Game state - in ECS components
type SquadData struct {
    SquadID   ecs.EntityID
    Formation FormationType
    Morale    int
}
```

### ECS and Service Layer Integration

Services encapsulate business logic and wrap system functions:

```go
// squads/squadservices/squad_builder_service.go:11-21
type SquadBuilderService struct {
    entityManager *common.EntityManager
}

func NewSquadBuilderService(manager *common.EntityManager) *SquadBuilderService {
    return &SquadBuilderService{entityManager: manager}
}

// Service method wraps system function with validation
func (sbs *SquadBuilderService) PlaceUnit(
    squadID ecs.EntityID,
    unit UnitTemplate,
    gridRow, gridCol int,
) *PlaceUnitResult {
    result := &PlaceUnitResult{UnitName: unit.Name}

    // Validation layer
    if gridRow < 0 || gridRow > 2 {
        result.Error = "invalid row"
        return result
    }

    // Call ECS system function
    err := squads.AddUnitToSquad(squadID, sbs.entityManager, unit, gridRow, gridCol)
    if err != nil {
        result.Error = err.Error()
        result.RemainingCapacity = squads.GetSquadRemainingCapacity(squadID, sbs.entityManager)
        return result
    }

    // Post-processing
    result.Success = true
    result.RemainingCapacity = squads.GetSquadRemainingCapacity(squadID, sbs.entityManager)
    return result
}
```

**Benefits:**
- Business logic separate from ECS mechanics
- Result structs for rich error handling
- Atomic operations with rollback support
- GUI-friendly return types

### ECS and Command Pattern Integration

Commands provide undo/redo support:

```go
// squads/squadcommands/rename_squad_command.go
type RenameSquadCommand struct {
    SquadID ecs.EntityID
    NewName string
    OldName string  // Stored during Execute for Undo
}

func (cmd *RenameSquadCommand) Execute(manager *common.EntityManager) error {
    squadData := common.GetComponentTypeByIDWithTag[*squads.SquadData](
        manager, cmd.SquadID, squads.SquadTag, squads.SquadComponent)
    if squadData == nil {
        return fmt.Errorf("squad not found")
    }

    // Store old value for undo
    cmd.OldName = squadData.Name

    // Apply change
    squadData.Name = cmd.NewName
    return nil
}

func (cmd *RenameSquadCommand) Undo(manager *common.EntityManager) error {
    squadData := common.GetComponentTypeByIDWithTag[*squads.SquadData](
        manager, cmd.SquadID, squads.SquadTag, squads.SquadComponent)
    if squadData == nil {
        return fmt.Errorf("squad not found")
    }

    squadData.Name = cmd.OldName
    return nil
}
```

**Command Executor:**
```go
// squads/squadcommands/command_executor.go
type CommandExecutor struct {
    manager        *common.EntityManager
    commandHistory []Command
    currentIndex   int
}

func (ce *CommandExecutor) Execute(cmd Command) error {
    if err := cmd.Execute(ce.manager); err != nil {
        return err
    }

    // Truncate history if we're not at the end
    ce.commandHistory = ce.commandHistory[:ce.currentIndex+1]

    // Add command to history
    ce.commandHistory = append(ce.commandHistory, cmd)
    ce.currentIndex++

    return nil
}

func (ce *CommandExecutor) Undo() error {
    if ce.currentIndex < 0 {
        return fmt.Errorf("nothing to undo")
    }

    cmd := ce.commandHistory[ce.currentIndex]
    if err := cmd.Undo(ce.manager); err != nil {
        return err
    }

    ce.currentIndex--
    return nil
}

func (ce *CommandExecutor) Redo() error {
    if ce.currentIndex >= len(ce.commandHistory)-1 {
        return fmt.Errorf("nothing to redo")
    }

    ce.currentIndex++
    cmd := ce.commandHistory[ce.currentIndex]
    return cmd.Execute(ce.manager)
}
```

### ECS and Turn Manager Integration

Turn-based systems use ECS components for state:

```go
// combat/turnmanager.go:11-19
type TurnManager struct {
    manager *common.EntityManager
}

func NewTurnManager(manager *common.EntityManager) *TurnManager {
    return &TurnManager{manager: manager}
}

// Initialize combat state as ECS entity
func (tm *TurnManager) InitializeCombat(factionIDs []ecs.EntityID) error {
    // Create turn state entity
    turnEntity := tm.manager.World.NewEntity()
    turnEntity.AddComponent(TurnStateComponent, &TurnStateData{
        CurrentRound:     1,
        TurnOrder:        shuffleFactionOrder(factionIDs),
        CurrentTurnIndex: 0,
        CombatActive:     true,
    })

    // Create action states for all squads
    for _, factionID := range factionIDs {
        factionSquads := GetSquadsForFaction(factionID, tm.manager)
        for _, squadID := range factionSquads {
            tm.createActionStateForSquad(squadID)
        }
    }

    return nil
}

// Query turn state component
func (tm *TurnManager) GetCurrentFaction() ecs.EntityID {
    turnEntity := findTurnStateEntity(tm.manager)
    if turnEntity == nil {
        return 0
    }

    turnState := common.GetComponentType[*TurnStateData](turnEntity, TurnStateComponent)
    return turnState.TurnOrder[turnState.CurrentTurnIndex]
}
```

**Pattern:** Game state stored in ECS components, turn manager provides game logic.

---

## Common Mistakes and Anti-Patterns

### Mistake #1: Logic in Components

**❌ WRONG:**
```go
type SquadData struct {
    SquadID ecs.EntityID
    Morale  int
}

// NO! Component methods are anti-pattern
func (s *SquadData) IncreaseMorale(amount int) {
    s.Morale += amount
}

func (s *SquadData) Attack(targetSquadID ecs.EntityID, manager *EntityManager) {
    // NO! Components can't access EntityManager
    // NO! Logic belongs in systems
}
```

**✅ CORRECT:**
```go
type SquadData struct {
    SquadID ecs.EntityID
    Morale  int
}

// System function
func IncreaseMorale(squadID ecs.EntityID, amount int, manager *EntityManager) {
    squadData := common.GetComponentTypeByIDWithTag[*SquadData](
        manager, squadID, SquadTag, SquadComponent)
    if squadData != nil {
        squadData.Morale += amount
    }
}
```

**Why This Is Wrong:**
- Components can't access EntityManager
- Components can't query for other entities
- Violates ECS separation of data and logic
- Makes testing harder

### Mistake #2: Storing Entity Pointers

**❌ WRONG:**
```go
type SquadMemberData struct {
    Squad *ecs.Entity  // ❌ Entity pointer
}

type Inventory struct {
    Items []*ecs.Entity  // ❌ Entity pointer array
}
```

**✅ CORRECT:**
```go
type SquadMemberData struct {
    SquadID ecs.EntityID  // ✅ Entity ID
}

type Inventory struct {
    ItemEntityIDs []ecs.EntityID  // ✅ Entity ID array
}
```

**Why This Is Wrong:**
- Entity pointers become invalid when entities are disposed
- Causes dangling pointer bugs
- Memory leaks
- Not serializable

**Historical Bug:**
Before 2025-10-21, Inventory stored `[]*ecs.Entity`. When items were removed, pointers became invalid, causing crashes. Refactor to `[]ecs.EntityID` eliminated the bug.

### Mistake #3: Caching Relationships

**❌ WRONG:**
```go
type SquadData struct {
    SquadID ecs.EntityID
    UnitIDs []ecs.EntityID  // ❌ Cached relationship
}

// Now requires manual synchronization
func AddUnitToSquad(squadID, unitID ecs.EntityID, manager *EntityManager) {
    // Create unit...

    // ❌ Must manually update cached list
    squadData := GetSquadData(squadID, manager)
    squadData.UnitIDs = append(squadData.UnitIDs, unitID)
}

func RemoveUnit(unitID ecs.EntityID, manager *EntityManager) {
    memberData := GetMemberData(unitID, manager)
    squadData := GetSquadData(memberData.SquadID, manager)

    // ❌ Must manually sync - can get out of sync!
    for i, id := range squadData.UnitIDs {
        if id == unitID {
            squadData.UnitIDs = append(squadData.UnitIDs[:i], squadData.UnitIDs[i+1:]...)
            break
        }
    }
}
```

**✅ CORRECT:**
```go
type SquadData struct {
    SquadID ecs.EntityID
    // No cached UnitIDs
}

// Query when needed
func GetUnitIDsInSquad(squadID ecs.EntityID, manager *EntityManager) []ecs.EntityID {
    var unitIDs []ecs.EntityID
    for _, result := range manager.World.Query(SquadMemberTag) {
        memberData := common.GetComponentType[*SquadMemberData](result.Entity, SquadMemberComponent)
        if memberData.SquadID == squadID {
            unitIDs = append(unitIDs, result.Entity.GetID())
        }
    }
    return unitIDs
}
```

**Why This Is Wrong:**
- Cached data can get out of sync
- Requires manual synchronization everywhere
- Queries are always up-to-date
- More code, more bugs

### Mistake #4: Pointer Map Keys

**❌ WRONG:**
```go
type PositionSystem struct {
    grid map[*coords.LogicalPosition][]ecs.EntityID  // ❌ Pointer key
}

// Lookup is slow
func (ps *PositionSystem) GetEntityAt(pos coords.LogicalPosition) ecs.EntityID {
    // Must create pointer for lookup
    key := &pos
    if ids, ok := ps.grid[key]; ok {
        return ids[0]
    }
    return 0
}
```

**✅ CORRECT:**
```go
type PositionSystem struct {
    grid map[coords.LogicalPosition][]ecs.EntityID  // ✅ Value key
}

// Lookup is 50x faster
func (ps *PositionSystem) GetEntityAt(pos coords.LogicalPosition) ecs.EntityID {
    if ids, ok := ps.grid[pos]; ok {
        return ids[0]
    }
    return 0
}
```

**Why This Is Wrong:**
- Pointer keys are 50x slower (measured)
- Must allocate temporary pointers for lookups
- Poor cache locality

### Mistake #5: Using AllEntitiesTag

**❌ WRONG (Slow):**
```go
// Searches ALL entities (1000+)
func GetSquadData(squadID ecs.EntityID, manager *EntityManager) *SquadData {
    for _, result := range manager.World.Query(AllEntitiesTag) {  // ❌ Searches all
        if result.Entity.GetID() == squadID {
            return common.GetComponentType[*SquadData](result.Entity, SquadComponent)
        }
    }
    return nil
}
```

**✅ CORRECT (Fast):**
```go
// Searches only squads (10-100)
func GetSquadData(squadID ecs.EntityID, manager *EntityManager) *SquadData {
    for _, result := range manager.World.Query(SquadTag) {  // ✅ Searches only squads
        squadData := common.GetComponentType[*SquadData](result.Entity, SquadComponent)
        if squadData.SquadID == squadID {
            return squadData
        }
    }
    return nil
}

// Even better: Use helper function
func GetSquadDataFast(squadID ecs.EntityID, manager *EntityManager) *SquadData {
    return common.GetComponentTypeByIDWithTag[*SquadData](
        manager, squadID, SquadTag, SquadComponent)
}
```

**Why This Is Wrong:**
- O(n) where n = all entities (slow)
- Use specific tags: O(n) where n = tagged entities (10-100x faster)

### Mistake #6: Manual Type Assertions

**❌ WRONG:**
```go
// Risky manual type assertion
data := entity.GetComponent(SquadComponent).(*SquadData)
```

**✅ CORRECT:**
```go
// Type-safe helper
data := common.GetComponentType[*SquadData](entity, SquadComponent)
```

**Why This Is Wrong:**
- Panics if component doesn't exist
- Panics if wrong type
- Type-safe helpers handle errors gracefully

### Mistake #7: Not Cleaning Up Position System

**❌ WRONG:**
```go
func DestroyEntity(entityID ecs.EntityID, manager *EntityManager) {
    entity := common.FindEntityByID(manager, entityID)

    // ❌ Forgot to remove from position system!
    manager.World.DisposeEntities(entity)
}
```

**✅ CORRECT:**
```go
func DestroyEntity(entityID ecs.EntityID, manager *EntityManager) error {
    entity := common.FindEntityByID(manager, entityID)
    if entity == nil {
        return fmt.Errorf("entity not found")
    }

    // ✅ Remove from position system first
    if entity.HasComponent(common.PositionComponent) {
        pos := common.GetComponentType[*coords.LogicalPosition](entity, common.PositionComponent)
        common.GlobalPositionSystem.RemoveEntity(entityID, *pos)
    }

    // Now dispose
    manager.World.DisposeEntities(entity)
    return nil
}
```

**Why This Is Wrong:**
- Position system keeps stale references
- Memory leaks
- Incorrect spatial queries

### Mistake #8: Uninitialized Components

**❌ WRONG:**
```go
// Component never initialized!
var MyComponent *ecs.Component  // nil!

func CreateEntity(manager *EntityManager) {
    entity := manager.World.CreateEntity()
    entity.AddComponent(MyComponent, &MyData{})  // PANIC: nil component
}
```

**✅ CORRECT:**
```go
var MyComponent *ecs.Component

func InitComponents(manager *EntityManager) {
    MyComponent = manager.World.NewComponent()  // ✅ Initialize
}

func CreateEntity(manager *EntityManager) {
    entity := manager.World.CreateEntity()
    entity.AddComponent(MyComponent, &MyData{})  // Works
}
```

**Why This Is Wrong:**
- Nil component causes panic
- Components must be registered before use

---

## Testing ECS Code

### Component Tests

Components are pure data - test by creating instances:

```go
func TestSquadData(t *testing.T) {
    squadData := &SquadData{
        SquadID:   1,
        Formation: FormationBalanced,
        Name:      "Test Squad",
        Morale:    100,
    }

    if squadData.Morale != 100 {
        t.Errorf("Expected morale 100, got %d", squadData.Morale)
    }

    if squadData.Name != "Test Squad" {
        t.Errorf("Expected 'Test Squad', got '%s'", squadData.Name)
    }
}
```

### Query Function Tests

Query functions need EntityManager setup:

```go
func TestGetUnitIDsInSquad(t *testing.T) {
    // Setup
    manager := common.CreateTestEntityManager()
    InitializeSquadData(manager)

    // Create test squad
    squadID := CreateSquad("Test Squad", FormationBalanced, manager)

    // Add units
    unit1ID := CreateUnit(squadID, testTemplate1, 0, 0, manager)
    unit2ID := CreateUnit(squadID, testTemplate2, 0, 1, manager)

    // Test query
    unitIDs := GetUnitIDsInSquad(squadID, manager)

    if len(unitIDs) != 2 {
        t.Errorf("Expected 2 units, got %d", len(unitIDs))
    }

    // Verify unit IDs
    found := make(map[ecs.EntityID]bool)
    for _, id := range unitIDs {
        found[id] = true
    }

    if !found[unit1ID] {
        t.Error("Unit 1 not found in squad")
    }
    if !found[unit2ID] {
        t.Error("Unit 2 not found in squad")
    }
}
```

### System Function Tests

System functions test game logic:

```go
func TestExecuteSquadAttack(t *testing.T) {
    // Setup
    manager := common.CreateTestEntityManager()
    InitializeSquadData(manager)

    // Create attacker squad
    attackerID := CreateSquad("Attacker", FormationOffensive, manager)
    CreateUnit(attackerID, strongUnitTemplate, 0, 0, manager)

    // Create defender squad
    defenderID := CreateSquad("Defender", FormationDefensive, manager)
    defenderUnitID := CreateUnit(defenderID, weakUnitTemplate, 0, 0, manager)

    // Get initial HP
    defenderAttr := common.GetAttributesByIDWithTag(manager, defenderUnitID, SquadMemberTag)
    initialHP := defenderAttr.CurrentHealth

    // Execute attack
    result := ExecuteSquadAttack(attackerID, defenderID, manager)

    // Verify result
    if result == nil {
        t.Fatal("Expected combat result, got nil")
    }

    if result.TotalDamage <= 0 {
        t.Error("Expected damage > 0")
    }

    // Verify defender took damage
    defenderAttr = common.GetAttributesByIDWithTag(manager, defenderUnitID, SquadMemberTag)
    if defenderAttr.CurrentHealth >= initialHP {
        t.Error("Defender should have taken damage")
    }
}
```

### Integration Tests

Test multiple systems working together:

```go
func TestSquadCombatIntegration(t *testing.T) {
    // Setup
    manager := common.CreateTestEntityManager()
    InitializeSquadData(manager)
    common.GlobalPositionSystem = systems.NewPositionSystem(manager.World)

    // Create positioned squads
    attackerID := CreateSquad("Attacker", FormationOffensive, manager)
    attackerEntity := GetSquadEntity(attackerID, manager)
    attackerPos := coords.LogicalPosition{X: 0, Y: 0}
    attackerEntity.AddComponent(common.PositionComponent, &attackerPos)
    common.GlobalPositionSystem.AddEntity(attackerEntity.GetID(), attackerPos)

    defenderID := CreateSquad("Defender", FormationDefensive, manager)
    defenderEntity := GetSquadEntity(defenderID, manager)
    defenderPos := coords.LogicalPosition{X: 1, Y: 0}
    defenderEntity.AddComponent(common.PositionComponent, &defenderPos)
    common.GlobalPositionSystem.AddEntity(defenderEntity.GetID(), defenderPos)

    // Add units with attack range
    CreateUnitWithRange(attackerID, meleeTemplate, 0, 0, 1, manager)
    CreateUnitWithRange(defenderID, meleeTemplate, 0, 0, 1, manager)

    // Calculate distance
    distance := GetSquadDistance(attackerID, defenderID, manager)
    if distance != 1 {
        t.Errorf("Expected distance 1, got %d", distance)
    }

    // Execute attack (should work - units in range)
    result := ExecuteSquadAttack(attackerID, defenderID, manager)
    if result.TotalDamage <= 0 {
        t.Error("Expected damage - units should be in range")
    }
}
```

### Test Helpers

Create test utilities for common setup:

```go
// testing/fixtures.go
func CreateTestEntityManager() *common.EntityManager {
    manager := common.NewEntityManager()

    // Initialize common components
    common.PositionComponent = manager.World.NewComponent()
    common.NameComponent = manager.World.NewComponent()
    common.AttributeComponent = manager.World.NewComponent()
    common.AllEntitiesTag = ecs.BuildTag()

    return manager
}

func CreateTestSquadWithUnits(manager *common.EntityManager, numUnits int) ecs.EntityID {
    squadID := CreateSquad("Test Squad", FormationBalanced, manager)

    for i := 0; i < numUnits; i++ {
        row := i / 3
        col := i % 3
        CreateUnit(squadID, defaultUnitTemplate, row, col, manager)
    }

    return squadID
}
```

### Benchmark Tests

Measure performance of query functions:

```go
func BenchmarkGetUnitIDsInSquad(b *testing.B) {
    // Setup
    manager := common.CreateTestEntityManager()
    InitializeSquadData(manager)

    // Create squad with 9 units
    squadID := CreateTestSquadWithUnits(manager, 9)

    b.ResetTimer()

    for i := 0; i < b.N; i++ {
        _ = GetUnitIDsInSquad(squadID, manager)
    }
}

func BenchmarkSpatialLookup(b *testing.B) {
    manager := common.CreateTestEntityManager()
    posSystem := systems.NewPositionSystem(manager.World)

    // Add 100 entities
    for i := 0; i < 100; i++ {
        pos := coords.LogicalPosition{X: i % 10, Y: i / 10}
        posSystem.AddEntity(ecs.EntityID(i+1), pos)
    }

    b.ResetTimer()

    for i := 0; i < b.N; i++ {
        pos := coords.LogicalPosition{X: i % 10, Y: i / 10}
        _ = posSystem.GetEntityIDAt(pos)
    }
}
```

---

## Migration Guide

### Migrating from Entity Pointers to EntityID

**Step 1: Identify Components with Entity Pointers**
```bash
grep -r "\*ecs.Entity" *.go
```

**Step 2: Change Component Definition**
```go
// Before
type SquadMemberData struct {
    Squad *ecs.Entity  // ❌
}

// After
type SquadMemberData struct {
    SquadID ecs.EntityID  // ✅
}
```

**Step 3: Update Creation Code**
```go
// Before
func CreateUnit(squad *ecs.Entity) {
    entity := manager.World.CreateEntity()
    entity.AddComponent(SquadMemberComponent, &SquadMemberData{
        Squad: squad,  // ❌
    })
}

// After
func CreateUnit(squadID ecs.EntityID) {
    entity := manager.World.CreateEntity()
    entity.AddComponent(SquadMemberComponent, &SquadMemberData{
        SquadID: squadID,  // ✅
    })
}
```

**Step 4: Update Access Code**
```go
// Before
memberData := common.GetComponentType[*SquadMemberData](entity, SquadMemberComponent)
squadEntity := memberData.Squad  // ❌
squadData := common.GetComponentType[*SquadData](squadEntity, SquadComponent)

// After
memberData := common.GetComponentType[*SquadMemberData](entity, SquadMemberComponent)
squadData := common.GetComponentTypeByIDWithTag[*SquadData](
    manager, memberData.SquadID, SquadTag, SquadComponent)  // ✅
```

### Migrating from Cached Relationships to Queries

**Step 1: Remove Cached Data**
```go
// Before
type SquadData struct {
    SquadID ecs.EntityID
    UnitIDs []ecs.EntityID  // ❌ Remove this
}

// After
type SquadData struct {
    SquadID ecs.EntityID
    // No cached UnitIDs
}
```

**Step 2: Create Query Function**
```go
func GetUnitIDsInSquad(squadID ecs.EntityID, manager *EntityManager) []ecs.EntityID {
    var unitIDs []ecs.EntityID
    for _, result := range manager.World.Query(SquadMemberTag) {
        memberData := common.GetComponentType[*SquadMemberData](result.Entity, SquadMemberComponent)
        if memberData.SquadID == squadID {
            unitIDs = append(unitIDs, result.Entity.GetID())
        }
    }
    return unitIDs
}
```

**Step 3: Replace All Access**
```go
// Before
squadData := GetSquadData(squadID, manager)
for _, unitID := range squadData.UnitIDs {  // ❌
    // Process unit
}

// After
unitIDs := GetUnitIDsInSquad(squadID, manager)
for _, unitID := range unitIDs {  // ✅
    // Process unit
}
```

**Step 4: Remove Synchronization Code**
```go
// Before - delete this
func AddUnitToSquad(squadID, unitID ecs.EntityID, manager *EntityManager) {
    // Create unit...

    // ❌ Delete manual sync code
    squadData := GetSquadData(squadID, manager)
    squadData.UnitIDs = append(squadData.UnitIDs, unitID)
}

// After - no sync needed
func AddUnitToSquad(squadID, unitID ecs.EntityID, manager *EntityManager) {
    // Create unit...
    // That's it - queries will automatically find it
}
```

### Migrating from Pointer Map Keys to Value Keys

**Step 1: Change Map Type**
```go
// Before
type PositionSystem struct {
    grid map[*coords.LogicalPosition][]ecs.EntityID  // ❌
}

// After
type PositionSystem struct {
    grid map[coords.LogicalPosition][]ecs.EntityID  // ✅
}
```

**Step 2: Update Add Function**
```go
// Before
func (ps *PositionSystem) AddEntity(entityID ecs.EntityID, pos *coords.LogicalPosition) {
    ps.grid[pos] = append(ps.grid[pos], entityID)  // ❌ Pointer key
}

// After
func (ps *PositionSystem) AddEntity(entityID ecs.EntityID, pos coords.LogicalPosition) {
    ps.grid[pos] = append(ps.grid[pos], entityID)  // ✅ Value key
}
```

**Step 3: Update Lookup Function**
```go
// Before
func (ps *PositionSystem) GetEntityAt(pos coords.LogicalPosition) ecs.EntityID {
    key := &pos  // ❌ Create temporary pointer
    if ids, ok := ps.grid[key]; ok {
        return ids[0]
    }
    return 0
}

// After
func (ps *PositionSystem) GetEntityAt(pos coords.LogicalPosition) ecs.EntityID {
    if ids, ok := ps.grid[pos]; ok {  // ✅ Use value directly
        return ids[0]
    }
    return 0
}
```

**Step 4: Update All Call Sites**
```go
// Before
pos := &coords.LogicalPosition{X: 10, Y: 20}  // ❌ Pointer
posSystem.AddEntity(entityID, pos)

// After
pos := coords.LogicalPosition{X: 10, Y: 20}  // ✅ Value
posSystem.AddEntity(entityID, pos)
```

### Migrating to Tag-Scoped Queries

**Step 1: Identify Slow Queries**
```go
// Before - slow
data := common.GetComponentTypeByID[*SquadData](manager, squadID, SquadComponent)
```

**Step 2: Use Tag-Scoped Version**
```go
// After - fast
data := common.GetComponentTypeByIDWithTag[*SquadData](
    manager, squadID, SquadTag, SquadComponent)
```

**Step 3: Benchmark**
```go
func BenchmarkComponentAccess(b *testing.B) {
    // ... setup ...

    b.Run("Without Tag", func(b *testing.B) {
        for i := 0; i < b.N; i++ {
            _ = common.GetComponentTypeByID[*SquadData](manager, squadID, SquadComponent)
        }
    })

    b.Run("With Tag", func(b *testing.B) {
        for i := 0; i < b.N; i++ {
            _ = common.GetComponentTypeByIDWithTag[*SquadData](
                manager, squadID, SquadTag, SquadComponent)
        }
    })
}
```

---

## Appendix: Quick Reference

### Component Checklist

- [ ] Pure data structure (no methods except pure calculations)
- [ ] Uses `ecs.EntityID` for relationships (not `*ecs.Entity`)
- [ ] Registered in `Init*Components()` function
- [ ] Tag created in `Init*Tags()` function
- [ ] Named with `*Data` suffix (e.g., `SquadData`)
- [ ] Component variable named with `*Component` suffix (e.g., `SquadComponent`)
- [ ] Tag variable named with `*Tag` suffix (e.g., `SquadTag`)

### System Function Checklist

- [ ] Receives `*EntityManager` as parameter (not global)
- [ ] Logic in function, not component method
- [ ] Uses query-based relationships (no cached data)
- [ ] Proper naming: `Get*`, `Find*`, `Is*`, `Execute*`, `Update*`
- [ ] Lives in `*queries.go` (read-only) or `*system.go` (modifies state)
- [ ] Entity cleanup includes `GlobalPositionSystem.RemoveEntity()`

### Query Checklist

- [ ] Uses specific tag, not `AllEntitiesTag`
- [ ] Uses `*WithTag` helpers when entity type known
- [ ] Returns `[]ecs.EntityID`, not `[]*ecs.Entity`
- [ ] Returns `nil` or empty slice when not found (not error)
- [ ] Filters applied during iteration (not separate pass)

### Performance Checklist

- [ ] Value-based map keys (not pointer-based)
- [ ] Tag-scoped queries (`*WithTag` functions)
- [ ] Spatial queries use `GlobalPositionSystem` (O(1))
- [ ] Pre-allocated slices when capacity known
- [ ] Early exit in boolean queries
- [ ] Deduplication when needed (multi-cell units, etc.)

### File Organization Checklist

- [ ] `components.go` - Only component definitions
- [ ] `*manager.go` - Only initialization functions
- [ ] `*queries.go` - Only read-only query functions
- [ ] `*system.go` - Only state-modifying logic
- [ ] `*_test.go` - Tests with proper setup
- [ ] Package contains one cohesive system

---

**End of ECS Architecture Guide**

This document is the definitive reference for ECS development in TinkerRogue. When in doubt, refer to the reference implementations and follow the patterns demonstrated in `squads/`, `gear/Inventory.go`, and `systems/positionsystem.go`.
