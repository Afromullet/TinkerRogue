# GUI Package Refactoring Analysis
**Generated**: 2025-11-10 15:47
**Target**: gui/ package (18 files, 4,794 LOC)
**Context**: Go roguelike using Ebiten + ebitenui, post-ButtonConfig pattern implementation

---

## EXECUTIVE SUMMARY

### Current State Assessment

The GUI package demonstrates **strong architectural foundations** with successful pattern implementations:
- **Mode system**: Well-designed UIMode interface with state machine (UIModeManager)
- **BaseMode pattern**: Excellent DRY application reducing boilerplate across 9 modes
- **ButtonConfig pattern**: Recently applied (2025-11-07), successfully reduces widget creation duplication
- **Functional options**: panelconfig.go uses modern Go patterns (BuildPanel with options)

**Overall Health**: 7.5/10
- Strengths: Clean abstractions, consistent patterns, good separation of UI modes
- Weaknesses: ECS query duplication, UI update logic scattered, some rendering concerns mixed

### Critical Issues Identified

1. **ECS Query Duplication** (High Impact)
   - Same queries repeated across 5+ mode files
   - Each mode independently queries squads, factions, units
   - No centralized query layer despite `squadqueries.go` existing

2. **UI Update Logic Scattered** (Medium Impact)
   - Multiple `update*()` methods duplicated across modes
   - `combatmode.go` has 6 update methods (577 LOC of updates)
   - No unified update pattern or reusable update components

3. **Mode State Synchronization** (Medium Impact)
   - Each mode refreshes on `Enter()`, but logic differs
   - No clear contract for when/how modes should refresh data
   - Potential stale data if transitions skip refresh

4. **Rendering Mixed with State** (Low-Medium Impact)
   - `combatmode.go` mixes rendering logic with mode logic (745 LOC total)
   - Viewport calculations duplicated in rendering methods
   - Opportunity to extract rendering systems

### Strengths to Preserve

1. **UIMode Interface** - Excellent lifecycle contract (Initialize/Enter/Exit/Update/Render/HandleInput)
2. **BaseMode Composition** - DRY achieved through embedding, not inheritance
3. **PanelBuilders** - High-level composition reduces boilerplate effectively
4. **Functional Options** - Modern Go pattern applied successfully in panelconfig.go
5. **Query Functions** - squadqueries.go shows the right direction (just needs expansion)

### Recommended Approach

**Incremental refactoring in 3 phases:**
1. **Phase 1**: Extract ECS query layer (unify squad/faction/unit queries)
2. **Phase 2**: Create reusable UI update components (squad lists, detail panels, etc.)
3. **Phase 3**: Extract rendering concerns from modes (viewport, highlights, overlays)

**Estimated Impact**: 15-20% LOC reduction, 40% complexity reduction in mode files

---

## DETAILED ANALYSIS BY FILE

### Architecture Overview

**File Breakdown** (18 files, 4,794 LOC):

**Core Infrastructure** (3 files, ~450 LOC):
- `uimode.go` - UIMode interface, UIContext, InputState (70 LOC)
- `modemanager.go` - UIModeManager state machine (175 LOC)
- `basemode.go` - BaseMode with common infrastructure (95 LOC)

**Widget Factories** (4 files, ~800 LOC):
- `createwidgets.go` - ButtonConfig, ListConfig, PanelConfig, TextAreaConfig (240 LOC)
- `guiresources.go` - Resource loading, font/image management (307 LOC)
- `panelconfig.go` - Functional options for BuildPanel (270 LOC)
- `panels.go` - PanelBuilders with high-level builders (262 LOC)

**Layout & Helpers** (3 files, ~150 LOC):
- `layout.go` - LayoutConfig for responsive positioning (48 LOC)
- `modehelpers.go` - CreateCloseButton, CreateBottomCenterButtonContainer (45 LOC)
- `squadqueries.go` - ECS query functions (FindAllSquads, GetSquadName, etc.) (80 LOC)

**UI Modes** (8 files, ~3,400 LOC):
- `explorationmode.go` - Default exploration UI (256 LOC)
- `combatmode.go` - Turn-based combat interface **(988 LOC) ⚠️ LARGEST**
- `squadmanagementmode.go` - Squad viewing/management (238 LOC)
- `squadbuilder.go` - Squad creation interface (~500 LOC estimated)
- `squaddeploymentmode.go` - Map-based squad placement (~400 LOC estimated)
- `formationeditormode.go` - 3x3 grid formation editing (~200 LOC estimated)
- `inventorymode.go` - Inventory browsing with filters (302 LOC)
- `infomode.go` - Right-click inspection (176 LOC)

---

## ISSUE ANALYSIS WITH CODE EXAMPLES

### Issue 1: ECS Query Duplication (HIGH PRIORITY)

**Problem**: Same ECS queries repeated across multiple mode files with slight variations.

**Evidence**:

**combatmode.go** (lines 263-272):
```go
func (cm *CombatMode) getFactionName(factionID ecs.EntityID) string {
    for _, result := range cm.context.ECSManager.World.Query(cm.context.ECSManager.Tags["faction"]) {
        factionData := common.GetComponentType[*combat.FactionData](result.Entity, combat.FactionComponent)
        if factionData.FactionID == factionID {
            return factionData.Name
        }
    }
    return "Unknown Faction"
}
```

**squadmanagementmode.go** (lines 78-86):
```go
func (smm *SquadManagementMode) Enter(fromMode UIMode) error {
    // Clear old panels
    smm.clearSquadPanels()

    // Find all squads in the game using shared query
    allSquads := FindAllSquads(smm.context.ECSManager)  // ✅ Uses squadqueries.go

    // Create panel for each squad
    for _, squadID := range allSquads {
        panel := smm.createSquadPanel(squadID)
```

**combatmode.go** (lines 444-464):
```go
func (cm *CombatMode) updateSquadList() {
    // ... code ...
    squadIDs := cm.factionManager.GetFactionSquads(currentFactionID)

    for _, squadID := range squadIDs {
        // Skip destroyed squads
        if squads.IsSquadDestroyed(squadID, cm.context.ECSManager) {
            continue
        }

        squadName := GetSquadName(cm.context.ECSManager, squadID)  // ✅ Uses squadqueries.go

        // Create button for each squad
        localSquadID := squadID
        squadButton := CreateButtonWithConfig(ButtonConfig{
            Text: squadName,
            OnClick: func() { cm.selectSquad(localSquadID) },
        })
```

**Analysis**:
- `squadqueries.go` exists but only has 4 functions (FindAllSquads, GetSquadName, GetSquadAtPosition, FindSquadsByFaction)
- `combatmode.go` has 7+ unique query patterns not in squadqueries.go:
  - `getFactionName()` - queries faction entities
  - `isPlayerFaction()` - checks faction control
  - `findActionStateEntity()` - finds action state for squad
  - `showAvailableTargets()` - finds enemy squads
  - Inline unit queries in `updateSquadDetail()` (lines 476-492)

**Impact**:
- **Complexity**: Each mode has 3-5 unique query methods (15-20 total duplicated queries)
- **Maintainability**: ECS schema changes require updates across 5+ files
- **Testability**: Cannot test query logic independently from UI

**Recommended Solution**: See Approach 1 below

---

### Issue 2: UI Update Logic Duplication (MEDIUM PRIORITY)

**Problem**: Multiple modes implement similar update patterns independently.

**Evidence**:

**combatmode.go** has 6 update methods (577 LOC):
```go
func (cm *CombatMode) updateSquadList()         // Lines 422-465 (44 LOC)
func (cm *CombatMode) updateSquadDetail()       // Lines 467-521 (55 LOC)
func (cm *CombatMode) updateTurnDisplay()       // Lines 583-600 (18 LOC)
func (cm *CombatMode) updateFactionDisplay()    // Lines 602-626 (25 LOC)
func (cm *CombatMode) renderMovementTiles()     // Lines 638-668 (31 LOC)
func (cm *CombatMode) renderAllSquadHighlights() // Lines 670-745 (76 LOC)
```

**inventorymode.go** has refresh pattern:
```go
func (im *InventoryMode) refreshItemList() // Lines 214-257 (44 LOC)
    // Query inventory based on filter
    switch im.currentFilter {
    case "Throwables":
        throwableEntries := gear.GetThrowableItems(...)
    case "All":
        allEntries := gear.GetInventoryForDisplay(...)
    }
    im.itemList.SetEntries(entries)
}
```

**squadmanagementmode.go** rebuilds panels on Enter:
```go
func (smm *SquadManagementMode) Enter(fromMode UIMode) error {
    // Clear old panels
    smm.clearSquadPanels()

    // Find all squads
    allSquads := FindAllSquads(smm.context.ECSManager)

    // Create panel for each squad
    for _, squadID := range allSquads {
        panel := smm.createSquadPanel(squadID)
        smm.squadPanels = append(smm.squadPanels, panel)
        smm.rootContainer.AddChild(panel.container)
    }
}
```

