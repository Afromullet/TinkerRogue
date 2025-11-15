# Code Duplication Summary

## Overview
Analysis of the entire TinkerRogue codebase identified **5 major duplication categories** affecting code maintainability. Total estimated savings: **300 LOC** through consolidation.



## 2. Faction/Squad/MapPosition Query Loop Duplications (High Priority - 120 LOC savings)

**Category:** Similar query patterns for specific entity relationships

### Identified Duplications

#### MapPosition Search Pattern (4+ occurrences)
- **gui/guiqueries.go**: lines 113-147, 140-161, 214-224, 233-244
- **combat/queries.go**: line 44

#### ActionState Search Pattern (2+ occurrences)
- **gui/guiqueries.go**: lines 154-162
- **combat/queries.go**: line 55

#### Faction Lookup Pattern (3+ occurrences)
- **gui/guiqueries.go**: lines 38, 72, 84
- **combat/queries.go**: line 24

### Pattern
```go
for _, result := range manager.World.Query(manager.Tags["mapposition"]) {
  mapPos := GetComponentType[*MapPositionData](result.Entity, MapPositionComponent)
  if mapPos.SquadID == squadID { /* process */ }
}
```

### Recommendation
**Extract to `combat/queries.go` as reusable public functions**:
- `FindMapPositionBySquadID(squadID, manager)`
- `FindActionStateBySquadID(squadID, manager)`
- `FindMapPositionByFactionID(factionID, manager)`

Replace all call sites in `gui/guiqueries.go` to use consolidated functions.

---

## 3. Text Widget Creation Duplication (Medium Priority - 60 LOC savings)

**Category:** Factory methods wrapping CreateTextWithConfig with fixed parameters

### Identified Duplications

**gui/combat_ui_factory.go**:
- `CreateTurnOrderLabel()` - Text + LargeFace + White
- `CreateFactionInfoText()` - Text + SmallFace + White
- `CreateSquadDetailText()` - Text + SmallFace + White

Similar pattern in `gui/squad_builder_ui_factory.go`.

### Pattern
```go
func (cuf *CombatUIFactory) CreateXxxText(text string) *widget.Text {
  return CreateTextWithConfig(TextConfig{
    Text: text, FontFace: SomeFont, Color: color.White,
  })
}
```

### Recommendation
**Delete wrapper methods** and replace with direct calls or single parameterized helper:
```go
func CreateLabelWithFont(text string, font font.Face) *widget.Text
```

---

## 4. Contains/Utility Helper Duplication (Low Priority - 25 LOC savings)

**Category:** Small helper functions reimplemented multiple times

### Identified Duplications

**combat/queries.go**:
- `contains()` line 237 - Check if LogicalPosition in slice
- `containsEntity()` line 247 - Check if EntityID in slice

**squads/squads_test.go**:
- `contains()` line 969 - String contains substring
- `containsHelper()` line 974 - Duplicate of above

### Recommendation
**Create `common/helpers.go`** with:
```go
func ContainsPosition(positions []LogicalPosition, pos LogicalPosition) bool
func ContainsEntityID(ids []ecs.EntityID, id ecs.EntityID) bool
```

Remove duplicates from combat/queries.go and test files.

---

## 5. Redundant Wrapper Functions (Very Low Priority - 15 LOC savings)

**Category:** Functions that just delegate to another identical function

### Identified Duplications

**coords/cordmanager.go**:
- `GetTilePositionsAsCommon()` (line 211) - Just calls `GetTilePositions()` and returns same type

### Recommendation
**Delete `GetTilePositionsAsCommon()`** entirely. Update 2 call sites:
- gear/itemactions.go line 71
- input/combatcontroller.go line 187

---

## Summary Table

| Category | LOC Saved | Difficulty | Priority | Files |
|----------|-----------|-----------|----------|-------|
| Entity Lookup Helpers | 80 | Low | High | 5 |
| MapPosition/ActionState/Faction Queries | 120 | Medium | High | 2 |
| Text Widget Factories | 60 | Low | Medium | 3 |
| Contains/Utility Helpers | 25 | Low | Low | 2 |
| Wrapper Functions | 15 | Very Low | Low | 1 |
| **TOTAL** | **300** | - | - | **12+** |

---

## Implementation Sequence

### Phase 1: High-Impact Entity Lookup (2 hours)
1. Add `FindEntityByIDWithTag()` to `common/ecsutil.go`
2. Replace all 8 similar functions across packages
3. Update imports; remove duplicates

**Impact:** -80 LOC, 5 files cleaned

### Phase 2: High-Impact Query Consolidation (2 hours)
1. Extract `FindMapPositionBySquadID()`, `FindActionStateBySquadID()` to `combat/queries.go`
2. Make `findFactionByID()` public
3. Update `gui/guiqueries.go` to use these functions

**Impact:** -120 LOC, 2 files heavily refactored

### Phase 3: Medium-Impact Factory Cleanup (1 hour)
1. Delete individual Create*Text() methods from factories
2. Replace with direct calls or single parameterized helper

**Impact:** -60 LOC, 3 files touched

### Phase 4: Low-Risk Utility Cleanup (30 min)
1. Move helpers to `common/helpers.go`
2. Delete `GetTilePositionsAsCommon()` from coords

**Impact:** -40 LOC, 4 files cleaned

---

## Notes
- All consolidations are low-risk: no breaking API changes, just internal refactoring
- Largest savings in gui/guiqueries.go: multiple similar query loops
- Post-refactoring: Consider QueryBuilder pattern to prevent future duplication
- Entity lookup functions are good candidates for a shared utility module
