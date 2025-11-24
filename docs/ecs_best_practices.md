# ECS Best Practices for TinkerRogue

**Last Updated:** 2025-11-23

This document defines the Entity Component System (ECS) architecture patterns used in TinkerRogue. All new code must follow these patterns to maintain consistency and performance.

---

## Core Principles

The squad and inventory systems demonstrate perfect ECS architecture. **Apply these patterns to all new code.**

### 1. Pure Data Components

Components should contain ZERO logic - only data fields.

```go
// ✅ CORRECT - Pure data
type SquadData struct {
    SquadID       ecs.EntityID
    Formation     FormationType
    Name          string
    Morale        int
}

// ❌ WRONG - Has methods
type SquadData struct {
    // ... fields ...
}
func (s *SquadData) GetMorale() int { return s.Morale } // NO METHODS!
```

**Why:**
- Components are pure data containers
- Logic belongs in system functions
- Keeps data and behavior properly separated
- Enables better testability and maintainability

---

### 2. Native EntityID - No Pointers

Always use `ecs.EntityID` for relationships, never store entity pointers.

```go
// ✅ CORRECT - Use EntityID
type SquadMemberData struct {
    SquadID ecs.EntityID  // Reference by ID
}

// ❌ WRONG - Entity pointers break ECS
type SquadMemberData struct {
    Squad *ecs.Entity  // NEVER store entity pointers!
}
```

**Why:**
- Entity pointers can become invalid when entities are disposed
- EntityIDs are stable and safe across the entity lifecycle
- Prevents memory leaks and dangling pointer bugs
- Forces explicit querying which is the ECS way

---

### 3. Query-Based Relationships

Discover relationships through ECS queries, don't cache them.

```go
// ✅ CORRECT - Query when needed
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

// ❌ WRONG - Storing cached relationships
type SquadData struct {
    UnitIDs []ecs.EntityID  // Don't cache - query instead!
}
```

**Why:**
- Cached relationships require manual synchronization
- Queries are always up-to-date
- Reduces component coupling
- Prevents stale data bugs

**Performance Note:**
- Query performance is excellent in practice (O(n) over relevant entities)
- If profiling shows a specific query is a bottleneck, optimize that specific case
- Don't prematurely optimize by caching

---

### 4. System-Based Logic

All behavior belongs in system functions, not component methods.

```go
// ✅ CORRECT - System function
func ExecuteSquadAttack(attackerSquad, defenderSquad ecs.EntityID, manager *common.EntityManager) *CombatResult {
    // Combat logic here
}

// ❌ WRONG - Logic in component
func (s *SquadData) Attack(target *SquadData) {
    // NO! Put this in a system function
}
```

**Why:**
- Components are data, systems are logic
- System functions can access multiple components and entities
- Enables better testing (mock EntityManager, not individual components)
- Follows ECS architectural principles

---

### 5. Value-Based Map Keys

Use value types as map keys for O(1) performance, not pointers.

```go
// ✅ CORRECT - Value-based keys
type PositionSystem struct {
    entityGrid map[coords.LogicalPosition][]ecs.EntityID  // Value key
}

// ❌ WRONG - Pointer keys are slower
type PositionSystem struct {
    entityGrid map[*coords.LogicalPosition][]ecs.EntityID  // Pointer key - 50x slower!
}
```

**Why:**
- Value-based keys use struct equality (fast)
- Pointer keys use pointer comparison (requires creating new pointers for lookups)
- Measured 50x performance improvement in position system refactor
- Go maps are optimized for value-type keys

---

## Component Registration & Initialization

### Component Declaration Pattern

Components must be registered with the ECS before use. Follow this exact pattern:

#### Step 1: Declare Component Variables (components.go)

```go
package mypackage

import "github.com/bytearena/ecs"

// Global component variables - initialized to nil
var (
    MyDataComponent      *ecs.Component
    MyOtherComponent     *ecs.Component

    MyTag       ecs.Tag  // For querying MyDataComponent entities
    MyOtherTag  ecs.Tag  // For querying MyOtherComponent entities
)

// Component data structs
type MyData struct {
    Field1 string
    Field2 int
}

type MyOtherData struct {
    Field string
}
```