**Analysis**:
- **Pattern**: Each mode has `update*()` or `refresh*()` methods that:
  1. Query ECS for data
  2. Format data for display
  3. Update widget state (SetText, SetEntries, etc.)
- **Duplication**: Squad list updates repeated in 3 modes (combat, management, deployment)
- **Consistency**: No unified approach - some use Update(), some use Enter(), some have refresh methods

**Impact**:
- **LOC**: ~200 lines of update logic that could be unified
- **Bugs**: Inconsistent update timing (some modes refresh on Update(), some on Enter())
- **Coupling**: Update logic tightly coupled to widget structure

**Recommended Solution**: See Approach 2 below

---

### Issue 3: Rendering Concerns Mixed with Mode Logic (LOW-MEDIUM PRIORITY)

**Problem**: Rendering logic embedded in mode files instead of separate rendering systems.

**Evidence**:

**combatmode.go** `Render()` method (lines 628-636):
```go
func (cm *CombatMode) Render(screen *ebiten.Image) {
    // Render all squad highlights (show player vs enemy squads with different colors)
    cm.renderAllSquadHighlights(screen)

    // Render valid movement tiles if in move mode
    if cm.inMoveMode && len(cm.validMoveTiles) > 0 {
        cm.renderMovementTiles(screen)
    }
}
```

**combatmode.go** rendering helper (lines 638-668):
```go
func (cm *CombatMode) renderMovementTiles(screen *ebiten.Image) {
    // Get player position for viewport centering
    playerPos := *cm.context.PlayerData.Pos

    // Create viewport centered on player using the initialized ScreenInfo
    // Update screen dimensions from current screen buffer
    screenData := graphics.ScreenInfo
    screenData.ScreenWidth = screen.Bounds().Dx()
    screenData.ScreenHeight = screen.Bounds().Dy()

    manager := coords.NewCoordinateManager(screenData)
    viewport := coords.NewViewport(manager, playerPos)

    tileSize := screenData.TileSize
    scaleFactor := screenData.ScaleFactor

    // Create a semi-transparent green overlay for valid tiles
    for _, pos := range cm.validMoveTiles {
        screenX, screenY := viewport.LogicalToScreen(pos)
        scaledTileSize := tileSize * scaleFactor
        rect := ebiten.NewImage(scaledTileSize, scaledTileSize)
        rect.Fill(color.RGBA{R: 0, G: 255, B: 0, A: 80})
        // ... draw rect
    }
}
```

**Analysis**:
- **Viewport Duplication**: Viewport creation repeated in 2 rendering methods (lines 638-649, lines 670-679)
- **Mixing Concerns**: CombatMode has:
  - State management (selected squad, attack mode, etc.)
  - Input handling (mouse clicks, keyboard)
  - Rendering (movement tiles, squad highlights)
- **Total Rendering LOC in CombatMode**: ~150 LOC (15% of file)

**Impact**:
- **Testability**: Cannot test rendering independently from combat logic
- **Reusability**: Squad highlighting logic could be reused in other modes
- **Performance**: Viewport recreated every frame instead of cached

**Recommended Solution**: See Approach 3 below

---

### Issue 4: Mode Enter/Exit Inconsistency (LOW PRIORITY)

**Problem**: Inconsistent approach to data refresh across mode lifecycle.

**Evidence**:

**explorationmode.go** refreshes on Enter:
```go
func (em *ExplorationMode) Enter(fromMode UIMode) error {
    fmt.Println("Entering Exploration Mode")

    // Refresh player stats
    if em.context.PlayerData != nil && em.statsTextArea != nil {
        em.statsTextArea.SetText(em.context.PlayerData.PlayerAttributes().DisplayString())
    }

    return nil
}
```

**inventorymode.go** refreshes on Enter:
```go
func (im *InventoryMode) Enter(fromMode UIMode) error {
    fmt.Println("Entering Inventory Mode")

    // Apply initial filter if one was set
    if im.initialFilter != "" {
        im.currentFilter = im.initialFilter
        im.initialFilter = "" // Reset after use
    }

    im.refreshItemList()
    return nil
}
```

**combatmode.go** initializes on Enter:
```go
func (cm *CombatMode) Enter(fromMode UIMode) error {
    fmt.Println("Entering Combat Mode")
    cm.addCombatLog("=== COMBAT STARTED ===")

    // Collect all factions
    var factionIDs []ecs.EntityID
    for _, result := range cm.context.ECSManager.World.Query(cm.context.ECSManager.Tags["faction"]) {
        factionData := common.GetComponentType[*combat.FactionData](result.Entity, combat.FactionComponent)
        factionIDs = append(factionIDs, factionData.FactionID)
    }

    // Initialize combat with all factions
    if len(factionIDs) > 0 {
        if err := cm.turnManager.InitializeCombat(factionIDs); err != nil {
            // ... error handling
        }
        // ... more initialization
    }
```

**squadmanagementmode.go** rebuilds UI on Enter:
```go
func (smm *SquadManagementMode) Enter(fromMode UIMode) error {
    fmt.Println("Entering Squad Management Mode")

    // Clear old panels
    smm.clearSquadPanels()

    // Find all squads in the game using shared query
    allSquads := FindAllSquads(smm.context.ECSManager)

    // Create panel for each squad
    for _, squadID := range allSquads {
        panel := smm.createSquadPanel(squadID)
        smm.squadPanels = append(smm.squadPanels, panel)
        smm.rootContainer.AddChild(panel.container)
    }

    return nil
}
```

**Analysis**:
- **Inconsistent Refresh**:
  - ExplorationMode: Simple text update
  - InventoryMode: Refresh list entries
  - CombatMode: Full combat initialization
  - SquadManagementMode: Rebuild entire UI
- **No Clear Contract**: UIMode interface doesn't specify refresh expectations
- **Performance**: SquadManagementMode rebuilds ALL panels every Enter (expensive if many squads)

**Impact**:
- **Bugs**: Potential stale data if Enter() implementation doesn't refresh properly
- **Performance**: Unnecessary rebuilds in some modes
- **Maintenance**: No clear pattern to follow when adding new modes

**Recommended Solution**: See Approach 2 below (includes refresh pattern)

---

## REFACTORING APPROACHES

### Approach 1: Unified ECS Query Layer

**Strategic Focus**: Create a centralized query service that eliminates ECS query duplication across all modes.

**Problem Statement**:
Currently, each UI mode independently queries the ECS for squads, factions, units, and combat data. This creates:
- 15-20 duplicated query methods across 5+ files
- Inconsistent query patterns (some use Tags, some use direct component access)
- Difficult to maintain when ECS schema changes
- Cannot test query logic independently

**Solution Overview**:
Expand `squadqueries.go` into a comprehensive `guiqueries.go` service that provides all ECS queries needed by UI modes. Apply the **Query Object Pattern** to encapsulate complex queries.

**Code Example**:

*Before* (combatmode.go, lines 263-283):
```go
func (cm *CombatMode) getFactionName(factionID ecs.EntityID) string {
    for _, result := range cm.context.ECSManager.World.Query(cm.context.ECSManager.Tags["faction"]) {
        factionData := common.GetComponentType[*combat.FactionData](result.Entity, combat.FactionComponent)
        if factionData.FactionID == factionID {
            return factionData.Name
        }
    }
    return "Unknown Faction"
}

func (cm *CombatMode) isPlayerFaction(factionID ecs.EntityID) bool {
    for _, result := range cm.context.ECSManager.World.Query(cm.context.ECSManager.Tags["faction"]) {
        factionData := common.GetComponentType[*combat.FactionData](result.Entity, combat.FactionComponent)
        if factionData.FactionID == factionID {
            return factionData.IsPlayerControlled
        }
    }
    return false
}
```

