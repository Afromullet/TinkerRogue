# TinkerRogue Input Reference

**Last Updated:** 2026-03-10

Complete reference for all keyboard, mouse, and user inputs in TinkerRogue. Covers the input system architecture, per-mode keybindings, and detailed interaction logic for complex modes.

---

## Table of Contents

1. [Input System Architecture](#input-system-architecture)
2. [Quick Reference: All Modes](#quick-reference-all-modes)
   - [Combat Mode](#combat-mode-bindings)
   - [Overworld Mode](#overworld-mode-bindings)
   - [Squad Editor Mode](#squad-editor-mode-bindings)
   - [Artifact Mode](#artifact-mode-bindings)
   - [Node Placement Mode](#node-placement-mode-bindings)
   - [Raid Mode](#raid-mode-bindings)
   - [Combat Animation Mode](#combat-animation-mode-bindings)
   - [Camera / Exploration](#camera--exploration-bindings)
   - [Unit Purchase Mode](#unit-purchase-mode-bindings)
3. [Combat Mode: Detailed Input Logic](#combat-mode-detailed-input-logic)
4. [Overworld Mode: Detailed Input Logic](#overworld-mode-detailed-input-logic)
5. [Squad Editor: Grid Interactions](#squad-editor-grid-interactions)
6. [Mouse Interaction Summary](#mouse-interaction-summary)
7. [Key Files Reference](#key-files-reference)

---

## Input System Architecture

### Overview

Input is processed through a layered pipeline each frame:

```
Physical Device (keyboard/mouse)
        ↓
UIModeManager  (captures raw key state per frame)
        ↓
ActionMap      (resolves physical keys → semantic InputActions)
        ↓
Mode Handler   (interprets active actions in context)
```

### Trigger Types

Every binding specifies when it fires, controlled by `InputTrigger` in `gui/framework/inputbinding.go`.

| Trigger | Constant | Fires When |
|---------|----------|-----------|
| Just Pressed | `TriggerJustPressed` | The frame the key first goes down (default) |
| Held | `TriggerHeld` | Every frame the key is held |
| Just Released | `TriggerJustReleased` | The frame the key comes back up |

Camera movement bindings use `TriggerJustReleased` to move the camera after the key is let go rather than continuously during hold.

### Modifier Keys

Modifier requirements are specified as a bitmask (`ModifierMask`).

| Constant | Meaning |
|----------|---------|
| `ModNone` | No modifier required |
| `ModCtrl` | Control key (or Meta on macOS) |
| `ModShift` | Shift key |
| `ModAlt` | Alt key |

**Important:** Plain (unmodified) key bindings are automatically rejected when any modifier is held. This prevents, for example, `S` from triggering `ActionSpellPanel` when the user types `Ctrl+S`.

### InputBinding Structure

Each binding maps a physical key or mouse button to a semantic `InputAction`:

```
InputBinding {
    Action    InputAction      // semantic meaning (e.g. ActionAttackMode)
    Key       ebiten.Key       // physical key (if keyboard binding)
    Mouse     ebiten.MouseButton // physical button (if mouse binding)
    IsMouse   bool             // distinguishes key vs. mouse binding
    Modifiers ModifierMask     // required modifier combination
    Trigger   InputTrigger     // when to fire
}
```

### ActionMap

`gui/framework/actionmap.go` holds a set of bindings for a given mode. Each frame, `ResolveInto()` evaluates all bindings against the current `InputState` and writes the active `InputAction` values into a map that handlers query.

Binding helpers on `ActionMap`:

| Method | Use |
|--------|-----|
| `Bind(key, action)` | Plain key, just-pressed |
| `BindMod(key, mod, action)` | Key with modifier, just-pressed |
| `BindMouse(btn, action)` | Mouse button, just-pressed |
| `BindRelease(key, action)` | Key on release |
| `BindHeld(key, action)` | Key while held |
| `MergeActionMaps(...)` | Combine multiple maps into one |

### Game Contexts

The `GameModeCoordinator` (`gui/framework/coordinator.go`) maintains two independent `UIModeManager` instances:

| Context | Manager | Layer |
|---------|---------|-------|
| `ContextOverworld` | `overworldManager` | Strategic (squad management, world map) |
| `ContextTactical` | `tacticalManager` | Tactical (dungeon exploration, combat) |

---

## Quick Reference: All Modes

### Combat Mode Bindings

> **File:** `gui/framework/defaultbindings.go` → `DefaultCombatBindings()`
> **Handler:** `gui/guicombat/combat_input_handler.go`

| Key / Input | Action | Description |
|-------------|--------|-------------|
| `A` | `ActionAttackMode` | Toggle attack mode |
| `M` | `ActionMoveMode` | Toggle move mode |
| `S` | `ActionSpellPanel` | Toggle spell panel |
| `D` | `ActionArtifactPanel` | Toggle artifact panel |
| `I` | `ActionInspectMode` | Toggle inspect mode |
| `Tab` | `ActionCycleSquad` | Cycle through squads |
| `Space` | `ActionEndTurn` | End current turn |
| `1` | `ActionSelectTarget1` | Select enemy target 1 (attack mode) |
| `2` | `ActionSelectTarget2` | Select enemy target 2 (attack mode) |
| `3` | `ActionSelectTarget3` | Select enemy target 3 (attack mode) |
| `H` | `ActionThreatToggle` | Toggle threat heat map |
| `Shift+H` | `ActionThreatCycleFact` | Cycle threat faction |
| `Ctrl+Right` | `ActionHealthBarToggle` | Toggle health bar display |
| `L` | `ActionLayerToggle` | Toggle layer visualizer |
| `Shift+L` | `ActionLayerCycleMode` | Cycle layer visualization mode |
| `Ctrl+Z` | `ActionUndoMove` | Undo last move |
| `Ctrl+K` | `ActionDebugKillAll` | Kill all enemies (debug only) |
| `ESC` | `ActionCancel` | Cancel / close panel |
| Left Mouse | `ActionMouseClick` | Context-sensitive click (see [Combat Detail](#combat-mode-detailed-input-logic)) |

**AoE Targeting sub-mode** (active during AoE spell selection):

| Key | Action | Description |
|-----|--------|-------------|
| `1` | `ActionAoERotateLeft` | Rotate AoE shape left |
| `2` | `ActionAoERotateRight` | Rotate AoE shape right |
| Left Mouse | — | Confirm AoE placement |

---

### Overworld Mode Bindings

> **File:** `gui/framework/defaultbindings.go` → `DefaultOverworldBindings()`
> **Handler:** `gui/guioverworld/overworld_input_handler.go`

| Key / Input | Action | Description |
|-------------|--------|-------------|
| `ESC` | `ActionCancel` | Cancel / close panel |
| `N` | `ActionNodePlacement` | Enter node placement mode |
| `Space` | `ActionEndOverworldTurn` | End overworld turn |
| `Enter` | `ActionEndOverworldTurn` | End overworld turn |
| `M` | `ActionOverworldMove` | Toggle movement mode |
| `Tab` | `ActionCycleCommander` | Cycle to next commander |
| `I` | `ActionToggleInfluence` | Toggle influence display |
| `G` | `ActionGarrison` | Open garrison management |
| `R` | `ActionRecruitCommander` | Recruit new commander |
| `S` | `ActionSquadManagement` | Open squad management |
| `E` | `ActionEngageThreat` | Engage threat at commander position |
| Left Mouse | `ActionMouseClick` | Context-sensitive click (see [Overworld Detail](#overworld-mode-detailed-input-logic)) |

---

### Squad Editor Mode Bindings

> **File:** `gui/framework/defaultbindings.go` → `DefaultSquadEditorBindings()`
> **Handler:** `gui/guisquads/squadeditormode.go`, `gui/guisquads/squadeditor_grid.go`

| Key / Input | Action | Description |
|-------------|--------|-------------|
| `ESC` | `ActionCancel` | Cancel / close panel |
| `U` | `ActionToggleUnits` | Toggle units panel |
| `R` | `ActionToggleRoster` | Toggle roster panel |
| `N` | `ActionNewSquad` | Create new squad |
| `V` | `ActionToggleAttackPattern` | Toggle attack pattern view |
| `B` | `ActionToggleSupportPattern` | Toggle support pattern view |
| `Tab` | `ActionCycleCommanderEditor` | Cycle to next commander |
| Left Mouse | — | Grid cell interaction (see [Squad Editor Grid](#squad-editor-grid-interactions)) |
| Right Mouse | — | Remove unit from cell |
| `Shift` + Left Mouse | — | View unit details |

---

### Artifact Mode Bindings

> **File:** `gui/framework/defaultbindings.go` → `DefaultArtifactBindings()`

| Key / Input | Action | Description |
|-------------|--------|-------------|
| `ESC` | `ActionCancel` | Cancel / close panel |
| `Left Arrow` | `ActionPrevSquad` | Previous squad |
| `Right Arrow` | `ActionNextSquad` | Next squad |
| `I` | `ActionTabInventory` | Switch to inventory tab |
| `E` | `ActionTabEquipment` | Switch to equipment tab |

---

### Node Placement Mode Bindings

> **File:** `gui/framework/defaultbindings.go` → `DefaultNodePlacementBindings()`
> **Mode:** `gui/guinodeplacement/nodeplacementmode.go`

| Key / Input | Action | Description |
|-------------|--------|-------------|
| `ESC` | `ActionCancel` | Cancel / exit node placement |
| `Tab` | `ActionCycleNodeType` | Cycle through node types |
| `1` | `ActionSelectNodeType1` | Select node type 1 |
| `2` | `ActionSelectNodeType2` | Select node type 2 |
| `3` | `ActionSelectNodeType3` | Select node type 3 |
| `4` | `ActionSelectNodeType4` | Select node type 4 |
| Left Mouse | `ActionMouseClick` | Place node at cursor position |

---

### Raid Mode Bindings

Raid mode has three sub-panels, each with separate bindings.

> **File:** `gui/guiraid/raidmode.go`

**Floor Map Panel** (`DefaultRaidFloorMapBindings()`):

| Key / Input | Action | Description |
|-------------|--------|-------------|
| `1`–`9` | `ActionSelectRoom1`–`ActionSelectRoom9` | Select room on floor map |
| Left Mouse | `ActionMouseClick` | Click to select room |

**Deploy Panel** (`DefaultRaidDeployBindings()`):

| Key / Input | Action | Description |
|-------------|--------|-------------|
| `Enter` | `ActionConfirm` | Confirm squad deployment |
| `ESC` | `ActionDeployBack` | Return to floor map |

**Summary Panel** (`DefaultRaidSummaryBindings()`):

| Key / Input | Action | Description |
|-------------|--------|-------------|
| `Enter` | `ActionConfirm` | Confirm and continue |
| `Space` | `ActionDismiss` | Dismiss summary |

---

### Combat Animation Mode Bindings

> **File:** `gui/framework/defaultbindings.go` → `DefaultCombatAnimationBindings()`

| Key / Input | Action | Description |
|-------------|--------|-------------|
| `Space` | `ActionReplayAnimation` | Replay the combat animation |
| `ESC` | `ActionCancel` | Skip / close animation |
| Left Mouse | `ActionMouseClick` | Click interaction |

---

### Camera / Exploration Bindings

> **File:** `gui/framework/defaultbindings.go` → `DefaultCameraBindings()`
> **Handler:** `input/cameracontroller.go`

All camera bindings use `TriggerJustReleased` — the camera moves when you release the key, not when you press it.

| Key | Action | Direction |
|-----|--------|-----------|
| `W` | `ActionCameraMoveUp` | Move up |
| `S` | `ActionCameraMoveDown` | Move down |
| `A` | `ActionCameraMoveLeft` | Move left |
| `D` | `ActionCameraMoveRight` | Move right |
| `Q` | `ActionCameraMoveUpLeft` | Move up-left (diagonal) |
| `E` | `ActionCameraMoveUpRight` | Move up-right (diagonal) |
| `Z` | `ActionCameraMoveDownLeft` | Move down-left (diagonal) |
| `C` | `ActionCameraMoveDownRight` | Move down-right (diagonal) |
| `B` | `ActionCameraHighlight` | Debug tile highlight |
| `M` | `ActionCameraToggleScroll` | Toggle map scrolling |

> **Note:** `W/A/S/D` are shared with combat mode. Combat bindings take priority in tactical context; camera bindings apply in exploration context.

---

### Unit Purchase Mode Bindings

> **File:** `gui/framework/defaultbindings.go` → `DefaultUnitPurchaseBindings()`

| Key / Input | Action | Description |
|-------------|--------|-------------|
| `ESC` | `ActionCancel` | Cancel / close purchase panel |
| `Ctrl+Z` | `ActionUndo` | Undo last purchase |
| `Ctrl+Y` | `ActionRedo` | Redo purchase |

---

## Combat Mode: Detailed Input Logic

The combat input handler (`gui/guicombat/combat_input_handler.go`) uses a strict priority stack. Higher-priority sub-modes suppress all lower-priority input when active.

### Input Priority Stack (Highest to Lowest)

**1. Debug Kill Mode** (activated by `Ctrl+K`)

| Input | Effect |
|-------|--------|
| Left-click on squad | Kill squad at that position |
| `ESC` | Exit debug kill mode |

---

**2. Spell Casting Mode** (activated by `S` or opening spell panel)

Suppresses all other input while active.

| Input | Effect |
|-------|--------|
| `ESC` | Cancel spell casting |
| Left-click | Cast single-target spell at clicked position |
| `1` | Rotate AoE shape left (AoE targeting only) |
| `2` | Rotate AoE shape right (AoE targeting only) |
| Left-click (AoE) | Confirm AoE placement at cursor |

---

**3. Artifact Mode** (activated by `D` or opening artifact panel)

Suppresses all other input while active.

| Input | Effect |
|-------|--------|
| `ESC` | Cancel artifact activation |
| Left-click | Apply artifact to target at clicked position |

---

**4. Inspect Mode** (activated by `I`)

Suppresses all other input while active.

| Input | Effect |
|-------|--------|
| `ESC` | Exit inspect mode |
| Left-click | Inspect squad at clicked position |

---

**5. Move Mode** (activated by `M` or double-clicking own squad)

| Input | Effect |
|-------|--------|
| Left-click on valid tile | Move selected squad to that tile |
| Right-click | Exit move mode |

---

**6. Normal Mode** (no active sub-mode)

| Input | Condition | Effect |
|-------|-----------|--------|
| Left-click | On allied squad | Select that squad |
| Left-click | Double-click on allied squad | Enter move mode for that squad |
| Left-click | On enemy squad | Execute attack against that squad |
| Left-click | On empty tile | Deselect current squad |

**Double-click threshold:** 300ms between clicks.

---

**7. Global Hotkeys** (always available regardless of sub-mode)

These hotkeys are processed after sub-mode input unless a sub-mode explicitly suppresses them.

| Key | Effect |
|-----|--------|
| `Tab` | Cycle to next squad |
| `Space` | End turn |
| `S` | Toggle spell panel |
| `D` | Toggle artifact panel |
| `A` | Toggle attack mode |
| `M` | Toggle move mode |
| `I` | Toggle inspect mode |
| `H` | Toggle threat visualization |
| `Shift+H` | Cycle threat faction |
| `Ctrl+Right` | Toggle health bar display |
| `L` | Toggle layer visualizer |
| `Shift+L` | Cycle layer visualization mode |
| `Ctrl+Z` | Undo last move |
| `Ctrl+K` | Kill all enemies (debug) |

---

## Overworld Mode: Detailed Input Logic

The overworld input handler (`gui/guioverworld/overworld_input_handler.go`) has two behavioral layers.

### Move Mode Active

When the player has toggled movement mode (`M`):

| Input | Condition | Effect |
|-------|-----------|--------|
| Left-click | On a valid (highlighted blue) tile | Move commander to that tile |
| Left-click | Outside valid tiles | Exit move mode |
| `ESC` | — | Exit move mode |

### Normal Mode (Mouse Click Handling)

When move mode is not active, left-click target determines what is selected:

| Click Target | Effect |
|--------------|--------|
| Commander | Select that commander |
| Threat | Select that threat |
| Node | Select that node |
| Empty space | Clear current selection |

### Global Hotkeys (Always Available)

| Key | Effect |
|-----|--------|
| `N` | Enter node placement mode |
| `Space` / `Enter` | End overworld turn |
| `M` | Toggle move mode |
| `Tab` | Cycle commanders |
| `I` | Toggle influence display |
| `G` | Open garrison dialog |
| `R` | Recruit a commander |
| `S` | Open squad management |
| `E` | Engage threat at commander's current position |

---

## Squad Editor: Grid Interactions

The squad formation grid (`gui/guisquads/squadeditor_grid.go`) interprets mouse input based on the current selection state and whether a cell is occupied.

### Left-Click Behavior

| Cell State | Selection State | Effect |
|------------|-----------------|--------|
| Unit present | Any | Select that unit; opens units panel |
| Empty cell | Unit selected | Move selected unit to this cell |
| Empty cell | Nothing selected | Open roster to choose unit for placement |

### Right-Click Behavior

| Input | Effect |
|-------|--------|
| Right-click on any cell | Remove unit from cell; unit returns to roster |

### Modified Click

| Input | Effect |
|-------|--------|
| `Shift` + Left-click | Open unit detail view for the unit in that cell |

---

## Mouse Interaction Summary

| Mode | Left Mouse | Right Mouse | Shift+Left |
|------|-----------|-------------|------------|
| Combat (normal) | Select / attack / deselect | — | — |
| Combat (move mode) | Move squad to tile | Exit move mode | — |
| Combat (spell mode) | Cast spell / confirm AoE | — | — |
| Combat (artifact mode) | Apply artifact | — | — |
| Combat (inspect mode) | Inspect squad | — | — |
| Combat (debug kill) | Kill squad at position | — | — |
| Overworld (normal) | Select commander/threat/node | — | — |
| Overworld (move mode) | Move commander / exit move mode | — | — |
| Squad Editor grid | Select / place unit | Remove unit | View unit detail |
| Node Placement | Place node | — | — |
| Raid (floor map) | Select room | — | — |
| Combat Animation | Click interaction | — | — |

---

## Key Files Reference

| File | Purpose |
|------|---------|
| `gui/framework/inputaction.go` | All `InputAction` semantic constants |
| `gui/framework/inputbinding.go` | `InputBinding`, `InputTrigger`, `ModifierMask` types |
| `gui/framework/defaultbindings.go` | Default key-to-action mapping functions per mode |
| `gui/framework/actionmap.go` | `ActionMap` — resolves physical input to semantic actions |
| `gui/framework/modemanager.go` | `UIModeManager` — captures per-frame input state |
| `gui/framework/coordinator.go` | `GameModeCoordinator` — switches between overworld and tactical contexts |
| `gui/guicombat/combat_input_handler.go` | Combat input dispatch and priority stack |
| `gui/guicombat/combatmode.go` | Combat mode setup and action map initialization |
| `gui/guioverworld/overworld_input_handler.go` | Overworld input dispatch |
| `gui/guisquads/squadeditor_grid.go` | Squad editor grid cell interaction logic |
| `gui/guisquads/squadeditormode.go` | Squad editor mode setup |
| `gui/guinodeplacement/nodeplacementmode.go` | Node placement mode and input |
| `gui/guiraid/raidmode.go` | Raid mode sub-panels and bindings |
| `gui/guiexploration/explorationmode.go` | Exploration mode setup |
| `input/cameracontroller.go` | Camera / player movement handler |
