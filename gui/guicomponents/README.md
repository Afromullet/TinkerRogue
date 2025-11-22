# GUI Components Pattern Guide

**Last Updated:** 2025-11-22
**Status:** Establishing consistency across all UI modes

---

## Overview

This package provides reusable UI components that abstract common widget update patterns. These components encapsulate refresh logic for dynamic content, making modes cleaner and reducing duplication.

**Available Components:**
- `SquadListComponent` - Dynamic squad button list with filtering
- `DetailPanelComponent` - Generic detail panel with formatting
- `TextDisplayComponent` - Text display with custom refresh logic
- `ItemListComponent` - Dynamic item list with filtering
- `PanelListComponent` - Dynamic panel list management
- `ButtonListComponent` - Dynamic button list management
- `ColorLabelComponent` - Color-coded label display
- `StatsDisplayComponent` - Player stats display

---

## Decision Matrix: When to Use Components vs Direct Widget Access

### ✅ USE A COMPONENT WHEN:

1. **Content refreshes frequently** (every frame or on events)
   - Squad list updates each turn
   - Item list filtered by type
   - Stats updated on player change
   - **Example:** `SquadListComponent` in `CombatMode`

2. **Multiple widgets update together** (synced refreshes)
   - Panel header + content change together
   - Squad name + HP bar + status all update
   - **Example:** `DetailPanelComponent` showing faction info

3. **Complex filtering/selection logic**
   - Need to filter squads by faction/status
   - Need to handle selection callbacks
   - **Example:** `SquadListComponent` with custom `SquadFilter`

4. **Same update pattern used in multiple modes**
   - Multiple modes show item lists → use `ItemListComponent`
   - Multiple modes show stat displays → use `StatsDisplayComponent`
   - **Example:** `StatsDisplayComponent` used in ExplorationMode and potentially other modes

5. **Widget acts as a reusable "container" for dynamic content**
   - Container children change, but container behavior is consistent
   - **Example:** Squad list panel in combat and squad management

---

### ✅ USE DIRECT WIDGET ACCESS WHEN:

1. **Static UI that rarely changes** after initialization
   - Combat log text area (created once, appended to)
   - Squad visualization grid (displayed per mode, not updated frequently)
   - **Example:** `gridDisplay` in `SquadManagementMode`

2. **Single widget updates** (not multiple synced widgets)
   - Setting a TextArea to show one thing
   - Updating a single Label
   - **Example:** `messageLog` in `ExplorationMode`

3. **Mode-specific one-off logic** (not shared pattern)
   - SquadManagementMode updates grid display specifically when squad changes
   - That grid display logic won't be reused elsewhere
   - **Example:** `panel.gridDisplay.SetText()` in `squadmanagementmode.go`

4. **Simple data binding** (no complex refresh logic)
   - Text = function result
   - No filtering, no callbacks, no side effects
   - **Example:** Squad counter label update

