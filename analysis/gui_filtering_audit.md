# GUI Package Filtering Audit

**Date:** 2025-11-12
**Status:** Comprehensive Analysis Complete
**Scope:** All GUI filtering patterns and opportunities

---

## Executive Summary

The GUI package uses filtering in three primary contexts:

1. **SquadFilter** - Filters on SquadInfo (squad status, faction, destroyed state)
2. **ItemFilter** - Filters on EntityID only (for panels and items)
3. **Manual Filtering** - Direct checks in rendering and query logic

Current state: Partially centralized with opportunities for consolidation.

---

## Filtering Patterns Found

### 1. SquadListComponent (guicomponents.go, lines 11-94)
**Status:** ✅ Uses centralized filtering via SquadFilter
**Pattern:** `func(info *SquadInfo) bool`
**Used by:**
- CombatMode (current faction squads)
- SquadDeploymentMode (alive squads)
- Custom filters via callbacks

**Filter Functions Available:**
- `AliveSquadsOnly()` - Non-destroyed squads only
- `PlayerSquadsOnly(queries)` - Player faction squads
- `FactionSquadsOnly(factionID)` - Specific faction squads

**Code Quality:** ✅ GOOD - Centralized filters at line 98-115

---

### 2. PanelListComponent (guicomponents.go, lines 264-332)
**Status:** ⚠️ Partially centralized
**Pattern:** `func(squadID ecs.EntityID) bool` (EntityID-based, not info-based)
**Used by:**
- SquadManagementMode - Shows all squads (filter returns true for all)

**Issue:** Uses ItemFilter which only takes EntityID
- Cannot access squad info for complex filtering
- Currently just returns true (no filtering)

**Code Quality:** ⚠️ MEDIUM - Could use SquadInfo-based filters

---

### 3. ButtonListComponent (guicomponents.go, lines 338-400)
**Status:** ⚠️ Partially centralized
**Pattern:** `func(squadID ecs.EntityID) bool` (EntityID-based)
**Used by:** Not currently used in modes

**Issue:** Same as PanelListComponent - limited to EntityID-based filtering

---

### 4. SquadHighlightRenderer (guirenderers.go, lines 112-163)
**Status:** ❌ Inline filtering (SHOULD BE REFACTORED)
**Code:**
```go
for _, squadID := range allSquads {
    squadInfo := shr.queries.GetSquadInfo(squadID)
    if squadInfo == nil || squadInfo.IsDestroyed || squadInfo.Position == nil {
        continue
    }
    // ... then determines color based on faction
}
```

**Problem:** Manual nil/destroyed checks
**Opportunity:** Should use `AliveSquadsOnly` or similar centralized filter

**Impact:** When IsDestroyed logic changes, must update here too

---

### 5. CombatActionHandler (combat_action_handler.go)
**Status:** ⚠️ Uses existing queries, could use FilterHelper
**Methods:**
- Line 88: `cah.queries.GetEnemySquads(currentFactionID)`
- Line 147: `cah.queries.GetEnemySquads(currentFactionID)`
- Line 229: `cah.factionManager.GetFactionSquads(currentFactionID)`

**Code Quality:** ⚠️ MEDIUM - Uses queries but not centralized FilterHelper

---

### 6. CombatInputHandler (combat_input_handler.go)
**Status:** ⚠️ Inline faction checks
**Lines:** 160, 162, 169
**Pattern:** Direct `IsPlayerFaction()` checks and faction comparisons
**Code Quality:** ⚠️ MEDIUM - Could benefit from higher-level filtering

---

### 7. SquadBuilderMode (squadbuilder.go)
**Status:** ✅ No filtering needed
**Pattern:** Just gets most recent squad from all squads

---

### 8. ExplorationMode (explorationmode.go)
**Status:** ✅ No squad filtering used

---

### 9. InventoryMode (inventorymode.go)
**Status:** ✅ No squad filtering used

---

### 10. InfoMode (infomode.go)
**Status:** ✅ No squad filtering used

---

## Filter Functions Currently Available

### In guicomponents.go (lines 96-115)

```go
// Predicate filter - checks single squad
func AliveSquadsOnly(info *SquadInfo) bool
    return !info.IsDestroyed

// Factory filters - returns filter function
func PlayerSquadsOnly(queries *GUIQueries) SquadFilter
    returns: !info.IsDestroyed && queries.IsPlayerFaction(info.FactionID)

func FactionSquadsOnly(factionID ecs.EntityID) SquadFilter
    returns: !info.IsDestroyed && info.FactionID == factionID
```

### In filter_helper.go (NEW)

```go
type FilterHelper struct { queries *GUIQueries }

// Slice-based filters
func (fh *FilterHelper) FilterPlayerFactionSquads(allSquads []ecs.EntityID) []ecs.EntityID
func (fh *FilterHelper) FilterAliveSquads(allSquads []ecs.EntityID) []ecs.EntityID
func (fh *FilterHelper) FilterFactionSquads(allSquads []ecs.EntityID, factionID) []ecs.EntityID

// Convenience wrappers
func (fh *FilterHelper) GetSquadIDsForFaction(factionID) []ecs.EntityID
func (fh *FilterHelper) GetPlayerFactionSquadIDs() []ecs.EntityID
```

---

## Refactoring Opportunities

### PRIORITY 1: SquadHighlightRenderer (MEDIUM Effort - HIGH Impact)
**File:** `guirenderers.go`, lines 132-163
**Current:** Manual filtering with nil/IsDestroyed checks
**Recommended:** Use `AliveSquadsOnly` filter