*After* (guiqueries.go):
```go
package gui

import (
    "game_main/combat"
    "game_main/common"
    "game_main/coords"
    "game_main/squads"
    "github.com/bytearena/ecs"
)

// GUIQueries provides centralized ECS query functions for all UI modes.
// This eliminates query duplication and provides a consistent query interface.
type GUIQueries struct {
    ecsManager *common.EntityManager
}

// NewGUIQueries creates a new query service
func NewGUIQueries(ecsManager *common.EntityManager) *GUIQueries {
    return &GUIQueries{ecsManager: ecsManager}
}

// ===== FACTION QUERIES =====

// FactionInfo encapsulates all faction data needed by UI
type FactionInfo struct {
    ID               ecs.EntityID
    Name             string
    IsPlayerControlled bool
    CurrentMana      int
    MaxMana          int
    SquadIDs         []ecs.EntityID
    AliveSquadCount  int
}

// GetFactionInfo returns complete faction information for UI display
func (gq *GUIQueries) GetFactionInfo(factionID ecs.EntityID) *FactionInfo {
    for _, result := range gq.ecsManager.World.Query(gq.ecsManager.Tags["faction"]) {
        factionData := common.GetComponentType[*combat.FactionData](result.Entity, combat.FactionComponent)
        if factionData.FactionID == factionID {
            // Get faction manager for additional data
            factionManager := combat.NewFactionManager(gq.ecsManager)
            currentMana, maxMana := factionManager.GetFactionMana(factionID)
            squadIDs := factionManager.GetFactionSquads(factionID)

            // Count alive squads
            aliveCount := 0
            for _, squadID := range squadIDs {
                if !squads.IsSquadDestroyed(squadID, gq.ecsManager) {
                    aliveCount++
                }
            }

            return &FactionInfo{
                ID:                 factionID,
                Name:               factionData.Name,
                IsPlayerControlled: factionData.IsPlayerControlled,
                CurrentMana:        currentMana,
                MaxMana:            maxMana,
                SquadIDs:           squadIDs,
                AliveSquadCount:    aliveCount,
            }
        }
    }
    return nil
}

// GetFactionName returns just the faction name (lightweight query)
func (gq *GUIQueries) GetFactionName(factionID ecs.EntityID) string {
    info := gq.GetFactionInfo(factionID)
    if info != nil {
        return info.Name
    }
    return "Unknown Faction"
}

// IsPlayerFaction checks if faction is player-controlled
func (gq *GUIQueries) IsPlayerFaction(factionID ecs.EntityID) bool {
    info := gq.GetFactionInfo(factionID)
    return info != nil && info.IsPlayerControlled
}

// ===== SQUAD QUERIES =====

// SquadInfo encapsulates all squad data needed by UI
type SquadInfo struct {
    ID            ecs.EntityID
    Name          string
    UnitIDs       []ecs.EntityID
    AliveUnits    int
    TotalUnits    int
    CurrentHP     int
    MaxHP         int
    Position      *coords.LogicalPosition
    FactionID     ecs.EntityID
    IsDestroyed   bool
    HasActed      bool
    HasMoved      bool
    MovementRemaining int
}

// GetSquadInfo returns complete squad information for UI display
func (gq *GUIQueries) GetSquadInfo(squadID ecs.EntityID) *SquadInfo {
    // Get squad name
    name := gq.GetSquadName(squadID)

    // Get unit IDs
    unitIDs := squads.GetUnitIDsInSquad(squadID, gq.ecsManager)

    // Calculate HP and alive units
    aliveUnits := 0
    totalHP := 0
    maxHP := 0
    for _, unitID := range unitIDs {
        for _, result := range gq.ecsManager.World.Query(gq.ecsManager.Tags["squadmember"]) {
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

    // Get position and faction
    var position *coords.LogicalPosition
    var factionID ecs.EntityID
    for _, result := range gq.ecsManager.World.Query(gq.ecsManager.Tags["mapposition"]) {
        mapPos := common.GetComponentType[*combat.MapPositionData](result.Entity, combat.MapPositionComponent)
        if mapPos.SquadID == squadID {
            position = &mapPos.Position
            factionID = mapPos.FactionID
            break
        }
    }

    // Get action state
    hasActed := false
    hasMoved := false
    movementRemaining := 0
    for _, result := range gq.ecsManager.World.Query(gq.ecsManager.Tags["actionstate"]) {
        actionState := common.GetComponentType[*combat.ActionStateData](result.Entity, combat.ActionStateComponent)
        if actionState.SquadID == squadID {
            hasActed = actionState.HasActed
            hasMoved = actionState.HasMoved
            movementRemaining = actionState.MovementRemaining
            break
        }
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
        IsDestroyed:       squads.IsSquadDestroyed(squadID, gq.ecsManager),
        HasActed:          hasActed,
        HasMoved:          hasMoved,
        MovementRemaining: movementRemaining,
    }
}

// GetSquadName returns the squad name (already exists in squadqueries.go, include here)
func (gq *GUIQueries) GetSquadName(squadID ecs.EntityID) string {
    return GetSquadName(gq.ecsManager, squadID)
}

// FindAllSquads returns all squad IDs (already exists, include here)
func (gq *GUIQueries) FindAllSquads() []ecs.EntityID {
    return FindAllSquads(gq.ecsManager)
}

// GetSquadAtPosition finds squad at given position (already exists, include here)
func (gq *GUIQueries) GetSquadAtPosition(pos coords.LogicalPosition) ecs.EntityID {
    return GetSquadAtPosition(gq.ecsManager, pos)
}

// FindSquadsByFaction returns squads for a faction (already exists, include here)
func (gq *GUIQueries) FindSquadsByFaction(factionID ecs.EntityID) []ecs.EntityID {
    return FindSquadsByFaction(gq.ecsManager, factionID)
}

// ===== COMBAT QUERIES =====

// GetEnemySquads returns all squads not in the given faction
func (gq *GUIQueries) GetEnemySquads(currentFactionID ecs.EntityID) []ecs.EntityID {
    enemySquads := []ecs.EntityID{}
    for _, result := range gq.ecsManager.World.Query(gq.ecsManager.Tags["mapposition"]) {
        mapPos := common.GetComponentType[*combat.MapPositionData](result.Entity, combat.MapPositionComponent)
        if mapPos.FactionID != currentFactionID {
            if !squads.IsSquadDestroyed(mapPos.SquadID, gq.ecsManager) {
                enemySquads = append(enemySquads, mapPos.SquadID)
            }
        }
    }
    return enemySquads
}

// GetAllFactions returns all faction IDs
func (gq *GUIQueries) GetAllFactions() []ecs.EntityID {
    factionIDs := []ecs.EntityID{}
    for _, result := range gq.ecsManager.World.Query(gq.ecsManager.Tags["faction"]) {
        factionData := common.GetComponentType[*combat.FactionData](result.Entity, combat.FactionComponent)
        factionIDs = append(factionIDs, factionData.FactionID)
    }
    return factionIDs
}
```

*After* (combatmode.go - simplified):
```go
type CombatMode struct {
    BaseMode

    // ... existing fields ...

    queries *GUIQueries // Add query service
}

func (cm *CombatMode) Initialize(ctx *UIContext) error {
    cm.InitializeBase(ctx)

    // Initialize query service
    cm.queries = NewGUIQueries(ctx.ECSManager)

    // ... rest of initialization ...
}

// REMOVED: getFactionName() - use cm.queries.GetFactionName()
// REMOVED: isPlayerFaction() - use cm.queries.IsPlayerFaction()
// REMOVED: findActionStateEntity() - use cm.queries.GetSquadInfo()
// REMOVED: showAvailableTargets() - use cm.queries.GetEnemySquads()

func (cm *CombatMode) updateFactionDisplay() {
    currentFactionID := cm.turnManager.GetCurrentFaction()
    if currentFactionID == 0 {
        cm.factionInfoText.Label = "No faction info"
        return
    }

    // Use unified query service
    factionInfo := cm.queries.GetFactionInfo(currentFactionID)
    if factionInfo == nil {
        return
    }

    infoText := fmt.Sprintf("%s\n", factionInfo.Name)
    infoText += fmt.Sprintf("Squads: %d/%d\n", factionInfo.AliveSquadCount, len(factionInfo.SquadIDs))
    infoText += fmt.Sprintf("Mana: %d/%d", factionInfo.CurrentMana, factionInfo.MaxMana)

    cm.factionInfoText.Label = infoText
}

func (cm *CombatMode) updateSquadDetail() {
    if cm.selectedSquadID == 0 {
        cm.squadDetailText.Label = "Select a squad\nto view details"
        return
    }

    // Use unified query service
    squadInfo := cm.queries.GetSquadInfo(cm.selectedSquadID)
    if squadInfo == nil {
        return
    }

    detailText := fmt.Sprintf("%s\n", squadInfo.Name)
    detailText += fmt.Sprintf("Units: %d/%d\n", squadInfo.AliveUnits, squadInfo.TotalUnits)
    detailText += fmt.Sprintf("HP: %d/%d\n", squadInfo.CurrentHP, squadInfo.MaxHP)
    detailText += fmt.Sprintf("Move: %d\n", squadInfo.MovementRemaining)

    if squadInfo.HasActed {
        detailText += "Status: Acted\n"
    } else if squadInfo.HasMoved {
        detailText += "Status: Moved\n"
    } else {
        detailText += "Status: Ready\n"
    }

    cm.squadDetailText.Label = detailText
}
```

**Key Changes**:
1. Created `GUIQueries` service with all ECS queries
2. Defined `FactionInfo` and `SquadInfo` data transfer objects
3. Eliminated 7+ query methods from `combatmode.go`
4. Single source of truth for query logic
5. Each mode gets `queries *GUIQueries` field in `BaseMode` or individually

**Value Proposition**:
- **Maintainability**: ECS schema changes require updates in ONE file (guiqueries.go)
- **Readability**: Mode files 20-30% smaller, focus on UI logic not queries
- **Extensibility**: New modes can use existing queries without duplication
- **Complexity Impact**:
  - **Before**: 15-20 query methods scattered across 5 files (~300 LOC)
  - **After**: 1 query service (~200 LOC), modes reduced by ~100 LOC total
  - **Net**: -200 LOC, centralized query logic