5. **Append-only or streaming content** (doesn't need reset)
   - Combat log (messages accumulate)
   - Message history (only grows)
   - **Example:** `combatLogArea` in `CombatMode`

---

## Code Examples

### Example 1: Squad List (USE COMPONENT)

```go
// ✅ GOOD: Using SquadListComponent for dynamic, filtered squad list
cm.squadListComponent = guicomponents.NewSquadListComponent(
    cm.squadListPanel,
    cm.Queries,
    func(info *guicomponents.SquadInfo) bool {
        // Filter to current faction's alive squads
        return !info.IsDestroyed && info.FactionID == currentFactionID
    },
    func(squadID ecs.EntityID) {
        cm.actionHandler.SelectSquad(squadID)
    },
)

// Later, refresh when turn changes:
cm.squadListComponent.Refresh()  // Rebuilds buttons with current filter
```

**Why Component?**
- Content changes every turn
- Needs filtering logic
- Multiple modes might show squads
- Selection callback pattern is reusable

---

### Example 2: Combat Log (USE DIRECT ACCESS)

```go
// ✅ GOOD: Direct TextArea access for append-only log
func (cm *CombatMode) addLog(message string) {
    cm.logManager.UpdateTextArea(cm.combatLogArea, message)
}

// ❌ BAD (don't do this):
combatLogComponent := guicomponents.NewTextDisplayComponent(
    cm.combatLogArea,
    func() string { return cm.logManager.GetAllMessages() },
)
combatLogComponent.Refresh()  // Overkill for simple append
```

**Why Direct?**
- Append-only content (doesn't need refresh)
- No filtering or complex logic
- Single widget
- Simple text updates

---

### Example 3: Squad Grid Display (USE DIRECT ACCESS)

```go
// ✅ GOOD: Direct TextArea for static-per-mode grid display
func (smm *SquadManagementMode) createSquadPanel(squadID ecs.EntityID) *SquadPanel {
    gridVisualization := squads.VisualizeSquad(squadID, smm.Context.ECSManager)
    panel.gridDisplay = widgets.CreateTextAreaWithConfig(gridConfig)
    panel.gridDisplay.SetText(gridVisualization)
    panel.container.AddChild(panel.gridDisplay)
    return panel
}

// ❌ BAD (over-engineering):
gridComponent := guicomponents.NewTextDisplayComponent(
    panel.gridDisplay,
    func() string {
        return squads.VisualizeSquad(smm.currentSquadID, smm.Context.ECSManager)
    },
)
// Call refresh every frame? No, grid doesn't change during mode
```

**Why Direct?**
- Display only changes when mode switches squads
- Created once per squad view
- Simple single-widget display
- Not refreshed during normal updates

---

### Example 4: Stat Display (USE COMPONENT)

```go
// ✅ GOOD: Using StatsDisplayComponent for dynamic player stats
em.statsComponent = guicomponents.NewStatsDisplayComponent(
    em.statsTextArea,
    nil,  // Use default formatter
)

// Refresh when player stats might have changed:
func (em *ExplorationMode) Enter(fromMode core.UIMode) error {
    em.statsComponent.RefreshStats(em.Context.PlayerData, em.Context.ECSManager)
    return nil
}
```

**Why Component?**
- Stat display might change during exploration
- Multiple modes might show player stats
- Encapsulates stat formatting logic
- Reusable across game modes

---

### Example 5: Faction Info (USE COMPONENT)

```go
// ✅ GOOD: DetailPanelComponent for multi-field faction display
cm.factionInfoComponent = guicomponents.NewDetailPanelComponent(
    cm.factionInfoText,
    cm.Queries,
    func(data interface{}) string {
        factionInfo := data.(*guicomponents.FactionInfo)
        return fmt.Sprintf("%s\nSquads: %d/%d\nMana: %d/%d",
            factionInfo.Name,
            factionInfo.AliveSquadCount,
            len(factionInfo.SquadIDs),
            factionInfo.CurrentMana,
            factionInfo.MaxMana,
        )
    },
)

// Update when faction changes:
cm.factionInfoComponent.ShowFaction(currentFactionID)
```

**Why Component?**
- Multiple fields (name, squads, mana) updated together
- Display changes frequently (each turn)
- Formatting logic is encapsulated
- Could be reused in other modes

---

## Anti-Patterns to Avoid

### ❌ Anti-Pattern 1: Component for Static Content
```go
// DON'T DO THIS - Overkill for static text
staticLabel := guicomponents.NewTextDisplayComponent(
    myLabel,
    func() string { return "Squad Management" },
)
staticLabel.Refresh()  // Called every frame for static content
```

### ❌ Anti-Pattern 2: Ignoring Shared Patterns
```go
// DON'T DO THIS - Use ItemListComponent instead
for _, item := range items {
    btn := widgets.CreateButton(item.Name)
    myContainer.AddChild(btn)
}
// Later: Need to filter items?
// Now you have to duplicate this logic in another mode
```

### ❌ Anti-Pattern 3: Component for Append-Only Content
```go
// DON'T DO THIS - Combat log doesn't need refresh logic
combatLog := guicomponents.NewTextDisplayComponent(
    logArea,
    func() string { return strings.Join(messages, "\n") },
)
// This rebuilds the entire log text every frame!
```

### ❌ Anti-Pattern 4: Tight Coupling Between Component and Widget
```go
// DON'T DO THIS - Component knows too much about widget implementation
type MyCustomComponent struct {
    textArea *widget.TextArea  // Should be generic
    button *widget.Button       // Should be generic
}
// Now this component only works with TextArea+Button combo
```

---

## When to Create a New Component

Before creating a new component, ask yourself:

1. **Will this pattern be used in multiple modes?** (Yes → Component)
2. **Is the content dynamic** (changes frequently)? (Yes → Component)
3. **Can I reuse an existing component?** (Yes → Use it)
4. **Does it encapsulate complex refresh logic?** (Yes → Component)
5. **Or is it a one-off widget?** (Yes → Direct access)

### Creating a New Component

If you answer "yes" to questions 1-4:

1. Define the component struct with fields for:
   - Widget(s) to update
   - Data/queries needed for refresh
   - Callbacks if needed

2. Implement `NewXxxComponent()` constructor
3. Implement `Refresh()` method (or `Show()`, `Update()`, etc.)
4. Add helper methods for common operations
5. Document the component's purpose and refresh pattern

**Example Template:**
```go
type MyComponent struct {
    widget    *widget.SomeWidget
    queries   *GUIQueries
    callback  func(data interface{})
}

func NewMyComponent(w *widget.SomeWidget, q *GUIQueries, cb func(interface{})) *MyComponent {
    return &MyComponent{
        widget: w,
        queries: q,
        callback: cb,
    }
}

func (mc *MyComponent) Refresh() {
    data := mc.queries.GetData()
    text := mc.formatData(data)
    mc.widget.SetText(text)
    if mc.callback != nil {
        mc.callback(data)
    }
}
```

---

## Current State by Mode

| Mode | Pattern | Assessment |
|------|---------|------------|
| CombatMode | ✅ Components | Perfect - uses 4 components for dynamic content |
| ExplorationMode | 🔄 Mixed | StatsDisplayComponent ✅ + direct messageLog ✅ (appropriate) |
| InventoryMode | 🔄 Mixed | ItemListComponent ✅ + direct filtering ✅ (appropriate) |
| SquadManagementMode | ✅ Direct | Appropriate - grid is static per squad view |
| FormationEditorMode | ⚠️ TBD | Check if formation display is dynamic |
| SquadBuilderMode | ⚠️ TBD | Check if unit list needs component |
| SquadDeploymentMode | ⚠️ TBD | Check if deployment preview is dynamic |

---

## Summary

**Use Components for:**
- Dynamic content (refreshes frequently)
- Multi-widget updates (synced refreshes)
- Reusable patterns (across modes)
- Complex logic (filtering, callbacks)

**Use Direct Access for:**
- Static content (created once)
- Single widget updates
- Mode-specific logic
- Simple data binding

**Best Practice:** Look at `CombatMode` - it demonstrates the right balance of components for dynamic content and direct access where appropriate.

