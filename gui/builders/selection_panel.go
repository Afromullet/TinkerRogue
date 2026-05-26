package builders

import (
	"image/color"

	"game_main/gui/specs"
	"game_main/gui/widgetresources"
	"game_main/gui/widgets"

	"github.com/ebitenui/ebitenui/widget"
)

// SelectionPanelConfig describes a right-anchored, vertically-stacked
// selection panel containing an optional status label, a list, a detail text
// area, an action button, and a cancel button. It is used by both the combat
// spell selection panel and the artifact activation panel — historically two
// 90-LOC copy-paste siblings inside combat_panels_registry.go.
//
// All fraction fields are screen-relative; pass values straight from specs/*.go
// (e.g. specs.CombatSpellPanelWidth).
//
// Scope — only adopt this for a new panel if ALL of the following hold:
//  1. Right-anchored, vertical list + detail + action + cancel layout.
//  2. Toggleable submenu (hidden by default, shown via a sub-menu controller).
//  3. Exactly one selection list.
//  4. Exactly one action button, enabled purely by "valid selection".
//  5. Exactly one cancel button.
//  6. Plain text detail area (no rich widgets, equipment slots, formation grids).
//
// If any criterion fails, build the panel inline or extract a different
// helper — do NOT widen this config with one-off flags. The guiprogression
// library panels (dual-list, no cancel, persistent) and the guisquads panels
// (multi-action, map-click activation, tab-coupled) are deliberate non-fits.
type SelectionPanelConfig struct {
	PanelWidthFrac   float64
	PanelHeightFrac  float64
	ListHeightFrac   float64
	DetailHeightFrac float64

	// StatusLabelText: when non-empty, a small label is added above the list
	// (used by the spell panel for "Mana: 0/0"). Empty string omits the label
	// entirely (the artifact panel does this).
	StatusLabelText string

	DetailPlaceholder string // e.g. "Select a spell to view details"
	ActionButtonText  string // e.g. "Cast", "Activate"
	CancelButtonText  string // e.g. "Cancel (ESC)"

	EntryLabelFunc  func(entry interface{}) string
	OnEntrySelected func(entry interface{})
	OnActionClicked func()
	OnCancelClicked func()
}

// SelectionPanelWidgets bundles the widget references created by
// BuildSelectionPanel so callers can wire them into their panel controllers.
// StatusLabel is nil when SelectionPanelConfig.StatusLabelText was empty.
type SelectionPanelWidgets struct {
	Container    *widget.Container
	StatusLabel  *widget.Text
	List         *widgets.CachedListWrapper
	Detail       *widgets.CachedTextAreaWrapper
	ActionButton *widget.Button
}

// BuildSelectionPanel constructs the standard list+detail+action+cancel layout
// shared by the spell and artifact combat panels. The returned container is
// hidden by default — callers register it with their sub-menu controller and
// show it via Toggle().
func BuildSelectionPanel(layout *specs.LayoutConfig, cfg SelectionPanelConfig) SelectionPanelWidgets {
	panelWidth := int(float64(layout.ScreenWidth) * cfg.PanelWidthFrac)
	panelHeight := int(float64(layout.ScreenHeight) * cfg.PanelHeightFrac)

	container := CreatePanelWithConfig(ContainerConfig{
		MinWidth:   panelWidth,
		MinHeight:  panelHeight,
		Background: widgetresources.PanelRes.Image,
		Layout: widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Spacing(5),
			widget.RowLayoutOpts.Padding(NewResponsiveRowPadding(layout, specs.PaddingExtraSmall)),
		),
	})

	rightPad := int(float64(layout.ScreenWidth) * specs.PaddingTight)
	container.GetWidget().LayoutData = AnchorEndCenter(rightPad)

	var statusLabel *widget.Text
	if cfg.StatusLabelText != "" {
		statusLabel = CreateSmallLabel(cfg.StatusLabelText)
		container.AddChild(statusLabel)
	}

	listWidth := panelWidth - 20
	listHeight := int(float64(layout.ScreenHeight) * cfg.ListHeightFrac)
	listWidget := CreateListWithConfig(ListConfig{
		Entries:         []interface{}{},
		MinWidth:        listWidth,
		MinHeight:       listHeight,
		EntryLabelFunc:  cfg.EntryLabelFunc,
		OnEntrySelected: cfg.OnEntrySelected,
	})
	cachedList := widgets.NewCachedListWrapper(listWidget)
	container.AddChild(listWidget)

	detailWidth := panelWidth - 20
	detailHeight := int(float64(layout.ScreenHeight) * cfg.DetailHeightFrac)
	rawDetail := CreateTextAreaWithConfig(TextAreaConfig{
		MinWidth:  detailWidth,
		MinHeight: detailHeight,
		FontColor: color.White,
	})
	rawDetail.SetText(cfg.DetailPlaceholder)
	container.AddChild(rawDetail)
	detailWrapper := widgets.NewCachedTextAreaWrapper(rawDetail)

	actionButton := CreateButtonWithConfig(ButtonConfig{
		Text:    cfg.ActionButtonText,
		OnClick: cfg.OnActionClicked,
	})
	actionButton.GetWidget().Disabled = true
	container.AddChild(actionButton)

	cancelButton := CreateButtonWithConfig(ButtonConfig{
		Text:    cfg.CancelButtonText,
		OnClick: cfg.OnCancelClicked,
	})
	container.AddChild(cancelButton)

	container.GetWidget().Visibility = widget.Visibility_Hide

	return SelectionPanelWidgets{
		Container:    container,
		StatusLabel:  statusLabel,
		List:         cachedList,
		Detail:       detailWrapper,
		ActionButton: actionButton,
	}
}