```go
// CURRENT
for _, squadID := range allSquads {
    squadInfo := shr.queries.GetSquadInfo(squadID)
    if squadInfo == nil || squadInfo.IsDestroyed || squadInfo.Position == nil {
        continue
    }
    // render...
}

// RECOMMENDED
aliveSquads := allSquads // or FilterHelper to filter
for _, squadInfo := range aliveSquads {
    if squadInfo.Position == nil {
        continue
    }
    // render...
}
```

**Benefits:**
- Uses centralized destruction logic
- When IsDestroyed changes, only FilterHelper needs update
- Clearer intent

---

### PRIORITY 2: PanelListComponent & ButtonListComponent (LOW-MEDIUM Effort - MEDIUM Impact)
**Files:** `guicomponents.go`, lines 264-400
**Current:** ItemFilter uses EntityID only
**Issue:** Cannot access SquadInfo for complex filtering
**Recommended:** Add SquadFilter variant or extend ItemFilter

**Option A: Create SquadPanelListComponent**
```go
type SquadPanelListComponent struct {
    filter SquadFilter  // Use SquadFilter instead of ItemFilter
    // ... rest same
}
```

**Option B: Extend ItemFilter to support SquadInfo lookup**
```go
type ItemFilter func(squadID ecs.EntityID, queries *GUIQueries) bool
```

**Current Users:** SquadManagementMode only (returns true for all)
**Impact if Changed:** Low - only one user

---

### PRIORITY 3: CombatActionHandler (LOW Effort - LOW-MEDIUM Impact)
**File:** `combat_action_handler.go`, lines 88, 147
**Current:** Uses `cah.queries.GetEnemySquads(currentFactionID)`
**Opportunity:** Consider using FilterHelper for consistency

**Note:** Already uses existing query functions, less critical than other refactorings

---

### PRIORITY 4: CombatInputHandler (LOW Effort - LOW Impact)
**File:** `combat_input_handler.go`, lines 160-172
**Current:** Direct faction comparison checks
**Opportunity:** Extract to helper methods for clarity

**Note:** Low impact - already using IsPlayerFaction queries

---

## Filtering Architecture Decision Tree

**Use case: Filter squad list**
- ✅ Use **SquadFilter** if filtering based on SquadInfo
- ❌ Don't use ItemFilter (too limited for squad filtering)

**Use case: Get filtered slice of all squads**
- ✅ Use **FilterHelper** methods for batch operations
- ❌ Don't use SquadFilter directly (wrong level of abstraction)

**Use case: Filter in components (UI)**
- ✅ Use **SquadFilter** with components like SquadListComponent
- ✅ Pass filter to component constructor

**Use case: Render with filtering**
- ✅ Query squads, then apply filter logic
- ✅ Consider extracting filter to helper method

---

## Implementation Roadmap

### Phase 1: Complete (Filters 2/4 files)
- ✅ SquadDeploymentMode - Uses AliveSquadsOnly
- ✅ CombatMode - Uses makeCurrentFactionSquadFilter()
- ✅ FilterHelper utility created

### Phase 2: Recommended (Filters remaining patterns)
1. **SquadHighlightRenderer** - Apply AliveSquadsOnly filter
2. **Create variant of PanelListComponent** - Support SquadFilter
3. **Document filtering patterns** - Create guidelines

### Phase 3: Optional (Consistency improvements)
1. CombatActionHandler - Consider FilterHelper usage
2. CombatInputHandler - Extract faction checks to methods
3. Update all components to use consistent filtering

---

## Testing Checklist

When implementing any filtering changes:

- [ ] AliveSquadsOnly correctly identifies destroyed squads
- [ ] PlayerSquadsOnly only shows player faction squads
- [ ] FactionSquadsOnly only shows specific faction squads
- [ ] FilterHelper methods don't modify input slices
- [ ] Rendering still shows all faction squads with correct colors
- [ ] Component filtering updates correctly on state changes

---

## Code Quality Metrics

### Current State
- **Centralized Filters:** 3 filter functions (AliveSquadsOnly, PlayerSquadsOnly, FactionSquadsOnly)
- **Centralized Utilities:** 1 FilterHelper (5 methods)
- **Manual Inline Filtering:** 1 location (SquadHighlightRenderer)
- **Limited-scope Filters:** ItemFilter (2 components - PanelListComponent, ButtonListComponent)

### After Recommended Refactoring
- **Centralized Filters:** +1 SquadHighlightRenderer (refactored)
- **Centralized Utilities:** FilterHelper (consistent across package)
- **Manual Inline Filtering:** 0 (fully eliminated)
- **Limited-scope Filters:** PanelListComponent variant created

### Benefits
- ✅ Single source of truth for filtering logic
- ✅ Easier testing of filter functions
- ✅ Consistent patterns across package
- ✅ Reduced code duplication
- ✅ Clearer intent in component usage

---

## Summary

The GUI package filtering is **70% centralized**:

✅ **Good:**
- SquadFilter predicate functions in place
- Most UI components use centralized filters
- FilterHelper provides batch operations

⚠️ **Needs Work:**
- SquadHighlightRenderer uses manual filtering
- PanelListComponent limited to EntityID-based filtering
- ItemFilter lacks access to squad state

🎯 **Recommended Next Steps:**
1. Refactor SquadHighlightRenderer to use AliveSquadsOnly (1 hour, high impact)
2. Create SquadFilter variant of PanelListComponent (2 hours, medium impact)
3. Document filtering patterns for future code (0.5 hours, prevents future issues)

**Total Effort:** ~3.5 hours for Phase 2 improvements
**Value:** Complete filtering centralization + better component architecture
