# GUI Game Logic Separation Analysis

**Generated:** 2025-11-26
**Analyst:** Claude (refactoring-synth coordinator)

---

## EXECUTIVE SUMMARY

### Claim Verification: "100% Complete" - **REJECTED**

The completion plan's claim that service layer extraction is "100% COMPLETE" is **FALSE**. While significant progress has been made, substantial game logic remains in GUI code, particularly in:

1. **Unit Purchase Mode** - Direct ECS manipulation, business logic, and resource management
2. **Squad Builder Mode** - Roster management, direct query functions bypassing services
3. **Squad Management Mode** - Direct query function calls
4. **Combat Action Handler** - Direct query function usage instead of service methods
5. **GUI Queries** - Contains business logic that should be in services

### Critical Finding

**The "100% complete" assessment was premature.** While 4 services exist and some GUI files use them, the integration is **incomplete and inconsistent**. Game logic continues to leak through in multiple ways:

- Direct squad package query function calls (`GetUnitIDsInSquad`, `IsSquadDestroyed`, `VisualizeSquad`)
- Direct roster manipulation (`MarkUnitInSquad`, `MarkUnitAvailable`)
- Resource management logic in UI (`CanAfford`, `SpendGold`)
- Entity creation and disposal in UI code
- Business validation logic in UI

---

## PHASE 1: COMPLETE GUI FILE INVENTORY

### Total Files Found: 31

#### **Core Infrastructure (6 files)**
- `gui/basemode.go` - Base mode infrastructure
- `gui/modehelpers.go` - Mode helper utilities
- `gui/core/uimode.go` - UI mode interface
- `gui/core/modemanager.go` - Mode manager
- `gui/core/gamemodecoordinator.go` - Context switching coordinator
- `gui/core/contextstate.go` - Context state management

#### **Combat GUI (5 files)**
1. `gui/guicombat/combatmode.go` - ✅ Uses CombatService (verified)
2. `gui/guicombat/combat_action_handler.go` - ⚠️ Uses CombatService BUT also direct queries
3. `gui/guicombat/combat_input_handler.go` - ✅ Pure input handling
4. `gui/guicombat/combat_log_manager.go` - Not analyzed (likely pure UI)
5. `gui/guicombat/combat_ui_factory.go` - Not analyzed (likely pure UI)

#### **Squad GUI (7 files)**
1. `gui/guisquads/squadbuilder.go` - ❌ **MAJOR ISSUES** - Direct game logic
2. `gui/guisquads/squadmanagementmode.go` - ⚠️ Direct query function calls
3. `gui/guisquads/squaddeploymentmode.go` - ✅ Uses SquadDeploymentService (verified)
4. `gui/guisquads/formationeditormode.go` - ✅ Pure UI (placeholder implementation)
5. `gui/guisquads/unitpurchasemode.go` - ❌ **MAJOR ISSUES** - Direct game logic
6. `gui/guisquads/squad_builder_ui_factory.go` - Not analyzed (likely pure UI)
7. `gui/guisquads/squad_builder_grid_manager.go` - Not analyzed (likely pure UI)

#### **General Modes (3 files)**
1. `gui/guimodes/explorationmode.go` - ✅ Pure UI with coordinate conversion
2. `gui/guimodes/inventorymode.go` - Not analyzed
3. `gui/guimodes/infomode.go` - Not analyzed
4. `gui/guimodes/guirenderers.go` - Not analyzed (likely pure rendering)

#### **Components & Queries (2 files)**
1. `gui/guicomponents/guicomponents.go` - ✅ Pure UI components
2. `gui/guicomponents/guiqueries.go` - ⚠️ **Contains business logic**

#### **Widgets & Resources (8 files)**
All in `gui/widgets/` and `gui/guiresources/` - Pure UI utilities (not analyzed in detail)

---

## PHASE 2: VERIFICATION OF "100% COMPLETE" CLAIM

### File 1: gui/guicombat/combatmode.go

**Status:** ✅ **MOSTLY CLEAN** (Service Integration Successful)

**Positive Findings:**
- Uses `CombatService` exclusively for game logic (lines 26, 66)
- No direct ECS queries for game state
- Proper separation of UI updates from game logic

**Minor Issues:**
- Lines 154, 224, 231: Gets `TurnManager` directly via `combatService.GetTurnManager()` instead of using service methods
- This is an **anti-pattern** - services should expose operations, not internal managers

**Code Examples:**
```go
// ANTI-PATTERN (line 154)
round := cm.combatService.GetTurnManager().GetCurrentRound()

// BETTER APPROACH - Service should provide:
round := cm.combatService.GetCurrentRound() // Already exists!
```

**Recommendation:**
- Remove `GetTurnManager()` exposure from CombatService
- Use existing service methods (`GetCurrentRound()`, etc.)

---

### File 2: gui/guicombat/combat_action_handler.go

**Status:** ⚠️ **PARTIALLY CLEAN** (Mixed Service/Direct Query Usage)

**Issues Found:**

1. **Line 206: Direct squad package query function**
   ```go
   if !squads.IsSquadDestroyed(squadID, entityManager) {
   ```
   - Should use service method or GUIQueries
   - Direct dependency on squads package

2. **Line 200: Gets FactionManager directly**
   ```go
   squadIDs := cah.combatService.GetFactionManager().GetFactionSquads(currentFactionID)
   ```
   - Service leakage anti-pattern

3. **Line 204: Gets EntityManager directly**
   ```go
   entityManager := cah.combatService.GetEntityManager()
   ```
   - Should not expose internal EntityManager

**Recommendation:**
- CombatService should provide: `GetAliveSquadsInFaction(factionID)` method
- Remove `GetEntityManager()` exposure
- All ECS queries should go through service or GUIQueries

---

### File 3: gui/guisquads/squadbuilder.go

**Status:** ❌ **MAJOR GAME LOGIC VIOLATIONS**

**Critical Issues:**

