package guiprogression

import (
	"image/color"

	"game_main/gui/builders"
	"game_main/gui/framework"
	"game_main/gui/specs"
	"game_main/gui/widgets"

	"github.com/ebitenui/ebitenui/widget"
)

// Panel type constants for progression mode.
const (
	ProgressionPanelHeader framework.PanelType = "progression_header"
	ProgressionPanelPerks  framework.PanelType = "progression_perks"
	ProgressionPanelSpells framework.PanelType = "progression_spells"
)

// libraryPanelPicker returns the libraryPanelController driving a given panel
// instance. Used by callbacks wired into the panel at build time, which fire
// only after the mode's controller has been constructed.
type libraryPanelPicker func(pm *ProgressionMode) *libraryPanelController

func init() {
	framework.RegisterPanel(ProgressionPanelHeader, framework.PanelDescriptor{
		Content:  framework.ContentCustom,
		OnCreate: buildHeaderPanel,
	})

	// Perks panel: anchored left, controller selected via pm.controller.perkPanel.
	framework.RegisterPanel(ProgressionPanelPerks, framework.PanelDescriptor{
		Content: framework.ContentCustom,
		OnCreate: buildLibraryPanel(
			perkLibrarySource,
			func(layout *specs.LayoutConfig, c *widget.Container) {
				leftPad := int(float64(layout.ScreenWidth) * specs.PaddingStandard)
				topOffset := int(float64(layout.ScreenHeight) * (specs.ProgressionHeaderHeight + specs.PaddingStandard))
				c.GetWidget().LayoutData = builders.AnchorStartStart(leftPad, topOffset)
			},
			func(pm *ProgressionMode) *libraryPanelController { return pm.controller.perkPanel },
		),
	})

	// Spells panel: anchored right, controller selected via pm.controller.spellPanel.
	framework.RegisterPanel(ProgressionPanelSpells, framework.PanelDescriptor{
		Content: framework.ContentCustom,
		OnCreate: buildLibraryPanel(
			spellLibrarySource,
			func(layout *specs.LayoutConfig, c *widget.Container) {
				rightPad := int(float64(layout.ScreenWidth) * specs.PaddingStandard)
				topOffset := int(float64(layout.ScreenHeight) * (specs.ProgressionHeaderHeight + specs.PaddingStandard))
				c.GetWidget().LayoutData = builders.AnchorEndStart(rightPad, topOffset)
			},
			func(pm *ProgressionMode) *libraryPanelController { return pm.controller.spellPanel },
		),
	})
}

func buildHeaderPanel(result *framework.PanelResult, mode framework.UIMode) error {
	pm := mode.(*ProgressionMode)
	layout := pm.Layout

	panelWidth := int(float64(layout.ScreenWidth) * specs.ProgressionHeaderWidth)
	panelHeight := int(float64(layout.ScreenHeight) * specs.ProgressionHeaderHeight)

	result.Container = builders.CreateStaticPanel(builders.ContainerConfig{
		MinWidth:  panelWidth,
		MinHeight: panelHeight,
		Layout: widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionHorizontal),
			widget.RowLayoutOpts.Spacing(20),
			widget.RowLayoutOpts.Padding(builders.NewResponsiveRowPadding(layout, specs.PaddingExtraSmall)),
		),
	})

	topPad := int(float64(layout.ScreenHeight) * specs.PaddingExtraSmall)
	result.Container.GetWidget().LayoutData = builders.AnchorCenterStart(topPad)

	title := builders.CreateSmallLabel("Progression Library")
	arcanaLabel := builders.CreateSmallLabel("Arcana: 0")
	skillLabel := builders.CreateSmallLabel("Skill: 0")

	result.Container.AddChild(title)
	result.Container.AddChild(arcanaLabel)
	result.Container.AddChild(skillLabel)

	result.Custom["arcanaLabel"] = arcanaLabel
	result.Custom["skillLabel"] = skillLabel
	return nil
}

// buildLibraryPanel returns an OnCreate callback that constructs a two-list
// library panel. Position is applied by `anchor` and the runtime controller is
// selected from the mode via `pickPanel` (safe because callbacks fire after
// the controller is constructed).
func buildLibraryPanel(
	source librarySource,
	anchor func(*specs.LayoutConfig, *widget.Container),
	pickPanel libraryPanelPicker,
) func(*framework.PanelResult, framework.UIMode) error {
	return func(result *framework.PanelResult, mode framework.UIMode) error {
		pm := mode.(*ProgressionMode)
		layout := pm.Layout

		panelWidth := int(float64(layout.ScreenWidth) * specs.ProgressionLibraryPanelWidth)
		panelHeight := int(float64(layout.ScreenHeight) * specs.ProgressionLibraryPanelHeight)

		result.Container = builders.CreateStaticPanel(builders.ContainerConfig{
			MinWidth:  panelWidth,
			MinHeight: panelHeight,
			Layout: widget.NewRowLayout(
				widget.RowLayoutOpts.Direction(widget.DirectionVertical),
				widget.RowLayoutOpts.Spacing(5),
			),
		})
		anchor(layout, result.Container)

		entryLabel := func(e interface{}) string { return e.(*libraryEntry).label }

		result.Container.AddChild(builders.CreateSmallLabel(source.label))

		result.Container.AddChild(builders.CreateSmallLabel("Unlocked:"))
		unlockedBase := builders.CreateListWithConfig(builders.ListConfig{
			Entries:        []interface{}{},
			MinWidth:       panelWidth - 20,
			MinHeight:      120,
			EntryLabelFunc: entryLabel,
			OnEntrySelected: func(e interface{}) {
				pickPanel(pm).onUnlockedSelected(e.(*libraryEntry))
			},
		})
		unlocked := widgets.NewCachedListWrapper(unlockedBase)
		result.Container.AddChild(unlockedBase)

		result.Container.AddChild(builders.CreateSmallLabel("Locked:"))
		lockedBase := builders.CreateListWithConfig(builders.ListConfig{
			Entries:        []interface{}{},
			MinWidth:       panelWidth - 20,
			MinHeight:      180,
			EntryLabelFunc: entryLabel,
			OnEntrySelected: func(e interface{}) {
				pickPanel(pm).onLockedSelected(e.(*libraryEntry))
			},
		})
		locked := widgets.NewCachedListWrapper(lockedBase)
		result.Container.AddChild(lockedBase)

		detail := builders.CreateCachedTextArea(builders.TextAreaConfig{
			MinWidth:  panelWidth - 20,
			MinHeight: 100,
			FontColor: color.White,
		})
		detail.SetText(source.detailPrompt)
		result.Container.AddChild(detail)

		unlockBtn := builders.CreateButtonWithConfig(builders.ButtonConfig{
			Text:    source.unlockBtnText,
			OnClick: func() { pickPanel(pm).onUnlockClicked() },
		})
		unlockBtn.GetWidget().Disabled = true
		result.Container.AddChild(unlockBtn)

		result.Custom["unlockedList"] = unlocked
		result.Custom["lockedList"] = locked
		result.Custom["detail"] = detail
		result.Custom["unlockBtn"] = unlockBtn
		return nil
	}
}
