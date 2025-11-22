# TinkerRogue Refactoring Checklist

**Last Updated**: 2025-11-21
**Total Issues**: 45 items across 8 categories
**Critical ECS Violations**: 8 instances
**Status**: Ready for systematic cleanup

---

## Priority Levels

- 🔴 **CRITICAL**: ECS violations, breaks architecture principles
- 🟡 **HIGH**: Affects functionality, blocks features
- 🟢 **MEDIUM**: Technical debt, code quality
- 🔵 **LOW**: Nice-to-have, polish

---

## 🔴 CRITICAL: ECS Anti-patterns (Must Fix)

### 1. Component Methods → System Functions

#### `gear/items.go` (Lines 49-97)
**Issue**: Item component has 7 logic methods, violates "pure data" principle

**Current (WRONG)**:
```go
func (item *Item) GetAction() *gear.ItemAction { ... }
func (item *Item) HasAction() bool { ... }
func (item *Item) GetActions() []gear.ItemAction { ... }
func (item *Item) GetThrowableAction() *gear.ThrowableAction { ... }
func (item *Item) HasThrowableAction() bool { ... }
func (item *Item) GetFirstActionOfType[T any]() *T { ... }
```

**Fix**: Move to system functions in `gear/gearutil.go` (following `Inventory.go` pattern)
```go
func GetItemAction(item *Item) *ItemAction { ... }
func HasItemAction(item *Item) bool { ... }
func GetItemActions(item *Item) []ItemAction { ... }
// etc.
```

**Files to change**:
- `gear/items.go` - Remove methods
- `gear/gearutil.go` - Add system functions
- All callers - Update to use system functions

**Estimate**: 2-3 hours

---

#### `gear/stateffect.go` (Lines 126-363)
**Issue**: Status effect components (`Sticky`, `Burning`, `Freezing`) have logic methods

**Current (WRONG)**:
```go
func (s *Sticky) ApplyToCreature(creature *ecs.Entity) { ... }  // Line 186
func (s *Sticky) StackEffect(other StatusEffect) { ... }        // Line 161
// Similar for Burning (lines 252, 268) and Freezing (lines 317, 333)
```

**Fix**: Extract to status effect system
```go
// New file: gear/statuseffectsystem.go
func ApplyStatusEffect(effect StatusEffect, target ecs.EntityID) { ... }
func StackStatusEffect(existing, new StatusEffect) StatusEffect { ... }
```

**Additional Work**: Extract quality interface (mentioned in CLAUDE.md as "85% complete")

**Files to change**:
- `gear/stateffect.go` - Remove ApplyToCreature, StackEffect methods
- Create `gear/statuseffectsystem.go` - Add system functions
- All callers - Update to use system functions

**Estimate**: 4-6 hours (includes interface extraction)

---

#### `common/playerdata.go` (Lines 40-62)
**Issue**: PlayerThrowable and PlayerData have logic methods

**Current (WRONG)**:
```go
func (p *PlayerThrowable) GetThrowableItemIndex() int { ... }  // Line 40
func (p *PlayerData) PlayerAttributes() *Attributes { ... }     // Line 54
```

**Fix**: Move to system functions in `common/ecsutil.go`
```go
func GetPlayerThrowableIndex(player *PlayerThrowable) int { ... }
func GetPlayerAttributes(player *PlayerData) *Attributes { ... }
```

**Files to change**:
- `common/playerdata.go` - Remove methods
- `common/ecsutil.go` - Add system functions
- All callers - Update to use system functions

**Estimate**: 1-2 hours

---

## 🟡 HIGH: Functionality & Feature Blockers

### 2. Throwable AOE Implementation

**Location**: `gear/itemactions.go` lines 68-88

**Issue**: Throwable effects commented out, not working with squad system
```go
//TODO, apply this to squads in the future
// Lines 68-88: Commented-out AOE application logic
```

**Fix**: Implement throwable effects for squad-based combat
- Integrate with squad damage system
- Apply AOE effects to squad formations
- Handle status effect application to squad members

