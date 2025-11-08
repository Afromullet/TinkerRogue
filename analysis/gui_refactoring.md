
### Approach 2: Shared Query Service with Practical Scope

**Strategic Focus**: "DRY Where It Matters, Avoid Service Layer Bloat"

**Problem Statement**:
The same ECS query patterns are repeated across multiple mode files:

1. **findAllSquads()** - Duplicated in:
   - `squadmanagementmode.go` (lines 121-136)
   - `squadbuilder.go` (lines 491-502)
   - Implicit in `squaddeploymentmode.go` (lines 183-187)

2. **getSquadName()** - Duplicated in:
   - `combatmode.go` (lines 372-380)
   - `squaddeploymentmode.go` (lines 165-173)
   - Similar logic in 2+ other modes

3. **Panel creation patterns** - Duplicated:
   - Close button creation in 5+ modes
   - Action button container setup in 4+ modes

This violates DRY (Don't Repeat Yourself) and creates maintenance burden:
- Bug fix in one query needs to be copied to all locations
- Inconsistent behavior (one mode formats squad names differently)
- More code to understand when reading a mode

**Solution Overview**:
Extract commonly duplicated queries and UI helpers into shared utility files. Scope is limited to what's actually duplicated 3+ times - avoiding over-abstraction.

**Code Example**:

*Before (Duplication across modes):*
```go
// squadmanagementmode.go
func (smm *SquadManagementMode) findAllSquads() []ecs.EntityID {
    allSquads := make([]ecs.EntityID, 0)
    entityIDs := smm.context.ECSManager.GetAllEntities()
    for _, entityID := range entityIDs {
        if smm.context.ECSManager.HasComponent(entityID, squads.SquadComponent) {
            allSquads = append(allSquads, entityID)
        }
    }
    return allSquads
}

// squadbuilder.go
func (sbm *SquadBuilderMode) findAllSquads() []ecs.EntityID {
    allSquads := make([]ecs.EntityID, 0)
    entityIDs := sbm.context.ECSManager.GetAllEntities()
    for _, entityID := range entityIDs {
        if sbm.context.ECSManager.HasComponent(entityID, squads.SquadComponent) {
            allSquads = append(allSquads, entityID)
        }
    }
    return allSquads
}

// squaddeploymentmode.go - slightly different approach!
for _, result := range sdm.context.ECSManager.World.Query(
    sdm.context.ECSManager.Tags["squad"]) {
    squadData := common.GetComponentType[*squads.SquadData](result.Entity, squads.SquadComponent)
    sdm.allSquads = append(sdm.allSquads, squadData.SquadID)
}
```

```go
// combatmode.go
func (cm *CombatMode) getSquadName(squadID ecs.EntityID) string {
    for _, result := range cm.context.ECSManager.World.Query(
        cm.context.ECSManager.Tags["squad"]) {
        squadData := common.GetComponentType[*squads.SquadData](
            result.Entity, squads.SquadComponent)
        if squadData.SquadID == squadID {
            return squadData.Name
        }
    }
    return "Unknown Squad"
}

// squaddeploymentmode.go
func (sdm *SquadDeploymentMode) getSquadName(squadID ecs.EntityID) string {
    for _, result := range sdm.context.ECSManager.World.Query(
        sdm.context.ECSManager.Tags["squad"]) {
        squadData := common.GetComponentType[*squads.SquadData](
            result.Entity, squads.SquadComponent)
        if squadData.SquadID == squadID {
            return squadData.Name
        }
    }
    return "Unknown Squad"
}
```

*After (Shared utilities):*

```go
// gui/squadqueries.go - NEW FILE, 80 LOC
package gui

import (
    "game_main/common"
    "game_main/squads"
    "github.com/bytearena/ecs"
)

// FindAllSquads returns all squad entity IDs in the game.
// Uses efficient ECS query pattern.
func FindAllSquads(ecsManager *common.EntityManager) []ecs.EntityID {
    allSquads := make([]ecs.EntityID, 0)

    for _, result := range ecsManager.World.Query(ecsManager.Tags["squad"]) {
        squadData := common.GetComponentType[*squads.SquadData](
            result.Entity, squads.SquadComponent)
        allSquads = append(allSquads, squadData.SquadID)
    }

    return allSquads
}

// GetSquadName returns the name of a squad by its ID.
// Returns "Unknown Squad" if squad not found.
func GetSquadName(ecsManager *common.EntityManager, squadID ecs.EntityID) string {
    for _, result := range ecsManager.World.Query(ecsManager.Tags["squad"]) {
        squadData := common.GetComponentType[*squads.SquadData](
            result.Entity, squads.SquadComponent)
        if squadData.SquadID == squadID {
            return squadData.Name
        }
    }
    return "Unknown Squad"
}

// GetSquadAtPosition returns the squad entity ID at the given position.
// Returns 0 if no squad at position or squad is destroyed.
func GetSquadAtPosition(ecsManager *common.EntityManager, pos coords.LogicalPosition) ecs.EntityID {
    for _, result := range ecsManager.World.Query(ecsManager.Tags["mapposition"]) {
        mapPos := common.GetComponentType[*combat.MapPositionData](
            result.Entity, combat.MapPositionComponent)

        if mapPos.Position.X == pos.X && mapPos.Position.Y == pos.Y {
            if !squads.IsSquadDestroyed(mapPos.SquadID, ecsManager) {
                return mapPos.SquadID
            }
        }
    }
    return 0
}

// FindSquadsByFaction returns all squad IDs belonging to a faction.
func FindSquadsByFaction(ecsManager *common.EntityManager, factionID ecs.EntityID) []ecs.EntityID {
    result := make([]ecs.EntityID, 0)

    for _, queryResult := range ecsManager.World.Query(ecsManager.Tags["mapposition"]) {
        mapPos := common.GetComponentType[*combat.MapPositionData](
            queryResult.Entity, combat.MapPositionComponent)

        if mapPos.FactionID == factionID {
            if !squads.IsSquadDestroyed(mapPos.SquadID, ecsManager) {
                result = append(result, mapPos.SquadID)
            }
        }
    }

    return result
}
```

```go
// gui/modehelpers.go - NEW FILE, 60 LOC
package gui

import (
    "github.com/ebitenui/ebitenui/widget"
)

// CreateCloseButton creates a standard close button that transitions to a target mode.
// All modes use this same pattern - centralize it.
func CreateCloseButton(modeManager *UIModeManager, targetModeName, buttonText string) *widget.Button {
    return CreateButtonWithConfig(ButtonConfig{
        Text: buttonText,
        OnClick: func() {
            if targetMode, exists := modeManager.GetMode(targetModeName); exists {
                modeManager.RequestTransition(targetMode, "Close button pressed")
            }
        },
    })
}

// CreateBottomCenterButtonContainer creates a standard bottom-center button container.
// Used by 4+ modes with identical layout.
func CreateBottomCenterButtonContainer(panelBuilders *PanelBuilders) *widget.Container {
    return panelBuilders.BuildPanel(
        BottomCenter(),
        HorizontalRowLayout(),
        CustomPadding(widget.Insets{
            Bottom: int(float64(panelBuilders.layout.ScreenHeight) * 0.08),
        }),
    )
}

// AddActionButton adds a button to an action button container with consistent styling.
func AddActionButton(container *widget.Container, text string, onClick func()) {
    btn := CreateButtonWithConfig(ButtonConfig{
        Text:    text,
        OnClick: onClick,
    })
    container.AddChild(btn)
}
```

*Updated Mode Usage:*
```go
// squadmanagementmode.go - SIMPLIFIED
func (smm *SquadManagementMode) Enter(fromMode UIMode) error {
    fmt.Println("Entering Squad Management Mode")

    // Clear old panels
    smm.clearSquadPanels()

    // Use shared query - ONE LINE instead of 15
    allSquads := FindAllSquads(smm.context.ECSManager)

    // Create panel for each squad
    for _, squadID := range allSquads {
        panel := smm.createSquadPanel(squadID)
        smm.squadPanels = append(smm.squadPanels, panel)
        smm.rootContainer.AddChild(panel.container)
    }

    return nil
}

func (smm *SquadManagementMode) createSquadPanel(squadID ecs.EntityID) *SquadPanel {
    panel := &SquadPanel{squadID: squadID}

    // Container for this squad's panel
    panel.container = CreatePanelWithConfig(/* ... */)

    // Squad name label - use shared query
    squadName := GetSquadName(smm.context.ECSManager, squadID)
    nameLabel := widget.NewText(
        widget.TextOpts.Text(fmt.Sprintf("Squad: %s", squadName), LargeFace, color.White),
    )
    panel.container.AddChild(nameLabel)

    // ... rest of panel creation
    return panel
}
```

```go
// combatmode.go - SIMPLIFIED close button
func (cm *CombatMode) buildActionButtons() {
    // Use shared helper for button container
    cm.actionButtons = CreateBottomCenterButtonContainer(cm.panelBuilders)

    // Add buttons using helper
    AddActionButton(cm.actionButtons, "Attack (A)", func() {
        cm.toggleAttackMode()
    })

    AddActionButton(cm.actionButtons, "Move (M)", func() {
        cm.toggleMoveMode()
    })

    AddActionButton(cm.actionButtons, "End Turn (Space)", func() {
        cm.handleEndTurn()
    })

    // Use shared close button helper
    fleeBtn := CreateCloseButton(cm.modeManager, "exploration", "Flee (ESC)")
    cm.actionButtons.AddChild(fleeBtn)

    cm.rootContainer.AddChild(cm.actionButtons)
}

// Remove local getSquadName method - use shared one
// func (cm *CombatMode) getSquadName(squadID ecs.EntityID) string { ... } DELETE THIS

// Update all calls:
// Before: squadName := cm.getSquadName(squadID)
// After:  squadName := GetSquadName(cm.context.ECSManager, squadID)
```

**Key Changes**:
1. **Two new files with focused utilities**:
   - `squadqueries.go`: Squad-related ECS queries (4 functions, 80 LOC)
   - `modehelpers.go`: Common UI patterns (3 functions, 60 LOC)

2. **Eliminated duplication**:
   - `findAllSquads()` removed from 3 files
   - `getSquadName()` removed from 4+ files
   - Close button creation unified across 5+ modes
   - ~150 LOC reduction across modes

3. **Consistent behavior**:
   - All modes now get squad names the same way
   - All modes use same query patterns
   - Bug fixes propagate to all users automatically

**Value Proposition**:
- **Maintainability**: Fix query once, all modes benefit
- **Readability**: `FindAllSquads(manager)` is clearer than 15 lines of boilerplate
- **Extensibility**: New modes just call helpers instead of reimplementing
- **Complexity Impact**:
  - Total LOC: 4871 → 4761 (2.3% reduction)
  - Duplication: High → Low
  - Cognitive load: Lower (less code to read per mode)

**Implementation Strategy**:
1. **Create utility files** (1 hour):
   - Create `gui/squadqueries.go` with 4 query functions
   - Create `gui/modehelpers.go` with 3 UI helper functions
   - Test each function in isolation

2. **Migrate modes one at a time** (4 hours):
   - CombatMode: Replace getSquadName calls, use CreateCloseButton
   - SquadManagementMode: Replace findAllSquads, use helpers
   - SquadBuilderMode: Same
   - SquadDeploymentMode: Same
   - InventoryMode: Use CreateCloseButton
   - Test each mode after migration

3. **Remove old code** (1 hour):
   - Delete local findAllSquads methods
   - Delete local getSquadName methods
   - Verify no references remain

4. **Documentation** (1 hour):
   - Add godoc comments to utility functions
   - Update mode documentation to reference helpers

**Advantages**:
- **Low Risk**: Adding utility functions doesn't break existing code
- **Incremental**: Can migrate one mode at a time
- **Immediate Value**: Each migrated mode becomes simpler
- **No Over-Engineering**: Only extracting what's actually duplicated 3+ times, not creating a full service layer

**Drawbacks & Risks**:
- **Import cycles**: Must be careful not to create circular dependencies
  - *Mitigation*: Utilities in `gui/` package, only depend on `common/` and `squads/`
- **Dependency on shared code**: Change to helper affects all modes
  - *Mitigation*: Keep helpers simple and well-tested, use semantic versioning if needed
- **Discoverability**: New developers might not know helpers exist
  - *Mitigation*: Clear documentation, code examples in comments

**Effort Estimate**:
- **Time**: 7 hours (~1 work day)
- **Complexity**: Low
- **Risk**: Low
- **Files Impacted**:
  - New: `gui/squadqueries.go` (80 LOC)
  - New: `gui/modehelpers.go` (60 LOC)
  - Modified: 6 mode files (150 LOC removed, replaced with helper calls)

**Critical Assessment** (from refactoring-critic):
This is MEDIUM priority and good pragmatic refactoring. The duplication is real and measurable. The solution is scoped appropriately - NOT creating a heavy service layer, just extracting what's actually repeated. Risk is low because it's additive (new files don't break existing code). The LOC reduction is modest but meaningful, and the consistency benefit is valuable. This is DRY principle applied correctly.
