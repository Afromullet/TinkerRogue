# TinkerRogue UI Design Principles

A checklist of principles to apply when designing or revising any UI screen in TinkerRogue. Each principle includes a reusable pattern and concrete examples from existing modes. Consult this guide before building a new mode or reworking an existing one.

---

## 1. Progressive Disclosure

Don't show everything at once. Start with the minimum information the player needs for their current task and reveal detail on demand. This reduces cognitive load and keeps the screen uncluttered during the most common interactions.

**Pattern:** Use `SubMenuController` panels that start hidden and toggle via hotkey or context action. Only surface a panel when the player signals they need it.

**Examples:**
- Combat mode: spell and artifact panels are hidden until the player presses a hotkey to open them.
- Squad editor: unit detail and roster panels are collapsed by default; the player opens them when they want to browse or assign units.

---

## 2. Visual Hierarchy

Not all elements are equally important. The primary interaction surface should dominate the screen; secondary tools appear in supporting positions at the edges. Size, placement, and contrast should make the most important element obvious at a glance.

**Pattern:** Center the primary content (map, formation grid, battlefield) and place controls, lists, and status displays at the periphery. The player's eye should land on the thing they interact with most.

**Examples:**
- Squad editor: the formation grid occupies the center of the screen; the squad list and action bars sit along the edges.
- Combat mode: the battle map fills the viewport; action bars and unit info panels frame the edges.

---

## 3. Gestalt Proximity (Group by Task)

Actions should be physically close to the objects they affect. If a button operates on the squad list, it belongs in the squad list panel, not in a shared action bar elsewhere on screen. Spatial proximity tells the player "these things belong together" without requiring labels or explanation.

**Pattern:** Embed object-specific actions in the same panel as the object they modify. Avoid dumping unrelated actions into a single shared toolbar.

**Examples:**
- Squad editor: New Squad and Rename buttons live inside the squad list panel, directly adjacent to the list they operate on.
- Combat mode: per-unit actions (attack, move, end turn) appear in a bar associated with the selected unit, not in a global menu.

---

## 4. Action Bar Clustering

When a bottom bar (or any toolbar) accumulates many buttons, split them into semantic groups separated by whitespace. Context actions (operate on the current selection) cluster on one side; navigation actions (leave the current screen, switch modes) cluster on the other. Spatial separation communicates different purposes at a glance.

**Pattern:** Left cluster = context actions that affect the current selection or state. Right cluster = navigation and mode-exit actions. Maintain this layout consistently across modes.

**Examples:**
- Squad editor bottom bar: left side holds selection-related actions, right side holds the exit/back button.
- Combat action bar: left side holds combat actions (attack, spell, item), right side holds end-turn and retreat.

---

## 5. ESC Cascade

ESC should undo the most recent UI expansion before exiting the mode entirely. If a sub-panel is open, the first ESC closes it. Only a second ESC (with nothing left to close) exits the screen. This gives the player a safe, predictable "back out one level" at every depth of the UI.

**Pattern:** In `HandleInput()`, check `subMenus.AnyActive()` before delegating ESC to `HandleCommonInput()`. If any sub-panel is open, close it and consume the key. Otherwise, let the common handler process the mode exit.

**Examples:**
- Combat mode: pressing ESC with the spell panel open closes the spell panel; pressing ESC again exits combat mode.
- Squad editor: pressing ESC with the roster panel open closes it; a second press returns to the overworld.

---

## 6. Context-Sensitive Panels

The UI should anticipate what the player needs based on their action rather than requiring explicit menu navigation. The interaction context (what was clicked, what is selected) determines which panel appears.

**Pattern:** Call `subMenus.Show("name")` in response to interaction context, not only in response to explicit hotkey presses. Inspect the state of the clicked or selected element to decide which panel is relevant.

**Examples:**
- Squad editor grid: clicking a populated cell opens the unit detail panel; clicking an empty cell opens the roster panel for unit placement.
- Combat mode: selecting a unit with available actions could auto-surface the action bar without requiring a separate keypress.

---

## 7. Keyboard + Mouse Parity

Every panel toggle or action should be reachable by both a keyboard shortcut and a clickable button. Neither input method should be a second-class citizen. Label buttons with their hotkey hint so the player discovers shortcuts organically through normal play.

