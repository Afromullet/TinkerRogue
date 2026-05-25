package widgets

import (
	"github.com/ebitenui/ebitenui/widget"
)

// SelectionPanelCore holds the widget references and selection state shared
// by the combat spell and artifact panels (built via
// builders.BuildSelectionPanel). Typed controllers embed it and provide the
// panel-specific bits — precondition checks, detail text formatting, and
// handler-typed selection wrappers.
//
// Selected is intentionally interface{}; typed wrappers cast it back to
// their domain type (*templates.SpellDefinition, *ArtifactOption, …).
type SelectionPanelCore struct {
	List         *CachedListWrapper
	Detail       *CachedTextAreaWrapper
	StatusLabel  *widget.Text // nil when the panel has no status field
	ActionButton *widget.Button
	Selected     interface{}

	ShowSubmenu  func()
	CloseSubmenu func()
}

// SetEntries replaces the list contents and marks it dirty so the next frame
// rerenders. Safe to call when the list is nil.
func (c *SelectionPanelCore) SetEntries(entries []interface{}) {
	if c.List == nil {
		return
	}
	c.List.GetList().SetEntries(entries)
	c.List.MarkDirty()
}

// SetStatusText updates the optional status label (e.g. "Mana: 12/30").
// No-op when the panel has no status label.
func (c *SelectionPanelCore) SetStatusText(text string) {
	if c.StatusLabel == nil {
		return
	}
	c.StatusLabel.Label = text
}

// SetDetail writes text to the detail area and toggles the action button's
// enabled state. Pass canExecute=false to disable the button (e.g. not enough
// mana, charge spent).
func (c *SelectionPanelCore) SetDetail(text string, canExecute bool) {
	if c.Detail != nil {
		c.Detail.SetText(text)
	}
	if c.ActionButton != nil {
		c.ActionButton.GetWidget().Disabled = !canExecute
	}
}

// ClearSelection resets Selected, restores the detail placeholder, and
// disables the action button. Called by Refresh after repopulating the list.
func (c *SelectionPanelCore) ClearSelection(detailPlaceholder string) {
	c.Selected = nil
	c.SetDetail(detailPlaceholder, false)
}

// Hide clears Selected and closes the sub-menu container.
func (c *SelectionPanelCore) Hide() {
	c.Selected = nil
	if c.CloseSubmenu != nil {
		c.CloseSubmenu()
	}
}