1. **Lines 185-196: Direct roster queries and entity retrieval**
   ```go
   roster := squads.GetPlayerRoster(sbm.Context.PlayerData.PlayerEntityID, sbm.Context.ECSManager)
   unitEntityID := roster.GetUnitEntityForTemplate(rosterEntry.TemplateName)
   ```
   - Direct roster access in UI
   - Should use SquadBuilderService

2. **Lines 205-207: Direct roster manipulation**
   ```go
   if err := roster.MarkUnitInSquad(unitEntityID, sbm.currentSquadID); err != nil {
       fmt.Printf("Warning: Failed to mark unit as in squad: %v\n", err)
   }
   ```
   - **Business logic in UI** - roster state management
   - Should be handled by service

3. **Lines 242-246: More roster manipulation**
   ```go
   if err := roster.MarkUnitAvailable(rosterEntryID); err != nil {
       fmt.Printf("Warning: Failed to return unit to roster: %v\n", err)
   }
   ```
   - UI code managing game state

4. **Lines 344, 373: Direct squad query functions**
   ```go
   unitIDs := squads.GetUnitIDsInSquad(sbm.currentSquadID, sbm.Context.ECSManager)
   visualization := squads.VisualizeSquad(sbm.currentSquadID, sbm.Context.ECSManager)
   ```
   - Should use service methods or GUIQueries

5. **Lines 441-454: Roster cleanup logic in UI**
   ```go
   roster := squads.GetPlayerRoster(...)
   for row := 0; row < 3; row++ {
       for col := 0; col < 3; col++ {
           if err := roster.MarkUnitAvailable(cell.rosterEntryID); err != nil {
               fmt.Printf("Warning: Failed to return unit to roster: %v\n", err)
           }
       }
   }
   ```
   - Complex business logic in UI code

**Missing Service Methods:**
SquadBuilderService needs:
- `GetAvailableRosterUnits(playerID) []RosterEntry`
- `PlaceRosterUnit(squadID, rosterUnitID, template, row, col) Result` (exists but UI still manages roster directly)
- `RemoveRosterUnit(squadID, rosterUnitID) Result`
- `ClearSquadAndReturnUnits(squadID, rosterUnitIDs) Result`
- `GetSquadVisualization(squadID) string`
- `GetSquadUnitCount(squadID) int`

---

### File 4: gui/guisquads/squaddeploymentmode.go

**Status:** ✅ **CLEAN** (Proper Service Usage)

**Positive Findings:**
- Uses `SquadDeploymentService` exclusively (lines 23, 53)
- No direct ECS access
- Proper coordinate conversion using viewport (lines 179-184)
- All game logic delegated to service

**Example of Good Pattern:**
```go
// Service call with result handling
result := sdm.deploymentService.PlaceSquadAtPosition(squadID, pos)
if !result.Success {
    fmt.Printf("DEBUG: ERROR - Failed to place squad: %s\n", result.Error)
    return
}
```

**Recommendation:** ✅ **NO CHANGES NEEDED** - This is the model to follow

---

### File 5: gui/guisquads/formationeditormode.go

**Status:** ✅ **PURE UI** (Placeholder Implementation)

**Analysis:**
- No game logic
- Pure UI button handling
- Placeholder save/load functionality
- No ECS access

**Recommendation:** ✅ **NO CHANGES NEEDED**

---

### File 6: gui/guisquads/squadmanagementmode.go

**Status:** ⚠️ **QUERY FUNCTION VIOLATIONS**

**Issues Found:**

1. **Line 291: Direct squad query function**
   ```go
   gridVisualization := squads.VisualizeSquad(squadID, smm.Context.ECSManager)
   ```
   - Should use GUIQueries or service method

2. **Line 320: Direct squad query function**
   ```go
   unitIDs := squads.GetUnitIDsInSquad(squadID, smm.Context.ECSManager)
   ```
   - Should use GUIQueries

**Recommendation:**
- Add to GUIQueries:
  - `GetSquadVisualization(squadID) string`
  - `GetSquadUnitDetails(squadID) []UnitDetail`

---

### File 7: gui/core/gamemodecoordinator.go

**Status:** ✅ **PURE UI COORDINATION** (Previously Not Investigated)

**Analysis:**
- Context switching logic only
- No game logic
- No ECS access
- Placeholder state save/restore (lines 215-239)

**Recommendation:** ✅ **NO CHANGES NEEDED**

---

## PHASE 3: DISCOVERY OF UNKNOWN GAME LOGIC

### File: gui/guisquads/unitpurchasemode.go

**Status:** ❌ **SEVERE GAME LOGIC VIOLATIONS**

This file was **NOT mentioned in the completion plan** and contains extensive game logic.

**Critical Issues:**

1. **Lines 242-245: Resource validation logic in UI**
   ```go
   resources := common.GetPlayerResources(upm.Context.PlayerData.PlayerEntityID, upm.Context.ECSManager)
   roster := squads.GetPlayerRoster(upm.Context.PlayerData.PlayerEntityID, upm.Context.ECSManager)
   canBuy := resources != nil && resources.CanAfford(cost) && roster != nil && roster.CanAddUnit()
   upm.buyButton.GetWidget().Disabled = !canBuy
   ```
   - **Business logic in UI** - purchase validation
   - Direct resource/roster access

2. **Lines 288-346: Complete purchase transaction in UI**
   ```go
   func (upm *UnitPurchaseMode) purchaseUnit() {
       resources := common.GetPlayerResources(playerID, upm.Context.ECSManager)
       roster := squads.GetPlayerRoster(playerID, upm.Context.ECSManager)

       if !resources.CanAfford(cost) {
           fmt.Printf("Cannot afford unit: need %d gold, have %d\n", cost, resources.Gold)
           return
       }

       unitEntity, err := squads.CreateUnitEntity(upm.Context.ECSManager, *upm.selectedTemplate)

       if err := roster.AddUnit(unitID, upm.selectedTemplate.Name); err != nil {
           // Rollback logic in UI!
           upm.Context.ECSManager.World.DisposeEntities(unitEntity)
           return
       }

       if err := resources.SpendGold(cost); err != nil {
           // More rollback logic in UI!
           roster.RemoveUnit(unitID)
           upm.Context.ECSManager.World.DisposeEntities(unitEntity)
           return
       }
   }
   ```

