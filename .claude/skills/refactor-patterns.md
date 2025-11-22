# Refactoring Pattern Skill

**Purpose**: Quick refactoring pattern suggestions for common simplification tasks
**Trigger**: When discussing code duplication, complexity, or technical debt

## Capabilities

- Identify extract function opportunities
- Suggest consolidation patterns
- Recommend separation of concerns
- Flag over-engineering
- Propose interface extractions

## Core Refactoring Patterns

### 1. Consolidation Pattern (Type Explosion → Unified Type)

**When to Use**: Multiple similar types with duplicated logic

**Example from TinkerRogue**: Graphics shapes consolidation (8+ types → 3 variants)

```go
// ❌ BEFORE: Type explosion (8+ shape types, code duplication)
type Rectangle struct {
    X, Y, Width, Height int
    Color color.Color
    BorderColor color.Color
}

type FilledRectangle struct {
    X, Y, Width, Height int
    Color color.Color
}

type Circle struct {
    X, Y, Radius int
    Color color.Color
    BorderColor color.Color
}
// ... 5+ more similar types

// ✅ AFTER: Consolidation (1 base type, 3 variants)
type BaseShape struct {
    Position coords.PixelPosition
    Color    color.Color
    Variant  ShapeVariant  // Rectangle, Circle, Line
}

type ShapeVariant int
const (
    RectangleVariant ShapeVariant = iota
    CircleVariant
    LineVariant
)
```

**Benefits**: 390 LOC (reduced from 800+ LOC), single draw function, easier maintenance

**Reference**: `graphics/drawableshapes.go` consolidation (completed 2025-10-08)

---

### 2. Extract Function Pattern (Code Duplication → Shared Logic)

**When to Use**: Duplicated logic across multiple functions or files

```go
// ❌ BEFORE: Duplicated coordinate calculation
func ProcessTileA(x, y int) {
    idx := y*dungeonWidth + x  // Duplicated calculation
    // ... logic A
}

func ProcessTileB(x, y int) {
    idx := y*dungeonWidth + x  // Duplicated calculation
    // ... logic B
}

// ✅ AFTER: Extract common logic
func (cm *CoordinateManager) LogicalToIndex(pos LogicalPosition) int {
    return pos.Y*cm.dungeonWidth + pos.X  // Single source of truth
}

func ProcessTileA(x, y int) {
    idx := coords.CoordManager.LogicalToIndex(coords.LogicalPosition{X: x, Y: y})
    // ... logic A
}
```

**Benefits**: Single source of truth, easier to test, prevents calculation errors

**Reference**: `coords/coordinatemanager.go` (O(1) spatial grid pattern)

---

### 3. Separation of Concerns (God Object → Specialized Components)

**When to Use**: Single type/function doing too many things

```go
// ❌ BEFORE: God object (mixed concerns)
type GameMap struct {
    Tiles            []*Tile
    TileImageSet     map[string]*ebiten.Image  // Graphics concern
    generator        MapGenerator              // Generation concern

    // ... 20+ fields mixing rendering, generation, logic
}

// ✅ AFTER: Separated concerns
type GameMap struct {
    Tiles  []*Tile
    Width  int
    Height int
    // Only map state data
}

// Graphics in separate struct
type TileImageSet struct {
    Floor  *ebiten.Image
    Wall   *ebiten.Image
    // ... rendering data only
}

// Generation in strategy pattern
type MapGenerator interface {
    Generate(width, height int) GenerationResult
}
```

**Benefits**: Each type has single responsibility, easier testing, clear boundaries

**Reference**: `worldmap/` package refactoring (Strategy Pattern, 2025-11-08)

---

### 4. Strategy Pattern (Hardcoded Algorithms → Pluggable Strategies)

**When to Use**: Multiple algorithms for the same task, algorithm selection needed

```go
// ❌ BEFORE: Hardcoded algorithm with flags
func GenerateMap(width, height int, useBSP bool, useCellular bool) {
    if useBSP {
        // BSP algorithm inline
    } else if useCellular {
        // Cellular automata inline
    } else {
        // Rooms and corridors inline
    }
}

// ✅ AFTER: Strategy pattern
type MapGenerator interface {
    Name() string
    Description() string
    Generate(width, height int, config GenerationConfig) *GenerationResult
}

type RoomsCorridorsGenerator struct{}
func (g *RoomsCorridorsGenerator) Generate(...) *GenerationResult { ... }

type TacticalBiomeGenerator struct{}
func (g *TacticalBiomeGenerator) Generate(...) *GenerationResult { ... }

// Registry for dynamic selection
var generators = map[string]MapGenerator{
    "rooms_corridors": &RoomsCorridorsGenerator{},
    "tactical_biome":  &TacticalBiomeGenerator{},
}

func NewGameMap(generatorName string) *GameMap {
    gen := generators[generatorName]
    result := gen.Generate(width, height, config)
    // ...
}
```