#### Step 2: Initialize Components (initialization function)

**CRITICAL:** Components must be registered with `manager.World.NewComponent()` before use. **Never skip this step or components will be nil!**

```go
// InitMyComponents registers all components with the ECS manager.
// Call this during game initialization, BEFORE creating any entities.
func InitMyComponents(manager *common.EntityManager) {
    // ✅ CORRECT - Register each component
    MyDataComponent = manager.World.NewComponent()
    MyOtherComponent = manager.World.NewComponent()
}

// InitMyTags creates tags for querying entities.
// Call this AFTER InitMyComponents.
func InitMyTags(manager *common.EntityManager) {
    // ✅ CORRECT - Build tags from components
    MyTag = ecs.BuildTag(MyDataComponent)
    MyOtherTag = ecs.BuildTag(MyOtherComponent)

    // Optional: Register tags in EntityManager for name-based lookup
    manager.WorldTags["mytag"] = MyTag
    manager.WorldTags["myothertag"] = MyOtherTag
}

// InitializeMySystem is the main initialization function.
// Call this from game_main during startup.
func InitializeMySystem(manager *common.EntityManager) error {
    InitMyComponents(manager)
    InitMyTags(manager)
    // Any other initialization...
    return nil
}
```

#### Step 3: Call Initialization During Game Startup

```go
// In game_main/gameinit.go or similar
func InitializeGame() {
    manager := common.NewEntityManager()

    // Initialize all systems
    if err := mypackage.InitializeMySystem(manager); err != nil {
        log.Fatal(err)
    }

    // Now safe to create entities with components
}
```

### Multi-Component Tags

Tags can combine multiple components for complex queries:

```go
// Single component tag
SquadTag = ecs.BuildTag(SquadComponent)

// Multi-component tag - matches entities with ALL specified components
LeaderTag = ecs.BuildTag(LeaderComponent, SquadMemberComponent)

// Usage: Query returns only entities with BOTH LeaderComponent AND SquadMemberComponent
for _, result := range manager.World.Query(LeaderTag) {
    // This entity is guaranteed to have both components
    leaderData := common.GetComponentType[*LeaderData](result.Entity, LeaderComponent)
    memberData := common.GetComponentType[*SquadMemberData](result.Entity, SquadMemberComponent)
}
```

### Common Initialization Mistakes

```go
// ❌ WRONG - Using component before initialization
var MyComponent *ecs.Component  // nil!
entity.AddComponent(MyComponent, data)  // PANIC: nil component

// ✅ CORRECT - Initialize first
MyComponent = manager.World.NewComponent()
entity.AddComponent(MyComponent, data)  // Works

// ❌ WRONG - Building tag before component initialization
var MyComponent *ecs.Component  // nil!
MyTag = ecs.BuildTag(MyComponent)  // PANIC: nil component

// ✅ CORRECT - Initialize component, then build tag
MyComponent = manager.World.NewComponent()
MyTag = ecs.BuildTag(MyComponent)

// ❌ WRONG - Trying to use component from wrong package without initialization
import "game_main/squads"
// squads.SquadComponent is nil if InitializeSquadData not called!
entity.AddComponent(squads.SquadComponent, data)  // PANIC

// ✅ CORRECT - Initialize system first
squads.InitializeSquadData(manager)
entity.AddComponent(squads.SquadComponent, data)  // Works
```

### Reference Implementation

See `squads/squadmanager.go` for perfect component initialization:

```go
// Lines 15-28: InitSquadComponents
func InitSquadComponents(manager *common.EntityManager) {
    SquadComponent = manager.World.NewComponent()
    SquadMemberComponent = manager.World.NewComponent()
    GridPositionComponent = manager.World.NewComponent()
    // ... more components
}

// Lines 32-40: InitSquadTags
func InitSquadTags(manager *common.EntityManager) {
    SquadTag = ecs.BuildTag(SquadComponent)
    SquadMemberTag = ecs.BuildTag(SquadMemberComponent)
    LeaderTag = ecs.BuildTag(LeaderComponent, SquadMemberComponent)

    manager.WorldTags["squad"] = SquadTag
    manager.WorldTags["squadmember"] = SquadMemberTag
    manager.WorldTags["leader"] = LeaderTag
}

// Lines 45-52: InitializeSquadData (main entry point)
func InitializeSquadData(manager *common.EntityManager) error {
    InitSquadComponents(manager)
    InitSquadTags(manager)
    // Additional initialization...
    return nil
}
```

