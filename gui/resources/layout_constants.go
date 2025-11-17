// Package resources provides GUI layout constants, asset resources, and panel specifications
// for the TinkerRogue game UI system.
package resources

// ========================================
// PANEL WIDTH PERCENTAGES
// ========================================

const (
	// Narrow side panels (stats display, faction info)
	PanelWidthNarrow = 0.15

	// Standard side panels (squad lists, inventory filters)
	PanelWidthStandard = 0.2

	// Medium panels (filters, secondary content)
	PanelWidthMedium = 0.3

	// Wide panels (top bars, main content areas)
	PanelWidthWide = 0.4

	// Extra wide panels (detail views, full-width content)
	PanelWidthExtraWide = 0.45
)

// ========================================
// PANEL HEIGHT PERCENTAGES
// ========================================

const (
	// Tiny panels (top bars, button containers)
	PanelHeightTiny = 0.08

	// Small panels (faction info, header sections)
	PanelHeightSmall = 0.12

	// Quarter screen height
	PanelHeightQuarter = 0.25

	// Third of screen height
	PanelHeightThird = 0.33

	// Half screen height
	PanelHeightHalf = 0.5

	// Tall panels (detail views, list containers)
	PanelHeightTall = 0.75

	// Nearly full screen (main content areas)
	PanelHeightFull = 0.85
)

// ========================================
// STANDARD PADDING PERCENTAGES
// ========================================

const (
	// Tight padding for most panels
	PaddingTight = 0.01

	// Standard padding between elements
	PaddingStandard = 0.02

	// Loose padding for spacious layouts
	PaddingLoose = 0.03
)

// ========================================
// LAYOUT OFFSET CONSTANTS
// ========================================

const (
	// Space reserved at bottom for button containers
	BottomButtonOffset = 0.08
)

// ========================================
// UI MODE LAYOUT SPECIFICATIONS
// ========================================

// ExplorationModeLayout defines layout constants for exploration mode
const (
	// Map display and main content
	ExplorationMapWidth = 0.7
)

// CombatModeLayout defines layout constants for combat mode
const (
	// Combat log (bottom)
	CombatLogHeight = 0.2
)

// InventoryModeLayout defines layout constants for inventory mode
const (
	// Item list (main area)
	InventoryListWidth  = 0.5
	InventoryListHeight = PanelHeightTall
)

// SquadManagementLayout defines layout constants for squad management mode
const (
	// Action buttons (bottom)
	SquadMgmtButtonHeight = 0.1
)

// SquadBuilderLayout defines layout constants for squad builder mode
const (
	// Grid area (left side)
	SquadBuilderGridWidth = 0.5

	// Unit list panel (right side)
	SquadBuilderUnitListWidth  = 0.25
	SquadBuilderUnitListHeight = PanelHeightTall

	// Squad info panel (right side, above unit list)
	SquadBuilderInfoWidth  = 0.25
	SquadBuilderInfoHeight = 0.2
)

// FormationEditorLayout defines layout constants for formation editor mode
const (
	// Formation grid preview (center)
	FormationGridWidth = 0.5
)

// InfoModeLayout defines layout constants for info display mode
const (
	// Main content area
	InfoContentWidth = PanelWidthExtraWide
)