**Dependencies**: Status effect system refactoring (Critical #1)

**Estimate**: 3-4 hours

---

### 3. Victory Condition Implementation

**Location**: `combat/victory.go` lines 37-49

**Issue**: `EliminationVictory.CheckVictory()` is a stub, combat never ends
```go
func (e EliminationVictory) CheckVictory(ecs *ecs.ECS) (int, bool) {
    //TODO: Implement actual faction squad counting logic
    return 0, false  // Always returns false!
}
```

**Fix**: Implement victory checking
- Count alive squads per faction
- Return winning faction when only one remains
- Implement `ObjectiveVictory` and `TurnLimitVictory` (lines 62-71)

**Files to change**:
- `combat/victory.go` - Implement all 3 victory condition types

**Estimate**: 2-3 hours

---

### 4. Entity Cleanup on Death

**Issue**: No automatic entity cleanup when units/squads die (noted in CLAUDE.md)

**Fix**: Implement entity lifecycle management
- Create death event system
- Remove entities from spatial grid
- Clean up component references
- Remove from squad rosters
- Clean up inventory items

**Files to create/modify**:
- Create `combat/lifecycle.go` or `common/lifecycle.go`
- Integrate with combat damage system
- Integrate with squad combat

**Estimate**: 3-4 hours

---

### 5. Wall Collision Detection

**Location**: `worldmap/dungeongen.go` line 318

**Issue**: TODO comment indicates incorrect collision check
```go
// TODO: Change this to check for WALL, not blocked
```

**Fix**: Update collision logic to check tile type properly

**Files to change**:
- `worldmap/dungeongen.go` - Line 318
- Verify collision checks in movement systems

**Estimate**: 1 hour

---

## 🟢 MEDIUM: Global State & Code Quality

### 6. Global Variable Dependency Injection

**Issue**: Multiple packages use global mutable state (15+ instances)

#### Instances Found:

**`coords/cordmanager.go:8`**
```go
var CoordManager *CoordinateManager  // Global instance
```

**`common/ecsutil.go:13-26`**
```go
var GlobalPositionSystem *systems.PositionSystem  // Global system
```

**`graphics/vx.go:15`**
```go
var VXHandler VisualEffectHandler  // Global handler
```

**`graphics/graphictypes.go:8-19`**
```go
var GreenColorMatrix = ColorMatrix{...}
var RedColorMatrix = ColorMatrix{...}
var ScreenInfo = coords.NewScreenData()
var CoordManager = coords.CoordManager
var ViewableSquareSize = 30
var MAP_SCROLLING_ENABLED = true
var StatsUIOffset int = 1000
```

**`spawning/loottables.go:12-21`**
```go
var ConsumableSpawnProb = 30
var ThrowableSpawnProb = 30
var RangedWeaponSpawnProb = 10
var RandThrowableOptions = []gear.StatusEffects{...}
var LootQualityTable = NewProbabilityTable[common.QualityType]()
var ThrowableEffectStatTable = NewProbabilityTable[gear.StatusEffects]()
var ThrowableAOEProbTable = NewProbabilityTable[graphics.BasicShapeType]()
```

**`squads/squadmanager.go:12`**
```go
var Units = make([]UnitTemplate, 0, len(entitytemplates.MonsterTemplates))
```

**Fix Strategy**:
1. Pass as struct fields to systems that need them
2. Use dependency injection in constructors
3. Create context objects for related globals
4. Constants should use `const` instead of `var`

**Estimate**: 8-12 hours (affects many files)

---

### 7. Large File Splitting

**Files exceeding 400 LOC** (excluding test files):

#### `graphics/vx.go` - 892 LOC 🔴
**Issue**: Single file handles all visual effects

**Fix**: Split into:
- `graphics/vx_core.go` - Core handler and types
- `graphics/vx_projectile.go` - Projectile effects
- `graphics/vx_impact.go` - Impact/explosion effects
- `graphics/vx_status.go` - Status effect visuals
- `graphics/vx_animation.go` - Animation helpers

**Estimate**: 3-4 hours

---

#### `gui/guicomponents/guicomponents.go` - 571 LOC
**Issue**: All GUI components in one file

**Fix**: Split into component types:
- `gui/guicomponents/panel.go`
- `gui/guicomponents/text.go`
- `gui/guicomponents/list.go`
- `gui/guicomponents/container.go`

**Estimate**: 2-3 hours

---

#### `gui/guisquads/squadbuilder.go` - 423 LOC
**Issue**: Squad UI builder needs organization

**Fix**: Extract UI factory functions:
- `gui/guisquads/squadbuilder.go` - Core builder
- `gui/guisquads/squadbuilder_panels.go` - Panel creation
- `gui/guisquads/squadbuilder_lists.go` - List widgets

**Estimate**: 2-3 hours

---

#### `gui/widgets/createwidgets.go` - 393 LOC
**Issue**: Large widget factory file

**Fix**: Already uses factory pattern, organize by widget type:
- `gui/widgets/button_factory.go`
- `gui/widgets/list_factory.go`
- `gui/widgets/panel_factory.go`

**Estimate**: 2 hours

---

### 8. Debug Print Cleanup

**Location**: `gui/guisquads/squaddeploymentmode.go` lines 170-274

**Issue**: Multiple debug printf statements throughout code

**Fix**:
- Remove debug prints or gate behind debug flag
- Use proper logging system if needed

**Estimate**: 30 minutes

---

## 🔵 LOW: Code Polish & Optional Improvements

### 9. GridPositionData Helper Methods

**Location**: `squads/components.go` lines 98-122

**Issue**: Helper methods on component (currently acceptable, but inconsistent)
```go
func (g *GridPositionData) GetOccupiedCells() []coords.LogicalPosition { ... }
func (g *GridPositionData) OccupiesCell(pos coords.LogicalPosition) bool { ... }
func (g *GridPositionData) GetRows() int { ... }
```

**Fix**: Move to query functions in `squads/squadqueries.go` for consistency

**Estimate**: 1 hour

---

### 10. Attributes Derived Stats

**Location**: `common/commoncomponents.go` lines 74-185

**Issue**: 12 derived stat methods on Attributes component

**Status**: Currently acceptable (pure calculations from data)

**Optional Fix**: Extract to `AttributesSystem` for perfect ECS consistency

**Estimate**: 2-3 hours

---

### 11. Tile SetColorMatrix Method

**Location**: `worldmap/dungeontile.go` line 65

**Issue**: `Tile` struct has `SetColorMatrix()` method

**Status**: Not a component (regular struct), but violates data/logic separation

**Fix**: Make ColorMatrix a direct field, remove setter method

**Estimate**: 30 minutes

---

## 📋 High-Priority TODO Comments

**Issues flagged in code with TODO comments:**

1. `combat/queries.go:250` - "TODO: Implement event system for UI"
2. `combat/victory.go:39` - "TODO: Implement actual faction squad counting logic" (see HIGH #3)
3. `combat/turnmanager.go:66` - "TODO: Do we really need to create a new system?"
4. `gear/itemactions.go:68` - "TODO, apply this to squads in the future" (see HIGH #2)
5. `squads/squadabilities.go:163` - "TODO: Track buff duration (requires turn/buff system)"
6. `worldmap/dungeongen.go:52` - "TODO: This is a temporary solution for spawning logic"
7. `worldmap/dungeongen.go:134` - "Todo need to add"
8. `worldmap/dungeongen.go:318` - "TODO: Change this to check for WALL, not blocked" (see HIGH #5)
9. `coords/cordmanager.go:41` - "TODO: This should only be calculated once instead of being called for every coordinate conversion"

---

## ✅ FALSE POSITIVES (Update CLAUDE.md)

### Formation Presets - ALREADY COMPLETE!

**CLAUDE.md Status**: "⚠️ Formation Presets (4-6h) - Balanced/Defensive/Offensive/Ranged templates (stubs exist)"

**Actual Status**: ✅ **FULLY IMPLEMENTED**

**Location**: `squads/squadcreation.go` lines 152-215

**Evidence**:
- `GetFormationPreset()` fully implemented for all 4 types
- FormationBalanced (lines 167-176)
- FormationDefensive (lines 178-187)
- FormationOffensive (lines 189-198)
- FormationRanged (lines 200-210)
- All include Position, Role, and Target data

**Action**: Update CLAUDE.md to mark Squad System as 100% complete!

---

### Equipment System EntityID Migration - ALREADY COMPLETE!

**CLAUDE.md Status**: "⚠️ Equipment system still uses entity pointers (scheduled for refactoring)"

**Actual Status**: ✅ **ALREADY REFACTORED**

**Evidence**:
- No separate equipment files exist (merged into `gear` package)
- `gear/items.go`: Item.Properties uses `ecs.EntityID` (line 43)
- `gear/Inventory.go`: Uses `[]ecs.EntityID` (line 25)
- All gear package already uses proper EntityIDs

**Action**: Remove this note from CLAUDE.md anti-patterns section!

---

## 📊 Summary Statistics

### By Priority
- 🔴 **CRITICAL**: 3 items (12-17 hours)
- 🟡 **HIGH**: 5 items (12-16 hours)
- 🟢 **MEDIUM**: 5 items (17-25 hours)
- 🔵 **LOW**: 3 items (4-7 hours)

### Total Estimated Work
- **Critical + High**: 24-33 hours
- **All Items**: 45-72 hours

### Largest Impact Items
1. Global variable dependency injection (8-12h) - Architecture improvement
2. Status effect system refactoring (4-6h) - Completes ECS migration
3. Large file splitting (9-12h) - Code organization
4. Item component methods extraction (2-3h) - ECS compliance

---

## 🎯 Recommended Implementation Order

### Phase 1: ECS Critical Path (2-3 days)
1. Item component methods → system functions
2. PlayerData methods → system functions
3. Status effect system refactoring

### Phase 2: Feature Completion (2-3 days)
4. Throwable AOE implementation (depends on Phase 1)
5. Victory condition implementation
6. Entity cleanup on death
7. Wall collision fix

### Phase 3: Code Quality (3-4 days)
8. Global variable dependency injection
9. Large file splitting (VX, GUI components)
10. Debug print cleanup

### Phase 4: Polish (1-2 days)
11. GridPositionData helpers → queries
12. Optional: Attributes system extraction
13. TODO comment cleanup

---

## 📝 Notes

- Squad system is actually **100% complete** (not 95% as CLAUDE.md states)
- Equipment system refactoring **already done** (not pending)
- Most critical ECS violations are in `gear` package
- Global state cleanup will touch many files but improve testability significantly
- Test files (1100+ LOC) are acceptable and don't need splitting

**Next Step**: Begin Phase 1 with item component methods extraction
