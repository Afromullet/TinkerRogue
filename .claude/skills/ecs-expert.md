# ECS Expert - Entity Component System Development Guide

**Purpose:** This skill provides comprehensive guidance for developing features using the Entity Component System (ECS) architecture in the TinkerRogue codebase. Follow these patterns to maintain code quality, performance, and consistency with the existing codebase.

---

## Table of Contents

1. [Core ECS Principles](#core-ecs-principles)
2. [Component Design Patterns](#component-design-patterns)
3. [System Function Patterns](#system-function-patterns)
4. [Entity Management](#entity-management)
5. [Query Patterns](#query-patterns)
6. [Common Anti-Patterns](#common-anti-patterns)
7. [Performance Best Practices](#performance-best-practices)
8. [Reference Implementations](#reference-implementations)
9. [Step-by-Step Workflows](#step-by-step-workflows)

---

## Core ECS Principles

### 1. Pure Data Components (No Logic)

**Rule:** Components must contain ONLY data fields. No methods (except simple helpers like String(), Copy()).

**✅ Good Example:**
```go
// From squads/components.go
type SquadData struct {
    SquadID       ecs.EntityID  // Native entity ID
    Formation     FormationType
    Name          string
    Morale        int
    SquadLevel    int
    TurnCount     int
    MaxUnits      int
    UsedCapacity  float64
    TotalCapacity int
}
```

**❌ Bad Example:**
```go
// DON'T DO THIS - logic in components
type Squad struct {
    Name string
    Units []ecs.EntityID
}

// ❌ BAD - methods on components
func (s *Squad) AddUnit(unitID ecs.EntityID) {
    s.Units = append(s.Units, unitID)
}
```

**Why:** Separating data from logic makes code testable, maintainable, and follows pure ECS architecture.

---

### 2. EntityID-Based Relationships (Never Pointers)

**Rule:** Always use `ecs.EntityID` for relationships between entities. NEVER store entity pointers or references.

**✅ Good Example:**
```go
// From gear/items.go
type Item struct {
    Properties ecs.EntityID    // ✅ Use EntityID
    Actions    []ItemAction
    Count      int
}

// From squads/components.go
type SquadMemberData struct {
    SquadID ecs.EntityID // ✅ Parent squad's entity ID
}
```

**❌ Bad Example:**
```go
// DON'T DO THIS
type Item struct {
    Properties *ecs.Entity  // ❌ Entity pointer
    Owner      *ecs.Entity  // ❌ Entity pointer
    Count      int
}

type SquadMember struct {
    Squad *Squad  // ❌ Direct struct pointer
}
```

**Why:** Entity pointers become invalid when entities are destroyed or moved in memory. EntityIDs remain valid and queries are efficient.

---

### 3. Query-Based Discovery (Don't Store References)

**Rule:** Discover entity relationships through ECS queries, not stored collections.

**✅ Good Example:**
```go
// From squads/squadqueries.go
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

**❌ Bad Example:**
```go
// DON'T DO THIS
type Squad struct {
    Members []*ecs.Entity  // ❌ Storing entity references
}

func (s *Squad) GetMembers() []*ecs.Entity {
    return s.Members
}
```

**Why:** Query-based discovery ensures you always work with current, valid entities and prevents stale references.

---

### 4. System-Based Logic (Functions, Not Methods)

**Rule:** All behavior lives in system functions that operate on components, never as component methods.

**✅ Good Example:**
```go
// From gear/Inventory.go - System functions
func AddItem(manager *ecs.Manager, inv *Inventory, itemEntityID ecs.EntityID) {
    // System logic here
}

func RemoveItem(manager *ecs.Manager, inv *Inventory, index int) {
    // System logic here
}

func GetItemEntityID(inv *Inventory, index int) (ecs.EntityID, error) {
    // System logic here
}
```

**❌ Bad Example:**
```go
// DON'T DO THIS
type Inventory struct {
    Items []ecs.EntityID
}

// ❌ Methods on components
func (inv *Inventory) AddItem(item ecs.EntityID) {
    inv.Items = append(inv.Items, item)
}

func (inv *Inventory) RemoveItem(index int) {
    inv.Items = append(inv.Items[:index], inv.Items[index+1:]...)
}
```

**Why:** System functions can be tested independently, reused across different components, and follow functional programming principles.

---

### 5. Value-Based Map Keys (Performance)

**Rule:** When using maps for lookups, use value types as keys, not pointers.

**✅ Good Example:**
```go
// From position system (50x performance improvement)
type PositionSystem struct {
    positionMap map[coords.LogicalPosition]ecs.EntityID  // ✅ Value key
}

func (ps *PositionSystem) AddEntity(entity *ecs.Entity, pos coords.LogicalPosition) {
    ps.positionMap[pos] = entity.GetID()  // ✅ O(1) lookup
}
```

**❌ Bad Example:**
```go
// DON'T DO THIS
type PositionTracker struct {
    positions map[*coords.LogicalPosition]ecs.EntityID  // ❌ Pointer key
}
```

**Why:** Value-based keys enable proper hashing and O(1) lookups. Pointer keys often fail equality checks even for identical values.

---

## Component Design Patterns

### Component Structure

Every component should follow this pattern:

```go
// 1. Component documentation
// Describe what the component represents and its role in the system

// 2. Data struct (pure data only)
type ComponentNameData struct {
    Field1 Type1  // Purpose of field1
    Field2 Type2  // Purpose of field2
}

// 3. Simple helper methods (optional, only if needed)
func (c *ComponentNameData) String() string {
    // String representation for debugging
}

// 4. Global component variable (in init section)
var ComponentNameComponent *ecs.Component
```

### Example: Squad Grid Position Component

```go
// From squads/components.go (lines 88-122)

// GridPositionData represents a unit's position within the 3x3 grid.
// Pure data - systems query for units at specific positions
// Supports multi-cell units (e.g., 1x2, 2x2, 2x1, etc.)
type GridPositionData struct {
    AnchorRow int // Top-left row (0-2)
    AnchorCol int // Top-left col (0-2)
    Width     int // Number of columns occupied (1-3)
    Height    int // Number of rows occupied (1-3)
}

// ✅ Simple helper methods are OK if they don't modify state or call systems
func (g *GridPositionData) GetOccupiedCells() [][2]int {
    var cells [][2]int
    for r := g.AnchorRow; r < g.AnchorRow+g.Height && r < 3; r++ {
        for c := g.AnchorCol; c < g.AnchorCol+g.Width && c < 3; c++ {
            cells = append(cells, [2]int{r, c})
        }
    }
    return cells
}

func (g *GridPositionData) OccupiesCell(row, col int) bool {
    return row >= g.AnchorRow && row < g.AnchorRow+g.Height &&
           col >= g.AnchorCol && col < g.AnchorCol+g.Width
}
```

### Component Naming Conventions

- **Component Type:** `ComponentNameData` (suffix with "Data")
- **Global Variable:** `ComponentNameComponent`
- **Tags:** `ComponentNameTag` (if needed for queries)

Example:
```go
var (
    SquadComponent        *ecs.Component  // For SquadData
    SquadMemberComponent  *ecs.Component  // For SquadMemberData
    GridPositionComponent *ecs.Component  // For GridPositionData

    SquadTag       ecs.Tag  // Query all squads
    SquadMemberTag ecs.Tag  // Query all squad members
)
```

---

## System Function Patterns

### Function Naming Conventions

System functions should use consistent naming patterns:

| Pattern | Purpose | Example |
|---------|---------|---------|
| `GetX` | Retrieve single value/entity | `GetSquadEntity()`, `GetLeaderID()` |
| `FindX` | Search for entity by criteria | `FindUnitByID()`, `FindItemEntityByID()` |
| `CheckX` / `IsX` | Boolean queries | `IsSquadDestroyed()`, `CanAddUnit()` |
| `GetXsInY` | Retrieve collections | `GetUnitIDsInSquad()`, `GetUnitIDsInRow()` |
| `ExecuteX` | Perform action | `ExecuteSquadAttack()` |
| `UpdateX` | Modify state | `UpdateSquadCapacity()` |
| `CalculateX` | Compute values | `CalculateTotalCover()` |

### Function Signatures

All system functions should follow this pattern:

```go
// Pattern 1: Query function (read-only)
func GetSomething(queryParam Type, manager *common.EntityManager) ReturnType {
    // Use ECS queries to find entities
    // Return computed result
}

// Pattern 2: Mutation function (modifies state)
func UpdateSomething(entityID ecs.EntityID, newValue Type, manager *common.EntityManager) {
    // Find entity
    // Get component
    // Modify component data
}

// Pattern 3: System action
func ExecuteSomething(param1 Type1, param2 Type2, manager *common.EntityManager) *ResultType {
    // Complex system logic
    // May involve multiple entities
    // Returns result struct
}
```

### Example: Query Function

```go
// From squads/squadqueries.go (lines 50-66)

// GetUnitIDsInSquad returns unit IDs belonging to a squad
// ✅ Returns ecs.EntityID (native type), not entity pointers
func GetUnitIDsInSquad(squadID ecs.EntityID, squadmanager *common.EntityManager) []ecs.EntityID {
    var unitIDs []ecs.EntityID

    // Query all entities with SquadMemberTag
    for _, result := range squadmanager.World.Query(SquadMemberTag) {
        unitEntity := result.Entity
        memberData := common.GetComponentType[*SquadMemberData](unitEntity, SquadMemberComponent)

        // Filter by squadID relationship
        if memberData.SquadID == squadID {
            unitID := unitEntity.GetID() // ✅ Native method!
            unitIDs = append(unitIDs, unitID)
        }
    }

    return unitIDs
}
```

### Example: System Action

```go
// From squads/squadcombat.go (lines 19-113)

// ExecuteSquadAttack performs row-based combat between two squads
// ✅ Works with ecs.EntityID internally
func ExecuteSquadAttack(attackerSquadID, defenderSquadID ecs.EntityID, squadmanager *common.EntityManager) *CombatResult {
    result := &CombatResult{
        DamageByUnit: make(map[ecs.EntityID]int),
        UnitsKilled:  []ecs.EntityID{},
    }

    // Query attacker units
    attackerUnitIDs := GetUnitIDsInSquad(attackerSquadID, squadmanager)

    // Process each attacker
    for _, attackerID := range attackerUnitIDs {
        attackerUnit := FindUnitByID(attackerID, squadmanager)
        if attackerUnit == nil {
            continue
        }

        // Get targeting data
        targetRowData := common.GetComponentType[*TargetRowData](attackerUnit, TargetRowComponent)

        // Find targets and apply damage
        // ... (complex system logic)
    }

    return result
}
```

### System Organization

Organize system functions into logical files:

- `componentname_queries.go` - Query functions (read-only)
- `componentname_actions.go` - Action functions (mutations)
- `componentname_combat.go` - Combat-specific systems
- `componentname_creation.go` - Entity/component creation helpers

Example from squads system:
```
squads/
├── components.go       # Pure data components
├── squadqueries.go     # Query functions
├── squadcombat.go      # Combat system
├── squadcreation.go    # Creation helpers
├── squadabilities.go   # Ability system
└── visualization.go    # Display/debug functions
```

---

## Entity Management

### Creating Entities with Components

Always use the builder pattern when creating entities:

```go
// From gear/items.go (lines 99-134)

func CreateItem(manager *ecs.Manager, name string, pos coords.LogicalPosition, imagePath string, effects ...StatusEffects) *ecs.Entity {

    img, _, err := ebitenutil.NewImageFromFile(imagePath)
    if err != nil {
        log.Fatal(err)
    }

    // Create properties entity to hold status effects
    propsEntity := manager.NewEntity()
    for _, prop := range effects {
        propsEntity.AddComponent(prop.StatusEffectComponent(), &prop)
    }

    // Create item component with EntityID reference (ECS best practice)
    item := &Item{
        Count:      1,
        Properties: propsEntity.GetID(), // ✅ Use EntityID instead of pointer
        Actions:    make([]ItemAction, 0),
    }

    // Build entity with chained AddComponent calls
    itemEntity := manager.NewEntity().
        AddComponent(rendering.RenderableComponent, &rendering.Renderable{
            Image:   img,
            Visible: true,
        }).
        AddComponent(common.PositionComponent, &coords.LogicalPosition{
            X: pos.X,
            Y: pos.Y,
        }).
        AddComponent(common.NameComponent, &common.Name{
            NameStr: name,
        }).
        AddComponent(ItemComponent, item)

    return itemEntity
}
```

### Adding Components to Existing Entities

```go
// Get entity
entity := FindEntityByID(entityID, manager)
if entity == nil {
    return
}

// Add new component
componentData := &NewComponentData{
    Field1: value1,
    Field2: value2,
}
entity.AddComponent(NewComponent, componentData)
```

### Removing Components

```go
// Check if component exists
if entity.HasComponent(SomeComponent) {
    entity.RemoveComponent(SomeComponent)
}
```

### Entity Lifecycle Management

```go
// Check if entity is valid before operations
func ProcessEntity(entityID ecs.EntityID, manager *common.EntityManager) {
    entity := FindEntityByID(entityID, manager)
    if entity == nil {
        // Entity doesn't exist, handle gracefully
        return
    }

    // Verify required components
    if !entity.HasComponent(RequiredComponent) {
        return
    }

    // Safe to proceed
    data := common.GetComponentType[*ComponentData](entity, RequiredComponent)
    // ... process data
}
```

---

## Query Patterns

### Basic Query: Find All Entities with Tag

```go
// From gear/gearutil.go (lines 12-23)

func FindItemEntityByID(manager *ecs.Manager, entityID ecs.EntityID) *ecs.Entity {
    // Build tag on-the-fly for Item entities
    itemTag := ecs.BuildTag(ItemComponent)
    for _, result := range manager.Query(itemTag) {
        if result.Entity.GetID() == entityID {
            return result.Entity
        }
    }
    return nil
}
```

### Multi-Component Query

```go
// Query entities that have multiple components
func FindEntitiesWithComponents(manager *ecs.Manager) []*ecs.Entity {
    var entities []*ecs.Entity

    // Build tag from multiple components
    tag := ecs.BuildTag(Component1, Component2, Component3)

    for _, result := range manager.Query(tag) {
        entities = append(entities, result.Entity)
    }

    return entities
}
```

### Filtered Query

```go
// From squads/squadqueries.go (lines 83-107)

func GetUnitIDsInRow(squadID ecs.EntityID, row int, squadmanager *common.EntityManager) []ecs.EntityID {
    var unitIDs []ecs.EntityID
    seen := make(map[ecs.EntityID]bool) // Prevents duplicates

    for col := 0; col < 3; col++ {
        idsAtPos := GetUnitIDsAtGridPosition(squadID, row, col, squadmanager)
        for _, unitID := range idsAtPos {
            if !seen[unitID] {
                unitEntity := FindUnitByID(unitID, squadmanager)
                if unitEntity == nil {
                    continue
                }

                // Filter by attribute check
                attr := common.GetAttributes(unitEntity)
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

### Query with Relationship Filtering

```go
// From squads/squadqueries.go (lines 22-48)

func GetUnitIDsAtGridPosition(squadID ecs.EntityID, row, col int, squadmanager *common.EntityManager) []ecs.EntityID {
    var unitIDs []ecs.EntityID

    for _, result := range squadmanager.World.Query(SquadMemberTag) {
        unitEntity := result.Entity

        // Filter by relationship
        memberData := common.GetComponentType[*SquadMemberData](unitEntity, SquadMemberComponent)
        if memberData.SquadID != squadID {
            continue
        }

        // Filter by component existence
        if !unitEntity.HasComponent(GridPositionComponent) {
            continue
        }

        gridPos := common.GetComponentType[*GridPositionData](unitEntity, GridPositionComponent)

        // Filter by position
        if gridPos.OccupiesCell(row, col) {
            unitID := unitEntity.GetID()
            unitIDs = append(unitIDs, unitID)
        }
    }

    return unitIDs
}
```

### Performance: Caching Query Results

For frequently accessed data, cache query results and invalidate on changes:

```go
type SystemCache struct {
    lastUpdate time.Time
    results    []ecs.EntityID
    isDirty    bool
}

func GetCachedUnits(cache *SystemCache, manager *common.EntityManager) []ecs.EntityID {
    if cache.isDirty {
        // Rebuild cache
        cache.results = QueryUnits(manager)
        cache.isDirty = false
        cache.lastUpdate = time.Now()
    }
    return cache.results
}

func InvalidateCache(cache *SystemCache) {
    cache.isDirty = true
}
```

---

## Common Anti-Patterns

### ❌ Anti-Pattern 1: Logic in Components

**Bad:**
```go
type Unit struct {
    Health int
    Damage int
}

func (u *Unit) Attack(target *Unit) {
    target.Health -= u.Damage
}
```

**Good:**
```go
// Component - pure data
type CombatStats struct {
    Health int
    Damage int
}

// System function
func ExecuteAttack(attackerID, defenderID ecs.EntityID, manager *ecs.Manager) {
    attacker := FindUnitByID(attackerID, manager)
    defender := FindUnitByID(defenderID, manager)

    attackerStats := common.GetComponentType[*CombatStats](attacker, CombatStatsComponent)
    defenderStats := common.GetComponentType[*CombatStats](defender, CombatStatsComponent)

    defenderStats.Health -= attackerStats.Damage
}
```

---

### ❌ Anti-Pattern 2: Storing Entity Pointers

**Bad:**
```go
type Inventory struct {
    Items []*ecs.Entity  // ❌ Pointers become invalid
}
```

**Good:**
```go
type Inventory struct {
    ItemEntityIDs []ecs.EntityID  // ✅ IDs remain valid
}

// Query when needed
func GetItems(inv *Inventory, manager *ecs.Manager) []*ecs.Entity {
    items := make([]*ecs.Entity, 0)
    for _, id := range inv.ItemEntityIDs {
        if item := FindItemByID(id, manager); item != nil {
            items = append(items, item)
        }
    }
    return items
}
```

---

### ❌ Anti-Pattern 3: Pointer Map Keys

**Bad:**
```go
type Tracker struct {
    positions map[*Position]ecs.EntityID  // ❌ Poor performance
}
```

**Good:**
```go
type Tracker struct {
    positions map[Position]ecs.EntityID  // ✅ Value keys, O(1) lookup
}
```

---

### ❌ Anti-Pattern 4: Storing Collections Instead of Querying

**Bad:**
```go
type Squad struct {
    Members []*Unit  // ❌ Manual bookkeeping required
}

func (s *Squad) AddMember(unit *Unit) {
    s.Members = append(s.Members, unit)
}
```

**Good:**
```go
type SquadData struct {
    SquadID ecs.EntityID
}

type SquadMemberData struct {
    SquadID ecs.EntityID  // Relationship
}

// Query when needed
func GetSquadMembers(squadID ecs.EntityID, manager *ecs.Manager) []ecs.EntityID {
    // Query-based discovery
    var members []ecs.EntityID
    for _, result := range manager.Query(SquadMemberTag) {
        memberData := common.GetComponentType[*SquadMemberData](result.Entity, SquadMemberComponent)
        if memberData.SquadID == squadID {
            members = append(members, result.Entity.GetID())
        }
    }
    return members
}
```

---

### ❌ Anti-Pattern 5: Methods on Component Structs

**Bad:**
```go
type Inventory struct {
    Items []ecs.EntityID
}

func (inv *Inventory) AddItem(item ecs.EntityID) {
    inv.Items = append(inv.Items, item)
}

func (inv *Inventory) RemoveItem(index int) {
    inv.Items = append(inv.Items[:index], inv.Items[index+1:]...)
}
```

**Good:**
```go
type Inventory struct {
    ItemEntityIDs []ecs.EntityID  // Pure data
}

// System functions
func AddItem(manager *ecs.Manager, inv *Inventory, itemEntityID ecs.EntityID) {
    // System logic
    inv.ItemEntityIDs = append(inv.ItemEntityIDs, itemEntityID)
}

func RemoveItem(manager *ecs.Manager, inv *Inventory, index int) {
    // System logic
    inv.ItemEntityIDs = append(inv.ItemEntityIDs[:index], inv.ItemEntityIDs[index+1:]...)
}
```

---

## Performance Best Practices

### 1. Use O(1) Spatial Lookups

For position-based queries, use a spatial map instead of linear search:

**Before (O(n)):**
```go
func GetEntityAtPosition(pos Position, manager *ecs.Manager) *ecs.Entity {
    for _, result := range manager.Query(allEntities) {
        entityPos := GetPosition(result.Entity)
        if entityPos.Equals(pos) {
            return result.Entity
        }
    }
    return nil
}
```

**After (O(1)):**
```go
type PositionSystem struct {
    positionMap map[coords.LogicalPosition]ecs.EntityID
}

func (ps *PositionSystem) GetEntityIDAt(pos coords.LogicalPosition) ecs.EntityID {
    return ps.positionMap[pos]  // O(1) lookup
}
```

### 2. Avoid Redundant Queries

Cache frequently accessed data within a system execution:

**Bad:**
```go
func ProcessUnits(squadID ecs.EntityID, manager *ecs.Manager) {
    for i := 0; i < 100; i++ {
        units := GetUnitIDsInSquad(squadID, manager)  // ❌ Queries 100 times
        // Process units
    }
}
```

**Good:**
```go
func ProcessUnits(squadID ecs.EntityID, manager *ecs.Manager) {
    units := GetUnitIDsInSquad(squadID, manager)  // ✅ Query once
    for i := 0; i < 100; i++ {
        // Process cached units
    }
}
```

### 3. Use Tags for Efficient Filtering

Pre-build tags for common queries:

```go
var (
    AliveUnitsTag  ecs.Tag
    DeadUnitsTag   ecs.Tag
    PlayerTag      ecs.Tag
    EnemyTag       ecs.Tag
)

func Init() {
    AliveUnitsTag = ecs.BuildTag(UnitComponent, AliveComponent)
    DeadUnitsTag = ecs.BuildTag(UnitComponent, DeadComponent)
}

func GetAliveUnits(manager *ecs.Manager) []*ecs.Entity {
    var units []*ecs.Entity
    for _, result := range manager.Query(AliveUnitsTag) {
        units = append(units, result.Entity)
    }
    return units
}
```

### 4. Batch Operations

Perform operations in batches to reduce query overhead:

```go
func ApplyDamageToMultipleUnits(damageMap map[ecs.EntityID]int, manager *ecs.Manager) {
    for unitID, damage := range damageMap {
        unit := FindUnitByID(unitID, manager)
        if unit == nil {
            continue
        }

        stats := common.GetComponentType[*CombatStats](unit, CombatStatsComponent)
        stats.Health -= damage
    }
}
```

---

## Reference Implementations

### Perfect ECS: Squad System (2358 LOC)

**Files:**
- `squads/components.go` - 8 pure data components, zero logic
- `squads/squadqueries.go` - 7 query functions, all return EntityIDs
- `squads/squadcombat.go` - Combat system with hit/dodge/crit mechanics
- `squads/visualization.go` - Display system (read-only)

**Key Features:**
- ✅ All components use `ecs.EntityID` for relationships
- ✅ Query-based discovery (no stored collections)
- ✅ System functions for all logic
- ✅ Value-based keys for performance

**Code Snippets:**

```go
// components.go (lines 40-50) - Pure data
type SquadData struct {
    SquadID       ecs.EntityID
    Formation     FormationType
    Name          string
    Morale        int
    SquadLevel    int
    TurnCount     int
    MaxUnits      int
    UsedCapacity  float64
    TotalCapacity int
}

// squadqueries.go (lines 50-66) - Query function
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

// squadcombat.go (lines 19-113) - System action
func ExecuteSquadAttack(attackerSquadID, defenderSquadID ecs.EntityID, squadmanager *common.EntityManager) *CombatResult {
    result := &CombatResult{
        DamageByUnit: make(map[ecs.EntityID]int),
        UnitsKilled:  []ecs.EntityID{},
    }

    attackerUnitIDs := GetUnitIDsInSquad(attackerSquadID, squadmanager)

    for _, attackerID := range attackerUnitIDs {
        // ... combat logic
    }

    return result
}
```

---

### Perfect ECS: Inventory System (241 LOC)

**File:** `gear/Inventory.go`

**Key Features:**
- ✅ Pure data component (Inventory struct)
- ✅ 9 system functions, zero component methods
- ✅ EntityID-based item references

**Code Snippets:**

```go
// Inventory.go (lines 18-22) - Pure data component
type Inventory struct {
    ItemEntityIDs []ecs.EntityID  // ✅ Use EntityID, not pointers
}

// Inventory.go (lines 27-66) - System function
func AddItem(manager *ecs.Manager, inv *Inventory, itemEntityID ecs.EntityID) {
    itemEntity := FindItemEntityByID(manager, itemEntityID)
    if itemEntity == nil {
        return
    }

    newItemName := common.GetComponentType[*common.Name](itemEntity, common.NameComponent).NameStr
    exists := false

    for _, existingID := range inv.ItemEntityIDs {
        existingEntity := FindItemEntityByID(manager, existingID)
        if existingEntity == nil {
            continue
        }

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

---

### Perfect ECS: Item System (177 LOC)

**File:** `gear/items.go`

**Key Features:**
- ✅ Item.Properties uses `ecs.EntityID` not pointer
- ✅ CreateItem functions follow builder pattern

**Code Snippets:**

```go
// items.go (lines 40-46) - Pure data with EntityID relationship
type Item struct {
    Properties ecs.EntityID    // ✅ Status effects entity ID
    Actions    []ItemAction
    Count      int
}

// items.go (lines 99-134) - Creation function
func CreateItem(manager *ecs.Manager, name string, pos coords.LogicalPosition, imagePath string, effects ...StatusEffects) *ecs.Entity {

    // Create properties entity
    propsEntity := manager.NewEntity()
    for _, prop := range effects {
        propsEntity.AddComponent(prop.StatusEffectComponent(), &prop)
    }

    // Create item with EntityID reference
    item := &Item{
        Count:      1,
        Properties: propsEntity.GetID(),  // ✅ Use EntityID
        Actions:    make([]ItemAction, 0),
    }

    // Builder pattern
    itemEntity := manager.NewEntity().
        AddComponent(rendering.RenderableComponent, &rendering.Renderable{...}).
        AddComponent(common.PositionComponent, &coords.LogicalPosition{...}).
        AddComponent(common.NameComponent, &common.Name{...}).
        AddComponent(ItemComponent, item)

    return itemEntity
}
```

---

### Perfect ECS: Utility Functions (115 LOC)

**File:** `gear/gearutil.go`

**Key Features:**
- ✅ Query-based entity lookup
- ✅ System functions for item operations

**Code Snippets:**

```go
// gearutil.go (lines 12-23) - Query-based lookup
func FindItemEntityByID(manager *ecs.Manager, entityID ecs.EntityID) *ecs.Entity {
    itemTag := ecs.BuildTag(ItemComponent)
    for _, result := range manager.Query(itemTag) {
        if result.Entity.GetID() == entityID {
            return result.Entity
        }
    }
    return nil
}

// gearutil.go (lines 25-33) - Component accessor
func GetItemByID(manager *ecs.Manager, entityID ecs.EntityID) *Item {
    entity := FindItemEntityByID(manager, entityID)
    if entity == nil {
        return nil
    }
    return common.GetComponentType[*Item](entity, ItemComponent)
}
```

---

## Step-by-Step Workflows

### Workflow 1: Adding a New Component

**Step 1:** Define the component data structure

```go
// In appropriate file (e.g., combat/components.go)
package combat

// ShieldData represents a unit's shield that absorbs damage before health
type ShieldData struct {
    CurrentShield int  // Current shield points
    MaxShield     int  // Maximum shield capacity
    RegenRate     int  // Shield regeneration per turn
}
```

**Step 2:** Declare global component variable

```go
var (
    ShieldComponent *ecs.Component
)
```

**Step 3:** Initialize component in package init or setup

```go
func InitComponents(manager *ecs.Manager) {
    ShieldComponent = manager.NewComponent()
}
```

**Step 4:** Add to entity creation functions

```go
func CreateUnitWithShield(manager *ecs.Manager, shieldAmount int) *ecs.Entity {
    shield := &ShieldData{
        CurrentShield: shieldAmount,
        MaxShield:     shieldAmount,
        RegenRate:     5,
    }

    entity := manager.NewEntity().
        AddComponent(common.AttributeComponent, &common.Attributes{...}).
        AddComponent(ShieldComponent, shield)  // Add new component

    return entity
}
```

**Step 5:** Create system functions

```go
// combat/shield_system.go

// ApplyDamageWithShield applies damage to shield first, then health
func ApplyDamageWithShield(unitID ecs.EntityID, damage int, manager *ecs.Manager) {
    unit := FindUnitByID(unitID, manager)
    if unit == nil {
        return
    }

    // Check if unit has shield
    if unit.HasComponent(ShieldComponent) {
        shield := common.GetComponentType[*ShieldData](unit, ShieldComponent)

        if shield.CurrentShield > 0 {
            // Shield absorbs damage
            shield.CurrentShield -= damage

            if shield.CurrentShield < 0 {
                // Overflow damage to health
                attr := common.GetAttributes(unit)
                attr.CurrentHealth += shield.CurrentShield  // Add negative value
                shield.CurrentShield = 0
            }
        } else {
            // No shield, damage health directly
            attr := common.GetAttributes(unit)
            attr.CurrentHealth -= damage
        }
    } else {
        // No shield component, damage health directly
        attr := common.GetAttributes(unit)
        attr.CurrentHealth -= damage
    }
}

// RegenerateShields restores shield points at the start of turn
func RegenerateShields(manager *ecs.Manager) {
    tag := ecs.BuildTag(ShieldComponent)

    for _, result := range manager.Query(tag) {
        shield := common.GetComponentType[*ShieldData](result.Entity, ShieldComponent)

        shield.CurrentShield += shield.RegenRate
        if shield.CurrentShield > shield.MaxShield {
            shield.CurrentShield = shield.MaxShield
        }
    }
}
```

**Step 6:** Integrate into existing systems

```go
// Update combat system to use new shield logic
func ExecuteAttack(attackerID, defenderID ecs.EntityID, manager *ecs.Manager) {
    damage := CalculateDamage(attackerID, defenderID, manager)

    // Use new shield-aware damage function
    ApplyDamageWithShield(defenderID, damage, manager)
}
```

---

### Workflow 2: Writing Query Functions

**Step 1:** Identify query requirements

Example: "Find all units with shields below 50%"

**Step 2:** Write query function signature

```go
func GetUnitsWithLowShields(threshold float64, manager *ecs.Manager) []ecs.EntityID
```

**Step 3:** Build appropriate tag

```go
func GetUnitsWithLowShields(threshold float64, manager *ecs.Manager) []ecs.EntityID {
    var lowShieldUnits []ecs.EntityID

    // Query entities with both Shield and Attribute components
    tag := ecs.BuildTag(ShieldComponent, common.AttributeComponent)
```

**Step 4:** Iterate and filter

```go
    for _, result := range manager.Query(tag) {
        shield := common.GetComponentType[*ShieldData](result.Entity, ShieldComponent)

        // Calculate shield percentage
        shieldPercent := float64(shield.CurrentShield) / float64(shield.MaxShield)

        // Filter by threshold
        if shieldPercent < threshold {
            lowShieldUnits = append(lowShieldUnits, result.Entity.GetID())
        }
    }

    return lowShieldUnits
}
```

**Step 5:** Document and test

```go
// GetUnitsWithLowShields returns entity IDs of units whose shield is below the threshold percentage
// threshold: 0.0-1.0 (e.g., 0.5 = 50%)
// Returns empty slice if no units match
func GetUnitsWithLowShields(threshold float64, manager *ecs.Manager) []ecs.EntityID {
    // ... implementation
}
```

---

### Workflow 3: Creating System Functions

**Step 1:** Determine function type (query, mutation, or action)

Example: "Restore shields to all units in a squad"

**Step 2:** Choose appropriate naming convention

```go
// UpdateX for mutations
func RestoreSquadShields(squadID ecs.EntityID, amount int, manager *ecs.Manager)
```

**Step 3:** Write function signature with manager parameter

```go
func RestoreSquadShields(squadID ecs.EntityID, amount int, manager *ecs.Manager) {
```

**Step 4:** Query for target entities

```go
    // Get all units in squad
    unitIDs := squads.GetUnitIDsInSquad(squadID, manager)
```

**Step 5:** Iterate and apply logic

```go
    for _, unitID := range unitIDs {
        unit := squads.FindUnitByID(unitID, manager)
        if unit == nil {
            continue
        }

        // Check if unit has shield component
        if !unit.HasComponent(ShieldComponent) {
            continue
        }

        // Apply shield restoration
        shield := common.GetComponentType[*ShieldData](unit, ShieldComponent)
        shield.CurrentShield += amount

        // Cap at maximum
        if shield.CurrentShield > shield.MaxShield {
            shield.CurrentShield = shield.MaxShield
        }
    }
}
```

**Step 6:** Add error handling and edge cases

```go
func RestoreSquadShields(squadID ecs.EntityID, amount int, manager *ecs.Manager) {
    // Validate amount
    if amount <= 0 {
        return
    }

    // Get squad entity to verify it exists
    squadEntity := squads.GetSquadEntity(squadID, manager)
    if squadEntity == nil {
        return
    }

    // Get all units in squad
    unitIDs := squads.GetUnitIDsInSquad(squadID, manager)

    for _, unitID := range unitIDs {
        // ... restoration logic
    }
}
```

---

### Workflow 4: Integrating with Existing Systems

**Step 1:** Identify integration points

Example: "Add shield regeneration to turn system"

**Step 2:** Find relevant system files

```
game_main/turn_system.go
```

**Step 3:** Add system call at appropriate phase

```go
func ProcessTurnStart(manager *ecs.Manager) {
    // Existing logic
    UpdateAbilityCooldowns(manager)
    ApplyStatusEffects(manager)

    // Add new shield regeneration
    combat.RegenerateShields(manager)  // ✅ New system integration
}
```

**Step 4:** Verify component dependencies

```go
// Ensure ShieldComponent is initialized before use
func InitGame() {
    manager := ecs.NewManager()

    // Initialize all components
    common.InitComponents(manager)
    combat.InitComponents(manager)  // ✅ Must initialize before use
    squads.InitComponents(manager)

    // ... rest of initialization
}
```

**Step 5:** Test integration

```go
func TestShieldRegeneration(t *testing.T) {
    manager := ecs.NewManager()
    combat.InitComponents(manager)

    // Create unit with damaged shield
    unit := combat.CreateUnitWithShield(manager, 100)
    shield := common.GetComponentType[*combat.ShieldData](unit, combat.ShieldComponent)
    shield.CurrentShield = 50  // Damaged

    // Process turn
    combat.RegenerateShields(manager)

    // Verify regeneration
    if shield.CurrentShield != 55 {  // 50 + 5 regen
        t.Errorf("Expected 55 shield, got %d", shield.CurrentShield)
    }
}
```

---

## Quick Reference Checklist

### Component Design ✅
- [ ] Pure data only (no logic methods)
- [ ] Use `ecs.EntityID` for relationships
- [ ] Name ends with "Data" (e.g., `ShieldData`)
- [ ] Global variable ends with "Component"
- [ ] Simple helpers OK (String, Copy, getters)

### System Functions ✅
- [ ] Named with Get/Find/Check/Update/Execute pattern
- [ ] Manager parameter included
- [ ] Returns EntityIDs, not entity pointers
- [ ] Uses query-based discovery
- [ ] Handles nil/missing entities gracefully

### Entity Management ✅
- [ ] Builder pattern for entity creation
- [ ] Verify component existence before access
- [ ] Use `entity.GetID()` for EntityID
- [ ] Never store entity pointers

### Performance ✅
- [ ] Value-based map keys
- [ ] Cache query results when appropriate
- [ ] Use tags for filtering
- [ ] Batch operations when possible
- [ ] O(1) spatial lookups for position queries

### Query Patterns ✅
- [ ] Build tags from components
- [ ] Filter results in loop
- [ ] Return EntityIDs, not pointers
- [ ] Handle empty results
- [ ] Use deduplication (map) if needed

---

## Common Mistakes and Solutions

| Mistake | Solution |
|---------|----------|
| Storing `*ecs.Entity` in component | Use `ecs.EntityID` instead |
| Methods on component structs | Move logic to system functions |
| Pointer map keys | Use value types as keys |
| Linear search for positions | Use spatial map with value keys |
| Storing collections in components | Use query-based discovery |
| Missing nil checks | Always verify entity exists before access |
| Querying inside loops | Cache query results before loop |
| Not checking component existence | Use `entity.HasComponent()` before access |

---

## Additional Resources

### Key Files to Reference

1. **Squad System (Perfect ECS)**
   - `squads/components.go` - Component design patterns
   - `squads/squadqueries.go` - Query function patterns
   - `squads/squadcombat.go` - System action patterns

2. **Inventory System (Perfect ECS)**
   - `gear/Inventory.go` - System function patterns
   - `gear/items.go` - EntityID relationships
   - `gear/gearutil.go` - Query utilities

3. **Common Utilities**
   - `common/commoncomponents.go` - Attributes component
   - `common/ecsutil.go` - Type-safe component access

### Documentation

- Project documentation: `CLAUDE.md`
- ECS best practices: `CLAUDE.md` lines 72-107
- Squad system: `analysis/MASTER_ROADMAP.md`

---

## When to Use This Skill

Invoke this skill when:

1. **Adding new components** - Follow pure data patterns
2. **Writing system functions** - Use correct naming and signatures
3. **Querying entities** - Use efficient query patterns
4. **Refactoring to ECS** - Convert old code to ECS best practices
5. **Performance optimization** - Apply O(1) lookup patterns
6. **Code review** - Verify ECS compliance

---

## Summary: The ECS Golden Rules

1. **Components = Pure Data** - No logic, only fields
2. **EntityID = Relationships** - Never use pointers
3. **Queries = Discovery** - Don't store collections
4. **Systems = Logic** - Functions, not methods
5. **Values = Map Keys** - Performance matters

Follow these rules for clean, maintainable, performant ECS code.