**This is a TRANSACTION with rollback logic in UI code!**

3. **Lines 316-328: Entity creation in UI**
   ```go
   unitEntity, err := squads.CreateUnitEntity(upm.Context.ECSManager, *upm.selectedTemplate)
   ```
   - Direct ECS manipulation
   - Entity lifecycle management in UI

4. **Lines 363-371: Cost calculation logic in UI**
   ```go
   func (upm *UnitPurchaseMode) getUnitCost(unitName string) int {
       baseCost := 100
       for _, c := range unitName {
           baseCost += int(c) % 50
       }
       return baseCost
   }
   ```
   - Business logic for pricing
   - Should be in game logic layer

**Missing Service:**
Need a **UnitPurchaseService** with:
- `GetAvailableUnitsForPurchase(playerID) []UnitTemplate`
- `GetUnitCost(unitTemplate) int`
- `CanPurchaseUnit(playerID, unitTemplate) (bool, string)` - returns can buy + reason
- `PurchaseUnit(playerID, unitTemplate) PurchaseResult` - handles entire transaction atomically
- `GetPlayerResources(playerID) ResourceInfo` - read-only DTO

---

### File: gui/guicomponents/guiqueries.go

**Status:** ⚠️ **CONTAINS BUSINESS LOGIC**

**Issue:**

While GUIQueries is meant to centralize queries, it contains some business logic:

**Lines 38-66: FactionInfo calculation**
```go
func (gq *GUIQueries) GetFactionInfo(factionID ecs.EntityID) *FactionInfo {
    // Direct ECS queries
    factionManager := combat.NewFactionManager(gq.ecsManager)
    currentMana, maxMana := factionManager.GetFactionMana(factionID)

    // Business logic - counting alive squads
    aliveCount := 0
    for _, squadID := range squadIDs {
        if !squads.IsSquadDestroyed(squadID, gq.ecsManager) {
            aliveCount++
        }
    }

    return &FactionInfo{...}
}
```

**Analysis:**
- Creating managers in query layer
- Business logic (alive squad counting)
- Should delegate to service

**Lines 108-168: SquadInfo calculation with complex logic**
```go
func (gq *GUIQueries) GetSquadInfo(squadID ecs.EntityID) *SquadInfo {
    // HP calculation logic
    for _, unitID := range unitIDs {
        for _, result := range gq.ecsManager.World.Query(squads.SquadMemberTag) {
            if result.Entity.GetID() == unitID {
                attrs := common.GetComponentType[*common.Attributes](result.Entity, common.AttributeComponent)
                if attrs.CanAct {
                    aliveUnits++
                }
                totalHP += attrs.CurrentHealth
                maxHP += attrs.MaxHealth
            }
        }
    }
}
```

**Issues:**
- Nested queries with business logic
- HP aggregation logic
- Should be service methods returning DTOs

**Recommendation:**
- GUIQueries should be **PURE READ-ONLY QUERIES** returning raw ECS data
- Business logic (aggregations, calculations, validations) should be in services
- Services return DTOs, GUIQueries returns ECS components

---

## PHASE 4: SPECIFIC GAME LOGIC PATTERNS FOUND

### Pattern 1: Direct ECS Access in GUI

**Locations:**
1. `gui/guisquads/unitpurchasemode.go:316` - `CreateUnitEntity`
2. `gui/guisquads/unitpurchasemode.go:328, 337` - `World.DisposeEntities`
3. `gui/guicomponents/guiqueries.go:120, 173, 204, 255` - `World.Query()`

**Issue:** UI code directly manipulates ECS entities, violating service layer abstraction.

---

### Pattern 2: Direct Roster Manipulation

**Locations:**
1. `gui/guisquads/squadbuilder.go:205` - `roster.MarkUnitInSquad()`
2. `gui/guisquads/squadbuilder.go:244, 447` - `roster.MarkUnitAvailable()`

**Issue:** UI manages roster state changes instead of delegating to service.

---

### Pattern 3: Direct Resource Management

**Locations:**
1. `gui/guisquads/unitpurchasemode.go:244` - `resources.CanAfford(cost)`
2. `gui/guisquads/unitpurchasemode.go:305` - `resources.CanAfford(cost)`
3. `gui/guisquads/unitpurchasemode.go:333` - `resources.SpendGold(cost)`

**Issue:** UI performs financial transactions and validates purchasing power.

---

### Pattern 4: Direct Squad Package Query Functions

**Locations:**
1. `gui/guicombat/combat_action_handler.go:206` - `squads.IsSquadDestroyed()`
2. `gui/guisquads/squadmanagementmode.go:291` - `squads.VisualizeSquad()`
3. `gui/guisquads/squadmanagementmode.go:320` - `squads.GetUnitIDsInSquad()`
4. `gui/guisquads/squadbuilder.go:344` - `squads.GetUnitIDsInSquad()`
5. `gui/guisquads/squadbuilder.go:373` - `squads.VisualizeSquad()`

**Issue:** UI bypasses services to query game state directly from squad package.

---

### Pattern 5: Business Logic in UI

**Locations:**
1. `gui/guisquads/unitpurchasemode.go:288-346` - Transaction with rollback logic
2. `gui/guisquads/unitpurchasemode.go:363-371` - Cost calculation formula
3. `gui/guisquads/squadbuilder.go:441-454` - Roster cleanup iteration
4. `gui/guicomponents/guiqueries.go:38-66` - Alive squad counting

**Issue:** Complex game rules and state management implemented in UI layer.

---

### Pattern 6: Service Leakage (Exposing Internal Managers)

**Locations:**
1. `gui/guicombat/combatmode.go:154, 224, 231` - `GetTurnManager()`
2. `gui/guicombat/combat_action_handler.go:200` - `GetFactionManager()`
3. `gui/guicombat/combat_action_handler.go:204` - `GetEntityManager()`

**Issue:** Services expose internal components (managers, EntityManager), breaking encapsulation.

---

## PHASE 5: INTEGRATION POINT ANALYSIS