---

## EntityManager & Component Access

The `common.EntityManager` wrapper provides type-safe component access functions. **NEVER** access the underlying ECS library directly - always use these helper functions.

### EntityManager Structure

```go
// EntityManager wraps ecs.Manager and provides utilities
type EntityManager struct {
    World     *ecs.Manager           // Underlying ECS manager
    WorldTags map[string]ecs.Tag     // Named tag registry
}

// Global systems (initialized during game setup)
var (
    GlobalPositionSystem *systems.PositionSystem  // O(1) spatial queries
)
```

### Component Access Functions (common/ecsutil.go)

#### GetComponentType - From Entity Pointer

Use when you already have an entity from a query:

```go
// ✅ CORRECT - Entity from query result
for _, result := range manager.World.Query(SquadTag) {
    entity := result.Entity
    squadData := common.GetComponentType[*SquadData](entity, SquadComponent)
    // Use squadData...
}

// Type parameter T must match component data type
// Returns zero value (nil for pointers) if component not found
```

#### GetComponentTypeByID - From EntityID

Use when you only have an EntityID:

```go
// ✅ CORRECT - Access component by EntityID
func ProcessUnit(unitID ecs.EntityID, manager *common.EntityManager) {
    attributes := common.GetComponentTypeByID[*Attributes](manager, unitID, AttributeComponent)
    if attributes == nil {
        // Component not found
        return
    }
    // Use attributes...
}
```

#### GetComponentTypeByIDWithTag - Optimized Query

Use when you know which tag the entity belongs to (faster than GetComponentTypeByID):

```go
// ✅ CORRECT - Query within specific tag
func GetSquadName(squadID ecs.EntityID, manager *common.EntityManager) string {
    squadData := common.GetComponentTypeByIDWithTag[*SquadData](
        manager, squadID, SquadTag, SquadComponent)
    if squadData == nil {
        return ""
    }
    return squadData.Name
}
```

**Why use WithTag version:**
- Searches only entities with SquadTag (typically 10-50 entities)
- Instead of AllEntitiesTag (potentially 1000+ entities)
- Can be 10-100x faster for large entity counts

#### HasComponent & HasComponentByIDWithTag

Check if entity has a component before accessing:

```go
// Check by EntityID (searches all entities)
if manager.HasComponent(entityID, PositionComponent) {
    pos := common.GetComponentTypeByID[*coords.LogicalPosition](
        manager, entityID, PositionComponent)
}

// Check by EntityID with tag (faster)
if manager.HasComponentByIDWithTag(unitID, SquadMemberTag, GridPositionComponent) {
    gridPos := common.GetComponentTypeByIDWithTag[*GridPositionData](
        manager, unitID, SquadMemberTag, GridPositionComponent)
}
```

### Entity Finding Functions (common/ecsutil.go)

#### FindEntityByID - Get Entity Pointer

**Use only when you need the entity pointer for operations that require it:**

```go
// ✅ CORRECT - Need entity pointer to add component
func AddLeaderComponent(unitID ecs.EntityID, manager *common.EntityManager) {
    entity := common.FindEntityByID(manager, unitID)
    if entity == nil {
        return  // Entity not found
    }
    // AddComponent requires entity pointer
    entity.AddComponent(LeaderComponent, &LeaderData{})
}

// ❌ WRONG - Don't use FindEntityByID just to get component
entity := common.FindEntityByID(manager, unitID)  // Wasteful
attrs := common.GetComponentType[*Attributes](entity, AttributeComponent)

// ✅ CORRECT - Use direct component access
attrs := common.GetComponentTypeByID[*Attributes](manager, unitID, AttributeComponent)
```