**Pattern:** Button text includes the hotkey in parentheses (e.g., "Units (U)"). The same key mapped in `HandleInput()` triggers the identical action as clicking the button. Keep hotkey assignments consistent across modes where the same concept applies.

**Examples:**
- Squad editor: "Units (U)" button toggles the units panel; pressing U does the same.
- Combat mode: "Spells (S)" button and the S key both open the spell panel.

---

## 8. Layout Widget Selection

Choose the right ebitenui layout widget for the job. Each of the three layout types solves a different spatial problem; using the wrong one leads to fighting the framework instead of leveraging it.

**AnchorLayout** — Position independent panels at screen edges. Each child gets its own anchor position (start/center/end on both axes) with padding offsets. Use as the root layout of every mode.
*(Current usage: root container of every mode — see `gui/builders/layout.go` anchor helpers)*

**RowLayout** — Stack content in a single direction — vertical lists, horizontal button rows, sequential form elements. Use inside panels for flowing content.
*(Current usage: internal layout of nearly all panels)*

**GridLayout** — Arrange content in a true 2D grid with uniform cell relationships. Children flow left-to-right and wrap at the column count. Use for formation grids, button matrices, or any content where rows and columns both matter.
*(Current usage: 3x3 formation grid in `gui/builders/panels.go`)*

**When NOT to use GridLayout:**
- **Toggle-visibility panels** (combat spell/artifact/inspect, squad editor units/roster): these occupy the same screen slot and only one is visible at a time via `SubMenuController`. Hidden GridLayout children still affect layout space, so use independent anchor-positioned panels with visibility toggling instead.
- **Screen-edge positioning**: panels anchored to different screen edges have no grid relationship. AnchorLayout handles this directly.
- **Single-row content**: a horizontal row of buttons or a vertical list is a RowLayout, not a 1×N grid.

**Pattern:** Default to AnchorLayout at the root, RowLayout inside panels, and reach for GridLayout only when content genuinely forms a multi-row, multi-column matrix.

---

## 9. Consistent Patterns Across Modes

Use the same structural patterns (SubMenuController, ESC cascade, split action bars, hotkey-labeled buttons) in every mode. When a player learns how one screen works, that knowledge should transfer to every other screen. Consistency reduces the learning curve for new features and makes the UI feel cohesive.

**Pattern:** Share `framework.SubMenuController` across all modes. Follow the same ESC cascade logic, the same action bar layout conventions, and the same button labeling style. When introducing a new mode, start from the same structural template.

**Examples:**
- Combat mode and squad editor both use `SubMenuController` for togglable side panels.
- Both modes place navigation/exit actions on the right side of the bottom bar.
- Future modes (inventory, shop, map overview) should adopt the same conventions.

---

## Applying the Checklist

When designing or revising a UI screen, walk through each principle:

1. **Progressive Disclosure** -- Is anything shown by default that the player only occasionally needs? Hide it behind a toggle.
2. **Visual Hierarchy** -- Is the primary interaction surface the most prominent element on screen?
3. **Gestalt Proximity** -- Are actions physically near the objects they affect?
4. **Action Bar Clustering** -- Are context actions separated from navigation actions?
5. **ESC Cascade** -- Does ESC close the innermost open panel before exiting the mode?
6. **Context-Sensitive Panels** -- Does the UI open the right panel automatically based on the player's action?
7. **Keyboard + Mouse Parity** -- Can every action be reached by both input methods? Are hotkeys labeled on buttons?
8. **Layout Widget Selection** -- Is each container using the right layout type? AnchorLayout at the root, RowLayout inside panels, GridLayout only for true 2D grids?
9. **Consistent Patterns** -- Does this screen follow the same structural conventions as existing modes?

---

## Key Files

| Concern | Location |
|---|---|
| SubMenuController | `gui/framework/submenu.go` |
| Mode manager / context switching | `gui/core/` |
| Combat mode (reference) | `gui/guicombat/` |
| Squad editor (reference) | `gui/guisquads/` |
| Layout specs | `gui/specs/layout.go` |
| Layout builder | `gui/builders/layout.go` |
