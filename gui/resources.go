// This file re-exports types and values from gui/resources/ for backward compatibility.
// New code should import "game_main/gui/resources" directly.
package gui

import "game_main/gui/resources"

// Re-export font resources
var (
	SmallFace  = resources.SmallFace
	LargeFace  = resources.LargeFace
)

// Re-export widget resources
var (
	PanelRes         = resources.PanelRes
	ListRes          = resources.ListRes
	TextAreaRes      = resources.TextAreaRes
	ButtonImage      = resources.ButtonImage
	DefaultWidgetColor = resources.DefaultWidgetColor
)

// Re-export helper functions
var (
	HexToColor = resources.HexToColor
)

// Re-export constants
const (
	TextDisabledColor = resources.TextDisabledColor
)

// Re-export panel width constants
const (
	PanelWidthNarrow    = resources.PanelWidthNarrow
	PanelWidthStandard  = resources.PanelWidthStandard
	PanelWidthMedium    = resources.PanelWidthMedium
	PanelWidthWide      = resources.PanelWidthWide
	PanelWidthExtraWide = resources.PanelWidthExtraWide
)

// Re-export panel height constants
const (
	PanelHeightTiny    = resources.PanelHeightTiny
	PanelHeightSmall   = resources.PanelHeightSmall
	PanelHeightQuarter = resources.PanelHeightQuarter
	PanelHeightThird   = resources.PanelHeightThird
	PanelHeightHalf    = resources.PanelHeightHalf
	PanelHeightTall    = resources.PanelHeightTall
	PanelHeightFull    = resources.PanelHeightFull
)

// Re-export padding constants
const (
	PaddingTight    = resources.PaddingTight
	PaddingStandard = resources.PaddingStandard
	PaddingLoose    = resources.PaddingLoose
)

// Re-export layout offset constants
const (
	BottomButtonOffset = resources.BottomButtonOffset
)

// Re-export mode layout constants
const (
	ExplorationMapWidth = resources.ExplorationMapWidth
	CombatLogHeight     = resources.CombatLogHeight
	InventoryListWidth  = resources.InventoryListWidth
	InventoryListHeight = resources.InventoryListHeight
	SquadMgmtButtonHeight = resources.SquadMgmtButtonHeight
	SquadBuilderGridWidth = resources.SquadBuilderGridWidth
	SquadBuilderUnitListWidth = resources.SquadBuilderUnitListWidth
	SquadBuilderUnitListHeight = resources.SquadBuilderUnitListHeight
	SquadBuilderInfoWidth = resources.SquadBuilderInfoWidth
	SquadBuilderInfoHeight = resources.SquadBuilderInfoHeight
	FormationGridWidth = resources.FormationGridWidth
	InfoContentWidth = resources.InfoContentWidth
)