**Implementation Strategy**:
1. Create `guiqueries.go` with all query functions
2. Add `queries *GUIQueries` to `BaseMode.InitializeBase()`
3. Update `combatmode.go` to use `cm.queries.*` instead of local methods
4. Update `squadmanagementmode.go`, `squaddeploymentmode.go`, `inventorymode.go`
5. Remove old query methods from mode files
6. **Optional**: Deprecate `squadqueries.go` (merge into guiqueries.go)

**Advantages**:
- **Testability**: Can test all queries independently with mock ECS
- **Performance**: Can add caching/memoization in one place
- **Consistency**: All modes use same query patterns
- **DRY**: Zero query duplication across modes
- **Type Safety**: Data transfer objects prevent accessing wrong component types

**Drawbacks & Risks**:
- **Initial Effort**: Must identify ALL queries across ALL modes (~2-3 hours)
- **Breaking Changes**: Modes must update to use new query service
- **Coupling**: All modes depend on GUIQueries (but already depend on ECSManager)
- **Mitigation**: Implement incrementally - start with combat queries, expand gradually

**Effort Estimate**:
- **Time**: 4-6 hours
- **Complexity**: Medium
- **Risk**: Low (backward compatible - can run old and new queries side-by-side)
- **Files Impacted**:
  - Create: `guiqueries.go` (~250 LOC)
  - Modify: `basemode.go` (add queries field), `combatmode.go` (-100 LOC), `squadmanagementmode.go` (-30 LOC), `squaddeploymentmode.go` (-20 LOC), `inventorymode.go` (-10 LOC)
  - **Net Change**: +250 LOC created, -160 LOC removed = +90 LOC (but much better organized)

**Critical Assessment**:
This is the **highest value refactoring** because:
1. Eliminates root cause of duplication (scattered ECS queries)
2. Improves testability and maintainability significantly
3. Low risk - can be implemented incrementally
4. Aligns with ECS best practices (query-based data access)
5. Foundation for future refactorings (update components, rendering systems)

**Recommendation**: **HIGH PRIORITY** - Implement first

---

### Approach 2: Reusable UI Update Components

**Strategic Focus**: Extract repeated UI update patterns into reusable components that modes can compose.

**Problem Statement**:
Multiple modes duplicate UI update logic:
- Squad list updates (combat, management, deployment)
- Detail panel updates (inventory, squads, combat)
- Text area refreshes (exploration stats, combat log)
- Inconsistent refresh timing (some on Enter, some on Update)

**Solution Overview**:
Create **UI Update Components** that encapsulate common update patterns. Apply **Observer Pattern** for automatic refresh when data changes.

**Code Example**:

*Before* (combatmode.go, lines 422-465):
```go
func (cm *CombatMode) updateSquadList() {
    // Clear existing buttons (keep label)
    children := cm.squadListPanel.Children()
    for len(children) > 1 {
        cm.squadListPanel.RemoveChild(children[len(children)-1])
        children = cm.squadListPanel.Children()
    }

    currentFactionID := cm.turnManager.GetCurrentFaction()
    if currentFactionID == 0 {
        return
    }

    // Only show squads if it's player's turn
    if !cm.isPlayerFaction(currentFactionID) {
        noSquadsText := widget.NewText(
            widget.TextOpts.Text("AI Turn", SmallFace, color.Gray{Y: 128}),
        )
        cm.squadListPanel.AddChild(noSquadsText)
        return
    }

    squadIDs := cm.factionManager.GetFactionSquads(currentFactionID)

    for _, squadID := range squadIDs {
        // Skip destroyed squads
        if squads.IsSquadDestroyed(squadID, cm.context.ECSManager) {
            continue
        }

        squadName := GetSquadName(cm.context.ECSManager, squadID)

        // Create button for each squad
        localSquadID := squadID
        squadButton := CreateButtonWithConfig(ButtonConfig{
            Text: squadName,
            OnClick: func() {
                cm.selectSquad(localSquadID)
            },
        })

        cm.squadListPanel.AddChild(squadButton)
    }
}
```

*Before* (squaddeploymentmode.go, similar pattern ~40 LOC):
```go
func (sdm *SquadDeploymentMode) Enter(fromMode UIMode) error {
    // Find all squads
    sdm.allSquads = FindAllSquads(sdm.context.ECSManager)
    sdm.squadNames = make([]string, 0, len(sdm.allSquads))

    // Get squad names
    for _, squadID := range sdm.allSquads {
        squadName := GetSquadName(sdm.context.ECSManager, squadID)
        sdm.squadNames = append(sdm.squadNames, squadName)
    }

    // Convert to list entries
    entries := make([]interface{}, len(sdm.squadNames))
    for i, name := range sdm.squadNames {
        entries[i] = name
    }

    sdm.squadList.SetEntries(entries)
    // ... rest
}
```

*After* (guicomponents.go - NEW FILE):
```go
package gui

import (
    "game_main/common"
    "github.com/bytearena/ecs"
    "github.com/ebitenui/ebitenui/widget"
)

// SquadListComponent manages a list widget displaying squads
type SquadListComponent struct {
    listWidget   *widget.List
    queries      *GUIQueries
    filter       SquadFilter
    onSelect     func(squadID ecs.EntityID)
}

// SquadFilter determines which squads to show
type SquadFilter func(squadInfo *SquadInfo) bool

// NewSquadListComponent creates a reusable squad list updater
func NewSquadListComponent(
    listWidget *widget.List,
    queries *GUIQueries,
    filter SquadFilter,
    onSelect func(ecs.EntityID),
) *SquadListComponent {
    return &SquadListComponent{
        listWidget: listWidget,
        queries:    queries,
        filter:     filter,
        onSelect:   onSelect,
    }
}

// Refresh updates the list with current squad data
func (slc *SquadListComponent) Refresh() {
    // Get all squads
    allSquads := slc.queries.FindAllSquads()

    // Filter squads
    filteredSquads := []ecs.EntityID{}
    for _, squadID := range allSquads {
        squadInfo := slc.queries.GetSquadInfo(squadID)
        if squadInfo != nil && slc.filter(squadInfo) {
            filteredSquads = append(filteredSquads, squadID)
        }
    }

    // Convert to list entries
    entries := make([]interface{}, len(filteredSquads))
    for i, squadID := range filteredSquads {
        entries[i] = slc.queries.GetSquadName(squadID)
    }

    // Update list widget
    slc.listWidget.SetEntries(entries)

    // Setup selection handler
    slc.listWidget.EntrySelectedEvent.AddHandler(func(args interface{}) {
        a := args.(*widget.ListEntrySelectedEventArgs)
        if squadName, ok := a.Entry.(string); ok {
            // Find squad ID by name
            for _, squadID := range filteredSquads {
                if slc.queries.GetSquadName(squadID) == squadName {
                    if slc.onSelect != nil {
                        slc.onSelect(squadID)
                    }
                    break
                }
            }
        }
    })
}

// Common filter functions
func AliveSquadsOnly(info *SquadInfo) bool {
    return !info.IsDestroyed
}

func PlayerSquadsOnly(queries *GUIQueries) SquadFilter {
    return func(info *SquadInfo) bool {
        return !info.IsDestroyed && queries.IsPlayerFaction(info.FactionID)
    }
}

func FactionSquadsOnly(factionID ecs.EntityID) SquadFilter {
    return func(info *SquadInfo) bool {
        return !info.IsDestroyed && info.FactionID == factionID
    }
}

// ===== DETAIL PANEL COMPONENT =====

// DetailPanelComponent manages a text area showing entity details
type DetailPanelComponent struct {
    textArea *widget.TextArea
    queries  *GUIQueries
    formatter DetailFormatter
}

// DetailFormatter converts entity data to display text
type DetailFormatter func(data interface{}) string

// NewDetailPanelComponent creates a reusable detail panel updater
func NewDetailPanelComponent(
    textArea *widget.TextArea,
    queries *GUIQueries,
    formatter DetailFormatter,
) *DetailPanelComponent {
    return &DetailPanelComponent{
        textArea:  textArea,
        queries:   queries,
        formatter: formatter,
    }
}

// ShowSquad displays squad details
func (dpc *DetailPanelComponent) ShowSquad(squadID ecs.EntityID) {
    squadInfo := dpc.queries.GetSquadInfo(squadID)
    if squadInfo == nil {
        dpc.textArea.SetText("Squad not found")
        return
    }

    if dpc.formatter != nil {
        dpc.textArea.SetText(dpc.formatter(squadInfo))
    } else {
        // Default formatter
        dpc.textArea.SetText(DefaultSquadFormatter(squadInfo))
    }
}

// ShowFaction displays faction details
func (dpc *DetailPanelComponent) ShowFaction(factionID ecs.EntityID) {
    factionInfo := dpc.queries.GetFactionInfo(factionID)
    if factionInfo == nil {
        dpc.textArea.SetText("Faction not found")
        return
    }

    if dpc.formatter != nil {
        dpc.textArea.SetText(dpc.formatter(factionInfo))
    } else {
        dpc.textArea.SetText(DefaultFactionFormatter(factionInfo))
    }
}

// Default formatters
func DefaultSquadFormatter(data interface{}) string {
    info := data.(*SquadInfo)
    return fmt.Sprintf("%s\n\nUnits: %d/%d\nHP: %d/%d\nMove: %d\nStatus: %s",
        info.Name,
        info.AliveUnits, info.TotalUnits,
        info.CurrentHP, info.MaxHP,
        info.MovementRemaining,
        getSquadStatus(info))
}

func DefaultFactionFormatter(data interface{}) string {
    info := data.(*FactionInfo)
    return fmt.Sprintf("%s\n\nSquads: %d/%d\nMana: %d/%d",
        info.Name,
        info.AliveSquadCount, len(info.SquadIDs),
        info.CurrentMana, info.MaxMana)
}

func getSquadStatus(info *SquadInfo) string {
    if info.HasActed {
        return "Acted"
    } else if info.HasMoved {
        return "Moved"
    } else {
        return "Ready"
    }
}
```