### How GUI Modes Interact

```
GameModeCoordinator
├── BattleMapManager (tactical context)
│   ├── ExplorationMode → CombatMode
│   ├── CombatMode ↔ BattleMapState (shared UI state)
│   └── SquadDeploymentMode
│
└── OverworldManager (strategic context)
    ├── SquadManagementMode → SquadBuilderMode
    ├── SquadBuilderMode
    ├── FormationEditorMode
    └── UnitPurchaseMode
```

### Shared State Issues

**BattleMapState** (in `gui/core/contextstate.go`):
- `SelectedSquadID` - Shared across combat modes ✅ Appropriate
- `SelectedTargetID` - Combat targeting ✅ Appropriate
- `InAttackMode`, `InMoveMode` - UI flags ✅ Appropriate
- `ValidMoveTiles` - Computed game state ⚠️ Should be ephemeral

**OverworldState** (in `gui/core/contextstate.go`):
- Currently placeholder (lines 52-61)
- No state leakage yet ✅

**Analysis:** State separation is good. BattleMapState correctly stores only UI state, not game state.

---

## COMPLETE FINDINGS SUMMARY

### Files by Service Integration Status

#### ✅ CLEAN (Proper Service Usage - 4 files)
1. `gui/guicombat/combatmode.go` - Uses CombatService (minor GetTurnManager issue)
2. `gui/guicombat/combat_input_handler.go` - Pure input handling
3. `gui/guisquads/squaddeploymentmode.go` - Uses SquadDeploymentService correctly
4. `gui/guisquads/formationeditormode.go` - Pure UI placeholder

#### ⚠️ PARTIAL (Mixed Service/Direct Usage - 3 files)
1. `gui/guicombat/combat_action_handler.go` - Uses service BUT also direct queries
2. `gui/guisquads/squadmanagementmode.go` - Direct query function calls
3. `gui/guicomponents/guiqueries.go` - Contains business logic

#### ❌ MAJOR ISSUES (Direct Game Logic - 2 files)
1. `gui/guisquads/squadbuilder.go` - Roster manipulation, direct queries
2. `gui/guisquads/unitpurchasemode.go` - Transactions, ECS manipulation, business logic

#### 🔍 NOT ANALYZED (20 files)
- All widgets, UI factories, renderers, log managers (likely pure UI)

---

## PRIORITIZED REFACTORING OPPORTUNITIES

### Priority 1: CRITICAL - Missing Service (High Impact)

**Create UnitPurchaseService**

**Impact:** Eliminates entire transaction system from UI

**File:** `gui/guisquads/unitpurchasemode.go`

**Required Methods:**
```go
type UnitPurchaseService struct {
    entityManager *common.EntityManager
}

// GetAvailableUnitsForPurchase returns templates purchaseable by player
func (ups *UnitPurchaseService) GetAvailableUnitsForPurchase(playerID ecs.EntityID) []UnitTemplate

// GetUnitCost returns cost of a unit template
func (ups *UnitPurchaseService) GetUnitCost(template UnitTemplate) int

// CanPurchaseUnit validates if player can purchase unit
func (ups *UnitPurchaseService) CanPurchaseUnit(playerID ecs.EntityID, template UnitTemplate) PurchaseValidationResult

// PurchaseUnit atomically handles purchase transaction with rollback on failure
func (ups *UnitPurchaseService) PurchaseUnit(playerID ecs.EntityID, template UnitTemplate) PurchaseResult

// GetPlayerPurchaseInfo returns DTO with gold, roster capacity, owned units
func (ups *UnitPurchaseService) GetPlayerPurchaseInfo(playerID ecs.EntityID) PlayerPurchaseInfo
```

**Estimated Effort:** 4-6 hours (medium complexity - transaction handling)

**Risk:** Medium (need to preserve transaction semantics)

---

### Priority 2: HIGH - Complete SquadBuilderService Integration

**Impact:** Removes roster manipulation from UI

**File:** `gui/guisquads/squadbuilder.go`

**Required Additional Methods:**
```go
// SquadBuilderService additions

// GetAvailableRosterUnits returns units player owns and can add to squad
func (sbs *SquadBuilderService) GetAvailableRosterUnits(playerID ecs.EntityID) []RosterUnitInfo

// AssignRosterUnitToSquad handles both placement AND roster marking atomically
func (sbs *SquadBuilderService) AssignRosterUnitToSquad(
    squadID, rosterUnitID ecs.EntityID,
    template UnitTemplate,
    row, col int,
) AssignmentResult

// UnassignRosterUnitFromSquad handles removal AND roster return atomically
func (sbs *SquadBuilderService) UnassignRosterUnitFromSquad(
    squadID, rosterUnitID ecs.EntityID,
    row, col int,
) UnassignmentResult

// ClearSquadAndReturnAllUnits bulk operation for grid clearing
func (sbs *SquadBuilderService) ClearSquadAndReturnAllUnits(
    squadID ecs.EntityID,
    rosterUnitIDs []ecs.EntityID,
) ClearResult

// GetSquadVisualization returns ASCII grid visualization
func (sbs *SquadBuilderService) GetSquadVisualization(squadID ecs.EntityID) string

// GetSquadUnitCount returns count of units in squad
func (sbs *SquadBuilderService) GetSquadUnitCount(squadID ecs.EntityID) int
```

**Changes to UI:**
- Lines 185-207: Replace roster access with `AssignRosterUnitToSquad()`
- Lines 241-248: Replace roster access with `UnassignRosterUnitFromSquad()`
- Lines 441-454: Replace loop with `ClearSquadAndReturnAllUnits()`
- Lines 344, 373: Replace direct queries with service methods

**Estimated Effort:** 6-8 hours (high complexity - multiple touchpoints)

**Risk:** Medium-High (roster state management is critical)

---

### Priority 3: MEDIUM - Remove Service Leakage

**Impact:** Improves encapsulation, prevents future violations

**Files:**
- `combat/combat_service.go`
- `gui/guicombat/combatmode.go`
- `gui/guicombat/combat_action_handler.go`

