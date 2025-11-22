# ECS Architecture Skill

**Purpose**: Quick ECS compliance checks and pattern suggestions for Go game code
**Trigger**: When working with component definitions, system functions, or entity relationships

## Capabilities

- Validate components are pure data (no methods)
- Check EntityID vs entity pointer usage
- Verify query-based relationships
- Suggest system function patterns
- Flag pointer map keys (performance)

## Quick ECS Check Pattern

When analyzing code for ECS compliance, check these key principles:

### 1. Pure Data Components
```go
// ✅ GOOD: Pure data, no methods
type Inventory struct {
    ItemEntityIDs []ecs.EntityID
}

// ❌ BAD: Component with logic
type Inventory struct {
    Items []*Item
}
func (inv *Inventory) AddItem(item *Item) { ... }  // ❌ No!
```

### 2. EntityID Usage
```go
// ✅ GOOD: Use EntityID
type Item struct {
    Properties ecs.EntityID
}

// ❌ BAD: Entity pointers
type Item struct {
    Properties *ecs.Entity  // ❌ No!
}
```

### 3. Query-Based Relationships
```go
// ✅ GOOD: Query on demand
func GetSquadMembers(manager *ecs.Manager, squadID ecs.EntityID) []*ecs.Entity {
    // Query for members with matching SquadID
    for _, entity := range manager.FilterByTag(SquadMemberTag) {
        memberData := GetSquadMemberData(entity)
        if memberData != nil && memberData.SquadID == squadID {
            members = append(members, entity)
        }
    }
    return members
}

// ❌ BAD: Stored references
type Squad struct {
    Members []*ecs.Entity  // ❌ Cached references
}
```

### 4. System-Based Logic
```go
// ✅ GOOD: System function
func AddItem(manager *ecs.Manager, inv *Inventory, itemID ecs.EntityID) {
    inv.ItemEntityIDs = append(inv.ItemEntityIDs, itemID)
}

// ❌ BAD: Component method
func (inv *Inventory) AddItem(itemID ecs.EntityID) { ... }  // ❌ No!
```

### 5. Value Map Keys
```go
// ✅ GOOD: Value keys (50x faster)
grid := make(map[coords.LogicalPosition]ecs.EntityID)
entity := grid[coords.LogicalPosition{X: 5, Y: 10}]  // O(1) lookup

// ❌ BAD: Pointer keys (50x slower)
grid := make(map[*coords.LogicalPosition]ecs.EntityID)
entity := grid[&coords.LogicalPosition{X: 5, Y: 10}]  // Won't find match!
```

## Reference Implementations

- `squads/*.go` - Perfect ECS: 2675 LOC, 8 pure data components, system-based combat
- `gear/Inventory.go` - Perfect ECS: 241 LOC, pure data component, 9 system functions
- `gear/items.go` - EntityID-based relationships (177 LOC)

## Usage Tips

When reviewing code for ECS violations:
1. Search for `func (.*\*.*) ` to find component methods
2. Search for `\*ecs.Entity` to find entity pointers
3. Search for `map[\*` to find pointer map keys
4. Check if components store references vs querying on demand
5. Verify all game logic is in system functions, not component methods

## Priority Levels

- **CRITICAL**: Pointer map keys (50x performance impact), entity pointers (crash risk)
- **HIGH**: Component methods (maintainability), missing EntityID usage
- **MEDIUM**: Inconsistent patterns, suboptimal queries
- **LOW**: Style preferences, documentation gaps