*After* (combatmode.go - simplified):
```go
type CombatMode struct {
    BaseMode

    // ... existing fields ...

    // Replace individual update methods with components
    squadListUpdater   *SquadListComponent
    squadDetailUpdater *DetailPanelComponent
    factionInfoUpdater *DetailPanelComponent
}

func (cm *CombatMode) Initialize(ctx *UIContext) error {
    cm.InitializeBase(ctx)

    // ... build UI widgets (squadList, squadDetailText, factionInfoText) ...

    // Create update components
    cm.squadListUpdater = NewSquadListComponent(
        cm.squadList,
        cm.queries,
        PlayerSquadsOnly(cm.queries), // Only show player squads
        func(squadID ecs.EntityID) {
            cm.selectSquad(squadID)
        },
    )

    cm.squadDetailUpdater = NewDetailPanelComponent(
        cm.squadDetailTextArea,
        cm.queries,
        DefaultSquadFormatter,
    )

    cm.factionInfoUpdater = NewDetailPanelComponent(
        cm.factionInfoTextArea,
        cm.queries,
        DefaultFactionFormatter,
    )

    // ... rest of initialization ...
}

// REMOVED: updateSquadList() - replaced by squadListUpdater.Refresh()
// REMOVED: updateSquadDetail() - replaced by squadDetailUpdater.ShowSquad()
// REMOVED: updateFactionDisplay() - replaced by factionInfoUpdater.ShowFaction()

func (cm *CombatMode) selectSquad(squadID ecs.EntityID) {
    cm.selectedSquadID = squadID
    cm.inAttackMode = false
    cm.inMoveMode = false
    cm.selectedTargetID = 0

    cm.addCombatLog(fmt.Sprintf("Selected: %s", cm.queries.GetSquadName(squadID)))

    // Update detail panel
    cm.squadDetailUpdater.ShowSquad(squadID)
}

func (cm *CombatMode) handleEndTurn() {
    // ... end turn logic ...

    // Refresh UI components
    currentFactionID := cm.turnManager.GetCurrentFaction()
    cm.squadListUpdater.Refresh()
    cm.factionInfoUpdater.ShowFaction(currentFactionID)
}

func (cm *CombatMode) Update(deltaTime float64) error {
    // Update displays each frame
    cm.updateTurnDisplay() // Still custom logic

    // Refresh components
    currentFactionID := cm.turnManager.GetCurrentFaction()
    cm.factionInfoUpdater.ShowFaction(currentFactionID)

    if cm.selectedSquadID != 0 {
        cm.squadDetailUpdater.ShowSquad(cm.selectedSquadID)
    }

    return nil
}
```

*After* (squaddeploymentmode.go - simplified):
```go
type SquadDeploymentMode struct {
    BaseMode

    squadListUpdater *SquadListComponent
}

func (sdm *SquadDeploymentMode) Initialize(ctx *UIContext) error {
    sdm.InitializeBase(ctx)

    // ... build squad list widget ...

    // Create update component
    sdm.squadListUpdater = NewSquadListComponent(
        sdm.squadList,
        sdm.queries,
        AliveSquadsOnly, // Show all alive squads
        func(squadID ecs.EntityID) {
            sdm.selectedSquadID = squadID
            sdm.isPlacingSquad = true
            sdm.updateInstructionText()
        },
    )

    // ... rest of initialization ...
}

func (sdm *SquadDeploymentMode) Enter(fromMode UIMode) error {
    // Refresh squad list
    sdm.squadListUpdater.Refresh()
    return nil
}
```

**Key Changes**:
1. Created `SquadListComponent` - reusable squad list updater with filters
2. Created `DetailPanelComponent` - reusable detail display with custom formatters
3. Eliminated 3 update methods from combatmode.go (~120 LOC)
4. Eliminated refresh logic from squaddeploymentmode.go (~40 LOC)
5. Modes compose components instead of implementing updates

**Value Proposition**:
- **Maintainability**: Update logic in ONE place, used by ALL modes
- **Readability**: Modes declare "what to show" not "how to show it"
- **Extensibility**: New modes get squad lists/detail panels for free
- **Complexity Impact**:
  - **Before**: ~200 LOC of update logic scattered across 5 modes
  - **After**: ~150 LOC in guicomponents.go, ~50 LOC in modes (component creation)
  - **Net**: -50 LOC, better organized, reusable

**Implementation Strategy**:
1. Create `guicomponents.go` with SquadListComponent and DetailPanelComponent
2. Update `combatmode.go` to use components (pilot implementation)
3. Update `squaddeploymentmode.go` to use SquadListComponent
4. Update `squadmanagementmode.go` to use components
5. Consider additional components:
   - `TextDisplayComponent` for simple text updates (exploration stats)
   - `LogComponent` for combat log with auto-trimming
   - `GridDisplayComponent` for 3x3 squad visualization

**Advantages**:
- **Consistency**: All modes update UI the same way
- **Testability**: Can test components independently
- **Flexibility**: Custom filters and formatters for each mode's needs
- **Performance**: Components can optimize updates (e.g., only refresh if data changed)

**Drawbacks & Risks**:
- **Abstraction Cost**: Another layer between modes and widgets
- **Flexibility vs Simplicity**: Components might not fit all use cases
- **Learning Curve**: Team needs to understand component pattern
- **Mitigation**:
  - Keep components simple and well-documented
  - Allow modes to bypass components for truly custom behavior
  - Start with common patterns (squad lists, detail panels)

**Effort Estimate**:
- **Time**: 6-8 hours
- **Complexity**: Medium-High
- **Risk**: Medium (requires refactoring multiple modes)
- **Files Impacted**:
  - Create: `guicomponents.go` (~200 LOC)
  - Modify: `combatmode.go` (-120 LOC, +50 LOC component usage), `squaddeploymentmode.go` (-40 LOC, +20 LOC), `squadmanagementmode.go` (-30 LOC, +15 LOC)
  - **Net Change**: +200 LOC created, -135 LOC removed = +65 LOC (better organized)

**Critical Assessment**:
This refactoring provides **high value** but requires:
1. Careful design of component interfaces
2. Multiple mode updates (more churn than Approach 1)
3. Team buy-in on component pattern

**Benefits**:
- Solves UI update duplication
- Natural evolution from current ButtonConfig/PanelBuilders pattern
- Foundation for more sophisticated UI (e.g., auto-refresh when data changes)

**Concerns**:
- More invasive than Approach 1 (changes more files)
- Risk of over-abstraction if components become too complex
- Depends on Approach 1 (needs GUIQueries first)

**Recommendation**: **MEDIUM PRIORITY** - Implement after Approach 1

---

### Approach 3: Extract Rendering Systems

**Strategic Focus**: Separate rendering concerns from mode logic into dedicated rendering systems.

**Problem Statement**:
Rendering logic is embedded in mode files:
- `combatmode.go` has 150 LOC of rendering (movement tiles, squad highlights)
- Viewport creation duplicated across rendering methods
- Rendering concerns mixed with combat logic
- Cannot reuse rendering logic across modes (e.g., squad highlighting in deployment mode)

**Solution Overview**:
Create **Rendering Systems** that modes can call to render game overlays. Apply **Strategy Pattern** for pluggable rendering behaviors.

**Code Example**:

*Before* (combatmode.go, lines 638-668):
```go
func (cm *CombatMode) renderMovementTiles(screen *ebiten.Image) {
    // Get player position for viewport centering
    playerPos := *cm.context.PlayerData.Pos

    // Create viewport centered on player using the initialized ScreenInfo
    screenData := graphics.ScreenInfo
    screenData.ScreenWidth = screen.Bounds().Dx()
    screenData.ScreenHeight = screen.Bounds().Dy()

    manager := coords.NewCoordinateManager(screenData)
    viewport := coords.NewViewport(manager, playerPos)

    tileSize := screenData.TileSize
    scaleFactor := screenData.ScaleFactor

    // Create a semi-transparent green overlay for valid tiles
    for _, pos := range cm.validMoveTiles {
        screenX, screenY := viewport.LogicalToScreen(pos)
        scaledTileSize := tileSize * scaleFactor
        rect := ebiten.NewImage(scaledTileSize, scaledTileSize)
        rect.Fill(color.RGBA{R: 0, G: 255, B: 0, A: 80})

        op := &ebiten.DrawImageOptions{}
        op.GeoM.Translate(screenX, screenY)
        screen.DrawImage(rect, op)
    }
}
```