**Required Changes:**

**1. Add missing service methods:**
```go
// CombatService additions

// GetAliveSquadsInFaction returns all alive squads for faction
func (cs *CombatService) GetAliveSquadsInFaction(factionID ecs.EntityID) []ecs.EntityID {
    squadIDs := cs.factionManager.GetFactionSquads(factionID)
    result := []ecs.EntityID{}
    for _, squadID := range squadIDs {
        if !squads.IsSquadDestroyed(squadID, cs.entityManager) {
            result = append(result, squadID)
        }
    }
    return result
}

// Note: GetCurrentRound() already exists, just use it!
```

**2. Remove exposed getters:**
```go
// REMOVE these from CombatService:
// func (cs *CombatService) GetTurnManager() *TurnManager
// func (cs *CombatService) GetFactionManager() *FactionManager
// func (cs *CombatService) GetEntityManager() *common.EntityManager
```

**3. Update UI code:**
- `combatmode.go:154` - Use `cs.GetCurrentRound()` (already exists!)
- `combat_action_handler.go:200` - Use `cs.GetAliveSquadsInFaction()`
- `combat_action_handler.go:206` - Use GUIQueries instead

**Estimated Effort:** 2-3 hours (low complexity)

**Risk:** Low (straightforward refactoring)

---

### Priority 4: MEDIUM - Centralize Squad Queries

**Impact:** Eliminates direct squad package dependencies in UI

**Files:**
- `gui/guicomponents/guiqueries.go` - Add methods
- `gui/guisquads/squadmanagementmode.go` - Update calls
- `gui/guisquads/squadbuilder.go` - Update calls

**Required Changes:**

**Add to GUIQueries:**
```go
// GetSquadVisualization returns ASCII grid of squad formation
func (gq *GUIQueries) GetSquadVisualization(squadID ecs.EntityID) string {
    return squads.VisualizeSquad(squadID, gq.ecsManager)
}

// GetSquadUnitIDs returns all unit entity IDs in squad
func (gq *GUIQueries) GetSquadUnitIDs(squadID ecs.EntityID) []ecs.EntityID {
    return squads.GetUnitIDsInSquad(squadID, gq.ecsManager)
}
```

**Update UI code:**
- `squadmanagementmode.go:291` - Use `gq.GetSquadVisualization()`
- `squadmanagementmode.go:320` - Use `gq.GetSquadUnitIDs()`
- `squadbuilder.go:344` - Use `gq.GetSquadUnitIDs()` OR service method
- `squadbuilder.go:373` - Use `gq.GetSquadVisualization()` OR service method
- `combat_action_handler.go:206` - Use `gq.GetSquadInfo().IsDestroyed` instead

**Estimated Effort:** 2-3 hours (low complexity)

**Risk:** Low (query centralization)

---

### Priority 5: LOW - Clarify GUIQueries vs Service Boundary

**Impact:** Architectural clarity, prevents future confusion

**Current Issue:** GUIQueries contains business logic (aggregations, calculations)

**Recommended Architecture:**

```
┌─────────────────────────────────────────────┐
│ UI Layer                                    │
│ - Mode classes                              │
│ - Handlers                                  │
│ - Components                                │
└────────────┬────────────────────────────────┘
             │
             ├─→ GUIQueries (READ-ONLY ECS QUERIES)
             │   - GetComponentByID()
             │   - FindEntitiesWithTag()
             │   - Raw ECS data only
             │
             └─→ Services (GAME LOGIC + DTOs)
                 - Business rules
                 - Aggregations
                 - Calculations
                 - State mutations
                 - Return rich DTOs
```

**Refactoring Decision:**

**Option A: Move business logic from GUIQueries to Services**
- Pro: Clearer separation
- Pro: Services own all logic
- Con: More service methods needed
- Con: Two query layers (GUIQueries + Service queries)

**Option B: Keep GUIQueries as "Read Service"**
- Pro: Single query interface for UI
- Pro: Fewer service methods
- Con: Blurs query vs logic boundary
- Con: Not true "queries" anymore

**Recommendation: Option A** (Move business logic to services)

**Estimated Effort:** 8-12 hours (refactoring multiple query methods)

**Risk:** Medium (architectural change affects multiple files)

---

## MISSING SERVICES SUMMARY

### 1. UnitPurchaseService ⭐⭐⭐ CRITICAL

**Purpose:** Handles unit purchasing transactions, resource validation, roster management

**Required By:** `gui/guisquads/unitpurchasemode.go`

**Key Methods:**
- `GetAvailableUnitsForPurchase(playerID)`
- `GetUnitCost(template)`
- `CanPurchaseUnit(playerID, template)`
- `PurchaseUnit(playerID, template)` - Atomic transaction
- `GetPlayerPurchaseInfo(playerID)`

---

### 2. Enhanced SquadBuilderService Methods ⭐⭐ HIGH

**Purpose:** Complete squad building with roster integration

**Required By:** `gui/guisquads/squadbuilder.go`

**Key Methods:**
- `GetAvailableRosterUnits(playerID)`
- `AssignRosterUnitToSquad(squadID, rosterUnitID, template, row, col)`
- `UnassignRosterUnitFromSquad(squadID, rosterUnitID, row, col)`
- `ClearSquadAndReturnAllUnits(squadID, rosterUnitIDs)`
- `GetSquadVisualization(squadID)`
- `GetSquadUnitCount(squadID)`

---

### 3. Enhanced CombatService Methods ⭐ MEDIUM

**Purpose:** Complete service encapsulation, no manager leakage

**Required By:** `gui/guicombat/*.go`

**Key Methods:**
- `GetAliveSquadsInFaction(factionID)`
- Remove: `GetTurnManager()`, `GetFactionManager()`, `GetEntityManager()`

---

## CODE EXAMPLES OF VIOLATIONS

### Example 1: Transaction Logic in UI

**File:** `gui/guisquads/unitpurchasemode.go:288-346`