**Valid use cases for FindEntityByID:**
- `entity.AddComponent(component, data)` - ECS library requires entity pointer
- `entity.RemoveComponent(component)` - ECS library requires entity pointer
- `entity.HasComponent(component)` - Though `manager.HasComponent()` is preferred

#### FindEntityByIDWithTag - Tag-Scoped Search

Faster version that searches within a specific tag:

```go
// ✅ CORRECT - Find entity within tag scope
func PromoteToLeader(unitID ecs.EntityID, manager *common.EntityManager) {
    entity := common.FindEntityByIDWithTag(manager, unitID, SquadMemberTag)
    if entity == nil {
        return  // Not a squad member
    }
    entity.AddComponent(LeaderComponent, &LeaderData{})
    entity.AddTag(LeaderTag)
}
```

#### FindEntityIDWithTag - Verify Entity Has Tag

Check if an EntityID belongs to a specific tag:

```go
// ✅ CORRECT - Verify entity has tag
func IsSquadMember(unitID ecs.EntityID, manager *common.EntityManager) bool {
    return common.FindEntityIDWithTag(manager, unitID, SquadMemberTag) != 0
}
```

### Specialized Helper Functions

#### Position Access

```go
// From entity pointer
pos := common.GetPosition(entity)

// From EntityID
pos := common.GetPositionByID(manager, entityID)

// From EntityID with tag (faster)
pos := common.GetPositionByIDWithTag(manager, entityID, MonsterTag)
```

#### Attributes Access

```go
// From entity pointer
attrs := common.GetAttributes(entity)

// From EntityID
attrs := common.GetAttributesByID(manager, entityID)

// From EntityID with tag (faster)
attrs := common.GetAttributesByIDWithTag(manager, entityID, SquadMemberTag)
```

#### Spatial Queries (O(1) Performance)

```go
// ✅ CORRECT - O(1) position lookup
entityIDs := common.GlobalPositionSystem.GetEntitiesAtPosition(logicalPos)

// Get single entity at position
entityID := common.GlobalPositionSystem.GetEntityIDAt(logicalPos)
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

### Function Selection Guide

| Scenario | Function to Use |
|----------|----------------|
| Have entity from query, need component | `common.GetComponentType[T](entity, component)` |
| Have EntityID, need component | `common.GetComponentTypeByID[T](manager, id, component)` |
| Have EntityID + tag, need component | `common.GetComponentTypeByIDWithTag[T](manager, id, tag, component)` |
| Need to add/remove component | `common.FindEntityByID(manager, id)` then `entity.AddComponent()` |
| Need to add component + have tag | `common.FindEntityByIDWithTag(manager, id, tag)` then `entity.AddComponent()` |
| Check if entity has component | `manager.HasComponent(id, component)` |
| Check if entity has component + tag | `manager.HasComponentByIDWithTag(id, tag, component)` |
| Verify entity exists with tag | `common.FindEntityIDWithTag(manager, id, tag) != 0` |
| Get entity at position | `common.GlobalPositionSystem.GetEntityIDAt(pos)` |
| Get all entities at position | `common.GlobalPositionSystem.GetEntitiesAtPosition(pos)` |

### Common Access Mistakes

```go
// ❌ WRONG - Manual type assertion (risky)
data := entity.GetComponent(SquadComponent).(*SquadData)

// ✅ CORRECT - Type-safe helper
data := common.GetComponentType[*SquadData](entity, SquadComponent)

// ❌ WRONG - Unnecessary entity lookup for component access
entity := common.FindEntityByID(manager, unitID)
attrs := common.GetComponentType[*Attributes](entity, AttributeComponent)

// ✅ CORRECT - Direct component access
attrs := common.GetComponentTypeByID[*Attributes](manager, unitID, AttributeComponent)

// ❌ WRONG - Using AllEntitiesTag when tag is known
data := common.GetComponentTypeByID[*SquadData](manager, squadID, SquadComponent)
// Searches ALL entities

// ✅ CORRECT - Use specific tag
data := common.GetComponentTypeByIDWithTag[*SquadData](manager, squadID, SquadTag, SquadComponent)
// Searches only squad entities (10-100x faster)