*Before* (combatmode.go, lines 670-745 - similar duplication for squad highlights, 76 LOC):
```go
func (cm *CombatMode) renderAllSquadHighlights(screen *ebiten.Image) {
    playerPos := *cm.context.PlayerData.Pos

    screenData := graphics.ScreenInfo
    screenData.ScreenWidth = screen.Bounds().Dx()
    screenData.ScreenHeight = screen.Bounds().Dy()

    manager := coords.NewCoordinateManager(screenData)
    viewport := coords.NewViewport(manager, playerPos)

    // ... viewport setup duplicated ...

    // Query all squads on map
    for _, result := range cm.context.ECSManager.World.Query(cm.context.ECSManager.Tags["mapposition"]) {
        mapPosData := common.GetComponentType[*combat.MapPositionData](result.Entity, combat.MapPositionComponent)

        // ... determine highlight color ...
        // ... draw 4 border rectangles ...
    }
}
```

*After* (guirenderers.go - NEW FILE):
```go
package gui

import (
    "game_main/coords"
    "game_main/graphics"
    "github.com/bytearena/ecs"
    "github.com/hajimehoshi/ebiten/v2"
    "image/color"
)

// ViewportRenderer provides viewport-centered rendering utilities
type ViewportRenderer struct {
    screenData graphics.ScreenData
    viewport   *coords.Viewport
}

// NewViewportRenderer creates a renderer for the current screen
func NewViewportRenderer(screen *ebiten.Image, centerPos coords.LogicalPosition) *ViewportRenderer {
    screenData := graphics.ScreenInfo
    screenData.ScreenWidth = screen.Bounds().Dx()
    screenData.ScreenHeight = screen.Bounds().Dy()

    manager := coords.NewCoordinateManager(screenData)
    viewport := coords.NewViewport(manager, centerPos)

    return &ViewportRenderer{
        screenData: screenData,
        viewport:   viewport,
    }
}

// TileSize returns the scaled tile size
func (vr *ViewportRenderer) TileSize() int {
    return vr.screenData.TileSize * vr.screenData.ScaleFactor
}

// LogicalToScreen converts logical position to screen coordinates
func (vr *ViewportRenderer) LogicalToScreen(pos coords.LogicalPosition) (float64, float64) {
    return vr.viewport.LogicalToScreen(pos)
}

// DrawTileOverlay draws a colored rectangle at a logical position
func (vr *ViewportRenderer) DrawTileOverlay(screen *ebiten.Image, pos coords.LogicalPosition, fillColor color.Color) {
    screenX, screenY := vr.LogicalToScreen(pos)
    tileSize := vr.TileSize()

    rect := ebiten.NewImage(tileSize, tileSize)
    rect.Fill(fillColor)

    op := &ebiten.DrawImageOptions{}
    op.GeoM.Translate(screenX, screenY)
    screen.DrawImage(rect, op)
}

// DrawTileBorder draws a colored border around a logical position
func (vr *ViewportRenderer) DrawTileBorder(screen *ebiten.Image, pos coords.LogicalPosition, borderColor color.Color, thickness int) {
    screenX, screenY := vr.LogicalToScreen(pos)
    tileSize := vr.TileSize()

    // Top border
    topBorder := ebiten.NewImage(tileSize, thickness)
    topBorder.Fill(borderColor)
    op := &ebiten.DrawImageOptions{}
    op.GeoM.Translate(screenX, screenY)
    screen.DrawImage(topBorder, op)

    // Bottom border
    bottomBorder := ebiten.NewImage(tileSize, thickness)
    bottomBorder.Fill(borderColor)
    op = &ebiten.DrawImageOptions{}
    op.GeoM.Translate(screenX, screenY+float64(tileSize-thickness))
    screen.DrawImage(bottomBorder, op)

    // Left border
    leftBorder := ebiten.NewImage(thickness, tileSize)
    leftBorder.Fill(borderColor)
    op = &ebiten.DrawImageOptions{}
    op.GeoM.Translate(screenX, screenY)
    screen.DrawImage(leftBorder, op)

    // Right border
    rightBorder := ebiten.NewImage(thickness, tileSize)
    rightBorder.Fill(borderColor)
    op = &ebiten.DrawImageOptions{}
    op.GeoM.Translate(screenX+float64(tileSize-thickness), screenY)
    screen.DrawImage(rightBorder, op)
}

// ===== COMBAT RENDERING SYSTEMS =====

// MovementTileRenderer renders valid movement tiles
type MovementTileRenderer struct {
    fillColor color.Color
}

// NewMovementTileRenderer creates a renderer for movement tiles
func NewMovementTileRenderer() *MovementTileRenderer {
    return &MovementTileRenderer{
        fillColor: color.RGBA{R: 0, G: 255, B: 0, A: 80}, // Semi-transparent green
    }
}

// Render draws all valid movement tiles
func (mtr *MovementTileRenderer) Render(screen *ebiten.Image, centerPos coords.LogicalPosition, validTiles []coords.LogicalPosition) {
    vr := NewViewportRenderer(screen, centerPos)

    for _, pos := range validTiles {
        vr.DrawTileOverlay(screen, pos, mtr.fillColor)
    }
}

// SquadHighlightRenderer renders squad position highlights
type SquadHighlightRenderer struct {
    queries           *GUIQueries
    selectedColor     color.Color
    playerColor       color.Color
    enemyColor        color.Color
    borderThickness   int
}

// NewSquadHighlightRenderer creates a renderer for squad highlights
func NewSquadHighlightRenderer(queries *GUIQueries) *SquadHighlightRenderer {
    return &SquadHighlightRenderer{
        queries:         queries,
        selectedColor:   color.RGBA{R: 255, G: 255, B: 255, A: 255}, // White
        playerColor:     color.RGBA{R: 0, G: 150, B: 255, A: 150},   // Blue
        enemyColor:      color.RGBA{R: 255, G: 0, B: 0, A: 150},     // Red
        borderThickness: 3,
    }
}

// Render draws highlights for all squads
func (shr *SquadHighlightRenderer) Render(
    screen *ebiten.Image,
    centerPos coords.LogicalPosition,
    currentFactionID ecs.EntityID,
    selectedSquadID ecs.EntityID,
) {
    vr := NewViewportRenderer(screen, centerPos)

    // Get all squads with positions
    allSquads := shr.queries.FindAllSquads()

    for _, squadID := range allSquads {
        squadInfo := shr.queries.GetSquadInfo(squadID)
        if squadInfo == nil || squadInfo.IsDestroyed || squadInfo.Position == nil {
            continue
        }

        // Determine highlight color
        var highlightColor color.Color
        if squadID == selectedSquadID {
            highlightColor = shr.selectedColor
        } else if squadInfo.FactionID == currentFactionID {
            highlightColor = shr.playerColor
        } else {
            highlightColor = shr.enemyColor
        }

        // Draw border
        vr.DrawTileBorder(screen, *squadInfo.Position, highlightColor, shr.borderThickness)
    }
}
```

*After* (combatmode.go - simplified):
```go
type CombatMode struct {
    BaseMode

    // ... existing fields ...

    // Rendering systems
    movementRenderer  *MovementTileRenderer
    highlightRenderer *SquadHighlightRenderer
}

func (cm *CombatMode) Initialize(ctx *UIContext) error {
    cm.InitializeBase(ctx)

    // ... other initialization ...

    // Create rendering systems
    cm.movementRenderer = NewMovementTileRenderer()
    cm.highlightRenderer = NewSquadHighlightRenderer(cm.queries)

    return nil
}

// REMOVED: renderMovementTiles() - replaced by movementRenderer.Render()
// REMOVED: renderAllSquadHighlights() - replaced by highlightRenderer.Render()

func (cm *CombatMode) Render(screen *ebiten.Image) {
    playerPos := *cm.context.PlayerData.Pos
    currentFactionID := cm.turnManager.GetCurrentFaction()

    // Render squad highlights (always shown)
    cm.highlightRenderer.Render(screen, playerPos, currentFactionID, cm.selectedSquadID)

    // Render valid movement tiles (only in move mode)
    if cm.inMoveMode && len(cm.validMoveTiles) > 0 {
        cm.movementRenderer.Render(screen, playerPos, cm.validMoveTiles)
    }
}
```

*After* (squaddeploymentmode.go - REUSES rendering):
```go
type SquadDeploymentMode struct {
    BaseMode

    highlightRenderer *SquadHighlightRenderer
}

func (sdm *SquadDeploymentMode) Initialize(ctx *UIContext) error {
    sdm.InitializeBase(ctx)

    // Create rendering system
    sdm.highlightRenderer = NewSquadHighlightRenderer(sdm.queries)

    // ... rest of initialization ...
}

func (sdm *SquadDeploymentMode) Render(screen *ebiten.Image) {
    playerPos := *sdm.context.PlayerData.Pos

    // Reuse squad highlighting from combat mode
    sdm.highlightRenderer.Render(screen, playerPos, 0, sdm.selectedSquadID)
}
```