**BEFORE (Current - BAD):**
```go
func (upm *UnitPurchaseMode) purchaseUnit() {
    // UI directly manages multi-step transaction
    playerID := upm.Context.PlayerData.PlayerEntityID
    resources := common.GetPlayerResources(playerID, upm.Context.ECSManager)
    roster := squads.GetPlayerRoster(playerID, upm.Context.ECSManager)

    // Business logic in UI
    if !resources.CanAfford(cost) {
        fmt.Printf("Cannot afford unit: need %d gold, have %d\n", cost, resources.Gold)
        return
    }

    // ECS manipulation in UI
    unitEntity, err := squads.CreateUnitEntity(upm.Context.ECSManager, *upm.selectedTemplate)
    if err != nil {
        fmt.Printf("Failed to create unit: %v\n", err)
        return
    }

    // More business logic
    if err := roster.AddUnit(unitID, upm.selectedTemplate.Name); err != nil {
        // Rollback in UI!
        upm.Context.ECSManager.World.DisposeEntities(unitEntity)
        return
    }

    // Financial transaction in UI
    if err := resources.SpendGold(cost); err != nil {
        // More rollback logic in UI!
        roster.RemoveUnit(unitID)
        upm.Context.ECSManager.World.DisposeEntities(unitEntity)
        return
    }

    fmt.Printf("Purchased unit: %s for %d gold\n", upm.selectedTemplate.Name, cost)
}
```

**AFTER (Proposed - GOOD):**
```go
func (upm *UnitPurchaseMode) purchaseUnit() {
    if upm.selectedTemplate == nil {
        return
    }

    playerID := upm.Context.PlayerData.PlayerEntityID

    // Single service call handles entire transaction atomically
    result := upm.purchaseService.PurchaseUnit(playerID, *upm.selectedTemplate)

    if !result.Success {
        // Just display error - no business logic
        upm.showPurchaseError(result.ErrorReason)
        return
    }

    // Update UI display only
    upm.refreshResourceDisplay()
    upm.updateDetailPanel()
    upm.showPurchaseSuccess(result.UnitName, result.CostPaid)
}
```

**Service Implementation:**
```go
// In UnitPurchaseService
func (ups *UnitPurchaseService) PurchaseUnit(playerID ecs.EntityID, template UnitTemplate) *PurchaseResult {
    result := &PurchaseResult{
        UnitName: template.Name,
    }

    // Validate
    resources := common.GetPlayerResources(playerID, ups.entityManager)
    roster := squads.GetPlayerRoster(playerID, ups.entityManager)
    cost := ups.GetUnitCost(template)

    if !resources.CanAfford(cost) {
        result.Success = false
        result.ErrorReason = fmt.Sprintf("Insufficient gold (need %d, have %d)", cost, resources.Gold)
        return result
    }

    if !roster.CanAddUnit() {
        result.Success = false
        result.ErrorReason = "Roster is full"
        return result
    }

    // Create unit
    unitEntity, err := squads.CreateUnitEntity(ups.entityManager, template)
    if err != nil {
        result.Success = false
        result.ErrorReason = fmt.Sprintf("Failed to create unit: %v", err)
        return result
    }
    unitID := unitEntity.GetID()

    // Add to roster (with rollback on failure)
    if err := roster.AddUnit(unitID, template.Name); err != nil {
        ups.entityManager.World.DisposeEntities(unitEntity)
        result.Success = false
        result.ErrorReason = fmt.Sprintf("Failed to add to roster: %v", err)
        return result
    }

    // Spend gold (with rollback on failure)
    if err := resources.SpendGold(cost); err != nil {
        roster.RemoveUnit(unitID)
        ups.entityManager.World.DisposeEntities(unitEntity)
        result.Success = false
        result.ErrorReason = fmt.Sprintf("Failed to spend gold: %v", err)
        return result
    }

    // Success - populate result
    result.Success = true
    result.UnitID = unitID
    result.CostPaid = cost
    result.RemainingGold = resources.Gold
    result.RosterCount = roster.GetUnitCount()

    return result
}
```

---

### Example 2: Roster Manipulation in UI

**File:** `gui/guisquads/squadbuilder.go:185-207`

**BEFORE (Current - BAD):**
```go
func (sbm *SquadBuilderMode) placeRosterUnitInCell(row, col int, rosterEntry *squads.UnitRosterEntry) {
    // ... validation ...

    // UI directly accesses roster
    roster := squads.GetPlayerRoster(sbm.Context.PlayerData.PlayerEntityID, sbm.Context.ECSManager)
    if roster == nil {
        fmt.Println("Roster not found")
        return
    }

    // UI gets entity from roster
    unitEntityID := roster.GetUnitEntityForTemplate(rosterEntry.TemplateName)
    if unitEntityID == 0 {
        fmt.Printf("No available units of type %s\n", rosterEntry.TemplateName)
        return
    }

    // Service call (good) but incomplete
    result := sbm.squadBuilderSvc.PlaceUnit(sbm.currentSquadID, unitEntityID, *unitTemplate, row, col)
    if !result.Success {
        fmt.Printf("Failed to place unit: %s\n", result.Error)
        return
    }

    // UI directly manipulates roster state (BAD!)
    if err := roster.MarkUnitInSquad(unitEntityID, sbm.currentSquadID); err != nil {
        fmt.Printf("Warning: Failed to mark unit as in squad: %v\n", err)
    }

    // UI manages grid display
    sbm.gridManager.UpdateDisplayForPlacedUnit(result.UnitID, unitTemplate, row, col, unitEntityID)
    sbm.refreshUnitPalette()
    sbm.updateCapacityDisplay()
}
```

**AFTER (Proposed - GOOD):**
```go
func (sbm *SquadBuilderMode) placeRosterUnitInCell(row, col int, rosterEntry *squads.UnitRosterEntry) {
    if sbm.selectedRosterEntry == nil {
        return
    }

    // Single service call handles BOTH placement and roster marking
    result := sbm.squadBuilderSvc.AssignRosterUnitToSquad(
        sbm.currentSquadID,
        rosterEntry.EntityID, // Roster unit entity ID
        rosterEntry.TemplateName,
        row, col,
    )

    if !result.Success {
        // Just display error
        sbm.showPlacementError(result.Error)
        return
    }

    // Update UI display only
    sbm.gridManager.UpdateDisplayForPlacedUnit(
        result.PlacedUnitID,
        result.Template,
        row, col,
        result.RosterUnitID,
    )
    sbm.refreshUnitPalette()
    sbm.updateCapacityDisplay()
}
```