**Benefits**: Open/Closed principle, add algorithms without modifying existing code

**Reference**: `worldmap/generator.go` and generators (2025-11-08)

---

### 5. Configuration Object Pattern (Parameter Explosion → Config Struct)

**When to Use**: Functions with 5+ parameters, especially with similar types

```go
// ❌ BEFORE: Parameter explosion
func CreateButton(text string, x, y, width, height int,
                 onClick func(), textColor color.Color,
                 bgColor color.Color, hoverColor color.Color,
                 disabled bool, tooltip string) *widget.Button { ... }

// ✅ AFTER: Configuration object
type ButtonConfig struct {
    Text        string
    Position    coords.PixelPosition
    Size        coords.PixelSize
    OnClick     func()
    Colors      ButtonColors
    Disabled    bool
    Tooltip     string
}

func CreateButtonWithConfig(config ButtonConfig) *widget.Button { ... }

// Usage with clear intent
button := CreateButtonWithConfig(ButtonConfig{
    Text:     "Deploy Squad",
    Position: coords.PixelPosition{X: 100, Y: 200},
    OnClick:  deployHandler,
    Colors:   defaultButtonColors,
})
```

**Benefits**: Named parameters (clarity), easier to extend, optional fields natural

**Reference**: `gui/widgets/buttons.go` ButtonConfig pattern (2025-11-07)

---

### 6. Interface Extraction Pattern (Concrete Type → Interface)

**When to Use**: Need flexibility, testing, or multiple implementations

```go
// ❌ BEFORE: Concrete dependency
type CombatSystem struct {
    damageCalc *DamageCalculator  // Concrete type
}

func (cs *CombatSystem) Attack(...) {
    damage := cs.damageCalc.Calculate(...)  // Tightly coupled
}

// ✅ AFTER: Interface dependency
type DamageCalculator interface {
    Calculate(attacker, defender *Unit) int
}

type CombatSystem struct {
    damageCalc DamageCalculator  // Interface
}

// Multiple implementations
type StandardDamageCalc struct{}
type CriticalDamageCalc struct{}
type MockDamageCalc struct{}  // For testing

func (cs *CombatSystem) Attack(...) {
    damage := cs.damageCalc.Calculate(...)  // Flexible, testable
}
```

**Benefits**: Testability (mock implementations), flexibility, dependency inversion

**Reference**: ECS system patterns throughout codebase

---

### 7. Entity Template Pattern (Factory → Data-Driven)

**When to Use**: Creating many similar objects with variations

```go
// ❌ BEFORE: Factory functions with hardcoded data
func CreateGoblin(manager *ecs.Manager) *ecs.Entity {
    entity := manager.NewEntity()
    entity.AddComponent(&StatsComponent{
        Health: 30,
        Damage: 5,
        Defense: 2,
    })
    // ... 10+ lines of hardcoded setup
    return entity
}

func CreateOrc(manager *ecs.Manager) *ecs.Entity {
    entity := manager.NewEntity()
    entity.AddComponent(&StatsComponent{
        Health: 50,  // Similar structure, different values
        Damage: 8,
        Defense: 4,
    })
    // ... 10+ lines of hardcoded setup
    return entity
}

// ✅ AFTER: Data-driven template
type EntityTemplate struct {
    Name   string
    Health int
    Damage int
    Defense int
    // ... all configuration data
}

// Templates in JSON
var templates = LoadFromJSON("monsterdata.json")

func CreateFromTemplate(manager *ecs.Manager, entityType EntityType) *ecs.Entity {
    template := templates[entityType]
    entity := manager.NewEntity()
    entity.AddComponent(&StatsComponent{
        Health:  template.Health,
        Damage:  template.Damage,
        Defense: template.Defense,
    })
    // ... apply template data
    return entity
}
```

**Benefits**: Data-driven, easy balancing, no code changes for new entities

**Reference**: `entitytemplates/` package (283 LOC, 2025-10-06)

---

## Anti-Patterns to Avoid

### 1. Premature Abstraction

```go
// ❌ DON'T: Abstract for 1-2 uses
func doThing(x int) int {
    return helper(x)  // Unnecessary wrapper
}
func helper(x int) int {
    return x * 2  // Only used once
}

// ✅ DO: Inline until 3+ uses (Rule of Three)
func doThing(x int) int {
    return x * 2  // Direct, clear
}
```

**Rule**: Don't abstract until you have 3+ similar uses (Rule of Three)

---

### 2. Over-Engineering

```go
// ❌ DON'T: Add complexity for "future flexibility"
type ConfigurableButtonFactoryBuilder struct {
    // 200 LOC of configuration options never used
}

// ✅ DO: Solve current problem simply
func CreateButton(text string, onClick func()) *widget.Button {
    // Simple solution that actually gets used
}
```