**Key Changes**:
1. Created `ViewportRenderer` - reusable viewport utilities
2. Created `MovementTileRenderer` - renders movement overlays
3. Created `SquadHighlightRenderer` - renders squad borders
4. Eliminated 150 LOC of rendering from combatmode.go
5. Modes call renderers instead of implementing rendering

**Value Proposition**:
- **Maintainability**: Rendering logic in ONE place
- **Readability**: Mode Render() methods are 5-10 LOC instead of 50-100 LOC
- **Extensibility**: New modes get rendering systems for free (e.g., deployment mode)
- **Complexity Impact**:
  - **Before**: ~150 LOC of rendering in combatmode.go, not reusable
  - **After**: ~200 LOC in guirenderers.go, reusable across modes
  - **Net**: +50 LOC but much better organized and reusable

**Implementation Strategy**:
1. Create `guirenderers.go` with ViewportRenderer base
2. Extract MovementTileRenderer from combatmode.go
3. Extract SquadHighlightRenderer from combatmode.go
4. Update combatmode.go to use renderers
5. Add rendering to squaddeploymentmode.go (reuse SquadHighlightRenderer)
6. Consider additional renderers:
   - `TargetingRenderer` for showing attack ranges
   - `GridRenderer` for formation editor overlays
   - `AOERenderer` for throwable/ability shapes

**Advantages**:
- **Separation of Concerns**: Modes handle logic, renderers handle visuals
- **Testability**: Can test rendering independently (mock screen, check draw calls)
- **Reusability**: Squad highlighting works in combat, deployment, and management modes
- **Performance**: Renderers can optimize (e.g., batch draws, cache images)

**Drawbacks & Risks**:
- **Complexity**: Another abstraction layer
- **Coupling**: Renderers depend on GUIQueries (but already needed for data)
- **Overkill**: Only 2-3 modes currently render custom graphics
- **Mitigation**:
  - Start with common patterns (movement tiles, highlights)
  - Allow modes to render custom graphics directly if needed
  - Keep renderers simple and focused

**Effort Estimate**:
- **Time**: 4-5 hours
- **Complexity**: Medium
- **Risk**: Low (rendering is isolated, easy to test visually)
- **Files Impacted**:
  - Create: `guirenderers.go` (~200 LOC)
  - Modify: `combatmode.go` (-150 LOC rendering, +30 LOC renderer usage), `squaddeploymentmode.go` (+20 LOC renderer usage)
  - **Net Change**: +200 LOC created, -100 LOC removed = +100 LOC (better organized)

**Critical Assessment**:
This refactoring provides **moderate value**:
1. Solves rendering duplication in combat mode
2. Enables reuse in deployment mode (currently has no rendering)
3. **BUT**: Only affects 1-2 modes currently (limited impact)
4. **AND**: Depends on Approach 1 (needs GUIQueries for squad data)

**Benefits**:
- Clean separation of rendering from mode logic
- Foundation for more sophisticated rendering (AOE shapes, threat ranges)
- Testable rendering (can verify draw calls)

**Concerns**:
- Lower impact than Approaches 1 & 2 (fewer modes benefit)
- Risk of over-engineering if only combat mode uses custom rendering
- Depends on Approach 1 being implemented first

**Recommendation**: **LOW PRIORITY** - Implement after Approaches 1 & 2, or when additional modes need custom rendering

---

## COMPARATIVE ANALYSIS OF FINAL APPROACHES

### Effort vs Impact Matrix

| Approach | Effort | Impact | Risk | Recommended Priority |
|----------|--------|--------|------|---------------------|
| **Approach 1: Unified ECS Query Layer** | Medium (4-6h) | **High** | Low | **1 - HIGHEST** |
| **Approach 2: Reusable UI Update Components** | High (6-8h) | **High** | Medium | **2 - HIGH** |
| **Approach 3: Extract Rendering Systems** | Medium (4-5h) | **Medium** | Low | **3 - MEDIUM** |

### Decision Guidance

**Choose Approach 1 if:**
- You want immediate, high-value improvement with low risk
- You're concerned about ECS query duplication across modes
- You plan to add new UI modes in the future
- You want a foundation for other refactorings
- **Recommended Context**: ANY project using the GUI package

**Choose Approach 2 if:**
- You've already implemented Approach 1 (Approach 2 depends on GUIQueries)
- You want to eliminate UI update duplication
- You're building more UI modes with squad/faction displays
- You're comfortable with component-based patterns
- **Recommended Context**: After Approach 1, when adding 2+ new modes

**Choose Approach 3 if:**
- You've implemented Approaches 1 & 2
- Multiple modes need custom rendering (currently only combat mode)
- You're adding gameplay features requiring overlays (AOE, threat ranges)
- Rendering performance is a concern
- **Recommended Context**: After Approaches 1 & 2, when adding rendering to 2+ modes

### Combination Opportunities

**Recommended Implementation Order**:

1. **Phase 1: Foundation** (Week 1)
   - Implement Approach 1 (Unified ECS Query Layer)
   - Immediate 20% reduction in mode complexity
   - Enables Approaches 2 & 3

2. **Phase 2: UI Components** (Week 2)
   - Implement Approach 2 (Reusable UI Update Components)
   - Additional 15% reduction in mode complexity
   - Natural evolution of ButtonConfig/PanelBuilders pattern

3. **Phase 3: Rendering** (Week 3 - Optional)
   - Implement Approach 3 (Extract Rendering Systems)
   - Only if 2+ modes need custom rendering
   - Deferred if low priority

**Combined Benefits** (All 3 Approaches):
- **LOC Reduction**: -200 to -250 lines from mode files
- **Complexity Reduction**: 40-50% reduction in mode file complexity
- **Maintainability**: ECS changes require 1 file update (guiqueries.go) instead of 5+
- **Reusability**: New modes compose from queries, components, renderers
- **Testability**: Can test queries, components, renderers independently

**Trade-offs**:
- **Initial Effort**: 14-19 hours total (spread over 2-3 weeks)
- **Abstraction Layers**: +3 new systems (queries, components, renderers)
- **Learning Curve**: Team must understand new patterns
- **Payoff**: Compounds over time - each new mode is 50% easier to implement

---

## APPENDIX: ADDITIONAL OBSERVATIONS

### Strengths to Preserve

1. **UIMode Interface** (uimode.go, lines 11-40)
   - Excellent lifecycle contract: Initialize → Enter → Update/Render/HandleInput → Exit
   - Clear separation of concerns (initialization vs. activation vs. updates)
   - **Recommendation**: Keep as-is, foundational design

2. **BaseMode Pattern** (basemode.go)
   - DRY achieved through composition (embedding) not inheritance
   - Common infrastructure (ui, context, layout, panelBuilders) shared automatically
   - HandleCommonInput() provides consistent ESC behavior
   - **Recommendation**: Keep as-is, exemplifies good Go design

3. **Functional Options Pattern** (panelconfig.go)
   - Modern Go pattern applied successfully
   - Composable, self-documenting panel configuration
   - Example: `BuildPanel(TopCenter(), Size(0.4, 0.08), Padding(0.01))`
   - **Recommendation**: Expand this pattern to other widget creation (buttons, lists)

4. **ButtonConfig Pattern** (createwidgets.go, lines 57-118)
   - Recently applied (2025-11-07), successfully eliminates button creation boilerplate
   - Declarative configuration: Text, OnClick, MinWidth, FontFace, etc.
   - Consistent across all modes
   - **Recommendation**: Apply similar pattern to more widget types (see below)

5. **PanelBuilders Abstraction** (panels.go)
   - High-level builders reduce duplication: BuildSquadListPanel, BuildDetailPanel, BuildGridEditor
   - Encapsulates common UI patterns
   - **Recommendation**: Evolve into Approach 2 (UI Update Components)

### Additional Refactoring Opportunities (Not Covered in Main Approaches)

#### 1. Expand Config Patterns to All Widgets

**Current State**:
- ButtonConfig ✅ (implemented)
- PanelConfig ✅ (implemented)
- ListConfig ✅ (implemented)
- TextAreaConfig ✅ (implemented)

**Missing**:
- TextInputConfig (for squad names in builder)
- SliderConfig (if adding settings)
- CheckboxConfig (if adding toggles)

**Recommendation**:
- Apply ButtonConfig pattern to TextInput widgets
- Keep pattern consistent across all widget types

**Effort**: 1-2 hours per widget type
**Priority**: Low (only implement when adding new widget types)

---

#### 2. Mode State Persistence

**Current Issue**:
Some modes lose state when exiting:
- CombatMode clears combat log on Exit (line 569-572)
- SquadManagementMode rebuilds all panels on Enter (lines 72-88)
- FormationEditorMode loses grid state