**Service Implementation:**
```go
// In SquadBuilderService
func (sbs *SquadBuilderService) AssignRosterUnitToSquad(
    squadID, rosterUnitID ecs.EntityID,
    templateName string,
    row, col int,
) *AssignmentResult {
    result := &AssignmentResult{}

    // Get roster
    roster := squads.GetPlayerRoster(playerID, sbs.entityManager)
    if roster == nil {
        result.Error = "roster not found"
        return result
    }

    // Validate roster has unit
    if !roster.HasAvailableUnit(rosterUnitID) {
        result.Error = "unit not available in roster"
        return result
    }

    // Get template
    template := squads.GetUnitTemplate(templateName)
    if template == nil {
        result.Error = "template not found"
        return result
    }

    // Place unit in squad (creates new unit entity)
    placeResult := sbs.PlaceUnit(squadID, rosterUnitID, *template, row, col)
    if !placeResult.Success {
        result.Error = placeResult.Error
        return result
    }

    // Mark roster unit as in squad (atomic with placement)
    if err := roster.MarkUnitInSquad(rosterUnitID, squadID); err != nil {
        // Rollback placement
        sbs.RemoveUnitFromGrid(squadID, row, col)
        result.Error = fmt.Sprintf("failed to mark roster unit: %v", err)
        return result
    }

    // Success
    result.Success = true
    result.PlacedUnitID = placeResult.UnitID
    result.RosterUnitID = rosterUnitID
    result.Template = template
    result.RemainingCapacity = placeResult.RemainingCapacity

    return result
}
```

---

### Example 3: Service Leakage

**File:** `gui/guicombat/combat_action_handler.go:200-210`

**BEFORE (Current - BAD):**
```go
func (cah *CombatActionHandler) CycleSquadSelection() {
    currentFactionID := cah.combatService.GetCurrentFaction()
    if currentFactionID == 0 || !cah.queries.IsPlayerFaction(currentFactionID) {
        return
    }

    // Accessing internal manager through service (BAD!)
    squadIDs := cah.combatService.GetFactionManager().GetFactionSquads(currentFactionID)

    // Filter out destroyed squads (business logic in UI)
    aliveSquads := []ecs.EntityID{}
    entityManager := cah.combatService.GetEntityManager() // Exposing EntityManager (BAD!)
    for _, squadID := range squadIDs {
        if !squads.IsSquadDestroyed(squadID, entityManager) {
            aliveSquads = append(aliveSquads, squadID)
        }
    }

    // ... selection logic ...
}
```

**AFTER (Proposed - GOOD):**
```go
func (cah *CombatActionHandler) CycleSquadSelection() {
    currentFactionID := cah.combatService.GetCurrentFaction()
    if currentFactionID == 0 || !cah.queries.IsPlayerFaction(currentFactionID) {
        return
    }

    // Single service call returns filtered list
    aliveSquads := cah.combatService.GetAliveSquadsInFaction(currentFactionID)

    if len(aliveSquads) == 0 {
        return
    }

    // Find current index
    currentIndex := -1
    selectedSquad := cah.battleMapState.SelectedSquadID
    for i, squadID := range aliveSquads {
        if squadID == selectedSquad {
            currentIndex = i
            break
        }
    }

    // Select next squad
    nextIndex := (currentIndex + 1) % len(aliveSquads)
    cah.SelectSquad(aliveSquads[nextIndex])
}
```

**Service Implementation:**
```go
// In CombatService - NEW METHOD
func (cs *CombatService) GetAliveSquadsInFaction(factionID ecs.EntityID) []ecs.EntityID {
    // Service owns the logic
    squadIDs := cs.factionManager.GetFactionSquads(factionID)

    aliveSquads := []ecs.EntityID{}
    for _, squadID := range squadIDs {
        if !squads.IsSquadDestroyed(squadID, cs.entityManager) {
            aliveSquads = append(aliveSquads, squadID)
        }
    }

    return aliveSquads
}

// REMOVE these methods from CombatService:
// func (cs *CombatService) GetTurnManager() *TurnManager { ... }
// func (cs *CombatService) GetFactionManager() *FactionManager { ... }
// func (cs *CombatService) GetEntityManager() *common.EntityManager { ... }
```

---

## REFACTORING RECOMMENDATIONS

### Immediate Actions (Next Sprint)

1. **Create UnitPurchaseService** (Priority 1)
   - Estimated: 4-6 hours
   - Risk: Medium
   - Impact: HIGH - Eliminates entire transaction system from UI
   - File: `combat/unit_purchase_service.go` (new)
   - Update: `gui/guisquads/unitpurchasemode.go`

2. **Complete SquadBuilderService Integration** (Priority 2)
   - Estimated: 6-8 hours
   - Risk: Medium-High
   - Impact: HIGH - Removes roster manipulation from UI
   - File: `squads/squad_builder_service.go` (enhance)
   - Update: `gui/guisquads/squadbuilder.go`

3. **Remove Service Leakage** (Priority 3)
   - Estimated: 2-3 hours
   - Risk: Low
   - Impact: MEDIUM - Improves encapsulation
   - File: `combat/combat_service.go` (enhance)
   - Update: `gui/guicombat/*.go`

**Total Estimated Effort for Immediate Actions:** 12-17 hours

---

### Medium-Term Actions (Next 2-3 Sprints)

4. **Centralize Squad Queries** (Priority 4)
   - Estimated: 2-3 hours
   - Risk: Low
   - Impact: MEDIUM - Reduces direct package dependencies
   - File: `gui/guicomponents/guiqueries.go` (enhance)
   - Update: Multiple GUI files

5. **Clarify GUIQueries vs Service Boundary** (Priority 5)
   - Estimated: 8-12 hours
   - Risk: Medium
   - Impact: MEDIUM - Architectural clarity
   - Files: Multiple (requires architectural decision first)