// ❌ WRONG - Calling ecs.Manager functions directly
component := manager.World.GetComponent(entity, SquadComponent)  // NO!

// ✅ CORRECT - Use EntityManager helpers
component := common.GetComponentTypeByID[*SquadData](manager, entityID, SquadComponent)
```

---

## ECS File Organization

### Package Structure

One major system per package with consistent file naming:

```
squads/
├── components.go      # Component data definitions
├── squadqueries.go    # Query functions
├── squadcombat.go     # Combat system logic
├── squadabilities.go  # Ability system logic
├── squadcreation.go   # Creation system logic
└── squads_test.go     # Tests
```

### File Responsibilities

#### `components.go` - Component Definitions Only

```go
package squads

// Component data struct
type SquadData struct {
    SquadID       ecs.EntityID
    Formation     FormationType
    Name          string
    Morale        int
}

// Component registration variable
var SquadComponent *ecs.Component

// Tag registration variable
var SquadTag ecs.Tag
```

**Rules:**
- Only data struct definitions
- Component registration variables
- Tag registration variables
- NO logic, NO methods, NO query functions

#### `*queries.go` - Query Functions

```go
package squads

// Query functions for finding entities
func GetSquadEntity(squadID ecs.EntityID, manager *common.EntityManager) *ecs.Entity {
    for _, result := range manager.World.Query(SquadTag) {
        squadData := common.GetComponentType[*SquadData](result.Entity, SquadComponent)
        if squadData.SquadID == squadID {
            return result.Entity
        }
    }
    return nil
}

func GetUnitIDsInSquad(squadID ecs.EntityID, manager *common.EntityManager) []ecs.EntityID {
    // Query implementation
}

func IsSquadDestroyed(squadID ecs.EntityID, manager *common.EntityManager) bool {
    // Query implementation
}
```

**Rules:**
- Query functions that search/filter entities
- Functions that check entity state
- Functions that extract data from components
- NO logic that modifies components

#### `*system.go` - System Logic

```go
package squads

// System functions that modify state
func ExecuteSquadAttack(attackerSquad, defenderSquad ecs.EntityID, manager *common.EntityManager) *CombatResult {
    // Get components
    attackerData := common.GetComponentTypeByIDWithTag[*SquadData](manager, attackerSquad, SquadTag, SquadComponent)
    defenderData := common.GetComponentTypeByIDWithTag[*SquadData](manager, defenderSquad, SquadTag, SquadComponent)

    // Perform combat logic
    damage := calculateDamage(attackerData, defenderData)
    applyDamage(defenderSquad, damage, manager)

    // Return result
    return &CombatResult{/* ... */}
}

func UpdateSquadCapacity(squadID ecs.EntityID, manager *common.EntityManager) {
    // Update logic
}
```

**Rules:**
- Functions that modify component data
- Functions that create/destroy entities
- Functions that implement game mechanics
- Can call query functions from `*queries.go`

---

## Naming Conventions

### Components

```go
// Data struct - suffix with "Data"
type SquadData struct { ... }
type ActionStateData struct { ... }
type InventoryData struct { ... }

// Component variable - suffix with "Component"
var SquadComponent *ecs.Component
var ActionStateComponent *ecs.Component
var InventoryComponent *ecs.Component

// Tag variable - suffix with "Tag"
var SquadTag ecs.Tag
var SquadMemberTag ecs.Tag
var ItemTag ecs.Tag
```

### Query Functions

```go
// Get single entity - prefix with "Get"
func GetSquadEntity(squadID ecs.EntityID, manager *common.EntityManager) *ecs.Entity

// Get multiple entities - plural noun
func GetUnitIDsInSquad(squadID ecs.EntityID, manager *common.EntityManager) []ecs.EntityID

// Find with search - prefix with "Find"
func FindActionStateBySquadID(squadID ecs.EntityID, manager *common.EntityManager) *ActionStateData

