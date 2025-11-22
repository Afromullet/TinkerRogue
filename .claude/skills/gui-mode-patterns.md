# GUI Mode Patterns Skill

**Purpose**: Quick UI mode creation and pattern suggestions
**Trigger**: When working with ebitenui widgets or mode system

## Capabilities

- Mode template generation (BaseMode embedding)
- Widget config pattern suggestions
- Component usage recommendations
- Layout constant selection
- GUIQueries integration patterns

## Core GUI Patterns

### 1. Mode Structure (BaseMode Embedding)

```go
type NewMode struct {
    gui.BaseMode  // Embed BaseMode for common functionality

    // Mode-specific state
    selectedSquadID ecs.EntityID

    // UI components
    mainContainer *widget.Container
    buttons       []*widget.Button
}

func NewNewMode(manager *ecs.Manager, inputCoord *inputmanager.InputCoordinator) *NewMode {
    mode := &NewMode{}
    mode.InitBaseMode(manager, inputCoord, "NewMode")  // Initialize base

    // Setup UI
    mode.setupUI()
    mode.setupInputHandlers()

    return mode
}
```

### 2. Widget Configuration Pattern

```go
// Button with config
button := widgets.CreateButtonWithConfig(widgets.ButtonConfig{
    Text:        "Squad Name",
    Width:       200,
    Height:      40,
    OnClick: func() {
        // IMPORTANT: Capture loop variable
        localID := squadID  // Capture in closure
        mode.handleSquadSelect(localID)
    },
})

// Container with layout
container := widget.NewContainer(
    widget.ContainerOpts.Layout(widget.NewGridLayout(
        widget.GridLayoutOpts.Columns(3),
        widget.GridLayoutOpts.Spacing(10, 10),
        widget.GridLayoutOpts.Padding(widget.Insets{
            Top: 20, Bottom: 20, Left: 20, Right: 20,
        }),
    )),
)
```

### 3. Component Usage

**SquadListComponent**:
```go
squadList := components.NewSquadListComponent(mode.ECSManager)
container.AddChild(squadList.GetContainer())

// Update data
squadList.UpdateData()
```

**DetailPanelComponent**:
```go
detailPanel := components.NewDetailPanelComponent(mode.ECSManager)
detailPanel.ShowSquadDetails(squadID)
```

### 4. GUIQueries Integration

```go
// Query ECS data for GUI display
squadListData := guiqueries.GetSquadListData(mode.ECSManager)

for _, squadData := range squadListData.Squads {
    // Create UI element per squad
    button := widgets.CreateButtonWithConfig(widgets.ButtonConfig{
        Text: squadData.Name,
        OnClick: func() {
            localID := squadData.ID  // Capture
            mode.selectSquad(localID)
        },
    })
}
```

### 5. Input Handling

```go
func (mode *NewMode) HandleInput(currentState *input.InputState) gui.InputResult {
    // Check hotkeys
    if currentState.IsKeyJustPressed(ebiten.KeyEscape) {
        return gui.InputResult{
            Handled:      true,
            RequestedMode: "exploration",  // Return to exploration
        }
    }

    if currentState.IsKeyJustPressed(ebiten.KeyS) {
        mode.handleSquadAction()
        return gui.InputResult{Handled: true}
    }

    return gui.InputResult{Handled: false}
}
```

### 6. Mode Transitions

```go
// Transition to another mode
func (mode *NewMode) transitionToFormationEditor() {
    mode.RequestModeChange("formationeditor")
}

// Clean up when leaving mode
func (mode *NewMode) OnExit() {
    // Clean up resources
    mode.mainContainer = nil
}

// Initialize when entering mode
func (mode *NewMode) OnEnter() {
    // Refresh data
    mode.refreshSquadList()
}
```

## Layout Constants

```go
// Common layout constants (from GUI_PATTERNS.md)
const (
    PanelPadding      = 20
    ButtonWidth       = 200
    ButtonHeight      = 40
    GridSpacing       = 10
    HeaderHeight      = 60
    FooterHeight      = 50
)
```

## Common Components

- **SquadListComponent**: Display list of squads with selection
- **DetailPanelComponent**: Show detailed info for selected entity
- **FormationGridComponent**: 3x3 grid for squad formations
- **StatDisplayComponent**: Show unit/squad stats
- **ButtonBarComponent**: Action buttons (OK, Cancel, etc.)

## Widget Best Practices

1. **Capture Loop Variables**: Always create local variable in loop when using closures
   ```go
   for _, squad := range squads {
       localSquad := squad  // Capture for closure
       button.OnClick = func() { useSquad(localSquad) }
   }
   ```

2. **Use ButtonConfig Pattern**: Consistent button creation across all modes
   ```go
   widgets.CreateButtonWithConfig(widgets.ButtonConfig{...})
   ```

3. **Separate UI Setup**: Create `setupUI()` method for clarity
   ```go
   func (mode *NewMode) setupUI() {
       mode.mainContainer = widget.NewContainer(...)
       mode.addButtons()
       mode.addPanels()
   }
   ```

4. **Query Data Through GUIQueries**: Don't access ECS directly in UI code
   ```go
   // ✅ GOOD
   data := guiqueries.GetSquadListData(manager)

   // ❌ BAD
   squads := manager.FilterByTag(SquadTag)  // Don't query directly in GUI
   ```

## Mode Lifecycle

```go
// 1. Creation
mode := NewNewMode(manager, inputCoord)

// 2. Enter mode
mode.OnEnter()

// 3. Update loop
mode.Update()
mode.Render(screen)
mode.HandleInput(inputState)

// 4. Exit mode
mode.OnExit()
```

## Reference Modes

- `gui/squadmanagement/squadmanagementmode.go` - List + selection pattern
- `gui/squaddeployment/squaddeploymentmode.go` - Grid + placement pattern
- `gui/formationeditor/formationeditormode.go` - 3x3 grid editor pattern
- `gui/squadbuilder/squadbuilder.go` - Multi-panel creation pattern

## Two-Phase Development

**Phase 1: Design & Planning**
1. Define layout structure
2. List required widgets
3. Identify ECS queries needed
4. Determine input handlers
5. Plan mode transitions

**Phase 2: Implementation**
- Create analysis/gui_{mode-name}_plan_[timestamp].md
- Ask for approval before coding
- Implement following patterns above
- Test integration with existing modes

## Usage Example

```markdown
User: "Need a button for squad selection"

Response:
Use widgets.CreateButtonWithConfig pattern:

```go
button := widgets.CreateButtonWithConfig(widgets.ButtonConfig{
    Text:    squadData.Name,  // Squad name from GUIQueries
    Width:   200,
    Height:  40,
    OnClick: func() {
        localID := squadID  // IMPORTANT: Capture loop variable
        mode.handleSquadSelect(localID)
    },
})
```

Reference: GUI_PATTERNS.md, SquadManagementMode for similar implementation
```
