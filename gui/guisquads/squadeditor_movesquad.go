package guisquads

import (
	"fmt"
	"game_main/gui/framework"
	"game_main/tactical/squadcommands"
	"game_main/tactical/squads"
	"image"

	"github.com/bytearena/ecs"
	"github.com/hajimehoshi/ebiten/v2"
)

// SwapState tracks squad selection for reordering
type SwapState struct {
	SelectedIndex    int  // Index of squad selected for move (-1 if none)
	PrevRightPressed bool // Previous frame's right mouse state (for edge detection)
}

// NewSwapState creates a new swap state
func NewSwapState() *SwapState {
	return &SwapState{
		SelectedIndex: -1,
	}
}

// Reset clears the swap state
func (ss *SwapState) Reset() {
	ss.SelectedIndex = -1
}

// handleSwapCancel checks for ESC key cancellation of swap selection.
// Returns true if input was consumed.
func (sem *SquadEditorMode) handleSwapCancel(inputState *framework.InputState) bool {
	if sem.swapState.SelectedIndex >= 0 {
		if escPressed, ok := inputState.KeysJustPressed[ebiten.KeyEscape]; ok && escPressed {
			sem.swapState.Reset()
			sem.updateStatusLabel("Selection cancelled")
			return true
		}
	}
	return false
}

// executeSquadReorder executes a reorder command and adjusts the current squad index.
func (sem *SquadEditorMode) executeSquadReorder(fromIndex, toIndex int) {
	cmd := squadcommands.NewReorderSquadsCommand(
		sem.Context.ECSManager,
		sem.Context.GetSquadRosterOwnerID(),
		fromIndex,
		toIndex,
	)

	if !sem.CommandHistory.Execute(cmd) {
		sem.updateStatusLabel("Error: Failed to move squad")
		return
	}

	// Sync from roster and refresh UI
	sem.syncSquadOrderFromRoster()
	sem.refreshSquadSelector()
	sem.updateStatusLabel("Squad moved")

	// Adjust current index if needed
	if sem.currentSquadIndex == fromIndex {
		sem.currentSquadIndex = toIndex
	} else if fromIndex < toIndex {
		// Squad moved down, indices in between shift up
		if sem.currentSquadIndex > fromIndex && sem.currentSquadIndex <= toIndex {
			sem.currentSquadIndex--
		}
	} else {
		// Squad moved up, indices in between shift down
		if sem.currentSquadIndex >= toIndex && sem.currentSquadIndex < fromIndex {
			sem.currentSquadIndex++
		}
	}
}

// handleSwapInput processes mouse input for right-click select and move squad reordering.
// Returns true if input was consumed (prevents pass-through to list selection).
func (sem *SquadEditorMode) handleSwapInput(inputState *framework.InputState) bool {
	// Disable swap if only one or zero squads
	if len(sem.allSquadIDs) <= 1 {
		return false
	}

	// Get list bounds (needed for hit testing)
	listRect := sem.getSquadListBounds()
	if listRect.Empty() {
		return false
	}

	mouseX, mouseY := inputState.MouseX, inputState.MouseY
	mouseInList := image.Pt(mouseX, mouseY).In(listRect)

	// Detect mouse button edges
	// Check directly from ebiten for more reliable right-click detection
	rightButtonPressed := ebiten.IsMouseButtonPressed(ebiten.MouseButtonRight)
	rightJustPressed := rightButtonPressed && !sem.swapState.PrevRightPressed

	// Update previous state for next frame
	defer func() {
		sem.swapState.PrevRightPressed = rightButtonPressed
	}()

	// ESC cancels selection
	if sem.handleSwapCancel(inputState) {
		return true
	}

	// Right-click: Select squad OR move selected squad
	if rightJustPressed && mouseInList {
		clickedIndex := sem.calculateEntryIndexAtPosition(mouseY, listRect)

		if clickedIndex < 0 || clickedIndex >= len(sem.allSquadIDs) {
			return false
		}

		// No squad selected yet: select this squad
		if sem.swapState.SelectedIndex < 0 {
			sem.swapState.SelectedIndex = clickedIndex
			squadName := sem.Queries.SquadCache.GetSquadName(sem.allSquadIDs[clickedIndex])
			sem.updateStatusLabel(fmt.Sprintf("Selected '%s' - right-click where to move", squadName))
			return true
		}

		// Right-click on same position: cancel selection
		if clickedIndex == sem.swapState.SelectedIndex {
			sem.swapState.Reset()
			sem.updateStatusLabel("Selection cancelled")
			return true
		}

		// Right-click on different position: execute move
		sem.executeSquadReorder(sem.swapState.SelectedIndex, clickedIndex)
		sem.swapState.Reset()
		return true
	}

	return false
}

// calculateEntryIndexAtPosition calculates which list entry is under the given Y position
func (sem *SquadEditorMode) calculateEntryIndexAtPosition(mouseY int, listRect image.Rectangle) int {
	// Entry height estimation: font (20px) + padding (10px) = 30px
	const entryHeight = 30

	relativeY := mouseY - listRect.Min.Y
	entryIndex := relativeY / entryHeight

	// Clamp to valid range
	if entryIndex < 0 {
		return -1
	}
	if entryIndex >= len(sem.allSquadIDs) {
		return -1
	}

	return entryIndex
}

// getSquadListBounds returns the screen bounds of the squad selector list
func (sem *SquadEditorMode) getSquadListBounds() image.Rectangle {
	if sem.squadSelector == nil {
		return image.Rectangle{}
	}

	// Get widget rect from ebitenui
	rect := sem.squadSelector.GetWidget().Rect
	return image.Rect(rect.Min.X, rect.Min.Y, rect.Max.X, rect.Max.Y)
}

// syncSquadOrderFromRoster updates allSquadIDs from the roster (source of truth)
func (sem *SquadEditorMode) syncSquadOrderFromRoster() {
	rosterOwnerID := sem.Context.GetSquadRosterOwnerID()
	manager := sem.Context.ECSManager

	// Get roster from active squad roster owner (commander or player)
	roster := squads.GetPlayerSquadRoster(rosterOwnerID, manager)
	if roster == nil {
		return
	}

	// Copy from roster to local state
	sem.allSquadIDs = make([]ecs.EntityID, len(roster.OwnedSquads))
	copy(sem.allSquadIDs, roster.OwnedSquads)
}

// updateStatusLabel updates the status label text
func (sem *SquadEditorMode) updateStatusLabel(text string) {
	if sem.StatusLabel != nil {
		sem.StatusLabel.Label = text
	}
}