// Boolean check - prefix with "Is" or "Can"
func IsSquadDestroyed(squadID ecs.EntityID, manager *common.EntityManager) bool
func CanSquadAct(squadID ecs.EntityID, manager *common.EntityManager) bool
```

### System Functions

```go
// Action functions - use verb
func ExecuteSquadAttack(attackerSquad, defenderSquad ecs.EntityID, manager *common.EntityManager) *CombatResult
func CheckAndTriggerAbilities(squadID ecs.EntityID, manager *common.EntityManager)
func UpdateSquadCapacity(squadID ecs.EntityID, manager *common.EntityManager)
func CreateSquad(name string, formation FormationType, manager *common.EntityManager) ecs.EntityID
```

---

## Common ECS Patterns

### Component Access Pattern

```go
// ✅ CORRECT - Type-safe component access
squadData := common.GetComponentType[*SquadData](entity, SquadComponent)

// ✅ CORRECT - Check if component exists
if entity.HasComponent(SquadComponent) {
    squadData := common.GetComponentType[*SquadData](entity, SquadComponent)
}

// ✅ CORRECT - Get component by entity ID
squadData := common.GetComponentTypeByIDWithTag[*SquadData](manager, entityID, SquadTag, SquadComponent)

// ❌ WRONG - Don't cast manually
data := entity.GetComponent(SquadComponent).(*SquadData)  // Risky!
```

### Query Pattern

```go
// Standard query pattern
for _, result := range manager.World.Query(SquadTag) {
    entity := result.Entity
    squadData := common.GetComponentType[*SquadData](entity, SquadComponent)

    // Process entity
    processSquad(entity, squadData)
}

// Query with filtering
func FindEntitiesByCondition(condition func(*SomeData) bool, manager *common.EntityManager) []*ecs.Entity {
    var matches []*ecs.Entity
    for _, result := range manager.World.Query(SomeTag) {
        data := common.GetComponentType[*SomeData](result.Entity, SomeComponent)
        if condition(data) {
            matches = append(matches, result.Entity)
        }
    }
    return matches
}
```

### Entity Creation Pattern

```go
func CreateSquad(name string, formation FormationType, manager *common.EntityManager) ecs.EntityID {
    // Create entity
    entity := manager.World.CreateEntity()

    // Generate unique ID
    squadID := ecs.EntityID(manager.NextID())

    // Add components
    entity.AddComponent(SquadComponent, &SquadData{
        SquadID:   squadID,
        Formation: formation,
        Name:      name,
        Morale:    100,
    })

    // Add tags
    entity.AddTag(SquadTag)

    return squadID
}
```

### Entity Cleanup Pattern

```go
func DestroySquad(squadID ecs.EntityID, manager *common.EntityManager) {
    // Find entity
    entity := GetSquadEntity(squadID, manager)
    if entity == nil {
        return
    }

    // Get position for spatial grid cleanup
    if entity.HasComponent(common.PositionComponent) {
        posData := common.GetComponentType[*common.PositionData](entity, common.PositionComponent)
        common.GlobalPositionSystem.RemoveEntity(entity.GetID(), posData.LogicalPosition)
    }

    // Dispose entity
    manager.World.DisposeEntities(entity)
}
```

---

## Reference Implementations

### Perfect ECS Examples

Study these files for proper ECS architecture:

#### Squad System (`squads/`)
- **Lines of Code:** 4,933
- **Components:** 8 components (SquadData, SquadMemberData, LeaderData, etc.)
- **Query Functions:** 7+ functions in `squadqueries.go`
- **Systems:** Combat, abilities, creation, visualization
- **Why Perfect:** Pure data components, query-based relationships, system-based logic

#### Inventory System (`gear/Inventory.go`)
- **Lines of Code:** 241
- **Components:** 1 pure data component (InventoryData)
- **System Functions:** 9 functions (AddItem, RemoveItem, GetItems, etc.)
- **Why Perfect:** EntityID-based relationships, no cached state, pure system functions

#### Item System (`gear/items.go`)
- **Lines of Code:** 177
- **Relationships:** Uses EntityID for item-owner relationships
- **Why Perfect:** No entity pointers, query-based item lookup

#### Position System (`systems/positionsystem.go`)
- **Lines of Code:** 183
- **Data Structure:** O(1) spatial grid with value-based keys
- **Why Perfect:** Value-based map keys (50x faster than pointer keys)

### Anti-Patterns Fixed

Learn from these refactoring case studies:

#### Position System Refactor (2025-10-15)
- **Before:** `map[*coords.LogicalPosition][]ecs.EntityID` (pointer keys)
- **After:** `map[coords.LogicalPosition][]ecs.EntityID` (value keys)
- **Result:** 50x performance improvement

#### Inventory System Refactor (2025-10-21)
- **Before:** Stored entity pointers in InventoryData
- **After:** Uses EntityID for relationships
- **Result:** No more dangling pointer bugs

#### TileContents Refactor (2025-11-08)
- **Before:** `[]*ecs.Entity` (entity pointer array)
- **After:** `[]ecs.EntityID` (entity ID array)
- **Result:** Safe entity references, no memory leaks

#### Equipment System (Pending)
- **Current:** Still uses entity pointers
- **Planned:** Refactor to use EntityID like inventory system
- **Status:** Scheduled for next refactoring sprint

---

## Code Review Checklist

Use this checklist when reviewing ECS code:

### Component Review
- [ ] Component struct has ZERO methods
- [ ] All fields are pure data (no functions, no interfaces with behavior)
- [ ] Uses `ecs.EntityID` for relationships, not entity pointers
- [ ] Follows naming convention: `TypeData` for struct, `TypeComponent` for variable

### Query Function Review
- [ ] Functions are read-only (don't modify components)
- [ ] Uses proper naming: `Get*`, `Find*`, `Is*`, `Can*`
- [ ] Uses `common.GetComponentType[T]()` for type-safe access
- [ ] Returns nil or empty slice when not found (not error)
- [ ] Lives in `*queries.go` file

### System Function Review
- [ ] Logic lives in system function, not component method
- [ ] Queries for relationships instead of using cached data
- [ ] Uses EntityManager parameter (not global)
- [ ] Properly cleans up entities (removes from spatial grids, etc.)
- [ ] Lives in `*system.go` file

### Architecture Review
- [ ] No cached relationships (uses queries instead)
- [ ] No global variables (except documented: CoordManager, GlobalPositionSystem)
- [ ] Value-based map keys, not pointer-based
- [ ] Follows package structure: `components.go`, `*queries.go`, `*system.go`

---

## Common Mistakes to Avoid

### Mistake #1: Adding Methods to Components

```go
// ❌ WRONG
type SquadData struct {
    Morale int
}
func (s *SquadData) GetMorale() int { return s.Morale }