**Potential Solution**:
```go
// Add to BaseMode
type ModeState interface {
    Save() interface{}
    Restore(state interface{})
}

// Modes can optionally implement state persistence
func (cm *CombatMode) Save() interface{} {
    return &CombatModeState{
        CombatLog: cm.combatLog,
        SelectedSquadID: cm.selectedSquadID,
    }
}

func (cm *CombatMode) Restore(state interface{}) {
    if savedState, ok := state.(*CombatModeState); ok {
        cm.combatLog = savedState.CombatLog
        cm.selectedSquadID = savedState.SelectedSquadID
    }
}
```

**Recommendation**:
- **NOT RECOMMENDED** for now - adds complexity without clear benefit
- Most modes SHOULD refresh on Enter (inventory, squad management)
- Combat log clearing is intentional (new battle)
- Only implement if users request "return to previous mode state"

**Effort**: 3-4 hours
**Priority**: Very Low (YAGNI - implement only if requested)

---

#### 3. Input Abstraction

**Current State**:
- InputState captured in UIModeManager (modemanager.go, lines 124-163)
- Each mode implements HandleInput() differently
- Some duplication in hotkey handling

**Example Duplication**:
```go
// ExplorationMode (lines 222-252)
if inputState.KeysJustPressed[ebiten.KeyE] {
    if squadMode, exists := em.modeManager.GetMode("squad_management"); exists {
        em.modeManager.RequestTransition(squadMode, "E key pressed")
    }
}

// SquadManagementMode (lines 228-233)
if inputState.KeysJustPressed[ebiten.KeyE] {
    if exploreMode, exists := smm.modeManager.GetMode("exploration"); exists {
        smm.modeManager.RequestTransition(exploreMode, "E key pressed")
    }
}
```

**Potential Solution**:
```go
// InputBinding maps keys to mode transitions
type InputBinding struct {
    Key        ebiten.Key
    TargetMode string
    Reason     string
}

// Add to BaseMode
func (bm *BaseMode) RegisterHotkey(key ebiten.Key, targetMode string) {
    bm.hotkeys[key] = InputBinding{
        Key:        key,
        TargetMode: targetMode,
        Reason:     fmt.Sprintf("%s key pressed", key),
    }
}

// BaseMode.HandleCommonInput checks hotkeys automatically
func (bm *BaseMode) HandleCommonInput(inputState *InputState) bool {
    // Check registered hotkeys
    for key, binding := range bm.hotkeys {
        if inputState.KeysJustPressed[key] {
            if targetMode, exists := bm.modeManager.GetMode(binding.TargetMode); exists {
                bm.modeManager.RequestTransition(targetMode, binding.Reason)
                return true
            }
        }
    }
    // ... ESC handling ...
}
```

**Recommendation**:
- **MAYBE** - reduces hotkey duplication across modes
- **BUT**: Only saves ~5 lines per mode, limited value
- **Alternative**: Document hotkey convention in UIMode interface comments

**Effort**: 2-3 hours
**Priority**: Low (nice-to-have, not critical)

---

#### 4. ECS Violation: findActionStateEntity Returns *ecs.Entity

**Issue** (combatmode.go, line 523):
```go
func (cm *CombatMode) findActionStateEntity(squadID ecs.EntityID) *ecs.Entity {
    for _, result := range cm.context.ECSManager.World.Query(cm.context.ECSManager.Tags["actionstate"]) {
        actionState := common.GetComponentType[*combat.ActionStateData](result.Entity, combat.ActionStateComponent)
        if actionState.SquadID == squadID {
            return result.Entity  // ❌ Returns entity pointer
        }
    }
    return nil
}
```

**ECS Best Practice**: Use EntityID, not entity pointers

**Solution** (Approach 1 solves this):
```go
// In GUIQueries (Approach 1)
func (gq *GUIQueries) GetSquadInfo(squadID ecs.EntityID) *SquadInfo {
    // ... other queries ...

    // Get action state (no entity pointer returned)
    hasActed := false
    hasMoved := false
    movementRemaining := 0
    for _, result := range gq.ecsManager.World.Query(gq.ecsManager.Tags["actionstate"]) {
        actionState := common.GetComponentType[*combat.ActionStateData](result.Entity, combat.ActionStateComponent)
        if actionState.SquadID == squadID {
            hasActed = actionState.HasActed
            hasMoved = actionState.HasMoved
            movementRemaining = actionState.MovementRemaining
            break
        }
    }

    return &SquadInfo{
        // ... other fields ...
        HasActed: hasActed,
        HasMoved: hasMoved,
        MovementRemaining: movementRemaining,
    }
}
```

**Recommendation**:
- Approach 1 naturally fixes this ECS violation
- No separate effort needed

---

#### 5. Mode Registration Verbosity

**Current State** (likely in game_main/main.go):
```go
modeManager := gui.NewUIModeManager(uiContext)

explorationMode := gui.NewExplorationMode(modeManager)
modeManager.RegisterMode(explorationMode)

combatMode := gui.NewCombatMode(modeManager)
modeManager.RegisterMode(combatMode)

squadMode := gui.NewSquadManagementMode(modeManager)
modeManager.RegisterMode(squadMode)

// ... 6+ more modes ...
```

**Potential Solution**:
```go
// Add to UIModeManager
func (umm *UIModeManager) RegisterModes(modeFactories ...func(*UIModeManager) UIMode) error {
    for _, factory := range modeFactories {
        mode := factory(umm)
        if err := umm.RegisterMode(mode); err != nil {
            return err
        }
    }
    return nil
}

// Usage
modeManager.RegisterModes(
    gui.NewExplorationMode,
    gui.NewCombatMode,
    gui.NewSquadManagementMode,
    gui.NewInventoryMode,
    gui.NewSquadBuilderMode,
    gui.NewSquadDeploymentMode,
    gui.NewFormationEditorMode,
    gui.NewInfoMode,
)
```

**Recommendation**:
- Nice-to-have, but minimal value (saves ~8 lines)
- Mode registration happens once at startup
- **SKIP** - not worth the effort

**Effort**: 30 minutes
**Priority**: Very Low (cosmetic improvement)

---

## SUMMARY & NEXT STEPS

### Quick Wins (Implement Now)

1. **Approach 1: Unified ECS Query Layer** ⭐ **HIGHEST PRIORITY**
   - 4-6 hours effort
   - Immediate 20% complexity reduction
   - Enables all other refactorings
   - Low risk, high value

### Medium-Term Improvements (Next Sprint)

2. **Approach 2: Reusable UI Update Components**
   - 6-8 hours effort
   - Requires Approach 1 first
   - 15% additional complexity reduction
   - Medium risk, high value

### Long-Term Enhancements (Future)

3. **Approach 3: Extract Rendering Systems**
   - 4-5 hours effort
   - Requires Approach 1 first
   - 10% complexity reduction
   - Low risk, medium value
   - Implement when 2+ modes need custom rendering

### Recommended Action Plan

**Week 1: Foundation**
1. Implement Approach 1 (GUIQueries)
2. Update combatmode.go to use queries
3. Update 2-3 other modes to use queries
4. Remove old query methods

**Week 2: Components**
1. Implement Approach 2 (UI Update Components)
2. Update combatmode.go to use components
3. Update squaddeploymentmode.go to use components
4. Remove old update methods

**Week 3: Optional Polish**
1. Evaluate if Approach 3 is needed
2. If yes, implement rendering systems
3. If no, consider additional refactorings from Appendix

### Validation Strategy

**Testing Approach**:
1. **Unit Tests**: Test GUIQueries independently with mock ECS
2. **Integration Tests**: Verify modes still function after refactoring
3. **Visual Tests**: Manually verify UI displays correctly
4. **Regression Tests**: Ensure no features broken

**Rollback Plan**:
- Git branch for each approach
- Keep old query methods during transition (mark deprecated)
- Can run old and new code side-by-side

**Success Metrics**:
- **LOC Reduction**: Target -200 lines from mode files
- **Complexity**: Fewer methods per mode (target 8-10, down from 15-20)
- **Duplication**: Zero query duplication across modes
- **Testability**: All query logic independently testable

---

## CONCLUSION

The GUI package demonstrates **strong architectural foundations** with the mode system, BaseMode pattern, and functional options. Recent work (ButtonConfig pattern, 2025-11-07) shows the team is actively improving code quality.

**Key Strengths**:
- Excellent UIMode interface and lifecycle
- Successful application of DRY through BaseMode
- Modern Go patterns (functional options, config structs)

**Key Weaknesses**:
- ECS query duplication across 5+ files
- UI update logic scattered and duplicated
- Rendering mixed with mode logic

**Recommended Path Forward**:
1. **Immediate**: Implement Approach 1 (Unified ECS Query Layer) - 4-6 hours
2. **Next Sprint**: Implement Approach 2 (Reusable UI Update Components) - 6-8 hours
3. **Future**: Consider Approach 3 (Rendering Systems) when needed

**Expected Outcome**:
- **15-20% LOC reduction** in mode files
- **40% complexity reduction** (fewer methods, clearer responsibilities)
- **100% query reusability** (zero duplication)
- **Foundation for future modes** (compose from queries, components, renderers)

The refactoring is **incremental, low-risk, and high-value**. Each approach can be implemented independently with immediate benefits.

---

END OF ANALYSIS