**Rule**: YAGNI (You Aren't Gonna Need It) - solve today's problem, not tomorrow's

---

### 3. Backwards-Compatibility Hacks

```go
// ❌ DON'T: Keep unused code "just in case"
type Component struct {
    NewField int
    _oldField int  // ❌ Unused, kept for "compatibility"
}

// ✅ DO: Delete unused code
type Component struct {
    NewField int
}
```

**Rule**: If it's unused, delete it. Version control keeps history.

---

## Refactoring Workflow

### Phase 1: Identify Pain Points
1. Code duplication (same logic in 3+ places)
2. Long functions (>50 lines, doing multiple things)
3. Parameter explosion (5+ parameters)
4. Type explosion (many similar types)
5. Hard to test (tight coupling, no interfaces)

### Phase 2: Choose Pattern
- **Duplication** → Extract Function or Consolidation
- **Complexity** → Separation of Concerns
- **Flexibility** → Strategy Pattern or Interface Extraction
- **Configuration** → Configuration Object
- **Data-Driven** → Entity Template Pattern

### Phase 3: Refactor Safely
1. Write tests FIRST (preserve behavior)
2. Make small changes (one pattern at a time)
3. Run tests after each change
4. Commit frequently (git checkpoints)
5. Document why (in analysis/ files)

### Phase 4: Validate Improvement
- ✅ Reduced LOC (simpler is better)
- ✅ Easier to test (less coupling)
- ✅ Easier to extend (Open/Closed)
- ✅ Easier to understand (clear responsibilities)

---

## TinkerRogue Refactoring Examples

### Completed Simplifications (Reference These!)

1. **Input System** (2025-09-15)
   - Pattern: Separation of Concerns
   - Before: Mixed keyboard/mouse handling
   - After: InputCoordinator + specialized controllers

2. **Coordinate System** (2025-09-20)
   - Pattern: Extract Function + Type Safety
   - Before: Manual calculations everywhere
   - After: CoordinateManager with LogicalPosition/PixelPosition types

3. **Entity Templates** (2025-10-06)
   - Pattern: Factory → Data-Driven
   - Before: 500+ LOC of factory functions
   - After: 283 LOC with JSON templates

4. **Graphics Shapes** (2025-10-08)
   - Pattern: Consolidation
   - Before: 8+ shape types, 800+ LOC
   - After: BaseShape + 3 variants, 390 LOC

5. **Position System** (2025-10-12)
   - Pattern: Extract Function + Value Keys
   - Before: O(n) searches, pointer map keys
   - After: O(1) spatial grid, value keys (50x faster!)

6. **Inventory System** (2025-10-21)
   - Pattern: ECS Refactor (Entity Pointers → EntityIDs)
   - Before: Component methods, entity pointers
   - After: Pure data component, system functions (533 LOC)

7. **GUI Button Factory** (2025-11-07)
   - Pattern: Configuration Object
   - Before: Parameter explosion (8+ parameters)
   - After: ButtonConfig consistently applied

8. **Worldmap Generator** (2025-11-08)
   - Pattern: Strategy Pattern
   - Before: Hardcoded algorithm in GameMap
   - After: MapGenerator interface + registry (2 generators, extensible)

**Total Simplification**: 8/8 completed items = 100%

**Key Lesson**: All simplifications followed a pattern from this skill

---

## Quick Reference: Pattern Selection

| Symptom | Pattern | Example |
|---------|---------|---------|
| Duplicated logic in 3+ places | Extract Function | CoordinateManager.LogicalToIndex() |
| 5+ similar types | Consolidation | BaseShape (8 types → 3 variants) |
| Single type doing too much | Separation of Concerns | GameMap refactor (rendering split out) |
| Hardcoded algorithm selection | Strategy Pattern | MapGenerator interface |
| 5+ function parameters | Configuration Object | ButtonConfig |
| Need testing flexibility | Interface Extraction | DamageCalculator interface |
| Many factory functions | Entity Template | monsterdata.json templates |

---

## Priority Levels

- **CRITICAL**: Code duplication causing bugs (fix immediately in multiple places)
- **HIGH**: Type explosion (maintenance burden), parameter explosion (error-prone)
- **MEDIUM**: Missing abstractions (3+ uses), tight coupling (hard to test)
- **LOW**: Premature optimization, style inconsistencies

---

## Usage Tips

1. **Always ask**: "Is this solving a real problem I have NOW?"
2. **Apply Rule of Three**: Don't abstract until 3+ similar uses
3. **Reference completed work**: Check CLAUDE.md simplification list for proven patterns
4. **Document decisions**: Create analysis/*.md files explaining WHY
5. **Test-driven**: Write tests before refactoring to preserve behavior
6. **Small steps**: One pattern at a time, commit frequently

---

**Remember**: The goal is **simplification**, not **complexity**. If a refactoring adds more code or concepts, question whether it's actually an improvement.