// ✅ CORRECT - Just access the field directly
squadData.Morale
```

### Mistake #2: Caching Relationships

```go
// ❌ WRONG - Cached relationship
type SquadData struct {
    UnitIDs []ecs.EntityID  // Must manually sync!
}

// ✅ CORRECT - Query when needed
func GetUnitIDsInSquad(squadID ecs.EntityID, manager *common.EntityManager) []ecs.EntityID {
    // Query implementation
}
```

### Mistake #3: Entity Pointer Storage

```go
// ❌ WRONG - Storing entity pointer
type EquipmentData struct {
    Owner *ecs.Entity  // Can become invalid!
}

// ✅ CORRECT - Store EntityID
type EquipmentData struct {
    OwnerID ecs.EntityID  // Always safe
}
```

### Mistake #4: Logic in Components

```go
// ❌ WRONG - Logic in component method
func (s *SquadData) TakeDamage(amount int) {
    s.Morale -= amount
}

// ✅ CORRECT - Logic in system function
func ApplyMoraleDamage(squadID ecs.EntityID, amount int, manager *common.EntityManager) {
    squadData := common.GetComponentTypeByIDWithTag[*SquadData](manager, squadID, SquadTag, SquadComponent)
    squadData.Morale -= amount
}
```

### Mistake #5: Pointer Map Keys

```go
// ❌ WRONG - Pointer-based map key
type PositionSystem struct {
    grid map[*coords.LogicalPosition][]ecs.EntityID
}