**Total Estimated Effort for Medium-Term:** 10-15 hours

---

### Long-Term Architectural Goals

**Define Clear Boundaries:**

```
GUI Layer (UI Code Only)
├─ No ECS access (except through services/queries)
├─ No business logic
├─ No direct package function calls
└─ Only UI state management

↓ (calls)

Query Layer (Read-Only)
├─ GUIQueries - Raw ECS component access
├─ No aggregations or calculations
├─ Returns ECS components/IDs directly
└─ Pure data retrieval

↓ (calls)

Service Layer (Game Logic)
├─ All business rules
├─ All state mutations
├─ All aggregations/calculations
├─ Transaction handling
├─ Returns rich DTOs
└─ Owns game managers

↓ (accesses)

ECS Layer (Data Storage)
├─ Components (pure data)
├─ Systems (pure logic functions)
├─ Queries (by tag/component)
└─ No business logic
```

---

## VALIDATION CHECKLIST

Before claiming "service separation complete", verify:

- [ ] **No direct ECS access in GUI** (`World.Query`, `NewEntity`, `DisposeEntities`)
- [ ] **No direct component manipulation** (`AddComponent`, `RemoveComponent`, `GetComponent` in UI)
- [ ] **No direct package query functions** (`GetUnitIDsInSquad`, `IsSquadDestroyed`, etc. in UI)
- [ ] **No roster manipulation in UI** (`MarkUnitInSquad`, `MarkUnitAvailable` in UI)
- [ ] **No resource management in UI** (`GetPlayerResources`, `CanAfford`, `SpendGold` in UI)
- [ ] **No entity creation in UI** (`CreateUnitEntity` in UI)
- [ ] **No business logic in UI** (validation, calculations, aggregations in UI)
- [ ] **No transaction logic in UI** (multi-step operations with rollback in UI)
- [ ] **No service leakage** (services don't expose internal managers/EntityManager)
- [ ] **Services are comprehensive** (cover all game logic needs of UI)
- [ ] **GUIQueries vs Services clearly defined** (pure queries vs DTOs with logic)

**Current Score: 3/11 ❌**

---

## CONCLUSION

### Is Service Separation "100% Complete"? NO.

**Actual Completion Estimate: 40-50%**

**What's Done:**
- 4 services exist (CombatService, SquadService, SquadBuilderService, SquadDeploymentService)
- Some GUI files use services correctly (combatmode.go, squaddeploymentmode.go)
- Service pattern is established and working

**What Remains:**
- 1 entire file with massive violations (unitpurchasemode.go - ~400 lines)
- 1 file with extensive violations (squadbuilder.go - 10+ violation sites)
- 2 files with partial violations (combat_action_handler.go, squadmanagementmode.go)
- 1 service missing entirely (UnitPurchaseService)
- Multiple service methods needed to complete integration
- Architectural clarity needed (GUIQueries vs Services)

**Recommended Next Steps:**

1. Create UnitPurchaseService (Priority 1 - Critical)
2. Complete SquadBuilderService integration (Priority 2 - High)
3. Remove service leakage (Priority 3 - Medium)
4. Update project plan to reflect actual status (40-50%, not 100%)
5. Continue systematic refactoring using priorities above

**Estimated Total Remaining Effort:** 22-32 hours of focused refactoring work

---

## APPENDIX: DETAILED FILE ANALYSIS

### Files with Direct ECS Access

1. **gui/guisquads/unitpurchasemode.go**
   - Line 316: `squads.CreateUnitEntity()`
   - Line 328: `World.DisposeEntities()`
   - Line 337: `World.DisposeEntities()`

2. **gui/guicomponents/guiqueries.go**
   - Line 120: `World.Query(squads.SquadMemberTag)`
   - Line 173: `World.Query(squads.SquadTag)`
   - Line 204: `World.Query(combat.MapPositionTag)`
   - Line 255: `World.Query(combat.FactionTag)`

### Files with Direct Squad Package Queries

1. **gui/guicombat/combat_action_handler.go**
   - Line 206: `squads.IsSquadDestroyed()`

2. **gui/guisquads/squadmanagementmode.go**
   - Line 291: `squads.VisualizeSquad()`
   - Line 320: `squads.GetUnitIDsInSquad()`

3. **gui/guisquads/squadbuilder.go**
   - Line 344: `squads.GetUnitIDsInSquad()`
   - Line 373: `squads.VisualizeSquad()`

4. **gui/guicomponents/guiqueries.go**
   - Line 52: `squads.IsSquadDestroyed()`
   - Line 113: `squads.GetUnitIDsInSquad()`
   - Line 163: `squads.IsSquadDestroyed()`
   - Line 209: `squads.IsSquadDestroyed()`
   - Line 226: `squads.IsSquadDestroyed()`

### Files with Roster Manipulation

1. **gui/guisquads/squadbuilder.go**
   - Line 110: `squads.GetPlayerRoster()`
   - Line 185: `squads.GetPlayerRoster()`
   - Line 205: `roster.MarkUnitInSquad()`
   - Line 242: `roster.MarkUnitAvailable()`
   - Line 441: `squads.GetPlayerRoster()`
   - Line 447: `roster.MarkUnitAvailable()`
   - Line 477: `squads.GetPlayerRoster()`

2. **gui/guisquads/unitpurchasemode.go**
   - Line 72: `squads.GetPlayerRoster()`
   - Line 243: `squads.GetPlayerRoster()`
   - Line 295: `squads.GetPlayerRoster()`
   - Line 351: `squads.GetPlayerRoster()`

### Files with Resource Management

1. **gui/guisquads/unitpurchasemode.go**
   - Line 242: `common.GetPlayerResources()`
   - Line 244: `resources.CanAfford()`
   - Line 294: `common.GetPlayerResources()`
   - Line 305: `resources.CanAfford()`
   - Line 333: `resources.SpendGold()`
   - Line 350: `common.GetPlayerResources()`

---

**END OF ANALYSIS**