// ✅ CORRECT - Value-based map key
type PositionSystem struct {
    grid map[coords.LogicalPosition][]ecs.EntityID
}
```

---

## Performance Considerations

### Query Performance

Queries are O(n) over entities with the tag, which is fast in practice:

```go
// This is fine - typically 10-100 entities
for _, result := range manager.World.Query(SquadTag) {
    // Process squad
}
```

**When queries become a problem:**
- Querying in tight inner loops (Update/Render every frame for all entities)
- Searching large entity sets (1000+ entities) repeatedly

**Solution:**
1. Profile first to confirm the bottleneck
2. Consider caching for that specific case
3. Document why caching is needed
4. Add synchronization logic carefully

### Spatial Queries

Use `common.GlobalPositionSystem` for O(1) position lookups:

```go
// ✅ CORRECT - O(1) spatial query
entityIDs := common.GlobalPositionSystem.GetEntitiesAtPosition(logicalPos)

// ❌ WRONG - O(n) brute force search
for _, result := range manager.World.Query(PositionTag) {
    posData := common.GetComponentType[*PositionData](result.Entity, PositionComponent)
    if posData.LogicalPosition == targetPos {
        // Found it!
    }
}
```

### Value vs Pointer Keys

Always use value types for map keys:

```go
// ✅ CORRECT - Value key (fast)
grid map[coords.LogicalPosition][]ecs.EntityID

// ❌ WRONG - Pointer key (50x slower)
grid map[*coords.LogicalPosition][]ecs.EntityID
```

**Why value keys are faster:**
- Value comparison uses struct equality (compiler optimized)
- Pointer comparison requires creating temporary pointers for lookups
- Measured 50x performance difference in position system

---

## Testing ECS Code

### Component Tests

Components are pure data - test by creating instances:

```go
func TestSquadData(t *testing.T) {
    squadData := &SquadData{
        SquadID:   1,
        Formation: FormationLine,
        Name:      "Test Squad",
        Morale:    100,
    }

    assert.Equal(t, 100, squadData.Morale)
    assert.Equal(t, "Test Squad", squadData.Name)
}
```

### Query Function Tests

Query functions need EntityManager setup:

```go
func TestGetSquadEntity(t *testing.T) {
    manager := common.CreateTestEntityManager()

    // Create test squad
    entity := manager.World.CreateEntity()
    squadID := ecs.EntityID(1)
    entity.AddComponent(SquadComponent, &SquadData{SquadID: squadID})
    entity.AddTag(SquadTag)

    // Test query
    result := GetSquadEntity(squadID, manager)
    assert.NotNil(t, result)

    // Test not found
    notFound := GetSquadEntity(ecs.EntityID(999), manager)
    assert.Nil(t, notFound)
}
```

### System Function Tests

System functions test game logic:

```go
func TestExecuteSquadAttack(t *testing.T) {
    manager := common.CreateTestEntityManager()

    // Create attacker squad
    attackerID := CreateSquad("Attacker", FormationLine, manager)

    // Create defender squad
    defenderID := CreateSquad("Defender", FormationLine, manager)

    // Execute attack
    result := ExecuteSquadAttack(attackerID, defenderID, manager)

    // Verify result
    assert.NotNil(t, result)
    assert.Greater(t, result.DamageDealt, 0)

    // Verify defender took damage
    defenderData := common.GetComponentTypeByIDWithTag[*SquadData](manager, defenderID, SquadTag, SquadComponent)
    assert.Less(t, defenderData.Morale, 100)
}
```

---

## Additional Resources

### Related Documentation
- **CLAUDE.md** - Main developer reference guide
- **Project Structure** - See CLAUDE.md "Project Structure & Core Files"
- **Utility Functions** - See CLAUDE.md "Utility Functions & Common Patterns"

### Reference Code
- **squads/components.go** - Perfect component definitions
- **squads/squadqueries.go** - Perfect query functions
- **squads/squadcombat.go** - Perfect system functions
- **gear/Inventory.go** - Perfect pure ECS component
- **systems/positionsystem.go** - Perfect value-based map keys

---

**Remember:** These patterns exist for a reason. Study the reference implementations and apply these patterns consistently across all new code.
